package ast

import (
	"github.com/mia0x75/pgparser/format"
)

var (
	_ DDLNode = &CreateResourceGroupStmt{}
	_ DDLNode = &AlterResourceGroupStmt{}
	_ DDLNode = &DropResourceGroupStmt{}
	_ Node    = &ResourceGroupVCPU{}
)

// Resource group states.
const (
	ResourceGroupStateNone = iota
	ResourceGroupStateEnable
	ResourceGroupStateDisable
)

// ResourceGroupVCPU is a single VCPU range in a resource group definition.
type ResourceGroupVCPU struct {
	node

	From uint64
	To   uint64
}

// Restore implements Node interface.
func (n *ResourceGroupVCPU) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	if n.From == n.To {
		ctx.WritePlainf("%d", n.From)
	} else {
		ctx.WritePlainf("%d-%d", n.From, n.To)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ResourceGroupVCPU) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*ResourceGroupVCPU)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*ResourceGroupVCPU)
	return v.Leave(n)
}

func restoreVCPUList(ctx *format.RestoreCtx, vcpu []*ResourceGroupVCPU) error {
	ctx.WriteKeyword(" VCPU = ")
	for i, v := range vcpu {
		if i > 0 {
			ctx.WritePlain(",")
		}
		if err := v.Restore(ctx); err != nil {
			return err
		}
	}
	return ctx.Err()
}

func restoreResourceGroupOptional(ctx *format.RestoreCtx, tp *int, state int) {
	if tp != nil {
		ctx.WriteKeyword(" THREAD_PRIORITY = ")
		ctx.WritePlainf("%d", *tp)
	}
	switch state {
	case ResourceGroupStateEnable:
		ctx.WriteKeyword(" ENABLE")
	case ResourceGroupStateDisable:
		ctx.WriteKeyword(" DISABLE")
	}
}

// CreateResourceGroupStmt is a statement to create a resource group.
type CreateResourceGroupStmt struct {
	ddlNode

	Name           string
	IsSystem       bool
	VCPU           []*ResourceGroupVCPU
	ThreadPriority *int
	State          int
}

// Restore implements Node interface.
func (n *CreateResourceGroupStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("CREATE RESOURCE GROUP ")
	ctx.WriteName(n.Name)
	ctx.WriteKeyword(" TYPE = ")
	if n.IsSystem {
		ctx.WriteKeyword("SYSTEM")
	} else {
		ctx.WriteKeyword("USER")
	}
	if len(n.VCPU) > 0 {
		if err := restoreVCPUList(ctx, n.VCPU); err != nil {
			return err
		}
	}
	restoreResourceGroupOptional(ctx, n.ThreadPriority, n.State)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CreateResourceGroupStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*CreateResourceGroupStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*CreateResourceGroupStmt)
	for i, vcpu := range n.VCPU {
		node, ok := vcpu.Accept(v)
		if !ok {
			return n, false
		}
		n.VCPU[i] = node.(*ResourceGroupVCPU)
	}
	return v.Leave(n)
}

// AlterResourceGroupStmt is a statement to alter a resource group.
type AlterResourceGroupStmt struct {
	ddlNode

	Name           string
	VCPU           []*ResourceGroupVCPU
	ThreadPriority *int
	State          int
}

// Restore implements Node interface.
func (n *AlterResourceGroupStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("ALTER RESOURCE GROUP ")
	ctx.WriteName(n.Name)
	if len(n.VCPU) > 0 {
		if err := restoreVCPUList(ctx, n.VCPU); err != nil {
			return err
		}
	}
	restoreResourceGroupOptional(ctx, n.ThreadPriority, n.State)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *AlterResourceGroupStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*AlterResourceGroupStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*AlterResourceGroupStmt)
	for i, vcpu := range n.VCPU {
		node, ok := vcpu.Accept(v)
		if !ok {
			return n, false
		}
		n.VCPU[i] = node.(*ResourceGroupVCPU)
	}
	return v.Leave(n)
}

// DropResourceGroupStmt is a statement to drop a resource group.
type DropResourceGroupStmt struct {
	ddlNode

	Name  string
	Force bool
}

// Restore implements Node interface.
func (n *DropResourceGroupStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	ctx.WriteKeyword("DROP RESOURCE GROUP ")
	ctx.WriteName(n.Name)
	if n.Force {
		ctx.WriteKeyword(" FORCE")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DropResourceGroupStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*DropResourceGroupStmt)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
	}
	n = newNode.(*DropResourceGroupStmt)
	return v.Leave(n)
}
