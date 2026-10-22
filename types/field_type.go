package types

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/mia0x75/pgparser/charset"
	"github.com/mia0x75/pgparser/format"
	"github.com/mia0x75/pgparser/postgres"
	"github.com/mia0x75/pgparser/types/json"
)

// UnspecifiedLength is unspecified length.
const (
	UnspecifiedLength = -1
)

// FieldType records field type information.
type FieldType struct {
	Tp      byte
	Flag    uint
	Flen    int
	Decimal int
	Charset string
	Collate string
	// Elems is the element list for enum and set type.
	Elems []string
}

// NewFieldType returns a FieldType,
// with a type and other information about field type.
func NewFieldType(tp byte) *FieldType {
	return &FieldType{
		Tp:      tp,
		Flen:    UnspecifiedLength,
		Decimal: UnspecifiedLength,
	}
}

// Clone returns a copy of itself.
func (ft *FieldType) Clone() *FieldType {
	ret := *ft
	return &ret
}

// Equal checks whether two FieldType objects are equal.
func (ft *FieldType) Equal(other *FieldType) bool {
	// We do not need to compare whole `ft.Flag == other.Flag` when wrapping cast upon an Expression.
	// but need compare unsigned_flag of ft.Flag.
	partialEqual := ft.Tp == other.Tp &&
		ft.Flen == other.Flen &&
		ft.Decimal == other.Decimal &&
		ft.Charset == other.Charset &&
		ft.Collate == other.Collate &&
		postgres.HasUnsignedFlag(ft.Flag) == postgres.HasUnsignedFlag(other.Flag)
	if !partialEqual || len(ft.Elems) != len(other.Elems) {
		return false
	}
	for i := range ft.Elems {
		if ft.Elems[i] != other.Elems[i] {
			return false
		}
	}
	return true
}

// EvalType gets the type in evaluation.
func (ft *FieldType) EvalType() EvalType {
	switch ft.Tp {
	case postgres.TypeTiny, postgres.TypeShort, postgres.TypeInt24, postgres.TypeLong, postgres.TypeLonglong,
		postgres.TypeBit, postgres.TypeYear:
		return ETInt
	case postgres.TypeFloat, postgres.TypeDouble:
		return ETReal
	case postgres.TypeNewDecimal:
		return ETDecimal
	case postgres.TypeDate, postgres.TypeDatetime:
		return ETDatetime
	case postgres.TypeTimestamp:
		return ETTimestamp
	case postgres.TypeDuration:
		return ETDuration
	case postgres.TypeJSON:
		return ETJson
	}
	return ETString
}

// Hybrid checks whether a type is a hybrid type, which can represent different types of value in specific context.
func (ft *FieldType) Hybrid() bool {
	return ft.Tp == postgres.TypeEnum || ft.Tp == postgres.TypeBit || ft.Tp == postgres.TypeSet
}

// Init initializes the FieldType data.
func (ft *FieldType) Init(tp byte) {
	ft.Tp = tp
	ft.Flen = UnspecifiedLength
	ft.Decimal = UnspecifiedLength
}

// CompactStr only considers Tp/CharsetBin/Flen/Deimal.
// This is used for showing column type in infoschema.
func (ft *FieldType) CompactStr() string {
	ts := TypeToStr(ft.Tp, ft.Charset)
	suffix := ""

	defaultFlen, defaultDecimal := postgres.GetDefaultFieldLengthAndDecimal(ft.Tp)
	isDecimalNotDefault := ft.Decimal != defaultDecimal && ft.Decimal != 0 && ft.Decimal != UnspecifiedLength

	// displayFlen and displayDecimal are flen and decimal values with `-1` substituted with default value.
	displayFlen, displayDecimal := ft.Flen, ft.Decimal
	if displayFlen == 0 || displayFlen == UnspecifiedLength {
		displayFlen = defaultFlen
	}
	if displayDecimal == 0 || displayDecimal == UnspecifiedLength {
		displayDecimal = defaultDecimal
	}

	switch ft.Tp {
	case postgres.TypeEnum, postgres.TypeSet:
		// Format is ENUM ('e1', 'e2') or SET ('e1', 'e2')
		es := make([]string, 0, len(ft.Elems))
		for _, e := range ft.Elems {
			e = format.OutputFormat(e)
			es = append(es, e)
		}
		suffix = fmt.Sprintf("('%s')", strings.Join(es, "','"))
	case postgres.TypeTimestamp, postgres.TypeDatetime, postgres.TypeDuration:
		if isDecimalNotDefault {
			suffix = fmt.Sprintf("(%d)", displayDecimal)
		}
	case postgres.TypeDouble, postgres.TypeFloat:
		// 1. Flen Not Default, Decimal Not Default -> Valid
		// 2. Flen Not Default, Decimal Default (-1) -> Invalid
		// 3. Flen Default, Decimal Not Default -> Valid
		// 4. Flen Default, Decimal Default -> Valid (hide)
		if isDecimalNotDefault {
			suffix = fmt.Sprintf("(%d,%d)", displayFlen, displayDecimal)
		}
	case postgres.TypeNewDecimal:
		suffix = fmt.Sprintf("(%d,%d)", displayFlen, displayDecimal)
	case postgres.TypeBit, postgres.TypeShort, postgres.TypeTiny, postgres.TypeInt24, postgres.TypeLong, postgres.TypeLonglong, postgres.TypeVarchar, postgres.TypeString, postgres.TypeVarString:
		// Flen is always shown.
		suffix = fmt.Sprintf("(%d)", displayFlen)
	case postgres.TypeYear:
		suffix = fmt.Sprintf("(%d)", ft.Flen)
	}
	return ts + suffix
}

