package parser_test

import (
	"fmt"
	"strings"
	"testing"

	parser "github.com/mia0x75/pgparser"
	"github.com/mia0x75/pgparser/ast"
	_ "github.com/mia0x75/pgparser/driver"
	"github.com/mia0x75/pgparser/format"
	json "github.com/mia0x75/pgparser/types/json"
)

func TestAuditRestoreRegression(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{"issue01_limit_all_offset", "SELECT 1 LIMIT ALL OFFSET 2", "SELECT 1 OFFSET 2"},
		{"issue01_braced_limit_offset", "(SELECT a FROM t LIMIT 3) OFFSET 2", `SELECT "a" FROM "t" LIMIT 3 OFFSET 2`},
		{"issue01_braced_offset_limit", "(SELECT a FROM t OFFSET 2) LIMIT 3", `SELECT "a" FROM "t" LIMIT 3 OFFSET 2`},
		{"issue01_braced_offset_all", "(SELECT a FROM t OFFSET 2) LIMIT ALL", `SELECT "a" FROM "t" OFFSET 2`},
		{"issue01_braced_ties_offset", "(SELECT a FROM t ORDER BY a FETCH FIRST 3 ROWS WITH TIES) OFFSET 2", `SELECT "a" FROM "t" ORDER BY "a" FETCH FIRST 3 ROWS WITH TIES OFFSET 2`},
		{"issue02_adjacent_comment", "SELECT 1+// extension\n2", "SELECT 1+2"},
		{"issue03_parameters", "SELECT $2, $1, $2147483647", "SELECT $2,$1,$2147483647"},
		{"issue04_numeric", "SELECT 9007199254740993, 0.10000000000000000001, 1e1000", "SELECT 9007199254740993,0.10000000000000000001,1e1000"},
		{"issue05_cast", "SELECT CAST('12' AS numeric(12, -2)), 1::integer", `SELECT CAST('12' AS "numeric"(12, -2)),CAST(1 AS "int4")`},
		{"issue05_cast_array", "SELECT CAST('{}' AS text[])", `SELECT CAST('{}' AS "text"[])`},
		{"issue05_qualified_cast", "SELECT '12'::app.money", `SELECT CAST('12' AS "app"."money")`},
		{"issue05_typed_name", "SELECT app.money '12'", `SELECT CAST('12' AS "app"."money")`},
		{"issue05_quoted_typed_name", `SELECT "app.schema"."Currency" '1.00'`, `SELECT CAST('1.00' AS "app.schema"."Currency")`},
		{"issue05_typed_modifiers", "SELECT app.vector(2, 3, 4) '1,2,3'", `SELECT CAST('1,2,3' AS "app"."vector"(2, 3, 4))`},
		{"issue05_typed_interval", "SELECT INTERVAL '1' DAY TO SECOND(3)", "SELECT CAST('1' AS INTERVAL DAY TO SECOND(3))"},
		{"issue05_typed_interval_precision", "SELECT INTERVAL(3) '1'", "SELECT CAST('1' AS INTERVAL(3))"},
		{"issue06_into", "SELECT * INTO TEMP saved FROM source", `SELECT * INTO TEMPORARY "saved" FROM "source"`},
		{"issue07_privilege", "GRANT MAINTAIN ON TABLE t TO reader", `GRANT MAINTAIN ON TABLE "t" TO "reader"`},
		{"issue07_role", "CREATE ROLE reader NOLOGIN NOSUPERUSER CONNECTION LIMIT 4", `CREATE ROLE "reader" NOLOGIN NOSUPERUSER CONNECTION LIMIT 4`},
		{"issue07_alter_role", "ALTER ROLE reader LOGIN BYPASSRLS", `ALTER ROLE "reader" LOGIN BYPASSRLS`},
		{"issue07_role_options", "CREATE ROLE reader PASSWORD 'MOCK_TOKEN' VALID UNTIL 'infinity' IN ROLE parent ADMIN manager", `CREATE ROLE "reader" PASSWORD 'MOCK_TOKEN' VALID UNTIL 'infinity' IN ROLE "parent" ADMIN "manager"`},
		{"issue07_alter_current", "ALTER ROLE CURRENT_USER LOGIN", "ALTER ROLE CURRENT_USER LOGIN"},
		{"issue07_alter_current_no_options", "ALTER ROLE CURRENT_USER", "ALTER ROLE CURRENT_USER"},
		{"issue07_role_member_current", "CREATE ROLE reader IN ROLE CURRENT_USER", `CREATE ROLE "reader" IN ROLE CURRENT_USER`},
		{"issue07_grant_options", "GRANT SELECT, UPDATE ON TABLE t TO PUBLIC WITH GRANT OPTION GRANTED BY CURRENT_USER", `GRANT SELECT, UPDATE ON TABLE "t" TO PUBLIC WITH GRANT OPTION GRANTED BY CURRENT_USER`},
		{"issue07_grant_all", "GRANT ALL PRIVILEGES ON TABLE t TO CURRENT_ROLE", `GRANT ALL ON TABLE "t" TO CURRENT_ROLE`},
		{"issue07_grant_all_columns", "GRANT ALL(a) ON TABLE t TO reader", `GRANT ALL ("a") ON TABLE "t" TO "reader"`},
		{"issue07_revoke_all", "REVOKE ALL ON TABLE t FROM PUBLIC CASCADE", `REVOKE ALL ON TABLE "t" FROM PUBLIC CASCADE`},
		{"issue07_revoke_all_columns", "REVOKE ALL PRIVILEGES(a) ON TABLE t FROM reader", `REVOKE ALL ("a") ON TABLE "t" FROM "reader" RESTRICT`},
		{"issue07_revoke_options", "REVOKE GRANT OPTION FOR SELECT ON TABLE t FROM CURRENT_USER GRANTED BY SESSION_USER CASCADE", `REVOKE GRANT OPTION FOR SELECT ON TABLE "t" FROM CURRENT_USER GRANTED BY SESSION_USER CASCADE`},
		{"issue07_grant_roles", `GRANT "Readers", writers TO member, CURRENT_USER WITH ADMIN OPTION, INHERIT TRUE, SET FALSE GRANTED BY SESSION_USER`, `GRANT "Readers", "writers" TO "member", CURRENT_USER WITH ADMIN TRUE, INHERIT TRUE, SET FALSE GRANTED BY SESSION_USER`},
		{"issue07_revoke_roles", "REVOKE ADMIN OPTION FOR readers FROM CURRENT_USER GRANTED BY SESSION_USER CASCADE", `REVOKE ADMIN OPTION FOR "readers" FROM CURRENT_USER GRANTED BY SESSION_USER CASCADE`},
		{"issue07_revoke_inherit", "REVOKE INHERIT OPTION FOR readers FROM member", `REVOKE INHERIT OPTION FOR "readers" FROM "member" RESTRICT`},
		{"issue07_revoke_set", "REVOKE SET OPTION FOR readers FROM member", `REVOKE SET OPTION FOR "readers" FROM "member" RESTRICT`},
		{"issue07_public_case", `GRANT SELECT ON TABLE t TO "PUBLIC"`, `GRANT SELECT ON TABLE "t" TO "PUBLIC"`},
		{"issue07_large_object", "GRANT SELECT ON LARGE OBJECT 42, 43 TO reader", `GRANT SELECT ON LARGE OBJECT 42, 43 TO "reader"`},
		{"issue08_aggregate", "SELECT count(DISTINCT a ORDER BY b) FILTER (WHERE b > 0) FROM t", `SELECT COUNT(DISTINCT "a" ORDER BY "b") FILTER (WHERE "b">0) FROM "t"`},
		{"issue08_custom_star", "SELECT f(*)", "SELECT F(*)"},
		{"issue08_custom_star_filter", "SELECT f(*) FILTER (WHERE $1 > 0)", "SELECT F(*) FILTER (WHERE $1>0)"},
		{"issue08_custom_star_window", "SELECT f(*) OVER ()", "SELECT F(*) OVER ()"},
		{"issue08_window", "SELECT count(*) FILTER (WHERE a > 0) OVER () FROM t", `SELECT COUNT(*) FILTER (WHERE "a">0) OVER () FROM "t"`},
		{"issue09_values", "SELECT CURRENT_TIME(3), LOCALTIME(2), CURRENT_ROLE, USER", "SELECT CURRENT_TIME(3),LOCALTIME(2),CURRENT_ROLE,USER"},
		{"issue09_typmod", "SELECT CURRENT_TIME(2147483647)", "SELECT CURRENT_TIME(2147483647)"},
		{"issue10_values_table", "VALUES (1) UNION ALL TABLE t", `VALUES (1) UNION ALL TABLE "t"`},
		{"issue10_nested", "SELECT 1 UNION (SELECT 2 EXCEPT SELECT 3)", "SELECT 1 UNION (SELECT 2 EXCEPT SELECT 3)"},
		{"issue10_precedence", "SELECT 1 UNION SELECT 2 INTERSECT SELECT 3", "SELECT 1 UNION SELECT 2 INTERSECT SELECT 3"},
		{"issue10_braced_left", "(SELECT 1 UNION SELECT 2 LIMIT 1) EXCEPT SELECT 3", "(SELECT 1 UNION SELECT 2 LIMIT 1) EXCEPT SELECT 3"},
		{"issue10_braced_limit_offset", "(SELECT 1 UNION SELECT 2 LIMIT 3) OFFSET 2", "(SELECT 1 UNION SELECT 2 LIMIT 3 OFFSET 2)"},
		{"issue10_braced_offset_limit", "(SELECT 1 UNION SELECT 2 OFFSET 2) LIMIT 3", "(SELECT 1 UNION SELECT 2 LIMIT 3 OFFSET 2)"},
		{"issue10_braced_values_limit_offset", "(VALUES (1), (2) LIMIT 3) OFFSET 2", "(VALUES (1), (2) LIMIT 3 OFFSET 2)"},
		{"issue10_braced_table_offset_limit", "(TABLE t OFFSET 2) LIMIT 3", `(TABLE "t" LIMIT 3 OFFSET 2)`},
		{"issue10_values_options", "WITH c AS (SELECT 1) VALUES (1) ORDER BY 1 LIMIT 2", `WITH "c" AS (SELECT 1) VALUES (1) ORDER BY 1 LIMIT 2`},
		{"issue10_table_options", "TABLE t LIMIT 2", "TABLE \"t\" LIMIT 2"},
		{"issue10_table_with", "WITH c AS (SELECT 1) TABLE c OFFSET 1", `WITH "c" AS (SELECT 1) TABLE "c" OFFSET 1`},
		{"issue10_values_braced", "SELECT 1 UNION (VALUES (2) OFFSET 1)", "SELECT 1 UNION (VALUES (2) OFFSET 1)"},
		{"issue10_values_with_braced", "SELECT 1 UNION (WITH c AS (SELECT 2) VALUES (3) OFFSET 1)", `SELECT 1 UNION (WITH "c" AS (SELECT 2) VALUES (3) OFFSET 1)`},
		{"issue10_with_select", "WITH c AS (SELECT 1) SELECT 2", `WITH "c" AS (SELECT 1) SELECT 2`},
		{"issue10_with_insert", "WITH c AS (SELECT 1) INSERT INTO t VALUES (2)", `WITH "c" AS (SELECT 1) INSERT INTO "t" VALUES (2)`},
		{"issue10_with_update", "WITH c AS (SELECT 1) UPDATE t SET a = 2 WHERE a = 1", `WITH "c" AS (SELECT 1) UPDATE "t" SET "a"=2 WHERE "a"=1`},
		{"issue10_with_delete", "WITH c AS (SELECT 1) DELETE FROM t WHERE a = 1", `WITH "c" AS (SELECT 1) DELETE FROM "t" WHERE "a"=1`},
		{"issue11_distinct", "SELECT DISTINCT a FROM t OFFSET 2", `SELECT DISTINCT "a" FROM "t" OFFSET 2`},
		{"issue11_ties", "SELECT a FROM t ORDER BY a OFFSET 2 ROWS FETCH FIRST 3 ROWS WITH TIES", `SELECT "a" FROM "t" ORDER BY "a" FETCH FIRST 3 ROWS WITH TIES OFFSET 2`},
		{"issue12_default", "SELECT $1 WHERE TRUE LIMIT 3 OFFSET 2", "SELECT $1 WHERE TRUE LIMIT 3 OFFSET 2"},
		{"issue12_quotes", "SELECT 'a\"b` c', \"Case\", \"a\"\"b\" FROM \"Order\"", "SELECT 'a\"b` c',\"Case\",\"a\"\"b\" FROM \"Order\""},
		{"issue13_json", `SELECT * FROM JSON_TABLE('{"a":1}', '$' COLUMNS (a int PATH '$.a')) AS j`, `SELECT * FROM JSON_TABLE('{"a":1}', '$' COLUMNS("a" "int4" PATH '$.a')) AS "j"`},
		{"issue13_xml", `SELECT * FROM XMLTABLE('/r' PASSING '<r/>' COLUMNS a int PATH '@a' DEFAULT 2 NOT NULL) AS x`, `SELECT * FROM XMLTABLE('/r' PASSING '<r/>' COLUMNS "a" "int4" PATH '@a' DEFAULT 2 NOT NULL) AS "x"`},
		{"issue13_xml_preserve", `SELECT XMLPARSE(DOCUMENT '<r/>' PRESERVE WHITESPACE)`, `SELECT XMLPARSE(DOCUMENT '<r/>' PRESERVE WHITESPACE)`},
		{"issue13_json_behavior", `SELECT * FROM JSON_TABLE('{}', '$' COLUMNS (a int EXISTS PATH '$.a' FALSE ON ERROR)) AS j`, `SELECT * FROM JSON_TABLE('{}', '$' COLUMNS("a" "int4" EXISTS PATH '$.a' FALSE ON ERROR)) AS "j"`},
		{"issue13_json_path_expr", `SELECT * FROM JSON_TABLE('{}', $1 AS root COLUMNS (a int)) AS j`, `SELECT * FROM JSON_TABLE('{}', $1 AS "root" COLUMNS("a" "int4")) AS "j"`},
		{"issue13_json_options", `SELECT * FROM JSON_TABLE('{}', '$' AS root COLUMNS (ord FOR ORDINALITY, a text PATH '$.a' WITH CONDITIONAL WRAPPER OMIT QUOTES NULL ON EMPTY ERROR ON ERROR, NESTED PATH '$.items[*]' AS items COLUMNS (b int PATH '$.b')) EMPTY ARRAY ON ERROR) AS j`, `SELECT * FROM JSON_TABLE('{}', '$' AS "root" COLUMNS("ord" FOR ORDINALITY, "a" "text" PATH '$.a' WITH CONDITIONAL WRAPPER OMIT QUOTES NULL ON EMPTY ERROR ON ERROR, NESTED PATH '$.items[*]' AS "items" COLUMNS ("b" "int4" PATH '$.b')) EMPTY ARRAY ON ERROR) AS "j"`},
		{"issue13_xml_options", `SELECT * FROM XMLTABLE(XMLNAMESPACES('urn:test' AS p, DEFAULT 'urn:default'), '/r' PASSING '<r/>' COLUMNS ord FOR ORDINALITY, a text PATH '@a' DEFAULT 'MOCK_TEXT' NOT NULL) AS x`, `SELECT * FROM XMLTABLE(XMLNAMESPACES('urn:test' AS "p", DEFAULT 'urn:default'), '/r' PASSING '<r/>' COLUMNS "ord" FOR ORDINALITY, "a" "text" PATH '@a' DEFAULT 'MOCK_TEXT' NOT NULL) AS "x"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n, err := parser.New().ParseOneStmt(tc.sql)
			if err != nil {
				t.Fatal(err)
			}
			got := restoreNode(t, n)
			if got != tc.want {
				t.Fatalf("restore = %q, want %q", got, tc.want)
			}
			again, err := parser.New().ParseOneStmt(got)
			if err != nil {
				t.Fatalf("reparse %q: %v", got, err)
			}
			if next := restoreNode(t, again); next != got {
				t.Fatalf("unstable restore: %q -> %q", got, next)
			}
		})
	}
}

func TestAuditTimePrecisionRegression(t *testing.T) {
	for _, name := range []string{"CURRENT_TIME", "CURRENT_TIMESTAMP", "LOCALTIME", "LOCALTIMESTAMP"} {
		t.Run(name, func(t *testing.T) {
			for _, precision := range []string{"0", "2147483647"} {
				t.Run(precision, func(t *testing.T) {
					sql := fmt.Sprintf("SELECT %s(%s)", name, precision)
					n, err := parser.New().ParseOneStmt(sql)
					if err != nil {
						t.Fatal(err)
					}
					if got := restoreNode(t, n); got != sql {
						t.Fatalf("restore = %q, want %q", got, sql)
					}
				})
			}
			for _, precision := range []string{"-1", "2147483648", "4294967295", "4294967296", "999999999999999999999"} {
				t.Run(precision, func(t *testing.T) {
					sql := fmt.Sprintf("SELECT %s(%s)", name, precision)
					if _, err := parser.New().ParseOneStmt(sql); err == nil {
						t.Fatalf("accepted %q", sql)
					}
				})
			}
		})
	}
}

func TestAuditLexerRegression(t *testing.T) {
	for _, sql := range []string{"SELECT $0", "SELECT $2147483648", "SELECT $999999999999999999999", "SELECT 1; /*", "SELECT 1; '", "SELECT 1; E'", "SELECT 1; $$", `SELECT 1; "`, "SELECT 1; B'", "SELECT 1; U&'"} {
		if _, _, err := parser.New().Parse(sql); err == nil {
			t.Errorf("accepted %q", sql)
		}
	}
	for _, tc := range []struct{ sql, value string }{
		{"SELECT 'abc   \nxyz'", "abc   \nxyz"},
		{`SELECT E'it\'s // data'`, "it's // data"},
		{"SELECT $$// preserved$$ /* // block */ // extension\n", "// preserved"},
	} {
		n, err := parser.New().ParseOneStmt(tc.sql)
		if err != nil {
			t.Fatal(err)
		}
		if got := n.(*ast.SelectStmt).Fields.Fields[0].Expr.(ast.ValueExpr).GetValue(); got != tc.value {
			t.Errorf("%q = %q, want %q", tc.sql, got, tc.value)
		}
	}
}

