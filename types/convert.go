package types

import (
	"math"
	"strconv"

	"github.com/mia0x75/pgparser/errors"
	"github.com/mia0x75/pgparser/postgres"
)

func truncateStr(str string, flen int) string {
	if flen != UnspecifiedLength && len(str) > flen {
		str = str[:flen]
	}
	return str
}

// UnsignedUpperBound indicates the max uint64 values of different mysql types.
var UnsignedUpperBound = map[byte]uint64{
	postgres.TypeTiny:     math.MaxUint8,
	postgres.TypeShort:    math.MaxUint16,
	postgres.TypeInt24:    postgres.MaxUint24,
	postgres.TypeLong:     math.MaxUint32,
	postgres.TypeLonglong: math.MaxUint64,
	postgres.TypeBit:      math.MaxUint64,
	postgres.TypeEnum:     math.MaxUint64,
	postgres.TypeSet:      math.MaxUint64,
}

// SignedUpperBound indicates the max int64 values of different mysql types.
var SignedUpperBound = map[byte]int64{
	postgres.TypeTiny:     math.MaxInt8,
	postgres.TypeShort:    math.MaxInt16,
	postgres.TypeInt24:    postgres.MaxInt24,
	postgres.TypeLong:     math.MaxInt32,
	postgres.TypeLonglong: math.MaxInt64,
}

// SignedLowerBound indicates the min int64 values of different mysql types.
var SignedLowerBound = map[byte]int64{
	postgres.TypeTiny:     math.MinInt8,
	postgres.TypeShort:    math.MinInt16,
	postgres.TypeInt24:    postgres.MinInt24,
	postgres.TypeLong:     math.MinInt32,
	postgres.TypeLonglong: math.MinInt64,
}

// ConvertFloatToInt converts a float64 value to a int value.
func ConvertFloatToInt(fval float64, lowerBound, upperBound int64, tp byte) (int64, error) {
	val := RoundFloat(fval)
	if val < float64(lowerBound) {
		return lowerBound, overflow(val, tp)
	}

	if val >= float64(upperBound) {
		if val == float64(upperBound) {
			return upperBound, nil
		}
		return upperBound, overflow(val, tp)
	}
	return int64(val), nil
}

// ConvertIntToInt converts an int value to another int value of different precision.
func ConvertIntToInt(val int64, lowerBound int64, upperBound int64, tp byte) (int64, error) {
	if val < lowerBound {
		return lowerBound, overflow(val, tp)
	}

	if val > upperBound {
		return upperBound, overflow(val, tp)
	}

	return val, nil
}

// ConvertUintToInt converts an uint value to an int value.
func ConvertUintToInt(val uint64, upperBound int64, tp byte) (int64, error) {
	if val > uint64(upperBound) {
		return upperBound, overflow(val, tp)
	}

	return int64(val), nil
}

// ConvertUintToUint converts an uint value to another uint value of different precision.
func ConvertUintToUint(val uint64, upperBound uint64, tp byte) (uint64, error) {
	if val > upperBound {
		return upperBound, overflow(val, tp)
	}

	return val, nil
}

// roundIntStr is to round int string base on the number following dot.
func roundIntStr(numNextDot byte, intStr string) string {
	if numNextDot < '5' {
		return intStr
	}
	retStr := []byte(intStr)
	for i := len(intStr) - 1; i >= 0; i-- {
		if retStr[i] != '9' {
			retStr[i]++
			break
		}
		if i == 0 {
			retStr[i] = '1'
			retStr = append(retStr, '0')
			break
		}
		retStr[i] = '0'
	}
	return string(retStr)
}

// ToString converts an interface to a string.
func ToString(value interface{}) (string, error) {
	switch v := value.(type) {
	case bool:
		if v {
			return "1", nil
		}
		return "0", nil
	case int:
		return strconv.FormatInt(int64(v), 10), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case uint64:
		return strconv.FormatUint(v, 10), nil
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	case Time:
		return v.String(), nil
	case Duration:
		return v.String(), nil
	case *MyDecimal:
		return v.String(), nil
	case BinaryLiteral:
		return v.ToString(), nil
	case Enum:
		return v.String(), nil
	case Set:
		return v.String(), nil
	default:
		return "", errors.Errorf("cannot convert %v(type %T) to string", value, value)
	}
}
