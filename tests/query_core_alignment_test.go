package parser_test

import (
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"
	dialectast "github.com/mia0x75/pgparser/ast"
)

func TestParseSharedQueryCore(t *testing.T) {
	const query = `WITH q AS (SELECT id FROM source) SELECT t.id, q.id FROM target AS t JOIN q ON t.id = q.id WHERE t.id > 0 ORDER BY t.id LIMIT 3`
	node := mustParseOne(t, query)
	stmt, ok := node.(*dialectast.SelectStmt)
	if !ok {
		t.Fatalf("statement type = %T, want *dialectast.SelectStmt", node)
	}
	if stmt.Fields == nil || len(stmt.Fields.Items) != 2 || stmt.From == nil || stmt.Where == nil || stmt.OrderBy == nil || stmt.Limit == nil {
		t.Fatalf("shared select core is incomplete: %#v", stmt.BaseSelectStmt)
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
