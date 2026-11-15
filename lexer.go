// Package parser implements a PostgreSQL-compatible SQL lexer and parser.
// This lexer is a direct translation of PostgreSQL's scan.l.
package parser

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Token types returned by the lexer (internal, prefixed with lex_).
// These are mapped to parser token constants in parse.go.
// Keywords are defined in keywords.go using constants from parser.go.
// Non-keyword tokens start after all keywords to avoid conflicts.
// Non-keyword token base - must be after all keyword tokens
// Keywords use 258 + 0..491 (492 keywords), so we start at 800
const nonKeywordTokenBase = 800

const maxIdentifierBytes = 63

const (
	// Special tokens
	lex_EOF = 0
)

// Internal lexer token types for non-keywords.
// These start at nonKeywordTokenBase and are mapped to parser constants in parse.go.
const (
	lex_ICONST = nonKeywordTokenBase + iota
	lex_FCONST
	lex_SCONST  // String constant
	lex_BCONST  // Bit string constant (B'...')
	lex_XCONST  // Hex string constant (X'...')
	lex_USCONST // Unicode string constant (U&'...')
	lex_IDENT
	lex_UIDENT         // Unicode identifier (U&"...")
	lex_TYPECAST       // ::
	lex_DOT_DOT        // ..
	lex_COLON_EQUALS   // :=
	lex_EQUALS_GREATER // =>
	lex_LESS_EQUALS    // <=
	lex_GREATER_EQUALS // >=
	lex_NOT_EQUALS     // <> or !=
	lex_PARAM          // $1, $2, etc.
	lex_Op
)

// LexerState represents the current state of the lexer (for string/comment handling).
type LexerState int

const (
	stateInitial LexerState = iota
	stateXB                 // bit string literal
	stateXD                 // delimited identifiers (double-quoted)
	stateXH                 // hexadecimal byte string
	stateXQ                 // standard quoted strings
	stateXQS                // quote stop (detect continued strings)
	stateXE                 // extended quoted strings (backslash escapes)
	stateXDOLQ              // dollar-quoted strings
	stateXUI                // quoted identifier with Unicode escapes
	stateXUS                // quoted string with Unicode escapes
	stateXEU                // Unicode surrogate pair in extended quoted string
)

// Token represents a lexical token.
type Token struct {
	Type int    // Token type (IDENT, ICONST, keyword token, etc.)
	Str  string // String value for identifiers, operators, string literals
	Ival int64  // Integer value for ICONST
	Loc  int    // Byte offset in the source text
}

// Lexer implements a PostgreSQL-compatible SQL lexer.
type Lexer struct {
	input string // Input SQL text
	pos   int    // Current position in input (byte offset)
	start int    // Start position of current token

	// Reuse the scanned boundary when SQL rules split an operator's suffix.
	operatorEnd     int
	operatorBodyEnd int

	state              LexerState // Current lexer state
	stateBeforeStrStop LexerState // State before entering xqs

	// Literal buffer for building string/identifier values
	literalbuf strings.Builder

	// Dollar quote delimiter
	dolqstart string

	// Unicode surrogate handling
	utf16FirstPart rune

	// Configuration
	StandardConformingStrings bool // If true, backslash is not escape in standard strings
	BackslashQuote            int  // 0=off, 1=on, 2=safe_encoding

	// Error handling
	Err error

	// newlineIdx caches the byte offsets of '\n' for lineAt, built on first use.
	newlineIdx []int
}

// BackslashQuote values
const (
	BackslashQuoteOff = iota
	BackslashQuoteOn
	BackslashQuoteSafeEncoding
)

// NewLexer creates a new lexer for the given input.
func NewLexer(input string) *Lexer {
	l := &Lexer{
		input:                     input,
		pos:                       0,
		state:                     stateInitial,
		StandardConformingStrings: true,
		BackslashQuote:            BackslashQuoteSafeEncoding,
	}
	if pos := strings.IndexByte(input, 0); pos >= 0 {
		l.pos = pos
		l.Err = fmt.Errorf("invalid NUL byte in SQL")
	} else if !utf8.ValidString(input) {
		l.Err = fmt.Errorf("invalid UTF-8 in SQL")
	}
	return l
}

// NextToken returns the next token from the input.
func (l *Lexer) NextToken() Token {
	if l.Err != nil {
		return Token{Type: lex_EOF, Loc: l.pos}
	}
	for {
		if l.pos >= len(l.input) {
			// At EOF - if we're in a string state, that's an error
			// If we're in quote-stop state, return the completed string
			if l.state == stateXQS {
				return l.lexQuoteContinue()
			}
			if l.state != stateInitial {
				l.Err = fmt.Errorf("unterminated SQL token")
				l.state = stateInitial
			}
			return Token{Type: lex_EOF, Loc: l.pos}
		}

		switch l.state {
		case stateInitial:
			l.start = l.pos
			return l.lexInitial()
		case stateXB, stateXH:
			return l.lexBitOrHexString()
		case stateXQ, stateXE, stateXUS:
			return l.lexQuotedString()
		case stateXQS:
			return l.lexQuoteContinue()
		case stateXEU:
			return l.lexUnicodeSurrogate()
		case stateXD, stateXUI:
			return l.lexDelimitedIdent()
		case stateXDOLQ:
			return l.lexDollarQuote()
		default:
			l.Err = fmt.Errorf("unknown lexer state: %d", l.state)
			return Token{Type: lex_EOF, Loc: l.start}
		}
	}
}

