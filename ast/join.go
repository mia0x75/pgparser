package ast

import (
	"fmt"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"

	"github.com/mia0x75/pgparser/format"
)

// JoinExprNode exposes join syntax shared with other dialects.
type JoinExprNode = pkast.JoinExprNode

// JoinType classifies a join without dialect keyword spelling.
type JoinType = pkast.JoinType

// Join keeps PostgreSQL's USING alias outside the shared relation.
type Join struct {
	pkast.BaseJoinExpr

	// UsingAlias is the optional alias of a USING clause, e.g. JOIN u USING (id) AS m.
	UsingAlias lex.Ident
}

// JoinExprCore returns the shared data without replacing this outer.
func (n *Join) JoinExprCore() *pkast.BaseJoinExpr {
	if n == nil {
		return nil
	}
	return &n.BaseJoinExpr
}

// Restore writes the join, keeping the parentheses of a nested relation.
func (n *Join) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return fmt.Errorf("restore join: nil join")
	}
	if !nodePresent(n.Left) {
		return fmt.Errorf("restore join: missing left relation")
	}
	if n.Right != nil && !nodePresent(n.Right) {
		return fmt.Errorf("restore join: missing right relation")
	}
	if n.On != nil && len(n.Using) != 0 {
		return fmt.Errorf("restore join: both ON and USING")
	}
	if n.UsingAlias.Present() && len(n.Using) == 0 {
		return fmt.Errorf("restore join: USING alias without USING columns")
	}
	switch n.Type {
	case pkast.JoinTypeInner, pkast.JoinTypeLeft, pkast.JoinTypeRight, pkast.JoinTypeFull, pkast.JoinTypeCross:
	default:
		return fmt.Errorf("restore join: unsupported join type %d", n.Type)
	}
	level := ctx.JoinLevel
	defer func() { ctx.JoinLevel = level }()
	if level != 0 {
		ctx.WritePlain("(")
		defer ctx.WritePlain(")")
	}
	ctx.JoinLevel = level + 1
	if err := n.Left.Restore(ctx); err != nil {
		return fmt.Errorf("restore join left: %w", err)
	}
	ctx.JoinLevel = level
	if n.Right == nil {
		return ctx.Err()
	}
	if n.Natural {
		ctx.WriteKeyword(" NATURAL")
	}
	switch n.Type {
	case pkast.JoinTypeInner:
		ctx.WriteKeyword(" INNER")
	case pkast.JoinTypeLeft:
		ctx.WriteKeyword(" LEFT")
	case pkast.JoinTypeRight:
		ctx.WriteKeyword(" RIGHT")
	case pkast.JoinTypeFull:
		ctx.WriteKeyword(" FULL")
	case pkast.JoinTypeCross:
		ctx.WriteKeyword(" CROSS")
	}
	ctx.WriteKeyword(" JOIN ")
	ctx.JoinLevel = level + 1
	if err := n.Right.Restore(ctx); err != nil {
		return fmt.Errorf("restore join right: %w", err)
	}
	ctx.JoinLevel = level
	if n.On != nil {
		ctx.WriteKeyword(" ON ")
		if err := n.On.Restore(ctx); err != nil {
			return fmt.Errorf("restore join on: %w", err)
		}
	}
	if len(n.Using) != 0 {
		ctx.WriteKeyword(" USING (")
		for i, name := range n.Using {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if err := restoreQualifiedName(ctx, name); err != nil {
				return fmt.Errorf("restore join using: %w", err)
			}
		}
		ctx.WritePlain(")")
		if n.UsingAlias.Present() {
			ctx.WriteKeyword(" AS ")
			ctx.WriteIdent(n.UsingAlias)
		}
	}
	return ctx.Err()
}

// Accept walks the left relation, the right relation and the ON expression.
func (n *Join) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	outer, ok := entered.(*Join)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	if !nodePresent(outer.Left) {
		return outer, false
	}
	updated, ok := outer.Left.Accept(v)
	if !ok {
		return outer, false
	}
	left, ok := updated.(ResultSetNode)
	if !ok || !nodePresent(left) {
		return outer, false
	}
	outer.Left = left
	if outer.Right != nil {
		if !nodePresent(outer.Right) {
			return outer, false
		}
		updated, ok = outer.Right.Accept(v)
		if !ok {
			return outer, false
		}
		right, ok := updated.(ResultSetNode)
		if !ok || !nodePresent(right) {
			return outer, false
		}
		outer.Right = right
	}
	if outer.On != nil {
		updated, ok = outer.On.Accept(v)
		if !ok {
			return outer, false
		}
		on, ok := updated.(ExprNode)
		if !ok || !nodePresent(on) {
			return outer, false
		}
		outer.On = on
	}
	return v.Leave(outer)
}

var (
	_ ResultSetNode = (*Join)(nil)
	_ JoinExprNode  = (*Join)(nil)
)
