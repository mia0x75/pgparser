package parser_test

import (
	"reflect"
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
	"github.com/mia0x75/pgparser/ast"
)

// Shared aliases must keep exact type identity with parserkit.
func TestProjectionSharedTypes(t *testing.T) {
	cases := map[string][2]reflect.Type{
		"SelectItem":   {reflect.TypeOf(ast.SelectItem{}), reflect.TypeOf(pkast.SelectItem{})},
		"FieldList":    {reflect.TypeOf(ast.FieldList{}), reflect.TypeOf(pkast.FieldList{})},
		"WildcardExpr": {reflect.TypeOf(ast.WildcardExpr{}), reflect.TypeOf(pkast.WildcardExpr{})},
	}
	for name, pair := range cases {
		if pair[0] != pair[1] {
			t.Fatalf("%s: ast type %v != parserkit type %v", name, pair[0], pair[1])
		}
	}
}

func TestProjectionSelectItemAliases(t *testing.T) {
	tests := []struct {
		name  string
		sql   string
		alias lex.Ident
		span  loc.Span
	}{
		{
			name: "no alias",
			sql:  "SELECT 1;",
			span: loc.InvalidSpan(),
		},
		{
			name:  "as foo",
			sql:   "SELECT 1 AS Foo;",
			alias: lex.Ident{Name: "foo", Raw: "Foo", Quote: lex.QuoteUnquoted},
			span:  loc.Span{Start: loc.ByteOffset(12), End: loc.ByteOffset(15)},
		},
		{
			name:  "implicit alias",
			sql:   "SELECT 1 foo;",
			alias: lex.Ident{Name: "foo", Raw: "foo", Quote: lex.QuoteUnquoted},
			span:  loc.Span{Start: loc.ByteOffset(9), End: loc.ByteOffset(12)},
		},
		{
			name:  "double quoted alias",
			sql:   `SELECT 1 AS "Result";`,
			alias: lex.Ident{Name: "Result", Raw: `"Result"`, Quote: lex.QuoteDouble},
			span:  loc.Span{Start: loc.ByteOffset(12), End: loc.ByteOffset(20)},
		},
		{
			name:  "unicode alias",
			sql:   `SELECT 1 AS U&"a\0062";`,
			alias: lex.Ident{Name: "ab", Raw: `U&"a\0062"`, Quote: lex.QuoteDouble},
			span:  loc.Span{Start: loc.ByteOffset(12), End: loc.ByteOffset(22)},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stmt := mustParseSelect(t, tc.sql)
			if len(stmt.Fields.Items) != 1 {
				t.Fatalf("fields = %d, want 1", len(stmt.Fields.Items))
			}
			item := stmt.Fields.Items[0]
			validateAlias(t, item, tc.alias, tc.span)
		})
	}
}

func TestProjectionWildcardQualifiers(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		wild     loc.Span
		qualPart lex.Ident
		qualSpan loc.Span
		itemSpan loc.Span
	}{
		{
			name:     "bare qualifier",
			sql:      `SELECT T.* AS "Result";`,
			wild:     loc.Span{Start: loc.ByteOffset(7), End: loc.ByteOffset(10)},
			qualPart: lex.Ident{Name: "t", Raw: "T", Quote: lex.QuoteUnquoted},
			qualSpan: loc.Span{Start: loc.ByteOffset(7), End: loc.ByteOffset(8)},
			itemSpan: loc.Span{Start: loc.ByteOffset(7), End: loc.ByteOffset(22)},
		},
		{
			name:     "quoted dotted qualifier",
			sql:      `SELECT "s.q".*;`,
			wild:     loc.Span{Start: loc.ByteOffset(7), End: loc.ByteOffset(14)},
			qualPart: lex.Ident{Name: "s.q", Raw: `"s.q"`, Quote: lex.QuoteDouble},
			qualSpan: loc.Span{Start: loc.ByteOffset(7), End: loc.ByteOffset(12)},
			itemSpan: loc.Span{Start: loc.ByteOffset(7), End: loc.ByteOffset(14)},
		},
		{
			name:     "unicode dotted qualifier",
			sql:      `SELECT U&"s\002Eq".*;`,
			wild:     loc.Span{Start: loc.ByteOffset(7), End: loc.ByteOffset(20)},
			qualPart: lex.Ident{Name: "s.q", Raw: `U&"s\002Eq"`, Quote: lex.QuoteDouble},
			qualSpan: loc.Span{Start: loc.ByteOffset(7), End: loc.ByteOffset(18)},
			itemSpan: loc.Span{Start: loc.ByteOffset(7), End: loc.ByteOffset(20)},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stmt := mustParseSelect(t, tc.sql)
			if len(stmt.Fields.Items) != 1 {
				t.Fatalf("fields = %d, want 1", len(stmt.Fields.Items))
			}
			item := stmt.Fields.Items[0]
			wild, ok := item.Expr.(*ast.WildcardExpr)
			if !ok {
				t.Fatalf("expr = %T, want *ast.WildcardExpr", item.Expr)
			}
			tc.qualPart.Span = tc.qualSpan
			if got, want := wild.Qualifier.Parts, []lex.Ident{tc.qualPart}; !reflect.DeepEqual(got, want) {
				t.Fatalf("qualifier = %#v, want %#v", got, want)
			}
			validateSpan(t, "wildcard", wild.Span(), tc.wild)
			validateSpan(t, "qualifier", wild.Qualifier.Span, tc.qualSpan)
			validateSpan(t, "item", item.Span(), tc.itemSpan)
		})
	}
}

