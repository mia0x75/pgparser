package ast

import (
	"github.com/mia0x75/pgparser/format"
)

// RowsFromExpr is a table function of a ROWS FROM call.
type RowsFromExpr struct {
	resultSetNode

	// Functions are the functions of the ROWS FROM call.
	Functions []*TableFunction
	// WithOrdinality is true for a WITH ORDINALITY call.
	WithOrdinality bool
}

// Restore implements Node interface.
func (n *RowsFromExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restoreNode(ctx, n)
	}

	ctx.WriteKeyword("ROWS FROM(")
	for i, fn := range n.Functions {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if err := fn.Restore(ctx); err != nil {
			return err
		}
	}
	ctx.WritePlain(")")
	if n.WithOrdinality {
		ctx.WriteKeyword(" WITH ORDINALITY")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *RowsFromExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	node, skip := v.Enter(n)
	outer, ok := node.(*RowsFromExpr)
	if !ok || outer == nil {
		return node, false
	}
	n = outer
	if skip {
		return v.Leave(n)
	}
	for i0 := range n.Functions {
		if !nodePresent(n.Functions[i0]) {
			return n, false
		}
		updated, ok := n.Functions[i0].Accept(v)
		if !ok || !nodePresent(updated) {
			return n, false
		}
		replacement, ok := updated.(*TableFunction)
		if !ok {
			return n, false
		}
		n.Functions[i0] = replacement
	}
	return v.Leave(n)
}
