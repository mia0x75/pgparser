package parser_test

import (
	"strings"
	"testing"

	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/format"
)

func restoreNode(t *testing.T, n ast.Node) string {
	t.Helper()
	var sb strings.Builder
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)
	if err := n.Restore(ctx); err != nil {
		t.Fatalf("restore %T: %v", n, err)
	}
	return sb.String()
}

func compactSQL(s string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(s, "`", "")), " ")
}

func squashSQL(s string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(s, "`", "")), "")
}

// Ported from parser_ir_select_test.go: projection expressions and aliases.
func TestParseSQLSelectExpressionsAST(t *testing.T) {
	cases := []struct {
		name   string
		sql    string
		checks []struct {
			expr  string
			alias string
		}
	}{
		{
			name: "functions",
			sql:  `SELECT COALESCE(email, 'unknown') AS email, LOWER(name) FROM users;`,
			checks: []struct {
				expr  string
				alias string
			}{
				{"COALESCE(email, 'unknown')", "email"},
				{"LOWER(name)", ""},
			},
		},
		{
			name: "arithmetic",
			sql:  `SELECT price * quantity AS total_cost FROM orders;`,
			checks: []struct {
				expr  string
				alias string
			}{
				{"price * quantity", "total_cost"},
			},
		},
		{
			name: "json operators",
			sql:  `SELECT data->>'name' AS username, data->'meta'->>'id' AS user_id FROM profiles;`,
			checks: []struct {
				expr  string
				alias string
			}{
				{"data ->> 'name'", "username"},
				{"data -> 'meta' ->> 'id'", "user_id"},
			},
		},
		{
			name: "window",
			sql:  `SELECT id, ROW_NUMBER() OVER (PARTITION BY tenant ORDER BY created_at DESC) AS seq FROM orders;`,
			checks: []struct {
				expr  string
				alias string
			}{
				{"id", ""},
				{"ROW_NUMBER() OVER (PARTITION BY tenant ORDER BY created_at DESC)", "seq"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := mustParseSelect(t, tc.sql)
			if st.Fields == nil || len(st.Fields.Fields) != len(tc.checks) {
				t.Fatalf("fields = %#v, want %d", st.Fields, len(tc.checks))
			}
			for i, want := range tc.checks {
				field := st.Fields.Fields[i]
				if got := squashSQL(restoreNode(t, field.Expr)); got != squashSQL(want.expr) {
					t.Fatalf("field %d expr = %q, want %q", i, got, want.expr)
				}
				if got := field.AsName.O; got != want.alias {
					t.Fatalf("field %d alias = %q, want %q", i, got, want.alias)
				}
			}
		})
	}

	caseSt := mustParseSelect(t, `SELECT CASE WHEN status = 'shipped' THEN 'done' WHEN status = 'pending' THEN 'waiting' ELSE 'other' END AS state FROM orders;`)
	if got := restoreNode(t, caseSt.Fields.Fields[0].Expr); !strings.Contains(got, "CASE") || !strings.Contains(got, "WHEN") {
		t.Fatalf("CASE expression not preserved: %q", got)
	}
	if got := caseSt.Fields.Fields[0].AsName.O; got != "state" {
		t.Fatalf("CASE alias = %q, want state", got)
	}
}

// Ported from parser_ir_select_test.go: GROUP BY / HAVING / ORDER BY / LIMIT.
func TestParseSQLSelectClausesAST(t *testing.T) {
	st := mustParseSelect(t, `SELECT tenant, COUNT(*) AS cnt FROM events WHERE occurred_at >= $1 GROUP BY tenant HAVING COUNT(*) > 5 ORDER BY cnt DESC NULLS LAST LIMIT 10 OFFSET 2;`)
	if st.GroupBy == nil || len(st.GroupBy.Items) != 1 {
		t.Fatalf("group by = %#v, want 1 item", st.GroupBy)
	}
	if got := squashSQL(restoreNode(t, st.GroupBy.Items[0].Expr)); got != "tenant" {
		t.Fatalf("group by = %q, want tenant", got)
	}
	if st.Having == nil {
		t.Fatal("expected HAVING")
	}
	if got := squashSQL(restoreNode(t, st.Having.Expr)); !strings.HasSuffix(got, ">5") {
		t.Fatalf("having = %q", got)
	}
	if st.OrderBy == nil || len(st.OrderBy.Items) != 1 {
		t.Fatalf("order by = %#v, want 1 item", st.OrderBy)
	}
	item := st.OrderBy.Items[0]
	if !item.Desc || item.NullOrder != 2 {
		t.Fatalf("order item = %#v, want DESC NULLS LAST", item)
	}
	if st.Limit == nil || st.Limit.Count == nil || st.Limit.Offset == nil {
		t.Fatalf("limit = %#v, want COUNT and OFFSET", st.Limit)
	}
	if got := countParams(st); got != 1 {
		t.Fatalf("params = %d, want 1", got)
	}
}

func walkJoins(n ast.ResultSetNode, fn func(*ast.Join)) {
	v, ok := n.(*ast.Join)
	if !ok {
		return
	}
	fn(v)
	if v.Left != nil {
		walkJoins(v.Left, fn)
	}
	if v.Right != nil {
		walkJoins(v.Right, fn)
	}
}

// Ported from parser_ir_select_test.go: join variants and conditions.
func TestParseSQLSelectJoinsAST(t *testing.T) {
	st := mustParseSelect(t, `SELECT a.id, b.id, c.id, d.id, e.id FROM alpha a RIGHT JOIN beta b ON a.id = b.alpha_id FULL JOIN gamma c ON b.id = c.beta_id LEFT OUTER JOIN delta d ON c.id = d.gamma_id NATURAL LEFT JOIN epsilon e;`)
	tables := collectTables(st)
	for _, name := range []string{"alpha", "beta", "gamma", "delta", "epsilon"} {
		if !hasTable(tables, "", name) {
			t.Fatalf("missing table %q in %#v", name, tables)
		}
	}

	seen := map[ast.JoinType]bool{}
	var natural bool
	var conditions []string
	walkJoins(st.From.TableRefs, func(j *ast.Join) {
		if j.Tp != 0 {
			seen[j.Tp] = true
		}
		if j.NaturalJoin {
			natural = true
		}
		if j.On != nil {
			conditions = append(conditions, squashSQL(restoreNode(t, j.On.Expr)))
		}
	})
	for _, wantTp := range []ast.JoinType{ast.RightJoin, ast.LeftJoin} {
		if !seen[wantTp] {
			t.Fatalf("join types = %#v, missing %v", seen, wantTp)
		}
	}
	if !natural {
		t.Fatal("expected NATURAL join")
	}
	for _, wantCond := range []string{"a.id = b.alpha_id", "b.id = c.beta_id", "c.id = d.gamma_id"} {
		found := false
		for _, got := range conditions {
			if got == squashSQL(wantCond) {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing condition %q in %#v", wantCond, conditions)
		}
	}
}

// Ported from parser_ir_select_test.go: CTE scoping and derived tables.
func TestParseSQLSelectCTEAndSubqueriesAST(t *testing.T) {
	st := mustParseSelect(t, `WITH active_users AS MATERIALIZED (SELECT id FROM users) SELECT active_users.id AS user_id FROM active_users ORDER BY active_users.id DESC NULLS LAST LIMIT 10 OFFSET 2;`)
	if st.With == nil || len(st.With.CTEs) != 1 {
		t.Fatalf("with = %#v, want 1 CTE", st.With)
	}
	if got := st.With.CTEs[0].Name.O; got != "active_users" {
		t.Fatalf("CTE name = %q", got)
	}
	if got := squashSQL(restoreNode(t, st.With.CTEs[0].Query)); !strings.Contains(got, "idFROMusers") {
		t.Fatalf("CTE query = %q", got)
	}
	tables := collectTables(st)
	if !hasTable(tables, "", "users") || !hasTable(tables, "", "active_users") {
		t.Fatalf("tables = %#v, want users and active_users", tables)
	}
	if st.OrderBy == nil || len(st.OrderBy.Items) != 1 || !st.OrderBy.Items[0].Desc || st.OrderBy.Items[0].NullOrder != 2 {
		t.Fatalf("order by = %#v", st.OrderBy)
	}

	sub := mustParseSelect(t, `SELECT id, name FROM users WHERE id IN (SELECT user_id FROM orders WHERE total > 500);`)
	subTables := collectTables(sub)
	if !hasTable(subTables, "", "users") || !hasTable(subTables, "", "orders") {
		t.Fatalf("subquery tables = %#v", subTables)
	}

	star := mustParseSelect(t, `SELECT sub.id, ids.id FROM (SELECT id FROM accounts) sub CROSS JOIN LATERAL unnest($1) AS ids(id);`)
	starTables := collectTables(star)
	if !hasTable(starTables, "", "accounts") {
		t.Fatalf("derived table base missing: %#v", starTables)
	}
	if got := countParams(star); got != 1 {
		t.Fatalf("params = %d, want 1", got)
	}
}

// Ported from parser_ir_select_test.go: a large real-world query stays parseable.
func TestParseSQLSelectComplexQueryAST(t *testing.T) {
	st := mustParseSelect(t, `
WITH recent_orders AS (
  SELECT o.id, SUM(oi.quantity * oi.price) AS total_amount
  FROM orders o JOIN order_items oi ON oi.order_id = o.id
  WHERE o.created_at >= NOW() - INTERVAL '30 days'
  GROUP BY o.id
),
expanded_data AS (
  SELECT tc.customer_id,
    jsonb_build_object('orders', (SELECT jsonb_agg(to_jsonb(ro.*)) FROM recent_orders ro)) AS full_data
  FROM top_customers tc JOIN customers c ON c.id = tc.customer_id
)
SELECT ed.customer_id, ed.full_data ->> 'info' AS customer_info,
  SUM((o.order_json ->> 'total_amount')::numeric) AS total_revenue
FROM expanded_data ed
CROSS JOIN LATERAL jsonb_array_elements(ed.full_data -> 'orders') AS o(order_json)
WHERE (o.order_json ->> 'total_amount')::numeric > 100
GROUP BY ed.customer_id, ed.full_data
HAVING SUM((o.order_json ->> 'total_amount')::numeric) > 5000
ORDER BY total_revenue DESC LIMIT 5;`)
	if st.With == nil || len(st.With.CTEs) != 2 {
		t.Fatalf("CTEs = %#v, want 2", st.With)
	}
	if st.Limit == nil {
		t.Fatal("expected LIMIT")
	}
	if got := countParams(st); got != 0 {
		t.Fatalf("params = %d, want 0", got)
	}

	foundLateral := false
	walkJoins(st.From.TableRefs, func(j *ast.Join) {
		if ts, ok := j.Right.(*ast.TableSource); ok && ts.Lateral {
			foundLateral = true
		}
	})
	if !foundLateral {
		t.Fatal("expected LATERAL table function")
	}
}

// Ported from parser_ir_select_test.go: double-slash comments are ignored.
func TestParseSQLSelectDoubleSlashComment(t *testing.T) {
	st := mustParseSelect(t, "SELECT id FROM users // extra metadata\n;")
	if st.Fields == nil || len(st.Fields.Fields) != 1 {
		t.Fatalf("fields = %#v, want 1", st.Fields)
	}
	if tables := collectTables(st); !hasTable(tables, "", "users") {
		t.Fatalf("tables = %#v", tables)
	}
}

func TestParseSQLSelectWildcardSchemaTable(t *testing.T) {
	st := mustParseSelect(t, `SELECT s.t.* FROM s.t;`)
	wc := st.Fields.Fields[0].WildCard
	if wc == nil || wc.Schema.O != "s" || wc.Table.O != "t" {
		t.Fatalf("wildcard = %#v, want s.t.*", wc)
	}
	if got := compactSQL(restoreNode(t, wc)); got != "s.t.*" {
		t.Fatalf("restored wildcard = %q", got)
	}
}
