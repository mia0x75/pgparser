package ast

import (
	"fmt"
	"io"
	"strings"

	"github.com/mia0x75/pgparser/format"
)

// ExprFlag derives syntax flags from pure AST structure. Execution-layer nodes
// no longer cache flags on the shared expression base.
func ExprFlag(expr ExprNode) uint64 {
	if expr == nil {
		return 0
	}
	if _, ok := expr.(ParamMarkerExpr); ok {
		return FlagHasParamMarker
	}
	switch x := expr.(type) {
	case *A_Indirection:
		flag := expressionFlags(x.Arg)
		if x.Indirection != nil {
			for _, item := range x.Indirection.Items {
				if index, ok := item.(*A_Indices); ok {
					flag |= expressionFlags(index.Lidx, index.Uidx)
				}
			}
		}
		return flag
	case *AggregateFuncExpr:
		flag := FlagHasAggregateFunc
		for _, val := range x.Args {
			flag |= ExprFlag(val)
		}
		return flag | functionOptionFlags(x.Filter, x.OrderBy, x.WithinGroup)
	case *WindowFuncExpr:
		flag := FlagHasWindowFunc
		for _, val := range x.Args {
			flag |= ExprFlag(val)
		}
		if x.Spec.PartitionBy != nil {
			for _, item := range x.Spec.PartitionBy.Items {
				flag |= ExprFlag(item.Expr)
			}
		}
		flag |= functionOptionFlags(nil, x.Spec.OrderBy, nil)
		if x.Spec.Frame != nil {
			for _, bound := range []*FrameBound{&x.Spec.Frame.Extent.Start, &x.Spec.Frame.Extent.End} {
				flag |= expressionFlags(bound.Expr, bound.Unit)
			}
		}
		return flag | functionOptionFlags(x.Filter, x.OrderBy, x.WithinGroup)
	case *BetweenExpr:
		return ExprFlag(x.Expr) | ExprFlag(x.Left) | ExprFlag(x.Right)
	case *BinaryOperationExpr:
		return ExprFlag(x.L) | ExprFlag(x.R)
	case *CaseExpr:
		var flag uint64
		if x.Value != nil {
			flag |= ExprFlag(x.Value)
		}
		for _, val := range x.WhenClauses {
			flag |= ExprFlag(val.Expr) | ExprFlag(val.Result)
		}
		if x.ElseClause != nil {
			flag |= ExprFlag(x.ElseClause)
		}
		return flag
	case *ColumnNameExpr:
		return FlagHasReference
	case *CompareSubqueryExpr:
		return ExprFlag(x.L) | ExprFlag(x.R)
	case *DefaultExpr:
		return FlagHasDefault
	case *ExistsSubqueryExpr:
		return ExprFlag(x.Sel)
	case *FuncCallExpr, *SQLValueFunction:
		flag := FlagHasFunc
		if call, ok := expr.(*FuncCallExpr); ok {
			for _, val := range call.Args {
				flag |= ExprFlag(val)
			}
			flag |= functionOptionFlags(call.Filter, call.OrderBy, call.WithinGroup)
		}
		return flag
	case *FuncCastExpr:
		return FlagHasFunc | ExprFlag(x.Expr)
	case *IsNullExpr, *IsTruthExpr, *ParenthesesExpr:
		switch e := expr.(type) {
		case *IsNullExpr:
			return ExprFlag(e.Expr)
		case *IsTruthExpr:
			return ExprFlag(e.Expr)
		default:
			return ExprFlag(e.(*ParenthesesExpr).Expr)
		}
	case *PatternInExpr:
		flag := ExprFlag(x.Expr)
		for _, val := range x.List {
			flag |= ExprFlag(val)
		}
		if x.Sel != nil {
			flag |= ExprFlag(x.Sel)
		}
		return flag
	case *PatternLikeExpr, *PatternRegexpExpr:
		switch e := expr.(type) {
		case *PatternLikeExpr:
			flag := ExprFlag(e.Pattern)
			if e.Expr != nil {
				flag |= ExprFlag(e.Expr)
			}
			return flag
		default:
			re := e.(*PatternRegexpExpr)
			flag := ExprFlag(re.Pattern)
			if re.Expr != nil {
				flag |= ExprFlag(re.Expr)
			}
			return flag
		}
	case *PositionExpr:
		return FlagHasReference | ExprFlag(x.P)
	case *ValuesExpr:
		return FlagHasReference
	case *RowExpr:
		var flag uint64
		for _, val := range x.Values {
			flag |= ExprFlag(val)
		}
		return flag
	case *SubqueryExpr:
		return FlagHasSubquery
	case *UnaryOperationExpr:
		return ExprFlag(x.V)
	case *VariableExpr:
		if x.Value == nil {
			return FlagHasVariable
		}
		return FlagHasVariable | ExprFlag(x.Value)
	}
	return 0
}

// FormatExpr preserves legacy rendering without widening the shared ExprNode.
// Use Restore when output errors must be reported.
func FormatExpr(expr ExprNode, w io.Writer) {
	if legacy, ok := expr.(interface{ Format(io.Writer) }); ok {
		legacy.Format(w)
		return
	}
	var out strings.Builder
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &out)
	if err := expr.Restore(ctx); err != nil {
		_, _ = fmt.Fprint(w, "<unsupported>")
		return
	}
	_, _ = io.WriteString(w, out.String())
}
