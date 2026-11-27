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

func TestParseSharedRecursiveCTECore(t *testing.T) {
	node := mustParseOne(t, "WITH RECURSIVE c(n) AS (SELECT 1 UNION ALL SELECT n FROM c) SELECT n FROM c")
	stmt, ok := node.(*dialectast.SelectStmt)
	if !ok || stmt.With == nil {
		t.Fatalf("statement = %T, want SELECT with WITH clause", node)
	}
	with := stmt.With.WithClauseCore()
	if with == nil || !with.IsRecursive || len(with.CTEs) != 1 {
		t.Fatalf("shared WITH core = %#v", with)
	}
	cte := with.CTEs[0].CTECore()
	if cte == nil || cte.Name.Name != "c" || len(cte.Columns) != 1 || cte.Columns[0].Name != "n" {
		t.Fatalf("shared CTE core = %#v", cte)
	}
	if _, ok := cte.Query.(pkast.SetOperationNode); !ok {
		t.Fatalf("recursive CTE query = %T, want shared set operation", cte.Query)
	}
}

func TestParseSharedCTEOnSetOperation(t *testing.T) {
	node := mustParseOne(t, "WITH c AS (SELECT 1) SELECT 1 UNION SELECT 2")
	stmt, ok := node.(*dialectast.SetOprStmt)
	if !ok || stmt.With == nil {
		t.Fatalf("statement = %T, want set operation with WITH clause", node)
	}
	with := stmt.With.WithClauseCore()
	if with == nil || len(with.CTEs) != 1 || with.CTEs[0].CTECore() == nil {
		t.Fatalf("shared WITH core = %#v", with)
	}
	branch, ok := stmt.Left.(*dialectast.SelectStmt)
	if !ok {
		t.Fatalf("first set-operation branch = %T, want SELECT", stmt.Left)
	}
	if branch.With != nil {
		t.Fatal("WITH should belong to the set operation, not its first branch")
	}
}

func TestParseSharedWindowCore(t *testing.T) {
	stmt, ok := mustParseOne(t, "SELECT 1 FROM t WINDOW w AS (PARTITION BY b ORDER BY c DESC ROWS BETWEEN 2 PRECEDING AND CURRENT ROW)").(*dialectast.SelectStmt)
	if !ok {
		t.Fatal("expected *dialectast.SelectStmt")
	}
	if len(stmt.WindowSpecs) != 1 || stmt.WindowSpecs[0].WindowSpecCore() == nil {
		t.Fatalf("window specs = %#v, want one shared window spec", stmt.WindowSpecs)
	}
	spec, ok := stmt.WindowSpecs[0].(*dialectast.WindowSpec)
	if !ok {
		t.Fatalf("window spec = %T, want *dialectast.WindowSpec", stmt.WindowSpecs[0])
	}
	core := spec.WindowSpecCore()
	if core.Name.Name != "w" || core.PartitionBy == nil || core.OrderBy == nil || core.Frame == nil {
		t.Fatalf("shared window spec core = %#v", core)
	}
	partition := core.PartitionBy.WindowPartitionClauseCore()
	if partition == nil || len(partition.Items) != 1 || partition.Items[0] == nil {
		t.Fatalf("shared window partition core = %#v", partition)
	}
	partitionItem := partition.Items[0].WindowPartitionItemCore()
	if partitionItem == nil || partitionItem.Expr == nil || len(spec.PartitionBy.Items) != 1 || partitionItem.Expr != spec.PartitionBy.Items[0].Expr {
		t.Fatalf("shared window partition item = %#v", partitionItem)
	}
	order := core.OrderBy.OrderByClauseCore()
	if order == nil || len(order.Items) != 1 || order.Items[0] == nil {
		t.Fatalf("shared window order core = %#v", order)
	}
	orderItem := order.Items[0].OrderItemCore()
	if orderItem == nil || !orderItem.Desc {
		t.Fatalf("shared window order item = %#v", orderItem)
	}
	frame := core.Frame.WindowFrameClauseCore()
	if frame == nil || frame.Type != pkast.WindowFrameRows || frame.Extent.Start == nil || frame.Extent.End == nil {
		t.Fatalf("shared window frame core = %#v", frame)
	}
	start, end := frame.Extent.Start.WindowFrameBoundCore(), frame.Extent.End.WindowFrameBoundCore()
	if start == nil || end == nil || start.Type != pkast.WindowFramePreceding || start.Expr == nil || end.Type != pkast.WindowFrameCurrentRow {
		t.Fatalf("shared window frame core = %#v", frame)
	}
	replacementExpr := spec.PartitionBy.Items[0].Expr
	replacement := &dialectast.PartitionByClause{Items: []*dialectast.OrderItem{{BaseOrderItem: pkast.BaseOrderItem{Expr: replacementExpr}}}}
	updated, ok := spec.Accept(&projectionVisitor{
		enter: func(node dialectast.Node) (dialectast.Node, bool) {
			if _, ok := node.(*dialectast.PartitionByClause); ok {
				return replacement, true
			}
			return node, false
		},
		leave: func(node dialectast.Node) (dialectast.Node, bool) { return node, true },
	})
	if !ok {
		t.Fatal("visitor rejected shared window spec")
	}
	updatedCore := updated.(*dialectast.WindowSpec).WindowSpecCore().PartitionBy.WindowPartitionClauseCore()
	if len(updatedCore.Items) != 1 || updatedCore.Items[0].WindowPartitionItemCore().Expr != replacementExpr {
		t.Fatalf("visitor replacement was not reflected in shared window core: %#v", updatedCore)
	}
}

