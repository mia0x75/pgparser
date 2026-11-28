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
- [ ] P2-03a: pin PostgreSQL LR conflict signatures and winners, not only the 416/37 totals.
- [ ] P2-03b: make astgen reject unsupported child-bearing embedded fields, with a generator regression test.
- [ ] P2-03c: select one PostgreSQL AST/Base integration at a time and validate AST, Span, Restore, and corpus behavior.
- [ ] P2-04b: run the MySQL MTR corpus in an authorized environment and retain version, coverage, decisions, and per-case differences.
- [ ] P3-01a: review `My RequireList` reversed storage/restore order; preserve compatibility unless consumers and tests permit migration.
- [ ] P3-01b: benchmark parser stack/boxing candidates before changing representation.
- [ ] P3-01c: define a compatible non-panicking behavior for unknown public PostgreSQL opcodes and test `String` and output paths.

## PG TODO Inventory

Each row is a separate follow-up task from the report's 34 production TODO comments. Remove or rewrite the source TODO only after the behavior has a test and an explicit support decision.

- [ ] DDL-1: multiple schema changes (`ast/ddl.go`, `AlterTableStmt`).
- [x] DDL-2: unsupported field case now returns `ErrUnsupportedRestore` (`ast/ddl.go`, `STATS_PERSISTENT`).
- [x] DDL-3: unsupported action case now returns `ErrUnsupportedRestore` (`ast/ddl.go`, `PACK_KEYS`).
- [ ] DDL-4: additional alter-table actions (`ast/ddl.go`, `AlterTableSpec`).
- [ ] DML-1: classify the old column/user-variable model (`ast/dml.go`, `ColumnNameOrUserVar`).
- [ ] DML-2: classify the first legacy display placeholder (`ast/dml.go`).
- [ ] DML-3: classify `FieldItem`'s legacy model (`ast/dml.go`).
- [ ] DML-4: classify the remaining legacy DML placeholder (`ast/dml.go`).
- [ ] Account-1: auth plugin support decision (`ast/misc.go`, `CreateUserStmt`).
- [ ] Account-2: `SET PASSWORD` representation (`ast/misc.go`).
- [ ] Account-3: `SetRoleStmt` public behavior and Restore support.
- [ ] Account-4: `SetDefaultRoleStmt` public behavior and Restore support.
- [ ] Account-5: first account/TLS placeholder (`ast/misc.go`).
- [x] Account-6: `TslOption` Restore now renders `SUBJECT` correctly and rejects nil receivers.
- [ ] Account-7: second account/resource placeholder (`ast/misc.go`).
- [ ] Account-8: `ResourceOption` support decision and validation.
- [ ] Account-9: third account/password placeholder (`ast/misc.go`).
- [x] Account-10: password expiration interval renders the interval and rejects non-positive values.
- [ ] Account-11: `LockTableStmt` support boundary and Restore behavior.
- [ ] Account-12: `UnlockTableStmt` support boundary and Restore behavior.
- [ ] Account-13: `PurgeStmt` support boundary and Restore behavior.
- [ ] Auth-1: escape usernames in identity display (`auth/auth.go`).
- [ ] Auth-2: escape hostnames in identity display (`auth/auth.go`).
- [ ] Auth-3: audit the third identity formatting path (`auth/auth.go`).
- [ ] Model-1: remove `OldSchemaID` compatibility only with migration evidence (`model/model.go`).
- [ ] Model-2: remove the paired legacy schema-ID path (`model/model.go`).
- [ ] Types-1: verify UTF-8 field length vs maximum encoded bytes (`types/field_type.go`).
- [ ] Types-2: pin JSON binary-function error code 1582 (`types/json/binary_functions.go`).
- [ ] Types-3: pin JSON binary-function error code 3149, first path.
- [ ] Types-4: pin JSON binary-function error code 3153.
- [ ] Types-5: pin JSON binary-function error code 3149, second path.
- [ ] Types-6: verify CEILING decimal behavior (`types/pgdecimal.go`).
- [ ] Types-7: specify PostgreSQL time-zone behavior (`types/pgtime.go`).
- [ ] Types-8: decide whether `NewTime` must establish default FSP (`types/time.go`).

## Grammar Consistency

- [ ] Align grammar section ordering and token/type declarations only when a generated-parser baseline proves the change is non-semantic.
- [ ] Add common-rule semantic fixtures for matching productions; preserve dialect-specific keyword classes, precedence, AST shape, and Restore behavior.
- [x] Normalize MySQL empty-production comments and split alternative markers; parser generation remains unchanged.
- [ ] Keep MySQL conflicts at 0/0; keep PostgreSQL conflict signatures stable unless an intentional semantic batch records the changed winner and corpus impact.
- [ ] Leave parserkit unchanged unless a separately reviewed shared-contract defect requires it.

## External Verification

- [x] PostgreSQL corpus baseline rerun: 40,644 statements and 141 known failures matched.
- [ ] MySQL MTR is not claimed complete until the authorized local corpus environment is available.
- [ ] Database-backed verification remains unrun; no database connection is configured for this task.
