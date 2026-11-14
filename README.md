# pgparser

纯 Go 实现的 PostgreSQL SQL 语法解析器。语法由 goyacc 从单一 `parser.y` 生成，无 cgo 依赖，解析结果为 `ast` 包中的语法树节点。

## 特性

- 覆盖 PostgreSQL 主要语句族：`SELECT`、`INSERT`、`UPDATE`、`DELETE`、`MERGE`、DDL、权限、事务、游标、`COPY`、`EXPLAIN`、`VACUUM` 等。
- 建模 `WITH`/CTE、集合运算、窗口函数、`GROUPING SETS`/`ROLLUP`/`CUBE`、`ON CONFLICT`、分区、`XMLTABLE`、`JSON_TABLE`、JSON 聚合等。
- PostgreSQL 特有节点与字段集中在 `ast/pg_ext.go`、`ast/pg_nodes_generated.go`；语法解析与 SQL 恢复的支持范围不同。
- 单文件语法定义：所有产生式集中在 `parser.y`，便于对照 PostgreSQL `gram.y` 维护。
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

`*Parser` 不是并发安全的：`Parse` 会把结果与词法状态写回接收者。每个 goroutine 请单独调用 `pgparser.New()` 创建 Parser，不要跨 goroutine 共享。

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

## 项目结构

```
parser.y            语法定义（goyacc），单文件包含全部产生式
parser.go           由 parser.y 生成的解析器（勿手工修改）
lexer.go            词法分析（含参数、注释、美元引用）
keywords.go         关键字表
lexer_adapter.go    goyacc 词法适配
ast_builder_*.go    语义动作使用的 AST 构造辅助
ast/                语法树节点与 Restore/Accept
tests/              单元测试与 PostgreSQL 回归语料
internal/astgen/    AST visitor 代码生成工具（make generate 调用）
```

## 开发

### 生成解析器

