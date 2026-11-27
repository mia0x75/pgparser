package parser_test

import (
	"strings"
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

func TestParseSharedPredicateExpressionCore(t *testing.T) {
	const inList = "a IN (1, 2)"
	const notInSubquery = "a NOT IN (SELECT b FROM t)"
	const between = "a BETWEEN 1 AND 3"
	const notBetween = "a NOT BETWEEN 1 AND 3"
	const isNull = "a IS NULL"
	const isNotNull = "a IS NOT NULL"
	const sql = "SELECT " + inList + ", " + notInSubquery + ", " + between + ", " + notBetween + ", " + isNull + ", " + isNotNull
	stmt := mustParseOne(t, sql).(*dialectast.SelectStmt)
	requireSpan := func(expr pkast.ExprNode, raw string) {
		t.Helper()
		span := expr.Span()
		if !span.Within(len(sql)) {
			t.Fatalf("%s: invalid span %v", raw, span)
		}
		if got := sql[span.Start:span.End]; got != raw {
			t.Fatalf("%s: span %v covers %q", raw, span, got)
		}
	}
	list, ok := stmt.Fields.Items[0].Expr.(*dialectast.PatternInExpr)
	if !ok || list.Not || list.Sel != nil || len(list.List) != 2 {
		t.Fatalf("IN list = %#v", stmt.Fields.Items[0].Expr)
	}
	requireSpan(list, inList)
	requireSpan(list.Expr, "a")
	requireSpan(list.List[0], "1")
	requireSpan(list.List[1], "2")

	subquery, ok := stmt.Fields.Items[1].Expr.(*dialectast.PatternInExpr)
	if !ok || !subquery.Not || subquery.Sel == nil || len(subquery.List) != 0 {
		t.Fatalf("NOT IN subquery = %#v", stmt.Fields.Items[1].Expr)
	}
	requireSpan(subquery, notInSubquery)
	requireSpan(subquery.Expr, "a")
	if query, ok := subquery.Sel.(*dialectast.SubqueryExpr); !ok || query.Query == nil {
		t.Fatalf("IN subquery = %#v", subquery.Sel)
	}

	for i, tc := range []struct {
		raw string
		not bool
	}{{between, false}, {notBetween, true}} {
		expr, ok := stmt.Fields.Items[i+2].Expr.(*dialectast.BetweenExpr)
		if !ok || expr.Not != tc.not {
			t.Fatalf("%s = %#v", tc.raw, stmt.Fields.Items[i+2].Expr)
		}
		requireSpan(expr, tc.raw)
		requireSpan(expr.Expr, "a")
		requireSpan(expr.Left, "1")
		requireSpan(expr.Right, "3")
	}

	for i, tc := range []struct {
		raw string
		not bool
	}{{isNull, false}, {isNotNull, true}} {
		expr, ok := stmt.Fields.Items[i+4].Expr.(*dialectast.IsNullExpr)
		if !ok || expr.Not != tc.not {
			t.Fatalf("%s = %#v", tc.raw, stmt.Fields.Items[i+4].Expr)
		}
		requireSpan(expr, tc.raw)
		requireSpan(expr.Expr, "a")
	}

	for _, malformed := range []string{"SELECT a IN ()", "SELECT a BETWEEN 1", "SELECT a IS"} {
		if _, err := parser.New().ParseOneStmt(malformed); err == nil {
			t.Errorf("accepted malformed predicate %q", malformed)
		}
	}
}

func TestParseSharedTruthPredicateCore(t *testing.T) {
	tests := []struct {
		name  string
		raw   string
		not   bool
		truth int64
	}{
		{name: "is true", raw: "a IS TRUE", truth: 1},
		{name: "is false", raw: "a IS FALSE"},
		{name: "is not true", raw: "a IS NOT TRUE", not: true, truth: 1},
		{name: "is not false", raw: "a IS NOT FALSE", not: true},
	}
	parts := make([]string, len(tests))
	for i, tc := range tests {
		parts[i] = tc.raw
	}
	sql := "SELECT " + strings.Join(parts, ", ")
	stmt := mustParseOne(t, sql).(*dialectast.SelectStmt)
	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expr, ok := stmt.Fields.Items[i].Expr.(*dialectast.IsTruthExpr)
			if !ok || expr.Not != tc.not || expr.True != tc.truth {
				t.Fatalf("%s = %#v", tc.raw, stmt.Fields.Items[i].Expr)
			}
			span := expr.Span()
			if !span.Within(len(sql)) || sql[span.Start:span.End] != tc.raw {
				t.Fatalf("%s has span %v", tc.raw, span)
			}
			childSpan := expr.Expr.Span()
			if !childSpan.Within(len(sql)) || sql[childSpan.Start:childSpan.End] != "a" {
				t.Fatalf("%s child has span %v", tc.raw, childSpan)
			}
		})
	}
}

