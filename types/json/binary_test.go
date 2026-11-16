package json

import (
	"bytes"
	stdjson "encoding/json"
	"fmt"
	"math"
	"math/big"
	"strings"
	"testing"
)

func TestBinaryJSONMarshalUnmarshal(t *testing.T) {
	strs := []string{
		`{"a": [1, "2", {"aa": "bb"}, 4, null], "b": true, "c": null}`,
		`{"aaaaaaaaaaa": [1, "2", {"aa": "bb"}, 4.1], "bbbbbbbbbb": true, "ccccccccc": "d"}`,
		`[{"a": 1, "b": true}, 3, 3.5, "hello, world", null, true]`,
	}
	for _, str := range strs {
		parsedBJ := mustParseBinaryFromString(t, str)
		if parsedBJ.String() != str {
			t.Fatalf("got %#v, want %#v", parsedBJ.String(), str)
		}
	}
}

func TestBinaryJSONExtract(t *testing.T) {
	bj1 := mustParseBinaryFromString(t, `{"\"hello\"": "world", "a": [1, "2", {"aa": "bb"}, 4.0, {"aa": "cc"}], "b": true, "c": ["d"]}`)
	bj2 := mustParseBinaryFromString(t, `[{"a": 1, "b": true}, 3, 3.5, "hello, world", null, true]`)

	var tests = []struct {
		bj              BinaryJSON
		pathExprStrings []string
		expected        BinaryJSON
		found           bool
		err             error
	}{
		// test extract with only one path expression.
		{bj1, []string{"$.a"}, mustParseBinaryFromString(t, `[1, "2", {"aa": "bb"}, 4.0, {"aa": "cc"}]`), true, nil},
		{bj2, []string{"$.a"}, mustParseBinaryFromString(t, "null"), false, nil},
		{bj1, []string{"$[0]"}, bj1, true, nil}, // in Extract, autowraped bj1 as an array.
		{bj2, []string{"$[0]"}, mustParseBinaryFromString(t, `{"a": 1, "b": true}`), true, nil},
		{bj1, []string{"$.a[2].aa"}, mustParseBinaryFromString(t, `"bb"`), true, nil},
		{bj1, []string{"$.a[*].aa"}, mustParseBinaryFromString(t, `["bb", "cc"]`), true, nil},
		{bj1, []string{"$.*[0]"}, mustParseBinaryFromString(t, `["world", 1, true, "d"]`), true, nil},
		{bj1, []string{`$.a[*]."aa"`}, mustParseBinaryFromString(t, `["bb", "cc"]`), true, nil},
		{bj1, []string{`$."\"hello\""`}, mustParseBinaryFromString(t, `"world"`), true, nil},
		{bj1, []string{`$**[1]`}, mustParseBinaryFromString(t, `"2"`), true, nil},

		// test extract with multi path expressions.
		{bj1, []string{"$.a", "$[5]"}, mustParseBinaryFromString(t, `[[1, "2", {"aa": "bb"}, 4.0, {"aa": "cc"}]]`), true, nil},
		{bj2, []string{"$.a", "$[0]"}, mustParseBinaryFromString(t, `[{"a": 1, "b": true}]`), true, nil},
	}

	for _, tt := range tests {
		var pathExprList = make([]PathExpression, 0)
		for _, peStr := range tt.pathExprStrings {
			pe, err := ParseJSONPathExpr(peStr)
			if err != nil {
				t.Fatal(err)
			}
			pathExprList = append(pathExprList, pe)
		}

		result, found := tt.bj.Extract(pathExprList)
		if found != tt.found {
			t.Fatalf("got %#v, want %#v", found, tt.found)
		}
		if found {
			if result.String() != tt.expected.String() {
				t.Fatalf("got %#v, want %#v", result.String(), tt.expected.String())
			}
		}
	}
}

