package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/username/plsql-parser/internal/plsql"
	"github.com/username/plsql-parser/pkg/optimizer"
	"github.com/username/plsql-parser/pkg/types"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// 先检查帮助参数
	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" {
			printUsage()
			os.Exit(0)
		}
	}

	filename := os.Args[1]

	// 检查参数
	outputJSON := false
	outputFileArg := ""
	verbose := false
	optLevel := optimizer.LevelClean // 默认 L1 去噪模式

	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--json", "-j":
			outputJSON = true
		case "--output", "-o":
			if i+1 < len(os.Args) {
				outputFileArg = os.Args[i+1]
				i++
			} else {
				fmt.Fprintf(os.Stderr, "错误: --output 需要指定文件路径\n")
				os.Exit(1)
			}
		case "--verbose", "-v":
			verbose = true
		case "-L":
			if i+1 < len(os.Args) {
				levelStr := strings.ToLower(os.Args[i+1])
				switch levelStr {
				case "1", "clean":
					optLevel = optimizer.LevelClean
				case "2", "lean":
					optLevel = optimizer.LevelLean
				default:
					fmt.Fprintf(os.Stderr, "未知级别: %s (支持: 1/2)\n", os.Args[i+1])
					os.Exit(1)
				}
				i++
			} else {
				fmt.Fprintf(os.Stderr, "错误: -L 需要指定级别 (1-2)\n")
				os.Exit(1)
			}
		case "--help", "-h":
			printUsage()
			os.Exit(0)
		default:
			fmt.Fprintf(os.Stderr, "未知参数: %s\n", os.Args[i])
			printUsage()
			os.Exit(1)
		}
	}

	// 读取文件
	// 检查文件大小，防止 OOM
	const maxFileSize = 10 * 1024 * 1024 // 10MB
	fileInfo, err := os.Stat(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 无法获取文件信息 %s: %v\n", filename, err)
		os.Exit(1)
	}
	if fileInfo.Size() > maxFileSize {
		fmt.Fprintf(os.Stderr, "错误: 文件过大 (最大 %d MB)\n", maxFileSize/(1024*1024))
		os.Exit(1)
	}
	
	code, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 无法读取文件 %s: %v\n", filename, err)
		os.Exit(1)
	}

	// 解析
	parser := plsql.NewANTLRRealParser(nil)
	result, err := parser.Parse(string(code))
	if err != nil {
		fmt.Fprintf(os.Stderr, "解析失败: %v\n", err)
		os.Exit(1)
	}

	// 根据级别优化结果
	optimizedResult := optimizer.Optimize(result, optimizer.GetOptions(optLevel))

	// 输出结果
	if outputJSON {
		outputJSONResult(optimizedResult, outputFileArg)
	} else {
		outputText(optimizedResult, verbose)
	}
}

func printUsage() {
	fmt.Println(`PL/SQL 解析器 - 基于 ANTLR v4.13.0

用法:
  plsql-parser <文件> [选项]

选项:
  -j, --json           JSON 格式输出
  -o, --output <文件>   输出到文件
  -L <1-2>             优化级别: 1(去噪), 2(精简)
  -v, --verbose        详细输出
  -h, --help           显示帮助信息

优化级别说明:
  L1 (Clean)  - 物理去噪：移除变量声明、调试过程
  L2 (Lean)   - 内容精简：折叠 SQL 字段列表、机械映射代码

示例:
  plsql-parser example/test.pkg -j -L 1
  plsql-parser example/test.pkg -j -L 2 -o result.json`)
}

func outputJSONResult(result *types.ParseResult, outputFile string) {
	// 使用 json.Encoder 替代 json.MarshalIndent，避免对 < > 进行 HTML 转义
	var buf strings.Builder
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false) // 禁用 HTML 特殊字符转义
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "JSON 编码失败: %v\n", err)
		os.Exit(1)
	}

	// Encode 会在末尾添加换行符，移除它
	output := strings.TrimSuffix(buf.String(), "\n")

	if outputFile != "" {
		err := os.WriteFile(outputFile, []byte(output), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "写入文件失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ 结果已保存到: %s\n", outputFile)
	} else {
		fmt.Println(output)
	}
}

func outputText(result *types.ParseResult, verbose bool) {
	var sb strings.Builder

	sb.WriteString("========================================\n")
	sb.WriteString("  PL/SQL 解析结果\n")
	sb.WriteString("========================================\n\n")

	sb.WriteString(fmt.Sprintf("代码行数: %d\n", result.LineCount))
	sb.WriteString(fmt.Sprintf("节点数量: %d\n", len(result.Nodes)))
	sb.WriteString(fmt.Sprintf("解析错误: %d\n", len(result.Errors)))
	sb.WriteString(fmt.Sprintf("解析警告: %d\n\n", len(result.Warnings)))

	// 统计各类节点
	stats := make(map[string]int)
	for _, node := range result.Nodes {
		stats[string(node.Type)]++
	}

	if len(stats) > 0 {
		sb.WriteString("节点统计:\n")
		for nodeType, count := range stats {
			icon := getIcon(nodeType)
			sb.WriteString(fmt.Sprintf("  %s %-20s %d\n", icon, nodeType, count))
		}
		sb.WriteString("\n")
	}

	if verbose && len(result.Nodes) > 0 {
		sb.WriteString("节点详情:\n")
		sb.WriteString(strings.Repeat("-", 80) + "\n")
		for i, node := range result.Nodes {
			sb.WriteString(fmt.Sprintf("%d. [%s]", i+1, node.Type))
			if node.Name != "" {
				sb.WriteString(fmt.Sprintf(" %s", node.Name))
			}
			sb.WriteString(fmt.Sprintf(" (行 %d)\n", node.Start.Line))

			// 显示元数据
			if len(node.Metadata) > 0 {
				for key, value := range node.Metadata {
					sb.WriteString(fmt.Sprintf("     %s: %v\n", key, value))
				}
			}
		}
		sb.WriteString("\n")
	}

	if len(result.Errors) > 0 {
		sb.WriteString("错误:\n")
		for _, err := range result.Errors {
			sb.WriteString(fmt.Sprintf("  ❌ 行 %d: %s\n", err.Position.Line, err.Message))
		}
		sb.WriteString("\n")
	}

	if len(result.Warnings) > 0 {
		sb.WriteString("警告:\n")
		for _, warn := range result.Warnings {
			sb.WriteString(fmt.Sprintf("  ⚠️  行 %d: %s\n", warn.Position.Line, warn.Message))
		}
		sb.WriteString("\n")
	}

	fmt.Print(sb.String())
}

func getIcon(nodeType string) string {
	switch nodeType {
	case "PACKAGE":
		return "📦"
	case "PROCEDURE":
		return "⚙️ "
	case "FUNCTION":
		return "🔧"
	case "VARIABLE":
		return "📝"
	case "SQL_STATEMENT":
		return "💾"
	default:
		return "📄"
	}
}
