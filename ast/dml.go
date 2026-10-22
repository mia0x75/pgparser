package ast

import (
	"fmt"
	"io"
	"strings"

	"github.com/mia0x75/pgparser/errors"

	"github.com/mia0x75/pgparser/auth"
	"github.com/mia0x75/pgparser/format"
	"github.com/mia0x75/pgparser/model"
	"github.com/mia0x75/pgparser/postgres"
)

var (
	_ DMLNode = &DeleteStmt{}
	_ DMLNode = &InsertStmt{}
	_ DMLNode = &SetOprStmt{}
	_ DMLNode = &UpdateStmt{}
	_ DMLNode = &SelectStmt{}
	_ DMLNode = &ValuesStmt{}
	_ DMLNode = &TableStmt{}
	_ DMLNode = &ShowStmt{}
	_ DMLNode = &LoadDataStmt{}

	_ Node = &Assignment{}
	_ Node = &ByItem{}
	_ Node = &FieldList{}
	_ Node = &GroupByClause{}
	_ Node = &HavingClause{}
	_ Node = &Join{}
	_ Node = &Limit{}
	_ Node = &OnCondition{}
	_ Node = &OrderByClause{}
	_ Node = &SelectField{}
	_ Node = &TableName{}
	_ Node = &TableRefsClause{}
	_ Node = &TableSource{}
	_ Node = &SetOprSelectList{}
	_ Node = &WildCardField{}
	_ Node = &WindowSpec{}
	_ Node = &PartitionByClause{}
	_ Node = &FrameClause{}
	_ Node = &FrameBound{}
)

// JoinType is join type, including cross/left/right/full.
type JoinType int

const (
	// CrossJoin is cross join type.
	CrossJoin JoinType = iota + 1
	// LeftJoin is left Join type.
	LeftJoin
	// RightJoin is right Join type.
	RightJoin
	// FullJoin is full outer join type.
	FullJoin
	// InnerJoin is inner join type.
	InnerJoin
)

// Join represents table join.
type Join struct {
	node
	resultSetNode

	// Left table can be TableSource or JoinNode.
	Left ResultSetNode
	// Right table can be TableSource or JoinNode or nil.
	Right ResultSetNode
	// Tp represents join type.
	Tp JoinType
	// On represents join on condition.
	On *OnCondition
	// Using represents join using clause.
	Using []*ColumnName
	// NaturalJoin represents join is natural join.
	NaturalJoin bool
	// StraightJoin represents a straight join.
	StraightJoin bool
}

// Restore implements Node interface.
func (n *Join) Restore(ctx *format.RestoreCtx) error {
	if ctx.JoinLevel != 0 {
		ctx.WritePlain("(")
		defer ctx.WritePlain(")")
	}
	ctx.JoinLevel++
	if err := n.Left.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore Join.Left")
	}
	ctx.JoinLevel--
	if n.Right == nil {
		return nil
	}
	if n.NaturalJoin {
		ctx.WriteKeyWord(" NATURAL")
	}
	switch n.Tp {
	case LeftJoin:
		ctx.WriteKeyWord(" LEFT")
	case RightJoin:
		ctx.WriteKeyWord(" RIGHT")
	}
	if n.StraightJoin {
		ctx.WriteKeyWord(" STRAIGHT_JOIN ")
	} else {
		ctx.WriteKeyWord(" JOIN ")
	}
	ctx.JoinLevel++
	if err := n.Right.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore Join.Right")
	}
	ctx.JoinLevel--

	if n.On != nil {
		ctx.WritePlain(" ")
		if err := n.On.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore Join.On")
		}
	}
	if len(n.Using) != 0 {
		ctx.WriteKeyWord(" USING ")
		ctx.WritePlain("(")
		for i, v := range n.Using {
			if i != 0 {
				ctx.WritePlain(",")
			}
			if err := v.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore Join.Using")
			}
		}
		ctx.WritePlain(")")
	}

	return nil
}

// Accept implements Node Accept interface.
func (n *Join) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*Join)
	node, ok := n.Left.Accept(v)
	if !ok {
		return n, false
	}
	n.Left = node.(ResultSetNode)
	if n.Right != nil {
		node, ok = n.Right.Accept(v)
		if !ok {
			return n, false
		}
		n.Right = node.(ResultSetNode)
	}
	if n.On != nil {
		node, ok = n.On.Accept(v)
		if !ok {
			return n, false
		}
		n.On = node.(*OnCondition)
	}
	return v.Leave(n)
}

// TableName represents a table name.
type TableName struct {
	node
	resultSetNode

	Schema model.CIStr
	Name   model.CIStr
	// Relpersistence is the persistence of a temporary table ('t' or 'u').
	Relpersistence byte

	DBInfo    *model.DBInfo
	TableInfo *model.TableInfo

	IndexHints     []*IndexHint
	PartitionNames []model.CIStr
}

// Restore implements Node interface.
func (n *TableName) Restore(ctx *format.RestoreCtx) error {
	if n.Schema.String() != "" {
		ctx.WriteName(n.Schema.String())
		ctx.WritePlain(".")
	}
	ctx.WriteName(n.Name.String())
	if len(n.PartitionNames) > 0 {
		ctx.WriteKeyWord(" PARTITION")
		ctx.WritePlain("(")
		for i, v := range n.PartitionNames {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			ctx.WriteName(v.String())
		}
		ctx.WritePlain(")")
	}
	for _, value := range n.IndexHints {
		ctx.WritePlain(" ")
		if err := value.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing IndexHints")
		}
	}

	return nil
}

// IndexHintType is the type for index hint use, ignore or force.
type IndexHintType int

// IndexHintUseType values.
const (
	HintUse    IndexHintType = 1
	HintIgnore IndexHintType = 2
	HintForce  IndexHintType = 3
)

// IndexHintScope is the type for index hint for join, order by or group by.
type IndexHintScope int

// Index hint scopes.
const (
	HintForScan    IndexHintScope = 1
	HintForJoin    IndexHintScope = 2
	HintForOrderBy IndexHintScope = 3
	HintForGroupBy IndexHintScope = 4
)

// IndexHint represents a hint for optimizer to use/ignore/force for join/order by/group by.
type IndexHint struct {
	IndexNames []model.CIStr
	HintType   IndexHintType
	HintScope  IndexHintScope
}

// Restore IndexHint (The const field uses switch to facilitate understanding)
func (n *IndexHint) Restore(ctx *format.RestoreCtx) error {
	indexHintType := ""
	switch n.HintType {
	case 1:
		indexHintType = "USE INDEX"
	case 2:
		indexHintType = "IGNORE INDEX"
	case 3:
		indexHintType = "FORCE INDEX"
	default: // Prevent accidents
		return errors.New("IndexHintType has an error while matching")
	}

	indexHintScope := ""
	switch n.HintScope {
	case 1:
		indexHintScope = ""
	case 2:
		indexHintScope = " FOR JOIN"
	case 3:
		indexHintScope = " FOR ORDER BY"
	case 4:
		indexHintScope = " FOR GROUP BY"
	default: // Prevent accidents
		return errors.New("IndexHintScope has an error while matching")
	}
	ctx.WriteKeyWord(indexHintType)
	ctx.WriteKeyWord(indexHintScope)
	ctx.WritePlain(" (")
	for i, value := range n.IndexNames {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		ctx.WriteName(value.O)
	}
	ctx.WritePlain(")")

	return nil
}

// Accept implements Node Accept interface.
func (n *TableName) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*TableName)
	return v.Leave(n)
}

// DeleteTableList is the tablelist used in delete statement multi-table mode.
type DeleteTableList struct {
	node
	Tables []*TableName
}

// Restore implements Node interface.
func (n *DeleteTableList) Restore(ctx *format.RestoreCtx) error {
	for i, t := range n.Tables {
		if i != 0 {
			ctx.WritePlain(",")
		}
		if err := t.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore DeleteTableList.Tables[%v]", i)
		}
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *DeleteTableList) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*DeleteTableList)
	if n != nil {
		for i, t := range n.Tables {
			node, ok := t.Accept(v)
			if !ok {
				return n, false
			}
			n.Tables[i] = node.(*TableName)
		}
	}
	return v.Leave(n)
}

// OnCondition represents JOIN on condition.
type OnCondition struct {
	node

	// Expr is the join condition of an `ON` clause.
	Expr ExprNode
	// Using holds the column names of a `USING` clause.
	Using []*ColumnName
	// JoinUsingAlias is the alias of a `JOIN ... USING (...) AS alias` clause.
	JoinUsingAlias *Alias
}

// Restore implements Node interface.
func (n *OnCondition) Restore(ctx *format.RestoreCtx) error {
	if len(n.Using) > 0 {
		ctx.WriteKeyWord("USING (")
		for i, col := range n.Using {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			if err := col.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore OnCondition.Using")
			}
		}
		ctx.WritePlain(")")
		if n.JoinUsingAlias != nil {
			ctx.WritePlain(" ")
			if err := n.JoinUsingAlias.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore OnCondition.JoinUsingAlias")
			}
		}
		return nil
	}
	ctx.WriteKeyWord("ON ")
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore OnCondition.Expr")
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *OnCondition) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*OnCondition)
	if n.Expr == nil {
		return v.Leave(n)
	}
	node, ok := n.Expr.Accept(v)
	if !ok {
		return n, false
	}
	n.Expr = node.(ExprNode)
	return v.Leave(n)
}

// TableSource represents table source with a name.
type TableSource struct {
	node

	// Source is the source of the data, can be a TableName,
	// a SelectStmt, a SetOprStmt, a JoinNode, or a TableFunction.
	Source ResultSetNode

	// AsName is the alias name of the table source.
	AsName model.CIStr

	// ColumnAliases holds the explicit output column names from a derived
	// table alias list, e.g. (SELECT ...) AS t(a, b) or f() AS t(a, b).
	ColumnAliases []model.CIStr

	// Lateral is true when the table source was introduced with LATERAL.
	Lateral bool

	// TableSample is the optional PostgreSQL TABLESAMPLE clause.
	TableSample *TableSample

	// Only is true for the ONLY table reference modifier.
	Only bool
}

// TableSample is a PostgreSQL TABLESAMPLE clause.
type TableSample struct {
	node

	// Method is the sampling method, e.g. system or bernoulli.
	Method string
	// Args are the arguments of the sampling method.
	Args []ExprNode
	// Repeatable is the optional REPEATABLE clause.
	Repeatable ExprNode
}

// Restore implements Node interface.
func (n *TableSample) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("TABLESAMPLE ")
	ctx.WriteKeyWord(n.Method)
	ctx.WritePlain("(")
	for i, arg := range n.Args {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := arg.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore TableSample.Args[%d]", i)
		}
	}
	ctx.WritePlain(")")
	if n.Repeatable != nil {
		ctx.WriteKeyWord(" REPEATABLE (")
		if err := n.Repeatable.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore TableSample.Repeatable")
		}
		ctx.WritePlain(")")
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *TableSample) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*TableSample)
	for i, arg := range n.Args {
		node, ok := arg.Accept(v)
		if !ok {
			return n, false
		}
		n.Args[i] = node.(ExprNode)
	}
	if n.Repeatable != nil {
		node, ok := n.Repeatable.Accept(v)
		if !ok {
			return n, false
		}
		n.Repeatable = node.(ExprNode)
	}
	return v.Leave(n)
}