func TestBinaryJSONLiteralStarPaths(t *testing.T) {
	base := mustParseBinaryFromString(t, `{"*": {"*": 1, "a": 2}, "a": {"*": 3}}`)
	for _, tc := range []struct {
		path, want string
		wildcard   bool
	}{
		{`$."*"."*"`, `1`, false},
		{`$."*".*`, `[1, 2]`, true},
		{`$.*."*"`, `[1, 3]`, true},
	} {
		t.Run(tc.path, func(t *testing.T) {
			path, err := ParseJSONPathExpr(tc.path)
			if err != nil {
				t.Fatal(err)
			}
			_, child := path.popOneLeg()
			if path.ContainsAnyAsterisk() != tc.wildcard || child.ContainsAnyAsterisk() != (tc.path == `$."*".*`) {
				t.Fatal("wildcard flag changed meaning")
			}
			got, found := base.Extract([]PathExpression{path})
			if !found || got.String() != tc.want {
				t.Fatalf("got %s, found=%v", got, found)
			}
		})
	}
	path, err := ParseJSONPathExpr(`$."*"."*"`)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := base.Modify([]PathExpression{path}, []BinaryJSON{mustParseBinaryFromString(t, "9")}, ModifySet)
	if err != nil || updated.String() != `{"*": {"*": 9, "a": 2}, "a": {"*": 3}}` {
		t.Fatalf("modify = %s, %v", updated, err)
	}
	removed, err := base.Remove([]PathExpression{path})
	if err != nil || removed.String() != `{"*": {"a": 2}, "a": {"*": 3}}` {
		t.Fatalf("remove = %s, %v", removed, err)
	}
	emoji := mustParseBinaryFromString(t, `{"😀": 7}`)
	path, err = ParseJSONPathExpr(`$."\uD83D\uDE00"`)
	if err != nil {
		t.Fatal(err)
	}
	got, found := emoji.Extract([]PathExpression{path})
	if !found || got.String() != "7" {
		t.Fatalf("emoji extraction = %s, %v", got, found)
	}
}

func TestBinaryJSONModifyKeyLength(t *testing.T) {
	base := mustParseBinaryFromString(t, `{}`)
	value := mustParseBinaryFromString(t, `1`)
	for _, size := range []int{math.MaxUint16, math.MaxUint16 + 1} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			key := strings.Repeat("a", size)
			path, err := ParseJSONPathExpr(`$."` + key + `"`)
			if err != nil {
				t.Fatal(err)
			}
			first, err := ParseJSONPathExpr("$.first")
			if err != nil {
				t.Fatal(err)
			}
			got, err := base.Modify([]PathExpression{first, path}, []BinaryJSON{value, value}, ModifySet)
			if size > math.MaxUint16 {
				if err == nil || got.String() != "{}" {
					t.Fatalf("long key modified object: %s, %v", got, err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				selected, found := got.Extract([]PathExpression{path})
				if !found || selected.String() != "1" {
					t.Fatal("maximum length key was lost")
				}
				if _, err := got.MarshalJSON(); err != nil {
					t.Fatal(err)
				}
			}
			if base.String() != "{}" {
				t.Fatal("original object mutated")
			}
		})
	}
}

func TestBinaryJSONType(t *testing.T) {
	var tests = []struct {
		In  string
		Out string
	}{
		{`{"a": "b"}`, "OBJECT"},
		{`["a", "b"]`, "ARRAY"},
		{`3`, "INTEGER"},
		{`3.0`, "DOUBLE"},
		{`null`, "NULL"},
		{`true`, "BOOLEAN"},
	}
	for _, tt := range tests {
		bj := mustParseBinaryFromString(t, tt.In)
		if bj.Type() != tt.Out {
			t.Fatalf("got %#v, want %#v", bj.Type(), tt.Out)
		}
	}
	bj, err := CreateBinary(uint64(1 << 63))
	if err != nil {
		t.Fatalf("CreateBinary failed: %v", err)
	}
	if bj.Type() != "UNSIGNED INTEGER" {
		t.Fatalf("got %#v, want %#v", bj.Type(), "UNSIGNED INTEGER")
	}
}