// lexInitial handles tokens in the initial state.
func (l *Lexer) lexInitial() Token {
	if err := l.skipWhitespaceAndComments(); err != nil {
		l.Err = err
		return Token{Type: lex_EOF, Loc: l.pos}
	}
	if l.pos >= len(l.input) {
		return Token{Type: lex_EOF, Loc: l.pos}
	}

	l.start = l.pos
	ch := l.input[l.pos]

	// Bit string: B'...'
	if (ch == 'b' || ch == 'B') && l.pos+1 < len(l.input) && l.input[l.pos+1] == '\'' {
		l.pos += 2
		l.state = stateXB
		l.literalbuf.Reset()
		l.literalbuf.WriteByte('b')
		return l.NextToken()
	}

	// Hex string: X'...'
	if (ch == 'x' || ch == 'X') && l.pos+1 < len(l.input) && l.input[l.pos+1] == '\'' {
		l.pos += 2
		l.state = stateXH
		l.literalbuf.Reset()
		l.literalbuf.WriteByte('x')
		return l.NextToken()
	}

	// National character: N'...' -> treat as NCHAR followed by string
	if (ch == 'n' || ch == 'N') && l.pos+1 < len(l.input) && l.input[l.pos+1] == '\'' {
		// Look up NCHAR keyword
		kw := LookupKeyword("nchar")
		if kw != nil {
			l.pos++ // consume only 'n'
			return Token{Type: kw.Token, Str: kw.Name, Loc: l.start}
		}
		// If NCHAR isn't a keyword, return 'n' as identifier
		l.pos++
		return Token{Type: lex_IDENT, Str: "n", Loc: l.start}
	}

	// Extended quoted string: E'...'
	if (ch == 'e' || ch == 'E') && l.pos+1 < len(l.input) && l.input[l.pos+1] == '\'' {
		l.pos += 2
		l.state = stateXE
		l.literalbuf.Reset()
		return l.NextToken()
	}

	// Unicode string: U&'...'
	if (ch == 'u' || ch == 'U') && l.pos+1 < len(l.input) && l.input[l.pos+1] == '&' {
		if l.pos+2 < len(l.input) && l.input[l.pos+2] == '\'' {
			if !l.StandardConformingStrings {
				l.Err = fmt.Errorf("unsafe use of string constant with Unicode escapes")
				return Token{Type: lex_EOF, Loc: l.start}
			}
			l.pos += 3
			l.state = stateXUS
			l.literalbuf.Reset()
			return l.NextToken()
		}
		// Unicode identifier: U&"..."
		if l.pos+2 < len(l.input) && l.input[l.pos+2] == '"' {
			l.pos += 3
			l.state = stateXUI
			l.literalbuf.Reset()
			return l.NextToken()
		}
		// Just 'U&' without quote - return 'u' as identifier
		l.pos++
		ident := l.downcase("u")
		return Token{Type: lex_IDENT, Str: ident, Loc: l.start}
	}

	// Standard quoted string: '...'
	if ch == '\'' {
		l.pos++
		if l.StandardConformingStrings {
			l.state = stateXQ
		} else {
			l.state = stateXE
		}
		l.literalbuf.Reset()
		return l.NextToken()
	}

	// Double-quoted identifier: "..."
	if ch == '"' {
		l.pos++
		l.state = stateXD
		l.literalbuf.Reset()
		return l.NextToken()
	}

	// Dollar-quoted string: $tag$...$tag$
	if ch == '$' {
		if tag := l.scanDollarTag(); tag != "" {
			l.dolqstart = tag
			l.state = stateXDOLQ
			l.literalbuf.Reset()
			return l.NextToken()
		}
		// Check for parameter: $1, $2, etc.
		if l.pos+1 < len(l.input) && isDigit(l.input[l.pos+1]) {
			return l.lexParam()
		}
		// Single $ is returned as itself
		l.pos++
		return Token{Type: int(ch), Str: string(ch), Loc: l.start}
	}

	// Two-character special tokens (must check before operators)
	if l.pos+1 < len(l.input) {
		ch2 := l.input[l.pos : l.pos+2]
		switch ch2 {
		case "::":
			l.pos += 2
			return Token{Type: lex_TYPECAST, Str: "::", Loc: l.start}
		case "..":
			l.pos += 2
			return Token{Type: lex_DOT_DOT, Str: "..", Loc: l.start}
		case ":=":
			l.pos += 2
			return Token{Type: lex_COLON_EQUALS, Str: ":=", Loc: l.start}
		}
	}

	// Operators (including multi-char operators like ->, ||, etc.)
	// Must check before self-delimiting because some chars overlap
	if isOpChar(ch) {
		return l.lexOperator()
	}

	// Numbers - check for leading dot numeric BEFORE self-delimiting
	// because .5 should be parsed as a numeric, not as . followed by 5
	if ch == '.' && l.pos+1 < len(l.input) && isDigit(l.input[l.pos+1]) {
		return l.lexNumber()
	}

	// Self-delimiting characters that are NOT operator chars
	// Note: +, -, *, /, %, ^, <, >, = are handled by lexOperator above
	if isSelfOnly(ch) {
		l.pos++
		return Token{Type: int(ch), Str: string(ch), Loc: l.start}
	}

	// Numbers starting with digit
	if isDigit(ch) {
		return l.lexNumber()
	}

	// Identifiers and keywords
	if isIdentStart(ch) {
		return l.lexIdent()
	}

	// Any other character
	if ch == 0 {
		l.Err = fmt.Errorf("NUL byte in input")
		return Token{Type: lex_EOF, Loc: l.start}
	}
	l.pos++
	return Token{Type: int(ch), Str: string(ch), Loc: l.start}
}

