package driver

import (
	"fmt"
	"io"
	"strconv"

	"github.com/mia0x75/pgparser/errors"
	"github.com/mia0x75/pgparser/hack"
	"github.com/mia0x75/pgparser/postgres"
	"github.com/mia0x75/pgparser/types"
)

// NewDecimal preserves the dialect's exact decimal representation.
func NewDecimal(str string) (any, error) {
	dec := new(types.PgDecimal)
	err := dec.FromString(hack.Slice(str))
	return dec, err
}

// NewHexLiteral keeps dialect decoding explicit instead of globally registered.
func NewHexLiteral(str string) (any, error) {
	return types.NewHexLiteral(str)
}

// NewBitLiteral keeps dialect decoding explicit instead of globally registered.
func NewBitLiteral(str string) (any, error) {
	return types.NewBitLiteral(str)
}

// ValueExpr is a runtime execution value, not an AST node.
type ValueExpr struct {
	types.Datum
	Type             types.FieldType
	flag             uint64
	projectionOffset int
}

// SetType preserves legacy literal metadata outside the shared syntax core.
func (n *ValueExpr) SetType(tp *types.FieldType) { n.Type = *tp }

// GetType preserves the legacy literal accessor during AST migration.
func (n *ValueExpr) GetType() *types.FieldType { return &n.Type }

// SetFlag keeps the legacy syntax cache local to the dialect.
func (n *ValueExpr) SetFlag(flag uint64) { n.flag = flag }

// GetFlag keeps the legacy syntax cache local to the dialect.
func (n *ValueExpr) GetFlag() uint64 { return n.flag }

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

// GetDatumString preserves the dialect ValueExpr string accessor.
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

// NewValueExpr retains existing literal pointers and assigns dialect defaults.
func NewValueExpr(value any) *ValueExpr {
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
