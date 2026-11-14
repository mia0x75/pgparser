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
	ctx.WriteKeyWord("BEGIN ATOMIC ")
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
	ctx.WriteKeyWord("END")
	return nil
}

func (n *BeginAtomicStmt) Accept(v Visitor) (Node, bool) {
	updated, skip := v.Enter(n)
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
