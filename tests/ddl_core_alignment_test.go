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

func TestParseSharedTableConstraintCore(t *testing.T) {
	const sql = "CREATE TABLE align_child (id INT, parent_id INT, CONSTRAINT pk PRIMARY KEY (id), CONSTRAINT uq UNIQUE (parent_id), CONSTRAINT fk FOREIGN KEY (parent_id) REFERENCES align_parent (id) ON DELETE CASCADE, CONSTRAINT ck CHECK (id > 0))"
	stmt := mustParseOne(t, sql).(*ast.CreateTableStmt)
	if len(stmt.Constraints) != 4 {
		t.Fatalf("table constraints = %#v", stmt.Constraints)
	}
	wantTypes := []ast.ConstraintType{ast.ConstraintPrimaryKey, ast.ConstraintUniq, ast.ConstraintForeignKey, ast.ConstraintCheck}
	for i, want := range wantTypes {
		if stmt.Constraints[i].Tp != want {
			t.Fatalf("constraint %d type = %v, want %v", i, stmt.Constraints[i].Tp, want)
		}
	}
	if len(stmt.Constraints[0].Keys) != 1 || len(stmt.Constraints[1].Keys) != 1 {
		t.Fatalf("key constraints = %#v", stmt.Constraints[:2])
	}
	fk := stmt.Constraints[2]
	if fk.Refer == nil || fk.Refer.OnDelete == nil || fk.Refer.OnDelete.ReferOpt != ast.ReferOptionCascade {
		t.Fatalf("foreign key reference = %#v", fk.Refer)
	}
	if stmt.Constraints[3].Expr == nil {
		t.Fatalf("check constraint = %#v", stmt.Constraints[3])
	}
}

func TestParseSharedCreateIndexCore(t *testing.T) {
	stmt := mustParseOne(t, "CREATE UNIQUE INDEX align_idx ON align_child (parent_id)").(*ast.CreateIndexStmt)
	if !stmt.Unique || len(stmt.IndexName.Parts) != 1 || stmt.IndexName.Parts[0].Name != "align_idx" ||
		stmt.Table == nil || len(stmt.IndexColNames) != 1 || stmt.IndexColNames[0].Column == nil || stmt.IndexColNames[0].Column.Name != "parent_id" {
		t.Fatalf("CREATE INDEX core = %#v", stmt)
	}
}

func TestParseSharedCreateDatabaseCore(t *testing.T) {
	stmt := mustParseOne(t, "CREATE DATABASE IF NOT EXISTS alignment_db").(*ast.CreateDatabaseStmt)
	if !stmt.IfNotExists || stmt.Name.Name != "alignment_db" {
		t.Fatalf("CREATE DATABASE core = %#v", stmt)
	}
}

func TestParseSharedCreateViewCore(t *testing.T) {
	stmt := mustParseOne(t, "CREATE VIEW alignment_view AS SELECT 1 AS id").(*ast.CreateViewStmt)
	_, isSelect := stmt.Select.(*ast.SelectStmt)
	if stmt.ViewName == nil || stmt.Select == nil || len(stmt.ViewName.TableRefCore().Name.Parts) != 1 ||
		stmt.ViewName.TableRefCore().Name.Parts[0].Name != "alignment_view" || !isSelect {
		t.Fatalf("CREATE VIEW core = %#v", stmt)
	}
}

func TestParseSharedCreateSequenceCore(t *testing.T) {
	stmt := mustParseOne(t, "CREATE SEQUENCE alignment_seq START WITH 1 INCREMENT BY 2").(*ast.CreateSequenceStmt)
	if stmt.Name == nil || len(stmt.Name.TableRefCore().Name.Parts) != 1 ||
		stmt.Name.TableRefCore().Name.Parts[0].Name != "alignment_seq" || stmt.Options == nil || len(stmt.Options.Items) != 2 {
		t.Fatalf("CREATE SEQUENCE core = %#v", stmt)
	}
}

func TestParseSharedCreateFunctionCore(t *testing.T) {
	stmt := mustParseOne(t, "CREATE FUNCTION alignment_fn() RETURNS INT RETURN 1").(*ast.CreateFunctionStmt)
	if stmt.Name == nil || len(stmt.Name.TableRefCore().Name.Parts) != 1 ||
		stmt.Name.TableRefCore().Name.Parts[0].Name != "alignment_fn" || stmt.Returns == nil || stmt.Body == nil {
		t.Fatalf("CREATE FUNCTION core = %#v", stmt)
	}
}