func TestAuditTypeRegression(t *testing.T) {
	n, err := parser.New().ParseOneStmt("CREATE TABLE t (id uuid, at timestamptz(3), tags text[][4], d app.money, n numeric(12, -2), i interval DAY TO SECOND(3), g app.vector(2,3,4), b bit(2+1))")
	if err != nil {
		t.Fatal(err)
	}
	stmt := n.(*ast.CreateTableStmt)
	wants := []string{`"uuid"`, `"timestamptz"(3)`, `"text"[][4]`, `"app"."money"`, `"numeric"(12, -2)`, `INTERVAL DAY TO SECOND(3)`, `"app"."vector"(2, 3, 4)`, `"bit"(2+1)`}
	for i, col := range stmt.Cols {
		if got := col.Tp.String(); got != wants[i] {
			t.Errorf("type %d = %q, want %q", i, got, wants[i])
		}
		clone := col.Tp.Clone()
		if !col.Tp.Equal(clone) {
			t.Fatal("clone differs")
		}
		clone.PGName[0] = "different"
		if col.Tp.Equal(clone) || col.Tp.PGName[0] == "different" {
			t.Fatal("type metadata ignored or aliased")
		}
	}
	if _, err := parser.New().ParseOneStmt(restoreNode(t, n)); err != nil {
		t.Fatal(err)
	}
}

