package format

import (
	"strings"
	"testing"
)

func TestRestoreCtx(t *testing.T) {
	testCases := []struct {
		flag   RestoreFlags
		expect string
	}{
		{0, "key`.'\"Word\\ str`.'\"ing\\ na`.'\"Me\\"},
		{RestoreStringSingleQuotes, "key`.'\"Word\\ 'str`.''\"ing\\' na`.'\"Me\\"},
		{RestoreStringDoubleQuotes, "key`.'\"Word\\ \"str`.'\"\"ing\\\" na`.'\"Me\\"},
		{RestoreStringEscapeBackslash, "key`.'\"Word\\ str`.'\"ing\\\\ na`.'\"Me\\"},
		{RestoreKeyWordUppercase, "KEY`.'\"WORD\\ str`.'\"ing\\ na`.'\"Me\\"},
		{RestoreKeyWordLowercase, "key`.'\"word\\ str`.'\"ing\\ na`.'\"Me\\"},
		{RestoreNameUppercase, "key`.'\"Word\\ str`.'\"ing\\ NA`.'\"ME\\"},
		{RestoreNameLowercase, "key`.'\"Word\\ str`.'\"ing\\ na`.'\"me\\"},
		{RestoreNameDoubleQuotes, "key`.'\"Word\\ str`.'\"ing\\ \"na`.'\"\"Me\\\""},
		{RestoreNameBackQuotes, "key`.'\"Word\\ str`.'\"ing\\ `na``.'\"Me\\`"},
		{DefaultRestoreFlags, "KEY`.'\"WORD\\ 'str`.''\"ing\\' \"na`.'\"\"Me\\\""},
		{RestoreStringSingleQuotes | RestoreStringDoubleQuotes, "key`.'\"Word\\ 'str`.''\"ing\\' na`.'\"Me\\"},
		{RestoreKeyWordUppercase | RestoreKeyWordLowercase, "KEY`.'\"WORD\\ str`.'\"ing\\ na`.'\"Me\\"},
		{RestoreNameUppercase | RestoreNameLowercase, "key`.'\"Word\\ str`.'\"ing\\ NA`.'\"ME\\"},
		{RestoreNameDoubleQuotes | RestoreNameBackQuotes, "key`.'\"Word\\ str`.'\"ing\\ \"na`.'\"\"Me\\\""},
	}
	var sb strings.Builder
	for _, testCase := range testCases {
		sb.Reset()
		ctx := NewRestoreCtx(testCase.flag, &sb)
		ctx.WriteKeyword("key`.'\"Word\\")
		ctx.WritePlain(" ")
		ctx.WriteString("str`.'\"ing\\")
		ctx.WritePlain(" ")
		ctx.WriteName("na`.'\"Me\\")
		if sb.String() != testCase.expect {
			t.Fatalf("got %#v, want %#v (case: %#v)", sb.String(), testCase.expect, testCase)
		}
		if err := ctx.Err(); err != nil {
			t.Fatal(err)
		}
	}
}
