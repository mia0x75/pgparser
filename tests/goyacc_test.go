package parser_test

import (
	"reflect"
	"testing"

	pgparser "github.com/mia0x75/pgparser"
	"github.com/mia0x75/pgparser/ast"
	_ "github.com/mia0x75/pgparser/driver"
)

func mustParseOne(t *testing.T, sql string) ast.StmtNode {
	t.Helper()
	stmt, err := pgparser.New().ParseOneStmt(sql)
	if err != nil {
		t.Fatalf("ParseOneStmt(%q) error: %v", sql, err)
	}
	return stmt
}

func mustParseSelect(t *testing.T, sql string) *ast.SelectStmt {
	t.Helper()
	stmt, ok := mustParseOne(t, sql).(*ast.SelectStmt)
	if !ok {
		t.Fatalf("expected *ast.SelectStmt, got %T", mustParseOne(t, sql))
	}
	return stmt
}

func leftmostTableSource(n ast.ResultSetNode) (*ast.TableSource, bool) {
	for {
		switch v := n.(type) {
		case *ast.TableSource:
			return v, true
		case *ast.Join:
			n = v.Left
		default:
			return nil, false
		}
	}
}

// Ported from parser_test.go: TestParseSQLSimpleSelect.
func TestParseSQLSimpleSelect(t *testing.T) {
	st := mustParseSelect(t, "SELECT id, name FROM users WHERE status = $1;")

	if st.Fields == nil || len(st.Fields.Fields) != 2 {
		t.Fatalf("expected 2 columns, got %#v", st.Fields)
	}
	if got := st.Fields.Fields[0].Expr.(*ast.ColumnNameExpr).Name.Name.O; got != "id" {
		t.Fatalf("column 1 = %q, want id", got)
	}
	if got := st.Fields.Fields[1].Expr.(*ast.ColumnNameExpr).Name.Name.O; got != "name" {
		t.Fatalf("column 2 = %q, want name", got)
	}
	if st.From == nil || st.From.TableRefs == nil {
		t.Fatal("expected FROM clause")
	}
	left, ok := st.From.TableRefs.Left.(*ast.TableSource)
	if !ok {
		t.Fatalf("expected TableSource, got %T", st.From.TableRefs.Left)
	}
	if got := left.Source.(*ast.TableName).Name.O; got != "users" {
		t.Fatalf("table = %q, want users", got)
	}
	if st.Where == nil {
		t.Fatal("expected WHERE clause")
	}
}

// Ported from parser_test.go: TestParseSQLJoinWithAlias.
func TestParseSQLJoinWithAlias(t *testing.T) {
	sql := `
SELECT o.id, c.name
FROM orders o
JOIN customers c ON o.customer_id = c.id
WHERE o.created_at > $1 AND c.active = true;`
	st := mustParseSelect(t, sql)

	if st.From == nil || st.From.TableRefs == nil {
		t.Fatal("expected FROM clause")
	}
	left, ok := leftmostTableSource(st.From.TableRefs)
	if !ok {
		t.Fatalf("expected left TableSource, got %T", st.From.TableRefs.Left)
	}
	if got := left.Source.(*ast.TableName).Name.O; got != "orders" {
		t.Fatalf("left table = %q, want orders", got)
	}
	if got := left.AsName.O; got != "o" {
		t.Fatalf("left alias = %q, want o", got)
	}
	right, ok := st.From.TableRefs.Right.(*ast.TableSource)
	if !ok {
		t.Fatalf("expected right TableSource, got %T", st.From.TableRefs.Right)
	}
	if got := right.Source.(*ast.TableName).Name.O; got != "customers" {
		t.Fatalf("right table = %q, want customers", got)
	}
	if got := right.AsName.O; got != "c" {
		t.Fatalf("right alias = %q, want c", got)
	}
	if st.From.TableRefs.On == nil {
		t.Fatal("expected ON condition")
	}
	if st.Where == nil {
		t.Fatal("expected WHERE clause")
	}
}

// Ported from parser_test.go: TestParseSQLGroupOrderLimit.
func TestParseSQLGroupOrderLimit(t *testing.T) {
	sql := `
SELECT tenant, COUNT(*) AS cnt
FROM events
WHERE occurred_at >= $1
GROUP BY tenant
HAVING COUNT(*) > 5
ORDER BY cnt DESC
LIMIT 10 OFFSET 2;`
	st := mustParseSelect(t, sql)

	if st.GroupBy == nil || len(st.GroupBy.Items) != 1 {
		t.Fatalf("expected 1 GROUP BY item, got %#v", st.GroupBy)
	}
	if st.Having == nil {
		t.Fatal("expected HAVING clause")
	}
	if st.OrderBy == nil || len(st.OrderBy.Items) != 1 {
		t.Fatalf("expected 1 ORDER BY item, got %#v", st.OrderBy)
	}
	if !st.OrderBy.Items[0].Desc {
		t.Fatal("expected ORDER BY ... DESC")
	}
	if st.Limit == nil {
		t.Fatal("expected LIMIT clause")
	}
	if st.Limit.Count == nil || st.Limit.Offset == nil {
		t.Fatalf("expected LIMIT and OFFSET, got %#v", st.Limit)
	}
}

// Ported from parser_test.go multi-statement behavior.
func TestParseSQLMultiStatement(t *testing.T) {
	stmts, _, err := pgparser.New().Parse("SELECT 1; SELECT 2;")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(stmts) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(stmts))
	}
	if _, ok := stmts[0].(*ast.SelectStmt); !ok {
		t.Fatalf("statement 0: expected *ast.SelectStmt, got %T", stmts[0])
	}
}

// Ported from parser_ir_dml_test.go: insert/update/delete statements.
func TestParseSQLDML(t *testing.T) {
	ins, ok := mustParseOne(t, "INSERT INTO t (a, b) VALUES (1, 2), (3, 4);").(*ast.InsertStmt)
	if !ok {
		t.Fatal("expected *ast.InsertStmt")
	}
	if len(ins.Columns) != 2 {
		t.Fatalf("expected 2 insert columns, got %d", len(ins.Columns))
	}
	if len(ins.Lists) != 2 {
		t.Fatalf("expected 2 value rows, got %d", len(ins.Lists))
	}

	upd, ok := mustParseOne(t, "UPDATE t SET a = 1, b = 2 WHERE c = 3;").(*ast.UpdateStmt)
	if !ok {
		t.Fatal("expected *ast.UpdateStmt")
	}
	if len(upd.List) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(upd.List))
	}
	if upd.Where == nil {
		t.Fatal("expected UPDATE WHERE clause")
	}

	del, ok := mustParseOne(t, "DELETE FROM t WHERE a = 1;").(*ast.DeleteStmt)
	if !ok {
		t.Fatal("expected *ast.DeleteStmt")
	}
	if del.Where == nil {
		t.Fatal("expected DELETE WHERE clause")
	}
}

