package parser_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/format"
	"github.com/mia0x75/pgparser/parser"
)

func TestGrammarCTEColumnList(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		sql  string
		fail bool
	}{
		{"plain", "WITH c AS (SELECT 1) SELECT * FROM c", false},
		{"columns", "WITH c(x) AS (SELECT 1) SELECT x FROM c", false},
		{"recursive", "WITH RECURSIVE c(x) AS (SELECT 1) SELECT x FROM c", false},
		{"keyword_name", "WITH recursive AS (SELECT 1) SELECT * FROM recursive", false},
		{"keyword_columns", "WITH recursive(x) AS (SELECT 1) SELECT x FROM recursive", false},
		{"missing_parentheses", "WITH c x AS (SELECT 1) SELECT x FROM c", true},
		{"missing_parentheses_list", "WITH c x, y AS (SELECT 1, 2) SELECT x FROM c", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parser.New().Parse(tc.sql)
			if (err != nil) != tc.fail {
				t.Fatalf("Parse(%q): error = %v, want failure %v", tc.sql, err, tc.fail)
			}
		})
	}
}

func TestGrammarFetchDirectionNames(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		sql    string
		cursor string
	}{
		{"forward", "FETCH 2 forward", "forward"},
		{"backward", "FETCH 2 backward", "backward"},
		{"move", "MOVE 2 backward", "backward"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stmt := mustParseOne(t, tc.sql).(*ast.FetchCursorStmt)
			if stmt.Count != 2 || stmt.Direction != "FORWARD" || stmt.Name != tc.cursor {
				t.Fatalf("unexpected fetch: %#v", stmt)
			}
		})
	}
}

func TestGrammarRoutineLegacyTransactions(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		sql  string
	}{
		{"begin", "BEGIN"},
		{"end", "END WORK"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mustParseOne(t, tc.sql)
			sql := "CREATE PROCEDURE p() LANGUAGE sql BEGIN ATOMIC " + tc.sql + "; END"
			if _, err := parser.New().Parse(sql); err == nil {
				t.Fatalf("legacy transaction accepted inside an atomic body: %s", sql)
			}
		})
	}
}

func TestGrammarDisambiguation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, sql string }{
		{"parameter_subscripts", "SELECT $1[1][2]"},
		{"column_subscripts", "SELECT a.b[1][2]"},
		{"parenthesized_slice", "SELECT (a)[1:2][1]"},
		{"cast_subscript", "SELECT (1::int[])[1]"},
		{"cast", "SELECT CAST(1 AS int)"},
		{"cast_collation", `SELECT CAST(1 AS int COLLATE "C")`},
		{"array_cast", "CREATE CAST (int[] AS text[]) WITHOUT FUNCTION"},
		{"array_type", "PREPARE stmt(int ARRAY[2], int[][]) AS SELECT $1"},
		{"explain_values", "EXPLAIN (VALUES (1))"},
		{"explain_keyword_option", "EXPLAIN (values) SELECT 1"},
		{"json_nested_path", `SELECT * FROM JSON_TABLE('[{}]', '$' COLUMNS (NESTED PATH '$.x' COLUMNS (v int PATH '$'))) AS jt`},
		{"json_nested", `SELECT * FROM JSON_TABLE('[{}]', '$' COLUMNS (NESTED '$.x' COLUMNS (v int PATH '$'))) AS jt`},
		{"reference_attributes", "CREATE DOMAIN d AS int REFERENCES t(x) DEFERRABLE INITIALLY DEFERRED NOT NULL"},
		{"index", "DROP INDEX CONCURRENTLY idx"},
		{"index_if_exists", `DROP INDEX CONCURRENTLY IF EXISTS "idx"`},
		{"qualified_index", "DROP INDEX CONCURRENTLY s.idx CASCADE"},
		{"index_behavior", "DROP INDEX CONCURRENTLY idx RESTRICT"},
		{"function_names", "DROP FUNCTION f, s.g CASCADE"},
		{"function_signatures", "DROP FUNCTION f(int), s.g() CASCADE"},
		{"procedure_names", "DROP PROCEDURE f, s.g CASCADE"},
		{"subscription", "DROP SUBSCRIPTION IF EXISTS sub CASCADE"},
		{"owned", "DROP OWNED BY a, b CASCADE"},
		{"atomic", "CREATE PROCEDURE p() LANGUAGE sql BEGIN ATOMIC SELECT 1; RETURN 2; END"},
		{"empty_atomic", "CREATE PROCEDURE p() LANGUAGE sql BEGIN ATOMIC END"},
		{"nested_atomic", "CREATE PROCEDURE p() LANGUAGE sql BEGIN ATOMIC CREATE FUNCTION f() RETURNS int LANGUAGE sql BEGIN ATOMIC RETURN 1; END; SELECT 2; END"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stmt := mustParseOne(t, tc.sql)
			if span := stmt.Span(); !span.Valid() || int(span.Start) != 0 || int(span.End) != len(tc.sql) {
				t.Fatalf("Span() = %v for %q", span, tc.sql)
			}
			if tc.name == "parameter_subscripts" || tc.name == "parenthesized_slice" {
				expr := stmt.(*ast.SelectStmt).Fields.Items[0].Expr
				indirection, ok := expr.(*ast.A_Indirection)
				if !ok || len(indirection.Indirection.Items) != 2 {
					t.Fatalf("subscripts did not extend the same indirection: %#v", expr)
				}
			}
			var out strings.Builder
			err := stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out))
			switch tc.name {
			case "array_cast", "qualified_index", "index_behavior", "subscription", "reference_attributes":
				if !errors.Is(err, ast.ErrUnsupportedRestore) {
					t.Fatalf("Restore() = %v, want unsupported restore", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			restored := out.String()
			if got := restoreString(t, mustParseOne(t, restored)); got != restored {
				t.Fatalf("unstable restore: %q -> %q", restored, got)
			}
		})
	}
}

