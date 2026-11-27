package parser_test

import (
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/loc"
	dialectast "github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/opcode"
	"github.com/mia0x75/pgparser/parser"
)

func TestParseSharedLiteralCore(t *testing.T) {
	const prefix = "SELECT "
	for _, tc := range []struct {
		raw   string
		kind  pkast.LiteralKind
		value string
	}{
		{raw: "TrUe", kind: pkast.LiteralBoolean, value: "true"},
		{raw: "FaLsE", kind: pkast.LiteralBoolean, value: "false"},
		{raw: "NuLl", kind: pkast.LiteralNull},
		{raw: "001", kind: pkast.LiteralInteger, value: "1"},
		{raw: "1.25", kind: pkast.LiteralFloat, value: "1.25"},
		{raw: "'a''b'", kind: pkast.LiteralString, value: "a'b"},
		{raw: "B'101'", kind: pkast.LiteralBit, value: "101"},
		{raw: "X'cafe'", kind: pkast.LiteralHex, value: "cafe"},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			sql := prefix + tc.raw + ", 0"
			stmt := mustParseOne(t, sql).(*dialectast.SelectStmt)
			expr := stmt.Fields.Items[0].Expr
			lit, ok := expr.(*dialectast.LiteralExpr)
			if !ok {
				t.Fatalf("expression = %T, want a literal", expr)
			}
			core := lit.BaseLiteralExpr
			wantSpan := loc.Span{Start: loc.ByteOffset(len(prefix)), End: loc.ByteOffset(len(prefix) + len(tc.raw))}
			if core.Kind != tc.kind || core.Value != tc.value || core.Raw != tc.raw || core.Span() != wantSpan {
				t.Fatalf("literal core = %#v, want kind %v, value %q, raw %q, span %v", core, tc.kind, tc.value, tc.raw, wantSpan)
			}
		})
	}
	if _, err := parser.New().ParseOneStmt("SELECT 'unterminated"); err == nil {
		t.Fatal("accepted an unterminated literal")
	}
}

func TestParseSharedParameterCore(t *testing.T) {
	const sql = "SELECT $1, $2"
	stmt := mustParseOne(t, sql).(*dialectast.SelectStmt)
	for i, marker := range []string{"$1", "$2"} {
		expr := stmt.Fields.Items[i].Expr
		param, ok := expr.(*dialectast.ParamMarkerExpr)
		if !ok {
			t.Fatalf("expression %d = %T, want a parameter", i, expr)
		}
		core := param.BaseParameterExpr
		span := core.Span()
		if core.Index != i || !span.Within(len(sql)) || sql[span.Start:span.End] != marker {
			t.Fatalf("parameter %d: index = %d, span = %v, want %s", i, core.Index, span, marker)
		}
	}
}

func TestParsePostgresParameterIndirection(t *testing.T) {
	const sql = "SELECT $2[1]"
	stmt := mustParseOne(t, sql).(*dialectast.SelectStmt)
	indirection := stmt.Fields.Items[0].Expr.(*dialectast.A_Indirection)
	param := indirection.Arg.(*dialectast.ParamMarkerExpr)
	span := param.Span()
	if param.Index != 1 || !span.Within(len(sql)) || sql[span.Start:span.End] != "$2" {
		t.Fatalf("subscripted parameter: index = %d, span = %v", param.Index, span)
	}
	if _, err := parser.New().ParseOneStmt("SELECT $0"); err == nil {
		t.Fatal("accepted a zero parameter number")
	}
}

func TestParseSharedBinaryExpressionCore(t *testing.T) {
	const mixed = "a + b * c = d AND e <> f OR g < h"
	const grouped = "(a + b) * c"
	const sql = "SELECT " + mixed + ", " + grouped + ", a - b - c"
	stmt := mustParseOne(t, sql).(*dialectast.SelectStmt)
	requireBinary := func(expr pkast.ExprNode, op opcode.Op, raw string) *dialectast.BinaryOperationExpr {
		t.Helper()
		node, ok := expr.(*dialectast.BinaryOperationExpr)
		if !ok {
			t.Fatalf("%s: expression = %T, want binary operation", raw, expr)
		}
		span := node.Span()
		if node.Op != op || !span.Within(len(sql)) || sql[span.Start:span.End] != raw {
			t.Fatalf("%s: operator = %v, span = %v; want %v and exact source", raw, node.Op, span, op)
		}
		return node
	}
	or := requireBinary(stmt.Fields.Items[0].Expr, opcode.LogicOr, mixed)
	and := requireBinary(or.Left, opcode.LogicAnd, "a + b * c = d AND e <> f")
	eq := requireBinary(and.Left, opcode.EQ, "a + b * c = d")
	plus := requireBinary(eq.Left, opcode.Plus, "a + b * c")
	requireBinary(plus.Right, opcode.Mul, "b * c")
	requireBinary(and.Right, opcode.NE, "e <> f")
	requireBinary(or.Right, opcode.LT, "g < h")

	mul := requireBinary(stmt.Fields.Items[1].Expr, opcode.Mul, grouped)
	paren, ok := mul.Left.(*dialectast.ParenthesesExpr)
	if !ok {
		t.Fatalf("left operand of %s = %T, want parentheses", grouped, mul.Left)
	}
	requireBinary(paren.Expr, opcode.Plus, "a + b")
	minus := requireBinary(stmt.Fields.Items[2].Expr, opcode.Minus, "a - b - c")
	requireBinary(minus.Left, opcode.Minus, "a - b")

	if _, err := parser.New().ParseOneStmt("SELECT a +"); err == nil {
		t.Fatal("accepted a missing right operand")
	}
}