// TableFunction represents a set-returning function used as a table source.
type TableFunction struct {
	node
	resultSetNode

	// Expr is the function call expression.
	Expr ExprNode
	// ColDefs are the column definitions of the function result.
	ColDefs []*ColumnDef
	// WithOrdinality is true for a WITH ORDINALITY function.
	WithOrdinality bool
}

// Restore implements Node interface.
func (n *TableFunction) Restore(ctx *format.RestoreCtx) error {
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore TableFunction.Expr")
	}
	if len(n.ColDefs) > 0 {
		ctx.WriteKeyWord(" AS (")
		for i, col := range n.ColDefs {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			if err := col.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore TableFunction.ColDefs")
			}
		}
		ctx.WritePlain(")")
	}
	if n.WithOrdinality {
		ctx.WriteKeyWord(" WITH ORDINALITY")
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *TableFunction) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*TableFunction)
	node, ok := n.Expr.Accept(v)
	if !ok {
		return n, false
	}
	n.Expr = node.(ExprNode)
	return v.Leave(n)
}

// Restore implements Node interface.
func (n *TableSource) Restore(ctx *format.RestoreCtx) error {
	if n.Lateral {
		ctx.WriteKeyWord("LATERAL ")
	}
	needParen := false
	switch n.Source.(type) {
	case *SelectStmt, *SetOprStmt:
		needParen = true
	}
	if needParen {
		ctx.WritePlain("(")
	}
	if err := n.Source.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore TableSource.Source")
	}
	if n.TableSample != nil {
		ctx.WriteKeyWord(" TABLESAMPLE ")
		ctx.WriteKeyWord(n.TableSample.Method)
		ctx.WritePlain("(")
		for i, arg := range n.TableSample.Args {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if err := arg.Restore(ctx); err != nil {
				return err
			}
		}
		ctx.WritePlain(")")
		if n.TableSample.Repeatable != nil {
			ctx.WriteKeyWord(" REPEATABLE (")
			if err := n.TableSample.Repeatable.Restore(ctx); err != nil {
				return err
			}
			ctx.WritePlain(")")
		}
	}
	if needParen {
		ctx.WritePlain(")")
	}
	if asName := n.AsName.String(); asName != "" {
		ctx.WriteKeyWord(" AS ")
		ctx.WriteName(asName)
	}
	if len(n.ColumnAliases) > 0 {
		ctx.WritePlain("(")
		for i, alias := range n.ColumnAliases {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			ctx.WriteName(alias.O)
		}
		ctx.WritePlain(")")
	}

	return nil
}

// Accept implements Node Accept interface.
func (n *TableSource) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*TableSource)
	node, ok := n.Source.Accept(v)
	if !ok {
		return n, false
	}
	n.Source = node.(ResultSetNode)
	if n.TableSample != nil {
		for i, arg := range n.TableSample.Args {
			node, ok := arg.Accept(v)
			if !ok {
				return n, false
			}
			n.TableSample.Args[i] = node.(ExprNode)
		}
		if n.TableSample.Repeatable != nil {
			node, ok := n.TableSample.Repeatable.Accept(v)
			if !ok {
				return n, false
			}
			n.TableSample.Repeatable = node.(ExprNode)
		}
	}
	return v.Leave(n)
}

// SelectLockType is the lock type for SelectStmt.
type SelectLockType int

// Select lock types.
const (
	SelectLockNone SelectLockType = iota
	SelectLockForUpdate
	SelectLockInShareMode
	SelectLockForShare
	SelectLockNoKeyUpdate
	SelectLockKeyShare
)

// String implements fmt.Stringer.
func (slt SelectLockType) String() string {
	switch slt {
	case SelectLockNone:
		return "none"
	case SelectLockForUpdate:
		return "for update"
	case SelectLockInShareMode:
		return "in share mode"
	case SelectLockForShare:
		return "for share"
	case SelectLockNoKeyUpdate:
		return "for no key update"
	case SelectLockKeyShare:
		return "for key share"
	}
	return "unsupported select lock type"
}

// SelectLockWaitPolicy is the wait policy of a locking read.
type SelectLockWaitPolicy int

// Wait policies for locking reads.
const (
	SelectLockWaitDefault SelectLockWaitPolicy = iota
	SelectLockWaitNoWait
	SelectLockWaitSkipLocked
)

// SelectLockInfo is the parsed SELECT locking clause.
type SelectLockInfo struct {
	// Type is the locking read type.
	Type SelectLockType
	// Tables is the optional `OF tbl_name [, ...]` list.
	Tables []*TableName
	// WaitPolicy is the optional NOWAIT / SKIP LOCKED policy.
	WaitPolicy SelectLockWaitPolicy
}

// WildCardField is a special type of select field content.
type WildCardField struct {
	exprNode

	Table  model.CIStr
	Schema model.CIStr
}

// Restore implements Node interface.
func (n *WildCardField) Restore(ctx *format.RestoreCtx) error {
	if schema := n.Schema.String(); schema != "" {
		ctx.WriteName(schema)
		ctx.WritePlain(".")
	}
	if table := n.Table.String(); table != "" {
		ctx.WriteName(table)
		ctx.WritePlain(".")
	}
	ctx.WritePlain("*")
	return nil
}

// Format implements ExprNode interface.
func (n *WildCardField) Format(w io.Writer) {
	if schema := n.Schema.String(); schema != "" {
		fmt.Fprintf(w, "%s.", schema)
	}
	if table := n.Table.String(); table != "" {
		fmt.Fprintf(w, "%s.", table)
	}
	fmt.Fprint(w, "*")
}

// Accept implements Node Accept interface.
func (n *WildCardField) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*WildCardField)
	return v.Leave(n)
}

// SelectField represents fields in select statement.
// There are two type of select field: wildcard
// and expression with optional alias name.
type SelectField struct {
	node

	// Offset is used to get original text.
	Offset int
	// WildCard is not nil, Expr will be nil.
	WildCard *WildCardField
	// Expr is not nil, WildCard will be nil.
	Expr ExprNode
	// AsName is alias name for Expr.
	AsName model.CIStr
	// Auxiliary stands for if this field is auxiliary.
	// When we add a Field into SelectField list which is used for having/orderby clause but the field is not in select clause,
	// we should set its Auxiliary to true. Then the TrimExec will trim the field.
	Auxiliary bool
}

// Restore implements Node interface.
func (n *SelectField) Restore(ctx *format.RestoreCtx) error {
	if n.WildCard != nil {
		if err := n.WildCard.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SelectField.WildCard")
		}
	}
	if n.Expr != nil {
		if err := n.Expr.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SelectField.Expr")
		}
	}
	if asName := n.AsName.String(); asName != "" {
		ctx.WriteKeyWord(" AS ")
		ctx.WriteName(asName)
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *SelectField) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*SelectField)
	if n.Expr != nil {
		node, ok := n.Expr.Accept(v)
		if !ok {
			return n, false
		}
		n.Expr = node.(ExprNode)
	}
	return v.Leave(n)
}

// FieldList represents field list in select statement.
type FieldList struct {
	node

	Fields []*SelectField
}

// Restore implements Node interface.
func (n *FieldList) Restore(ctx *format.RestoreCtx) error {
	for i, v := range n.Fields {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore FieldList.Fields[%d]", i)
		}
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *FieldList) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*FieldList)
	for i, val := range n.Fields {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Fields[i] = node.(*SelectField)
	}
	return v.Leave(n)
}

// TableRefsClause represents table references clause in dml statement.
type TableRefsClause struct {
	node

	TableRefs *Join
}

// Restore implements Node interface.
func (n *TableRefsClause) Restore(ctx *format.RestoreCtx) error {
	if err := n.TableRefs.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore TableRefsClause.TableRefs")
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *TableRefsClause) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*TableRefsClause)
	node, ok := n.TableRefs.Accept(v)
	if !ok {
		return n, false
	}
	n.TableRefs = node.(*Join)
	return v.Leave(n)
}

// ByItem represents an item in order by or group by.
type ByItem struct {
	node

	Expr ExprNode
	Desc bool
	// NullOrder indicates the NULLS ordering: 0 for default, 1 for NULLS FIRST, 2 for NULLS LAST.
	NullOrder int
	// GroupingSet is set for PostgreSQL GROUPING SETS / ROLLUP / CUBE items.
	GroupingSet *GroupingSet
}

// Restore implements Node interface.
func (n *ByItem) Restore(ctx *format.RestoreCtx) error {
	if n.GroupingSet != nil {
		switch n.GroupingSet.Type {
		case GroupingSetEmpty:
			ctx.WritePlain("()")
		case GroupingSetRollup:
			ctx.WriteKeyWord("ROLLUP (")
			restoreExprSets(ctx, n.GroupingSet.Sets)
			ctx.WritePlain(")")
		case GroupingSetCube:
			ctx.WriteKeyWord("CUBE (")
			restoreExprSets(ctx, n.GroupingSet.Sets)
			ctx.WritePlain(")")
		case GroupingSetSets:
			ctx.WriteKeyWord("GROUPING SETS (")
			for i, set := range n.GroupingSet.Sets {
				if i > 0 {
					ctx.WritePlain(", ")
				}
				ctx.WritePlain("(")
				for j, e := range set {
					if j > 0 {
						ctx.WritePlain(", ")
					}
					if err := e.Restore(ctx); err != nil {
						return err
					}
				}
				ctx.WritePlain(")")
			}
			ctx.WritePlain(")")
		}
		return nil
	}
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore ByItem.Expr")
	}
	if n.Desc {
		ctx.WriteKeyWord(" DESC")
	}
	switch n.NullOrder {
	case 1:
		ctx.WriteKeyWord(" NULLS FIRST")
	case 2:
		ctx.WriteKeyWord(" NULLS LAST")
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *ByItem) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*ByItem)
	if n.Expr == nil {
		return v.Leave(n)
	}
	node, ok := n.Expr.Accept(v)
	if !ok {
		return n, false
	}
	n.Expr = node.(ExprNode)
	return v.Leave(n)
}

// GroupByClause represents group by clause.
type GroupByClause struct {
	node
	Items []*ByItem
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

// Restore implements Node interface.
func (n *GroupByClause) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("GROUP BY ")
	for i, v := range n.Items {
		if i != 0 {
			ctx.WritePlain(",")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore GroupByClause.Items[%d]", i)
		}
	}
	if n.WithRollup {
		ctx.WriteKeyWord(" WITH ROLLUP")
	}
	if n.WithCube {
		ctx.WriteKeyWord(" WITH CUBE")
	}
	if n.HasCheckOption {
		if n.CheckOption == model.CheckOptionLocal {
			ctx.WriteKeyWord(" WITH LOCAL CHECK OPTION")
		} else {
			ctx.WriteKeyWord(" WITH CASCADED CHECK OPTION")
		}
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *GroupByClause) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*GroupByClause)
	for i, val := range n.Items {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Items[i] = node.(*ByItem)
	}
	return v.Leave(n)
}

