package ast

import (
	"strings"
	"testing"

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
