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
- [x] P2-03a: pin the PostgreSQL conflict signature in `make generate`; count remains 416/37.
- [x] P2-03b: astgen rejects unsupported child-bearing embedded fields, with generator regression coverage.
- [x] P2-03c: validate `CREATE EXTENSION` AST type, exact statement Span, Restore round-trip, and the recorded corpus baseline.
- [ ] P2-04b: run the MySQL MTR corpus in an authorized environment and retain version, coverage, decisions, and per-case differences.
- [ ] P3-01a: review `My RequireList` reversed storage/restore order; preserve compatibility unless consumers and tests permit migration.
- [ ] P3-01b: benchmark parser stack/boxing candidates before changing representation.
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
- [x] Keep MySQL conflicts at 0/0; PostgreSQL conflict signature is pinned and the recorded corpus baseline is unchanged.
- [x] Leave parserkit unchanged; no shared-contract defect was required for this batch.

## External Verification

- [x] PostgreSQL corpus baseline rerun: 40,644 statements and 141 known failures matched.
- [ ] MySQL MTR is not claimed complete until the authorized local corpus environment is available.
- [ ] Database-backed verification remains unrun; no database connection is configured for this task.
