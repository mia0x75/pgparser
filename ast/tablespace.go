package ast

import (
	"fmt"

	"github.com/mia0x75/pgparser/format"
)

var (
	_ DDLNode = &CreateTablespaceStmt{}
	_ DDLNode = &AlterTablespaceStmt{}
	_ DDLNode = &DropTablespaceStmt{}
)

// CreateTablespaceStmt is a PostgreSQL `CREATE TABLESPACE` statement.
// See https://www.postgresql.org/docs/current/sql-createtablespace.html
type CreateTablespaceStmt struct {
	ddlNode

	Name     string
	Owner    *RoleSpec
	Location string
	// Options is the `WITH (...)` storage parameter list.
	Options *List
}

// Restore implements Node interface.
func (n *CreateTablespaceStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("CREATE TABLESPACE ")
	ctx.WriteName(n.Name)
	if n.Owner != nil {
		ctx.WriteKeyWord(" OWNER ")
		if err := n.Owner.Restore(ctx); err != nil {
			return err
		}
	}
	// LOCATION is mandatory syntax, so it is written even when empty.
	ctx.WriteKeyWord(" LOCATION ")
	ctx.WriteString(n.Location)
	if n.Options != nil {
		ctx.WriteKeyWord(" WITH ")
		if err := restoreRelOptions(ctx, n.Options); err != nil {
			return err
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CreateTablespaceStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*CreateTablespaceStmt)
	if n.Owner != nil {
		child, ok := n.Owner.Accept(v)
		if !ok {
			return n, false
		}
		n.Owner = child.(*RoleSpec)
	}
	if n.Options != nil {
		child, ok := n.Options.Accept(v)
		if !ok {
			return n, false
		}
		n.Options = child.(*List)
	}
	return v.Leave(n)
}

// AlterTablespaceStmt is a statement to alter a tablespace.
type AlterTablespaceStmt struct {
	ddlNode

	Name string
	// Action is `SET` or `RESET` when Options are present.
	Action  string
	Options *List
	// RenameTo is set by `ALTER TABLESPACE ... RENAME TO`.
	RenameTo string
}

// Restore implements Node interface.
func (n *AlterTablespaceStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("ALTER TABLESPACE ")
	ctx.WriteName(n.Name)
	if n.Action != "" {
		if n.Options == nil {
			return fmt.Errorf("ALTER TABLESPACE %s has no options", n.Action)
		}
		ctx.WriteKeyWord(" " + n.Action + " ")
		if err := restoreRelOptions(ctx, n.Options); err != nil {
			return err
		}
	}
	if n.RenameTo != "" {
		ctx.WriteKeyWord(" RENAME TO ")
		ctx.WriteName(n.RenameTo)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *AlterTablespaceStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*AlterTablespaceStmt)
	if n.Options != nil {
		child, ok := n.Options.Accept(v)
		if !ok {
			return n, false
		}
		n.Options = child.(*List)
	}
	return v.Leave(n)
}

// DropTablespaceStmt is a statement to drop a tablespace.
type DropTablespaceStmt struct {
	ddlNode

	IfExists bool
	Name     string
}

// Restore implements Node interface.
func (n *DropTablespaceStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("DROP TABLESPACE ")
	if n.IfExists {
		ctx.WriteKeyWord("IF EXISTS ")
	}
	ctx.WriteName(n.Name)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DropTablespaceStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	return v.Leave(newNode.(*DropTablespaceStmt))
}
