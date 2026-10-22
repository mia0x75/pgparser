package parser

import (
	"fmt"

	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/errors"
)

// ErrSyntax returns for sql syntax error.
var ErrSyntax = errors.New("You have an error in your SQL syntax")

// TrimComment trims MySQL/MariaDB style special comment markers. PostgreSQL
// does not use them, but the helper is kept for API parity.
func TrimComment(txt string) string {
	return txt
}

// Parser represents a parser instance. Some temporary objects are stored in it
// to reduce object allocations during Parse.
type Parser struct {
	result []ast.StmtNode
	src    string
	lexer  Scanner

	// The following fields are used by yyParse to reduce allocation.
	cache  []yySymType
	yylval yySymType
	yyVAL  yySymType
}

// New returns a Parser object.
func New() *Parser {
	if ast.NewValueExpr == nil ||
		ast.NewParamMarkerExpr == nil ||
		ast.NewHexLiteral == nil ||
		ast.NewBitLiteral == nil {
		panic("no parser driver (forgotten import?)")
	}
	return &Parser{
		cache: make([]yySymType, 200),
	}
}

// Parse parses a query string to raw ast.StmtNode.
func (parser *Parser) Parse(sql string) (stmt []ast.StmtNode, warns []error, err error) {
	sql = preprocessSQLInput(sql)
	parser.src = sql
	parser.result = parser.result[:0]

	parser.lexer.reset(sql)
	ret := yyParse(&parser.lexer)

	warns, errs := parser.lexer.Errors()
	if len(warns) > 0 {
		warns = append([]error(nil), warns...)
	} else {
		warns = nil
	}
	if len(errs) != 0 {
		return nil, warns, errors.Trace(errs[0])
	}
	if ret != 0 {
		return nil, warns, errors.Trace(ErrSyntax)
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
	return parser.result, warns, nil
}

// ParseOneStmt parses a query and returns an ast.StmtNode.
// The query must have exactly one statement, otherwise ErrSyntax is returned.
func (parser *Parser) ParseOneStmt(sql string) (ast.StmtNode, error) {
	stmts, _, err := parser.Parse(sql)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if len(stmts) != 1 {
		return nil, ErrSyntax
	}
	ast.SetFlag(stmts[0])
	return stmts[0], nil
}

// ParseErrorWith returns a syntax error message compatible with PostgreSQL.
func ParseErrorWith(errstr string, lineno int) error {
	return fmt.Errorf("syntax error at line %d near %q", lineno, errstr)
}
