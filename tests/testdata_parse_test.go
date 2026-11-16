package parser_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	pgparser "github.com/mia0x75/pgparser"
	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/format"
)

type testdataStatement struct {
	sql  string
	line int
}

// TestParseTestdataSQL parses every SQL statement in tests/testdata/*.sql,
// restores each successful parse back to SQL, and re-parses the result to
// check stability. Failures already recorded in testdata/known_failures.txt
// are tolerated; anything else fails the test. Set PGPARSER_TESTDATA=1 for a
// per-statement log.
func TestParseTestdataSQL(t *testing.T) {
	verbose := os.Getenv("PGPARSER_TESTDATA") != ""
	known, err := loadKnownFailures(filepath.Join("testdata", "known_failures.txt"))
	if err != nil {
		t.Fatalf("load known failures: %v", err)
	}
	files, err := filepath.Glob("testdata/*.sql")
	if err != nil {
		t.Fatalf("glob testdata: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no testdata SQL files found")
	}

	regenerate := os.Getenv("PGPARSER_REGENERATE") != ""
	// These upstream negative fixtures must be rejected, not hidden as parser gaps.
	invalidNumerics := map[string]bool{
		"SELECT 100_": true, "SELECT 100__000": true, "SELECT 1_000_.5": true,
		"SELECT 1_000._5": true, "SELECT 1_000.5_": true, "SELECT 1_000.5e_1": true,
	}
	var failures []corpusFailure
	total, skipped, failed, unsupported, knownHits, rejected := 0, 0, 0, 0, 0, 0
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
			key := fmt.Sprintf("%s:%d", file, stmt.line)
			parseErr, restoreErr := parseTestdataStatement(stmt.sql)
			if file == filepath.Join("testdata", "numerology.sql") && invalidNumerics[stmt.sql] {
				if parseErr == nil {
					t.Errorf("%s: invalid numeric fixture accepted: %s", key, stmt.sql)
				}
				rejected++
				continue
			}
			if errors.Is(restoreErr, ast.ErrUnsupportedRestore) {
				unsupported++
				continue
			}
			if err := errors.Join(parseErr, restoreErr); err != nil {
				if known[key] != "" && !regenerate {
					knownHits++
					if verbose {
						t.Logf("%s:%d: [known: %s] %v", file, stmt.line, known[key], err)
					}
					continue
				}
				failures = append(failures, corpusFailure{key: key, reason: failureReason(parseErr, restoreErr)})
				if regenerate {
					continue
				}
				failed++
				fileFailed++
				t.Errorf("%s:%d: %v", file, stmt.line, err)
				if verbose {
					t.Logf("%s:%d: failing statement:\n%s", file, stmt.line, stmt.sql)
				}
			}
		}
		if verbose && fileFailed > 0 {
			t.Logf("%s: %d/%d statements failed", file, fileFailed, len(stmts))
		}
	}
	if rejected != len(invalidNumerics) {
		t.Errorf("expected %d negative numeric fixtures, found %d", len(invalidNumerics), rejected)
	}
	if regenerate {
		if err := writeKnownFailures(filepath.Join("testdata", "known_failures.txt"), failures); err != nil {
			t.Fatalf("write known failures: %v", err)
		}
		t.Logf("regenerated known_failures.txt with %d entries", len(failures))
		return
	}
	stale := len(known) - knownHits
	t.Logf("%d statements; %d explicitly unsupported restores; %d psql-dependent skipped; %d expected lexical rejections; %d known failures matched (%d stale list entries, %d unexpected failures)",
		total, unsupported, skipped, rejected, knownHits, stale, failed)

	if failed > 0 {
		t.Errorf("%d SQL statements failed without a known_failures.txt entry (%d skipped); add each to the list or fix the parser/restore", failed, skipped)
	}
	if stale > 0 {
		t.Errorf("%d known_failures.txt entries no longer fail; run PGPARSER_REGENERATE=1 go test -run TestParseTestdataSQL ./tests to prune them", stale)
	}
}

// corpusFailure is one known_failures.txt entry.
type corpusFailure struct {
	key    string
	reason string
}

// failureReason classifies a corpus failure for known_failures.txt.
func failureReason(parseErr, restoreErr error) string {
	if parseErr != nil {
		return "expected parse rejection (unsupported PostgreSQL syntax)"
	}
	if strings.Contains(restoreErr.Error(), "Restore is not stable") {
		return "restore instability"
	}
	return "restore round-trip gap"
}

// writeKnownFailures rewrites the known-failure list, sorted by key.
func writeKnownFailures(path string, failures []corpusFailure) error {
	sort.Slice(failures, func(i, j int) bool { return failures[i].key < failures[j].key })
	var b strings.Builder
	b.WriteString("# Known corpus failures for TestParseTestdataSQL.\n")
	b.WriteString("# Format: <file.sql>:<line>\t<reason>\n")
	b.WriteString("# \"expected parse rejection\" = the original SQL is not accepted by the parser.\n")
	b.WriteString("# \"restore round-trip gap\" / \"restore instability\" = Restore output does not re-parse.\n")
	b.WriteString("# Regenerate: PGPARSER_REGENERATE=1 go test -run TestParseTestdataSQL ./tests\n")
	for _, f := range failures {
		b.WriteString(f.key)
		b.WriteString("\t")
		b.WriteString(f.reason)
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// loadKnownFailures reads the known-failure list of the corpus gate.
func loadKnownFailures(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	known := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		reason := ""
		if len(parts) == 2 {
			reason = parts[1]
		}
		known[parts[0]] = reason
	}
	return known, nil
}

type restoredSQLStmt struct {
	ast.StmtNode
	sql string
	err error
}

func (n restoredSQLStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WritePlain(n.sql)
	return n.err
}

func TestCorpusRestoreErrorClassification(t *testing.T) {
	first := restoreTestdataStatement(restoredSQLStmt{err: ast.ErrUnsupportedRestore})
	if !errors.Is(first, ast.ErrUnsupportedRestore) {
		t.Fatalf("first unsupported restore lost its classification: %v", first)
	}
	second := restoreTestdataStatement(restoredSQLStmt{sql: "ALTER TABLE t DROP CONSTRAINT c"})
	if second == nil || errors.Is(second, ast.ErrUnsupportedRestore) || !strings.Contains(second.Error(), "second Restore") {
		t.Fatalf("second unsupported restore was allowed: %v", second)
	}
}

func parseTestdataStatement(sql string) (parseErr, restoreErr error) {
	defer func() {
		if r := recover(); r != nil {
			restoreErr = fmt.Errorf("panic: %v", r)
		}
	}()
	stmt, err := pgparser.New().ParseOneStmt(sql + ";")
	if err != nil {
		return err, nil
	}
	return nil, restoreTestdataStatement(stmt)
}

func restoreTestdataStatement(stmt ast.StmtNode) error {
	var out strings.Builder
	if err := stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil {
		return err
	}
	reparsed, err := pgparser.New().ParseOneStmt(out.String())
	if err != nil {
		return fmt.Errorf("restored SQL cannot be parsed (%q): %v", out.String(), err)
	}
	var second strings.Builder
	if err := reparsed.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &second)); err != nil {
		// Only an unsupported first Restore may be skipped by the corpus gate.
		return fmt.Errorf("second Restore: %v", err)
	}
	if second.String() != out.String() {
		return fmt.Errorf("Restore is not stable: %q then %q", out.String(), second.String())
	}
	return nil
}

// psql commands and inline COPY data are not SQL statements.
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