func TestBinaryJSONIntegerBoundaries(t *testing.T) {
	var okCases = []struct {
		in   string
		want string
	}{
		{"9223372036854775808", "9223372036854775808"}, // maxInt64+1
		{"9223372036854775809", "9223372036854775809"},
		{"18446744073709551615", "18446744073709551615"}, // maxUint64
		{"-9223372036854775808", "-9223372036854775808"}, // minInt64
	}
	for _, tc := range okCases {
		t.Run(tc.in, func(t *testing.T) {
			bj := mustParseBinaryFromString(t, tc.in)
			if got, want := bj.String(), tc.want; got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
			out, err := bj.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON(%s) failed: %v", tc.in, err)
			}
			if got, want := string(out), tc.want; got != want {
				t.Fatalf("MarshalJSON = %q, want %q", got, want)
			}
		})
	}
	for _, bad := range []string{
		"18446744073709551616", // > maxUint64
		"-9223372036854775809", // < minInt64
	} {
		t.Run(bad, func(t *testing.T) {
			if _, err := ParseBinaryFromString(bad); err == nil {
				t.Fatalf("ParseBinaryFromString(%s) accepted an out-of-range integer", bad)
			}
		})
	}
}

func TestBinaryJSONUnquote(t *testing.T) {
	var tests = []struct {
		j        string
		unquoted string
	}{
		{j: `3`, unquoted: "3"},
		{j: `"3"`, unquoted: "3"},
		{j: `"hello, \"escaped quotes\" world"`, unquoted: "hello, \"escaped quotes\" world"},
		{j: "\"\\u4f60\"", unquoted: "你"},
		{j: `true`, unquoted: "true"},
		{j: `null`, unquoted: "null"},
		{j: `{"a": [1, 2]}`, unquoted: `{"a": [1, 2]}`},
		{j: `"\""`, unquoted: `"`},
		{j: `"'"`, unquoted: `'`},
		{j: `"''"`, unquoted: `''`},
		{j: `""`, unquoted: ``},
		{j: `"\\n"`, unquoted: `\n`},
		{j: `"\\u123"`, unquoted: `\u123`},
		{j: `"\\uD800"`, unquoted: `\uD800`},
		{j: `"\"hello\""`, unquoted: `"hello"`},
	}
	for _, tt := range tests {
		bj := mustParseBinaryFromString(t, tt.j)
		unquoted, err := bj.Unquote()
		if err != nil {
			t.Fatal(err)
		}
		if unquoted != tt.unquoted {
			t.Fatalf("got %#v, want %#v", unquoted, tt.unquoted)
		}
	}
}

func TestBinaryJSONModify(t *testing.T) {
	var tests = []struct {
		base     string
		setField string
		setValue string
		mt       ModifyType
		expected string
		success  bool
	}{
		{`null`, "$", `{}`, ModifySet, `{}`, true},
		{`{}`, "$.a", `3`, ModifySet, `{"a": 3}`, true},
		{`{"a": 3}`, "$.a", `[]`, ModifyReplace, `{"a": []}`, true},
		{`{"a": 3}`, "$.b", `"3"`, ModifySet, `{"a": 3, "b": "3"}`, true},
		{`{"a": []}`, "$.a[0]", `3`, ModifySet, `{"a": [3]}`, true},
		{`{"a": [3]}`, "$.a[1]", `4`, ModifyInsert, `{"a": [3, 4]}`, true},
		{`{"a": [3]}`, "$[0]", `4`, ModifySet, `4`, true},
		{`{"a": [3]}`, "$[1]", `4`, ModifySet, `[{"a": [3]}, 4]`, true},
		{`{"b": true}`, "$.b", `false`, ModifySet, `{"b": false}`, true},

		// nothing changed because the path is empty and we want to insert.
		{`{}`, "$", `1`, ModifyInsert, `{}`, true},
		// nothing changed because the path without last leg doesn't exist.
		{`{"a": [3, 4]}`, "$.b[1]", `3`, ModifySet, `{"a": [3, 4]}`, true},
		// nothing changed because the path without last leg doesn't exist.
		{`{"a": [3, 4]}`, "$.a[2].b", `3`, ModifySet, `{"a": [3, 4]}`, true},
		// nothing changed because we want to insert but the full path exists.
		{`{"a": [3, 4]}`, "$.a[0]", `30`, ModifyInsert, `{"a": [3, 4]}`, true},
		// nothing changed because we want to replace but the full path doesn't exist.
		{`{"a": [3, 4]}`, "$.a[2]", `30`, ModifyReplace, `{"a": [3, 4]}`, true},

		// bad path expression.
		{"null", "$.*", "{}", ModifySet, "null", false},
		{"null", "$[*]", "{}", ModifySet, "null", false},
		{"null", "$**.a", "{}", ModifySet, "null", false},
		{"null", "$**[3]", "{}", ModifySet, "null", false},
	}
	for _, tt := range tests {
		pathExpr, err := ParseJSONPathExpr(tt.setField)
		if err != nil {
			t.Fatal(err)
		}

		base := mustParseBinaryFromString(t, tt.base)
		value := mustParseBinaryFromString(t, tt.setValue)
		expected := mustParseBinaryFromString(t, tt.expected)
		obtain, err := base.Modify([]PathExpression{pathExpr}, []BinaryJSON{value}, tt.mt)
		if tt.success {
			if err != nil {
				t.Fatal(err)
			}
			if obtain.String() != expected.String() {
				t.Fatalf("got %#v, want %#v", obtain.String(), expected.String())
			}
		} else if err == nil {
			t.Fatal("got nil, want error")
		}
	}
}