// HavingClause represents having clause.
type HavingClause struct {
	node
	Expr ExprNode
}

// Restore implements Node interface.
func (n *HavingClause) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("HAVING ")
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore HavingClause.Expr")
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *HavingClause) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*HavingClause)
	node, ok := n.Expr.Accept(v)
	if !ok {
		return n, false
	}
	n.Expr = node.(ExprNode)
	return v.Leave(n)
}

// OrderByClause represents order by clause.
type OrderByClause struct {
	node
	Items    []*ByItem
	ForUnion bool
}

// Restore implements Node interface.
func (n *OrderByClause) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("ORDER BY ")
	for i, item := range n.Items {
		if i != 0 {
			ctx.WritePlain(",")
		}
		if err := item.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore OrderByClause.Items[%d]", i)
		}
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *OrderByClause) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*OrderByClause)
	for i, val := range n.Items {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Items[i] = node.(*ByItem)
	}
	return v.Leave(n)
}

// SelectStmt represents the select query node.
// See https://dev.postgres.com/doc/refman/5.7/en/select.html
type SelectStmt struct {
	dmlNode
	resultSetNode

	// SelectStmtOpts wraps around select hints and switches.
	*SelectStmtOpts
	// Distinct represents whether the select has distinct option.
	Distinct bool
	// From is the from clause of the query.
	From *TableRefsClause
	// Where is the where clause in select statement.
	Where ExprNode
	// Fields is the select expression list.
	Fields *FieldList
	// GroupBy is the group by expression list.
	GroupBy *GroupByClause
	// Having is the having condition.
	Having *HavingClause
	// WindowSpecs is the window specification list.
	WindowSpecs []WindowSpec
	// OrderBy is the ordering expression list.
	OrderBy *OrderByClause
	// Limit is the limit clause.
	Limit *Limit
	// LockTp is the lock type
	LockTp SelectLockType
	// LockedTables is the optional `OF tbl_name [, ...]` list of a locking read.
	LockedTables []*TableName
	// LockWaitPolicy is the wait policy of a locking read (NOWAIT / SKIP LOCKED).
	LockWaitPolicy SelectLockWaitPolicy
	// TableHints represents the table level Optimizer Hint for join type
	TableHints []*TableOptimizerHint
	// IntoVars is the variable list of a SELECT ... INTO var_list statement,
	// only meaningful inside a stored program.
	IntoVars []string
	// Into is the optional SELECT ... INTO OUTFILE / DUMPFILE clause.
	Into *SelectIntoStatement
	// With is the optional common table expression clause.
	With *WithClause
	// IsInBraces indicates whether it's a stmt in brace.
	IsInBraces bool
	// AfterSetOperator indicates the SelectStmt after which type of set operator
	// (UNION/EXCEPT/INTERSECT) it is. It is nil when the SelectStmt is the first
	// operand of a set operation or when it is not part of one.
	AfterSetOperator *SetOprType
}

// Restore implements Node interface.
func (n *SelectStmt) Restore(ctx *format.RestoreCtx) error {
	if n.With != nil {
		if err := n.With.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SelectStmt.With")
		}
		ctx.WritePlain(" ")
	}
	ctx.WriteKeyWord("SELECT ")

	if n.SelectStmtOpts.Priority > 0 {
		ctx.WriteKeyWord(postgres.Priority2Str[n.SelectStmtOpts.Priority])
		ctx.WritePlain(" ")
	}

	if n.SelectStmtOpts.SQLSmallResult {
		ctx.WriteKeyWord("SQL_SMALL_RESULT ")
	}

	if n.SelectStmtOpts.SQLBigResult {
		ctx.WriteKeyWord("SQL_BIG_RESULT ")
	}

	if n.SelectStmtOpts.SQLBufferResult {
		ctx.WriteKeyWord("SQL_BUFFER_RESULT ")
	}

	if !n.SelectStmtOpts.SQLCache {
		ctx.WriteKeyWord("SQL_NO_CACHE ")
	}

	if n.TableHints != nil && len(n.TableHints) != 0 {
		ctx.WritePlain("/*+ ")
		for i, tableHint := range n.TableHints {
			if err := tableHint.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore SelectStmt.TableHints[%d]", i)
			}
		}
		ctx.WritePlain("*/ ")
	}

	if n.Distinct {
		ctx.WriteKeyWord("DISTINCT ")
		if n.SelectStmtOpts != nil && len(n.SelectStmtOpts.DistinctOn) > 0 {
			ctx.WriteKeyWord("ON (")
			for i, item := range n.SelectStmtOpts.DistinctOn {
				if i > 0 {
					ctx.WritePlain(", ")
				}
				if err := item.Expr.Restore(ctx); err != nil {
					return errors.Annotate(err, "An error occurred while restore SelectStmt.DistinctOn")
				}
			}
			ctx.WritePlain(") ")
		}
	}
	if n.SelectStmtOpts.StraightJoin {
		ctx.WriteKeyWord("STRAIGHT_JOIN ")
	}
	if n.Fields != nil {
		for i, field := range n.Fields.Fields {
			if i != 0 {
				ctx.WritePlain(",")
			}
			if err := field.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore SelectStmt.Fields[%d]", i)
			}
		}
	}

	if len(n.IntoVars) > 0 {
		ctx.WriteKeyWord(" INTO ")
		for i, v := range n.IntoVars {
			if i != 0 {
				ctx.WritePlain(",")
			}
			if strings.HasPrefix(v, "@") {
				ctx.WritePlain(v)
			} else {
				ctx.WriteName(v)
			}
		}
	}

	if n.From != nil {
		ctx.WriteKeyWord(" FROM ")
		if err := n.From.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SelectStmt.From")
		}
	}

	if n.From == nil && n.Where != nil {
		ctx.WriteKeyWord(" FROM DUAL")
	}
	if n.Where != nil {
		ctx.WriteKeyWord(" WHERE ")
		if err := n.Where.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SelectStmt.Where")
		}
	}

	if n.GroupBy != nil {
		ctx.WritePlain(" ")
		if err := n.GroupBy.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SelectStmt.GroupBy")
		}
	}

	if n.Having != nil {
		ctx.WritePlain(" ")
		if err := n.Having.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SelectStmt.Having")
		}
	}

	if n.WindowSpecs != nil {
		ctx.WriteKeyWord(" WINDOW ")
		for i, windowsSpec := range n.WindowSpecs {
			if i != 0 {
				ctx.WritePlain(",")
			}
			if err := windowsSpec.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore SelectStmt.WindowSpec[%d]", i)
			}
		}
	}

	if n.OrderBy != nil {
		ctx.WritePlain(" ")
		if err := n.OrderBy.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SelectStmt.OrderBy")
		}
	}

	if n.Limit != nil {
		ctx.WritePlain(" ")
		if err := n.Limit.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SelectStmt.Limit")
		}
	}

	switch n.LockTp {
	case SelectLockInShareMode:
		ctx.WriteKeyWord(" LOCK ")
		ctx.WriteKeyWord(n.LockTp.String())
	case SelectLockForUpdate:
		ctx.WritePlain(" ")
		ctx.WriteKeyWord("FOR UPDATE")
	case SelectLockForShare:
		ctx.WritePlain(" ")
		ctx.WriteKeyWord("FOR SHARE")
	case SelectLockNoKeyUpdate:
		ctx.WritePlain(" ")
		ctx.WriteKeyWord("FOR NO KEY UPDATE")
	case SelectLockKeyShare:
		ctx.WritePlain(" ")
		ctx.WriteKeyWord("FOR KEY SHARE")
	}
	if len(n.LockedTables) > 0 {
		ctx.WriteKeyWord(" OF ")
		for i, tbl := range n.LockedTables {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			if err := tbl.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore SelectStmt.LockedTables")
			}
		}
	}
	switch n.LockWaitPolicy {
	case SelectLockWaitNoWait:
		ctx.WriteKeyWord(" NOWAIT")
	case SelectLockWaitSkipLocked:
		ctx.WriteKeyWord(" SKIP LOCKED")
	}
	if n.Into != nil {
		ctx.WritePlain(" ")
		if err := n.Into.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SelectStmt.Into")
		}
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *SelectStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}

	n = newNode.(*SelectStmt)
	if n.With != nil {
		node, ok := n.With.Accept(v)
		if !ok {
			return n, false
		}
		n.With = node.(*WithClause)
	}
	if n.TableHints != nil && len(n.TableHints) != 0 {
		newHints := make([]*TableOptimizerHint, len(n.TableHints))
		for i, hint := range n.TableHints {
			node, ok := hint.Accept(v)
			if !ok {
				return n, false
			}
			newHints[i] = node.(*TableOptimizerHint)
		}
		n.TableHints = newHints
	}

	if n.From != nil {
		node, ok := n.From.Accept(v)
		if !ok {
			return n, false
		}
		n.From = node.(*TableRefsClause)
	}

	if n.Where != nil {
		node, ok := n.Where.Accept(v)
		if !ok {
			return n, false
		}
		n.Where = node.(ExprNode)
	}

	if n.Fields != nil {
		node, ok := n.Fields.Accept(v)
		if !ok {
			return n, false
		}
		n.Fields = node.(*FieldList)
	}

	if n.GroupBy != nil {
		node, ok := n.GroupBy.Accept(v)
		if !ok {
			return n, false
		}
		n.GroupBy = node.(*GroupByClause)
	}

	if n.Having != nil {
		node, ok := n.Having.Accept(v)
		if !ok {
			return n, false
		}
		n.Having = node.(*HavingClause)
	}

	for i, spec := range n.WindowSpecs {
		node, ok := spec.Accept(v)
		if !ok {
			return n, false
		}
		n.WindowSpecs[i] = *node.(*WindowSpec)
	}

	if n.OrderBy != nil {
		node, ok := n.OrderBy.Accept(v)
		if !ok {
			return n, false
		}
		n.OrderBy = node.(*OrderByClause)
	}

	if n.Limit != nil {
		node, ok := n.Limit.Accept(v)
		if !ok {
			return n, false
		}
		n.Limit = node.(*Limit)
	}

	return v.Leave(n)
}

// ValuesStmt is a table value constructor statement.
// See https://dev.postgres.com/doc/refman/8.0/en/values.html
type ValuesStmt struct {
	dmlNode

	Rows []*RowExpr
}

// Restore implements Node interface.
func (n *ValuesStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("VALUES ")
	for i, row := range n.Rows {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if err := row.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while splicing ValuesStmt Row[%v]", i)
		}
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *ValuesStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*ValuesStmt)
	for i, row := range n.Rows {
		node, ok := row.Accept(v)
		if !ok {
			return n, false
		}
		n.Rows[i] = node.(*RowExpr)
	}
	return v.Leave(n)
}

