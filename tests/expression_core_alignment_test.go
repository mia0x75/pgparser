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

func TestParseSharedExistsSubqueryCore(t *testing.T) {
	const existsSQL = "EXISTS (SELECT 1)"
	const notExistsSQL = "NOT EXISTS (SELECT 2)"
	const sql = "SELECT " + existsSQL + ", " + notExistsSQL
	stmt := mustParseOne(t, sql).(*dialectast.SelectStmt)

	expr, ok := stmt.Fields.Items[0].Expr.(*dialectast.ExistsSubqueryExpr)
	if !ok || expr.Not {
		t.Fatalf("EXISTS expression = %#v", stmt.Fields.Items[0].Expr)
	}
	if _, ok := expr.Sel.(*dialectast.SubqueryExpr); !ok {
		t.Fatalf("EXISTS subquery = %T", expr.Sel)
	}
	span := expr.Span()
	if !span.Within(len(sql)) || sql[span.Start:span.End] != existsSQL {
		t.Fatalf("EXISTS span = %v", span)
	}

	notExpr, ok := stmt.Fields.Items[1].Expr.(*dialectast.UnaryOperationExpr)
	if !ok || notExpr.Op != opcode.Not {
		t.Fatalf("NOT EXISTS expression = %#v", stmt.Fields.Items[1].Expr)
	}
	inner, ok := notExpr.Expr.(*dialectast.ExistsSubqueryExpr)
	if !ok || inner.Not {
		t.Fatalf("NOT EXISTS inner expression = %#v", notExpr.Expr)
	}
	span = notExpr.Span()
	if !span.Within(len(sql)) || sql[span.Start:span.End] != notExistsSQL {
		t.Fatalf("NOT EXISTS span = %v", span)
	}
}

func TestParseSharedCompareSubqueryCore(t *testing.T) {
	const anySQL = "1 = ANY (SELECT 1)"
	const allSQL = "2 <> ALL (SELECT 2)"
	const sql = "SELECT " + anySQL + ", " + allSQL
	stmt := mustParseOne(t, sql).(*dialectast.SelectStmt)
	for i, tc := range []struct {
		raw string
		op  opcode.Op
		all bool
	}{{raw: anySQL, op: opcode.EQ}, {raw: allSQL, op: opcode.NE, all: true}} {
		expr, ok := stmt.Fields.Items[i].Expr.(*dialectast.CompareSubqueryExpr)
		if !ok || expr.Op != tc.op || expr.All != tc.all {
			t.Fatalf("%s = %#v", tc.raw, stmt.Fields.Items[i].Expr)
		}
		if _, ok := expr.R.(*dialectast.SubqueryExpr); !ok {
			t.Fatalf("%s right operand = %T", tc.raw, expr.R)
		}
		span := expr.Span()
		if !span.Within(len(sql)) || sql[span.Start:span.End] != tc.raw {
			t.Fatalf("%s has span %v", tc.raw, span)
		}
	}
}

func TestParseSharedRowConstructorCore(t *testing.T) {
	tests := []struct {
		name  string
		raw   string
		items []string
	}{
		{name: "explicit", raw: "ROW(1, 2)", items: []string{"1", "2"}},
		{name: "implicit", raw: "(3, 4)", items: []string{"3", "4"}},
	}
	sql := "SELECT " + tests[0].raw + ", " + tests[1].raw
	stmt := mustParseOne(t, sql).(*dialectast.SelectStmt)
	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expr, ok := stmt.Fields.Items[i].Expr.(*dialectast.RowExpr)
			if !ok || len(expr.Items) != 2 {
				t.Fatalf("%s = %#v", tc.raw, stmt.Fields.Items[i].Expr)
			}
			span := expr.Span()
			if !span.Within(len(sql)) || sql[span.Start:span.End] != tc.raw {
				t.Fatalf("%s has span %v", tc.raw, span)
			}
			for j, want := range tc.items {
				childSpan := expr.Items[j].Span()
				if !childSpan.Within(len(sql)) || sql[childSpan.Start:childSpan.End] != want {
					t.Fatalf("%s item %d has span %v", tc.raw, j, childSpan)
				}
			}
		})
	}
}

