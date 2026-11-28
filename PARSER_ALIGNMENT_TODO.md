# Parser Alignment TODO

Source: `/tmp/opencode/three-sql-audit.uFdtjN/三仓像素级审计与文法对齐报告.md`.

## Closed

- [x] P1-01: preserve `CREATE EXTENSION` name, options, and restore semantics.
- [x] P1-02: validate Visitor replacements before skip, walk `RawStmt` and embedded value children, and reject nil collection entries.
- [x] P1-03: reject nil Restore inputs and invalid required structure without panicking.
- [x] P1-04: reconcile shared and PostgreSQL window fields after Visitor callbacks.
- [x] Phase 4a formatting: align alternative and terminator indentation; follow up with consistent empty markers. MySQL parser output is unchanged; PostgreSQL has only the P1-01 semantic action change plus the recorded Span fix.
- [x] P2-02: MySQL empty `ColumnDef` now errors; the 36 audited typed-nil roots return `(nil, false)`.

## Active

- [x] P2-01: preserve exact spans for explicit PostgreSQL window bounds and parenthesized expressions; keep synthetic bounds invalid.
- [x] P2-04a: rerun PostgreSQL's 40,644-statement corpus and compare the recorded baseline after semantic parser changes; counts and all 141 known failures match.
- [x] P2-03a: pin the PostgreSQL conflict signature in `make generate`; current baseline is 417/24 after factoring duplicate `ColId`/`ColLabel` token reductions. The sole added shift/reduce is in the same dotted-name state and retains shift preference.
- [x] P2-03b: astgen rejects unsupported child-bearing embedded fields, with generator regression coverage.
- [x] P2-03c: validate `CREATE EXTENSION` AST type, exact statement Span, Restore round-trip, and the recorded corpus baseline.
- [x] P2-04b: run the extracted MySQL/MariaDB MTR SQL corpus with `cmd/mtrcheck`; retain source revisions, coverage and per-case results. This is an offline parser check, not `mysql-test-run.pl`.
- [x] P3-01a: preserve MySQL `RequireList` reverse storage because Restore iterates in reverse; a regression test pins stored and restored source order.
- [x] P3-01b: benchmarked long parser inputs before considering stack/boxing changes; 5 runs on amd64/AMD Ryzen 7 5700U, `-benchtime=300ms`: unary chain 32,768 operators 28.6-45.4 ms/op, 66.06 MB/op, 26-29 allocs/op; UNION chain 8,000 terms 40.1-49.7 ms/op, 7.31 MB/op, 88,003 allocs/op. No representation change is justified by this baseline.
- [x] P3-01c: unknown public opcodes return `Op(<n>)` from `String` and output paths; `Restore` continues to return an error.

## PG TODO Inventory

Each row is a separate follow-up task from the report's 34 production TODO comments. Remove or rewrite the source TODO only after the behavior has a test and an explicit support decision.

- [x] DDL-1: `AlterTableStmt` restores multiple schema changes; parser/Restore regression coverage added.
- [x] DDL-2: unsupported field case now returns `ErrUnsupportedRestore` (`ast/ddl.go`, `STATS_PERSISTENT`).
- [x] DDL-3: unsupported action case now returns `ErrUnsupportedRestore` (`ast/ddl.go`, `PACK_KEYS`).
- [x] DDL-4: unknown `AlterTableSpec` actions return `ErrUnsupportedRestore`; regression coverage pins the unsupported boundary.
- [x] DML-1: retain `ColumnNameOrUserVar` as a legacy LOAD DATA shape; do not migrate without consumer evidence.
- [x] DML-2: retain display-position values for source compatibility.
- [x] DML-3: retain `FieldItem` as a legacy LOAD DATA representation.
- [x] DML-4: retain profiling type values for source compatibility.
- [x] Account-1: auth-plugin clauses remain unsupported because the grammar has no production for them.
- [x] Account-2: keep the legacy `VariableAssignment.ExtendValue` payload pending a compatible `SetStmt` migration.
- [x] Account-3: `SetRoleStmt` Restore behavior is covered.
- [x] Account-4: `SetDefaultRoleStmt` Restore behavior is covered.
- [x] Account-5: TLS requirement values remain represented by `TslOption`; Restore behavior is covered by `legacy_restore_test.go`.
- [x] Account-6: `TslOption` Restore now renders `SUBJECT` correctly and rejects nil receivers.
- [x] Account-7: resource limit kinds remain represented by `ResourceOption`.
- [x] Account-8: `ResourceOption` rejects nil receivers, negative counts, and unknown kinds; valid zero is preserved.
- [x] Account-9: password expiration and account lock kinds remain represented by `PasswordOrLockOption`; Restore behavior is covered by `legacy_restore_test.go`.
- [x] Account-10: password expiration interval renders the interval and rejects non-positive values.
- [x] Account-11: `LockTableStmt` is retained but Restore returns `ErrUnsupportedRestore`.
- [x] Account-12: `UnlockTableStmt` is retained but Restore returns `ErrUnsupportedRestore`.
- [x] Account-13: `PurgeStmt` is retained but Restore returns `ErrUnsupportedRestore`.
- [x] Auth-1: identity display escapes usernames.
- [x] Auth-2: identity display escapes hostnames.
- [x] Auth-3: `RoleIdentity.String` is the third identity-display path; it escapes both components and has regression coverage.
- [x] Model-1: retain serialized `OldSchemaID`; legacy metadata selects the allocator schema ID, and no migration evidence permits removing that behavior. A compatibility regression test covers old JSON metadata.
- [x] Model-2: retain `GetDBID` as the paired path for legacy `OldSchemaID`; its fallback and compatibility behavior are tested.
- [x] Types-1: string `Flen` is the encoded byte length of the actual value, not the maximum storage width; a multibyte UTF-8 test pins this contract.
- [x] Types-2: pin JSON binary-function error code 1582 and SQLSTATE `42000`.
- [x] Types-3: pin JSON binary-function error code 3149 for modification wildcard paths.
- [x] Types-4: pin JSON binary-function error code 3153 for empty removal paths.
- [x] Types-5: pin JSON binary-function error code 3149 for removal wildcard paths.
- [x] Types-6: implement and test exact decimal CEILING for positive/negative values, positive/negative scales and in-place rounding; precision-limit behavior returns the existing `ErrTruncated` (`types/decimal.go`).
- [x] Types-7: `PgTime.Weekday` uses UTC calendar fields because `PgTime` represents a timezone-free wall clock; regression coverage pins the result.
- [x] Types-8: keep FSP explicit; zero-value `Time` and `CurrentTime` use precision zero, covered by tests. No `NewTime` constructor is needed.

