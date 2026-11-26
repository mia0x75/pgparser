package ast

import (
	"fmt"
	"io"
	"strings"

	"github.com/mia0x75/pgparser/errors"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"

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
	_ Node = &GroupByClause{}
	_ Node = &JoinExpr{}
	_ Node = &Limit{}
	_ Node = &OrderByClause{}
	_ Node = &TableName{}
	_ Node = &FromClause{}
	_ Node = &TableSource{}
	_ Node = &WindowSpec{}
	_ Node = &PartitionByClause{}
	_ Node = &FrameClause{}
	_ Node = &FrameBound{}
)

// TableName represents a table name.
type TableName struct {
	pkast.BaseTableRef

	// Relpersistence is the persistence of a temporary table ('t' or 'u').
	Relpersistence byte

	DBInfo    *model.DBInfo
	TableInfo *model.TableInfo

	IndexHints     []*IndexHint
	PartitionNames []lex.Ident
}

// TableRefCore lets role-based parents check for a missing table before dispatch.
func (n *TableName) TableRefCore() *pkast.BaseTableRef {
	if n == nil {
		return nil
	}
	return &n.BaseTableRef
}

// Restore implements Node interface.
func (n *TableName) Restore(ctx *format.RestoreCtx) error {
	if err := restoreQualifiedName(ctx, n.Name); err != nil {
		return errors.Annotate(err, "An error occurred while restore TableName.Name")
	}
	if len(n.PartitionNames) > 0 {
		ctx.WriteKeyword(" PARTITION")
		ctx.WritePlain("(")
		for i, v := range n.PartitionNames {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			ctx.WriteIdent(v)
		}
		ctx.WritePlain(")")
	}
	for _, value := range n.IndexHints {
		ctx.WritePlain(" ")
		if err := value.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing IndexHints")
		}
	}

	return ctx.Err()
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
	ctx.WriteKeyword(indexHintType)
	ctx.WriteKeyword(indexHintScope)
	ctx.WritePlain(" (")
	for i, value := range n.IndexNames {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		ctx.WriteName(value.O)
	}
	ctx.WritePlain(")")

	return ctx.Err()
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
	return ctx.Err()
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

// TableSample is a PostgreSQL TABLESAMPLE clause.
type TableSample struct {
	node

	// Method is the sampling method, e.g. system or bernoulli.
	Method      string
	MethodParts []string
	// Args are the arguments of the sampling method.
	Args []ExprNode
	// Repeatable is the optional REPEATABLE clause.
	Repeatable ExprNode
}