func TestAuditInvalidTypeLiteralModifiers(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{
		{"order", "SELECT app.vector(2 ORDER BY 1) '1,2'"},
		{"named_argument", "SELECT app.vector(size => 2) '1,2'"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parser.New().ParseOneStmt(tc.sql); err == nil {
				t.Fatalf("accepted %q", tc.sql)
			}
		})
	}
}

type auditVisitor struct {
	replace bool
	seen    int
	formats []ast.JsonEncoding
}

func (v *auditVisitor) Enter(n ast.Node) (ast.Node, bool) {
	if f, ok := n.(*ast.JsonFormat); ok {
		v.formats = append(v.formats, f.Encoding)
	}
	if role, ok := n.(*ast.RoleSpec); ok && role.Rolename == "reader" {
		v.seen++
		if v.replace {
			copy := *role
			copy.Rolename = "renamed"
			return &copy, false
		}
	}
	if col, ok := n.(*ast.ColumnName); ok && col.Name.O == "source" {
		v.seen++
	}
	if ref, ok := n.(*ast.TableName); ok && ref.Name.O == "saved" {
		v.seen++
		if v.replace {
			copy := *ref
			copy.Name.O = "renamed"
			return &copy, false
		}
	}
	if _, ok := n.(*ast.NumericLiteral); ok {
		v.seen++
		if v.replace {
			return &ast.NumericLiteral{Value: "2.5"}, false
		}
	}
	return n, false
}