// InfoSchemaStr joins the CompactStr with unsigned flag and
// returns a string.
func (ft *FieldType) InfoSchemaStr() string {
	suffix := ""
	if postgres.HasUnsignedFlag(ft.Flag) {
		suffix = " unsigned"
	}
	return ft.CompactStr() + suffix
}

// String joins the information of FieldType and returns a string.
// Note: when flen or decimal is unspecified, this function will use the default value instead of -1.
func (ft *FieldType) String() string {
	strs := []string{ft.CompactStr()}
	if postgres.HasUnsignedFlag(ft.Flag) {
		strs = append(strs, "UNSIGNED")
	}
	if postgres.HasZerofillFlag(ft.Flag) {
		strs = append(strs, "ZEROFILL")
	}
	if postgres.HasBinaryFlag(ft.Flag) && ft.Tp != postgres.TypeString {
		strs = append(strs, "BINARY")
	}

	if IsTypeChar(ft.Tp) || IsTypeBlob(ft.Tp) {
		if ft.Charset != "" && ft.Charset != charset.CharsetBin {
			strs = append(strs, fmt.Sprintf("CHARACTER SET %s", ft.Charset))
		}
		if ft.Collate != "" && ft.Collate != charset.CharsetBin {
			strs = append(strs, fmt.Sprintf("COLLATE %s", ft.Collate))
		}
	}

	return strings.Join(strs, " ")
}

// Restore implements Node interface.
func (ft *FieldType) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord(TypeToStr(ft.Tp, ft.Charset))

	switch ft.Tp {
	case postgres.TypeEnum, postgres.TypeSet:
		ctx.WritePlain("(")
		for i, e := range ft.Elems {
			if i != 0 {
				ctx.WritePlain(",")
			}
			ctx.WriteString(e)
		}
		ctx.WritePlain(")")
	case postgres.TypeTimestamp, postgres.TypeDatetime, postgres.TypeDuration:
		if ft.Flen > 0 && ft.Decimal > 0 {
			ctx.WritePlainf("(%d)", ft.Decimal)
		}
	case postgres.TypeDouble, postgres.TypeFloat:
		if ft.Flen > 0 && ft.Decimal > 0 {
			ctx.WritePlainf("(%d,%d)", ft.Flen, ft.Decimal)
		}
	case postgres.TypeNewDecimal:
		if ft.Flen > 0 && ft.Decimal > 0 {
			ctx.WritePlainf("(%d,%d)", ft.Flen, ft.Decimal)
		}
	case postgres.TypeBit, postgres.TypeShort, postgres.TypeTiny, postgres.TypeInt24, postgres.TypeLong, postgres.TypeLonglong, postgres.TypeVarchar, postgres.TypeString, postgres.TypeVarString, postgres.TypeTinyBlob, postgres.TypeMediumBlob, postgres.TypeBlob, postgres.TypeLongBlob, postgres.TypeYear:
		if ft.Flen > 0 {
			ctx.WritePlainf("(%d)", ft.Flen)
		}
	}

	if postgres.HasUnsignedFlag(ft.Flag) {
		ctx.WriteKeyWord(" UNSIGNED")
	}
	if postgres.HasZerofillFlag(ft.Flag) {
		ctx.WriteKeyWord(" ZEROFILL")
	}
	if postgres.HasBinaryFlag(ft.Flag) && ft.Charset != charset.CharsetBin {
		ctx.WriteKeyWord(" BINARY")
	}

	if IsTypeChar(ft.Tp) || IsTypeBlob(ft.Tp) {
		if ft.Charset != "" && ft.Charset != charset.CharsetBin {
			ctx.WriteKeyWord(" CHARACTER SET ")
			ctx.WritePlain(ft.Charset)
		}
		if ft.Collate != "" && ft.Collate != charset.CharsetBin {
			ctx.WriteKeyWord(" COLLATE ")
			ctx.WritePlain(ft.Collate)
		}
	}

	return nil
}

