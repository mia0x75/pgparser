package parser_test

import (
	"reflect"
	"testing"

	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/parser"
	"github.com/mia0x75/pgparser/postgres"
)

func TestParseSharedPreparedStatementCore(t *testing.T) {
	stmt := mustParseOne(t, "PREPARE alignment_p (INT) AS SELECT $1").(*ast.PrepareStmt)
	if stmt.Name.Name != "alignment_p" || len(stmt.Params) != 1 || stmt.Stmt == nil {
		t.Fatalf("PREPARE core = %#v", stmt)
	}
	exec := mustParseOne(t, "EXECUTE alignment_p (1)").(*ast.ExecuteStmt)
	if exec.Name.Name != "alignment_p" || len(exec.Args) != 1 {
		t.Fatalf("EXECUTE core = %#v", exec)
	}
	dealloc := mustParseOne(t, "DEALLOCATE PREPARE alignment_p").(*ast.DeallocateStmt)
	if dealloc.Name.Name != "alignment_p" || dealloc.All {
		t.Fatalf("DEALLOCATE core = %#v", dealloc)
	}
}

func TestParseSharedTransactionCore(t *testing.T) {
	for _, tc := range []struct {
		sql  string
		want ast.StmtNode
	}{
		{"BEGIN", &ast.BeginStmt{}},
		{"COMMIT WORK", &ast.CommitStmt{}},
		{"ROLLBACK", &ast.RollbackStmt{}},
	} {
		t.Run(tc.sql, func(t *testing.T) {
			stmt := mustParseOne(t, tc.sql)
			if reflect.TypeOf(stmt) != reflect.TypeOf(tc.want) {
				t.Fatalf("statement = %T, want %T", stmt, tc.want)
			}
		})
	}
	save := mustParseOne(t, "SAVEPOINT alignment_s").(*ast.SavepointStmt)
	rollback := mustParseOne(t, "ROLLBACK TO SAVEPOINT alignment_s").(*ast.RollbackStmt)
	release := mustParseOne(t, "RELEASE SAVEPOINT alignment_s").(*ast.ReleaseSavepointStmt)
	if save.Name != "alignment_s" || rollback.SavepointName != save.Name || release.Name != save.Name {
		t.Fatalf("savepoint names = %#v / %#v / %#v", save, rollback, release)
	}
}

func TestParseSharedGrantCore(t *testing.T) {
	stmt := mustParseOne(t, "GRANT SELECT ON alignment_t TO alignment_user WITH GRANT OPTION").(*ast.GrantStmt)
	if len(stmt.Privs) != 1 || stmt.Privs[0].Priv != postgres.SelectPriv || !stmt.GrantOption ||
		len(stmt.Objects) != 1 || len(stmt.Users) != 1 {
		t.Fatalf("GRANT core = %#v", stmt)
	}
	name := stmt.Objects[0].TableRefCore().Name.Parts
	if len(name) != 1 || name[0].Name != "alignment_t" {
		t.Fatalf("GRANT object = %#v", name)
	}
}

func TestParseSharedMaintenanceCore(t *testing.T) {
	explain := mustParseOne(t, "EXPLAIN SELECT 1").(*ast.ExplainStmt)
	if _, ok := explain.Stmt.(*ast.SelectStmt); !ok {
		t.Fatalf("EXPLAIN body = %T", explain.Stmt)
	}
	analyze := mustParseOne(t, "ANALYZE alignment_t").(*ast.AnalyzeTableStmt)
	if analyze.Relations == nil || len(analyze.Relations.Items) != 1 {
		t.Fatalf("ANALYZE targets = %#v", analyze.Relations)
	}
}

func TestParseUtilityDialectBoundaries(t *testing.T) {
	for _, sql := range []string{
		"PREPARE alignment_p FROM 'SELECT 1'",
		"EXECUTE alignment_p USING @a",
		"LOAD DATA INFILE 'mock.csv' INTO TABLE alignment_t",
		"SHOW REPLICA STATUS",
	} {
		t.Run(sql, func(t *testing.T) {
			if _, err := parser.New().ParseOneStmt(sql); err == nil {
				t.Fatal("accepted MySQL utility syntax")
			}
		})
	}
}

func TestParseDialectImportCore(t *testing.T) {
	stmt := mustParseOne(t, "COPY alignment_t (id) FROM STDIN").(*ast.CopyStmt)
	if !stmt.IsFrom || !stmt.Std || stmt.Table == nil || len(stmt.Columns) != 1 || stmt.Columns[0].Name != "id" {
		t.Fatalf("COPY core = %#v", stmt)
	}
	name := stmt.Table.TableRefCore().Name.Parts
	if len(name) != 1 || name[0].Name != "alignment_t" {
		t.Fatalf("COPY target = %#v", name)
	}
}