// TableStmt is a `TABLE tbl_name` statement, which is equivalent to
// `SELECT * FROM tbl_name`.
// See https://dev.postgres.com/doc/refman/8.0/en/table.html
type TableStmt struct {
	dmlNode

	Table *TableName
}

// Restore implements Node interface.
func (n *TableStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("TABLE ")
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while splicing TableStmt Table")
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *TableStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*TableStmt)
	node, ok := n.Table.Accept(v)
	if !ok {
		return n, false
	}
	n.Table = node.(*TableName)
	return v.Leave(n)
}

// WithClause is a common table expression clause.
// See https://dev.postgres.com/doc/refman/8.0/en/with.html
type WithClause struct {
	node

	IsRecursive bool
	CTEs        []*CommonTableExpression
}

// Restore implements Node interface.
func (n *WithClause) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("WITH ")
	if n.IsRecursive {
		ctx.WriteKeyWord("RECURSIVE ")
	}
	for i, cte := range n.CTEs {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if err := cte.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore WithClause.CTEs[%d]", i)
		}
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *WithClause) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*WithClause)
	for i, cte := range n.CTEs {
		node, ok := cte.Accept(v)
		if !ok {
			return n, false
		}
		n.CTEs[i] = node.(*CommonTableExpression)
	}
	return v.Leave(n)
}

// CommonTableExpression is a single CTE definition inside a WithClause.
type CommonTableExpression struct {
	node

	Name     model.CIStr
	ColNames []model.CIStr
	Query    ResultSetNode
}

// Restore implements Node interface.
func (n *CommonTableExpression) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteName(n.Name.O)
	if len(n.ColNames) > 0 {
		ctx.WritePlain(" (")
		for i, col := range n.ColNames {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			ctx.WriteName(col.O)
		}
		ctx.WritePlain(")")
	}
	ctx.WriteKeyWord(" AS ")
	ctx.WritePlain("(")
	if err := n.Query.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CommonTableExpression.Query")
	}
	ctx.WritePlain(")")
	return nil
}

// Accept implements Node Accept interface.
func (n *CommonTableExpression) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*CommonTableExpression)
	node, ok := n.Query.Accept(v)
	if !ok {
		return n, false
	}
	n.Query = node.(ResultSetNode)
	return v.Leave(n)
}

// SetOprType is the type of a set operator.
type SetOprType uint8

const (
	// Union is "UNION" (distinct).
	Union SetOprType = iota
	// UnionAll is "UNION ALL".
	UnionAll
	// Except is "EXCEPT" (distinct).
	Except
	// ExceptAll is "EXCEPT ALL".
	ExceptAll
	// Intersect is "INTERSECT" (distinct).
	Intersect
	// IntersectAll is "INTERSECT ALL".
	IntersectAll
)

// String implements fmt.Stringer.
func (s *SetOprType) String() string {
	switch *s {
	case Union:
		return "UNION"
	case UnionAll:
		return "UNION ALL"
	case Except:
		return "EXCEPT"
	case ExceptAll:
		return "EXCEPT ALL"
	case Intersect:
		return "INTERSECT"
	case IntersectAll:
		return "INTERSECT ALL"
	}
	return ""
}

// SetOprSelectList represents the select list in a union/except/intersect statement.
type SetOprSelectList struct {
	node

	With             *WithClause
	AfterSetOperator *SetOprType
	Selects          []Node
	Limit            *Limit
	OrderBy          *OrderByClause
}

// Restore implements Node interface.
func (n *SetOprSelectList) Restore(ctx *format.RestoreCtx) error {
	if n.With != nil {
		if err := n.With.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SetOprSelectList.With")
		}
		ctx.WritePlain(" ")
	}

	for i, stmt := range n.Selects {
		switch selectStmt := stmt.(type) {
		case *SelectStmt:
			if i != 0 {
				ctx.WriteKeyWord(" " + selectStmt.AfterSetOperator.String() + " ")
			}
			if err := selectStmt.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore SetOprSelectList.SelectStmt")
			}
		case *SetOprSelectList:
			if i != 0 {
				ctx.WriteKeyWord(" " + selectStmt.AfterSetOperator.String() + " ")
			}
			ctx.WritePlain("(")
			if err := selectStmt.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore SetOprSelectList.SetOprSelectList")
			}
			ctx.WritePlain(")")
		case *SetOprStmt:
			if i != 0 && selectStmt.SelectList != nil && len(selectStmt.SelectList.Selects) > 0 {
				if first, ok := selectStmt.SelectList.Selects[0].(*SelectStmt); ok && first.AfterSetOperator != nil {
					ctx.WriteKeyWord(" " + first.AfterSetOperator.String() + " ")
				}
			}
			if err := selectStmt.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore SetOprSelectList.SetOprStmt")
			}
		}
	}

	if n.OrderBy != nil {
		ctx.WritePlain(" ")
		if err := n.OrderBy.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SetOprSelectList.OrderBy")
		}
	}

	if n.Limit != nil {
		ctx.WritePlain(" ")
		if err := n.Limit.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SetOprSelectList.Limit")
		}
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *SetOprSelectList) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*SetOprSelectList)
	if n.With != nil {
		node, ok := n.With.Accept(v)
		if !ok {
			return n, false
		}
		n.With = node.(*WithClause)
	}
	for i, sel := range n.Selects {
		node, ok := sel.Accept(v)
		if !ok {
			return n, false
		}
		n.Selects[i] = node
	}
	if n.OrderBy != nil {
		node, ok := n.OrderBy.Accept(v)
		if !ok {
			return n, false
		}
		n.OrderBy = node.(*OrderByClause)
	}
	if n.Limit != nil {
		node, ok := n.Limit.Accept(v)
		if !ok {
			return n, false
		}
		n.Limit = node.(*Limit)
	}
	return v.Leave(n)
}

// SetOprStmt represents a "union/except/intersect statement"
// See https://dev.postgres.com/doc/refman/8.0/en/union.html
// See https://mariadb.com/kb/en/intersect/
// See https://mariadb.com/kb/en/except/
type SetOprStmt struct {
	dmlNode
	resultSetNode

	// IsInBraces indicates whether the whole statement is in braces.
	IsInBraces bool
	SelectList *SetOprSelectList
	OrderBy    *OrderByClause
	Limit      *Limit
	// With is the optional common table expression clause.
	With *WithClause
}

// Restore implements Node interface.
func (n *SetOprStmt) Restore(ctx *format.RestoreCtx) error {
	if n.With != nil {
		if err := n.With.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SetOprStmt.With")
		}
		ctx.WritePlain(" ")
	}
	if n.IsInBraces {
		ctx.WritePlain("(")
	}
	if err := n.SelectList.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore SetOprStmt.SelectList")
	}
	if n.IsInBraces {
		ctx.WritePlain(")")
	}

	if n.OrderBy != nil {
		ctx.WritePlain(" ")
		if err := n.OrderBy.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SetOprStmt.OrderBy")
		}
	}

	if n.Limit != nil {
		ctx.WritePlain(" ")
		if err := n.Limit.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SetOprStmt.Limit")
		}
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *SetOprStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*SetOprStmt)
	if n.With != nil {
		node, ok := n.With.Accept(v)
		if !ok {
			return n, false
		}
		n.With = node.(*WithClause)
	}
	if n.SelectList != nil {
		node, ok := n.SelectList.Accept(v)
		if !ok {
			return n, false
		}
		n.SelectList = node.(*SetOprSelectList)
	}
	if n.OrderBy != nil {
		node, ok := n.OrderBy.Accept(v)
		if !ok {
			return n, false
		}
		n.OrderBy = node.(*OrderByClause)
	}
	if n.Limit != nil {
		node, ok := n.Limit.Accept(v)
		if !ok {
			return n, false
		}
		n.Limit = node.(*Limit)
	}
	return v.Leave(n)
}

// Assignment is the expression for assignment, like a = 1.
type Assignment struct {
	node
	// Column is the column name to be assigned.
	Column *ColumnName
	// MultiColumns holds the column names of a row assignment.
	MultiColumns []*ColumnName
	// Expr is the expression assigning to ColName.
	Expr ExprNode
}

// Restore implements Node interface.
func (n *Assignment) Restore(ctx *format.RestoreCtx) error {
	if err := n.Column.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore Assignment.Column")
	}
	ctx.WritePlain("=")
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore Assignment.Expr")
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *Assignment) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*Assignment)
	node, ok := n.Column.Accept(v)
	if !ok {
		return n, false
	}
	n.Column = node.(*ColumnName)
	node, ok = n.Expr.Accept(v)
	if !ok {
		return n, false
	}
	n.Expr = node.(ExprNode)
	return v.Leave(n)
}

// ColumnNameOrUserVar TODO
type ColumnNameOrUserVar struct {
	ColumnName *ColumnName
	UserVar    *VariableExpr
}

// LoadDataStmt is a statement to load data from a specified file, then insert this rows into an existing table.
// See https://dev.postgres.com/doc/refman/5.7/en/load-data.html
type LoadDataStmt struct {
	dmlNode

	IsLocal           bool
	Path              string
	OnDuplicate       OnDuplicateKeyHandlingType
	Table             *TableName
	Columns           []*ColumnName
	FieldsInfo        *FieldsClause
	LinesInfo         *LinesClause
	IgnoreLines       uint64
	ColumnAssignments []*Assignment

	ColumnsAndUserVars []*ColumnNameOrUserVar
}

// Restore implements Node interface.
func (n *LoadDataStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("LOAD DATA ")
	if n.IsLocal {
		ctx.WriteKeyWord("LOCAL ")
	}
	ctx.WriteKeyWord("INFILE ")
	ctx.WriteString(n.Path)
	if n.OnDuplicate == OnDuplicateKeyHandlingReplace {
		ctx.WriteKeyWord(" REPLACE")
	} else if n.OnDuplicate == OnDuplicateKeyHandlingIgnore {
		ctx.WriteKeyWord(" IGNORE")
	}
	ctx.WriteKeyWord(" INTO TABLE ")
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore LoadDataStmt.Table")
	}
	n.FieldsInfo.Restore(ctx)
	n.LinesInfo.Restore(ctx)
	if n.IgnoreLines != 0 {
		ctx.WriteKeyWord(" IGNORE ")
		ctx.WritePlainf("%d", n.IgnoreLines)
		ctx.WriteKeyWord(" LINES")
	}
	if len(n.ColumnsAndUserVars) != 0 {
		ctx.WritePlain(" (")
		for i, c := range n.ColumnsAndUserVars {
			if i != 0 {
				ctx.WritePlain(",")
			}
			if c.ColumnName != nil {
				if err := c.ColumnName.Restore(ctx); err != nil {
					return errors.Annotate(err, "An error occurred while restore LoadDataStmt.ColumnsAndUserVars")
				}
			}
			if c.UserVar != nil {
				if err := c.UserVar.Restore(ctx); err != nil {
					return errors.Annotate(err, "An error occurred while restore LoadDataStmt.ColumnsAndUserVars")
				}
			}
		}
		ctx.WritePlain(")")
	}

	if n.ColumnAssignments != nil {
		ctx.WriteKeyWord(" SET")
		for i, assign := range n.ColumnAssignments {
			if i != 0 {
				ctx.WritePlain(",")
			}
			ctx.WritePlain(" ")
			if err := assign.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore LoadDataStmt.ColumnAssignments")
			}
		}
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *LoadDataStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*LoadDataStmt)
	if n.Table != nil {
		node, ok := n.Table.Accept(v)
		if !ok {
			return n, false
		}
		n.Table = node.(*TableName)
	}
	for i, val := range n.Columns {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Columns[i] = node.(*ColumnName)
	}

	for i, assignment := range n.ColumnAssignments {
		node, ok := assignment.Accept(v)
		if !ok {
			return n, false
		}
		n.ColumnAssignments[i] = node.(*Assignment)
	}
	return v.Leave(n)
}

