package ast

import (
	"fmt"

	pkast "github.com/mia0x75/parserkit/ast"

	"github.com/mia0x75/pgparser/format"
)

// SetOperationNode preserves the concrete dialect node's identity.
type SetOperationNode = pkast.SetOperationNode

// QueryTailNode preserves the concrete dialect node's identity.
type QueryTailNode = pkast.QueryTailNode

// SetOprType is the type of a set operator.
type SetOprType = pkast.SetOperationType

const (
	// Union is "UNION" (distinct).
	Union = pkast.SetOperationUnion
	// UnionAll is "UNION ALL".
	UnionAll = pkast.SetOperationUnionAll
	// Except is "EXCEPT" (distinct).
	Except = pkast.SetOperationExcept
	// ExceptAll is "EXCEPT ALL".
	ExceptAll = pkast.SetOperationExceptAll
	// Intersect is "INTERSECT" (distinct).
	Intersect = pkast.SetOperationIntersect
	// IntersectAll is "INTERSECT ALL".
	IntersectAll = pkast.SetOperationIntersectAll
)

// SetOprStmt is a set operation with the tail of its query level.
type SetOprStmt struct {
	pkast.BaseSetOperation

	// Tail is the ORDER BY and LIMIT of the set operation.
	Tail pkast.BaseQueryTail

	// With is the optional common table expression clause.
	With *WithClause
	// Locking is the optional locking clause of the query level.
	Locking *List
}

// SetOperationCore returns the shared set operation data.
func (n *SetOprStmt) SetOperationCore() *pkast.BaseSetOperation {
	if n == nil {
		return nil
	}
	return &n.BaseSetOperation
}

// Restore writes the operands, the operator and the tail of the set operation.
func (n *SetOprStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return fmt.Errorf("restore set operation: nil statement")
	}
	if !nodePresent(n.Left) || !nodePresent(n.Right) {
		return fmt.Errorf("restore set operation: missing operand")
	}
	if n.Operator.String() == "" {
		return fmt.Errorf("restore set operation: invalid operator %d", n.Operator)
	}
	if n.With != nil {
		if err := n.With.Restore(ctx); err != nil {
			return fmt.Errorf("restore set operation with clause: %w", err)
		}
		ctx.WritePlain(" ")
	}
	if err := restoreSetOperand(ctx, n.Left, n.Operator, false); err != nil {
		return fmt.Errorf("restore set operation left operand: %w", err)
	}
	ctx.WriteKeyword(" " + n.Operator.String() + " ")
	if err := restoreSetOperand(ctx, n.Right, n.Operator, true); err != nil {
		return fmt.Errorf("restore set operation right operand: %w", err)
	}
	if err := restoreQueryTail(ctx, &n.Tail); err != nil {
		return err
	}
	if n.Locking != nil {
		if err := restoreLocking(ctx, n.Locking); err != nil {
			return err
		}
	}
	return ctx.Err()
}

// Accept walks both operands and the tail of the set operation.
func (n *SetOprStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	outer, ok := entered.(*SetOprStmt)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	if outer.With != nil {
		node, ok := outer.With.Accept(v)
		if !ok {
			return outer, false
		}
		with, ok := node.(*WithClause)
		if !ok || with == nil {
			return outer, false
		}
		outer.With = with
	}
	if !nodePresent(outer.Left) || !nodePresent(outer.Right) {
		return outer, false
	}
	updated, ok := outer.Left.Accept(v)
	if !ok {
		return outer, false
	}
	left, ok := updated.(QueryNode)
	if !ok || !nodePresent(left) {
		return outer, false
	}
	outer.Left = left
	updated, ok = outer.Right.Accept(v)
	if !ok {
		return outer, false
	}
	right, ok := updated.(QueryNode)
	if !ok || !nodePresent(right) {
		return outer, false
	}
	outer.Right = right
	if outer.Tail.OrderBy != nil {
		updated, ok = outer.Tail.OrderBy.Accept(v)
		if !ok {
			return outer, false
		}
		order, ok := updated.(OrderByClauseNode)
		if !ok || order.OrderByClauseCore() == nil {
			return outer, false
		}
		outer.Tail.OrderBy = order
	}
	if outer.Tail.Limit != nil {
		updated, ok = outer.Tail.Limit.Accept(v)
		if !ok {
			return outer, false
		}
		limit, ok := updated.(LimitNode)
		if !ok || limit.LimitCore() == nil {
			return outer, false
		}
		outer.Tail.Limit = limit
	}
	if outer.Locking != nil {
		updated, ok = outer.Locking.Accept(v)
		if !ok {
			return outer, false
		}
		locking, ok := updated.(*List)
		if !ok || locking == nil {
			return outer, false
		}
		outer.Locking = locking
	}
	return v.Leave(outer)
}

