package types

import (
	"github.com/mia0x75/pgparser/errors"
	"github.com/mia0x75/pgparser/opcode"
)

// ComputePlus computes the result of a+b.
func ComputePlus(a, b Datum) (d Datum, err error) {
	switch a.Kind() {
	case KindInt64:
		switch b.Kind() {
		case KindInt64:
			r, err1 := AddInt64(a.GetInt64(), b.GetInt64())
			d.SetInt64(r)
			return d, errors.Trace(err1)
		case KindUint64:
			r, err1 := AddInteger(b.GetUint64(), a.GetInt64())
			d.SetUint64(r)
			return d, errors.Trace(err1)
		}
	case KindUint64:
		switch b.Kind() {
		case KindInt64:
			r, err1 := AddInteger(a.GetUint64(), b.GetInt64())
			d.SetUint64(r)
			return d, errors.Trace(err1)
		case KindUint64:
			r, err1 := AddUint64(a.GetUint64(), b.GetUint64())
			d.SetUint64(r)
			return d, errors.Trace(err1)
		}
	case KindFloat64:
		if b.Kind() == KindFloat64 {
			r := a.GetFloat64() + b.GetFloat64()
			d.SetFloat64(r)
			return d, nil
		}
	case KindDecimal:
		if b.Kind() == KindDecimal {
			r := new(PgDecimal)
			err = DecimalAdd(a.GetPgDecimal(), b.GetPgDecimal(), r)
			d.SetPgDecimal(r)
			d.SetFrac(max(a.Frac(), b.Frac()))
			return d, err
		}
	}
	_, err = InvOp2(a.GetValue(), b.GetValue(), opcode.Plus)
	return d, err
}
