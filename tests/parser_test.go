package parser_test

import (
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/pgparser/ast"
	_ "github.com/mia0x75/pgparser/driver"
	pgparser "github.com/mia0x75/pgparser/parser"
)

func TestParse(t *testing.T) {
	stmts, err := pgparser.New().Parse("SELECT 1")
	if err != nil || len(stmts) != 1 {
		t.Fatalf("Parse returned %d statements, error %v", len(stmts), err)
	}
}

func TestParseResultNotOverwrittenByNextCall(t *testing.T) {
	shared := pgparser.New()
	first, err := shared.Parse("SELECT 1")
	if err != nil {
		t.Fatal(err)
	}
	firstStmt := first[0]
	if _, err := shared.Parse("SELECT 2"); err != nil {
		t.Fatal(err)
	}
	if first[0] != firstStmt {
		t.Fatal("a later parse replaced an earlier result")
	}
}

func TestParserConcurrentUse(t *testing.T) {
	const workers, iterations = 8, 20
	shared := pgparser.New()
	var group sync.WaitGroup
	group.Add(workers)
	for worker := range workers {
		go func(worker int) {
			defer group.Done()
			for iteration := range iterations {
				sql := "SELECT " + strings.Repeat("1", worker*iterations+iteration+1)
				stmt, err := shared.ParseOneStmt(sql)
				if err != nil {
					t.Errorf("ParseOneStmt returned error: %v", err)
					return
				}
				if got, want := stmt.Span(), (loc.Span{Start: 0, End: loc.ByteOffset(len(sql))}); got != want {
					t.Errorf("statement span = %v, want %v", got, want)
				}
			}
		}(worker)
	}
	group.Wait()
}

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

// limitClauseNode returns the dialect limit of a query block.
func limitClauseNode(t *testing.T, node ast.LimitNode) *ast.Limit {
	t.Helper()
	limit, ok := node.(*ast.Limit)
	if !ok {
		t.Fatalf("limit = %T, want *ast.Limit", node)
	}
	return limit
}

// fromClause returns the dialect from clause of a query block.
func fromClause(t *testing.T, node ast.FromClauseNode) *ast.FromClause {
	t.Helper()
	clause, ok := node.(*ast.FromClause)
	if !ok {
		t.Fatalf("from clause = %T, want *ast.FromClause", node)
	}
	return clause
}

// groupClause returns the dialect grouping clause of a query block.
func groupClause(t *testing.T, node ast.GroupByClauseNode) *ast.GroupByClause {
	t.Helper()
	clause, ok := node.(*ast.GroupByClause)
	if !ok {
		t.Fatalf("group by clause = %T, want *ast.GroupByClause", node)
	}
	return clause
}

// orderClause returns the dialect ordering clause of a query block.
func orderClause(t *testing.T, node ast.OrderByClauseNode) *ast.OrderByClause {
	t.Helper()
	clause, ok := node.(*ast.OrderByClause)
	if !ok {
		t.Fatalf("order by clause = %T, want *ast.OrderByClause", node)
	}
	return clause
}

// windowSpec returns the dialect window specification of a query block.
func windowSpec(t *testing.T, node ast.WindowSpecNode) *ast.WindowSpec {
	t.Helper()
	spec, ok := node.(*ast.WindowSpec)
	if !ok {
		t.Fatalf("window spec = %T, want *ast.WindowSpec", node)
	}
	return spec
}

// groupItem returns the dialect grouping item of a group by list.
func groupItem(t *testing.T, node ast.GroupItemNode) *ast.GroupItem {
	t.Helper()
	item, ok := node.(*ast.GroupItem)
	if !ok {
		t.Fatalf("group item = %T, want *ast.GroupItem", node)
	}
	return item
}

