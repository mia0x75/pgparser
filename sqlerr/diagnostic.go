package sqlerr

import (
	"github.com/mia0x75/parserkit/diagnostic"
	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/pgparser/postgres"
)

// Diagnostic converts a PostgreSQL error into a parserkit diagnostic. The
// message stays unchanged and the code is the dialect SQLSTATE.
func (e *Error) Diagnostic(span loc.Span, cause error) *diagnostic.Diagnostic {
	if e == nil {
		return nil
	}
	state, ok := postgres.PgState[e.getPgErrCode()]
	if !ok {
		state = postgres.DefaultPgState
	}
	return diagnosticError(state, e.getMsg(), span, cause)
}

// SyntaxDiagnostic builds the diagnostic of a PostgreSQL syntax error.
func SyntaxDiagnostic(message string, span loc.Span, cause error) *diagnostic.Diagnostic {
	return diagnosticError(postgres.PgState[postgres.ErrSyntax], message, span, cause)
}

func diagnosticError(code, message string, span loc.Span, cause error) *diagnostic.Diagnostic {
	if code == "" {
		code = postgres.DefaultPgState
	}
	return &diagnostic.Diagnostic{
		Severity: diagnostic.SeverityError,
		Code:     code,
		Message:  message,
		Span:     span,
		Cause:    cause,
	}
}
