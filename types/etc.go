package types

import (
	"io"
	"strings"

	"github.com/mia0x75/pgparser/charset"
	"github.com/mia0x75/pgparser/errors"
	"github.com/mia0x75/pgparser/opcode"
	"github.com/mia0x75/pgparser/postgres"
	"github.com/mia0x75/pgparser/sqlerr"
)

// IsTypeBlob returns a boolean indicating whether the tp is a blob type.
func IsTypeBlob(tp byte) bool {
	switch tp {
	case postgres.TypeTinyBlob, postgres.TypeMediumBlob, postgres.TypeBlob, postgres.TypeLongBlob:
		return true
	default:
		return false
	}
}

// IsTypeChar returns a boolean indicating
// whether the tp is the char type like a string type or a varchar type.
func IsTypeChar(tp byte) bool {
	return tp == postgres.TypeString || tp == postgres.TypeVarchar
}

var type2Str = map[byte]string{
	postgres.TypeBit:        "bit",
	postgres.TypeBlob:       "text",
	postgres.TypeDate:       "date",
	postgres.TypeDatetime:   "datetime",
	postgres.TypeDecimal:    "unspecified",
	postgres.TypeNewDecimal: "decimal",
	postgres.TypeDouble:     "double",
	postgres.TypeEnum:       "enum",
	postgres.TypeFloat:      "float",
	postgres.TypeGeometry:   "geometry",
	postgres.TypeInt24:      "mediumint",
	postgres.TypeJSON:       "json",
	postgres.TypeLong:       "int",
	postgres.TypeLonglong:   "bigint",
	postgres.TypeLongBlob:   "longtext",
	postgres.TypeMediumBlob: "mediumtext",
	postgres.TypeNull:       "null",
	postgres.TypeSet:        "set",
	postgres.TypeShort:      "smallint",
	postgres.TypeString:     "char",
	postgres.TypeDuration:   "time",
	postgres.TypeTimestamp:  "timestamp",
	postgres.TypeTiny:       "tinyint",
	postgres.TypeTinyBlob:   "tinytext",
	postgres.TypeVarchar:    "varchar",
	postgres.TypeVarString:  "var_string",
	postgres.TypeYear:       "year",
}

// TypeStr converts tp to a string.
func TypeStr(tp byte) (r string) {
	return type2Str[tp]
}

// TypeToStr converts a field to a string.
// It is used for converting Text to Blob,
// or converting Char to Binary.
// Args:
//
//	tp: type enum
//	cs: charset
func TypeToStr(tp byte, cs string) (r string) {
	ts := type2Str[tp]
	if cs != "binary" {
		return ts
	}
	if IsTypeBlob(tp) {
		ts = strings.Replace(ts, "text", "blob", 1)
	} else if IsTypeChar(tp) {
		ts = strings.Replace(ts, "char", "binary", 1)
	}
	return ts
}

// ErrInvalidDefault is returned when meet a invalid default value.
var ErrInvalidDefault = sqlerr.ClassTypes.New(codeInvalidDefault, "Invalid default value for '%s'")

// IsTypeVarchar returns a boolean indicating
// whether the tp is the varchar type like a varstring type or a varchar type.
func IsTypeVarchar(tp byte) bool {
	return tp == postgres.TypeVarString || tp == postgres.TypeVarchar
}

// IsTypeUnspecified returns a boolean indicating whether the tp is the Unspecified type.
func IsTypeUnspecified(tp byte) bool {
	return tp == postgres.TypeUnspecified
}

// IsTypePrefixable returns a boolean indicating
// whether an index on a column with the tp can be defined with a prefix.
func IsTypePrefixable(tp byte) bool {
	return IsTypeBlob(tp) || IsTypeChar(tp)
}

// IsTypeFractionable returns a boolean indicating
// whether the tp can has time fraction.
func IsTypeFractionable(tp byte) bool {
	return tp == postgres.TypeDatetime || tp == postgres.TypeDuration || tp == postgres.TypeTimestamp
}

// IsTypeTime returns a boolean indicating
// whether the tp is time type like datetime, date or timestamp.
func IsTypeTime(tp byte) bool {
	return tp == postgres.TypeDatetime || tp == postgres.TypeDate || tp == postgres.TypeTimestamp
}

// IsTypeNumeric returns a boolean indicating whether the tp is numeric type.
func IsTypeNumeric(tp byte) bool {
	switch tp {
	case postgres.TypeBit, postgres.TypeTiny, postgres.TypeInt24, postgres.TypeLong, postgres.TypeLonglong, postgres.TypeNewDecimal,
		postgres.TypeDecimal, postgres.TypeFloat, postgres.TypeDouble, postgres.TypeShort:
		return true
	}
	return false
}

// IsTemporalWithDate returns a boolean indicating
// whether the tp is time type with date.
func IsTemporalWithDate(tp byte) bool {
	return IsTypeTime(tp)
}

// IsBinaryStr returns a boolean indicating
// whether the field type is a binary string type.
func IsBinaryStr(ft *FieldType) bool {
	if ft.Collate == charset.CollationBin && IsString(ft.Tp) {
		return true
	}
	return false
}

// IsNonBinaryStr returns a boolean indicating
// whether the field type is a non-binary string type.
func IsNonBinaryStr(ft *FieldType) bool {
	if ft.Collate != charset.CollationBin && IsString(ft.Tp) {
		return true
	}
	return false
}

// IsString returns a boolean indicating
// whether the field type is a string type.
func IsString(tp byte) bool {
	return IsTypeChar(tp) || IsTypeBlob(tp) || IsTypeVarchar(tp) || IsTypeUnspecified(tp)
}

var kind2Str = map[byte]string{
	KindNull:          "null",
	KindInt64:         "bigint",
	KindUint64:        "unsigned bigint",
	KindFloat32:       "float",
	KindFloat64:       "double",
	KindString:        "char",
	KindBytes:         "bytes",
	KindBinaryLiteral: "bit/hex literal",
	KindMysqlDecimal:  "decimal",
	KindMysqlDuration: "time",
	KindMysqlEnum:     "enum",
	KindMysqlBit:      "bit",
	KindMysqlSet:      "set",
	KindMysqlTime:     "datetime",
	KindInterface:     "interface",
	KindMinNotNull:    "min_not_null",
	KindMaxValue:      "max_value",
	KindRaw:           "raw",
	KindMysqlJSON:     "json",
}

// KindStr converts kind to a string.
func KindStr(kind byte) (r string) {
	return kind2Str[kind]
}

// EOFAsNil filtrates errors,
// If err is equal to io.EOF returns nil.
func EOFAsNil(err error) error {
	if sqlerr.ErrorEqual(err, io.EOF) {
		return nil
	}
	return errors.Trace(err)
}

// InvOp2 returns an invalid operation error.
func InvOp2(x, y interface{}, o opcode.Op) (interface{}, error) {
	return nil, errors.Errorf("Invalid operation: %v %v %v (mismatched types %T and %T)", x, o, y, x, y)
}

// overflow returns an overflowed error.
func overflow(v interface{}, tp byte) error {
	return ErrOverflow.GenWithStack("constant %v overflows %s", v, TypeStr(tp))
}

// IsTypeTemporal checks if a type is a temporal type.
func IsTypeTemporal(tp byte) bool {
	switch tp {
	case postgres.TypeDuration, postgres.TypeDatetime, postgres.TypeTimestamp,
		postgres.TypeDate, postgres.TypeNewDate:
		return true
	}
	return false
}