// TODO:
const (
	Terminated = iota
	Enclosed
	Escaped
)

// FieldItem TODO:
type FieldItem struct {
	Type  int
	Value string
}

// FieldsClause represents fields references clause in load data statement.
type FieldsClause struct {
	Terminated string
	Enclosed   byte
	Escaped    byte
}

// Restore for FieldsClause
func (n *FieldsClause) Restore(ctx *format.RestoreCtx) error {
	if n.Terminated != "\t" || n.Escaped != '\\' {
		ctx.WriteKeyWord(" FIELDS")
		if n.Terminated != "\t" {
			ctx.WriteKeyWord(" TERMINATED BY ")
			ctx.WriteString(n.Terminated)
		}
		if n.Enclosed != 0 {
			ctx.WriteKeyWord(" ENCLOSED BY ")
			ctx.WriteString(string(n.Enclosed))
		}
		if n.Escaped != '\\' {
			ctx.WriteKeyWord(" ESCAPED BY ")
			if n.Escaped == 0 {
				ctx.WritePlain("''")
			} else {
				ctx.WriteString(string(n.Escaped))
			}
		}
	}
	return nil
}

// LinesClause represents lines references clause in load data statement.
type LinesClause struct {
	Starting   string
	Terminated string
}

// Restore for LinesClause
func (n *LinesClause) Restore(ctx *format.RestoreCtx) error {
	if n.Starting != "" || n.Terminated != "\n" {
		ctx.WriteKeyWord(" LINES")
		if n.Starting != "" {
			ctx.WriteKeyWord(" STARTING BY ")
			ctx.WriteString(n.Starting)
		}
		if n.Terminated != "\n" {
			ctx.WriteKeyWord(" TERMINATED BY ")
			ctx.WriteString(n.Terminated)
		}
	}
	return nil
}

// SelectIntoType is the type of a SELECT ... INTO clause.
type SelectIntoType int

// SELECT ... INTO clause types.
const (
	SelectIntoVarList SelectIntoType = iota + 1
	SelectIntoOutfile
	SelectIntoDumpfile
)

// SelectIntoStatement represents the INTO clause of a SELECT statement,
// covering INTO var_list, INTO OUTFILE and INTO DUMPFILE.
type SelectIntoStatement struct {
	node

	Tp       SelectIntoType
	VarList  []string
	FileName string
	Charset  string
	Fields   *FieldsClause
	Lines    *LinesClause
}

// Restore implements Node interface.
func (n *SelectIntoStatement) Restore(ctx *format.RestoreCtx) error {
	switch n.Tp {
	case SelectIntoVarList:
		ctx.WriteKeyWord("INTO ")
		for i, v := range n.VarList {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			if strings.HasPrefix(v, "@") {
				ctx.WritePlain(v)
			} else {
				ctx.WriteName(v)
			}
		}
	case SelectIntoOutfile:
		ctx.WriteKeyWord("INTO OUTFILE ")
		ctx.WriteString(n.FileName)
		if n.Charset != "" {
			ctx.WriteKeyWord(" CHARACTER SET ")
			ctx.WritePlain(n.Charset)
		}
		if n.Fields != nil {
			if err := n.Fields.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore SelectIntoStatement.Fields")
			}
		}
		if n.Lines != nil {
			if err := n.Lines.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore SelectIntoStatement.Lines")
			}
		}
	case SelectIntoDumpfile:
		ctx.WriteKeyWord("INTO DUMPFILE ")
		ctx.WriteString(n.FileName)
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *SelectIntoStatement) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*SelectIntoStatement)
	return v.Leave(n)
}

// InsertStmt is a statement to insert new rows into an existing table.
// See https://dev.postgres.com/doc/refman/5.7/en/insert.html
type InsertStmt struct {
	dmlNode

	// With is the optional common table expression clause.
	With        *WithClause
	IsReplace   bool
	IgnoreErr   bool
	Table       *TableRefsClause
	Columns     []*ColumnName
	Lists       [][]ExprNode
	Setlist     []*Assignment
	Priority    postgres.PriorityEnum
	OnDuplicate []*Assignment
	Select      ResultSetNode
	// OnConflict is the optional PostgreSQL `ON CONFLICT` clause.
	OnConflict *OnConflictClause
	// Returning is the optional RETURNING field list (MariaDB).
	Returning *FieldList
}

// OnConflictClause is the PostgreSQL `ON CONFLICT` clause of an INSERT.
type OnConflictClause struct {
	node

	// Constraint is the optional constraint name of an `ON CONFLICT ON CONSTRAINT` target.
	Constraint string
	// IndexCols is the optional conflict target column list.
	IndexCols []*IndexColName
	// TargetWhere is the optional predicate of a partial-index conflict target.
	TargetWhere ExprNode
	// DoNothing is true for `DO NOTHING`.
	DoNothing bool
	// UpdateList is the assignment list of `DO UPDATE SET ...`.
	UpdateList []*Assignment
	// UpdateWhere is the optional WHERE of `DO UPDATE`.
	UpdateWhere ExprNode
}

// Restore implements Node interface.
func (n *OnConflictClause) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord(" ON CONFLICT")
	if n.Constraint != "" {
		ctx.WriteKeyWord(" ON CONSTRAINT ")
		ctx.WriteName(n.Constraint)
	}
	if len(n.IndexCols) > 0 {
		ctx.WritePlain(" (")
		for i, col := range n.IndexCols {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if err := col.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore OnConflictClause.IndexCols[%d]", i)
			}
		}
		ctx.WritePlain(")")
	}
	if n.TargetWhere != nil {
		ctx.WriteKeyWord(" WHERE ")
		if err := n.TargetWhere.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore OnConflictClause.TargetWhere")
		}
	}
	if n.DoNothing {
		ctx.WriteKeyWord(" DO NOTHING")
		return nil
	}
	ctx.WriteKeyWord(" DO UPDATE SET ")
	for i, a := range n.UpdateList {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if err := a.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore OnConflictClause.UpdateList[%d]", i)
		}
	}
	if n.UpdateWhere != nil {
		ctx.WriteKeyWord(" WHERE ")
		if err := n.UpdateWhere.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore OnConflictClause.UpdateWhere")
		}
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *OnConflictClause) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*OnConflictClause)
	for i, col := range n.IndexCols {
		node, ok := col.Accept(v)
		if !ok {
			return n, false
		}
		n.IndexCols[i] = node.(*IndexColName)
	}
	if n.TargetWhere != nil {
		node, ok := n.TargetWhere.Accept(v)
		if !ok {
			return n, false
		}
		n.TargetWhere = node.(ExprNode)
	}
	for i, a := range n.UpdateList {
		node, ok := a.Accept(v)
		if !ok {
			return n, false
		}
		n.UpdateList[i] = node.(*Assignment)
	}
	if n.UpdateWhere != nil {
		node, ok := n.UpdateWhere.Accept(v)
		if !ok {
			return n, false
		}
		n.UpdateWhere = node.(ExprNode)
	}
	return v.Leave(n)
}

// Restore implements Node interface.
func (n *InsertStmt) Restore(ctx *format.RestoreCtx) error {
	if n.With != nil {
		if err := n.With.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore InsertStmt.With")
		}
		ctx.WritePlain(" ")
	}
	if n.IsReplace {
		ctx.WriteKeyWord("REPLACE ")
	} else {
		ctx.WriteKeyWord("INSERT ")
	}
	if err := n.Priority.Restore(ctx); err != nil {
		return errors.Trace(err)
	}
	if n.Priority != postgres.NoPriority {
		ctx.WritePlain(" ")
	}
	if n.IgnoreErr {
		ctx.WriteKeyWord("IGNORE ")
	}
	ctx.WriteKeyWord("INTO ")
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore InsertStmt.Table")
	}
	if n.Columns != nil {
		ctx.WritePlain(" (")
		for i, v := range n.Columns {
			if i != 0 {
				ctx.WritePlain(",")
			}
			if err := v.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore InsertStmt.Columns[%d]", i)
			}
		}
		ctx.WritePlain(")")
	}
	if n.Lists != nil {
		ctx.WriteKeyWord(" VALUES ")
		for i, row := range n.Lists {
			if i != 0 {
				ctx.WritePlain(",")
			}
			ctx.WritePlain("(")
			for j, v := range row {
				if j != 0 {
					ctx.WritePlain(",")
				}
				if err := v.Restore(ctx); err != nil {
					return errors.Annotatef(err, "An error occurred while restore InsertStmt.Lists[%d][%d]", i, j)
				}
			}
			ctx.WritePlain(")")
		}
	}
	if n.Select != nil {
		ctx.WritePlain(" ")
		switch v := n.Select.(type) {
		case *SelectStmt, *SetOprStmt:
			if err := v.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore InsertStmt.Select")
			}
		default:
			return errors.Errorf("Incorrect type for InsertStmt.Select: %T", v)
		}
	}
	if n.Setlist != nil {
		ctx.WriteKeyWord(" SET ")
		for i, v := range n.Setlist {
			if i != 0 {
				ctx.WritePlain(",")
			}
			if err := v.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore InsertStmt.Setlist[%d]", i)
			}
		}
	}
	if n.OnDuplicate != nil {
		ctx.WriteKeyWord(" ON DUPLICATE KEY UPDATE ")
		for i, v := range n.OnDuplicate {
			if i != 0 {
				ctx.WritePlain(",")
			}
			if err := v.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore InsertStmt.OnDuplicate[%d]", i)
			}
		}
	}
	if n.OnConflict != nil {
		if err := n.OnConflict.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore InsertStmt.OnConflict")
		}
	}
	if n.Returning != nil {
		ctx.WriteKeyWord(" RETURNING ")
		if err := n.Returning.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore InsertStmt.Returning")
		}
	}

	return nil
}