// Restore implements Node interface.
func (n *TableSample) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("TABLESAMPLE ")
	writeFuncName(ctx, n.MethodParts, n.Method)
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
		ctx.WriteKeyword(" REPEATABLE (")
		if err := n.Repeatable.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore TableSample.Repeatable")
		}
		ctx.WritePlain(")")
	}
	return ctx.Err()
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
	if n.WithOrdinality {
		ctx.WriteKeyword(" WITH ORDINALITY")
	}
	if len(n.ColDefs) > 0 {
		ctx.WriteKeyword(" AS (")
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
	return ctx.Err()
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
	for i, col := range n.ColDefs {
		node, ok := col.Accept(v)
		if !ok {
			return n, false
		}
		n.ColDefs[i] = node.(*ColumnDef)
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

// SelectStmt represents the select query node.
type SelectStmt struct {
	pkast.BaseSelectStmt

	// SelectStmtOpts wraps around select hints and switches.
	*SelectStmtOpts
	// LockTp is the lock type
	LockTp SelectLockType
	// LockedTables is the optional `OF tbl_name [, ...]` list of a locking read.
	LockedTables []*TableName
	// LockWaitPolicy is the wait policy of a locking read (NOWAIT / SKIP LOCKED).
	LockWaitPolicy SelectLockWaitPolicy
	Locking        *List
	// TableHints represents the table level Optimizer Hint for join type
	TableHints []*TableOptimizerHint
	// IntoVars is the variable list of a SELECT ... INTO var_list statement,
	// only meaningful inside a stored program.
	IntoVars []string
	// Into is the optional SELECT ... INTO OUTFILE / DUMPFILE clause.
	Into      *SelectIntoStatement
	IntoTable *IntoClause
	// With is the optional common table expression clause.
	With *WithClause
	// IsInBraces indicates whether it's a stmt in brace.
	IsInBraces bool
}

// Restore implements Node interface.
func (n *SelectStmt) Restore(ctx *format.RestoreCtx) error {
	if n.With != nil {
		if err := n.With.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SelectStmt.With")
		}
		ctx.WritePlain(" ")
	}
	ctx.WriteKeyword("SELECT ")

	if n.Priority > 0 {
		ctx.WriteKeyword(postgres.Priority2Str[n.Priority])
		ctx.WritePlain(" ")
	}

	if n.SQLSmallResult {
		ctx.WriteKeyword("SQL_SMALL_RESULT ")
	}

	if n.SQLBigResult {
		ctx.WriteKeyword("SQL_BIG_RESULT ")
	}

	if n.SQLBufferResult {
		ctx.WriteKeyword("SQL_BUFFER_RESULT ")
	}

	if len(n.TableHints) != 0 {
		ctx.WritePlain("/*+ ")
		for i, tableHint := range n.TableHints {
			if err := tableHint.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore SelectStmt.TableHints[%d]", i)
			}
		}
		ctx.WritePlain("*/ ")
	}

	if n.Distinct {
		ctx.WriteKeyword("DISTINCT ")
		if n.SelectStmtOpts != nil && len(n.DistinctOn) > 0 {
			ctx.WriteKeyword("ON (")
			for i, item := range n.DistinctOn {
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
	if n.StraightJoin {
		ctx.WriteKeyword("STRAIGHT_JOIN ")
	}
	if n.Fields != nil {
		for i, field := range n.Fields.Items {
			if i != 0 {
				ctx.WritePlain(",")
			}
			if err := field.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore SelectStmt.Fields[%d]", i)
			}
		}
	}

	if len(n.IntoVars) > 0 {
		ctx.WriteKeyword(" INTO ")
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
	if n.IntoTable != nil {
		ctx.WriteKeyword(" INTO ")
		if err := n.IntoTable.Restore(ctx); err != nil {
			return err
		}
	}

	if n.From != nil {
		ctx.WriteKeyword(" FROM ")
		if err := n.From.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SelectStmt.From")
		}
	}

	if n.Where != nil {
		ctx.WriteKeyword(" WHERE ")
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
		ctx.WriteKeyword("HAVING ")
		if err := n.Having.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SelectStmt.Having")
		}
	}

	if n.WindowSpecs != nil {
		ctx.WriteKeyword(" WINDOW ")
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

	if n.Locking != nil {
		if err := restoreLocking(ctx, n.Locking); err != nil {
			return err
		}
	} else {
		switch n.LockTp {
		case SelectLockInShareMode:
			ctx.WriteKeyword(" LOCK ")
			ctx.WriteKeyword(n.LockTp.String())
		case SelectLockForUpdate:
			ctx.WritePlain(" ")
			ctx.WriteKeyword("FOR UPDATE")
		case SelectLockForShare:
			ctx.WritePlain(" ")
			ctx.WriteKeyword("FOR SHARE")
		case SelectLockNoKeyUpdate:
			ctx.WritePlain(" ")
			ctx.WriteKeyword("FOR NO KEY UPDATE")
		case SelectLockKeyShare:
			ctx.WritePlain(" ")
			ctx.WriteKeyword("FOR KEY SHARE")
		}
		if len(n.LockedTables) > 0 {
			ctx.WriteKeyword(" OF ")
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
			ctx.WriteKeyword(" NOWAIT")
		case SelectLockWaitSkipLocked:
			ctx.WriteKeyword(" SKIP LOCKED")
		}
	}
	if n.Into != nil {
		ctx.WritePlain(" ")
		if err := n.Into.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SelectStmt.Into")
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *SelectStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}

	n = newNode.(*SelectStmt)
	if n.Locking != nil {
		child, ok := n.Locking.Accept(v)
		if !ok {
			return n, false
		}
		n.Locking = child.(*List)
	}
	if n.IntoTable != nil {
		node, ok := n.IntoTable.Accept(v)
		if !ok {
			return n, false
		}
		n.IntoTable = node.(*IntoClause)
	}
	for i, item := range n.DistinctOn {
		node, ok := item.Accept(v)
		if !ok {
			return n, false
		}
		n.DistinctOn[i] = node.(*GroupItem)
	}
	if n.With != nil {
		node, ok := n.With.Accept(v)
		if !ok {
			return n, false
		}
		n.With = node.(*WithClause)
	}
	if len(n.TableHints) != 0 {
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
		n.From = node.(*FromClause)
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
		if !nodePresent(n.Having) {
			return n, false
		}
		node, ok := n.Having.Accept(v)
		if !ok {
			return n, false
		}
		having, ok := node.(ExprNode)
		if !ok || !nodePresent(having) {
			return n, false
		}
		n.Having = having
	}

	for i, spec := range n.WindowSpecs {
		if !nodePresent(spec) || spec.WindowSpecCore() == nil {
			return n, false
		}
		node, ok := spec.Accept(v)
		if !ok {
			return n, false
		}
		replacement, ok := node.(WindowSpecNode)
		if !ok || replacement.WindowSpecCore() == nil {
			return n, false
		}
		n.WindowSpecs[i] = replacement
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
type ValuesStmt struct {
	pkast.BaseQuery

	Rows []*RowExpr
}

// Restore implements Node interface.
func (n *ValuesStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("VALUES ")
	for i, row := range n.Rows {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		ctx.WritePlain("(")
		for j, value := range row.Items {
			if j > 0 {
				ctx.WritePlain(", ")
			}
			if err := value.Restore(ctx); err != nil {
				return err
			}
		}
		ctx.WritePlain(")")
	}
	return ctx.Err()
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
type TableStmt struct {
	pkast.BaseQuery

	Table *TableName
}

// Restore implements Node interface.
func (n *TableStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("TABLE ")
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while splicing TableStmt Table")
	}
	return ctx.Err()
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

// ColumnNameOrUserVar TODO
type ColumnNameOrUserVar struct {
	ColumnName *ColumnName
	UserVar    *VariableExpr
}

// LoadDataStmt is a statement to load data from a specified file, then insert this rows into an existing table.
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
	ctx.WriteKeyword("LOAD DATA ")
	if n.IsLocal {
		ctx.WriteKeyword("LOCAL ")
	}
	ctx.WriteKeyword("INFILE ")
	ctx.WriteString(n.Path)
	switch n.OnDuplicate {
	case OnDuplicateKeyHandlingReplace:
		ctx.WriteKeyword(" REPLACE")
	case OnDuplicateKeyHandlingIgnore:
		ctx.WriteKeyword(" IGNORE")
	}
	ctx.WriteKeyword(" INTO TABLE ")
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore LoadDataStmt.Table")
	}
	if err := n.FieldsInfo.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore LoadDataStmt.FieldsInfo")
	}
	if err := n.LinesInfo.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore LoadDataStmt.LinesInfo")
	}
	if n.IgnoreLines != 0 {
		ctx.WriteKeyword(" IGNORE ")
		ctx.WritePlainf("%d", n.IgnoreLines)
		ctx.WriteKeyword(" LINES")
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
		ctx.WriteKeyword(" SET")
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
	return ctx.Err()
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
		ctx.WriteKeyword(" FIELDS")
		if n.Terminated != "\t" {
			ctx.WriteKeyword(" TERMINATED BY ")
			ctx.WriteString(n.Terminated)
		}
		if n.Enclosed != 0 {
			ctx.WriteKeyword(" ENCLOSED BY ")
			ctx.WriteString(string(n.Enclosed))
		}
		if n.Escaped != '\\' {
			ctx.WriteKeyword(" ESCAPED BY ")
			if n.Escaped == 0 {
				ctx.WritePlain("''")
			} else {
				ctx.WriteString(string(n.Escaped))
			}
		}
	}
	return ctx.Err()
}

// LinesClause represents lines references clause in load data statement.
type LinesClause struct {
	Starting   string
	Terminated string
}

// Restore for LinesClause
func (n *LinesClause) Restore(ctx *format.RestoreCtx) error {
	if n.Starting != "" || n.Terminated != "\n" {
		ctx.WriteKeyword(" LINES")
		if n.Starting != "" {
			ctx.WriteKeyword(" STARTING BY ")
			ctx.WriteString(n.Starting)
		}
		if n.Terminated != "\n" {
			ctx.WriteKeyword(" TERMINATED BY ")
			ctx.WriteString(n.Terminated)
		}
	}
	return ctx.Err()
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
		ctx.WriteKeyword("INTO ")
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
		ctx.WriteKeyword("INTO OUTFILE ")
		ctx.WriteString(n.FileName)
		if n.Charset != "" {
			ctx.WriteKeyword(" CHARACTER SET ")
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
		ctx.WriteKeyword("INTO DUMPFILE ")
		ctx.WriteString(n.FileName)
	}
	return ctx.Err()
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
// InsertStmt is a statement to insert rows into a table.
type InsertStmt struct {
	pkast.BaseInsertStmt

	// With is the optional common table expression clause.
	With *WithClause
	// TargetAlias is the optional alias of the insert target.
	TargetAlias lex.Ident
	// OnConflict is the optional PostgreSQL `ON CONFLICT` clause.
	OnConflict *OnConflictClause
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
	ctx.WriteKeyword(" ON CONFLICT")
	if n.Constraint != "" {
		ctx.WriteKeyword(" ON CONSTRAINT ")
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
		ctx.WriteKeyword(" WHERE ")
		if err := n.TargetWhere.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore OnConflictClause.TargetWhere")
		}
	}
	if n.DoNothing {
		ctx.WriteKeyword(" DO NOTHING")
		return ctx.Err()
	}
	ctx.WriteKeyword(" DO UPDATE SET ")
	for i, a := range n.UpdateList {
		if !assignmentValid(a) {
			return errors.Errorf("restore OnConflictClause.UpdateList[%d]: invalid assignment", i)
		}
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if err := a.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore OnConflictClause.UpdateList[%d]", i)
		}
	}
	if n.UpdateWhere != nil {
		ctx.WriteKeyword(" WHERE ")
		if err := n.UpdateWhere.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore OnConflictClause.UpdateWhere")
		}
	}
	return ctx.Err()
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
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return errors.New("restore InsertStmt: nil statement")
	}
	if !nodePresent(n.Table) {
		return errors.New("restore InsertStmt: missing table")
	}
	if n.With != nil {
		if err := n.With.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore InsertStmt.With")
		}
		ctx.WritePlain(" ")
	}
	ctx.WriteKeyword("INSERT INTO ")
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore InsertStmt.Table")
	}
	if n.TargetAlias.Present() {
		ctx.WriteKeyword(" AS ")
		ctx.WriteIdent(n.TargetAlias)
	}
	if n.Columns != nil {
		ctx.WritePlain(" (")
		for i, v := range n.Columns {
			if i != 0 {
				ctx.WritePlain(",")
			}
			if !nodePresent(v) {
				return errors.Errorf("restore InsertStmt.Columns[%d]: missing target", i)
			}
			if err := v.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore InsertStmt.Columns[%d]", i)
			}
		}
		ctx.WritePlain(")")
	}
	if n.Values != nil {
		ctx.WriteKeyword(" VALUES ")
		for i, row := range n.Values {
			if i != 0 {
				ctx.WritePlain(",")
			}
			ctx.WritePlain("(")
			for j, v := range row {
				if j != 0 {
					ctx.WritePlain(",")
				}
				if !nodePresent(v) {
					return errors.Errorf("restore InsertStmt.Values[%d][%d]: missing expression", i, j)
				}
				if err := v.Restore(ctx); err != nil {
					return errors.Annotatef(err, "An error occurred while restore InsertStmt.Values[%d][%d]", i, j)
				}
			}
			ctx.WritePlain(")")
		}
	}
	if n.Values == nil && n.Query == nil {
		ctx.WriteKeyword(" DEFAULT VALUES")
	}
	if n.Query != nil {
		ctx.WritePlain(" ")
		if !nodePresent(n.Query) {
			return errors.New("restore InsertStmt.Query: nil query")
		}
		if err := n.Query.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore InsertStmt.Query")
		}
	}
	if n.OnConflict != nil {
		if err := n.OnConflict.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore InsertStmt.OnConflict")
		}
	}
	if n.Returning != nil {
		ctx.WriteKeyword(" RETURNING ")
		if err := n.Returning.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore InsertStmt.Returning")
		}
	}

	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *InsertStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	outer, ok := entered.(*InsertStmt)
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
	if outer.Table != nil {
		if !nodePresent(outer.Table) {
			return outer, false
		}
		node, ok := outer.Table.Accept(v)
		if !ok {
			return outer, false
		}
		table, ok := node.(TableRefNode)
		if !ok || !nodePresent(table) {
			return outer, false
		}
		outer.Table = table
	}
	for i, val := range outer.Columns {
		if !nodePresent(val) {
			return outer, false
		}
		node, ok := val.Accept(v)
		if !ok {
			return outer, false
		}
		column, ok := node.(ExprNode)
		if !ok || !nodePresent(column) {
			return outer, false
		}
		outer.Columns[i] = column
	}
	for i, list := range outer.Values {
		for j, val := range list {
			if !nodePresent(val) {
				return outer, false
			}
			node, ok := val.Accept(v)
			if !ok {
				return outer, false
			}
			expr, ok := node.(ExprNode)
			if !ok || !nodePresent(expr) {
				return outer, false
			}
			outer.Values[i][j] = expr
		}
	}
	if outer.Query != nil {
		if !nodePresent(outer.Query) {
			return outer, false
		}
		node, ok := outer.Query.Accept(v)
		if !ok {
			return outer, false
		}
		query, ok := node.(QueryNode)
		if !ok || !nodePresent(query) {
			return outer, false
		}
		outer.Query = query
	}
	if outer.OnConflict != nil {
		node, ok := outer.OnConflict.Accept(v)
		if !ok {
			return outer, false
		}
		conflict, ok := node.(*OnConflictClause)
		if !ok || conflict == nil {
			return outer, false
		}
		outer.OnConflict = conflict
	}
	if outer.Returning != nil {
		if !nodePresent(outer.Returning) {
			return outer, false
		}
		node, ok := outer.Returning.Accept(v)
		if !ok {
			return outer, false
		}
		returning, ok := node.(*FieldList)
		if !ok || returning == nil {
			return outer, false
		}
		outer.Returning = returning
	}
	return v.Leave(outer)
}

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

	Table     *FromClause
	Source    *FromClause
	On        ExprNode
	Whens     []*MergeWhenClause
	Returning *FieldList
	// With is the optional common table expression clause.
	With *WithClause
}

