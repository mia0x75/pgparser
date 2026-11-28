package ast

import (
	"io"

	"github.com/mia0x75/pgparser/format"
)

// CollateExpr is an expression with an explicit collation.
type CollateExpr struct {
	exprNode

	Expr      ExprNode
	Collation string
}

// Restore implements Node interface.
func (n *CollateExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restoreNode(ctx, n)
	}

	if err := n.Expr.Restore(ctx); err != nil {
		return err
	}
	ctx.WriteKeyword(" COLLATE ")
	ctx.WriteName(n.Collation)
	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *CollateExpr) Format(w io.Writer) {
	FormatExpr(n.Expr, w)
}

// Accept implements Node Accept interface.
func (n *CollateExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	node, skip := v.Enter(n)
	outer, ok := node.(*CollateExpr)
	if !ok || outer == nil {
		return node, false
	}
	n = outer
	if skip {
		return v.Leave(n)
	}
	if n.Expr != nil {
		if !nodePresent(n.Expr) {
			return n, false
		}
		updated, ok := n.Expr.Accept(v)
		if !ok || !nodePresent(updated) {
			return n, false
		}
		replacement, ok := updated.(ExprNode)
		if !ok {
			return n, false
		}
		n.Expr = replacement
	}
	return v.Leave(n)
}
