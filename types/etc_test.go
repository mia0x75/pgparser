package types

import (
	"io"
	"testing"

	"github.com/mia0x75/pgparser/postgres"
	"github.com/mia0x75/pgparser/sqlerr"
)

func testIsTypeBlob(t *testing.T, tp byte, expect bool) {
	v := IsTypeBlob(tp)
	if v != expect {
		t.Fatalf("got %#v, want %#v", v, expect)
	}
}

func testIsTypeChar(t *testing.T, tp byte, expect bool) {
	v := IsTypeChar(tp)
	if v != expect {
		t.Fatalf("got %#v, want %#v", v, expect)
	}
}

func TestIsType(t *testing.T) {
	testIsTypeBlob(t, postgres.TypeTinyBlob, true)
	testIsTypeBlob(t, postgres.TypeMediumBlob, true)
	testIsTypeBlob(t, postgres.TypeBlob, true)
	testIsTypeBlob(t, postgres.TypeLongBlob, true)
	testIsTypeBlob(t, postgres.TypeInt24, false)

	testIsTypeChar(t, postgres.TypeString, true)
	testIsTypeChar(t, postgres.TypeVarchar, true)
	testIsTypeChar(t, postgres.TypeLong, false)
}

func testTypeStr(t *testing.T, tp byte, expect string) {
	v := TypeStr(tp)
	if v != expect {
		t.Fatalf("got %#v, want %#v", v, expect)
	}
}

func testTypeToStr(t *testing.T, tp byte, charset string, expect string) {
	v := TypeToStr(tp, charset)
	if v != expect {
		t.Fatalf("got %#v, want %#v", v, expect)
	}
}

func TestTypeToStr(t *testing.T) {
	testTypeStr(t, postgres.TypeYear, "year")
	testTypeStr(t, 0xdd, "")

	testTypeToStr(t, postgres.TypeBlob, "utf8", "text")
	testTypeToStr(t, postgres.TypeLongBlob, "utf8", "longtext")
	testTypeToStr(t, postgres.TypeTinyBlob, "utf8", "tinytext")
	testTypeToStr(t, postgres.TypeMediumBlob, "utf8", "mediumtext")
	testTypeToStr(t, postgres.TypeVarchar, "binary", "varbinary")
	testTypeToStr(t, postgres.TypeString, "binary", "binary")
	testTypeToStr(t, postgres.TypeTiny, "binary", "tinyint")
	testTypeToStr(t, postgres.TypeBlob, "binary", "blob")
	testTypeToStr(t, postgres.TypeLongBlob, "binary", "longblob")
	testTypeToStr(t, postgres.TypeTinyBlob, "binary", "tinyblob")
	testTypeToStr(t, postgres.TypeMediumBlob, "binary", "mediumblob")
	testTypeToStr(t, postgres.TypeVarchar, "utf8", "varchar")
	testTypeToStr(t, postgres.TypeString, "utf8", "char")
	testTypeToStr(t, postgres.TypeShort, "binary", "smallint")
	testTypeToStr(t, postgres.TypeInt24, "binary", "mediumint")
	testTypeToStr(t, postgres.TypeLong, "binary", "int")
	testTypeToStr(t, postgres.TypeLonglong, "binary", "bigint")
	testTypeToStr(t, postgres.TypeFloat, "binary", "float")
	testTypeToStr(t, postgres.TypeDouble, "binary", "double")
	testTypeToStr(t, postgres.TypeYear, "binary", "year")
	testTypeToStr(t, postgres.TypeDuration, "binary", "time")
	testTypeToStr(t, postgres.TypeDatetime, "binary", "datetime")
	testTypeToStr(t, postgres.TypeDate, "binary", "date")
	testTypeToStr(t, postgres.TypeTimestamp, "binary", "timestamp")
	testTypeToStr(t, postgres.TypeNewDecimal, "binary", "decimal")
	testTypeToStr(t, postgres.TypeUnspecified, "binary", "unspecified")
	testTypeToStr(t, 0xdd, "binary", "")
	testTypeToStr(t, postgres.TypeBit, "binary", "bit")
	testTypeToStr(t, postgres.TypeEnum, "binary", "enum")
	testTypeToStr(t, postgres.TypeSet, "binary", "set")
}

func TestEOFAsNil(t *testing.T) {
	err := EOFAsNil(io.EOF)
	if err != nil {
		t.Fatalf("got %#v, want nil", err)
	}
}

func TestMaxFloat(t *testing.T) {
	tbl := []struct {
		Flen    int
		Decimal int
		Expect  float64
	}{
		{3, 2, 9.99},
		{5, 2, 999.99},
		{10, 1, 999999999.9},
		{5, 5, 0.99999},
	}

	for _, tt := range tbl {
		f := GetMaxFloat(tt.Flen, tt.Decimal)
		if f != tt.Expect {
			t.Fatalf("got %#v, want %#v", f, tt.Expect)
		}
	}
}

func TestRoundFloat(t *testing.T) {
	tbl := []struct {
		Input  float64
		Expect float64
	}{
		{2.5, 3},
		{1.5, 2},
		{0.5, 1},
		{0.49999999999999997, 0},
		{0, 0},
		{-0.49999999999999997, 0},
		{-0.5, -1},
		{-2.5, -3},
		{-1.5, -2},
	}

	for _, tt := range tbl {
		f := RoundFloat(tt.Input)
		if f != tt.Expect {
			t.Fatalf("got %#v, want %#v", f, tt.Expect)
		}
	}
}

func TestRound(t *testing.T) {
	tbl := []struct {
		Input  float64
		Dec    int
		Expect float64
	}{
		{-1.23, 0, -1},
		{-1.58, 0, -2},
		{1.58, 0, 2},
		{1.298, 1, 1.3},
		{1.298, 0, 1},
		{23.298, -1, 20},
	}

	for _, tt := range tbl {
		f := Round(tt.Input, tt.Dec)
		if f != tt.Expect {
			t.Fatalf("got %#v, want %#v", f, tt.Expect)
		}
	}
}

func TestTruncate(t *testing.T) {
	tbl := []struct {
		Input   float64
		Flen    int
		Decimal int
		Expect  float64
		Err     error
	}{
		{100.114, 10, 2, 100.11, nil},
		{100.115, 10, 2, 100.12, nil},
		{100.1156, 10, 3, 100.116, nil},
		{100.1156, 3, 1, 99.9, ErrOverflow},
		{1.36, 10, 2, 1.36, nil},
	}

	for _, tt := range tbl {
		f, err := TruncateFloat(tt.Input, tt.Flen, tt.Decimal)
		if f != tt.Expect {
			t.Fatalf("got %#v, want %#v", f, tt.Expect)
		}
		if !sqlerr.ErrorEqual(err, tt.Err) {
			t.Fatalf("err %v", err)
		}
	}
}