// Ported from parser_ir_select_test.go: expression coverage.
func TestParseSQLExpressions(t *testing.T) {
	tests := []struct {
		name  string
		sql   string
		check func(t *testing.T, st *ast.SelectStmt)
	}{
		{
			name: "in list",
			sql:  "SELECT a FROM t WHERE a IN (1, 2, 3);",
			check: func(t *testing.T, st *ast.SelectStmt) {
				if _, ok := st.Where.(*ast.PatternInExpr); !ok {
					t.Fatalf("expected *ast.PatternInExpr, got %T", st.Where)
				}
			},
		},
		{
			name: "between",
			sql:  "SELECT a FROM t WHERE a BETWEEN 1 AND 2;",
			check: func(t *testing.T, st *ast.SelectStmt) {
				if _, ok := st.Where.(*ast.BetweenExpr); !ok {
					t.Fatalf("expected *ast.BetweenExpr, got %T", st.Where)
				}
			},
		},
		{
			name: "like",
			sql:  "SELECT a FROM t WHERE a LIKE 'x%';",
			check: func(t *testing.T, st *ast.SelectStmt) {
				if _, ok := st.Where.(*ast.PatternLikeExpr); !ok {
					t.Fatalf("expected *ast.PatternLikeExpr, got %T", st.Where)
				}
			},
		},
		{
			name: "is null",
			sql:  "SELECT a FROM t WHERE a IS NOT NULL;",
			check: func(t *testing.T, st *ast.SelectStmt) {
				e, ok := st.Where.(*ast.IsNullExpr)
				if !ok {
					t.Fatalf("expected *ast.IsNullExpr, got %T", st.Where)
				}
				if !e.Not {
					t.Fatal("expected IS NOT NULL")
				}
			},
		},
		{
			name: "case",
			sql:  "SELECT CASE WHEN a = 1 THEN 2 ELSE 3 END FROM t;",
			check: func(t *testing.T, st *ast.SelectStmt) {
				if _, ok := st.Fields.Fields[0].Expr.(*ast.CaseExpr); !ok {
					t.Fatalf("expected *ast.CaseExpr, got %T", st.Fields.Fields[0].Expr)
				}
			},
		},
		{
			name: "function call",
			sql:  "SELECT COUNT(*) FROM t;",
			check: func(t *testing.T, st *ast.SelectStmt) {
				if _, ok := st.Fields.Fields[0].Expr.(*ast.AggregateFuncExpr); !ok {
					t.Fatalf("expected *ast.AggregateFuncExpr, got %T", st.Fields.Fields[0].Expr)
				}
			},
		},
		{
			name: "qualified column",
			sql:  "SELECT t.a FROM t;",
			check: func(t *testing.T, st *ast.SelectStmt) {
				e, ok := st.Fields.Fields[0].Expr.(*ast.ColumnNameExpr)
				if !ok {
					t.Fatalf("expected *ast.ColumnNameExpr, got %T", st.Fields.Fields[0].Expr)
				}
				if e.Name.Table.O != "t" || e.Name.Name.O != "a" {
					t.Fatalf("got %s.%s, want t.a", e.Name.Table.O, e.Name.Name.O)
				}
			},
		},
		{
			name: "qualified wildcard",
			sql:  "SELECT t.* FROM t;",
			check: func(t *testing.T, st *ast.SelectStmt) {
				w := st.Fields.Fields[0].WildCard
				if w == nil || w.Table.O != "t" {
					t.Fatalf("expected t.*, got %#v", st.Fields.Fields[0])
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.check(t, mustParseSelect(t, tc.sql))
		})
	}
}

// Ported from parser_ir_setops_test.go.
func TestParseSQLSetOperations(t *testing.T) {
	for _, sql := range []string{
		"SELECT 1 UNION SELECT 2;",
		"SELECT 1 UNION ALL SELECT 2;",
		"SELECT 1 INTERSECT SELECT 2;",
		"SELECT 1 EXCEPT SELECT 2;",
	} {
		stmt := mustParseOne(t, sql)
		if _, ok := stmt.(*ast.SetOprStmt); !ok {
			t.Fatalf("%q: expected *ast.SetOprStmt, got %T", sql, stmt)
		}
	}
}

// Ported from parser_ir_placeholder_test.go.
func TestParseSQLParameters(t *testing.T) {
	st := mustParseSelect(t, "SELECT a FROM t WHERE a = $1 AND b = $2;")
	if st.Where == nil {
		t.Fatal("expected WHERE clause")
	}
}

// Ported from parser_test.go: error handling.
func TestParseSQLErrors(t *testing.T) {
	if _, err := pgparser.New().ParseOneStmt("SELECT FROM;"); err == nil {
		t.Fatal("expected syntax error for SELECT FROM;")
	}
	if _, err := pgparser.New().ParseOneStmt("SELECT a, FROM t;"); err == nil {
		t.Fatal("expected syntax error for trailing comma")
	}
}

// Ported from parser_jsonb_compound_tokens_test.go / preprocess_test.go:
// lexer-level checks that remain valid for the goyacc scanner.
func TestScannerDollarQuoteAndComments(t *testing.T) {
	for _, sql := range []string{
		"SELECT $$hello$$;",
		"SELECT $tag$hello$tag$;",
		"SELECT 1; -- trailing comment",
		"/* leading */ SELECT 1;",
	} {
		if _, err := pgparser.New().ParseOneStmt(sql); err != nil {
			t.Fatalf("%q: unexpected error: %v", sql, err)
		}
	}
}

func TestParseSQLEmptyInput(t *testing.T) {
	stmts, _, err := pgparser.New().Parse("")
	if err != nil {
		t.Fatalf("empty input: unexpected error: %v", err)
	}
	if len(stmts) != 0 {
		t.Fatalf("empty input: expected 0 statements, got %d", len(stmts))
	}
}

func TestParseSQLQuotedIdentifier(t *testing.T) {
	st := mustParseSelect(t, `SELECT "MyColumn" FROM t;`)
	e, ok := st.Fields.Fields[0].Expr.(*ast.ColumnNameExpr)
	if !ok {
		t.Fatalf("expected *ast.ColumnNameExpr, got %T", st.Fields.Fields[0].Expr)
	}
	if got := e.Name.Name.O; got != "MyColumn" {
		t.Fatalf("quoted identifier = %q, want MyColumn", got)
	}
}

func TestParseSQLAlterTable(t *testing.T) {
	stmt, ok := mustParseOne(t, "ALTER TABLE s.foo ADD COLUMN bar INT NOT NULL, DROP COLUMN baz;").(*ast.AlterTableStmt)
	if !ok {
		t.Fatalf("expected *ast.AlterTableStmt, got %T", mustParseOne(t, "ALTER TABLE s.foo ADD COLUMN bar INT;"))
	}
	if stmt.Table.Schema.O != "s" || stmt.Table.Name.O != "foo" {
		t.Fatalf("table = %s.%s, want s.foo", stmt.Table.Schema.O, stmt.Table.Name.O)
	}
	if len(stmt.Specs) != 2 {
		t.Fatalf("specs = %d, want 2", len(stmt.Specs))
	}
	if stmt.Specs[0].Tp != ast.AlterTableAddColumns {
		t.Fatalf("spec[0].Tp = %v, want AlterTableAddColumns", stmt.Specs[0].Tp)
	}
	if stmt.Specs[1].Tp != ast.AlterTableDropColumn {
		t.Fatalf("spec[1].Tp = %v, want AlterTableDropColumn", stmt.Specs[1].Tp)
	}
	// RENAME TO is a separate statement in PostgreSQL.
	rename, ok := mustParseOne(t, "ALTER TABLE s.foo RENAME TO qux;").(*ast.RenameStmt)
	if !ok || rename.Relation == nil || rename.Relation.Name.O != "foo" || rename.Newname != "qux" {
		t.Fatalf("unexpected rename: %#v", mustParseOne(t, "ALTER TABLE s.foo RENAME TO qux;"))
	}
}

func TestParseSQLDropAndTruncate(t *testing.T) {
	dropTable, ok := mustParseOne(t, "DROP TABLE IF EXISTS foo, s.bar;").(*ast.DropTableStmt)
	if !ok {
		t.Fatal("expected *ast.DropTableStmt")
	}
	if !dropTable.IfExists || len(dropTable.Tables) != 2 || dropTable.Tables[1].Schema.O != "s" {
		t.Fatalf("unexpected drop table: %#v", dropTable)
	}

	dropView, ok := mustParseOne(t, "DROP VIEW v;").(*ast.DropTableStmt)
	if !ok || !dropView.IsView {
		t.Fatalf("expected view drop, got %#v", dropView)
	}

	dropIndex, ok := mustParseOne(t, "DROP INDEX IF EXISTS s.ix;").(*ast.DropIndexStmt)
	if !ok || !dropIndex.IfExists || dropIndex.IndexName != "ix" {
		t.Fatalf("unexpected drop index: %#v", dropIndex)
	}

	truncate, ok := mustParseOne(t, "TRUNCATE TABLE s.t;").(*ast.TruncateTableStmt)
	if !ok || truncate.Table.Schema.O != "s" || truncate.Table.Name.O != "t" {
		t.Fatalf("unexpected truncate: %#v", truncate)
	}
}

func TestParseSQLCreateIndexAndView(t *testing.T) {
	idx, ok := mustParseOne(t, "CREATE UNIQUE INDEX IF NOT EXISTS ix ON s.t (a, b);").(*ast.CreateIndexStmt)
	if !ok || !idx.Unique || idx.IndexName != "ix" || idx.Table.Schema.O != "s" {
		t.Fatalf("unexpected create index: %#v", idx)
	}
	if len(idx.IndexColNames) != 2 {
		t.Fatalf("index cols = %d, want 2", len(idx.IndexColNames))
	}

	view, ok := mustParseOne(t, "CREATE OR REPLACE VIEW v (a, b) AS SELECT a, b FROM t;").(*ast.CreateViewStmt)
	if !ok || !view.OrReplace || view.ViewName.Name.O != "v" {
		t.Fatalf("unexpected create view: %#v", view)
	}
	if len(view.Cols) != 2 || view.Cols[0].O != "a" {
		t.Fatalf("view cols = %#v, want [a b]", view.Cols)
	}
	if _, ok := view.Select.(*ast.SelectStmt); !ok {
		t.Fatalf("expected select body, got %T", view.Select)
	}
}

func TestParseSQLSequence(t *testing.T) {
	create, ok := mustParseOne(t, "CREATE SEQUENCE IF NOT EXISTS s.seq;").(*ast.CreateSequenceStmt)
	if !ok || !create.IfNotExists || create.Name.Schema.O != "s" || create.Name.Name.O != "seq" {
		t.Fatalf("unexpected create sequence: %#v", create)
	}
	drop, ok := mustParseOne(t, "DROP SEQUENCE IF EXISTS seq;").(*ast.DropSequenceStmt)
	if !ok || !drop.IfExists || len(drop.Names) != 1 {
		t.Fatalf("unexpected drop sequence: %#v", drop)
	}
}

func TestParseSQLWithClause(t *testing.T) {
	st := mustParseSelect(t, "WITH RECURSIVE t(n) AS (SELECT 1 UNION ALL SELECT n + 1 FROM t WHERE n < 10) SELECT n FROM t;")
	if st.With == nil || !st.With.IsRecursive || len(st.With.CTEs) != 1 {
		t.Fatalf("unexpected with clause: %#v", st.With)
	}
	cte := st.With.CTEs[0]
	if cte.Name.O != "t" || len(cte.ColNames) != 1 || cte.ColNames[0].O != "n" {
		t.Fatalf("unexpected cte: %#v", cte)
	}
	if _, ok := cte.Query.(*ast.SetOprStmt); !ok {
		t.Fatalf("expected set operation in cte, got %T", cte.Query)
	}
}

func TestParseSQLInsertSelectAndReturning(t *testing.T) {
	ins, ok := mustParseOne(t, "INSERT INTO s.t (a, b) SELECT a, b FROM u RETURNING a;").(*ast.InsertStmt)
	if !ok || ins.Table.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Schema.O != "s" {
		t.Fatalf("unexpected insert: %#v", ins)
	}
	if len(ins.Columns) != 2 || ins.Select == nil || ins.Returning == nil {
		t.Fatalf("unexpected insert fields: %#v", ins)
	}

	upd, ok := mustParseOne(t, "UPDATE t SET a = 1 WHERE b = 2 RETURNING a;").(*ast.UpdateStmt)
	if !ok || upd.Returning == nil || upd.Where == nil {
		t.Fatalf("unexpected update: %#v", upd)
	}

	del, ok := mustParseOne(t, "DELETE FROM t WHERE a = 1 RETURNING id;").(*ast.DeleteStmt)
	if !ok || del.Returning == nil || del.Where == nil {
		t.Fatalf("unexpected delete: %#v", del)
	}
}

func TestParseSQLSetOperationsChain(t *testing.T) {
	st, ok := mustParseOne(t, "SELECT 1 UNION ALL SELECT 2 EXCEPT SELECT 3;").(*ast.SetOprStmt)
	if !ok {
		t.Fatal("expected *ast.SetOprStmt")
	}
	selects := st.SelectList.Selects
	if len(selects) != 3 {
		t.Fatalf("selects = %d, want 3", len(selects))
	}
	second := selects[1].(*ast.SelectStmt)
	if second.AfterSetOperator == nil || *second.AfterSetOperator != ast.UnionAll {
		t.Fatalf("second set operator = %v, want UNION ALL", second.AfterSetOperator)
	}
	third := selects[2].(*ast.SelectStmt)
	if third.AfterSetOperator == nil || *third.AfterSetOperator != ast.Except {
		t.Fatalf("third set operator = %v, want EXCEPT", third.AfterSetOperator)
	}
}

func TestParseSQLUtilityAndTransactions(t *testing.T) {
	set, ok := mustParseOne(t, "SET search_path TO public;").(*ast.SetStmt)
	if !ok || len(set.Variables) != 1 || set.Variables[0].Name != "search_path" {
		t.Fatalf("unexpected set: %#v", set)
	}
	if _, ok := mustParseOne(t, "SET a.b = 'c';").(*ast.SetStmt); !ok {
		t.Fatal("expected *ast.SetStmt")
	}
	if _, ok := mustParseOne(t, "SHOW all;").(*ast.ShowStmt); !ok {
		t.Fatal("expected *ast.ShowStmt")
	}
	if _, ok := mustParseOne(t, "BEGIN;").(*ast.BeginStmt); !ok {
		t.Fatal("expected *ast.BeginStmt")
	}
	if _, ok := mustParseOne(t, "COMMIT WORK;").(*ast.CommitStmt); !ok {
		t.Fatal("expected *ast.CommitStmt")
	}
	if _, ok := mustParseOne(t, "ROLLBACK;").(*ast.RollbackStmt); !ok {
		t.Fatal("expected *ast.RollbackStmt")
	}
}

func TestParseSQLUpdateFromAndDeleteUsing(t *testing.T) {
	upd, ok := mustParseOne(t, "UPDATE t SET a = u.a FROM u, v WHERE t.id = u.id RETURNING t.a;").(*ast.UpdateStmt)
	if !ok || upd.From == nil || upd.Returning == nil || upd.Where == nil {
		t.Fatalf("unexpected update: %#v", upd)
	}
	if upd.From.TableRefs.Right == nil {
		t.Fatal("expected joined FROM tables")
	}

	del, ok := mustParseOne(t, "DELETE FROM t USING u, v WHERE t.id = u.id RETURNING *;").(*ast.DeleteStmt)
	if !ok || del.Using == nil || del.Returning == nil || del.Where == nil {
		t.Fatalf("unexpected delete: %#v", del)
	}
}

func TestParseSQLInsertOnConflict(t *testing.T) {
	ins, ok := mustParseOne(t, "INSERT INTO t (a, b) VALUES (1, 2) ON CONFLICT DO NOTHING;").(*ast.InsertStmt)
	if !ok || ins.OnConflict == nil || !ins.OnConflict.DoNothing {
		t.Fatalf("unexpected on conflict: %#v", ins)
	}

	ins, ok = mustParseOne(t, "INSERT INTO t (a) VALUES (1) ON CONFLICT (a) WHERE a > 0 DO UPDATE SET a = excluded.a, b = 2 WHERE t.a > 0 RETURNING a;").(*ast.InsertStmt)
	if !ok || ins.OnConflict == nil || ins.OnConflict.DoNothing {
		t.Fatalf("unexpected on conflict update: %#v", ins)
	}
	c := ins.OnConflict
	if len(c.IndexCols) != 1 || c.TargetWhere == nil || len(c.UpdateList) != 2 || c.UpdateWhere == nil || ins.Returning == nil {
		t.Fatalf("unexpected on conflict fields: %#v", c)
	}
}

func TestParseSQLCommentOn(t *testing.T) {
	c, ok := mustParseOne(t, "COMMENT ON COLUMN s.t.c IS 'hello';").(*ast.CommentStmt)
	if !ok || c.ObjectType != "column" || c.ObjectName != "s.t.c" || c.Comment == nil || *c.Comment != "hello" {
		t.Fatalf("unexpected comment: %#v", c)
	}
	c, ok = mustParseOne(t, "COMMENT ON TABLE t IS NULL;").(*ast.CommentStmt)
	if !ok || c.ObjectType != "table" || c.Comment != nil {
		t.Fatalf("unexpected comment removal: %#v", c)
	}
}

func TestParseSQLExplainAnalyzeAndCreateDatabase(t *testing.T) {
	exp, ok := mustParseOne(t, "EXPLAIN ANALYZE SELECT * FROM t;").(*ast.ExplainStmt)
	if !ok || !exp.Analyze {
		t.Fatalf("unexpected explain: %#v", exp)
	}
	if _, ok := exp.Stmt.(*ast.SelectStmt); !ok {
		t.Fatalf("expected select body, got %T", exp.Stmt)
	}

	if _, ok := mustParseOne(t, "ANALYZE s.t, u;").(*ast.AnalyzeTableStmt); !ok {
		t.Fatal("expected *ast.AnalyzeTableStmt")
	}

	db, ok := mustParseOne(t, "CREATE DATABASE IF NOT EXISTS db;").(*ast.CreateDatabaseStmt)
	if !ok || !db.IfNotExists || db.Name != "db" {
		t.Fatalf("unexpected create database: %#v", db)
	}
}

func TestParseSQLMerge(t *testing.T) {
	m, ok := mustParseOne(t, "MERGE INTO t USING u ON t.id = u.id WHEN MATCHED AND u.a > 0 THEN UPDATE SET a = u.a WHEN NOT MATCHED THEN INSERT (id, a) VALUES (u.id, u.a);").(*ast.MergeStmt)
	if !ok {
		t.Fatal("expected *ast.MergeStmt")
	}
	if m.On == nil || m.Table == nil || m.Source == nil {
		t.Fatalf("unexpected merge: %#v", m)
	}
	if len(m.Whens) != 2 {
		t.Fatalf("whens = %d, want 2", len(m.Whens))
	}
	if !m.Whens[0].Matched || m.Whens[0].Tp != ast.MergeWhenMatchedUpdate || m.Whens[0].Condition == nil {
		t.Fatalf("unexpected first when: %#v", m.Whens[0])
	}
	if m.Whens[1].Matched || m.Whens[1].Tp != ast.MergeWhenNotMatchedInsert || len(m.Whens[1].Values) != 2 {
		t.Fatalf("unexpected second when: %#v", m.Whens[1])
	}
}

func TestParseSQLWithDML(t *testing.T) {
	ins, ok := mustParseOne(t, "WITH t AS (SELECT 1) INSERT INTO x SELECT * FROM t;").(*ast.InsertStmt)
	if !ok || ins.With == nil || len(ins.With.CTEs) != 1 {
		t.Fatalf("unexpected with insert: %#v", ins)
	}
	upd, ok := mustParseOne(t, "WITH t AS (SELECT 1) UPDATE x SET a = 1 FROM t;").(*ast.UpdateStmt)
	if !ok || upd.With == nil || upd.From == nil {
		t.Fatalf("unexpected with update: %#v", upd)
	}
	del, ok := mustParseOne(t, "WITH t AS (SELECT 1) DELETE FROM x USING t;").(*ast.DeleteStmt)
	if !ok || del.With == nil || del.Using == nil {
		t.Fatalf("unexpected with delete: %#v", del)
	}
}

func TestParseSQLCaseExpressions(t *testing.T) {
	simple := mustParseSelect(t, "SELECT CASE a WHEN 1 THEN 'x' WHEN 2 THEN 'y' ELSE 'z' END FROM t;")
	ce, ok := simple.Fields.Fields[0].Expr.(*ast.CaseExpr)
	if !ok {
		t.Fatalf("expected *ast.CaseExpr, got %T", simple.Fields.Fields[0].Expr)
	}
	if ce.Value == nil || len(ce.WhenClauses) != 2 || ce.ElseClause == nil {
		t.Fatalf("unexpected simple case: %#v", ce)
	}

	searched := mustParseSelect(t, "SELECT CASE WHEN a = 1 THEN 'x' WHEN a = 2 THEN 'y' END FROM t;")
	ce, ok = searched.Fields.Fields[0].Expr.(*ast.CaseExpr)
	if !ok || ce.Value != nil || len(ce.WhenClauses) != 2 || ce.ElseClause != nil {
		t.Fatalf("unexpected searched case: %#v", ce)
	}
}

func TestParseSQLGroupOrderAndIn(t *testing.T) {
	st := mustParseSelect(t, "SELECT a FROM t GROUP BY a, b, c ORDER BY a DESC, b, c ASC, d LIMIT ALL;")
	if st.GroupBy == nil || len(st.GroupBy.Items) != 3 {
		t.Fatalf("group by items = %#v, want 3", st.GroupBy)
	}
	if st.OrderBy == nil || len(st.OrderBy.Items) != 4 {
		t.Fatalf("order by items = %#v, want 4", st.OrderBy)
	}
	if st.Limit != nil {
		t.Fatalf("LIMIT ALL should not produce a limit, got %#v", st.Limit)
	}

	in := mustParseSelect(t, "SELECT * FROM t WHERE a NOT IN (SELECT a FROM u);")
	pin, ok := in.Where.(*ast.PatternInExpr)
	if !ok || !pin.Not || pin.Sel == nil {
		t.Fatalf("unexpected NOT IN subquery: %#v", in.Where)
	}
}

func TestParseSQLParenthesizedSetOperations(t *testing.T) {
	st, ok := mustParseOne(t, "(SELECT 1) UNION (SELECT 2);").(*ast.SetOprStmt)
	if !ok || len(st.SelectList.Selects) != 2 {
		t.Fatalf("unexpected set operation: %#v", st)
	}
	single, ok := mustParseOne(t, "(SELECT 1);").(*ast.SelectStmt)
	if !ok || !single.IsInBraces {
		t.Fatalf("unexpected braced select: %#v", single)
	}
}

func TestParseSQLSelectDistinctAndJoinUsing(t *testing.T) {
	st := mustParseSelect(t, "SELECT DISTINCT a FROM t JOIN u USING (id, tenant_id);")
	if !st.SelectStmtOpts.Distinct {
		t.Fatal("expected DISTINCT")
	}
	if st.From == nil || st.From.TableRefs == nil || st.From.TableRefs.Using == nil {
		t.Fatalf("expected join USING, got %#v", st.From)
	}
	if len(st.From.TableRefs.Using) != 2 {
		t.Fatalf("using columns = %d, want 2", len(st.From.TableRefs.Using))
	}
}

func TestParseSQLWindowAndNullsOrder(t *testing.T) {
	st := mustParseSelect(t, "SELECT ROW_NUMBER() OVER (PARTITION BY dept ORDER BY salary DESC NULLS LAST) FROM employees;")
	w, ok := st.Fields.Fields[0].Expr.(*ast.WindowFuncExpr)
	if !ok {
		t.Fatalf("expected *ast.WindowFuncExpr, got %T", st.Fields.Fields[0].Expr)
	}
	if w.Spec.PartitionBy == nil || w.Spec.OrderBy == nil || len(w.Spec.OrderBy.Items) != 1 {
		t.Fatalf("unexpected window spec: %#v", w.Spec)
	}
	if w.Spec.OrderBy.Items[0].NullOrder != 2 || !w.Spec.OrderBy.Items[0].Desc {
		t.Fatalf("unexpected null ordering: %#v", w.Spec.OrderBy.Items[0])
	}
}

func TestParseSQLJsonAndLikeOperators(t *testing.T) {
	st := mustParseSelect(t, "SELECT data->'a'->>'b' FROM t WHERE name ILIKE 'x%' AND data @> '{\"k\":1}';")
	if _, ok := st.Fields.Fields[0].Expr.(*ast.BinaryOperationExpr); !ok {
		t.Fatalf("expected json binary expression, got %T", st.Fields.Fields[0].Expr)
	}
	if _, ok := st.Where.(*ast.BinaryOperationExpr); !ok {
		t.Fatalf("expected and expression, got %T", st.Where)
	}
}

func TestParseSQLCastAndExtract(t *testing.T) {
	st := mustParseSelect(t, "SELECT length(name)::int, EXTRACT(year FROM now()) FROM t WHERE a::int < 5;")
	if _, ok := st.Fields.Fields[0].Expr.(*ast.FuncCastExpr); !ok {
		t.Fatalf("expected cast expression, got %T", st.Fields.Fields[0].Expr)
	}
	fc, ok := st.Fields.Fields[1].Expr.(*ast.FuncCallExpr)
	if !ok || fc.FnName.L != "extract" {
		t.Fatalf("expected extract call, got %#v", st.Fields.Fields[1].Expr)
	}
	if _, ok := st.Where.(*ast.BinaryOperationExpr); !ok {
		t.Fatalf("expected comparison with cast, got %T", st.Where)
	}
}

func TestParseSQLCharacterVaryingSynonym(t *testing.T) {
	stmt := mustParseOne(t, "CREATE TABLE t (a CHARACTER VARYING(20), b VARCHAR(20));")
	ct, ok := stmt.(*ast.CreateTableStmt)
	if !ok {
		t.Fatalf("expected *ast.CreateTableStmt, got %T", stmt)
	}
	if len(ct.Cols) != 2 {
		t.Fatalf("columns = %d, want 2", len(ct.Cols))
	}
	for i, col := range ct.Cols {
		if got := col.Tp.String(); got != "varchar(20)" {
			t.Fatalf("column %d type = %q, want varchar(20)", i, got)
		}
	}

	st := mustParseSelect(t, "SELECT 'x'::CHARACTER VARYING(20), CAST('x' AS CHARACTER VARYING(20)), a::int b FROM t;")
	for i := 0; i < 2; i++ {
		fc, ok := st.Fields.Fields[i].Expr.(*ast.FuncCastExpr)
		if !ok {
			t.Fatalf("field %d: expected cast, got %T", i, st.Fields.Fields[i].Expr)
		}
		if fc.Tp == nil || fc.Tp.String() != "varchar(20)" {
			t.Fatalf("field %d: type = %#v, want varchar(20)", i, fc.Tp)
		}
	}
	if got := st.Fields.Fields[2].AsName.O; got != "b" {
		t.Fatalf("alias = %q, want b", got)
	}
}

func TestParseSQLQualifiedFunctionAndSoftKeywords(t *testing.T) {
	st := mustParseSelect(t, "SELECT pg_catalog.lower(name), myschema.length(col) FROM t WHERE role = 'admin';")
	if len(st.Fields.Fields) != 2 {
		t.Fatalf("fields = %d, want 2", len(st.Fields.Fields))
	}
	fc, ok := st.Fields.Fields[0].Expr.(*ast.FuncCallExpr)
	if !ok || fc.FnName.O != "pg_catalog.lower" {
		t.Fatalf("expected qualified function, got %#v", st.Fields.Fields[0].Expr)
	}
	if st.Where == nil {
		t.Fatal("expected where clause using soft keyword")
	}
}

func TestParseSQLDistinctOnAndWindowFrame(t *testing.T) {
	st := mustParseSelect(t, "SELECT DISTINCT ON (user_id) user_id FROM events ORDER BY user_id;")
	if !st.Distinct || st.SelectStmtOpts == nil || len(st.DistinctOn) != 1 {
		t.Fatalf("unexpected distinct on: %#v", st.SelectStmtOpts)
	}
	w := mustParseSelect(t, "SELECT SUM(amount) OVER (ORDER BY created_at ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) FROM t;")
	wf, ok := w.Fields.Fields[0].Expr.(*ast.WindowFuncExpr)
	if !ok || wf.Spec.Frame == nil || wf.Spec.Frame.Type != ast.Rows {
		t.Fatalf("unexpected window frame: %#v", w.Fields.Fields[0].Expr)
	}
	if !wf.Spec.Frame.Extent.Start.UnBounded || wf.Spec.Frame.Extent.End.Type != ast.CurrentRow {
		t.Fatalf("unexpected frame extent: %#v", wf.Spec.Frame.Extent)
	}
}

func TestParseSQLLikeInternalAndNaturalJoin(t *testing.T) {
	st := mustParseSelect(t, "SELECT * FROM a NATURAL LEFT JOIN b WHERE name !~~ 'x%';")
	if st.From == nil || !st.From.TableRefs.NaturalJoin {
		t.Fatalf("expected natural join, got %#v", st.From)
	}
	pl, ok := st.Where.(*ast.PatternLikeExpr)
	if !ok || !pl.Not {
		t.Fatalf("expected negated like, got %#v", st.Where)
	}
}

func TestParseSQLCreateIndexOptionsAndDefaults(t *testing.T) {
	idx, ok := mustParseOne(t, "CREATE UNIQUE INDEX CONCURRENTLY idx ON ONLY t (a) INCLUDE (b, c) WHERE a > 0;").(*ast.CreateIndexStmt)
	if !ok || !idx.Unique || len(idx.Include) != 2 || idx.Where == nil {
		t.Fatalf("unexpected create index: %#v", idx)
	}
	ct, ok := mustParseOne(t, "CREATE TABLE t (id integer DEFAULT 1 NOT NULL, tags text[], created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP);").(*ast.CreateTableStmt)
	if !ok || len(ct.Cols) != 3 {
		t.Fatalf("unexpected create table: %#v", ct)
	}
}

func TestParseSQLCommentFunctionAndSetScope(t *testing.T) {
	c, ok := mustParseOne(t, "COMMENT ON FUNCTION public.f(integer) IS 'x';").(*ast.CommentStmt)
	if !ok || c.ObjectName == "" {
		t.Fatalf("unexpected comment: %#v", c)
	}
	for _, sql := range []string{
		"SET LOCAL client_min_messages = DEBUG;",
		"SET SESSION client_min_messages TO notice;",
		"SET SESSION AUTHORIZATION postgres;",
	} {
		if _, ok := mustParseOne(t, sql).(*ast.SetStmt); !ok {
			t.Fatalf("expected *ast.SetStmt for %q", sql)
		}
	}
}

func TestParseSQLDropBehaviorAndMaterialized(t *testing.T) {
	if _, ok := mustParseOne(t, "DROP TABLE IF EXISTS t CASCADE;").(*ast.DropTableStmt); !ok {
		t.Fatal("expected *ast.DropTableStmt")
	}
	if _, ok := mustParseOne(t, "DROP INDEX CONCURRENTLY IF EXISTS idx;").(*ast.DropIndexStmt); !ok {
		t.Fatal("expected *ast.DropIndexStmt")
	}
	ins, ok := mustParseOne(t, "WITH x AS MATERIALIZED (SELECT 1) SELECT * FROM x;").(*ast.SelectStmt)
	if !ok || ins.With == nil || len(ins.With.CTEs) != 1 {
		t.Fatalf("unexpected materialized CTE: %#v", ins)
	}
}

func TestParseSQLAtTimeZone(t *testing.T) {
	st := mustParseSelect(t, "SELECT created_at AT TIME ZONE 'UTC' FROM events;")
	fc, ok := st.Fields.Fields[0].Expr.(*ast.FuncCallExpr)
	if !ok || fc.FnName.L != "timezone" {
		t.Fatalf("expected timezone call, got %#v", st.Fields.Fields[0].Expr)
	}
}

func TestParseSQLLateralAndTableFunctions(t *testing.T) {
	st := mustParseSelect(t, "SELECT * FROM products p CROSS JOIN LATERAL unnest(p.tags) AS tag(name);")
	join := st.From.TableRefs
	right, ok := join.Right.(*ast.TableSource)
	if !ok {
		t.Fatalf("expected right TableSource, got %T", join.Right)
	}
	if !right.Lateral {
		t.Fatal("expected LATERAL table source")
	}
	if _, ok := right.Source.(*ast.TableFunction); !ok {
		t.Fatalf("expected TableFunction, got %T", right.Source)
	}
	if right.AsName.L != "tag" {
		t.Fatalf("alias = %q, want tag", right.AsName.O)
	}
	if len(right.ColumnAliases) != 1 || right.ColumnAliases[0].L != "name" {
		t.Fatalf("column aliases = %#v, want [name]", right.ColumnAliases)
	}

	st = mustParseSelect(t, "SELECT * FROM t, LATERAL (SELECT * FROM u WHERE u.tid = t.id) sub;")
	right, ok = st.From.TableRefs.Right.(*ast.TableSource)
	if !ok || !right.Lateral {
		t.Fatalf("expected lateral subquery, got %#v", st.From.TableRefs.Right)
	}
}

func TestParseSQLJSONExistsAndWildcardArgs(t *testing.T) {
	st := mustParseSelect(t, "SELECT to_jsonb(ro.*) FROM recent_orders ro WHERE ro.data ? 'key';")
	if st.Where == nil {
		t.Fatal("expected WHERE clause")
	}
	if _, ok := st.Fields.Fields[0].Expr.(*ast.FuncCallExpr); !ok {
		t.Fatalf("expected function call, got %T", st.Fields.Fields[0].Expr)
	}

	st = mustParseSelect(t, "SELECT * FROM t WHERE meta ? 'key' AND id = $1;")
	if st.Where == nil {
		t.Fatal("expected WHERE clause")
	}
}

func TestParseSQLSetSearchPathAndTruncateOptions(t *testing.T) {
	if _, ok := mustParseOne(t, "SET search_path = public, pg_catalog;").(*ast.SetStmt); !ok {
		t.Fatal("expected *ast.SetStmt")
	}
	if _, ok := mustParseOne(t, "SET search_path FROM CURRENT;").(*ast.SetStmt); !ok {
		t.Fatal("expected *ast.SetStmt")
	}
	tr, ok := mustParseOne(t, "TRUNCATE ONLY public.users, orders CASCADE;").(*ast.TruncateTableStmt)
	if !ok {
		t.Fatal("expected *ast.TruncateTableStmt")
	}
	if !tr.Only || tr.Behavior != "CASCADE" || len(tr.Tables) != 2 {
		t.Fatalf("unexpected truncate: %#v", tr)
	}
}

func TestParseSQLWithDMLAndCTEDML(t *testing.T) {
	if _, ok := mustParseOne(t, "WITH source AS (SELECT id FROM staging) INSERT INTO users (id) SELECT id FROM source;").(*ast.InsertStmt); !ok {
		t.Fatal("expected *ast.InsertStmt")
	}
	sel, ok := mustParseOne(t, "WITH t AS (UPDATE orders SET total = total + 1 RETURNING id) SELECT * FROM t;").(*ast.SelectStmt)
	if !ok || sel.With == nil {
		t.Fatalf("expected select with CTE, got %#v", sel)
	}
	if _, ok := sel.With.CTEs[0].Query.(*ast.UpdateStmt); !ok {
		t.Fatalf("expected update CTE query, got %T", sel.With.CTEs[0].Query)
	}
}

func TestParseSQLUtilitySetAndSearchPath(t *testing.T) {
	valid := []string{
		"SET client_min_messages = warning;",
		"SET SESSION client_min_messages TO notice;",
		"SET LOCAL client_min_messages = DEBUG;",
		"SET search_path = public;",
		"SET search_path = public, pg_catalog;",
		"SET search_path TO public;",
		"SET search_path FROM CURRENT;",
		"SET ROLE postgres;",
		"SET SESSION AUTHORIZATION postgres;",
		"ALTER SYSTEM SET client_min_messages = warning;",
		"SHOW search_path;",
		"RESET TIME ZONE;",
	}
	for _, sql := range valid {
		if _, err := pgparser.New().ParseOneStmt(sql); err != nil {
			t.Errorf("ParseOneStmt(%q) error: %v", sql, err)
		}
	}

	invalid := []string{
		"SET",
		"SET ;",
		"SET LOCAL",
		"SET SESSION",
		"SET =",
		"SET ROLE",
		"SET SESSION AUTHORIZATION",
		"SET client_min_messages",
		"SET client_min_messages TO",
		"SET client_min_messages =",
		"SET client_min_messages = warning]",
		"SET client_min_messages == warning",
		"SHOW",
		"SHOW ALL extra",
		"RESET",
		"RESET ALL extra",
		"SETTINGS foo = bar",
	}
	for _, sql := range invalid {
		if _, err := pgparser.New().ParseOneStmt(sql); err == nil {
			t.Errorf("ParseOneStmt(%q) expected error", sql)
		}
	}
}

type tableCollector struct {
	tables []*ast.TableName
}

func (c *tableCollector) Enter(n ast.Node) (ast.Node, bool) {
	if tn, ok := n.(*ast.TableName); ok {
		c.tables = append(c.tables, tn)
	}
	return n, false
}

func (c *tableCollector) Leave(n ast.Node) (ast.Node, bool) { return n, true }

func collectTables(n ast.Node) []*ast.TableName {
	c := &tableCollector{}
	n.Accept(c)
	return c.tables
}

type paramCollector struct {
	count int
}

func (c *paramCollector) Enter(n ast.Node) (ast.Node, bool) {
	if _, ok := n.(ast.ParamMarkerExpr); ok {
		c.count++
	}
	return n, false
}

func (c *paramCollector) Leave(n ast.Node) (ast.Node, bool) { return n, true }

func countParams(n ast.Node) int {
	c := &paramCollector{}
	n.Accept(c)
	return c.count
}

func hasTable(tables []*ast.TableName, schema, name string) bool {
	for _, tbl := range tables {
		if tbl.Name.L == name && (schema == "" || tbl.Schema.L == schema) {
			return true
		}
	}
	return false
}

func TestParseSQLCTESelect(t *testing.T) {
	st := mustParseSelect(t, `WITH active_users AS (
    SELECT users.id, users.name FROM users WHERE users.active = $1
)
SELECT active_users.id, active_users.name
FROM active_users
WHERE active_users.id = $2;`)

	if st.With == nil || len(st.With.CTEs) != 1 {
		t.Fatalf("expected 1 CTE, got %#v", st.With)
	}
	cte, ok := st.With.CTEs[0].Query.(*ast.SelectStmt)
	if !ok {
		t.Fatalf("expected select CTE query, got %T", st.With.CTEs[0].Query)
	}
	if len(cte.Fields.Fields) != 2 {
		t.Fatalf("expected 2 CTE fields, got %d", len(cte.Fields.Fields))
	}
	if got := cte.Fields.Fields[0].Expr.(*ast.ColumnNameExpr).Name.String(); got != "users.id" {
		t.Fatalf("first CTE field = %q, want users.id", got)
	}
	if !hasTable(collectTables(cte), "", "users") {
		t.Fatal("expected users table inside CTE")
	}
	if got := countParams(cte); got != 1 {
		t.Fatalf("CTE parameters = %d, want 1", got)
	}
}

func TestParseSQLCTENestedTables(t *testing.T) {
	st := mustParseSelect(t, `WITH ro AS (SELECT id FROM orders) SELECT * FROM ro JOIN customers c ON c.id = ro.id;`)
	cte, ok := st.With.CTEs[0].Query.(*ast.SelectStmt)
	if !ok {
		t.Fatal("expected select CTE query")
	}
	if !hasTable(collectTables(cte), "", "orders") {
		t.Fatal("expected orders in CTE body")
	}
	mainTables := collectTables(st.From)
	if hasTable(mainTables, "", "orders") {
		t.Fatal("orders should not appear in the outer FROM")
	}
	if !hasTable(mainTables, "", "ro") || !hasTable(mainTables, "", "customers") {
		t.Fatalf("unexpected outer tables: %#v", mainTables)
	}
}

func TestParseSQLCTEUpdate(t *testing.T) {
	st := mustParseSelect(t, `WITH updated_users AS (
    UPDATE users SET active = true WHERE id = $1 RETURNING id
)
SELECT * FROM updated_users;`)
	upd, ok := st.With.CTEs[0].Query.(*ast.UpdateStmt)
	if !ok {
		t.Fatalf("expected update CTE query, got %T", st.With.CTEs[0].Query)
	}
	if len(upd.List) == 0 {
		t.Fatal("expected SET clauses in CTE update")
	}
	if upd.Returning == nil || len(upd.Returning.Fields) == 0 {
		t.Fatal("expected RETURNING in CTE update")
	}
	if !hasTable(collectTables(upd), "", "users") {
		t.Fatal("expected users table in CTE update")
	}
	if got := countParams(upd); got != 1 {
		t.Fatalf("CTE update parameters = %d, want 1", got)
	}
}

func TestParseSQLCTETablesExtraction(t *testing.T) {
	st := mustParseSelect(t, `WITH expanded AS (
    SELECT payload FROM slow_smoke.unindexed WHERE payload LIKE $1
)
SELECT COUNT(*) FROM expanded e1 JOIN expanded e2 ON e2.payload >= e1.payload;`)
	if len(st.With.CTEs) != 1 || st.With.CTEs[0].Name.L != "expanded" {
		t.Fatalf("unexpected CTEs: %#v", st.With.CTEs)
	}
	all := collectTables(st)
	if !hasTable(all, "slow_smoke", "unindexed") {
		t.Fatal("expected slow_smoke.unindexed base table")
	}
	refs := 0
	for _, tbl := range all {
		if tbl.Name.L == "expanded" && tbl.Schema.L == "" {
			refs++
		}
	}
	if refs != 2 {
		t.Fatalf("expanded references = %d, want 2", refs)
	}
}

func TestParseSQLMultipleCTETables(t *testing.T) {
	st := mustParseSelect(t, `WITH
	users_cte AS (SELECT * FROM users WHERE active = true),
	orders_cte AS (SELECT * FROM orders JOIN products ON orders.product_id = products.id)
SELECT * FROM users_cte JOIN orders_cte ON users_cte.id = orders_cte.user_id;`)
	all := collectTables(st)
	for _, name := range []string{"users", "orders", "products"} {
		if !hasTable(all, "", name) {
			t.Fatalf("expected base table %q", name)
		}
	}
}

func TestParseSQLMergeStatement(t *testing.T) {
	st, ok := mustParseOne(t, `
MERGE INTO customers c
USING (
  SELECT customer_id, total_spent
  FROM summary
) s
ON c.id = s.customer_id
WHEN MATCHED AND s.total_spent > 1000 THEN UPDATE SET vip = true
WHEN NOT MATCHED THEN INSERT (id, total_spent) VALUES (s.customer_id, s.total_spent)
WHEN MATCHED THEN DELETE;`).(*ast.MergeStmt)
	if !ok {
		t.Fatal("expected *ast.MergeStmt")
	}
	left, ok := st.Table.TableRefs.Left.(*ast.TableSource)
	if !ok {
		t.Fatalf("expected target TableSource, got %T", st.Table.TableRefs.Left)
	}
	target, ok := left.Source.(*ast.TableName)
	if !ok || target.Name.L != "customers" || left.AsName.L != "c" {
		t.Fatalf("unexpected MERGE target: %#v", left)
	}
	src, ok := st.Source.TableRefs.Left.(*ast.TableSource)
	if !ok {
		t.Fatalf("expected source TableSource, got %T", st.Source.TableRefs.Left)
	}
	if _, ok := src.Source.(*ast.SelectStmt); !ok || src.AsName.L != "s" {
		t.Fatalf("unexpected MERGE source: %#v", src)
	}
	if st.On == nil {
		t.Fatal("expected MERGE ON condition")
	}
	if len(st.Whens) != 3 {
		t.Fatalf("MERGE actions = %d, want 3", len(st.Whens))
	}
	if st.Whens[0].Tp != ast.MergeWhenMatchedUpdate || len(st.Whens[0].UpdateList) != 1 || st.Whens[0].Condition == nil {
		t.Fatalf("unexpected update action: %#v", st.Whens[0])
	}
	if st.Whens[1].Tp != ast.MergeWhenNotMatchedInsert || len(st.Whens[1].Columns) != 2 || len(st.Whens[1].Values) != 2 {
		t.Fatalf("unexpected insert action: %#v", st.Whens[1])
	}
	if st.Whens[2].Tp != ast.MergeWhenMatchedDelete {
		t.Fatalf("unexpected delete action: %#v", st.Whens[2])
	}
	all := collectTables(st)
	if !hasTable(all, "", "customers") || !hasTable(all, "", "summary") {
		t.Fatalf("unexpected MERGE tables: %#v", all)
	}
}

func TestParseSQLSubqueryMetadataFromClause(t *testing.T) {
	st := mustParseSelect(t, `
SELECT t.id, sq.total
FROM tenants t
JOIN (
  SELECT tenant_id, SUM(amount) AS total
  FROM invoices
  GROUP BY tenant_id
) sq ON sq.tenant_id = t.id;`)
	all := collectTables(st)
	if !hasTable(all, "", "tenants") || !hasTable(all, "", "invoices") {
		t.Fatalf("unexpected tables: %#v", all)
	}
	var sq *ast.TableSource
	var find func(ast.ResultSetNode)
	find = func(n ast.ResultSetNode) {
		switch v := n.(type) {
		case *ast.TableSource:
			if v.AsName.L == "sq" {
				sq = v
			}
			find(v.Source)
		case *ast.Join:
			find(v.Left)
			if v.Right != nil {
				find(v.Right)
			}
		}
	}
	find(st.From.TableRefs)
	if sq == nil {
		t.Fatal("expected derived table alias sq")
	}
	sub, ok := sq.Source.(*ast.SelectStmt)
	if !ok {
		t.Fatalf("expected subquery, got %T", sq.Source)
	}
	if sub.GroupBy == nil || len(sub.GroupBy.Items) == 0 {
		t.Fatal("expected GROUP BY in subquery")
	}
}

func TestParseSQLSetOperationsMetadata(t *testing.T) {
	st, ok := mustParseOne(t, `
(
  SELECT user_id FROM logins
  INTERSECT
  SELECT user_id FROM payments
)
EXCEPT ALL
SELECT user_id FROM banned_users;`).(*ast.SetOprStmt)
	if !ok {
		t.Fatal("expected *ast.SetOprStmt")
	}
	if st.SelectList == nil || len(st.SelectList.Selects) != 2 {
		t.Fatalf("unexpected set operation list: %#v", st.SelectList)
	}
	nested, ok := st.SelectList.Selects[0].(*ast.SetOprStmt)
	if !ok {
		t.Fatalf("expected nested set operation, got %T", st.SelectList.Selects[0])
	}
	if len(nested.SelectList.Selects) != 2 {
		t.Fatalf("unexpected nested selects: %#v", nested.SelectList.Selects)
	}
	if op := nested.SelectList.Selects[1].(*ast.SelectStmt).AfterSetOperator; op == nil || *op != ast.Intersect {
		t.Fatalf("nested operator = %v, want INTERSECT", op)
	}
	if op := st.SelectList.Selects[1].(*ast.SelectStmt).AfterSetOperator; op == nil || *op != ast.ExceptAll {
		t.Fatalf("outer operator = %v, want EXCEPT ALL", op)
	}
	all := collectTables(st)
	for _, name := range []string{"logins", "payments", "banned_users"} {
		if !hasTable(all, "", name) {
			t.Fatalf("expected table %q", name)
		}
	}
}

func TestParseSQLSetOperationsTableDriven(t *testing.T) {
	tests := []struct {
		name       string
		sql        string
		op         ast.SetOprType
		wantTables []string
	}{
		{"union", "SELECT account_id FROM current_accounts UNION SELECT account_id FROM archived_accounts;", ast.Union, []string{"current_accounts", "archived_accounts"}},
		{"intersect all", "SELECT user_id FROM service_a_events INTERSECT ALL SELECT user_id FROM service_b_events;", ast.IntersectAll, []string{"service_a_events", "service_b_events"}},
		{"except", "SELECT user_id FROM granted_permissions EXCEPT SELECT user_id FROM revoked_permissions;", ast.Except, []string{"granted_permissions", "revoked_permissions"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st, ok := mustParseOne(t, tc.sql).(*ast.SetOprStmt)
			if !ok {
				t.Fatalf("expected *ast.SetOprStmt")
			}
			right := st.SelectList.Selects[1].(*ast.SelectStmt)
			if right.AfterSetOperator == nil || *right.AfterSetOperator != tc.op {
				t.Fatalf("operator = %v, want %v", right.AfterSetOperator, tc.op)
			}
			all := collectTables(st)
			for _, name := range tc.wantTables {
				if !hasTable(all, "", name) {
					t.Fatalf("expected table %q", name)
				}
			}
		})
	}
}

func TestParseSQLSetOpBranchesKeepWhere(t *testing.T) {
	for _, op := range []string{"UNION", "INTERSECT", "EXCEPT"} {
		t.Run(op, func(t *testing.T) {
			st, ok := mustParseOne(t, "SELECT a FROM t1 WHERE x = 1 "+op+" SELECT a FROM t2 WHERE y = 2;").(*ast.SetOprStmt)
			if !ok {
				t.Fatal("expected *ast.SetOprStmt")
			}
			for _, sel := range st.SelectList.Selects {
				if sel.(*ast.SelectStmt).Where == nil {
					t.Fatalf("%s branch is missing WHERE", op)
				}
			}
		})
	}
}

func TestParseSQLSetOpLeadingSubquery(t *testing.T) {
	st, ok := mustParseOne(t, "SELECT a FROM (SELECT a FROM inner_t) s WHERE x = 1 UNION SELECT a FROM t2 WHERE y = 2;").(*ast.SetOprStmt)
	if !ok {
		t.Fatal("expected *ast.SetOprStmt")
	}
	all := collectTables(st)
	if !hasTable(all, "", "inner_t") || !hasTable(all, "", "t2") {
		t.Fatalf("unexpected tables: %#v", all)
	}
}

func TestParseSQLSetOpBranchLimit(t *testing.T) {
	st, ok := mustParseOne(t, "(SELECT id FROM a) UNION ALL (SELECT id FROM b LIMIT 5) LIMIT 10;").(*ast.SetOprStmt)
	if !ok {
		t.Fatal("expected *ast.SetOprStmt")
	}
	if st.Limit == nil || st.Limit.Count == nil {
		t.Fatal("expected top-level LIMIT")
	}
	branch := st.SelectList.Selects[1].(*ast.SelectStmt)
	if branch.Limit == nil || branch.Limit.Count == nil {
		t.Fatal("expected branch LIMIT")
	}
}

func TestParseSQLPlaceholders(t *testing.T) {
	tests := []struct {
		name  string
		sql   string
		count int
	}{
		{"dollar placeholder", "SELECT * FROM t WHERE id = $1;", 1},
		{"dollar placeholder", "SELECT * FROM t WHERE id = $2;", 1},
		{"jsonb question operator", "SELECT * FROM t WHERE data ? 'key';", 0},
		{"interval placeholder", "SELECT * FROM t WHERE created_at > now() - INTERVAL '1 day';", 0},
		{"extract unit", "SELECT extract(year FROM created_at) FROM t;", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := mustParseOne(t, tc.sql)
			if got := countParams(st); got != tc.count {
				t.Fatalf("parameter count = %d, want %d", got, tc.count)
			}
		})
	}
}

