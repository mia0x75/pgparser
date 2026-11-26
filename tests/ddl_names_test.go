package parser_test

import (
	"strings"
	"testing"

	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/pgparser/ast"
)

func TestDDLNameSpelling(t *testing.T) {
	for _, tc := range []struct {
		sql string
		get func(ast.StmtNode) lex.Ident
		raw string
	}{
		{`CREATE DATABASE "Db"`, func(n ast.StmtNode) lex.Ident { return n.(*ast.CreateDatabaseStmt).Name }, `"Db"`},
		{`CREATE SCHEMA "Schema"`, func(n ast.StmtNode) lex.Ident { return n.(*ast.CreateSchemaStmt).Name }, `"Schema"`},
		{`ALTER DATABASE db RENAME TO "New"`, func(n ast.StmtNode) lex.Ident { return n.(*ast.AlterDatabaseStmt).RenameTo }, `"New"`},
		{`ALTER DATABASE db SET TABLESPACE "Space"`, func(n ast.StmtNode) lex.Ident { return n.(*ast.AlterDatabaseStmt).Tablespace }, `"Space"`},
		{`CREATE INDEX "Index" ON t USING "btree" (c)`, func(n ast.StmtNode) lex.Ident { return n.(*ast.CreateIndexStmt).Using }, `"btree"`},
		{`CREATE INDEX "Index" ON t (c) TABLESPACE "Space"`, func(n ast.StmtNode) lex.Ident { return n.(*ast.CreateIndexStmt).Tablespace }, `"Space"`},
		{`CREATE SCHEMA AUTHORIZATION "Role"`, func(n ast.StmtNode) lex.Ident { return n.(*ast.CreateSchemaStmt).Authorization.Rolename }, `"Role"`},
		{`ALTER TABLESPACE ts RENAME TO "Space"`, func(n ast.StmtNode) lex.Ident { return n.(*ast.AlterTablespaceStmt).RenameTo }, `"Space"`},
		{`PREPARE "Plan" AS SELECT 1`, func(n ast.StmtNode) lex.Ident { return n.(*ast.PrepareStmt).Name }, `"Plan"`},
		{`EXECUTE "Plan"`, func(n ast.StmtNode) lex.Ident { return n.(*ast.ExecuteStmt).Name }, `"Plan"`},
		{`DEALLOCATE "ALL"`, func(n ast.StmtNode) lex.Ident { return n.(*ast.DeallocateStmt).Name }, `"ALL"`},
	} {
		t.Run(tc.sql, func(t *testing.T) {
			n := mustParseOne(t, tc.sql)
			ident := tc.get(n)
			start := loc.ByteOffset(strings.Index(tc.sql, tc.raw))
			if ident.Name != strings.Trim(tc.raw, `"`) || ident.Raw != tc.raw || ident.Quote != lex.QuoteDouble || ident.Span != (loc.Span{Start: start, End: start + loc.ByteOffset(len(tc.raw))}) {
				t.Fatalf("identifier = %+v", ident)
			}
			restored := restoreString(t, n)
			if again := restoreString(t, mustParseOne(t, restored)); again != restored {
				t.Fatalf("unstable restore: %q -> %q", restored, again)
			}
		})
	}
}

func TestConstraintAndViewNameSpelling(t *testing.T) {
	const sql = `CREATE TABLE t (CONSTRAINT "U" UNIQUE ("c") INCLUDE ("d") USING INDEX TABLESPACE "S")`
	n := mustParseOne(t, sql).(*ast.CreateTableStmt)
	constraint := n.Constraints[0]
	if constraint.Name.Raw != `"U"` || len(constraint.Including) != 1 || constraint.Including[0].Raw != `"d"` || constraint.IndexSpace.Raw != `"S"` {
		t.Fatalf("constraint names = %+v", constraint)
	}
	if got := restoreString(t, n); !strings.Contains(got, `CONSTRAINT "U" UNIQUE ("c") INCLUDE ("d") USING INDEX TABLESPACE "S"`) {
		t.Fatalf("restore = %q", got)
	}

	view := mustParseOne(t, `CREATE VIEW v ("C") AS SELECT 1`).(*ast.CreateViewStmt)
	if len(view.Cols) != 1 || view.Cols[0].Raw != `"C"` || view.Cols[0].Span.Start != loc.ByteOffset(strings.Index(`CREATE VIEW v ("C") AS SELECT 1`, `"C"`)) {
		t.Fatalf("view cols = %+v", view.Cols)
	}
}

func TestTablespaceNameSpelling(t *testing.T) {
	for _, tc := range []struct {
		sql  string
		get  func(ast.StmtNode) lex.Ident
		want string
	}{
		{`CREATE TABLESPACE "Ts" LOCATION '/data'`, func(n ast.StmtNode) lex.Ident { return n.(*ast.CreateTablespaceStmt).Name }, `"Ts"`},
		{`ALTER TABLESPACE ts RENAME TO "Ts"`, func(n ast.StmtNode) lex.Ident { return n.(*ast.AlterTablespaceStmt).RenameTo }, `"Ts"`},
		{`DROP TABLESPACE "Ts"`, func(n ast.StmtNode) lex.Ident { return n.(*ast.DropTablespaceStmt).Name }, `"Ts"`},
		{`DROP DATABASE "Db"`, func(n ast.StmtNode) lex.Ident { return n.(*ast.DropdbStmt).Dbname }, `"Db"`},
		{`ALTER DATABASE "Db" SET work_mem TO 1`, func(n ast.StmtNode) lex.Ident { return n.(*ast.AlterDatabaseSetStmt).Dbname }, `"Db"`},
	} {
		t.Run(tc.sql, func(t *testing.T) {
			ident := tc.get(mustParseOne(t, tc.sql))
			if ident.Raw != tc.want {
				t.Fatalf("identifier = %+v, want %q", ident, tc.want)
			}
		})
	}
}

func TestDropIndexQualifiedName(t *testing.T) {
	const sql = `DROP INDEX IF EXISTS "Schema"."Index"`
	n := mustParseOne(t, sql).(*ast.DropIndexStmt)
	if len(n.IndexName.Parts) != 2 || n.IndexName.Parts[0].Raw != `"Schema"` || n.IndexName.Parts[1].Raw != `"Index"` {
		t.Fatalf("index name = %+v", n.IndexName)
	}
	if got := restoreString(t, n); got != sql {
		t.Fatalf("restore = %q", got)
	}
}
