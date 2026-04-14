package optimizer

import (
	"github.com/username/plsql-parser/pkg/types"
	"fmt"
	"regexp"
	"strings"
)

// OutputLevel 输出级别
type OutputLevel int

const (
	LevelClean OutputLevel = 1 // L1: 物理去噪
	LevelLean  OutputLevel = 2 // L2: 内容精简
)

// 预编译正则表达式以提高性能
var (
	// singleLineCommentRegex 匹配 -- 单行注释
	singleLineCommentRegex = regexp.MustCompile(`--[^\r\n]*`)
	// stringLiteralRegex 匹配字符串字面量（包括转义的单引号 ''）
	stringLiteralRegex = regexp.MustCompile(`'(?:''|[^'])*'`)
)

// OptimizeOptions 优化选项
type OptimizeOptions struct {
	Level OutputLevel
}

// GetOptions 根据级别获取配置
func GetOptions(level OutputLevel) *OptimizeOptions {
	return &OptimizeOptions{Level: level}
}

// Optimize 优化解析结果
func Optimize(result *types.ParseResult, options *OptimizeOptions) *types.ParseResult {
	if result == nil {
		return nil
	}
	
	if options == nil {
		options = GetOptions(LevelClean)
	}

	// 创建优化后的结果，不包含 Source 字段
	optimized := &types.ParseResult{
		Root:      cloneNode(result.Root),
		Nodes:     make([]*types.Node, 0),
		Errors:    result.Errors,
		Warnings:  result.Warnings,
		LineCount: result.LineCount,
		// 不再包含 Source 字段
	}

	// 逐层处理节点
	for _, node := range result.Nodes {
		processed := processNode(node, options.Level)
		if processed != nil {
			optimized.Nodes = append(optimized.Nodes, processed)
		}
	}

	return optimized
}

// cloneNode 简单克隆节点
func cloneNode(node *types.Node) *types.Node {
	if node == nil {
		return nil
	}
	return &types.Node{
		Type:     node.Type,
		Name:     node.Name,
		Text:     node.Text,
		Start:    node.Start,
		End:      node.End,
		Metadata: node.Metadata,
	}
}

// cleanNoise 物理去噪 (L1)
func cleanNoise(node *types.Node) *types.Node {
	n := &types.Node{
		Type: node.Type,
		Name: node.Name,
		Text: node.Text,
		Start: types.Position{
			Line:   node.Start.Line,
			Column: node.Start.Column,
		},
		End: types.Position{
			Line:   node.End.Line,
			Column: node.End.Column,
		},
	}

	// 仅保留有实际内容的 Metadata
	if len(node.Metadata) > 0 {
		n.Metadata = make(types.Metadata)
		for k, v := range node.Metadata {
			n.Metadata[k] = v
		}
	}

	return n
}

// processNode 核心分发逻辑
func processNode(node *types.Node, level OutputLevel) *types.Node {
	if node == nil {
		return nil
	}

	// L1: 物理去噪 - 剔除无用节点
	if shouldDiscard(node) {
		return nil
	}

	// 克隆节点（浅拷贝）
	cleaned := cleanNoise(node)

	if level < LevelLean {
		return cleaned
	}

	// L2: 内容精简 - 对保留的节点进行文本压缩
	precomputeUpperAndSimplify(cleaned)

	return cleaned
}

// shouldDiscard 判断是否应该丢弃节点 (L1)
func shouldDiscard(node *types.Node) bool {
	// 仅剔除变量与常量声明节点
	if node.Type == types.NodeTypeVariable || node.Type == types.NodeTypeConstant {
		return true
	}

	// 不清除任何过程或函数，保留所有业务逻辑
	return false
}

