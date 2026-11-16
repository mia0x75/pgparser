package postgres

import "fmt"

// SQLError records an error information, from executing SQL.
type SQLError struct {
	Code    uint16
	Message string
	State   string
}

// Error prints errors, with a formatted string.
func (e *SQLError) Error() string {
	return fmt.Sprintf("ERROR %d (%s): %s", e.Code, e.State, e.Message)
}

// NewErr generates a SQL error, with an error code and default format specifier defined in PgErrName.
func NewErr(errCode uint16, args ...any) *SQLError {
	e := &SQLError{Code: errCode}

	if s, ok := PgState[errCode]; ok {
		e.State = s
	} else {
		e.State = DefaultPgState
	}

	if format, ok := PgErrName[errCode]; ok {
		e.Message = fmt.Sprintf(format, args...)
	} else {
		e.Message = fmt.Sprint(args...)
	}

	return e
}

// NewErrf creates a SQL error, with an error code and a format specifier.
func NewErrf(errCode uint16, format string, args ...any) *SQLError {
	e := &SQLError{Code: errCode}

	if s, ok := PgState[errCode]; ok {
		e.State = s
	} else {
		e.State = DefaultPgState
	}

	e.Message = fmt.Sprintf(format, args...)

	return e
}
