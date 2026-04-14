# PL/SQL Parser - 基于 ANTLR 的 PL/SQL 解析器

**版本**: v2.1.0 | **架构**: 纯 ANTLR 实现 | **用途**: PL/SQL 语法解析与代码分析

---

## 📖 项目简介

PL/SQL Parser 是一个基于 **ANTLR v4.13.0** 构建的完整 PL/SQL 解析器，专为 Oracle E-Business Suite 代码分析与审计而设计。

### ✨ 核心特性

- ✅ **纯 ANTLR 实现** - 基于形式化语法的精准解析
- ✅ **Listener 模式** - 自动 AST 遍历，类型安全
- ✅ **结构识别** - 包、变量、SQL 语句等自动检测
- ✅ **无 UNKNOWN 节点** - 所有节点都有明确类型
- ✅ **独立模块** - 可作为库或 CLI 工具使用
- ✅ **双层输出优化** - Clean/Lean 两级优化，节省 LLM Token

---

## 🏗️ 项目结构

```
plsql-parser/
├── grammar/                 # ANTLR 语法文件
│   ├── PlSqlLexer.g4        # 词法分析器语法
│   └── PlSqlParser.g4       # 语法分析器规则
├── tools/
│   └── antlr-4.13.0-complete.jar  # ANTLR 工具
├── internal/
│   ├── antlr/               # ANTLR 生成的 Go 代码
│   │   ├── plsql_lexer.go
│   │   ├── plsql_parser.go
│   │   ├── plsqlparser_listener.go
│   │   └── plsqlparser_base_listener.go
│   └── plsql/
│       ├── parser.go        # 解析器封装
│       └── listener.go      # Listener 实现
├── pkg/
│   ├── types/               # 类型定义
│   └── optimizer/           # 输出优化器
│       ├── optimizer.go     # 三层优化逻辑
│       └── optimizer_test.go # 单元测试
├── example/                 # 示例文件
│   ├── simple_package.sql
│   ├── simple_function.sql
│   └── simple_procedure.sql
├── cmd/                     # CLI 工具
├── go.mod
└── README.md
```

---

## 🚀 快速开始

### 1. 编译二进制文件

```bash
cd plsql-parser
go build -o plsql-parser.exe ./cmd
```

### 2. 通过 Go 安装

```bash
go install github.com/username/plsql-parser/cmd/plsql-parser@latest
```

### 3. CLI 使用方式

```bash
# 基础用法（文本输出）
./plsql-parser example/simple_package.sql

# JSON 输出与优化级别
./plsql-parser example/simple_package.sql --json              # 默认无优化
./plsql-parser example/simple_package.sql --json -L 1         # L1: 物理去噪
./plsql-parser example/simple_package.sql --json -L 2         # L2: 内容精简

# 保存到文件
./plsql-parser example/simple_package.sql --json -L 2 -o result.json
```

**优化级别说明：**

| 级别 | 参数 | 优化内容 | 适用场景 |
|------|------|---------|----------|
| **无优化** | 无 | 保留所有节点和源代码 | 调试、完整分析 |
| **L1 Clean** | `-L 1` | 移除变量/常量声明节点 | 关注业务逻辑 |
| **L2 Lean** | `-L 2` | L1 + 折叠SQL字段列表、压缩机械代码 | LLM 调用、Token 节省 |

**文本输出示例：**
```
========================================
  PL/SQL 解析结果
========================================

代码行数: 935
节点数量: 57
解析错误: 0
解析警告: 0

节点统计:
  📦 PACKAGE              2
  🔧 FUNCTION             3
  ⚙️  PROCEDURE            8
  📝 VARIABLE             38
  💾 SQL_STATEMENT        6
```

**JSON 输出示例：**
```json
{
  "root": {...},
  "nodes": [
    {
      "type": "PACKAGE",
      "name": "CUX_WS_TPM_SHPREQ_PKG",
      "start": {"line": 1, "column": 0},
      "metadata": {"packageType": "spec"}
    },
    {
      "type": "PROCEDURE",
      "name": "MAIN_SHPREQ",
      "start": {"line": 30, "column": 2},
      "metadata": {"objectType": "spec"}
    }
  ]
}
```

---

## 💻 编程接口

```go
package main

import (
    "fmt"
    "github.com/username/plsql-parser/internal/plsql"
)

func main() {
    code := `
    CREATE OR REPLACE PACKAGE BODY xx_test_pkg AS
        PROCEDURE test_proc IS
        BEGIN
            INSERT INTO fnd_user VALUES (1, 'test');
        END;
    END;
    `
    
    parser := plsql.NewANTLRRealParser(nil)
    result, err := parser.Parse(code)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("检测到 %d 个节点\n", len(result.Nodes))
    for _, node := range result.Nodes {
        fmt.Printf("- [%s] %s (行 %d)\n", node.Type, node.Name, node.Start.Line)
    }
}
```

**输出：**
```
检测到 3 个节点
- [PACKAGE] XX_TEST_PKG (行 2)
- [PROCEDURE] TEST_PROC (行 3)
- [SQL_STATEMENT]  (行 5)
```

---

## 📊 支持的结构识别

### ✅ 已实现功能

#### 1. 包（Packages）
- **包规范** (`CREATE PACKAGE`)
  - 自动提取签名部分（到 IS/AS 为止）
  - 元数据: `packageType: "spec"`
- **包体** (`CREATE PACKAGE BODY`)
  - 自动提取签名部分
  - 元数据: `packageType: "body"`

#### 2. 过程（Procedures）
- **独立过程** (`CREATE PROCEDURE`)
  - 完整代码提取
  - 元数据: `objectType: "body"`
