package postgres

import "testing"

func TestCompatibilityErrorTemplates(t *testing.T) {
	tests := []struct {
		name string
		err  *SQLError
		want string
	}{
		{
			name: "stack usage",
			err:  NewErr(ErrStackOverrun, 64, 128),
			want: "Thread stack overrun: Used 64 of a 128 stack",
		},
		{
			name: "stack shortage",
			err:  NewErr(ErrStackOverrunNeedMore, 64, 128, 256),
			want: "Thread stack overrun: 64 bytes used of 128, 256 needed",
		},
		{
			name: "feature support",
			err:  NewErr(ErrFeatureDisabled, "MOCK_FEATURE", "MOCK_SUPPORT"),
			want: "The 'MOCK_FEATURE' feature is disabled; it requires support for 'MOCK_SUPPORT'",
		},
		{
			name: "metadata version",
			err: NewErr(
				ErrColCountDoesntMatchPleaseUpdate,
				"MOCK_TABLE",
				2,
				1,
				1,
				2,
			),
			want: "Table 'MOCK_TABLE': expected 2 columns, found 1; created v1, running v2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Message != tt.want {
				t.Fatalf("got message %q, want %q", tt.err.Message, tt.want)
			}
		})
	}
}