// Accept implements Node Accept interface.
func (n *InsertStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}

	n = newNode.(*InsertStmt)
	if n.With != nil {
		node, ok := n.With.Accept(v)
		if !ok {
			return n, false
		}
		n.With = node.(*WithClause)
	}
	if n.Select != nil {
		node, ok := n.Select.Accept(v)
		if !ok {
			return n, false
		}
		n.Select = node.(ResultSetNode)
	}

	node, ok := n.Table.Accept(v)
	if !ok {
		return n, false
	}
	n.Table = node.(*TableRefsClause)

	for i, val := range n.Columns {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Columns[i] = node.(*ColumnName)
	}
	for i, list := range n.Lists {
		for j, val := range list {
			node, ok := val.Accept(v)
			if !ok {
				return n, false
			}
			n.Lists[i][j] = node.(ExprNode)
		}
	}
	for i, val := range n.Setlist {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Setlist[i] = node.(*Assignment)
	}
	for i, val := range n.OnDuplicate {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.OnDuplicate[i] = node.(*Assignment)
	}
	if n.OnConflict != nil {
		node, ok := n.OnConflict.Accept(v)
		if !ok {
			return n, false
		}
		n.OnConflict = node.(*OnConflictClause)
	}
	if n.Returning != nil {
		node, ok := n.Returning.Accept(v)
		if !ok {
			return n, false
		}
		n.Returning = node.(*FieldList)
	}
	return v.Leave(n)
}

// DeleteStmt is a statement to delete rows from table.
// See https://dev.postgres.com/doc/refman/5.7/en/delete.html
// MergeWhenType is the action type of a MERGE WHEN clause.
type MergeWhenType int

const (
	// MergeWhenMatchedUpdate updates the target row.
	MergeWhenMatchedUpdate MergeWhenType = iota + 1
	// MergeWhenMatchedDelete deletes the target row.
	MergeWhenMatchedDelete
	// MergeWhenNotMatchedInsert inserts a new row.
	MergeWhenNotMatchedInsert
	// MergeWhenDoNothing skips the row.
	MergeWhenDoNothing
)

// MergeStmt is a PostgreSQL `MERGE INTO ... USING ... ON ... WHEN ...` statement.
type MergeStmt struct {
	dmlNode

	Table     *TableRefsClause
	Source    *TableRefsClause
	On        ExprNode
	Whens     []*MergeWhenClause
	Returning *FieldList
	// With is the optional common table expression clause.
	With *WithClause
}

// Restore implements Node interface.
func (n *MergeStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("MERGE INTO ")
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore MergeStmt.Table")
	}
	ctx.WriteKeyWord(" USING ")
	if err := n.Source.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore MergeStmt.Source")
	}
	ctx.WriteKeyWord(" ON ")
	if err := n.On.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore MergeStmt.On")
	}
	for i, w := range n.Whens {
		ctx.WritePlain(" ")
		if err := w.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore MergeStmt.Whens[%d]", i)
		}
	}
	if n.Returning != nil {
		ctx.WriteKeyWord(" RETURNING ")
		if err := n.Returning.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore MergeStmt.Returning")
		}
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *MergeStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*MergeStmt)
	if n.Table != nil {
		node, ok := n.Table.Accept(v)
		if !ok {
			return n, false
		}
		n.Table = node.(*TableRefsClause)
	}
	if n.Source != nil {
		node, ok := n.Source.Accept(v)
		if !ok {
			return n, false
		}
		n.Source = node.(*TableRefsClause)
	}
	if n.On != nil {
		node, ok := n.On.Accept(v)
		if !ok {
			return n, false
		}
		n.On = node.(ExprNode)
	}
	for i, when := range n.Whens {
		node, ok := when.Accept(v)
		if !ok {
			return n, false
		}
		n.Whens[i] = node.(*MergeWhenClause)
	}
	if n.Returning != nil {
		node, ok := n.Returning.Accept(v)
		if !ok {
			return n, false
		}
		n.Returning = node.(*FieldList)
	}
	return v.Leave(n)
}

// MergeWhenClause is a single WHEN clause of a MERGE statement.
type MergeWhenClause struct {
	node

	Matched    bool
	Condition  ExprNode
	Tp         MergeWhenType
	UpdateList []*Assignment
	Columns    []*ColumnName
	Values     []ExprNode
}

// Restore implements Node interface.
func (n *MergeWhenClause) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("WHEN ")
	if !n.Matched {
		ctx.WriteKeyWord("NOT ")
	}
	ctx.WriteKeyWord("MATCHED")
	if n.Condition != nil {
		ctx.WriteKeyWord(" AND ")
		if err := n.Condition.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore MergeWhenClause.Condition")
		}
	}
	ctx.WriteKeyWord(" THEN ")
	switch n.Tp {
	case MergeWhenMatchedUpdate:
		ctx.WriteKeyWord("UPDATE SET ")
		for i, a := range n.UpdateList {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if err := a.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore MergeWhenClause.UpdateList[%d]", i)
			}
		}
	case MergeWhenMatchedDelete:
		ctx.WriteKeyWord("DELETE")
	case MergeWhenNotMatchedInsert:
		ctx.WriteKeyWord("INSERT")
		if len(n.Columns) > 0 {
			ctx.WritePlain(" (")
			for i, c := range n.Columns {
				if i > 0 {
					ctx.WritePlain(", ")
				}
				if err := c.Restore(ctx); err != nil {
					return errors.Annotatef(err, "An error occurred while restore MergeWhenClause.Columns[%d]", i)
				}
			}
			ctx.WritePlain(")")
		}
		ctx.WriteKeyWord(" VALUES (")
		for i, e := range n.Values {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if err := e.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore MergeWhenClause.Values[%d]", i)
			}
		}
		ctx.WritePlain(")")
	default:
		ctx.WriteKeyWord("DO NOTHING")
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *MergeWhenClause) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*MergeWhenClause)
	if n.Condition != nil {
		node, ok := n.Condition.Accept(v)
		if !ok {
			return n, false
		}
		n.Condition = node.(ExprNode)
	}
	for i, a := range n.UpdateList {
		node, ok := a.Accept(v)
		if !ok {
			return n, false
		}
		n.UpdateList[i] = node.(*Assignment)
	}
	for i, e := range n.Values {
		node, ok := e.Accept(v)
		if !ok {
			return n, false
		}
		n.Values[i] = node.(ExprNode)
	}
	return v.Leave(n)
}

type DeleteStmt struct {
	dmlNode

	// With is the optional common table expression clause.
	With *WithClause
	// TableRefs is used in both single table and multiple table delete statement.
	TableRefs *TableRefsClause
	// Using is the optional PostgreSQL `USING` clause.
	Using *TableRefsClause
	// Tables is only used in multiple table delete statement.
	Tables       *DeleteTableList
	Where        ExprNode
	Order        *OrderByClause
	Limit        *Limit
	Priority     postgres.PriorityEnum
	IgnoreErr    bool
	Quick        bool
	IsMultiTable bool
	BeforeFrom   bool
	// TableHints represents the table level Optimizer Hint for join type.
	TableHints []*TableOptimizerHint
	// Returning is the optional RETURNING field list (MariaDB).
	Returning *FieldList
}

// Restore implements Node interface.
func (n *DeleteStmt) Restore(ctx *format.RestoreCtx) error {
	if n.With != nil {
		if err := n.With.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore DeleteStmt.With")
		}
		ctx.WritePlain(" ")
	}
	ctx.WriteKeyWord("DELETE ")

	if n.TableHints != nil && len(n.TableHints) != 0 {
		ctx.WritePlain("/*+ ")
		for i, tableHint := range n.TableHints {
			if err := tableHint.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore UpdateStmt.TableHints[%d]", i)
			}
		}
		ctx.WritePlain("*/ ")
	}

	if err := n.Priority.Restore(ctx); err != nil {
		return errors.Trace(err)
	}
	if n.Priority != postgres.NoPriority {
		ctx.WritePlain(" ")
	}
	if n.Quick {
		ctx.WriteKeyWord("QUICK ")
	}
	if n.IgnoreErr {
		ctx.WriteKeyWord("IGNORE ")
	}

	if n.IsMultiTable { // Multiple-Table Syntax
		if n.BeforeFrom {
			if err := n.Tables.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore DeleteStmt.Tables")
			}

			ctx.WriteKeyWord(" FROM ")
			if err := n.TableRefs.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore DeleteStmt.TableRefs")
			}
		} else {
			ctx.WriteKeyWord("FROM ")
			if err := n.Tables.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore DeleteStmt.Tables")
			}

			ctx.WriteKeyWord(" USING ")
			if err := n.TableRefs.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore DeleteStmt.TableRefs")
			}
		}
	} else { // Single-Table Syntax
		ctx.WriteKeyWord("FROM ")

		if err := n.TableRefs.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore DeleteStmt.TableRefs")
		}
	}

	if n.Using != nil {
		ctx.WriteKeyWord(" USING ")
		if err := n.Using.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore DeleteStmt.Using")
		}
	}

	if n.Where != nil {
		ctx.WriteKeyWord(" WHERE ")
		if err := n.Where.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore DeleteStmt.Where")
		}
	}

	if n.Order != nil {
		ctx.WritePlain(" ")
		if err := n.Order.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore DeleteStmt.Order")
		}
	}

	if n.Limit != nil {
		ctx.WritePlain(" ")
		if err := n.Limit.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore DeleteStmt.Limit")
		}
	}

	if n.Returning != nil {
		ctx.WriteKeyWord(" RETURNING ")
		if err := n.Returning.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore DeleteStmt.Returning")
		}
	}

	return nil
}

// Accept implements Node Accept interface.
func (n *DeleteStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}

	n = newNode.(*DeleteStmt)
	if n.With != nil {
		node, ok := n.With.Accept(v)
		if !ok {
			return n, false
		}
		n.With = node.(*WithClause)
	}
	node, ok := n.TableRefs.Accept(v)
	if !ok {
		return n, false
	}
	n.TableRefs = node.(*TableRefsClause)

	if n.Tables != nil {
		node, ok = n.Tables.Accept(v)
		if !ok {
			return n, false
		}
		n.Tables = node.(*DeleteTableList)
	}
	if n.Using != nil {
		node, ok = n.Using.Accept(v)
		if !ok {
			return n, false
		}
		n.Using = node.(*TableRefsClause)
	}

	if n.Where != nil {
		node, ok = n.Where.Accept(v)
		if !ok {
			return n, false
		}
		n.Where = node.(ExprNode)
	}
	if n.Order != nil {
		node, ok = n.Order.Accept(v)
		if !ok {
			return n, false
		}
		n.Order = node.(*OrderByClause)
	}
	if n.Limit != nil {
		node, ok = n.Limit.Accept(v)
		if !ok {
			return n, false
		}
		n.Limit = node.(*Limit)
	}
	if n.Returning != nil {
		node, ok = n.Returning.Accept(v)
		if !ok {
			return n, false
		}
		n.Returning = node.(*FieldList)
	}
	return v.Leave(n)
}

