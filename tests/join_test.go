package parser_test

import (
	"reflect"
	"strings"
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/format"
)

func TestJoinSharedCore(t *testing.T) {
	stmt := mustParseSelect(t, `SELECT * FROM "S"."T" AS a RIGHT JOIN u ON a.id = u.id;`)
	join, ok := fromClause(t, stmt.From).Table.(*ast.Join)
	if !ok {
		t.Fatalf("FROM = %T, want *ast.Join", fromClause(t, stmt.From).Table)
	}
	var shared pkast.JoinExprNode = join
	if shared.JoinExprCore() != &join.BaseJoinExpr {
		t.Fatal("shared core does not refer to the dialect node's data")
	}
	if join.Type != pkast.JoinTypeRight || join.Natural {
		t.Fatalf("join type/natural = %v/%v", join.Type, join.Natural)
	}
	if join.On == nil {
		t.Fatal("expected ON condition")
	}
	left, ok := join.Left.(*ast.TableSource)
	if !ok {
		t.Fatalf("left = %T, want *ast.TableSource", join.Left)
	}
	if left.Alias.Name != "a" || left.Alias.Quote != lex.QuoteUnquoted {
		t.Fatalf("left alias = %#v", left.Alias)
	}
	natural := mustParseSelect(t, `SELECT * FROM t NATURAL JOIN u;`)
	naturalJoin, ok := fromClause(t, natural.From).Table.(*ast.Join)
	if !ok || !naturalJoin.Natural || naturalJoin.Type != pkast.JoinTypeInner {
		t.Fatalf("natural join = %#v", fromClause(t, natural.From).Table)
	}
	if reflect.TypeFor[ast.JoinExprNode]() != reflect.TypeFor[pkast.JoinExprNode]() || reflect.TypeFor[ast.JoinType]() != reflect.TypeFor[pkast.JoinType]() {
		t.Fatal("join interfaces are not shared aliases")
	}
}

func TestJoinRestore(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want string
	}{
		{"inner", `SELECT * FROM t JOIN u ON t.id = u.id;`, `SELECT * FROM "t" INNER JOIN "u" ON "t"."id"="u"."id"`},
		{"cross", `SELECT * FROM t CROSS JOIN u;`, `SELECT * FROM "t" CROSS JOIN "u"`},
		{"left outer", `SELECT * FROM t LEFT OUTER JOIN u USING (id);`, `SELECT * FROM "t" LEFT JOIN "u" USING ("id")`},
		{"full", `SELECT * FROM t FULL JOIN u ON true;`, `SELECT * FROM "t" FULL JOIN "u" ON TRUE`},
		{"natural", `SELECT * FROM t NATURAL JOIN u;`, `SELECT * FROM "t" NATURAL INNER JOIN "u"`},
		{"using alias", `SELECT * FROM t JOIN u USING (id, "Tenant") AS m;`, `SELECT * FROM "t" INNER JOIN "u" USING ("id", "Tenant") AS "m"`},
		{"nested", `SELECT * FROM (t JOIN u USING (id)) AS x JOIN v ON true;`, `SELECT * FROM ("t" INNER JOIN "u" USING ("id")) AS "x" INNER JOIN "v" ON TRUE`},
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

func TestJoinUsingAlias(t *testing.T) {
	stmt := mustParseSelect(t, `SELECT * FROM t JOIN u USING (id) AS "Merged";`)
	join, ok := fromClause(t, stmt.From).Table.(*ast.Join)
	if !ok {
		t.Fatalf("FROM = %T, want *ast.Join", fromClause(t, stmt.From).Table)
	}
	if len(join.Using) != 1 || join.Using[0].Parts[0].Name != "id" {
		t.Fatalf("using = %#v", join.Using)
	}
	if join.UsingAlias.Name != "Merged" || join.UsingAlias.Quote != lex.QuoteDouble {
		t.Fatalf("using alias = %#v", join.UsingAlias)
	}
}

func TestJoinVisitor(t *testing.T) {
	stmt := mustParseSelect(t, `SELECT * FROM t JOIN u ON t.id = u.id;`)
	join, ok := fromClause(t, stmt.From).Table.(*ast.Join)
	if !ok {
		t.Fatalf("FROM = %T, want *ast.Join", fromClause(t, stmt.From).Table)
	}
	replacement := &ast.TableName{BaseTableRef: pkast.BaseTableRef{Name: ast.QualifiedName{Parts: []lex.Ident{{Name: "new"}}}}}
	visitor := &projectionVisitor{
		enter: func(n ast.Node) (ast.Node, bool) { return n, false },
		leave: func(n ast.Node) (ast.Node, bool) {
			if _, ok := n.(*ast.TableName); ok {
				return replacement, true
			}
			return n, true
		},
	}
	if _, ok := join.Accept(visitor); !ok {
		t.Fatal("visitor failed")
	}
	left, ok := join.Left.(*ast.TableSource)
	if !ok {
		t.Fatalf("left = %T", join.Left)
	}
	if name, ok := left.Source.(*ast.TableName); !ok || name.Name.Parts[0].Name != "new" {
		t.Fatalf("left source = %#v", left.Source)
	}
	visited := 0
	visitor.enter = func(n ast.Node) (ast.Node, bool) {
		visited++
		return n, false
	}
	visitor.leave = func(n ast.Node) (ast.Node, bool) { return n, true }
	if _, ok := join.Accept(visitor); !ok || visited == 0 {
		t.Fatalf("visitor: ok=%v visited=%d", ok, visited)
	}
	stopped := 0
	visitor.enter = func(n ast.Node) (ast.Node, bool) {
		stopped++
		return n, false
	}
	visitor.leave = func(n ast.Node) (ast.Node, bool) {
		stopped++
		return n, false
	}
	if _, ok := join.Accept(visitor); ok {
		t.Fatal("visitor stop was ignored")
	}
}

func TestJoinInvalidRestore(t *testing.T) {
	tests := []struct {
		name string
		node ast.Node
	}{
		{"nil join", (*ast.Join)(nil)},
		{"missing left", &ast.Join{}},
		{"typed nil left", &ast.Join{BaseJoinExpr: pkast.BaseJoinExpr{Left: (*ast.TableSource)(nil)}}},
		{"typed nil right", &ast.Join{BaseJoinExpr: pkast.BaseJoinExpr{Left: &ast.TableSource{}, Right: (*ast.TableSource)(nil)}}},
		{"unknown type", &ast.Join{BaseJoinExpr: pkast.BaseJoinExpr{Left: &ast.TableSource{}, Right: &ast.TableSource{}}}},
		{"on and using", &ast.Join{BaseJoinExpr: pkast.BaseJoinExpr{
			Left:  &ast.TableSource{},
			Right: &ast.TableSource{},
			On:    &ast.ColumnNameExpr{},
			Using: []ast.QualifiedName{{Parts: []lex.Ident{{Name: "id"}}}},
		}}},
		{"using alias without columns", &ast.Join{BaseJoinExpr: pkast.BaseJoinExpr{
			Left:  &ast.TableSource{},
			Right: &ast.TableSource{},
		}, UsingAlias: lex.Ident{Name: "m"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out strings.Builder
			ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &out)
			if err := tt.node.Restore(ctx); err == nil {
				t.Fatal("invalid join restored without error")
			}
		})
	}
}