func (*auditVisitor) Leave(n ast.Node) (ast.Node, bool) { return n, true }

func TestAuditVisitorRegression(t *testing.T) {
	for _, sql := range []string{"SELECT 1.5 INTO saved", "SELECT (1.5)[1]", "SELECT JSON_SCALAR(1.5)", "GRANT SELECT ON TABLE t TO reader", "REVOKE SELECT ON TABLE t FROM reader", "GRANT readers TO reader", "REVOKE readers FROM reader"} {
		n, err := parser.New().ParseOneStmt(sql)
		if err != nil {
			t.Fatal(err)
		}
		v := &auditVisitor{replace: true}
		if _, ok := n.Accept(v); !ok || v.seen == 0 {
			t.Fatalf("visitor missed %q", sql)
		}
		if sql == "SELECT 1.5 INTO saved" && restoreNode(t, n) != `SELECT 2.5 INTO "renamed"` {
			t.Fatal(restoreNode(t, n))
		}
		if strings.HasPrefix(sql, "GRANT") || strings.HasPrefix(sql, "REVOKE") {
			if got := restoreNode(t, n); !strings.Contains(got, `"renamed"`) {
				t.Fatalf("role replacement discarded: %q", got)
			}
		}
	}
}

func TestAuditExpressionFlags(t *testing.T) {
	for _, tc := range []struct {
		name, sql string
		flag      uint64
	}{
		{"sql_value", "SELECT CURRENT_TIMESTAMP", ast.FlagHasFunc},
		{"function_order", "SELECT f(1 ORDER BY $1)", ast.FlagHasFunc | ast.FlagHasParamMarker},
		{"aggregate_order", "SELECT count(1 ORDER BY $1)", ast.FlagHasAggregateFunc | ast.FlagHasParamMarker},
		{"aggregate_filter", "SELECT count(*) FILTER (WHERE $1 > 0)", ast.FlagHasAggregateFunc | ast.FlagHasParamMarker},
		{"aggregate_within_group", "SELECT percentile_cont(0.5) WITHIN GROUP (ORDER BY $1)", ast.FlagHasAggregateFunc | ast.FlagHasParamMarker},
		{"window_filter", "SELECT count(*) FILTER (WHERE $1 > 0) OVER ()", ast.FlagHasWindowFunc | ast.FlagHasParamMarker},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := mustParseSelect(t, tc.sql)
			if got := n.Fields.Fields[0].Expr.GetFlag(); got != tc.flag {
				t.Fatalf("expression flags = %d, want %d", got, tc.flag)
			}
		})
	}
}