func TestParseSQLPlaceholdersMultiStatement(t *testing.T) {
	stmts, _, err := pgparser.New().Parse("SELECT * FROM a WHERE id = $1; SELECT * FROM b WHERE status = $1;")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(stmts) != 2 {
		t.Fatalf("statement count = %d, want 2", len(stmts))
	}
	for i, st := range stmts {
		if got := countParams(st); got != 1 {
			t.Fatalf("statement %d parameter count = %d, want 1", i, got)
		}
	}
}

func TestParseSQLPlaceholderEscapes(t *testing.T) {
	st := mustParseSelect(t, "SELECT 'é?' AS marker WHERE id = $1;")
	if got := countParams(st); got != 1 {
		t.Fatalf("parameter count = %d, want 1", got)
	}
}

func TestParseSQLAdvancedExpressionsAndSetOps(t *testing.T) {
	setOp, ok := mustParseOne(t, `
	(
	  SELECT id, CASE WHEN amount > 100 THEN 'high' ELSE 'low' END AS spending_level
	  FROM orders WHERE status = 'paid'
	)
	UNION ALL
	(
	  SELECT id, 'archived' AS spending_level FROM archived_orders
	)
	ORDER BY id;`).(*ast.SetOprStmt)
	if !ok {
		t.Fatal("expected *ast.SetOprStmt")
	}
	right := setOp.SelectList.Selects[1].(*ast.SelectStmt)
	if right.AfterSetOperator == nil || *right.AfterSetOperator != ast.UnionAll {
		t.Fatalf("operator = %v, want UNION ALL", right.AfterSetOperator)
	}
	if setOp.OrderBy == nil {
		t.Fatal("expected ORDER BY on set operation")
	}
	if !hasTable(collectTables(setOp), "", "archived_orders") {
		t.Fatal("expected archived_orders table")
	}

	ins, ok := mustParseOne(t, `INSERT INTO customers (name, email, signup_date)
		VALUES ('John', 'john@example.com', NOW()) RETURNING id, created_at;`).(*ast.InsertStmt)
	if !ok || len(ins.Columns) != 3 || ins.Returning == nil || len(ins.Returning.Fields) != 2 {
		t.Fatalf("unexpected insert: %#v", ins)
	}

	upd, ok := mustParseOne(t, `UPDATE products p
		SET price = price * (1 - d.discount_rate), updated_at = NOW()
		FROM discounts d
		WHERE p.category_id = d.category_id AND d.expires_at > NOW()
		RETURNING p.id, p.price;`).(*ast.UpdateStmt)
	if !ok || len(upd.List) != 2 || upd.From == nil || upd.Returning == nil {
		t.Fatalf("unexpected update: %#v", upd)
	}

	del, ok := mustParseOne(t, `DELETE FROM sessions s USING users u
		WHERE s.user_id = u.id AND u.active = false RETURNING s.id;`).(*ast.DeleteStmt)
	if !ok || del.Using == nil || del.Returning == nil {
		t.Fatalf("unexpected delete: %#v", del)
	}
}

