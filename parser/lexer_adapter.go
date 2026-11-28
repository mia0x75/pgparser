package parser

import (
	"fmt"
	"strings"

	"github.com/mia0x75/parserkit/diagnostic"
	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/pgparser/postgres"
)

// Scanner adapts the PostgreSQL lexer to the goyacc-generated yyLexer
// interface. It also implements the one-token lookahead logic used by
// PostgreSQL's parser to disambiguate NOT, WITH, NULLS, WITHOUT and FORMAT.
type Scanner struct {
	lexer        *Lexer
	parser       *parseState
	errs         []error
	stmtStartPos int

	haveLookahead    bool
	lookaheadToken   Token
	lookaheadTokType int
	lookaheadSpan    loc.Span
}

func (s *Scanner) reset(input string) {
	s.lexer = NewLexer(input)
	s.parser = nil
	s.errs = nil
	s.stmtStartPos = 0
	s.haveLookahead = false
	s.lookaheadToken = Token{}
	s.lookaheadTokType = 0
	s.lookaheadSpan = loc.InvalidSpan()
}

func (s *Scanner) stmtText() string {
	endPos := s.lexer.pos
	if s.haveLookahead && s.lookaheadTokType != ';' && s.lookaheadTokType != 0 {
		// A prefetched token can already belong to the next statement.
		endPos = int(s.lookaheadSpan.Start)
	}
	if s.stmtStartPos >= endPos {
		s.stmtStartPos = endPos
		return ""
	}
	if s.lexer.input[endPos-1] == '\n' {
		endPos--
	}
	if s.stmtStartPos < endPos && s.lexer.input[s.stmtStartPos] == '\n' {
		s.stmtStartPos++
	}
	text := s.lexer.input[s.stmtStartPos:endPos]
	s.stmtStartPos = endPos
	return text
}

func (s *Scanner) Lex(lval *yySymType) int {
	var tok Token
	var tokType int
	if s.haveLookahead {
		tok = s.lookaheadToken
		tokType = s.lookaheadTokType
		s.haveLookahead = false
		lval.span = s.lookaheadSpan
	} else {
		tok = s.lexer.NextToken()
		tokType = s.mapTokenType(tok)
		lval.span = loc.Span{Start: loc.ByteOffset(s.lexer.start), End: loc.ByteOffset(s.lexer.pos)}
	}
	if s.lexer.Err != nil {
		s.Error(s.lexer.Err.Error())
		s.lexer.Err = nil
		return 0
	}
	if s.needsLookahead(tokType) {
		nextTok := s.lexer.NextToken()
		nextTokType := s.mapTokenType(nextTok)
		s.haveLookahead = true
		s.lookaheadToken = nextTok
		s.lookaheadTokType = nextTokType
		s.lookaheadSpan = loc.Span{Start: loc.ByteOffset(s.lexer.start), End: loc.ByteOffset(s.lexer.pos)}
		tokType = s.applyLookahead(tokType, nextTokType)
	}
	switch tokType {
	case IDENT:
		lval.str = tok.Str
	case ICONST, PARAM:
		lval.ival = tok.Ival
	case FCONST, SCONST, BCONST, XCONST:
		lval.str = tok.Str
	case Op:
		lval.str = tok.Str
	default:
		if tok.Str != "" {
			lval.str = tok.Str
		}
	}
	return tokType
}

func (s *Scanner) needsLookahead(tokType int) bool {
	switch tokType {
	case NOT, WITH, NULLS_P, WITHOUT, FORMAT:
		return true
	}
	return false
}

func (s *Scanner) applyLookahead(curToken, nextToken int) int {
	switch curToken {
	case NOT:
		switch nextToken {
		case BETWEEN, IN_P, LIKE, ILIKE, SIMILAR:
			return NOT_LA
		}
	case WITH:
		switch nextToken {
		case TIME, ORDINALITY:
			return WITH_LA
		}
	case NULLS_P:
		switch nextToken {
		case FIRST_P, LAST_P, DISTINCT, NOT:
			return NULLS_LA
		}
	case FORMAT:
		if nextToken == JSON {
			return FORMAT_LA
		}
	case WITHOUT:
		if nextToken == TIME {
			return WITHOUT_LA
		}
	}
	return curToken
}

func (s *Scanner) Error(msg string) {
	line := s.lexer.lineAt(s.lexer.pos)
	near := s.lexer.contextAt(s.lexer.pos)
	// Lexer and grammar-action errors arrive here already worded; keep that
	// wording but add the same position detail as a syntax error.
	span := loc.Span{Start: loc.ByteOffset(s.lexer.start), End: loc.ByteOffset(s.lexer.pos)}
	if !strings.HasPrefix(msg, "syntax error") {
		s.errs = append(s.errs, &ParseError{
			Message:  fmt.Sprintf("%s at line %d near %q", msg, line, near),
			Position: s.lexer.pos,
			Span:     span,
		})
		return
	}
	s.errs = append(s.errs, &ParseError{
		Message:  fmt.Sprintf("syntax error at line %d near %q", line, near),
		Position: s.lexer.pos,
		Span:     span,
	})
}

func (s *Scanner) Errors() []error {
	return s.errs
}

func (s *Scanner) mapTokenType(tok Token) int {
	if tok.Type == 0 {
		return 0
	}
	if tok.Type > 0 && tok.Type < 256 {
		return tok.Type
	}
	if tok.Type >= nonKeywordTokenBase && tok.Type < nonKeywordTokenBase+100 {
		switch tok.Type - nonKeywordTokenBase {
		case 0:
			return ICONST
		case 1:
			return FCONST
		case 2:
			return SCONST
		case 3:
			return BCONST
		case 4:
			return XCONST
		case 5:
			return SCONST
		case 6:
			return IDENT
		case 7:
			return IDENT
		case 8:
			return TYPECAST
		case 9:
			return DOT_DOT
		case 10:
			return COLON_EQUALS
		case 11:
			return EQUALS_GREATER
		case 12:
			return LESS_EQUALS
		case 13:
			return GREATER_EQUALS
		case 14:
			return NOT_EQUALS
		case 15:
			return PARAM
		case 16:
			return Op
		}
		return 0
	}
	return tok.Type
}

// ParseError represents a parse error with position information.
type ParseError struct {
	Message  string
	Position int
	Span     loc.Span
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("%s at byte %d", e.Message, e.Position)
}

// Diagnostic returns the parserkit diagnostic with the PG syntax SQLSTATE.
func (e *ParseError) Diagnostic() *diagnostic.Diagnostic {
	if e == nil {
		return nil
	}
	span := e.Span
	if !span.Valid() {
		span = loc.InvalidSpan()
	}
	return postgres.SyntaxDiagnostic(e.Message, span, e)
}