func TestAuditUnsupportedRestore(t *testing.T) {
	for _, n := range []ast.Node{&ast.A_ArrayExpr{}, &ast.JsonObjectConstructor{}, &ast.XmlExpr{}, &ast.PrivElem{}} {
		var out strings.Builder
		if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil {
			t.Fatalf("silently restored unsupported %T: %q", n, out.String())
		}
	}
	for _, tc := range []struct{ name, sql string }{
		{"passing", `SELECT * FROM JSON_TABLE('{}', '$' PASSING 1 AS x COLUMNS(a int)) AS j`},
		{"input_format", `SELECT * FROM JSON_TABLE('{}' FORMAT JSON, '$' COLUMNS(a int)) AS j`},
		{"column_format", `SELECT * FROM JSON_TABLE('{}', '$' COLUMNS(a text FORMAT JSON)) AS j`},
		{"default_empty", `SELECT * FROM JSON_TABLE('{}', '$' COLUMNS(a int DEFAULT $1 ON EMPTY)) AS j`},
		{"default_error", `SELECT * FROM JSON_TABLE('{}', '$' COLUMNS(a int DEFAULT $1 ON ERROR)) AS j`},
		{"default_table_error", `SELECT * FROM JSON_TABLE('{}', '$' COLUMNS(a int) DEFAULT $1 ON ERROR) AS j`},
		{"rollup_child", `SELECT 1 GROUP BY ROLLUP (JSON_ARRAY(1))`},
		{"cube_child", `SELECT 1 GROUP BY CUBE (JSON_ARRAY(1))`},
		{"function_privilege", "GRANT EXECUTE ON FUNCTION f(integer) TO reader"},
		{"procedure_privilege", "REVOKE EXECUTE ON PROCEDURE p(integer) FROM reader"},
		{"routine_privilege", "GRANT EXECUTE ON ROUTINE f(integer) TO reader"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n, err := parser.New().ParseOneStmt(tc.sql)
			if err != nil {
				t.Fatal(err)
			}
			var out strings.Builder
			if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil {
				t.Fatalf("silently discarded %s: %q", tc.name, out.String())
			}
		})
	}
}

