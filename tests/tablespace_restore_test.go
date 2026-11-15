package parser_test

import "testing"

func TestRestoreTablespaceLocation(t *testing.T) {
	for _, tc := range []struct{ name, sql, want string }{
		{"empty_location", `CREATE TABLESPACE ts LOCATION ''`,
			`CREATE TABLESPACE "ts" LOCATION ''`},
		{"empty_location_with_options", `CREATE TABLESPACE ts LOCATION '' WITH (seq_page_cost = 2)`,
			`CREATE TABLESPACE "ts" LOCATION '' WITH ("seq_page_cost" = 2)`},
		{"location", `CREATE TABLESPACE ts LOCATION 'mock_dir'`,
			`CREATE TABLESPACE "ts" LOCATION 'mock_dir'`},
		{"owner", `CREATE TABLESPACE ts OWNER app LOCATION 'mock_dir'`,
			`CREATE TABLESPACE "ts" OWNER "app" LOCATION 'mock_dir'`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := restoreString(t, mustParseOne(t, tc.sql)); got != tc.want {
				t.Errorf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}
