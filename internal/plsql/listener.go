package plsql

import (
	"strings"

	plsqlantlr "github.com/username/plsql-parser/internal/antlr"
	"github.com/username/plsql-parser/pkg/types"

	"github.com/antlr4-go/antlr/v4"
)

// 注意：实际包名是parser，但使用alias plsqlantlr保持代码清晰

// LineOffsetCache 行偏移量缓存 - 预计算每行的起始偏移，避免重复遍历
type LineOffsetCache struct {
	lines []int // 每行的起始偏移（不含换行符）
}

// NewLineOffsetCache 创建行偏移缓存
func NewLineOffsetCache(source string) *LineOffsetCache {
	lines := []int{0} // 第1行从偏移0开始
	for i := 0; i < len(source); i++ {
		if source[i] == '\n' {
			// 换行符属于当前行，下一行从 i+1 开始
			lines = append(lines, i+1)
		} else if source[i] == '\r' && (i+1 >= len(source) || source[i+1] != '\n') {
			// 单独的 \r（不是 \r\n），下一行从 i+1 开始
			lines = append(lines, i+1)
		}
	}
	return &LineOffsetCache{lines: lines}
}

// GetOffset 获取指定行列号对应的字符偏移（O(1) 查询）
// line: 行号（1-based）
// col: 列号（0-based）
// 返回: 字符偏移，如果位置无效返回 -1
func (c *LineOffsetCache) GetOffset(line, col int) int {
	if line < 1 || line > len(c.lines) {
		return -1
	}
	if col < 0 {
		return -1
	}

	lineIdx := line - 1
	offset := c.lines[lineIdx] + col

	// 验证 col 是否在该行范围内（简单检查：不超过下一行的开始）
	if line < len(c.lines) {
		lineLength := c.lines[line] - c.lines[lineIdx] - 1 // 减1排除换行符
		if col > lineLength {
			return -1
		}
	}

	return offset
}

// LineCount 返回总行数
func (c *LineOffsetCache) LineCount() int {
	return len(c.lines)
}

// PureANTLRCollector 纯ANTLR AST收集器 - 完全基于listener机制
type PureANTLRCollector struct {
	*plsqlantlr.BasePlSqlParserListener
	nodes []*types.Node
	source string       // 源代码，用于提取原始文本
	cache  *LineOffsetCache // 行偏移缓存
}

// NewPureANTLRCollector 创建纯ANTLR收集器
func NewPureANTLRCollector(source string) *PureANTLRCollector {
	return &PureANTLRCollector{
		nodes:  make([]*types.Node, 0),
		source: source,
		cache:  NewLineOffsetCache(source),
	}
}

// lineColToOffset 将行列号转换为字符偏移量（使用缓存优化）
func (c *PureANTLRCollector) lineColToOffset(line, col int) int {
	offset := c.cache.GetOffset(line, col)
	if offset < 0 {
		return len(c.source) // 无效位置返回末尾
	}
	return offset
}

// extractOriginalText 从源代码中提取原始文本（保留空格和格式）
func (c *PureANTLRCollector) extractOriginalText(startLine, startCol, endLine, endCol int) string {
	if c.source == "" || startLine < 1 || endLine < 1 {
		return ""
	}
	
	startIndex := c.lineColToOffset(startLine, startCol)
	endIndex := c.lineColToOffset(endLine, endCol)
	
	if startIndex >= endIndex || startIndex >= len(c.source) {
		return ""
	}
	
	if endIndex > len(c.source) {
		endIndex = len(c.source)
	}
	
	return c.source[startIndex:endIndex]
}

// createNode 创建节点（自动提取原始文本）
func (c *PureANTLRCollector) createNode(nodeType types.NodeType, name string, ctx interface{ GetStart() antlr.Token; GetStop() antlr.Token }) *types.Node {
	startToken := ctx.GetStart()
	stopToken := ctx.GetStop()
	
	// 空指针检查：防止解析失败或遇到 EOF 时 panic
	if startToken == nil || stopToken == nil {
		return nil
	}
	
	startLine := startToken.GetLine()
	startCol := startToken.GetColumn()
	endLine := stopToken.GetLine()
	endCol := stopToken.GetColumn() + len(stopToken.GetText())
	
	text := c.extractOriginalText(startLine, startCol, endLine, endCol)
	
	return &types.Node{
		Type:     nodeType,
		Name:     name,
		Text:     text,
		Start:    types.Position{Line: startLine, Column: startCol},
		End:      types.Position{Line: endLine, Column: endCol},
		Metadata: make(map[string]interface{}),
	}
}