func TestBinaryJSONRemove(t *testing.T) {
	var tests = []struct {
		base     string
		path     string
		expected string
		success  bool
	}{
		{`null`, "$", `{}`, false},
		{`{"a":[3]}`, "$.a[*]", `{"a":[3]}`, false},
		{`{}`, "$.a", `{}`, true},
		{`{"a":3}`, "$.a", `{}`, true},
		{`{"a":1,"b":2,"c":3}`, "$.b", `{"a":1,"c":3}`, true},
		{`{"a":1,"b":2,"c":3}`, "$.d", `{"a":1,"b":2,"c":3}`, true},
		{`{"a":3}`, "$[0]", `{"a":3}`, true},
		{`{"a":[3,4,5]}`, "$.a[0]", `{"a":[4,5]}`, true},
		{`{"a":[3,4,5]}`, "$.a[1]", `{"a":[3,5]}`, true},
		{`{"a":[3,4,5]}`, "$.a[4]", `{"a":[3,4,5]}`, true},
		{`{"a": [1, 2, {"aa": "xx"}]}`, "$.a[2].aa", `{"a": [1, 2, {}]}`, true},
	}
	for _, tt := range tests {
		pathExpr, err := ParseJSONPathExpr(tt.path)
		if err != nil {
			t.Fatal(err)
		}

		base := mustParseBinaryFromString(t, tt.base)
		expected := mustParseBinaryFromString(t, tt.expected)
		obtain, err := base.Remove([]PathExpression{pathExpr})
		if tt.success {
			if err != nil {
				t.Fatal(err)
			}
			if obtain.String() != expected.String() {
				t.Fatalf("got %#v, want %#v", obtain.String(), expected.String())
			}
		} else if err == nil {
			t.Fatal("got nil, want error")
		}
	}
}

func TestCompareBinary(t *testing.T) {
	jNull := mustParseBinaryFromString(t, `null`)
	jBoolTrue := mustParseBinaryFromString(t, `true`)
	jBoolFalse := mustParseBinaryFromString(t, `false`)
	jIntegerLarge, err := CreateBinary(uint64(1 << 63))
	if err != nil {
		t.Fatalf("CreateBinary failed: %v", err)
	}
	jIntegerSmall := mustParseBinaryFromString(t, `3`)
	jStringLarge := mustParseBinaryFromString(t, `"hello, world"`)
	jStringSmall := mustParseBinaryFromString(t, `"hello"`)
	jArrayLarge := mustParseBinaryFromString(t, `["a", "c"]`)
	jArraySmall := mustParseBinaryFromString(t, `["a", "b"]`)
	jObject := mustParseBinaryFromString(t, `{"a": "b"}`)

	var tests = []struct {
		left  BinaryJSON
		right BinaryJSON
	}{
		{jNull, jIntegerSmall},
		{jIntegerSmall, jIntegerLarge},
		{jIntegerLarge, jStringSmall},
		{jStringSmall, jStringLarge},
		{jStringLarge, jObject},
		{jObject, jArraySmall},
		{jArraySmall, jArrayLarge},
		{jArrayLarge, jBoolFalse},
		{jBoolFalse, jBoolTrue},
	}
	for _, tt := range tests {
		cmp := CompareBinary(tt.left, tt.right)
		if cmp >= 0 {
			t.Fatalf("got %#v, want < 0", cmp)
		}
	}
}

