package ast

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/mia0x75/parserkit/lex"

	"github.com/mia0x75/pgparser/errors"

	"github.com/mia0x75/pgparser/auth"
	"github.com/mia0x75/pgparser/driver"
	"github.com/mia0x75/pgparser/format"
	"github.com/mia0x75/pgparser/model"
	"github.com/mia0x75/pgparser/postgres"
	"github.com/mia0x75/pgparser/types"
)

var (
	_ StmtNode = &AlterUserStmt{}
	_ StmtNode = &BeginStmt{}
	_ StmtNode = &BinlogStmt{}
	_ StmtNode = &CommitStmt{}
	_ StmtNode = &CreateUserStmt{}
	_ StmtNode = &DeallocateStmt{}
	_ StmtNode = &DoStmt{}
	_ StmtNode = &ExecuteStmt{}
	_ StmtNode = &ExplainStmt{}
	_ StmtNode = &GrantStmt{}
	_ StmtNode = &GrantRoleStmt{}
	_ StmtNode = &PrepareStmt{}
	_ StmtNode = &RollbackStmt{}
	_ StmtNode = &SavepointStmt{}
	_ StmtNode = &ReleaseSavepointStmt{}
	_ StmtNode = &XAStmt{}
	_ StmtNode = &SetPwdStmt{}
	_ StmtNode = &SetRoleStmt{}
	_ StmtNode = &SetDefaultRoleStmt{}
	_ StmtNode = &SetStmt{}
	_ StmtNode = &UseStmt{}
	_ StmtNode = &PurgeStmt{}
	_ StmtNode = &FlushStmt{}
	_ StmtNode = &KillStmt{}
	_ StmtNode = &LockTableStmt{}
	_ StmtNode = &UnlockTableStmt{}

	_ Node = &PrivElem{}
	_ Node = &VariableAssignment{}
)

// Isolation level constants.
const (
	ReadCommitted   = "READ-COMMITTED"
	ReadUncommitted = "READ-UNCOMMITTED"
	Serializable    = "SERIALIZABLE"
	RepeatableRead  = "REPEATABLE-READ"

	// Valid formats for explain statement.
	ExplainFormatROW = "row"
	ExplainFormatDOT = "dot"
)

var (
	// ExplainFormats stores the valid formats for explain statement, used by validator.
	ExplainFormats = []string{
		ExplainFormatROW,
		ExplainFormatDOT,
	}
)

// TypeOpt is used for parsing data type option from SQL.
type TypeOpt struct {
	IsUnsigned bool
	IsZerofill bool
}

// FloatOpt is used for parsing floating-point type option from SQL.
type FloatOpt struct {
	Flen    int
	Decimal int
}

// AuthOption is used for parsing create use statement.
type AuthOption struct {
	// ByAuthString set as true, if AuthString is used for authorization. Otherwise, authorization is done by HashString.
	ByAuthString bool
	AuthString   string
	HashString   string
	// Password indicates PostgreSQL PASSWORD syntax instead of IDENTIFIED BY.
	Password bool
	// TODO: support auth_plugin
}

// Restore implements Node interface.
func (n *AuthOption) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("PASSWORD ")
	if n.ByAuthString {
		ctx.WriteString(n.AuthString)
	} else {
		ctx.WriteString(n.HashString)
	}
	return ctx.Err()
}

// ExplainForStmt is a statement to provite information about how is SQL statement executeing
// in connection #ConnectionID
type ExplainForStmt struct {
	stmtNode

	Format       string
	ConnectionID uint64
}

// Restore implements Node interface.
func (n *ExplainForStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("EXPLAIN ")
	ctx.WriteKeyword("FORMAT ")
	ctx.WritePlain("= ")
	ctx.WriteString(n.Format)
	ctx.WritePlain(" ")
	ctx.WriteKeyword("FOR ")
	ctx.WriteKeyword("CONNECTION ")
	ctx.WritePlain(strconv.FormatUint(n.ConnectionID, 10))
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ExplainForStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*ExplainForStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*ExplainForStmt)
	return v.Leave(n)
}

// ExplainStmt is a statement to provide information about how is SQL statement executed
// or get columns information in a table.
type ExplainStmt struct {
	stmtNode

	Stmt    StmtNode
	Format  string
	Analyze bool
	Options []string
}

// Restore implements Node interface.
func (n *ExplainStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	if showStmt, ok := n.Stmt.(*ShowStmt); ok {
		ctx.WriteKeyword("DESC ")
		if err := showStmt.Table.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore ExplainStmt.ShowStmt.Table")
		}
		if showStmt.Column != nil {
			ctx.WritePlain(" ")
			if err := showStmt.Column.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore ExplainStmt.ShowStmt.Column")
			}
		}
		return ctx.Err()
	}
	ctx.WriteKeyword("EXPLAIN ")
	// `EXPLAIN ANALYZE` sets both the flag and the option; write it once.
	analyzeInOptions := false
	for _, opt := range n.Options {
		if strings.EqualFold(opt, "analyze") {
			analyzeInOptions = true
			break
		}
	}
	if n.Analyze && !analyzeInOptions {
		ctx.WriteKeyword("ANALYZE ")
	}
	if len(n.Options) > 0 {
		ctx.WritePlain("(")
		for i, opt := range n.Options {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			ctx.WriteKeyword(opt)
		}
		ctx.WritePlain(") ")
	} else if !n.Analyze && n.Format != "" {
		ctx.WriteKeyword("FORMAT ")
		ctx.WritePlain("= ")
		ctx.WriteString(n.Format)
		ctx.WritePlain(" ")
	}
	if err := n.Stmt.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore ExplainStmt.Stmt")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ExplainStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*ExplainStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*ExplainStmt)
	node, ok := n.Stmt.Accept(v)
	if !ok {
		return n, false
	}
	if stmt, ok := node.(StmtNode); ok {
		n.Stmt = stmt
	}
	return v.Leave(n)
}

// PrepareStmt is a statement to prepares a SQL statement which contains placeholders,
// and it is executed with ExecuteStmt and released with DeallocateStmt.
type PrepareStmt struct {
	stmtNode

	Name    lex.Ident
	Params  []*types.FieldType
	Stmt    StmtNode
	SQLText string
	SQLVar  *VariableExpr
}

// Restore implements Node interface.
func (n *PrepareStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("PREPARE ")
	ctx.WriteIdent(n.Name)
	if len(n.Params) > 0 {
		ctx.WritePlain(" (")
		for i, tp := range n.Params {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			if err := tp.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore PrepareStmt.Params[%d]", i)
			}
		}
		ctx.WritePlain(")")
	}
	ctx.WriteKeyword(" AS ")
	if n.Stmt != nil {
		return n.Stmt.Restore(ctx)
	}
	if n.SQLText != "" {
		ctx.WriteString(n.SQLText)
		return ctx.Err()
	}
	if n.SQLVar != nil {
		if err := n.SQLVar.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore PrepareStmt.SQLVar")
		}
		return ctx.Err()
	}
	return errors.New("An error occurred while restore PrepareStmt")
}