// skipLineComment skips a -- comment to end of line.
func (l *Lexer) skipLineComment() {
	l.pos += 2 // skip --
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		l.pos++
		if ch == '\n' || ch == '\r' {
			break
		}
	}
}

// skipBlockComment scans a /* ... */ comment starting just after its opening
// delimiter.  PostgreSQL allows nested block comments.
func (l *Lexer) skipBlockComment() bool {
	depth := 1
	for l.pos < len(l.input) {
		switch {
		case l.input[l.pos] == '/' && l.pos+1 < len(l.input) && l.input[l.pos+1] == '*':
			depth++
			l.pos += 2
		case l.input[l.pos] == '*' && l.pos+1 < len(l.input) && l.input[l.pos+1] == '/':
			depth--
			l.pos += 2
			if depth == 0 {
				return true
			}
		default:
			l.pos++
		}
	}
	return false
}

// lexBitOrHexString handles B'...' and X'...' strings.
func (l *Lexer) lexBitOrHexString() Token {
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == '\'' {
			l.pos++
			l.stateBeforeStrStop = l.state
			l.state = stateXQS
			return l.NextToken()
		}
		l.literalbuf.WriteByte(ch)
		l.pos++
	}

	if l.state == stateXB {
		l.Err = fmt.Errorf("unterminated bit string literal")
	} else {
		l.Err = fmt.Errorf("unterminated hexadecimal string literal")
	}
	l.state = stateInitial
	return Token{Type: lex_EOF, Loc: l.start}
}

// lexQuotedString handles '...' E'...' and U&'...' strings.
func (l *Lexer) lexQuotedString() Token {
	for l.pos < len(l.input) {
		ch := l.input[l.pos]

		// End quote - check for doubled quote first
		if ch == '\'' {
			// Check for doubled quote (escaped single quote)
			if l.pos+1 < len(l.input) && l.input[l.pos+1] == '\'' {
				l.literalbuf.WriteByte('\'')
				l.pos += 2
				continue
			}
			// Single quote - end of string, check for continuation
			l.pos++
			l.stateBeforeStrStop = l.state
			l.state = stateXQS
			return l.NextToken()
		}

		// For extended strings, handle escape sequences
		if l.state == stateXE && ch == '\\' && l.pos+1 < len(l.input) {
			l.pos++
			l.handleEscapeSequence()
			if l.Err != nil {
				return Token{Type: lex_EOF, Loc: l.start}
			}
			if l.state == stateXEU {
				return l.lexUnicodeSurrogate()
			}
			continue
		}

		l.literalbuf.WriteByte(ch)
		l.pos++
	}

	l.Err = fmt.Errorf("unterminated quoted string")
	l.state = stateInitial
	return Token{Type: lex_EOF, Loc: l.start}
}

// handleEscapeSequence processes escape sequences in E'...' strings.
func (l *Lexer) handleEscapeSequence() {
	if l.pos >= len(l.input) {
		return
	}

	ch := l.input[l.pos]
	l.pos++

	switch ch {
	case 'b':
		l.literalbuf.WriteByte('\b')
	case 'f':
		l.literalbuf.WriteByte('\f')
	case 'n':
		l.literalbuf.WriteByte('\n')
	case 'r':
		l.literalbuf.WriteByte('\r')
	case 't':
		l.literalbuf.WriteByte('\t')
	case 'v':
		l.literalbuf.WriteByte('\v')
	case '0', '1', '2', '3', '4', '5', '6', '7':
		// Octal escape: \0 through \777
		l.pos--
		val := l.scanOctal()
		l.literalbuf.WriteByte(byte(val))
	case 'x':
		// Hex escape: \xNN
		if l.pos < len(l.input) && isHexDigit(l.input[l.pos]) {
			val := l.scanHex(2)
			l.literalbuf.WriteByte(byte(val))
		} else {
			l.literalbuf.WriteByte('x')
		}
	case 'u', 'U':
		digits := 4
		if ch == 'U' {
			digits = 8
		}
		value, err := l.scanUnicodeEscape(digits)
		if err != nil {
			l.Err = err
			return
		}
		if value >= 0xD800 && value <= 0xDBFF {
			l.utf16FirstPart = value
			l.state = stateXEU
		} else {
			l.writeUnicodeChar(value)
		}
	default:
		l.literalbuf.WriteByte(ch)
	}
}

