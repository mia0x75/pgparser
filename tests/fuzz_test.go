package parser_test

import (
	"errors"
	"strings"
	"testing"

	pgparser "github.com/mia0x75/pgparser"
	"github.com/mia0x75/pgparser/ast"
	_ "github.com/mia0x75/pgparser/driver"
	"github.com/mia0x75/pgparser/format"
)

// FuzzParseRestore parses arbitrary bytes, restores them back to SQL and
// re-parses the result. A panic anywhere means the fuzzer records a failure.
// This is the input-space check the corpus gate cannot give: it explores
// malformed and truncated SQL that the hand-written fixtures miss.
func FuzzParseRestore(f *testing.F) {
	// Seeds from the classes that used to panic or lose data: indirection,
	// quoted function names, reserved-word names, named windows, IS DISTINCT
	// FROM, COPY, MERGE, TRUNCATE, DO blocks and array subscripts.
	seeds := []string{
		`SELECT (t.a).b FROM t`,
		`SELECT $1[1]`,
		`SELECT a[1:2] FROM t`,
		`SELECT "MyFunc"(a) FROM t`,
		`SELECT "0"()`,
		`SELECT "cast"(a) FROM t`,
		`SELECT "exists"(a) FROM t`,
		`SELECT rank() OVER w FROM t WINDOW w AS (PARTITION BY a)`,
		`SELECT 1 WHERE a IS DISTINCT FROM 1`,
		`COPY t FROM STDIN WITH (HEADER true)`,
		`MERGE INTO t USING s ON t.id = s.id WHEN NOT MATCHED THEN INSERT DEFAULT VALUES`,
		`TRUNCATE ONLY t`,
		`DO $$ BEGIN NULL; END $$ LANGUAGE plpgsql`,
		`DO LANGUAGE a`,
		"DO $$\xdf$$",
		`EXPLAIN SELECT 1`,
		`ALTER SERVER s VERSION '1.0'`,
		`SELECT lower(a) FROM t`,
		`SELECT count(*) FROM t`,
		`SELECT substring(b, 2, 4) FROM t`,
		"SELECT 1",
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		stmts, err := pgparser.New().Parse(string(data))
		if err != nil {
			return
		}
		for _, stmt := range stmts {
			var sb strings.Builder
			if err := stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err != nil {
				// ErrUnsupportedRestore is an allowed outcome; anything else
				// (a panic recovered here, a real error on a supported
				// restore) is a failure.
				if !errors.Is(err, ast.ErrUnsupportedRestore) {
					t.Fatalf("restore error for %q: %v", data, err)
				}
				continue
			}
			restored := sb.String()
			next, err := pgparser.New().Parse(restored)
			if err != nil {
				t.Fatalf("restored SQL is not parseable: %q: %v", restored, err)
			}
			if len(next) != 1 {
				t.Fatalf("restore changed statement count: %q", restored)
			}
			var again strings.Builder
			if err := next[0].Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &again)); err != nil {
				t.Fatalf("second restore failed: %q: %v", restored, err)
			}
			if again.String() != restored {
				t.Fatalf("unstable restore: %q -> %q", restored, again.String())
			}
		}
	})
}
