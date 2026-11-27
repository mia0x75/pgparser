# 双仓 grammar 规则映射基线

这是 B0 的可复核映射清单。它描述当前产生式职责，不承诺同名规则等价，也不授权把方言语法合并。规则名来自两仓 `parser/grammar.y` 的规则头；生成基线见 [GRAMMAR_ALIGNMENT_BATCHES.md](GRAMMAR_ALIGNMENT_BATCHES.md)。

## 判断标准

- **精确同名**：两仓都有同名规则，但仍须读取产生式和语义动作后才能迁移。
- **部分重叠**：只对齐共同 AST 字段或语法子集；入口、消歧和方言扩展继续分离。
- **方言专属**：只登记锚点，不补齐另一仓的规则。
- **待核实**：名称或 AST 看似相关，但当前证据不足；禁止机械改名。

## 精确同名规则

两仓共有 92 个规则头，按当前主要职责分组如下。入口规则也登记在内，不因同名而视为相同入口。

### 入口与语句

`Start`、`Statement`、`StatementList`、`InsertStmt`、`UpdateStmt`、`DeleteStmt`、`PreparedStmt`、`ExecuteStmt`、`DeallocateStmt`、`ExplainStmt`、`ExplainableStmt`、`CallStmt`、`DoStmt`、`ReturnStmt`、`TruncateTableStmt`、`LockType`、`ResourceGroupType`。

### 查询、投影、表源与集合

`SelectStmt`、`Field`、`FieldList`、`ExpressionList`、`TableName`、`TableRef`、`TableRefs`、`TableRefsClause`、`TableAsNameOpt`、`TableElement`、`TableElementList`、`JoinTable`、`JoinType`、`Order`、`OrderBy`、`OrderByOpt`、`ByList`、`ByItem`、`LimitClause`、`GroupByClause`、`HavingClause`、`WhereClause`、`WhereClauseOpt`、`WithClause`、`WithList`、`WithClauseItem`、`WindowClauseOpt`、`WindowDefinitionList`、`WindowDefinition`、`WindowSpec`、`WindowFrameExtent`、`WindowFrameBound`、`ReturningOpt`。

### DDL、DML 与共享节点边界

`Assignment`、`AssignmentList`、`ColumnDef`、`ColumnList`、`ConstraintElem`、`CreateTableStmt`、`CreateDatabaseStmt`、`DropDatabaseStmt`、`CreateSequenceStmt`、`CreateTablespaceStmt`、`AlterTablespaceStmt`、`DropTablespaceStmt`、`CreateFunctionStmt`、`CreateRoleStmt`、`CreateUserStmt`、`CreateTriggerStmt`、`CreateViewStmt`、`AlterDatabaseStmt`、`AlterFunctionStmt`、`AlterTableStmt`、`GrantStmt`、`GrantRoleStmt`、`RevokeStmt`、`RevokeRoleStmt`、`DropRoleStmt`。

### 词法、选项与辅助规则

`OptBinary`、`OptCollate`、`OptExistingWindowName`、`OptFull`、`OptPartitionClause`、`OptTable`、`OptWindowFrameClause`、`OptWindowFrameExclusion`、`OuterOpt`、`RoleSpec`、`UnReservedKeyword`、`WhenClause`、`WhenClauseList`、`JSONTableExpr`、`JSONTableColumn`、`JSONTableColumnList`、`DropServerStmt`。

## 可实施映射

