package parser_test

import (
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"

	"github.com/mia0x75/pgparser/ast"
)

func TestRestoreRoutineDropTargets(t *testing.T) {
	for _, tc := range []struct{ name, sql, want string }{
		{"function", `DROP FUNCTION public.f(int) CASCADE`, `DROP FUNCTION "public"."f"("int4") CASCADE`},
		{"procedure", `DROP PROCEDURE IF EXISTS p(int), q() CASCADE`, `DROP PROCEDURE IF EXISTS "p"("int4"), "q"() CASCADE`},
		{"empty_signature", `DROP FUNCTION f(), g(int) RESTRICT`, `DROP FUNCTION "f"(), "g"("int4")`},
		{"omitted_signatures", `DROP FUNCTION f, g CASCADE`, `DROP FUNCTION "f", "g" CASCADE`},
		{"quoted_name", `DROP FUNCTION "s.q"."f"(int)`, `DROP FUNCTION "s.q"."f"("int4")`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := restoreString(t, mustParseOne(t, tc.sql)); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRoutineDropVisitor(t *testing.T) {
	for _, kind := range []string{"FUNCTION", "PROCEDURE"} {
		t.Run(kind, func(t *testing.T) {
			stmt := mustParseOne(t, "DROP "+kind+" f(int), g(), h CASCADE")
			var names, arguments int
			updated, ok := stmt.Accept(pgRewriteVisitor{rewrite: func(n ast.Node) ast.Node {
				switch child := n.(type) {
				case *ast.TableName:
					names++
					copy := *child
					copy.Name = pkast.QualifiedName{Parts: []pkast.Ident{{Name: "renamed_" + child.Name.Parts[0].Name}}}
					return &copy
				case *ast.DataType:
					arguments++
					return &ast.DataType{Type: mustParseOne(t, "SELECT 1::bigint").(*ast.SelectStmt).Fields.Items[0].Expr.(*ast.FuncCastExpr).Tp}
				}
				return n
			}})
			if !ok || names != 3 || arguments != 1 {
				t.Fatalf("visited names=%d arguments=%d, ok=%v", names, arguments, ok)
			}
			want := "DROP " + kind + ` "renamed_f"("int8"), "renamed_g"(), "renamed_h" CASCADE`
			if got := restoreString(t, updated.(ast.StmtNode)); got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	}
}

func TestRestoreRecursiveViewWildcard(t *testing.T) {
	for _, prefix := range []string{"CREATE", "CREATE OR REPLACE"} {
		t.Run(prefix, func(t *testing.T) {
			stmt := mustParseOne(t, prefix+" RECURSIVE VIEW v(n) AS SELECT 1").(*ast.CreateViewStmt)
			field := stmt.Select.(*ast.SelectStmt).Fields.Items[0]
			wc, ok := field.Expr.(*ast.WildcardExpr)
			if !ok || wc == nil {
				t.Fatal("recursive view projection is a column name instead of a wildcard")
			}
			if len(wc.Qualifier.Parts) != 0 {
				t.Fatal("recursive view projection is a column name instead of a wildcard")
			}
			if field.Alias.Present() || field.Alias.Span.Valid() || field.Span().Valid() || wc.Span().Valid() {
				t.Fatal("synthetic projection must not claim source positions or an alias")
			}
			restoreString(t, stmt)
		})
	}
}
