package parser_test

import (
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"
	dialectast "github.com/mia0x75/pgparser/ast"
)

func TestParseSharedQueryCore(t *testing.T) {
	const query = `WITH q AS (SELECT id FROM source) SELECT t.id, q.id FROM target AS t JOIN q ON t.id = q.id WHERE t.id > 0 GROUP BY t.id, q.id HAVING COUNT(*) > 0 ORDER BY t.id LIMIT 3`
	node := mustParseOne(t, query)
	stmt, ok := node.(*dialectast.SelectStmt)
	if !ok {
		t.Fatalf("statement type = %T, want *dialectast.SelectStmt", node)
	}
	if stmt.Fields == nil || len(stmt.Fields.Items) != 2 || stmt.From == nil || stmt.Where == nil || stmt.GroupBy == nil || stmt.Having == nil || stmt.OrderBy == nil || stmt.Limit == nil {
		t.Fatalf("shared select core is incomplete: %#v", stmt.BaseSelectStmt)
	}
	groupBy := stmt.GroupBy.GroupByClauseCore()
	if groupBy == nil || len(groupBy.Items) != 2 {
		t.Fatalf("shared GROUP BY core = %#v", groupBy)
	}

	from := stmt.From.FromClauseCore()
	if from == nil {
		t.Fatal("shared FROM core is nil")
	}
	join, ok := from.Table.(pkast.JoinExprNode)
	if !ok || join.JoinExprCore() == nil {
		t.Fatalf("FROM relation = %T, want shared JoinExprNode", from.Table)
	}
	joinCore := join.JoinExprCore()
	if joinCore.Type != pkast.JoinTypeInner || joinCore.On == nil {
		t.Fatalf("shared JOIN core = %#v", joinCore)
	}

	if stmt.With == nil {
		t.Fatalf("shared WITH core = %#v", stmt.With)
	}
	with := stmt.With.WithClauseCore()
	if with == nil || len(with.CTEs) != 1 {
		t.Fatalf("shared WITH core = %#v", with)
	}
	cte := with.CTEs[0].CTECore()
	if cte == nil || cte.Name.Name != "q" || cte.Query == nil {
		t.Fatalf("shared CTE core = %#v", cte)
	}
}

func TestParseSharedSetOperationCore(t *testing.T) {
	for _, tc := range []struct {
		name               string
		sql                string
		operator           pkast.SetOperationType
		rightOperator      pkast.SetOperationType
		rightParenthesized bool
	}{
		{
			name:          "intersect precedence",
			sql:           "SELECT 1 UNION SELECT 2 INTERSECT SELECT 3",
			operator:      pkast.SetOperationUnion,
			rightOperator: pkast.SetOperationIntersect,
		},
		{
			name:               "parenthesized right operand",
			sql:                "SELECT 1 UNION (SELECT 2 INTERSECT SELECT 3)",
			operator:           pkast.SetOperationUnion,
			rightOperator:      pkast.SetOperationIntersect,
			rightParenthesized: true,
		},
		{
			name:     "values operands",
			sql:      "VALUES (1) UNION VALUES (2)",
			operator: pkast.SetOperationUnion,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			node := mustParseOne(t, tc.sql)
			setOperation, ok := node.(pkast.SetOperationNode)
			if !ok || setOperation.SetOperationCore() == nil {
				t.Fatalf("query = %T, want shared SetOperationNode", node)
			}
			core := setOperation.SetOperationCore()
			if core.Operator != tc.operator || core.Left == nil || core.Right == nil {
				t.Fatalf("shared set-operation core = %#v", core)
			}
			if tc.rightOperator == pkast.SetOperationUnknown {
				return
			}
			right := core.Right
			if tc.rightParenthesized {
				wrapper, ok := right.(pkast.QueryTailNode)
				if !ok || wrapper.QueryTailStmtCore() == nil || !wrapper.QueryTailStmtCore().Parenthesized {
					t.Fatalf("right operand = %T, want shared parenthesized query wrapper", right)
				}
				right = wrapper.QueryTailStmtCore().Query
			}
			nested, ok := right.(pkast.SetOperationNode)
			if !ok || nested.SetOperationCore() == nil || nested.SetOperationCore().Operator != tc.rightOperator {
				t.Fatalf("right operand = %T, want nested %s", right, tc.rightOperator)
			}
		})
	}
}

func TestParseSharedQueryTailCore(t *testing.T) {
	stmt, ok := mustParseOne(t, "SELECT 1 ORDER BY 1 DESC LIMIT 3 OFFSET 2").(*dialectast.SelectStmt)
	if !ok {
		t.Fatal("expected *dialectast.SelectStmt")
	}
	assertSharedQueryTailCore(t, stmt.OrderBy, stmt.Limit)

	setOperation, ok := mustParseOne(t, "SELECT 1 UNION SELECT 2 ORDER BY 1 DESC LIMIT 3 OFFSET 2").(*dialectast.SetOprStmt)
	if !ok {
		t.Fatal("expected *dialectast.SetOprStmt")
	}
	assertSharedQueryTailCore(t, setOperation.Tail.OrderBy, setOperation.Tail.Limit)
}

func TestLimitCommaKeepsOffsetAndCountOrder(t *testing.T) {
	stmt := mustParseOne(t, "SELECT 1 LIMIT 2, 3")
	if got, want := restoreString(t, stmt), "SELECT 1 LIMIT 3 OFFSET 2"; got != want {
		t.Fatalf("restored LIMIT = %q, want %q", got, want)
	}
}

func assertSharedQueryTailCore(t *testing.T, orderBy pkast.OrderByClauseNode, limit pkast.LimitNode) {
	t.Helper()
	if orderBy == nil || orderBy.OrderByClauseCore() == nil {
		t.Fatal("shared ORDER BY core is nil")
	}
	items := orderBy.OrderByClauseCore().Items
	if len(items) != 1 {
		t.Fatalf("shared ORDER BY item count = %d, want 1", len(items))
	}
	item := items[0].OrderItemCore()
	if item == nil || !item.Desc {
		t.Fatalf("shared ORDER BY item core = %#v", item)
	}
	if limit == nil || limit.LimitCore() == nil || limit.LimitCore().Count == nil || limit.LimitCore().Offset == nil {
		t.Fatalf("shared LIMIT core = %#v", limit)
	}
}