| 族 | PostgreSQL | MySQL / MariaDB | 共同 AST 核对点 | 当前结论 |
| --- | --- | --- | --- | --- |
| 投影 | `Field`、`FieldList`、`OptTargetList` | `Field`、`FieldList`、`SelectStmtFieldList` | `SelectItem`、别名、通配符、字段顺序、Span | `OptTargetList` 与 `SelectStmtFieldList` 都是入口适配器，不合并；Q1a 只改 PG 名称。 |
| SELECT 主体 | `SimpleSelect`、`SelectClause` | `SelectStmtBasic`、`SelectStmtFromDualTable`、`SelectStmtFromTable` | `BaseSelectStmt` 的 Fields/From/Where/GroupBy/Having | 部分重叠；Q2 只核对字段挂载，不改方言子句。 |
| FROM / JOIN | `TableRefsClause`、`TableRefs`、`TableRef`、`JoinTable`、`JoinQual` | `TableRefsClause`、`TableRefs`、`TableRef`、`TableFactor`、`JoinTable` | `BaseTableSource`、`BaseJoinExpr` 的 Left/Right/Type/On/Using/Natural | 部分重叠；Q3 先保留两仓消歧结构。另有顶层表列表的 AST 形状差异，见 Q3 进度，需另立 AST/行为决策。 |
| 查询尾部 | `OrderBy`、`OrderByOpt`、`SelectLimit`、`ForLockingClause` | `OrderBy`、`OrderByOpt`、`SelectStmtLimit`、`SelectLockClause` | OrderBy、Limit、锁等待属性 | 部分重叠；PG `FETCH` 与 MySQL `LOCK IN SHARE MODE` 不互换。 |
| CTE | `WithClause`、`WithList`、`WithClauseItem`、`OptSearchClause`、`OptCycleClause` | `WithClause`、`WithList`、`WithClauseItem`、`CTEBody`、`WithCycleOpt` | `BaseWithClause`、`BaseCommonTableExpression` | 核心字段重叠；主体类型和递归扩展分离。 |
| 窗口 | `WindowClauseOpt`、`WindowDefinition`、`WindowSpec`、`WindowFrameExtent`、`WindowFrameBound` | 同名规则及 `WindowSpecDetails`、`WindowFrameUnits` 等 | `BaseWindowSpec`、frame 边界、命名窗口 | 部分重叠；Q7 逐子项验证。 |
| INSERT | `InsertStmt`、`InsertTarget`、`InsertRest`、`ValuesClause` | `InsertStmt`、`InsertValues`、`ValuesStmt`、`ReplaceIntoStmt` | `BaseInsertStmt` 的目标、列、Values/Query | 核心重叠；`ON CONFLICT`、`REPLACE`、`ON DUPLICATE KEY` 分离。 |
| UPDATE / DELETE | `UpdateStmt`、`DeleteStmt`、`AssignmentList`、`ReturningOpt` | `UpdateStmt`、`DeleteStmt`、`AssignmentList`、`ReturningOpt` | `BaseUpdateStmt`、`BaseDeleteStmt`、Assignment、Where、Returning | 核心重叠；PG `FROM`/`USING` 与 MySQL 多表规则分离。 |
| 约束 | `ConstraintElem`、`KeyActions`、`ConstraintAttributeSpec` | `ConstraintElem`、`ReferDef`、`Constraint` | Constraint 类型、外键动作、命名与选项 | 仅比对外键动作；PG 可延迟/排斥约束保留。 |
| 表与索引 | `CreateTableStmt`、`ColumnDef`、`TableConstraint`、`IndexStmt` | `CreateTableStmt`、`ColumnDef`、`CreateTableDefinition`、`CreateIndexStmt` | ColumnDef、Constraint、Index 共享字段 | 部分重叠；存储、分区、索引前缀各自保留。 |
| 表达式 / 类型 | `AExpr`、`BExpr`、`CExpr`、`ColumnRef`、`AExprConst`、`FuncApplication`、`Typename` | `Expression`、`PredicateExpr`、`BitExpr`、`SimpleExpr`、`ColumnRef`、`Literal`、`FunctionCallGeneric`、`Type` | Expr、Literal、TypeName、FunctionCall | 语义映射而非规则合并；E1–E3 分开推进。 |

## 明确的方言锚点

