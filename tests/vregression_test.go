package parser_test

import (
	"strings"
	"testing"

	pgparser "github.com/mia0x75/pgparser"
	"github.com/mia0x75/pgparser/ast"
	_ "github.com/mia0x75/pgparser/driver"
	"github.com/mia0x75/pgparser/format"
)

// These tests pin the restore fixes from ISSUES.md V-1 to V-6: restored SQL
// must re-parse to the same statement and keep its semantics.

func TestVRestoreIndirection(t *testing.T) {
	for _, tc := range []struct{ sql, want string }{
		{`SELECT (t.a).b FROM t`, `SELECT ("t"."a")."b" FROM "t"`},
		{`SELECT (a).* FROM t`, `SELECT ("a").* FROM "t"`},
		{`SELECT $1[1]`, `SELECT $1[1]`},
		{`SELECT a[1] FROM t`, `SELECT "a"[1] FROM "t"`},
		{`SELECT a[1:2] FROM t`, `SELECT "a"[1:2] FROM "t"`},
		{`SELECT a[:2], a[1:] FROM t`, `SELECT "a"[:2],"a"[1:] FROM "t"`},
	} {
		got := restoreString(t, mustParseOne(t, tc.sql))
		if got != tc.want {
			t.Errorf("restore(%q) = %q, want %q", tc.sql, got, tc.want)
		}
	}
}

func TestVRestoreFunctionNameFidelity(t *testing.T) {
	// Names that fold away case or quoting must be preserved.
	for _, tc := range []struct{ sql, want string }{
		{`SELECT "MyFunc"(a) FROM t`, `SELECT "MyFunc"("a") FROM "t"`},
		{`SELECT "Weird Func"() FROM t`, `SELECT "Weird Func"() FROM "t"`},
		{`SELECT s."My.Table".f(a) FROM t`, `SELECT "s"."My.Table"."f"("a") FROM "t"`},
		// Reserved words used as a function name stay quoted so they re-parse.
		{`SELECT "cast"(a) FROM t`, `SELECT "cast"("a") FROM "t"`},
		{`SELECT "values"(a) FROM t`, `SELECT "values"("a") FROM "t"`},
		{`SELECT "exists"(a) FROM t`, `SELECT "exists"("a") FROM "t"`},
		{`SELECT "interval"(a)`, `SELECT "interval"("a")`},
		{`SELECT "user"(a) FROM t`, `SELECT "user"("a") FROM "t"`},
		// Plain built-ins stay keywords.
		{`SELECT count(*) FROM t`, `SELECT COUNT(*) FROM "t"`},
		{`SELECT coalesce(a, b) FROM t`, `SELECT COALESCE("a", "b") FROM "t"`},
	} {
		got := restoreString(t, mustParseOne(t, tc.sql))
		if got != tc.want {
			t.Errorf("restore(%q) = %q, want %q", tc.sql, got, tc.want)
		}
	}
}

// No quoted name may panic Restore. (Names that collide with a reserved keyword
// form, e.g. "extract", are documented in ISSUES.md V-3 as restoring to the
// keyword form; they must at least not crash.)
func TestVRestoreQuotedNamesDoNotPanic(t *testing.T) {
	for _, sql := range []string{
		`SELECT "extract"(a) FROM t`,
		`SELECT "extract"() FROM t`,
		`SELECT "position"(a) FROM t`,
		`SELECT "trim"(a, b, c) FROM t`,
		`SELECT "substring"(a) FROM t`,
		`SELECT "MyFunc"(a) FROM t`,
	} {
		stmt := mustParseOne(t, sql)
		var sb strings.Builder
		if err := stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err != nil {
			t.Fatalf("Restore(%q): %v", sql, err)
		}
		_ = sb.String()
	}
}

func TestVRestoreDistinctFromSpacing(t *testing.T) {
	for _, tc := range []struct{ sql, want string }{
		{`SELECT 1 WHERE a IS DISTINCT FROM 1`, `SELECT 1 WHERE "a" IS DISTINCT FROM 1`},
		{`SELECT 1 WHERE a IS NOT DISTINCT FROM NULL`, `SELECT 1 WHERE "a" IS NOT DISTINCT FROM NULL`},
		{`SELECT 1 WHERE a = 1`, `SELECT 1 WHERE "a"=1`},
	} {
		got := restoreString(t, mustParseOne(t, tc.sql))
		if got != tc.want {
			t.Errorf("restore(%q) = %q, want %q", tc.sql, got, tc.want)
		}
	}
}

func TestVRestoreNamedWindow(t *testing.T) {
	sql := `SELECT rank() OVER w FROM t WINDOW w AS (PARTITION BY a)`
	want := `SELECT RANK() OVER "w" FROM "t" WINDOW "w" AS (PARTITION BY "a")`
	got := restoreString(t, mustParseOne(t, sql))
	if got != want {
		t.Errorf("restore = %q, want %q", got, want)
	}
}

