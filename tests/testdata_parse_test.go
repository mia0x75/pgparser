package parser_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pgparser "github.com/mia0x75/pgparser"
)

type testdataStatement struct {
	sql  string
	line int
}

// TestParseTestdataSQL parses every SQL statement in tests/testdata/*.sql and
// reports the ones that fail. psql meta-commands, COPY data blocks, and
// statements that depend on psql variables are skipped.
func TestParseTestdataSQL(t *testing.T) {
	if os.Getenv("PGPARSER_TESTDATA") == "" {
		t.Skip("diagnostic test; set PGPARSER_TESTDATA=1 to run")
	}
	files, err := filepath.Glob("testdata/*.sql")
	if err != nil {
		t.Fatalf("glob testdata: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no testdata SQL files found")
	}

	total, skipped, failed := 0, 0, 0
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		stmts := splitTestdataSQL(string(data))
		var fileFailed int
		for _, stmt := range stmts {
			if hasPsqlVariable(stmt.sql) {
				skipped++
				continue
			}
			total++
			if err := parseTestdataStatement(stmt.sql); err != nil {
				failed++
				fileFailed++
				t.Logf("%s:%d: %v\n%s", file, stmt.line, err, stmt.sql)
			}
		}
		if fileFailed > 0 {
			t.Logf("%s: %d/%d statements failed", file, fileFailed, len(stmts))
		}
	}

	if failed > 0 {
		t.Errorf("%d/%d SQL statements failed to parse (%d skipped)", failed, total, skipped)
	}
}

// splitTestdataSQL splits PostgreSQL regression-test input into statements. It
// removes comments and psql meta-commands, and skips COPY FROM STDIN data.
func parseTestdataStatement(sql string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	_, err = pgparser.New().ParseOneStmt(sql + ";")
	return err
}

func splitTestdataSQL(src string) []testdataStatement {
	var (
		stmts      []testdataStatement
		b          strings.Builder
		line       = 1
		startLine  = 1
		inSingle   bool
		inDouble   bool
		dollarTag  string
		blockDeep  int
		parenDeep  int
		copyBlocks int
		ifDepth    int
		atomicDeep int
		caseDeep   int
		scsOn      bool
		bsEscape   bool
	)
	scsOn = true

	appendCode := func(s string) {
		if strings.TrimSpace(b.String()) == "" && strings.TrimSpace(s) != "" {
			startLine = line
		}
		b.WriteString(s)
	}
	flush := func() {
		sql := strings.TrimSpace(b.String())
		b.Reset()
		parenDeep = 0
		if sql == "" {
			return
		}
		stmts = append(stmts, testdataStatement{sql: sql, line: startLine})
		upper := strings.ToUpper(sql)
		switch {
		case strings.HasPrefix(upper, "SET STANDARD_CONFORMING_STRINGS"):
			switch {
			case strings.HasSuffix(upper, " OFF"):
				scsOn = false
			case strings.HasSuffix(upper, " ON"):
				scsOn = true
			}
		case upper == "RESET STANDARD_CONFORMING_STRINGS":
			scsOn = true
		}
		if strings.HasPrefix(upper, "COPY ") && !strings.HasPrefix(upper, "COPY (") &&
			(strings.Contains(upper, "FROM STDIN") || strings.Contains(upper, "FROM STDOUT")) {
			copyBlocks++
		}
	}

	for i := 0; i < len(src); {
		if i == 0 || src[i-1] == '\n' {
			j := i
			for j < len(src) && (src[j] == ' ' || src[j] == '\t') {
				j++
			}
			if strings.HasPrefix(src[j:], "**") {
				end := strings.IndexByte(src[i:], '\n')
				if end < 0 {
					break
				}
				line++
				i += end + 1
				continue
			}
		}
		if copyBlocks > 0 {
			end := strings.IndexByte(src[i:], '\n')
			if end < 0 {
				break
			}
			lineText := src[i : i+end]
			line++
			i += end + 1
			if strings.TrimSpace(lineText) == `\.` {
				copyBlocks--
			}
			continue
		}
		if ifDepth > 0 {
			end := strings.IndexByte(src[i:], '\n')
			if end < 0 {
				break
			}
			lineText := strings.TrimSpace(src[i : i+end])
			if strings.HasPrefix(lineText, `\if`) {
				ifDepth++
			} else if strings.HasPrefix(lineText, `\endif`) {
				ifDepth--
			}
			line++
			i += end + 1
			continue
		}

		c := src[i]
		if inDollarTag := dollarTag; inDollarTag != "" {
			if strings.HasPrefix(src[i:], inDollarTag) {
				appendCode(inDollarTag)
				i += len(inDollarTag)
				dollarTag = ""
				continue
			}
			appendCode(string(c))
			if c == '\n' {
				line++
			}
			i++
			continue
		}
		if blockDeep > 0 {
			if strings.HasPrefix(src[i:], "/*") {
				blockDeep++
				i += 2
				continue
			}
			if strings.HasPrefix(src[i:], "*/") {
				blockDeep--
				i += 2
				continue
			}
			if c == '\n' {
				line++
			}
			i++
			continue
		}
		if inSingle {
			appendCode(string(c))
			if bsEscape && c == '\\' && i+1 < len(src) {
				i++
				appendCode(string(src[i]))
				if src[i] == '\n' {
					line++
				}
				i++
				continue
			}
			if c == '\'' {
				if i+1 < len(src) && src[i+1] == '\'' {
					appendCode("'")
					i += 2
					continue
				}
				inSingle = false
			}
			if c == '\n' {
				line++
			}
			i++
			continue
		}
		if inDouble {
			appendCode(string(c))
			if c == '"' {
				if i+1 < len(src) && src[i+1] == '"' {
					appendCode(`"`)
					i += 2
					continue
				}
				inDouble = false
			}
			if c == '\n' {
				line++
			}
			i++
			continue
		}

		switch {
		case c == '-' && i+1 < len(src) && src[i+1] == '-':
			end := strings.IndexByte(src[i:], '\n')
			if end < 0 {
				i = len(src)
			} else {
				line++
				i += end + 1
			}
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			blockDeep = 1
			i += 2
		case c == '\'':
			appendCode("'")
			inSingle = true
			uStr := i >= 2 && src[i-1] == '&' && (src[i-2] == 'U' || src[i-2] == 'u')
			eStr := i >= 1 && (src[i-1] == 'E' || src[i-1] == 'e') && (i < 2 || !isIdentPartByte(src[i-2]))
			bsEscape = !uStr && (eStr || !scsOn)
			i++
		case c == '"':
			appendCode(`"`)
			inDouble = true
			i++
		case c == '$':
			if tag, size := scanDollarQuoteTag(src[i:]); size > 0 {
				appendCode(tag)
				dollarTag = tag
				i += size
			} else {
				appendCode("$")
				i++
			}
		case c == '\\':
			end := strings.IndexByte(src[i:], '\n')
			lineEnd := len(src)
			if end >= 0 {
				lineEnd = i + end
			}
			cmd := strings.TrimSpace(src[i:lineEnd])
			if strings.HasPrefix(cmd, `\if`) {
				ifDepth = 1
				i = lineEnd
				if end >= 0 {
					line++
					i++
				}
				continue
			}
			if strings.HasPrefix(cmd, `\copy`) && strings.Contains(strings.ToLower(cmd), "from stdin") {
				copyBlocks++
			}
			if i+1 < len(src) && src[i+1] == ';' {
				rest := strings.ToLower(src[i+2 : lineEnd])
				copyBlocks += strings.Count(rest, "from stdin") + strings.Count(rest, "from stdout")
				flush()
				i = lineEnd
				if end >= 0 {
					line++
					i++
				}
				continue
			}
			flush()
			if end < 0 {
				i = len(src)
			} else {
				line++
				i += end + 1
			}
		case c == '(':
			appendCode("(")
			parenDeep++
			i++
		case c == ')':
			appendCode(")")
			if parenDeep > 0 {
				parenDeep--
			}
			i++
		case c == ';':
			if parenDeep == 0 && atomicDeep == 0 {
				flush()
			} else {
				appendCode(";")
			}
			i++
		default:
			if isIdentStartByte(c) && (i == 0 || !isIdentPartByte(src[i-1])) {
				j := i
				for j < len(src) && isIdentPartByte(src[j]) {
					j++
				}
				switch strings.ToLower(src[i:j]) {
				case "begin":
					if atomicDeep == 0 && hasAtomicAhead(src[j:]) {
						atomicDeep++
					}
				case "case":
					if atomicDeep > 0 {
						caseDeep++
					}
				case "end":
					if atomicDeep > 0 {
						if caseDeep > 0 {
							caseDeep--
						} else {
							atomicDeep--
						}
					}
				}
			}
			appendCode(string(c))
			if c == '\n' {
				line++
			}
			i++
		}
	}
	flush()
	return stmts
}