| PostgreSQL 侧 | MySQL / MariaDB 侧 | 处理 |
| --- | --- | --- |
| `DistinctClause`、`GroupingSetsClause`、`OptOnConflict`、`OptSearchClause`、`OptCycleClause`、`ForLockingClause`、`CopyStmt`、`CreateFdwStmt`、`CreatePublicationStmt`、`CreateSubscriptionStmt`、`MergeStmt` | `SelectOptionList`、`OnDuplicateKeyUpdate`、`ReplaceIntoStmt`、`WithCycleOpt`、`SelectLockClause`、`LoadDataStmt`、`ChangeMasterStmt`、`ShowStmt`、`SpStmt` | 记录为方言扩展，不为表面命名相似而迁移。 |
| `Typename`、`SimpleTypename`、`ConstTypename`、`AExpr`／`BExpr`／`CExpr` | `Type`、`CastType`、`CastDatetimeType`、`Expression`／`PredicateExpr`／`BitExpr`／`SimpleExpr` | 保留各自的优先级和消歧层次；只核对 AST 结果。 |
| `INHERITS`、`ATTACH PARTITION`、`EXCLUDE`、`WHERE CURRENT OF`、`FETCH ... WITH TIES` | `FROM DUAL`、`STRAIGHT_JOIN`、索引提示、`LOCK IN SHARE MODE`、`LIMIT offset,count`、SQL mode 选项 | 作为正反例加入对应批次，不能用共同规则覆盖。 |

## 测试与缺口

| 族 | PG 现有护栏 | MySQL / MariaDB 现有护栏 | B0 结论 |
| --- | --- | --- | --- |
| 查询核心 | `tests/query_core_alignment_test.go`、`cte_test.go`、`join_test.go`、`table_source_test.go`、`set_operation_test.go`、`projection_test.go` | `tests/query_core_alignment_test.go`，另有 parser 内部 SELECT、JOIN、集合和 Span 测试 | 共享核心已有最小正例；MySQL 需要后续补齐同名族定向测试。 |
| DML | `tests/dml_test.go` | `parser/dml_syntax_test.go`、`parser/insert_test.go`、`parser/parser_span_test.go` | 先复用现有测试，D1–D3 不改变动作。 |
| 语料 | `tests/testdata_parse_test.go`，含 `known_failures.txt` 分类 | `cmd/mtrcheck`，按 case ID 记录 MTR 结果；保留 `tests/local_*_test.go` | 每批都跑；已知失败不能扩大豁免。 |
| 生成器 | `make generate`，PG 基线 418/37 | `make parser`，要求 0/0 | 生成表差异是方言事实，不作为结构等价证明。 |

## B0 验收记录

- 规则头：PG 698，MySQL 738；精确同名 92。
- 生成基线：PG 3,012 产生式、6,204 状态、126,217 `yyAct`、418 shift/reduce、37 reduce/reduce；MySQL 2,891 产生式、4,906 状态、77,825 `yyAct`、0/0 冲突。
- 共享 AST：两仓使用 `parserkit v1.0.0`；已核对的 `select.go`、`cte.go`、`join.go`、`table_source.go`、`insert.go`、`update.go`、`delete.go`、`window.go` 与发布版一致。
- 迁移范围：B0 只增加映射文档；未改 `grammar.y`、生成 parser、AST、测试和依赖。

## Q3 结构核对记录

- 双仓共享护栏覆盖 `RIGHT JOIN ... ON`、`LEFT JOIN ... USING`、`NATURAL JOIN`；断言 `BaseJoinExpr` 字段与左右 `BaseTableSource` 别名。
- 尚存 AST 形状差异：PG `TableRefs` 在单一来源时直接保留表源，逗号列表用 `JoinTypeCross` 连接；MySQL `TableRefs` 先构造左单边 `JoinExpr`，逗号列表再用 `JoinTypeInner` 拼接，并以括号组记录保护显式连接分组。共享 `BaseFromClause.Table` 类型允许这两种树形，但消费者观察到的节点不同；本批只锁定真实 JOIN 的共同核心，不折叠或改写该行为。若要统一树形，先审计 Restore、Visitor 和查询规划消费者，列为独立 AST/行为决策。

## Q4 结构核对记录

- PG 在 `SelectClause` 中接入 SELECT、集合运算、`VALUES` 和 `TABLE`；MySQL 将 `SelectStmt`、`SetOprStmt`、`ValuesSetOprStmt` 与 `TableStmt` 作为不同入口。保持入口分离，避免 `VALUES` 表达式歧义和入口消歧变化。
- 双仓 `TestParseSharedSetOperationCore` 断言共享 `BaseSetOperation`、`INTERSECT` 优先级、显式括号组和 `VALUES ... UNION VALUES ...` 的共同核心。单 SELECT 分支、VALUES 构造和 TABLE 查询仍由各自现有方言测试覆盖。
- 两仓用共享 `QueryTailStmt` 的 `Parenthesized` 标记包装显式括号内的集合操作；PG 还以 `SelectStmt.IsInBraces` 标记简单 SELECT 分支。继续保留此 AST 层次，不为规则同名而折叠。

