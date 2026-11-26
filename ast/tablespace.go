package ast

import (
	"fmt"

	"github.com/mia0x75/parserkit/lex"

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

	Name     lex.Ident
	Owner    *RoleSpec
	Location string
	// Options is the `WITH (...)` storage parameter list.
	Options *List
}

// Restore implements Node interface.
func (n *CreateTablespaceStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("CREATE TABLESPACE ")
	ctx.WriteIdent(n.Name)
	if n.Owner != nil {
		ctx.WriteKeyword(" OWNER ")
		if err := n.Owner.Restore(ctx); err != nil {
			return err
		}
	}
	// LOCATION is mandatory syntax, so it is written even when empty.
	ctx.WriteKeyword(" LOCATION ")
	ctx.WriteString(n.Location)
	if n.Options != nil {
		ctx.WriteKeyword(" WITH ")
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

	Name lex.Ident
	// Action is `SET` or `RESET` when Options are present.
	Action  string
	Options *List
	// RenameTo is set by `ALTER TABLESPACE ... RENAME TO`.
	RenameTo lex.Ident
}

// Restore implements Node interface.
func (n *AlterTablespaceStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("ALTER TABLESPACE ")
	ctx.WriteIdent(n.Name)
	if n.Action != "" {
		if n.Options == nil {
			return fmt.Errorf("ALTER TABLESPACE %s has no options", n.Action)
		}
		ctx.WriteKeyword(" " + n.Action + " ")
		if err := restoreRelOptions(ctx, n.Options); err != nil {
			return err
		}
	}
	if n.RenameTo.Present() {
		ctx.WriteKeyword(" RENAME TO ")
		ctx.WriteIdent(n.RenameTo)
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
	Name     lex.Ident
}

// Restore implements Node interface.
func (n *DropTablespaceStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("DROP TABLESPACE ")
	if n.IfExists {
		ctx.WriteKeyword("IF EXISTS ")
	}
	ctx.WriteIdent(n.Name)
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