// QueryTailStmt carries a query that owns an ORDER BY or LIMIT tail.
type QueryTailStmt struct {
	pkast.BaseQueryTailStmt

	// With is the optional common table expression clause.
	With *WithClause
}

// QueryTailStmtCore returns the shared query tail data.
func (n *QueryTailStmt) QueryTailStmtCore() *pkast.BaseQueryTailStmt {
	if n == nil {
		return nil
	}
	return &n.BaseQueryTailStmt
}

// Restore writes the query, its parentheses and its tail.
func (n *QueryTailStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil || !nodePresent(n.Query) {
		return fmt.Errorf("restore query tail: missing query")
	}
	if n.With != nil {
		if err := n.With.Restore(ctx); err != nil {
			return fmt.Errorf("restore query tail with clause: %w", err)
		}
		ctx.WritePlain(" ")
	}
	if n.Parenthesized {
		ctx.WritePlain("(")
	}
	if err := n.Query.Restore(ctx); err != nil {
		return fmt.Errorf("restore query tail query: %w", err)
	}
	if n.Parenthesized {
		ctx.WritePlain(")")
	}
	return restoreQueryTail(ctx, &n.Tail)
}

// Accept walks the query and the tail.
func (n *QueryTailStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	outer, ok := entered.(*QueryTailStmt)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	if outer.With != nil {
		node, ok := outer.With.Accept(v)
		if !ok {
			return outer, false
		}
		with, ok := node.(*WithClause)
		if !ok || with == nil {
			return outer, false
		}
		outer.With = with
	}
	if !nodePresent(outer.Query) {
		return outer, false
	}
	updated, ok := outer.Query.Accept(v)
	if !ok {
		return outer, false
	}
	query, ok := updated.(QueryNode)
	if !ok || !nodePresent(query) {
		return outer, false
	}
	outer.Query = query
	if outer.Tail.OrderBy != nil {
		updated, ok = outer.Tail.OrderBy.Accept(v)
		if !ok {
			return outer, false
		}
		order, ok := updated.(OrderByClauseNode)
		if !ok || order.OrderByClauseCore() == nil {
			return outer, false
		}
		outer.Tail.OrderBy = order
	}
	if outer.Tail.Limit != nil {
		updated, ok = outer.Tail.Limit.Accept(v)
		if !ok {
			return outer, false
		}
		limit, ok := updated.(LimitNode)
		if !ok || limit.LimitCore() == nil {
			return outer, false
		}
		outer.Tail.Limit = limit
	}
	return v.Leave(outer)
}

// restoreSetOperand writes one operand of a set operation. An operand that
// carries its own ORDER BY or LIMIT needs parentheses to stay one query block.
func restoreSetOperand(ctx *format.RestoreCtx, operand QueryNode, operator SetOprType, right bool) error {
	if !nodePresent(operand) {
		return fmt.Errorf("restore set operation: nil operand")
	}
	switch v := operand.(type) {
	case *QueryTailStmt:
		if v.Parenthesized {
			return v.Restore(ctx)
		}
	case *SetOprStmt:
		precedence := setOperationPrecedence(v.Operator) - setOperationPrecedence(operator)
		if precedence >= 0 && (!right || precedence != 0) && v.With == nil && v.Locking == nil && v.Tail.OrderBy == nil && v.Tail.Limit == nil {
			return v.Restore(ctx)
		}
	case *SelectStmt:
		if v.With == nil && v.OrderBy == nil && v.Limit == nil {
			return v.Restore(ctx)
		}
	default:
		return operand.Restore(ctx)
	}
	ctx.WritePlain("(")
	if err := operand.Restore(ctx); err != nil {
		return err
	}
	ctx.WritePlain(")")
	return ctx.Err()
}

func setOperationPrecedence(operator SetOprType) int {
	if operator == Intersect || operator == IntersectAll {
		return 2
	}
	return 1
}

// restoreQueryTail writes the ORDER BY and LIMIT of one query level.
func restoreQueryTail(ctx *format.RestoreCtx, tail *pkast.BaseQueryTail) error {
	if tail == nil {
		return nil
	}
	if tail.OrderBy != nil {
		if tail.OrderBy.OrderByClauseCore() == nil {
			return fmt.Errorf("restore query tail: invalid ORDER BY clause")
		}
		ctx.WritePlain(" ")
		if err := tail.OrderBy.Restore(ctx); err != nil {
			return fmt.Errorf("restore query tail order by: %w", err)
		}
	}
	if tail.Limit != nil {
		if tail.Limit.LimitCore() == nil {
			return fmt.Errorf("restore query tail: invalid LIMIT clause")
		}
		ctx.WritePlain(" ")
		if err := tail.Limit.Restore(ctx); err != nil {
			return fmt.Errorf("restore query tail limit: %w", err)
		}
	}
	return ctx.Err()
}

var (
	_ SetOperationNode = (*SetOprStmt)(nil)
	_ QueryTailNode    = (*QueryTailStmt)(nil)
)
