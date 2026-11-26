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
	ctx.WriteKeyword(" COLLATE ")
	ctx.WriteName(n.Collation)
	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *CollateExpr) Format(w io.Writer) {
	FormatExpr(n.Expr, w)
}

// Accept implements Node Accept interface.
func (n *CollateExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	node, skip := v.Enter(n)
	if !nodePresent(node) {
		return n, false
	}
	if skip {
		return v.Leave(node)
	}
	outer, ok := node.(*CollateExpr)
	if !ok || outer == nil {
		return node, false
	}
	n = outer
	if n.Expr != nil {
		if !nodePresent(n.Expr) {
			return n, false
		}
		updated, ok := n.Expr.Accept(v)
		if !ok {
			return n, false
		}
		replacement, ok := updated.(ExprNode)
		if !ok || !nodePresent(replacement) {
			return n, false
		}
		n.Expr = replacement
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
		ctx.WriteKeyword("ROLLUP (")
		if err := restoreExprList(ctx, n.Sets[0]); err != nil {
			return err
		}
		ctx.WritePlain(")")
	case GroupingSetCube:
		ctx.WriteKeyword("CUBE (")
		if err := restoreExprList(ctx, n.Sets[0]); err != nil {
			return err
		}
		ctx.WritePlain(")")
	case GroupingSetSets:
		ctx.WriteKeyword("GROUPING SETS (")
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
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *GroupingSet) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	node, skip := v.Enter(n)
	if !nodePresent(node) {
		return n, false
	}
	if skip {
		return v.Leave(node)
	}
	outer, ok := node.(*GroupingSet)
	if !ok || outer == nil {
		return node, false
	}
	n = outer
	for i0 := range n.Sets {
		for i1 := range n.Sets[i0] {
			if n.Sets[i0][i1] != nil {
				if !nodePresent(n.Sets[i0][i1]) {
					return n, false
				}
				updated, ok := n.Sets[i0][i1].Accept(v)
				if !ok {
					return n, false
				}
				replacement, ok := updated.(ExprNode)
				if !ok || !nodePresent(replacement) {
					return n, false
				}
				n.Sets[i0][i1] = replacement
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
	return ctx.Err()
}

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
	if !nodePresent(node) {
		return n, false
	}
	if skip {
		return v.Leave(node)
	}
	outer, ok := node.(*XMLTableExpr)
	if !ok || outer == nil {
		return node, false
	}
	n = outer
	if n.RowExpr != nil {
		if !nodePresent(n.RowExpr) {
			return n, false
		}
		updated, ok := n.RowExpr.Accept(v)
		if !ok {
			return n, false
		}
		replacement, ok := updated.(ExprNode)
		if !ok || !nodePresent(replacement) {
			return n, false
		}
		n.RowExpr = replacement
	}
	if n.DocExpr != nil {
		if !nodePresent(n.DocExpr) {
			return n, false
		}
		updated, ok := n.DocExpr.Accept(v)
		if !ok {
			return n, false
		}
		replacement, ok := updated.(ExprNode)
		if !ok || !nodePresent(replacement) {
			return n, false
		}
		n.DocExpr = replacement
	}
	for i0 := range n.Columns {
		if n.Columns[i0] != nil {
			if !nodePresent(n.Columns[i0]) {
				return n, false
			}
			updated, ok := n.Columns[i0].Accept(v)
			if !ok {
				return n, false
			}
			replacement, ok := updated.(*XMLTableColumn)
			if !ok || !nodePresent(replacement) {
				return n, false
			}
			n.Columns[i0] = replacement
		}
	}
	for i0 := range n.Namespaces {
		if n.Namespaces[i0] != nil {
			if !nodePresent(n.Namespaces[i0]) {
				return n, false
			}
			updated, ok := n.Namespaces[i0].Accept(v)
			if !ok {
				return n, false
			}
			replacement, ok := updated.(*XMLTableNamespace)
			if !ok || !nodePresent(replacement) {
				return n, false
			}
			n.Namespaces[i0] = replacement
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
	if !nodePresent(node) {
		return n, false
	}
	if skip {
		return v.Leave(node)
	}
	outer, ok := node.(*XMLTableColumn)
	if !ok || outer == nil {
		return node, false
	}
	n = outer
	if n.DefaultExpr != nil {
		if !nodePresent(n.DefaultExpr) {
			return n, false
		}
		updated, ok := n.DefaultExpr.Accept(v)
		if !ok {
			return n, false
		}
		replacement, ok := updated.(ExprNode)
		if !ok || !nodePresent(replacement) {
			return n, false
		}
		n.DefaultExpr = replacement
	}
	if n.PathExpr != nil {
		if !nodePresent(n.PathExpr) {
			return n, false
		}
		updated, ok := n.PathExpr.Accept(v)
		if !ok {
			return n, false
		}
		replacement, ok := updated.(ExprNode)
		if !ok || !nodePresent(replacement) {
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
	if !nodePresent(node) {
		return n, false
	}
	if skip {
		return v.Leave(node)
	}
	outer, ok := node.(*XMLTableNamespace)
	if !ok || outer == nil {
		return node, false
	}
	n = outer
	if n.Expr != nil {
		if !nodePresent(n.Expr) {
			return n, false
		}
		updated, ok := n.Expr.Accept(v)
		if !ok {
			return n, false
		}
		replacement, ok := updated.(ExprNode)
		if !ok || !nodePresent(replacement) {
			return n, false
		}
		n.Expr = replacement
	}
	return v.Leave(n)
}

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
	if !nodePresent(node) {
		return n, false
	}
	if skip {
		return v.Leave(node)
	}
	outer, ok := node.(*JSONTableExpr)
	if !ok || outer == nil {
		return node, false
	}
	n = outer
	if n.ContextItem != nil {
		if !nodePresent(n.ContextItem) {
			return n, false
		}
		updated, ok := n.ContextItem.Accept(v)
		if !ok {
			return n, false
		}
		replacement, ok := updated.(ExprNode)
		if !ok || !nodePresent(replacement) {
			return n, false
		}
		n.ContextItem = replacement
	}
	if n.PathSpec != nil {
		if !nodePresent(n.PathSpec) {
			return n, false
		}
		updated, ok := n.PathSpec.Accept(v)
		if !ok {
			return n, false
		}
		replacement, ok := updated.(*JSONTablePathSpec)
		if !ok || !nodePresent(replacement) {
			return n, false
		}
		n.PathSpec = replacement
	}
	for i0 := range n.Passing {
		if n.Passing[i0] != nil {
			if !nodePresent(n.Passing[i0]) {
				return n, false
			}
			updated, ok := n.Passing[i0].Accept(v)
			if !ok || !nodePresent(updated) {
				return n, false
			}
			n.Passing[i0] = updated
		}
	}
	for i0 := range n.Columns {
		if n.Columns[i0] != nil {
			if !nodePresent(n.Columns[i0]) {
				return n, false
			}
			updated, ok := n.Columns[i0].Accept(v)
			if !ok {
				return n, false
			}
			replacement, ok := updated.(*JSONTableColumn)
			if !ok || !nodePresent(replacement) {
				return n, false
			}
			n.Columns[i0] = replacement
		}
	}
	if n.OnError != nil {
		if !nodePresent(n.OnError) {
			return n, false
		}
		updated, ok := n.OnError.Accept(v)
		if !ok {
			return n, false
		}
		replacement, ok := updated.(*JsonBehavior)
		if !ok || !nodePresent(replacement) {
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
	if !nodePresent(node) {
		return n, false
	}
	if skip {
		return v.Leave(node)
	}
	outer, ok := node.(*JSONTablePathSpec)
	if !ok || outer == nil {
		return node, false
	}
	n = outer
	if n.Expr != nil {
		if !nodePresent(n.Expr) {
			return n, false
		}
		updated, ok := n.Expr.Accept(v)
		if !ok {
			return n, false
		}
		replacement, ok := updated.(ExprNode)
		if !ok || !nodePresent(replacement) {
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
	if !nodePresent(node) {
		return n, false
	}
	if skip {
		return v.Leave(node)
	}
	outer, ok := node.(*JSONTableColumn)
	if !ok || outer == nil {
		return node, false
	}
	n = outer
	if n.Format != nil {
		if !nodePresent(n.Format) {
			return n, false
		}
		updated, ok := n.Format.Accept(v)
		if !ok {
			return n, false
		}
		replacement, ok := updated.(*JsonFormat)
		if !ok || !nodePresent(replacement) {
			return n, false
		}
		n.Format = replacement
	}
	if n.PathSpec != nil {
		if !nodePresent(n.PathSpec) {
			return n, false
		}
		updated, ok := n.PathSpec.Accept(v)
		if !ok {
			return n, false
		}
		replacement, ok := updated.(*JSONTablePathSpec)
		if !ok || !nodePresent(replacement) {
			return n, false
		}
		n.PathSpec = replacement
	}
	if n.OnEmpty != nil {
		if !nodePresent(n.OnEmpty) {
			return n, false
		}
		updated, ok := n.OnEmpty.Accept(v)
		if !ok {
			return n, false
		}
		replacement, ok := updated.(*JsonBehavior)
		if !ok || !nodePresent(replacement) {
			return n, false
		}
		n.OnEmpty = replacement
	}
	if n.OnError != nil {
		if !nodePresent(n.OnError) {
			return n, false
		}
		updated, ok := n.OnError.Accept(v)
		if !ok {
			return n, false
		}
		replacement, ok := updated.(*JsonBehavior)
		if !ok || !nodePresent(replacement) {
			return n, false
		}
		n.OnError = replacement
	}
	for i0 := range n.Columns {
		if n.Columns[i0] != nil {
			if !nodePresent(n.Columns[i0]) {
				return n, false
			}
			updated, ok := n.Columns[i0].Accept(v)
			if !ok {
				return n, false
			}
			replacement, ok := updated.(*JSONTableColumn)
			if !ok || !nodePresent(replacement) {
				return n, false
			}
			n.Columns[i0] = replacement
		}
	}
	return v.Leave(n)
}

// RowsFromExpr is a table function of a ROWS FROM call.
type RowsFromExpr struct {
	resultSetNode

	// Functions are the functions of the ROWS FROM call.
	Functions []*TableFunction
	// WithOrdinality is true for a WITH ORDINALITY call.
	WithOrdinality bool
}

// Restore implements Node interface.
func (n *RowsFromExpr) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("ROWS FROM(")
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
		ctx.WriteKeyword(" WITH ORDINALITY")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *RowsFromExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	node, skip := v.Enter(n)
	if !nodePresent(node) {
		return n, false
	}
	if skip {
		return v.Leave(node)
	}
	outer, ok := node.(*RowsFromExpr)
	if !ok || outer == nil {
		return node, false
	}
	n = outer
	for i0 := range n.Functions {
		if n.Functions[i0] != nil {
			if !nodePresent(n.Functions[i0]) {
				return n, false
			}
			updated, ok := n.Functions[i0].Accept(v)
			if !ok {
				return n, false
			}
			replacement, ok := updated.(*TableFunction)
			if !ok || !nodePresent(replacement) {
				return n, false
			}
			n.Functions[i0] = replacement
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
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *PrivilegeTarget) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	node, skip := v.Enter(n)
	if !nodePresent(node) {
		return n, false
	}
	if skip {
		return v.Leave(node)
	}
	outer, ok := node.(*PrivilegeTarget)
	if !ok || outer == nil {
		return node, false
	}
	n = outer
	for i0 := range n.Objects {
		if n.Objects[i0] != nil {
			if !nodePresent(n.Objects[i0]) {
				return n, false
			}
			updated, ok := n.Objects[i0].Accept(v)
			if !ok {
				return n, false
			}
			replacement, ok := updated.(*TableName)
			if !ok || !nodePresent(replacement) {
				return n, false
			}
			n.Objects[i0] = replacement
		}
	}
	if n.TargetNodes != nil {
		if !nodePresent(n.TargetNodes) {
			return n, false
		}
		updated, ok := n.TargetNodes.Accept(v)
		if !ok {
			return n, false
		}
		replacement, ok := updated.(*List)
		if !ok || !nodePresent(replacement) {
			return n, false
		}
		n.TargetNodes = replacement
	}
	return v.Leave(n)
}
