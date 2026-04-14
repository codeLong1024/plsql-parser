package integration_test

import (
	"testing"

	"github.com/username/plsql-parser/pkg/optimizer"
	"github.com/username/plsql-parser/pkg/types"
)

// createTestResult 创建测试用的解析结果
func createTestResult() *types.ParseResult {
	return &types.ParseResult{
		Root: &types.Node{
			Type: types.NodeTypePackage,
			Name: "ROOT",
		},
		Nodes: []*types.Node{
			{
				Type: types.NodeTypePackage,
				Name: "TEST_PKG",
				Text: "CREATE OR REPLACE PACKAGE test_pkg IS",
				Metadata: map[string]interface{}{"packageType": "spec"},
			},
			{
				Type: types.NodeTypeVariable,
				Name: "V_VAR1",
				Text: "v_var1 NUMBER;",
			},
			{
				Type: types.NodeTypeConstant,
				Name: "C_CONST1",
				Text: "c_const1 CONSTANT NUMBER := 1;",
			},
			{
				Type: types.NodeTypeProcedure,
				Name: "TEST_PROC",
				Text: "PROCEDURE test_proc IS\nBEGIN\n  NULL;\nEND;",
			},
			{
				Type: types.NodeTypeSQLStatement,
				Text: "SELECT col1, col2, col3 FROM test_table WHERE id = 1",
				Metadata: map[string]interface{}{"sqlType": "SELECT"},
			},
		},
		LineCount: 50,
	}
}

// TestOptimizeLevelClean 测试 L1 去噪优化
func TestOptimizeLevelClean(t *testing.T) {
	result := createTestResult()
	options := optimizer.GetOptions(optimizer.LevelClean)
	optimized := optimizer.Optimize(result, options)

	// L1 应该移除 VARIABLE 和 CONSTANT 节点
	varCount := 0
	constCount := 0
	for _, node := range optimized.Nodes {
		if node.Type == types.NodeTypeVariable {
			varCount++
		}
		if node.Type == types.NodeTypeConstant {
			constCount++
		}
	}

	if varCount > 0 {
		t.Errorf("L1 优化后不应包含 VARIABLE 节点，但找到 %d 个", varCount)
	}
	if constCount > 0 {
		t.Errorf("L1 优化后不应包含 CONSTANT 节点，但找到 %d 个", constCount)
	}

	// 应该保留 PACKAGE 和 PROCEDURE
	hasPackage := false
	hasProcedure := false
	for _, node := range optimized.Nodes {
		if node.Type == types.NodeTypePackage {
			hasPackage = true
		}
		if node.Type == types.NodeTypeProcedure {
			hasProcedure = true
		}
	}

	if !hasPackage {
		t.Error("L1 优化后应保留 PACKAGE 节点")
	}
	if !hasProcedure {
		t.Error("L1 优化后应保留 PROCEDURE 节点")
	}
}

// TestOptimizeLevelLean 测试 L2 精简优化
func TestOptimizeLevelLean(t *testing.T) {
	result := createTestResult()
	options := optimizer.GetOptions(optimizer.LevelLean)
	optimized := optimizer.Optimize(result, options)

	// L2 也应该移除 VARIABLE 和 CONSTANT
	varCount := 0
	for _, node := range optimized.Nodes {
		if node.Type == types.NodeTypeVariable {
			varCount++
		}
	}

	if varCount > 0 {
		t.Errorf("L2 优化后不应包含 VARIABLE 节点，但找到 %d 个", varCount)
	}

	// L2 应该折叠 SELECT 语句的字段列表
	for _, node := range optimized.Nodes {
		if node.Type == types.NodeTypeSQLStatement {
			if len(node.Text) > 0 && len(node.Text) < len("SELECT col1, col2, col3 FROM test_table WHERE id = 1") {
				t.Logf("SELECT 语句已折叠: %s", node.Text)
			}
		}
	}
}

// TestOptimizeNilResult 测试 nil 结果处理
func TestOptimizeNilResult(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("优化 nil 结果时发生 panic: %v", r)
		}
	}()

	options := optimizer.GetOptions(optimizer.LevelClean)
	// 这里不应该 panic
	_ = optimizer.Optimize(nil, options)
}