## Q5 结构核对记录

- `TestParseSharedQueryTailCore` 断言普通 SELECT 和集合整体上的共享 `OrderByClauseCore`、排序方向及 `LimitCore` 的 Count/Offset；两仓均把集合整体的尾部保存在 `SetOprStmt.Tail`。
- `TestLimitCommaKeepsOffsetAndCountOrder` 固定 `LIMIT 2, 3` 的现有映射：共享 Count 为 3、Offset 为 2。PG 恢复为 `LIMIT 3 OFFSET 2`，MySQL 保留 `LIMIT 2,3`；仅输出拼写不同。
- 括号分支的尾部作用域已有方言测试：PG `TestSetOpParenthesizedScopeAndTail` 将尾部留在 `SelectStmt`；MySQL `TestSetOperationPaginationSpans` 检查括号分支和集合整体尾部的各自归属。PG `FETCH ... WITH TIES` 与 MySQL 锁尾语法继续分离。

## Q6 结构核对记录

- 双仓 `TestParseSharedRecursiveCTECore` 断言 `BaseWithClause.IsRecursive`、`BaseCommonTableExpression.Name`／`Columns`／`Query`，并验证递归主体是共享集合操作；`TestParseSharedCTEOnSetOperation` 确认 WITH 附着于集合整体而非首个 SELECT 分支。
- PG `WithClauseItem` 的主体为 `PreparableStmt`，允许写入型 CTE；MySQL `CTEBody` 只接查询分支。保留这一语法接受范围差异。
- PG 的 `MATERIALIZED`、`SEARCH` 和标准 `CYCLE`，以及 MariaDB `CYCLE ... RESTRICT` 均继续由各自扩展承载；已有方言用例覆盖，不改语义动作。

## Q7 结构核对记录

- PG `WindowSpec` 的方言字段原先遮蔽 `BaseWindowSpec` 中的分区、排序和 frame 字段；窗口解析动作只填入方言视图。Q7 补齐解析时的共享字段，并为 PG 分区项提供 `WindowPartitionClauseNode`／`WindowPartitionItemNode` 适配，同时保留现有 `PartitionByClause.Items` 供 PG AST 使用。
- 双仓 `TestParseSharedWindowCore` 固定命名窗口的共享名称、分区表达式、排序方向以及 ROWS frame 的边界和 offset 表达式；PG 另验证 visitor 替换分区节点后共享视图仍同步。
- PG `OptPartitionClause` 接受 `ExpressionList`；MySQL `OptPartitionClause` 使用 `ByList`，其分区项另存方向。MySQL frame 起始项还保留数字、字符串、参数和 `INTERVAL` 形式的语义动作；不折叠这些解析差异。产生式分支与 SQL 接受范围不变。

## D1a 结构核对记录

- 双仓 `TestParseSharedInsertCore` 覆盖共享 INSERT 目标、两列、两行 values，以及单列查询来源；Values 与 Query 两个来源槽位保持互斥。
- PG `InsertRest` 经 `SelectStmt` 承接 VALUES，再由 `InsertStmt` 动作转存到共享 `Values`；MySQL `InsertValues` 直接填充共享 `Values`。两者对外共享相同核心形状。
- PG 目标别名独立存入 `TargetAlias`，MySQL 目标表保持直接表名并支持目标分区；`ON CONFLICT`、`ON DUPLICATE KEY UPDATE`、`REPLACE`、`INSERT ... SET`、行别名及 `OVERRIDING` 仍是方言扩展，本子批不调整语法或动作。

## D2a 结构核对记录

- 双仓 `TestParseSharedAssignmentCore` 断言两条 UPDATE 赋值均为共享 `pkast.Assignment`，且共享 Target／Expr 槽位非空。
- PG `Assignment` 允许行目标、列下标和普通列目标；MySQL `Assignment` 目标由 `ColumnName` 构造，支持方言限定列名和 `DEFAULT` 值。保留 AST 目标表达式的具体差异，共享节点只统一 Target 与 Expr 字段。

