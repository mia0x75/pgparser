package parser_test

import "testing"

func TestRestoreServerOptionNames(t *testing.T) {
	for _, tc := range []struct{ name, sql, want string }{
		{"plain_name", `CREATE SERVER svr FOREIGN DATA WRAPPER mock_fdw OPTIONS (dbname 'MOCK_VALUE')`,
			`CREATE SERVER "svr" FOREIGN DATA WRAPPER "mock_fdw" OPTIONS("dbname" 'MOCK_VALUE')`},
		{"space_in_name", `CREATE SERVER svr FOREIGN DATA WRAPPER mock_fdw OPTIONS ("option name" 'MOCK_VALUE')`,
			`CREATE SERVER "svr" FOREIGN DATA WRAPPER "mock_fdw" OPTIONS("option name" 'MOCK_VALUE')`},
		{"mixed_case", `CREATE SERVER svr FOREIGN DATA WRAPPER mock_fdw OPTIONS ("OptionName" 'MOCK_VALUE')`,
			`CREATE SERVER "svr" FOREIGN DATA WRAPPER "mock_fdw" OPTIONS("OptionName" 'MOCK_VALUE')`},
		{"embedded_quote", `CREATE SERVER svr FOREIGN DATA WRAPPER mock_fdw OPTIONS ("op""x" 'MOCK_VALUE')`,
			`CREATE SERVER "svr" FOREIGN DATA WRAPPER "mock_fdw" OPTIONS("op""x" 'MOCK_VALUE')`},
		{"alter_option", `ALTER SERVER svr OPTIONS ("option name" 'MOCK_VALUE')`,
			`ALTER SERVER "svr" OPTIONS("option name" 'MOCK_VALUE')`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := restoreString(t, mustParseOne(t, tc.sql)); got != tc.want {
				t.Errorf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}