func TestParseSQLExampleLibraryScenarios(t *testing.T) {
	sel, ok := mustParseOne(t, "SELECT * FROM accounts WHERE status = $1;").(*ast.SelectStmt)
	if !ok || sel.From == nil || sel.Where == nil {
		t.Fatal("expected select with FROM and WHERE")
	}
	if len(collectTables(sel)) != 1 || !hasTable(collectTables(sel), "", "accounts") {
		t.Fatalf("unexpected tables: %#v", collectTables(sel))
	}

	join := mustParseSelect(t, "SELECT orders.id, customers.name FROM orders JOIN customers ON customers.id = orders.customer_id;")
	if join.From.TableRefs.On == nil {
		t.Fatal("expected JOIN ON condition")
	}
	if all := collectTables(join); !hasTable(all, "", "orders") || !hasTable(all, "", "customers") {
		t.Fatalf("unexpected join tables: %#v", all)
	}

	cte := mustParseSelect(t, `WITH ranked AS (
		SELECT id, RANK() OVER (ORDER BY created_at) AS r FROM items
	) SELECT * FROM ranked WHERE r <= 10;`)
	if cte.With == nil || len(cte.With.CTEs) != 1 || cte.Where == nil {
		t.Fatalf("unexpected CTE select: %#v", cte)
	}
	if all := collectTables(cte); !hasTable(all, "", "items") || !hasTable(all, "", "ranked") {
		t.Fatalf("unexpected CTE tables: %#v", all)
	}

	upd, ok := mustParseOne(t, "UPDATE products SET price = price * 1.1 WHERE category = $1 RETURNING id;").(*ast.UpdateStmt)
	if !ok || len(upd.List) != 1 || upd.Returning == nil {
		t.Fatalf("unexpected update: %#v", upd)
	}

	del, ok := mustParseOne(t, `DELETE FROM sessions USING users
		WHERE sessions.user_id = users.id AND users.active = false RETURNING sessions.id;`).(*ast.DeleteStmt)
	if !ok || del.Using == nil || del.Returning == nil {
		t.Fatalf("unexpected delete: %#v", del)
	}
	if all := collectTables(del); !hasTable(all, "", "users") {
		t.Fatalf("expected users in USING: %#v", all)
	}

	ins, ok := mustParseOne(t, "INSERT INTO metrics (id, payload) VALUES ($1, $2) RETURNING id;").(*ast.InsertStmt)
	if !ok || len(ins.Columns) != 2 || ins.Returning == nil || len(ins.Lists) != 1 || len(ins.Lists[0]) != 2 {
		t.Fatalf("unexpected insert: %#v", ins)
	}
	if got := countParams(ins); got != 2 {
		t.Fatalf("insert parameters = %d, want 2", got)
	}
}

