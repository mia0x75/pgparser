package parser

import (
	stderrors "errors"
	"fmt"

	"github.com/mia0x75/parserkit/diagnostic"
	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/errors"
	"github.com/mia0x75/pgparser/sqlerr"
)

// ParseErrors merges the errors produced by one parse. Messages and cause
// identity stay unchanged; only structured metadata is collected.
type ParseErrors struct {
	Errs []error
}

// Error returns the joined messages of all parse errors.
func (e *ParseErrors) Error() string {
	if e != nil {
		if joined := stderrors.Join(e.Errs...); joined != nil {
			return joined.Error()
		}
	}
	return ""
}

// Unwrap exposes every parse error to errors.Is and errors.As.
func (e *ParseErrors) Unwrap() []error {
	if e == nil {
		return nil
	}
	return e.Errs
}

// ParseDiagnostics collects the parserkit diagnostics of a parse error.
func ParseDiagnostics(err error) []*diagnostic.Diagnostic {
	return appendParseDiagnostics(nil, err)
}

func appendParseDiagnostics(result []*diagnostic.Diagnostic, err error) []*diagnostic.Diagnostic {
	switch e := err.(type) {
	case *diagnostic.Diagnostic:
		if e != nil {
			return append(result, e)
		}
	case *ParseError:
		if e != nil {
			return append(result, e.Diagnostic())
		}
	case interface{ Unwrap() []error }:
		for _, child := range e.Unwrap() {
			result = appendParseDiagnostics(result, child)
		}
	case interface{ Unwrap() error }:
		return appendParseDiagnostics(result, e.Unwrap())
	}
	return result
}

// ErrSyntax returns for sql syntax error.
var ErrSyntax = errors.New("You have an error in your SQL syntax")

// Parser represents a parser instance. It is safe for concurrent use because
// each parse keeps its mutable state locally.
type Parser struct{}

// New returns a Parser object.
func New() *Parser {
	return &Parser{}
}

// Parse parses a query string to raw ast.StmtNode.
func (*Parser) Parse(sql string) (stmt []ast.StmtNode, err error) {
	state := parseState{}
	state.lexer.reset(sql)
	ret := yyParse(&state.lexer)

	errs := state.lexer.Errors()
	if len(errs) != 0 {
		// Keep a single error unwrappable so sqlerr class checks still match.
		if len(errs) == 1 {
			return nil, errors.Trace(errs[0])
		}
		return nil, errors.Trace(&ParseErrors{Errs: errs})
	}
	if ret != 0 {
		return nil, errors.Trace(sqlerr.SyntaxDiagnostic(ErrSyntax.Error(), loc.InvalidSpan(), ErrSyntax))
	}
	if state.lexer.result != nil {
		for _, item := range state.lexer.result.Items {
			if raw, ok := item.(*ast.RawStmt); ok {
				if stmt, ok := raw.Stmt.(ast.StmtNode); ok {
					state.result = append(state.result, stmt)
				}
				continue
			}
			if stmt, ok := item.(ast.StmtNode); ok {
				state.result = append(state.result, stmt)
			}
		}
	}
	return state.result, nil
}

// ParseOneStmt parses a query and returns an ast.StmtNode.
// The query must have exactly one statement, otherwise ErrSyntax is returned.
func (parser *Parser) ParseOneStmt(sql string) (ast.StmtNode, error) {
	stmts, err := parser.Parse(sql)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if len(stmts) != 1 {
		return nil, errors.Trace(sqlerr.SyntaxDiagnostic(ErrSyntax.Error(), loc.InvalidSpan(), ErrSyntax))
	}
	return stmts[0], nil
}

// ParseErrorWith returns a syntax error message compatible with PostgreSQL.
func ParseErrorWith(errstr string, lineno int) error {
	return fmt.Errorf("syntax error at line %d near %q", lineno, errstr)
}
