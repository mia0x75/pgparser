package parser_test

import (
	"fmt"
	"strings"
	"testing"

	pgparser "github.com/mia0x75/pgparser"
)

func BenchmarkScanOperatorChain(b *testing.B) {
	for _, size := range []int{8192, 16384, 32768} {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			input := strings.Repeat("+", size)
			b.ReportAllocs()
			b.SetBytes(int64(len(input)))
			for b.Loop() {
				l := pgparser.NewLexer(input)
				count := 0
				for token := l.NextToken(); token.Type != 0; token = l.NextToken() {
					count++
				}
				if l.Err != nil || count != size {
					b.Fatalf("scan count %d, error %v", count, l.Err)
				}
			}
		})
	}
}

func BenchmarkParseUnaryChain(b *testing.B) {
	for _, size := range []int{8192, 16384, 32768} {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			input := "SELECT " + strings.Repeat("+", size) + "1"
			p := pgparser.New()
			b.ReportAllocs()
			for b.Loop() {
				if _, err := p.ParseOneStmt(input); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkParseUnionChain(b *testing.B) {
	for _, size := range []int{1000, 2000, 4000, 8000} {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			input := strings.Repeat("SELECT 1 UNION ALL ", size-1) + "SELECT 1"
			p := pgparser.New()
			b.ReportAllocs()
			for b.Loop() {
				if _, err := p.ParseOneStmt(input); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
