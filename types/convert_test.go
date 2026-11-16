package types

import (
	"math"
	"testing"

	"github.com/mia0x75/pgparser/postgres"
)

func TestConvertFloatToIntBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name               string
		value              float64
		lower, upper, want int64
		wantErr            bool
	}{
		{"nan", math.NaN(), math.MinInt64, math.MaxInt64, 0, true},
		{"positive_infinity", math.Inf(1), math.MinInt64, math.MaxInt64, math.MaxInt64, true},
		{"negative_infinity", math.Inf(-1), math.MinInt64, math.MaxInt64, math.MinInt64, true},
		{"exclusive_upper", math.Exp2(63), math.MinInt64, math.MaxInt64, math.MaxInt64, true},
		{"last_float_in_range", math.Nextafter(math.Exp2(63), 0), math.MinInt64, math.MaxInt64, 9223372036854774784, false},
		{"minimum", float64(math.MinInt64), math.MinInt64, math.MaxInt64, math.MinInt64, false},
		{"below_minimum", math.Nextafter(float64(math.MinInt64), math.Inf(-1)), math.MinInt64, math.MaxInt64, math.MinInt64, true},
		{"inexact_upper", 9007199254740992, math.MinInt64, 9007199254740991, 9007199254740991, true},
		{"inexact_lower", -9007199254740992, -9007199254740991, math.MaxInt64, -9007199254740991, true},
		{"positive_half", 1.5, -10, 10, 2, false},
		{"negative_half", -1.5, -10, 10, -2, false},
		{"small_upper", 10.5, -10, 10, 10, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ConvertFloatToInt(tc.value, tc.lower, tc.upper, postgres.TypeLonglong)
			if got != tc.want || (err != nil) != tc.wantErr {
				t.Fatalf("got (%d, %v), want (%d, error=%v)", got, err, tc.want, tc.wantErr)
			}
		})
	}
}
