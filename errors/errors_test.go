package errors

import (
	stderrors "errors"
	"testing"

	"github.com/mia0x75/parserkit/diagnostic"
	"github.com/mia0x75/parserkit/loc"
)

func TestDiagnosticCarriesDialectCode(t *testing.T) {
	cause := stderrors.New("scanner failure")
	span := loc.Span{Start: 3, End: 7}
	diag := Diagnostic("42000", "syntax error", span, cause)
	if diag.Severity != diagnostic.SeverityError {
		t.Fatalf("severity = %v", diag.Severity)
	}
	if diag.Code != "42000" || diag.Message != "syntax error" || diag.Span != span {
		t.Fatalf("diagnostic = %+v", diag)
	}
	if diag.Error() != "syntax error" {
		t.Fatalf("message was rewritten: %q", diag.Error())
	}
	if !stderrors.Is(diag, cause) {
		t.Fatal("diagnostic lost its cause")
	}
	found, ok := AsDiagnostic(stderrors.Join(diag))
	if !ok || found != diag {
		t.Fatalf("AsDiagnostic = %v, %v", found, ok)
	}
}

func TestWrappersPreserveChains(t *testing.T) {
	root := New("root")
	wrapped := Annotatef(root, "step %d", 1)
	if wrapped.Error() != "step 1: root" {
		t.Fatalf("message = %q", wrapped.Error())
	}
	if Cause(wrapped) != root {
		t.Fatal("Cause did not reach the root")
	}
	if !stderrors.Is(wrapped, root) {
		t.Fatal("wrapping broke errors.Is")
	}
	if Trace(wrapped) != wrapped || AddStack(wrapped) != wrapped {
		t.Fatal("Trace and AddStack must not rewrite the error")
	}
	if ErrorStack(nil) != "" {
		t.Fatal("empty stack expected for nil")
	}
}
