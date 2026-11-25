package driver

import (
	"strings"
	"testing"

	"github.com/mia0x75/pgparser/types"
)

func TestValueExprFormat(t *testing.T) {
	testCases := []struct {
		datum  types.Datum
		expect string
	}{
		{types.NewDatum(nil), "NULL"},
		{types.NewIntDatum(1), "1"},
		{types.NewIntDatum(-1), "-1"},
		{types.NewUintDatum(1), "1"},
		{types.NewFloat32Datum(1.1), "1.1e+00"},
		{types.NewFloat64Datum(1.1), "1.1e+00"},
		{types.NewStringDatum("abc"), `"abc"`},
		{types.NewDecimalDatum(types.NewDecFromInt(321)), "321"},
	}
	for i, testCase := range testCases {
		var sb strings.Builder
		(&ValueExpr{Datum: testCase.datum}).Format(&sb)
		if sb.String() != testCase.expect {
			t.Fatalf("case %d: Format got %q, want %q (Datum: %#v)", i, sb.String(), testCase.expect, testCase.datum)
		}
	}
}

func TestValueExprProjectionOffsetAndGetDatumString(t *testing.T) {
	v := &ValueExpr{Datum: types.NewStringDatum("abc")}
	v.SetProjectionOffset(7)
	if got := v.GetProjectionOffset(); got != 7 {
		t.Fatalf("GetProjectionOffset: got %d, want 7", got)
	}
	if got := v.GetDatumString(); got != "abc" {
		t.Fatalf("GetDatumString: got %#v, want %q", got, "abc")
	}
}
