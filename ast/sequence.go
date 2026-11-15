package ast

import (
	"github.com/mia0x75/pgparser/errors"
	"github.com/mia0x75/pgparser/format"
)

var (
	_ DDLNode = &CreateSequenceStmt{}
	_ DDLNode = &DropSequenceStmt{}
)

// CreateSequenceStmt is a statement to create a sequence.
type CreateSequenceStmt struct {
	ddlNode

	IfNotExists bool
	Temporary   bool
	Name        *TableName
	// Options is the parsed sequence option list (DefElem entries).
	Options *List
}

// Restore implements Node interface.
func (n *CreateSequenceStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("CREATE ")
	if n.Temporary {
		ctx.WriteKeyWord("TEMPORARY ")
	}
	ctx.WriteKeyWord("SEQUENCE ")
	if n.IfNotExists {
		ctx.WriteKeyWord("IF NOT EXISTS ")
	}
	if err := n.Name.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while splicing CreateSequenceStmt Name")
	}
	return restoreSeqOptions(ctx, n.Options)
}

// Accept implements Node Accept interface.
func (n *CreateSequenceStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*CreateSequenceStmt)
	node, ok := n.Name.Accept(v)
	if !ok {
		return n, false
	}
	n.Name = node.(*TableName)
	if n.Options != nil {
		node, ok := n.Options.Accept(v)
		if !ok {
			return n, false
		}
		n.Options = node.(*List)
	}
	return v.Leave(n)
}

// DropSequenceStmt is a statement to drop one or more sequences.
type DropSequenceStmt struct {
	ddlNode

	IfExists bool
	Names    []*TableName
}

// Restore implements Node interface.
func (n *DropSequenceStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("DROP SEQUENCE ")
	if n.IfExists {
		ctx.WriteKeyWord("IF EXISTS ")
	}
	for i, name := range n.Names {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if err := name.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while splicing DropSequenceStmt Names[%d]", i)
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DropSequenceStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*DropSequenceStmt)
	for i, name := range n.Names {
		node, ok := name.Accept(v)
		if !ok {
			return n, false
		}
		n.Names[i] = node.(*TableName)
	}
	return v.Leave(n)
}
