package ast

import (
	"fmt"

	pkast "github.com/mia0x75/parserkit/ast"

	"github.com/mia0x75/pgparser/format"
)

// CTENode exposes the shared CTE syntax without hiding the dialect outer.
type CTENode = pkast.CTENode

// WithClauseNode preserves the dialect WITH clause identity.
type WithClauseNode = pkast.WithClauseNode

// CommonTableExpression keeps PostgreSQL's MATERIALIZED, SEARCH and CYCLE options.
type CommonTableExpression struct {
	pkast.BaseCommonTableExpression

	// Materialized is the materialization mode of the CTE body.
	Materialized CTEMaterialize
	// Search is the optional SEARCH clause of a recursive CTE.
	Search *CTESearchClause
	// Cycle is the optional CYCLE clause of a recursive CTE.
	Cycle *CTECycleClause
}

// CTECore returns the shared data without replacing this outer.
func (n *CommonTableExpression) CTECore() *pkast.BaseCommonTableExpression {
	if n == nil {
		return nil
	}
	return &n.BaseCommonTableExpression
}

// Restore writes the CTE name, its columns, its body and the recursive options.
func (n *CommonTableExpression) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restoreNode(ctx, n)
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return fmt.Errorf("restore common table expression: nil cte")
	}
	if !nodePresent(n.Query) {
		return fmt.Errorf("restore common table expression: missing query")
	}
	ctx.WriteIdent(n.Name)
	if len(n.Columns) > 0 {
		ctx.WritePlain(" (")
		for i, column := range n.Columns {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			ctx.WriteIdent(column)
		}
		ctx.WritePlain(")")
	}
	ctx.WriteKeyword(" AS ")
	switch n.Materialized {
	case CTEMaterializeDefault:
	case CTEMaterializeAlways:
		ctx.WriteKeyword("MATERIALIZED ")
	case CTEMaterializeNever:
		ctx.WriteKeyword("NOT MATERIALIZED ")
	default:
		return fmt.Errorf("restore common table expression: unknown materialization %d", n.Materialized)
	}
	ctx.WritePlain("(")
	if err := n.Query.Restore(ctx); err != nil {
		return fmt.Errorf("restore common table expression query: %w", err)
	}
	ctx.WritePlain(")")
	if n.Search != nil {
		ctx.WritePlain(" ")
		if err := n.Search.Restore(ctx); err != nil {
			return fmt.Errorf("restore common table expression search: %w", err)
		}
	}
	if n.Cycle != nil {
		ctx.WritePlain(" ")
		if err := n.Cycle.Restore(ctx); err != nil {
			return fmt.Errorf("restore common table expression cycle: %w", err)
		}
	}
	return ctx.Err()
}

// Accept walks the CTE body and the recursive options.
func (n *CommonTableExpression) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	outer, ok := entered.(*CommonTableExpression)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	if !nodePresent(outer.Query) {
		return outer, false
	}
	updated, ok := outer.Query.Accept(v)
	if !ok {
		return outer, false
	}
	query, ok := updated.(StmtNode)
	if !ok || !nodePresent(query) {
		return outer, false
	}
	outer.Query = query
	if outer.Search != nil {
		updated, ok = outer.Search.Accept(v)
		if !ok {
			return outer, false
		}
		search, ok := updated.(*CTESearchClause)
		if !ok || search == nil {
			return outer, false
		}
		outer.Search = search
	}
	if outer.Cycle != nil {
		updated, ok = outer.Cycle.Accept(v)
		if !ok {
			return outer, false
		}
		cycle, ok := updated.(*CTECycleClause)
		if !ok || cycle == nil {
			return outer, false
		}
		outer.Cycle = cycle
	}
	return v.Leave(outer)
}

// WithClause shares the CTE list with the other dialects.
type WithClause struct {
	pkast.BaseWithClause
}

// WithClauseCore returns the shared data without replacing this outer.
func (n *WithClause) WithClauseCore() *pkast.BaseWithClause {
	if n == nil {
		return nil
	}
	return &n.BaseWithClause
}

// Restore writes the WITH clause and every CTE it holds.
func (n *WithClause) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restoreNode(ctx, n)
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return fmt.Errorf("restore with clause: nil clause")
	}
	ctx.WriteKeyword("WITH ")
	if n.IsRecursive {
		ctx.WriteKeyword("RECURSIVE ")
	}
	for i, cte := range n.CTEs {
		if !nodePresent(cte) || cte.CTECore() == nil {
			return fmt.Errorf("restore with clause: nil cte at %d", i)
		}
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if err := cte.Restore(ctx); err != nil {
			return fmt.Errorf("restore with clause cte %d: %w", i, err)
		}
	}
	return ctx.Err()
}

// Accept walks every CTE of the clause.
func (n *WithClause) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	clause, ok := entered.(*WithClause)
	if !ok || clause == nil {
		return entered, false
	}
	if skip {
		return v.Leave(clause)
	}
	for i, cte := range clause.CTEs {
		if !nodePresent(cte) || cte.CTECore() == nil {
			return clause, false
		}
		updated, ok := cte.Accept(v)
		if !ok {
			return clause, false
		}
		replacement, ok := updated.(CTENode)
		if !ok || replacement.CTECore() == nil {
			return clause, false
		}
		clause.CTEs[i] = replacement
	}
	return v.Leave(clause)
}

var (
	_ Node           = (*CommonTableExpression)(nil)
	_ CTENode        = (*CommonTableExpression)(nil)
	_ Node           = (*WithClause)(nil)
	_ WithClauseNode = (*WithClause)(nil)
)
