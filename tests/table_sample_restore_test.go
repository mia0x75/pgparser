package parser_test

import (
	"testing"

	"github.com/mia0x75/pgparser/ast"
)

func TestRestoreTableSampleAlias(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{
			"alias_before_sample",
			`SELECT t.a FROM data AS t TABLESAMPLE SYSTEM (50) REPEATABLE (0)`,
			`SELECT "t"."a" FROM "data" AS "t" TABLESAMPLE SYSTEM(50) REPEATABLE (0)`,
		},
		{"sample_without_alias", `SELECT * FROM data TABLESAMPLE SYSTEM (50)`, `SELECT * FROM "data" TABLESAMPLE SYSTEM(50)`},
		{
			"qualified_method",
			`SELECT * FROM data t TABLESAMPLE app."Sampler"(1)`,
			`SELECT * FROM "data" AS "t" TABLESAMPLE "app"."Sampler"(1)`,
		},
		{"dotted_method_part", `SELECT * FROM data TABLESAMPLE "a.b"(1)`, `SELECT * FROM "data" TABLESAMPLE "a.b"(1)`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := restoreString(t, mustParseOne(t, tc.sql)); got != tc.want {
				t.Fatalf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTableSampleVisitorWritesBack(t *testing.T) {
	for _, tc := range []struct{ name, want string }{
		{"sample", `SELECT * FROM "data" AS "t" TABLESAMPLE OTHER(50) REPEATABLE (0)`},
		{"arguments", `SELECT * FROM "data" AS "t" TABLESAMPLE SYSTEM(2) REPEATABLE (2)`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stmt := mustParseOne(t, `SELECT * FROM data t TABLESAMPLE SYSTEM(50) REPEATABLE (0)`)
			visitor := pgRewriteVisitor{rewrite: func(n ast.Node) ast.Node {
				if sample, ok := n.(*ast.TableSample); ok && tc.name == "sample" {
					copy := *sample
					copy.MethodParts = []string{"other"}
					return &copy
				}
				if _, ok := n.(ast.ValueExpr); ok && tc.name == "arguments" {
					return ast.NewValueExpr(int64(2))
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
