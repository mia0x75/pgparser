package ast

import (
	"strings"

	"github.com/mia0x75/pgparser/auth"
	"github.com/mia0x75/pgparser/errors"
	"github.com/mia0x75/pgparser/format"
)

var (
	_ StmtNode = &VacuumStmt{}
	_ StmtNode = &ReindexStmt{}
	_ StmtNode = &ClusterStmt{}
	_ StmtNode = &RefreshMatViewStmt{}
	_ StmtNode = &CheckpointStmt{}
	_ StmtNode = &DiscardStmt{}
	_ StmtNode = &LoadStmt{}
	_ StmtNode = &SecurityLabelStmt{}
	_ StmtNode = &ImportForeignSchemaStmt{}
	_ StmtNode = &ReassignOwnedStmt{}
	_ StmtNode = &DropOwnedStmt{}
	_ StmtNode = &FetchCursorStmt{}
	_ StmtNode = &CloseCursorStmt{}
	_ StmtNode = &ListenStmt{}
	_ StmtNode = &NotifyStmt{}
	_ StmtNode = &UnlistenStmt{}
	_ StmtNode = &CopyStmt{}
)

// VacuumStmt is a VACUUM statement.
type VacuumStmt struct {
	stmtNode

	Options   *List
	Relations *List
}

// Restore implements Node interface.
func (n *VacuumStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("VACUUM")
	if err := restoreUtilityOptions(ctx, n.Options); err != nil {
		return err
	}
	if n.Relations != nil {
		ctx.WritePlain(" ")
		return n.Relations.Restore(ctx)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *VacuumStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*VacuumStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*VacuumStmt)
	for _, list := range []**List{&n.Options, &n.Relations} {
		if *list != nil {
			child, ok := (*list).Accept(v)
			if !ok {
				return n, false
			}
			*list = child.(*List)
		}
	}
	return v.Leave(n)
}

// ReindexStmt is a REINDEX statement.
type ReindexStmt struct {
	stmtNode

	ObjectType   string
	Name         string
	Concurrently bool
	Verbose      bool
}

// Restore implements Node interface.
func (n *ReindexStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("REINDEX")
	if n.Verbose {
		ctx.WritePlain(" (VERBOSE)")
	}
	if n.ObjectType != "" {
		ctx.WritePlain(" ")
		ctx.WriteKeyword(n.ObjectType)
	}
	// The grammar places CONCURRENTLY between the object type and the name.
	if n.Concurrently {
		ctx.WriteKeyword(" CONCURRENTLY")
	}
	if n.Name != "" {
		ctx.WritePlain(" ")
		ctx.WriteName(n.Name)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ReindexStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*ReindexStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*ReindexStmt)
	return v.Leave(n)
}

// ClusterStmt is a CLUSTER statement.
type ClusterStmt struct {
	stmtNode

	Table   *TableName
	Index   *TableName
	Verbose bool
}

