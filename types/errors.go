package types

import (
	"github.com/mia0x75/pgparser/postgres"
	"github.com/mia0x75/pgparser/sqlerr"
)

var (
	// ErrDataTooLong is returned when converts a string value that is longer than field type length.
	ErrDataTooLong = sqlerr.ClassTypes.New(codeDataTooLong, "Data Too Long")
	// ErrIllegalValueForType is returned when value of type is illegal.
	ErrIllegalValueForType = sqlerr.ClassTypes.New(codeIllegalValueForType, postgres.MySQLErrName[postgres.ErrIllegalValueForType])
	// ErrTruncated is returned when data has been truncated during conversion.
	ErrTruncated = sqlerr.ClassTypes.New(codeTruncated, "Data Truncated")
	// ErrTruncatedWrongVal is returned when data has been truncated during conversion.
	ErrTruncatedWrongVal = sqlerr.ClassTypes.New(codeTruncatedWrongValue, msgTruncatedWrongVal)
	// ErrOverflow is returned when data is out of range for a field type.
	ErrOverflow = sqlerr.ClassTypes.New(codeOverflow, msgOverflow)
	// ErrDivByZero is return when do division by 0.
	ErrDivByZero = sqlerr.ClassTypes.New(codeDivByZero, "Division by 0")
	// ErrTooBigDisplayWidth is return when display width out of range for column.
	ErrTooBigDisplayWidth = sqlerr.ClassTypes.New(codeTooBigDisplayWidth, "Too Big Display width")
	// ErrTooBigFieldLength is return when column length too big for column.
	ErrTooBigFieldLength = sqlerr.ClassTypes.New(codeTooBigFieldLength, "Too Big Field length")
	// ErrTooBigSet is returned when too many strings for column.
	ErrTooBigSet = sqlerr.ClassTypes.New(codeTooBigSet, "Too Big Set")
	// ErrTooBigScale is returned when type DECIMAL/NUMERIC scale is bigger than postgres.MaxDecimalScale.
	ErrTooBigScale = sqlerr.ClassTypes.New(codeTooBigScale, postgres.MySQLErrName[postgres.ErrTooBigScale])
	// ErrTooBigPrecision is returned when type DECIMAL/NUMERIC precision is bigger than postgres.MaxDecimalWidth
	ErrTooBigPrecision = sqlerr.ClassTypes.New(codeTooBigPrecision, postgres.MySQLErrName[postgres.ErrTooBigPrecision])
	// ErrWrongFieldSpec is return when incorrect column specifier for column.
	ErrWrongFieldSpec = sqlerr.ClassTypes.New(codeWrongFieldSpec, "Wrong Field Spec")
	// ErrBadNumber is return when parsing an invalid binary decimal number.
	ErrBadNumber = sqlerr.ClassTypes.New(codeBadNumber, "Bad Number")
	// ErrInvalidDefault is returned when meet a invalid default value.
	// ErrInvalidDefault = ErrInvalidDefault
	// ErrCastAsSignedOverflow is returned when positive out-of-range integer, and convert to it's negative complement.
	ErrCastAsSignedOverflow = sqlerr.ClassTypes.New(codeUnknown, msgCastAsSignedOverflow)
	// ErrCastNegIntAsUnsigned is returned when a negative integer be casted to an unsigned int.
	ErrCastNegIntAsUnsigned = sqlerr.ClassTypes.New(codeUnknown, msgCastNegIntAsUnsigned)
	// ErrMBiggerThanD is returned when precision less than the scale.
	ErrMBiggerThanD = sqlerr.ClassTypes.New(codeMBiggerThanD, postgres.MySQLErrName[postgres.ErrMBiggerThanD])
	// ErrWarnDataOutOfRange is returned when the value in a numeric column that is outside the permissible range of the column data type.
	// See https://dev.postgres.com/doc/refman/5.5/en/out-of-range-and-overflow.html for details
	ErrWarnDataOutOfRange = sqlerr.ClassTypes.New(codeDataOutOfRange, postgres.MySQLErrName[postgres.ErrWarnDataOutOfRange])
	// ErrDuplicatedValueInType is returned when enum column has duplicated value.
	ErrDuplicatedValueInType = sqlerr.ClassTypes.New(codeDuplicatedValueInType, postgres.MySQLErrName[postgres.ErrDuplicatedValueInType])
)