// precomputeUpperAndSimplify 预计算大写文本并执行 L2 精简
func precomputeUpperAndSimplify(node *types.Node) {
	if node.Text == "" {
		return
	}

	// 先清理注释（在所有其他处理之前）
	cleanComments(node)

	upperText := strings.ToUpper(node.Text)

	// 1. SQL SELECT 语句折叠：保留表名和关键结构，折叠字段列表
	if node.Type == types.NodeTypeSQLStatement && strings.HasPrefix(upperText, "SELECT") {
		fromIdx := strings.Index(upperText, " FROM ")
		// 确保找到 FROM 且 SELECT 和 FROM 之间有足够多的内容才折叠
		if fromIdx > 10 {
			node.Text = fmt.Sprintf("SELECT [Fields List Collapsed]%s", node.Text[fromIdx:])
		}
		return
	}

	// 2. 机械式映射折叠：识别 PUT_JSON_FIELD 或大量拼接
	if strings.Count(upperText, "PUT_JSON_FIELD") > 5 || strings.Count(upperText, "L_REQUEST_DATA :=") > 10 {
		lines := strings.Split(node.Text, "\n")
		if len(lines) > 10 {
			var sb strings.Builder
			sb.WriteString(lines[0])
			sb.WriteString("\n")
			sb.WriteString(fmt.Sprintf("... [Mechanical Mapping: %d lines collapsed] ...\n", len(lines)-2))
			sb.WriteString(lines[len(lines)-1])
			node.Text = sb.String()
		}
		return
	}

	// 3. 简单包装函数隐藏实现
	if (node.Type == types.NodeTypeFunction || node.Type == types.NodeTypeProcedure) && len(node.Text) < 200 {
		complexKeywords := []string{"CURSOR", "LOOP", "UPDATE", "INSERT", "DELETE", "MERGE"}
		for _, kw := range complexKeywords {
			if strings.Contains(upperText, kw) {
				return // 包含复杂逻辑，不折叠
			}
		}
		// 执行折叠
		lines := strings.Split(node.Text, "\n")
		if len(lines) > 2 {
			node.Text = fmt.Sprintf("%s\n  [Implementation Hidden: Simple Wrapper]\n%s", lines[0], lines[len(lines)-1])
		}
	}
}

// cleanComments 清理注释（不误删字符串字面量中的内容）
// 使用状态机正确处理嵌套注释
func cleanComments(node *types.Node) {
	if node.Text == "" {
		return
	}

	// 第一步：使用占位符保护字符串字面量
	placeholders := make([]string, 0)
	protected := node.Text

	protected = stringLiteralRegex.ReplaceAllStringFunc(protected, func(match string) string {
		placeholders = append(placeholders, match)
		return fmt.Sprintf("\x00SP%d\x00", len(placeholders)-1)
	})

	// 第二步：使用状态机移除注释
	var result strings.Builder
	i := 0

	for i < len(protected) {
		// 检查是否到达占位符 \x00SP{n}\x00
		if i+4 < len(protected) && protected[i] == '\x00' && protected[i+1] == 'S' && protected[i+2] == 'P' {
			// 找到第二个 \x00 的位置
			end := strings.Index(protected[i+3:], "\x00")
			if end > 0 {
				idxStr := protected[i+3 : i+3+end]
				var idx int
				if _, err := fmt.Sscanf(idxStr, "%d", &idx); err == nil && idx < len(placeholders) {
					result.WriteString(placeholders[idx])
				}
				i += 3 + end + 1 // 跳过 \x00SP{idx}\x00
				continue
			}
		}

		// 检查单行注释 --
		if i+1 < len(protected) && protected[i] == '-' && protected[i+1] == '-' {
			// 跳过到行尾
			for i < len(protected) && protected[i] != '\n' && protected[i] != '\r' {
				i++
			}
			continue
		}

		// 检查多行注释 /*（支持嵌套）*/
		if i+1 < len(protected) && protected[i] == '/' && protected[i+1] == '*' {
			depth := 1
			i += 2
			for i < len(protected) && depth > 0 {
				// 检查嵌套的 /*
				if i+1 < len(protected) && protected[i] == '/' && protected[i+1] == '*' {
					depth++
					i += 2
					continue
				}
				// 检查结束的 */
				if i+1 < len(protected) && protected[i] == '*' && protected[i+1] == '/' {
					depth--
					i += 2
					continue
				}
				// 多行注释内嵌套的单行注释 -- 也需要跳过
				if i+1 < len(protected) && protected[i] == '-' && protected[i+1] == '-' {
					// 跳过到行尾
					for i < len(protected) && protected[i] != '\n' && protected[i] != '\r' {
						i++
					}
					continue
				}
				// 占位符在注释内，直接跳过
				if i+4 < len(protected) && protected[i] == '\x00' && protected[i+1] == 'S' && protected[i+2] == 'P' {
					end := strings.Index(protected[i+3:], "\x00")
					if end > 0 {
						i += 3 + end + 1
						continue
					}
				}
				i++
			}
			// 移除注释后，添加一个空格来分隔周围的内容（避免连续单词合并）
			result.WriteByte(' ')
			continue
		}

		// 普通字符，直接复制
		result.WriteByte(protected[i])
		i++
	}

	text := result.String()

	// 第三步：清理多余的空行和空格
	lines := strings.Split(text, "\n")
	var cleanedLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cleanedLines = append(cleanedLines, trimmed)
		}
	}

	if len(cleanedLines) > 0 {
		text = strings.Join(cleanedLines, "\n")
		// 规范化连续空格为单空格
		text = regexp.MustCompile(`[ \t]+`).ReplaceAllString(text, " ")
		node.Text = text
	} else {
		node.Text = ""
	}
}