// Accept implements Node Accept interface.
func (n *PrepareStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*PrepareStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*PrepareStmt)
	if n.Stmt != nil {
		node, ok := n.Stmt.Accept(v)
		if !ok {
			return n, false
		}
		n.Stmt = node.(StmtNode)
	}
	if n.SQLVar != nil {
		node, ok := n.SQLVar.Accept(v)
		if !ok {
			return n, false
		}
		n.SQLVar = node.(*VariableExpr)
	}
	return v.Leave(n)
}

// DeallocateStmt is a statement to release PreparedStmt.
type DeallocateStmt struct {
	stmtNode

	Name lex.Ident
	All  bool
}

// Restore implements Node interface.
func (n *DeallocateStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("DEALLOCATE PREPARE ")
	if n.All {
		ctx.WriteKeyword("ALL")
		return ctx.Err()
	}
	ctx.WriteIdent(n.Name)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DeallocateStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*DeallocateStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*DeallocateStmt)
	return v.Leave(n)
}

// Prepared represents a prepared statement.
type Prepared struct {
	Stmt          StmtNode
	Params        []ParamMarkerExpr
	SchemaVersion int64
	UseCache      bool
}

// ExecuteStmt is a statement to execute PreparedStmt.
type ExecuteStmt struct {
	stmtNode

	Name      lex.Ident
	Args      []ExprNode
	UsingVars []ExprNode
	ExecID    uint32
}

// Restore implements Node interface.
func (n *ExecuteStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("EXECUTE ")
	ctx.WriteIdent(n.Name)
	if len(n.Args) > 0 {
		ctx.WritePlain(" (")
		for i, val := range n.Args {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			if err := val.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore ExecuteStmt.Args index %d", i)
			}
		}
		ctx.WritePlain(")")
	}
	if len(n.UsingVars) > 0 {
		ctx.WriteKeyword(" USING ")
		for i, val := range n.UsingVars {
			if i != 0 {
				ctx.WritePlain(",")
			}
			if err := val.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore ExecuteStmt.UsingVars index %d", i)
			}
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ExecuteStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*ExecuteStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*ExecuteStmt)
	for i, val := range n.Args {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Args[i] = node.(ExprNode)
	}
	for i, val := range n.UsingVars {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.UsingVars[i] = node.(ExprNode)
	}
	return v.Leave(n)
}

// BeginStmt is a statement to start a new transaction.
type BeginStmt struct {
	stmtNode

	IsolationLevel string
	ReadOnly       *bool
	Deferrable     *bool
}

// Restore implements Node interface.
func (n *BeginStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("START TRANSACTION")
	if n.IsolationLevel != "" {
		ctx.WriteKeyword(" ISOLATION LEVEL ")
		ctx.WriteKeyword(n.IsolationLevel)
	}
	if n.ReadOnly != nil {
		if *n.ReadOnly {
			ctx.WriteKeyword(" READ ONLY")
		} else {
			ctx.WriteKeyword(" READ WRITE")
		}
	}
	if n.Deferrable != nil {
		if *n.Deferrable {
			ctx.WriteKeyword(" DEFERRABLE")
		} else {
			ctx.WriteKeyword(" NOT DEFERRABLE")
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *BeginStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*BeginStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*BeginStmt)
	return v.Leave(n)
}

// BinlogStmt is an internal-use statement.
// We just parse and ignore it.
type BinlogStmt struct {
	stmtNode
	Str string
}

// Restore implements Node interface.
func (n *BinlogStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("BINLOG ")
	ctx.WriteString(n.Str)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *BinlogStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*BinlogStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*BinlogStmt)
	return v.Leave(n)
}

// CommitStmt is a statement to commit the current transaction.
type CommitStmt struct {
	stmtNode
}

// Restore implements Node interface.
func (n *CommitStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("COMMIT")
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CommitStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*CommitStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*CommitStmt)
	return v.Leave(n)
}

// RollbackStmt is a statement to roll back the current transaction.
type RollbackStmt struct {
	stmtNode

	// SavepointName is the target savepoint of `ROLLBACK TO [SAVEPOINT] name`.
	// It is empty for a full transaction rollback.
	SavepointName string
}

// Restore implements Node interface.
func (n *RollbackStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("ROLLBACK")
	if n.SavepointName != "" {
		ctx.WriteKeyword(" TO SAVEPOINT ")
		ctx.WriteName(n.SavepointName)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *RollbackStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*RollbackStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*RollbackStmt)
	return v.Leave(n)
}

// SavepointStmt is a `SAVEPOINT identifier` statement.
type SavepointStmt struct {
	stmtNode

	Name string
}

// Restore implements Node interface.
func (n *SavepointStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("SAVEPOINT ")
	ctx.WriteName(n.Name)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *SavepointStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*SavepointStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*SavepointStmt)
	return v.Leave(n)
}

// ReleaseSavepointStmt is a `RELEASE SAVEPOINT identifier` statement.
type ReleaseSavepointStmt struct {
	stmtNode

	Name string
}

// Restore implements Node interface.
func (n *ReleaseSavepointStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("RELEASE SAVEPOINT ")
	ctx.WriteName(n.Name)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ReleaseSavepointStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*ReleaseSavepointStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*ReleaseSavepointStmt)
	return v.Leave(n)
}

// XAStmt is a statement to execute an XA transaction command.
type XAStmt struct {
	stmtNode

	// Command is one of START, BEGIN, END, PREPARE, COMMIT, ROLLBACK, RECOVER.
	Command string
	// XID is the literal transaction identifier. It is empty for RECOVER.
	XID string
	// Extra holds trailing keywords such as JOIN, RESUME, SUSPEND, FOR MIGRATE
	// or ONE PHASE.
	Extra string
}

// Restore implements Node interface.
func (n *XAStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("XA ")
	ctx.WriteKeyword(n.Command)
	if n.XID != "" {
		ctx.WritePlain(" ")
		ctx.WriteString(n.XID)
	}
	if n.Extra != "" {
		ctx.WritePlain(" ")
		ctx.WriteKeyword(n.Extra)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *XAStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*XAStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*XAStmt)
	return v.Leave(n)
}

// UseStmt is a statement to use the DBName database as the current database.
type UseStmt struct {
	stmtNode

	DBName string
}

