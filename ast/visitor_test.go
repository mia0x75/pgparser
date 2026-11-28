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

type nodeTypeVisitor struct {
	tableName   bool
	columnDef   bool
	columnName  bool
	genericStmt bool
	rawStmt     bool
}

func (v *nodeTypeVisitor) Enter(n Node) (Node, bool) {
	switch n.(type) {
	case *TableName:
		v.tableName = true
	case *ColumnDef:
		v.columnDef = true
	case *ColumnName:
		v.columnName = true
	case *GenericStmt:
		v.genericStmt = true
	case *RawStmt:
		v.rawStmt = true
	}
	return n, false
}

func (*nodeTypeVisitor) Leave(n Node) (Node, bool) { return n, true }

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

func TestCreateForeignTableVisitsEmbeddedCreateStmt(t *testing.T) {
	table := &TableName{BaseTableRef: pkast.BaseTableRef{
		Name: pkast.QualifiedName{Parts: []pkast.Ident{{Name: "remote_table"}}},
	}}
	column := &ColumnDef{Name: &ColumnName{Name: "id"}}
	n := &CreateForeignTableStmt{Base: CreateStmt{
		Relation:  table,
		TableElts: &List{Items: []Node{column}},
	}}
	visitor := &nodeTypeVisitor{}
	if _, ok := n.Accept(visitor); !ok {
		t.Fatal("visitor rejected CREATE FOREIGN TABLE")
	}
	if !visitor.tableName || !visitor.columnDef || !visitor.columnName {
		t.Fatalf("embedded CREATE TABLE children were not visited: %#v", visitor)
	}
}

func TestRawStmtVisitsNestedStatement(t *testing.T) {
	n := &RawStmt{SQL: "SELECT 1", Stmt: &GenericStmt{Kind: "SELECT", Raw: "1"}}
	visitor := &nodeTypeVisitor{}
	if _, ok := n.Accept(visitor); !ok {
		t.Fatal("visitor rejected RawStmt")
	}
	if !visitor.rawStmt || !visitor.genericStmt {
		t.Fatalf("RawStmt nested statement was not visited: %#v", visitor)
	}
}
