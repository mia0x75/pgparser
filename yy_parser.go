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
	if len(e.Errs) == 0 {
		return ""
	}
	return stderrors.Join(e.Errs...).Error()
}

// Unwrap exposes every parse error to errors.Is and errors.As.
func (e *ParseErrors) Unwrap() []error { return e.Errs }

// ParseDiagnostics collects the parserkit diagnostics of a parse error.
func ParseDiagnostics(err error) []*diagnostic.Diagnostic {
	if err == nil {
		return nil
	}
	switch e := err.(type) {
	case *ParseErrors:
		var result []*diagnostic.Diagnostic
		for _, child := range e.Errs {
			result = append(result, ParseDiagnostics(child)...)
		}
		return result
	case *ParseError:
		return []*diagnostic.Diagnostic{e.Diagnostic()}
	}
	var found *diagnostic.Diagnostic
	if stderrors.As(err, &found) {
		return []*diagnostic.Diagnostic{found}
	}
	return nil
}

// ErrSyntax returns for sql syntax error.
var ErrSyntax = errors.New("You have an error in your SQL syntax")

// Parser represents a parser instance. Some temporary objects are stored in it
// to reduce object allocations during Parse.
//
// A Parser is not safe for concurrent use: Parse stores its result and lexer
// state on the receiver. Create one Parser per goroutine.
type Parser struct {
	result []ast.StmtNode
	lexer  Scanner
}

// New returns a Parser object.
func New() *Parser {
	return &Parser{}
}

// Parse parses a query string to raw ast.StmtNode.
func (parser *Parser) Parse(sql string) (stmt []ast.StmtNode, err error) {
	parser.result = nil

	parser.lexer.reset(sql)
	ret := yyParse(&parser.lexer)

	errs := parser.lexer.Errors()
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
	if parser.lexer.result != nil {
		for _, item := range parser.lexer.result.Items {
			if raw, ok := item.(*ast.RawStmt); ok {
				if stmt, ok := raw.Stmt.(ast.StmtNode); ok {
					parser.result = append(parser.result, stmt)
				}
				continue
			}
			if stmt, ok := item.(ast.StmtNode); ok {
				parser.result = append(parser.result, stmt)
			}
		}
	}
	return parser.result, nil
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