## D2b 结构核对记录

- 双仓 `TestParseSharedReturningCore` 断言 INSERT、UPDATE、DELETE 的 `Returning` 均为共享 `FieldList`，且按序包含两个非空表达式项。
- 两仓 `ReturningOpt` 都复用投影字段列表；PG 通过 `returningFieldList` 填充共享 AST，MySQL 在其 INSERT／单表 UPDATE／单表 DELETE 动作中附着字段列表。MySQL/MariaDB 的产生式位置与方言支持边界保持原样，不改 grammar 或动作。

## D3a 结构核对记录

- 双仓 `TestParseSharedUpdateCore` 检查共享 `BaseUpdateStmt` 的目标、赋值和过滤条件，并确认共同 CTE 挂在 UPDATE 外层节点。
- PG `UPDATE ... FROM` 仍由 `UpdateStmt.From` 承载；MySQL JOIN 仍在共享 `BaseUpdateStmt.Table` 关系树内，单表 `ORDER BY`／`LIMIT` 与多表限制保留。现有方言用例继续覆盖这些差异；本子批不改产生式和动作。

## D3b 结构核对记录

- 双仓 `TestParseSharedDeleteCore` 核对共享目标、过滤和 WITH CTE 核心；单目标删除中共享 `Using` 槽位保持 nil。
- PG `DELETE ... USING` 与 MySQL 多表 DELETE 的目标列表／来源语法由既有方言用例单独覆盖；MySQL 单表 `ORDER BY`／`LIMIT` 和 RETURNING 附着也不移动或改写。本子批不改产生式和动作。

## E1a 列名核心

- PG `ColumnRef` 与 MySQL `SimpleIdent` 的单段表达式路径统一使用 `columnRef`、`syntaxName`、`syntaxIdent` 构造。MySQL 的 `ColumnName` 是独立规则，返回 `QualifiedName` 供 DDL 等调用方使用；PG `ColumnRef` 还支持目录限定、下标和通配符。两仓保留各自产生式。
- PG `ColumnNameExpr.Name` 现在是共享 `QualifiedName`；处理列引用的访问器应访问 `ColumnNameExpr`，不再等待它产生 `ColumnName` 子节点。DDL 的 `ColumnName.Name` 改为 `string`；本批不移除其他节点的 `CIStr`。既有 `Format` 和 `COMMENT ON COLUMN` 输出不变。
- `TestParseSharedColumnRefCore` 检查单段和含引号内点号的多段名称、原文和跨度，也检查不完整限定名被拒绝。PG 还覆盖目录限定和继续走独立节点的长后缀、下标及通配符。
- E1a 之前／之后：PG `5a0a726` 的解析表不变（3,012 个产生式、6,204 个状态）；生成文件为 1,238,265 字节。MySQL `3a0ed33` 的生成文件不变（894,175 字节）。
- PG 交错运行 `BenchmarkParse` 各 12 次：小／中／宽查询的耗时无显著变化；每次解析的分配字节分别减少 3.0%、6.4%、18.5%。

## E1a 后插入：SQL 错误包合并

- PG 调用方由 `pgparser/sqlerr` 改用 `pgparser/postgres`；类到错误码的映射、SQLSTATE、诊断信息及原因链不变。MySQL 已在 `1424c86` 迁移至 `myparser/mysql`，本批无需修改。两仓 grammar 与生成表不变。

## E1b 字面量和参数核心

- PG `AExprConst` 的普通常量与 MySQL `Literal` 共享 `BaseLiteralExpr`；布尔常量以及浮点令牌的构造动作同形。`Raw` 保留原文，`Kind`／`Value` 维持旧行为。PG 的类型前缀常量仍走类型转换；MySQL 保留字符集引导符与引号样式。`NULL` 的值字段及小数字面量的分类也维持各自现状。
- 两仓均使用 `BaseParameterExpr` 和 `NewParamMarkerExprAt`。PG 下标表达式中的参数只记录参数令牌跨度，索引由 `$n` 指定；MySQL `?` 按出现顺序编号。共享测试检查字面量原文、值与跨度，并覆盖参数以及两仓的拒绝边界。
- PG 生成文件更新至 1,238,597 字节，解析表不变。交错运行 `BenchmarkParse` 各 12 次，耗时无显著变化，分配量和分配次数不变；MySQL 生成文件不变。

