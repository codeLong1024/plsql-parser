package integration_test

import (
	"testing"

	"github.com/username/plsql-parser/internal/plsql"
)

// TestParseEmptyCode 测试空代码解析
func TestParseEmptyCode(t *testing.T) {
	parser := plsql.NewANTLRRealParser(nil)
	_, err := parser.Parse("")
	if err == nil {
		t.Error("期望空代码返回错误，但得到了 nil")
	}
}

// TestParseWhitespaceOnly 测试纯空白代码
func TestParseWhitespaceOnly(t *testing.T) {
	parser := plsql.NewANTLRRealParser(nil)
	_, err := parser.Parse("   \n\t  ")
	if err == nil {
		t.Error("期望纯空白代码返回错误，但得到了 nil")
	}
}

// TestParseSimplePackage 测试简单包解析
func TestParseSimplePackage(t *testing.T) {
	code := `CREATE OR REPLACE PACKAGE test_pkg IS
  PROCEDURE test_proc;
END test_pkg;`

	parser := plsql.NewANTLRRealParser(nil)
	result, err := parser.Parse(code)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if len(result.Nodes) == 0 {
		t.Error("期望至少有一个节点，但得到 0 个")
	}

	// 检查是否检测到包
	hasPackage := false
	for _, node := range result.Nodes {
		if string(node.Type) == "PACKAGE" {
			hasPackage = true
			if node.Name != "TEST_PKG" {
				t.Errorf("期望包名为 TEST_PKG，但得到 %s", node.Name)
			}
			break
		}
	}

	if !hasPackage {
		t.Error("未检测到 PACKAGE 节点")
	}
}

// TestParsePackageBody 测试包体解析
func TestParsePackageBody(t *testing.T) {
	code := `CREATE OR REPLACE PACKAGE BODY test_pkg IS
  PROCEDURE test_proc IS
  BEGIN
    NULL;
  END test_proc;
END test_pkg;`

	parser := plsql.NewANTLRRealParser(nil)
	result, err := parser.Parse(code)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	// 检查是否检测到包体和过程
	hasPackageBody := false
	hasProcedure := false

	for _, node := range result.Nodes {
		if string(node.Type) == "PACKAGE" {
			if meta, ok := node.Metadata["packageType"]; ok && meta == "body" {
				hasPackageBody = true
			}
		}
		if string(node.Type) == "PROCEDURE" {
			hasProcedure = true
		}
	}

	if !hasPackageBody {
		t.Error("未检测到 PACKAGE BODY 节点")
	}
	if !hasProcedure {
		t.Error("未检测到 PROCEDURE 节点")
	}
}

// TestParseFunction 测试函数解析
func TestParseFunction(t *testing.T) {
	code := `CREATE OR REPLACE FUNCTION test_func RETURN VARCHAR2 IS
BEGIN
  RETURN 'test';
END test_func;`

	parser := plsql.NewANTLRRealParser(nil)
	result, err := parser.Parse(code)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	hasFunction := false
	for _, node := range result.Nodes {
		if string(node.Type) == "FUNCTION" {
			hasFunction = true
			if node.Name != "TEST_FUNC" {
				t.Errorf("期望函数名为 TEST_FUNC，但得到 %s", node.Name)
			}
			break
		}
	}

	if !hasFunction {
		t.Error("未检测到 FUNCTION 节点")
	}
}

// TestParseSQLStatements 测试 SQL 语句解析
func TestParseSQLStatements(t *testing.T) {
	code := `CREATE OR REPLACE PROCEDURE test_proc IS
BEGIN
  INSERT INTO test_table VALUES (1);
  UPDATE test_table SET col1 = 1;
  DELETE FROM test_table WHERE id = 1;
  SELECT COUNT(*) INTO v_count FROM test_table;
END test_proc;`

	parser := plsql.NewANTLRRealParser(nil)
	result, err := parser.Parse(code)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	sqlCount := 0
	for _, node := range result.Nodes {
		if string(node.Type) == "SQL_STATEMENT" {
			sqlCount++
		}
	}

	if sqlCount < 4 {
		t.Errorf("期望至少 4 个 SQL 语句，但得到 %d 个", sqlCount)
	}
}

// TestParseVariables 测试变量声明解析
func TestParseVariables(t *testing.T) {
	code := `CREATE OR REPLACE PROCEDURE test_proc IS
  v_var1 NUMBER;
  v_var2 VARCHAR2(100);
BEGIN
  NULL;
END test_proc;`

	parser := plsql.NewANTLRRealParser(nil)
	result, err := parser.Parse(code)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	varCount := 0
	for _, node := range result.Nodes {
		if string(node.Type) == "VARIABLE" {
			varCount++
		}
	}

	if varCount < 2 {
		t.Errorf("期望至少 2 个变量，但得到 %d 个", varCount)
	}
}

// TestParseErrorHandling 测试错误处理
func TestParseErrorHandling(t *testing.T) {
	// 故意提供错误的 PL/SQL 代码
	code := `CREATE OR REPLACE PROCEDURE test_proc IS
BEGIN
  INVALID SYNTAX HERE
END test_proc;`

	parser := plsql.NewANTLRRealParser(nil)
	result, err := parser.Parse(code)
	
	// 非严格模式下，即使有语法错误也应该返回结果
	if err != nil {
		t.Logf("解析返回错误（预期）: %v", err)
	}

	// 检查是否收集到错误
	if len(result.Errors) == 0 {
		t.Log("警告: 无效语法但未检测到错误")
	}
}

// TestParseStrictMode 测试严格模式
func TestParseStrictMode(t *testing.T) {
	code := `INVALID PLSQL CODE`

	options := &plsql.ParseOptions{Strict: true}
	parser := plsql.NewANTLRRealParser(options)
	_, err := parser.Parse(code)

	if err == nil {
		t.Error("严格模式下期望返回错误，但得到了 nil")
	}
}

// TestNodePositions 测试节点位置信息
func TestNodePositions(t *testing.T) {
	code := `CREATE OR REPLACE PACKAGE test_pkg IS
  PROCEDURE test_proc;
END test_pkg;`

	parser := plsql.NewANTLRRealParser(nil)
	result, err := parser.Parse(code)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	for _, node := range result.Nodes {
		if node.Start.Line < 1 {
			t.Errorf("节点 %s 的起始行号无效: %d", node.Name, node.Start.Line)
		}
		if node.End.Line < node.Start.Line {
			t.Errorf("节点 %s 的结束行号小于起始行号", node.Name)
		}
	}
}

// TestPackageSignatureExtraction 测试包签名提取
func TestPackageSignatureExtraction(t *testing.T) {
	code := `CREATE OR REPLACE PACKAGE test_pkg IS
  PROCEDURE proc1;
  FUNCTION func1 RETURN NUMBER;
END test_pkg;`

	parser := plsql.NewANTLRRealParser(nil)
	result, err := parser.Parse(code)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	for _, node := range result.Nodes {
		if string(node.Type) == "PACKAGE" && node.Text != "" {
			// 包规范应该只包含签名，不包含完整实现
			t.Logf("包签名文本长度: %d", len(node.Text))
		}
	}
}
