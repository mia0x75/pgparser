package ast

import (
	"errors"
	"strings"
	"testing"

	"github.com/mia0x75/pgparser/auth"
	"github.com/mia0x75/pgparser/format"
)

func TestGroupingSetRestoreChildError(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind GroupingSetType
	}{
		{"rollup", GroupingSetRollup},
		{"cube", GroupingSetCube},
		{"sets", GroupingSetSets},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := &GroupingSet{Type: tc.kind, Sets: [][]ExprNode{{&JsonArrayConstructor{}}}}
			var out strings.Builder
			if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil {
				t.Fatalf("ignored unsupported child: %q", out.String())
			}
		})
	}
}

func TestUnsupportedStatementRestoreBoundary(t *testing.T) {
	for _, node := range []Node{&LockTableStmt{}, &UnlockTableStmt{}, &PurgeStmt{}} {
		var out strings.Builder
		err := node.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out))
		if !errors.Is(err, ErrUnsupportedRestore) {
			t.Errorf("%T Restore() error = %v, want ErrUnsupportedRestore", node, err)
		}
	}
}

func TestResourceOptionValidation(t *testing.T) {
	for _, tc := range []struct {
		name string
		opt  *ResourceOption
		want string
	}{
		{name: "unknown option", opt: &ResourceOption{Type: -1}},
		{name: "negative count", opt: &ResourceOption{Type: MaxQueriesPerHour, Count: -1}},
		{name: "valid zero", opt: &ResourceOption{Type: MaxQueriesPerHour}, want: "MAX_QUERIES_PER_HOUR 0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			err := tc.opt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out))
			if tc.want == "" {
				if err == nil {
					t.Fatalf("Restore() = %q, want error", out.String())
				}
				return
			}
			if err != nil || out.String() != tc.want {
				t.Fatalf("Restore() = %q, %v; want %q", out.String(), err, tc.want)
			}
		})
	}
}

func TestSetRoleRestore(t *testing.T) {
	for _, tc := range []struct {
		name string
		node Node
		want string
	}{
		{
			name: "set role",
			node: &SetRoleStmt{SetRoleOpt: SetRoleRegular, RoleList: []*auth.RoleIdentity{{Username: "analyst"}}},
			want: "SET ROLE \"analyst\"",
		},
		{
			name: "set default role",
			node: &SetDefaultRoleStmt{
				SetRoleOpt: SetRoleRegular,
				RoleList:   []*auth.RoleIdentity{{Username: "analyst"}},
				UserList:   []*auth.UserIdentity{{Username: "operator"}},
			},
			want: "SET DEFAULT ROLE \"analyst\" TO \"operator\"",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			if err := tc.node.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil {
				t.Fatal(err)
			}
			if out.String() != tc.want {
				t.Fatalf("Restore() = %q, want %q", out.String(), tc.want)
			}
		})
	}
}
