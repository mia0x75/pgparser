package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestVisitField(t *testing.T) {
	for _, tc := range []struct {
		typ  string
		want bool
	}{
		{"*Child", true}, {"Node", true}, {"[][]ExprNode", true}, {"[]*Child", true},
		{"string", false}, {"[]int", false}, {"*External", false},
		{"QueryNode", true}, {"CTENode", true}, {"OrderByClauseNode", true},
	} {
		expr, err := parser.ParseExpr(tc.typ)
		if err != nil {
			t.Fatal(err)
		}
		var body strings.Builder
		if err := visitField(&body, token.NewFileSet(), expr, "n.Children", map[string]bool{"Child": true}, 0); err != nil {
			t.Fatal(err)
		}
		if got := body.Len() > 0; got != tc.want {
			t.Fatalf("%s: body = %q", tc.typ, body.String())
		}
		if tc.want {
			file, err := parser.ParseFile(token.NewFileSet(), "generated.go", "package ast; func visit() {"+body.String()+"}", 0)
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			ast.Inspect(file, func(n ast.Node) bool {
				if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == "Accept" {
					calls++
				}
				return true
			})
			if calls != 1 {
				t.Fatalf("%s: generated %d Accept calls", tc.typ, calls)
			}
		}
	}
}

func TestFieldTypeName(t *testing.T) {
	for _, tc := range []struct {
		typ  string
		want string
	}{
		{"*Child", "Child"}, {"Node", "Node"}, {"QueryNode", "QueryNode"},
		{"*External", "External"},
	} {
		expr, err := parser.ParseExpr(tc.typ)
		if err != nil {
			t.Fatal(err)
		}
		if got := fieldTypeName(expr, token.NewFileSet()); got != tc.want {
			t.Fatalf("%s: got %q", tc.typ, got)
		}
	}
}
