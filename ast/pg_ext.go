package ast

import (
	"fmt"
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
	node, skip := v.Enter(n)
	if skip {
		return v.Leave(node)
	}
	n = node.(*CollateExpr)
	if n.Expr != nil {
		updated, ok := n.Expr.Accept(v)
		if !ok {
			return n, false
		}
		n.Expr = updated.(ExprNode)
	}
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
		if err := restoreExprList(ctx, n.Sets[0]); err != nil {
			return err
		}
		ctx.WritePlain(")")
	case GroupingSetCube:
		ctx.WriteKeyWord("CUBE (")
		if err := restoreExprList(ctx, n.Sets[0]); err != nil {
			return err
		}
		ctx.WritePlain(")")
	case GroupingSetSets:
		ctx.WriteKeyWord("GROUPING SETS (")
		for i, set := range n.Sets {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			ctx.WritePlain("(")
			if err := restoreExprList(ctx, set); err != nil {
				return err
			}
			ctx.WritePlain(")")
		}
		ctx.WritePlain(")")
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *GroupingSet) Accept(v Visitor) (Node, bool) {
	node, skip := v.Enter(n)
	if skip {
		return v.Leave(node)
	}
	n = node.(*GroupingSet)
	for i0 := range n.Sets {
		for i1 := range n.Sets[i0] {
			if n.Sets[i0][i1] != nil {
				updated, ok := n.Sets[i0][i1].Accept(v)
				if !ok {
					return n, false
				}
				n.Sets[i0][i1] = updated.(ExprNode)
			}
		}
	}
	return v.Leave(n)
}

