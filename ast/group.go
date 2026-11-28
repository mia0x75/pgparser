package ast

import (
	"fmt"
	"io"

	pkast "github.com/mia0x75/parserkit/ast"

	"github.com/mia0x75/pgparser/format"
	"github.com/mia0x75/pgparser/model"
)

// GroupItemNode admits dialect extensions without accepting ordering items.
type GroupItemNode = pkast.GroupItemNode

// GroupByClauseNode preserves the concrete dialect clause identity.
type GroupByClauseNode = pkast.GroupByClauseNode

// GroupItem keeps PostgreSQL's grouping sets.
type GroupItem struct {
	pkast.BaseGroupItem

	// GroupingSet is set for PostgreSQL GROUPING SETS / ROLLUP / CUBE items.
	GroupingSet *GroupingSet
}

// GroupItemCore returns the shared grouping data without replacing this outer.
func (n *GroupItem) GroupItemCore() *pkast.BaseGroupItem {
	if n == nil {
		return nil
	}
	return &n.BaseGroupItem
}

// Restore writes the grouped expression or its grouping set.
func (n *GroupItem) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return fmt.Errorf("restore group item: nil item")
	}
	if n.GroupingSet != nil {
		switch n.GroupingSet.Type {
		case GroupingSetEmpty:
			ctx.WritePlain("()")
		case GroupingSetRollup:
			ctx.WriteKeyword("ROLLUP (")
			if err := restoreExprList(ctx, n.GroupingSet.Sets[0]); err != nil {
				return err
			}
			ctx.WritePlain(")")
		case GroupingSetCube:
			ctx.WriteKeyword("CUBE (")
			if err := restoreExprList(ctx, n.GroupingSet.Sets[0]); err != nil {
				return err
			}
			ctx.WritePlain(")")
		case GroupingSetSets:
			ctx.WriteKeyword("GROUPING SETS (")
			if err := restoreExprSets(ctx, n.GroupingSet.Sets); err != nil {
				return err
			}
			ctx.WritePlain(")")
		}
		return ctx.Err()
	}
	if n.Expr == nil {
		return fmt.Errorf("restore group item: missing expression")
	}
	if err := n.Expr.Restore(ctx); err != nil {
		return fmt.Errorf("restore group item expression: %w", err)
	}
	return ctx.Err()
}

// Format writes the grouped expression or its grouping set.
func (n *GroupItem) Format(w io.Writer) {
	if n == nil {
		return
	}
	if n.GroupingSet != nil {
		switch n.GroupingSet.Type {
		case GroupingSetEmpty:
			_, _ = fmt.Fprint(w, "()")
		case GroupingSetRollup:
			_, _ = fmt.Fprint(w, "ROLLUP (")
			formatExprSets(w, n.GroupingSet.Sets, false)
			_, _ = fmt.Fprint(w, ")")
		case GroupingSetCube:
			_, _ = fmt.Fprint(w, "CUBE (")
			formatExprSets(w, n.GroupingSet.Sets, false)
			_, _ = fmt.Fprint(w, ")")
		case GroupingSetSets:
			_, _ = fmt.Fprint(w, "GROUPING SETS (")
			formatExprSets(w, n.GroupingSet.Sets, true)
			_, _ = fmt.Fprint(w, ")")
		}
		return
	}
	if n.Expr != nil {
		FormatExpr(n.Expr, w)
	}
}

