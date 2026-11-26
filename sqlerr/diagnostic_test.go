package sqlerr

import (
	stderrors "errors"
	"testing"

	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/pgparser/postgres"
)

func TestErrorDiagnosticUsesPgState(t *testing.T) {
	source := ClassParser.New(CodeUnknown, "unsupported option %s")
	err := source.GenWithStack("bad option %s", "x")
	pgErr, ok := stderrors.Unwrap(err).(*Error)
	if !ok {
		pgErr, ok = err.(*Error)
	}
	if !ok {
		t.Fatalf("error = %T", err)
	}
	span := loc.Span{Start: 1, End: 2}
	diag := pgErr.Diagnostic(span, err)
	if diag == nil {
		t.Fatal("Diagnostic returned nil")
	}
	wantCode, ok := postgres.PgState[pgErr.getPgErrCode()]
	if !ok {
		wantCode = postgres.DefaultPgState
	}
	if diag.Code != wantCode || diag.Code == "" {
		t.Fatalf("code = %q, want %q", diag.Code, wantCode)
	}
	if diag.Message != "bad option x" || diag.Span != span {
		t.Fatalf("diagnostic = %+v", diag)
	}
	if !stderrors.Is(diag, err) {
		t.Fatal("diagnostic lost its cause")
	}

	var nilErr *Error
	if nilErr.Diagnostic(loc.InvalidSpan(), nil) != nil {
		t.Fatal("nil error must not produce a diagnostic")
	}
}

func TestSyntaxDiagnostic(t *testing.T) {
	diag := SyntaxDiagnostic("syntax error", loc.InvalidSpan(), nil)
	if diag.Code != postgres.PgState[postgres.ErrSyntax] || diag.Code != "42000" {
		t.Fatalf("code = %q", diag.Code)
	}
	if diag.Span.Valid() {
		t.Fatal("unknown span must stay invalid")
	}
}