func TestBinaryJSONMerge(t *testing.T) {
	var tests = []struct {
		suffixes []string
		expected string
	}{
		{[]string{`{"a": 1}`, `{"b": 2}`}, `{"a": 1, "b": 2}`},
		{[]string{`{"a": 1}`, `{"a": 2}`}, `{"a": [1, 2]}`},
		{[]string{`[1]`, `[2]`}, `[1, 2]`},
		{[]string{`{"a": 1}`, `[1]`}, `[{"a": 1}, 1]`},
		{[]string{`[1]`, `{"a": 1}`}, `[1, {"a": 1}]`},
		{[]string{`{"a": 1}`, `4`}, `[{"a": 1}, 4]`},
		{[]string{`[1]`, `4`}, `[1, 4]`},
		{[]string{`4`, `{"a": 1}`}, `[4, {"a": 1}]`},
		{[]string{`4`, `1`}, `[4, 1]`},
		{[]string{`{}`, `[]`}, `[{}]`},
	}

	for _, tt := range tests {
		suffixes := make([]BinaryJSON, 0, len(tt.suffixes)+1)
		for _, s := range tt.suffixes {
			suffixes = append(suffixes, mustParseBinaryFromString(t, s))
		}
		result := MergeBinary(suffixes)
		cmp := CompareBinary(result, mustParseBinaryFromString(t, tt.expected))
		if cmp != 0 {
			t.Fatalf("got %#v, want %#v", cmp, 0)
		}
	}
}

func TestBinaryJSONKeyLengthLimit(t *testing.T) {
	bj, err := CreateBinary(map[string]any{strings.Repeat("k", math.MaxUint16): "v"})
	if err != nil {
		t.Fatalf("CreateBinary with %d-byte key: %v", math.MaxUint16, err)
	}
	if want := `{"` + strings.Repeat("k", math.MaxUint16) + `": "v"}`; bj.String() != want {
		t.Fatalf("got %q, want %q", bj.String(), want)
	}
	if _, err := CreateBinary(map[string]any{strings.Repeat("k", math.MaxUint16+1): "v"}); err == nil {
		t.Fatal("CreateBinary accepted a 65536-byte key")
	}
	// The binary length counts bytes, not runes.
	if _, err := CreateBinary(map[string]any{strings.Repeat("é", 32768): "v"}); err == nil {
		t.Fatal("CreateBinary accepted a 65536-byte multibyte key")
	}
	if _, err := CreateBinary(map[string]any{strings.Repeat("é", 32767): "v"}); err != nil {
		t.Fatalf("CreateBinary rejected a 65534-byte multibyte key: %v", err)
	}
}

func mustParseBinaryFromString(t *testing.T, s string) BinaryJSON {
	bj, err := ParseBinaryFromString(s)
	if err != nil {
		t.Fatal(err)
	}
	return bj
}

const benchStr = `{"a":[1,"2",{"aa":"bb"},4,null],"b":true,"c":null}`

func BenchmarkBinaryMarshal(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(benchStr)))
	bj, _ := ParseBinaryFromString(benchStr)
	for i := 0; i < b.N; i++ {
		_, _ = bj.MarshalJSON()
	}
}

