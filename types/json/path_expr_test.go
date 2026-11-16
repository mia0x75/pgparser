package json

import (
	"testing"
)

func TestContainsAnyAsterisk(t *testing.T) {
	var tests = []struct {
		exprString        string
		containsAsterisks bool
	}{
		{"$.a[0]", false},
		{"$.a[*]", true},
		{"$.*[0]", true},
		{"$**.a[0]", true},
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

func TestPathRejectsMalformedSuffixesAndUnicode(t *testing.T) {
	for _, path := range []string{
		`$.a!`, `$.a[`, `$.a]junk`, `$.a[b]`, `$.*[b]`,
		`$."\u123"`, `$."\u"`, `$."\u12xz"`, `$."\uD800"`, `$."\uDC00"`,
		`$."\uD800\u0041"`, `$."\uD800\uDC0"`,
	} {
		t.Run(path, func(t *testing.T) {
			if _, err := ParseJSONPathExpr(path); err == nil {
				t.Fatalf("accepted malformed path %q", path)
			}
		})
	}
	for _, tc := range []struct{ path, key string }{
		{`$."\uD83D\uDE00"`, "😀"},
		{`$."a\uD83D\uDE00b\u0041"`, "a😀bA"},
		{`$."\uFFFF"`, "\uFFFF"},
		{`$."\q"`, "q"},
		{`$.a  `, "a"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			pe, err := ParseJSONPathExpr(tc.path)
			if err != nil || len(pe.legs) != 1 || pe.legs[0].dotKey != tc.key {
				t.Fatalf("decoded path = %#v, %v", pe, err)
			}
		})
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
		} else if err == nil {
			t.Fatal("got nil, want error")
		}
	}
}

func FuzzJSONPathExpression(f *testing.F) {
	for _, path := range []string{`$.a!`, `$."\u123"`, `$."\uD800"`, `$."\uD83D\uDE00"`, `$."*"`, `$.*`} {
		f.Add(path)
	}
	f.Fuzz(func(t *testing.T, path string) {
		pe, err := ParseJSONPathExpr(path)
		if err != nil {
			return
		}
		if len(pe.legs) > 0 {
			_, pe = pe.popOneLeg()
			var wildcard bool
			for _, leg := range pe.legs {
				wildcard = wildcard || leg.typ == pathLegKeyAsterisk || leg.typ == pathLegDoubleAsterisk ||
					(leg.typ == pathLegIndex && leg.arrayIndex == arrayIndexAsterisk)
			}
			if pe.ContainsAnyAsterisk() != wildcard {
				t.Fatal("wildcard flag does not match remaining path")
			}
		}
	})
}
