package ast

import "github.com/mia0x75/pgparser/format"

var _ StmtNode = &CallStmt{}

// CallStmt is a CALL statement.
type CallStmt struct {
	stmtNode

	Proc *FuncCallExpr
}

// Restore implements Node interface.
func (n *CallStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("CALL ")
	if err := n.Proc.Restore(ctx); err != nil {
		return err
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CallStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*CallStmt)
	node, ok := n.Proc.Accept(v)
	if !ok {
		return n, false
	}
	n.Proc = node.(*FuncCallExpr)
	return v.Leave(n)
}
