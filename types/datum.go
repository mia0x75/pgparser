package types

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/mia0x75/pgparser/errors"
	"github.com/mia0x75/pgparser/hack"
	"github.com/mia0x75/pgparser/types/json"
)

// Kind constants.
const (
	KindNull          byte = 0
	KindInt64         byte = 1
	KindUint64        byte = 2
	KindFloat32       byte = 3
	KindFloat64       byte = 4
	KindString        byte = 5
	KindBytes         byte = 6
	KindBinaryLiteral byte = 7 // Used for BIT / HEX literals.
	KindDecimal       byte = 8
	KindDuration      byte = 9
	KindEnum          byte = 10
	KindBit           byte = 11 // Used for BIT table column values.
	KindSet           byte = 12
	KindTime          byte = 13
	KindInterface     byte = 14
	KindMinNotNull    byte = 15
	KindMaxValue      byte = 16
	KindRaw           byte = 17
	KindJSON          byte = 18
)

// Datum is a data box holds different kind of data.
// It has better performance and is easier to use than `interface{}`.
type Datum struct {
	k         byte   // datum kind.
	collation uint8  // collation can hold uint8 values.
	decimal   uint16 // decimal can hold uint16 values.
	length    uint32 // length can hold uint32 values.
	i         int64  // i can hold int64 uint64 float64 values.
	b         []byte // b can hold string or []byte values.
	x         any    // x hold all other types.
}

// Copy deep copies a Datum.
func (d *Datum) Copy() *Datum {
	ret := *d
	if d.b != nil {
		ret.b = make([]byte, len(d.b))
		copy(ret.b, d.b)
	}
	switch ret.Kind() {
	case KindDecimal:
		d := *d.GetDecimal()
		ret.SetDecimal(&d)
	case KindTime:
		ret.SetPgTime(d.GetPgTime())
	}
	return &ret
}

// Kind gets the kind of the datum.
func (d *Datum) Kind() byte {
	return d.k
}

// Collation gets the collation of the datum.
func (d *Datum) Collation() byte {
	return d.collation
}

// Frac gets the frac of the datum.
func (d *Datum) Frac() int {
	return int(d.decimal)
}

// SetFrac sets the frac of the datum.
func (d *Datum) SetFrac(frac int) {
	d.decimal = uint16(frac)
}

// Length gets the length of the datum.
func (d *Datum) Length() int {
	return int(d.length)
}

// IsNull checks if datum is null.
func (d *Datum) IsNull() bool {
	return d.k == KindNull
}

// GetInt64 gets int64 value.
func (d *Datum) GetInt64() int64 {
	return d.i
}

// SetInt64 sets int64 value.
func (d *Datum) SetInt64(i int64) {
	d.k = KindInt64
	d.i = i
}

// GetUint64 gets uint64 value.
func (d *Datum) GetUint64() uint64 {
	return uint64(d.i)
}

// SetUint64 sets uint64 value.
func (d *Datum) SetUint64(i uint64) {
	d.k = KindUint64
	d.i = int64(i)
}

// GetFloat64 gets float64 value.
func (d *Datum) GetFloat64() float64 {
	return math.Float64frombits(uint64(d.i))
}

// SetFloat64 sets float64 value.
func (d *Datum) SetFloat64(f float64) {
	d.k = KindFloat64
	d.i = int64(math.Float64bits(f))
}

// GetFloat32 gets float32 value.
func (d *Datum) GetFloat32() float32 {
	return float32(math.Float64frombits(uint64(d.i)))
}

// SetFloat32 sets float32 value.
func (d *Datum) SetFloat32(f float32) {
	d.k = KindFloat32
	d.i = int64(math.Float64bits(float64(f)))
}

// GetString gets string value.
func (d *Datum) GetString() string {
	return string(d.b)
}

// SetString sets string value.
func (d *Datum) SetString(s string) {
	d.k = KindString
	d.b = []byte(s)
}

// sink prevents s from being allocated on the stack.
var sink = func(s string) {
}

// GetBytes gets bytes value.
func (d *Datum) GetBytes() []byte {
	return d.b
}

// SetBytes sets bytes value to datum.
func (d *Datum) SetBytes(b []byte) {
	d.k = KindBytes
	d.b = b
}

// GetInterface gets interface value.
func (d *Datum) GetInterface() any {
	return d.x
}

// SetInterface sets interface to datum.
func (d *Datum) SetInterface(x any) {
	d.k = KindInterface
	d.x = x
}

// SetNull sets datum to nil.
func (d *Datum) SetNull() {
	d.k = KindNull
	d.x = nil
}

