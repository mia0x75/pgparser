package json

import (
	"testing"
)

func TestContainsAnyAsterisk(t *testing.T) {
	var tests = []struct {
		exprString        string
		containsAsterisks bool
	}{
		{"$.a[b]", false},
		{"$.a[*]", true},
		{"$.*[b]", true},
		{"$**.a[b]", true},
	}
	for _, tt := range tests {
		pe, err := ParseJSONPathExpr(tt.exprString)
		if err != nil {
			t.Fatal(err)
		}
		if pe.flags.containsAnyAsterisk() != tt.containsAsterisks {
			t.Fatalf("got %#v, want %#v", pe.flags.containsAnyAsterisk(), tt.containsAsterisks)
		}
	}
}

func TestValidatePathExpr(t *testing.T) {
	var tests = []struct {
		exprString string
		success    bool
		legs       int
	}{
		{`   $  `, true, 0},
		{"   $ .   key1  [  3  ]\t[*].*.key3", true, 5},
		{"   $ .   key1  [  3  ]**[*].*.key3", true, 6},
		{`$."key1 string"[  3  ][*].*.key3`, true, 5},
		{`$."hello \"escaped quotes\" world\\n"[3][*].*.key3`, true, 5},

		{`$.\"escaped quotes\"[3][*].*.key3`, false, 0},
		{`$.hello \"escaped quotes\" world[3][*].*.key3`, false, 0},
		{`$NoValidLegsHere`, false, 0},
		{`$        No Valid Legs Here .a.b.c`, false, 0},
	}

	for _, tt := range tests {
		pe, err := ParseJSONPathExpr(tt.exprString)
		if tt.success {
			if err != nil {
				t.Fatal(err)
			}
			if len(pe.legs) != tt.legs {
				t.Fatalf("got %#v, want %#v", len(pe.legs), tt.legs)
			}
		} else {
			if err == nil {
				t.Fatal("got nil, want error")
			}
		}
	}
}
