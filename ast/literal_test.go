package ast_test

import (
	"strings"
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/format"

	"github.com/mia0x75/pgparser/ast"
)

func restoreLiteral(t *testing.T, literal *ast.LiteralExpr) (string, error) {
	t.Helper()
	var sb strings.Builder
	ctx := format.NewRestoreCtx(&sb, format.DialectConfig{}, format.DefaultRestoreFlags)
	err := literal.Restore(ctx)
	return sb.String(), err
}

func TestLiteralRestore(t *testing.T) {
	t.Parallel()

	cases := []struct {
		kind  pkast.LiteralKind
		value string
		want  string
	}{
		{pkast.LiteralNull, "", "NULL"},
		{pkast.LiteralBoolean, "true", "TRUE"},
		{pkast.LiteralBoolean, "FALSE", "FALSE"},
		{pkast.LiteralInteger, "-12", "-12"},
		{pkast.LiteralFloat, "1.5e+2", "1.5e+2"},
		{pkast.LiteralDecimal, "0.001", "0.001"},
		{pkast.LiteralString, "it's", "'it''s'"},
		{pkast.LiteralBytes, "raw", "'raw'"},
		{pkast.LiteralHex, "ff", "x'ff'"},
		{pkast.LiteralBit, "1010", "b'1010'"},
		{pkast.LiteralBit, "", "b''"},
		{pkast.LiteralHex, "", "x''"},
	}
	for _, tc := range cases {
		got, err := restoreLiteral(t, ast.NewLiteralExpr(tc.kind, tc.value))
		if err != nil {
			t.Fatalf("kind %v: %v", tc.kind, err)
		}
		if got != tc.want {
			t.Fatalf("kind %v: got %q, want %q", tc.kind, got, tc.want)
		}
	}
}

func TestLiteralRestoreRejectsInvalidText(t *testing.T) {
	t.Parallel()

	cases := []*ast.LiteralExpr{
		{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralBoolean, Value: "yes"}},
		{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: "1; DROP TABLE t"}},
		{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralHex, Value: "fg"}},
		{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralBit, Value: "102"}},
		{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralUnknown}},
	}
	for _, literal := range cases {
		if _, err := restoreLiteral(t, literal); err == nil {
			t.Fatalf("kind %v value %q: expected an error", literal.Kind, literal.Value)
		}
	}
}

func TestLiteralGetValue(t *testing.T) {
	t.Parallel()

	if got := ast.NewLiteralExpr(pkast.LiteralInteger, "42").GetValue(); got != int64(42) {
		t.Fatalf("integer value is %#v", got)
	}
	if got := ast.NewLiteralExpr(pkast.LiteralFloat, "1.5").GetValue(); got != 1.5 {
		t.Fatalf("float value is %#v", got)
	}
	if got := ast.NewLiteralExpr(pkast.LiteralBoolean, "true").GetValue(); got != true {
		t.Fatalf("boolean value is %#v", got)
	}
	if got := ast.NewLiteralExpr(pkast.LiteralNull, "").GetValue(); got != nil {
		t.Fatalf("null value is %#v", got)
	}
}