func scanDollarQuoteTag(s string) (string, int) {
	if len(s) < 2 || s[0] != '$' {
		return "", 0
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if c == '$' {
			return s[:i+1], i + 1
		}
		if c != '_' && (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (i <= 1 || c < '0' || c > '9') {
			return "", 0
		}
	}
	return "", 0
}

func hasPsqlVariable(sql string) bool {
	var inSingle, inDouble bool
	var dollarTag string
	for i := 0; i < len(sql); {
		c := sql[i]
		if dollarTag != "" {
			if strings.HasPrefix(sql[i:], dollarTag) {
				i += len(dollarTag)
				dollarTag = ""
			} else {
				i++
			}
			continue
		}
		if inSingle {
			if c == '\\' && i+1 < len(sql) {
				i += 2
				continue
			}
			if c == '\'' {
				if i+1 < len(sql) && sql[i+1] == '\'' {
					i += 2
					continue
				}
				inSingle = false
			}
			i++
			continue
		}
		if inDouble {
			if c == '"' {
				if i+1 < len(sql) && sql[i+1] == '"' {
					i += 2
					continue
				}
				inDouble = false
			}
			i++
			continue
		}
		switch c {
		case '\'':
			inSingle = true
			i++
		case '"':
			inDouble = true
			i++
		case '$':
			if tag, size := scanDollarQuoteTag(sql[i:]); size > 0 {
				dollarTag = tag
				i += size
			} else {
				i++
			}
		case ':':
			if i+1 < len(sql) && sql[i+1] != ':' && (isIdentStartByte(sql[i+1]) || sql[i+1] == '{' || sql[i+1] == '\'' || sql[i+1] == '"') {
				return true
			}
			i++
		default:
			i++
		}
	}
	return false
}

func isIdentStartByte(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func isIdentPartByte(c byte) bool {
	return isIdentStartByte(c) || c >= '0' && c <= '9'
}

// hasAtomicAhead reports whether s starts with the ATOMIC keyword.
func hasAtomicAhead(s string) bool {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	const kw = "atomic"
	if len(s)-i < len(kw) || !strings.EqualFold(s[i:i+len(kw)], kw) {
		return false
	}
	end := i + len(kw)
	return end == len(s) || !isIdentPartByte(s[end])
}