// RestoreAsCastType is used for write AST back to string.
func (ft *FieldType) RestoreAsCastType(ctx *format.RestoreCtx) {
	switch ft.Tp {
	case postgres.TypeVarString:
		if ft.Charset == charset.CharsetBin && ft.Collate == charset.CollationBin {
			ctx.WriteKeyWord("BINARY")
		} else {
			ctx.WriteKeyWord("CHAR")
		}
		if ft.Flen != UnspecifiedLength {
			ctx.WritePlainf("(%d)", ft.Flen)
		}
		if ft.Flag&postgres.BinaryFlag != 0 {
			ctx.WriteKeyWord(" BINARY")
		}
		if ft.Charset != charset.CharsetBin && ft.Charset != postgres.DefaultCharset {
			ctx.WriteKeyWord(" CHARACTER SET ")
			ctx.WritePlain(ft.Charset)
		}
	case postgres.TypeDate:
		ctx.WriteKeyWord("DATE")
	case postgres.TypeDatetime:
		ctx.WriteKeyWord("DATETIME")
		if ft.Decimal > 0 {
			ctx.WritePlainf("(%d)", ft.Decimal)
		}
	case postgres.TypeNewDecimal:
		ctx.WriteKeyWord("DECIMAL")
		if ft.Flen > 0 && ft.Decimal > 0 {
			ctx.WritePlainf("(%d, %d)", ft.Flen, ft.Decimal)
		} else if ft.Flen > 0 {
			ctx.WritePlainf("(%d)", ft.Flen)
		}
	case postgres.TypeDuration:
		ctx.WriteKeyWord("TIME")
		if ft.Decimal > 0 {
			ctx.WritePlainf("(%d)", ft.Decimal)
		}
	case postgres.TypeLonglong:
		if ft.Flag&postgres.UnsignedFlag != 0 {
			ctx.WriteKeyWord("UNSIGNED")
		} else {
			ctx.WriteKeyWord("SIGNED")
		}
	case postgres.TypeJSON:
		ctx.WriteKeyWord("JSON")
	}
}

// FormatAsCastType is used for write AST back to string.
func (ft *FieldType) FormatAsCastType(w io.Writer) {
	var sb strings.Builder
	restoreCtx := format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)
	ft.RestoreAsCastType(restoreCtx)
	fmt.Fprint(w, sb.String())
}

// VarStorageLen indicates this column is a variable length column.
const VarStorageLen = -1

// StorageLength is the length of stored value for the type.
func (ft *FieldType) StorageLength() int {
	switch ft.Tp {
	case postgres.TypeTiny, postgres.TypeShort, postgres.TypeInt24, postgres.TypeLong,
		postgres.TypeLonglong, postgres.TypeDouble, postgres.TypeFloat, postgres.TypeYear, postgres.TypeDuration,
		postgres.TypeDate, postgres.TypeDatetime, postgres.TypeTimestamp, postgres.TypeEnum, postgres.TypeSet,
		postgres.TypeBit:
		// This may not be the accurate length, because we may encode them as varint.
		return 8
	case postgres.TypeNewDecimal:
		precision, frac := ft.Flen-ft.Decimal, ft.Decimal
		return precision/digitsPerWord*wordSize + dig2bytes[precision%digitsPerWord] + frac/digitsPerWord*wordSize + dig2bytes[frac%digitsPerWord]
	default:
		return VarStorageLen
	}
}

// HasCharset indicates if a COLUMN has an associated charset. Returning false here prevents some information
// statements(like `SHOW CREATE TABLE`) from attaching a CHARACTER SET clause to the column.
func HasCharset(ft *FieldType) bool {
	switch ft.Tp {
	case postgres.TypeVarchar, postgres.TypeString, postgres.TypeVarString, postgres.TypeBlob,
		postgres.TypeTinyBlob, postgres.TypeMediumBlob, postgres.TypeLongBlob:
		return !postgres.HasBinaryFlag(ft.Flag)
	case postgres.TypeEnum, postgres.TypeSet:
		return true
	}
	return false
}

// AggFieldType aggregates field types for a multi-argument function like `IF`, `IFNULL`, `COALESCE`
// whose return type is determined by the arguments' FieldTypes.
// Aggregation is performed by MergeFieldType function.
func AggFieldType(tps []*FieldType) *FieldType {
	var currType FieldType
	for i, t := range tps {
		if i == 0 && currType.Tp == postgres.TypeUnspecified {
			currType = *t
			continue
		}
		mtp := MergeFieldType(currType.Tp, t.Tp)
		currType.Tp = mtp
	}

	return &currType
}

// AggregateEvalType aggregates arguments' EvalType of a multi-argument function.
func AggregateEvalType(fts []*FieldType, flag *uint) EvalType {
	var (
		aggregatedEvalType = ETString
		unsigned           bool
		gotFirst           bool
		gotBinString       bool
	)
	lft := fts[0]
	for _, ft := range fts {
		if ft.Tp == postgres.TypeNull {
			continue
		}
		et := ft.EvalType()
		rft := ft
		if (IsTypeBlob(ft.Tp) || IsTypeVarchar(ft.Tp) || IsTypeChar(ft.Tp)) && postgres.HasBinaryFlag(ft.Flag) {
			gotBinString = true
		}
		if !gotFirst {
			gotFirst = true
			aggregatedEvalType = et
			unsigned = postgres.HasUnsignedFlag(ft.Flag)
		} else {
			aggregatedEvalType = mergeEvalType(aggregatedEvalType, et, lft, rft, unsigned, postgres.HasUnsignedFlag(ft.Flag))
			unsigned = unsigned && postgres.HasUnsignedFlag(ft.Flag)
		}
		lft = rft
	}
	setTypeFlag(flag, postgres.UnsignedFlag, unsigned)
	setTypeFlag(flag, postgres.BinaryFlag, !aggregatedEvalType.IsStringKind() || gotBinString)
	return aggregatedEvalType
}