// Restore implements Node interface.
func (n *MergeStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("MERGE INTO ")
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore MergeStmt.Table")
	}
	ctx.WriteKeyword(" USING ")
	if err := n.Source.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore MergeStmt.Source")
	}
	ctx.WriteKeyword(" ON ")
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
		ctx.WriteKeyword(" RETURNING ")
		if err := n.Returning.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore MergeStmt.Returning")
		}
	}
	return ctx.Err()
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
		n.Table = node.(*FromClause)
	}
	if n.Source != nil {
		node, ok := n.Source.Accept(v)
		if !ok {
			return n, false
		}
		n.Source = node.(*FromClause)
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
	Columns    []ExprNode
	Values     []ExprNode
}

// Restore implements Node interface.
func (n *MergeWhenClause) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("WHEN ")
	if !n.Matched {
		ctx.WriteKeyword("NOT ")
	}
	ctx.WriteKeyword("MATCHED")
	if n.Condition != nil {
		ctx.WriteKeyword(" AND ")
		if err := n.Condition.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore MergeWhenClause.Condition")
		}
	}
	ctx.WriteKeyword(" THEN ")
	switch n.Tp {
	case MergeWhenMatchedUpdate:
		ctx.WriteKeyword("UPDATE SET ")
		for i, a := range n.UpdateList {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if err := a.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore MergeWhenClause.UpdateList[%d]", i)
			}
		}
	case MergeWhenMatchedDelete:
		ctx.WriteKeyword("DELETE")
	case MergeWhenNotMatchedInsert:
		ctx.WriteKeyword("INSERT")
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
		if len(n.Values) == 0 {
			ctx.WriteKeyword(" DEFAULT VALUES")
			break
		}
		ctx.WriteKeyword(" VALUES (")
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
		ctx.WriteKeyword("DO NOTHING")
	}
	return ctx.Err()
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

