package ast

import (
	"github.com/mia0x75/pgparser/format"
)

// GroupingSetType is the type of a PostgreSQL grouping set.
type GroupingSetType int

// Grouping set types.
const (
	GroupingSetNone GroupingSetType = iota
	GroupingSetEmpty
	GroupingSetRollup
	GroupingSetCube
	GroupingSetSets
)

// GroupingSet is a PostgreSQL GROUPING SETS / ROLLUP / CUBE item.
type GroupingSet struct {
	node

	// Type is the kind of the grouping set.
	Type GroupingSetType
	// Sets holds the expressions of the grouping set.
	Sets [][]ExprNode
}

// Restore implements Node interface.
func (n *GroupingSet) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restoreNode(ctx, n)
	}

	switch n.Type {
	case GroupingSetEmpty:
		ctx.WritePlain("()")
	case GroupingSetRollup:
		ctx.WriteKeyword("ROLLUP (")
		if err := restoreExprList(ctx, n.Sets[0]); err != nil {
			return err
		}
		ctx.WritePlain(")")
	case GroupingSetCube:
		ctx.WriteKeyword("CUBE (")
		if err := restoreExprList(ctx, n.Sets[0]); err != nil {
			return err
		}
		ctx.WritePlain(")")
	case GroupingSetSets:
		ctx.WriteKeyword("GROUPING SETS (")
		for i, set := range n.Sets {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			ctx.WritePlain("(")
			if err := restoreExprList(ctx, set); err != nil {
				return err
			}
			ctx.WritePlain(")")
		}
		ctx.WritePlain(")")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *GroupingSet) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	node, skip := v.Enter(n)
	outer, ok := node.(*GroupingSet)
	if !ok || outer == nil {
		return node, false
	}
	n = outer
	if skip {
		return v.Leave(n)
	}
	for i0 := range n.Sets {
		for i1 := range n.Sets[i0] {
			if !nodePresent(n.Sets[i0][i1]) {
				return n, false
			}
			updated, ok := n.Sets[i0][i1].Accept(v)
			if !ok || !nodePresent(updated) {
				return n, false
			}
			replacement, ok := updated.(ExprNode)
			if !ok {
				return n, false
			}
			n.Sets[i0][i1] = replacement
		}
	}
	return v.Leave(n)
}

func restoreExprList(ctx *format.RestoreCtx, exprs []ExprNode) error {
	for i, e := range exprs {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if err := e.Restore(ctx); err != nil {
			return err
		}
	}
	return ctx.Err()
}
