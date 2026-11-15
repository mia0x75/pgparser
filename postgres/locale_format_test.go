package postgres

import (
	"math"
	"strconv"
	"testing"
)

// The platform int bound prevents a precision from wrapping on 32-bit builds.
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

func TestLocaleFormatShortInputs(t *testing.T) {
	var tests = []struct {
		name      string
		number    string
		precision string
		want      string
		wantErr   bool
	}{
		{name: "positive int", number: "1", precision: "0", want: "1"},
		{name: "negative", number: "-1234", precision: "0", want: "-1,234"},
		{name: "dotless half", number: ".5", precision: "2", want: "0.50"},
		{name: "negative dotless half", number: "-.5", precision: "2", want: "-0.50"},
		{name: "rounds down", number: "12345", precision: "0", want: "12,345"},
		{name: "precision trims fraction", number: "12.345", precision: "2", want: "12.34"},
		{name: "precision pads fraction", number: "12.3", precision: "2", want: "12.30"},
		{name: "garbage treated as zero", number: "abc", precision: "0", want: "0"},
		{name: "empty precision", number: "1", precision: "", want: "1"},
		{name: "empty number", number: "", precision: "2", wantErr: true},
		{name: "lone minus", number: "-", precision: "2", wantErr: true},
		{name: "empty number and precision", number: "", precision: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := GetLocaleFormatFunction("en_US")
			got, err := fn(tt.number, tt.precision)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("fn(%q, %q) = %q, want error", tt.number, tt.precision, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("fn(%q, %q) failed: %v", tt.number, tt.precision, err)
			}
			if got != tt.want {
				t.Fatalf("fn(%q, %q) = %q, want %q", tt.number, tt.precision, got, tt.want)
			}
		})
	}
}