// GetBinaryLiteral gets Bit value
func (d *Datum) GetBinaryLiteral() BinaryLiteral {
	return d.b
}

// SetBinaryLiteral sets Bit value
func (d *Datum) SetBinaryLiteral(b BinaryLiteral) {
	d.k = KindBinaryLiteral
	d.b = b
}

// SetPgBit stores a BinaryLiteral with KindBit rather than KindBinaryLiteral.
func (d *Datum) SetPgBit(b BinaryLiteral) {
	d.k = KindBit
	d.b = b
}

// GetDecimal exposes the stored pointer; use Copy before modifying a shared datum.
func (d *Datum) GetDecimal() *Decimal {
	return d.x.(*Decimal)
}

// SetDecimal retains the pointer, so subsequent changes to b are visible through the datum.
func (d *Datum) SetDecimal(b *Decimal) {
	d.k = KindDecimal
	d.x = b
}

// GetPgDecimal is kept for source compatibility.
func (d *Datum) GetPgDecimal() *PgDecimal { return d.GetDecimal() }

// SetPgDecimal is kept for source compatibility.
func (d *Datum) SetPgDecimal(b *PgDecimal) { d.SetDecimal(b) }

// GetPgDuration gets Duration value
func (d *Datum) GetPgDuration() Duration {
	return Duration{Duration: time.Duration(d.i), Fsp: int(d.decimal)}
}

// SetPgDuration sets Duration value
func (d *Datum) SetPgDuration(b Duration) {
	d.k = KindDuration
	d.i = int64(b.Duration)
	d.decimal = uint16(b.Fsp)
}

// GetPgEnum gets Enum value
func (d *Datum) GetPgEnum() Enum {
	str := string(hack.String(d.b))
	return Enum{Value: uint64(d.i), Name: str}
}

// SetPgEnum sets Enum value
func (d *Datum) SetPgEnum(b Enum) {
	d.k = KindEnum
	d.i = int64(b.Value)
	sink(b.Name)
	d.b = hack.Slice(b.Name)
}

// GetPgSet gets Set value
func (d *Datum) GetPgSet() Set {
	str := string(hack.String(d.b))
	return Set{Value: uint64(d.i), Name: str}
}

// SetPgSet sets Set value
func (d *Datum) SetPgSet(b Set) {
	d.k = KindSet
	d.i = int64(b.Value)
	sink(b.Name)
	d.b = hack.Slice(b.Name)
}

// GetPgJSON gets json.BinaryJSON value
func (d *Datum) GetPgJSON() json.BinaryJSON {
	return json.BinaryJSON{TypeCode: byte(d.i), Value: d.b}
}

// SetPgJSON sets json.BinaryJSON value
func (d *Datum) SetPgJSON(b json.BinaryJSON) {
	d.k = KindJSON
	d.i = int64(b.TypeCode)
	d.b = b.Value
}

// GetPgTime gets types.Time value
func (d *Datum) GetPgTime() Time {
	return d.x.(Time)
}

// SetPgTime sets types.Time value
func (d *Datum) SetPgTime(b Time) {
	d.k = KindTime
	d.x = b
}

// GetValue gets the value of the datum of any kind.
func (d *Datum) GetValue() any {
	switch d.k {
	case KindInt64:
		return d.GetInt64()
	case KindUint64:
		return d.GetUint64()
	case KindFloat32:
		return d.GetFloat32()
	case KindFloat64:
		return d.GetFloat64()
	case KindString:
		return d.GetString()
	case KindBytes:
		return d.GetBytes()
	case KindDecimal:
		return d.GetDecimal()
	case KindDuration:
		return d.GetPgDuration()
	case KindEnum:
		return d.GetPgEnum()
	case KindBinaryLiteral, KindBit:
		return d.GetBinaryLiteral()
	case KindSet:
		return d.GetPgSet()
	case KindJSON:
		return d.GetPgJSON()
	case KindTime:
		return d.GetPgTime()
	default:
		return d.GetInterface()
	}
}

// SetValue sets any kind of value.
func (d *Datum) SetValue(val any) {
	switch x := val.(type) {
	case nil:
		d.SetNull()
	case bool:
		if x {
			d.SetInt64(1)
		} else {
			d.SetInt64(0)
		}
	case int:
		d.SetInt64(int64(x))
	case int64:
		d.SetInt64(x)
	case uint64:
		d.SetUint64(x)
	case float32:
		d.SetFloat32(x)
	case float64:
		d.SetFloat64(x)
	case string:
		d.SetString(x)
	case []byte:
		d.SetBytes(x)
	case *Decimal:
		d.SetDecimal(x)
	case Duration:
		d.SetPgDuration(x)
	case Enum:
		d.SetPgEnum(x)
	case BinaryLiteral:
		d.SetBinaryLiteral(x)
	case BitLiteral: // Store as BinaryLiteral for Bit and Hex literals
		d.SetBinaryLiteral(BinaryLiteral(x))
	case HexLiteral:
		d.SetBinaryLiteral(BinaryLiteral(x))
	case Set:
		d.SetPgSet(x)
	case json.BinaryJSON:
		d.SetPgJSON(x)
	case Time:
		d.SetPgTime(x)
	default:
		d.SetInterface(x)
	}
}

