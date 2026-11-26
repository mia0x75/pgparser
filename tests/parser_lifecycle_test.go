package parser_test

import (
	"strings"
	"testing"

	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/pgparser/ast"
	parser "github.com/mia0x75/pgparser/parser"
)

func TestStatementSpansArePopulated(t *testing.T) {
	for _, sql := range []string{
		`SELECT a FROM t WHERE a = 1`,
		`INSERT INTO t (a) VALUES (1)`,
		`UPDATE t SET a = 1`,
		`DELETE FROM t WHERE a = 1`,
		`CREATE TABLE t (a int)`,
		`CREATE INDEX ix ON t (a)`,
		`DROP TABLE t`,
		`SET work_mem TO 1`,
		`BEGIN`,
	} {
		t.Run(sql, func(t *testing.T) {
			n := mustParseOne(t, sql)
			if want := (loc.Span{Start: 0, End: loc.ByteOffset(len(sql))}); n.Span() != want {
				t.Fatalf("statement span = %v, want %v", n.Span(), want)
			}
		})
	}
}

func TestStatementSpansInBatch(t *testing.T) {
	const first = "INSERT INTO t (a) VALUES (1)"
	const nested = "INSERT INTO t (a) VALUES (2)"
	const second = "PREPARE p AS " + nested
	const sql = "; /* lead */ " + first + ";; -- gap\n " + second + " /* trailing */ ;"
	stmts, err := parser.New().Parse(sql)
	if err != nil || len(stmts) != 2 {
		t.Fatalf("Parse returned %d statements, error %v", len(stmts), err)
	}
	for _, tc := range []struct {
		node ast.Node
		text string
	}{
		{stmts[0], first},
		{stmts[1], second},
		{stmts[1].(*ast.PrepareStmt).Stmt, nested},
	} {
		start := strings.Index(sql, tc.text)
		want := loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(tc.text))}
		if got := tc.node.Span(); got != want {
			t.Errorf("%s span = %v, want %v", tc.text, got, want)
		}
	}
}

func TestNodePresentRejectsTypedNil(t *testing.T) {
	var stmt *ast.SelectStmt
	if ast.NodePresent(stmt) {
		t.Fatal("typed nil SelectStmt reported present")
	}
	if !ast.NodePresent(&ast.SelectStmt{}) {
		t.Fatal("non-nil SelectStmt reported absent")
	}
}