## E1c 列引用规则命名

- MySQL `SimpleIdent` 的所有分支都构造 `*ast.ColumnRef`，因此规则头、`%type` 与调用方同步改名为 `ColumnRef`。保留 PG 的目录限定和下标，以及 MySQL 的前导点和关键字分支；没有合并名称值规则 `ColumnName`。
- PG `e9f810a` 和 MySQL `4293f33` 的生成文件逐字节不变（分别为 1,238,597 和 894,175 字节）；MySQL 87,549 条 MTR 逐用例结果一致，PG 语料无新增异常。

## E2a 二元运算构造

- PG `AExpr`／`BExpr` 与 MySQL `Expression`／`BoolPri`／`BitExpr` 的普通二元运算统一调用 `binaryExpr(op, left, right, yyVAL.span)`。PG 保留 AST 上的方言 `Op`，MySQL 仍写入 `BaseBinaryExpr.Op`。PG 自定义运算符、MySQL `MOD()` 原有跨度行为和 SQL mode 不动。
- `TestParseSharedBinaryExpressionCore` 核对算术、比较、`AND`／`OR` 的优先级、左结合、括号 AST、各节点的源码跨度，以及缺少右操作数时的拒绝行为。
- PG `364af20` → E2a 生成文件从 1,238,597 变为 1,238,493 字节；MySQL `b59f8e6` → E2a 从 894,175 变为 892,356 字节。双仓生成表逐字节不变；SR／RR 仍为 418／37 和 0／0。变化仅来自构造动作，不是解析表缩小。
- 双仓现有查询基准前后各交错运行 12 次；耗时和每次分配字节均无显著变化，分配次数不变。

## E2b IN / BETWEEN / NULL 谓词

- 两仓普通 `IN` 列表及子查询都经 `patternInExpr` 构造；PG `AExpr` 的普通与否定 `BETWEEN`、MySQL `PredicateExpr` 的 `BetweenOrNotOp` 经 `betweenExpr` 构造；普通 `IS [NOT] NULL` 都经 `isNullExpr` 构造。helper 同时赋上产生式完整跨度。
- PG `BETWEEN SYMMETRIC` 仍是带独立 `Symmetric` 字段的方言分支；PG `IS UNKNOWN` 与 MySQL `IS UNKNOWN` 既有 AST 映射不同，均不移动。无文法产生式、优先级或接受范围变化。
- 双仓 `TestParseSharedPredicateExpressionCore` 核验谓词节点、否定标记、列表／子查询槽位、谓词与子表达式源码跨度，并拒绝空 `IN`、不完整 `BETWEEN`／`IS`。
- PG `deac12c` → E2b 生成文件由 1,238,493 变为 1,238,485 字节；MySQL `75ee3e9` → E2b 由 892,356 变为 892,126 字节。双仓解析表逐字节不变，PG 冲突 418／37、MySQL 0／0；改变量来自动作代码。
- PG 40,644 条 testdata 与 MySQL 87,549 条 MTR 结果和基线一致。最终源码交错基准各 12 次，耗时无显著变化，分配字节及分配次数不变。

## E2c IS TRUE / FALSE 谓词

- PG `AExpr` 的 TRUE／FALSE 四种形式与 MySQL `Expression` 中对应的 `IsOrNotOp` 分支统一调用 `isTruthExpr(expr, not, truth, span)`；四种形式都检查 AST 标志与完整源码跨度。
- `UNKNOWN` 仍按旧动作映射：PG 使用 `IsTruthExpr`，MySQL 使用 `IsNullExpr`。方言用例分别固定其字段和值，未改写成对方仓库的表达。
- PG `d0de4e8` → E2c 生成文件由 1,238,485 变为 1,238,488 字节；MySQL `bd89b62` → E2c 由 892,126 变为 892,024 字节。解析表逐字节不变，PG 冲突 418／37、MySQL 0／0；双仓语料结果与 E2b 基线一致。交错基准各 12 次，耗时无显著变化，分配字节及分配次数不变。