// ToString gets the string representation of the datum.
func (d *Datum) ToString() (string, error) {
	switch d.Kind() {
	case KindInt64:
		return strconv.FormatInt(d.GetInt64(), 10), nil
	case KindUint64:
		return strconv.FormatUint(d.GetUint64(), 10), nil
	case KindFloat32:
		return strconv.FormatFloat(float64(d.GetFloat32()), 'f', -1, 32), nil
	case KindFloat64:
		return strconv.FormatFloat(d.GetFloat64(), 'f', -1, 64), nil
	case KindString:
		return d.GetString(), nil
	case KindBytes:
		return d.GetString(), nil
	case KindTime:
		return d.GetPgTime().String(), nil
	case KindDuration:
		return d.GetPgDuration().String(), nil
	case KindDecimal:
		return d.GetDecimal().String(), nil
	case KindEnum:
		return d.GetPgEnum().String(), nil
	case KindSet:
		return d.GetPgSet().String(), nil
	case KindJSON:
		return d.GetPgJSON().String(), nil
	case KindBinaryLiteral, KindBit:
		return d.GetBinaryLiteral().ToString(), nil
	default:
		return "", errors.Errorf("cannot convert %v(type %T) to string", d.GetValue(), d.GetValue())
	}
}

// NewDatum creates a new Datum from an interface{}.
func NewDatum(in any) (d Datum) {
	switch x := in.(type) {
	case []any:
		d.SetValue(MakeDatums(x...))
	default:
		d.SetValue(in)
	}
	return d
}

// NewIntDatum creates a new Datum from an int64 value.
func NewIntDatum(i int64) (d Datum) {
	d.SetInt64(i)
	return d
}

// NewUintDatum creates a new Datum from an uint64 value.
func NewUintDatum(i uint64) (d Datum) {
	d.SetUint64(i)
	return d
}

// NewBytesDatum creates a new Datum from a byte slice.
func NewBytesDatum(b []byte) (d Datum) {
	d.SetBytes(b)
	return d
}

// NewStringDatum creates a new Datum from a string.
func NewStringDatum(s string) (d Datum) {
	d.SetString(s)
	return d
}

// NewFloat64Datum creates a new Datum from a float64 value.
func NewFloat64Datum(f float64) (d Datum) {
	d.SetFloat64(f)
	return d
}

// NewFloat32Datum creates a new Datum from a float32 value.
func NewFloat32Datum(f float32) (d Datum) {
	d.SetFloat32(f)
	return d
}

// NewTimeDatum creates a new Time from a Time value.
func NewTimeDatum(t Time) (d Datum) {
	d.SetPgTime(t)
	return d
}

// NewDecimalDatum creates a new Datum form a Decimal value.
func NewDecimalDatum(dec *Decimal) (d Datum) {
	d.SetDecimal(dec)
	return d
}

// NewBinaryLiteralDatum creates a new BinaryLiteral Datum for a BinaryLiteral value.
func NewBinaryLiteralDatum(b BinaryLiteral) (d Datum) {
	d.SetBinaryLiteral(b)
	return d
}

// MakeDatums creates datum slice from interfaces.
func MakeDatums(args ...any) []Datum {
	datums := make([]Datum, len(args))
	for i, v := range args {
		datums[i] = NewDatum(v)
	}
	return datums
}

// DatumsToString converts several datums to formatted string.
func DatumsToString(datums []Datum, handleSpecialValue bool) (string, error) {
	var strs []string
	for _, datum := range datums {
		if handleSpecialValue {
			switch datum.Kind() {
			case KindNull:
				strs = append(strs, "NULL")
				continue
			case KindMinNotNull:
				strs = append(strs, "-inf")
				continue
			case KindMaxValue:
				strs = append(strs, "+inf")
				continue
			}
		}
		str, err := datum.ToString()
		if err != nil {
			return "", errors.Trace(err)
		}
		strs = append(strs, str)
	}
	size := len(datums)
	if size > 1 {
		strs[0] = "(" + strs[0]
		strs[size-1] += ")"
	}
	return strings.Join(strs, ", "), nil
}
