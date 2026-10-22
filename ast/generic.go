package ast

import (
	"github.com/mia0x75/pgparser/format"
)

// GenericStmt is a syntax-only statement node used for PostgreSQL statements
// whose dedicated AST representation has not been implemented yet. It keeps
// the statement kind so callers can still distinguish them.
type GenericStmt struct {
	stmtNode

	Kind string
	Raw  string
}

// Restore implements Node interface.
func (n *GenericStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord(n.Kind)
	if n.Raw != "" {
		ctx.WritePlain(" ")
		ctx.WritePlain(n.Raw)
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *GenericStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*GenericStmt)
	return v.Leave(n)
}

// RawStmt is a nested statement body whose SQL text is kept verbatim.
// It is used for routine and trigger bodies, not as a top-level fallback.
type RawStmt struct {
	stmtNode

	SQL string
	// Stmt is the statement of the raw statement.
	Stmt StmtNode
}

// Restore implements Node interface.
func (n *RawStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WritePlain(n.SQL)
	return nil
}

// Accept implements Node Accept interface.
func (n *RawStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*RawStmt)
	return v.Leave(n)
}