// Restore implements Node interface.
func (n *ClusterStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("CLUSTER")
	if n.Verbose {
		ctx.WriteKeyword(" VERBOSE")
	}
	if n.Table != nil {
		ctx.WritePlain(" ")
		if err := n.Table.Restore(ctx); err != nil {
			return err
		}
	}
	if n.Index != nil {
		ctx.WriteKeyword(" USING ")
		if err := n.Index.Restore(ctx); err != nil {
			return err
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ClusterStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*ClusterStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*ClusterStmt)
	if n.Table != nil {
		node, ok := n.Table.Accept(v)
		if !ok {
			return n, false
		}
		n.Table = node.(*TableName)
	}
	if n.Index != nil {
		node, ok := n.Index.Accept(v)
		if !ok {
			return n, false
		}
		n.Index = node.(*TableName)
	}
	return v.Leave(n)
}

// RefreshMatViewStmt is a REFRESH MATERIALIZED VIEW statement.
type RefreshMatViewStmt struct {
	stmtNode

	Concurrently  bool
	Table         *TableName
	NoData        bool
	DataSpecified bool
}

// Restore implements Node interface.
func (n *RefreshMatViewStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("REFRESH MATERIALIZED VIEW ")
	if n.Concurrently {
		ctx.WriteKeyword("CONCURRENTLY ")
	}
	if err := n.Table.Restore(ctx); err != nil {
		return err
	}
	if n.DataSpecified {
		if n.NoData {
			ctx.WriteKeyword(" WITH NO DATA")
		} else {
			ctx.WriteKeyword(" WITH DATA")
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *RefreshMatViewStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*RefreshMatViewStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*RefreshMatViewStmt)
	node, ok := n.Table.Accept(v)
	if !ok {
		return n, false
	}
	n.Table = node.(*TableName)
	return v.Leave(n)
}

// CheckpointStmt is a CHECKPOINT statement.
type CheckpointStmt struct{ stmtNode }

// Restore implements Node interface.
func (n *CheckpointStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("CHECKPOINT")
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CheckpointStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*CheckpointStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	return v.Leave(newNode.(*CheckpointStmt))
}

// DiscardStmt is a DISCARD statement.
type DiscardStmt struct {
	stmtNode

	Mode string
}

// Restore implements Node interface.
func (n *DiscardStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("DISCARD ")
	ctx.WriteKeyword(n.Mode)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DiscardStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*DiscardStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	return v.Leave(newNode.(*DiscardStmt))
}

// LoadStmt is a LOAD statement.
type LoadStmt struct {
	stmtNode

	Path string
}

// Restore implements Node interface.
func (n *LoadStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("LOAD ")
	ctx.WriteString(n.Path)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *LoadStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*LoadStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	return v.Leave(newNode.(*LoadStmt))
}

// SecurityLabelStmt is a SECURITY LABEL statement.
type SecurityLabelStmt struct {
	stmtNode

	Provider   string
	ObjectType string
	ObjectName string
	Label      string
}

// Restore implements Node interface.
func (n *SecurityLabelStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("SECURITY LABEL")
	if n.Provider != "" {
		ctx.WriteKeyword(" FOR ")
		ctx.WriteName(n.Provider)
	}
	ctx.WriteKeyword(" ON ")
	ctx.WriteKeyword(n.ObjectType)
	ctx.WritePlain(" ")
	ctx.WriteName(n.ObjectName)
	ctx.WriteKeyword(" IS ")
	ctx.WriteString(n.Label)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *SecurityLabelStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*SecurityLabelStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	return v.Leave(newNode.(*SecurityLabelStmt))
}

// ImportForeignSchemaStmt is an IMPORT FOREIGN SCHEMA statement.
type ImportForeignSchemaStmt struct {
	stmtNode

	RemoteSchema string
	Server       string
	Into         string
	Options      []string
	ListType     ImportForeignSchemaType
	TableList    *List
	OptionDefs   *List
}

// Restore implements Node interface.
func (n *ImportForeignSchemaStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("IMPORT FOREIGN SCHEMA ")
	ctx.WriteName(n.RemoteSchema)
	switch n.ListType {
	case FDW_IMPORT_SCHEMA_ALL:
	case FDW_IMPORT_SCHEMA_LIMIT_TO, FDW_IMPORT_SCHEMA_EXCEPT:
		if n.TableList == nil || len(n.TableList.Items) == 0 {
			return errors.New("foreign schema import has no table list")
		}
		if n.ListType == FDW_IMPORT_SCHEMA_LIMIT_TO {
			ctx.WriteKeyword(" LIMIT TO (")
		} else {
			ctx.WriteKeyword(" EXCEPT (")
		}
		if err := n.TableList.Restore(ctx); err != nil {
			return err
		}
		ctx.WritePlain(")")
	default:
		return errors.New("invalid foreign schema import selection")
	}
	ctx.WriteKeyword(" FROM SERVER ")
	ctx.WriteName(n.Server)
	ctx.WriteKeyword(" INTO ")
	ctx.WriteName(n.Into)
	if n.OptionDefs != nil {
		ctx.WritePlain(" ")
		return restoreAlterOptions(ctx, n.OptionDefs)
	}
	for _, opt := range n.Options {
		ctx.WritePlain(" ")
		ctx.WriteKeyword(opt)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ImportForeignSchemaStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*ImportForeignSchemaStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*ImportForeignSchemaStmt)
	for _, list := range []**List{&n.TableList, &n.OptionDefs} {
		if *list != nil {
			child, ok := (*list).Accept(v)
			if !ok {
				return n, false
			}
			*list = child.(*List)
		}
	}
	return v.Leave(n)
}

// ReassignOwnedStmt is a REASSIGN OWNED statement.
type ReassignOwnedStmt struct {
	stmtNode

	Roles   []*auth.RoleIdentity
	NewRole *auth.RoleIdentity
}

// Restore implements Node interface.
func (n *ReassignOwnedStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("REASSIGN OWNED BY ")
	restoreRoleList(ctx, n.Roles)
	ctx.WriteKeyword(" TO ")
	ctx.WriteName(n.NewRole.Username)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ReassignOwnedStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*ReassignOwnedStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	return v.Leave(newNode.(*ReassignOwnedStmt))
}

// DropOwnedStmt is a DROP OWNED statement.
type DropOwnedStmt struct {
	stmtNode

	Roles   []*auth.RoleIdentity
	Cascade bool
}

// Restore implements Node interface.
func (n *DropOwnedStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("DROP OWNED BY ")
	restoreRoleList(ctx, n.Roles)
	if n.Cascade {
		ctx.WriteKeyword(" CASCADE")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DropOwnedStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*DropOwnedStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	return v.Leave(newNode.(*DropOwnedStmt))
}

// FetchCursorStmt is a FETCH or MOVE cursor statement.
type FetchCursorStmt struct {
	stmtNode

	IsMove    bool
	Direction string
	Count     int64
	Name      string
}

// Restore implements Node interface.
func (n *FetchCursorStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	if n.IsMove {
		ctx.WriteKeyword("MOVE ")
	} else {
		ctx.WriteKeyword("FETCH ")
	}
	switch n.Direction {
	case "NEXT", "PRIOR", "FIRST", "LAST":
		// Count is always 1 for these directions; writing it would produce
		// SQL the grammar does not accept, which breaks restore stability.
		ctx.WriteKeyword(n.Direction + " ")
	case "ALL":
		ctx.WriteKeyword("ALL ")
	default:
		if n.Direction != "" {
			ctx.WriteKeyword(n.Direction)
			ctx.WritePlain(" ")
		}
		switch {
		// The ALL sentinel maps back to the ALL keyword, not an integer.
		case n.Count == FETCH_ALL:
			ctx.WriteKeyword("ALL ")
		case n.Count > 0 || n.Direction == "ABSOLUTE" || n.Direction == "RELATIVE":
			// ABSOLUTE and RELATIVE accept 0 as a meaningful quantity, so
			// the count is written unconditionally for them.
			ctx.WritePlainf("%d", n.Count)
			ctx.WritePlain(" ")
		}
	}
	ctx.WriteKeyword("FROM ")
	ctx.WriteName(n.Name)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *FetchCursorStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*FetchCursorStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	return v.Leave(newNode.(*FetchCursorStmt))
}

// CloseCursorStmt is a CLOSE cursor statement.
type CloseCursorStmt struct {
	stmtNode

	Name string
	All  bool
}

// Restore implements Node interface.
func (n *CloseCursorStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("CLOSE ")
	if n.All {
		ctx.WriteKeyword("ALL")
	} else {
		ctx.WriteName(n.Name)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CloseCursorStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*CloseCursorStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	return v.Leave(newNode.(*CloseCursorStmt))
}

// ListenStmt is a LISTEN statement.
type ListenStmt struct {
	stmtNode

	Channel string
}

// Restore implements Node interface.
func (n *ListenStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("LISTEN ")
	ctx.WriteName(n.Channel)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ListenStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*ListenStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	return v.Leave(newNode.(*ListenStmt))
}

// NotifyStmt is a NOTIFY statement.
type NotifyStmt struct {
	stmtNode

	Channel string
	Payload string
}

// Restore implements Node interface.
func (n *NotifyStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("NOTIFY ")
	ctx.WriteName(n.Channel)
	if n.Payload != "" {
		ctx.WritePlain(", ")
		ctx.WriteString(n.Payload)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *NotifyStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*NotifyStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	return v.Leave(newNode.(*NotifyStmt))
}

// UnlistenStmt is an UNLISTEN statement.
type UnlistenStmt struct {
	stmtNode

	Channel string
	All     bool
}

// Restore implements Node interface.
func (n *UnlistenStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("UNLISTEN ")
	if n.All {
		ctx.WritePlain("*")
	} else {
		ctx.WriteName(n.Channel)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *UnlistenStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*UnlistenStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	return v.Leave(newNode.(*UnlistenStmt))
}

// CopyStmt is a COPY statement.
type CopyStmt struct {
	stmtNode

	Table    *TableName
	Columns  []*ColumnName
	IsFrom   bool
	FileName string
	Program  bool
	Std      bool
	// Query is the statement of a `COPY (query) TO` statement.
	Query Node
	// WhereClause is the filter of a `COPY` statement.
	WhereClause ExprNode
	// Options holds the copy options of the statement.
	Options []*DefElem
}

// Restore implements Node interface.
func (n *CopyStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	if (n.Table == nil) == (n.Query == nil) {
		return errors.New("COPY requires exactly one table or query")
	}
	if n.Program && n.Std {
		return errors.New("COPY PROGRAM requires a file name")
	}
	if n.Query != nil && (n.IsFrom || len(n.Columns) > 0 || n.WhereClause != nil) {
		return errors.New("query COPY cannot have FROM, columns or WHERE")
	}
	if n.Std && n.FileName != "" {
		return errors.New("COPY has both a file and a standard stream")
	}
	ctx.WriteKeyword("COPY ")
	switch {
	case n.Query != nil:
		ctx.WritePlain("(")
		if err := n.Query.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore CopyStmt.Query")
		}
		ctx.WritePlain(")")
	case n.Table != nil:
		if err := n.Table.Restore(ctx); err != nil {
			return err
		}
	default:
		return errors.New("CopyStmt has neither Table nor Query")
	}
	if len(n.Columns) > 0 {
		ctx.WritePlain(" (")
		for i, col := range n.Columns {
			if col == nil {
				return errors.New("COPY has a nil column")
			}
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if err := col.Restore(ctx); err != nil {
				return err
			}
		}
		ctx.WritePlain(")")
	}
	if n.IsFrom {
		ctx.WriteKeyword(" FROM ")
	} else {
		ctx.WriteKeyword(" TO ")
	}
	if n.Program {
		ctx.WriteKeyword("PROGRAM ")
	}
	switch {
	case n.Std && n.IsFrom:
		ctx.WriteKeyword("STDIN")
	case n.Std:
		ctx.WriteKeyword("STDOUT")
	default:
		ctx.WriteString(n.FileName)
	}
	if len(n.Options) > 0 {
		ctx.WriteKeyword(" WITH (")
		for i, option := range n.Options {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if err := restoreUtilityOption(ctx, option); err != nil {
				return errors.Annotatef(err, "COPY option %d", i)
			}
		}
		ctx.WritePlain(")")
	}
	if n.WhereClause != nil {
		ctx.WriteKeyword(" WHERE ")
		if err := n.WhereClause.Restore(ctx); err != nil {
			return errors.Annotate(err, "COPY filter")
		}
	}
	return ctx.Err()
}

func restoreUtilityOptions(ctx *format.RestoreCtx, options *List) error {
	if options == nil {
		return ctx.Err()
	}
	ctx.WritePlain(" (")
	for i, item := range options.Items {
		option, ok := item.(*DefElem)
		if !ok {
			return errors.New("invalid utility option")
		}
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if err := restoreUtilityOption(ctx, option); err != nil {
			return err
		}
	}
	ctx.WritePlain(")")
	return ctx.Err()
}

// writeBoolKeyword writes TRUE or FALSE.
func writeBoolKeyword(ctx *format.RestoreCtx, b bool) {
	if b {
		ctx.WriteKeyword("TRUE")
	} else {
		ctx.WriteKeyword("FALSE")
	}
}

// isBooleanOption reports whether a utility option takes a boolean value.
// Values like DELIMITER '1' or NULL '0' are strings, not booleans, so the
// textual-boolean normalisation must stay limited to these names.
func isBooleanOption(name string) bool {
	switch name {
	case "header", "freeze", "analyze", "verbose", "costs", "buffers", "wal", "timing",
		"summary", "settings", "memory", "serialize", "generic_plan", "truncate":
		return true
	}
	return false
}

// isBoolLiteral reports whether s is a textual boolean accepted by COPY options.
func isBoolLiteral(s string) bool {
	switch strings.ToLower(s) {
	case "true", "false", "on", "off", "1", "0":
		return true
	}
	return false
}

// boolLiteralValue maps the accepted boolean spellings to a value.
func boolLiteralValue(s string) bool {
	switch strings.ToLower(s) {
	case "true", "on", "1":
		return true
	}
	return false
}

func restoreUtilityOption(ctx *format.RestoreCtx, option *DefElem) error {
	if option == nil || option.Defname == "" {
		return errors.New("utility option has no name")
	}
	if option.Defnamespace != "" || option.Defaction != int(DEFELEM_UNSPEC) {
		return errors.New("invalid utility option metadata")
	}
	switch option.Defname {
	case "format", "freeze", "delimiter", "null", "default", "header", "quote", "escape",
		"force_quote", "force_not_null", "force_null", "encoding", "on_error", "reject_limit", "log_verbosity",
		"full", "verbose", "analyze", "parallel", "buffer_usage_limit":
		ctx.WriteKeyword(option.Defname)
	default:
		ctx.WriteName(option.Defname)
	}
	if option.Arg == nil {
		return ctx.Err()
	}
	ctx.WritePlain(" ")
	if boolean, ok := option.Arg.(*Boolean); ok {
		// A valueless boolean option (e.g. COPY ... (HEADER)) arrives as a
		// Boolean, while an explicit `HEADER true` arrives as a String. Always
		// write the TRUE/FALSE form so both restore to the same text.
		writeBoolKeyword(ctx, boolean.Boolval)
		return ctx.Err()
	}
	if str, ok := option.Arg.(*String); ok && isBooleanOption(option.Defname) && isBoolLiteral(str.Str) {
		writeBoolKeyword(ctx, boolLiteralValue(str.Str))
		return ctx.Err()
	}
	if list, ok := option.Arg.(*List); ok {
		ctx.WritePlain("(")
		for i, item := range list.Items {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if item == nil {
				return errors.New("COPY option has a nil value")
			}
			if name, ok := item.(*String); ok && strings.HasPrefix(option.Defname, "force_") {
				ctx.WriteName(name.Str)
			} else if err := item.Restore(ctx); err != nil {
				return err
			}
		}
		ctx.WritePlain(")")
		return ctx.Err()
	}
	return option.Arg.Restore(ctx)
}

// Accept implements Node Accept interface.
func (n *CopyStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*CopyStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*CopyStmt)
	if n.Table != nil {
		node, ok := n.Table.Accept(v)
		if !ok {
			return n, false
		}
		n.Table = node.(*TableName)
	}
	if n.Query != nil {
		node, ok := n.Query.Accept(v)
		if !ok {
			return n, false
		}
		n.Query = node
	}
	for i, col := range n.Columns {
		if col == nil {
			continue
		}
		node, ok := col.Accept(v)
		if !ok {
			return n, false
		}
		n.Columns[i] = node.(*ColumnName)
	}
	for i, option := range n.Options {
		if option == nil {
			continue
		}
		node, ok := option.Accept(v)
		if !ok {
			return n, false
		}
		n.Options[i] = node.(*DefElem)
	}
	if n.WhereClause != nil {
		node, ok := n.WhereClause.Accept(v)
		if !ok {
			return n, false
		}
		n.WhereClause = node.(ExprNode)
	}
	return v.Leave(n)
}

func restoreRoleList(ctx *format.RestoreCtx, roles []*auth.RoleIdentity) {
	for i, role := range roles {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		ctx.WriteName(role.Username)
	}
}
