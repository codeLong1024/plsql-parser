package types

import "fmt"

// NodeType AST 节点类型
type NodeType string

const (
	NodeTypePackage      NodeType = "PACKAGE"
	NodeTypePackageBody  NodeType = "PACKAGE_BODY"
	NodeTypeProcedure    NodeType = "PROCEDURE"
	NodeTypeFunction     NodeType = "FUNCTION"
	NodeTypeVariable     NodeType = "VARIABLE"
	NodeTypeConstant     NodeType = "CONSTANT"
	NodeTypeCursor       NodeType = "CURSOR"
	NodeTypeType         NodeType = "TYPE"
	NodeTypeException    NodeType = "EXCEPTION"
	NodeTypeIfStatement  NodeType = "IF_STATEMENT"
	NodeTypeLoopStatement NodeType = "LOOP_STATEMENT"
	NodeTypeSQLStatement NodeType = "SQL_STATEMENT"
	NodeTypeAssignment   NodeType = "ASSIGNMENT"
	NodeTypeCall         NodeType = "CALL"
	NodeTypeReturn       NodeType = "RETURN"
	NodeTypeRaise        NodeType = "RAISE"
	NodeTypeComment      NodeType = "COMMENT"
	NodeTypeUnknown      NodeType = "UNKNOWN"
)

// Position 源代码位置
type Position struct {
	Line   int `json:"line,omitempty"`   // 行号
	Column int `json:"column,omitempty"` // 列号
	Offset int `json:"offset,omitempty"` // 字符偏移
}

// String 格式化位置信息
func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

// Node AST 节点
type Node struct {
	Type     NodeType   `json:"type"`              // 节点类型
	Name     string     `json:"name,omitempty"`    // 节点名称(如有)
	Text     string     `json:"text,omitempty"`    // 源代码文本
	Start    Position   `json:"start"`             // 起始位置
	End      Position   `json:"end"`               // 结束位置
	Children []*Node    `json:"children,omitempty"` // 子节点
	Parent   *Node      `json:"-"`                 // 父节点(不序列化)
	Metadata Metadata   `json:"metadata,omitempty"` // 元数据
}

// Metadata 节点元数据
type Metadata map[string]interface{}

// ParseResult 解析结果
type ParseResult struct {
	Root      *Node   `json:"root"`               // 根节点
	Nodes     []*Node `json:"nodes"`              // 所有节点(扁平化)
	Errors    []Error `json:"errors"`             // 解析错误
	Warnings  []Error `json:"warnings"`           // 解析警告
	Source    string  `json:"source,omitempty"`   // 原始代码
	LineCount int     `json:"lineCount"`          // 总行数
}

// Error 解析错误
type Error struct {
	Message  string   `json:"message"`  // 错误消息
	Position Position `json:"position"` // 错误位置
	Severity string   `json:"severity"` // 错误级别: error, warning
	Code     string   `json:"code"`     // 错误代码
}

// Issue 审计问题
type Issue struct {
	RuleID      string   `json:"ruleId"`      // 规则 ID
	RuleName    string   `json:"ruleName"`    // 规则名称
	Severity    string   `json:"severity"`    // 严重性: error, warning, info
	Message     string   `json:"message"`     // 问题描述
	Position    Position `json:"position"`    // 代码位置
	Suggestion  string   `json:"suggestion"`  // 修复建议
	Category    string   `json:"category"`    // 问题分类
	Reference   string   `json:"reference"`   // 参考文档
}

// AuditResult 审计结果
type AuditResult struct {
	File        string   `json:"file"`        // 文件路径
	Issues      []Issue  `json:"issues"`      // 问题列表
	Score       int      `json:"score"`       // 代码质量分数 (0-100)
	Summary     Summary  `json:"summary"`     // 汇总信息
}

// Summary 审计汇总
type Summary struct {
	TotalIssues   int `json:"totalIssues"`   // 总问题数
	ErrorCount    int `json:"errorCount"`    // 错误数
	WarningCount  int `json:"warningCount"`  // 警告数
	InfoCount     int `json:"infoCount"`     // 提示数
	LineCount     int `json:"lineCount"`     // 代码行数
	AnalyzedNodes int `json:"analyzedNodes"` // 分析节点数
}

// SQLStatementInfo SQL 语句信息
type SQLStatementInfo struct {
	Type        string   `json:"type"`        // SELECT, INSERT, UPDATE, DELETE, MERGE
	Tables      []string `json:"tables"`      // 涉及的表
	Columns     []string `json:"columns"`     // 涉及的列
	WhereClause string   `json:"whereClause"` // WHERE 子句
	IsDynamic   bool     `json:"isDynamic"`   // 是否动态 SQL
	Position    Position `json:"position"`    // 位置
}

// ProcedureInfo 存储过程信息
type ProcedureInfo struct {
	Name       string     `json:"name"`       // 过程名
	Schema     string     `json:"schema"`     // 模式名
	Parameters []ParamInfo `json:"parameters"` // 参数列表
	Position   Position   `json:"position"`   // 位置
}

// FunctionInfo 函数信息
type FunctionInfo struct {
	Name       string      `json:"name"`       // 函数名
	Schema     string      `json:"schema"`     // 模式名
	ReturnType string      `json:"returnType"` // 返回类型
	Parameters []ParamInfo `json:"parameters"` // 参数列表
	Position   Position    `json:"position"`   // 位置
}

// ParamInfo 参数信息
type ParamInfo struct {
	Name      string `json:"name"`      // 参数名
	Type      string `json:"type"`      // 数据类型
	Direction string `json:"direction"` // IN, OUT, IN OUT
	Default   string `json:"default"`   // 默认值
}

// PackageInfo 包信息
type PackageInfo struct {
	Name       string          `json:"name"`       // 包名
	Schema     string          `json:"schema"`     // 模式名
	Procedures []ProcedureInfo `json:"procedures"` // 存储过程
	Functions  []FunctionInfo  `json:"functions"`  // 函数
	Variables  []VariableInfo  `json:"variables"`  // 变量
	Types      []TypeInfo      `json:"types"`      // 类型
	Position   Position        `json:"position"`   // 位置
}

// VariableInfo 变量信息
type VariableInfo struct {
	Name      string `json:"name"`      // 变量名
	Type      string `json:"type"`      // 数据类型
	Default   string `json:"default"`   // 默认值
	IsConstant bool  `json:"isConstant"` // 是否常量
	Position  Position `json:"position"` // 位置
}

// TypeInfo 类型信息
type TypeInfo struct {
	Name       string   `json:"name"`       // 类型名
	BaseType   string   `json:"baseType"`   // 基础类型
	IsRecord   bool     `json:"isRecord"`   // 是否记录类型
	IsTable    bool     `json:"isTable"`    // 是否表类型
	IsRefCursor bool    `json:"isRefCursor"` // 是否引用游标
	Fields     []FieldInfo `json:"fields"`  // 字段列表(记录类型)
	Position   Position `json:"position"`   // 位置
}

// FieldInfo 字段信息
type FieldInfo struct {
	Name     string `json:"name"`     // 字段名
	Type     string `json:"type"`     // 数据类型
	Position Position `json:"position"` // 位置
}