const (
	codeBadNumber sqlerr.ErrCode = 1

	codeDataTooLong           = sqlerr.ErrCode(postgres.ErrDataTooLong)
	codeIllegalValueForType   = sqlerr.ErrCode(postgres.ErrIllegalValueForType)
	codeTruncated             = sqlerr.ErrCode(postgres.WarnDataTruncated)
	codeOverflow              = sqlerr.ErrCode(postgres.ErrDataOutOfRange)
	codeDivByZero             = sqlerr.ErrCode(postgres.ErrDivisionByZero)
	codeTooBigDisplayWidth    = sqlerr.ErrCode(postgres.ErrTooBigDisplaywidth)
	codeTooBigFieldLength     = sqlerr.ErrCode(postgres.ErrTooBigFieldlength)
	codeTooBigSet             = sqlerr.ErrCode(postgres.ErrTooBigSet)
	codeTooBigScale           = sqlerr.ErrCode(postgres.ErrTooBigScale)
	codeTooBigPrecision       = sqlerr.ErrCode(postgres.ErrTooBigPrecision)
	codeWrongFieldSpec        = sqlerr.ErrCode(postgres.ErrWrongFieldSpec)
	codeTruncatedWrongValue   = sqlerr.ErrCode(postgres.ErrTruncatedWrongValue)
	codeUnknown               = sqlerr.ErrCode(postgres.ErrUnknown)
	codeInvalidDefault        = sqlerr.ErrCode(postgres.ErrInvalidDefault)
	codeMBiggerThanD          = sqlerr.ErrCode(postgres.ErrMBiggerThanD)
	codeDataOutOfRange        = sqlerr.ErrCode(postgres.ErrWarnDataOutOfRange)
	codeDuplicatedValueInType = sqlerr.ErrCode(postgres.ErrDuplicatedValueInType)
)

var (
	msgOverflow             = postgres.MySQLErrName[postgres.ErrDataOutOfRange]
	msgTruncatedWrongVal    = postgres.MySQLErrName[postgres.ErrTruncatedWrongValue]
	msgCastAsSignedOverflow = "Cast to signed converted positive out-of-range integer to it's negative complement"
	msgCastNegIntAsUnsigned = "Cast to unsigned converted negative integer to it's positive complement"
)

func init() {
	typesMySQLErrCodes := map[sqlerr.ErrCode]uint16{
		codeDataTooLong:           postgres.ErrDataTooLong,
		codeIllegalValueForType:   postgres.ErrIllegalValueForType,
		codeTruncated:             postgres.WarnDataTruncated,
		codeOverflow:              postgres.ErrDataOutOfRange,
		codeDivByZero:             postgres.ErrDivisionByZero,
		codeTooBigDisplayWidth:    postgres.ErrTooBigDisplaywidth,
		codeTooBigFieldLength:     postgres.ErrTooBigFieldlength,
		codeTooBigSet:             postgres.ErrTooBigSet,
		codeTooBigScale:           postgres.ErrTooBigScale,
		codeTooBigPrecision:       postgres.ErrTooBigPrecision,
		codeWrongFieldSpec:        postgres.ErrWrongFieldSpec,
		codeTruncatedWrongValue:   postgres.ErrTruncatedWrongValue,
		codeUnknown:               postgres.ErrUnknown,
		codeInvalidDefault:        postgres.ErrInvalidDefault,
		codeMBiggerThanD:          postgres.ErrMBiggerThanD,
		codeDataOutOfRange:        postgres.ErrWarnDataOutOfRange,
		codeDuplicatedValueInType: postgres.ErrDuplicatedValueInType,
	}
	sqlerr.ErrClassToMySQLCodes[sqlerr.ClassTypes] = typesMySQLErrCodes
}
