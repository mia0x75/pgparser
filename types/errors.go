package types

import "github.com/mia0x75/pgparser/postgres"

var (
	// ErrDataTooLong is returned when converts a string value that is longer than field type length.
	ErrDataTooLong = postgres.ClassTypes.New(codeDataTooLong, "Data Too Long")
	// ErrIllegalValueForType is returned when value of type is illegal.
	ErrIllegalValueForType = postgres.ClassTypes.New(
		codeIllegalValueForType,
		postgres.PgErrName[postgres.ErrIllegalValueForType],
	)
	// ErrTruncated is returned when data has been truncated during conversion.
	ErrTruncated = postgres.ClassTypes.New(codeTruncated, "Data Truncated")
	// ErrTruncatedWrongVal is returned when data has been truncated during conversion.
	ErrTruncatedWrongVal = postgres.ClassTypes.New(codeTruncatedWrongValue, msgTruncatedWrongVal)
	// ErrOverflow is returned when data is out of range for a field type.
	ErrOverflow = postgres.ClassTypes.New(codeOverflow, msgOverflow)
	// ErrDivByZero is return when do division by 0.
	ErrDivByZero = postgres.ClassTypes.New(codeDivByZero, "Division by 0")
	// ErrTooBigDisplayWidth is return when display width out of range for column.
	ErrTooBigDisplayWidth = postgres.ClassTypes.New(codeTooBigDisplayWidth, "Too Big Display width")
	// ErrTooBigFieldLength is return when column length too big for column.
	ErrTooBigFieldLength = postgres.ClassTypes.New(codeTooBigFieldLength, "Too Big Field length")
	// ErrTooBigSet is returned when too many strings for column.
	ErrTooBigSet = postgres.ClassTypes.New(codeTooBigSet, "Too Big Set")
	// ErrTooBigScale is returned when type DECIMAL/NUMERIC scale is bigger than postgres.MaxDecimalScale.
	ErrTooBigScale = postgres.ClassTypes.New(codeTooBigScale, postgres.PgErrName[postgres.ErrTooBigScale])
	// ErrTooBigPrecision is returned when type DECIMAL/NUMERIC precision is bigger than postgres.MaxDecimalWidth
	ErrTooBigPrecision = postgres.ClassTypes.New(codeTooBigPrecision, postgres.PgErrName[postgres.ErrTooBigPrecision])
	// ErrWrongFieldSpec is return when incorrect column specifier for column.
	ErrWrongFieldSpec = postgres.ClassTypes.New(codeWrongFieldSpec, "Wrong Field Spec")
	// ErrBadNumber is return when parsing an invalid binary decimal number.
	ErrBadNumber = postgres.ClassTypes.New(codeBadNumber, "Bad Number")
	// ErrInvalidDefault is returned when meet a invalid default value.
	// ErrInvalidDefault = ErrInvalidDefault
	// ErrCastAsSignedOverflow is returned when positive out-of-range integer, and convert to it's negative complement.
	ErrCastAsSignedOverflow = postgres.ClassTypes.New(codeUnknown, msgCastAsSignedOverflow)
	// ErrCastNegIntAsUnsigned is returned when a negative integer be casted to an unsigned int.
	ErrCastNegIntAsUnsigned = postgres.ClassTypes.New(codeUnknown, msgCastNegIntAsUnsigned)
	// ErrMBiggerThanD is returned when precision less than the scale.
	ErrMBiggerThanD = postgres.ClassTypes.New(codeMBiggerThanD, postgres.PgErrName[postgres.ErrMBiggerThanD])
	// ErrWarnDataOutOfRange is returned when the value in a numeric column that is outside the permissible range of the column data type.
	ErrWarnDataOutOfRange = postgres.ClassTypes.New(codeDataOutOfRange, postgres.PgErrName[postgres.ErrWarnDataOutOfRange])
	// ErrDuplicatedValueInType is returned when enum column has duplicated value.
	ErrDuplicatedValueInType = postgres.ClassTypes.New(
		codeDuplicatedValueInType,
		postgres.PgErrName[postgres.ErrDuplicatedValueInType],
	)
)

const (
	codeBadNumber postgres.ErrCode = 1

	codeDataTooLong           = postgres.ErrCode(postgres.ErrDataTooLong)
	codeIllegalValueForType   = postgres.ErrCode(postgres.ErrIllegalValueForType)
	codeTruncated             = postgres.ErrCode(postgres.WarnDataTruncated)
	codeOverflow              = postgres.ErrCode(postgres.ErrDataOutOfRange)
	codeDivByZero             = postgres.ErrCode(postgres.ErrDivisionByZero)
	codeTooBigDisplayWidth    = postgres.ErrCode(postgres.ErrTooBigDisplaywidth)
	codeTooBigFieldLength     = postgres.ErrCode(postgres.ErrTooBigFieldlength)
	codeTooBigSet             = postgres.ErrCode(postgres.ErrTooBigSet)
	codeTooBigScale           = postgres.ErrCode(postgres.ErrTooBigScale)
	codeTooBigPrecision       = postgres.ErrCode(postgres.ErrTooBigPrecision)
	codeWrongFieldSpec        = postgres.ErrCode(postgres.ErrWrongFieldSpec)
	codeTruncatedWrongValue   = postgres.ErrCode(postgres.ErrTruncatedWrongValue)
	codeUnknown               = postgres.ErrCode(postgres.ErrUnknown)
	codeInvalidDefault        = postgres.ErrCode(postgres.ErrInvalidDefault)
	codeMBiggerThanD          = postgres.ErrCode(postgres.ErrMBiggerThanD)
	codeDataOutOfRange        = postgres.ErrCode(postgres.ErrWarnDataOutOfRange)
	codeDuplicatedValueInType = postgres.ErrCode(postgres.ErrDuplicatedValueInType)
)

var (
	msgOverflow             = postgres.PgErrName[postgres.ErrDataOutOfRange]
	msgTruncatedWrongVal    = postgres.PgErrName[postgres.ErrTruncatedWrongValue]
	msgCastAsSignedOverflow = "Cast to signed converted positive out-of-range integer to it's negative complement"
	msgCastNegIntAsUnsigned = "Cast to unsigned converted negative integer to it's positive complement"
)

func init() {
	typesPgErrCodes := map[postgres.ErrCode]uint16{
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
	postgres.ErrClassToPgCodes[postgres.ClassTypes] = typesPgErrCodes
}
