package optimizer

import (
	"github.com/username/plsql-parser/pkg/types"
	"strings"
	"testing"
)

func TestOptimizeClean(t *testing.T) {
	result := createTestResult()
	options := GetOptions(LevelClean)
	optimized := Optimize(result, options)

	// L1 应该剔除变量节点
	for _, node := range optimized.Nodes {
		if node.Type == types.NodeTypeVariable {
			t.Error("Clean mode should remove variable nodes")
		}
	}
	// Source 字段不应该存在
	if optimized.Source != "" {
		t.Error("Source field should be empty")
	}
}

func TestOptimizeLean(t *testing.T) {
	result := createTestResultWithSQL()
	options := GetOptions(LevelLean)
	optimized := Optimize(result, options)

	// L2 应该折叠长 SQL
	found := false
	for _, node := range optimized.Nodes {
		if node.Type == types.NodeTypeSQLStatement && strings.Contains(node.Text, "[Fields List Collapsed]") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Lean mode should fold long SQL statements")
	}
}

func TestOptimizeFold(t *testing.T) {
	// L3 已移除，此测试不再需要
	t.Skip("LevelFold has been removed")
}

// TestCleanComments 测试注释清理功能
func TestCleanComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "单行注释被移除",
			input:    "SELECT * FROM t -- 这是一条注释\nWHERE id = 1",
			expected: "SELECT * FROM t\nWHERE id = 1",
		},
		{
			name:     "多行注释被移除且不留下多余空格",
			input:    "SELECT * FROM t /* 这是\n多行\n注释 */ WHERE id = 1",
			expected: "SELECT * FROM t WHERE id = 1",
		},
		{
			name:     "字符串字面量中的单引号被保留",
			input:    "SELECT * FROM t WHERE name = 'O''Brien'",
			expected: "SELECT * FROM t WHERE name = 'O''Brien'",
		},
		{
			name:     "字符串字面量中的内容不被误删",
			input:    "SELECT * FROM t -- 测试注释\nWHERE status = 'N'\nAND type = 'O'",
			expected: "SELECT * FROM t\nWHERE status = 'N'\nAND type = 'O'",
		},
		{
			name:     "多行注释内的SQL不被保留",
			input:    "SELECT * FROM t\n/* AND EXISTS (SELECT 1 FROM other) */\nWHERE id = 1",
			expected: "SELECT * FROM t\nWHERE id = 1",
		},
		// 验证用户报告的问题：注释中的 'N' 不应该被当作字符串保留
		{
			name:     "注释中的N不会被保留",
			input:    "And h.Return_Status = 'N'\n         --And h.Return_Status = 'N'",
			expected: "And h.Return_Status = 'N'",
		},
		{
			name:     "字符串字面量应该被保留而不是注释",
			input:    "WHERE type = 'O' -- O是字符串",
			expected: "WHERE type = 'O'",
		},
		{
			name:     "嵌套的多行注释被正确移除",
			input:    "SELECT * FROM t /* 外层 /* 嵌套注释 */ 继续外层 */ WHERE id = 1",
			expected: "SELECT * FROM t WHERE id = 1",
		},
		{
			name:     "多行注释内嵌套单行注释被正确移除",
			input:    "SELECT * FROM t /* 外层\n  -- 这是单行注释\n  继续外层 */ WHERE id = 1",
			expected: "SELECT * FROM t WHERE id = 1",
		},
		{
			name:     "复杂嵌套组合",
			input:    "SELECT * FROM t /* 外层 /* 嵌套1 /* 最内层 */ */ 继续 */ WHERE id = 1",
			expected: "SELECT * FROM t WHERE id = 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &types.ParseResult{
				Root:      &types.Node{Type: types.NodeTypePackage, Name: "ROOT", Metadata: make(map[string]interface{})},
				Nodes:     []*types.Node{},
				Errors:    []types.Error{},
				Warnings:  []types.Error{},
				LineCount: 1,
			}

			// 计算正确的行数
			lineCount := 1 + strings.Count(tt.input, "\n")

			node := &types.Node{
				Type:     types.NodeTypeSQLStatement,
				Text:     tt.input,
				Start:    types.Position{Line: 1, Column: 0, Offset: 0},
				End:      types.Position{Line: lineCount, Column: len(tt.input), Offset: len(tt.input)},
				Metadata: map[string]interface{}{"sqlType": "SELECT"},
			}
			result.Nodes = append(result.Nodes, node)

			options := GetOptions(LevelLean)
			optimized := Optimize(result, options)

			if len(optimized.Nodes) != 1 {
				t.Fatalf("expected 1 node, got %d", len(optimized.Nodes))
			}

			actual := optimized.Nodes[0].Text
			if actual != tt.expected {
				t.Errorf("cleanComments 失败:\n  输入:    %q\n  期望:    %q\n  实际:    %q", tt.input, tt.expected, actual)
			}
		})
	}
}

// createTestResultWithSQL 创建包含 SQL 的测试结果用于 L2 测试
func createTestResultWithSQL() *types.ParseResult {
	return &types.ParseResult{
		Root: &types.Node{
			Type:     types.NodeTypePackage,
			Name:     "ROOT",
			Metadata: make(map[string]interface{}),
		},
		Nodes: []*types.Node{
			{
				Type: types.NodeTypeSQLStatement,
				Text: "SELECT col1, col2, col3, col4, col5, col6 FROM test_table WHERE id = 1",
				Start: types.Position{Line: 1, Column: 0, Offset: 0},
				End:   types.Position{Line: 1, Column: 70, Offset: 70},
				Metadata: map[string]interface{}{"sqlType": "SELECT"},
			},
		},
		Errors:    []types.Error{},
		Warnings:  []types.Error{},
		LineCount: 1,
	}
}

// createTestResult 创建测试结果用于测试
func createTestResult() *types.ParseResult {
	return &types.ParseResult{
		Root: &types.Node{
			Type:     types.NodeTypePackage,
			Name:     "ROOT",
			Metadata: make(map[string]interface{}),
		},
		Nodes: []*types.Node{
			{
				Type:     types.NodeTypePackage,
				Name:     "TEST_PKG",
				Text:     "Create Or Replace Package TEST_PKG Is",
				Start:    types.Position{Line: 1, Column: 0, Offset: 0},
				End:      types.Position{Line: 10, Column: 25, Offset: 100},
				Metadata: map[string]interface{}{"packageType": "spec"},
			},
			{
				Type:     types.NodeTypeVariable,
				Name:     "G_VAR",
				Text:     "g_var Number;",
				Start:    types.Position{Line: 3, Column: 2, Offset: 50},
				End:      types.Position{Line: 3, Column: 15, Offset: 63},
				Metadata: map[string]interface{}{}, // 空metadata
			},
			{
				Type:     types.NodeTypeFunction,
				Name:     "TEST_FUNC",
				Text:     "Function Test_Func Return Number;",
				Start:    types.Position{Line: 5, Column: 2, Offset: 70},
				End:      types.Position{Line: 5, Column: 35, Offset: 103},
				Metadata: map[string]interface{}{"objectType": "spec"},
			},
		},
		Errors:    []types.Error{},
		Warnings:  []types.Error{},
		Source:    "Create Or Replace Package TEST_PKG Is\n...\nEnd TEST_PKG;",
		LineCount: 10,
	}
}
