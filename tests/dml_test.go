package parser_test

import (
	"reflect"
	"strings"
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/format"
)

func TestDMLSharedCore(t *testing.T) {
	insert := mustParseOne(t, `INSERT INTO "S"."T" AS x ("a") VALUES (1) RETURNING a;`).(*ast.InsertStmt)
	if insert.Table == nil || insert.Table.TableRefCore() == nil {
		t.Fatalf("insert table = %#v", insert.Table)
	}
	if insert.TargetAlias.Name != "x" || insert.TargetAlias.Quote != lex.QuoteUnquoted {
		t.Fatalf("insert alias = %#v", insert.TargetAlias)
	}
	if len(insert.Columns) != 1 || len(insert.Values) != 1 || len(insert.Values[0]) != 1 {
		t.Fatalf("insert columns/values = %#v %#v", insert.Columns, insert.Values)
	}
	if insert.Returning == nil || len(insert.Returning.Items) != 1 {
		t.Fatalf("insert returning = %#v", insert.Returning)
	}

	update := mustParseOne(t, `UPDATE t SET a = 1 WHERE b = 2 RETURNING a;`).(*ast.UpdateStmt)
	if update.Table == nil || len(update.Assignments) != 1 || update.Where == nil {
		t.Fatalf("update = %#v", update)
	}
	if update.Assignments[0].Target == nil || update.Assignments[0].Expr == nil {
		t.Fatalf("assignment = %#v", update.Assignments[0])
	}

	del := mustParseOne(t, `DELETE FROM t USING u WHERE t.id = u.id RETURNING a;`).(*ast.DeleteStmt)
	if del.Table == nil || del.Using == nil || del.Where == nil {
		t.Fatalf("delete = %#v", del)
	}
}

func TestAssignmentTargets(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want string
	}{
		{"column", `UPDATE t SET a = 1;`, `UPDATE "t" SET "a"=1`},
		{"row", `UPDATE t SET (a, b) = ROW(1, 2);`, `UPDATE "t" SET ("a", "b")=ROW(1,2)`},
		{"subscript", `UPDATE t SET a[1] = 1;`, `UPDATE "t" SET "a"[1]=1`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt := mustParseOne(t, tt.sql).(*ast.UpdateStmt)
			if len(stmt.Assignments) != 1 {
				t.Fatalf("assignments = %d", len(stmt.Assignments))
			}
			target := stmt.Assignments[0].Target
			if target == nil {
				t.Fatal("assignment has no target")
			}
			switch tt.name {
			case "row":
				row, ok := target.(*ast.RowTarget)
				if !ok || len(row.Columns) != 2 {
					t.Fatalf("row target = %#v", target)
				}
			case "subscript":
				if _, ok := target.(*ast.A_Indirection); !ok {
					t.Fatalf("subscript target = %T", target)
				}
			default:
				if _, ok := target.(*ast.ColumnNameExpr); !ok {
					t.Fatalf("column target = %T", target)
				}
			}
			if got := restoreString(t, stmt); got != tt.want {
				t.Fatalf("restore = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLimitRestore(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want string
	}{
		{"limit", `SELECT 1 LIMIT 10;`, `SELECT 1 LIMIT 10`},
		{"offset", `SELECT 1 OFFSET 5;`, `SELECT 1 OFFSET 5`},
		{"limit offset", `SELECT 1 LIMIT 10 OFFSET 5;`, `SELECT 1 LIMIT 10 OFFSET 5`},
		{"fetch only", `SELECT 1 FETCH FIRST 3 ROWS ONLY;`, `SELECT 1 LIMIT 3`},
		{"fetch with ties", `SELECT 1 FETCH FIRST 3 ROWS WITH TIES;`, `SELECT 1 FETCH FIRST 3 ROWS WITH TIES`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt := mustParseSelect(t, tt.sql)
			if stmt.Limit == nil || stmt.Limit.LimitCore() == nil {
				t.Fatalf("limit = %#v", stmt.Limit)
			}
			if got := restoreString(t, stmt); got != tt.want {
				t.Fatalf("restore = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDMLRestore(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want string
	}{
		{"insert values", `INSERT INTO t (a, b) VALUES (1, 2);`, `INSERT INTO "t" ("a","b") VALUES (1,2)`},
		{"insert default values", `INSERT INTO t DEFAULT VALUES;`, `INSERT INTO "t" DEFAULT VALUES`},
		{"insert select", `INSERT INTO t SELECT 1;`, `INSERT INTO "t" SELECT 1`},
		{"insert alias", `INSERT INTO t AS x VALUES (1);`, `INSERT INTO "t" AS "x" VALUES (1)`},
		{"insert on conflict", `INSERT INTO t VALUES (1) ON CONFLICT DO NOTHING;`, `INSERT INTO "t" VALUES (1) ON CONFLICT DO NOTHING`},
		{"update", `UPDATE t SET a = 1 WHERE b = 2;`, `UPDATE "t" SET "a"=1 WHERE "b"=2`},
		{"update from", `UPDATE t SET a = 1 FROM u WHERE t.id = u.id;`, `UPDATE "t" SET "a"=1 FROM "u" WHERE "t"."id"="u"."id"`},
		{"delete", `DELETE FROM t WHERE a = 1;`, `DELETE FROM "t" WHERE "a"=1`},
		{"delete using", `DELETE FROM t USING u WHERE t.id = u.id;`, `DELETE FROM "t" USING "u" WHERE "t"."id"="u"."id"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt := mustParseOne(t, tt.sql)
			got := restoreString(t, stmt)
			if got != tt.want {
				t.Fatalf("restore = %q, want %q", got, tt.want)
			}
			if again := restoreString(t, mustParseOne(t, got)); again != got {
				t.Fatalf("second restore = %q, want %q", again, got)
			}
		})
	}
}

func TestDMLInvalidRestore(t *testing.T) {
	tests := []struct {
		name string
		node ast.Node
	}{
		{"nil insert", (*ast.InsertStmt)(nil)},
		{"insert without table", &ast.InsertStmt{}},
		{"nil update", (*ast.UpdateStmt)(nil)},
		{"update without table", &ast.UpdateStmt{}},
		{"update without assignments", &ast.UpdateStmt{Table: &ast.TableSource{}}},
		{"nil delete", (*ast.DeleteStmt)(nil)},
		{"delete without table", &ast.DeleteStmt{}},
		{"nil limit", (*ast.Limit)(nil)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out strings.Builder
			ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &out)
			if err := tt.node.Restore(ctx); err == nil {
				t.Fatal("invalid statement restored without error")
			}
		})
	}
}

func TestSharedTypeIdentity(t *testing.T) {
	if reflect.TypeFor[ast.Assignment]() != reflect.TypeFor[pkast.Assignment]() {
		t.Error("Assignment is not the shared type")
	}
	if reflect.TypeFor[ast.LimitNode]() != reflect.TypeFor[pkast.LimitNode]() {
		t.Error("LimitNode is not the shared type")
	}
	if reflect.TypeFor[ast.OrderItemNode]() != reflect.TypeFor[pkast.OrderItemNode]() {
		t.Error("OrderItemNode is not the shared type")
	}
	if reflect.TypeFor[ast.GroupItemNode]() != reflect.TypeFor[pkast.GroupItemNode]() {
		t.Error("GroupItemNode is not the shared type")
	}
}
