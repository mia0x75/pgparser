package parser_test

import (
	"fmt"
	"os"
	"testing"
)

func TestRestoreCorpusRegressions(t *testing.T) {
	// Keep targets fixed so pruning known failures cannot shrink regression coverage.
	targets := map[string][]int{
		"testdata/create_operator.sql": {26, 27, 28},
		"testdata/foreign_data.sql":    {188, 389, 436, 757},
		"testdata/insert_conflict.sql": {240, 243, 246},
		"testdata/int4.sql":            {99, 101},
		"testdata/limit.sql":           {201},
		"testdata/partition_join.sql":  {104, 418},
		"testdata/plpgsql.sql":         {1542, 1543, 1558, 1559},
		"testdata/privileges.sql":      {241},
		"testdata/rangefuncs.sql": {
			25, 26, 34, 35, 40, 44, 51, 148, 149, 153, 162, 163, 167,
			197, 202, 208, 253, 292, 294, 296, 299, 675, 676, 679, 827,
		},
		"testdata/stats_ext.sql":   {1647, 1662, 1675, 1688, 1700, 1713},
		"testdata/tablesample.sql": {7},
		"testdata/tablespace.sql":  {5, 14, 15},
		"testdata/test_setup.sql":  {28},
	}
	found := 0
	for file, lines := range targets {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		want := make(map[int]testdataStatement)
		for _, stmt := range splitTestdataSQL(string(data)) {
			want[stmt.line] = stmt
		}
		for _, line := range lines {
			stmt, ok := want[line]
			if !ok {
				t.Errorf("%s:%d: target line not found in fixture", file, line)
				continue
			}
			found++
			t.Run(fmt.Sprintf("%s:%d", file, line), func(t *testing.T) {
				parseErr, restoreErr := parseTestdataStatement(stmt.sql)
				if parseErr != nil {
					t.Errorf("parse: %v", parseErr)
				}
				if restoreErr != nil {
					t.Errorf("restore: %v", restoreErr)
				}
			})
		}
	}
	if found != 56 {
		t.Fatalf("expected all 56 targets found in fixtures, got %d", found)
	}
}