// createSQLNode 创建 SQL 语句节点
func (c *PureANTLRCollector) createSQLNode(sqlType string, ctx interface{ GetStart() antlr.Token; GetStop() antlr.Token }) *types.Node {
	node := c.createNode(types.NodeTypeSQLStatement, "", ctx)
	if node == nil {
		return nil
	}
	
	node.Metadata["sqlType"] = sqlType
	return node
}

// EnterInsert_statement 进入INSERT语句节点
func (c *PureANTLRCollector) EnterInsert_statement(ctx *plsqlantlr.Insert_statementContext) {
	node := c.createSQLNode("INSERT", ctx)
	if node == nil {
		return
	}

	// 纯ANTLR AST遍历获取表名
	if singleTableInsert := ctx.Single_table_insert(); singleTableInsert != nil {
		if insertIntoClause := singleTableInsert.Insert_into_clause(); insertIntoClause != nil {
			if tableRef := insertIntoClause.General_table_ref(); tableRef != nil {
				if tableExpr := tableRef.Dml_table_expression_clause(); tableExpr != nil {
					tableName := c.extractTableNameFromContext(tableExpr)
					if tableName != "" {
						node.Metadata["table"] = tableName
					}
				}
			}
		}
	}

	c.nodes = append(c.nodes, node)
}

// EnterUpdate_statement 进入UPDATE语句节点
func (c *PureANTLRCollector) EnterUpdate_statement(ctx *plsqlantlr.Update_statementContext) {
	node := c.createSQLNode("UPDATE", ctx)
	if node == nil {
		return
	}

	// 纯ANTLR AST遍历获取表名
	if tableRef := ctx.General_table_ref(); tableRef != nil {
		if tableExpr := tableRef.Dml_table_expression_clause(); tableExpr != nil {
			tableName := c.extractTableNameFromContext(tableExpr)
			if tableName != "" {
				node.Metadata["table"] = tableName
			}
		}
	}

	c.nodes = append(c.nodes, node)
}

// EnterDelete_statement 进入DELETE语句节点
func (c *PureANTLRCollector) EnterDelete_statement(ctx *plsqlantlr.Delete_statementContext) {
	node := c.createSQLNode("DELETE", ctx)
	if node == nil {
		return
	}

	// 纯ANTLR AST遍历获取表名
	if tableRef := ctx.General_table_ref(); tableRef != nil {
		if tableExpr := tableRef.Dml_table_expression_clause(); tableExpr != nil {
			tableName := c.extractTableNameFromContext(tableExpr)
			if tableName != "" {
				node.Metadata["table"] = tableName
			}
		}
	}

	c.nodes = append(c.nodes, node)
}

// EnterSelect_statement 进入SELECT语句节点
func (c *PureANTLRCollector) EnterSelect_statement(ctx *plsqlantlr.Select_statementContext) {
	node := c.createSQLNode("SELECT", ctx)
	if node == nil {
		return
	}

	c.nodes = append(c.nodes, node)
}

// extractPackageSignature 提取包/包体的签名部分（只包含头部声明，不包含实现）
func (c *PureANTLRCollector) extractPackageSignature(startLine, startCol int, ctx antlr.ParserRuleContext) string {
	// 查找 IS 或 AS 关键字的位置，只提取到那里为止
	// 遍历子节点查找 IS/AS 关键字
	for i := 0; i < ctx.GetChildCount(); i++ {
		child := ctx.GetChild(i)
		if terminal, ok := child.(antlr.TerminalNode); ok {
			text := strings.ToUpper(terminal.GetText())
			if text == "IS" || text == "AS" {
				// 找到 IS/AS，提取到这个关键字结束的位置
				token := terminal.GetSymbol()
				endLine := token.GetLine()
				endCol := token.GetColumn() + len(terminal.GetText())
				return c.extractOriginalText(startLine, startCol, endLine, endCol)
			}
		}
	}
	
	// 如果没找到 IS/AS，返回空字符串
	return ""
}

