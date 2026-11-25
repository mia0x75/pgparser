package ast

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	pkast "github.com/mia0x75/parserkit/ast"

	"github.com/mia0x75/pgparser/errors"
	"github.com/mia0x75/pgparser/format"
	"github.com/mia0x75/pgparser/types"
)

// LiteralExpr keeps literal syntax without an evaluated value or inferred type.
type LiteralExpr struct {
	pkast.BaseLiteralExpr
}

// NewLiteralExpr returns a syntax literal with canonical text.
func NewLiteralExpr(kind pkast.LiteralKind, value string) *LiteralExpr {
	return &LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: kind, Value: value}}
}

// NewValueExpr converts a decoded value into a syntax literal. It is the
// grammar-facing constructor: callers that already hold a LiteralExpr keep it.
func NewValueExpr(value any) ValueExpr {
	switch v := value.(type) {
	case nil:
		return NewLiteralExpr(pkast.LiteralNull, "")
	case bool:
		return NewLiteralExpr(pkast.LiteralBoolean, strconv.FormatBool(v))
	case string:
		return NewLiteralExpr(pkast.LiteralString, v)
	case []byte:
		return NewLiteralExpr(pkast.LiteralBytes, string(v))
	case int:
		return NewLiteralExpr(pkast.LiteralInteger, strconv.Itoa(v))
	case int64:
		return NewLiteralExpr(pkast.LiteralInteger, strconv.FormatInt(v, 10))
	case uint64:
		return NewLiteralExpr(pkast.LiteralInteger, strconv.FormatUint(v, 10))
	case float64:
		return NewLiteralExpr(pkast.LiteralFloat, strconv.FormatFloat(v, 'e', -1, 64))
	case float32:
		return NewLiteralExpr(pkast.LiteralFloat, strconv.FormatFloat(float64(v), 'e', -1, 32))
	case *types.PgDecimal:
		return NewLiteralExpr(pkast.LiteralDecimal, v.String())
	case ValueExpr:
		return v
	default:
		return NewLiteralExpr(pkast.LiteralString, fmt.Sprint(value))
	}
}

// GetValue decodes the literal for legacy callers that inspect parsed values.
func (n *LiteralExpr) GetValue() any {
	if n == nil {
		return nil
	}
	switch n.Kind {
	case pkast.LiteralNull:
		return nil
	case pkast.LiteralBoolean:
		return strings.EqualFold(n.Value, "true")
	case pkast.LiteralInteger:
		if value, err := strconv.ParseInt(n.Value, 10, 64); err == nil {
			return value
		}
	case pkast.LiteralFloat, pkast.LiteralDecimal:
		if value, err := strconv.ParseFloat(n.Value, 64); err == nil {
			return value
		}
	}
	return n.Value
}

// GetString returns the literal text.
func (n *LiteralExpr) GetString() string {
	if n == nil {
		return ""
	}
	return n.Value
}

// Restore implements Node interface.
func (n *LiteralExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore LiteralExpr: nil literal")
	}
	switch n.Kind {
	case pkast.LiteralNull:
		ctx.WriteKeyword("NULL")
	case pkast.LiteralBoolean:
		switch {
		case strings.EqualFold(n.Value, "true"):
			ctx.WriteKeyword("TRUE")
		case strings.EqualFold(n.Value, "false"):
			ctx.WriteKeyword("FALSE")
		default:
			return errors.Errorf("unsupported boolean literal value %q", n.Value)
		}
	case pkast.LiteralInteger, pkast.LiteralDecimal, pkast.LiteralFloat:
		if !validNumericLiteral(n.Value) {
			return errors.Errorf("invalid numeric literal %q", n.Value)
		}
		ctx.WritePlain(n.Value)
	case pkast.LiteralString, pkast.LiteralBytes:
		ctx.WriteString(n.Value)
	case pkast.LiteralHex, pkast.LiteralBit:
		for _, digit := range n.Value {
			valid := digit == '0' || digit == '1'
			if n.Kind == pkast.LiteralHex {
				valid = digit >= '0' && digit <= '9' || digit >= 'a' && digit <= 'f' || digit >= 'A' && digit <= 'F'
			}
			if !valid {
				return errors.New("invalid binary literal digits")
			}
		}
		if n.Kind == pkast.LiteralHex {
			ctx.WritePlain("x'")
		} else {
			ctx.WritePlain("b'")
		}
		ctx.WritePlain(n.Value)
		ctx.WritePlain("'")
	default:
		return errors.New("unsupported literal kind")
	}
	return ctx.Err()
}

// Format formats the literal into a Writer for diagnostics.
func (n *LiteralExpr) Format(w io.Writer) {
	switch n.Kind {
	case pkast.LiteralString, pkast.LiteralBytes:
		_, _ = fmt.Fprint(w, strconv.Quote(n.Value))
	case pkast.LiteralHex:
		_, _ = fmt.Fprintf(w, "x'%s'", n.Value)
	case pkast.LiteralBit:
		_, _ = fmt.Fprintf(w, "b'%s'", n.Value)
	default:
		_, _ = fmt.Fprint(w, n.Value)
	}
}

// Accept implements Node Accept interface.
func (n *LiteralExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, _ := v.Enter(n)
	outer, ok := entered.(*LiteralExpr)
	if !ok || outer == nil {
		return entered, false
	}
	return v.Leave(outer)
}

func validNumericLiteral(value string) bool {
	if value == "" {
		return false
	}
	i := 0
	if value[i] == '+' || value[i] == '-' {
		i++
	}
	if strings.HasPrefix(value[i:], "0x") || strings.HasPrefix(value[i:], "0X") {
		return validDigits(value[i+2:], 16)
	}
	if strings.HasPrefix(value[i:], "0o") || strings.HasPrefix(value[i:], "0O") {
		return validDigits(value[i+2:], 8)
	}
	if strings.HasPrefix(value[i:], "0b") || strings.HasPrefix(value[i:], "0B") {
		return validDigits(value[i+2:], 2)
	}
	integerStart := i
	for i < len(value) && value[i] >= '0' && value[i] <= '9' {
		i++
	}
	integerDigits := i - integerStart
	fractionDigits := 0
	if i < len(value) && value[i] == '.' {
		i++
		fractionStart := i
		for i < len(value) && value[i] >= '0' && value[i] <= '9' {
			i++
		}
		fractionDigits = i - fractionStart
	}
	if integerDigits == 0 && fractionDigits == 0 {
		return false
	}
	if i < len(value) && (value[i] == 'e' || value[i] == 'E') {
		i++
		if i < len(value) && (value[i] == '+' || value[i] == '-') {
			i++
		}
		exponentStart := i
		for i < len(value) && value[i] >= '0' && value[i] <= '9' {
			i++
		}
		if i == exponentStart {
			return false
		}
	}
	return i == len(value)
}

// validDigits accepts the optional separators the lexer has already validated.
func validDigits(value string, base int) bool {
	digits := 0
	for _, digit := range value {
		if digit == '_' {
			continue
		}
		digits++
		valid := digit >= '0' && digit <= '9'
		switch base {
		case 2:
			valid = digit == '0' || digit == '1'
		case 8:
			valid = digit >= '0' && digit <= '7'
		case 16:
			valid = digit >= '0' && digit <= '9' || digit >= 'a' && digit <= 'f' || digit >= 'A' && digit <= 'F'
		}
		if !valid {
			return false
		}
	}
	return digits != 0
}
