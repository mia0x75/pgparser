package ast

import (
	"fmt"

	"github.com/mia0x75/pgparser/format"
	"github.com/mia0x75/pgparser/types"
)

type DataType struct {
	node
	Type *types.FieldType
}

func (n *DataType) Restore(ctx *format.RestoreCtx) error {
	if n.Type == nil {
		return fmt.Errorf("data type is missing")
	}
	return n.Type.Restore(ctx)
}

func (n *DataType) Accept(v Visitor) (Node, bool) {
	updated, _ := v.Enter(n)
	return v.Leave(updated)
}