// EnterCreate_package 进入包规范节点
func (c *PureANTLRCollector) EnterCreate_package(ctx *plsqlantlr.Create_packageContext) {
	// 空指针检查
	if ctx.GetStart() == nil || ctx.GetStop() == nil {
		return
	}
	
	if pkgName := ctx.Package_name(0); pkgName != nil {
		startLine := ctx.GetStart().GetLine()
		startCol := ctx.GetStart().GetColumn()
		endLine := ctx.GetStop().GetLine()
		endCol := ctx.GetStop().GetColumn() + len(ctx.GetStop().GetText())
		
		// 只提取签名部分，不包含内部实现
		signature := c.extractPackageSignature(startLine, startCol, ctx)
		if signature == "" {
			// 如果提取签名失败，使用完整文本作为降级方案
			signature = c.extractOriginalText(startLine, startCol, endLine, endCol)
		}
		
		node := &types.Node{
			Type:     types.NodeTypePackage,
			Name:     strings.ToUpper(pkgName.GetText()),
			Text:     signature,
			Start:    types.Position{Line: startLine, Column: startCol},
			End:      types.Position{Line: endLine, Column: endCol},
			Metadata: map[string]interface{}{"packageType": "spec"},
		}
		c.nodes = append(c.nodes, node)
	}
}

// EnterCreate_package_body 进入包体节点
func (c *PureANTLRCollector) EnterCreate_package_body(ctx *plsqlantlr.Create_package_bodyContext) {
	// 空指针检查
	if ctx.GetStart() == nil || ctx.GetStop() == nil {
		return
	}
	
	if pkgName := ctx.Package_name(0); pkgName != nil {
		startLine := ctx.GetStart().GetLine()
		startCol := ctx.GetStart().GetColumn()
		endLine := ctx.GetStop().GetLine()
		endCol := ctx.GetStop().GetColumn() + len(ctx.GetStop().GetText())
		
		// 只提取签名部分，不包含内部实现
		signature := c.extractPackageSignature(startLine, startCol, ctx)
		if signature == "" {
			// 如果提取签名失败，使用完整文本作为降级方案
			signature = c.extractOriginalText(startLine, startCol, endLine, endCol)
		}
		
		node := &types.Node{
			Type:     types.NodeTypePackage,
			Name:     strings.ToUpper(pkgName.GetText()),
			Text:     signature,
			Start:    types.Position{Line: startLine, Column: startCol},
			End:      types.Position{Line: endLine, Column: endCol},
			Metadata: map[string]interface{}{"packageType": "body"},
		}
		c.nodes = append(c.nodes, node)
	}
}

// EnterCreate_procedure_body 进入过程体节点
func (c *PureANTLRCollector) EnterCreate_procedure_body(ctx *plsqlantlr.Create_procedure_bodyContext) {
	// 空指针检查
	if ctx.GetStart() == nil || ctx.GetStop() == nil {
		return
	}
	
	if procName := ctx.Procedure_name(); procName != nil {
		startLine := ctx.GetStart().GetLine()
		startCol := ctx.GetStart().GetColumn()
		endLine := ctx.GetStop().GetLine()
		endCol := ctx.GetStop().GetColumn() + len(ctx.GetStop().GetText())
		
		node := &types.Node{
			Type:     types.NodeTypeProcedure,
			Name:     strings.ToUpper(procName.GetText()),
			Text:     c.extractOriginalText(startLine, startCol, endLine, endCol),
			Start:    types.Position{Line: startLine, Column: startCol},
			End:      types.Position{Line: endLine, Column: endCol},
			Metadata: map[string]interface{}{"objectType": "body"},
		}
		c.nodes = append(c.nodes, node)
	}
}

