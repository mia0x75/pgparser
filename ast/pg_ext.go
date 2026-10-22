package ast

import (
	"io"

	"github.com/mia0x75/pgparser/format"
	"github.com/mia0x75/pgparser/types"
)

// CollateExpr is an expression with an explicit collation.
type CollateExpr struct {
	exprNode

	Expr      ExprNode
	Collation string
}

// Restore implements Node interface.
func (n *CollateExpr) Restore(ctx *format.RestoreCtx) error {
	if err := n.Expr.Restore(ctx); err != nil {
		return err
	}
	ctx.WriteKeyWord(" COLLATE ")
	ctx.WriteName(n.Collation)
	return nil
}

// Format the ExprNode into a Writer.
func (n *CollateExpr) Format(w io.Writer) {
	n.Expr.Format(w)
}

// Accept implements Node Accept interface.
func (n *CollateExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*CollateExpr)
	node, ok := n.Expr.Accept(v)
	if !ok {
		return n, false
	}
	n.Expr = node.(ExprNode)
	return v.Leave(n)
}

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
	switch n.Type {
	case GroupingSetEmpty:
		ctx.WritePlain("()")
	case GroupingSetRollup:
		ctx.WriteKeyWord("ROLLUP (")
		restoreExprList(ctx, n.Sets[0])
		ctx.WritePlain(")")
	case GroupingSetCube:
		ctx.WriteKeyWord("CUBE (")
		restoreExprList(ctx, n.Sets[0])
		ctx.WritePlain(")")
	case GroupingSetSets:
		ctx.WriteKeyWord("GROUPING SETS (")
		for i, set := range n.Sets {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			ctx.WritePlain("(")
			restoreExprList(ctx, set)
			ctx.WritePlain(")")
		}
		ctx.WritePlain(")")
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *GroupingSet) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*GroupingSet)
	for i, set := range n.Sets {
		for j, e := range set {
			node, ok := e.Accept(v)
			if !ok {
				return n, false
			}
			n.Sets[i][j] = node.(ExprNode)
		}
	}
	return v.Leave(n)
}

func restoreExprList(ctx *format.RestoreCtx, exprs []ExprNode) {
	for i, e := range exprs {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		_ = e.Restore(ctx)
	}
}

// XMLTableExpr is a table function of an XMLTABLE call.
type XMLTableExpr struct {
	node

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
	ctx.WriteKeyWord("XMLTABLE(")
	if n.RowExpr != nil {
		if err := n.RowExpr.Restore(ctx); err != nil {
			return err
		}
	}
	ctx.WriteKeyWord(" COLUMNS ")
	for i, col := range n.Columns {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if err := col.Restore(ctx); err != nil {
			return err
		}
	}
	ctx.WritePlain(")")
	return nil
}

// Accept implements Node Accept interface.
func (n *XMLTableExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*XMLTableExpr)
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
	ctx.WriteName(n.Name)
	if n.TypeName != nil {
		ctx.WritePlain(" ")
		if err := n.TypeName.Restore(ctx); err != nil {
			return err
		}
	}
	if n.ForOrdinality {
		ctx.WriteKeyWord(" FOR ORDINALITY")
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *XMLTableColumn) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*XMLTableColumn)
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
	if n.Expr != nil {
		if err := n.Expr.Restore(ctx); err != nil {
			return err
		}
	}
	if n.Name != "" {
		ctx.WriteKeyWord(" AS ")
		ctx.WriteName(n.Name)
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *XMLTableNamespace) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*XMLTableNamespace)
	return v.Leave(n)
}

// JSONTableExpr is a table function of a JSON_TABLE call.
type JSONTableExpr struct {
	node

	// ContextItem is the context item of the JSON_TABLE call.
	ContextItem ExprNode
	// PathSpec is the path specification of the JSON_TABLE call.
	PathSpec *JSONTablePathSpec
	// Passing holds the passing arguments of the JSON_TABLE call.
	Passing []ExprNode
	// Columns are the column definitions of the table function.
	Columns []*JSONTableColumn
	// OnError is the error behavior of the JSON_TABLE call.
	OnError string
}

// Restore implements Node interface.
func (n *JSONTableExpr) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("JSON_TABLE(")
	if n.ContextItem != nil {
		if err := n.ContextItem.Restore(ctx); err != nil {
			return err
		}
	}
	ctx.WritePlain(", ")
	if n.PathSpec != nil {
		ctx.WriteString(n.PathSpec.String)
	}
	ctx.WriteKeyWord(" COLUMNS(")
	for i, col := range n.Columns {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if err := col.Restore(ctx); err != nil {
			return err
		}
	}
	ctx.WritePlain(")")
	return nil
}

// Accept implements Node Accept interface.
func (n *JSONTableExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*JSONTableExpr)
	return v.Leave(n)
}

// JSONTablePathSpec is a path specification of a JSON_TABLE column.
type JSONTablePathSpec struct {
	node

	// String is the path expression.
	String string
	// Name is the optional path name.
	Name string
}

// Restore implements Node interface.
func (n *JSONTablePathSpec) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteString(n.String)
	if n.Name != "" {
		ctx.WriteKeyWord(" AS ")
		ctx.WriteName(n.Name)
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *JSONTablePathSpec) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*JSONTablePathSpec)
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
	// PathSpec is the path specification of the column.
	PathSpec *JSONTablePathSpec
	// Wrapper is the wrapper behavior of the column.
	Wrapper string
	// Quotes is the quotes behavior of the column.
	Quotes string
	// OnEmpty is the empty behavior of the column.
	OnEmpty string
	// OnError is the error behavior of the column.
	OnError string
	// Columns are the nested column definitions.
	Columns []*JSONTableColumn
}

// Restore implements Node interface.
func (n *JSONTableColumn) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteName(n.Name)
	if n.TypeName != nil {
		ctx.WritePlain(" ")
		if err := n.TypeName.Restore(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *JSONTableColumn) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*JSONTableColumn)
	return v.Leave(n)
}

// RowsFromExpr is a table function of a ROWS FROM call.
type RowsFromExpr struct {
	node

	// Functions are the functions of the ROWS FROM call.
	Functions []*TableFunction
	// WithOrdinality is true for a WITH ORDINALITY call.
	WithOrdinality bool
}

// Restore implements Node interface.
func (n *RowsFromExpr) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("ROWS FROM(")
	for i, fn := range n.Functions {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if err := fn.Restore(ctx); err != nil {
			return err
		}
	}
	ctx.WritePlain(")")
	if n.WithOrdinality {
		ctx.WriteKeyWord(" WITH ORDINALITY")
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *RowsFromExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*RowsFromExpr)
	return v.Leave(n)
}

// PrivilegeTarget is the target of a GRANT or REVOKE statement.
type PrivilegeTarget struct {
	node

	// ObjectType is the kind of the target objects.
	ObjectType ObjectTypeType
	// Objects are the target objects.
	Objects []*TableName
}

// Restore implements Node interface.
func (n *PrivilegeTarget) Restore(ctx *format.RestoreCtx) error {
	if err := writeGrantTargets(ctx, n.Objects, nil); err != nil {
		return err
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *PrivilegeTarget) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*PrivilegeTarget)
	return v.Leave(n)
}