// Restore implements Node interface.
func (n *UseStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("USE ")
	ctx.WriteName(n.DBName)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *UseStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*UseStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*UseStmt)
	return v.Leave(n)
}

const (
	// SetNames is the const for set names/charset stmt.
	// If VariableAssignment.Name == Names, it should be set names/charset stmt.
	SetNames = "SetNAMES"
)

// VariableAssignment is a variable assignment struct.
type VariableAssignment struct {
	node
	Name     string
	Value    ExprNode
	IsGlobal bool
	IsSystem bool

	// ExtendValue is a way to store extended info.
	// VariableAssignment should be able to store information for SetCharset/SetPWD Stmt.
	// For SetCharsetStmt, Value is charset, ExtendValue is collation.
	// TODO: Use SetStmt to implement set password statement.
	ExtendValue ValueExpr
}

// Restore implements Node interface.
func (n *VariableAssignment) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	if n.IsSystem {
		ctx.WritePlain("@@")
		if n.IsGlobal {
			ctx.WriteKeyword("GLOBAL")
		} else {
			ctx.WriteKeyword("SESSION")
		}
		ctx.WritePlain(".")
	} else if n.Name != SetNames {
		ctx.WriteKeyword("@")
	}
	if n.Name == SetNames {
		ctx.WriteKeyword("NAMES ")
	} else {
		ctx.WriteName(n.Name)
		ctx.WritePlain("=")
	}
	if err := n.Value.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore VariableAssignment.Value")
	}
	if n.ExtendValue != nil {
		ctx.WriteKeyword(" COLLATE ")
		if err := n.ExtendValue.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore VariableAssignment.ExtendValue")
		}
	}
	return ctx.Err()
}

// Accept implements Node interface.
func (n *VariableAssignment) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*VariableAssignment)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*VariableAssignment)
	node, ok := n.Value.Accept(v)
	if !ok {
		return n, false
	}
	n.Value = node.(ExprNode)
	return v.Leave(n)
}

// FlushStmtType is the type for FLUSH statement.
type FlushStmtType int

// Flush statement types.
const (
	FlushNone FlushStmtType = iota
	FlushTables
	FlushPrivileges
	FlushStatus
)

// FlushStmt is a statement to flush tables/privileges/optimizer costs and so on.
type FlushStmt struct {
	stmtNode

	Tp              FlushStmtType // Privileges/Tables/...
	NoWriteToBinLog bool
	Tables          []*TableName // For FlushTableStmt, if Tables is empty, it means flush all tables.
	ReadLock        bool
}

// Restore implements Node interface.
func (n *FlushStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("FLUSH ")
	if n.NoWriteToBinLog {
		ctx.WriteKeyword("NO_WRITE_TO_BINLOG ")
	}
	switch n.Tp {
	case FlushTables:
		ctx.WriteKeyword("TABLES")
		for i, v := range n.Tables {
			if i == 0 {
				ctx.WritePlain(" ")
			} else {
				ctx.WritePlain(", ")
			}
			if err := v.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore FlushStmt.Tables[%d]", i)
			}
		}
		if n.ReadLock {
			ctx.WriteKeyword(" WITH READ LOCK")
		}
	case FlushPrivileges:
		ctx.WriteKeyword("PRIVILEGES")
	case FlushStatus:
		ctx.WriteKeyword("STATUS")
	default:
		return errors.New("Unsupported type of FlushTables")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *FlushStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*FlushStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*FlushStmt)
	return v.Leave(n)
}

// KillStmt is a statement to kill a query or connection.
type KillStmt struct {
	stmtNode

	// Query indicates whether terminate a single query on this connection or the whole connection.
	// If Query is true, terminates the statement the connection is currently executing, but leaves the connection itself intact.
	// If Query is false, terminates the connection associated with the given ConnectionID, after terminating any statement the connection is executing.
	Query        bool
	ConnectionID uint64
}

// Restore implements Node interface.
func (n *KillStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("KILL")
	if n.Query {
		ctx.WriteKeyword(" QUERY")
	}
	ctx.WritePlainf(" %d", n.ConnectionID)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *KillStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*KillStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*KillStmt)
	return v.Leave(n)
}

// SetStmt is the statement to set variables.
type SetStmt struct {
	stmtNode
	// Variables is the list of variable assignment.
	Variables []*VariableAssignment
}

// Restore implements Node interface.
func (n *SetStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("SET ")
	for i, v := range n.Variables {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore SetStmt.Variables[%d]", i)
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *SetStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*SetStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*SetStmt)
	for i, val := range n.Variables {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Variables[i] = node.(*VariableAssignment)
	}
	return v.Leave(n)
}

/*
// SetCharsetStmt is a statement to assign values to character and collation variables.
type SetCharsetStmt struct {
	stmtNode

	Charset string
	Collate string
}

// Accept implements Node Accept interface.
func (n *SetCharsetStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*SetCharsetStmt)
	return v.Leave(n)
}
*/

// SetPwdStmt is a statement to assign a password to user account.
type SetPwdStmt struct {
	stmtNode

	User     *auth.UserIdentity
	Password string
}

// Restore implements Node interface.
func (n *SetPwdStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("SET PASSWORD")
	if n.User != nil {
		ctx.WriteKeyword(" FOR ")
		if err := n.User.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SetPwdStmt.User")
		}
	}
	ctx.WritePlain("=")
	ctx.WriteString(n.Password)
	return ctx.Err()
}

// SecureText implements SensitiveStatement interface.
func (n *SetPwdStmt) SecureText() string {
	return fmt.Sprintf("set password for user %s", n.User)
}

// Accept implements Node Accept interface.
func (n *SetPwdStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*SetPwdStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*SetPwdStmt)
	return v.Leave(n)
}

// SetRoleStmtType is the type for FLUSH statement.
type SetRoleStmtType int

// SetRole statement types.
const (
	SetRoleDefault SetRoleStmtType = iota
	SetRoleNone
	SetRoleAll
	SetRoleAllExcept
	SetRoleRegular
)

// SetRoleStmt TODO:
type SetRoleStmt struct {
	stmtNode

	SetRoleOpt SetRoleStmtType
	RoleList   []*auth.RoleIdentity
}

