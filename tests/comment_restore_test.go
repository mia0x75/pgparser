package parser_test

import "testing"

func TestRestoreCommentOperatorSignatures(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{
			"prefix",
			`COMMENT ON OPERATOR ###### (NONE, int4) IS 'MOCK_COMMENT'`,
			`COMMENT ON OPERATOR ######(NONE, "int4") IS 'MOCK_COMMENT'`,
		},
		{
			"postfix",
			`COMMENT ON OPERATOR ###### (int4, NONE) IS 'MOCK_COMMENT'`,
			`COMMENT ON OPERATOR ######("int4", NONE) IS 'MOCK_COMMENT'`,
		},
		{
			"binary",
			`COMMENT ON OPERATOR ###### (int4, int8) IS 'MOCK_COMMENT'`,
			`COMMENT ON OPERATOR ######("int4", "int8") IS 'MOCK_COMMENT'`,
		},
		{
			"qualified",
			`COMMENT ON OPERATOR "Operator Schema".=== (integer, text) IS NULL`,
			`COMMENT ON OPERATOR "Operator Schema".===("int4", "text") IS NULL`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := restoreString(t, mustParseOne(t, tc.sql)); got != tc.want {
				t.Fatalf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}
