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
	if _, ok := stmts[0].(*ast.InsertStmt); !ok {
		t.Fatalf("first statement = %T, want *ast.InsertStmt", stmts[0])
	}
	prepared, ok := stmts[1].(*ast.PrepareStmt)
	if !ok {
		t.Fatalf("second statement = %T, want *ast.PrepareStmt", stmts[1])
	}
	for i, want := range []string{
		"; /* lead */ " + first + ";",
		"; -- gap\n " + second + " /* trailing */ ;",
	} {
		if got := stmts[i].Text(); got != want {
			t.Errorf("statement %d text = %q, want %q", i, got, want)
		}
	}
	for _, tc := range []struct {
		node ast.Node
		text string
	}{
		{stmts[0], first},
		{stmts[1], second},
		{prepared.Stmt, nested},
	} {
		start := strings.Index(sql, tc.text)
		want := loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(tc.text))}
		if got := tc.node.Span(); got != want {
			t.Errorf("%s span = %v, want %v", tc.text, got, want)
		}
	}
}

func TestStatementTextBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name string
		sql  string
		want []string
	}{
		{
			name: "one newline at each end",
			sql:  "\n\nSELECT 1\n\n",
			want: []string{"\nSELECT 1\n"},
		},
		{
			name: "prefetched semicolon",
			sql:  "LISTEN format /* first */ ; SELECT 2 /* trailing */",
			want: []string{"LISTEN format /* first */ ;", " SELECT 2 /* trailing */"},
		},
		{
			name: "prefetched end of input",
			sql:  "LISTEN format /* trailing */\n",
			want: []string{"LISTEN format /* trailing */"},
		},
		{
			name: "reduction before semicolon",
			sql:  "RESET ALL; SELECT 2",
			want: []string{"RESET ALL", "; SELECT 2"},
		},
		{
			name: "quoted semicolons",
			sql:  "SELECT ';'; SELECT $$;$$; SELECT 3 -- trailing\n",
			want: []string{"SELECT ';';", " SELECT $$;$$;", " SELECT 3 -- trailing"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stmts, err := parser.New().Parse(tc.sql)
			if err != nil || len(stmts) != len(tc.want) {
				t.Fatalf("Parse returned %d statements, error %v, want %d statements", len(stmts), err, len(tc.want))
			}
			for i, want := range tc.want {
				if got := stmts[i].Text(); got != want {
					t.Errorf("statement %d text = %q, want %q", i, got, want)
				}
			}
		})
	}
}

func TestParseEmptyStatements(t *testing.T) {
	for _, tc := range []struct {
		name string
		sql  string
	}{
		{name: "empty"},
		{name: "comments", sql: "/* lead */ -- trailing\n"},
		{name: "semicolons", sql: ";;;"},
		{name: "comments and semicolons", sql: "; /* gap */ ; -- trailing\n;"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			shared := parser.New()
			if _, err := shared.Parse("SELECT 1"); err != nil {
				t.Fatal(err)
			}
			stmts, err := shared.Parse(tc.sql)
			if err != nil || stmts != nil {
				t.Fatalf("Parse(%q) = %v, %v, want nil, nil", tc.sql, stmts, err)
			}
		})
	}
}

func TestParseRecoversAfterError(t *testing.T) {
	for _, tc := range []struct {
		name string
		sql  string
	}{
		{name: "syntax", sql: "SELECT 1; SELECT FROM;"},
		{name: "lexer", sql: "SELECT 1; SELECT 'unterminated"},
		{name: "prefetched token", sql: "CLOSE format /* gap */ SELECT 2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			shared := parser.New()
			if _, err := shared.Parse("SELECT 1"); err != nil {
				t.Fatal(err)
			}
			if stmts, err := shared.Parse(tc.sql); err == nil || stmts != nil {
				t.Fatalf("failed Parse = %v, %v, want nil statements and an error", stmts, err)
			}
			const sql = "DELETE FROM t"
			stmts, err := shared.Parse(sql)
			if err != nil || len(stmts) != 1 {
				t.Fatalf("recovered Parse returned %d statements, error %v", len(stmts), err)
			}
			if _, ok := stmts[0].(*ast.DeleteStmt); !ok {
				t.Fatalf("recovered statement = %T, want *ast.DeleteStmt", stmts[0])
			}
			if got := stmts[0].Text(); got != sql {
				t.Fatalf("recovered statement text = %q, want %q", got, sql)
			}
			if got, want := stmts[0].Span(), (loc.Span{Start: 0, End: loc.ByteOffset(len(sql))}); got != want {
				t.Fatalf("recovered statement span = %v, want %v", got, want)
			}
		})
	}
}

func TestParseBatchPreservesParameterNumbers(t *testing.T) {
	shared := parser.New()
	if stmts, err := shared.Parse("SELECT $1, $2; SELECT FROM;"); err == nil || stmts != nil {
		t.Fatalf("failed Parse = %v, %v, want nil statements and an error", stmts, err)
	}
	if stmts, err := shared.Parse("SELECT $0"); err == nil || stmts != nil {
		t.Fatalf("zero parameter Parse = %v, %v, want nil statements and an error", stmts, err)
	}
	const sql = "SELECT $2, $1, $2;; SELECT $1, $2, $1;"
	stmts, err := shared.Parse(sql)
	if err != nil || len(stmts) != 2 {
		t.Fatalf("Parse returned %d statements, error %v", len(stmts), err)
	}
	for i, want := range [][]int{{1, 0, 1}, {0, 1, 0}} {
		stmt, ok := stmts[i].(*ast.SelectStmt)
		if !ok {
			t.Fatalf("statement %d = %T, want *ast.SelectStmt", i, stmts[i])
		}
		if len(stmt.Fields.Items) != len(want) {
			t.Fatalf("statement %d has %d fields, want %d", i, len(stmt.Fields.Items), len(want))
		}
		for j, index := range want {
			param, ok := stmt.Fields.Items[j].Expr.(*ast.ParamMarkerExpr)
			if !ok {
				t.Fatalf("statement %d field %d = %T, want *ast.ParamMarkerExpr", i, j, stmt.Fields.Items[j].Expr)
			}
			if param.Index != index {
				t.Errorf("statement %d parameter %d index = %d, want %d", i, j, param.Index, index)
			}
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
