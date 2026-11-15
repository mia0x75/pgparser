package parser_test

import (
	"math"
	"strings"
	"testing"

	"github.com/mia0x75/pgparser/ast"
)

func TestRestoreOperatorTokenBoundaries(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{"subtract_negative", `SELECT 2- -1 AS three`, `SELECT 2- -1 AS "three"`},
		{"compare_negative", `SELECT a != -1`, `SELECT "a"!= -1`},
		{"power_negative", `SELECT a ^ -1`, `SELECT "a"^ -1`},
		{"negative_decimal", `SELECT a != -1.5`, `SELECT "a"!= -1.5`},
		{"negative_product", `SELECT 2 - -1 * 2`, `SELECT 2- -1*2`},
		{"negative_collation", `SELECT 2 - -1 COLLATE c`, `SELECT 2- -1 COLLATE "c"`},
		{"nested_unary", `SELECT - -a`, `SELECT - -"a"`},
		{"whole_row_equal", `SELECT t.* = s.* FROM t, s`, `SELECT "t".* = "s".* FROM "t" CROSS JOIN "s"`},
		{
			"indirected_whole_row",
			`SELECT (t.a).* = (s.a).* FROM t, s`,
			`SELECT ("t"."a").* = ("s"."a").* FROM "t" CROSS JOIN "s"`,
		},
		{"whole_row_custom", `SELECT t.* <<< ROW(1, 1) FROM t`, `SELECT "t".* <<< ROW(1,1) FROM "t"`},
		{"nested_whole_row", `SELECT a + t.* = s.* FROM t, s`, `SELECT "a"+"t".* = "s".* FROM "t" CROSS JOIN "s"`},
		{"update_not", `UPDATE t SET a = NOT a`, `UPDATE "t" SET "a"=NOT "a"`},
		{"select_not", `SELECT NOT a FROM t`, `SELECT NOT "a" FROM "t"`},
		{"nested_not", `SELECT NOT NOT a`, `SELECT NOT NOT "a"`},
		{
			"insert_whole_rows",
			`INSERT INTO t AS i VALUES (1) ON CONFLICT (a) DO UPDATE SET a = excluded.a WHERE i.* != excluded.*`,
			`INSERT INTO "t" AS "i" VALUES (1) ON CONFLICT ("a") ` +
				`DO UPDATE SET "a"="excluded"."a" WHERE "i".* != "excluded".*`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := restoreString(t, mustParseOne(t, tc.sql)); got != tc.want {
				t.Fatalf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestUnaryOperatorFormat(t *testing.T) {
	for _, tc := range []struct{ name, sql, want string }{
		{"not", `SELECT NOT a`, "NOT `a`"},
		{"nested_not", `SELECT NOT NOT a`, "NOT NOT `a`"},
		{"nested_minus", `SELECT - -a`, "- -`a`"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stmt := mustParseOne(t, tc.sql).(*ast.SelectStmt)
			var out strings.Builder
			stmt.Fields.Fields[0].Expr.Format(&out)
			if got := out.String(); got != tc.want {
				t.Fatalf("format = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRestoreNegativeZeroOperand(t *testing.T) {
	stmt := mustParseOne(t, `SELECT 2 - 0`).(*ast.SelectStmt)
	expr := stmt.Fields.Fields[0].Expr.(*ast.BinaryOperationExpr)
	expr.R = ast.NewValueExpr(math.Copysign(0, -1))
	if got := restoreString(t, stmt); got != `SELECT 2- -0e+00` {
		t.Fatalf("restore = %q, want SELECT 2- -0e+00", got)
	}
}

func TestRestoreUnaryWordOperator(t *testing.T) {
	stmt := mustParseOne(t, `SELECT NOT a`).(*ast.SelectStmt)
	expr := stmt.Fields.Fields[0].Expr.(*ast.UnaryOperationExpr)
	expr.OpName = "NOT"
	if got := restoreString(t, stmt); got != `SELECT NOT "a"` {
		t.Fatalf("restore = %q, want SELECT NOT \"a\"", got)
	}
	var out strings.Builder
	expr.Format(&out)
	if got := out.String(); got != "NOT `a`" {
		t.Fatalf("format = %q, want NOT `a`", got)
	}
}

func TestRestoreNullLimit(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{"limit_null", `SELECT 1 LIMIT NULL`, `SELECT 1 LIMIT NULL`},
		{"fetch_null", `SELECT 1 FETCH FIRST NULL ROWS ONLY`, `SELECT 1 LIMIT NULL`},
		{"null_with_offset", `SELECT 1 OFFSET 2 FETCH FIRST NULL ROWS ONLY`, `SELECT 1 LIMIT NULL OFFSET 2`},
		{
			"null_with_ties",
			`SELECT a FROM t ORDER BY a FETCH FIRST NULL ROWS WITH TIES`,
			`SELECT "a" FROM "t" ORDER BY "a" FETCH FIRST NULL ROWS WITH TIES`,
		},
		{"limit_all", `SELECT 1 LIMIT ALL`, `SELECT 1`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := restoreString(t, mustParseOne(t, tc.sql)); got != tc.want {
				t.Fatalf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}
