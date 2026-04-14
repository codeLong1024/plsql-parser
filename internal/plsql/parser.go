package plsql

import (
	"fmt"
	"strings"

	plsqlantlr "github.com/username/plsql-parser/internal/antlr"
	"github.com/username/plsql-parser/pkg/types"

	"github.com/antlr4-go/antlr/v4"
)

// ParseOptions 解析选项
type ParseOptions struct {
	Strict bool // 严格模式(遇到错误立即返回)
}

// parseError 解析错误
type parseError struct {
	line   int
	column int
	msg    string
}

// ErrorListener 错误监听器
type ErrorListener struct {
	errors []*parseError
}

// NewErrorListener 创建错误监听器
func NewErrorListener() *ErrorListener {
	return &ErrorListener{
		errors: make([]*parseError, 0),
	}
}

// SyntaxError 语法错误处理
func (l *ErrorListener) SyntaxError(recognizer antlr.Recognizer, offendingSymbol interface{}, line, column int, msg string, e antlr.RecognitionException) {
	l.errors = append(l.errors, &parseError{
		line:   line,
		column: column,
		msg:    msg,
	})
}

// ReportAmbiguity 报告歧义
func (l *ErrorListener) ReportAmbiguity(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex int, exact bool, ambigAlts *antlr.BitSet, configs *antlr.ATNConfigSet) {
	// 简化实现:忽略歧义报告
}

// ReportAttemptingFullContext 报告尝试全上下文
func (l *ErrorListener) ReportAttemptingFullContext(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex int, conflictingAlts *antlr.BitSet, configs *antlr.ATNConfigSet) {
	// 简化实现:忽略
}

// ReportContextSensitivity 报告上下文敏感性
func (l *ErrorListener) ReportContextSensitivity(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex, prediction int, configs *antlr.ATNConfigSet) {
	// 简化实现:忽略
}

// ANTLRRealParser 真正使用 ANTLR 的解析器
type ANTLRRealParser struct {
	options *ParseOptions
}

// NewANTLRRealParser 创建 ANTLR 真实解析器
func NewANTLRRealParser(options *ParseOptions) *ANTLRRealParser {
	if options == nil {
		options = &ParseOptions{
			Strict: false,
		}
	}
	return &ANTLRRealParser{options: options}
}



// Parse 使用 ANTLR 解析 PLSQL 代码
func (p *ANTLRRealParser) Parse(code string) (*types.ParseResult, error) {
	if strings.TrimSpace(code) == "" {
		return nil, fmt.Errorf("代码为空")
	}

	// 创建输入流
	input := antlr.NewInputStream(code)

	// 创建词法分析器
	lexer := plsqlantlr.NewPlSqlLexer(input)
	lexer.RemoveErrorListeners()

	// 添加错误监听器
	lexerErrors := NewErrorListener()
	lexer.AddErrorListener(lexerErrors)

	// 创建 token 流
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	// 创建语法分析器
	parser := plsqlantlr.NewPlSqlParser(tokens)
	parser.RemoveErrorListeners()

	// 添加错误监听器
	parserErrors := NewErrorListener()
	parser.AddErrorListener(parserErrors)

	// 解析 (使用 sql_script 规则作为入口)
	tree := parser.Sql_script()

	// 创建结果
	result := &types.ParseResult{
		Root: &types.Node{
			Type:     types.NodeTypePackage,
			Name:     "ROOT",
			Children: make([]*types.Node, 0),
			Metadata: make(map[string]interface{}),
		},
		Nodes:     make([]*types.Node, 0),
		Errors:    make([]types.Error, 0),
		Warnings:  make([]types.Error, 0),
		Source:    code,
		LineCount: len(strings.Split(code, "\n")),
	}
	


	// 收集错误
	for _, err := range lexerErrors.errors {
		result.Errors = append(result.Errors, types.Error{
			Message:  err.msg,
			Position: types.Position{Line: err.line, Column: err.column},
			Severity: "error",
		})
	}

	for _, err := range parserErrors.errors {
		result.Errors = append(result.Errors, types.Error{
			Message:  err.msg,
			Position: types.Position{Line: err.line, Column: err.column},
			Severity: "error",
		})
	}

	// 如果有错误且是严格模式，返回错误
	if len(result.Errors) > 0 && p.options.Strict {
		return result, fmt.Errorf("解析发现 %d 个错误", len(result.Errors))
	}

	// 遍历 AST 收集信息 - 纯ANTLR实现
	collector := NewPureANTLRCollector(code)
	antlr.ParseTreeWalkerDefault.Walk(collector, tree)

	// 将收集的节点添加到结果
	result.Nodes = collector.GetNodes()

	return result, nil
}
