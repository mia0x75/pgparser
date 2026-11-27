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
| 表达式 / 类型 | `AExpr`、`BExpr`、`CExpr`、`ColumnRef`、`AExprConst`、`FuncApplication`、`Typename` | `Expression`、`PredicateExpr`、`BitExpr`、`SimpleExpr`、`Literal`、`FunctionCallGeneric`、`Type` | Expr、Literal、TypeName、FunctionCall | 语义映射而非规则合并；E1–E3 分开推进。 |

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
