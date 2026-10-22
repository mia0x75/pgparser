package parser_test

import "testing"

// Ported from parser_jsonb_compound_tokens_test.go: compound JSON operators
// must parse as operators and not be consumed as placeholders.
func TestParseSQLJSONBCompoundOperators(t *testing.T) {
	queries := []string{
		"SELECT * FROM t WHERE data ?| array['a','b'];",
		"SELECT * FROM t WHERE data ?& array['a','b'];",
		"SELECT * FROM t WHERE meta ? 'key';",
	}
	for _, sql := range queries {
		stmt := mustParseOne(t, sql)
		if countParams(stmt) != 0 {
			t.Fatalf("ParseOneStmt(%q) unexpectedly recorded placeholders", sql)
		}
	}
	if got := countParams(mustParseOne(t, "SELECT * FROM t WHERE id = $1;")); got != 1 {
		t.Fatalf("bare ? params = %d, want 1", got)
	}
}
