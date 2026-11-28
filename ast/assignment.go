package ast

import (
	"fmt"
	"io"

	pkast "github.com/mia0x75/parserkit/ast"

	"github.com/mia0x75/pgparser/format"
)

// Assignment keeps the shared write target and value.
type Assignment = pkast.Assignment

// assignmentValid reports whether an assignment can be restored.
func assignmentValid(a *Assignment) bool {
	if a == nil || a.Expr == nil || a.Target == nil {
		return false
	}
	return nodePresent(a.Target)
}

// RowTarget is the parenthesized column list of a row assignment,
// for example SET (a, b) = ROW(1, 2).
type RowTarget struct {
	exprNode

	// Columns are the assigned column names.
	Columns []*ColumnName
}

// Restore writes the parenthesized column list.
func (n *RowTarget) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return fmt.Errorf("restore row target: nil target")
	}
	ctx.WritePlain("(")
	for i, col := range n.Columns {
		if col == nil {
			return fmt.Errorf("restore row target: nil column %d", i)
		}
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if err := col.Restore(ctx); err != nil {
			return fmt.Errorf("restore row target column %d: %w", i, err)
		}
	}
	ctx.WritePlain(")")
	return ctx.Err()
}

// Format writes the parenthesized column list.
func (n *RowTarget) Format(w io.Writer) {
	if n == nil {
		return
	}
	_, _ = fmt.Fprint(w, "(")
	for i, col := range n.Columns {
		if i > 0 {
			_, _ = fmt.Fprint(w, ", ")
		}
		if col != nil {
			col.Format(w)
		}
	}
	_, _ = fmt.Fprint(w, ")")
}

// Accept walks every assigned column.
func (n *RowTarget) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	outer, ok := entered.(*RowTarget)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	for i, col := range outer.Columns {
		if col == nil {
			return outer, false
		}
		updated, ok := col.Accept(v)
		if !ok {
			return outer, false
		}
		column, ok := updated.(*ColumnName)
		if !ok || column == nil {
			return outer, false
		}
		outer.Columns[i] = column
	}
	return v.Leave(outer)
}

var _ ExprNode = (*RowTarget)(nil)
