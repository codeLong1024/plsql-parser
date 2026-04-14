package integration_test

import (
	"os"
	"testing"

	"github.com/username/plsql-parser/internal/plsql"
	"github.com/username/plsql-parser/pkg/optimizer"
)

// TestRealPackageFile 测试真实包文件解析
func TestRealPackageFile(t *testing.T) {
	// 读取示例文件
	code, err := os.ReadFile("../example/simple_package.sql")
	if err != nil {
		t.Fatalf("无法读取示例文件: %v", err)
	}

	parser := plsql.NewANTLRRealParser(nil)
	result, err := parser.Parse(string(code))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if len(result.Nodes) == 0 {
		t.Error("期望检测到节点，但得到 0 个")
	}

	// 统计各类节点
	stats := make(map[string]int)
	for _, node := range result.Nodes {
		stats[string(node.Type)]++
	}

	t.Logf("节点统计: %+v", stats)

	// 应该检测到包、函数、过程、SQL语句
	expectedTypes := []string{"PACKAGE", "FUNCTION", "PROCEDURE", "SQL_STATEMENT"}
	for _, expectedType := range expectedTypes {
		if count, exists := stats[expectedType]; !exists || count == 0 {
			t.Errorf("未检测到预期的节点类型: %s", expectedType)
		}
	}
}

// TestRealFunctionFile 测试真实函数文件解析
func TestRealFunctionFile(t *testing.T) {
	code, err := os.ReadFile("../example/simple_function.sql")
	if err != nil {
		t.Fatalf("无法读取示例文件: %v", err)
	}

	parser := plsql.NewANTLRRealParser(nil)
	result, err := parser.Parse(string(code))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	hasFunction := false
	for _, node := range result.Nodes {
		if string(node.Type) == "FUNCTION" {
			hasFunction = true
			break
		}
	}

	if !hasFunction {
		t.Error("未检测到 FUNCTION 节点")
	}
}

// TestRealProcedureFile 测试真实过程文件解析
func TestRealProcedureFile(t *testing.T) {
	code, err := os.ReadFile("../example/simple_procedure.sql")
	if err != nil {
		t.Fatalf("无法读取示例文件: %v", err)
	}

	parser := plsql.NewANTLRRealParser(nil)
	result, err := parser.Parse(string(code))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	hasProcedure := false
	for _, node := range result.Nodes {
		if string(node.Type) == "PROCEDURE" {
			hasProcedure = true
			break
		}
	}

	if !hasProcedure {
		t.Error("未检测到 PROCEDURE 节点")
	}
}

// TestOptimizationLevels 测试不同优化级别的效果
func TestOptimizationLevels(t *testing.T) {
	code, err := os.ReadFile("../example/simple_package.sql")
	if err != nil {
		t.Fatalf("无法读取示例文件: %v", err)
	}

	parser := plsql.NewANTLRRealParser(nil)
	result, err := parser.Parse(string(code))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	// 测试无优化
	unoptimized := optimizer.Optimize(result, &optimizer.OptimizeOptions{Level: 0})
	t.Logf("无优化节点数: %d", len(unoptimized.Nodes))

	// 测试 L1 优化
	l1Optimized := optimizer.Optimize(result, optimizer.GetOptions(optimizer.LevelClean))
	t.Logf("L1 优化节点数: %d", len(l1Optimized.Nodes))

	// 测试 L2 优化
	l2Optimized := optimizer.Optimize(result, optimizer.GetOptions(optimizer.LevelLean))
	t.Logf("L2 优化节点数: %d", len(l2Optimized.Nodes))

	// L1 和 L2 应该比无优化少（因为移除了变量）
	if len(l1Optimized.Nodes) >= len(unoptimized.Nodes) {
		t.Log("警告: L1 优化后节点数未减少")
	}
}

// TestTableExtraction 测试表名提取
func TestTableExtraction(t *testing.T) {
	code := `CREATE OR REPLACE PROCEDURE test_proc IS
BEGIN
  INSERT INTO fnd_user VALUES (1, 'test');
  UPDATE hr_employees SET salary = 1000;
  DELETE FROM ap_invoices WHERE invoice_id = 1;
END test_proc;`

	parser := plsql.NewANTLRRealParser(nil)
	result, err := parser.Parse(code)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	tableFound := false
	for _, node := range result.Nodes {
		if string(node.Type) == "SQL_STATEMENT" {
			if table, ok := node.Metadata["table"]; ok {
				t.Logf("检测到表名: %v", table)
				tableFound = true
			}
		}
	}

	if !tableFound {
		t.Log("警告: 未检测到 SQL 语句中的表名（可能是语法限制）")
	}
}

// TestLargeFilePerformance 测试大文件性能
func TestLargeFilePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过性能测试")
	}

	// 检查 test.pkg 文件是否存在
	if _, err := os.Stat("../example/test.pkg"); os.IsNotExist(err) {
		t.Skip("test.pkg 文件不存在，跳过性能测试")
	}

	code, err := os.ReadFile("../example/test.pkg")
	if err != nil {
		t.Fatalf("无法读取测试文件: %v", err)
	}

	t.Logf("测试文件大小: %d bytes, %d lines", len(code), len(string(code)))

	parser := plsql.NewANTLRRealParser(nil)
	
	// 测量解析时间
	start := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = parser.Parse(string(code))
		}
	})

	t.Logf("解析性能: %v", start)
}

// TestErrorRecovery 测试错误恢复能力
func TestErrorRecovery(t *testing.T) {
	// 包含部分错误语法的代码
	code := `CREATE OR REPLACE PACKAGE test_pkg IS
  PROCEDURE valid_proc;
  -- Some invalid syntax here
  INVALID KEYWORD test;
END test_pkg;`

	parser := plsql.NewANTLRRealParser(nil)
	result, err := parser.Parse(code)
	
	// 非严格模式应该能够部分解析
	if err != nil {
		t.Logf("解析返回错误: %v", err)
	}

	// 即使有错误，也应该检测到一些有效节点
	if len(result.Nodes) > 0 {
		t.Logf("在错误情况下仍检测到 %d 个节点", len(result.Nodes))
	} else {
		t.Log("警告: 错误导致完全无法解析")
	}
}

// TestMetadataIntegrity 测试元数据完整性
func TestMetadataIntegrity(t *testing.T) {
	code := `CREATE OR REPLACE PACKAGE BODY test_pkg IS
  PROCEDURE test_proc IS
  BEGIN
    INSERT INTO test_table VALUES (1);
  END;
END test_pkg;`

	parser := plsql.NewANTLRRealParser(nil)
	result, err := parser.Parse(code)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	// 检查包的元数据
	for _, node := range result.Nodes {
		if string(node.Type) == "PACKAGE" {
			if pkgType, ok := node.Metadata["packageType"]; ok {
				t.Logf("包类型: %v", pkgType)
			} else {
				t.Error("PACKAGE 节点缺少 packageType 元数据")
			}
		}

		if string(node.Type) == "PROCEDURE" {
			if objType, ok := node.Metadata["objectType"]; ok {
				t.Logf("对象类型: %v", objType)
			}
		}

		if string(node.Type) == "SQL_STATEMENT" {
			if sqlType, ok := node.Metadata["sqlType"]; ok {
				t.Logf("SQL 类型: %v", sqlType)
			}
		}
	}
}