func TestVRestoreStatementTemplates(t *testing.T) {
	for _, tc := range []struct{ sql, want string }{
		{`EXPLAIN SELECT 1`, `EXPLAIN SELECT 1`},
		{`EXPLAIN ANALYZE SELECT 1`, `EXPLAIN (ANALYZE) SELECT 1`},
		{`TRUNCATE ONLY t`, `TRUNCATE TABLE ONLY "t" RESTRICT`},
		{`TRUNCATE t`, `TRUNCATE TABLE "t" RESTRICT`},
		{`DO $$ BEGIN NULL; END $$ LANGUAGE plpgsql`, `DO ' BEGIN NULL; END ' LANGUAGE "plpgsql"`},
		{`DO $$ BEGIN NULL; END $$`, `DO ' BEGIN NULL; END '`},
		{`MERGE INTO t USING s ON t.id = s.id WHEN NOT MATCHED THEN INSERT DEFAULT VALUES`,
			`MERGE INTO "t" USING "s" ON "t"."id"="s"."id" WHEN NOT MATCHED THEN INSERT DEFAULT VALUES`},
		{`ALTER SERVER s VERSION '1.0'`, `ALTER SERVER "s" VERSION '1.0'`},
		{`CREATE SERVER s TYPE 'oracle' VERSION '1.0' FOREIGN DATA WRAPPER w`,
			`CREATE SERVER "s" TYPE 'oracle' VERSION '1.0' FOREIGN DATA WRAPPER "w"`},
	} {
		got := restoreString(t, mustParseOne(t, tc.sql))
		if got != tc.want {
			t.Errorf("restore(%q) = %q, want %q", tc.sql, got, tc.want)
		}
	}
}

// ISSUES.md V-18: ExprNode.Format must not drop grouping-set expressions.
func TestVFormatGroupingSets(t *testing.T) {
	for _, tc := range []struct{ sql, want string }{
		{`SELECT a FROM t GROUP BY ROLLUP (a, b)`, "ROLLUP (`a`, `b`)"},
		{`SELECT a FROM t GROUP BY CUBE (a, b)`, "CUBE (`a`, `b`)"},
		{`SELECT a FROM t GROUP BY GROUPING SETS ((a), (a, b))`, "GROUPING SETS ((`a`), (`a`, `b`))"},
	} {
		stmt := mustParseOne(t, tc.sql).(*ast.SelectStmt)
		if stmt.GroupBy == nil || len(stmt.GroupBy.Items) != 1 {
			t.Fatalf("%q: no group by item", tc.sql)
		}
		var sb strings.Builder
		stmt.GroupBy.Items[0].Format(&sb)
		if sb.String() != tc.want {
			t.Errorf("Format(%q) = %q, want %q", tc.sql, sb.String(), tc.want)
		}
	}
}

// ISSUES.md V-13: a boolean COPY option must restore to one stable form whether
// it was written with or without a value.
func TestVRestoreCopyBooleans(t *testing.T) {
	for _, tc := range []struct{ sql, want string }{
		{`COPY t FROM STDIN WITH (HEADER)`, `COPY "t" FROM STDIN WITH (HEADER)`},
		{`COPY t FROM STDIN WITH (HEADER true)`, `COPY "t" FROM STDIN WITH (HEADER TRUE)`},
		{`COPY t FROM STDIN WITH (HEADER FALSE)`, `COPY "t" FROM STDIN WITH (HEADER FALSE)`},
		// A quoted '1'/'0' is a delimiter or NULL marker, not a boolean.
		{`COPY t FROM STDIN WITH (DELIMITER '1')`, `COPY "t" FROM STDIN WITH (DELIMITER '1')`},
		{`COPY t FROM STDIN WITH (NULL '0')`, `COPY "t" FROM STDIN WITH (NULL '0')`},
	} {
		got := restoreString(t, mustParseOne(t, tc.sql))
		if got != tc.want {
			t.Errorf("restore(%q) = %q, want %q", tc.sql, got, tc.want)
		}
	}
}

// The function-name quote table must stay in sync with the keyword table. A word
// that the parser rejects as a bare 2-argument function name must be quoted on
// restore, or the restored SQL reads it back as a keyword.
func TestVFuncNameQuoteTableMatchesKeywords(t *testing.T) {
	parses := func(sql string) bool {
		_, err := pgparser.New().Parse(sql)
		return err == nil
	}
	for _, kw := range pgparser.Keywords {
		bare := "SELECT " + strings.ToUpper(kw.Name) + "(1, 2)"
		quoted := `SELECT "` + kw.Name + `"(1, 2)`
		if parses(bare) || !parses(quoted) {
			continue
		}
		if !ast.NeedsQuotedFuncName(kw.Name) {
			t.Errorf("keyword %q cannot be a bare function name but NeedsQuotedFuncName is false", kw.Name)
		}
	}
	// Common built-ins must stay bare for readable output.
	for _, name := range []string{"count", "sum", "lower", "upper", "coalesce", "substring", "convert"} {
		if ast.NeedsQuotedFuncName(name) {
			t.Errorf("%q should not need quoting", name)
		}
	}
}