// UpdateStmt is a statement to update columns of existing rows in tables with new values.
// See https://dev.postgres.com/doc/refman/5.7/en/update.html
type UpdateStmt struct {
	dmlNode

	// With is the optional common table expression clause.
	With      *WithClause
	TableRefs *TableRefsClause
	// From is the optional PostgreSQL `FROM` clause.
	From          *TableRefsClause
	List          []*Assignment
	Where         ExprNode
	Order         *OrderByClause
	Limit         *Limit
	Priority      postgres.PriorityEnum
	IgnoreErr     bool
	MultipleTable bool
	TableHints    []*TableOptimizerHint
	// Returning is the optional RETURNING field list (MariaDB).
	Returning *FieldList
}

// Restore implements Node interface.
func (n *UpdateStmt) Restore(ctx *format.RestoreCtx) error {
	if n.With != nil {
		if err := n.With.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore UpdateStmt.With")
		}
		ctx.WritePlain(" ")
	}
	ctx.WriteKeyWord("UPDATE ")

	if n.TableHints != nil && len(n.TableHints) != 0 {
		ctx.WritePlain("/*+ ")
		for i, tableHint := range n.TableHints {
			if err := tableHint.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore UpdateStmt.TableHints[%d]", i)
			}
		}
		ctx.WritePlain("*/ ")
	}

	if err := n.Priority.Restore(ctx); err != nil {
		return errors.Trace(err)
	}
	if n.Priority != postgres.NoPriority {
		ctx.WritePlain(" ")
	}
	if n.IgnoreErr {
		ctx.WriteKeyWord("IGNORE ")
	}

	if err := n.TableRefs.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occur while restore UpdateStmt.TableRefs")
	}

	ctx.WriteKeyWord(" SET ")
	for i, assignment := range n.List {
		if i != 0 {
			ctx.WritePlain(", ")
		}

		if err := assignment.Column.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occur while restore UpdateStmt.List[%d].Column", i)
		}

		ctx.WritePlain("=")

		if err := assignment.Expr.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occur while restore UpdateStmt.List[%d].Expr", i)
		}
	}

	if n.From != nil {
		ctx.WriteKeyWord(" FROM ")
		if err := n.From.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore UpdateStmt.From")
		}
	}

	if n.Where != nil {
		ctx.WriteKeyWord(" WHERE ")
		if err := n.Where.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occur while restore UpdateStmt.Where")
		}
	}

	if n.Order != nil {
		ctx.WritePlain(" ")
		if err := n.Order.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occur while restore UpdateStmt.Order")
		}
	}

	if n.Limit != nil {
		ctx.WritePlain(" ")
		if err := n.Limit.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occur while restore UpdateStmt.Limit")
		}
	}

	if n.Returning != nil {
		ctx.WriteKeyWord(" RETURNING ")
		if err := n.Returning.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore UpdateStmt.Returning")
		}
	}

	return nil
}

// Accept implements Node Accept interface.
func (n *UpdateStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*UpdateStmt)
	if n.With != nil {
		node, ok := n.With.Accept(v)
		if !ok {
			return n, false
		}
		n.With = node.(*WithClause)
	}
	node, ok := n.TableRefs.Accept(v)
	if !ok {
		return n, false
	}
	n.TableRefs = node.(*TableRefsClause)
	for i, val := range n.List {
		node, ok = val.Accept(v)
		if !ok {
			return n, false
		}
		n.List[i] = node.(*Assignment)
	}
	if n.Where != nil {
		node, ok = n.Where.Accept(v)
		if !ok {
			return n, false
		}
		n.Where = node.(ExprNode)
	}
	if n.Order != nil {
		node, ok = n.Order.Accept(v)
		if !ok {
			return n, false
		}
		n.Order = node.(*OrderByClause)
	}
	if n.Limit != nil {
		node, ok = n.Limit.Accept(v)
		if !ok {
			return n, false
		}
		n.Limit = node.(*Limit)
	}
	if n.Returning != nil {
		node, ok = n.Returning.Accept(v)
		if !ok {
			return n, false
		}
		n.Returning = node.(*FieldList)
	}
	return v.Leave(n)
}

// Limit is the limit clause.
type Limit struct {
	node

	Count  ExprNode
	Offset ExprNode
}

// Restore implements Node interface.
func (n *Limit) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("LIMIT ")
	if n.Offset != nil {
		if err := n.Offset.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore Limit.Offset")
		}
		ctx.WritePlain(",")
	}
	if err := n.Count.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore Limit.Count")
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *Limit) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	if n.Count != nil {
		node, ok := n.Count.Accept(v)
		if !ok {
			return n, false
		}
		n.Count = node.(ExprNode)
	}
	if n.Offset != nil {
		node, ok := n.Offset.Accept(v)
		if !ok {
			return n, false
		}
		n.Offset = node.(ExprNode)
	}

	n = newNode.(*Limit)
	return v.Leave(n)
}

// ShowStmtType is the type for SHOW statement.
type ShowStmtType int

// Show statement types.
const (
	ShowNone = iota
	ShowEngines
	ShowDatabases
	ShowTables
	ShowTableStatus
	ShowColumns
	ShowWarnings
	ShowCharset
	ShowVariables
	ShowStatus
	ShowCollation
	ShowCreateTable
	ShowCreateView
	ShowCreateUser
	ShowGrants
	ShowTriggers
	ShowProcedureStatus
	ShowIndex
	ShowProcessList
	ShowCreateDatabase
	ShowEvents
	ShowStatsMeta
	ShowStatsHistograms
	ShowStatsBuckets
	ShowStatsHealthy
	ShowPlugins
	ShowProfile
	ShowProfiles
	ShowMasterStatus
	ShowPrivileges
	ShowErrors
	ShowBindings
	ShowOpenTables
)

// TODO:
const (
	ProfileTypeInvalid = iota
	ProfileTypeCPU
	ProfileTypeMemory
	ProfileTypeBlockIo
	ProfileTypeContextSwitch
	ProfileTypePageFaults
	ProfileTypeIpc
	ProfileTypeSwaps
	ProfileTypeSource
	ProfileTypeAll
)

// ShowStmt is a statement to provide information about databases, tables, columns and so on.
// See https://dev.postgres.com/doc/refman/5.7/en/show.html
type ShowStmt struct {
	dmlNode
	resultSetNode

	Tp          ShowStmtType // Databases/Tables/Columns/....
	DBName      string
	Table       *TableName  // Used for showing columns.
	Column      *ColumnName // Used for `desc table column`.
	Flag        int         // Some flag parsed from sql, such as FULL.
	Full        bool
	User        *auth.UserIdentity   // Used for show grants/create user.
	Roles       []*auth.RoleIdentity // Used for show grants .. using
	IfNotExists bool                 // Used for `show create database if not exists`

	// GlobalScope is used by `show variables`
	GlobalScope bool
	Pattern     *PatternLikeExpr
	Where       ExprNode

	ShowProfileTypes []int  // Used for `SHOW PROFILE` syntax
	ShowProfileArgs  *int64 // Used for `SHOW PROFILE` syntax
	ShowProfileLimit *Limit // Used for `SHOW PROFILE` syntax
}

