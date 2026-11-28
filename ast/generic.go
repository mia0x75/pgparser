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
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword(n.Kind)
	if n.Raw != "" {
		ctx.WritePlain(" ")
		ctx.WritePlain(n.Raw)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *GenericStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, ok := newNode.(*GenericStmt)
	if !ok || outer == nil {
		return newNode, false
	}
	n = outer
	if skipChildren {
		return v.Leave(n)
	}
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
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WritePlain(n.SQL)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *RawStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, ok := newNode.(*RawStmt)
	if !ok || outer == nil {
		return newNode, false
	}
	n = outer
	if skipChildren {
		return v.Leave(n)
	}
	if n.Stmt != nil {
		if !nodePresent(n.Stmt) {
			return n, false
		}
		updated, ok := n.Stmt.Accept(v)
		if !ok || !nodePresent(updated) {
			return n, false
		}
		stmt, ok := updated.(StmtNode)
		if !ok || !nodePresent(stmt) {
			return n, false
		}
		n.Stmt = stmt
	}
	return v.Leave(n)
}