## E2d LIKE 谓词

- 双仓增加普通 `LIKE`／`NOT LIKE` 的节点与跨度护栏；PG 另测 `NOT ILIKE ... ESCAPE`，MySQL 另测单字符 ESCAPE，既有 MySQL 表达式 ESCAPE 回归测试仍覆盖 `EscapeExpr`。
- 尝试把 PG 与 MySQL 的共同节点构造提取为 `patternLikeExpr`。固定 CPU、交错 12 次的 PG 基准显示耗时增加 3.6%–8.3%，分配不变；MySQL 未显示显著变化。PG 生成解析器已经统一补 AST span，helper 方案被撤回以避免热路径回退；grammar、helper 和生成文件现恢复基线。
- E2d 最终只保留双仓测试与本记录，不改变解析行为；生成文件逐字节不变。撤回后固定 CPU、每项 1 秒的交错基准各 8 次，双仓耗时无显著变化，分配字节及次数不变。双仓 build、vet、race、lint、tidy、生成检查和语料核对通过；PG 40,644 条无新增异常，MySQL 87,549 条 MTR 逐用例一致。

## E2e PatternRegexp 核心

- PG `SIMILAR TO`（含 `NOT`／`ESCAPE`）和 MySQL `REGEXP/RLIKE`（含 `NOT`）均构造 `PatternRegexpExpr`，双仓新增测试锁定核心节点、Not 标记及整个谓词源码跨度。运算符语法和恢复文本仍保持方言差异。
- 只增加测试与批次记录，不改 grammar、AST 或生成文件；双仓 build、vet、race、lint、tidy、生成检查通过，PG 40,644 条无新增异常，MySQL 87,549 条 MTR 逐用例与基线一致。

## E2f EXISTS 子查询

- 双仓正向 `EXISTS (SELECT ...)` 均产生 `ExistsSubqueryExpr`，其 Sel 为 `SubqueryExpr`；共享测试检查节点和完整表达式跨度。
- 否定形式保留既有 AST 差异：PG `NOT EXISTS` 使用 `UnaryOperationExpr{Op: Not}` 包裹 `ExistsSubqueryExpr`；MySQL 将 `Not` 写入该节点。测试分别固定两种形状及覆盖完整 SQL 的跨度，不做不必要的 grammar 动作改写。
- 双仓 build、vet、race、lint、tidy、生成检查通过；PG 40,644 条语料无新增异常，MySQL 87,549 条 MTR 逐用例与基线一致。

## E2g ANY / ALL 比较子查询

- 双仓以 `= ANY (SELECT ...)` 与 `<> ALL (SELECT ...)` 核对共享 `CompareSubqueryExpr` 的比较 opcode、`All` 标志、右侧 `SubqueryExpr` 和完整 SQL span。PG `OpName` 方言扩展不变。
- 仅增加测试与批次记录；双仓 build、vet、race、lint、tidy、生成检查通过，PG 40,644 条语料无新增异常，MySQL 87,549 条 MTR 逐用例与基线一致。

## E2h ROW 构造与 VALUES 行

- 双仓 `ROW(1, 2)` 与隐式 `(3, 4)` 均构造 `RowExpr`，共享 `BaseRowExpr.Items` 和完整表达式跨度；PG 的 `ExplicitRow`／`ImplicitRow` 与 MySQL 的 `SimpleExpr` 规则保留各自语法组织。
- `VALUES (1, 2), (3, 4)` 的行数据和恢复文本作为共同形式固定；PG 继续拒绝 `VALUES ROW(...)`，MySQL 继续支持 `VALUES ROW(...)` 并通过 `OmitRowKeyword` 区分两种行拼写，同时拒绝混用 `ROW` 与括号行。
- 仅增加双仓测试与本记录，不改 AST、grammar 或生成文件；双仓 build、vet、race、lint、tidy、生成检查通过，PG 40,644 条语料无新增异常，MySQL 87,549 条 MTR 逐用例与基线一致。
