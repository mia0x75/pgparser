package ast

import (
	"github.com/mia0x75/pgparser/errors"
	"github.com/mia0x75/pgparser/format"
)

var (
	_ DDLNode = &CreateTablespaceStmt{}
	_ DDLNode = &AlterTablespaceStmt{}
	_ DDLNode = &DropTablespaceStmt{}
	_ Node    = &TablespaceOption{}
)

// TablespaceOptionType is the type for tablespace options.
type TablespaceOptionType int

// Tablespace option types.
const (
	TablespaceOptionNone TablespaceOptionType = iota
	TablespaceOptionDataFile
	TablespaceOptionFileBlockSize
	TablespaceOptionExtentSize
	TablespaceOptionInitialSize
	TablespaceOptionAutoextendSize
	TablespaceOptionMaxSize
	TablespaceOptionNodegroup
	TablespaceOptionWait
	TablespaceOptionNoWait
	TablespaceOptionComment
	TablespaceOptionEngine
	TablespaceOptionLocation
)

// TablespaceOption represents a single option of a tablespace definition.
type TablespaceOption struct {
	node

	Tp  TablespaceOptionType
	Str string
	Val uint64
}

// Restore implements Node interface.
func (n *TablespaceOption) Restore(ctx *format.RestoreCtx) error {
	switch n.Tp {
	case TablespaceOptionDataFile:
		ctx.WriteKeyWord("ADD DATAFILE ")
		ctx.WriteString(n.Str)
	case TablespaceOptionFileBlockSize:
		ctx.WriteKeyWord("FILE_BLOCK_SIZE = ")
		ctx.WritePlainf("%d", n.Val)
	case TablespaceOptionExtentSize:
		ctx.WriteKeyWord("EXTENT_SIZE = ")
		ctx.WritePlainf("%d", n.Val)
	case TablespaceOptionInitialSize:
		ctx.WriteKeyWord("INITIAL_SIZE = ")
		ctx.WritePlainf("%d", n.Val)
	case TablespaceOptionAutoextendSize:
		ctx.WriteKeyWord("AUTOEXTEND_SIZE = ")
		ctx.WritePlainf("%d", n.Val)
	case TablespaceOptionMaxSize:
		ctx.WriteKeyWord("MAX_SIZE = ")
		ctx.WritePlainf("%d", n.Val)
	case TablespaceOptionNodegroup:
		ctx.WriteKeyWord("NODEGROUP = ")
		ctx.WritePlainf("%d", n.Val)
	case TablespaceOptionWait:
		ctx.WriteKeyWord("WAIT")
	case TablespaceOptionNoWait:
		ctx.WriteKeyWord("NO WAIT")
	case TablespaceOptionComment:
		ctx.WriteKeyWord("COMMENT = ")
		ctx.WriteString(n.Str)
	case TablespaceOptionEngine:
		ctx.WriteKeyWord("ENGINE = ")
		ctx.WritePlain(n.Str)
	case TablespaceOptionLocation:
		ctx.WriteKeyWord("LOCATION ")
		ctx.WriteString(n.Str)
	default:
		return errors.New("An error occurred while splicing TablespaceOption")
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *TablespaceOption) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*TablespaceOption)
	return v.Leave(n)
}

// CreateTablespaceStmt is a statement to create a tablespace.
// See https://dev.postgres.com/doc/refman/8.0/en/create-tablespace.html
type CreateTablespaceStmt struct {
	ddlNode

	Undo    bool
	Name    string
	Options []*TablespaceOption
}

// Restore implements Node interface.
func (n *CreateTablespaceStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("CREATE ")
	if n.Undo {
		ctx.WriteKeyWord("UNDO ")
	}
	ctx.WriteKeyWord("TABLESPACE ")
	ctx.WriteName(n.Name)
	for _, opt := range n.Options {
		ctx.WritePlain(" ")
		if err := opt.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing CreateTablespaceStmt")
		}
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *CreateTablespaceStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*CreateTablespaceStmt)
	for i, opt := range n.Options {
		node, ok := opt.Accept(v)
		if !ok {
			return n, false
		}
		n.Options[i] = node.(*TablespaceOption)
	}
	return v.Leave(n)
}

// AlterTablespaceStmt is a statement to alter a tablespace.
// See https://dev.postgres.com/doc/refman/8.0/en/alter-tablespace.html
type AlterTablespaceStmt struct {
	ddlNode

	Undo     bool
	Name     string
	Options  []*TablespaceOption
	RenameTo string
}

// Restore implements Node interface.
func (n *AlterTablespaceStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("ALTER ")
	if n.Undo {
		ctx.WriteKeyWord("UNDO ")
	}
	ctx.WriteKeyWord("TABLESPACE ")
	ctx.WriteName(n.Name)
	for _, opt := range n.Options {
		ctx.WritePlain(" ")
		if err := opt.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing AlterTablespaceStmt")
		}
	}
	if n.RenameTo != "" {
		ctx.WriteKeyWord(" RENAME TO ")
		ctx.WriteName(n.RenameTo)
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *AlterTablespaceStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*AlterTablespaceStmt)
	for i, opt := range n.Options {
		node, ok := opt.Accept(v)
		if !ok {
			return n, false
		}
		n.Options[i] = node.(*TablespaceOption)
	}
	return v.Leave(n)
}

// DropTablespaceStmt is a statement to drop a tablespace.
// See https://dev.postgres.com/doc/refman/8.0/en/drop-tablespace.html
type DropTablespaceStmt struct {
	ddlNode

	Undo     bool
	IfExists bool
	Name     string
	Option   *TablespaceOption
}

// Restore implements Node interface.
func (n *DropTablespaceStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("DROP ")
	if n.Undo {
		ctx.WriteKeyWord("UNDO ")
	}
	ctx.WriteKeyWord("TABLESPACE ")
	if n.IfExists {
		ctx.WriteKeyWord("IF EXISTS ")
	}
	ctx.WriteName(n.Name)
	if n.Option != nil {
		ctx.WritePlain(" ")
		if err := n.Option.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing DropTablespaceStmt")
		}
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *DropTablespaceStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*DropTablespaceStmt)
	return v.Leave(n)
}
