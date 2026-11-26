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

func TestTableSourceSharedCore(t *testing.T) {
	const sql = `SELECT * FROM "S"."T" AS "Alias"("A", B);`
	stmt := mustParseSelect(t, sql)
	source, ok := fromClause(t, stmt.From).Table.(*ast.TableSource)
	if !ok {
		t.Fatalf("single FROM = %T, want *ast.TableSource", fromClause(t, stmt.From).Table)
	}
	var sharedSource pkast.TableSourceNode = source
	sharedFrom := stmt.From
	if sharedSource.TableSourceCore() != &source.BaseTableSource || sharedFrom.FromClauseCore() != &fromClause(t, stmt.From).BaseFromClause {
		t.Fatal("shared core does not refer to the dialect node's data")
	}
	checkIdent := func(got lex.Ident, name, raw string, quote lex.QuoteStyle) {
		t.Helper()
		start := strings.Index(sql, raw)
		want := lex.Ident{Name: name, Raw: raw, Quote: quote, Span: loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(raw))}}
		if got != want {
			t.Fatalf("identifier = %#v, want %#v", got, want)
		}
	}
	checkIdent(source.Alias, "Alias", `"Alias"`, lex.QuoteDouble)
	if len(source.Columns) != 2 {
		t.Fatalf("columns = %#v", source.Columns)
	}
	checkIdent(source.Columns[0], "A", `"A"`, lex.QuoteDouble)
	checkIdent(source.Columns[1], "b", "B", lex.QuoteUnquoted)
	table := source.Source.(*ast.TableName)
	if len(table.Name.Parts) != 2 {
		t.Fatalf("table name = %#v", table.Name)
	}
	checkIdent(table.Name.Parts[0], "S", `"S"`, lex.QuoteDouble)
	checkIdent(table.Name.Parts[1], "T", `"T"`, lex.QuoteDouble)
	if source.Span() != (loc.Span{Start: 14, End: loc.ByteOffset(len(sql) - 1)}) || stmt.From.Span().Start != 9 {
		t.Fatalf("source/from spans = %v/%v", source.Span(), stmt.From.Span())
	}
	if reflect.TypeFor[ast.TableSourceNode]() != reflect.TypeFor[pkast.TableSourceNode]() || reflect.TypeFor[ast.FromClauseNode]() != reflect.TypeFor[pkast.FromClauseNode]() {
		t.Fatal("source/from interfaces are not shared aliases")
	}
}

func TestTableSourceDialectRestore(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want string
	}{
		{"only sample", `SELECT * FROM ONLY "S"."T" AS t TABLESAMPLE system (1) REPEATABLE (2);`, `SELECT * FROM ONLY "S"."T" AS "t" TABLESAMPLE SYSTEM(1) REPEATABLE (2)`},
		{"lateral values", `SELECT * FROM LATERAL (VALUES (1)) AS v(x);`, `SELECT * FROM LATERAL (VALUES (1)) AS "v"("x")`},
		{"joined source alias", `SELECT * FROM (t JOIN u USING (id)) AS j(a, b);`, `SELECT * FROM ("t" INNER JOIN "u" USING ("id")) AS "j"("a", "b")`},
		{"function definitions", `SELECT * FROM f() AS t(x int);`, `SELECT * FROM F() AS "t"("x" "int4")`},
		{"using alias", `SELECT * FROM t JOIN u USING (id) AS "Merged";`, `SELECT * FROM "t" INNER JOIN "u" USING ("id") AS "Merged"`},
		{"nested query joins", `SELECT * FROM t JOIN (SELECT * FROM u JOIN v USING (id)) AS s ON true;`, `SELECT * FROM "t" INNER JOIN (SELECT * FROM "u" INNER JOIN "v" USING ("id")) AS "s" ON TRUE`},
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

func TestTableSourceVisitor(t *testing.T) {
	source := &ast.TableSource{BaseTableSource: pkast.BaseTableSource{Source: &ast.TableName{}}}
	clause := &ast.FromClause{BaseFromClause: pkast.BaseFromClause{Table: source}}
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
	if updated, ok := clause.Accept(visitor); !ok || updated != clause || source.Source != replacement {
		t.Fatalf("visitor replacement: node=%T ok=%v source=%#v", updated, ok, source.Source)
	}
	visitor.leave = func(n ast.Node) (ast.Node, bool) {
		if _, ok := n.(*ast.TableName); ok {
			return &ast.WildcardExpr{}, true
		}
		return n, true
	}
	if _, ok := clause.Accept(visitor); ok {
		t.Fatal("incompatible source replacement was accepted")
	}
	visited := 0
	visitor.enter = func(n ast.Node) (ast.Node, bool) {
		visited++
		return n, true
	}
	visitor.leave = func(n ast.Node) (ast.Node, bool) { return n, true }
	if _, ok := clause.Accept(visitor); !ok || visited != 1 {
		t.Fatalf("skip: ok=%v visited=%d", ok, visited)
	}
	visitor.enter = func(n ast.Node) (ast.Node, bool) { return n, false }
	visitor.leave = func(n ast.Node) (ast.Node, bool) { return n, false }
	if _, ok := clause.Accept(visitor); ok {
		t.Fatal("visitor stop was ignored")
	}
}

func TestTableSourceInvalidRestore(t *testing.T) {
	tests := []struct {
		name string
		node ast.Node
	}{
		{"nil source", (*ast.TableSource)(nil)},
		{"missing source", &ast.TableSource{}},
		{"typed nil source", &ast.TableSource{BaseTableSource: pkast.BaseTableSource{Source: (*ast.TableName)(nil)}}},
		{"missing from", &ast.FromClause{}},
		{"typed nil from", &ast.FromClause{BaseFromClause: pkast.BaseFromClause{Table: (*ast.TableSource)(nil)}}},
		{"columns without alias", &ast.TableSource{BaseTableSource: pkast.BaseTableSource{Source: &ast.TableName{}, Columns: []lex.Ident{{Name: "x"}}}}},
		{"only query", &ast.TableSource{BaseTableSource: pkast.BaseTableSource{Source: &ast.ValuesStmt{}}, Only: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out strings.Builder
			ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &out)
			if err := tt.node.Restore(ctx); err == nil {
				t.Fatal("invalid source restored without error")
			}
		})
	}
}
