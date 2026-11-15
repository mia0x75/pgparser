package parser_test

import (
	"testing"

	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/model"
)

func TestRestoreTableFunctionDefinitions(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{"anonymous_record", `SELECT * FROM f() AS (a integer, b text)`, `SELECT * FROM F() AS ("a" "int4", "b" "text")`},
		{"named_record", `SELECT * FROM f() AS r(a integer, b text)`, `SELECT * FROM F() AS "r"("a" "int4", "b" "text")`},
		{
			"lateral_record",
			`SELECT * FROM LATERAL f(t.a) r(a integer)`,
			`SELECT * FROM LATERAL F("t"."a") AS "r"("a" "int4")`,
		},
		{
			"ordinal_record",
			`SELECT * FROM f() WITH ORDINALITY AS r(a integer, n bigint)`,
			`SELECT * FROM F() WITH ORDINALITY AS "r"("a" "int4", "n" "int8")`,
		},
		{"rows_from_calls", `SELECT * FROM ROWS FROM(f(), g())`, `SELECT * FROM ROWS FROM(F(), G())`},
		{
			"rows_from_records",
			`SELECT * FROM ROWS FROM(f() AS (a integer), g() AS (b text)) WITH ORDINALITY AS r`,
			`SELECT * FROM ROWS FROM(F() AS ("a" "int4"), G() AS ("b" "text")) WITH ORDINALITY AS "r"`,
		},
		{
			"rows_from_outer_record",
			`SELECT * FROM ROWS FROM(f()) AS r(a integer)`,
			`SELECT * FROM ROWS FROM(F()) AS "r"("a" "int4")`,
		},
		{
			"column_aliases",
			`SELECT * FROM f() WITH ORDINALITY AS r(a, n)`,
			`SELECT * FROM F() WITH ORDINALITY AS "r"("a", "n")`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := restoreString(t, mustParseOne(t, tc.sql))
			if got != tc.want {
				t.Fatalf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTableFunctionVisitorDefinitions(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{"outer_definition", `SELECT * FROM f() AS r(a integer)`, `SELECT * FROM F() AS "r"("renamed" "int4")`},
		{
			"inner_definition",
			`SELECT * FROM ROWS FROM(f() AS (a integer))`,
			`SELECT * FROM ROWS FROM(F() AS ("renamed" "int4"))`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stmt := mustParseOne(t, tc.sql)
			visitor := pgRewriteVisitor{rewrite: func(n ast.Node) ast.Node {
				if col, ok := n.(*ast.ColumnDef); ok {
					copy := *col
					copy.Name = &ast.ColumnName{Name: model.NewCIStr("renamed")}
					return &copy
				}
				return n
			}}
			updated, ok := stmt.Accept(visitor)
			if !ok {
				t.Fatal("visitor stopped")
			}
			if got := restoreString(t, updated.(ast.StmtNode)); got != tc.want {
				t.Fatalf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}
