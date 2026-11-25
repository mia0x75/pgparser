package ast

import (
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"
)

type replaceVisitor struct {
	visits int
	skip   bool
	stop   bool
}

func (v *replaceVisitor) Enter(n Node) (Node, bool) {
	if _, ok := n.(*LiteralExpr); ok {
		v.visits++
		return &LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralFloat, Value: "2.5"}}, false
	}
	return n, v.skip
}

func (v *replaceVisitor) Leave(n Node) (Node, bool) { return n, !v.stop }

func TestGeneratedVisitorChildren(t *testing.T) {
	value := &LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralFloat, Value: "1.5"}}
	n := &JsonValueExpr{RawExpr: value, FormattedExpr: value}
	v := &replaceVisitor{}
	if _, ok := n.Accept(v); !ok || v.visits != 2 {
		t.Fatalf("visits = %d", v.visits)
	}
	if n.RawExpr.(*LiteralExpr).Value != "2.5" || n.FormattedExpr.(*LiteralExpr).Value != "2.5" {
		t.Fatal("replacement discarded")
	}
	v = &replaceVisitor{skip: true}
	if _, ok := n.Accept(v); !ok || v.visits != 0 {
		t.Fatal("skip ignored")
	}
	v = &replaceVisitor{stop: true}
	if _, ok := n.Accept(v); ok || v.visits != 1 {
		t.Fatal("visitor stop ignored")
	}
	g := &GroupingSet{Sets: [][]ExprNode{{value}, {value}}}
	v = &replaceVisitor{}
	if _, ok := g.Accept(v); !ok || v.visits != 2 || g.Sets[1][0].(*LiteralExpr).Value != "2.5" {
		t.Fatal("nested slice replacement failed")
	}
}
