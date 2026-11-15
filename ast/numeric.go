package ast

import (
	"io"

	"github.com/mia0x75/pgparser/format"
)

// NumericLiteral keeps SQL text to avoid float64 rounding.
type NumericLiteral struct {
	exprNode
	Value string
}

func (n *NumericLiteral) Restore(ctx *format.RestoreCtx) error {
	ctx.WritePlain(n.Value)
	return ctx.Err()
}

func (n *NumericLiteral) Format(w io.Writer) {
	_, _ = io.WriteString(w, n.Value)
}

func (n *NumericLiteral) Accept(v Visitor) (Node, bool) {
	node, _ := v.Enter(n)
	return v.Leave(node)
}