func TestParseSQLUpsertTargetAndActionWhere(t *testing.T) {
	ins, ok := mustParseOne(t, `INSERT INTO accounts (id, email, active)
		VALUES ($1, $2, true)
		ON CONFLICT (id) WHERE active = true
		DO UPDATE SET email = EXCLUDED.email
		WHERE accounts.active = true;`).(*ast.InsertStmt)
	if !ok || ins.OnConflict == nil {
		t.Fatalf("expected ON CONFLICT, got %#v", ins)
	}
	oc := ins.OnConflict
	if oc.DoNothing || len(oc.UpdateList) != 1 || oc.TargetWhere == nil || oc.UpdateWhere == nil {
		t.Fatalf("unexpected ON CONFLICT clause: %#v", oc)
	}
	if len(oc.IndexCols) != 1 || oc.IndexCols[0].Column.Name.L != "id" {
		t.Fatalf("unexpected conflict target: %#v", oc.IndexCols)
	}
}

func TestParseSQLInsertOnConflictCapturesSetClauses(t *testing.T) {
	ins, ok := mustParseOne(t, `INSERT INTO account_balances (id, balance)
		VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE
		SET balance = EXCLUDED.balance + $3, updated_at = NOW()
		RETURNING id, balance;`).(*ast.InsertStmt)
	if !ok || ins.OnConflict == nil {
		t.Fatal("expected insert with ON CONFLICT")
	}
	oc := ins.OnConflict
	if len(ins.Columns) != 2 || len(oc.UpdateList) != 2 || len(oc.IndexCols) != 1 {
		t.Fatalf("unexpected insert: %#v", ins)
	}
	if got := countParams(ins); got != 3 {
		t.Fatalf("parameters = %d, want 3", got)
	}
	if ins.Returning == nil || len(ins.Returning.Fields) != 2 {
		t.Fatalf("unexpected RETURNING: %#v", ins.Returning)
	}
}

