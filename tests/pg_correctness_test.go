package parser_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/pgparser/ast"
	_ "github.com/mia0x75/pgparser/driver"
	"github.com/mia0x75/pgparser/format"
	pgparser "github.com/mia0x75/pgparser/parser"
)

func restoreString(t *testing.T, stmt ast.StmtNode) string {
	t.Helper()
	var out strings.Builder
	if err := stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil {
		t.Fatalf("Restore error: %v", err)
	}
	reparsed := mustParseOne(t, out.String())
	if reflect.TypeOf(reparsed) != reflect.TypeOf(stmt) {
		t.Fatalf("restored statement changed type: %T to %T", stmt, reparsed)
	}
	var second strings.Builder
	if err := reparsed.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &second)); err != nil {
		t.Fatalf("second Restore error: %v", err)
	}
	if second.String() != out.String() {
		t.Fatalf("Restore is not stable: %q then %q", out.String(), second.String())
	}
	return out.String()
}

func TestIsReadOnlyRejectsWritesAndLocks(t *testing.T) {
	cases := []struct {
		sql  string
		want bool
	}{
		{"SELECT 1", true},
		{"EXPLAIN SELECT 1", true},
		{"EXPLAIN ANALYZE SELECT 1", true},
		{"EXPLAIN ANALYZE DELETE FROM t", false},
		{"EXPLAIN (ANALYZE) DELETE FROM t", false},
		{"EXPLAIN SELECT * FROM t FOR SHARE", false},
		{"DO $$ BEGIN DELETE FROM t; END $$", false},
		{"SELECT 1 INTO newtab", false},
		{"SELECT * FROM t FOR UPDATE", false},
		{"SELECT * FROM t FOR SHARE", false},
		{"SELECT * FROM t FOR NO KEY UPDATE", false},
		{"SELECT * FROM t FOR KEY SHARE", false},
		{"SELECT * FROM (SELECT * FROM t FOR SHARE) AS q", false},
		{"SELECT (SELECT 1 INTO newtab)", false},
		{"SELECT 1 UNION SELECT 2", true},
		{"SELECT 1 UNION SELECT 2 FOR KEY SHARE", false},
		{"WITH d AS (DELETE FROM t RETURNING *) SELECT 1 UNION SELECT 2", false},
		{"WITH q AS (SELECT * FROM t FOR KEY SHARE) SELECT * FROM q", false},
		{"WITH d AS (DELETE FROM t RETURNING *) SELECT * FROM d", false},
		{"WITH d AS (UPDATE t SET a = 1 RETURNING *) SELECT * FROM d", false},
		{"WITH d AS (INSERT INTO t VALUES (1) RETURNING *) SELECT * FROM d", false},
		{"WITH d AS (MERGE INTO t USING s ON t.id = s.id WHEN MATCHED THEN DELETE RETURNING *) SELECT * FROM d", false},
	}
	for _, tc := range cases {
		t.Run(tc.sql, func(t *testing.T) {
			stmt := mustParseOne(t, tc.sql)
			if got := ast.IsReadOnly(stmt); got != tc.want {
				t.Errorf("IsReadOnly(%q) = %v, want %v", tc.sql, got, tc.want)
			}
		})
	}
}

func TestParseRejectsNUL(t *testing.T) {
	for _, sql := range []string{
		"SELECT 1\x00; DROP TABLE t;",
		"SELECT '\x00'",
		"SELECT \"a\x00\"",
		"SELECT $$a\x00$$",
		"SELECT 1 /* \x00 */",
		"SELECT 1 -- \x00\n; SELECT 2",
	} {
		t.Run(sql, func(t *testing.T) {
			if _, err := pgparser.New().Parse(sql); err == nil {
				t.Fatal("expected an error for a NUL byte")
			}
		})
	}
}

