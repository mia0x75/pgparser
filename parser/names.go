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
	case 'u', 'U':
		// Unicode-escaped identifiers are double quoted after the U& prefix.
		if strings.HasPrefix(ident.Raw[1:], "&\"") {
			ident.Quote = lex.QuoteDouble
		}
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

// syntaxIdents converts a name list into identifiers with spelling.
func syntaxIdents(l yyLexer, list *ast.List) []lex.Ident {
	if list == nil {
		return nil
	}
	idents := make([]lex.Ident, 0, len(list.Items))
	for _, item := range list.Items {
		if name, ok := item.(*ast.String); ok {
			idents = append(idents, syntaxIdent(l, name.Str, name.Span()))
		}
	}
	return idents
}

// syntaxQualifiedNames converts a name list into qualified names with spelling.
func syntaxQualifiedNames(l yyLexer, list *ast.List) []ast.QualifiedName {
	idents := syntaxIdents(l, list)
	if idents == nil {
		return nil
	}
	names := make([]ast.QualifiedName, 0, len(idents))
	for _, ident := range idents {
		names = append(names, syntaxName(ident))
	}
	return names
}

// lastNameIdent returns the last part of a qualified name.
func lastNameIdent(name ast.QualifiedName) lex.Ident {
	if len(name.Parts) == 0 {
		return lex.Ident{Span: loc.InvalidSpan()}
	}
	return name.Parts[len(name.Parts)-1]
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

func syntaxTableNames(l yyLexer, list *ast.List) []*ast.TableName {
	if list == nil {
		return nil
	}
	names := make([]*ast.TableName, 0, len(list.Items))
	for _, item := range list.Items {
		switch name := item.(type) {
		case *ast.List:
			names = append(names, syntaxTableName(syntaxName(syntaxIdents(l, name)...)))
		case *ast.String:
			names = append(names, syntaxTableName(syntaxName(syntaxIdent(l, name.Str, name.Span()))))
		case *ast.DataType:
			names = append(names, syntaxTableName(nameFromString(name.Type.String())))
		}
	}
	return names
}

func syntaxSelectItem(expr ast.ExprNode, alias lex.Ident, span loc.Span) *ast.SelectItem {
	item := &ast.SelectItem{Expr: expr, Alias: alias}
	item.SetSpan(span)
	return item
}

func syntaxWildcard(qualifier ast.QualifiedName, span loc.Span) *ast.WildcardExpr {
	wildcard := &ast.WildcardExpr{Qualifier: qualifier}
	wildcard.SetSpan(span)
	return wildcard
}

func indexNameFromTableName(table *ast.TableName) ast.QualifiedName {
	if table == nil {
		return ast.QualifiedName{}
	}
	return table.Name
}

func syntaxTableSource(source ast.ResultSetNode, span loc.Span) *ast.TableSource {
	table := &ast.TableSource{BaseTableSource: pkast.BaseTableSource{
		Source: source,
		Alias:  lex.Ident{Span: loc.InvalidSpan()},
	}}
	table.SetSpan(span)
	return table
}

func syntaxFromClause(table ast.ResultSetNode) *ast.FromClause {
	clause := &ast.FromClause{BaseFromClause: pkast.BaseFromClause{Table: table}}
	clause.SetSpan(table.Span())
	return clause
}

type sourceAlias struct {
	name    lex.Ident
	columns []lex.Ident
	defs    []*ast.ColumnDef
}

func syntaxSourceAlias(l yyLexer, name string, span loc.Span, columns *ast.List) *sourceAlias {
	alias := &sourceAlias{name: syntaxIdent(l, name, span)}
	if columns != nil {
		alias.columns = make([]lex.Ident, 0, len(columns.Items))
		for _, item := range columns.Items {
			if column, ok := item.(*ast.String); ok {
				alias.columns = append(alias.columns, syntaxIdent(l, column.Str, column.Span()))
			}
		}
	}
	return alias
}

func applySourceAlias(table *ast.TableSource, alias *sourceAlias) {
	if alias != nil {
		table.Alias = alias.name
		table.Columns = alias.columns
		table.ColDefs = alias.defs
	}
}
