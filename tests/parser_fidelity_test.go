package parser_test

import (
	"errors"
	"io"
	"math"
	"strings"
	"testing"

	pgparser "github.com/mia0x75/pgparser"
	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/format"
)

func TestRestoreParserFidelity(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{"catalog_delete", `DELETE FROM catalog_name.schema_name.table_name WHERE a = 1`, `DELETE FROM "catalog_name"."schema_name"."table_name" WHERE "a"=1`},
		{"catalog_select", `SELECT catalog_name.schema_name.table_name.column_name FROM catalog_name.schema_name.table_name`, `SELECT "catalog_name"."schema_name"."table_name"."column_name" FROM "catalog_name"."schema_name"."table_name"`},
		{"catalog_wildcard", `SELECT catalog_name.schema_name.table_name.* FROM catalog_name.schema_name.table_name`, `SELECT "catalog_name"."schema_name"."table_name".* FROM "catalog_name"."schema_name"."table_name"`},
		{"catalog_subscript", `SELECT catalog_name.schema_name.table_name.column_name[1] FROM catalog_name.schema_name.table_name`, `SELECT "catalog_name"."schema_name"."table_name"."column_name"[1] FROM "catalog_name"."schema_name"."table_name"`},
		{"qualified_subscript", `SELECT t.a[1] FROM t`, `SELECT "t"."a"[1] FROM "t"`},
		{"schema_subscript", `SELECT s.t.a[1] FROM s.t`, `SELECT "s"."t"."a"[1] FROM "s"."t"`},
		{"field_after_subscript", `SELECT a[1].b FROM t`, `SELECT "a"[1]."b" FROM "t"`},
		{"star_after_subscript", `SELECT a[1].* FROM t`, `SELECT "a"[1].* FROM "t"`},
		{"mixed_subscripts", `SELECT t.a[1].b[2:3] FROM t`, `SELECT "t"."a"[1]."b"[2:3] FROM "t"`},
		{"window_name_parts", `SELECT "a.b"(1) OVER ()`, `SELECT "a.b"(1) OVER ()`},
		{"aggregate_name_parts", `SELECT "a.b"(1) FILTER (WHERE TRUE)`, `SELECT "a.b"(1) FILTER (WHERE TRUE)`},
		{"qualified_window_name", `SELECT s."a.b"(1) OVER ()`, `SELECT "s"."a.b"(1) OVER ()`},
		{"quoted_position", `SELECT "position"(a,b)`, `SELECT "position"("a", "b")`},
		{"quoted_extract", `SELECT "extract"('year',ts)`, `SELECT "extract"('year', "ts")`},
		{"quoted_trim", `SELECT "trim"(a,b)`, `SELECT "trim"("a", "b")`},
		{"numeric_function_name", `SELECT "0"()`, `SELECT "0"()`},
		{"numeric_function_prefix", `SELECT "0func"(1)`, `SELECT "0func"(1)`},
		{"trim_both", `SELECT trim(BOTH 'x' FROM c)`, `SELECT "pg_catalog"."btrim"("c", 'x')`},
		{"trim_leading", `SELECT trim(LEADING 'x' FROM c)`, `SELECT "pg_catalog"."ltrim"("c", 'x')`},
		{"trim_trailing", `SELECT trim(TRAILING 'x' FROM c)`, `SELECT "pg_catalog"."rtrim"("c", 'x')`},
		{"array_subquery", `SELECT ARRAY(SELECT 1)`, `SELECT ARRAY(SELECT 1)`},
		{"scalar_subquery", `SELECT (SELECT 1)`, `SELECT (SELECT 1)`},
		{"distinct_grouping", `SELECT count(*) FROM t GROUP BY DISTINCT GROUPING SETS ((a),(a))`, `SELECT COUNT(*) FROM "t" GROUP BY DISTINCT GROUPING SETS (("a"), ("a"))`},
		{"aggregate_star", `COMMENT ON AGGREGATE agg(*) IS 'MOCK_COMMENT'`, `COMMENT ON AGGREGATE "agg"(*) IS 'MOCK_COMMENT'`},
		{"slash_operator", `SELECT 1 // 2`, `SELECT 1//2`},
		{"utf8_identifier", `SELECT Ä, AÄ FROM t`, `SELECT "Ä","aÄ" FROM "t"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := mustParseOne(t, tc.sql)
			got := restoreString(t, n)
			if got != tc.want {
				t.Fatalf("restore = %q, want %q", got, tc.want)
			}
			if next := restoreString(t, mustParseOne(t, got)); next != got {
				t.Fatalf("unstable restore: %q -> %q", got, next)
			}
		})
	}
}

func TestIndirectionAndWindowFlags(t *testing.T) {
	cases := []struct {
		name, sql string
		flag      uint64
	}{
		{"indirection_arg", `SELECT $1[1]`, ast.FlagHasParamMarker},
		{"indirection_index", `SELECT a[$1]`, ast.FlagHasReference | ast.FlagHasParamMarker},
		{"slice_bounds", `SELECT a[$1:$2]`, ast.FlagHasReference | ast.FlagHasParamMarker},
		{"parenthesized_arg", `SELECT ($1+1)[1]`, ast.FlagHasParamMarker},
		{"partition", `SELECT sum(a) OVER (PARTITION BY $1)`, ast.FlagHasReference | ast.FlagHasParamMarker | ast.FlagHasWindowFunc},
		{"order", `SELECT sum(a) OVER (ORDER BY $1)`, ast.FlagHasReference | ast.FlagHasParamMarker | ast.FlagHasWindowFunc},
		{"frame", `SELECT sum(a) OVER (ORDER BY a ROWS $1 PRECEDING)`, ast.FlagHasReference | ast.FlagHasParamMarker | ast.FlagHasWindowFunc},
		{"frame_end", `SELECT sum(a) OVER (ORDER BY a ROWS BETWEEN CURRENT ROW AND $1 FOLLOWING)`, ast.FlagHasReference | ast.FlagHasParamMarker | ast.FlagHasWindowFunc},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := mustParseOne(t, tc.sql).(*ast.SelectStmt)
			if got := n.Fields.Fields[0].Expr.GetFlag(); got != tc.flag {
				t.Fatalf("flags = %d, want %d", got, tc.flag)
			}
		})
	}
}

type failingWriter struct {
	err       error
	remaining int
	short     bool
}

func (w *failingWriter) Write(p []byte) (int, error) {
	if len(p) <= w.remaining {
		w.remaining -= len(p)
		return len(p), nil
	}
	n := w.remaining
	w.remaining = 0
	if w.short {
		return n, nil
	}
	return n, w.err
}

func TestRestoreWriterErrors(t *testing.T) {
	writeErr := errors.New("MOCK_WRITE_ERROR")
	cases := []struct {
		name string
		w    failingWriter
		want error
	}{
		{"first_write", failingWriter{err: writeErr}, writeErr},
		{"partial_output", failingWriter{err: writeErr, remaining: 12}, writeErr},
		{"short_write", failingWriter{short: true, remaining: 12}, io.ErrShortWrite},
		{"final_write", failingWriter{err: writeErr, remaining: math.MaxInt}, writeErr},
		{"final_short_write", failingWriter{short: true, remaining: math.MaxInt}, io.ErrShortWrite},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, sql := range []string{
				`DELETE FROM t WHERE a = 1`, `SELECT 1`, `SELECT $1`, `SELECT a[$1]`,
				`SELECT sum(a) OVER ()`, `SELECT "a.b"(1) FILTER (WHERE TRUE)`,
				`CREATE TABLE t (a integer)`, `COMMENT ON AGGREGATE agg(*) IS 'MOCK_COMMENT'`,
				`BEGIN`, `NOTIFY c, 'MOCK_PAYLOAD'`, `EXPLAIN SELECT 1`,
				`SELECT * FROM ROWS FROM(f() AS (a integer))`,
				`SELECT * FROM f() AS r(a integer)`,
				`SELECT * FROM t AS r TABLESAMPLE SYSTEM(1)`,
				`ALTER FOREIGN TABLE t ADD COLUMN a integer OPTIONS (p 'MOCK_VALUE')`,
				`IMPORT FOREIGN SCHEMA s EXCEPT (t) FROM SERVER srv INTO d OPTIONS (p 'MOCK_VALUE')`,
				`COMMENT ON OPERATOR ### (NONE, integer) IS 'MOCK_COMMENT'`,
				`CREATE SERVER s FOREIGN DATA WRAPPER mock_fdw OPTIONS ("Option Name" 'MOCK_VALUE')`,
				`CREATE TABLESPACE ts LOCATION ''`,
			} {
				t.Run(sql, func(t *testing.T) {
					w := tc.w
					n := mustParseOne(t, sql)
					if w.remaining != 0 {
						w.remaining = min(w.remaining, len(restoreString(t, n))-1)
					}
					err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &w))
					if !errors.Is(err, tc.want) {
						t.Fatalf("restore error = %v, want %v", err, tc.want)
					}
				})
			}
		})
	}
}

func TestOperatorAndSetChains(t *testing.T) {
	t.Run("unary_plus", func(t *testing.T) {
		const size = 32768
		sql := "SELECT " + strings.Repeat("+", size) + "1"
		if _, err := pgparser.New().ParseOneStmt(sql); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("union_chain", func(t *testing.T) {
		const count = 1000
		sql := strings.Repeat("SELECT 1 UNION ALL ", count-1) + "SELECT 1"
		n := mustParseOne(t, sql).(*ast.SetOprStmt)
		if got := len(n.SelectList.Selects); got != count {
			t.Fatalf("select count = %d, want %d", got, count)
		}
		if got := restoreString(t, n); got != sql {
			t.Fatal("set operation order or ALL changed")
		}
	})
}