// DeleteStmt keeps PostgreSQL's multi-target delete outside the shared syntax.
type DeleteStmt struct {
	pkast.BaseDeleteStmt

	// With is the optional common table expression clause.
	With *WithClause
	// Tables is only used in multiple table delete statement.
	Tables       *DeleteTableList
	IsMultiTable bool
	BeforeFrom   bool
}

// Restore implements Node interface.
func (n *DeleteStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return errors.New("restore DeleteStmt: nil statement")
	}
	if !nodePresent(n.Table) {
		return errors.New("restore DeleteStmt: missing table")
	}
	if n.With != nil {
		if err := n.With.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore DeleteStmt.With")
		}
		ctx.WritePlain(" ")
	}
	ctx.WriteKeyword("DELETE ")

	if n.IsMultiTable { // Multiple-Table Syntax
		if n.Tables == nil {
			return errors.New("restore DeleteStmt.Tables: missing tables")
		}
		if n.BeforeFrom {
			if err := n.Tables.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore DeleteStmt.Tables")
			}

			ctx.WriteKeyword(" FROM ")
			if err := n.Table.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore DeleteStmt.Table")
			}
		} else {
			ctx.WriteKeyword("FROM ")
			if err := n.Tables.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore DeleteStmt.Tables")
			}

			ctx.WriteKeyword(" USING ")
			if err := n.Table.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore DeleteStmt.Table")
			}
		}
	} else { // Single-Table Syntax
		ctx.WriteKeyword("FROM ")

		if err := n.Table.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore DeleteStmt.Table")
		}
	}

	if n.Using != nil {
		ctx.WriteKeyword(" USING ")
		if err := n.Using.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore DeleteStmt.Using")
		}
	}

	if n.Where != nil {
		ctx.WriteKeyword(" WHERE ")
		if err := n.Where.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore DeleteStmt.Where")
		}
	}

	if n.OrderBy != nil {
		ctx.WritePlain(" ")
		if err := n.OrderBy.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore DeleteStmt.OrderBy")
		}
	}

	if n.Limit != nil {
		ctx.WritePlain(" ")
		if err := n.Limit.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore DeleteStmt.Limit")
		}
	}

	if n.Returning != nil {
		ctx.WriteKeyword(" RETURNING ")
		if err := n.Returning.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore DeleteStmt.Returning")
		}
	}

	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DeleteStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	outer, ok := entered.(*DeleteStmt)
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
	if outer.Table != nil {
		if !nodePresent(outer.Table) {
			return outer, false
		}
		node, ok := outer.Table.Accept(v)
		if !ok {
			return outer, false
		}
		table, ok := node.(ResultSetNode)
		if !ok || !nodePresent(table) {
			return outer, false
		}
		outer.Table = table
	}
	if outer.Tables != nil {
		node, ok := outer.Tables.Accept(v)
		if !ok {
			return outer, false
		}
		tables, ok := node.(*DeleteTableList)
		if !ok || tables == nil {
			return outer, false
		}
		outer.Tables = tables
	}
	if outer.Using != nil {
		if !nodePresent(outer.Using) {
			return outer, false
		}
		node, ok := outer.Using.Accept(v)
		if !ok {
			return outer, false
		}
		using, ok := node.(ResultSetNode)
		if !ok || !nodePresent(using) {
			return outer, false
		}
		outer.Using = using
	}
	if outer.Where != nil {
		if !nodePresent(outer.Where) {
			return outer, false
		}
		node, ok := outer.Where.Accept(v)
		if !ok {
			return outer, false
		}
		where, ok := node.(ExprNode)
		if !ok || !nodePresent(where) {
			return outer, false
		}
		outer.Where = where
	}
	if outer.OrderBy != nil {
		if !nodePresent(outer.OrderBy) {
			return outer, false
		}
		node, ok := outer.OrderBy.Accept(v)
		if !ok {
			return outer, false
		}
		order, ok := node.(OrderByClauseNode)
		if !ok || order.OrderByClauseCore() == nil {
			return outer, false
		}
		outer.OrderBy = order
	}
	if outer.Limit != nil {
		if !nodePresent(outer.Limit) {
			return outer, false
		}
		node, ok := outer.Limit.Accept(v)
		if !ok {
			return outer, false
		}
		limit, ok := node.(LimitNode)
		if !ok || limit.LimitCore() == nil {
			return outer, false
		}
		outer.Limit = limit
	}
	if outer.Returning != nil {
		if !nodePresent(outer.Returning) {
			return outer, false
		}
		node, ok := outer.Returning.Accept(v)
		if !ok {
			return outer, false
		}
		returning, ok := node.(*FieldList)
		if !ok || returning == nil {
			return outer, false
		}
		outer.Returning = returning
	}
	return v.Leave(outer)
}