func TestBinaryJSONContains(t *testing.T) {
	var tests = []struct {
		input    string
		target   string
		expected bool
	}{
		{`{}`, `{}`, true},
		{`{"a":1}`, `{}`, true},
		{`{"a":1}`, `1`, false},
		{`{"a":[1]}`, `[1]`, false},
		{`{"b":2, "c":3}`, `{"c":3}`, true},
		{`1`, `1`, true},
		{`[1]`, `1`, true},
		{`[1,2]`, `[1]`, true},
		{`[1,2]`, `[1,3]`, false},
		{`[1,2]`, `["1"]`, false},
		{`[1,2,[1,3]]`, `[1,3]`, true},
		{`[1,2,[1,[5,[3]]]]`, `[1,3]`, true},
		{`[1,2,[1,[5,{"a":[2,3]}]]]`, `[1,{"a":[3]}]`, true},
		{`[{"a":1}]`, `{"a":1}`, true},
		{`[{"a":1,"b":2}]`, `{"a":1}`, true},
		{`[{"a":{"a":1},"b":2}]`, `{"a":1}`, false},
	}

	for _, tt := range tests {
		obj := mustParseBinaryFromString(t, tt.input)
		target := mustParseBinaryFromString(t, tt.target)
		if ContainsBinary(obj, target) != tt.expected {
			t.Fatalf("got %#v, want %#v", ContainsBinary(obj, target), tt.expected)
		}
	}
}

func TestBinaryJSONDepth(t *testing.T) {
	var tests = []struct {
		input    string
		expected int
	}{
		{`{}`, 1},
		{`[]`, 1},
		{`true`, 1},
		{`[10, 20]`, 2},
		{`[[], {}]`, 2},
		{`[10, {"a": 20}]`, 3},
		{`{"Person": {"Name": "Homer", "Age": 39, "Hobbies": ["Eating", "Sleeping"]} }`, 4},
	}

	for _, tt := range tests {
		obj := mustParseBinaryFromString(t, tt.input)
		if obj.GetElemDepth() != tt.expected {
			t.Fatalf("got %#v, want %#v", obj.GetElemDepth(), tt.expected)
		}
	}
}

func TestBinaryJSONNumberComparisonExact(t *testing.T) {
	var numbers []BinaryJSON
	add := func(bj BinaryJSON, err error) {
		if err != nil {
			t.Fatal(err)
		}
		numbers = append(numbers, bj)
	}
	for _, n := range []int64{math.MinInt64, -9007199254740993, -1, 0, 1, 9007199254740992, 9007199254740993, math.MaxInt64} {
		add(CreateBinary(n))
	}
	for _, n := range []uint64{0, 1, 9007199254740993, 1 << 63, 1<<63 + 1, math.MaxUint64} {
		add(CreateBinary(n))
	}
	for _, n := range []float64{
		math.Nextafter(float64(math.MinInt64), math.Inf(-1)), float64(math.MinInt64), -1.5, -0.5, 0, 0.5, 1, 1.5,
		1 + 1e-9, math.Nextafter(math.Exp2(63), 0), math.Exp2(63), math.Nextafter(math.Exp2(64), 0), math.Exp2(64),
	} {
		add(CreateBinary(n))
	}
	rational := func(bj BinaryJSON) *big.Rat {
		switch bj.TypeCode {
		case TypeCodeInt64:
			return new(big.Rat).SetInt64(bj.GetInt64())
		case TypeCodeUint64:
			return new(big.Rat).SetInt(new(big.Int).SetUint64(bj.GetUint64()))
		default:
			return new(big.Rat).SetFloat64(bj.GetFloat64())
		}
	}
	for i, left := range numbers {
		for j, right := range numbers {
			t.Run(fmt.Sprintf("%d/%d", i, j), func(t *testing.T) {
				want := rational(left).Cmp(rational(right))
				got := CompareBinary(left, right)
				if got != want || got != -CompareBinary(right, left) || ContainsBinary(left, right) != (want == 0) {
					t.Fatalf("%s vs %s: compare=%d, want=%d", left, right, got, want)
				}
			})
		}
	}
}

