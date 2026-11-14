package types

import (
	"reflect"
	"testing"
	"time"

	"github.com/mia0x75/pgparser/postgres"
)

func TestDatumKindValues(t *testing.T) {
	tests := []struct {
		name string
		kind byte
		want byte
		text string
	}{
		{name: "decimal", kind: KindDecimal, want: 8, text: "decimal"},
		{name: "duration", kind: KindDuration, want: 9, text: "time"},
		{name: "enum", kind: KindEnum, want: 10, text: "enum"},
		{name: "bit", kind: KindBit, want: 11, text: "bit"},
		{name: "set", kind: KindSet, want: 12, text: "set"},
		{name: "time", kind: KindTime, want: 13, text: "datetime"},
		{name: "json", kind: KindJSON, want: 18, text: "json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.kind != tt.want {
				t.Fatalf("got kind %d, want %d", tt.kind, tt.want)
			}
			if got := KindStr(tt.kind); got != tt.text {
				t.Fatalf("got kind name %q, want %q", got, tt.text)
			}
		})
	}
}

func TestDatum(t *testing.T) {
	values := []any{
		int64(1),
		uint64(1),
		1.1,
		"abc",
		[]byte("abc"),
		[]int{1},
	}
	for _, val := range values {
		var d Datum
		d.SetValue(val)
		x := d.GetValue()
		if !reflect.DeepEqual(x, val) {
			t.Fatalf("got %#v, want %#v", x, val)
		}
	}
}

func TestDatumDecimalCopy(t *testing.T) {
	dec := NewDecFromStringForTest("1.25")
	d := NewDecimalDatum(dec)
	if d.Kind() != KindDecimal || d.GetPgDecimal() != dec {
		t.Fatal("decimal datum did not retain its kind and value")
	}
	copied := d.Copy()
	if err := copied.GetPgDecimal().FromString([]byte("9")); err != nil {
		t.Fatal(err)
	}
	if got := d.GetPgDecimal().String(); got != "1.25" {
		t.Fatalf("copy changed the original decimal to %q", got)
	}
	sum, err := ComputePlus(d, d)
	if err != nil {
		t.Fatal(err)
	}
	if got := sum.GetPgDecimal().String(); got != "2.50" {
		t.Fatalf("got decimal sum %q, want 2.50", got)
	}
}

func TestIsNull(t *testing.T) {
	tests := []struct {
		data   any
		isnull bool
	}{
		{nil, true},
		{0, false},
		{1, false},
		{1.1, false},
		{"string", false},
		{"", false},
	}
	for _, tt := range tests {
		testIsNull(t, tt.data, tt.isnull)
	}
}

func testIsNull(t *testing.T, data any, isnull bool) {
	d := NewDatum(data)
	if d.IsNull() != isnull {
		t.Fatalf("data: %v, isnull: %v", data, isnull)
	}
}

func prepareCompareDatums() ([]Datum, []Datum) {
	vals := make([]Datum, 0, 5)
	vals = append(vals, NewIntDatum(1))
	vals = append(vals, NewFloat64Datum(1.23))
	vals = append(vals, NewStringDatum("abcde"))
	vals = append(vals, NewDecimalDatum(NewDecFromStringForTest("1.2345")))
	vals = append(vals, NewTimeDatum(Time{Time: FromGoTime(time.Date(2018, 3, 8, 16, 1, 0, 315313000, time.UTC)), Fsp: 6, Type: postgres.TypeTimestamp}))

	vals1 := make([]Datum, 0, 5)
	vals1 = append(vals1, NewIntDatum(1))
	vals1 = append(vals1, NewFloat64Datum(1.23))
	vals1 = append(vals1, NewStringDatum("abcde"))
	vals1 = append(vals1, NewDecimalDatum(NewDecFromStringForTest("1.2345")))
	vals1 = append(vals1, NewTimeDatum(Time{Time: FromGoTime(time.Date(2018, 3, 8, 16, 1, 0, 315313000, time.UTC)), Fsp: 6, Type: postgres.TypeTimestamp}))
	return vals, vals1
}

func BenchmarkCompareDatumByReflect(b *testing.B) {
	vals, vals1 := prepareCompareDatums()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reflect.DeepEqual(vals, vals1)
	}
}