func TestParsePostgresUnknownTruthCompatibility(t *testing.T) {
	const sql = "SELECT a IS UNKNOWN, a IS NOT UNKNOWN"
	stmt := mustParseOne(t, sql).(*dialectast.SelectStmt)
	for i, want := range []struct {
		not   bool
		truth int64
	}{{not: true}, {truth: 1}} {
		expr, ok := stmt.Fields.Items[i].Expr.(*dialectast.IsTruthExpr)
		if !ok || expr.Not != want.not || expr.True != want.truth {
			t.Fatalf("unknown predicate %d = %#v", i, stmt.Fields.Items[i].Expr)
		}
	}
}

func TestParseSharedLikePredicateCore(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		pattern string
		not     bool
	}{
		{name: "like", raw: "a LIKE 'x%'", pattern: "'x%'"},
		{name: "not like", raw: "a NOT LIKE 'y%'", pattern: "'y%'", not: true},
	}
	sql := "SELECT " + tests[0].raw + ", " + tests[1].raw
	stmt := mustParseOne(t, sql).(*dialectast.SelectStmt)
	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expr, ok := stmt.Fields.Items[i].Expr.(*dialectast.PatternLikeExpr)
			if !ok || expr.Not != tc.not {
				t.Fatalf("%s = %#v", tc.raw, stmt.Fields.Items[i].Expr)
			}
			span := expr.Span()
			if !span.Within(len(sql)) || sql[span.Start:span.End] != tc.raw {
				t.Fatalf("%s has span %v", tc.raw, span)
			}
			childSpan := expr.Expr.Span()
			if !childSpan.Within(len(sql)) || sql[childSpan.Start:childSpan.End] != "a" {
				t.Fatalf("%s subject has span %v", tc.raw, childSpan)
			}
			patternSpan := expr.Pattern.Span()
			if !patternSpan.Within(len(sql)) || sql[patternSpan.Start:patternSpan.End] != tc.pattern {
				t.Fatalf("%s pattern has span %v", tc.raw, patternSpan)
			}
		})
	}
}

func TestParsePostgresILikeCompatibility(t *testing.T) {
	const sql = "SELECT a NOT ILIKE 'x%' ESCAPE '!'"
	stmt := mustParseOne(t, sql).(*dialectast.SelectStmt)
	expr, ok := stmt.Fields.Items[0].Expr.(*dialectast.PatternLikeExpr)
	if !ok || !expr.ILike || !expr.Not || expr.Escape == nil {
		t.Fatalf("ILIKE expression = %#v", stmt.Fields.Items[0].Expr)
	}
	span := expr.Span()
	if !span.Within(len(sql)) || sql[span.Start:span.End] != "a NOT ILIKE 'x%' ESCAPE '!'" {
		t.Fatalf("ILIKE expression has span %v", span)
	}
}

func TestParsePostgresSimilarToCompatibility(t *testing.T) {
	const similar = "a SIMILAR TO 'x%'"
	const notSimilarEscape = "a NOT SIMILAR TO 'y%' ESCAPE '!'"
	const sql = "SELECT " + similar + ", " + notSimilarEscape
	stmt := mustParseOne(t, sql).(*dialectast.SelectStmt)
	for i, tc := range []struct {
		raw string
		not bool
	}{{raw: similar}, {raw: notSimilarEscape, not: true}} {
		expr, ok := stmt.Fields.Items[i].Expr.(*dialectast.PatternRegexpExpr)
		if !ok || expr.Not != tc.not {
			t.Fatalf("%s = %#v", tc.raw, stmt.Fields.Items[i].Expr)
		}
		span := expr.Span()
		if !span.Within(len(sql)) || sql[span.Start:span.End] != tc.raw {
			t.Fatalf("%s has span %v", tc.raw, span)
		}
	}
	if expr := stmt.Fields.Items[1].Expr.(*dialectast.PatternRegexpExpr); expr.Escape == nil {
		t.Fatal("SIMILAR TO ESCAPE expression was not preserved")
	}
}
