package ast

import (
	"fmt"

	pkast "github.com/mia0x75/parserkit/ast"

	"github.com/mia0x75/pgparser/format"
)

// LimitNode keeps pagination syntax attached to its concrete dialect outer.
type LimitNode = pkast.LimitNode

// Limit keeps PostgreSQL's FETCH ... WITH TIES outside the shared pagination.
type Limit struct {
	pkast.BaseLimit

	// WithTies is true for FETCH FIRST ... ROWS WITH TIES.
	WithTies bool
}

// LimitCore returns the shared pagination data without replacing this outer.
func (n *Limit) LimitCore() *pkast.BaseLimit {
	if n == nil {
		return nil
	}
	return &n.BaseLimit
}

// Restore writes the LIMIT, FETCH or OFFSET clause.
func (n *Limit) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restoreNode(ctx, n)
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return fmt.Errorf("restore limit: nil limit")
	}
	if n.WithTies {
		ctx.WriteKeyword("FETCH FIRST ")
	} else if n.Count != nil {
		ctx.WriteKeyword("LIMIT ")
	}
	if n.Count != nil {
		if err := n.Count.Restore(ctx); err != nil {
			return fmt.Errorf("restore limit count: %w", err)
		}
	}
	if n.WithTies {
		ctx.WriteKeyword(" ROWS WITH TIES")
	}
	if n.Offset != nil {
		if n.Count != nil || n.WithTies {
			ctx.WritePlain(" ")
		}
		ctx.WriteKeyword("OFFSET ")
		if err := n.Offset.Restore(ctx); err != nil {
			return fmt.Errorf("restore limit offset: %w", err)
		}
	}
	return ctx.Err()
}

// Accept walks the count and the offset expressions.
func (n *Limit) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	outer, ok := entered.(*Limit)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	if outer.Count != nil {
		if !nodePresent(outer.Count) {
			return outer, false
		}
		updated, ok := outer.Count.Accept(v)
		if !ok {
			return outer, false
		}
		count, ok := updated.(ExprNode)
		if !ok || !nodePresent(count) {
			return outer, false
		}
		outer.Count = count
	}
	if outer.Offset != nil {
		if !nodePresent(outer.Offset) {
			return outer, false
		}
		updated, ok := outer.Offset.Accept(v)
		if !ok {
			return outer, false
		}
		offset, ok := updated.(ExprNode)
		if !ok || !nodePresent(offset) {
			return outer, false
		}
		outer.Offset = offset
	}
	return v.Leave(outer)
}

var _ LimitNode = (*Limit)(nil)