// scanOctal scans an octal escape sequence.
func (l *Lexer) scanOctal() int {
	val := 0
	for i := 0; i < 3 && l.pos < len(l.input); i++ {
		ch := l.input[l.pos]
		if ch < '0' || ch > '7' {
			break
		}
		val = val*8 + int(ch-'0')
		l.pos++
	}
	return val
}

// scanHex scans n hex digits.
func (l *Lexer) scanHex(n int) int {
	val := 0
	for i := 0; i < n && l.pos < len(l.input); i++ {
		ch := l.input[l.pos]
		if !isHexDigit(ch) {
			break
		}
		val = val*16 + hexValue(ch)
		l.pos++
	}
	return val
}

func (l *Lexer) scanUnicodeEscape(digits int) (rune, error) {
	start := l.pos
	value := l.scanHex(digits)
	if l.pos-start != digits || value <= 0 || value > utf8.MaxRune {
		return 0, fmt.Errorf("invalid Unicode escape sequence")
	}
	return rune(value), nil
}

// writeUnicodeChar writes a Unicode code point to the literal buffer.
func (l *Lexer) writeUnicodeChar(r rune) {
	if !validCodePoint(int64(r)) {
		l.Err = fmt.Errorf("invalid Unicode escape sequence")
		return
	}
	var buf [4]byte
	n := utf8.EncodeRune(buf[:], r)
	l.literalbuf.Write(buf[:n])
}

// lexQuoteContinue handles quote continuation (string literal spanning lines).
func (l *Lexer) lexQuoteContinue() Token {
	// Check if there's whitespace with newline followed by quote
	// SQL requires at least one newline in the whitespace to continue a string
	hasNewline := false
	if l.stateBeforeStrStop == stateXUI {
		goto done
	}

	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		switch {
		case ch == ' ' || ch == '\t' || ch == '\f' || ch == '\v':
			l.pos++
		case ch == '\n' || ch == '\r':
			hasNewline = true
			l.pos++
		case ch == '-' && l.pos+1 < len(l.input) && l.input[l.pos+1] == '-':
			// SQL comment - after comment there's an implicit newline
			l.skipLineComment()
			hasNewline = true
		default:
			goto done
		}
	}
done:

	// If we see a quote after whitespace with newline, continue the string
	if hasNewline && l.pos < len(l.input) && l.input[l.pos] == '\'' {
		l.pos++
		l.state = l.stateBeforeStrStop
		return l.NextToken()
	}

	// No continuation - return the completed string
	// Don't rewind position, just return from current spot
	l.state = stateInitial

	str := l.literalbuf.String()

	switch l.stateBeforeStrStop {
	case stateXB:
		return Token{Type: lex_BCONST, Str: str, Loc: l.start}
	case stateXH:
		return Token{Type: lex_XCONST, Str: str, Loc: l.start}
	case stateXQ, stateXE:
		if strings.IndexByte(str, 0) >= 0 || !utf8.ValidString(str) {
			l.Err = fmt.Errorf("invalid UTF-8 string literal")
			return Token{Type: lex_EOF, Loc: l.start}
		}
		return Token{Type: lex_SCONST, Str: str, Loc: l.start}
	case stateXUS:
		// Unicode string: U&'...'
		// Check for UESCAPE clause
		escapeChar, err := l.scanUESCAPE()
		if err != nil {
			l.Err = err
			return Token{Type: lex_EOF, Loc: l.start}
		}

		// Decode the string using the determined escape char
		decoded, err := l.decodeUnicodeString(str, escapeChar)
		if err != nil {
			l.Err = err
			return Token{Type: lex_EOF, Loc: l.start}
		}

		return Token{Type: lex_USCONST, Str: decoded, Loc: l.start}
	case stateXUI:
		// Unicode identifier: U&"..."
		// Check for UESCAPE clause
		escapeChar, err := l.scanUESCAPE()
		if err != nil {
			l.Err = err
			return Token{Type: lex_EOF, Loc: l.start}
		}

		// Decode the string using the determined escape char
		decoded, err := l.decodeUnicodeString(str, escapeChar)
		if err != nil {
			l.Err = err
			return Token{Type: lex_EOF, Loc: l.start}
		}

		if decoded == "" {
			l.Err = fmt.Errorf("zero-length delimited identifier")
			return Token{Type: lex_EOF, Loc: l.start}
		}
		return Token{Type: lex_IDENT, Str: truncateIdentifier(decoded), Loc: l.start}
	default:
		return Token{Type: lex_SCONST, Str: str, Loc: l.start}
	}
}

