package parser_test

import (
	"testing"

	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/model"
)

func TestRestoreForeignColumnOptions(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{
			"column_options",
			`ALTER FOREIGN TABLE t ADD COLUMN c integer OPTIONS (p 'MOCK_VALUE')`,
			`ALTER FOREIGN TABLE "t" ADD COLUMN "c" "int4" OPTIONS ("p" 'MOCK_VALUE')`,
		},
		{
			"conditional_column_options",
			`ALTER FOREIGN TABLE IF EXISTS t ADD COLUMN c integer OPTIONS (p 'MOCK_VALUE')`,
			`ALTER FOREIGN TABLE IF EXISTS "t" ADD COLUMN "c" "int4" OPTIONS ("p" 'MOCK_VALUE')`,
		},
		{
			"quoted_options",
			`ALTER FOREIGN TABLE t ADD COLUMN c integer OPTIONS ("Option Name" 'MOCK_VALUE', q 'MOCK_OTHER') NOT NULL`,
			`ALTER FOREIGN TABLE "t" ADD COLUMN "c" "int4" ` +
				`OPTIONS ("Option Name" 'MOCK_VALUE', "q" 'MOCK_OTHER') NOT NULL`,
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

func TestForeignOptionsVisitor(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{
			"column_option",
			`ALTER FOREIGN TABLE t ADD COLUMN c integer OPTIONS (p 'MOCK_VALUE')`,
			`ALTER FOREIGN TABLE "renamed" ADD COLUMN "c" "int4" OPTIONS ("p" 'MOCK_REWRITTEN')`,
		},
		{
			"import_options_and_tables",
			`IMPORT FOREIGN SCHEMA s EXCEPT (t) FROM SERVER srv INTO d OPTIONS (p 'MOCK_VALUE')`,
			`IMPORT FOREIGN SCHEMA "s" EXCEPT ("renamed") FROM SERVER "srv" INTO "d" OPTIONS ("p" 'MOCK_REWRITTEN')`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stmt := mustParseOne(t, tc.sql)
			visitor := pgRewriteVisitor{rewrite: func(n ast.Node) ast.Node {
				switch child := n.(type) {
				case *ast.String:
					return &ast.String{Str: "MOCK_REWRITTEN"}
				case *ast.TableName:
					copy := *child
					copy.Name = model.NewCIStr("renamed")
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

func TestRestoreImportForeignSchema(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{
			"all_tables",
			`IMPORT FOREIGN SCHEMA s FROM SERVER srv INTO d`,
			`IMPORT FOREIGN SCHEMA "s" FROM SERVER "srv" INTO "d"`,
		},
		{
			"limit_tables",
			`IMPORT FOREIGN SCHEMA s LIMIT TO (t, u) FROM SERVER srv INTO d`,
			`IMPORT FOREIGN SCHEMA "s" LIMIT TO ("t", "u") FROM SERVER "srv" INTO "d"`,
		},
		{
			"except_tables_and_options",
			`IMPORT FOREIGN SCHEMA s EXCEPT (t, u) FROM SERVER srv INTO d OPTIONS (p 'MOCK_VALUE', q 'MOCK_OTHER')`,
			`IMPORT FOREIGN SCHEMA "s" EXCEPT ("t", "u") FROM SERVER "srv" INTO "d" ` +
				`OPTIONS ("p" 'MOCK_VALUE', "q" 'MOCK_OTHER')`,
		},
		{
			"quoted_options",
			`IMPORT FOREIGN SCHEMA s FROM SERVER srv INTO d OPTIONS ("Option Name" 'MOCK_VALUE')`,
			`IMPORT FOREIGN SCHEMA "s" FROM SERVER "srv" INTO "d" OPTIONS ("Option Name" 'MOCK_VALUE')`,
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
