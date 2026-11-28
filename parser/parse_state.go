package parser

import (
	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/pgparser/ast"
)

type parseState struct {
	result []ast.StmtNode
	lexer  Scanner
}

func yyGetParser(lex yyLexer) *parseState {
	return lex.(*Scanner).parser
}

func setParseResult(lex yyLexer, statements []ast.StmtNode) {
	yyGetParser(lex).result = statements
}

func appendStatement(lex yyLexer, statements []ast.StmtNode, node ast.Node, span loc.Span) []ast.StmtNode {
	if node == nil {
		return statements
	}
	node.SetSpan(span)
	if raw, ok := node.(*ast.RawStmt); ok {
		node = raw.Stmt
	}
	if stmt, ok := node.(ast.StmtNode); ok {
		stmt.SetText(lex.(*Scanner).stmtText())
		return append(statements, stmt)
	}
	return statements
}
