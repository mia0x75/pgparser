package driver

import (
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/errors"
	"github.com/mia0x75/pgparser/format"
	"github.com/mia0x75/pgparser/hack"
	"github.com/mia0x75/pgparser/postgres"
	"github.com/mia0x75/pgparser/types"
)

// The purpose of driver package is to decompose the dependency of the parser and
// types package.
// It provides the NewValueExpr function for the ast package, so the ast package
// do not depends on the concrete definition of `types.Datum`, thus get rid of
// the dependency of the types package.
// The parser package depends on the ast package, but not the types package.
// The whole relationship:
// ast imports []
// types imports [parser/types]
// parser imports [ast, parser/types]
// driver imports [ast, types]
// callers import [parser, driver]

func init() {
	ast.NewValueExpr = newValueExpr
	ast.NewParamMarkerExpr = newParamMarkerExpr
	ast.NewDecimal = func(str string) (any, error) {
		dec := new(types.PgDecimal)
		err := dec.FromString(hack.Slice(str))
		return dec, err
	}
	ast.NewHexLiteral = func(str string) (any, error) {
		h, err := types.NewHexLiteral(str)
		return h, err
	}
	ast.NewBitLiteral = func(str string) (any, error) {
		b, err := types.NewBitLiteral(str)
		return b, err
	}
}

var (
	_ ast.ParamMarkerExpr = &ParamMarkerExpr{}
	_ ast.ValueExpr       = &ValueExpr{}
)

// ValueExpr is the simple value expression.
type ValueExpr struct {
	ast.TexprNode
	types.Datum
	projectionOffset int
}

// valueExprString returns the string rendering of a ValueExpr's value.
// Restore and Format share this helper so the Kind-to-string mapping stays
// in sync. Callers handle kind-specific output details (charset prefix,
// quoting, ctx vs Writer).
func valueExprString(n *ValueExpr) (string, error) {
	switch n.Kind() {
	case types.KindNull:
		return "NULL", nil
	case types.KindInt64:
		if n.Type.Flag&postgres.IsBooleanFlag != 0 {
			if n.GetInt64() > 0 {
				return "TRUE", nil
			}
			return "FALSE", nil
		}
		return strconv.FormatInt(n.GetInt64(), 10), nil
	case types.KindUint64:
		return strconv.FormatUint(n.GetUint64(), 10), nil
	case types.KindFloat32:
		return strconv.FormatFloat(n.GetFloat64(), 'e', -1, 32), nil
	case types.KindFloat64:
		return strconv.FormatFloat(n.GetFloat64(), 'e', -1, 64), nil
	case types.KindString, types.KindBytes:
		return n.GetString(), nil
	case types.KindDecimal:
		return n.GetPgDecimal().String(), nil
	case types.KindBinaryLiteral:
		if n.Type.Flag&postgres.UnsignedFlag != 0 {
			return fmt.Sprintf("x'%x'", n.GetBytes()), nil
		}
		return n.GetBinaryLiteral().ToBitLiteralString(true), nil
	case types.KindDuration, types.KindEnum,
		types.KindBit, types.KindSet, types.KindTime,
		types.KindInterface, types.KindMinNotNull, types.KindMaxValue,
		types.KindRaw, types.KindJSON:
		return "", errors.New("Not implemented")
	default:
		return "", errors.New("can't format to string")
	}
}

// Restore implements Node interface.
func (n *ValueExpr) Restore(ctx *format.RestoreCtx) error {
	s, err := valueExprString(n)
	if err != nil {
		return err
	}
	if n.Kind() == types.KindString {
		if n.Type.Charset != "" && n.Type.Charset != postgres.DefaultCharset {
			ctx.WritePlain("_")
			ctx.WriteKeyWord(n.Type.Charset)
		}
		ctx.WriteString(s)
		return ctx.Err()
	}
	if n.Kind() == types.KindBytes {
		ctx.WriteString(s)
		return ctx.Err()
	}
	ctx.WritePlain(s)
	return ctx.Err()
}

// GetDatumString implements the ast.ValueExpr interface.
func (n *ValueExpr) GetDatumString() string {
	return n.GetString()
}

// Format the ExprNode into a Writer.
// Unsupported kinds write a placeholder instead of panicking, so a caller
// that formats an expression cannot crash the process.
func (n *ValueExpr) Format(w io.Writer) {
	s, err := valueExprString(n)
	if err != nil {
		_, _ = fmt.Fprint(w, "<unsupported>")
		return
	}
	if n.Kind() == types.KindString || n.Kind() == types.KindBytes {
		s = strconv.Quote(s)
	}
	_, _ = fmt.Fprint(w, s)
}

// newValueExpr creates a ValueExpr with value, and sets default field type.
func newValueExpr(value any) ast.ValueExpr {
	if ve, ok := value.(*ValueExpr); ok {
		return ve
	}
	ve := &ValueExpr{}
	ve.SetValue(value)
	types.DefaultTypeForValue(value, &ve.Type)
	ve.projectionOffset = -1
	return ve
}

// SetProjectionOffset sets ValueExpr.projectionOffset for logical plan builder.
func (n *ValueExpr) SetProjectionOffset(offset int) {
	n.projectionOffset = offset
}

// GetProjectionOffset returns ValueExpr.projectionOffset.
func (n *ValueExpr) GetProjectionOffset() int {
	return n.projectionOffset
}

// Accept implements Node interface.
func (n *ValueExpr) Accept(v ast.Visitor) (ast.Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*ValueExpr)
	return v.Leave(n)
}

// ParamMarkerExpr expression holds a place for another expression.
// Used in parsing prepare statement.
type ParamMarkerExpr struct {
	ValueExpr
	Offset int
	Order  int
}

// Restore implements Node interface.
func (n *ParamMarkerExpr) Restore(ctx *format.RestoreCtx) error {
	if n.Offset < 0 || n.Offset >= math.MaxInt32 {
		return errors.New("invalid PostgreSQL parameter number")
	}
	ctx.WritePlainf("$%d", n.Offset+1)
	return ctx.Err()
}

func newParamMarkerExpr(offset int) ast.ParamMarkerExpr {
	return &ParamMarkerExpr{
		Offset: offset,
		Order:  offset,
	}
}

// Format the ExprNode into a Writer.
func (n *ParamMarkerExpr) Format(w io.Writer) {
	_, _ = fmt.Fprintf(w, "$%d", n.Offset+1)
}

// Accept implements Node Accept interface.
func (n *ParamMarkerExpr) Accept(v ast.Visitor) (ast.Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*ParamMarkerExpr)
	return v.Leave(n)
}

// SetOrder implements the ast.ParamMarkerExpr interface.
func (n *ParamMarkerExpr) SetOrder(order int) {
	n.Order = order
}
