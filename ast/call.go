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
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("CALL ")
	if err := n.Proc.Restore(ctx); err != nil {
		return err
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CallStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*CallStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*CallStmt)
	node, ok := n.Proc.Accept(v)
	if !ok {
		return n, false
	}
	n.Proc = node.(*FuncCallExpr)
	return v.Leave(n)
}