// lexUnicodeSurrogate handles Unicode surrogate pairs in E'...' strings.
func (l *Lexer) lexUnicodeSurrogate() Token {
	// Looking for the second part of a surrogate pair
	if l.pos+1 < len(l.input) && l.input[l.pos] == '\\' {
		l.pos++
		ch := l.input[l.pos]
		if ch == 'u' || ch == 'U' {
			l.pos++
			n := 4
			if ch == 'U' {
				n = 8
			}
			val, err := l.scanUnicodeEscape(n)
			if err == nil && isUTF16SurrogateSecond(val) {
				combined := surrogateToCodepoint(l.utf16FirstPart, val)
				l.writeUnicodeChar(combined)
				l.state = stateXE
				return l.NextToken()
			}
		}
	}

	l.Err = fmt.Errorf("invalid Unicode surrogate pair")
	l.state = stateInitial
	return Token{Type: lex_EOF, Loc: l.start}
}

// lexDelimitedIdent handles "..." identifiers.
func (l *Lexer) lexDelimitedIdent() Token {
	for l.pos < len(l.input) {
		ch := l.input[l.pos]

		// End quote
		if ch == '"' {
			l.pos++
			// Check for "" (escaped quote)
			if l.pos < len(l.input) && l.input[l.pos] == '"' {
				l.literalbuf.WriteByte('"')
				l.pos++
				continue
			}
			// End of identifier
			oldState := l.state
			l.state = stateInitial

			// If it's a Unicode identifier (U&"..."), we need to handle optional UESCAPE
			if oldState == stateXUI {
				l.stateBeforeStrStop = stateXUI
				return l.lexQuoteContinue()
			}

			str := l.literalbuf.String()
			if str == "" {
				l.Err = fmt.Errorf("zero-length delimited identifier")
				return Token{Type: lex_EOF, Loc: l.start}
			}
			return Token{Type: lex_IDENT, Str: truncateIdentifier(str), Loc: l.start}
		}

		l.literalbuf.WriteByte(ch)
		l.pos++
	}

	l.Err = fmt.Errorf("unterminated quoted identifier")
	l.state = stateInitial
	return Token{Type: lex_EOF, Loc: l.start}
}

// lexDollarQuote handles $tag$...$tag$ strings.
func (l *Lexer) lexDollarQuote() Token {
	for l.pos < len(l.input) {
		ch := l.input[l.pos]

		if ch == '$' {
			// Check for ending delimiter
			endTag := l.peekDollarTag()
			if endTag == l.dolqstart {
				l.pos += len(endTag)
				l.state = stateInitial
				return Token{Type: lex_SCONST, Str: l.literalbuf.String(), Loc: l.start}
			}
		}

		l.literalbuf.WriteByte(ch)
		l.pos++
	}

	l.Err = fmt.Errorf("unterminated dollar-quoted string")
	l.state = stateInitial
	return Token{Type: lex_EOF, Loc: l.start}
}

// scanDollarTag scans and returns a dollar tag ($...$) if present.
func (l *Lexer) scanDollarTag() string {
	if l.pos >= len(l.input) || l.input[l.pos] != '$' {
		return ""
	}

	start := l.pos
	l.pos++ // skip first $

	// Tag can be empty ($$ ... $$) or have identifier chars
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == '$' {
			l.pos++
			return l.input[start:l.pos]
		}
		if l.pos == start+1 {
			// First char must be letter or underscore
			if !isIdentStart(ch) {
				l.pos = start
				return ""
			}
		} else {
			// Subsequent chars can include digits
			if !isIdentCont(ch) {
				l.pos = start
				return ""
			}
		}
		l.pos++
	}

	l.pos = start
	return ""
}

// peekDollarTag peeks at a dollar tag without advancing position.
func (l *Lexer) peekDollarTag() string {
	savedPos := l.pos
	tag := l.scanDollarTag()
	l.pos = savedPos
	return tag
}

// lexParam handles parameter tokens ($1, $2, etc.)
func (l *Lexer) lexParam() Token {
	l.pos++ // skip $
	start := l.pos
	for l.pos < len(l.input) && isDigit(l.input[l.pos]) {
		l.pos++
	}

	// Check for trailing junk
	if l.pos < len(l.input) && isIdentStart(l.input[l.pos]) {
		l.Err = fmt.Errorf("trailing junk after parameter")
		return Token{Type: lex_EOF, Loc: l.start}
	}

	val, err := strconv.ParseInt(l.input[start:l.pos], 10, 32)
	if err != nil || val == 0 {
		l.Err = fmt.Errorf("parameter number must be between 1 and 2147483647")
		return Token{Type: lex_EOF, Loc: l.start}
	}
	return Token{Type: lex_PARAM, Ival: val, Loc: l.start}
}

