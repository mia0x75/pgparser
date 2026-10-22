package parser_test

import "testing"

// Ported from preprocess_test.go: URL and dollar-quoted literals are preserved
// verbatim while line comments are ignored.
func TestParseSQLStringLiteralsAndComments(t *testing.T) {
	queries := []string{
		"  SELECT id FROM users   \n;",
		"SELECT 'http://example.com' AS url // comment\n;",
		"SELECT $$http://example.com//keep$$ AS url // comment\n;",
		"SELECT $$line1\n//not comment\nline3$$ AS body // trailing\n;",
	}
	for _, sql := range queries {
		st := mustParseSelect(t, sql)
		if st.Fields == nil || len(st.Fields.Fields) == 0 {
			t.Fatalf("ParseOneStmt(%q) produced no fields", sql)
		}
	}
}