// UpdateStmt is a statement to update columns of existing rows in tables with new values.
// UpdateStmt keeps PostgreSQL's CTE prefix and FROM clause outside the shared syntax.
type UpdateStmt struct {
	pkast.BaseUpdateStmt

	// With is the optional common table expression clause.
	With *WithClause
	// From is the optional PostgreSQL `FROM` clause.
	From *FromClause
}

// Restore implements Node interface.
func (n *UpdateStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return errors.New("restore UpdateStmt: nil statement")
	}
	if !nodePresent(n.Table) {
		return errors.New("restore UpdateStmt: missing table")
	}
	if len(n.Assignments) == 0 {
		return errors.New("restore UpdateStmt: missing assignments")
	}
	if n.With != nil {
		if err := n.With.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore UpdateStmt.With")
		}
		ctx.WritePlain(" ")
	}
	ctx.WriteKeyword("UPDATE ")

	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occur while restore UpdateStmt.Table")
	}

	ctx.WriteKeyword(" SET ")
	for i, assignment := range n.Assignments {
		if !assignmentValid(assignment) {
			return errors.Errorf("restore UpdateStmt.Assignments[%d]: invalid assignment", i)
		}
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := assignment.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occur while restore UpdateStmt.Assignments[%d]", i)
		}
	}

	if n.From != nil {
		ctx.WriteKeyword(" FROM ")
		if err := n.From.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore UpdateStmt.From")
		}
	}

	if n.Where != nil {
		ctx.WriteKeyword(" WHERE ")
		if err := n.Where.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occur while restore UpdateStmt.Where")
		}
	}

	if n.OrderBy != nil {
		ctx.WritePlain(" ")
		if err := n.OrderBy.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occur while restore UpdateStmt.OrderBy")
		}
	}

	if n.Limit != nil {
		ctx.WritePlain(" ")
		if err := n.Limit.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occur while restore UpdateStmt.Limit")
		}
	}

	if n.Returning != nil {
		ctx.WriteKeyword(" RETURNING ")
		if err := n.Returning.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore UpdateStmt.Returning")
		}
	}

	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *UpdateStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	outer, ok := entered.(*UpdateStmt)
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
	if outer.Table != nil {
		if !nodePresent(outer.Table) {
			return outer, false
		}
		node, ok := outer.Table.Accept(v)
		if !ok {
			return outer, false
		}
		table, ok := node.(ResultSetNode)
		if !ok || !nodePresent(table) {
			return outer, false
		}
		outer.Table = table
	}
	if outer.From != nil {
		if !nodePresent(outer.From) {
			return outer, false
		}
		node, ok := outer.From.Accept(v)
		if !ok {
			return outer, false
		}
		from, ok := node.(*FromClause)
		if !ok || from == nil {
			return outer, false
		}
		outer.From = from
	}
	for i, assignment := range outer.Assignments {
		if !nodePresent(assignment) {
			return outer, false
		}
		node, ok := assignment.Accept(v)
		if !ok {
			return outer, false
		}
		replacement, ok := node.(*Assignment)
		if !ok || replacement == nil {
			return outer, false
		}
		outer.Assignments[i] = replacement
	}
	if outer.Where != nil {
		if !nodePresent(outer.Where) {
			return outer, false
		}
		node, ok := outer.Where.Accept(v)
		if !ok {
			return outer, false
		}
		where, ok := node.(ExprNode)
		if !ok || !nodePresent(where) {
			return outer, false
		}
		outer.Where = where
	}
	if outer.OrderBy != nil {
		if !nodePresent(outer.OrderBy) {
			return outer, false
		}
		node, ok := outer.OrderBy.Accept(v)
		if !ok {
			return outer, false
		}
		order, ok := node.(OrderByClauseNode)
		if !ok || order.OrderByClauseCore() == nil {
			return outer, false
		}
		outer.OrderBy = order
	}
	if outer.Limit != nil {
		if !nodePresent(outer.Limit) {
			return outer, false
		}
		node, ok := outer.Limit.Accept(v)
		if !ok {
			return outer, false
		}
		limit, ok := node.(LimitNode)
		if !ok || limit.LimitCore() == nil {
			return outer, false
		}
		outer.Limit = limit
	}
	if outer.Returning != nil {
		if !nodePresent(outer.Returning) {
			return outer, false
		}
		node, ok := outer.Returning.Accept(v)
		if !ok {
			return outer, false
		}
		returning, ok := node.(*FieldList)
		if !ok || returning == nil {
			return outer, false
		}
		outer.Returning = returning
	}
	return v.Leave(outer)
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
type ShowStmt struct {
	dmlNode

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
			ctx.WriteKeyword("FULL ")
		}
	}
	restoreShowLikeOrWhereOpt := func() error {
		if n.Pattern != nil && n.Pattern.Pattern != nil {
			ctx.WriteKeyword(" LIKE ")
			if err := n.Pattern.Pattern.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore ShowStmt.Pattern")
			}
		} else if n.Where != nil {
			ctx.WriteKeyword(" WHERE ")
			if err := n.Where.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore ShowStmt.Where")
			}
		}
		return nil
	}

	ctx.WriteKeyword("SHOW ")
	switch n.Tp {
	case ShowCreateTable:
		ctx.WriteKeyword("CREATE TABLE ")
		if err := n.Table.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore ShowStmt.Table")
		}
	case ShowCreateView:
		ctx.WriteKeyword("CREATE VIEW ")
		if err := n.Table.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore ShowStmt.VIEW")
		}
	case ShowCreateDatabase:
		ctx.WriteKeyword("CREATE DATABASE ")
		if n.IfNotExists {
			ctx.WriteKeyword("IF NOT EXISTS ")
		}
		ctx.WriteName(n.DBName)
	case ShowCreateUser:
		ctx.WriteKeyword("CREATE USER ")
		if err := n.User.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore ShowStmt.User")
		}
	case ShowGrants:
		ctx.WriteKeyword("GRANTS")
		if n.User != nil {
			ctx.WriteKeyword(" FOR ")
			if err := n.User.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore ShowStmt.User")
			}
		}
		if n.Roles != nil {
			ctx.WriteKeyword(" USING ")
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
		ctx.WriteKeyword("MASTER STATUS")
	case ShowProcessList:
		restoreOptFull()
		ctx.WriteKeyword("PROCESSLIST")
	case ShowStatsMeta:
		ctx.WriteKeyword("STATS_META")
		if err := restoreShowLikeOrWhereOpt(); err != nil {
			return err
		}
	case ShowStatsHistograms:
		ctx.WriteKeyword("STATS_HISTOGRAMS")
		if err := restoreShowLikeOrWhereOpt(); err != nil {
			return err
		}
	case ShowStatsBuckets:
		ctx.WriteKeyword("STATS_BUCKETS")
		if err := restoreShowLikeOrWhereOpt(); err != nil {
			return err
		}
	case ShowStatsHealthy:
		ctx.WriteKeyword("STATS_HEALTHY")
		if err := restoreShowLikeOrWhereOpt(); err != nil {
			return err
		}
	case ShowProfiles:
		ctx.WriteKeyword("PROFILES")
	case ShowProfile:
		ctx.WriteKeyword("PROFILE")
		if len(n.ShowProfileTypes) > 0 {
			for i, tp := range n.ShowProfileTypes {
				if i != 0 {
					ctx.WritePlain(",")
				}
				ctx.WritePlain(" ")
				switch tp {
				case ProfileTypeCPU:
					ctx.WriteKeyword("CPU")
				case ProfileTypeMemory:
					ctx.WriteKeyword("MEMORY")
				case ProfileTypeBlockIo:
					ctx.WriteKeyword("BLOCK IO")
				case ProfileTypeContextSwitch:
					ctx.WriteKeyword("CONTEXT SWITCHES")
				case ProfileTypeIpc:
					ctx.WriteKeyword("IPC")
				case ProfileTypePageFaults:
					ctx.WriteKeyword("PAGE FAULTS")
				case ProfileTypeSource:
					ctx.WriteKeyword("SOURCE")
				case ProfileTypeSwaps:
					ctx.WriteKeyword("SWAPS")
				case ProfileTypeAll:
					ctx.WriteKeyword("ALL")
				}
			}
		}
		if n.ShowProfileArgs != nil {
			ctx.WriteKeyword(" FOR QUERY ")
			ctx.WritePlainf("%d", *n.ShowProfileArgs)
		}
		if n.ShowProfileLimit != nil {
			ctx.WritePlain(" ")
			if err := n.ShowProfileLimit.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore ShowStmt.WritePlain")
			}
		}
	case ShowPrivileges:
		ctx.WriteKeyword("PRIVILEGES")
	// ShowTargetFilterable
	default:
		if n.DBName == "" {
			return errors.New("SHOW statement has no variable name")
		}
		// PostgreSQL SHOW always names a GUC, so restoring the variable
		// keeps the statement re-parseable and semantically identical.
		ctx.WriteName(n.DBName)
		if err := restoreShowLikeOrWhereOpt(); err != nil {
			return err
		}
	}
	return ctx.Err()
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

