package parser_test

import (
	"strings"
	"testing"

	pgparser "github.com/mia0x75/pgparser"
)

// Ported from parser_ir_error_test.go and parser_test.go: syntax errors expose
// line information and never panic.
func TestParseSQLSyntaxErrors(t *testing.T) {
	cases := []string{
		"SELECT FROM WHERE broken = true;",
		"SELECT * FROM UNION SELECT;",
		"SELECT 1; SELECT FROM; SELECT 2;",
	}
	for _, sql := range cases {
		_, _, err := pgparser.New().Parse(sql)
		if err == nil {
			t.Fatalf("Parse(%q) expected error", sql)
		}
		if !strings.Contains(err.Error(), "line") {
			t.Fatalf("Parse(%q) error = %q, want line information", sql, err)
		}
	}

	if _, err := pgparser.New().ParseOneStmt("SELECT 1; SELECT 2;"); err == nil {
		t.Fatal("ParseOneStmt with multiple statements should fail")
	}
	if _, err := pgparser.New().ParseOneStmt(""); err == nil {
		t.Fatal("ParseOneStmt with empty input should fail")
	}
	stmts, _, err := pgparser.New().Parse("")
	if err != nil || len(stmts) != 0 {
		t.Fatalf("Parse(empty) = %d stmts, err %v", len(stmts), err)
	}
}

// Ported from parser_test.go: multi-statement batches parse in order.
func TestParseSQLBatchStatements(t *testing.T) {
	sql := `SELECT u.id, COUNT(o.id) AS order_count FROM users u LEFT JOIN orders o ON o.user_id = u.id WHERE u.active = true AND o.created_at > $1 GROUP BY u.id HAVING COUNT(o.id) > 1 ORDER BY order_count DESC LIMIT 10;
UPDATE users SET status = 'active' WHERE id = $2 RETURNING id;
DELETE FROM sessions WHERE expires_at < NOW();`
	stmts, _, err := pgparser.New().Parse(sql)
	if err != nil {
		t.Fatalf("Parse batch: %v", err)
	}
	if len(stmts) != 3 {
		t.Fatalf("statements = %d, want 3", len(stmts))
	}
	if got := countParams(stmts[0]) + countParams(stmts[1]); got != 2 {
		t.Fatalf("parameters = %d, want 2", got)
	}
}