// orderItem returns the dialect sorting item of an order by list.
func orderItem(t *testing.T, node ast.OrderItemNode) *ast.OrderItem {
	t.Helper()
	item, ok := node.(*ast.OrderItem)
	if !ok {
		t.Fatalf("order item = %T, want *ast.OrderItem", node)
	}
	return item
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

	if st.Fields == nil || len(st.Fields.Items) != 2 {
		t.Fatalf("expected 2 columns, got %#v", st.Fields)
	}
	if got := st.Fields.Items[0].Expr.(*ast.ColumnNameExpr).Name.Parts[0].Name; got != "id" {
		t.Fatalf("column 1 = %q, want id", got)
	}
	if got := st.Fields.Items[1].Expr.(*ast.ColumnNameExpr).Name.Parts[0].Name; got != "name" {
		t.Fatalf("column 2 = %q, want name", got)
	}
	if st.From == nil || fromClause(t, st.From).Table == nil {
		t.Fatal("expected FROM clause")
	}
	left, ok := fromClause(t, st.From).Table.(*ast.TableSource)
	if !ok {
		t.Fatalf("expected TableSource, got %T", fromClause(t, st.From).Table)
	}
	if got := left.Source.(*ast.TableName).Name.Parts[0].Name; got != "users" {
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

	if st.From == nil || fromClause(t, st.From).Table == nil {
		t.Fatal("expected FROM clause")
	}
	left, ok := leftmostTableSource(fromClause(t, st.From).Table)
	if !ok {
		t.Fatalf("expected left TableSource, got %T", fromClause(t, st.From).Table)
	}
	if got := left.Source.(*ast.TableName).Name.Parts[0].Name; got != "orders" {
		t.Fatalf("left table = %q, want orders", got)
	}
	if got := left.Alias.Name; got != "o" {
		t.Fatalf("left alias = %q, want o", got)
	}
	join, ok := fromClause(t, st.From).Table.(*ast.Join)
	if !ok {
		t.Fatalf("expected join, got %T", fromClause(t, st.From).Table)
	}
	right, ok := join.Right.(*ast.TableSource)
	if !ok {
		t.Fatalf("expected right TableSource, got %T", join.Right)
	}
	if got := right.Source.(*ast.TableName).Name.Parts[0].Name; got != "customers" {
		t.Fatalf("right table = %q, want customers", got)
	}
	if got := right.Alias.Name; got != "c" {
		t.Fatalf("right alias = %q, want c", got)
	}
	if join.On == nil {
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

	if st.GroupBy == nil || len(groupClause(t, st.GroupBy).Items) != 1 {
		t.Fatalf("expected 1 GROUP BY item, got %#v", st.GroupBy)
	}
	if st.Having == nil {
		t.Fatal("expected HAVING clause")
	}
	if st.OrderBy == nil || len(orderClause(t, st.OrderBy).Items) != 1 {
		t.Fatalf("expected 1 ORDER BY item, got %#v", st.OrderBy)
	}
	if !orderItem(t, orderClause(t, st.OrderBy).Items[0]).Desc {
		t.Fatal("expected ORDER BY ... DESC")
	}
	if st.Limit == nil {
		t.Fatal("expected LIMIT clause")
	}
	if limitClauseNode(t, st.Limit).Count == nil || limitClauseNode(t, st.Limit).Offset == nil {
		t.Fatalf("expected LIMIT and OFFSET, got %#v", st.Limit)
	}
}

// Ported from parser_test.go multi-statement behavior.
func TestParseSQLMultiStatement(t *testing.T) {
	stmts, err := pgparser.New().Parse("SELECT 1; SELECT 2;")
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
	if len(ins.Values) != 2 {
		t.Fatalf("expected 2 value rows, got %d", len(ins.Values))
	}

	upd, ok := mustParseOne(t, "UPDATE t SET a = 1, b = 2 WHERE c = 3;").(*ast.UpdateStmt)
	if !ok {
		t.Fatal("expected *ast.UpdateStmt")
	}
	if len(upd.Assignments) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(upd.Assignments))
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
				if _, ok := st.Fields.Items[0].Expr.(*ast.CaseExpr); !ok {
					t.Fatalf("expected *ast.CaseExpr, got %T", st.Fields.Items[0].Expr)
				}
			},
		},
		{
			name: "function call",
			sql:  "SELECT COUNT(*) FROM t;",
			check: func(t *testing.T, st *ast.SelectStmt) {
				if _, ok := st.Fields.Items[0].Expr.(*ast.AggregateFuncExpr); !ok {
					t.Fatalf("expected *ast.AggregateFuncExpr, got %T", st.Fields.Items[0].Expr)
				}
			},
		},
		{
			name: "qualified column",
			sql:  "SELECT t.a FROM t;",
			check: func(t *testing.T, st *ast.SelectStmt) {
				e, ok := st.Fields.Items[0].Expr.(*ast.ColumnNameExpr)
				if !ok {
					t.Fatalf("expected *ast.ColumnNameExpr, got %T", st.Fields.Items[0].Expr)
				}
				if len(e.Name.Parts) != 2 || e.Name.Parts[0].Name != "t" || e.Name.Parts[1].Name != "a" {
					t.Fatalf("got %#v, want t.a", e.Name)
				}
			},
		},
		{
			name: "qualified wildcard",
			sql:  "SELECT t.* FROM t;",
			check: func(t *testing.T, st *ast.SelectStmt) {
				w, ok := st.Fields.Items[0].Expr.(*ast.WildcardExpr)
				if !ok || w == nil {
					t.Fatalf("expected t.*, got %#v", st.Fields.Items[0])
				}
				if len(w.Qualifier.Parts) != 1 || w.Qualifier.Parts[0].Name != "t" {
					t.Fatalf("expected t.*, got %#v", st.Fields.Items[0])
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
	stmts, err := pgparser.New().Parse("")
	if err != nil {
		t.Fatalf("empty input: unexpected error: %v", err)
	}
	if len(stmts) != 0 {
		t.Fatalf("empty input: expected 0 statements, got %d", len(stmts))
	}
}

func TestParseSQLQuotedIdentifier(t *testing.T) {
	st := mustParseSelect(t, `SELECT "MyColumn" FROM t;`)
	e, ok := st.Fields.Items[0].Expr.(*ast.ColumnNameExpr)
	if !ok {
		t.Fatalf("expected *ast.ColumnNameExpr, got %T", st.Fields.Items[0].Expr)
	}
	if got := e.Name.Parts[0].Name; got != "MyColumn" {
		t.Fatalf("quoted identifier = %q, want MyColumn", got)
	}
}

func TestParseSQLAlterTable(t *testing.T) {
	stmt, ok := mustParseOne(t, "ALTER TABLE s.foo ADD COLUMN bar INT NOT NULL, DROP COLUMN baz;").(*ast.AlterTableStmt)
	if !ok {
		t.Fatalf("expected *ast.AlterTableStmt, got %T", mustParseOne(t, "ALTER TABLE s.foo ADD COLUMN bar INT;"))
	}
	if stmt.Table.Name.Parts[0].Name != "s" || stmt.Table.Name.Parts[1].Name != "foo" {
		t.Fatalf("table = %s.%s, want s.foo", stmt.Table.Name.Parts[0].Name, stmt.Table.Name.Parts[1].Name)
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
	if !ok || rename.Relation == nil || rename.Relation.Name.Parts[len(rename.Relation.Name.Parts)-1].Name != "foo" || rename.Newname != "qux" {
		t.Fatalf("unexpected rename: %#v", mustParseOne(t, "ALTER TABLE s.foo RENAME TO qux;"))
	}
}

func TestParseSQLDropAndTruncate(t *testing.T) {
	dropTable, ok := mustParseOne(t, "DROP TABLE IF EXISTS foo, s.bar;").(*ast.DropTableStmt)
	if !ok {
		t.Fatal("expected *ast.DropTableStmt")
	}
	if !dropTable.IfExists || len(dropTable.Tables) != 2 || dropTable.Tables[1].Name.Parts[0].Name != "s" {
		t.Fatalf("unexpected drop table: %#v", dropTable)
	}

	dropView, ok := mustParseOne(t, "DROP VIEW v;").(*ast.DropTableStmt)
	if !ok || !dropView.IsView {
		t.Fatalf("expected view drop, got %#v", dropView)
	}

	dropIndex, ok := mustParseOne(t, "DROP INDEX IF EXISTS s.ix;").(*ast.DropIndexStmt)
	if !ok || !dropIndex.IfExists || len(dropIndex.IndexName.Parts) != 2 || dropIndex.IndexName.Parts[1].Name != "ix" {
		t.Fatalf("unexpected drop index: %#v", dropIndex)
	}

	truncate, ok := mustParseOne(t, "TRUNCATE TABLE s.t;").(*ast.TruncateTableStmt)
	if !ok || truncate.Table.Name.Parts[0].Name != "s" || truncate.Table.Name.Parts[1].Name != "t" {
		t.Fatalf("unexpected truncate: %#v", truncate)
	}
}

func TestParseSQLCreateIndexAndView(t *testing.T) {
	idx, ok := mustParseOne(t, "CREATE UNIQUE INDEX IF NOT EXISTS ix ON s.t (a, b);").(*ast.CreateIndexStmt)
	if !ok || !idx.Unique || len(idx.IndexName.Parts) != 1 || idx.IndexName.Parts[0].Name != "ix" || idx.Table.Name.Parts[0].Name != "s" {
		t.Fatalf("unexpected create index: %#v", idx)
	}
	if len(idx.IndexColNames) != 2 {
		t.Fatalf("index cols = %d, want 2", len(idx.IndexColNames))
	}

	view, ok := mustParseOne(t, "CREATE OR REPLACE VIEW v (a, b) AS SELECT a, b FROM t;").(*ast.CreateViewStmt)
	if !ok || !view.OrReplace || view.ViewName.Name.Parts[0].Name != "v" {
		t.Fatalf("unexpected create view: %#v", view)
	}
	if len(view.Cols) != 2 || view.Cols[0].Name != "a" {
		t.Fatalf("view cols = %#v, want [a b]", view.Cols)
	}
	if _, ok := view.Select.(*ast.SelectStmt); !ok {
		t.Fatalf("expected select body, got %T", view.Select)
	}
}

func TestParseSQLSequence(t *testing.T) {
	create, ok := mustParseOne(t, "CREATE SEQUENCE IF NOT EXISTS s.seq;").(*ast.CreateSequenceStmt)
	if !ok || !create.IfNotExists || create.Name.Name.Parts[0].Name != "s" || create.Name.Name.Parts[1].Name != "seq" {
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
	cte, ok := st.With.CTEs[0].(*ast.CommonTableExpression)
	if !ok {
		t.Fatalf("unexpected cte type %T", st.With.CTEs[0])
	}
	if cte.Name.Name != "t" || len(cte.Columns) != 1 || cte.Columns[0].Name != "n" {
		t.Fatalf("unexpected cte: %#v", cte)
	}
	if _, ok := cte.Query.(*ast.SetOprStmt); !ok {
		t.Fatalf("expected set operation in cte, got %T", cte.Query)
	}
}

func TestParseSQLInsertSelectAndReturning(t *testing.T) {
	ins, ok := mustParseOne(t, "INSERT INTO s.t (a, b) SELECT a, b FROM u RETURNING a;").(*ast.InsertStmt)
	if !ok || ins.Table.(*ast.TableName).Name.Parts[0].Name != "s" {
		t.Fatalf("unexpected insert: %#v", ins)
	}
	if len(ins.Columns) != 2 || ins.Query == nil || ins.Returning == nil {
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
	if st.Operator != ast.Except {
		t.Fatalf("outer set operator = %v, want EXCEPT", st.Operator)
	}
	inner, ok := st.Left.(*ast.SetOprStmt)
	if !ok {
		t.Fatalf("left operand = %T, want *ast.SetOprStmt", st.Left)
	}
	if inner.Operator != ast.UnionAll {
		t.Fatalf("inner set operator = %v, want UNION ALL", inner.Operator)
	}
}

func TestParseSQLUtilityAndTransactions(t *testing.T) {
	set, ok := mustParseOne(t, "SET search_path TO public;").(*ast.VariableSetStmt)
	if !ok || set.Name != "search_path" || set.Args == nil || len(set.Args.Items) != 1 {
		t.Fatalf("unexpected set: %#v", set)
	}
	if _, ok := mustParseOne(t, "SET a.b = 'c';").(*ast.VariableSetStmt); !ok {
		t.Fatal("expected *ast.VariableSetStmt")
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
	if _, ok := fromClause(t, upd.From).Table.(*ast.Join); !ok {
		t.Fatalf("expected joined FROM tables, got %#v", fromClause(t, upd.From).Table)
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
	if !ok || c.ObjectType != "column" || !reflect.DeepEqual(c.ObjectName, []string{"s", "t", "c"}) || c.Comment == nil || *c.Comment != "hello" {
		t.Fatalf("unexpected comment: %#v", c)
	}
	c = mustParseOne(t, `COMMENT ON COLUMN db."MySchema"."MyTable"."MyColumn" IS NULL`).(*ast.CommentStmt)
	if want := []string{"MySchema", "MyTable", "MyColumn"}; !reflect.DeepEqual(c.ObjectName, want) {
		t.Fatalf("legacy column comment target = %v, want %v", c.ObjectName, want)
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
	if !ok || !db.IfNotExists || db.Name.Name != "db" {
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
	ce, ok := simple.Fields.Items[0].Expr.(*ast.CaseExpr)
	if !ok {
		t.Fatalf("expected *ast.CaseExpr, got %T", simple.Fields.Items[0].Expr)
	}
	if ce.Value == nil || len(ce.WhenClauses) != 2 || ce.ElseClause == nil {
		t.Fatalf("unexpected simple case: %#v", ce)
	}

	searched := mustParseSelect(t, "SELECT CASE WHEN a = 1 THEN 'x' WHEN a = 2 THEN 'y' END FROM t;")
	ce, ok = searched.Fields.Items[0].Expr.(*ast.CaseExpr)
	if !ok || ce.Value != nil || len(ce.WhenClauses) != 2 || ce.ElseClause != nil {
		t.Fatalf("unexpected searched case: %#v", ce)
	}
}

func TestParseSQLGroupOrderAndIn(t *testing.T) {
	st := mustParseSelect(t, "SELECT a FROM t GROUP BY a, b, c ORDER BY a DESC, b, c ASC, d LIMIT ALL;")
	if st.GroupBy == nil || len(groupClause(t, st.GroupBy).Items) != 3 {
		t.Fatalf("group by items = %#v, want 3", st.GroupBy)
	}
	if st.OrderBy == nil || len(orderClause(t, st.OrderBy).Items) != 4 {
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
	if !ok || st.Left == nil || st.Right == nil {
		t.Fatalf("unexpected set operation: %#v", st)
	}
	single, ok := mustParseOne(t, "(SELECT 1);").(*ast.SelectStmt)
	if !ok || !single.IsInBraces {
		t.Fatalf("unexpected braced select: %#v", single)
	}
}

func TestParseSQLSelectDistinctAndJoinUsing(t *testing.T) {
	st := mustParseSelect(t, "SELECT DISTINCT a FROM t JOIN u USING (id, tenant_id);")
	if !st.Distinct {
		t.Fatal("expected DISTINCT")
	}
	if st.From == nil || fromClause(t, st.From).Table == nil {
		t.Fatalf("expected join USING, got %#v", st.From)
	}
	join, ok := fromClause(t, st.From).Table.(*ast.Join)
	if !ok || len(join.Using) != 2 {
		t.Fatalf("expected join USING, got %#v", fromClause(t, st.From).Table)
	}
	if got := join.Using[0].Parts[0].Name + "," + join.Using[1].Parts[0].Name; got != "id,tenant_id" {
		t.Fatalf("using columns = %q, want id,tenant_id", got)
	}
}

func TestParseSQLWindowAndNullsOrder(t *testing.T) {
	st := mustParseSelect(t, "SELECT ROW_NUMBER() OVER (PARTITION BY dept ORDER BY salary DESC NULLS LAST) FROM employees;")
	w, ok := st.Fields.Items[0].Expr.(*ast.WindowFuncExpr)
	if !ok {
		t.Fatalf("expected *ast.WindowFuncExpr, got %T", st.Fields.Items[0].Expr)
	}
	if w.Spec.PartitionBy == nil || w.Spec.OrderBy == nil || len(w.Spec.OrderBy.Items) != 1 {
		t.Fatalf("unexpected window spec: %#v", w.Spec)
	}
	if orderItem(t, w.Spec.OrderBy.Items[0]).NullOrder != 2 || !orderItem(t, w.Spec.OrderBy.Items[0]).Desc {
		t.Fatalf("unexpected null ordering: %#v", w.Spec.OrderBy.Items[0])
	}
}

func TestParseSQLJsonAndLikeOperators(t *testing.T) {
	st := mustParseSelect(t, "SELECT data->'a'->>'b' FROM t WHERE name ILIKE 'x%' AND data @> '{\"k\":1}';")
	if _, ok := st.Fields.Items[0].Expr.(*ast.BinaryOperationExpr); !ok {
		t.Fatalf("expected json binary expression, got %T", st.Fields.Items[0].Expr)
	}
	if _, ok := st.Where.(*ast.BinaryOperationExpr); !ok {
		t.Fatalf("expected and expression, got %T", st.Where)
	}
}

func TestParseSQLCastAndExtract(t *testing.T) {
	st := mustParseSelect(t, "SELECT length(name)::int, EXTRACT(year FROM now()) FROM t WHERE a::int < 5;")
	if _, ok := st.Fields.Items[0].Expr.(*ast.FuncCastExpr); !ok {
		t.Fatalf("expected cast expression, got %T", st.Fields.Items[0].Expr)
	}
	fc, ok := st.Fields.Items[1].Expr.(*ast.FuncCallExpr)
	if !ok || len(fc.Name.Parts) != 1 || fc.Name.Parts[0].Name != "extract" {
		t.Fatalf("expected extract call, got %#v", st.Fields.Items[1].Expr)
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
		if got := col.Tp.String(); got != `"varchar"(20)` {
			t.Fatalf("column %d type = %q, want varchar(20)", i, got)
		}
	}

	st := mustParseSelect(t, "SELECT 'x'::CHARACTER VARYING(20), CAST('x' AS CHARACTER VARYING(20)), a::int b FROM t;")
	for i := range 2 {
		fc, ok := st.Fields.Items[i].Expr.(*ast.FuncCastExpr)
		if !ok {
			t.Fatalf("field %d: expected cast, got %T", i, st.Fields.Items[i].Expr)
		}
		if fc.Tp == nil || fc.Tp.String() != `"varchar"(20)` {
			t.Fatalf("field %d: type = %#v, want varchar(20)", i, fc.Tp)
		}
	}
	if got := st.Fields.Items[2].Alias.Name; got != "b" {
		t.Fatalf("alias = %q, want b", got)
	}
}

func TestParseSQLQualifiedFunctionAndSoftKeywords(t *testing.T) {
	st := mustParseSelect(t, "SELECT pg_catalog.lower(name), myschema.length(col) FROM t WHERE role = 'admin';")
	if len(st.Fields.Items) != 2 {
		t.Fatalf("fields = %d, want 2", len(st.Fields.Items))
	}
	fc, ok := st.Fields.Items[0].Expr.(*ast.FuncCallExpr)
	if !ok || len(fc.Name.Parts) != 2 || fc.Name.Parts[0].Name != "pg_catalog" || fc.Name.Parts[1].Name != "lower" {
		t.Fatalf("expected qualified function, got %#v", st.Fields.Items[0].Expr)
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
	wf, ok := w.Fields.Items[0].Expr.(*ast.WindowFuncExpr)
	if !ok || wf.Spec.Frame == nil || wf.Spec.Frame.Type != ast.Rows {
		t.Fatalf("unexpected window frame: %#v", w.Fields.Items[0].Expr)
	}
	start := wf.Spec.Frame.Extent.Start.WindowFrameBoundCore()
	end := wf.Spec.Frame.Extent.End.WindowFrameBoundCore()
	if !start.Unbounded || end.Type != ast.CurrentRow {
		t.Fatalf("unexpected frame extent: %#v", wf.Spec.Frame.Extent)
	}
}

func TestParseSQLLikeInternalAndNaturalJoin(t *testing.T) {
	st := mustParseSelect(t, "SELECT * FROM a NATURAL LEFT JOIN b WHERE name !~~ 'x%';")
	if st.From == nil || fromClause(t, st.From).Table == nil {
		t.Fatalf("expected natural join, got %#v", st.From)
	}
	join, ok := fromClause(t, st.From).Table.(*ast.Join)
	if !ok || !join.Natural {
		t.Fatalf("expected natural join, got %#v", fromClause(t, st.From).Table)
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
	if !ok || len(c.ObjectName) == 0 {
		t.Fatalf("unexpected comment: %#v", c)
	}
	for _, sql := range []string{
		"SET LOCAL client_min_messages = DEBUG;",
		"SET SESSION client_min_messages TO notice;",
		"SET SESSION AUTHORIZATION postgres;",
	} {
		if _, ok := mustParseOne(t, sql).(*ast.VariableSetStmt); !ok {
			t.Fatalf("expected *ast.VariableSetStmt for %q", sql)
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
	fc, ok := st.Fields.Items[0].Expr.(*ast.FuncCallExpr)
	if !ok || len(fc.Name.Parts) != 1 || fc.Name.Parts[0].Name != "timezone" {
		t.Fatalf("expected timezone call, got %#v", st.Fields.Items[0].Expr)
	}
}

func TestParseSQLLateralAndTableFunctions(t *testing.T) {
	st := mustParseSelect(t, "SELECT * FROM products p CROSS JOIN LATERAL unnest(p.tags) AS tag(name);")
	join, ok := fromClause(t, st.From).Table.(*ast.Join)
	if !ok {
		t.Fatalf("expected lateral join, got %#v", fromClause(t, st.From).Table)
	}
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
	if right.Alias.Name != "tag" {
		t.Fatalf("alias = %q, want tag", right.Alias.Name)
	}
	if len(right.Columns) != 1 || right.Columns[0].Name != "name" {
		t.Fatalf("column aliases = %#v, want [name]", right.Columns)
	}

	st = mustParseSelect(t, "SELECT * FROM t, LATERAL (SELECT * FROM u WHERE u.tid = t.id) sub;")
	join, ok = fromClause(t, st.From).Table.(*ast.Join)
	if !ok {
		t.Fatalf("expected lateral join, got %#v", fromClause(t, st.From).Table)
	}
	right, ok = join.Right.(*ast.TableSource)
	if !ok || !right.Lateral {
		t.Fatalf("expected lateral subquery, got %#v", fromClause(t, st.From).Table)
	}
}

func TestParseSQLJSONExistsAndWildcardArgs(t *testing.T) {
	st := mustParseSelect(t, "SELECT to_jsonb(ro.*) FROM recent_orders ro WHERE ro.data ? 'key';")
	if st.Where == nil {
		t.Fatal("expected WHERE clause")
	}
	if _, ok := st.Fields.Items[0].Expr.(*ast.FuncCallExpr); !ok {
		t.Fatalf("expected function call, got %T", st.Fields.Items[0].Expr)
	}

	st = mustParseSelect(t, "SELECT * FROM t WHERE meta ? 'key' AND id = $1;")
	if st.Where == nil {
		t.Fatal("expected WHERE clause")
	}
}

func TestParseSQLSetSearchPathAndTruncateOptions(t *testing.T) {
	if _, ok := mustParseOne(t, "SET search_path = public, pg_catalog;").(*ast.VariableSetStmt); !ok {
		t.Fatal("expected *ast.VariableSetStmt")
	}
	if _, ok := mustParseOne(t, "SET search_path FROM CURRENT;").(*ast.VariableSetStmt); !ok {
		t.Fatal("expected *ast.VariableSetStmt")
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
	if _, ok := sel.With.CTEs[0].CTECore().Query.(*ast.UpdateStmt); !ok {
		t.Fatalf("expected update CTE query, got %T", sel.With.CTEs[0].CTECore().Query)
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
	if _, ok := n.(*ast.ParamMarkerExpr); ok {
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
		if tbl.Name.Parts[len(tbl.Name.Parts)-1].Name == name && (schema == "" || (len(tbl.Name.Parts) > 1 && tbl.Name.Parts[0].Name == schema)) {
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
	cte, ok := st.With.CTEs[0].CTECore().Query.(*ast.SelectStmt)
	if !ok {
		t.Fatalf("expected select CTE query, got %T", st.With.CTEs[0].CTECore().Query)
	}
	if len(cte.Fields.Items) != 2 {
		t.Fatalf("expected 2 CTE fields, got %d", len(cte.Fields.Items))
	}
	if got := strings.Join([]string{cte.Fields.Items[0].Expr.(*ast.ColumnNameExpr).Name.Parts[0].Name, cte.Fields.Items[0].Expr.(*ast.ColumnNameExpr).Name.Parts[1].Name}, "."); got != "users.id" {
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
	cte, ok := st.With.CTEs[0].CTECore().Query.(*ast.SelectStmt)
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
	upd, ok := st.With.CTEs[0].CTECore().Query.(*ast.UpdateStmt)
	if !ok {
		t.Fatalf("expected update CTE query, got %T", st.With.CTEs[0].CTECore().Query)
	}
	if len(upd.Assignments) == 0 {
		t.Fatal("expected SET clauses in CTE update")
	}
	if upd.Returning == nil || len(upd.Returning.Items) == 0 {
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
	if len(st.With.CTEs) != 1 || st.With.CTEs[0].CTECore().Name.Name != "expanded" {
		t.Fatalf("unexpected CTEs: %#v", st.With.CTEs)
	}
	all := collectTables(st)
	if !hasTable(all, "slow_smoke", "unindexed") {
		t.Fatal("expected slow_smoke.unindexed base table")
	}
	refs := 0
	for _, tbl := range all {
		if tbl.Name.Parts[len(tbl.Name.Parts)-1].Name == "expanded" && len(tbl.Name.Parts) == 1 {
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
	left, ok := st.Table.Table.(*ast.TableSource)
	if !ok {
		t.Fatalf("expected target TableSource, got %T", st.Table.Table)
	}
	target, ok := left.Source.(*ast.TableName)
	if !ok || target.Name.Parts[0].Name != "customers" || left.Alias.Name != "c" {
		t.Fatalf("unexpected MERGE target: %#v", left)
	}
	src, ok := st.Source.Table.(*ast.TableSource)
	if !ok {
		t.Fatalf("expected source TableSource, got %T", st.Source.Table)
	}
	if _, ok := src.Source.(*ast.SelectStmt); !ok || src.Alias.Name != "s" {
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
			if v.Alias.Name == "sq" {
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
	find(fromClause(t, st.From).Table)
	if sq == nil {
		t.Fatal("expected derived table alias sq")
	}
	sub, ok := sq.Source.(*ast.SelectStmt)
	if !ok {
		t.Fatalf("expected subquery, got %T", sq.Source)
	}
	if sub.GroupBy == nil || len(groupClause(t, sub.GroupBy).Items) == 0 {
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
	if st.Operator != ast.ExceptAll {
		t.Fatalf("outer operator = %v, want EXCEPT ALL", st.Operator)
	}
	grouped, ok := st.Left.(*ast.QueryTailStmt)
	if !ok || !grouped.Parenthesized {
		t.Fatalf("expected parenthesized query, got %T", st.Left)
	}
	nested, ok := grouped.Query.(*ast.SetOprStmt)
	if !ok {
		t.Fatalf("expected nested set operation, got %T", st.Left)
	}
	if nested.Operator != ast.Intersect {
		t.Fatalf("nested operator = %v, want INTERSECT", nested.Operator)
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
			if st.Operator != tc.op {
				t.Fatalf("operator = %v, want %v", st.Operator, tc.op)
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
			for _, branch := range []ast.QueryNode{st.Left, st.Right} {
				if branch.(*ast.SelectStmt).Where == nil {
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
	if st.Tail.Limit == nil || limitClauseNode(t, st.Tail.Limit).Count == nil {
		t.Fatal("expected top-level LIMIT")
	}
	branch, ok := st.Right.(*ast.SelectStmt)
	if !ok {
		t.Fatalf("right operand = %T, want *ast.SelectStmt", st.Right)
	}
	if branch.Limit == nil || limitClauseNode(t, branch.Limit).Count == nil {
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
	stmts, err := pgparser.New().Parse("SELECT * FROM a WHERE id = $1; SELECT * FROM b WHERE status = $1;")
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
	if setOp.Operator != ast.UnionAll {
		t.Fatalf("operator = %v, want UNION ALL", setOp.Operator)
	}
	if setOp.Tail.OrderBy == nil {
		t.Fatal("expected ORDER BY on set operation")
	}
	if !hasTable(collectTables(setOp), "", "archived_orders") {
		t.Fatal("expected archived_orders table")
	}

	ins, ok := mustParseOne(t, `INSERT INTO customers (name, email, signup_date)
		VALUES ('John', 'john@example.com', NOW()) RETURNING id, created_at;`).(*ast.InsertStmt)
	if !ok || len(ins.Columns) != 3 || ins.Returning == nil || len(ins.Returning.Items) != 2 {
		t.Fatalf("unexpected insert: %#v", ins)
	}

	upd, ok := mustParseOne(t, `UPDATE products p
		SET price = price * (1 - d.discount_rate), updated_at = NOW()
		FROM discounts d
		WHERE p.category_id = d.category_id AND d.expires_at > NOW()
		RETURNING p.id, p.price;`).(*ast.UpdateStmt)
	if !ok || len(upd.Assignments) != 2 || upd.From == nil || upd.Returning == nil {
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
	if j, ok := fromClause(t, join.From).Table.(*ast.Join); !ok || j.On == nil {
		t.Fatalf("expected JOIN ON condition, got %#v", fromClause(t, join.From).Table)
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
	if !ok || len(upd.Assignments) != 1 || upd.Returning == nil {
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
	if !ok || len(ins.Columns) != 2 || ins.Returning == nil || len(ins.Values) != 1 || len(ins.Values[0]) != 2 {
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
	if len(oc.IndexCols) != 1 || oc.IndexCols[0].Column.Name != "id" {
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
	if ins.Returning == nil || len(ins.Returning.Items) != 2 {
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
	if len(oc.IndexCols) != 1 || oc.IndexCols[0].Column.Name != "id" {
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
	if st, ok := mustParseOne(t, "DROP FUNCTION IF EXISTS public.f(int);").(*ast.DropFunctionStmt); !ok || !st.IfExists || st.Name.Name.Parts[len(st.Name.Name.Parts)-1].Name != "f" {
		t.Fatalf("drop function = %#v", mustParseOne(t, "DROP FUNCTION IF EXISTS public.f(int);"))
	}
	if st, ok := mustParseOne(t, "DROP PROCEDURE p();").(*ast.DropProcedureStmt); !ok || st.Name.Name.Parts[len(st.Name.Name.Parts)-1].Name != "p" {
		t.Fatalf("drop procedure = %#v", mustParseOne(t, "DROP PROCEDURE p();"))
	}
	if st, ok := mustParseOne(t, "DROP TRIGGER IF EXISTS trg ON t;").(*ast.DropTriggerStmt); !ok || !st.IfExists {
		t.Fatalf("drop trigger = %#v", mustParseOne(t, "DROP TRIGGER IF EXISTS trg ON t;"))
	}
}

func TestParseSQLUserRoleAndPreparedStatements(t *testing.T) {
	if st, ok := mustParseOne(t, "CREATE USER u PASSWORD 'MOCK_TOKEN';").(*ast.CreateUserStmt); !ok || st.Specs[0].AuthOpt == nil || !st.Specs[0].AuthOpt.Password {
		t.Fatalf("create user = %#v", mustParseOne(t, "CREATE USER u PASSWORD 'MOCK_TOKEN';"))
	}
	if st, ok := mustParseOne(t, "CREATE ROLE r WITH PASSWORD 'MOCK_TOKEN';").(*ast.CreateUserStmt); !ok || !st.IsCreateRole || !st.Specs[0].AuthOpt.Password {
		t.Fatalf("create role = %#v", mustParseOne(t, "CREATE ROLE r WITH PASSWORD 'MOCK_TOKEN';"))
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
	if !ok || st.Name.Name != "p" || len(st.Params) != 2 || st.Stmt == nil {
		t.Fatalf("prepare = %#v", prep)
	}
	if st.Params[0].String() != `"int4"` || st.Params[1].String() != `"text"` {
		t.Fatalf("prepare params = %#v", st.Params)
	}
	if st, ok := mustParseOne(t, "EXECUTE p(1, 'x');").(*ast.ExecuteStmt); !ok || len(st.Args) != 2 {
		t.Fatalf("execute = %#v", mustParseOne(t, "EXECUTE p(1, 'x');"))
	}
	if st, ok := mustParseOne(t, "DEALLOCATE PREPARE p;").(*ast.DeallocateStmt); !ok || st.Name.Name != "p" || st.All {
		t.Fatalf("deallocate = %#v", mustParseOne(t, "DEALLOCATE PREPARE p;"))
	}
	if st, ok := mustParseOne(t, "DEALLOCATE ALL;").(*ast.DeallocateStmt); !ok || !st.All {
		t.Fatalf("deallocate all = %#v", mustParseOne(t, "DEALLOCATE ALL;"))
	}

	if got := compactSQL(restoreNode(t, prep)); got != `PREPARE "p" ("int4", "text") AS INSERT INTO "t" VALUES ($1,$2)` {
		t.Fatalf("prepare restore = %q", got)
	}
	if got := compactSQL(restoreNode(t, mustParseOne(t, "CREATE USER u PASSWORD 'MOCK_TOKEN';"))); got != `CREATE USER "u" PASSWORD 'MOCK_TOKEN'` {
		t.Fatalf("create user restore = %q", got)
	}
}

func TestParseSQLReferenceUtilityStatementFamilies(t *testing.T) {
	cases := []struct {
		sql  string
		want any
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
	if st, ok := mustParseOne(t, `SET ROLE app;`).(*ast.VariableSetStmt); !ok || st.Name != "role" || st.Args == nil || len(st.Args.Items) != 1 {
		t.Fatalf("set role = %#v", mustParseOne(t, `SET ROLE app;`))
	}
	if st, ok := mustParseOne(t, `SET ROLE NONE;`).(*ast.VariableSetStmt); !ok || st.Name != "role" || st.Kind != ast.VAR_SET_VALUE {
		t.Fatalf("set role none = %#v", mustParseOne(t, `SET ROLE NONE;`))
	}
	if st, ok := mustParseOne(t, `SET ROLE DEFAULT;`).(*ast.VariableSetStmt); !ok || st.Name != "role" || st.Kind != ast.VAR_SET_DEFAULT {
		t.Fatalf("set role default = %#v", mustParseOne(t, `SET ROLE DEFAULT;`))
	}
	if st, ok := mustParseOne(t, `SET SESSION AUTHORIZATION app;`).(*ast.VariableSetStmt); !ok || st.Name != "session_authorization" {
		t.Fatalf("set session authorization = %#v", mustParseOne(t, `SET SESSION AUTHORIZATION app;`))
	}
	if st, ok := mustParseOne(t, `PREPARE p AS SELECT 1;`).(*ast.PrepareStmt); !ok || st.Name.Name != "p" || st.Stmt == nil {
		t.Fatalf("prepare = %#v", mustParseOne(t, `PREPARE p AS SELECT 1;`))
	}
}