func TestParseSQLInsertOnConflictDoNothing(t *testing.T) {
	ins, ok := mustParseOne(t, `INSERT INTO accounts (id, email)
		VALUES ($1, $2)
		ON CONFLICT (id) WHERE is_active = TRUE DO NOTHING;`).(*ast.InsertStmt)
	if !ok || ins.OnConflict == nil {
		t.Fatal("expected insert with ON CONFLICT")
	}
	oc := ins.OnConflict
	if !oc.DoNothing || len(oc.UpdateList) != 0 || oc.TargetWhere == nil {
		t.Fatalf("unexpected ON CONFLICT: %#v", oc)
	}
	if len(oc.IndexCols) != 1 || oc.IndexCols[0].Column.Name.L != "id" {
		t.Fatalf("unexpected conflict target: %#v", oc.IndexCols)
	}
}

func TestParseSQLSavepointAndRoutineStatements(t *testing.T) {
	if st, ok := mustParseOne(t, "SAVEPOINT sp;").(*ast.SavepointStmt); !ok || st.Name != "sp" {
		t.Fatalf("savepoint = %#v", mustParseOne(t, "SAVEPOINT sp;"))
	}
	if st, ok := mustParseOne(t, "RELEASE SAVEPOINT sp;").(*ast.ReleaseSavepointStmt); !ok || st.Name != "sp" {
		t.Fatalf("release savepoint = %#v", mustParseOne(t, "RELEASE SAVEPOINT sp;"))
	}
	if st, ok := mustParseOne(t, "RELEASE sp;").(*ast.ReleaseSavepointStmt); !ok || st.Name != "sp" {
		t.Fatalf("release = %#v", mustParseOne(t, "RELEASE sp;"))
	}
	if st, ok := mustParseOne(t, "DO $$ BEGIN NULL; END $$;").(*ast.DoStmt); !ok || len(st.Exprs) != 1 {
		t.Fatalf("do = %#v", mustParseOne(t, "DO $$ BEGIN NULL; END $$;"))
	}
	if st, ok := mustParseOne(t, "DROP FUNCTION IF EXISTS public.f(int);").(*ast.DropFunctionStmt); !ok || !st.IfExists || st.Name.Name.O != "f" {
		t.Fatalf("drop function = %#v", mustParseOne(t, "DROP FUNCTION IF EXISTS public.f(int);"))
	}
	if st, ok := mustParseOne(t, "DROP PROCEDURE p();").(*ast.DropProcedureStmt); !ok || st.Name.Name.O != "p" {
		t.Fatalf("drop procedure = %#v", mustParseOne(t, "DROP PROCEDURE p();"))
	}
	if st, ok := mustParseOne(t, "DROP TRIGGER IF EXISTS trg ON t;").(*ast.DropTriggerStmt); !ok || !st.IfExists {
		t.Fatalf("drop trigger = %#v", mustParseOne(t, "DROP TRIGGER IF EXISTS trg ON t;"))
	}
}