func TestParsePostgresValuesRowCompatibility(t *testing.T) {
	const sql = "VALUES (1, 2), (3, 4)"
	stmt := mustParseOne(t, sql).(*dialectast.ValuesStmt)
	if len(stmt.Rows) != 2 {
		t.Fatalf("VALUES rows = %d, want 2", len(stmt.Rows))
	}
	for i, row := range stmt.Rows {
		if row == nil || len(row.Items) != 2 {
			t.Fatalf("VALUES row %d = %#v", i, row)
		}
	}
	if got := compactSQL(restoreNode(t, stmt)); got != sql {
		t.Fatalf("VALUES restore = %q, want %q", got, sql)
	}
	if _, err := parser.New().ParseOneStmt("VALUES ROW(1, 2)"); err == nil {
		t.Fatal("accepted MySQL-only ROW syntax in VALUES")
	}
}

func TestParseSharedCaseExpressionCore(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		value     string
		condition string
		result    string
		elseExpr  string
	}{
		{
			name:      "searched",
			raw:       "CASE WHEN a > 0 THEN 1 ELSE 2 END",
			condition: "a > 0",
			result:    "1",
			elseExpr:  "2",
		},
		{
			name:      "simple",
			raw:       "CASE a WHEN 1 THEN 'x' ELSE 'y' END",
			value:     "a",
			condition: "1",
			result:    "'x'",
			elseExpr:  "'y'",
		},
	}
	parts := make([]string, len(tests))
	for i, tc := range tests {
		parts[i] = tc.raw
	}
	sql := "SELECT " + strings.Join(parts, ", ")
	stmt := mustParseOne(t, sql).(*dialectast.SelectStmt)
	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expr, ok := stmt.Fields.Items[i].Expr.(*dialectast.CaseExpr)
			if !ok || len(expr.WhenClauses) != 1 {
				t.Fatalf("%s = %#v", tc.raw, stmt.Fields.Items[i].Expr)
			}
			span := expr.Span()
			if !span.Within(len(sql)) || sql[span.Start:span.End] != tc.raw {
				t.Fatalf("%s has span %v", tc.raw, span)
			}
			if tc.value == "" {
				if expr.Value != nil {
					t.Fatalf("%s value = %#v, want nil", tc.raw, expr.Value)
				}
			} else if expr.Value == nil || sql[expr.Value.Span().Start:expr.Value.Span().End] != tc.value {
				t.Fatalf("%s value = %#v", tc.raw, expr.Value)
			}
			clause := expr.WhenClauses[0]
			if clause == nil || clause.Expr == nil || clause.Result == nil || expr.ElseClause == nil {
				t.Fatalf("%s clause = %#v", tc.raw, clause)
			}
			for label, item := range map[string]struct {
				node pkast.Node
				want string
			}{
				"when":   {node: clause.Expr, want: tc.condition},
				"result": {node: clause.Result, want: tc.result},
				"else":   {node: expr.ElseClause, want: tc.elseExpr},
			} {
				childSpan := item.node.Span()
				if !childSpan.Within(len(sql)) || strings.TrimSpace(sql[childSpan.Start:childSpan.End]) != item.want {
					t.Fatalf("%s %s span = %v", tc.raw, label, childSpan)
				}
			}
		})
	}
}

func TestParseSharedCollateExpressionCore(t *testing.T) {
	const raw = "a COLLATE utf8mb4_bin"
	const sql = "SELECT " + raw
	stmt := mustParseOne(t, sql).(*dialectast.SelectStmt)
	expr, ok := stmt.Fields.Items[0].Expr.(*dialectast.CollateExpr)
	if !ok || expr.Collation != "utf8mb4_bin" || expr.Expr == nil {
		t.Fatalf("COLLATE expression = %#v", stmt.Fields.Items[0].Expr)
	}
	span := expr.Span()
	if !span.Within(len(sql)) || sql[span.Start:span.End] != raw {
		t.Fatalf("COLLATE span = %v", span)
	}
	childSpan := expr.Expr.Span()
	if !childSpan.Within(len(sql)) || sql[childSpan.Start:childSpan.End] != "a" {
		t.Fatalf("COLLATE child span = %v", childSpan)
	}
}

