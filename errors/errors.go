// Package errors maps PostgreSQL dialect errors onto parserkit diagnostics.
//
// Generic wrapping uses the standard library. This package keeps only the
// helpers the codebase calls and the mapping from PG SQLSTATE codes to
// diagnostic metadata. Messages are never rewritten.
package errors

import (
	stderrors "errors"
	"fmt"

	"github.com/mia0x75/parserkit/diagnostic"
	"github.com/mia0x75/parserkit/loc"
)

// New returns an error with the supplied message.
func New(message string) error { return stderrors.New(message) }

// Errorf formats a message according to a format specifier.
func Errorf(format string, args ...any) error { return fmt.Errorf(format, args...) }

// Annotate prefixes err with message and keeps errors.Is and errors.As working.
// If err is nil, Annotate returns nil.
func Annotate(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// Annotatef formats the annotation message.
func Annotatef(err error, format string, args ...any) error {
	return Annotate(err, fmt.Sprintf(format, args...))
}

// Trace is retained for API compatibility. Error chains already carry their
// context, so there is no separate stack to add.
func Trace(err error) error { return err }

// AddStack is retained for API compatibility.
func AddStack(err error) error { return err }

// Cause returns the root cause of err, following standard error unwrapping.
func Cause(err error) error {
	for err != nil {
		next := stderrors.Unwrap(err)
		if next == nil {
			return err
		}
		err = next
	}
	return nil
}

// ErrorStack returns the error message. Parse locations live in diagnostics.
func ErrorStack(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// Diagnostic builds a diagnostic with PG error severity and an unchanged
// message. Code is the SQLSTATE of the dialect error.
func Diagnostic(code, message string, span loc.Span, cause error) *diagnostic.Diagnostic {
	return &diagnostic.Diagnostic{
		Severity: diagnostic.SeverityError,
		Code:     code,
		Message:  message,
		Span:     span,
		Cause:    cause,
	}
}

// AsDiagnostic returns the first diagnostic in err's chain.
func AsDiagnostic(err error) (*diagnostic.Diagnostic, bool) {
	var found *diagnostic.Diagnostic
	if stderrors.As(err, &found) {
		return found, true
	}
	return nil, false
}
