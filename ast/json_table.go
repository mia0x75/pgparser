package ast

import (
	"fmt"

	"github.com/mia0x75/pgparser/format"
	"github.com/mia0x75/pgparser/types"
)

// JSONTableExpr is a table function of a JSON_TABLE call.
type JSONTableExpr struct {
	resultSetNode

	// ContextItem is the context item of the JSON_TABLE call.
	ContextItem ExprNode
	// PathSpec is the path specification of the JSON_TABLE call.
	PathSpec *JSONTablePathSpec
	// Passing holds the passing arguments of the JSON_TABLE call.
	Passing []Node
	// Columns are the column definitions of the table function.
	Columns []*JSONTableColumn
	// OnError is the error behavior of the JSON_TABLE call.
	OnError *JsonBehavior
}

// Restore implements Node interface.
func (n *JSONTableExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restoreNode(ctx, n)
	}

	ctx.WriteKeyword("JSON_TABLE(")
	if n.ContextItem != nil {
		if err := n.ContextItem.Restore(ctx); err != nil {
			return err
		}
	}
	ctx.WritePlain(", ")
	if n.PathSpec == nil {
		return fmt.Errorf("JSON_TABLE has no path")
	}
	if err := n.PathSpec.Restore(ctx); err != nil {
		return err
	}
	if len(n.Passing) > 0 {
		return fmt.Errorf("%w: JSON_TABLE PASSING", ErrUnsupportedRestore)
	}
	ctx.WriteKeyword(" COLUMNS(")
	for i, col := range n.Columns {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if err := col.Restore(ctx); err != nil {
			return err
		}
	}
	ctx.WritePlain(")")
	if n.OnError != nil {
		ctx.WritePlain(" ")
		if err := n.OnError.Restore(ctx); err != nil {
			return err
		}
		ctx.WriteKeyword(" ON ERROR")
	}
	ctx.WritePlain(")")
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *JSONTableExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	node, skip := v.Enter(n)
	outer, ok := node.(*JSONTableExpr)
	if !ok || outer == nil {
		return node, false
	}
	n = outer
	if skip {
		return v.Leave(n)
	}
	if n.ContextItem != nil {
		if !nodePresent(n.ContextItem) {
			return n, false
		}
		updated, ok := n.ContextItem.Accept(v)
		if !ok || !nodePresent(updated) {
			return n, false
		}
		replacement, ok := updated.(ExprNode)
		if !ok {
			return n, false
		}
		n.ContextItem = replacement
	}
	if n.PathSpec != nil {
		if !nodePresent(n.PathSpec) {
			return n, false
		}
		updated, ok := n.PathSpec.Accept(v)
		if !ok || !nodePresent(updated) {
			return n, false
		}
		replacement, ok := updated.(*JSONTablePathSpec)
		if !ok {
			return n, false
		}
		n.PathSpec = replacement
	}
	for i0 := range n.Passing {
		if !nodePresent(n.Passing[i0]) {
			return n, false
		}
		updated, ok := n.Passing[i0].Accept(v)
		if !ok || !nodePresent(updated) {
			return n, false
		}
		n.Passing[i0] = updated
	}
	for i0 := range n.Columns {
		if !nodePresent(n.Columns[i0]) {
			return n, false
		}
		updated, ok := n.Columns[i0].Accept(v)
		if !ok || !nodePresent(updated) {
			return n, false
		}
		replacement, ok := updated.(*JSONTableColumn)
		if !ok {
			return n, false
		}
		n.Columns[i0] = replacement
	}
	if n.OnError != nil {
		if !nodePresent(n.OnError) {
			return n, false
		}
		updated, ok := n.OnError.Accept(v)
		if !ok || !nodePresent(updated) {
			return n, false
		}
		replacement, ok := updated.(*JsonBehavior)
		if !ok {
			return n, false
		}
		n.OnError = replacement
	}
	return v.Leave(n)
}

// JSONTablePathSpec is a path specification of a JSON_TABLE column.
type JSONTablePathSpec struct {
	node

	Expr ExprNode
	// Name is the optional path name.
	Name string
}

// Restore implements Node interface.
func (n *JSONTablePathSpec) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restoreNode(ctx, n)
	}

	if n.Expr == nil {
		return fmt.Errorf("JSON_TABLE path has no expression")
	}
	if err := n.Expr.Restore(ctx); err != nil {
		return err
	}
	if n.Name != "" {
		ctx.WriteKeyword(" AS ")
		ctx.WriteName(n.Name)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *JSONTablePathSpec) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	node, skip := v.Enter(n)
	outer, ok := node.(*JSONTablePathSpec)
	if !ok || outer == nil {
		return node, false
	}
	n = outer
	if skip {
		return v.Leave(n)
	}
	if n.Expr != nil {
		if !nodePresent(n.Expr) {
			return n, false
		}
		updated, ok := n.Expr.Accept(v)
		if !ok || !nodePresent(updated) {
			return n, false
		}
		replacement, ok := updated.(ExprNode)
		if !ok {
			return n, false
		}
		n.Expr = replacement
	}
	return v.Leave(n)
}