func TestParseSharedUnaryExpressionCore(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		op   opcode.Op
		arg  string
	}{
		{name: "not", raw: "NOT a", op: opcode.Not, arg: "a"},
		{name: "negative", raw: "-b", op: opcode.Minus, arg: "b"},
	}
	sql := "SELECT " + tests[0].raw + ", " + tests[1].raw
	stmt := mustParseOne(t, sql).(*dialectast.SelectStmt)
	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expr, ok := stmt.Fields.Items[i].Expr.(*dialectast.UnaryOperationExpr)
			if !ok || opcode.Op(expr.Op) != tc.op || expr.Expr == nil {
				t.Fatalf("%s = %#v", tc.raw, stmt.Fields.Items[i].Expr)
			}
			span := expr.Span()
			if !span.Within(len(sql)) || sql[span.Start:span.End] != tc.raw {
				t.Fatalf("%s has span %v", tc.raw, span)
			}
			childSpan := expr.Expr.Span()
			if !childSpan.Within(len(sql)) || sql[childSpan.Start:childSpan.End] != tc.arg {
				t.Fatalf("%s child span = %v", tc.raw, childSpan)
			}
		})
	}
}

func TestParseSharedCastExpressionCore(t *testing.T) {
	const raw = "CAST(1 AS CHAR)"
	const sql = "SELECT " + raw
	stmt := mustParseOne(t, sql).(*dialectast.SelectStmt)
	expr, ok := stmt.Fields.Items[0].Expr.(*dialectast.FuncCastExpr)
	if !ok || expr.FunctionType != dialectast.CastFunction || expr.Expr == nil || expr.Tp == nil {
		t.Fatalf("CAST expression = %#v", stmt.Fields.Items[0].Expr)
	}
	span := expr.Span()
	if !span.Within(len(sql)) || sql[span.Start:span.End] != raw {
		t.Fatalf("CAST span = %v", span)
	}
	childSpan := expr.Expr.Span()
	if !childSpan.Within(len(sql)) || sql[childSpan.Start:childSpan.End] != "1" {
		t.Fatalf("CAST child span = %v", childSpan)
	}
}

func TestParseSharedFunctionCallCore(t *testing.T) {
	const raw = "coalesce(a, b)"
	const sql = "SELECT " + raw
	stmt := mustParseOne(t, sql).(*dialectast.SelectStmt)
	expr, ok := stmt.Fields.Items[0].Expr.(*dialectast.FuncCallExpr)
	if !ok || len(expr.Name.Parts) != 1 || expr.Name.Parts[0].Name != "coalesce" || len(expr.Args) != 2 {
		t.Fatalf("function call = %#v", stmt.Fields.Items[0].Expr)
	}
	span := expr.Span()
	if !span.Within(len(sql)) || sql[span.Start:span.End] != raw {
		t.Fatalf("function call span = %v", span)
	}
	for i, want := range []string{"a", "b"} {
		argSpan := expr.Args[i].Span()
		if !argSpan.Within(len(sql)) || sql[argSpan.Start:argSpan.End] != want {
			t.Fatalf("argument %d span = %v", i, argSpan)
		}
	}
}

func TestParseSharedParenthesesAndScalarSubqueryCore(t *testing.T) {
	const parenSQL = "(a + b)"
	const subquerySQL = "(SELECT 1)"
	const sql = "SELECT " + parenSQL + ", " + subquerySQL
	stmt := mustParseOne(t, sql).(*dialectast.SelectStmt)

	paren, ok := stmt.Fields.Items[0].Expr.(*dialectast.ParenthesesExpr)
	if !ok || paren.Expr == nil {
		t.Fatalf("parentheses expression = %#v", stmt.Fields.Items[0].Expr)
	}
	span := paren.Span()
	if !span.Within(len(sql)) || sql[span.Start:span.End] != parenSQL {
		t.Fatalf("parentheses span = %v", span)
	}
	childSpan := paren.Expr.Span()
	if !childSpan.Within(len(sql)) || sql[childSpan.Start:childSpan.End] != "a + b" {
		t.Fatalf("parentheses child span = %v", childSpan)
	}

	subquery, ok := stmt.Fields.Items[1].Expr.(*dialectast.SubqueryExpr)
	if !ok || subquery.Query == nil {
		t.Fatalf("scalar subquery = %#v", stmt.Fields.Items[1].Expr)
	}
	span = subquery.Span()
	if !span.Within(len(sql)) || sql[span.Start:span.End] != subquerySQL {
		t.Fatalf("scalar subquery span = %v", span)
	}
}