func TestGrammarColumnReferenceAttributes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, attributes, deferrable, initially string }{
		{"deferred", "DEFERRABLE INITIALLY DEFERRED", "deferrable", "initially deferred"},
		{"immediate", "NOT DEFERRABLE INITIALLY IMMEDIATE", "not deferrable", "initially immediate"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stmt := mustParseOne(t, "CREATE TABLE child (c int REFERENCES parent(id) "+tc.attributes+" NOT NULL)").(*ast.CreateTableStmt)
			opts := stmt.Cols[0].Options
			if len(opts) != 4 || opts[0].Tp != ast.ColumnOptionReference || opts[1].StrValue != tc.deferrable ||
				opts[2].StrValue != tc.initially || opts[3].Tp != ast.ColumnOptionNotNull {
				t.Fatalf("reference attributes or the following constraint were lost: %#v", opts)
			}
		})
	}
}

func TestGrammarUnsignedRoutineDrop(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"FUNCTION", "PROCEDURE", "ROUTINE"} {
		t.Run(kind, func(t *testing.T) {
			stmt := mustParseOne(t, "DROP "+kind+" f, s.g CASCADE")
			var names []*ast.TableName
			switch stmt := stmt.(type) {
			case *ast.DropFunctionStmt:
				names = stmt.Names
				if stmt.Arguments != nil {
					t.Fatalf("omitted signatures became %#v", stmt.Arguments)
				}
			case *ast.DropProcedureStmt:
				names = stmt.Names
				if stmt.Arguments != nil {
					t.Fatalf("omitted signatures became %#v", stmt.Arguments)
				}
			case *ast.DropStmt:
				names = stmt.Objects
				if stmt.Behavior != int(ast.DROP_CASCADE) {
					t.Fatalf("drop behavior = %v", stmt.Behavior)
				}
			default:
				t.Fatalf("unexpected statement %T", stmt)
			}
			if len(names) != 2 {
				t.Fatalf("got %d names, want 2", len(names))
			}
			for _, name := range names {
				if name.Span() != loc.InvalidSpan() {
					t.Fatalf("synthetic drop name acquired span %v", name.Span())
				}
			}
		})
	}
}
