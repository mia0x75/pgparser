package parser_test

import (
	"testing"

	"github.com/mia0x75/pgparser/ast"
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
			if !n.Span().Valid() {
				t.Fatalf("%q: statement span missing", sql)
			}
			if n.Span().End <= n.Span().Start {
				t.Fatalf("%q: statement span = %v", sql, n.Span())
			}
		})
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
