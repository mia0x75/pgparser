package parser

import (
	"strings"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/auth"
	"github.com/mia0x75/pgparser/model"
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

// tableNameFromList returns the table name of a qualified name list.
func tableNameFromList(l *ast.List) *ast.TableName {
	name := &ast.TableName{}
	if l == nil {
		return name
	}
	parts := make([]lex.Ident, 0, len(l.Items))
	for _, item := range l.Items {
		if s, ok := item.(*ast.String); ok {
			parts = append(parts, lex.Ident{Name: s.Str})
		}
	}
	name.Name = syntaxName(parts...)
	return name
}

// tableNamesFromList returns the table names of a list.
func tableNamesFromList(l *ast.List) []*ast.TableName {
	if l == nil {
		return nil
	}
	var result []*ast.TableName
	for _, item := range l.Items {
		switch v := item.(type) {
		case *ast.TableName:
			result = append(result, v)
		case *ast.String:
			result = append(result, syntaxTableName(nameFromString(v.Str)))
		case *ast.List:
			result = append(result, tableNameFromList(v))
		case *ast.RangeVar:
			result = append(result, syntaxTableName(nameFromParts(v.Schemaname, v.Relname)))
		case *ast.TableSource:
			if name, ok := v.Source.(*ast.TableName); ok {
				result = append(result, name)
			}
		case *ast.VacuumRelation:
			result = append(result, tableNameFromList(nil))
		}
	}
	return result
}

// nameFromList returns the name of a qualified name list.
func nameFromList(l *ast.List) string {
	name := tableNameFromList(l)
	return nameString(name.Name)
}

// indexName returns the index name of a cluster index specification.
func indexName(name string) *ast.TableName {
	if name == "" {
		return nil
	}
	return syntaxTableName(nameFromString(name))
}

// roleIdentity returns the role identity of a role specification.
func roleIdentity(n ast.Node) *auth.RoleIdentity {
	if spec, ok := n.(*ast.RoleSpec); ok {
		return &auth.RoleIdentity{Username: spec.Rolename.Name}
	}
	return nil
}

// roleIdentities returns the role identities of a role list.
func roleIdentities(l *ast.List) []*auth.RoleIdentity {
	if l == nil {
		return nil
	}
	var result []*auth.RoleIdentity
	for _, item := range l.Items {
		switch v := item.(type) {
		case *ast.RoleSpec:
			result = append(result, &auth.RoleIdentity{Username: v.Rolename.Name})
		case *ast.String:
			result = append(result, &auth.RoleIdentity{Username: v.Str})
		}
	}
	return result
}

// relationName returns the table name of a relation expression.
func relationName(n ast.Node) *ast.TableName {
	switch v := n.(type) {
	case *ast.TableName:
		return v
	case *ast.TableSource:
		if name, ok := v.Source.(*ast.TableName); ok {
			return name
		}
	case *ast.RangeVar:
		table := syntaxTableName(nameFromParts(v.Catalogname, v.Schemaname, v.Relname))
		table.Relpersistence = v.Relpersistence
		return table
	}
	return nil
}

// tableNamesFromNameList returns the table names of a name list.
func tableNamesFromNameList(l *ast.List) []*ast.TableName {
	if l == nil {
		return nil
	}
	var result []*ast.TableName
	for _, item := range l.Items {
		switch v := item.(type) {
		case *ast.String:
			result = append(result, syntaxTableName(nameFromString(v.Str)))
		case *ast.DataType:
			result = append(result, syntaxTableName(nameFromString(v.Type.String())))
		case *ast.List:
			result = append(result, tableNameFromList(v))
		}
	}
	return result
}

// tableNameFromSource returns the table name of a table source.
func tableNameFromSource(n ast.Node) *ast.TableName {
	if src, ok := n.(*ast.TableSource); ok {
		if name, ok := src.Source.(*ast.TableName); ok {
			return name
		}
	}
	return nil
}

// objectWithArgsNames returns the names of an object list.
func objectWithArgsNames(l *ast.List) []*ast.TableName {
	if l == nil {
		return nil
	}
	var names []*ast.TableName
	for _, item := range l.Items {
		obj, ok := item.(*ast.ObjectWithArgs)
		if !ok {
			continue
		}
		var parts []string
		for _, name := range obj.Objname.Items {
			s, ok := name.(*ast.String)
			if !ok {
				continue
			}
			parts = append(parts, s.Str)
		}
		switch len(parts) {
		case 0:
			continue
		case 1:
			names = append(names, syntaxTableName(nameFromString(parts[0])))
		case 3:
			names = append(names, syntaxTableName(nameFromParts(parts[0], parts[1], parts[2])))
		default:
			names = append(names, syntaxTableName(nameFromParts(parts[0], parts[1])))
		}
	}
	return names
}

// tableNamesFromObjectWithArgs returns the names of an object with args list.
func tableNamesFromObjectWithArgs(l *ast.List) []*ast.TableName {
	return objectWithArgsNames(l)
}

// firstName returns the first name of a name list.
func firstName(l *ast.List) string {
	if l == nil {
		return ""
	}
	for _, item := range l.Items {
		if s, ok := item.(*ast.String); ok {
			return s.Str
		}
	}
	return ""
}

// indexNameFromList returns the index name of a name list.
func indexNameFromList(l yyLexer, list *ast.List) ast.QualifiedName {
	return syntaxName(syntaxIdents(l, list)...)
}

// columnNameListFromStrings returns the column names of a string list.
func columnNameListFromStrings(l *ast.List) []*ast.ColumnName {
	if l == nil {
		return nil
	}
	var cols []*ast.ColumnName
	for _, item := range l.Items {
		s, ok := item.(*ast.String)
		if !ok {
			continue
		}
		cols = append(cols, &ast.ColumnName{Name: model.NewCIStr(s.Str)})
	}
	return cols
}