// WindowSpecNode preserves the concrete dialect window identity.
type WindowSpecNode = pkast.WindowSpecNode

// WindowSpec is the specification of a window.
type WindowSpec struct {
	pkast.BaseWindowSpec

	PartitionBy *PartitionByClause
	OrderBy     *OrderByClause
	Frame       *FrameClause
}

// WindowSpecCore returns the shared window data without replacing this outer.
func (n *WindowSpec) WindowSpecCore() *pkast.BaseWindowSpec {
	if n == nil {
		return nil
	}
	return &n.BaseWindowSpec
}

// Restore implements Node interface.
func (n *WindowSpec) Restore(ctx *format.RestoreCtx) error {
	if n.Name.Present() {
		ctx.WriteIdent(n.Name)
		if n.OnlyAlias {
			return ctx.Err()
		}
		ctx.WriteKeyword(" AS ")
	}
	ctx.WritePlain("(")
	sep := ""
	if n.Ref.Present() {
		ctx.WriteIdent(n.Ref)
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

	return ctx.Err()
}

// Format formats the WindowSpec into a Writer.
func (n *WindowSpec) Format(w io.Writer) {
	if n.Name.Present() {
		name := n.Name.Name
		_, _ = fmt.Fprint(w, name)
		if n.OnlyAlias {
			return
		}
		_, _ = fmt.Fprint(w, " AS ")
	}
	_, _ = fmt.Fprint(w, "(")
	sep := ""
	if n.Ref.Present() {
		refName := n.Ref.Name
		_, _ = fmt.Fprint(w, refName)
		sep = " "
	}
	if n.PartitionBy != nil {
		_, _ = fmt.Fprint(w, sep)
		n.PartitionBy.Format(w)
		sep = " "
	}
	if n.OrderBy != nil {
		_, _ = fmt.Fprint(w, sep)
		n.OrderBy.Format(w)
		sep = " "
	}
	if n.Frame != nil {
		_, _ = fmt.Fprint(w, sep)
		n.Frame.Format(w)
	}
	_, _ = fmt.Fprint(w, ")")
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

	Items []*OrderItem
}

// Restore implements Node interface.
func (n *PartitionByClause) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("PARTITION BY ")
	for i, v := range n.Items {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore PartitionByClause.Items[%d]", i)
		}
	}
	return ctx.Err()
}

