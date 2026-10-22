package ast

import (
	"github.com/mia0x75/pgparser/auth"
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
	_ StmtNode = &CallStmt{}
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

	Options []string
	Tables  []*TableName
}

// Restore implements Node interface.
func (n *VacuumStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("VACUUM")
	if len(n.Options) > 0 {
		ctx.WritePlain(" (")
		for i, opt := range n.Options {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			ctx.WriteKeyWord(opt)
		}
		ctx.WritePlain(")")
	}
	restoreTableList(ctx, n.Tables)
	return nil
}

// Accept implements Node Accept interface.
func (n *VacuumStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*VacuumStmt)
	for i, table := range n.Tables {
		node, ok := table.Accept(v)
		if !ok {
			return n, false
		}
		n.Tables[i] = node.(*TableName)
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
	ctx.WriteKeyWord("REINDEX")
	if n.Verbose {
		ctx.WritePlain(" (VERBOSE)")
	}
	if n.Concurrently {
		ctx.WriteKeyWord(" CONCURRENTLY")
	}
	if n.ObjectType != "" {
		ctx.WritePlain(" ")
		ctx.WriteKeyWord(n.ObjectType)
	}
	if n.Name != "" {
		ctx.WritePlain(" ")
		ctx.WriteName(n.Name)
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *ReindexStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
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
	ctx.WriteKeyWord("CLUSTER")
	if n.Verbose {
		ctx.WriteKeyWord(" VERBOSE")
	}
	if n.Table != nil {
		ctx.WritePlain(" ")
		if err := n.Table.Restore(ctx); err != nil {
			return err
		}
	}
	if n.Index != nil {
		ctx.WriteKeyWord(" USING ")
		if err := n.Index.Restore(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *ClusterStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
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
	ctx.WriteKeyWord("REFRESH MATERIALIZED VIEW ")
	if n.Concurrently {
		ctx.WriteKeyWord("CONCURRENTLY ")
	}
	if err := n.Table.Restore(ctx); err != nil {
		return err
	}
	if n.DataSpecified {
		if n.NoData {
			ctx.WriteKeyWord(" WITH NO DATA")
		} else {
			ctx.WriteKeyWord(" WITH DATA")
		}
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *RefreshMatViewStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
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
	ctx.WriteKeyWord("CHECKPOINT")
	return nil
}

// Accept implements Node Accept interface.
func (n *CheckpointStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
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
	ctx.WriteKeyWord("DISCARD ")
	ctx.WriteKeyWord(n.Mode)
	return nil
}

// Accept implements Node Accept interface.
func (n *DiscardStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
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
	ctx.WriteKeyWord("LOAD ")
	ctx.WriteString(n.Path)
	return nil
}

// Accept implements Node Accept interface.
func (n *LoadStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
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
	ctx.WriteKeyWord("SECURITY LABEL")
	if n.Provider != "" {
		ctx.WriteKeyWord(" FOR ")
		ctx.WriteName(n.Provider)
	}
	ctx.WriteKeyWord(" ON ")
	ctx.WriteKeyWord(n.ObjectType)
	ctx.WritePlain(" ")
	ctx.WriteName(n.ObjectName)
	ctx.WriteKeyWord(" IS ")
	ctx.WriteString(n.Label)
	return nil
}

// Accept implements Node Accept interface.
func (n *SecurityLabelStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
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
}

// Restore implements Node interface.
func (n *ImportForeignSchemaStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("IMPORT FOREIGN SCHEMA ")
	ctx.WriteName(n.RemoteSchema)
	ctx.WriteKeyWord(" FROM SERVER ")
	ctx.WriteName(n.Server)
	ctx.WriteKeyWord(" INTO ")
	ctx.WriteName(n.Into)
	for _, opt := range n.Options {
		ctx.WritePlain(" ")
		ctx.WriteKeyWord(opt)
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *ImportForeignSchemaStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	return v.Leave(newNode.(*ImportForeignSchemaStmt))
}

// ReassignOwnedStmt is a REASSIGN OWNED statement.
type ReassignOwnedStmt struct {
	stmtNode

	Roles   []*auth.RoleIdentity
	NewRole *auth.RoleIdentity
}

// Restore implements Node interface.
func (n *ReassignOwnedStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("REASSIGN OWNED BY ")
	restoreRoleList(ctx, n.Roles)
	ctx.WriteKeyWord(" TO ")
	ctx.WriteName(n.NewRole.Username)
	return nil
}

// Accept implements Node Accept interface.
func (n *ReassignOwnedStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
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
	ctx.WriteKeyWord("DROP OWNED BY ")
	restoreRoleList(ctx, n.Roles)
	if n.Cascade {
		ctx.WriteKeyWord(" CASCADE")
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *DropOwnedStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	return v.Leave(newNode.(*DropOwnedStmt))
}

// CallStmt is a CALL statement.
type CallStmt struct {
	stmtNode

	Proc *FuncCallExpr
}

// Restore implements Node interface.
func (n *CallStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("CALL ")
	if err := n.Proc.Restore(ctx); err != nil {
		return err
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *CallStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*CallStmt)
	node, ok := n.Proc.Accept(v)
	if !ok {
		return n, false
	}
	n.Proc = node.(*FuncCallExpr)
	return v.Leave(n)
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
	if n.IsMove {
		ctx.WriteKeyWord("MOVE ")
	} else {
		ctx.WriteKeyWord("FETCH ")
	}
	if n.Direction != "" {
		ctx.WriteKeyWord(n.Direction)
		if n.Count != 0 {
			ctx.WritePlainf(" %d", n.Count)
		}
		ctx.WritePlain(" ")
	}
	ctx.WriteKeyWord("FROM ")
	ctx.WriteName(n.Name)
	return nil
}

// Accept implements Node Accept interface.
func (n *FetchCursorStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
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
	ctx.WriteKeyWord("CLOSE ")
	if n.All {
		ctx.WriteKeyWord("ALL")
	} else {
		ctx.WriteName(n.Name)
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *CloseCursorStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
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
	ctx.WriteKeyWord("LISTEN ")
	ctx.WriteName(n.Channel)
	return nil
}

// Accept implements Node Accept interface.
func (n *ListenStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
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
	ctx.WriteKeyWord("NOTIFY ")
	ctx.WriteName(n.Channel)
	if n.Payload != "" {
		ctx.WritePlain(", ")
		ctx.WriteString(n.Payload)
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *NotifyStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
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
	ctx.WriteKeyWord("UNLISTEN ")
	if n.All {
		ctx.WritePlain("*")
	} else {
		ctx.WriteName(n.Channel)
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *UnlistenStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
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
	Options []string
	// Attlist holds the attributes of the copy options.
	Attlist []string
}

// Restore implements Node interface.
func (n *CopyStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("COPY ")
	if err := n.Table.Restore(ctx); err != nil {
		return err
	}
	if len(n.Columns) > 0 {
		ctx.WritePlain(" (")
		for i, col := range n.Columns {
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
		ctx.WriteKeyWord(" FROM ")
	} else {
		ctx.WriteKeyWord(" TO ")
	}
	if n.Program {
		ctx.WriteKeyWord("PROGRAM ")
	}
	if n.Std {
		ctx.WriteKeyWord("STDIN")
	} else {
		ctx.WriteString(n.FileName)
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *CopyStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*CopyStmt)
	node, ok := n.Table.Accept(v)
	if !ok {
		return n, false
	}
	n.Table = node.(*TableName)
	for i, col := range n.Columns {
		node, ok := col.Accept(v)
		if !ok {
			return n, false
		}
		n.Columns[i] = node.(*ColumnName)
	}
	return v.Leave(n)
}

func restoreTableList(ctx *format.RestoreCtx, tables []*TableName) {
	for i, table := range tables {
		if i == 0 {
			ctx.WritePlain(" ")
		} else {
			ctx.WritePlain(", ")
		}
		_ = table.Restore(ctx)
	}
}

func restoreRoleList(ctx *format.RestoreCtx, roles []*auth.RoleIdentity) {
	for i, role := range roles {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		ctx.WriteName(role.Username)
	}
}