func mergeEvalType(lhs, rhs EvalType, lft, rft *FieldType, isLHSUnsigned, isRHSUnsigned bool) EvalType {
	if lft.Tp == postgres.TypeUnspecified || rft.Tp == postgres.TypeUnspecified {
		if lft.Tp == rft.Tp {
			return ETString
		}
		if lft.Tp == postgres.TypeUnspecified {
			lhs = rhs
		} else {
			rhs = lhs
		}
	}
	if lhs.IsStringKind() || rhs.IsStringKind() {
		return ETString
	} else if lhs == ETReal || rhs == ETReal {
		return ETReal
	} else if lhs == ETDecimal || rhs == ETDecimal || isLHSUnsigned != isRHSUnsigned {
		return ETDecimal
	}
	return ETInt
}

func setTypeFlag(flag *uint, flagItem uint, on bool) {
	if on {
		*flag |= flagItem
	} else {
		*flag &= ^flagItem
	}
}

// DefaultParamTypeForValue returns the default FieldType for the parameterized value.
func DefaultParamTypeForValue(value interface{}, tp *FieldType) {
	switch value.(type) {
	case nil:
		tp.Tp = postgres.TypeVarString
		tp.Flen = UnspecifiedLength
		tp.Decimal = UnspecifiedLength
	default:
		DefaultTypeForValue(value, tp)
		if hasVariantFieldLength(tp) {
			tp.Flen = UnspecifiedLength
		}
		if tp.Tp == postgres.TypeUnspecified {
			tp.Tp = postgres.TypeVarString
		}
	}
}

func hasVariantFieldLength(tp *FieldType) bool {
	switch tp.Tp {
	case postgres.TypeLonglong, postgres.TypeVarString, postgres.TypeDouble, postgres.TypeBlob,
		postgres.TypeBit, postgres.TypeDuration, postgres.TypeNewDecimal, postgres.TypeEnum, postgres.TypeSet:
		return true
	}
	return false
}

// DefaultTypeForValue returns the default FieldType for the value.
func DefaultTypeForValue(value interface{}, tp *FieldType) {
	switch x := value.(type) {
	case nil:
		tp.Tp = postgres.TypeNull
		tp.Flen = 0
		tp.Decimal = 0
		SetBinChsClnFlag(tp)
	case bool:
		tp.Tp = postgres.TypeLonglong
		tp.Flen = 1
		tp.Decimal = 0
		tp.Flag |= postgres.IsBooleanFlag
		SetBinChsClnFlag(tp)
	case int:
		tp.Tp = postgres.TypeLonglong
		tp.Flen = len(strconv.FormatInt(int64(x), 10))
		tp.Decimal = 0
		SetBinChsClnFlag(tp)
	case int64:
		tp.Tp = postgres.TypeLonglong
		tp.Flen = len(strconv.FormatInt(x, 10))
		tp.Decimal = 0
		SetBinChsClnFlag(tp)
	case uint64:
		tp.Tp = postgres.TypeLonglong
		tp.Flag |= postgres.UnsignedFlag
		tp.Flen = len(strconv.FormatUint(x, 10))
		tp.Decimal = 0
		SetBinChsClnFlag(tp)
	case string:
		tp.Tp = postgres.TypeVarString
		// TODO: tp.Flen should be len(x) * 3 (max bytes length of CharsetUTF8)
		tp.Flen = len(x)
		tp.Decimal = UnspecifiedLength
		tp.Charset, tp.Collate = charset.GetDefaultCharsetAndCollate()
	case float64:
		tp.Tp = postgres.TypeDouble
		s := strconv.FormatFloat(x, 'f', -1, 64)
		tp.Flen = len(s)
		tp.Decimal = UnspecifiedLength
		SetBinChsClnFlag(tp)
	case []byte:
		tp.Tp = postgres.TypeBlob
		tp.Flen = len(x)
		tp.Decimal = UnspecifiedLength
		SetBinChsClnFlag(tp)
	case BitLiteral:
		tp.Tp = postgres.TypeVarString
		tp.Flen = len(x)
		tp.Decimal = 0
		SetBinChsClnFlag(tp)
	case HexLiteral:
		tp.Tp = postgres.TypeVarString
		tp.Flen = len(x)
		tp.Decimal = 0
		tp.Flag |= postgres.UnsignedFlag
		SetBinChsClnFlag(tp)
	case BinaryLiteral:
		tp.Tp = postgres.TypeBit
		tp.Flen = len(x) * 8
		tp.Decimal = 0
		SetBinChsClnFlag(tp)
		tp.Flag &= ^postgres.BinaryFlag
		tp.Flag |= postgres.UnsignedFlag
	case Time:
		tp.Tp = x.Type
		switch x.Type {
		case postgres.TypeDate:
			tp.Flen = postgres.MaxDateWidth
			tp.Decimal = UnspecifiedLength
		case postgres.TypeDatetime, postgres.TypeTimestamp:
			tp.Flen = postgres.MaxDatetimeWidthNoFsp
			if x.Fsp > DefaultFsp { // consider point('.') and the fractional part.
				tp.Flen += x.Fsp + 1
			}
			tp.Decimal = x.Fsp
		}
		SetBinChsClnFlag(tp)
	case Duration:
		tp.Tp = postgres.TypeDuration
		tp.Flen = len(x.String())
		if x.Fsp > DefaultFsp { // consider point('.') and the fractional part.
			tp.Flen = x.Fsp + 1
		}
		tp.Decimal = x.Fsp
		SetBinChsClnFlag(tp)
	case *MyDecimal:
		tp.Tp = postgres.TypeNewDecimal
		tp.Flen = len(x.ToString())
		tp.Decimal = int(x.digitsFrac)
		SetBinChsClnFlag(tp)
	case Enum:
		tp.Tp = postgres.TypeEnum
		tp.Flen = len(x.Name)
		tp.Decimal = UnspecifiedLength
		SetBinChsClnFlag(tp)
	case Set:
		tp.Tp = postgres.TypeSet
		tp.Flen = len(x.Name)
		tp.Decimal = UnspecifiedLength
		SetBinChsClnFlag(tp)
	case json.BinaryJSON:
		tp.Tp = postgres.TypeJSON
		tp.Flen = UnspecifiedLength
		tp.Decimal = 0
		tp.Charset = charset.CharsetBin
		tp.Collate = charset.CollationBin
	default:
		tp.Tp = postgres.TypeUnspecified
		tp.Flen = UnspecifiedLength
		tp.Decimal = UnspecifiedLength
	}
}