func TestAuditJSONTableExpressionChildren(t *testing.T) {
	n, err := parser.New().ParseOneStmt(`SELECT * FROM JSON_TABLE('{}', $1 AS root PASSING $2 AS value COLUMNS(a int DEFAULT $3 ON EMPTY DEFAULT $4 ON ERROR)) AS j`)
	if err != nil {
		t.Fatal(err)
	}
	if got := countParams(n); got != 4 {
		t.Fatalf("parameters = %d, want 4", got)
	}
}

func TestAuditJSONFormatAST(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{
		{"table", `SELECT * FROM JSON_TABLE('{}' FORMAT JSON ENCODING UTF16, '$' COLUMNS(a text FORMAT JSON ENCODING UTF8)) AS j`},
		{"serialize", `SELECT JSON_SERIALIZE('{}' FORMAT JSON ENCODING UTF16 RETURNING text FORMAT JSON ENCODING UTF8)`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n, err := parser.New().ParseOneStmt(tc.sql)
			if err != nil {
				t.Fatal(err)
			}
			v := &auditVisitor{}
			if _, ok := n.Accept(v); !ok || len(v.formats) != 2 || v.formats[0] != ast.JS_ENC_UTF16 || v.formats[1] != ast.JS_ENC_UTF8 {
				t.Fatalf("JSON format encodings = %v, want UTF16 and UTF8", v.formats)
			}
		})
	}
}