// Accept walks the grouped expression.
func (n *GroupItem) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	outer, ok := entered.(*GroupItem)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	if outer.GroupingSet != nil {
		updated, ok := outer.GroupingSet.Accept(v)
		if !ok {
			return outer, false
		}
		grouping, ok := updated.(*GroupingSet)
		if !ok || grouping == nil {
			return outer, false
		}
		outer.GroupingSet = grouping
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

// formatExprSets writes grouping-set expressions; each set is parenthesised when
// withParens is true (GROUPING SETS) and left flat for ROLLUP/CUBE.
func formatExprSets(w io.Writer, sets [][]ExprNode, withParens bool) {
	for i, set := range sets {
		if i > 0 {
			_, _ = fmt.Fprint(w, ", ")
		}
		if withParens {
			_, _ = fmt.Fprint(w, "(")
		}
		for j, e := range set {
			if j > 0 {
				_, _ = fmt.Fprint(w, ", ")
			}
			FormatExpr(e, w)
		}
		if withParens {
			_, _ = fmt.Fprint(w, ")")
		}
	}
}

// GroupByClause keeps PostgreSQL's grouping modifiers and the view check option.
type GroupByClause struct {
	pkast.BaseGroupByClause

	// Distinct requests grouping of distinct values.
	Distinct bool
	// WithRollup indicates whether `WITH ROLLUP` is specified.
	WithRollup bool
	// WithCube indicates whether `WITH CUBE` is specified.
	WithCube bool
	// HasCheckOption indicates whether a view `WITH CASCADED|LOCAL CHECK OPTION`
	// clause directly follows the GROUP BY clause.
	HasCheckOption bool
	// CheckOption is the view check option following the GROUP BY clause.
	// It is only meaningful when HasCheckOption is true.
	CheckOption model.ViewCheckOption
}

// GroupByClauseCore returns the shared grouping data without replacing this outer.
func (n *GroupByClause) GroupByClauseCore() *pkast.BaseGroupByClause {
	if n == nil {
		return nil
	}
	return &n.BaseGroupByClause
}

// Restore writes the GROUP BY clause and its modifiers.
func (n *GroupByClause) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return fmt.Errorf("restore group by clause: nil clause")
	}
	ctx.WriteKeyword("GROUP BY ")
	if n.Distinct {
		ctx.WriteKeyword("DISTINCT ")
	}
	for i, item := range n.Items {
		if i != 0 {
			ctx.WritePlain(",")
		}
		if err := item.Restore(ctx); err != nil {
			return fmt.Errorf("restore group by item %d: %w", i, err)
		}
	}
	if n.WithRollup {
		ctx.WriteKeyword(" WITH ROLLUP")
	}
	if n.WithCube {
		ctx.WriteKeyword(" WITH CUBE")
	}
	if n.HasCheckOption {
		if n.CheckOption == model.CheckOptionLocal {
			ctx.WriteKeyword(" WITH LOCAL CHECK OPTION")
		} else {
			ctx.WriteKeyword(" WITH CASCADED CHECK OPTION")
		}
	}
	return ctx.Err()
}

// Format writes the GROUP BY clause and its modifiers.
func (n *GroupByClause) Format(w io.Writer) {
	if n == nil {
		return
	}
	_, _ = fmt.Fprint(w, "GROUP BY ")
	if n.Distinct {
		_, _ = fmt.Fprint(w, "DISTINCT ")
	}
	for i, item := range n.Items {
		if i != 0 {
			_, _ = fmt.Fprint(w, ", ")
		}
		if group, ok := item.(*GroupItem); ok {
			group.Format(w)
			continue
		}
		if core := item.GroupItemCore(); core != nil && core.Expr != nil {
			FormatExpr(core.Expr, w)
		}
	}
	if n.WithRollup {
		_, _ = fmt.Fprint(w, " WITH ROLLUP")
	}
	if n.WithCube {
		_, _ = fmt.Fprint(w, " WITH CUBE")
	}
	if n.HasCheckOption {
		if n.CheckOption == model.CheckOptionLocal {
			_, _ = fmt.Fprint(w, " WITH LOCAL CHECK OPTION")
		} else {
			_, _ = fmt.Fprint(w, " WITH CASCADED CHECK OPTION")
		}
	}
}

// Accept walks every grouping item.
func (n *GroupByClause) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	clause, ok := entered.(*GroupByClause)
	if !ok || clause == nil {
		return entered, false
	}
	if skip {
		return v.Leave(clause)
	}
	for i, item := range clause.Items {
		if !nodePresent(item) || item.GroupItemCore() == nil {
			return clause, false
		}
		updated, ok := item.Accept(v)
		if !ok {
			return clause, false
		}
		replacement, ok := updated.(GroupItemNode)
		if !ok || replacement.GroupItemCore() == nil {
			return clause, false
		}
		clause.Items[i] = replacement
	}
	return v.Leave(clause)
}

var (
	_ GroupItemNode     = (*GroupItem)(nil)
	_ GroupByClauseNode = (*GroupByClause)(nil)
)
