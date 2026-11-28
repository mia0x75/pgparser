# pgparser

纯 Go 实现的 PostgreSQL SQL 语法解析器。语法由 goyacc 从单一 `parser/grammar.y` 生成，无 cgo 依赖，解析结果为 `ast` 包中的语法树节点。

## 特性

- 覆盖 PostgreSQL 主要语句族：`SELECT`、`INSERT`、`UPDATE`、`DELETE`、`MERGE`、DDL、权限、事务、游标、`COPY`、`EXPLAIN`、`VACUUM` 等。
- 建模 `WITH`/CTE、集合运算、窗口函数、`GROUPING SETS`/`ROLLUP`/`CUBE`、`ON CONFLICT`、分区、`XMLTABLE`、`JSON_TABLE`、JSON 聚合等。
- PostgreSQL 特有节点按职责分布在 `ast/` 中，生成节点位于 `ast/nodes_generated.go`；语法解析与 SQL 恢复的支持范围不同。
- 单文件语法定义：所有产生式集中在 `parser/grammar.y`，便于对照 PostgreSQL `gram.y` 维护。
- 无 cgo、无外部解析器运行时依赖。

## 安装

要求 Go 1.27.3 或更新版本。

```bash
go get github.com/mia0x75/pgparser
```

## 快速开始

根包名为 `parser`，建议导入时起别名。使用前需要导入 `driver` 子包以注册 `ast` 的值表达式构造函数。

```go
package main

import (
	"fmt"

	pgparser "github.com/mia0x75/pgparser/parser"
	"github.com/mia0x75/pgparser/ast"
	_ "github.com/mia0x75/pgparser/driver"
)

func main() {
	stmts, err := pgparser.New().Parse("SELECT a, b FROM t WHERE a = 1;")
	if err != nil {
		panic(err)
	}
	fmt.Printf("stmts=%d\n", len(stmts))

	for _, stmt := range stmts {
		if sel, ok := stmt.(*ast.SelectStmt); ok {
			fmt.Printf("select fields: %d\n", len(sel.Fields.Items))
		}
	}
}
```

单语句解析：

```go
stmt, err := pgparser.New().ParseOneStmt("SELECT 1;")
```

> 多条语句需以 `;` 分隔，并以 `;` 结尾，例如 `SELECT 1;SELECT 2;`。

解析器的 `Parse` 返回 `([]ast.StmtNode, error)`；`ParseOneStmt` 要求恰好一条语句，否则返回 `ErrSyntax`。

`*Parser` 可并发复用；每次 `Parse` 都使用独立的解析状态。

## 支持的 SQL

以下列出语法解析范围；使用 `Restore` 时仍须检查返回错误，不能使用报错时已经写出的部分 SQL。

### 查询与 DML

- `SELECT`：`DISTINCT [ON (...)]`、`FROM`/`JOIN`（含 `NATURAL`、`USING`、`LATERAL`、`TABLESAMPLE`）、`WHERE`、`GROUP BY`（含 `GROUPING SETS`/`ROLLUP`/`CUBE`）、`HAVING`、`WINDOW`、`ORDER BY`、`LIMIT`/`OFFSET`/`FETCH`、行锁。
- 集合运算：`UNION`、`INTERSECT`、`EXCEPT`，支持 `ALL` 与链式组合。
- `INSERT`：多行 `VALUES`、`INSERT ... SELECT`、`OVERRIDING`、`ON CONFLICT`、`RETURNING`。
- `UPDATE`：`FROM`、`RETURNING`；`DELETE`：`USING`、`RETURNING`。
- `MERGE`：`WHEN [NOT] MATCHED [BY SOURCE]` 的 `UPDATE`/`DELETE`/`INSERT`/`DO NOTHING`、`RETURNING`。
- 公共表表达式 `WITH [RECURSIVE]`，支持 `AS [NOT] MATERIALIZED`。
- 子查询、`LATERAL`、表函数、`ROWS FROM`、`WITH ORDINALITY`。

### DDL

- `CREATE/ALTER/DROP TABLE`、`CREATE [UNIQUE] INDEX`、`CREATE/ALTER/DROP VIEW`、`CREATE/ALTER/DROP SEQUENCE`、`CREATE SCHEMA`。
- `CREATE/ALTER/DROP DATABASE`、`TABLESPACE`、`SERVER`、`FOREIGN DATA WRAPPER`、`PUBLICATION`、`SUBSCRIPTION`、`STATISTICS`、`POLICY`、`TRIGGER`、`RULE`、`OPERATOR`/`OPERATOR CLASS`/`OPERATOR FAMILY`、`TYPE`/`DOMAIN`、`COLLATION`、`EXTENSION` 等。
- 分区：`PARTITION BY`、`PARTITION OF`、`ATTACH/DETACH PARTITION`、`PartitionBoundSpec`。
- 列与表约束、`GENERATED`、`COLLATE`、`INHERITS`、`WITH (...)`、`TABLESPACE`、`ON COMMIT`。

### 事务、权限与运维