func TestParseSharedInsertCore(t *testing.T) {
	stmt, ok := mustParseOne(t, "INSERT INTO t (a, b) VALUES (1, 2), (3, 4)").(*dialectast.InsertStmt)
	if !ok {
		t.Fatal("expected *dialectast.InsertStmt")
	}
	if stmt.Table == nil || stmt.Table.TableRefCore() == nil || len(stmt.Columns) != 2 || len(stmt.Values) != 2 || stmt.Query != nil {
		t.Fatalf("shared INSERT values core = %#v", stmt.BaseInsertStmt)
	}
	for i, column := range stmt.Columns {
		if column == nil {
			t.Fatalf("shared INSERT column %d is nil", i)
		}
	}
	for i, row := range stmt.Values {
		if len(row) != 2 || row[0] == nil || row[1] == nil {
			t.Fatalf("shared INSERT values row %d = %#v", i, row)
		}
	}

	query, ok := mustParseOne(t, "INSERT INTO t (a) SELECT 1").(*dialectast.InsertStmt)
	if !ok || query.Table == nil || query.Table.TableRefCore() == nil || len(query.Columns) != 1 || query.Values != nil || query.Query == nil {
		t.Fatalf("shared INSERT query core = %#v", query)
	}
	if _, ok := query.Query.(*dialectast.SelectStmt); !ok || query.Columns[0] == nil {
		t.Fatalf("shared INSERT query source/column = %T / %#v", query.Query, query.Columns)
	}
}

func TestParseSharedAssignmentCore(t *testing.T) {
	stmt, ok := mustParseOne(t, "UPDATE t SET a = 1, b = a + 1").(*dialectast.UpdateStmt)
	if !ok {
		t.Fatal("expected *dialectast.UpdateStmt")
	}
	if len(stmt.Assignments) != 2 {
		t.Fatalf("statement = %T, assignments = %d, want UPDATE with two assignments", stmt, len(stmt.Assignments))
	}
	for i, assignment := range stmt.Assignments {
		shared := assignment
		if shared == nil || shared.Target == nil || shared.Expr == nil {
			t.Fatalf("shared assignment %d = %#v", i, shared)
		}
	}
}

func TestParseSharedReturningCore(t *testing.T) {
	insert, ok := mustParseOne(t, "INSERT INTO t (a, b) VALUES (1, 2) RETURNING a, b").(*dialectast.InsertStmt)
	if !ok {
		t.Fatal("expected *dialectast.InsertStmt")
	}
	assertSharedReturningCore(t, insert.Returning)

	update, ok := mustParseOne(t, "UPDATE t SET a = 1 WHERE b = 2 RETURNING a, b").(*dialectast.UpdateStmt)
	if !ok {
		t.Fatal("expected *dialectast.UpdateStmt")
	}
	assertSharedReturningCore(t, update.Returning)

	deleteStmt, ok := mustParseOne(t, "DELETE FROM t WHERE a = 1 RETURNING a, b").(*dialectast.DeleteStmt)
	if !ok {
		t.Fatal("expected *dialectast.DeleteStmt")
	}
	assertSharedReturningCore(t, deleteStmt.Returning)
}

func TestLimitCommaKeepsOffsetAndCountOrder(t *testing.T) {
	stmt := mustParseOne(t, "SELECT 1 LIMIT 2, 3")
	if got, want := restoreString(t, stmt), "SELECT 1 LIMIT 3 OFFSET 2"; got != want {
		t.Fatalf("restored LIMIT = %q, want %q", got, want)
	}
}

func assertSharedReturningCore(t *testing.T, fields *pkast.FieldList) {
	t.Helper()
	if fields == nil || len(fields.Items) != 2 {
		t.Fatalf("shared RETURNING fields = %#v, want two items", fields)
	}
	for i, item := range fields.Items {
		if item == nil || item.Expr == nil {
			t.Fatalf("shared RETURNING item %d = %#v", i, item)
		}
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