// lexOperator handles operator tokens.
func (l *Lexer) lexOperator() Token {
	start := l.pos
	if start >= l.operatorEnd {
		end := start
		special := false
		for end < len(l.input) && isOpChar(l.input[end]) {
			ch := l.input[end]
			if end+1 < len(l.input) && ((ch == '/' && l.input[end+1] == '*') ||
				(ch == '-' && l.input[end+1] == '-')) {
				break
			}
			special = special || strings.ContainsRune("~!@#^&|`?%", rune(ch))
			end++
		}
		l.operatorEnd = end
		if !special {
			for end > start+1 && (l.input[end-1] == '+' || l.input[end-1] == '-') {
				end--
			}
		}
		l.operatorBodyEnd = end
	}
	l.pos = max(start+1, l.operatorBodyEnd)
	op := l.input[start:l.pos]

	// Single-char ops that should be returned as their char value
	if len(op) == 1 && isSelf(op[0]) {
		return Token{Type: int(op[0]), Str: op, Loc: l.start}
	}

	// Two-char special tokens
	if len(op) == 2 {
		switch op {
		case "=>":
			return Token{Type: lex_EQUALS_GREATER, Str: op, Loc: l.start}
		case ">=":
			return Token{Type: lex_GREATER_EQUALS, Str: op, Loc: l.start}
		case "<=":
			return Token{Type: lex_LESS_EQUALS, Str: op, Loc: l.start}
		case "<>":
			return Token{Type: lex_NOT_EQUALS, Str: op, Loc: l.start}
		case "!=":
			return Token{Type: lex_NOT_EQUALS, Str: op, Loc: l.start}
		}
	}

	if len(op) >= 63 { // NAMEDATALEN
		l.Err = fmt.Errorf("operator too long")
		return Token{Type: lex_EOF, Loc: l.start}
	}

	return Token{Type: lex_Op, Str: op, Loc: l.start}
}

// lexNumber handles integer and floating-point literals.
func (l *Lexer) lexNumber() Token {
	start := l.pos

	// Check for hex, octal, or binary prefix
	if l.input[l.pos] == '0' && l.pos+1 < len(l.input) {
		next := l.input[l.pos+1]
		if next == 'x' || next == 'X' {
			return l.lexBaseNumber(16)
		}
		if next == 'o' || next == 'O' {
			return l.lexBaseNumber(8)
		}
		if next == 'b' || next == 'B' {
			return l.lexBaseNumber(2)
		}
	}

	// Decimal integer or numeric
	if !l.scanNumberDigits(10) {
		l.Err = fmt.Errorf("invalid numeric separator")
		return Token{Type: lex_EOF, Loc: l.start}
	}

	isFloat := false

	// Check for decimal point
	if l.pos < len(l.input) && l.input[l.pos] == '.' {
		// Check for .. (DOT_DOT)
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == '.' {
			// Return the integer part, let .. be handled separately
			goto done
		}
		l.pos++
		if !l.scanNumberDigits(10) {
			l.Err = fmt.Errorf("invalid numeric separator")
			return Token{Type: lex_EOF, Loc: l.start}
		}
		isFloat = true
	}

	// Check for exponent
	if l.pos < len(l.input) && (l.input[l.pos] == 'e' || l.input[l.pos] == 'E') {
		l.pos++
		if l.pos < len(l.input) && (l.input[l.pos] == '+' || l.input[l.pos] == '-') {
			l.pos++
		}
		if l.pos >= len(l.input) || !isDigit(l.input[l.pos]) {
			l.Err = fmt.Errorf("trailing junk after numeric literal")
			return Token{Type: lex_EOF, Loc: l.start}
		}
		if !l.scanNumberDigits(10) {
			l.Err = fmt.Errorf("invalid numeric separator")
			return Token{Type: lex_EOF, Loc: l.start}
		}
		isFloat = true
	}

done:
	// Check for trailing identifier
	if l.pos < len(l.input) && isIdentStart(l.input[l.pos]) {
		l.Err = fmt.Errorf("trailing junk after numeric literal")
		return Token{Type: lex_EOF, Loc: l.start}
	}

	numStr := l.input[start:l.pos]
	// Remove underscores for parsing
	numStr = strings.ReplaceAll(numStr, "_", "")

	if isFloat {
		return Token{Type: lex_FCONST, Str: numStr, Loc: l.start}
	}

	// Try to parse as integer
	val, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		// Too large, treat as float
		return Token{Type: lex_FCONST, Str: numStr, Loc: l.start}
	}

	return Token{Type: lex_ICONST, Ival: val, Str: numStr, Loc: l.start}
}

func (l *Lexer) scanNumberDigits(base int) bool {
	start := l.pos
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		switch {
		case isHexDigit(ch) && hexValue(ch) < base:
			l.pos++
		case ch == '_':
			if l.pos == start || l.pos+1 >= len(l.input) ||
				!isHexDigit(l.input[l.pos+1]) || hexValue(l.input[l.pos+1]) >= base {
				return false
			}
			l.pos++
		default:
			return true
		}
	}
	return true
}