// Restore implements Node interface.
func (n *ShowStmt) Restore(ctx *format.RestoreCtx) error {
	restoreOptFull := func() {
		if n.Full {
			ctx.WriteKeyWord("FULL ")
		}
	}
	restoreShowDatabaseNameOpt := func() {
		if n.DBName != "" {
			// FROM OR IN
			ctx.WriteKeyWord(" IN ")
			ctx.WriteName(n.DBName)
		}
	}
	restoreGlobalScope := func() {
		if n.GlobalScope {
			ctx.WriteKeyWord("GLOBAL ")
		} else {
			ctx.WriteKeyWord("SESSION ")
		}
	}
	restoreShowLikeOrWhereOpt := func() error {
		if n.Pattern != nil && n.Pattern.Pattern != nil {
			ctx.WriteKeyWord(" LIKE ")
			if err := n.Pattern.Pattern.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore ShowStmt.Pattern")
			}
		} else if n.Where != nil {
			ctx.WriteKeyWord(" WHERE ")
			if err := n.Where.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore ShowStmt.Where")
			}
		}
		return nil
	}

	ctx.WriteKeyWord("SHOW ")
	switch n.Tp {
	case ShowCreateTable:
		ctx.WriteKeyWord("CREATE TABLE ")
		if err := n.Table.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore ShowStmt.Table")
		}
	case ShowCreateView:
		ctx.WriteKeyWord("CREATE VIEW ")
		if err := n.Table.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore ShowStmt.VIEW")
		}
	case ShowCreateDatabase:
		ctx.WriteKeyWord("CREATE DATABASE ")
		if n.IfNotExists {
			ctx.WriteKeyWord("IF NOT EXISTS ")
		}
		ctx.WriteName(n.DBName)
	case ShowCreateUser:
		ctx.WriteKeyWord("CREATE USER ")
		if err := n.User.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore ShowStmt.User")
		}
	case ShowGrants:
		ctx.WriteKeyWord("GRANTS")
		if n.User != nil {
			ctx.WriteKeyWord(" FOR ")
			if err := n.User.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore ShowStmt.User")
			}
		}
		if n.Roles != nil {
			ctx.WriteKeyWord(" USING ")
			for i, r := range n.Roles {
				if err := r.Restore(ctx); err != nil {
					return errors.Annotate(err, "An error occurred while restore ShowStmt.User")
				}
				if i != len(n.Roles)-1 {
					ctx.WritePlain(", ")
				}
			}
		}
	case ShowMasterStatus:
		ctx.WriteKeyWord("MASTER STATUS")
	case ShowProcessList:
		restoreOptFull()
		ctx.WriteKeyWord("PROCESSLIST")
	case ShowStatsMeta:
		ctx.WriteKeyWord("STATS_META")
		if err := restoreShowLikeOrWhereOpt(); err != nil {
			return err
		}
	case ShowStatsHistograms:
		ctx.WriteKeyWord("STATS_HISTOGRAMS")
		if err := restoreShowLikeOrWhereOpt(); err != nil {
			return err
		}
	case ShowStatsBuckets:
		ctx.WriteKeyWord("STATS_BUCKETS")
		if err := restoreShowLikeOrWhereOpt(); err != nil {
			return err
		}
	case ShowStatsHealthy:
		ctx.WriteKeyWord("STATS_HEALTHY")
		if err := restoreShowLikeOrWhereOpt(); err != nil {
			return err
		}
	case ShowProfiles:
		ctx.WriteKeyWord("PROFILES")
	case ShowProfile:
		ctx.WriteKeyWord("PROFILE")
		if len(n.ShowProfileTypes) > 0 {
			for i, tp := range n.ShowProfileTypes {
				if i != 0 {
					ctx.WritePlain(",")
				}
				ctx.WritePlain(" ")
				switch tp {
				case ProfileTypeCPU:
					ctx.WriteKeyWord("CPU")
				case ProfileTypeMemory:
					ctx.WriteKeyWord("MEMORY")
				case ProfileTypeBlockIo:
					ctx.WriteKeyWord("BLOCK IO")
				case ProfileTypeContextSwitch:
					ctx.WriteKeyWord("CONTEXT SWITCHES")
				case ProfileTypeIpc:
					ctx.WriteKeyWord("IPC")
				case ProfileTypePageFaults:
					ctx.WriteKeyWord("PAGE FAULTS")
				case ProfileTypeSource:
					ctx.WriteKeyWord("SOURCE")
				case ProfileTypeSwaps:
					ctx.WriteKeyWord("SWAPS")
				case ProfileTypeAll:
					ctx.WriteKeyWord("ALL")
				}
			}
		}
		if n.ShowProfileArgs != nil {
			ctx.WriteKeyWord(" FOR QUERY ")
			ctx.WritePlainf("%d", *n.ShowProfileArgs)
		}
		if n.ShowProfileLimit != nil {
			ctx.WritePlain(" ")
			if err := n.ShowProfileLimit.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore ShowStmt.WritePlain")
			}
		}
	case ShowPrivileges:
		ctx.WriteKeyWord("PRIVILEGES")
	// ShowTargetFilterable
	default:
		switch n.Tp {
		case ShowEngines:
			ctx.WriteKeyWord("ENGINES")
		case ShowDatabases:
			ctx.WriteKeyWord("DATABASES")
		case ShowCharset:
			ctx.WriteKeyWord("CHARSET")
		case ShowTables:
			restoreOptFull()
			ctx.WriteKeyWord("TABLES")
			restoreShowDatabaseNameOpt()
		case ShowOpenTables:
			ctx.WriteKeyWord("OPEN TABLES")
			restoreShowDatabaseNameOpt()
		case ShowTableStatus:
			ctx.WriteKeyWord("TABLE STATUS")
			restoreShowDatabaseNameOpt()
		case ShowIndex:
			// here can be INDEX INDEXES KEYS
			// FROM or IN
			ctx.WriteKeyWord("INDEX IN ")
			if err := n.Table.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while resotre ShowStmt.Table")
			} // TODO: remember to check this case
		case ShowColumns: // equivalent to SHOW FIELDS
			restoreOptFull()
			ctx.WriteKeyWord("COLUMNS")
			if n.Table != nil {
				// FROM or IN
				ctx.WriteKeyWord(" IN ")
				if err := n.Table.Restore(ctx); err != nil {
					return errors.Annotate(err, "An error occurred while resotre ShowStmt.Table")
				}
			}
			restoreShowDatabaseNameOpt()
		case ShowWarnings:
			ctx.WriteKeyWord("WARNINGS")
		case ShowErrors:
			ctx.WriteKeyWord("ERRORS")
		case ShowVariables:
			restoreGlobalScope()
			ctx.WriteKeyWord("VARIABLES")
		case ShowStatus:
			restoreGlobalScope()
			ctx.WriteKeyWord("STATUS")
		case ShowCollation:
			ctx.WriteKeyWord("COLLATION")
		case ShowTriggers:
			ctx.WriteKeyWord("TRIGGERS")
			restoreShowDatabaseNameOpt()
		case ShowProcedureStatus:
			ctx.WriteKeyWord("PROCEDURE STATUS")
		case ShowEvents:
			ctx.WriteKeyWord("EVENTS")
			restoreShowDatabaseNameOpt()
		case ShowPlugins:
			ctx.WriteKeyWord("PLUGINS")
		default:
			return errors.New("Unknown ShowStmt type")
		}
		restoreShowLikeOrWhereOpt()
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *ShowStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*ShowStmt)
	if n.Table != nil {
		node, ok := n.Table.Accept(v)
		if !ok {
			return n, false
		}
		n.Table = node.(*TableName)
	}
	if n.Column != nil {
		node, ok := n.Column.Accept(v)
		if !ok {
			return n, false
		}
		n.Column = node.(*ColumnName)
	}
	if n.Pattern != nil {
		node, ok := n.Pattern.Accept(v)
		if !ok {
			return n, false
		}
		n.Pattern = node.(*PatternLikeExpr)
	}

	switch n.Tp {
	case ShowTriggers, ShowProcedureStatus, ShowProcessList, ShowEvents:
		// We don't have any data to return for those types,
		// but visiting Where may cause resolving error, so return here to avoid error.
		return v.Leave(n)
	}

	if n.Where != nil {
		node, ok := n.Where.Accept(v)
		if !ok {
			return n, false
		}
		n.Where = node.(ExprNode)
	}
	return v.Leave(n)
}

// WindowSpec is the specification of a window.
type WindowSpec struct {
	node

	Name model.CIStr
	// Ref is the reference window of this specification. For example, in `w2 as (w1 order by a)`,
	// the definition of `w2` references `w1`.
	Ref model.CIStr

	PartitionBy *PartitionByClause
	OrderBy     *OrderByClause
	Frame       *FrameClause

	// OnlyAlias will set to true of the first following case.
	// To make compatible with MySQL, we need to distinguish `select func over w` from `select func over (w)`.
	OnlyAlias bool
}

// Restore implements Node interface.
func (n *WindowSpec) Restore(ctx *format.RestoreCtx) error {
	if name := n.Name.String(); name != "" {
		ctx.WriteName(name)
		if n.OnlyAlias {
			return nil
		}
		ctx.WriteKeyWord(" AS ")
	}
	ctx.WritePlain("(")
	sep := ""
	if refName := n.Ref.String(); refName != "" {
		ctx.WriteName(refName)
		sep = " "
	}
	if n.PartitionBy != nil {
		ctx.WritePlain(sep)
		if err := n.PartitionBy.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore WindowSpec.PartitionBy")
		}
		sep = " "
	}
	if n.OrderBy != nil {
		ctx.WritePlain(sep)
		if err := n.OrderBy.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore WindowSpec.OrderBy")
		}
		sep = " "
	}
	if n.Frame != nil {
		ctx.WritePlain(sep)
		if err := n.Frame.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore WindowSpec.Frame")
		}
	}
	ctx.WritePlain(")")

	return nil
}

// Accept implements Node Accept interface.
func (n *WindowSpec) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*WindowSpec)
	if n.PartitionBy != nil {
		node, ok := n.PartitionBy.Accept(v)
		if !ok {
			return n, false
		}
		n.PartitionBy = node.(*PartitionByClause)
	}
	if n.OrderBy != nil {
		node, ok := n.OrderBy.Accept(v)
		if !ok {
			return n, false
		}
		n.OrderBy = node.(*OrderByClause)
	}
	if n.Frame != nil {
		node, ok := n.Frame.Accept(v)
		if !ok {
			return n, false
		}
		n.Frame = node.(*FrameClause)
	}
	return v.Leave(n)
}

// PartitionByClause represents partition by clause.
type PartitionByClause struct {
	node

	Items []*ByItem
}

// Restore implements Node interface.
func (n *PartitionByClause) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("PARTITION BY ")
	for i, v := range n.Items {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore PartitionByClause.Items[%d]", i)
		}
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *PartitionByClause) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*PartitionByClause)
	for i, val := range n.Items {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Items[i] = node.(*ByItem)
	}
	return v.Leave(n)
}

// FrameType is the type of window function frame.
type FrameType int

// Window function frame types.
// MySQL only supports `ROWS` and `RANGES`.
const (
	Rows = iota
	Ranges
	Groups
)

// FrameClause represents frame clause.
type FrameClause struct {
	node

	Type   FrameType
	Extent FrameExtent
}

// Restore implements Node interface.
func (n *FrameClause) Restore(ctx *format.RestoreCtx) error {
	switch n.Type {
	case Rows:
		ctx.WriteKeyWord("ROWS")
	case Ranges:
		ctx.WriteKeyWord("RANGE")
	default:
		return errors.New("Unsupported window function frame type")
	}
	ctx.WriteKeyWord(" BETWEEN ")
	if err := n.Extent.Start.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore FrameClause.Extent.Start")
	}
	ctx.WriteKeyWord(" AND ")
	if err := n.Extent.End.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore FrameClause.Extent.End")
	}

	return nil
}

// Accept implements Node Accept interface.
func (n *FrameClause) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*FrameClause)
	node, ok := n.Extent.Start.Accept(v)
	if !ok {
		return n, false
	}
	n.Extent.Start = *node.(*FrameBound)
	node, ok = n.Extent.End.Accept(v)
	if !ok {
		return n, false
	}
	n.Extent.End = *node.(*FrameBound)
	return v.Leave(n)
}

// FrameExtent represents frame extent.
type FrameExtent struct {
	Start FrameBound
	End   FrameBound
}

// BoundType is the type of window function frame bound.
type BoundType int

// Frame bound types.
const (
	Following = iota
	Preceding
	CurrentRow
)

// FrameBound represents frame bound.
type FrameBound struct {
	node

	Type      BoundType
	UnBounded bool
	Expr      ExprNode
	// `Unit` is used to indicate the units in which the `Expr` should be interpreted.
	// For example: '2:30' MINUTE_SECOND.
	Unit ExprNode
}

// Restore implements Node interface.
func (n *FrameBound) Restore(ctx *format.RestoreCtx) error {
	if n.UnBounded {
		ctx.WriteKeyWord("UNBOUNDED")
	}
	switch n.Type {
	case CurrentRow:
		ctx.WriteKeyWord("CURRENT ROW")
	case Preceding, Following:
		if n.Unit != nil {
			ctx.WriteKeyWord("INTERVAL ")
		}
		if n.Expr != nil {
			if err := n.Expr.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore FrameBound.Expr")
			}
		}
		if n.Unit != nil {
			// Here the Unit string should not be quoted.
			// TODO: This is a temporary workaround that should be changed once something like "Keyword Expression" is implemented.
			var sb strings.Builder
			n.Unit.Restore(format.NewRestoreCtx(0, &sb))
			ctx.WritePlain(" ")
			ctx.WriteKeyWord(sb.String())
		}
		if n.Type == Preceding {
			ctx.WriteKeyWord(" PRECEDING")
		} else {
			ctx.WriteKeyWord(" FOLLOWING")
		}
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *FrameBound) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*FrameBound)
	if n.Expr != nil {
		node, ok := n.Expr.Accept(v)
		if !ok {
			return n, false
		}
		n.Expr = node.(ExprNode)
	}
	if n.Unit != nil {
		node, ok := n.Unit.Accept(v)
		if !ok {
			return n, false
		}
		n.Unit = node.(ExprNode)
	}
	return v.Leave(n)
}

func restoreExprSets(ctx *format.RestoreCtx, sets [][]ExprNode) {
	for i, set := range sets {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		ctx.WritePlain("(")
		for j, e := range set {
			if j > 0 {
				ctx.WritePlain(", ")
			}
			_ = e.Restore(ctx)
		}
		ctx.WritePlain(")")
	}
}
