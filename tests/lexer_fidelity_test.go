package parser_test

import (
	"strings"
	"testing"

	pgparser "github.com/mia0x75/pgparser/parser"
)

func TestNumericLiteralSeparators(t *testing.T) {
	invalid := []string{
		"1_", "1__2", "1_.2", "1._2", "1e2_", "1e2__3", "1e_2",
		"0x_", "0x__1", "0x1_", "0x1__2", "0b_", "0b102", "0b1_0_",
		"0o_", "0o78", "0o1__2", "0x1g", "0b1a", "0o1a",
	}
	for _, literal := range invalid {
		t.Run("invalid_"+literal, func(t *testing.T) {
			if _, err := pgparser.New().ParseOneStmt("SELECT " + literal); err == nil {
				t.Fatal("invalid numeric literal accepted")
			}
		})
	}
	valid := []string{
		"1_000", "1_2.3_4", "1.2e+3_4", ".5_6", "1.",
		"0x_FF", "0x1_2", "0b_10", "0b1_0", "0o_77", "0o1_2",
	}
	for _, literal := range valid {
		t.Run("valid_"+literal, func(t *testing.T) {
			n := mustParseOne(t, "SELECT "+literal)
			mustParseOne(t, restoreString(t, n))
		})
	}
}

func TestOperatorTokenBoundaries(t *testing.T) {
	cases := []struct {
		input string
		want  []string
	}{
		{"++++", []string{"+", "+", "+", "+"}},
		{"+*++", []string{"+*", "+", "+"}},
		{"+~++", []string{"+~++"}},
		{"++/* comment */+", []string{"+", "+", "+"}},
		{"++-- comment\n+", []string{"+", "+", "+"}},
		{"+/* first -- */~+", []string{"+", "~+"}},
		{"// +//+", []string{"//", "+//", "+"}},
		{"++ +~+ ++", []string{"+", "+", "+~+", "+", "+"}},
		{strings.Repeat("+", 65536), nil},
	}
	for _, tc := range cases {
		name := tc.input
		if tc.want == nil {
			name = "long_plus_chain"
		}
		t.Run(name, func(t *testing.T) {
			l := pgparser.NewLexer(tc.input)
			count := 0
			for token := l.NextToken(); token.Type != 0; token = l.NextToken() {
				if tc.want == nil {
					if token.Str != "+" || token.Loc != count {
						t.Fatalf("token %d = %+v", count, token)
					}
				} else if count >= len(tc.want) || token.Str != tc.want[count] {
					t.Fatalf("token %d = %+v, want %v", count, token, tc.want)
				}
				count++
			}
			if l.Err != nil {
				t.Fatal(l.Err)
			}
			wantCount := len(tc.want)
			if tc.want == nil {
				wantCount = len(tc.input)
			}
			if count != wantCount {
				t.Fatalf("token count = %d, want %d", count, wantCount)
			}
		})
	}
}

func TestRelationNameBoundaries(t *testing.T) {
	for _, name := range []string{"a.b.c.d", "a[1]", "a.b[1]", "a.*"} {
		t.Run(name, func(t *testing.T) {
			if _, err := pgparser.New().ParseOneStmt("SELECT 1 FROM " + name); err == nil {
				t.Fatal("invalid relation name accepted")
			}
		})
	}
}

func TestDoStatementRequiresCode(t *testing.T) {
	if _, err := pgparser.New().ParseOneStmt("DO LANGUAGE a"); err == nil {
		t.Fatal("DO without a code block accepted")
	}
}

func TestRejectInvalidUTF8(t *testing.T) {
	for _, sql := range []string{"DO $$\xdf$$", "SELECT \xff", "SELECT '\xc0'"} {
		t.Run(sql, func(t *testing.T) {
			if _, err := pgparser.New().ParseOneStmt(sql); err == nil {
				t.Fatal("invalid UTF-8 accepted")
			}
		})
	}
}
