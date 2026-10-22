package postgres

import (
	"errors"
	"fmt"
)

// Portable analogs of some common call errors.
var (
	ErrBadConn       = errors.New("connection was bad")
	ErrMalformPacket = errors.New("malform packet error")
)

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

// NewErr generates a SQL error, with an error code and default format specifier defined in MySQLErrName.
func NewErr(errCode uint16, args ...interface{}) *SQLError {
	e := &SQLError{Code: errCode}

	if s, ok := MySQLState[errCode]; ok {
		e.State = s
	} else {
		e.State = DefaultMySQLState
	}

	if format, ok := MySQLErrName[errCode]; ok {
		e.Message = fmt.Sprintf(format, args...)
	} else {
		e.Message = fmt.Sprint(args...)
	}

	return e
}

// NewErrf creates a SQL error, with an error code and a format specifier.
func NewErrf(errCode uint16, format string, args ...interface{}) *SQLError {
	e := &SQLError{Code: errCode}

	if s, ok := MySQLState[errCode]; ok {
		e.State = s
	} else {
		e.State = DefaultMySQLState
	}

	e.Message = fmt.Sprintf(format, args...)

	return e
}