// EnterCreate_function_body 进入函数体节点
func (c *PureANTLRCollector) EnterCreate_function_body(ctx *plsqlantlr.Create_function_bodyContext) {
	// 空指针检查
	if ctx.GetStart() == nil || ctx.GetStop() == nil {
		return
	}
	
	if funcName := ctx.Function_name(); funcName != nil {
		startLine := ctx.GetStart().GetLine()
		startCol := ctx.GetStart().GetColumn()
		endLine := ctx.GetStop().GetLine()
		endCol := ctx.GetStop().GetColumn() + len(ctx.GetStop().GetText())
		
		node := &types.Node{
			Type:     types.NodeTypeFunction,
			Name:     strings.ToUpper(funcName.GetText()),
			Text:     c.extractOriginalText(startLine, startCol, endLine, endCol),
			Start:    types.Position{Line: startLine, Column: startCol},
			End:      types.Position{Line: endLine, Column: endCol},
			Metadata: map[string]interface{}{"objectType": "body", "scope": "standalone"},
		}
		c.nodes = append(c.nodes, node)
	}
}

// EnterProcedure_spec 进入过程规范节点（包括包内过程）
func (c *PureANTLRCollector) EnterProcedure_spec(ctx *plsqlantlr.Procedure_specContext) {
	// 空指针检查
	if ctx.GetStart() == nil || ctx.GetStop() == nil {
		return
	}
	
	if procName := ctx.Identifier(); procName != nil {
		startLine := ctx.GetStart().GetLine()
		startCol := ctx.GetStart().GetColumn()
		endLine := ctx.GetStop().GetLine()
		endCol := ctx.GetStop().GetColumn() + len(ctx.GetStop().GetText())
		
		node := &types.Node{
			Type:     types.NodeTypeProcedure,
			Name:     strings.ToUpper(procName.GetText()),
			Text:     c.extractOriginalText(startLine, startCol, endLine, endCol),
			Start:    types.Position{Line: startLine, Column: startCol},
			End:      types.Position{Line: endLine, Column: endCol},
			Metadata: map[string]interface{}{"objectType": "spec"},
		}
		c.nodes = append(c.nodes, node)
	}
}

// EnterFunction_spec 进入函数规范节点（包括包内函数）
func (c *PureANTLRCollector) EnterFunction_spec(ctx *plsqlantlr.Function_specContext) {
	// 空指针检查：防止解析失败或遇到 EOF 时 panic
	if ctx.GetStart() == nil || ctx.GetStop() == nil {
		return
	}
	
	if funcName := ctx.Identifier(); funcName != nil {
		startLine := ctx.GetStart().GetLine()
		startCol := ctx.GetStart().GetColumn()
		endLine := ctx.GetStop().GetLine()
		endCol := ctx.GetStop().GetColumn() + len(ctx.GetStop().GetText())
		
		node := &types.Node{
			Type:     types.NodeTypeFunction,
			Name:     strings.ToUpper(funcName.GetText()),
			Text:     c.extractOriginalText(startLine, startCol, endLine, endCol),
			Start:    types.Position{Line: startLine, Column: startCol},
			End:      types.Position{Line: endLine, Column: endCol},
			Metadata: map[string]interface{}{"objectType": "spec"},
		}
		c.nodes = append(c.nodes, node)
	}
}

// EnterProcedure_body 进入过程体节点（包括包内过程）
func (c *PureANTLRCollector) EnterProcedure_body(ctx *plsqlantlr.Procedure_bodyContext) {
	// 空指针检查：防止解析失败或遇到 EOF 时 panic
	if ctx.GetStart() == nil || ctx.GetStop() == nil {
		return
	}
	
	if procName := ctx.Identifier(); procName != nil {
		startLine := ctx.GetStart().GetLine()
		startCol := ctx.GetStart().GetColumn()
		endLine := ctx.GetStop().GetLine()
		endCol := ctx.GetStop().GetColumn() + len(ctx.GetStop().GetText())
		
		node := &types.Node{
			Type:     types.NodeTypeProcedure,
			Name:     strings.ToUpper(procName.GetText()),
			Text:     c.extractOriginalText(startLine, startCol, endLine, endCol),
			Start:    types.Position{Line: startLine, Column: startCol},
			End:      types.Position{Line: endLine, Column: endCol},
			Metadata: map[string]interface{}{"objectType": "body", "scope": "package"},
		}
		c.nodes = append(c.nodes, node)
	}
}

