package parser

import (
	"strings"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/pgparser/ast"
)

func syntaxIdent(l yyLexer, name string, span loc.Span) lex.Ident {
	ident := lex.Ident{Name: name, Span: loc.InvalidSpan()}
	scanner, ok := l.(*Scanner)
	if !ok || scanner.lexer == nil {
		return ident
	}
	src := scanner.lexer.input
	if !span.Within(len(src)) || span.Empty() {
		return ident
	}
	ident.Span = span
	ident.Raw = src[span.Start:span.End]
	switch ident.Raw[0] {
	case '`':
		ident.Quote = lex.QuoteBacktick
	case '"':
		ident.Quote = lex.QuoteDouble
	case '\'':
		ident.Quote = lex.QuoteSingle
	}
	if ident.Quote != lex.QuoteUnquoted && len(ident.Raw) >= 2 {
		body := ident.Raw[1 : len(ident.Raw)-1]
		if body == name && ident.Raw[len(ident.Raw)-1] == ident.Raw[0] {
			ident.Name = body
		}
	}
	return ident
}

func syntaxName(parts ...lex.Ident) ast.QualifiedName {
	name := ast.QualifiedName{Parts: parts, Span: loc.InvalidSpan()}
	for _, part := range parts {
		name.Span = loc.Cover(name.Span, part.Span)
	}
	return name
}

func nameFromString(name string) ast.QualifiedName {
	return nameFromParts(name)
}

func nameFromParts(parts ...string) ast.QualifiedName {
	idents := make([]lex.Ident, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			idents = append(idents, lex.Ident{Name: part})
		}
	}
	return syntaxName(idents...)
}

func nameString(name ast.QualifiedName) string {
	return strings.Join(nameStrings(name), ".")
}

func nameStrings(name ast.QualifiedName) []string {
	parts := make([]string, 0, len(name.Parts))
	for _, part := range name.Parts {
		parts = append(parts, part.Name)
	}
	return parts
}

func syntaxTableName(name ast.QualifiedName) *ast.TableName {
	table := &ast.TableName{BaseTableRef: pkast.BaseTableRef{Name: name}}
	table.SetSpan(name.Span)
	return table
}

func indexNameFromTableName(table *ast.TableName) string {
	if table == nil || len(table.Name.Parts) == 0 {
		return ""
	}
	return table.Name.Parts[len(table.Name.Parts)-1].Name
}
