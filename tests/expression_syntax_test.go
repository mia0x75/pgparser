package parser_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/parser"
)

func TestTypeModifierIntegerBounds(t *testing.T) {
	t.Parallel()
	for _, typ := range []struct {
		name, format string
		scale        bool
	}{
		{"generic", "widget(%d)", false},
		{"qualified", "app.widget(%d)", false},
		{"numeric_precision", "numeric(%d, 1)", false},
		{"numeric_scale", "numeric(1, %d)", true},
		{"decimal", "decimal(%d, 1)", false},
		{"dec", "dec(%d, 1)", false},
		{"bit", "bit(%d)", false},
		{"varbit", "bit varying(%d)", false},
	} {
		for _, tc := range []struct {
			name  string
			value int64
		}{
			{"zero", 0},
			{"max_int32", math.MaxInt32},
			{"above_int32", 2147483648},
			{"wrap_to_zero", 4294967296},
			{"max_int64", math.MaxInt64},
		} {
			t.Run(typ.name+"/"+tc.name, func(t *testing.T) {
				sql := "CREATE TABLE t (c " + fmt.Sprintf(typ.format, tc.value) + ")"
				stmt, err := parser.New().ParseOneStmt(sql)
				if tc.value > math.MaxInt {
					if err == nil || stmt != nil {
						t.Fatalf("ParseOneStmt(%q) = %T, %v; want overflow rejection", sql, stmt, err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				tp := stmt.(*ast.CreateTableStmt).Cols[0].Tp
				got := tp.Flen
				if typ.scale {
					got = tp.Decimal
				}
				if int64(got) != tc.value {
					t.Fatalf("modifier = %d, want %d", got, tc.value)
				}
			})
		}
	}
}