// Restore implements Node interface.
func (n *SetRoleStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("SET ROLE")
	switch n.SetRoleOpt {
	case SetRoleDefault:
		ctx.WriteKeyword(" DEFAULT")
	case SetRoleNone:
		ctx.WriteKeyword(" NONE")
	case SetRoleAll:
		ctx.WriteKeyword(" ALL")
	case SetRoleAllExcept:
		ctx.WriteKeyword(" ALL EXCEPT")
	}
	for i, role := range n.RoleList {
		ctx.WritePlain(" ")
		err := role.Restore(ctx)
		if err != nil {
			return errors.Annotate(err, "An error occurred while restore SetRoleStmt.RoleList")
		}
		if i != len(n.RoleList)-1 {
			ctx.WritePlain(",")
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *SetRoleStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*SetRoleStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*SetRoleStmt)
	return v.Leave(n)
}

// SetDefaultRoleStmt TODO:
type SetDefaultRoleStmt struct {
	stmtNode

	SetRoleOpt SetRoleStmtType
	RoleList   []*auth.RoleIdentity
	UserList   []*auth.UserIdentity
}

// Restore implements Node interface.
func (n *SetDefaultRoleStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("SET DEFAULT ROLE")
	switch n.SetRoleOpt {
	case SetRoleNone:
		ctx.WriteKeyword(" NONE")
	case SetRoleAll:
		ctx.WriteKeyword(" ALL")
	default:
	}
	for i, role := range n.RoleList {
		ctx.WritePlain(" ")
		err := role.Restore(ctx)
		if err != nil {
			return errors.Annotate(err, "An error occurred while restore SetDefaultRoleStmt.RoleList")
		}
		if i != len(n.RoleList)-1 {
			ctx.WritePlain(",")
		}
	}
	ctx.WritePlain(" TO")
	for i, user := range n.UserList {
		ctx.WritePlain(" ")
		err := user.Restore(ctx)
		if err != nil {
			return errors.Annotate(err, "An error occurred while restore SetDefaultRoleStmt.UserList")
		}
		if i != len(n.UserList)-1 {
			ctx.WritePlain(",")
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *SetDefaultRoleStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*SetDefaultRoleStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*SetDefaultRoleStmt)
	return v.Leave(n)
}

// UserSpec is used for parsing create user statement.
type UserSpec struct {
	User    *auth.UserIdentity
	AuthOpt *AuthOption
	Role    *RoleSpec
	IsRole  bool
}

// Restore implements Node interface.
func (n *UserSpec) Restore(ctx *format.RestoreCtx) error {
	if n.Role != nil {
		if err := n.Role.Restore(ctx); err != nil {
			return err
		}
	} else {
		if err := n.User.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore UserSpec.User")
		}
	}
	if n.AuthOpt != nil {
		ctx.WritePlain(" ")
		if err := n.AuthOpt.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore UserSpec.AuthOpt")
		}
	}
	return ctx.Err()
}

// SecurityString formats the UserSpec without password information.
func (n *UserSpec) SecurityString() string {
	withPassword := false
	if opt := n.AuthOpt; opt != nil {
		if len(opt.AuthString) > 0 || len(opt.HashString) > 0 {
			withPassword = true
		}
	}
	if withPassword {
		return fmt.Sprintf("{%s password = ***}", n.User)
	}
	return n.User.String()
}

// TODO:
const (
	TslNone = iota
	Ssl
	X509
	Cipher
	Issuer
	Subject
)

// TslOption TODO:
type TslOption struct {
	Type  int
	Value string
}

// Restore implements Node interface.
func (t *TslOption) Restore(ctx *format.RestoreCtx) error {
	switch t.Type {
	case TslNone:
		ctx.WriteKeyword("NONE")
	case Ssl:
		ctx.WriteKeyword("SSL")
	case X509:
		ctx.WriteKeyword("X509")
	case Cipher:
		ctx.WriteKeyword("CIPHER ")
		ctx.WriteString(t.Value)
	case Issuer:
		ctx.WriteKeyword("ISSUER ")
		ctx.WriteString(t.Value)
	case Subject:
		ctx.WriteKeyword("CIPHER")
		ctx.WriteString(t.Value)
	default:
		return errors.Errorf("Unsupported TslOption.Type %d", t.Type)
	}
	return ctx.Err()
}

// TODO:
const (
	MaxQueriesPerHour = iota + 1
	MaxUpdatesPerHour
	MaxConnectionsPerHour
	MaxUserConnections
)

// ResourceOption TODO:
type ResourceOption struct {
	Type  int
	Count int64
}

// Restore implements Node interface.
func (r *ResourceOption) Restore(ctx *format.RestoreCtx) error {
	switch r.Type {
	case MaxQueriesPerHour:
		ctx.WriteKeyword("MAX_QUERIES_PER_HOUR ")
	case MaxUpdatesPerHour:
		ctx.WriteKeyword("MAX_UPDATES_PER_HOUR ")
	case MaxConnectionsPerHour:
		ctx.WriteKeyword("MAX_CONNECTIONS_PER_HOUR ")
	case MaxUserConnections:
		ctx.WriteKeyword("MAX_USER_CONNECTIONS ")
	default:
		return errors.Errorf("Unsupported ResourceOption.Type %d", r.Type)
	}
	ctx.WritePlainf("%d", r.Count)
	return ctx.Err()
}

// TODO:
const (
	PasswordExpire = iota + 1
	PasswordExpireDefault
	PasswordExpireNever
	PasswordExpireInterval
	Lock
	Unlock
)

// PasswordOrLockOption TODO:
type PasswordOrLockOption struct {
	Type  int
	Count int64
}

// Restore implements Node interface.
func (p *PasswordOrLockOption) Restore(ctx *format.RestoreCtx) error {
	switch p.Type {
	case PasswordExpire:
		ctx.WriteKeyword("PASSWORD EXPIRE")
	case PasswordExpireDefault:
		ctx.WriteKeyword("PASSWORD EXPIRE DEFAULT")
	case PasswordExpireNever:
		ctx.WriteKeyword("PASSWORD EXPIRE NEVER")
	case PasswordExpireInterval:
		ctx.WriteKeyword("PASSWORD EXPIRE NEVER")
		ctx.WritePlainf(" %d", p.Count)
		ctx.WriteKeyword(" DAY")
	case Lock:
		ctx.WriteKeyword("ACCOUNT LOCK")
	case Unlock:
		ctx.WriteKeyword("ACCOUNT UNLOCK")
	default:
		return errors.Errorf("Unsupported PasswordOrLockOption.Type %d", p.Type)
	}
	return ctx.Err()
}

// CreateUserStmt creates user account.
type CreateUserStmt struct {
	stmtNode

	IsCreateRole          bool
	IfNotExists           bool
	Specs                 []*UserSpec
	TslOptions            []*TslOption
	ResourceOptions       []*ResourceOption
	PasswordOrLockOptions []*PasswordOrLockOption
	RoleOptions           *List
	Role                  *RoleSpec
}

