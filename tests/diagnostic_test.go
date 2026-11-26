package parser_test

import (
	"errors"
	"testing"

	pgparser "github.com/mia0x75/pgparser"
)

func TestParseErrorDiagnostic(t *testing.T) {
	_, err := pgparser.New().ParseOneStmt("SELECT FROM")
	if err == nil {
		t.Fatal("expected a parse error")
	}
	diags := pgparser.ParseDiagnostics(err)
	if len(diags) == 0 {
		t.Fatal("no diagnostics on parse error")
	}
	var parseErr *pgparser.ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("error = %T, want *ParseError", err)
	}
	diag := parseErr.Diagnostic()
	if diag.Code != "42000" {
		t.Fatalf("code = %q, want PG syntax SQLSTATE", diag.Code)
	}
	if diag.Message != parseErr.Message {
		t.Fatalf("message = %q, want %q", diag.Message, parseErr.Message)
	}
	if diag.Span != parseErr.Span {
		t.Fatalf("span = %v, want %v", diag.Span, parseErr.Span)
	}
	if !errors.Is(diag, parseErr) {
		t.Fatal("diagnostic lost its parse error cause")
	}
}

func TestParseErrorsMerge(t *testing.T) {
	merged := &pgparser.ParseErrors{Errs: []error{
		&pgparser.ParseError{Message: "first", Position: 1},
		&pgparser.ParseError{Message: "second", Position: 2},
	}}
	diags := pgparser.ParseDiagnostics(merged)
	if len(diags) != 2 || diags[0].Message != "first" || diags[1].Message != "second" {
		t.Fatalf("diagnostics = %+v", diags)
	}
	if !errors.Is(merged, merged.Errs[0]) || !errors.Is(merged, merged.Errs[1]) {
		t.Fatal("merged error lost a cause")
	}
}