// EnterFunction_body 进入函数体节点（包括包内函数）
func (c *PureANTLRCollector) EnterFunction_body(ctx *plsqlantlr.Function_bodyContext) {
	// 空指针检查：防止解析失败或遇到 EOF 时 panic
	if ctx.GetStart() == nil || ctx.GetStop() == nil {
		return
	}
	
	if funcName := ctx.Identifier(); funcName != nil {
		startLine := ctx.GetStart().GetLine()
		startCol := ctx.GetStart().GetColumn()
		endLine := ctx.GetStop().GetLine()
		endCol := ctx.GetStop().GetColumn() + len(ctx.GetStop().GetText())
		
		node := &types.Node{
			Type:     types.NodeTypeFunction,
			Name:     strings.ToUpper(funcName.GetText()),
			Text:     c.extractOriginalText(startLine, startCol, endLine, endCol),
			Start:    types.Position{Line: startLine, Column: startCol},
			End:      types.Position{Line: endLine, Column: endCol},
			Metadata: map[string]interface{}{"objectType": "body", "scope": "package"},
		}
		c.nodes = append(c.nodes, node)
	}
}

// EnterVariable_declaration 进入变量声明节点
func (c *PureANTLRCollector) EnterVariable_declaration(ctx *plsqlantlr.Variable_declarationContext) {
	// 空指针检查
	if ctx.GetStart() == nil || ctx.GetStop() == nil {
		return
	}
	
	if varName := ctx.Identifier(); varName != nil {
		startLine := ctx.GetStart().GetLine()
		startCol := ctx.GetStart().GetColumn()
		endLine := ctx.GetStop().GetLine()
		endCol := ctx.GetStop().GetColumn() + len(ctx.GetStop().GetText())
		
		node := &types.Node{
			Type:     types.NodeTypeVariable,
			Name:     strings.ToUpper(varName.GetText()),
			Text:     c.extractOriginalText(startLine, startCol, endLine, endCol),
			Start:    types.Position{Line: startLine, Column: startCol},
			End:      types.Position{Line: endLine, Column: endCol},
			Metadata: map[string]interface{}{},
		}
		c.nodes = append(c.nodes, node)
	}
}

// extractTableNameFromContext 从上下文中提取表名 - 纯ANTLR AST遍历
func (c *PureANTLRCollector) extractTableNameFromContext(ctx plsqlantlr.IDml_table_expression_clauseContext) string {
	// 收集所有标识符（包括点和可能的schema前缀）
	var identifiers []string
	
	// 遍历所有终端节点收集非关键字标识符
	var collectIdentifiers func(antlr.Tree)
	collectIdentifiers = func(node antlr.Tree) {
		switch v := node.(type) {
		case antlr.TerminalNode:
			text := v.GetText()
			// 跳过SQL关键字、空白、括号等
			if !c.isSQLKeyword(text) && text != "" && text != "." && text != "(" && text != ")" {
				identifiers = append(identifiers, strings.ToUpper(text))
			}
		case antlr.ParserRuleContext:
			// 递归处理子节点
			for i := 0; i < v.GetChildCount(); i++ {
				collectIdentifiers(v.GetChild(i))
			}
		}
	}
	
	collectIdentifiers(ctx)
	
	// 如果有多个标识符，可能是 schema.table 格式
	// 尝试重建完整的表名（如 SCHEMA.TABLE）
	if len(identifiers) >= 2 {
		// 简单启发式：如果最后两个标识符可能是 schema.table
		// 或者如果有 "." 分隔符的痕迹
		// 这里我们返回最后一个标识符作为表名（最右边的部分）
		return identifiers[len(identifiers)-1]
	} else if len(identifiers) == 1 {
		return identifiers[0]
	}
	
	return ""
}

// isSQLKeyword 检查是否是SQL关键字
func (c *PureANTLRCollector) isSQLKeyword(text string) bool {
	upper := strings.ToUpper(text)
	keywords := map[string]bool{
		"INSERT": true, "INTO": true, "VALUES": true, "SELECT": true,
		"UPDATE": true, "SET": true, "DELETE": true, "FROM": true,
		"WHERE": true, "AND": true, "OR": true, "NOT": true,
		"NULL": true, "TRUE": true, "FALSE": true, "DUAL": true,
		"ONLY": true, "LEFT": true, "RIGHT": true, "INNER": true,
		"OUTER": true, "JOIN": true, "ON": true, "AS": true,
	}
	return keywords[upper]
}

// GetNodes 获取收集的节点
func (c *PureANTLRCollector) GetNodes() []*types.Node {
	return c.nodes
}