func TestBinaryJSONRejectsMalformedLayout(t *testing.T) {
	array := mustParseBinaryFromString(t, `[1, [2], "x"]`)
	object := mustParseBinaryFromString(t, `{"a": 1, "b": true}`)
	for _, tc := range []struct {
		name   string
		base   BinaryJSON
		mutate func([]byte)
	}{
		{"count", array, func(b []byte) { endian.PutUint32(b, math.MaxUint32) }},
		{"unreferenced_payload", array, func(b []byte) { endian.PutUint32(b, 0) }},
		{"short_size", array, func(b []byte) { endian.PutUint32(b[dataSizeOff:], 1) }},
		{"long_size", array, func(b []byte) { endian.PutUint32(b[dataSizeOff:], math.MaxUint32) }},
		{"offset", array, func(b []byte) { endian.PutUint32(b[headerSize+1:], math.MaxUint32) }},
		{"back_reference", array, func(b []byte) { b[headerSize] = TypeCodeArray; endian.PutUint32(b[headerSize+1:], 0) }},
		{"overlap", array, func(b []byte) {
			copy(b[headerSize+valEntrySize:headerSize+2*valEntrySize], b[headerSize:headerSize+valEntrySize])
		}},
		{"unknown_child_type", array, func(b []byte) { b[headerSize] = 0xff }},
		{"string_length", array, func(b []byte) { off := endian.Uint32(b[headerSize+2*valEntrySize+1:]); b[off] = 127 }},
		{"key_offset", object, func(b []byte) { endian.PutUint32(b[headerSize:], 0) }},
		{"key_length", object, func(b []byte) { endian.PutUint16(b[headerSize+keyLenOff:], math.MaxUint16) }},
		{"bad_literal", object, func(b []byte) { b[headerSize+2*keyEntrySize+valEntrySize+1] = 9 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bj := tc.base.Copy()
			tc.mutate(bj.Value)
			if out, err := bj.MarshalJSON(); err == nil || out != nil {
				t.Fatalf("malformed layout emitted %q, %v", out, err)
			}
			if _, err := PeekBytesAsJSON(append([]byte{bj.TypeCode}, bj.Value...)); err == nil {
				t.Fatal("peek accepted malformed layout")
			}
		})
	}
	for _, bj := range []BinaryJSON{array, object, {TypeCode: TypeCodeInt64, Value: make([]byte, 8)}, {TypeCode: TypeCodeString, Value: []byte{3, 'a', 'b', 'c'}}} {
		for size := 0; size < len(bj.Value); size++ {
			t.Run(fmt.Sprintf("truncate/%d/%d", bj.TypeCode, size), func(t *testing.T) {
				if out, err := (BinaryJSON{TypeCode: bj.TypeCode, Value: bj.Value[:size]}).MarshalJSON(); err == nil || out != nil {
					t.Fatalf("truncation emitted %q, %v", out, err)
				}
			})
		}
	}
	for _, bj := range []BinaryJSON{{}, {TypeCode: 0xff, Value: []byte{0}}, {TypeCode: TypeCodeLiteral, Value: []byte{9}}, {TypeCode: TypeCodeString, Value: bytes.Repeat([]byte{0xff}, 12)}} {
		if _, err := bj.MarshalJSON(); err == nil {
			t.Fatal("invalid scalar accepted")
		}
	}
	for _, number := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		value := make([]byte, 8)
		endian.PutUint64(value, math.Float64bits(number))
		bj := BinaryJSON{TypeCode: TypeCodeFloat64, Value: value}
		if _, err := bj.MarshalJSON(); err == nil {
			t.Fatal("non-finite JSON number was accepted")
		}
		if _, err := PeekBytesAsJSON(append([]byte{bj.TypeCode}, value...)); err == nil {
			t.Fatal("peek accepted a non-finite JSON number")
		}
	}
	stream := append([]byte{array.TypeCode}, array.Value...)
	stream = append(stream, 0xff)
	if n, err := PeekBytesAsJSON(stream); err != nil || n != len(stream)-1 {
		t.Fatalf("stream peek = %d, %v", n, err)
	}
	path, err := ParseJSONPathExpr("$.x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := object.Modify([]PathExpression{path}, []BinaryJSON{{}}, ModifySet); err == nil {
		t.Fatal("invalid replacement accepted")
	}
}

