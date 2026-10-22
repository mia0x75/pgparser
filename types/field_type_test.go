package types

import (
	"testing"

	"github.com/mia0x75/pgparser/charset"
	"github.com/mia0x75/pgparser/postgres"
)

func TestFieldType(t *testing.T) {
	ft := NewFieldType(postgres.TypeDuration)
	if ft.Flen != UnspecifiedLength {
		t.Fatalf("got %#v, want %#v", ft.Flen, UnspecifiedLength)
	}
	if ft.Decimal != UnspecifiedLength {
		t.Fatalf("got %#v, want %#v", ft.Decimal, UnspecifiedLength)
	}
	ft.Decimal = 5
	if ft.String() != "time(5)" {
		t.Fatalf("got %#v, want %#v", ft.String(), "time(5)")
	}

	ft = NewFieldType(postgres.TypeLong)
	ft.Flen = 5
	ft.Flag = postgres.UnsignedFlag | postgres.ZerofillFlag
	if ft.String() != "int(5) UNSIGNED ZEROFILL" {
		t.Fatalf("got %#v, want %#v", ft.String(), "int(5) UNSIGNED ZEROFILL")
	}
	if ft.InfoSchemaStr() != "int(5) unsigned" {
		t.Fatalf("got %#v, want %#v", ft.InfoSchemaStr(), "int(5) unsigned")
	}

	ft = NewFieldType(postgres.TypeFloat)
	ft.Flen = 12   // Default
	ft.Decimal = 3 // Not Default
	if ft.String() != "float(12,3)" {
		t.Fatalf("got %#v, want %#v", ft.String(), "float(12,3)")
	}
	ft = NewFieldType(postgres.TypeFloat)
	ft.Flen = 12    // Default
	ft.Decimal = -1 // Default
	if ft.String() != "float" {
		t.Fatalf("got %#v, want %#v", ft.String(), "float")
	}
	ft = NewFieldType(postgres.TypeFloat)
	ft.Flen = 5     // Not Default
	ft.Decimal = -1 // Default
	if ft.String() != "float" {
		t.Fatalf("got %#v, want %#v", ft.String(), "float")
	}
	ft = NewFieldType(postgres.TypeFloat)
	ft.Flen = 7    // Not Default
	ft.Decimal = 3 // Not Default
	if ft.String() != "float(7,3)" {
		t.Fatalf("got %#v, want %#v", ft.String(), "float(7,3)")
	}

	ft = NewFieldType(postgres.TypeDouble)
	ft.Flen = 22   // Default
	ft.Decimal = 3 // Not Default
	if ft.String() != "double(22,3)" {
		t.Fatalf("got %#v, want %#v", ft.String(), "double(22,3)")
	}
	ft = NewFieldType(postgres.TypeDouble)
	ft.Flen = 22    // Default
	ft.Decimal = -1 // Default
	if ft.String() != "double" {
		t.Fatalf("got %#v, want %#v", ft.String(), "double")
	}
	ft = NewFieldType(postgres.TypeDouble)
	ft.Flen = 5     // Not Default
	ft.Decimal = -1 // Default
	if ft.String() != "double" {
		t.Fatalf("got %#v, want %#v", ft.String(), "double")
	}
	ft = NewFieldType(postgres.TypeDouble)
	ft.Flen = 7    // Not Default
	ft.Decimal = 3 // Not Default
	if ft.String() != "double(7,3)" {
		t.Fatalf("got %#v, want %#v", ft.String(), "double(7,3)")
	}

	ft = NewFieldType(postgres.TypeBlob)
	ft.Flen = 10
	ft.Charset = "UTF8"
	ft.Collate = "UTF8_UNICODE_GI"
	if ft.String() != "text CHARACTER SET UTF8 COLLATE UTF8_UNICODE_GI" {
		t.Fatalf("got %#v, want %#v", ft.String(), "text CHARACTER SET UTF8 COLLATE UTF8_UNICODE_GI")
	}

	ft = NewFieldType(postgres.TypeVarchar)
	ft.Flen = 10
	ft.Flag |= postgres.BinaryFlag
	if ft.String() != "varchar(10) BINARY" {
		t.Fatalf("got %#v, want %#v", ft.String(), "varchar(10) BINARY")
	}

	ft = NewFieldType(postgres.TypeString)
	ft.Charset = charset.CollationBin
	ft.Flag |= postgres.BinaryFlag
	if ft.String() != "binary(1)" {
		t.Fatalf("got %#v, want %#v", ft.String(), "binary(1)")
	}

	ft = NewFieldType(postgres.TypeEnum)
	ft.Elems = []string{"a", "b"}
	if ft.String() != "enum('a','b')" {
		t.Fatalf("got %#v, want %#v", ft.String(), "enum('a','b')")
	}

	ft = NewFieldType(postgres.TypeEnum)
	ft.Elems = []string{"'a'", "'b'"}
	if ft.String() != "enum('''a''','''b''')" {
		t.Fatalf("got %#v, want %#v", ft.String(), "enum('''a''','''b''')")
	}

	ft = NewFieldType(postgres.TypeEnum)
	ft.Elems = []string{"a\nb", "a\tb", "a\rb"}
	if ft.String() != "enum('a\\nb','a\tb','a\\rb')" {
		t.Fatalf("got %#v", ft.String())
	}

	ft = NewFieldType(postgres.TypeEnum)
	ft.Elems = []string{"a\nb", "a'\t\r\nb", "a\rb"}
	if ft.String() != "enum('a\\nb','a''	\\r\\nb','a\\rb')" {
		t.Fatalf("got %#v", ft.String())
	}

	ft = NewFieldType(postgres.TypeSet)
	ft.Elems = []string{"a", "b"}
	if ft.String() != "set('a','b')" {
		t.Fatalf("got %#v, want %#v", ft.String(), "set('a','b')")
	}

	ft = NewFieldType(postgres.TypeSet)
	ft.Elems = []string{"'a'", "'b'"}
	if ft.String() != "set('''a''','''b''')" {
		t.Fatalf("got %#v, want %#v", ft.String(), "set('''a''','''b''')")
	}

	ft = NewFieldType(postgres.TypeSet)
	ft.Elems = []string{"a\nb", "a'\t\r\nb", "a\rb"}
	if ft.String() != "set('a\\nb','a''	\\r\\nb','a\\rb')" {
		t.Fatalf("got %#v", ft.String())
	}

	ft = NewFieldType(postgres.TypeSet)
	ft.Elems = []string{"a'\nb", "a'b\tc"}
	if ft.String() != "set('a''\\nb','a''b	c')" {
		t.Fatalf("got %#v", ft.String())
	}

	ft = NewFieldType(postgres.TypeTimestamp)
	ft.Flen = 8
	ft.Decimal = 2
	if ft.String() != "timestamp(2)" {
		t.Fatalf("got %#v, want %#v", ft.String(), "timestamp(2)")
	}
	ft = NewFieldType(postgres.TypeTimestamp)
	ft.Flen = 8
	ft.Decimal = 0
	if ft.String() != "timestamp" {
		t.Fatalf("got %#v, want %#v", ft.String(), "timestamp")
	}

	ft = NewFieldType(postgres.TypeDatetime)
	ft.Flen = 8
	ft.Decimal = 2
	if ft.String() != "datetime(2)" {
		t.Fatalf("got %#v, want %#v", ft.String(), "datetime(2)")
	}
	ft = NewFieldType(postgres.TypeDatetime)
	ft.Flen = 8
	ft.Decimal = 0
	if ft.String() != "datetime" {
		t.Fatalf("got %#v, want %#v", ft.String(), "datetime")
	}

	ft = NewFieldType(postgres.TypeDate)
	ft.Flen = 8
	ft.Decimal = 2
	if ft.String() != "date" {
		t.Fatalf("got %#v, want %#v", ft.String(), "date")
	}
	ft = NewFieldType(postgres.TypeDate)
	ft.Flen = 8
	ft.Decimal = 0
	if ft.String() != "date" {
		t.Fatalf("got %#v, want %#v", ft.String(), "date")
	}

	ft = NewFieldType(postgres.TypeYear)
	ft.Flen = 4
	ft.Decimal = 0
	if ft.String() != "year(4)" {
		t.Fatalf("got %#v, want %#v", ft.String(), "year(4)")
	}
	ft = NewFieldType(postgres.TypeYear)
	ft.Flen = 2
	ft.Decimal = 2
	if ft.String() != "year(2)" { // Note: Invalid year.
		t.Fatalf("got %#v, want %#v", ft.String(), "year(2)")
	}
}
