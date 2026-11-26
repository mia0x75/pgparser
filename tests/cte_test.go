package parser_test

import (
	"reflect"
	"strings"
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/format"
)

func TestCTESharedCore(t *testing.T) {
	stmt := mustParseSelect(t, `WITH "Cte" ("Col") AS (SELECT 1) SELECT 1;`)
	if stmt.With == nil || len(stmt.With.CTEs) != 1 {
		t.Fatalf("with = %#v", stmt.With)
	}
	var shared pkast.WithClauseNode = stmt.With
	if shared.WithClauseCore() != &stmt.With.BaseWithClause {
		t.Fatal("shared with core does not refer to the dialect node's data")
	}
	cte, ok := stmt.With.CTEs[0].(*ast.CommonTableExpression)
	if !ok {
		t.Fatalf("cte type = %T", stmt.With.CTEs[0])
	}
	var cteNode pkast.CTENode = cte
	if cteNode.CTECore() != &cte.BaseCommonTableExpression {
		t.Fatal("shared cte core does not refer to the dialect node's data")
	}
	if cte.Name.Name != "Cte" || cte.Name.Raw != `"Cte"` || cte.Name.Quote != lex.QuoteDouble {
		t.Fatalf("cte name = %#v", cte.Name)
	}
	if len(cte.Columns) != 1 || cte.Columns[0].Name != "Col" || cte.Columns[0].Quote != lex.QuoteDouble {
		t.Fatalf("cte columns = %#v", cte.Columns)
	}
	if reflect.TypeFor[ast.CTENode]() != reflect.TypeFor[pkast.CTENode]() || reflect.TypeFor[ast.WithClauseNode]() != reflect.TypeFor[pkast.WithClauseNode]() {
		t.Fatal("CTE interfaces are not shared aliases")
	}
}

func TestCTEPositions(t *testing.T) {
	const sql = `WITH "Cte" ("Col") AS (SELECT 1) SELECT 1;`
	stmt := mustParseSelect(t, sql)
	cte := stmt.With.CTEs[0].(*ast.CommonTableExpression)
	nameStart := strings.Index(sql, `"Cte"`)
	if cte.Name.Span != (loc.Span{Start: loc.ByteOffset(nameStart), End: loc.ByteOffset(nameStart + 5)}) {
		t.Fatalf("cte name span = %v", cte.Name.Span)
	}
	columnStart := strings.Index(sql, `"Col"`)
	if cte.Columns[0].Span != (loc.Span{Start: loc.ByteOffset(columnStart), End: loc.ByteOffset(columnStart + 5)}) {
		t.Fatalf("cte column span = %v", cte.Columns[0].Span)
	}
}

func TestCTERestore(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want string
	}{
		{"plain", `WITH c AS (SELECT 1) SELECT 1;`, `WITH "c" AS (SELECT 1) SELECT 1`},
		{"materialized", `WITH c AS MATERIALIZED (SELECT 1) SELECT 1;`, `WITH "c" AS MATERIALIZED (SELECT 1) SELECT 1`},
		{"not materialized", `WITH c AS NOT MATERIALIZED (SELECT 1) SELECT 1;`, `WITH "c" AS NOT MATERIALIZED (SELECT 1) SELECT 1`},
		{"recursive", `WITH RECURSIVE c(n) AS (SELECT 1 UNION ALL SELECT n FROM c) SELECT 1;`, `WITH RECURSIVE "c" ("n") AS (SELECT 1 UNION ALL SELECT "n" FROM "c") SELECT 1`},
		{"search", `WITH RECURSIVE c(n) AS (SELECT 1) SEARCH DEPTH FIRST BY n SET seq SELECT 1;`, `WITH RECURSIVE "c" ("n") AS (SELECT 1) SEARCH DEPTH FIRST BY "n" SET "seq" SELECT 1`},
		{"cycle", `WITH RECURSIVE c(n) AS (SELECT 1) CYCLE n SET mark TO true DEFAULT false USING path SELECT 1;`, `WITH RECURSIVE "c" ("n") AS (SELECT 1) CYCLE "n" SET "mark" TO TRUE DEFAULT FALSE USING "path" SELECT 1`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt := mustParseSelect(t, tt.sql)
			got := restoreString(t, stmt)
			if got != tt.want {
				t.Fatalf("restore = %q, want %q", got, tt.want)
			}
			if again := restoreString(t, mustParseOne(t, got)); again != got {
				t.Fatalf("second restore = %q, want %q", again, got)
			}
		})
	}
}

func TestCTEVisitor(t *testing.T) {
	stmt := mustParseSelect(t, `WITH c AS (SELECT 1) SELECT 1;`)
	visited := 0
	visitor := &projectionVisitor{
		enter: func(n ast.Node) (ast.Node, bool) {
			visited++
			return n, false
		},
		leave: func(n ast.Node) (ast.Node, bool) { return n, true },
	}
	if _, ok := stmt.With.Accept(visitor); !ok || visited == 0 {
		t.Fatalf("visitor: ok=%v visited=%d", ok, visited)
	}
	replacement := &ast.CommonTableExpression{BaseCommonTableExpression: pkast.BaseCommonTableExpression{
		Name:  lex.Ident{Name: "other"},
		Query: &ast.SelectStmt{},
	}}
	visitor.enter = func(n ast.Node) (ast.Node, bool) { return n, false }
	visitor.leave = func(n ast.Node) (ast.Node, bool) {
		if _, ok := n.(*ast.CommonTableExpression); ok {
			return replacement, true
		}
		return n, true
	}
	if _, ok := stmt.With.Accept(visitor); !ok {
		t.Fatal("visitor failed")
	}
	if stmt.With.CTEs[0] != ast.CTENode(replacement) {
		t.Fatalf("cte = %#v", stmt.With.CTEs[0])
	}
}

func TestCTEInvalidRestore(t *testing.T) {
	tests := []struct {
		name string
		node ast.Node
	}{
		{"nil clause", (*ast.WithClause)(nil)},
		{"nil cte", &ast.WithClause{BaseWithClause: pkast.BaseWithClause{CTEs: []ast.CTENode{(*ast.CommonTableExpression)(nil)}}}},
		{"nil cte body", &ast.CommonTableExpression{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out strings.Builder
			ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &out)
			if err := tt.node.Restore(ctx); err == nil {
				t.Fatal("invalid CTE restored without error")
			}
		})
	}
}