func TestBinaryJSONRejectsPayloadGaps(t *testing.T) {
	path, err := ParseJSONPathExpr("$[0]")
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{`[1, 2]`, `{"a": 1, "b": 2}`} {
		base := mustParseBinaryFromString(t, text)
		count := base.GetElemCount()
		entryStart := headerSize
		payloadStart := headerSize + count*valEntrySize
		if base.TypeCode == TypeCodeObject {
			entryStart += count * keyEntrySize
			payloadStart += count * keyEntrySize
		}
		for gap := payloadStart; gap < len(base.Value); gap++ {
			t.Run(fmt.Sprintf("%s/gap=%d", text, gap), func(t *testing.T) {
				bj := base.Copy()
				bj.Value = append(bj.Value[:gap], append([]byte{0}, base.Value[gap:]...)...)
				endian.PutUint32(bj.Value[dataSizeOff:], uint32(len(bj.Value)))
				for i := range count {
					entries := []int{entryStart + i*valEntrySize + valTypeSize}
					if bj.TypeCode == TypeCodeObject {
						entries = append(entries, headerSize+i*keyEntrySize)
					}
					for _, entry := range entries {
						off := endian.Uint32(bj.Value[entry:])
						if int(off) >= gap {
							endian.PutUint32(bj.Value[entry:], off+1)
						}
					}
				}
				if _, err := bj.MarshalJSON(); err == nil {
					t.Fatal("marshal accepted a payload gap")
				}
				if _, err := PeekBytesAsJSON(append([]byte{bj.TypeCode}, bj.Value...)); err == nil {
					t.Fatal("peek accepted a payload gap")
				}
				if _, err := bj.Unquote(); err == nil {
					t.Fatal("unquote accepted a payload gap")
				}
				if _, err := bj.Modify([]PathExpression{path}, []BinaryJSON{base}, ModifySet); err == nil {
					t.Fatal("modify accepted a payload gap")
				}
				if _, err := bj.Remove([]PathExpression{path}); err == nil {
					t.Fatal("remove accepted a payload gap")
				}
			})
		}
	}
}

func TestBinaryJSONNestingLimit(t *testing.T) {
	levels := maxBinaryJSONDepth + 2
	value := make([]byte, levels*(headerSize+valEntrySize))
	for i := range levels {
		part := value[i*(headerSize+valEntrySize):]
		endian.PutUint32(part, 1)
		endian.PutUint32(part[dataSizeOff:], uint32(len(part)))
		if i == levels-1 {
			part[headerSize] = TypeCodeLiteral
		} else {
			part[headerSize] = TypeCodeArray
			endian.PutUint32(part[headerSize+valTypeSize:], headerSize+valEntrySize)
		}
	}
	if _, err := (BinaryJSON{TypeCode: TypeCodeArray, Value: value}).MarshalJSON(); err == nil {
		t.Fatal("excessive nesting was accepted")
	}
}

func FuzzBinaryJSONMarshal(f *testing.F) {
	for _, text := range []string{`null`, `1`, `"x"`, `[1, [2], "x"]`, `{"a": true}`} {
		bj, err := ParseBinaryFromString(text)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(append([]byte{bj.TypeCode}, bj.Value...))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			return
		}
		bj := BinaryJSON{TypeCode: data[0], Value: data[1:]}
		out, err := bj.MarshalJSON()
		if err != nil {
			return
		}
		if !stdjson.Valid(out) {
			t.Fatalf("invalid JSON output %q", out)
		}
		if n, err := PeekBytesAsJSON(data); err != nil || n != len(data) {
			t.Fatalf("valid marshal cannot peek: n=%d, %v", n, err)
		}
	})
}

// Truncated or hand-built values must not panic the public accessors.
func TestBinaryJSONTruncatedValues(t *testing.T) {
	for _, bj := range []BinaryJSON{
		{},
		{TypeCode: TypeCodeString},
		{TypeCode: TypeCodeInt64},
		{TypeCode: TypeCodeLiteral},
		{TypeCode: TypeCodeObject, Value: []byte{1}},
		{TypeCode: TypeCodeArray, Value: []byte{1}},
		{TypeCode: TypeCodeObject, Value: []byte{0, 0, 0, 1}},
	} {
		_ = bj.Type()
		_ = bj.GetString()
		_ = bj.GetInt64()
		_ = bj.GetUint64()
		_ = bj.GetFloat64()
		_ = bj.GetElemCount()
		_ = bj.String()
	}
	if got := (BinaryJSON{}).Type(); got != "UNKNOWN" {
		t.Fatalf("Type() = %q, want UNKNOWN", got)
	}
}