func TestParseSQLUserRoleAndPreparedStatements(t *testing.T) {
	if st, ok := mustParseOne(t, "CREATE USER u PASSWORD 'x';").(*ast.CreateUserStmt); !ok || st.Specs[0].AuthOpt == nil || !st.Specs[0].AuthOpt.Password {
		t.Fatalf("create user = %#v", mustParseOne(t, "CREATE USER u PASSWORD 'x';"))
	}
	if st, ok := mustParseOne(t, "CREATE ROLE r WITH PASSWORD 'x';").(*ast.CreateUserStmt); !ok || !st.IsCreateRole || !st.Specs[0].AuthOpt.Password {
		t.Fatalf("create role = %#v", mustParseOne(t, "CREATE ROLE r WITH PASSWORD 'x';"))
	}
	if st, ok := mustParseOne(t, "ALTER ROLE r;").(*ast.AlterUserStmt); !ok || !st.IsAlterRole {
		t.Fatalf("alter role = %#v", mustParseOne(t, "ALTER ROLE r;"))
	}
	if st, ok := mustParseOne(t, "DROP USER IF EXISTS u, v;").(*ast.DropUserStmt); !ok || !st.IfExists || len(st.UserList) != 2 {
		t.Fatalf("drop user = %#v", mustParseOne(t, "DROP USER IF EXISTS u, v;"))
	}
	if st, ok := mustParseOne(t, "DROP ROLE r;").(*ast.DropUserStmt); !ok || !st.IsDropRole {
		t.Fatalf("drop role = %#v", mustParseOne(t, "DROP ROLE r;"))
	}

	prep := mustParseOne(t, "PREPARE p (int, text) AS INSERT INTO t VALUES ($1, $2);")
	st, ok := prep.(*ast.PrepareStmt)
	if !ok || st.Name != "p" || len(st.Params) != 2 || st.Stmt == nil {
		t.Fatalf("prepare = %#v", prep)
	}
	if st.Params[0].String() != "int(11)" || st.Params[1].String() != "text" {
		t.Fatalf("prepare params = %#v", st.Params)
	}
	if st, ok := mustParseOne(t, "EXECUTE p(1, 'x');").(*ast.ExecuteStmt); !ok || len(st.Args) != 2 {
		t.Fatalf("execute = %#v", mustParseOne(t, "EXECUTE p(1, 'x');"))
	}
	if st, ok := mustParseOne(t, "DEALLOCATE PREPARE p;").(*ast.DeallocateStmt); !ok || st.Name != "p" || st.All {
		t.Fatalf("deallocate = %#v", mustParseOne(t, "DEALLOCATE PREPARE p;"))
	}
	if st, ok := mustParseOne(t, "DEALLOCATE ALL;").(*ast.DeallocateStmt); !ok || !st.All {
		t.Fatalf("deallocate all = %#v", mustParseOne(t, "DEALLOCATE ALL;"))
	}

	if got := compactSQL(restoreNode(t, prep)); got != "PREPARE p (INT, TEXT) AS INSERT INTO t VALUES (?,?)" {
		t.Fatalf("prepare restore = %q", got)
	}
	if got := compactSQL(restoreNode(t, mustParseOne(t, "CREATE USER u PASSWORD 'x';"))); got != "CREATE USER u PASSWORD 'x'" {
		t.Fatalf("create user restore = %q", got)
	}
}

