package format

import (
	"bytes"
	"errors"
	"io"

	"strings"
	"testing"
)

func checkFormat(t *testing.T, f Formatter, buf *bytes.Buffer, str, expect string) {
	_, err := f.Format(str, 3)
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(buf)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != expect {
		t.Fatalf("got %#v, want %#v", string(b), expect)
	}
}

func TestFormat(t *testing.T) {
	str := "abc%d%%e%i\nx\ny\n%uz\n"
	buf := &bytes.Buffer{}
	f := IndentFormatter(buf, "\t")
	expect := `abc3%e
	x
	y
z
`
	checkFormat(t, f, buf, str, expect)

	str = "abc%d%%e%i\nx\ny\n%uz\n%i\n"
	buf = &bytes.Buffer{}
	f = FlatFormatter(buf)
	expect = "abc3%e x y z\n "
	checkFormat(t, f, buf, str, expect)
}

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
		ctx.WriteKeyWord("key`.'\"Word\\")
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

type errorWriter struct {
	err   error
	calls int
}

func (w *errorWriter) Write([]byte) (int, error) {
	w.calls++
	return 0, w.err
}

func TestRestoreCtxFirstError(t *testing.T) {
	writeErr := errors.New("mock write error")
	w := &errorWriter{err: writeErr}
	ctx := NewRestoreCtx(DefaultRestoreFlags, w)
	ctx.WriteName("name")
	ctx.WriteKeyWord("SELECT")
	ctx.WriteString("value")
	ctx.WritePlain("plain")
	ctx.WritePlainf("%d", 1)
	if !errors.Is(ctx.Err(), writeErr) || w.calls != 1 {
		t.Fatalf("error = %v, writer calls = %d", ctx.Err(), w.calls)
	}
}

func TestRestoreCtxNilWriter(t *testing.T) {
	ctx := NewRestoreCtx(DefaultRestoreFlags, nil)
	ctx.WritePlain("SELECT")
	if ctx.Err() == nil {
		t.Fatal("nil writer accepted")
	}
}