func (l *Lexer) lexBaseNumber(base int) Token {
	start := l.pos
	l.pos += 2
	// PostgreSQL permits one separator immediately after a base prefix.
	if l.pos < len(l.input) && l.input[l.pos] == '_' {
		l.pos++
	}
	if l.pos >= len(l.input) || !isHexDigit(l.input[l.pos]) || hexValue(l.input[l.pos]) >= base ||
		!l.scanNumberDigits(base) {
		l.Err = fmt.Errorf("invalid base-%d integer", base)
		return Token{Type: lex_EOF, Loc: l.start}
	}

	if l.pos < len(l.input) && isIdentCont(l.input[l.pos]) {
		l.Err = fmt.Errorf("trailing junk after numeric literal")
		return Token{Type: lex_EOF, Loc: l.start}
	}

	numStr := l.input[start:l.pos]
	numStr = strings.ReplaceAll(numStr, "_", "")

	val, err := strconv.ParseInt(numStr[2:], base, 64)
	if err != nil {
		return Token{Type: lex_FCONST, Str: numStr, Loc: l.start}
	}

	return Token{Type: lex_ICONST, Ival: val, Str: numStr, Loc: l.start}
}

// lexIdent handles identifiers and keywords.
func (l *Lexer) lexIdent() Token {
	start := l.pos

	// Scan identifier
	for l.pos < len(l.input) && isIdentCont(l.input[l.pos]) {
		l.pos++
	}

	ident := l.downcase(l.input[start:l.pos])

	// Keywords are stored lowercase, so one lowering serves both lookups.
	if kw := lookupKeywordLower(ident); kw != nil {
		return Token{Type: kw.Token, Str: kw.Name, Loc: l.start}
	}

	return Token{Type: lex_IDENT, Str: truncateIdentifier(ident), Loc: l.start}
}

func truncateIdentifier(ident string) string {
	if len(ident) <= maxIdentifierBytes {
		return ident
	}
	cut := maxIdentifierBytes
	for cut > 0 && !utf8.RuneStart(ident[cut]) {
		cut--
	}
	return ident[:cut]
}

// downcase converts an identifier to lowercase.
func (l *Lexer) downcase(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			buf := []byte(s)
			for j := i; j < len(buf); j++ {
				if buf[j] >= 'A' && buf[j] <= 'Z' {
					buf[j] += 'a' - 'A'
				}
			}
			return string(buf)
		}
	}
	return s
}