// Restore implements Node interface.
func (n *CreateUserStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	if n.Role != nil {
		return restoreRoleStatement(ctx, n.IsCreateRole, false, n.Role, n.RoleOptions)
	}
	if n.IsCreateRole {
		ctx.WriteKeyword("CREATE ROLE ")
	} else {
		ctx.WriteKeyword("CREATE USER ")
	}
	if n.IfNotExists {
		ctx.WriteKeyword("IF NOT EXISTS ")
	}
	for i, v := range n.Specs {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore CreateUserStmt.Specs[%d]", i)
		}
	}

	tslOptionLen := len(n.TslOptions)

	if tslOptionLen != 0 {
		ctx.WriteKeyword(" REQUIRE ")
	}

	// Restore `tslOptions` reversely to keep order the same with original sql
	for i := tslOptionLen; i > 0; i-- {
		if i != tslOptionLen {
			ctx.WriteKeyword(" AND ")
		}
		if err := n.TslOptions[i-1].Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore CreateUserStmt.TslOptions[%d]", i)
		}
	}

	if len(n.ResourceOptions) != 0 {
		ctx.WriteKeyword(" WITH")
	}

	for i, v := range n.ResourceOptions {
		ctx.WritePlain(" ")
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore CreateUserStmt.ResourceOptions[%d]", i)
		}
	}

	for i, v := range n.PasswordOrLockOptions {
		ctx.WritePlain(" ")
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore CreateUserStmt.PasswordOrLockOptions[%d]", i)
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CreateUserStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*CreateUserStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*CreateUserStmt)
	if n.Role != nil {
		node, ok := n.Role.Accept(v)
		if !ok {
			return n, false
		}
		n.Role = node.(*RoleSpec)
	}
	if n.RoleOptions != nil {
		node, ok := n.RoleOptions.Accept(v)
		if !ok {
			return n, false
		}
		n.RoleOptions = node.(*List)
	}
	return v.Leave(n)
}

// SecureText implements SensitiveStatement interface.
func (n *CreateUserStmt) SecureText() string {
	var buf bytes.Buffer
	buf.WriteString("create user")
	for _, user := range n.Specs {
		buf.WriteString(" ")
		buf.WriteString(user.SecurityString())
	}
	return buf.String()
}

// AlterUserStmt modifies user account.
type AlterUserStmt struct {
	stmtNode

	IfExists    bool
	IsAlterRole bool
	CurrentAuth *AuthOption
	Specs       []*UserSpec
	RoleOptions *List
	Role        *RoleSpec
}

// Restore implements Node interface.
func (n *AlterUserStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	if n.Role != nil {
		return restoreRoleStatement(ctx, n.IsAlterRole, true, n.Role, n.RoleOptions)
	}
	if n.IsAlterRole {
		ctx.WriteKeyword("ALTER ROLE ")
	} else {
		ctx.WriteKeyword("ALTER USER ")
	}
	if n.IfExists {
		ctx.WriteKeyword("IF EXISTS ")
	}
	if n.CurrentAuth != nil {
		ctx.WriteKeyword("USER")
		ctx.WritePlain("() ")
		if err := n.CurrentAuth.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore AlterUserStmt.CurrentAuth")
		}
	}
	for i, v := range n.Specs {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore AlterUserStmt.Specs[%d]", i)
		}
	}
	return ctx.Err()
}

// SecureText implements SensitiveStatement interface.
func (n *AlterUserStmt) SecureText() string {
	var buf bytes.Buffer
	buf.WriteString("alter user")
	for _, user := range n.Specs {
		buf.WriteString(" ")
		buf.WriteString(user.SecurityString())
	}
	return buf.String()
}

// Accept implements Node Accept interface.
func (n *AlterUserStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*AlterUserStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*AlterUserStmt)
	if n.Role != nil {
		node, ok := n.Role.Accept(v)
		if !ok {
			return n, false
		}
		n.Role = node.(*RoleSpec)
	}
	if n.RoleOptions != nil {
		node, ok := n.RoleOptions.Accept(v)
		if !ok {
			return n, false
		}
		n.RoleOptions = node.(*List)
	}
	return v.Leave(n)
}

// DropUserStmt creates user account.
type DropUserStmt struct {
	stmtNode

	IfExists   bool
	IsDropRole bool
	UserList   []*auth.UserIdentity
}

// Restore implements Node interface.
func (n *DropUserStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	if n.IsDropRole {
		ctx.WriteKeyword("DROP ROLE ")
	} else {
		ctx.WriteKeyword("DROP USER ")
	}
	if n.IfExists {
		ctx.WriteKeyword("IF EXISTS ")
	}
	for i, v := range n.UserList {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore DropUserStmt.UserList[%d]", i)
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DropUserStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*DropUserStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*DropUserStmt)
	return v.Leave(n)
}

// DoStmt is the struct for DO statement.
type DoStmt struct {
	stmtNode

	// Exprs holds the code literal.
	Exprs []ExprNode
	// Language is the optional LANGUAGE clause; empty means the default.
	Language string
}

// Restore implements Node interface.
func (n *DoStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	if len(n.Exprs) == 0 {
		return errors.New("DO statement has no code")
	}
	ctx.WriteKeyword("DO ")
	if err := n.Exprs[0].Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore DoStmt.Exprs[0]")
	}
	if n.Language != "" {
		ctx.WriteKeyword(" LANGUAGE ")
		ctx.WriteName(n.Language)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DoStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*DoStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*DoStmt)
	for i, val := range n.Exprs {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Exprs[i] = node.(ExprNode)
	}
	return v.Leave(n)
}

// HandleRange represents a range where handle value >= Begin and < End.
type HandleRange struct {
	Begin int64
	End   int64
}

// PrivElem is the privilege type and optional column list.
type PrivElem struct {
	node

	Priv postgres.PrivilegeType
	Cols []*ColumnName
	Name string
}

