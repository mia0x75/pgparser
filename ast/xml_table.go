package ast

import (
	"fmt"

	"github.com/mia0x75/pgparser/format"
	"github.com/mia0x75/pgparser/types"
)

// XMLTableExpr is a table function of an XMLTABLE call.
type XMLTableExpr struct {
	resultSetNode

	// RowExpr is the row expression of the XMLTABLE call.
	RowExpr ExprNode
	// DocExpr is the XML document expression of the XMLTABLE call.
	DocExpr ExprNode
	// Columns are the column definitions of the table function.
	Columns []*XMLTableColumn
	// Namespaces are the XML namespace definitions of the table function.
	Namespaces []*XMLTableNamespace
}

// Restore implements Node interface.
func (n *XMLTableExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restoreNode(ctx, n)
	}

	ctx.WriteKeyword("XMLTABLE(")
	if len(n.Namespaces) > 0 {
		ctx.WriteKeyword("XMLNAMESPACES(")
		for i, namespace := range n.Namespaces {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if err := namespace.Restore(ctx); err != nil {
				return err
			}
		}
		ctx.WritePlain("), ")
	}
	if n.RowExpr != nil {
		if err := n.RowExpr.Restore(ctx); err != nil {
			return err
		}
	}
	ctx.WriteKeyword(" PASSING ")
	if n.DocExpr == nil {
		return fmt.Errorf("XMLTABLE has no document expression")
	}
	if err := n.DocExpr.Restore(ctx); err != nil {
		return err
	}
	ctx.WriteKeyword(" COLUMNS ")
	for i, col := range n.Columns {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if err := col.Restore(ctx); err != nil {
			return err
		}
	}
	ctx.WritePlain(")")
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *XMLTableExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	node, skip := v.Enter(n)
	outer, ok := node.(*XMLTableExpr)
	if !ok || outer == nil {
		return node, false
	}
	n = outer
	if skip {
		return v.Leave(n)
	}
	if n.RowExpr != nil {
		if !nodePresent(n.RowExpr) {
			return n, false
		}
		updated, ok := n.RowExpr.Accept(v)
		if !ok || !nodePresent(updated) {
			return n, false
		}
		replacement, ok := updated.(ExprNode)
		if !ok {
			return n, false
		}
		n.RowExpr = replacement
	}
	if n.DocExpr != nil {
		if !nodePresent(n.DocExpr) {
			return n, false
		}
		updated, ok := n.DocExpr.Accept(v)
		if !ok || !nodePresent(updated) {
			return n, false
		}
		replacement, ok := updated.(ExprNode)
		if !ok {
			return n, false
		}
		n.DocExpr = replacement
	}
	for i0 := range n.Columns {
		if !nodePresent(n.Columns[i0]) {
			return n, false
		}
		updated, ok := n.Columns[i0].Accept(v)
		if !ok || !nodePresent(updated) {
			return n, false
		}
		replacement, ok := updated.(*XMLTableColumn)
		if !ok {
			return n, false
		}
		n.Columns[i0] = replacement
	}
	for i0 := range n.Namespaces {
		if !nodePresent(n.Namespaces[i0]) {
			return n, false
		}
		updated, ok := n.Namespaces[i0].Accept(v)
		if !ok || !nodePresent(updated) {
			return n, false
		}
		replacement, ok := updated.(*XMLTableNamespace)
		if !ok {
			return n, false
		}
		n.Namespaces[i0] = replacement
	}
	return v.Leave(n)
}

// XMLTableColumn is a column definition of an XMLTABLE table function.
type XMLTableColumn struct {
	node

	// Name is the column name.
	Name string
	// TypeName is the column type.
	TypeName *types.FieldType
	// ForOrdinality is true for a FOR ORDINALITY column.
	ForOrdinality bool
	// DefaultExpr is the default expression of the column.
	DefaultExpr ExprNode
	// PathExpr is the path expression of the column.
	PathExpr ExprNode
	// IsNotNull is true for a NOT NULL column.
	IsNotNull bool
}

// Restore implements Node interface.
func (n *XMLTableColumn) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restoreNode(ctx, n)
	}

	ctx.WriteName(n.Name)
	if n.TypeName != nil {
		ctx.WritePlain(" ")
		if err := n.TypeName.Restore(ctx); err != nil {
			return err
		}
	}
	if n.ForOrdinality {
		ctx.WriteKeyword(" FOR ORDINALITY")
	}
	if n.PathExpr != nil {
		ctx.WriteKeyword(" PATH ")
		if err := n.PathExpr.Restore(ctx); err != nil {
			return err
		}
	}
	if n.DefaultExpr != nil {
		ctx.WriteKeyword(" DEFAULT ")
		if err := n.DefaultExpr.Restore(ctx); err != nil {
			return err
		}
	}
	if n.IsNotNull {
		ctx.WriteKeyword(" NOT NULL")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *XMLTableColumn) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	node, skip := v.Enter(n)
	outer, ok := node.(*XMLTableColumn)
	if !ok || outer == nil {
		return node, false
	}
	n = outer
	if skip {
		return v.Leave(n)
	}
	if n.DefaultExpr != nil {
		if !nodePresent(n.DefaultExpr) {
			return n, false
		}
		updated, ok := n.DefaultExpr.Accept(v)
		if !ok || !nodePresent(updated) {
			return n, false
		}
		replacement, ok := updated.(ExprNode)
		if !ok {
			return n, false
		}
		n.DefaultExpr = replacement
	}
	if n.PathExpr != nil {
		if !nodePresent(n.PathExpr) {
			return n, false
		}
		updated, ok := n.PathExpr.Accept(v)
		if !ok || !nodePresent(updated) {
			return n, false
		}
		replacement, ok := updated.(ExprNode)
		if !ok {
			return n, false
		}
		n.PathExpr = replacement
	}
	return v.Leave(n)
}

// XMLTableNamespace is a namespace definition of an XMLTABLE table function.
type XMLTableNamespace struct {
	node

	// Expr is the namespace expression.
	Expr ExprNode
	// Name is the namespace name.
	Name string
}

// Restore implements Node interface.
func (n *XMLTableNamespace) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restoreNode(ctx, n)
	}

	if n.Name == "" {
		ctx.WriteKeyword("DEFAULT ")
	}
	if n.Expr != nil {
		if err := n.Expr.Restore(ctx); err != nil {
			return err
		}
	}
	if n.Name != "" {
		ctx.WriteKeyword(" AS ")
		ctx.WriteName(n.Name)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *XMLTableNamespace) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	node, skip := v.Enter(n)
	outer, ok := node.(*XMLTableNamespace)
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