// Character classification functions

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isHexDigit(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func hexValue(ch byte) int {
	if ch >= '0' && ch <= '9' {
		return int(ch - '0')
	}
	if ch >= 'a' && ch <= 'f' {
		return int(ch - 'a' + 10)
	}
	return int(ch - 'A' + 10)
}

func isIdentStart(ch byte) bool {
	// A-Z, a-z, _, and high bytes (for UTF-8)
	return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || ch == '_' || ch >= 128
}

func isIdentCont(ch byte) bool {
	// A-Z, a-z, 0-9, _, $, and high bytes
	return isIdentStart(ch) || (ch >= '0' && ch <= '9') || ch == '$'
}

func isSelf(ch byte) bool {
	// Characters that are self-delimiting tokens
	return ch == ',' || ch == '(' || ch == ')' || ch == '[' || ch == ']' ||
		ch == '.' || ch == ';' || ch == ':' || ch == '+' || ch == '-' ||
		ch == '*' || ch == '/' || ch == '%' || ch == '^' || ch == '<' ||
		ch == '>' || ch == '='
}

func isSelfOnly(ch byte) bool {
	// Characters that are ONLY self-delimiting (not part of operators)
	return ch == ',' || ch == '(' || ch == ')' || ch == '[' || ch == ']' ||
		ch == '.' || ch == ';' || ch == ':'
}

func isOpChar(ch byte) bool {
	// Characters that can appear in operators
	return ch == '~' || ch == '!' || ch == '@' || ch == '#' || ch == '^' ||
		ch == '&' || ch == '|' || ch == '`' || ch == '?' || ch == '+' ||
		ch == '-' || ch == '*' || ch == '/' || ch == '%' || ch == '<' ||
		ch == '>' || ch == '='
}

// Unicode surrogate pair handling

func isUTF16SurrogateSecond(r rune) bool {
	return r >= 0xDC00 && r <= 0xDFFF
}

func surrogateToCodepoint(high, low rune) rune {
	return ((high - 0xD800) << 10) + (low - 0xDC00) + 0x10000
}

func isSpaceByte(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '\v'
}

func (l *Lexer) scanUESCAPE() (rune, error) {
	savedPos := l.pos
	if err := l.skipWhitespaceAndComments(); err != nil {
		return 0, err
	}
	const keyword = "UESCAPE"
	end := l.pos + len(keyword)
	if end > len(l.input) || !strings.EqualFold(l.input[l.pos:end], keyword) ||
		(end < len(l.input) && isIdentCont(l.input[end])) {
		l.pos = savedPos
		return '\\', nil
	}
	l.pos = end
	if err := l.skipWhitespaceAndComments(); err != nil {
		return 0, err
	}
	if l.pos+3 > len(l.input) || l.input[l.pos] != '\'' || l.input[l.pos+2] != '\'' {
		return 0, fmt.Errorf("invalid Unicode escape character")
	}
	ch := l.input[l.pos+1]
	if ch >= utf8.RuneSelf || isHexDigit(ch) || isSpaceByte(ch) || ch == '+' || ch == '\'' || ch == '"' {
		return 0, fmt.Errorf("invalid Unicode escape character")
	}
	l.pos += 3
	return rune(ch), nil
}

// skipWhitespaceAndComments skips whitespace and comments (both -- and /* */).
func (l *Lexer) skipWhitespaceAndComments() error {
	for l.pos < len(l.input) {
		ch := l.input[l.pos]

		if isSpaceByte(ch) {
			l.pos++
			continue
		}

		if ch == '-' && l.pos+1 < len(l.input) && l.input[l.pos+1] == '-' {
			l.skipLineComment()
			continue
		}

		if ch == '/' && l.pos+1 < len(l.input) && l.input[l.pos+1] == '*' {
			l.pos += 2
			if !l.skipBlockComment() {
				return fmt.Errorf("unterminated /* comment")
			}
			continue
		}

		break
	}
	return nil
}

// decodeUnicodeString decodes a U& string with the given escape character.
func (l *Lexer) decodeUnicodeString(s string, escape rune) (string, error) {
	var buf strings.Builder
	runes := []rune(s)
	n := len(runes)
	var high rune

	for i := 0; i < n; i++ {
		r := runes[i]

		if r == escape {
			if i+1 >= n {
				return "", fmt.Errorf("invalid Unicode escape sequence at end of string")
			}

			next := runes[i+1]

			// Escaped escape char
			if next == escape {
				if high != 0 {
					return "", fmt.Errorf("invalid Unicode surrogate pair")
				}
				buf.WriteRune(escape)
				i++
				continue
			}

			start, digits := i+1, 4
			if next == '+' {
				start, digits = i+2, 6
			}
			if start+digits > n {
				return "", fmt.Errorf("invalid Unicode escape sequence")
			}
			var value rune
			for _, digit := range runes[start : start+digits] {
				if digit > 127 || !isHexDigit(byte(digit)) {
					return "", fmt.Errorf("invalid Unicode escape sequence")
				}
				value = value*16 + rune(hexValue(byte(digit)))
			}
			i = start + digits - 1
			if high != 0 {
				if !isUTF16SurrogateSecond(value) {
					return "", fmt.Errorf("invalid Unicode surrogate pair")
				}
				value = surrogateToCodepoint(high, value)
				high = 0
			} else if value >= 0xD800 && value <= 0xDBFF {
				high = value
				continue
			}
			if !validCodePoint(int64(value)) {
				return "", fmt.Errorf("invalid Unicode escape sequence")
			}
			buf.WriteRune(value)
			continue
		}
		if high != 0 {
			return "", fmt.Errorf("invalid Unicode surrogate pair")
		}
		buf.WriteRune(r)
	}
	if high != 0 {
		return "", fmt.Errorf("invalid Unicode surrogate pair")
	}
	return buf.String(), nil
}

// validCodePoint reports whether v is a Unicode scalar value.  Surrogates and
// values above U+10FFFF are rejected rather than silently replaced by U+FFFD.
func validCodePoint(v int64) bool {
	return v > 0 && v <= utf8.MaxRune && (v < 0xD800 || v > 0xDFFF)
}

// lineAt returns the line number of the given position.
func (l *Lexer) lineAt(pos int) int {
	if pos > len(l.input) {
		pos = len(l.input)
	}
	if l.newlineIdx == nil {
		l.newlineIdx = make([]int, 0, strings.Count(l.input, "\n"))
		for i := 0; i < len(l.input); i++ {
			if l.input[i] == '\n' {
				l.newlineIdx = append(l.newlineIdx, i)
			}
		}
	}
	// Number of newlines before pos, plus one.
	return 1 + sort.SearchInts(l.newlineIdx, pos)
}

// contextAt returns the text around the given position.
func (l *Lexer) contextAt(pos int) string {
	if pos > len(l.input) {
		pos = len(l.input)
	}
	start := max(pos-16, 0)
	// Do not begin in the middle of a word: back up to the previous space or
	// self-delimiting character, bounded so a long token cannot bloat the message.
	for start > 0 && start > pos-32 && !isSpaceByte(l.input[start-1]) && !isSelf(l.input[start-1]) {
		start--
	}
	for start < pos && !utf8.RuneStart(l.input[start]) {
		start++
	}
	end := min(pos+16, len(l.input))
	for end < len(l.input) && !utf8.RuneStart(l.input[end]) {
		end++
	}
	return strings.TrimSpace(l.input[start:end])
}
