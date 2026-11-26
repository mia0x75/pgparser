package parser_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/mia0x75/parserkit/diagnostic"

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
	first := &pgparser.ParseError{Message: "first", Position: 1}
	second := &diagnostic.Diagnostic{Code: "42000", Message: "second"}
	merged := &pgparser.ParseErrors{Errs: []error{
		first,
		errors.Join(errors.New("not a diagnostic"), second),
	}}
	wrapped := fmt.Errorf("parse batch: %w", merged)
	diags := pgparser.ParseDiagnostics(wrapped)
	if len(diags) != 2 || diags[0].Message != "first" || diags[1] != second {
		t.Fatalf("diagnostics = %+v", diags)
	}
	if !errors.Is(wrapped, first) || !errors.Is(wrapped, second) {
		t.Fatal("merged error lost a cause")
	}
}

func TestEmptyParseErrors(t *testing.T) {
	for _, err := range []*pgparser.ParseErrors{nil, {}, {Errs: []error{nil}}} {
		if err.Error() != "" || len(pgparser.ParseDiagnostics(err)) != 0 {
			t.Fatalf("empty ParseErrors returned a message or diagnostic: %#v", err)
		}
	}
}
