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
		if err := visitField(&body, token.NewFileSet(), expr, "n.Children", map[string]bool{"Child": true}, 0, true); err != nil {
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

func TestVisitFieldNilAndValueContracts(t *testing.T) {
	tests := []struct {
		name string
		typ  string
		want []string
		deny []string
	}{
		{
			name: "nil collection entries fail",
			typ:  "[]Node",
			want: []string{"if !nodePresent(n.Children[i0])", "n.Children[i0].Accept(v)"},
			deny: []string{"if n.Children[i0] != nil"},
		},
		{
			name: "optional node fields may be absent",
			typ:  "*Child",
			want: []string{"if n.Children != nil", "nodePresent(n.Children)"},
		},
		{
			name: "value node fields are traversed and written back",
			typ:  "Child",
			want: []string{"(&n.Children).Accept(v)", "updated.(*Child)", "n.Children = *replacement"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tc.typ)
			if err != nil {
				t.Fatal(err)
			}
			var body strings.Builder
			if err := visitField(&body, token.NewFileSet(), expr, "n.Children", map[string]bool{"Child": true}, 0, true); err != nil {
				t.Fatal(err)
			}
			for _, text := range tc.want {
				if !strings.Contains(body.String(), text) {
					t.Errorf("generated code %q does not contain %q", body.String(), text)
				}
			}
			for _, text := range tc.deny {
				if strings.Contains(body.String(), text) {
					t.Errorf("generated code %q unexpectedly contains %q", body.String(), text)
				}
			}
		})
	}
}

func TestAcceptPrefixValidatesReplacementBeforeSkip(t *testing.T) {
	prefix := acceptPrefix("Child")
	assertions := []string{
		"node, skip := v.Enter(n)",
		"outer, ok := node.(*Child)",
		"if !ok || outer == nil",
		"n = outer",
		"if skip { return v.Leave(n) }",
	}
	last := -1
	for _, assertion := range assertions {
		current := strings.Index(prefix, assertion)
		if current <= last {
			t.Fatalf("%q appears out of order in %q", assertion, prefix)
		}
		last = current
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "generated.go", "package ast; func visit(v Visitor, n *Child) (Node, bool) "+prefix+"return v.Leave(n) }", 0); err != nil {
		t.Fatalf("generated prefix is invalid Go: %v", err)
	}
}

func TestFieldTypeName(t *testing.T) {
	for _, tc := range []struct {
		typ  string
		want string
	}{
		{"*Child", "Child"}, {"Child", "Child"}, {"Node", "Node"}, {"QueryNode", "QueryNode"},
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