// Restore implements Node interface.
func (n *PrivElem) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	if n.Name != "" {
		switch strings.ToLower(n.Name) {
		case "all", "select", "insert", "update", "delete", "truncate", "references", "trigger", "maintain",
			"usage", "create", "connect", "temporary", "temp", "execute", "set", "alter system":
			ctx.WriteKeyword(n.Name)
		default:
			return errors.New("unsupported privilege name")
		}
	} else {
		switch n.Priv {
		case 0:
			return errors.New("unsupported privilege type")
		case postgres.AllPriv:
			ctx.WriteKeyword("ALL")
		case postgres.AlterPriv:
			ctx.WriteKeyword("ALTER")
		case postgres.CreatePriv:
			ctx.WriteKeyword("CREATE")
		case postgres.CreateUserPriv:
			ctx.WriteKeyword("CREATE USER")
		case postgres.CreateRolePriv:
			ctx.WriteKeyword("CREATE ROLE")
		case postgres.TriggerPriv:
			ctx.WriteKeyword("TRIGGER")
		case postgres.DeletePriv:
			ctx.WriteKeyword("DELETE")
		case postgres.DropPriv:
			ctx.WriteKeyword("DROP")
		case postgres.ProcessPriv:
			ctx.WriteKeyword("PROCESS")
		case postgres.ExecutePriv:
			ctx.WriteKeyword("EXECUTE")
		case postgres.IndexPriv:
			ctx.WriteKeyword("INDEX")
		case postgres.InsertPriv:
			ctx.WriteKeyword("INSERT")
		case postgres.SelectPriv:
			ctx.WriteKeyword("SELECT")
		case postgres.SuperPriv:
			ctx.WriteKeyword("SUPER")
		case postgres.ShowDBPriv:
			ctx.WriteKeyword("SHOW DATABASES")
		case postgres.UpdatePriv:
			ctx.WriteKeyword("UPDATE")
		case postgres.GrantPriv:
			ctx.WriteKeyword("GRANT OPTION")
		case postgres.ReferencesPriv:
			ctx.WriteKeyword("REFERENCES")
		case postgres.CreateViewPriv:
			ctx.WriteKeyword("CREATE VIEW")
		case postgres.ShowViewPriv:
			ctx.WriteKeyword("SHOW VIEW")
		case postgres.UsagePriv:
			ctx.WriteKeyword("USAGE")
		case postgres.ConnectPriv:
			ctx.WriteKeyword("CONNECT")
		case postgres.TemporaryPriv:
			ctx.WriteKeyword("TEMPORARY")
		case postgres.TruncatePriv:
			ctx.WriteKeyword("TRUNCATE")
		default:
			return errors.New("Undefined privilege type")
		}
	}
	if n.Cols != nil {
		ctx.WritePlain(" (")
		for i, v := range n.Cols {
			if i != 0 {
				ctx.WritePlain(",")
			}
			if err := v.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore PrivElem.Cols[%d]", i)
			}
		}
		ctx.WritePlain(")")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *PrivElem) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*PrivElem)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*PrivElem)
	for i, val := range n.Cols {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Cols[i] = node.(*ColumnName)
	}
	return v.Leave(n)
}

// ObjectTypeType is the type for object type.
type ObjectTypeType int

const (
	// ObjectTypeNone is for empty object type.
	ObjectTypeNone ObjectTypeType = iota + 1
	// ObjectTypeTable means the following object is a table.
	ObjectTypeTable
	ObjectTypeSequence
	ObjectTypeFunction
	ObjectTypeProcedure
	ObjectTypeSchema
	ObjectTypeDatabase
	ObjectTypeUserType
	ObjectTypeTablespace
	ObjectTypeLanguage
	ObjectTypeDomain
	ObjectTypeForeignDataWrapper
	ObjectTypeForeignServer
	ObjectTypeRoutine
	ObjectTypeLargeObject
	ObjectTypeAllTablesInSchema
	ObjectTypeAllSequencesInSchema
	ObjectTypeAllFunctionsInSchema
	ObjectTypeAllProceduresInSchema
	ObjectTypeAllRoutinesInSchema
)

// Restore implements Node interface.
func (n ObjectTypeType) Restore(ctx *format.RestoreCtx) error {
	switch n {
	case ObjectTypeNone:
		// do nothing
	case ObjectTypeTable:
		ctx.WriteKeyword("TABLE")
	case ObjectTypeSequence:
		ctx.WriteKeyword("SEQUENCE")
	case ObjectTypeFunction:
		ctx.WriteKeyword("FUNCTION")
	case ObjectTypeProcedure:
		ctx.WriteKeyword("PROCEDURE")
	case ObjectTypeSchema:
		ctx.WriteKeyword("SCHEMA")
	case ObjectTypeDatabase:
		ctx.WriteKeyword("DATABASE")
	case ObjectTypeUserType:
		ctx.WriteKeyword("TYPE")
	case ObjectTypeTablespace:
		ctx.WriteKeyword("TABLESPACE")
	case ObjectTypeLanguage:
		ctx.WriteKeyword("LANGUAGE")
	case ObjectTypeDomain:
		ctx.WriteKeyword("DOMAIN")
	case ObjectTypeForeignDataWrapper:
		ctx.WriteKeyword("FOREIGN DATA WRAPPER")
	case ObjectTypeForeignServer:
		ctx.WriteKeyword("FOREIGN SERVER")
	case ObjectTypeRoutine:
		ctx.WriteKeyword("ROUTINE")
	case ObjectTypeLargeObject:
		ctx.WriteKeyword("LARGE OBJECT")
	case ObjectTypeAllTablesInSchema:
		ctx.WriteKeyword("ALL TABLES IN SCHEMA")
	case ObjectTypeAllSequencesInSchema:
		ctx.WriteKeyword("ALL SEQUENCES IN SCHEMA")
	case ObjectTypeAllFunctionsInSchema:
		ctx.WriteKeyword("ALL FUNCTIONS IN SCHEMA")
	case ObjectTypeAllProceduresInSchema:
		ctx.WriteKeyword("ALL PROCEDURES IN SCHEMA")
	case ObjectTypeAllRoutinesInSchema:
		ctx.WriteKeyword("ALL ROUTINES IN SCHEMA")
	default:
		return errors.New("Unsupported object type")
	}
	return ctx.Err()
}

// GrantLevelType is the type for grant level.
type GrantLevelType int

const (
	// GrantLevelNone is the dummy const for default value.
	GrantLevelNone GrantLevelType = iota + 1
	// GrantLevelGlobal means the privileges are administrative or apply to all databases on a given server.
	GrantLevelGlobal
	// GrantLevelDB means the privileges apply to all objects in a given database.
	GrantLevelDB
	// GrantLevelTable means the privileges apply to all columns in a given table.
	GrantLevelTable
)

// GrantLevel is used for store the privilege scope.
type GrantLevel struct {
	Level     GrantLevelType
	DBName    string
	TableName string
}

// Restore implements Node interface.
func (n *GrantLevel) Restore(ctx *format.RestoreCtx) error {
	switch n.Level {
	case GrantLevelDB:
		if n.DBName == "" {
			ctx.WritePlain("*")
		} else {
			ctx.WriteName(n.DBName)
			ctx.WritePlain(".*")
		}
	case GrantLevelGlobal:
		ctx.WritePlain("*.*")
	case GrantLevelTable:
		if n.DBName != "" {
			ctx.WriteName(n.DBName)
			ctx.WritePlain(".")
		}
		ctx.WriteName(n.TableName)
	}
	return ctx.Err()
}