- `BEGIN`/`START TRANSACTION`/`COMMIT`/`ROLLBACK`/`SAVEPOINT`/`RELEASE`，以及事务模式。
- `SET`/`RESET`/`SHOW`、`SET ROLE`、`SET SESSION AUTHORIZATION`。
- `GRANT`/`REVOKE` 全目标（表、序列、函数、类型、模式、语言、大对象、FDW/SERVER 等）。
- `PREPARE`/`EXECUTE`/`DEALLOCATE`、`DECLARE`/`FETCH`/`MOVE`/`CLOSE`、`LISTEN`/`NOTIFY`/`UNLISTEN`。
- `COPY`、`VACUUM`、`ANALYZE`、`REINDEX`、`CLUSTER`、`REFRESH MATERIALIZED VIEW`、`EXPLAIN`、`COMMENT ON`、`IMPORT FOREIGN SCHEMA`。
- `CREATE/ALTER FUNCTION|PROCEDURE`、`DO`、事件触发器、`CALL`。

表达式覆盖字面量、列引用、函数/聚合/窗口调用、`CASE`、`CAST`/`::`、`COLLATE`、`AT TIME ZONE`、数组与行构造器、子查询、`ANY`/`ALL`/`SOME`、`BETWEEN`、`LIKE`/`ILIKE`/`SIMILAR TO`、JSON 运算符等。

> `CREATE EXTENSION` 目前仅返回 `*ast.GenericStmt`（保留语句种类），其余语句均映射为具体 AST 节点。

### 恢复与兼容边界

`CompareSubqueryExpr.OpName` 保留自定义及显式 `OPERATOR(...)`，非空时优先于旧 `Op` 字段；`BetweenExpr.Symmetric`、`ByItem.UseOp` 保留对应表达式与排序语义。`DROP FUNCTION/PROCEDURE` 有无签名均返回对应专用节点：`Arguments` 与 `Names` 对齐，nil 项表示省略签名，空列表表示 `()`；`Behavior` 保留 CASCADE，旧 `Name` 字段仍对应首个目标。

SQL 错误类型、错误类和 SQLSTATE 诊断统一由 `github.com/mia0x75/pgparser/postgres` 提供；原 `sqlerr` 包已移除。解析入口和返回的 SQLSTATE 不变。

`types/json` 使用内部随机访问格式，并非 PostgreSQL 的 jsonb 存储。数字比较不再使用浮点容差或将整数转为 float64；`Unquote` 返回已解码字符串，路径中的 `"*"` 是字面键，裸 `.*` 才是通配符。`MarshalJSON`、`PeekBytesAsJSON` 及返回错误的修改入口拒绝损坏布局、重叠载荷与过深嵌套；无错误返回的随机访问接口仍要求有效的内部值，不应直接传入未经校验的外部字节。

## 项目结构

```
parser/             解析器、词法分析、语法定义与 AST 构造辅助
parser/grammar.y    语法定义（goyacc），单文件包含全部产生式
parser/parser_generated.go 由 grammar.y 生成的解析器（勿手工修改）
ast/                语法树节点与 Restore/Accept
tests/              解析器黑盒单测与 PostgreSQL 官方 SQL 回归语料门禁
internal/astgen/    AST visitor 代码生成工具（make generate 调用）
```

## 开发

### 生成解析器

解析器由 `parser/grammar.y` 通过 [golang.org/x/tools/cmd/goyacc](https://pkg.go.dev/golang.org/x/tools/cmd/goyacc) 生成：

```bash
make generate
# 等价于：
#   go run ./internal/astgen            # 重新生成 AST visitor
#   go tool goyacc -o parser/parser_generated.go -l -p yy parser/grammar.y   # go.mod pins v0.52.0
```

不要在提交中手工修改 `parser/parser_generated.go`。若需查看冲突，可附加 `-v y.output`（该文件已被忽略）。

### 测试与检查

```bash
make            # build + vet + test + lint
make generate-check  # Verify generated files match the grammar
make test       # go test -race -count=1 -cover ./...
make vet        # go vet ./...
make lint       # golangci-lint run
```

解析器对 PostgreSQL 官方回归测试集的覆盖情况由默认门禁 `TestParseTestdataSQL` 检查：

```bash
go test ./tests -run TestParseTestdataSQL            # 默认门禁：新出现的失败即报错
PGPARSER_TESTDATA=1 go test -v ./tests -run TestParseTestdataSQL   # 附加逐语句诊断
```

该门禁遍历 `tests/testdata/*.sql`：支持恢复的语句必须能 `Restore` 回 SQL、再次解析并稳定恢复。只有首次恢复显式返回 `ast.ErrUnsupportedRestore` 的语句单独统计，二次恢复报错不可跳过；其他已知缺口记录在 `tests/testdata/known_failures.txt`。新增失败使门禁失败；不再复现的清单条目单独统计，应及时清理。定向回归另行断言预期 SQL 与 AST 属性，避免把稳定的语义丢失当成成功。

### 性能验证

```bash
go test ./tests -run '^$' -bench '^Benchmark(Parse|LexerOperator)$' -benchmem -benchtime=100ms -count=10 -cpu=1
```

## 许可证

本项目采用 **MIT** 与 **Anti-996** 双协议，你可任选其一：

- [MIT License](LICENSE)
- [Anti-996 License 1.0](LICENSE-ANTI-996)

## 致谢

- 语法参考 [PostgreSQL](https://www.postgresql.org/) 的 `gram.y` 及其移植版本 [pgplex/pgparser](https://github.com/pgplex/pgparser)。
- 解析器生成器 [golang.org/x/tools/cmd/goyacc](https://pkg.go.dev/golang.org/x/tools/cmd/goyacc)。
- `ast` 与部分基础设施按 PostgreSQL 语法解析器的通用设计独立实现。