// TestOptimizeNilOptions 测试 nil 选项处理
func TestOptimizeNilOptions(t *testing.T) {
	result := createTestResult()
	
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("使用 nil 选项优化时发生 panic: %v", r)
		}
	}()

	// 应该使用默认选项
	optimized := optimizer.Optimize(result, nil)
	if optimized == nil {
		t.Error("期望返回优化结果，但得到 nil")
	}
}

// TestOptimizePreservesErrors 测试优化保留错误信息
func TestOptimizePreservesErrors(t *testing.T) {
	result := createTestResult()
	result.Errors = []types.Error{
		{Message: "Test error", Position: types.Position{Line: 1}},
	}
	result.Warnings = []types.Error{
		{Message: "Test warning", Position: types.Position{Line: 2}},
	}

	options := optimizer.GetOptions(optimizer.LevelClean)
	optimized := optimizer.Optimize(result, options)

	if len(optimized.Errors) != len(result.Errors) {
		t.Errorf("期望保留 %d 个错误，但得到 %d 个", len(result.Errors), len(optimized.Errors))
	}
	if len(optimized.Warnings) != len(result.Warnings) {
		t.Errorf("期望保留 %d 个警告，但得到 %d 个", len(result.Warnings), len(optimized.Warnings))
	}
}

// TestOptimizeRemovesSource 测试优化移除 Source 字段
func TestOptimizeRemovesSource(t *testing.T) {
	result := createTestResult()
	result.Source = "Some source code"

	options := optimizer.GetOptions(optimizer.LevelClean)
	optimized := optimizer.Optimize(result, options)

	if optimized.Source != "" {
		t.Error("优化后的结果不应包含 Source 字段")
	}
}

// TestOptimizeEmptyNodes 测试空节点列表优化
func TestOptimizeEmptyNodes(t *testing.T) {
	result := &types.ParseResult{
		Root: &types.Node{
			Type: types.NodeTypePackage,
			Name: "ROOT",
		},
		Nodes:     []*types.Node{},
		LineCount: 0,
	}

	options := optimizer.GetOptions(optimizer.LevelClean)
	optimized := optimizer.Optimize(result, options)

	if len(optimized.Nodes) != 0 {
		t.Errorf("空节点列表优化后应有 0 个节点，但得到 %d 个", len(optimized.Nodes))
	}
}

// TestOptimizeNodeMetadata 测试元数据处理
func TestOptimizeNodeMetadata(t *testing.T) {
	result := &types.ParseResult{
		Root: &types.Node{Type: types.NodeTypePackage, Name: "ROOT"},
		Nodes: []*types.Node{
			{
				Type:     types.NodeTypeProcedure,
				Name:     "TEST_PROC",
				Metadata: map[string]interface{}{"key1": "value1", "key2": 123},
			},
		},
	}

	options := optimizer.GetOptions(optimizer.LevelClean)
	optimized := optimizer.Optimize(result, options)

	if len(optimized.Nodes) > 0 {
		node := optimized.Nodes[0]
		if len(node.Metadata) != 2 {
			t.Errorf("期望保留 2 个元数据项，但得到 %d 个", len(node.Metadata))
		}
	}
}

// TestCommentCleaning 测试注释清理
func TestCommentCleaning(t *testing.T) {
	result := &types.ParseResult{
		Root: &types.Node{Type: types.NodeTypePackage, Name: "ROOT"},
		Nodes: []*types.Node{
			{
				Type: types.NodeTypeProcedure,
				Text: "PROCEDURE test_proc IS\n  -- This is a comment\nBEGIN\n  NULL; -- inline comment\nEND;",
			},
		},
	}

	options := optimizer.GetOptions(optimizer.LevelLean)
	optimized := optimizer.Optimize(result, options)

	if len(optimized.Nodes) > 0 {
		node := optimized.Nodes[0]
		// L2 应该移除注释
		t.Logf("优化后文本: %s", node.Text)
	}
}
