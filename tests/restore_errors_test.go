package parser_test

import (
	"errors"
	"testing"

	"github.com/mia0x75/pgparser/format"
)

func TestRestorePropagatesWriterErrors(t *testing.T) {
	writeErr := errors.New("mock write failure")
	for _, sql := range []string{
		`SELECT a FROM t WHERE a = 1`,
		`SELECT 1 UNION SELECT 2 LIMIT 3`,
		`INSERT INTO t (a) VALUES (1)`,
		`UPDATE t SET a = 1`,
		`DELETE FROM t WHERE a = 1`,
		`CREATE TABLE t (a int)`,
		`CREATE INDEX ix ON t (a)`,
		`CREATE DATABASE db`,
		`CREATE SCHEMA s`,
		`WITH c AS (SELECT 1) SELECT * FROM c`,
	} {
		t.Run(sql, func(t *testing.T) {
			stmt := mustParseOne(t, sql)
			ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &failingWriter{err: writeErr})
			if err := stmt.Restore(ctx); !errors.Is(err, writeErr) {
				t.Fatalf("Restore error = %v, want %v", err, writeErr)
			}
		})
	}
}