// Format formats the PartitionByClause into a Writer.
func (n *PartitionByClause) Format(w io.Writer) {
	_, _ = fmt.Fprint(w, "PARTITION BY ")
	for i, v := range n.Items {
		if i != 0 {
			_, _ = fmt.Fprint(w, ", ")
		}
		v.Format(w)
	}
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
		n.Items[i] = node.(*OrderItem)
	}
	return v.Leave(n)
}

// FrameType is the type of window function frame.
type FrameType = pkast.WindowFrameType

// Window function frame types: ROWS, RANGE and GROUPS.
const (
	Rows   = pkast.WindowFrameRows
	Ranges = pkast.WindowFrameRange
	Groups = pkast.WindowFrameGroups
)

// FrameClause represents frame clause.
type FrameClause struct {
	pkast.BaseWindowFrameClause
}

// Restore implements Node interface.
func (n *FrameClause) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore FrameClause: nil frame")
	}
	if n.Extent.Start == nil || n.Extent.Start.WindowFrameBoundCore() == nil ||
		n.Extent.End == nil || n.Extent.End.WindowFrameBoundCore() == nil {
		return errors.New("window frame requires start and end bounds")
	}
	switch n.Type {
	case Rows:
		ctx.WriteKeyword("ROWS")
	case Ranges:
		ctx.WriteKeyword("RANGE")
	case Groups:
		ctx.WriteKeyword("GROUPS")
	default:
		return errors.New("invalid window frame type")
	}
	ctx.WriteKeyword(" BETWEEN ")
	if err := n.Extent.Start.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore FrameClause.Extent.Start")
	}
	ctx.WriteKeyword(" AND ")
	if err := n.Extent.End.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore FrameClause.Extent.End")
	}
	switch n.Exclusion {
	case pkast.WindowFrameExcludeNone:
	case pkast.WindowFrameExcludeCurrentRow:
		ctx.WriteKeyword(" EXCLUDE CURRENT ROW")
	case pkast.WindowFrameExcludeGroup:
		ctx.WriteKeyword(" EXCLUDE GROUP")
	case pkast.WindowFrameExcludeTies:
		ctx.WriteKeyword(" EXCLUDE TIES")
	case pkast.WindowFrameExcludeNoOthers:
		ctx.WriteKeyword(" EXCLUDE NO OTHERS")
	default:
		return errors.New("invalid window frame exclusion")
	}

	return ctx.Err()
}