func restoreExprList(ctx *format.RestoreCtx, exprs []ExprNode) error {
	for i, e := range exprs {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if err := e.Restore(ctx); err != nil {
			return err
		}
	}
	return nil
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
	if len(n.Namespaces) > 0 {
		ctx.WriteKeyWord("XMLNAMESPACES(")
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
	ctx.WriteKeyWord(" PASSING ")
	if n.DocExpr == nil {
		return fmt.Errorf("XMLTABLE has no document expression")
	}
	if err := n.DocExpr.Restore(ctx); err != nil {
		return err
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
	node, skip := v.Enter(n)
	if skip {
		return v.Leave(node)
	}
	n = node.(*XMLTableExpr)
	if n.RowExpr != nil {
		updated, ok := n.RowExpr.Accept(v)
		if !ok {
			return n, false
		}
		n.RowExpr = updated.(ExprNode)
	}
	if n.DocExpr != nil {
		updated, ok := n.DocExpr.Accept(v)
		if !ok {
			return n, false
		}
		n.DocExpr = updated.(ExprNode)
	}
	for i0 := range n.Columns {
		if n.Columns[i0] != nil {
			updated, ok := n.Columns[i0].Accept(v)
			if !ok {
				return n, false
			}
			n.Columns[i0] = updated.(*XMLTableColumn)
		}
	}
	for i0 := range n.Namespaces {
		if n.Namespaces[i0] != nil {
			updated, ok := n.Namespaces[i0].Accept(v)
			if !ok {
				return n, false
			}
			n.Namespaces[i0] = updated.(*XMLTableNamespace)
		}
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
	if n.PathExpr != nil {
		ctx.WriteKeyWord(" PATH ")
		if err := n.PathExpr.Restore(ctx); err != nil {
			return err
		}
	}
	if n.DefaultExpr != nil {
		ctx.WriteKeyWord(" DEFAULT ")
		if err := n.DefaultExpr.Restore(ctx); err != nil {
			return err
		}
	}
	if n.IsNotNull {
		ctx.WriteKeyWord(" NOT NULL")
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *XMLTableColumn) Accept(v Visitor) (Node, bool) {
	node, skip := v.Enter(n)
	if skip {
		return v.Leave(node)
	}
	n = node.(*XMLTableColumn)
	if n.DefaultExpr != nil {
		updated, ok := n.DefaultExpr.Accept(v)
		if !ok {
			return n, false
		}
		n.DefaultExpr = updated.(ExprNode)
	}
	if n.PathExpr != nil {
		updated, ok := n.PathExpr.Accept(v)
		if !ok {
			return n, false
		}
		n.PathExpr = updated.(ExprNode)
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
	if n.Name == "" {
		ctx.WriteKeyWord("DEFAULT ")
	}
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
	node, skip := v.Enter(n)
	if skip {
		return v.Leave(node)
	}
	n = node.(*XMLTableNamespace)
	if n.Expr != nil {
		updated, ok := n.Expr.Accept(v)
		if !ok {
			return n, false
		}
		n.Expr = updated.(ExprNode)
	}
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
	Passing []Node
	// Columns are the column definitions of the table function.
	Columns []*JSONTableColumn
	// OnError is the error behavior of the JSON_TABLE call.
	OnError *JsonBehavior
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
	if n.PathSpec == nil {
		return fmt.Errorf("JSON_TABLE has no path")
	}
	if err := n.PathSpec.Restore(ctx); err != nil {
		return err
	}
	if len(n.Passing) > 0 {
		return fmt.Errorf("%w: JSON_TABLE PASSING", ErrUnsupportedRestore)
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
	if n.OnError != nil {
		ctx.WritePlain(" ")
		if err := n.OnError.Restore(ctx); err != nil {
			return err
		}
		ctx.WriteKeyWord(" ON ERROR")
	}
	ctx.WritePlain(")")
	return nil
}

// Accept implements Node Accept interface.
func (n *JSONTableExpr) Accept(v Visitor) (Node, bool) {
	node, skip := v.Enter(n)
	if skip {
		return v.Leave(node)
	}
	n = node.(*JSONTableExpr)
	if n.ContextItem != nil {
		updated, ok := n.ContextItem.Accept(v)
		if !ok {
			return n, false
		}
		n.ContextItem = updated.(ExprNode)
	}
	if n.PathSpec != nil {
		updated, ok := n.PathSpec.Accept(v)
		if !ok {
			return n, false
		}
		n.PathSpec = updated.(*JSONTablePathSpec)
	}
	for i0 := range n.Passing {
		if n.Passing[i0] != nil {
			updated, ok := n.Passing[i0].Accept(v)
			if !ok {
				return n, false
			}
			n.Passing[i0] = updated.(Node)
		}
	}
	for i0 := range n.Columns {
		if n.Columns[i0] != nil {
			updated, ok := n.Columns[i0].Accept(v)
			if !ok {
				return n, false
			}
			n.Columns[i0] = updated.(*JSONTableColumn)
		}
	}
	if n.OnError != nil {
		updated, ok := n.OnError.Accept(v)
		if !ok {
			return n, false
		}
		n.OnError = updated.(*JsonBehavior)
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
	if n.Expr == nil {
		return fmt.Errorf("JSON_TABLE path has no expression")
	}
	if err := n.Expr.Restore(ctx); err != nil {
		return err
	}
	if n.Name != "" {
		ctx.WriteKeyWord(" AS ")
		ctx.WriteName(n.Name)
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *JSONTablePathSpec) Accept(v Visitor) (Node, bool) {
	node, skip := v.Enter(n)
	if skip {
		return v.Leave(node)
	}
	n = node.(*JSONTablePathSpec)
	if n.Expr != nil {
		updated, ok := n.Expr.Accept(v)
		if !ok {
			return n, false
		}
		n.Expr = updated.(ExprNode)
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
	if n.ColType == "nested" {
		ctx.WriteKeyWord("NESTED PATH ")
		if n.PathSpec == nil {
			return fmt.Errorf("nested JSON_TABLE column has no path")
		}
		if err := n.PathSpec.Restore(ctx); err != nil {
			return err
		}
		ctx.WriteKeyWord(" COLUMNS (")
		for i, column := range n.Columns {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if err := column.Restore(ctx); err != nil {
				return err
			}
		}
		ctx.WritePlain(")")
		return nil
	}
	ctx.WriteName(n.Name)
	if n.ColType == "ordinality" {
		ctx.WriteKeyWord(" FOR ORDINALITY")
		return nil
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
		ctx.WriteKeyWord(" EXISTS")
	}
	if n.PathSpec != nil {
		ctx.WriteKeyWord(" PATH ")
		if err := n.PathSpec.Restore(ctx); err != nil {
			return err
		}
	}
	switch n.Wrapper {
	case "conditional":
		ctx.WriteKeyWord(" WITH CONDITIONAL WRAPPER")
	case "unconditional":
		ctx.WriteKeyWord(" WITH UNCONDITIONAL WRAPPER")
	case "none":
		ctx.WriteKeyWord(" WITHOUT WRAPPER")
	}
	if n.Quotes != "" {
		ctx.WriteKeyWord(" " + n.Quotes + " QUOTES")
	}
	if n.OnEmpty != nil {
		ctx.WritePlain(" ")
		if err := n.OnEmpty.Restore(ctx); err != nil {
			return err
		}
		ctx.WriteKeyWord(" ON EMPTY")
	}
	if n.OnError != nil {
		ctx.WritePlain(" ")
		if err := n.OnError.Restore(ctx); err != nil {
			return err
		}
		ctx.WriteKeyWord(" ON ERROR")
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *JSONTableColumn) Accept(v Visitor) (Node, bool) {
	node, skip := v.Enter(n)
	if skip {
		return v.Leave(node)
	}
	n = node.(*JSONTableColumn)
	if n.Format != nil {
		updated, ok := n.Format.Accept(v)
		if !ok {
			return n, false
		}
		n.Format = updated.(*JsonFormat)
	}
	if n.PathSpec != nil {
		updated, ok := n.PathSpec.Accept(v)
		if !ok {
			return n, false
		}
		n.PathSpec = updated.(*JSONTablePathSpec)
	}
	if n.OnEmpty != nil {
		updated, ok := n.OnEmpty.Accept(v)
		if !ok {
			return n, false
		}
		n.OnEmpty = updated.(*JsonBehavior)
	}
	if n.OnError != nil {
		updated, ok := n.OnError.Accept(v)
		if !ok {
			return n, false
		}
		n.OnError = updated.(*JsonBehavior)
	}
	for i0 := range n.Columns {
		if n.Columns[i0] != nil {
			updated, ok := n.Columns[i0].Accept(v)
			if !ok {
				return n, false
			}
			n.Columns[i0] = updated.(*JSONTableColumn)
		}
	}
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
	node, skip := v.Enter(n)
	if skip {
		return v.Leave(node)
	}
	n = node.(*RowsFromExpr)
	for i0 := range n.Functions {
		if n.Functions[i0] != nil {
			updated, ok := n.Functions[i0].Accept(v)
			if !ok {
				return n, false
			}
			n.Functions[i0] = updated.(*TableFunction)
		}
	}
	return v.Leave(n)
}

// PrivilegeTarget is the target of a GRANT or REVOKE statement.
type PrivilegeTarget struct {
	node

	// ObjectType is the kind of the target objects.
	ObjectType ObjectTypeType
	// Objects are the target objects.
	Objects     []*TableName
	TargetNodes *List
}

// Restore implements Node interface.
func (n *PrivilegeTarget) Restore(ctx *format.RestoreCtx) error {
	if err := writeGrantTargets(ctx, n.Objects, nil, n.TargetNodes); err != nil {
		return err
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *PrivilegeTarget) Accept(v Visitor) (Node, bool) {
	node, skip := v.Enter(n)
	if skip {
		return v.Leave(node)
	}
	n = node.(*PrivilegeTarget)
	for i0 := range n.Objects {
		if n.Objects[i0] != nil {
			updated, ok := n.Objects[i0].Accept(v)
			if !ok {
				return n, false
			}
			n.Objects[i0] = updated.(*TableName)
		}
	}
	if n.TargetNodes != nil {
		updated, ok := n.TargetNodes.Accept(v)
		if !ok {
			return n, false
		}
		n.TargetNodes = updated.(*List)
	}
	return v.Leave(n)
}
