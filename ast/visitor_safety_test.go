package ast

import (
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"
)

// misbehavingVisitor violates the Visitor contract: Enter may return a nil
// replacement, a replacement of the wrong concrete type, or a typed-nil
// pointer boxed in the interface, and children.Accept may stop traversal.
type misbehavingVisitor struct {
	skip      bool
	stop      bool
	enterNil  bool
	enterType pkast.Node
}

func (v *misbehavingVisitor) Enter(n Node) (Node, bool) {
	switch {
	case v.enterNil:
		return nil, v.skip
	case v.enterType != nil:
		return v.enterType, v.skip
	default:
		return n, v.skip
	}
}

func (v *misbehavingVisitor) Leave(n Node) (Node, bool) {
	if v.stop {
		return n, false
	}
	return n, true
}

func TestVisitorSafetyGuards(t *testing.T) {
	cases := []struct {
		name string
		node Node
		vis  *misbehavingVisitor
		want bool
	}{
		{"nil receiver", (*List)(nil), &misbehavingVisitor{}, false},
		{"typed-nil List entry", &List{Items: []Node{(*RoleSpec)(nil)}}, &misbehavingVisitor{}, false},
		{"nil List entry skipped", &List{Items: []Node{nil, &RoleSpec{}}}, &misbehavingVisitor{}, true},
		{"Enter returns nil", &DefElem{}, &misbehavingVisitor{enterNil: true}, false},
		{"Enter wrong type", &DefElem{}, &misbehavingVisitor{enterType: &List{}}, false},
		{"Enter typed-nil", &DefElem{}, &misbehavingVisitor{enterType: (*RoleSpec)(nil)}, false},
		{"child stops", &DefElem{Arg: &String{}}, &misbehavingVisitor{stop: true}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var ok bool
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panicked: %v", r)
				}
			}()
			_, ok = tc.node.Accept(tc.vis)
			if ok != tc.want {
				t.Fatalf("Accept = %v, want %v", ok, tc.want)
			}
		})
	}
}
