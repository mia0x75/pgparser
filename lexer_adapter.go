package parser

import (
	"fmt"

	"github.com/mia0x75/pgparser/ast"
)

// Scanner adapts the PostgreSQL lexer to the goyacc-generated yyLexer
// interface. It also implements the one-token lookahead logic used by
// PostgreSQL's parser to disambiguate NOT, WITH, NULLS, WITHOUT and FORMAT.
type Scanner struct {
	lexer    *Lexer
	result   *ast.List
	errs     []error
	warnings []error

	haveLookahead    bool
	lookaheadToken   Token
	lookaheadTokType int

	// params counts the placeholder parameters of the parsed statement.
	params int
}

func (s *Scanner) reset(input string) {
	s.lexer = NewLexer(input)
	s.params = 0
	s.result = nil
	s.errs = nil
	s.warnings = nil
	s.haveLookahead = false
	s.lookaheadToken = Token{}
	s.lookaheadTokType = 0
}

func (s *Scanner) Lex(lval *yySymType) int {
	var tok Token
	var tokType int
	if s.haveLookahead {
		tok = s.lookaheadToken
		tokType = s.lookaheadTokType
		s.haveLookahead = false
	} else {
		tok = s.lexer.NextToken()
		tokType = s.mapTokenType(tok)
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
		tokType = s.applyLookahead(tokType, nextTokType)
	}
	switch tokType {
	case IDENT:
		lval.str = tok.Str
	case ICONST:
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
	s.errs = append(s.errs, &ParseError{
		Message:  fmt.Sprintf("syntax error at line %d near %q", line, near),
		Position: s.lexer.pos,
	})
}

func (s *Scanner) Errors() ([]error, []error) {
	return s.warnings, s.errs
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
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("%s at byte %d", e.Message, e.Position)
}