// RevokeStmt is the struct for REVOKE statement.
type RevokeStmt struct {
	stmtNode

	Privs      []*PrivElem
	ObjectType ObjectTypeType
	Level      *GrantLevel
	Users      []*UserSpec
	// Objects are the objects the privileges are revoked from.
	Objects     []*TableName
	TargetNodes *List
	// Behavior is the drop behavior of the statement.
	Behavior    DropBehavior
	GrantOption bool
	Grantor     *RoleSpec
}

// Restore implements Node interface.
func (n *RevokeStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("REVOKE ")
	if n.GrantOption {
		ctx.WriteKeyword("GRANT OPTION FOR ")
	}
	for i, v := range n.Privs {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore RevokeStmt.Privs[%d]", i)
		}
	}
	ctx.WriteKeyword(" ON ")
	if n.ObjectType != ObjectTypeNone {
		if err := n.ObjectType.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore RevokeStmt.ObjectType")
		}
		ctx.WritePlain(" ")
	}
	if err := writeGrantTargets(ctx, n.Objects, n.Level, n.TargetNodes); err != nil {
		return errors.Annotate(err, "An error occurred while restore RevokeStmt.Objects")
	}
	ctx.WriteKeyword(" FROM ")
	for i, v := range n.Users {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore RevokeStmt.Users[%d]", i)
		}
	}
	if n.Grantor != nil {
		ctx.WriteKeyword(" GRANTED BY ")
		if err := n.Grantor.Restore(ctx); err != nil {
			return err
		}
	}
	switch n.Behavior {
	case DROP_CASCADE:
		ctx.WriteKeyword(" CASCADE")
	case DROP_RESTRICT:
		ctx.WriteKeyword(" RESTRICT")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *RevokeStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*RevokeStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*RevokeStmt)
	if n.TargetNodes != nil {
		node, ok := n.TargetNodes.Accept(v)
		if !ok {
			return n, false
		}
		n.TargetNodes = node.(*List)
	}
	for i, val := range n.Privs {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Privs[i] = node.(*PrivElem)
	}
	for i, object := range n.Objects {
		node, ok := object.Accept(v)
		if !ok {
			return n, false
		}
		n.Objects[i] = node.(*TableName)
	}
	for _, user := range n.Users {
		if user != nil && user.Role != nil {
			node, ok := user.Role.Accept(v)
			if !ok {
				return n, false
			}
			user.Role = node.(*RoleSpec)
		}
	}
	if n.Grantor != nil {
		node, ok := n.Grantor.Accept(v)
		if !ok {
			return n, false
		}
		n.Grantor = node.(*RoleSpec)
	}
	return v.Leave(n)
}

// RevokeRoleStmt is the struct for REVOKE statement.
type RevokeRoleStmt struct {
	stmtNode

	Roles       *List
	Users       *List
	GrantOption string
	Grantor     *RoleSpec
	Behavior    DropBehavior
}

// Restore implements Node interface.
func (n *RevokeRoleStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	if n.Roles == nil || n.Users == nil {
		return errors.New("role revoke has no roles or grantees")
	}
	ctx.WriteKeyword("REVOKE ")
	if n.GrantOption != "" {
		switch strings.ToLower(n.GrantOption) {
		case "admin", "inherit", "set":
		default:
			return errors.New("unsupported role revoke option")
		}
		ctx.WriteKeyword(n.GrantOption + " OPTION FOR ")
	}
	if err := n.Roles.Restore(ctx); err != nil {
		return err
	}
	ctx.WriteKeyword(" FROM ")
	if err := n.Users.Restore(ctx); err != nil {
		return err
	}
	if n.Grantor != nil {
		ctx.WriteKeyword(" GRANTED BY ")
		if err := n.Grantor.Restore(ctx); err != nil {
			return err
		}
	}
	switch n.Behavior {
	case DROP_CASCADE:
		ctx.WriteKeyword(" CASCADE")
	case DROP_RESTRICT:
		ctx.WriteKeyword(" RESTRICT")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *RevokeRoleStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*RevokeRoleStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*RevokeRoleStmt)
	if n.Roles != nil {
		node, ok := n.Roles.Accept(v)
		if !ok {
			return n, false
		}
		n.Roles = node.(*List)
	}
	if n.Users != nil {
		node, ok := n.Users.Accept(v)
		if !ok {
			return n, false
		}
		n.Users = node.(*List)
	}
	if n.Grantor != nil {
		node, ok := n.Grantor.Accept(v)
		if !ok {
			return n, false
		}
		n.Grantor = node.(*RoleSpec)
	}
	return v.Leave(n)
}

// writeGrantTargets writes the targets of a GRANT or REVOKE statement.
func writeGrantTargets(ctx *format.RestoreCtx, objects []*TableName, level *GrantLevel, targets *List) error {
	if targets != nil {
		return targets.Restore(ctx)
	}
	if len(objects) > 0 {
		for i, obj := range objects {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			if err := obj.Restore(ctx); err != nil {
				return err
			}
		}
		return ctx.Err()
	}
	if level == nil {
		return ctx.Err()
	}
	return level.Restore(ctx)
}

// GrantStmt is the struct for GRANT statement.
type GrantStmt struct {
	stmtNode

	Privs      []*PrivElem
	ObjectType ObjectTypeType
	Level      *GrantLevel
	Users      []*UserSpec
	WithGrant  bool
	// Objects are the objects the privileges are granted on.
	Objects     []*TableName
	TargetNodes *List
	// IsGrant is true for GRANT and false for REVOKE.
	IsGrant bool
	// GrantOption is true when the grant option is given.
	GrantOption bool
	// Grantor is the role granting the privileges.
	Grantor *RoleSpec
	// Behavior is the drop behavior of a REVOKE statement.
	Behavior DropBehavior
}

// Restore implements Node interface.
func (n *GrantStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("GRANT ")
	for i, v := range n.Privs {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore GrantStmt.Privs[%d]", i)
		}
	}
	ctx.WriteKeyword(" ON ")
	if n.ObjectType != ObjectTypeNone {
		if err := n.ObjectType.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore GrantStmt.ObjectType")
		}
		ctx.WritePlain(" ")
	}
	if err := writeGrantTargets(ctx, n.Objects, n.Level, n.TargetNodes); err != nil {
		return errors.Annotate(err, "An error occurred while restore GrantStmt.Objects")
	}
	ctx.WriteKeyword(" TO ")
	for i, v := range n.Users {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore GrantStmt.Users[%d]", i)
		}
	}
	if n.WithGrant || n.GrantOption {
		ctx.WriteKeyword(" WITH GRANT OPTION")
	}
	if n.Grantor != nil {
		ctx.WriteKeyword(" GRANTED BY ")
		if err := n.Grantor.Restore(ctx); err != nil {
			return err
		}
	}
	return ctx.Err()
}

