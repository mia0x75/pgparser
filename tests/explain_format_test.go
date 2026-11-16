package parser_test

import (
	"testing"

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

func TestExplainLastFormatWins(t *testing.T) {
	for _, tc := range []struct{ name, sql, want string }{
		{"yaml_last", `EXPLAIN (FORMAT JSON, FORMAT YAML) SELECT 1`, `yaml`},
		{"json_last", `EXPLAIN (FORMAT YAML, FORMAT JSON) SELECT 1`, `json`},
		{"case_insensitive", `EXPLAIN (Format yaml, format JSON) SELECT 1`, `json`},
		{"same_format", `EXPLAIN (FORMAT JSON, FORMAT JSON) SELECT 1`, `json`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stmt := mustParseOne(t, tc.sql).(*ast.ExplainStmt)
			if stmt.Format != tc.want {
				t.Fatalf("format = %q, want %q", stmt.Format, tc.want)
			}
			restoreString(t, stmt)
		})
	}
}