## Grammar Consistency

- [x] 统一两仓 grammar 的声明分区、产生式格式、动作缩进、显式结束符与空产生式标记；93 个同名产生式的相对顺序及 19 个模块锚点的顺序全部对齐。
- [x] 统一 `Start → StatementList → Statement` 的类型化语句列表流程：`appendStatement` 完成单条语句收尾和累积，`setParseResult` 仅在 `Start` 一次性交付结果；空输入仍返回 nil，错误批次不返回部分结果。
- [x] MySQL 保留原有语句 Text、精确 Span、`?` 编号与逐语句清零语义；PostgreSQL 在相同收尾阶段保留原有 Span 和显式 `$n` 编号，并补齐顶层语句 Text。未给 PostgreSQL 增加无用途的隐式参数计数器。
- [x] 双仓统一提供 `make generate` 与 `make generate-check`；检查先保存生成器可能改写的文件摘要再重新生成并比较，PostgreSQL 同时覆盖普通 AST 文件中的 visitor 防护，MySQL 保留 `make parser` 作为兼容入口。
- [x] 逐规则核对重构前后的右部与动作签名：MySQL 738 条规则、2,890 个分支，PostgreSQL 698 条规则、3,008 个分支；除顶层三个规则外无变化。公共分发使用 goyacc 的默认首项传递，不再重复书写无作用的转发动作。
- [x] 删除 PostgreSQL `%union` 中无声明引用或动作引用的 `ivalp`、`setop`、`colnames`，缩减语义栈元素；生成后的冲突数、冲突签名和语料结果均保持不变。
- [x] Add a matching common SELECT semantic fixture in both parser repositories; dialect-specific AST and Restore behavior remain local.
- [x] Normalize MySQL empty-production comments and split alternative markers; parser generation remains unchanged.
- [x] Keep MySQL conflicts at 0/0; PostgreSQL conflict signature is pinned. Left-factor the shared identifier classes in `ColLabel`, reducing reduce/reduce conflicts from 37 to 24; the parser corpus must remain unchanged.
- [x] Pin both repositories to `golang.org/x/tools/cmd/goyacc@v0.52.0` so grammar generation uses the same reproducible parser-generator version.
- [x] Leave parserkit unchanged; no shared-contract defect was required for this batch.

## External Verification

- [x] PostgreSQL corpus baseline rerun: 40,644 statements and 141 known failures matched.
- [x] MySQL/MariaDB 离线 MTR SQL 语料复核：87,549 条语句，86,872 条成功解析、677 条拒绝、0 跳过、0 预期不匹配、0 panic；全部逐例 JSON 结果与重构前基线相同。最终报告：`/tmp/opencode/myparser-mtr-grammar-flow-delivery-20261128/`（未运行数据库服务端 MTR）。
- [x] 2026-11-28 grammar 重构最终门禁：双仓 `make generate-check`、`go build ./...`、`go vet ./...`、`go test -race ./...`、`golangci-lint run ./...` 均通过且静态分析 0 问题；PostgreSQL 40,644 条语料及全部 141 个已知失败匹配，冲突数和签名保持 417/24 基线。
- [x] 2026-11-28 最终基准复核：同一 Go 工具链、linux/amd64、AMD Ryzen 7 5700U、固定 CPU 8、`-cpu=1`、每项 250ms，预热后交错运行新旧版本各 12 轮。四项解析耗时均无显著变化；MySQL 简单解析多 384 B/op（+8.86%，类型化语句列表字段的固定成本），分配次数不变；PostgreSQL 32,768 层一元链由约 63 MiB/op 降至 60 MiB/op（-4.76%），分配次数由 26 降至 24。原始结果和 benchstat：`/tmp/opencode/grammar-flow-bench-interleaved-20261128/`。
- [x] Database-backed verification was attempted but no local MySQL/PostgreSQL server is listening; no server-backed behavior is claimed.
