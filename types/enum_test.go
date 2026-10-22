package types

import (
	"testing"
)

func TestEnum(t *testing.T) {
	tbl := []struct {
		Elems    []string
		Name     string
		Expected int
	}{
		{[]string{"a", "b"}, "a", 1},
		{[]string{"a"}, "b", 0},
		{[]string{"a"}, "1", 1},
	}

	for _, tt := range tbl {
		e, err := ParseEnumName(tt.Elems, tt.Name)
		if tt.Expected == 0 {
			if err == nil {
				t.Fatalf("expected error: %#v", tt)
			}
			if e.ToNumber() != float64(0) {
				t.Fatalf("got %#v, want %#v", e.ToNumber(), float64(0))
			}
			if e.String() != "" {
				t.Fatalf("got %#v, want %#v", e.String(), "")
			}
			continue
		}

		if err != nil {
			t.Fatalf("unexpected error %v: %#v", err, tt)
		}
		if e.String() != tt.Elems[tt.Expected-1] {
			t.Fatalf("got %#v, want %#v", e.String(), tt.Elems[tt.Expected-1])
		}
		if e.ToNumber() != float64(tt.Expected) {
			t.Fatalf("got %#v, want %#v", e.ToNumber(), float64(tt.Expected))
		}
	}

	tblNumber := []struct {
		Elems    []string
		Number   uint64
		Expected int
	}{
		{[]string{"a"}, 1, 1},
		{[]string{"a"}, 0, 0},
	}

	for _, tt := range tblNumber {
		e, err := ParseEnumValue(tt.Elems, tt.Number)
		if tt.Expected == 0 {
			if err == nil {
				t.Fatalf("expected error: %#v", tt)
			}
			continue
		}

		if err != nil {
			t.Fatalf("unexpected error %v: %#v", err, tt)
		}
		if e.ToNumber() != float64(tt.Expected) {
			t.Fatalf("got %#v, want %#v", e.ToNumber(), float64(tt.Expected))
		}
	}
}
