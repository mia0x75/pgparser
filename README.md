# pgparser

纯 Go 实现的 PostgreSQL SQL 语法解析器。语法由 goyacc 从单一 `parser.y` 生成，无 cgo 依赖，解析结果为 `ast` 包中的语法树节点。

## 特性

- 覆盖 PostgreSQL 主要语句族：`SELECT`、`INSERT`、`UPDATE`、`DELETE`、`MERGE`、DDL、权限、事务、游标、`COPY`、`EXPLAIN`、`VACUUM` 等。
- 完整建模 `WITH`/CTE、集合运算、窗口函数、`GROUPING SETS`/`ROLLUP`/`CUBE`、`ON CONFLICT`、分区、`XMLTABLE`、`JSON_TABLE`、JSON 聚合等。
- 保留 PostgreSQL 语义：独有的节点与字段集中在 `ast/pg_ext.go`、`ast/pg_nodes_generated.go`，不为了兼容而丢弃信息。
- 单文件语法定义：所有产生式集中在 `parser.y`，便于对照 PostgreSQL `gram.y` 维护。
- 无 cgo、无外部解析器运行时依赖。

## 安装

```bash
go get github.com/mia0x75/pgparser
```

## 快速开始

根包名为 `parser`，建议导入时起别名。使用前需要导入 `driver` 子包以注册 `ast` 的值表达式构造函数。

```go
package main

import (
	"fmt"

	pgparser "github.com/mia0x75/pgparser"
	"github.com/mia0x75/pgparser/ast"
	_ "github.com/mia0x75/pgparser/driver"
)

func main() {
	stmts, warns, err := pgparser.New().Parse("SELECT a, b FROM t WHERE a = 1;")
	if err != nil {
		panic(err)
	}
	fmt.Printf("stmts=%d warns=%d\n", len(stmts), len(warns))

	for _, stmt := range stmts {
		if sel, ok := stmt.(*ast.SelectStmt); ok {
			fmt.Printf("select fields: %d\n", len(sel.Fields.Fields))
		}
	}
}
```

单语句解析：

```go
stmt, err := pgparser.New().ParseOneStmt("SELECT 1;")
```

> 多条语句需以 `;` 分隔，并以 `;` 结尾，例如 `SELECT 1;SELECT 2;`。

`Parse` 返回 `([]ast.StmtNode, []error, error)`；`ParseOneStmt` 要求恰好一条语句，否则返回 `ErrSyntax`。

## 支持的 SQL

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

## 项目结构

```
parser.y            语法定义（goyacc），单文件包含全部产生式
parser.go           由 parser.y 生成的解析器（勿手工修改）
lexer.go            词法分析
keywords.go         关键字表
lexer_adapter.go    goyacc 词法适配
preprocess.go       输入预处理（$/美元引用感知）
ast_builder_*.go    语义动作使用的 AST 构造辅助
ast/                语法树节点与 Restore/Accept
tests/              单元测试与 PostgreSQL 回归语料
```

## 开发

### 生成解析器

解析器由 `parser.y` 通过 [golang.org/x/tools/cmd/goyacc](https://pkg.go.dev/golang.org/x/tools/cmd/goyacc) 生成：

```bash
go run golang.org/x/tools/cmd/goyacc@v0.38.0 -o parser.go -l -p yy parser.y
```

不要在提交中手工修改 `parser.go`。若需查看冲突，可附加 `-v y.output`（该文件已被忽略）。

### 测试与检查

```bash
make            # vet + test + lint
make test       # go test -race -count=1 ./...
make vet        # go vet ./...
make lint       # golangci-lint run
```

解析器对 PostgreSQL 官方回归测试集的覆盖情况可通过诊断测试查看：

```bash
PGPARSER_TESTDATA=1 go test ./tests -run TestParseTestdataSQL -v
```

该测试默认跳过；启用后会遍历 `tests/testdata/*.sql`，报告无法解析的语句。当前约 4 万条语句中失败约 123 条，绝大多数是 PostgreSQL 本身也会拒绝的负向用例。

## 许可证

本项目采用 **MIT** 与 **Anti-996** 双协议，你可任选其一：

- [MIT License](LICENSE)
- [Anti-996 License 1.0](LICENSE-ANTI-996)

## 致谢

- 语法参考 [PostgreSQL](https://www.postgresql.org/) 的 `gram.y` 及其移植版本 [pgplex/pgparser](https://github.com/pgplex/pgparser)。
- 解析器生成器 [golang.org/x/tools/cmd/goyacc](https://pkg.go.dev/golang.org/x/tools/cmd/goyacc)。
- `ast` 与部分基础设施源自 [TiDB](https://github.com/pingcap/tidb) 的 parser。
