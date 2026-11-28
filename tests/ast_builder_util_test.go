package parser_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/parser"
)

func TestResourceGroupThreadPriorityIntegerBounds(t *testing.T) {
	t.Parallel()
	for _, operation := range []struct{ name, sql string }{
		{"create", "CREATE RESOURCE GROUP rg TYPE = USER THREAD_PRIORITY = %d"},
		{"alter", "ALTER RESOURCE GROUP rg THREAD_PRIORITY = %d"},
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
			t.Run(operation.name+"/"+tc.name, func(t *testing.T) {
				sql := fmt.Sprintf(operation.sql, tc.value)
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
				var priority *int
				switch stmt := stmt.(type) {
				case *ast.CreateResourceGroupStmt:
					priority = stmt.ThreadPriority
				case *ast.AlterResourceGroupStmt:
					priority = stmt.ThreadPriority
				default:
					t.Fatalf("unexpected statement %T", stmt)
				}
				if priority == nil || int64(*priority) != tc.value {
					t.Fatalf("priority = %v, want %d", priority, tc.value)
				}
			})
		}
	}
}