func TestParseSQLReferenceUtilityStatementFamilies(t *testing.T) {
	cases := []struct {
		sql  string
		want interface{}
	}{
		{`GRANT SELECT, UPDATE ON t TO u WITH GRANT OPTION;`, &ast.GrantStmt{}},
		{`REVOKE SELECT ON t FROM u;`, &ast.RevokeStmt{}},
		{`GRANT admin TO u;`, &ast.GrantRoleStmt{}},
		{`REVOKE admin FROM u;`, &ast.RevokeRoleStmt{}},
		{`VALUES (1), (2);`, &ast.ValuesStmt{}},
		{`TABLE t;`, &ast.TableStmt{}},
		{`CREATE TABLESPACE ts LOCATION '/data/ts';`, &ast.CreateTablespaceStmt{}},
		{`ALTER TABLESPACE ts RENAME TO ts2;`, &ast.AlterTablespaceStmt{}},
		{`DROP TABLESPACE IF EXISTS ts2;`, &ast.DropTablespaceStmt{}},
		{`CREATE SERVER s FOREIGN DATA WRAPPER fdw OPTIONS (host 'localhost');`, &ast.CreateServerStmt{}},
		{`ALTER SERVER s OPTIONS (host '127.0.0.1');`, &ast.AlterServerStmt{}},
		{`DROP SERVER IF EXISTS s;`, &ast.DropServerStmt{}},
		{`CREATE RESOURCE GROUP rg TYPE = SYSTEM;`, &ast.CreateResourceGroupStmt{}},
		{`ALTER RESOURCE GROUP rg;`, &ast.AlterResourceGroupStmt{}},
		{`DROP RESOURCE GROUP rg FORCE;`, &ast.DropResourceGroupStmt{}},
		{`CREATE DATABASE db;`, &ast.CreateDatabaseStmt{}},
		{`ALTER DATABASE db RENAME TO db2;`, &ast.AlterDatabaseStmt{}},
	}
	for _, tt := range cases {
		got := mustParseOne(t, tt.sql)
		if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
			t.Errorf("%q: got %T, want %T", tt.sql, got, tt.want)
		}
	}
}

func TestParseSQLSetRoleAndPrepare(t *testing.T) {
	if st, ok := mustParseOne(t, `SET ROLE app;`).(*ast.SetRoleStmt); !ok || len(st.RoleList) != 1 {
		t.Fatalf("set role = %#v", mustParseOne(t, `SET ROLE app;`))
	}
	if st, ok := mustParseOne(t, `SET ROLE NONE;`).(*ast.SetRoleStmt); !ok || st.SetRoleOpt != ast.SetRoleNone {
		t.Fatalf("set role none = %#v", mustParseOne(t, `SET ROLE NONE;`))
	}
	if st, ok := mustParseOne(t, `SET ROLE DEFAULT;`).(*ast.SetRoleStmt); !ok || st.SetRoleOpt != ast.SetRoleDefault {
		t.Fatalf("set role default = %#v", mustParseOne(t, `SET ROLE DEFAULT;`))
	}
	if st, ok := mustParseOne(t, `SET SESSION AUTHORIZATION app;`).(*ast.SetStmt); !ok || len(st.Variables) != 1 || st.Variables[0].Name != "session_authorization" {
		t.Fatalf("set session authorization = %#v", mustParseOne(t, `SET SESSION AUTHORIZATION app;`))
	}
	if st, ok := mustParseOne(t, `PREPARE p AS SELECT 1;`).(*ast.PrepareStmt); !ok || st.Name != "p" || st.Stmt == nil {
		t.Fatalf("prepare = %#v", mustParseOne(t, `PREPARE p AS SELECT 1;`))
	}
}