func TestAuditJSONEncodingRegression(t *testing.T) {
	for _, tc := range []struct {
		name, clause string
		encoding     ast.JsonEncoding
		wantErr      bool
	}{
		{"default", "", ast.JS_ENC_DEFAULT, false},
		{"utf8", "ENCODING utf8", ast.JS_ENC_UTF8, false},
		{"utf16", "ENCODING UTF16", ast.JS_ENC_UTF16, false},
		{"utf32", "ENCODING UTF32", ast.JS_ENC_UTF32, false},
		{"invalid", "ENCODING invalid", ast.JS_ENC_DEFAULT, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n, err := parser.New().ParseOneStmt(fmt.Sprintf("SELECT JSON('[]' FORMAT JSON %s)", tc.clause))
			if tc.wantErr {
				if err == nil {
					t.Fatal("accepted unknown JSON encoding")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			v := &auditVisitor{}
			if _, ok := n.Accept(v); !ok || len(v.formats) != 1 || v.formats[0] != tc.encoding {
				t.Fatalf("JSON format encodings = %v, want %d", v.formats, tc.encoding)
			}
		})
	}
}

func TestAuditInvalidRoleNames(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{
		{"grant_columns", "GRANT readers(a) TO member"},
		{"revoke_columns", "REVOKE readers(a) FROM member"},
		{"public_role", "CREATE ROLE public"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parser.New().ParseOneStmt(tc.sql); err == nil {
				t.Fatalf("accepted %q", tc.sql)
			}
		})
	}
}