// JSONTableColumn is a column definition of a JSON_TABLE table function.
type JSONTableColumn struct {
	node

	// ColType is the kind of the column.
	ColType string
	// Name is the column name.
	Name string
	// TypeName is the column type.
	TypeName *types.FieldType
	Format   *JsonFormat
	// PathSpec is the path specification of the column.
	PathSpec *JSONTablePathSpec
	// Wrapper is the wrapper behavior of the column.
	Wrapper string
	// Quotes is the quotes behavior of the column.
	Quotes string
	// OnEmpty is the empty behavior of the column.
	OnEmpty *JsonBehavior
	// OnError is the error behavior of the column.
	OnError *JsonBehavior
	// Columns are the nested column definitions.
	Columns []*JSONTableColumn
}

// Restore implements Node interface.
func (n *JSONTableColumn) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restoreNode(ctx, n)
	}

	if n.ColType == "nested" {
		ctx.WriteKeyword("NESTED PATH ")
		if n.PathSpec == nil {
			return fmt.Errorf("nested JSON_TABLE column has no path")
		}
		if err := n.PathSpec.Restore(ctx); err != nil {
			return err
		}
		ctx.WriteKeyword(" COLUMNS (")
		for i, column := range n.Columns {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if err := column.Restore(ctx); err != nil {
				return err
			}
		}
		ctx.WritePlain(")")
		return ctx.Err()
	}
	ctx.WriteName(n.Name)
	if n.ColType == "ordinality" {
		ctx.WriteKeyword(" FOR ORDINALITY")
		return ctx.Err()
	}
	if n.ColType == "formatted" || n.Format != nil {
		return fmt.Errorf("%w: JSON_TABLE FORMAT", ErrUnsupportedRestore)
	}
	if n.TypeName != nil {
		ctx.WritePlain(" ")
		if err := n.TypeName.Restore(ctx); err != nil {
			return err
		}
	}
	if n.ColType == "exists" {
		ctx.WriteKeyword(" EXISTS")
	}
	if n.PathSpec != nil {
		ctx.WriteKeyword(" PATH ")
		if err := n.PathSpec.Restore(ctx); err != nil {
			return err
		}
	}
	switch n.Wrapper {
	case "conditional":
		ctx.WriteKeyword(" WITH CONDITIONAL WRAPPER")
	case "unconditional":
		ctx.WriteKeyword(" WITH UNCONDITIONAL WRAPPER")
	case "none":
		ctx.WriteKeyword(" WITHOUT WRAPPER")
	}
	if n.Quotes != "" {
		ctx.WriteKeyword(" " + n.Quotes + " QUOTES")
	}
	if n.OnEmpty != nil {
		ctx.WritePlain(" ")
		if err := n.OnEmpty.Restore(ctx); err != nil {
			return err
		}
		ctx.WriteKeyword(" ON EMPTY")
	}
	if n.OnError != nil {
		ctx.WritePlain(" ")
		if err := n.OnError.Restore(ctx); err != nil {
			return err
		}
		ctx.WriteKeyword(" ON ERROR")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *JSONTableColumn) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	node, skip := v.Enter(n)
	outer, ok := node.(*JSONTableColumn)
	if !ok || outer == nil {
		return node, false
	}
	n = outer
	if skip {
		return v.Leave(n)
	}
	if n.Format != nil {
		if !nodePresent(n.Format) {
			return n, false
		}
		updated, ok := n.Format.Accept(v)
		if !ok || !nodePresent(updated) {
			return n, false
		}
		replacement, ok := updated.(*JsonFormat)
		if !ok {
			return n, false
		}
		n.Format = replacement
	}
	if n.PathSpec != nil {
		if !nodePresent(n.PathSpec) {
			return n, false
		}
		updated, ok := n.PathSpec.Accept(v)
		if !ok || !nodePresent(updated) {
			return n, false
		}
		replacement, ok := updated.(*JSONTablePathSpec)
		if !ok {
			return n, false
		}
		n.PathSpec = replacement
	}
	if n.OnEmpty != nil {
		if !nodePresent(n.OnEmpty) {
			return n, false
		}
		updated, ok := n.OnEmpty.Accept(v)
		if !ok || !nodePresent(updated) {
			return n, false
		}
		replacement, ok := updated.(*JsonBehavior)
		if !ok {
			return n, false
		}
		n.OnEmpty = replacement
	}
	if n.OnError != nil {
		if !nodePresent(n.OnError) {
			return n, false
		}
		updated, ok := n.OnError.Accept(v)
		if !ok || !nodePresent(updated) {
			return n, false
		}
		replacement, ok := updated.(*JsonBehavior)
		if !ok {
			return n, false
		}
		n.OnError = replacement
	}
	for i0 := range n.Columns {
		if !nodePresent(n.Columns[i0]) {
			return n, false
		}
		updated, ok := n.Columns[i0].Accept(v)
		if !ok || !nodePresent(updated) {
			return n, false
		}
		replacement, ok := updated.(*JSONTableColumn)
		if !ok {
			return n, false
		}
		n.Columns[i0] = replacement
	}
	return v.Leave(n)
}