// DefaultCharsetForType returns the default charset/collation for mysql type.
func DefaultCharsetForType(tp byte) (string, string) {
	switch tp {
	case postgres.TypeVarString, postgres.TypeString, postgres.TypeVarchar:
		// Default charset for string types is utf8.
		return postgres.DefaultCharset, postgres.DefaultCollationName
	}
	return charset.CharsetBin, charset.CollationBin
}

// MergeFieldType merges two MySQL type to a new type.
// This is used in hybrid field type expression.
// For example "select case c when 1 then 2 when 2 then 'x' from t;"
// The result field type of the case expression is the merged type of the two when clause.
// See https://github.com/mysql/mysql-server/blob/5.7/sql/field.cc#L1042
func MergeFieldType(a byte, b byte) byte {
	ia := getFieldTypeIndex(a)
	ib := getFieldTypeIndex(b)
	return fieldTypeMergeRules[ia][ib]
}

func getFieldTypeIndex(tp byte) int {
	itp := int(tp)
	if itp < fieldTypeTearFrom {
		return itp
	}
	return fieldTypeTearFrom + itp - fieldTypeTearTo - 1
}

const (
	fieldTypeTearFrom = int(postgres.TypeBit) + 1
	fieldTypeTearTo   = int(postgres.TypeJSON) - 1
	fieldTypeNum      = fieldTypeTearFrom + (255 - fieldTypeTearTo)
)