func TestProjectionVisitorContract(t *testing.T) {
	tests := []struct {
		name  string
		enter func(n ast.Node) (ast.Node, bool)
		leave func(n ast.Node) (ast.Node, bool)
		want  int
	}{
		{
			name: "enter replace",
			enter: func(n ast.Node) (ast.Node, bool) {
				if _, ok := n.(*ast.WildcardExpr); ok {
					return &ast.WildcardExpr{Qualifier: ast.QualifiedName{Parts: []lex.Ident{{Name: "u"}}}}, false
				}
				return n, false
			},
			leave: func(n ast.Node) (ast.Node, bool) { return n, true },
			want:  1,
		},
		{
			name:  "leave replace",
			enter: func(n ast.Node) (ast.Node, bool) { return n, false },
			leave: func(n ast.Node) (ast.Node, bool) {
				if w, ok := n.(*ast.WildcardExpr); ok && len(w.Qualifier.Parts) == 0 {
					return &ast.WildcardExpr{Qualifier: ast.QualifiedName{Parts: []lex.Ident{{Name: "u"}}}}, true
				}
				return n, true
			},
			want: 1,
		},
		{
			name: "skip preserves original",
			enter: func(n ast.Node) (ast.Node, bool) {
				if _, ok := n.(*ast.FieldList); ok {
					return n, true
				}
				return n, false
			},
			leave: func(n ast.Node) (ast.Node, bool) { return n, true },
			want:  0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := &projectionVisitor{enter: tc.enter, leave: tc.leave}
			item := &ast.SelectItem{
				Expr:  &ast.WildcardExpr{},
				Alias: lex.Ident{Name: "Result", Raw: `"Result"`, Quote: lex.QuoteDouble},
			}
			list := &ast.FieldList{Items: []*ast.SelectItem{item}}
			updated, ok := list.Accept(v)
			if !ok {
				t.Fatal("Accept returned ok = false")
			}
			plist, ok := updated.(*ast.FieldList)
			if !ok {
				t.Fatalf("Accept = %T, want *ast.FieldList", updated)
			}
			got := 0
			for _, it := range plist.Items {
				if w, ok := it.Expr.(*ast.WildcardExpr); ok && len(w.Qualifier.Parts) == 1 {
					got++
				}
			}
			if got != tc.want {
				t.Fatalf("replacements = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestProjectionReturningWildcard(t *testing.T) {
	stmt, ok := mustParseOne(t, `INSERT INTO t VALUES (1) RETURNING t.* AS "Result";`).(*ast.InsertStmt)
	if !ok {
		t.Fatal("expected *ast.InsertStmt")
	}
	if stmt.Returning == nil || len(stmt.Returning.Items) != 1 {
		t.Fatalf("returning = %#v", stmt.Returning)
	}
	item := stmt.Returning.Items[0]
	validateSpan(t, "returning list", stmt.Returning.Span(), loc.Span{Start: 35, End: 50})
	wild, ok := item.Expr.(*ast.WildcardExpr)
	if !ok {
		t.Fatalf("expr = %T, want *ast.WildcardExpr", item.Expr)
	}
	if got, want := wild.Qualifier.Parts, []lex.Ident{{Name: "t", Raw: "t", Quote: lex.QuoteUnquoted, Span: loc.Span{Start: 35, End: 36}}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("qualifier = %#v, want %#v", got, want)
	}
	if item.Alias != (lex.Ident{Name: "Result", Raw: `"Result"`, Quote: lex.QuoteDouble, Span: loc.Span{Start: 42, End: 50}}) {
		t.Fatalf("alias = %#v, want Result", item.Alias)
	}
	if got := restoreString(t, stmt); got != `INSERT INTO "t" VALUES (1) RETURNING "t".* AS "Result"` {
		t.Fatalf("restore = %q", got)
	}
}

func TestProjectionVisitorStops(t *testing.T) {
	visited := 0
	visitor := &projectionVisitor{
		enter: func(n ast.Node) (ast.Node, bool) { return n, false },
		leave: func(n ast.Node) (ast.Node, bool) {
			if _, ok := n.(*ast.WildcardExpr); ok {
				visited++
				return n, false
			}
			return n, true
		},
	}
	list := &ast.FieldList{Items: []*ast.SelectItem{
		{Expr: &ast.WildcardExpr{}},
		{Expr: &ast.WildcardExpr{}},
	}}
	if _, ok := list.Accept(visitor); ok || visited != 1 {
		t.Fatalf("stop: ok=%v visited=%d, want false/1", ok, visited)
	}
}

type projectionVisitor struct {
	enter func(ast.Node) (ast.Node, bool)
	leave func(ast.Node) (ast.Node, bool)
}

func (v *projectionVisitor) Enter(n ast.Node) (ast.Node, bool) { return v.enter(n) }
func (v *projectionVisitor) Leave(n ast.Node) (ast.Node, bool) { return v.leave(n) }

func validateAlias(t *testing.T, item *ast.SelectItem, want lex.Ident, wantSpan loc.Span) {
	t.Helper()
	want.Span = wantSpan
	if item.Alias != want {
		t.Fatalf("alias = %#v, want %#v", item.Alias, want)
	}
	validateSpan(t, "alias", item.Alias.Span, wantSpan)
	if item.Alias.Present() != want.Present() {
		t.Fatalf("alias present = %v, want %v", item.Alias.Present(), want.Present())
	}
}

func validateSpan(t *testing.T, label string, got, want loc.Span) {
	t.Helper()
	if got != want {
		t.Fatalf("%s span = %v, want %v", label, got, want)
	}
}
