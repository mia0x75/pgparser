package ast

import (
	"fmt"
	"io"

	pkast "github.com/mia0x75/parserkit/ast"

	"github.com/mia0x75/pgparser/format"
)

// OrderItemNode admits dialect extensions without accepting grouping items.
type OrderItemNode = pkast.OrderItemNode

// OrderByClauseNode preserves the concrete dialect clause identity.
type OrderByClauseNode = pkast.OrderByClauseNode

// OrderItem keeps PostgreSQL's USING operator and NULLS ordering.
type OrderItem struct {
	pkast.BaseOrderItem

	// UseOp is the optional `USING <operator>` of the item.
	UseOp string
	// NullOrder indicates the NULLS ordering: 0 for default, 1 for NULLS FIRST, 2 for NULLS LAST.
	NullOrder int
}

// OrderItemCore returns the shared sorting data without replacing this outer.
func (n *OrderItem) OrderItemCore() *pkast.BaseOrderItem {
	if n == nil {
		return nil
	}
	return &n.BaseOrderItem
}

// Restore writes the sorted expression and its modifiers.
func (n *OrderItem) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil || n.Expr == nil {
		return fmt.Errorf("restore order item: missing expression")
	}
	if err := n.Expr.Restore(ctx); err != nil {
		return fmt.Errorf("restore order item expression: %w", err)
	}
	if n.UseOp != "" {
		ctx.WriteKeyword(" USING ")
		ctx.WritePlain(n.UseOp)
	} else if n.Desc {
		ctx.WriteKeyword(" DESC")
	}
	switch n.NullOrder {
	case 1:
		ctx.WriteKeyword(" NULLS FIRST")
	case 2:
		ctx.WriteKeyword(" NULLS LAST")
	}
	return ctx.Err()
}

// Format writes the sorted expression and its modifiers.
func (n *OrderItem) Format(w io.Writer) {
	if n == nil {
		return
	}
	if n.Expr != nil {
		FormatExpr(n.Expr, w)
	}
	if n.UseOp != "" {
		_, _ = fmt.Fprint(w, " USING ", n.UseOp)
	} else if n.Desc {
		_, _ = fmt.Fprint(w, " DESC")
	}
	switch n.NullOrder {
	case 1:
		_, _ = fmt.Fprint(w, " NULLS FIRST")
	case 2:
		_, _ = fmt.Fprint(w, " NULLS LAST")
	}
}

// Accept walks the sorted expression.
func (n *OrderItem) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	outer, ok := entered.(*OrderItem)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	if outer.Expr == nil {
		return v.Leave(outer)
	}
	updated, ok := outer.Expr.Accept(v)
	if !ok {
		return outer, false
	}
	expr, ok := updated.(ExprNode)
	if !ok || !nodePresent(expr) {
		return outer, false
	}
	outer.Expr = expr
	return v.Leave(outer)
}

// OrderByClause keeps the marker of the clause that belongs to a set operation.
type OrderByClause struct {
	pkast.BaseOrderByClause

	// ForUnion marks the clause of a set operation.
	ForUnion bool
}

// OrderByClauseCore returns the shared sorting data without replacing this outer.
func (n *OrderByClause) OrderByClauseCore() *pkast.BaseOrderByClause {
	if n == nil {
		return nil
	}
	return &n.BaseOrderByClause
}

// Restore writes the ORDER BY clause and its items.
func (n *OrderByClause) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return fmt.Errorf("restore order by clause: nil clause")
	}
	ctx.WriteKeyword("ORDER BY ")
	for i, item := range n.Items {
		if i != 0 {
			ctx.WritePlain(",")
		}
		if err := item.Restore(ctx); err != nil {
			return fmt.Errorf("restore order by item %d: %w", i, err)
		}
	}
	return ctx.Err()
}

// Format writes the ORDER BY clause and its items.
func (n *OrderByClause) Format(w io.Writer) {
	if n == nil {
		return
	}
	_, _ = fmt.Fprint(w, "ORDER BY ")
	for i, item := range n.Items {
		if i != 0 {
			_, _ = fmt.Fprint(w, ", ")
		}
		if order, ok := item.(*OrderItem); ok {
			order.Format(w)
			continue
		}
		if core := item.OrderItemCore(); core != nil && core.Expr != nil {
			FormatExpr(core.Expr, w)
			if core.Desc {
				_, _ = fmt.Fprint(w, " DESC")
			}
		}
	}
}

// Accept walks every sorting item.
func (n *OrderByClause) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	clause, ok := entered.(*OrderByClause)
	if !ok || clause == nil {
		return entered, false
	}
	if skip {
		return v.Leave(clause)
	}
	for i, item := range clause.Items {
		if !nodePresent(item) || item.OrderItemCore() == nil {
			return clause, false
		}
		updated, ok := item.Accept(v)
		if !ok {
			return clause, false
		}
		replacement, ok := updated.(OrderItemNode)
		if !ok || replacement.OrderItemCore() == nil {
			return clause, false
		}
		clause.Items[i] = replacement
	}
	return v.Leave(clause)
}

var (
	_ OrderItemNode     = (*OrderItem)(nil)
	_ OrderByClauseNode = (*OrderByClause)(nil)
)
