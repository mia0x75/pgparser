package parser

import (
	stderrors "errors"
	"fmt"

	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/errors"
)

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
	if ast.NewValueExpr == nil ||
		ast.NewParamMarkerExpr == nil ||
		ast.NewHexLiteral == nil ||
		ast.NewBitLiteral == nil {
		return nil, errors.New("no parser driver (forgotten import?)")
	}
	parser.result = nil

	parser.lexer.reset(sql)
	ret := yyParse(&parser.lexer)

	errs := parser.lexer.Errors()
	if len(errs) != 0 {
		// Keep a single error unwrappable so sqlerr class checks still match.
		if len(errs) == 1 {
			return nil, errors.Trace(errs[0])
		}
		return nil, errors.Trace(stderrors.Join(errs...))
	}
	if ret != 0 {
		return nil, errors.Trace(ErrSyntax)
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
	for _, stmt := range parser.result {
		ast.SetFlag(stmt)
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
		return nil, ErrSyntax
	}
	return stmts[0], nil
}

// ParseErrorWith returns a syntax error message compatible with PostgreSQL.
func ParseErrorWith(errstr string, lineno int) error {
	return fmt.Errorf("syntax error at line %d near %q", lineno, errstr)
}
