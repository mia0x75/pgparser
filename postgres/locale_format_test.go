package postgres

import (
	"math"
	"strconv"
	"testing"
)

// precisionToInt must reject values above the platform int range and negative
// values; a MaxInt64 bound would let a 32-bit build wrap a large precision to
// a negative int.
func TestPrecisionToIntBounds(t *testing.T) {
	maxDigits := strconv.FormatUint(uint64(math.MaxInt), 10)
	if got, err := precisionToInt(maxDigits); err != nil || got != math.MaxInt {
		t.Fatalf("precisionToInt(%q) = (%d, %v), want (%d, nil)", maxDigits, got, err, math.MaxInt)
	}
	over := strconv.FormatUint(uint64(math.MaxInt)+1, 10)
	if _, err := precisionToInt(over); err == nil {
		t.Fatalf("precisionToInt(%q) accepted a value above the platform int range", over)
	}
	if _, err := precisionToInt("-1"); err == nil {
		t.Fatal("precisionToInt accepted a negative precision")
	}
}
