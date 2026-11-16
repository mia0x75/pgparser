package parser_test

import (
	"strings"
	"testing"

	pgparser "github.com/mia0x75/pgparser"
	"github.com/mia0x75/pgparser/ast"
)

func TestExplainFormatOption(t *testing.T) {
	for _, tc := range []struct {
		name, sql, want string
	}{
		{"plain", `EXPLAIN SELECT 1`, ``},
		{"explicit", `EXPLAIN (FORMAT JSON) SELECT 1`, `json`},
		{"mixed_case_key", `EXPLAIN (Format yaml) SELECT 1`, `yaml`},
		{"options_without_format", `EXPLAIN (ANALYZE, VERBOSE) SELECT 1`, `text`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stmt, ok := mustParseOne(t, tc.sql).(*ast.ExplainStmt)
			if !ok {
				t.Fatal("expected *ast.ExplainStmt")
			}
			if stmt.Format != tc.want {
				t.Fatalf("format = %q, want %q", stmt.Format, tc.want)
			}
		})
	}
}

// PostgreSQL reports "conflicting or redundant options" for a repeated FORMAT
// rather than silently choosing one of the two values.
func TestExplainRejectsRepeatedFormat(t *testing.T) {
	_, err := pgparser.New().ParseOneStmt(`EXPLAIN (FORMAT JSON, FORMAT YAML) SELECT 1`)
	if err == nil {
		t.Fatal("expected an error for a repeated FORMAT option")
	}
	if !strings.Contains(err.Error(), "conflicting or redundant options") {
		t.Fatalf("error = %v, want the conflicting-options message", err)
	}
}