func TestRestoreFunctionAndProcedure(t *testing.T) {
	cases := []struct{ sql, want string }{
		{`CREATE FUNCTION f() RETURNS int LANGUAGE sql AS $$ SELECT 1 $$`,
			`CREATE FUNCTION "f"() RETURNS "int4" LANGUAGE "sql" AS ' SELECT 1 '`},
		{`CREATE OR REPLACE PROCEDURE p() LANGUAGE sql AS $$ SELECT 1 $$`,
			`CREATE OR REPLACE PROCEDURE "p"() LANGUAGE "sql" AS ' SELECT 1 '`},
		{`CREATE FUNCTION f() RETURNS int RETURN 1`,
			`CREATE FUNCTION "f"() RETURNS "int4" RETURN 1`},
		{`CREATE FUNCTION f(int DEFAULT 1, VARIADIC xs text[]) RETURNS text RETURN 'ok'`,
			`CREATE FUNCTION "f"("int4" DEFAULT 1, VARIADIC "xs" "text"[]) RETURNS "text" RETURN 'ok'`},
		{`CREATE FUNCTION f() RETURNS TABLE (n int, s text) LANGUAGE sql BEGIN ATOMIC SELECT 1, 'x'; END`,
			`CREATE FUNCTION "f"() RETURNS TABLE ("n" "int4", "s" "text") LANGUAGE "sql" BEGIN ATOMIC SELECT 1,'x'; END`},
		{`ALTER FUNCTION public.f(int) IMMUTABLE STRICT`,
			`ALTER FUNCTION "public"."f"("int4") IMMUTABLE STRICT`},
		{`ALTER PROCEDURE p(text) SECURITY INVOKER`,
			`ALTER PROCEDURE "p"("text") SECURITY INVOKER`},
		{`ALTER ROUTINE f() SET work_mem = '2MB' RESET ALL`,
			`ALTER ROUTINE "f"() SET "work_mem" TO '2MB' RESET ALL`},
		{`CREATE FUNCTION f(x app.t.c%TYPE) RETURNS SETOF app.t.c%TYPE LANGUAGE sql RETURN x`,
			`CREATE FUNCTION "f"("x" "app"."t"."c"%TYPE) RETURNS SETOF "app"."t"."c"%TYPE LANGUAGE "sql" RETURN "x"`},
		{`CREATE FUNCTION f(IN x numeric(10,2), INOUT y text, OUT z int) LANGUAGE sql RETURN x`,
			`CREATE FUNCTION "f"("x" "numeric"(10, 2), INOUT "y" "text", OUT "z" "int4") LANGUAGE "sql" RETURN "x"`},
		{`CREATE FUNCTION f() RETURNS int AS 'mock_object', 'mock_symbol' LANGUAGE c TRANSFORM FOR TYPE int, FOR TYPE app.kind WINDOW IMMUTABLE LEAKPROOF STRICT SECURITY DEFINER PARALLEL SAFE COST 1.5 ROWS 10 SUPPORT app.support_fn SET search_path TO app, public SET work_mem FROM CURRENT`,
			`CREATE FUNCTION "f"() RETURNS "int4" AS 'mock_object', 'mock_symbol' LANGUAGE "c" TRANSFORM FOR TYPE "int4", FOR TYPE "app"."kind" WINDOW IMMUTABLE LEAKPROOF STRICT SECURITY DEFINER PARALLEL SAFE COST 1.5 ROWS 10 SUPPORT "app"."support_fn" SET "search_path" TO 'app', 'public' SET "work_mem" FROM CURRENT`},
		{`ALTER FUNCTION f(int) VOLATILE NOT LEAKPROOF CALLED ON NULL INPUT SECURITY INVOKER PARALLEL RESTRICTED COST 2 ROWS 3 SUPPORT support_fn RESET work_mem`,
			`ALTER FUNCTION "f"("int4") VOLATILE NOT LEAKPROOF CALLED ON NULL INPUT SECURITY INVOKER PARALLEL RESTRICTED COST 2 ROWS 3 SUPPORT "support_fn" RESET "work_mem"`},
		{`CREATE PROCEDURE p(x int DEFAULT 1) LANGUAGE sql BEGIN ATOMIC SELECT x; SELECT 2; END`,
			`CREATE PROCEDURE "p"("x" "int4" DEFAULT 1) LANGUAGE "sql" BEGIN ATOMIC SELECT "x"; SELECT 2; END`},
	}
	for _, tc := range cases {
		t.Run(tc.sql, func(t *testing.T) {
			if got := restoreString(t, mustParseOne(t, tc.sql)); got != tc.want {
				t.Errorf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRestoreRoutinePrivileges(t *testing.T) {
	for _, tc := range []struct{ sql, want string }{
		{`GRANT EXECUTE ON FUNCTION f(integer) TO reader`, `GRANT EXECUTE ON FUNCTION "f"("int4") TO "reader"`},
		{`REVOKE EXECUTE ON PROCEDURE p(integer) FROM reader`, `REVOKE EXECUTE ON PROCEDURE "p"("int4") FROM "reader" RESTRICT`},
		{`GRANT EXECUTE ON ROUTINE f(integer) TO reader`, `GRANT EXECUTE ON ROUTINE "f"("int4") TO "reader"`},
	} {
		t.Run(tc.sql, func(t *testing.T) {
			if got := restoreString(t, mustParseOne(t, tc.sql)); got != tc.want {
				t.Fatalf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRestoreTrigger(t *testing.T) {
	for _, tc := range []struct{ sql, want string }{
		{`CREATE TRIGGER tr BEFORE INSERT ON t EXECUTE FUNCTION f()`,
			`CREATE TRIGGER "tr" BEFORE INSERT ON "t" FOR EACH STATEMENT EXECUTE FUNCTION "f"()`},
		{`CREATE OR REPLACE TRIGGER tr AFTER UPDATE OF a, b OR DELETE ON app.t FOR EACH ROW WHEN (new.a > 1) EXECUTE PROCEDURE app.f('mock', 2)`,
			`CREATE OR REPLACE TRIGGER "tr" AFTER UPDATE OF "a", "b" OR DELETE ON "app"."t" FOR EACH ROW WHEN ("new"."a">1) EXECUTE FUNCTION "app"."f"('mock', '2')`},
		{`CREATE TRIGGER tr AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON t REFERENCING OLD TABLE AS old_rows NEW TABLE AS new_rows FOR EACH STATEMENT EXECUTE FUNCTION f()`,
			`CREATE TRIGGER "tr" AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON "t" REFERENCING OLD TABLE AS "old_rows" NEW TABLE AS "new_rows" FOR EACH STATEMENT EXECUTE FUNCTION "f"()`},
		{`CREATE CONSTRAINT TRIGGER tr AFTER INSERT ON t FROM ref DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION f()`,
			`CREATE CONSTRAINT TRIGGER "tr" AFTER INSERT ON "t" FROM "ref" DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION "f"()`},
		{`CREATE TRIGGER tr INSTEAD OF UPDATE ON v FOR EACH ROW EXECUTE FUNCTION f()`,
			`CREATE TRIGGER "tr" INSTEAD OF UPDATE ON "v" FOR EACH ROW EXECUTE FUNCTION "f"()`},
	} {
		t.Run(tc.sql, func(t *testing.T) {
			if got := restoreString(t, mustParseOne(t, tc.sql)); got != tc.want {
				t.Fatalf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRestoreRoleIdentity(t *testing.T) {
	for _, tc := range []struct{ name, sql, want string }{
		{"schema current role", `CREATE SCHEMA s AUTHORIZATION CURRENT_ROLE`,
			`CREATE SCHEMA "s" AUTHORIZATION CURRENT_ROLE`},
		{"schema quoted current role", `CREATE SCHEMA s AUTHORIZATION "current_role"`,
			`CREATE SCHEMA "s" AUTHORIZATION "current_role"`},
		{"schema current user", `CREATE SCHEMA s AUTHORIZATION CURRENT_USER`,
			`CREATE SCHEMA "s" AUTHORIZATION CURRENT_USER`},
		{"schema quoted current user", `CREATE SCHEMA s AUTHORIZATION "current_user"`,
			`CREATE SCHEMA "s" AUTHORIZATION "current_user"`},
		{"schema session user", `CREATE SCHEMA s AUTHORIZATION SESSION_USER`,
			`CREATE SCHEMA "s" AUTHORIZATION SESSION_USER`},
		{"schema quoted session user", `CREATE SCHEMA s AUTHORIZATION "session_user"`,
			`CREATE SCHEMA "s" AUTHORIZATION "session_user"`},
		{"unnamed schema", `CREATE SCHEMA IF NOT EXISTS AUTHORIZATION "current_user"`,
			`CREATE SCHEMA IF NOT EXISTS AUTHORIZATION "current_user"`},
		{"tablespace current role", `CREATE TABLESPACE ts OWNER CURRENT_ROLE LOCATION 'mock_location'`,
			`CREATE TABLESPACE "ts" OWNER CURRENT_ROLE LOCATION 'mock_location'`},
		{"tablespace quoted current role", `CREATE TABLESPACE ts OWNER "current_role" LOCATION 'mock_location'`,
			`CREATE TABLESPACE "ts" OWNER "current_role" LOCATION 'mock_location'`},
		{"tablespace current user", `CREATE TABLESPACE ts OWNER CURRENT_USER LOCATION 'mock_location'`,
			`CREATE TABLESPACE "ts" OWNER CURRENT_USER LOCATION 'mock_location'`},
		{"tablespace quoted current user", `CREATE TABLESPACE ts OWNER "current_user" LOCATION 'mock_location'`,
			`CREATE TABLESPACE "ts" OWNER "current_user" LOCATION 'mock_location'`},
		{"tablespace session user", `CREATE TABLESPACE ts OWNER SESSION_USER LOCATION 'mock_location'`,
			`CREATE TABLESPACE "ts" OWNER SESSION_USER LOCATION 'mock_location'`},
		{"tablespace quoted session user", `CREATE TABLESPACE ts OWNER "session_user" LOCATION 'mock_location'`,
			`CREATE TABLESPACE "ts" OWNER "session_user" LOCATION 'mock_location'`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := restoreString(t, mustParseOne(t, tc.sql)); got != tc.want {
				t.Fatalf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRestoreSetModes(t *testing.T) {
	for _, tc := range []struct{ sql, want string }{
		{`SET LOCAL search_path TO app, public`, `SET LOCAL "search_path" TO 'app', 'public'`},
		{`SET SESSION work_mem TO DEFAULT`, `SET "work_mem" TO DEFAULT`},
		{`SET work_mem FROM CURRENT`, `SET "work_mem" FROM CURRENT`},
		{`SET ROLE app`, `SET "role" TO 'app'`},
		{`SET ROLE NONE`, `SET "role" TO 'none'`},
		{`SET LOCAL ROLE DEFAULT`, `SET LOCAL "role" TO DEFAULT`},
		{`SET SESSION AUTHORIZATION DEFAULT`, `SET "session_authorization" TO DEFAULT`},
		{`SET TIME ZONE DEFAULT`, `SET "timezone" TO DEFAULT`},
		{`SET TRANSACTION ISOLATION LEVEL SERIALIZABLE, READ ONLY, NOT DEFERRABLE`,
			`SET TRANSACTION ISOLATION LEVEL SERIALIZABLE, READ ONLY, NOT DEFERRABLE`},
		{`SET LOCAL TRANSACTION READ WRITE DEFERRABLE`, `SET LOCAL TRANSACTION READ WRITE, DEFERRABLE`},
		{`SET SESSION CHARACTERISTICS AS TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ WRITE`,
			`SET SESSION CHARACTERISTICS AS TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ WRITE`},
		{`SET TRANSACTION SNAPSHOT 'MOCK_SNAPSHOT'`, `SET TRANSACTION SNAPSHOT 'MOCK_SNAPSHOT'`},
		{`RESET ALL`, `RESET ALL`},
		{`RESET ROLE`, `RESET "role"`},
		{`RESET SESSION AUTHORIZATION`, `RESET "session_authorization"`},
	} {
		t.Run(tc.sql, func(t *testing.T) {
			stmt := mustParseOne(t, tc.sql)
			if _, ok := stmt.(*ast.VariableSetStmt); !ok {
				t.Fatalf("SET/RESET returned %T", stmt)
			}
			if got := restoreString(t, stmt); got != tc.want {
				t.Fatalf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRestoreLocking(t *testing.T) {
	for _, tc := range []struct{ sql, want string }{
		{`SELECT * FROM t, u FOR UPDATE OF t NOWAIT FOR KEY SHARE OF u SKIP LOCKED`,
			`SELECT * FROM "t" CROSS JOIN "u" FOR UPDATE OF "t" NOWAIT FOR KEY SHARE OF "u" SKIP LOCKED`},
		{`SELECT 1 UNION SELECT 2 FOR SHARE`, `SELECT 1 UNION SELECT 2 FOR SHARE`},
		{`(SELECT 1 UNION SELECT 2 FOR UPDATE) UNION SELECT 3`, `(SELECT 1 UNION SELECT 2 FOR UPDATE) UNION SELECT 3`},
	} {
		t.Run(tc.sql, func(t *testing.T) {
			stmt := mustParseOne(t, tc.sql)
			if ast.IsReadOnly(stmt) {
				t.Fatal("locking query was classified read-only")
			}
			if got := restoreString(t, stmt); got != tc.want {
				t.Fatalf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRestoreCopy(t *testing.T) {
	plain := mustParseOne(t, `COPY t (a, b) FROM STDIN WITH (FORMAT csv)`).(*ast.CopyStmt)
	if plain.Table == nil {
		t.Fatal("COPY table was not stored")
	}
	if got, want := restoreString(t, plain), `COPY "t" ("a", "b") FROM STDIN WITH (FORMAT 'csv')`; got != want {
		t.Errorf("plain COPY restore = %q, want %q", got, want)
	}

	query := mustParseOne(t, `COPY (SELECT 1) TO STDOUT`).(*ast.CopyStmt)
	if query.Query == nil {
		t.Fatal("COPY query was not stored")
	}
	if got, want := restoreString(t, query), `COPY (SELECT 1) TO STDOUT`; got != want {
		t.Errorf("query COPY restore = %q, want %q", got, want)
	}
	for _, tc := range []struct{ sql, want string }{
		{`COPY t FROM ''`, `COPY "t" FROM ''`},
		{`COPY t FROM 'input.csv' WITH (FORMAT csv, DELIMITER ';', HEADER true) WHERE a > 1`,
			`COPY "t" FROM 'input.csv' WITH (FORMAT 'csv', DELIMITER ';', HEADER TRUE) WHERE "a">1`},
		{`COPY (SELECT a FROM t) TO PROGRAM 'MOCK_COMMAND' CSV FORCE QUOTE a`,
			`COPY (SELECT "a" FROM "t") TO PROGRAM 'MOCK_COMMAND' WITH (FORMAT 'csv', FORCE_QUOTE ("a"))`},
		{`COPY t TO STDOUT WITH (FORMAT csv, FORCE_QUOTE *)`,
			`COPY "t" TO STDOUT WITH (FORMAT 'csv', FORCE_QUOTE *)`},
	} {
		t.Run(tc.sql, func(t *testing.T) {
			if got := restoreString(t, mustParseOne(t, tc.sql)); got != tc.want {
				t.Errorf("restore = %q, want %q", got, tc.want)
			}
		})
	}
	if _, err := pgparser.New().ParseOneStmt(`COPY t FROM PROGRAM STDIN`); err == nil {
		t.Fatal("COPY PROGRAM without a file name was accepted")
	}
}

func TestRestoreAlterColumnSetDefault(t *testing.T) {
	for _, tc := range []struct{ sql, want string }{
		{`ALTER TABLE t ALTER COLUMN c SET DEFAULT 1`, `ALTER TABLE "t" ALTER COLUMN "c" SET DEFAULT 1`},
		{`ALTER VIEW v ALTER COLUMN c SET DEFAULT 1`, `ALTER VIEW "v" ALTER COLUMN "c" SET DEFAULT 1`},
	} {
		t.Run(tc.sql, func(t *testing.T) {
			if got := restoreString(t, mustParseOne(t, tc.sql)); got != tc.want {
				t.Errorf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRestoreAlterCommands(t *testing.T) {
	for _, tc := range []struct{ sql, want string }{
		{`ALTER INDEX IF EXISTS i SET (fillfactor = 70)`, `ALTER INDEX IF EXISTS "i" SET ("fillfactor" = 70)`},
		{`ALTER INDEX i ATTACH PARTITION p`, `ALTER INDEX "i" ATTACH PARTITION "p"`},
		{`ALTER SEQUENCE s SET UNLOGGED`, `ALTER SEQUENCE "s" SET UNLOGGED`},
		{`ALTER VIEW IF EXISTS v ALTER COLUMN c DROP DEFAULT`, `ALTER VIEW IF EXISTS "v" ALTER COLUMN "c" DROP DEFAULT`},
		{`ALTER MATERIALIZED VIEW v SET (fillfactor = 70)`, `ALTER MATERIALIZED VIEW "v" SET ("fillfactor" = 70)`},
		{`ALTER FOREIGN TABLE t OPTIONS (ADD schema_name 'app', SET table_name 'source', DROP old)`,
			`ALTER FOREIGN TABLE "t" OPTIONS (ADD "schema_name" 'app', SET "table_name" 'source', DROP "old")`},
		{`ALTER FOREIGN TABLE t ALTER COLUMN c OPTIONS (SET option_name 'value')`,
			`ALTER FOREIGN TABLE "t" ALTER COLUMN "c" OPTIONS (SET "option_name" 'value')`},
		{`ALTER TABLE t ALTER COLUMN c TYPE bigint USING c::bigint`,
			`ALTER TABLE "t" ALTER COLUMN "c" TYPE "int8" USING CAST("c" AS "int8")`},
		{`ALTER TABLE t ALTER COLUMN c SET STATISTICS -1`, `ALTER TABLE "t" ALTER COLUMN "c" SET STATISTICS -1`},
		{`ALTER INDEX i ALTER COLUMN 2 SET STATISTICS 100000`, `ALTER INDEX "i" ALTER COLUMN 2 SET STATISTICS 100000`},
		{`ALTER TABLE t ALTER COLUMN c SET STORAGE DEFAULT`, `ALTER TABLE "t" ALTER COLUMN "c" SET STORAGE DEFAULT`},
		{`ALTER TABLE t ALTER COLUMN c SET (n_distinct = -0.5)`, `ALTER TABLE "t" ALTER COLUMN "c" SET ("n_distinct" = -0.5)`},
		{`ALTER TABLE t RESET (toast.autovacuum_enabled)`, `ALTER TABLE "t" RESET ("toast"."autovacuum_enabled")`},
		{`ALTER TABLE t ENABLE TRIGGER ALL`, `ALTER TABLE "t" ENABLE TRIGGER ALL`},
		{`ALTER TABLE t DISABLE TRIGGER USER`, `ALTER TABLE "t" DISABLE TRIGGER USER`},
		{`ALTER TABLE t ENABLE ALWAYS RULE r`, `ALTER TABLE "t" ENABLE ALWAYS RULE "r"`},
		{`ALTER TABLE t ENABLE REPLICA RULE r`, `ALTER TABLE "t" ENABLE REPLICA RULE "r"`},
		{`ALTER TABLE t DISABLE RULE r`, `ALTER TABLE "t" DISABLE RULE "r"`},
		{`ALTER TABLE t DETACH PARTITION p CONCURRENTLY`, `ALTER TABLE "t" DETACH PARTITION "p" CONCURRENTLY`},
		{`ALTER TABLE t DETACH PARTITION p FINALIZE`, `ALTER TABLE "t" DETACH PARTITION "p" FINALIZE`},
		{`ALTER TABLE t REPLICA IDENTITY DEFAULT`, `ALTER TABLE "t" REPLICA IDENTITY DEFAULT`},
		{`ALTER TABLE t REPLICA IDENTITY FULL`, `ALTER TABLE "t" REPLICA IDENTITY FULL`},
		{`ALTER TABLE t REPLICA IDENTITY NOTHING`, `ALTER TABLE "t" REPLICA IDENTITY NOTHING`},
		{`ALTER TABLE t REPLICA IDENTITY USING INDEX i`, `ALTER TABLE "t" REPLICA IDENTITY USING INDEX "i"`},
		{`ALTER TYPE app.row_type ADD ATTRIBUTE c int CASCADE`, `ALTER TYPE "app"."row_type" ADD ATTRIBUTE "c" "int4" CASCADE`},
		{`ALTER TYPE row_type DROP ATTRIBUTE IF EXISTS c CASCADE`, `ALTER TYPE "row_type" DROP ATTRIBUTE IF EXISTS "c" CASCADE`},
		{`ALTER TYPE row_type ALTER ATTRIBUTE c TYPE bigint CASCADE`, `ALTER TYPE "row_type" ALTER ATTRIBUTE "c" TYPE "int8" CASCADE`},
		{`ALTER TABLE ONLY t ALTER COLUMN c SET DEFAULT 1`, `ALTER TABLE ONLY "t" ALTER COLUMN "c" SET DEFAULT 1`},
		{`ALTER FOREIGN TABLE IF EXISTS ONLY t OWNER TO CURRENT_USER`, `ALTER FOREIGN TABLE IF EXISTS ONLY "t" OWNER TO CURRENT_USER`},
		{`ALTER TABLE t OWNER TO SESSION_USER`, `ALTER TABLE "t" OWNER TO SESSION_USER`},
		{`ALTER TABLE t OWNER TO CURRENT_ROLE`, `ALTER TABLE "t" OWNER TO CURRENT_ROLE`},
	} {
		t.Run(tc.sql, func(t *testing.T) {
			if got := restoreString(t, mustParseOne(t, tc.sql)); got != tc.want {
				t.Errorf("restore = %q, want %q", got, tc.want)
			}
		})
	}
	for _, sql := range []string{
		`ALTER TABLE t DROP CONSTRAINT c`,
		`ALTER TABLE t ALTER COLUMN c DROP EXPRESSION`,
		`ALTER TABLE t ALTER COLUMN c ADD GENERATED ALWAYS AS IDENTITY`,
	} {
		t.Run(sql, func(t *testing.T) {
			var out strings.Builder
			if err := mustParseOne(t, sql).Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); !errors.Is(err, ast.ErrUnsupportedRestore) {
				t.Fatalf("unsupported operation: output %q, error %v", out.String(), err)
			}
		})
	}
}

type pgRewriteVisitor struct {
	rewrite func(ast.Node) ast.Node
}

func (v pgRewriteVisitor) Enter(n ast.Node) (ast.Node, bool) { return v.rewrite(n), false }
func (pgRewriteVisitor) Leave(n ast.Node) (ast.Node, bool)   { return n, true }

func TestAlterVisitorWritesBackChildren(t *testing.T) {
	for _, tc := range []struct{ sql, want string }{
		{`ALTER TABLE t ADD COLUMN c int`, `ALTER TABLE "t" ADD COLUMN "renamed" "int4"`},
		{`ALTER TABLE t ALTER COLUMN c SET DEFAULT 1`, `ALTER TABLE "t" ALTER COLUMN "c" SET DEFAULT 2`},
		{`ALTER TABLE t ALTER COLUMN c TYPE bigint USING 1`, `ALTER TABLE "t" ALTER COLUMN "c" TYPE "int8" USING 2`},
		{`ALTER TABLE t SET (fillfactor = 1)`, `ALTER TABLE "t" SET ("fillfactor" = 2)`},
	} {
		t.Run(tc.sql, func(t *testing.T) {
			stmt := mustParseOne(t, tc.sql)
			visitor := pgRewriteVisitor{rewrite: func(n ast.Node) ast.Node {
				switch child := n.(type) {
				case *ast.ColumnDef:
					copy := *child
					copy.Name = &ast.ColumnName{}
					copy.Name.Name = "renamed"
					return &copy
				case ast.ValueExpr:
					return ast.NewValueExpr(int64(2))
				case *ast.Integer:
					return &ast.Integer{Ival: 2}
				}
				return n
			}}
			if _, ok := stmt.Accept(visitor); !ok {
				t.Fatal("visitor failed")
			}
			if got := restoreString(t, stmt); got != tc.want {
				t.Errorf("rewritten = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRestoreOptionalChildren(t *testing.T) {
	cases := []struct{ sql, want string }{
		{`CREATE TABLE fktable (id int REFERENCES pktable(id))`,
			`CREATE TABLE "fktable" ("id" "int4" REFERENCES "pktable"("id"))`},
		{`CREATE TABLE fktable (id int REFERENCES pktable)`,
			`CREATE TABLE "fktable" ("id" "int4" REFERENCES "pktable")`},
		{`CREATE TABLE fktable (id int REFERENCES pktable ON DELETE CASCADE ON UPDATE RESTRICT)`,
			`CREATE TABLE "fktable" ("id" "int4" REFERENCES "pktable" ON DELETE CASCADE ON UPDATE RESTRICT)`},
		{`CREATE TABLE fktable (id int REFERENCES pktable MATCH FULL ON DELETE SET NULL (id) ON UPDATE NO ACTION)`,
			`CREATE TABLE "fktable" ("id" "int4" REFERENCES "pktable" MATCH FULL ON DELETE SET NULL ("id") ON UPDATE NO ACTION)`},
		{`CREATE TABLE fktable (id int REFERENCES pktable MATCH PARTIAL ON DELETE SET DEFAULT)`,
			`CREATE TABLE "fktable" ("id" "int4" REFERENCES "pktable" MATCH PARTIAL ON DELETE SET DEFAULT)`},
		{`CREATE INDEX i ON t (lower(a))`, `CREATE INDEX "i" ON "t" (LOWER("a"))`},
		{`INSERT INTO t VALUES (1) ON CONFLICT (lower(fruit)) DO NOTHING`,
			`INSERT INTO "t" VALUES (1) ON CONFLICT (LOWER("fruit")) DO NOTHING`},
		{`UPDATE aaa SET (b, a) = (SELECT 1, 2)`, `UPDATE "aaa" SET ("b", "a")=(SELECT 1,2)`},
		{`UPDATE aaa SET (b, a) = ROW(1, 2)`, `UPDATE "aaa" SET ("b", "a")=ROW(1,2)`},
	}
	for _, tc := range cases {
		t.Run(tc.sql, func(t *testing.T) {
			if got := restoreString(t, mustParseOne(t, tc.sql)); got != tc.want {
				t.Errorf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRestoreGenericFunctionNames(t *testing.T) {
	for _, tc := range []struct{ sql, want string }{
		{`SELECT convert(a, b)`, `SELECT CONVERT("a", "b")`},
		{`SELECT date_add(x, INTERVAL '1 day')`, `SELECT DATE_ADD("x", CAST('1 day' AS INTERVAL))`},
		{`SELECT date_sub(x, y), adddate(x, y), subdate(x, y)`,
			`SELECT DATE_SUB("x", "y"),ADDDATE("x", "y"),SUBDATE("x", "y")`},
		{`SELECT timestampdiff(day, a, b), timestampadd(day, a, b)`,
			`SELECT TIMESTAMPDIFF("day", "a", "b"),TIMESTAMPADD("day", "a", "b")`},
	} {
		t.Run(tc.sql, func(t *testing.T) {
			if got := restoreString(t, mustParseOne(t, tc.sql)); got != tc.want {
				t.Errorf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestIdentifierTruncationKeepsValidUTF8(t *testing.T) {
	for _, tc := range []struct{ name, sql, want string }{
		{"ascii", `SELECT ` + strings.Repeat("a", 64), strings.Repeat("a", 63)},
		{"two-byte split", `SELECT ` + strings.Repeat("a", 62) + "é", strings.Repeat("a", 62)},
		{"two-byte fits", `SELECT ` + strings.Repeat("a", 61) + "é", strings.Repeat("a", 61) + "é"},
		{"three-byte split", `SELECT ` + strings.Repeat("a", 62) + "名", strings.Repeat("a", 62)},
		{"four-byte split", `SELECT ` + strings.Repeat("a", 61) + "😀", strings.Repeat("a", 61)},
		{"quoted", `SELECT "` + strings.Repeat("A", 62) + `é"`, strings.Repeat("A", 62)},
		{"unicode quoted", `SELECT U&"` + strings.Repeat("A", 62) + `\00E9"`, strings.Repeat("A", 62)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sel := mustParseOne(t, tc.sql).(*ast.SelectStmt)
			got := sel.Fields.Items[0].Expr.(*ast.ColumnNameExpr).Name.Parts[0].Name
			if !utf8.ValidString(got) || got != tc.want {
				t.Fatalf("identifier = %q, want %q", got, tc.want)
			}
			restoreString(t, sel)
		})
	}
}

func TestRejectsInvalidUnicodeEscapes(t *testing.T) {
	for _, sql := range []string{
		`SELECT U&'\+00D800'`,
		`SELECT U&'\D800'`,
		`SELECT U&'\+110000'`,
		`SELECT U&'\+FFFFFF'`,
		`SELECT U&'\DC00'`,
		`SELECT U&'\D800x\DC00'`,
		`SELECT U&'\+000000'`,
		`SELECT U&'\0000'`,
		`SELECT U&'\+0012'`,
		`SELECT U&'\+00+001'`,
		`SELECT E'\uD800'`,
		`SELECT E'\uDC00'`,
		`SELECT E'\uD800x\uDC00'`,
		`SELECT E'\uD800\u123'`,
		`SELECT E'\U00110000'`,
		`SELECT E'\u123'`,
		`SELECT E'\u+001'`,
		`SELECT E'\U000000GG'`,
		`SELECT E'\u0000'`,
		`SELECT E'\000'`,
		`SELECT E'\x00'`,
		`SELECT U&'' UESCAPE '0'`,
		`SELECT U&'' UESCAPE '+'`,
		`SELECT U&'' UESCAPE ' '`,
		`SELECT U&'' UESCAPE 'ab'`,
		`SELECT U&'' UESCAPE 'é'`,
		`SELECT U&'' UESCAPE '"'`,
		`SELECT U&'' UESCAPE ''''`,
		`SELECT U&""`,
	} {
		t.Run(sql, func(t *testing.T) {
			if _, err := pgparser.New().ParseOneStmt(sql); err == nil {
				t.Errorf("expected an error for %q", sql)
			}
		})
	}
	for _, tc := range []struct{ sql, want string }{
		{`SELECT E'\uD7FF'`, "\uD7FF"},
		{`SELECT E'\U0010FFFF'`, "\U0010FFFF"},
		{`SELECT E'\uD83D\uDE00'`, "😀"},
		{`SELECT E'\U0000D83D\uDE00'`, "😀"},
		{`SELECT U&'\D83D\DE00'`, "😀"},
		{`SELECT U&'\+00D83D\+00DE00'`, "😀"},
		{`SELECT U&'!!' UESCAPE '!'`, "!"},
		{`SELECT U&'!0061' /* outer /* nested */ */ UESCAPE /* nested /* comment */ */ '!'`, "a"},
	} {
		t.Run(tc.sql, func(t *testing.T) {
			sel := mustParseOne(t, tc.sql).(*ast.SelectStmt)
			if got := sel.Fields.Items[0].Expr.(ast.ValueExpr).GetValue(); got != tc.want {
				t.Fatalf("decoded = %q, want %q", got, tc.want)
			}
			restoreString(t, sel)
		})
	}
}

func TestNestedComments(t *testing.T) {
	for _, sql := range []string{
		`SELECT 1 /* /* nested */ */`,
		`SELECT U&'\0061' /* /* nested */ */ AS a`,
		`SELECT U&"!0061" /* /* nested */ */ UESCAPE '!'`,
	} {
		if _, err := pgparser.New().ParseOneStmt(sql); err != nil {
			t.Errorf("nested comment rejected in %q: %v", sql, err)
		}
	}
	for _, sql := range []string{
		`SELECT 1 /* /* nested */`,
		`SELECT U&'\0061' /* /* nested */`,
		`SELECT U&'!0061' UESCAPE /* unclosed`,
	} {
		t.Run(sql, func(t *testing.T) {
			if _, err := pgparser.New().ParseOneStmt(sql); err == nil {
				t.Fatal("unterminated comment was accepted")
			}
		})
	}
}

func TestRestoreDatabaseOptions(t *testing.T) {
	for _, tc := range []struct{ sql, want string }{
		{`CREATE DATABASE d LC_COLLATE 'C' LC_CTYPE 'C' LOCATION 'mock_location'`,
			`CREATE DATABASE "d" LC_COLLATE = 'C' LC_CTYPE = 'C' "location" = 'mock_location'`},
		{`CREATE DATABASE d OWNER DEFAULT TEMPLATE DEFAULT ENCODING DEFAULT CONNECTION LIMIT DEFAULT`,
			`CREATE DATABASE "d" OWNER = DEFAULT TEMPLATE = DEFAULT ENCODING = DEFAULT CONNECTION LIMIT = DEFAULT`},
		{`CREATE DATABASE d OWNER app TEMPLATE template0 ENCODING 'UTF8' CONNECTION LIMIT -1 "custom_option" 1.5`,
			`CREATE DATABASE "d" OWNER = "app" TEMPLATE = "template0" ENCODING = 'UTF8' CONNECTION LIMIT = -1 "custom_option" = '1.5'`},
		{`ALTER DATABASE d ALLOW_CONNECTIONS true CONNECTION LIMIT 5`,
			`ALTER DATABASE "d" "allow_connections" = 'true' CONNECTION LIMIT = 5`},
		{`ALTER DATABASE d SET TABLESPACE ts`, `ALTER DATABASE "d" SET TABLESPACE "ts"`},
	} {
		t.Run(tc.sql, func(t *testing.T) {
			if got := restoreString(t, mustParseOne(t, tc.sql)); got != tc.want {
				t.Fatalf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRestoreRelOptions(t *testing.T) {
	for _, tc := range []struct{ sql, want string }{
		{`CREATE TABLE t (a int) WITH (fillfactor = 70, toast.autovacuum_enabled = false)`,
			`CREATE TABLE "t" ("a" "int4") WITH ("fillfactor" = 70, "toast"."autovacuum_enabled" = 'false')`},
		{`CREATE INDEX i ON t (a) WITH (fillfactor = 70)`,
			`CREATE INDEX "i" ON "t" ("a") WITH ("fillfactor" = 70)`},
		{`CREATE VIEW v WITH (security_barrier = true) AS SELECT 1`,
			`CREATE VIEW "v" WITH ("security_barrier" = 'true') AS SELECT 1`},
	} {
		t.Run(tc.sql, func(t *testing.T) {
			if got := restoreString(t, mustParseOne(t, tc.sql)); got != tc.want {
				t.Fatalf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRestoreIndexNullTreatment(t *testing.T) {
	for _, tc := range []struct{ name, sql, want string }{
		{"include before nulls", `CREATE UNIQUE INDEX i ON t (a) INCLUDE (b) NULLS NOT DISTINCT`,
			`CREATE UNIQUE INDEX "i" ON "t" ("a") INCLUDE ("b") NULLS NOT DISTINCT`},
		{"if not exists", `CREATE UNIQUE INDEX IF NOT EXISTS i ON t (a) INCLUDE (b) NULLS NOT DISTINCT`,
			`CREATE UNIQUE INDEX IF NOT EXISTS "i" ON "t" ("a") INCLUDE ("b") NULLS NOT DISTINCT`},
		{"without include", `CREATE UNIQUE INDEX i ON t (a) NULLS NOT DISTINCT`,
			`CREATE UNIQUE INDEX "i" ON "t" ("a") NULLS NOT DISTINCT`},
		{"explicit default", `CREATE UNIQUE INDEX i ON t (a) INCLUDE (b) NULLS DISTINCT`,
			`CREATE UNIQUE INDEX "i" ON "t" ("a") INCLUDE ("b")`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := restoreString(t, mustParseOne(t, tc.sql)); got != tc.want {
				t.Fatalf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTypeNameMetadataPreserved(t *testing.T) {
	for _, tc := range []struct {
		name string
		sql  string
		want []string
	}{
		{
			name: "drop cast",
			sql:  `DROP CAST (app.widget[] AS text)`,
			want: []string{`"app"."widget"[]`, `"text"`},
		},
		{
			name: "drop transform",
			sql:  `DROP TRANSFORM FOR app.widget LANGUAGE plpgsql`,
			want: []string{`"app"."widget"`, "plpgsql"},
		},
		{
			name: "comment cast",
			sql:  `COMMENT ON CAST (int AS text) IS 'mock_comment'`,
			want: []string{`"int4" AS "text"`},
		},
		{
			name: "comment transform",
			sql:  `COMMENT ON TRANSFORM FOR app.widget LANGUAGE plpgsql IS NULL`,
			want: []string{`"app"."widget"`, "plpgsql"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			switch stmt := mustParseOne(t, tc.sql).(type) {
			case *ast.DropStmt:
				for _, object := range stmt.Objects {
					got = append(got, object.Name.Parts[len(object.Name.Parts)-1].Name)
				}
			case *ast.CommentStmt:
				if stmt.Suffix != "" {
					got = append(got, stmt.Suffix)
				}
				got = append(got, stmt.ObjectName...)
				got = append(got, stmt.ExtraName...)
			default:
				t.Fatalf("unexpected statement type %T", stmt)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("type metadata = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRestoreRejectsInvalidAST(t *testing.T) {
	table := &ast.TableName{BaseTableRef: pkast.BaseTableRef{Name: pkast.QualifiedName{Parts: []pkast.Ident{{Name: "t"}}}}}
	column := &ast.ColumnName{Name: "c"}
	for _, tc := range []struct {
		name string
		node ast.Node
	}{
		{"missing COPY target", &ast.CopyStmt{}},
		{"conflicting COPY targets", &ast.CopyStmt{Table: table, Query: mustParseOne(t, "SELECT 1")}},
		{"query COPY FROM", &ast.CopyStmt{Query: mustParseOne(t, "SELECT 1"), IsFrom: true}},
		{"query COPY columns", &ast.CopyStmt{Query: mustParseOne(t, "SELECT 1"), Columns: []*ast.ColumnName{column}}},
		{"COPY file and stream", &ast.CopyStmt{Table: table, Std: true, FileName: "mock_file"}},
		{"COPY nil column", &ast.CopyStmt{Table: table, Columns: []*ast.ColumnName{nil}}},
		{"COPY nil option", &ast.CopyStmt{Table: table, Options: []*ast.DefElem{nil}}},
		{"COPY option metadata", &ast.CopyStmt{Table: table, Options: []*ast.DefElem{{Defname: "format", Defnamespace: "app"}}}},
		{"missing reference", &ast.ReferenceDef{}},
		{"nil reference column", &ast.ReferenceDef{Table: table, IndexColNames: []*ast.IndexColName{nil}}},
		{"invalid FK action", &ast.OnDeleteOpt{ReferOpt: ast.ReferOptionType(100)}},
		{"invalid FK update", &ast.OnUpdateOpt{ReferOpt: ast.ReferOptionType(100)}},
		{"invalid FK action columns", &ast.OnDeleteOpt{ReferOpt: ast.ReferOptionCascade, Columns: []*ast.ColumnName{column}}},
		{"missing index target", &ast.IndexColName{}},
		{"conflicting index targets", &ast.IndexColName{Column: column, Expr: ast.NewValueExpr(int64(1))}},
		{"missing assignment value", &ast.UpdateStmt{Table: &ast.TableSource{}, Assignments: []*ast.Assignment{{Target: ast.NewColumnNameExpr("", "", "c")}}}},
		{"nil assignment target", &ast.UpdateStmt{Table: &ast.TableSource{}, Assignments: []*ast.Assignment{{Target: (*ast.ColumnNameExpr)(nil), Expr: ast.NewValueExpr(int64(1))}}}},
		{"nil database option", &ast.CreateDatabaseStmt{Name: lex.Ident{Name: "d"}, Options: []*ast.DatabaseOption{nil}}},
		{"missing database option name", &ast.CreateDatabaseStmt{Name: lex.Ident{Name: "d"}, Options: []*ast.DatabaseOption{{}}}},
		{"invalid connection limit", &ast.CreateDatabaseStmt{Name: lex.Ident{Name: "d"}, Options: []*ast.DatabaseOption{{Tp: ast.DatabaseOptionConnectionLimit, Value: "1; SELECT 1"}}}},
		{"conflicting database actions", &ast.AlterDatabaseStmt{Name: lex.Ident{Name: "d"}, RenameTo: lex.Ident{Name: "renamed"}, Tablespace: lex.Ident{Name: "ts"}}},
		{"missing routine name", &ast.CreateFunctionStmt{}},
		{"missing extension name", &ast.CreateExtensionStmt{}},
		{"invalid extension option", &ast.CreateExtensionStmt{Extname: "demo", Options: &ast.List{Items: []ast.Node{&ast.DefElem{Defname: "schema"}}}}},
		{"nil routine option", &ast.CreateFunctionStmt{Name: table, Options: []*ast.DefElem{nil}}},
		{"invalid routine WINDOW", &ast.CreateFunctionStmt{Name: table, Options: []*ast.DefElem{{Defname: "window", Arg: &ast.Integer{Ival: 0}}}}},
		{"missing trigger target", &ast.CreateTrigStmt{}},
		{"missing SET values", &ast.VariableSetStmt{Kind: ast.VAR_SET_VALUE, Name: "work_mem"}},
		{"missing ALTER default", &ast.AlterTableSpec{Tp: ast.AlterTableModifyColumn, OldColumnName: column}},
		{"missing ALTER type", &ast.AlterTableSpec{Tp: ast.AlterTableAlterColumnType, OldColumnName: column}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			if err := tc.node.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil {
				t.Fatalf("invalid AST restored without error: %q", out.String())
			}
		})
	}
}

func TestCreateExtensionRestoreRetainsOptions(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want string
	}{
		{
			name: "simple name",
			sql:  "CREATE EXTENSION demo",
			want: `CREATE EXTENSION "demo"`,
		},
		{
			name: "if not exists and all options",
			sql:  `CREATE EXTENSION IF NOT EXISTS "Pkg.Name" WITH SCHEMA public VERSION '1.2' CASCADE`,
			want: `CREATE EXTENSION IF NOT EXISTS "Pkg.Name" WITH SCHEMA "public" VERSION '1.2' CASCADE`,
		},
		{
			name: "version option only",
			sql:  "CREATE EXTENSION demo VERSION v1",
			want: `CREATE EXTENSION "demo" WITH VERSION 'v1'`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parsed := mustParseOne(t, tc.sql)
			stmt, ok := parsed.(*ast.CreateExtensionStmt)
			if !ok {
				t.Fatalf("ParseOneStmt returned %T", parsed)
			}
			span := stmt.Span()
			if !span.Within(len(tc.sql)) || tc.sql[span.Start:span.End] != tc.sql {
				t.Fatalf("statement span = %v, want full input span", span)
			}
			if span != (loc.Span{Start: 0, End: loc.ByteOffset(len(tc.sql))}) {
				t.Fatalf("statement span = %v, want exact input bounds", span)
			}
			got := restoreString(t, stmt)
			if got != tc.want {
				t.Fatalf("Restore = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAlterTableRestoresMultipleCommands(t *testing.T) {
	stmt := mustParseOne(t, "ALTER TABLE t ADD COLUMN a integer, ADD COLUMN b text")
	alter, ok := stmt.(*ast.AlterTableStmt)
	if !ok || len(alter.Specs) != 2 {
		t.Fatalf("parsed ALTER TABLE = %#v, want two commands", stmt)
	}
	if got, want := restoreString(t, alter), "ALTER TABLE \"t\" ADD COLUMN \"a\" \"int4\", ADD COLUMN \"b\" \"text\""; got != want {
		t.Fatalf("Restore = %q, want %q", got, want)
	}
}

func TestAlterTableRejectsUnknownCommand(t *testing.T) {
	var output strings.Builder
	err := (&ast.AlterTableSpec{Tp: ast.AlterTableType(999)}).Restore(
		format.NewRestoreCtx(format.DefaultRestoreFlags, &output),
	)
	if !errors.Is(err, ast.ErrUnsupportedRestore) {
		t.Fatalf("Restore error = %v, want ErrUnsupportedRestore", err)
	}
}

func TestRestoreRejectsNilReceivers(t *testing.T) {
	for _, tc := range []struct {
		name string
		node ast.Node
	}{
		{name: "generated node", node: (*ast.CreateExtensionStmt)(nil)},
		{name: "column definition", node: (*ast.ColumnDef)(nil)},
		{name: "select statement", node: (*ast.SelectStmt)(nil)},
		{name: "window specification", node: (*ast.WindowSpec)(nil)},
		{name: "raw list", node: (*ast.List)(nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.node.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &strings.Builder{})); err == nil {
				t.Fatal("Restore accepted a nil receiver")
			}
		})
	}
}

func TestVisitorWritesBackRestoredChildren(t *testing.T) {
	for _, tc := range []struct{ sql, want string }{
		{`CREATE FUNCTION f(x int DEFAULT 1) RETURNS int LANGUAGE sql RETURN 1`,
			`CREATE FUNCTION "f"("x" "int4" DEFAULT 2) RETURNS "int4" LANGUAGE "sql" RETURN 2`},
		{`CREATE FUNCTION f() RETURNS TABLE (c int) LANGUAGE sql BEGIN ATOMIC SELECT 1; END`,
			`CREATE FUNCTION "f"() RETURNS TABLE ("renamed" "int4") LANGUAGE "sql" BEGIN ATOMIC SELECT 2; END`},
		{`ALTER FUNCTION f(int) COST 1 SET work_mem TO 1`,
			`ALTER FUNCTION "f"("int4") COST 2 SET "work_mem" TO 2`},
		{`SET LOCAL work_mem TO 1`, `SET LOCAL "work_mem" TO 2`},
		{`COPY t (c) TO STDOUT WITH (reject_limit 1)`, `COPY "t" ("renamed") TO STDOUT WITH (REJECT_LIMIT 2)`},
		{`COPY (SELECT 1) TO STDOUT`, `COPY (SELECT 2) TO STDOUT`},
		{`SELECT 1 FROM t FOR UPDATE OF c FOR SHARE`, `SELECT 2 FROM "t" FOR UPDATE OF "renamed" FOR SHARE`},
		{`SELECT 1 UNION SELECT 1 FOR UPDATE OF c`, `SELECT 2 UNION SELECT 2 FOR UPDATE OF "renamed"`},
		{`SELECT 1 GROUP BY GROUPING SETS ((1), ())`, `SELECT 2 GROUP BY GROUPING SETS ((2), ())`},
		{`CREATE TABLE t (c int REFERENCES parent(c) ON DELETE SET NULL (c))`,
			`CREATE TABLE "t" ("renamed" "int4" REFERENCES "parent"("renamed") ON DELETE SET NULL ("renamed"))`},
		{`UPDATE t SET (c, b) = ROW(1, 1)`, `UPDATE "t" SET ("renamed", "b")=ROW(2,2)`},
		{`CREATE INDEX i ON t (lower(c)) INCLUDE (c) WITH (fillfactor = 1) WHERE c = 1`,
			`CREATE INDEX "i" ON "t" (LOWER("renamed")) INCLUDE ("renamed") WITH ("fillfactor" = 2) WHERE "renamed"=2`},
		{`CREATE TABLE t (c int) WITH (fillfactor = 1)`, `CREATE TABLE "t" ("renamed" "int4") WITH ("fillfactor" = 2)`},
		{`CREATE VIEW v WITH (check_option = 1) AS SELECT 1`, `CREATE VIEW "v" WITH ("check_option" = 2) AS SELECT 2`},
		{`ALTER TABLE t OWNER TO app`, `ALTER TABLE "t" OWNER TO "rewritten_role"`},
		{`CREATE SCHEMA s AUTHORIZATION CURRENT_USER`, `CREATE SCHEMA "s" AUTHORIZATION "rewritten_role"`},
		{`CREATE TABLESPACE ts OWNER CURRENT_ROLE LOCATION 'mock_location'`,
			`CREATE TABLESPACE "ts" OWNER "rewritten_role" LOCATION 'mock_location'`},
		// The shared join core keeps USING columns as names, so a visitor cannot rewrite them.
		{`SELECT 1 FROM t INNER JOIN u USING (c) AS j`, `SELECT 2 FROM "t" INNER JOIN "u" USING ("c") AS "j"`},
	} {
		t.Run(tc.sql, func(t *testing.T) {
			stmt := mustParseOne(t, tc.sql)
			visitor := pgRewriteVisitor{rewrite: func(n ast.Node) ast.Node {
				switch child := n.(type) {
				case ast.ValueExpr:
					if child.GetValue() == int64(1) {
						return ast.NewValueExpr(int64(2))
					}
				case *ast.Integer:
					return &ast.Integer{Ival: 2}
				case *ast.ColumnName:
					if child.Name == "c" {
						return &ast.ColumnName{Name: "renamed"}
					}
				case *ast.ColumnNameExpr:
					if len(child.Name.Parts) != 0 && child.Name.Parts[len(child.Name.Parts)-1].Name == "c" {
						copy := *child
						copy.Name.Parts = append([]lex.Ident(nil), child.Name.Parts...)
						copy.Name.Parts[len(copy.Name.Parts)-1].Name = "renamed"
						return &copy
					}
				case *ast.String:
					if child.Str == "c" {
						return &ast.String{Str: "renamed"}
					}
				case *ast.RangeVar:
					copy := *child
					copy.Relname = "renamed"
					return &copy
				case *ast.FunctionParameter:
					copy := *child
					if copy.Name == "c" {
						copy.Name = "renamed"
					}
					return &copy
				case *ast.RoleSpec:
					return &ast.RoleSpec{Roletype: int(ast.ROLESPEC_CSTRING), Rolename: lex.Ident{Name: "rewritten_role"}}
				}
				return n
			}}
			if _, ok := stmt.Accept(visitor); !ok {
				t.Fatal("visitor failed")
			}
			if got := restoreString(t, stmt); got != tc.want {
				t.Fatalf("rewritten = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRestoreAdditionalPostgreSQLSyntax(t *testing.T) {
	for _, tc := range []struct {
		name, sql string
		parts     []string
	}{
		{"nested arrays", `SELECT ARRAY[[1, 2], [3, 4]], ARRAY[]`, []string{`ARRAY[ARRAY[1,2],ARRAY[3,4]]`, `ARRAY[]`}},
		{"inner join", `SELECT * FROM t JOIN u ON t.i = u.i`, []string{`INNER JOIN`, ` ON `}},
		{"natural join", `SELECT * FROM t NATURAL JOIN u`, []string{`NATURAL INNER JOIN`}},
		{"empty escape", `SELECT s LIKE p ESCAPE ''`, []string{`ESCAPE ''`}},
		{"unicode escape", `SELECT s ILIKE p ESCAPE 'é'`, []string{`ILIKE`, `ESCAPE 'é'`}},
		{"expression escape", `SELECT s NOT LIKE p ESCAPE escape_column`, []string{`NOT LIKE`, `ESCAPE "escape_column"`}},
		{"similar escape", `SELECT s NOT SIMILAR TO p ESCAPE escape_column`, []string{`NOT SIMILAR TO`, `ESCAPE "escape_column"`}},
		{"copy null", `COPY t TO STDOUT WITH (NULL 'missing')`, []string{`WITH (NULL 'missing')`}},
		{"groups exclusion", `SELECT sum(i) OVER (ORDER BY i GROUPS BETWEEN 1 PRECEDING AND CURRENT ROW EXCLUDE TIES) FROM t`, []string{`GROUPS BETWEEN 1 PRECEDING AND CURRENT ROW EXCLUDE TIES`}},
		{"rows exclusion", `SELECT sum(i) OVER (ROWS UNBOUNDED PRECEDING EXCLUDE CURRENT ROW) FROM t`, []string{`EXCLUDE CURRENT ROW`}},
		{"range exclusion", `SELECT sum(i) OVER (RANGE UNBOUNDED PRECEDING EXCLUDE GROUP) FROM t`, []string{`EXCLUDE GROUP`}},
		{"no others exclusion", `SELECT sum(i) OVER (ROWS UNBOUNDED PRECEDING EXCLUDE NO OTHERS) FROM t`, []string{`EXCLUDE NO OTHERS`}},
		{"default insert", `INSERT INTO t DEFAULT VALUES RETURNING i`, []string{`DEFAULT VALUES RETURNING`}},
		{"index metadata", `CREATE INDEX i ON t ((a + 1) COLLATE "C" s.ops (distance = 2) DESC NULLS LAST)`, []string{`COLLATE "C" "s"."ops"`, `("distance" = 2) DESC NULLS LAST`}},
		{"empty table", `CREATE TABLE t ()`, []string{`CREATE TABLE "t" ()`}},
		{"unlogged table", `CREATE UNLOGGED TABLE t (i int) USING heap`, []string{`CREATE UNLOGGED TABLE`, `USING "heap"`}},
		{"table options order", `CREATE TEMP TABLE t (i int) WITH (fillfactor = 70) ON COMMIT DROP`, []string{`WITH ("fillfactor" = 70) ON COMMIT DROP`}},
		{"typed table", `CREATE TABLE t OF s.typ`, []string{`OF "s"."typ"`}},
		{"partition keys", `CREATE TABLE t (i int) PARTITION BY RANGE (i)`, []string{`PARTITION BY RANGE ("i")`}},
		{"hash bounds", `CREATE TABLE p PARTITION OF t FOR VALUES WITH (MODULUS 8, REMAINDER 3)`, []string{`PARTITION OF "t"`, `MODULUS 8, REMAINDER 3`}},
		{"default bounds", `CREATE TABLE p PARTITION OF t DEFAULT`, []string{`PARTITION OF "t" DEFAULT`}},
		{"deferred trigger", `CREATE CONSTRAINT TRIGGER tr AFTER INSERT ON t INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION f()`, []string{`DEFERRABLE INITIALLY DEFERRED`}},
		{"analyze columns", `ANALYZE VERBOSE s.t (a, b), u (c)`, []string{`ANALYZE (VERBOSE)`, `"s"."t" ("a", "b"), "u" ("c")`}},
		{"analyze options", `ANALYZE (VERBOSE false, BUFFER_USAGE_LIMIT '2MB') t`, []string{`VERBOSE FALSE`, `BUFFER_USAGE_LIMIT '2MB'`}},
		{"vacuum columns", `VACUUM (ANALYZE false, PARALLEL 2) s.t (a)`, []string{`ANALYZE FALSE`, `PARALLEL 2`, `"s"."t" ("a")`}},
		{"sequence options", `CREATE SEQUENCE s AS bigint OWNED BY t.c NO MINVALUE INCREMENT 2 CACHE 5`, []string{`AS "int8"`, `OWNED BY "t"."c"`, `NO MINVALUE`, `INCREMENT 2`, `CACHE 5`}},
		{"alter sequence", `ALTER SEQUENCE s RESTART 100`, []string{`RESTART 100`}},
		{"deferrable unique column", `CREATE TABLE t (i int UNIQUE DEFERRABLE)`, []string{`UNIQUE DEFERRABLE`}},
		{"nulls not distinct", `CREATE TABLE t (i int UNIQUE NULLS NOT DISTINCT)`, []string{`UNIQUE NULLS NOT DISTINCT`}},
		{"any array", `SELECT a = ANY('{1,2}') FROM t`, []string{`= ANY('{1,2}')`}},
		{"show variable", `SHOW datestyle`, []string{`SHOW "datestyle"`}},
		{"drop trigger", `DROP TRIGGER tr ON t`, []string{`DROP TRIGGER "tr" ON "t"`}},
		{"create tablespace", `CREATE TABLESPACE ts LOCATION '/x' WITH (seq_page_cost = 2)`, []string{`LOCATION '/x'`, `("seq_page_cost" = 2)`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := restoreString(t, mustParseOne(t, tc.sql))
			for _, part := range tc.parts {
				if !strings.Contains(got, part) {
					t.Errorf("Restore omitted %q: %s", part, got)
				}
			}
		})
	}
}

func TestDatabaseRestoreValidationDoesNotWrite(t *testing.T) {
	for _, stmt := range []ast.StmtNode{
		&ast.CreateDatabaseStmt{Name: lex.Ident{Name: "d"}, Options: []*ast.DatabaseOption{{Tp: ast.DatabaseOptionOwner, Value: "app"}, nil}},
		&ast.CreateDatabaseStmt{Name: lex.Ident{Name: "d"}, Options: []*ast.DatabaseOption{{Tp: ast.DatabaseOptionConnectionLimit, Value: "invalid"}}},
		&ast.AlterDatabaseStmt{Name: lex.Ident{Name: "d"}, Options: []*ast.DatabaseOption{{}}},
		&ast.AlterDatabaseStmt{Name: lex.Ident{Name: "d"}, RenameTo: lex.Ident{Name: "other"}, Tablespace: lex.Ident{Name: "ts"}},
	} {
		var out strings.Builder
		if err := stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil || out.Len() != 0 {
			t.Fatalf("invalid database AST wrote %q with error %v", out.String(), err)
		}
	}
}