var fieldTypeMergeRules = [fieldTypeNum][fieldTypeNum]byte{
	/* postgres.TypeDecimal -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeNewDecimal, postgres.TypeNewDecimal,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeNewDecimal, postgres.TypeNewDecimal,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeDouble, postgres.TypeDouble,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeNewDecimal, postgres.TypeVarchar,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeDecimal, postgres.TypeDecimal,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeVarchar,
		//postgres.TypeJSON
		postgres.TypeVarchar,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeNewDecimal, postgres.TypeVarchar,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeVarchar, postgres.TypeTinyBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeBlob, postgres.TypeVarchar,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeString, postgres.TypeVarchar,
	},
	/* postgres.TypeTiny -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeNewDecimal, postgres.TypeTiny,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeShort, postgres.TypeLong,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeFloat, postgres.TypeDouble,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeTiny, postgres.TypeVarchar,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeLonglong, postgres.TypeInt24,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeVarchar, postgres.TypeTiny,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeVarchar,
		//postgres.TypeJSON
		postgres.TypeVarchar,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeNewDecimal, postgres.TypeVarchar,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeVarchar, postgres.TypeTinyBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeBlob, postgres.TypeVarchar,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeString, postgres.TypeVarchar,
	},
	/* postgres.TypeShort -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeNewDecimal, postgres.TypeShort,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeShort, postgres.TypeLong,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeFloat, postgres.TypeDouble,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeShort, postgres.TypeVarchar,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeLonglong, postgres.TypeInt24,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeVarchar, postgres.TypeShort,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeVarchar,
		//postgres.TypeJSON
		postgres.TypeVarchar,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeNewDecimal, postgres.TypeVarchar,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeVarchar, postgres.TypeTinyBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeBlob, postgres.TypeVarchar,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeString, postgres.TypeVarchar,
	},
	/* postgres.TypeLong -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeNewDecimal, postgres.TypeLong,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeLong, postgres.TypeLong,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeDouble, postgres.TypeDouble,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeLong, postgres.TypeVarchar,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeLonglong, postgres.TypeLong,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeVarchar, postgres.TypeLong,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeVarchar,
		//postgres.TypeJSON
		postgres.TypeVarchar,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeNewDecimal, postgres.TypeVarchar,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeVarchar, postgres.TypeTinyBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeBlob, postgres.TypeVarchar,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeString, postgres.TypeVarchar,
	},
	/* postgres.TypeFloat -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeDouble, postgres.TypeFloat,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeFloat, postgres.TypeDouble,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeFloat, postgres.TypeDouble,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeFloat, postgres.TypeVarchar,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeFloat, postgres.TypeFloat,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeVarchar, postgres.TypeFloat,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeVarchar,
		//postgres.TypeJSON
		postgres.TypeVarchar,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeDouble, postgres.TypeVarchar,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeVarchar, postgres.TypeTinyBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeBlob, postgres.TypeVarchar,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeString, postgres.TypeVarchar,
	},
	/* postgres.TypeDouble -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeDouble, postgres.TypeDouble,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeDouble, postgres.TypeDouble,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeDouble, postgres.TypeDouble,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeDouble, postgres.TypeVarchar,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeDouble, postgres.TypeDouble,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeVarchar, postgres.TypeDouble,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeVarchar,
		//postgres.TypeJSON
		postgres.TypeVarchar,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeDouble, postgres.TypeVarchar,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeVarchar, postgres.TypeTinyBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeBlob, postgres.TypeVarchar,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeString, postgres.TypeVarchar,
	},
	/* postgres.TypeNull -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeNewDecimal, postgres.TypeTiny,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeShort, postgres.TypeLong,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeFloat, postgres.TypeDouble,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeNull, postgres.TypeTimestamp,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeLonglong, postgres.TypeLonglong,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeDate, postgres.TypeDuration,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeDatetime, postgres.TypeYear,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeNewDate, postgres.TypeVarchar,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeBit,
		//postgres.TypeJSON
		postgres.TypeJSON,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeNewDecimal, postgres.TypeEnum,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeSet, postgres.TypeTinyBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeBlob, postgres.TypeVarchar,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeString, postgres.TypeGeometry,
	},
	/* postgres.TypeTimestamp -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeTimestamp, postgres.TypeTimestamp,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeDatetime, postgres.TypeDatetime,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeDatetime, postgres.TypeVarchar,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeNewDate, postgres.TypeVarchar,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeVarchar,
		//postgres.TypeJSON
		postgres.TypeVarchar,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeVarchar, postgres.TypeTinyBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeBlob, postgres.TypeVarchar,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeString, postgres.TypeVarchar,
	},
	/* postgres.TypeLonglong -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeNewDecimal, postgres.TypeLonglong,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeLonglong, postgres.TypeLonglong,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeDouble, postgres.TypeDouble,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeLonglong, postgres.TypeVarchar,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeLonglong, postgres.TypeLong,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeVarchar, postgres.TypeLonglong,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeNewDate, postgres.TypeVarchar,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeVarchar,
		//postgres.TypeJSON
		postgres.TypeVarchar,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeNewDecimal, postgres.TypeVarchar,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeVarchar, postgres.TypeTinyBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeBlob, postgres.TypeVarchar,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeString, postgres.TypeVarchar,
	},
	/* postgres.TypeInt24 -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeNewDecimal, postgres.TypeInt24,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeInt24, postgres.TypeLong,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeFloat, postgres.TypeDouble,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeInt24, postgres.TypeVarchar,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeLonglong, postgres.TypeInt24,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeVarchar, postgres.TypeInt24,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeNewDate, postgres.TypeVarchar,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeVarchar,
		//postgres.TypeJSON
		postgres.TypeVarchar,
		//postgres.TypeNewDecimal    postgres.TypeEnum
		postgres.TypeNewDecimal, postgres.TypeVarchar,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeVarchar, postgres.TypeTinyBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeBlob, postgres.TypeVarchar,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeString, postgres.TypeVarchar,
	},
	/* postgres.TypeDate -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeDate, postgres.TypeDatetime,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeDate, postgres.TypeDatetime,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeDatetime, postgres.TypeVarchar,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeNewDate, postgres.TypeVarchar,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeVarchar,
		//postgres.TypeJSON
		postgres.TypeVarchar,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeVarchar, postgres.TypeTinyBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeBlob, postgres.TypeVarchar,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeString, postgres.TypeVarchar,
	},
	/* postgres.TypeTime -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeDuration, postgres.TypeDatetime,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeDatetime, postgres.TypeDuration,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeDatetime, postgres.TypeVarchar,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeNewDate, postgres.TypeVarchar,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeVarchar,
		//postgres.TypeJSON
		postgres.TypeVarchar,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeVarchar, postgres.TypeTinyBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeBlob, postgres.TypeVarchar,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeString, postgres.TypeVarchar,
	},
	/* postgres.TypeDatetime -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeDatetime, postgres.TypeDatetime,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeDatetime, postgres.TypeDatetime,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeDatetime, postgres.TypeVarchar,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeNewDate, postgres.TypeVarchar,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeVarchar,
		//postgres.TypeJSON
		postgres.TypeVarchar,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeVarchar, postgres.TypeTinyBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeBlob, postgres.TypeVarchar,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeString, postgres.TypeVarchar,
	},
	/* postgres.TypeYear -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeDecimal, postgres.TypeTiny,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeShort, postgres.TypeLong,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeFloat, postgres.TypeDouble,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeYear, postgres.TypeVarchar,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeLonglong, postgres.TypeInt24,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeVarchar, postgres.TypeYear,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeVarchar,
		//postgres.TypeJSON
		postgres.TypeVarchar,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeNewDecimal, postgres.TypeVarchar,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeVarchar, postgres.TypeTinyBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeBlob, postgres.TypeVarchar,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeString, postgres.TypeVarchar,
	},
	/* postgres.TypeNewDate -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeNewDate, postgres.TypeDatetime,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeNewDate, postgres.TypeDatetime,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeDatetime, postgres.TypeVarchar,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeNewDate, postgres.TypeVarchar,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeVarchar,
		//postgres.TypeJSON
		postgres.TypeVarchar,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeVarchar, postgres.TypeTinyBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeBlob, postgres.TypeVarchar,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeString, postgres.TypeVarchar,
	},
	/* postgres.TypeVarchar -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeVarchar,
		//postgres.TypeJSON
		postgres.TypeVarchar,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeVarchar, postgres.TypeTinyBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeBlob, postgres.TypeVarchar,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeVarchar, postgres.TypeVarchar,
	},
	/* postgres.TypeBit -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeBit, postgres.TypeVarchar,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeBit,
		//postgres.TypeJSON
		postgres.TypeVarchar,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeVarchar, postgres.TypeTinyBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeBlob, postgres.TypeVarchar,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeString, postgres.TypeVarchar,
	},
	/* postgres.TypeJSON -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeNewFloat     postgres.TypeDouble
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeJSON, postgres.TypeVarchar,
		//postgres.TypeLongLONG     postgres.TypeInt24
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDate         MYSQL_TYPE_TIME
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDatetime     MYSQL_TYPE_YEAR
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeVarchar,
		//postgres.TypeJSON
		postgres.TypeJSON,
		//postgres.TypeNewDecimal   MYSQL_TYPE_ENUM
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeVarchar, postgres.TypeLongBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeLongBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeLongBlob, postgres.TypeVarchar,
		//postgres.TypeString       MYSQL_TYPE_GEOMETRY
		postgres.TypeString, postgres.TypeVarchar,
	},
	/* postgres.TypeNewDecimal -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeNewDecimal, postgres.TypeNewDecimal,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeNewDecimal, postgres.TypeNewDecimal,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeDouble, postgres.TypeDouble,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeNewDecimal, postgres.TypeVarchar,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeNewDecimal, postgres.TypeNewDecimal,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeVarchar, postgres.TypeNewDecimal,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeVarchar,
		//postgres.TypeJSON
		postgres.TypeVarchar,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeNewDecimal, postgres.TypeVarchar,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeVarchar, postgres.TypeTinyBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeBlob, postgres.TypeVarchar,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeString, postgres.TypeVarchar,
	},
	/* postgres.TypeEnum -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeEnum, postgres.TypeVarchar,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeVarchar,
		//postgres.TypeJSON
		postgres.TypeVarchar,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeVarchar, postgres.TypeTinyBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeBlob, postgres.TypeVarchar,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeString, postgres.TypeVarchar,
	},
	/* postgres.TypeSet -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeSet, postgres.TypeVarchar,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeVarchar,
		//postgres.TypeJSON
		postgres.TypeVarchar,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeVarchar, postgres.TypeTinyBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeBlob, postgres.TypeVarchar,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeString, postgres.TypeVarchar,
	},
	/* postgres.TypeTinyBlob -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeTinyBlob, postgres.TypeTinyBlob,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeTinyBlob, postgres.TypeTinyBlob,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeTinyBlob, postgres.TypeTinyBlob,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeTinyBlob, postgres.TypeTinyBlob,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeTinyBlob, postgres.TypeTinyBlob,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeTinyBlob, postgres.TypeTinyBlob,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeTinyBlob, postgres.TypeTinyBlob,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeTinyBlob, postgres.TypeTinyBlob,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeTinyBlob,
		//postgres.TypeJSON
		postgres.TypeLongBlob,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeTinyBlob, postgres.TypeTinyBlob,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeTinyBlob, postgres.TypeTinyBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeBlob, postgres.TypeTinyBlob,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeTinyBlob, postgres.TypeTinyBlob,
	},
	/* postgres.TypeMediumBlob -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeMediumBlob, postgres.TypeMediumBlob,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeMediumBlob, postgres.TypeMediumBlob,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeMediumBlob, postgres.TypeMediumBlob,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeMediumBlob, postgres.TypeMediumBlob,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeMediumBlob, postgres.TypeMediumBlob,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeMediumBlob, postgres.TypeMediumBlob,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeMediumBlob, postgres.TypeMediumBlob,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeMediumBlob, postgres.TypeMediumBlob,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeMediumBlob,
		//postgres.TypeJSON
		postgres.TypeLongBlob,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeMediumBlob, postgres.TypeMediumBlob,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeMediumBlob, postgres.TypeMediumBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeMediumBlob, postgres.TypeMediumBlob,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeMediumBlob, postgres.TypeMediumBlob,
	},
	/* postgres.TypeLongBlob -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeLongBlob, postgres.TypeLongBlob,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeLongBlob, postgres.TypeLongBlob,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeLongBlob, postgres.TypeLongBlob,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeLongBlob, postgres.TypeLongBlob,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeLongBlob, postgres.TypeLongBlob,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeLongBlob, postgres.TypeLongBlob,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeLongBlob, postgres.TypeLongBlob,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeLongBlob, postgres.TypeLongBlob,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeLongBlob,
		//postgres.TypeJSON
		postgres.TypeLongBlob,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeLongBlob, postgres.TypeLongBlob,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeLongBlob, postgres.TypeLongBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeLongBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeLongBlob, postgres.TypeLongBlob,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeLongBlob, postgres.TypeLongBlob,
	},
	/* postgres.TypeBlob -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeBlob, postgres.TypeBlob,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeBlob, postgres.TypeBlob,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeBlob, postgres.TypeBlob,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeBlob, postgres.TypeBlob,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeBlob, postgres.TypeBlob,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeBlob, postgres.TypeBlob,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeBlob, postgres.TypeBlob,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeBlob, postgres.TypeBlob,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeBlob,
		//postgres.TypeJSON
		postgres.TypeLongBlob,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeBlob, postgres.TypeBlob,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeBlob, postgres.TypeBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeBlob, postgres.TypeBlob,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeBlob, postgres.TypeBlob,
	},
	/* postgres.TypeVarString -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeVarchar,
		//postgres.TypeJSON
		postgres.TypeVarchar,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeVarchar, postgres.TypeTinyBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeBlob, postgres.TypeVarchar,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeVarchar, postgres.TypeVarchar,
	},
	/* postgres.TypeString -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeString, postgres.TypeString,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeString, postgres.TypeString,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeString, postgres.TypeString,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeString, postgres.TypeString,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeString, postgres.TypeString,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeString, postgres.TypeString,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeString, postgres.TypeString,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeString, postgres.TypeVarchar,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeString,
		//postgres.TypeJSON
		postgres.TypeString,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeString, postgres.TypeString,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeString, postgres.TypeTinyBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeBlob, postgres.TypeVarchar,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeString, postgres.TypeString,
	},
	/* postgres.TypeGeometry -> */
	{
		//postgres.TypeDecimal      postgres.TypeTiny
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeShort        postgres.TypeLong
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeFloat        postgres.TypeDouble
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeNull         postgres.TypeTimestamp
		postgres.TypeGeometry, postgres.TypeVarchar,
		//postgres.TypeLonglong     postgres.TypeInt24
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDate         postgres.TypeTime
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeDatetime     postgres.TypeYear
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeNewDate      postgres.TypeVarchar
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeBit          <16>-<244>
		postgres.TypeVarchar,
		//postgres.TypeJSON
		postgres.TypeVarchar,
		//postgres.TypeNewDecimal   postgres.TypeEnum
		postgres.TypeVarchar, postgres.TypeVarchar,
		//postgres.TypeSet          postgres.TypeTinyBlob
		postgres.TypeVarchar, postgres.TypeTinyBlob,
		//postgres.TypeMediumBlob  postgres.TypeLongBlob
		postgres.TypeMediumBlob, postgres.TypeLongBlob,
		//postgres.TypeBlob         postgres.TypeVarString
		postgres.TypeBlob, postgres.TypeVarchar,
		//postgres.TypeString       postgres.TypeGeometry
		postgres.TypeString, postgres.TypeGeometry,
	},
}

// SetBinChsClnFlag sets charset, collation as 'binary' and adds binaryFlag to FieldType.
func SetBinChsClnFlag(ft *FieldType) {
	ft.Charset = charset.CharsetBin
	ft.Collate = charset.CollationBin
	ft.Flag |= postgres.BinaryFlag
}
