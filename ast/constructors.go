package ast

import (
	"io"

	"github.com/mia0x75/pgparser/errors"
	"github.com/mia0x75/pgparser/format"
	"github.com/mia0x75/pgparser/model"
	"github.com/mia0x75/pgparser/opcode"
	"github.com/mia0x75/pgparser/types"
)

// NewColumnNameExpr returns a ColumnNameExpr for the given qualified name.
func NewColumnNameExpr(schema, table, name string) *ColumnNameExpr {
	return &ColumnNameExpr{
		Name: &ColumnName{
			Schema: model.NewCIStr(schema),
			Table:  model.NewCIStr(table),
			Name:   model.NewCIStr(name),
		},
	}
}

// NewBinaryOperationExpr returns a BinaryOperationExpr for the given opcode.
func NewBinaryOperationExpr(op opcode.Op, l, r ExprNode) *BinaryOperationExpr {
	return &BinaryOperationExpr{Op: op, L: l, R: r}
}

// NewBinaryOperationExprByName returns a BinaryOperationExpr for operators
// that have no opcode equivalent, such as `->>` or `@@`.
func NewBinaryOperationExprByName(name string, l, r ExprNode) *BinaryOperationExpr {
	return &BinaryOperationExpr{OpName: name, L: l, R: r}
}

// NewUnaryOperationExpr returns a UnaryOperationExpr for the given opcode.
func NewUnaryOperationExpr(op opcode.Op, v ExprNode) *UnaryOperationExpr {
	return &UnaryOperationExpr{Op: op, V: v}
}

// NewUnaryOperationExprByName returns a UnaryOperationExpr for operators that
// have no opcode equivalent.
func NewUnaryOperationExprByName(name string, v ExprNode) *UnaryOperationExpr {
	return &UnaryOperationExpr{OpName: name, V: v}
}

// NewFuncCallExpr returns a FuncCallExpr for the given function name.
func NewFuncCallExpr(name string, args ...ExprNode) *FuncCallExpr {
	return &FuncCallExpr{FnName: model.NewCIStr(name), Args: args}
}

// NewTypeCastExpr returns a FuncCastExpr casting expr to tp.
func NewTypeCastExpr(expr ExprNode, tp *types.FieldType) *FuncCastExpr {
	return &FuncCastExpr{Expr: expr, Tp: tp, FunctionType: CastBinaryOperator}
}

// NewArrayExpr returns an ArrayExpr holding the given elements.
func NewArrayExpr(elements []ExprNode) *ArrayExpr {
	return &ArrayExpr{Elements: elements}
}

// ArrayExpr is a PostgreSQL ARRAY[...] expression.
type ArrayExpr struct {
	exprNode

	// Elements are the array elements.
	Elements []ExprNode
}

// Restore implements Node interface.
func (n *ArrayExpr) Restore(ctx *format.RestoreCtx) error {
	ctx.WritePlain("[")
	for i, e := range n.Elements {
		if i != 0 {
			ctx.WritePlain(",")
		}
		if err := e.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred when restore ArrayExpr.Elements[%v]", i)
		}
	}
	ctx.WritePlain("]")
	return nil
}

// Format implements ExprNode interface.
func (n *ArrayExpr) Format(w io.Writer) {
	_, _ = io.WriteString(w, "[")
	for i, e := range n.Elements {
		if i != 0 {
			_, _ = io.WriteString(w, ",")
		}
		e.Format(w)
	}
	_, _ = io.WriteString(w, "]")
}

// Accept implements Node Accept interface.
func (n *ArrayExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*ArrayExpr)
	for i, val := range n.Elements {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Elements[i] = node.(ExprNode)
	}
	return v.Leave(n)
}