// SecureText implements SensitiveStatement interface.
func (n *GrantStmt) SecureText() string {
	text := n.Text()
	// Filter "identified by xxx" because it would expose password information.
	idx := strings.Index(strings.ToLower(text), "identified")
	if idx > 0 {
		text = text[:idx]
	}
	return text
}

// Accept implements Node Accept interface.
func (n *GrantStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*GrantStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*GrantStmt)
	if n.TargetNodes != nil {
		node, ok := n.TargetNodes.Accept(v)
		if !ok {
			return n, false
		}
		n.TargetNodes = node.(*List)
	}
	for i, val := range n.Privs {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Privs[i] = node.(*PrivElem)
	}
	for i, object := range n.Objects {
		node, ok := object.Accept(v)
		if !ok {
			return n, false
		}
		n.Objects[i] = node.(*TableName)
	}
	for _, user := range n.Users {
		if user != nil && user.Role != nil {
			node, ok := user.Role.Accept(v)
			if !ok {
				return n, false
			}
			user.Role = node.(*RoleSpec)
		}
	}
	if n.Grantor != nil {
		node, ok := n.Grantor.Accept(v)
		if !ok {
			return n, false
		}
		n.Grantor = node.(*RoleSpec)
	}
	return v.Leave(n)
}

// GrantRoleStmt is the struct for GRANT TO statement.
type GrantRoleStmt struct {
	stmtNode

	Roles   *List
	Users   *List
	Options *List
	Grantor *RoleSpec
}

// Accept implements Node Accept interface.
func (n *GrantRoleStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*GrantRoleStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*GrantRoleStmt)
	if n.Roles != nil {
		node, ok := n.Roles.Accept(v)
		if !ok {
			return n, false
		}
		n.Roles = node.(*List)
	}
	if n.Users != nil {
		node, ok := n.Users.Accept(v)
		if !ok {
			return n, false
		}
		n.Users = node.(*List)
	}
	if n.Options != nil {
		node, ok := n.Options.Accept(v)
		if !ok {
			return n, false
		}
		n.Options = node.(*List)
	}
	if n.Grantor != nil {
		node, ok := n.Grantor.Accept(v)
		if !ok {
			return n, false
		}
		n.Grantor = node.(*RoleSpec)
	}
	return v.Leave(n)
}

// Restore implements Node interface.
func (n *GrantRoleStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	if n.Roles == nil || n.Users == nil {
		return errors.New("role grant has no roles or grantees")
	}
	ctx.WriteKeyword("GRANT ")
	if err := n.Roles.Restore(ctx); err != nil {
		return err
	}
	ctx.WriteKeyword(" TO ")
	if err := n.Users.Restore(ctx); err != nil {
		return err
	}
	if n.Options != nil && len(n.Options.Items) > 0 {
		ctx.WriteKeyword(" WITH ")
		for i, item := range n.Options.Items {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			option, ok := item.(*DefElem)
			if !ok || option.Arg == nil {
				return errors.New("invalid role grant option")
			}
			switch strings.ToLower(option.Defname) {
			case "admin", "inherit", "set":
			default:
				return errors.New("unsupported role grant option")
			}
			ctx.WriteKeyword(option.Defname + " ")
			if err := option.Arg.Restore(ctx); err != nil {
				return err
			}
		}
	}
	if n.Grantor != nil {
		ctx.WriteKeyword(" GRANTED BY ")
		if err := n.Grantor.Restore(ctx); err != nil {
			return err
		}
	}
	return ctx.Err()
}

// SecureText implements SensitiveStatement interface.
func (n *GrantRoleStmt) SecureText() string {
	text := n.Text()
	// Filter "identified by xxx" because it would expose password information.
	idx := strings.Index(strings.ToLower(text), "identified")
	if idx > 0 {
		text = text[:idx]
	}
	return text
}

// LockTableStmt TODO:
type LockTableStmt struct {
	stmtNode
}

// Restore implements Node interface.
func (n *LockTableStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *LockTableStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*LockTableStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*LockTableStmt)
	return v.Leave(n)
}

// UnlockTableStmt TODO:
type UnlockTableStmt struct {
	stmtNode
}

// Restore implements Node interface.
func (n *UnlockTableStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *UnlockTableStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*UnlockTableStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*UnlockTableStmt)
	return v.Leave(n)
}

// PurgeStmt TODO:
type PurgeStmt struct {
	stmtNode
}

// Restore implements Node interface.
func (n *PurgeStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	return errors.New("Not implemented")
}

// Accept implements Node Accept interface.
func (n *PurgeStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*PurgeStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*PurgeStmt)
	return v.Leave(n)
}

// Ident is the table identifier composed of schema name and table name.
type Ident struct {
	Schema model.CIStr
	Name   model.CIStr
}

// String implements fmt.Stringer interface.
func (i Ident) String() string {
	if i.Schema.O == "" {
		return i.Name.O
	}
	return fmt.Sprintf("%s.%s", i.Schema, i.Name)
}

// SelectStmtOpts wrap around select hints and switches
type SelectStmtOpts struct {
	DistinctOn      []*GroupItem
	SQLBigResult    bool
	SQLBufferResult bool
	SQLCache        bool
	SQLSmallResult  bool
	CalcFoundRows   bool
	StraightJoin    bool
	Priority        postgres.PriorityEnum
	TableHints      []*TableOptimizerHint
}

// TableOptimizerHint is Table level optimizer hint
type TableOptimizerHint struct {
	node
	// HintName is the name or alias of the table(s) which the hint will affect.
	// Table hints has no schema info
	// It allows only table name or alias (if table has an alias)
	HintName model.CIStr
	Tables   []model.CIStr
	// Statement Execution Time Optimizer Hints
	MaxExecutionTime uint64
}

// Restore implements Node interface.
func (n *TableOptimizerHint) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword(n.HintName.String())
	ctx.WritePlain("(")
	if n.HintName.L == "max_execution_time" {
		ctx.WritePlainf("%d", n.MaxExecutionTime)
	} else {
		for i, table := range n.Tables {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			ctx.WriteName(table.String())
		}
	}
	ctx.WritePlain(")")
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *TableOptimizerHint) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*TableOptimizerHint)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*TableOptimizerHint)
	return v.Leave(n)
}

// NewDecimal creates a types.Decimal value, it's provided by parser driver.
func NewDecimal(str string) (any, error) {
	return driver.NewDecimal(str)
}

// NewHexLiteral creates a types.HexLiteral value, it's provided by parser driver.
func NewHexLiteral(str string) (any, error) {
	return driver.NewHexLiteral(str)
}

// NewBitLiteral creates a types.BitLiteral value, it's provided by parser driver.
func NewBitLiteral(str string) (any, error) {
	return driver.NewBitLiteral(str)
}
