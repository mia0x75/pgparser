package types

import (
	"reflect"
	"testing"
	"time"

	"github.com/mia0x75/pgparser/postgres"
)

func TestDatum(t *testing.T) {
	values := []interface{}{
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

func TestIsNull(t *testing.T) {
	tests := []struct {
		data   interface{}
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

func testIsNull(t *testing.T, data interface{}, isnull bool) {
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
