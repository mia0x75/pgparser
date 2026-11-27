package parser_test

import (
	"testing"

	"github.com/mia0x75/pgparser/ast"
)

func TestParseSharedCreateTableCore(t *testing.T) {
	stmt := mustParseOne(t, "CREATE TABLE alignment_t (id INT NOT NULL, name VARCHAR(20))").(*ast.CreateTableStmt)
	if stmt.Table == nil || len(stmt.Cols) != 2 {
		t.Fatalf("CREATE TABLE core = %#v", stmt)
	}
	if stmt.Cols[0].Name == nil || stmt.Cols[0].Name.Name != "id" || stmt.Cols[0].Tp == nil {
		t.Fatalf("first column = %#v", stmt.Cols[0])
	}
	if stmt.Cols[1].Name == nil || stmt.Cols[1].Name.Name != "name" || stmt.Cols[1].Tp == nil {
		t.Fatalf("second column = %#v", stmt.Cols[1])
	}
}

func TestParseSharedCreateTableAsCore(t *testing.T) {
	stmt := mustParseOne(t, "CREATE TABLE alignment_t AS SELECT 1 AS id").(*ast.CreateTableStmt)
	if stmt.Table == nil || stmt.Select == nil {
		t.Fatalf("CREATE TABLE AS core = %#v", stmt)
	}
}

func TestParseSharedAlterTableAddColumnCore(t *testing.T) {
	stmt := mustParseOne(t, "ALTER TABLE alignment_t ADD COLUMN extra INT").(*ast.AlterTableStmt)
	if stmt.Table == nil || len(stmt.Specs) != 1 || len(stmt.Specs[0].NewColumns) != 1 {
		t.Fatalf("ALTER TABLE core = %#v", stmt)
	}
	column := stmt.Specs[0].NewColumns[0]
	if column.Name == nil || column.Name.Name != "extra" || column.Tp == nil {
		t.Fatalf("added column = %#v", column)
	}
}
