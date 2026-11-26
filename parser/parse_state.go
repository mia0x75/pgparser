package parser

import "github.com/mia0x75/pgparser/ast"

type parseState struct {
	result []ast.StmtNode
	lexer  Scanner
}
