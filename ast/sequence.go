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
	if n == nil {
		return restoreNode(ctx, n)
	}

	ctx.WriteKeyword("CREATE ")
	if n.Temporary {
		ctx.WriteKeyword("TEMPORARY ")
	}
	ctx.WriteKeyword("SEQUENCE ")
	if n.IfNotExists {
		ctx.WriteKeyword("IF NOT EXISTS ")
	}
	if err := n.Name.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while splicing CreateSequenceStmt Name")
	}
	return restoreSeqOptions(ctx, n.Options)
}

// Accept implements Node Accept interface.
func (n *CreateSequenceStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*CreateSequenceStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
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
	if n == nil {
		return restoreNode(ctx, n)
	}

	ctx.WriteKeyword("DROP SEQUENCE ")
	if n.IfExists {
		ctx.WriteKeyword("IF EXISTS ")
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
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*DropSequenceStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
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