func TestAuditGrantRestoreNameValidation(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{
		{"privilege", `GRANT "SELECT ON TABLE t TO reader; DROP TABLE t; --" ON TABLE t TO reader`},
		{"grant_option", `GRANT readers TO member WITH "ADMIN TRUE; DROP TABLE t; --" TRUE`},
		{"revoke_option", `REVOKE "ADMIN OPTION FOR readers FROM member; DROP TABLE t; --" OPTION FOR readers FROM member`},
		{"unknown_privilege", "GRANT unknown_privilege ON TABLE t TO reader"},
		{"unknown_grant_option", "GRANT readers TO member WITH unknown_option TRUE"},
		{"unknown_revoke_option", "REVOKE unknown_option OPTION FOR readers FROM member"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n, err := parser.New().ParseOneStmt(tc.sql)
			if err != nil {
				t.Fatal(err)
			}
			var out strings.Builder
			if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil {
				t.Fatalf("restored unsafe name: %q", out.String())
			}
			if strings.Contains(out.String(), ";") {
				t.Fatalf("unsafe name reached output: %q", out.String())
			}
		})
	}
}

func TestAuditPrivilegeRestoreNames(t *testing.T) {
	for _, name := range []string{
		"all", "select", "insert", "update", "delete", "truncate", "references", "trigger", "maintain",
		"usage", "create", "connect", "temporary", "temp", "execute", "set", "alter system",
	} {
		t.Run(name, func(t *testing.T) {
			var out strings.Builder
			n := &ast.PrivElem{Name: name}
			if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil {
				t.Fatal(err)
			}
			if got := out.String(); got != strings.ToUpper(name) {
				t.Fatalf("privilege = %q, want %q", got, strings.ToUpper(name))
			}
		})
	}
}

func TestAuditParserReuse(t *testing.T) {
	p := parser.New()
	first, _, err := p.Parse("SELECT 1")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := p.Parse("DELETE FROM t"); err != nil {
		t.Fatal(err)
	}
	if _, ok := first[0].(*ast.SelectStmt); !ok {
		t.Fatalf("old result replaced by %T", first[0])
	}
}

func TestAuditJSONTrailingInput(t *testing.T) {
	for _, text := range []string{`{} {}`, `{} garbage`, `1 2`} {
		if _, err := json.ParseBinaryFromString(text); err == nil {
			t.Errorf("accepted %q", text)
		}
	}
	if _, err := json.ParseBinaryFromString("{} \n\t"); err != nil {
		t.Fatal(err)
	}
}

var parserSink *parser.Parser

func BenchmarkParserNew(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		parserSink = parser.New()
	}
}

func BenchmarkLexerOperator(b *testing.B) {
	for _, size := range []int{8192, 16384, 32768} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			input := strings.Repeat("+", size)
			b.SetBytes(int64(size))
			b.ReportAllocs()
			for b.Loop() {
				_ = parser.NewLexer(input).NextToken()
			}
		})
	}
}
