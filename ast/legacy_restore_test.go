package ast

import (
	"errors"
	"strings"
	"testing"

	"github.com/mia0x75/pgparser/format"
)

func restoreLegacyNode(t *testing.T, restore func(*format.RestoreCtx) error) (string, error) {
	t.Helper()
	var out strings.Builder
	err := restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out))
	return out.String(), err
}

func TestLegacyRestoreCasesAreFaithfulOrExplicitlyUnsupported(t *testing.T) {
	tests := []struct {
		name  string
		call  func(*format.RestoreCtx) error
		want  string
		isErr error
	}{
		{
			name: "TLS subject",
			call: func(ctx *format.RestoreCtx) error {
				return (&TslOption{Type: Subject, Value: "MOCK_VALUE"}).Restore(ctx)
			},
			want: "SUBJECT 'MOCK_VALUE'",
		},
		{
			name: "password expiration interval",
			call: func(ctx *format.RestoreCtx) error {
				return (&PasswordOrLockOption{Type: PasswordExpireInterval, Count: 14}).Restore(ctx)
			},
			want: "PASSWORD EXPIRE INTERVAL 14 DAY",
		},
		{
			name: "zero password expiration interval",
			call: func(ctx *format.RestoreCtx) error {
				return (&PasswordOrLockOption{Type: PasswordExpireInterval}).Restore(ctx)
			},
			isErr: errors.New("password expiration interval must be positive"),
		},
		{
			name: "unsupported stats option",
			call: func(ctx *format.RestoreCtx) error {
				return (&TableOption{Tp: TableOptionStatsPersistent}).Restore(ctx)
			},
			isErr: ErrUnsupportedRestore,
		},
		{
			name: "unsupported pack keys option",
			call: func(ctx *format.RestoreCtx) error {
				return (&TableOption{Tp: TableOptionPackKeys}).Restore(ctx)
			},
			isErr: ErrUnsupportedRestore,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := restoreLegacyNode(t, tt.call)
			if tt.isErr != nil {
				if err == nil || (tt.isErr == ErrUnsupportedRestore && !errors.Is(err, tt.isErr)) || (tt.isErr != ErrUnsupportedRestore && err.Error() != tt.isErr.Error()) {
					t.Fatalf("Restore() error = %v, want %v", err, tt.isErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Restore(): %v", err)
			}
			if got != tt.want {
				t.Fatalf("Restore() = %q, want %q", got, tt.want)
			}
		})
	}
}
