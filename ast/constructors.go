package ast

import (
	"io"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/pgparser/errors"
	"github.com/mia0x75/pgparser/format"
	"github.com/mia0x75/pgparser/opcode"
	"github.com/mia0x75/pgparser/types"
)

// NewColumnNameExpr returns a ColumnNameExpr for the given qualified name.
func NewColumnNameExpr(schema, table, name string) *ColumnNameExpr {
	qualified := nameFromStrings([]string{schema, table, name})
	if name == "" {
		qualified.Parts = append(qualified.Parts, pkast.Ident{})
	}
	qualified.Span = loc.InvalidSpan()
	for i := range qualified.Parts {
		qualified.Parts[i].Span = loc.InvalidSpan()
	}
	return &ColumnNameExpr{
		BaseColumnRef: pkast.BaseColumnRef{Name: qualified},
	}
}

// NewBinaryOperationExpr returns a BinaryOperationExpr for the given opcode.
func NewBinaryOperationExpr(op opcode.Op, l, r ExprNode) *BinaryOperationExpr {
	return &BinaryOperationExpr{Op: op, BaseBinaryExpr: pkast.BaseBinaryExpr{Left: l, Right: r}}
}

// NewBinaryOperationExprByName preserves an operator's text without requiring an opcode.
func NewBinaryOperationExprByName(name string, l, r ExprNode) *BinaryOperationExpr {
	return &BinaryOperationExpr{OpName: name, BaseBinaryExpr: pkast.BaseBinaryExpr{Left: l, Right: r}}
}

// NewUnaryOperationExpr returns a UnaryOperationExpr for the given opcode.
func NewUnaryOperationExpr(op opcode.Op, v ExprNode) *UnaryOperationExpr {
	return &UnaryOperationExpr{Op: op, BaseUnaryExpr: pkast.BaseUnaryExpr{Expr: v}}
}

// NewUnaryOperationExprByName preserves an operator's text without requiring an opcode.
func NewUnaryOperationExprByName(name string, v ExprNode) *UnaryOperationExpr {
	return &UnaryOperationExpr{OpName: name, BaseUnaryExpr: pkast.BaseUnaryExpr{Expr: v}}
}

// NewFuncCallExpr returns a FuncCallExpr for the given function name.
func NewFuncCallExpr(name string, args ...ExprNode) *FuncCallExpr {
	return &FuncCallExpr{BaseCallExpr: pkast.BaseCallExpr{Name: nameFromDotted(name), Args: args}}
}

// NewFuncCallExprParts returns a FuncCallExpr for the given name parts.
func NewFuncCallExprParts(parts []string, args ...ExprNode) *FuncCallExpr {
	return &FuncCallExpr{BaseCallExpr: pkast.BaseCallExpr{Name: nameFromStrings(parts), Args: args}}
}

// NewTypeCastExpr returns a FuncCastExpr casting expr to tp.
func NewTypeCastExpr(expr ExprNode, tp *types.FieldType) *FuncCastExpr {
	return &FuncCastExpr{Expr: expr, Tp: tp, FunctionType: CastFunction}
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
	if n == nil {
		return restoreNode(ctx, n)
	}

	ctx.WriteKeyword("ARRAY[")
	for i, e := range n.Elements {
		if i != 0 {
			ctx.WritePlain(",")
		}
		if err := e.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred when restore ArrayExpr.Elements[%v]", i)
		}
	}
	ctx.WritePlain("]")
	return ctx.Err()
}

// Format implements ExprNode interface.
func (n *ArrayExpr) Format(w io.Writer) {
	_, _ = io.WriteString(w, "ARRAY[")
	for i, e := range n.Elements {
		if i != 0 {
			_, _ = io.WriteString(w, ",")
		}
		FormatExpr(e, w)
	}
	_, _ = io.WriteString(w, "]")
}

// Accept implements Node Accept interface.
func (n *ArrayExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*ArrayExpr)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
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