解析器由 `parser.y` 通过 [golang.org/x/tools/cmd/goyacc](https://pkg.go.dev/golang.org/x/tools/cmd/goyacc) 生成：

```bash
make generate
# 等价于：
#   go run ./internal/astgen            # 重新生成 AST visitor
#   go tool goyacc -o parser.go -l -p yy parser.y   # go.mod pins v0.38.0
```

不要在提交中手工修改 `parser.go`。若需查看冲突，可附加 `-v y.output`（该文件已被忽略）。

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

该门禁遍历 `tests/testdata/*.sql`：支持恢复的语句必须能 `Restore` 回 SQL、再次解析并稳定恢复。显式返回 `ast.ErrUnsupportedRestore` 的语句单独统计；其他已知缺口记录在 `tests/testdata/known_failures.txt`。新增失败使门禁失败；不再复现的清单条目单独统计，应及时清理。

### 当前 ISSUE #1–#16 修复与接口变化

本轮修复覆盖只读判定、NUL 与 Unicode 校验、标识符截断、PostgreSQL 例程及 COPY 恢复、ALTER COLUMN、可选 AST 子节点、数据库选项、默认语料门禁与生成一致性。回归测试集中在 `tests/pg_correctness_test.go`。

- `CreateFunctionStmt` 同时表示函数与过程，参数、返回表、选项及 SQL body 保留为可遍历节点；触发器使用 `CreateTrigStmt`。不再输出 MySQL `DEFINER =`，同名用户函数使用普通参数恢复。序列类型和例程签名中的类型节点使用 `DataType`，依赖原字符串节点的调用方需要适配。
- `CreateSchemaStmt.Authorization`、`CreateTablespaceStmt.Owner` 使用 `*RoleSpec` 区分角色关键字与同名带引号角色，依赖原字符串字段的调用方需要适配。
- PostgreSQL-only：移除 MySQL 密码／握手辅助 API、旧字符集查询 API 和 `TableMaintenanceStmt`（CHECK／REPAIR／OPTIMIZE／CHECKSUM TABLE）。`charset` 保留 `CharsetBin`、`CollationBin`、`GetDefaultCharsetAndCollate`；移除运行时 `x/text` 依赖。仓库外使用旧 API 的调用方需要迁移。
- `make generate` 使用锁定的 goyacc 工具依赖，`make generate-check` 检查生成结果；缺少 golangci-lint 时 `make lint` 明确失败。

当前语料门禁检查 **40,644** 条语句：4,332 条显式不支持恢复、4,620 条 psql 依赖跳过、427 条已知失败；过期清单项与意外失败均为 **0**。这是解析与恢复稳定性检查，不代表所有 PostgreSQL 语法或数据库执行语义均受支持。

### 上一轮 ISSUE #1–#20（历史记录）

上一轮修复覆盖分页与集合括号、原文与精确数字、类型元数据、SELECT INTO、角色与权限、聚合/窗口属性、SQL 值函数、visitor 遍历、词法 EOF、JSON 尾随输入、解析结果独立性及无用分配/依赖。回归测试位于 `tests/audit_regression_test.go`、`ast/visitor_test.go`、`ast/pg_ext_test.go` 和 `internal/astgen/main_test.go`。

- 默认恢复使用双引号标识符、`$n` 参数和 PostgreSQL 分页语法。保留 `//` 注释扩展和无值 `OPTIONS (x)`。
- 小数、指数及超出 int64 的数字使用 `*ast.NumericLiteral` 保存数字文本；SQL 值函数使用 `*ast.SQLValueFunction`。依赖旧具体类型或输出的调用方需要适配。
- `types.FieldType` 新增 `PGName`、`PGModifiers`、`PGArrayBounds`、`PGSetOf`、`PGInterval`；Clone 深拷贝切片，Equal 比较这些元数据。
- CAST、`::` 和类型字面量统一恢复为 CAST，保留限定类型名、修饰符及 interval 范围；`NewTypeCastExpr` 使用 `CastFunction`，不再生成 `BINARY`。
- 带 WITH、排序、分页或括号的 VALUES/TABLE 使用 `*ast.SetOprStmt` 包装；SELECT INTO 目标保存在 `SelectStmt.IntoTable`。
- 角色的语义身份保存在 `RoleSpec`；`GrantStmt.Grantor` 改为 `*RoleSpec`，角色授予/撤销的 `Roles`、`Users` 改为 `*List`。函数权限目标的完整节点保存在 `TargetNodes`；未知权限名或角色授权选项会拒绝恢复。
- `JSONTablePathSpec.String` 改为 `Expr`；JSON_TABLE 的 `Passing` 改为 `[]Node`，`OnEmpty`/`OnError` 改为 `*JsonBehavior`，FORMAT、编码和 DEFAULT 表达式保留为可遍历节点。
- 尚未实现的 PostgreSQL 节点恢复及 JSON_TABLE 的 PASSING/FORMAT/DEFAULT 返回明确错误，而非静默省略内容；函数／过程／例程权限目标已在本轮支持恢复。

上一轮历史语料诊断为 **125/40644** 条解析失败，跳过 4620 条；基线为 123 条。新增两条均为 `INVALID_ENCODING` 负向用例被正确拒绝，其余解析失败集合不变。该历史结果仅检查解析，不包含本轮恢复门禁。

### 性能验证（上一轮）

基线与修复版在同一台 AMD Ryzen 7 5700U、linux/amd64、Go 1.27.3 上串行运行，单 CPU、每项 10 次采样，使用 benchstat 比较中位数：

| Benchmark | 基线耗时 | 修复后耗时 | 基线 → 修复后分配 |
| --- | --- | --- | --- |
| ParserNew | 47.47µs ±64% | 139.3ns ±27% | 107776 → 160 B/op；2 → 1 allocs/op |
| LexerOperator/8192 | 65.99ms ±80% | 37.50µs ±63% | 均为 0 B/op、0 allocs/op |
| LexerOperator/16384 | 248.08ms ±3% | 69.95µs ±41% | 均为 0 B/op、0 allocs/op |
| LexerOperator/32768 | 1.310s ±40% | 158.6µs ±82% | 均为 0 B/op、0 allocs/op |

耗时差异的 benchstat 检验均为 p<0.001、n=10。操作符 benchmark 只测首 token，不代表整条 SQL 的解析吞吐量；环境波动较大，绝对耗时不作为性能保证。复测命令：

```bash
go test ./tests -run '^$' -bench '^Benchmark(ParserNew|LexerOperator)$' -benchmem -benchtime=100ms -count=10 -cpu=1
```

## 许可证

本项目采用 **MIT** 与 **Anti-996** 双协议，你可任选其一：

- [MIT License](LICENSE)
- [Anti-996 License 1.0](LICENSE-ANTI-996)

## 致谢

- 语法参考 [PostgreSQL](https://www.postgresql.org/) 的 `gram.y` 及其移植版本 [pgplex/pgparser](https://github.com/pgplex/pgparser)。
- 解析器生成器 [golang.org/x/tools/cmd/goyacc](https://pkg.go.dev/golang.org/x/tools/cmd/goyacc)。
- `ast` 与部分基础设施源自 [TiDB](https://github.com/pingcap/tidb) 的 parser。
