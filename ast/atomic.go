package ast

import (
	"fmt"

	"github.com/mia0x75/pgparser/format"
)

type BeginAtomicStmt struct {
	stmtNode
	List *List
}

func (n *BeginAtomicStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("BEGIN ATOMIC ")
	if n.List != nil {
		for _, stmt := range n.List.Items {
			if stmt == nil {
				return fmt.Errorf("atomic body has a nil statement")
			}
			if err := stmt.Restore(ctx); err != nil {
				return fmt.Errorf("atomic body: %w", err)
			}
			ctx.WritePlain("; ")
		}
	}
	ctx.WriteKeyword("END")
	return ctx.Err()
}

func (n *BeginAtomicStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	updated, skip := v.Enter(n)

	enteredNode, enteredOK := updated.(*BeginAtomicStmt)
	if !enteredOK || enteredNode == nil {
		return updated, false
	}
	n = enteredNode
	if skip {
		return v.Leave(updated)
	}
	n = updated.(*BeginAtomicStmt)
	if n.List != nil {
		child, ok := n.List.Accept(v)
		if !ok {
			return n, false
		}
		n.List = child.(*List)
	}
	return v.Leave(n)
}
