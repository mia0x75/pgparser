package ast

import (
	"fmt"
	"io"
	"math"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/pgparser/errors"
	"github.com/mia0x75/pgparser/format"
)

// ParamMarkerExpr represents a parameter placeholder without execution data.
type ParamMarkerExpr struct {
	pkast.BaseParameterExpr
}

// NewParamMarkerExpr creates a parameter with an unknown source span.
func NewParamMarkerExpr(index int) *ParamMarkerExpr {
	return NewParamMarkerExprAt(loc.InvalidSpan(), index)
}

// NewParamMarkerExprAt creates a parameter with independent source and index data.
func NewParamMarkerExprAt(span loc.Span, index int) *ParamMarkerExpr {
	n := &ParamMarkerExpr{BaseParameterExpr: *pkast.NewBaseParameterExpr(index)}
	n.SetSpan(span)
	return n
}

// Restore implements Node interface.
func (n *ParamMarkerExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore ParamMarkerExpr: nil parameter")
	}
	if n.Index < 0 || n.Index >= math.MaxInt32 {
		return errors.New("invalid PostgreSQL parameter number")
	}
	ctx.WritePlainf("$%d", n.Index+1)
	return ctx.Err()
}

// Format implements ExprNode interface.
func (n *ParamMarkerExpr) Format(w io.Writer) {
	_, _ = fmt.Fprintf(w, "$%d", n.Index+1)
}

// Accept implements Node Accept interface.
func (n *ParamMarkerExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, _ := v.Enter(n)
	marker, ok := entered.(*ParamMarkerExpr)
	if !ok || marker == nil {
		return entered, false
	}
	return v.Leave(marker)
}

// SetOrder preserves the existing API; the syntax index is canonical.
func (n *ParamMarkerExpr) SetOrder(index int) { n.Index = index }
