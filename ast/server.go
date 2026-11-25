package ast

import (
	"github.com/mia0x75/pgparser/errors"
	"github.com/mia0x75/pgparser/format"
)

var (
	_ DDLNode = &CreateServerStmt{}
	_ DDLNode = &AlterServerStmt{}
	_ DDLNode = &DropServerStmt{}
	_ Node    = &ServerOption{}
)

// ServerOption represents a single option of a foreign server definition.
type ServerOption struct {
	node

	Name  string
	Value string
}

// Restore implements Node interface.
func (n *ServerOption) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteName(n.Name)
	ctx.WritePlain(" ")
	ctx.WriteString(n.Value)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ServerOption) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*ServerOption)
	return v.Leave(n)
}

// CreateServerStmt is a statement to create a foreign server.
type CreateServerStmt struct {
	ddlNode

	Name    string
	Wrapper string
	// ServerType is the optional TYPE clause; empty when absent.
	ServerType string
	// Version is the optional VERSION clause; empty when absent.
	Version string
	Options []*ServerOption
}

// Restore implements Node interface.
func (n *CreateServerStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("CREATE SERVER ")
	ctx.WriteName(n.Name)
	if n.ServerType != "" {
		ctx.WriteKeyword(" TYPE ")
		ctx.WriteString(n.ServerType)
	}
	if n.Version != "" {
		ctx.WriteKeyword(" VERSION ")
		ctx.WriteString(n.Version)
	}
	ctx.WriteKeyword(" FOREIGN DATA WRAPPER ")
	ctx.WriteName(n.Wrapper)
	if len(n.Options) > 0 {
		ctx.WriteKeyword(" OPTIONS(")
		for i, opt := range n.Options {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if err := opt.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while splicing CreateServerStmt")
			}
		}
		ctx.WritePlain(")")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CreateServerStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*CreateServerStmt)
	for i, opt := range n.Options {
		node, ok := opt.Accept(v)
		if !ok {
			return n, false
		}
		n.Options[i] = node.(*ServerOption)
	}
	return v.Leave(n)
}

// AlterServerStmt is a statement to alter a foreign server.
type AlterServerStmt struct {
	ddlNode

	Name string
	// Version is the optional VERSION clause; empty when absent.
	Version string
	Options []*ServerOption
}

// Restore implements Node interface.
func (n *AlterServerStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("ALTER SERVER ")
	ctx.WriteName(n.Name)
	if n.Version != "" {
		ctx.WriteKeyword(" VERSION ")
		ctx.WriteString(n.Version)
	}
	if len(n.Options) > 0 {
		ctx.WriteKeyword(" OPTIONS(")
		for i, opt := range n.Options {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if err := opt.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while splicing AlterServerStmt")
			}
		}
		ctx.WritePlain(")")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *AlterServerStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*AlterServerStmt)
	for i, opt := range n.Options {
		node, ok := opt.Accept(v)
		if !ok {
			return n, false
		}
		n.Options[i] = node.(*ServerOption)
	}
	return v.Leave(n)
}

// DropServerStmt is a statement to drop a foreign server.
type DropServerStmt struct {
	ddlNode

	IfExists bool
	Name     string
}

// Restore implements Node interface.
func (n *DropServerStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("DROP SERVER ")
	if n.IfExists {
		ctx.WriteKeyword("IF EXISTS ")
	}
	ctx.WriteName(n.Name)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DropServerStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*DropServerStmt)
	return v.Leave(n)
}