// Format formats the FrameClause into a Writer.
func (n *FrameClause) Format(w io.Writer) {
	switch n.Type {
	case Rows:
		_, _ = fmt.Fprint(w, "ROWS")
	case Ranges:
		_, _ = fmt.Fprint(w, "RANGE")
	case Groups:
		_, _ = fmt.Fprint(w, "GROUPS")
	}
	_, _ = fmt.Fprint(w, " BETWEEN ")
	formatFrameBound(w, n.Extent.Start)
	_, _ = fmt.Fprint(w, " AND ")
	formatFrameBound(w, n.Extent.End)
	switch n.Exclusion {
	case pkast.WindowFrameExcludeNone:
	case pkast.WindowFrameExcludeCurrentRow:
		_, _ = fmt.Fprint(w, " EXCLUDE CURRENT ROW")
	case pkast.WindowFrameExcludeGroup:
		_, _ = fmt.Fprint(w, " EXCLUDE GROUP")
	case pkast.WindowFrameExcludeTies:
		_, _ = fmt.Fprint(w, " EXCLUDE TIES")
	case pkast.WindowFrameExcludeNoOthers:
		_, _ = fmt.Fprint(w, " EXCLUDE NO OTHERS")
	}
}

// Accept implements Node Accept interface.
func (n *FrameClause) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*FrameClause)
	if n.Extent.Start != nil {
		node, ok := n.Extent.Start.Accept(v)
		if !ok {
			return n, false
		}
		bound, ok := node.(pkast.WindowFrameBoundNode)
		if !ok || bound == nil || bound.WindowFrameBoundCore() == nil {
			return n, false
		}
		n.Extent.Start = bound
	}
	if n.Extent.End != nil {
		node, ok := n.Extent.End.Accept(v)
		if !ok {
			return n, false
		}
		bound, ok := node.(pkast.WindowFrameBoundNode)
		if !ok || bound == nil || bound.WindowFrameBoundCore() == nil {
			return n, false
		}
		n.Extent.End = bound
	}
	return v.Leave(n)
}

// FrameExtent represents frame extent.
type FrameExtent = pkast.WindowFrameExtent

func formatFrameBound(w io.Writer, bound pkast.WindowFrameBoundNode) {
	if b, ok := bound.(*FrameBound); ok {
		b.Format(w)
	}
}

// BoundType is the type of window function frame bound.
type BoundType = pkast.WindowFrameBoundType

// Frame bound types.
const (
	Following  = pkast.WindowFrameFollowing
	Preceding  = pkast.WindowFramePreceding
	CurrentRow = pkast.WindowFrameCurrentRow
)

// FrameBound represents frame bound.
type FrameBound struct {
	pkast.BaseWindowFrameBound
}

// WindowFrameBoundCore reports a missing bound before role-based dispatch.
func (n *FrameBound) WindowFrameBoundCore() *pkast.BaseWindowFrameBound {
	if n == nil {
		return nil
	}
	return &n.BaseWindowFrameBound
}

// Restore implements Node interface.
func (n *FrameBound) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore FrameBound: nil bound")
	}
	if n.Type != CurrentRow && n.Type != Preceding && n.Type != Following {
		return errors.New("invalid window frame bound type")
	}
	if n.Unbounded {
		ctx.WriteKeyword("UNBOUNDED")
	}
	switch n.Type {
	case CurrentRow:
		ctx.WriteKeyword("CURRENT ROW")
	case Preceding, Following:
		if n.Unit.Present() {
			ctx.WriteKeyword("INTERVAL ")
		}
		if n.Expr != nil {
			if err := n.Expr.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore FrameBound.Expr")
			}
		}
		if n.Unit.Present() {
			ctx.WritePlain(" ")
			ctx.WriteKeyword(n.Unit.Name)
		}
		if n.Type == Preceding {
			ctx.WriteKeyword(" PRECEDING")
		} else {
			ctx.WriteKeyword(" FOLLOWING")
		}
	}
	return ctx.Err()
}

// Format formats the FrameBound into a Writer.
func (n *FrameBound) Format(w io.Writer) {
	if n.Unbounded {
		_, _ = fmt.Fprint(w, "UNBOUNDED ")
	}
	switch n.Type {
	case CurrentRow:
		_, _ = fmt.Fprint(w, "CURRENT ROW")
	case Preceding, Following:
		if n.Unit.Present() {
			_, _ = fmt.Fprint(w, "INTERVAL ")
		}
		if n.Expr != nil {
			FormatExpr(n.Expr, w)
		}
		if n.Unit.Present() {
			_, _ = fmt.Fprint(w, " ")
			_, _ = fmt.Fprint(w, n.Unit.Name)
		}
		if n.Type == Preceding {
			_, _ = fmt.Fprint(w, " PRECEDING")
		} else {
			_, _ = fmt.Fprint(w, " FOLLOWING")
		}
	}
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
	return v.Leave(n)
}

func restoreExprSets(ctx *format.RestoreCtx, sets [][]ExprNode) error {
	for i, set := range sets {
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
	return ctx.Err()
}
