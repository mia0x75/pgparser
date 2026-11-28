package ast

import (
	"github.com/mia0x75/pgparser/format"
)

// PrivilegeTarget is the target of a GRANT or REVOKE statement.
type PrivilegeTarget struct {
	node

	// ObjectType is the kind of the target objects.
	ObjectType ObjectTypeType
	// Objects are the target objects.
	Objects     []*TableName
	TargetNodes *List
}

// Restore implements Node interface.
func (n *PrivilegeTarget) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restoreNode(ctx, n)
	}

	if err := writeGrantTargets(ctx, n.Objects, nil, n.TargetNodes); err != nil {
		return err
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *PrivilegeTarget) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	node, skip := v.Enter(n)
	outer, ok := node.(*PrivilegeTarget)
	if !ok || outer == nil {
		return node, false
	}
	n = outer
	if skip {
		return v.Leave(n)
	}
	for i0 := range n.Objects {
		if !nodePresent(n.Objects[i0]) {
			return n, false
		}
		updated, ok := n.Objects[i0].Accept(v)
		if !ok || !nodePresent(updated) {
			return n, false
		}
		replacement, ok := updated.(*TableName)
		if !ok {
			return n, false
		}
		n.Objects[i0] = replacement
	}
	if n.TargetNodes != nil {
		if !nodePresent(n.TargetNodes) {
			return n, false
		}
		updated, ok := n.TargetNodes.Accept(v)
		if !ok || !nodePresent(updated) {
			return n, false
		}
		replacement, ok := updated.(*List)
		if !ok {
			return n, false
		}
		n.TargetNodes = replacement
	}
	return v.Leave(n)
}