- **包内过程规范** (`PROCEDURE spec`)
  - 仅头部声明
  - 元数据: `objectType: "spec"`
- **包内过程体** (`PROCEDURE body`)
  - 完整实现代码
  - 元数据: `objectType: "body", scope: "package"`

#### 3. 函数（Functions）
- **独立函数** (`CREATE FUNCTION`)
  - 完整代码提取
  - 元数据: `objectType: "body", scope: "standalone"`
- **包内函数规范** (`FUNCTION spec`)
  - 仅头部声明
  - 元数据: `objectType: "spec"`
- **包内函数体** (`FUNCTION body`)
  - 完整实现代码
  - 元数据: `objectType: "body", scope: "package"`

#### 4. 变量声明（Variables）
- 所有变量声明节点
- L1 优化时会被移除
- 包含变量名和完整声明文本

#### 5. SQL 语句
- **INSERT** - 自动提取表名到 `metadata.table`
- **UPDATE** - 自动提取表名到 `metadata.table`
- **DELETE** - 自动提取表名到 `metadata.table`
- **SELECT** - 识别但不提取具体字段
- 元数据: `sqlType: "INSERT|UPDATE|DELETE|SELECT"`

### ⚠️ 当前局限

#### 1. 未识别的节点类型
以下 NodeType 在 types.go 中定义但 **listener.go 中未实现**：
- `CONSTANT` - 常量声明（虽定义了 EnterVariable_declaration，但未区分常量）
- `CURSOR` - 游标声明
- `TYPE` - 类型定义
- `EXCEPTION` - 异常声明
- `IF_STATEMENT` - IF 语句
- `LOOP_STATEMENT` - 循环语句
- `ASSIGNMENT` - 赋值语句
- `CALL` - 函数调用
- `RETURN` - 返回语句
- `RAISE` - 异常抛出
- `COMMENT` - 注释
- `UNKNOWN` - 未知节点

**影响**: 这些结构虽然存在于 PL/SQL 代码中，但解析器不会生成对应的节点。

#### 2. 表名提取限制
- 使用启发式方法提取表名
- 只返回最后一个标识符（如 `SCHEMA.TABLE` 只返回 `TABLE`）
- 复杂查询（JOIN、子查询）可能提取不准确

#### 3. 包签名提取
- 尝试提取到第一个 IS/AS 关键字
- 如果找不到 IS/AS，降级为完整文本
- 对于复杂的包规范可能不够精确

---

## 📈 性能基准测试

### 实际测试结果

**小文件（1.3KB, ~50 行）**：
- 解析时间：**~4ms**
- 检测节点：9 个（2 PACKAGE + 2 FUNCTION + 2 PROCEDURE + 3 SQL）

**中等文件（43KB, 935 行）**：
- 解析时间：**~1.27s**
- 吞吐量：约 34 KB/s

**大文件（test.pkg, 1.3KB, 1337 行）**：
- 解析时间：**~4.2ms** (单次)
- Benchmark: 291 次迭代，平均 4.2ms/op

> 💡 **性能建议**：
> - 对于批量处理，建议实现缓存机制
> - ANTLR 解析器初始化开销较大，可复用 parser 实例
> - 大文件（>10MB）会被 CLI 拒绝（防止 OOM）

### 优化效果

| 优化级别 | 节点过滤 | 文本压缩 | Token 节省 |
|---------|---------|---------|----------|
| **无优化** | 无 | 无 | 0% |
| **L1 Clean** | 移除 VARIABLE/CONSTANT | 无 | ~10-20% |
| **L2 Lean** | L1 + 折叠简单函数 | 去注释+折叠SQL | ~30-50% |

---

## 🔧 重新生成 ANTLR 代码

如果修改了 `.g4` 语法文件：

```bash
cd plsql-parser

# 1. 清理旧文件
Remove-Item internal\antlr\*.go -Force

# 2. 生成新代码
java -jar tools\antlr-4.13.0-complete.jar `
  -Dlanguage=Go `
  -package antlr `
  -o internal\antlr `
  grammar\PlSqlLexer.g4 grammar\PlSqlParser.g4

# 3. 修复 Java 特定代码
(Get-Content internal\antlr\plsql_parser.go) `
  -replace '\bthis\b', 'p' | `
  Set-Content internal\antlr\plsql_parser.go
```

---

## 🎯 核心架构

### 解析流程

```
PL/SQL 源代码
    ↓
词法分析器 (Lexer)
    ↓
语法分析器 (Parser)
    ↓
AST 抽象语法树
    ↓
Listener 遍历收集节点
    ↓
结构化节点列表
    ↓
输出优化器 (Optimizer)
    ↓
JSON/文本输出
```

### 双层优化架构

**L1 - 物理去噪（Clean）**
```bash
./plsql-parser test.pkg --json -L 1
```
- 移除 `VARIABLE` 和 `CONSTANT` 类型节点
- 保留所有过程、函数、SQL 语句等业务逻辑
- 清理空 Metadata 对象

**L2 - 内容精简（Lean）**
```bash
./plsql-parser test.pkg --json -L 2
```
- 包含 L1 的所有优化
- 折叠 SELECT 语句的字段列表：`SELECT [Fields List Collapsed] FROM ...`
- 压缩机械映射代码（如大量 `PUT_JSON_FIELD` 调用）
- 隐藏简单包装函数的实现细节
- 移除行内注释（`--` 开头的注释）

---

## 📝 许可证

MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。

---

## 📚 相关文档

- **[项目深度审计报告](docs/PROJECT_AUDIT_REPORT.md)** - 完整源码审查、风险识别、测试体系、改进建议

---

**最后更新**: 2026-04-11  
**ANTLR 版本**: 4.13.0  
**状态**: ✅ 生产就绪
