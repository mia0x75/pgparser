package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
)

var spanTargets = []string{"node", "expr", "list", "bound"}

func generateParserSpans() error {
	const path = "parser/parser_generated.go"
	source, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	formatted, err := injectParserSpans(source)
	if err != nil {
		return err
	}
	return os.WriteFile(path, formatted, 0o644)
}

func injectParserSpans(source []byte) ([]byte, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "parser/parser_generated.go", source, 0)
	if err != nil {
		return nil, err
	}
	type edit struct {
		offset int
		indent string
		fields []string
	}
	var edits []edit
	ast.Inspect(file, func(node ast.Node) bool {
		stmt, ok := node.(*ast.SwitchStmt)
		if !ok {
			return true
		}
		tag, ok := stmt.Tag.(*ast.Ident)
		if !ok || tag.Name != "yynt" {
			return true
		}
		for _, node := range stmt.Body.List {
			clause, ok := node.(*ast.CaseClause)
			if !ok {
				continue
			}
			assigned := make(map[string]bool)
			existingFields := make(map[string]bool)
			var existing []ast.Stmt
			for _, statement := range clause.Body {
				ast.Inspect(statement, func(node ast.Node) bool {
					if _, field, ok := spanCallFromNode(node); ok {
						existingFields[field] = true
					}
					if field, ok := assignedField(node); ok && slices.Contains(spanTargets, field) {
						assigned[field] = true
					}
					return true
				})
				if _, _, ok := spanCall(statement); ok {
					existing = append(existing, statement)
				}
			}
			var wanted []string
			for _, field := range spanTargets {
				if assigned[field] && !existingFields[field] {
					wanted = append(wanted, field)
				}
			}
			if len(wanted) == 0 {
				continue
			}
			edit := edit{fields: wanted}
			if len(existing) > 0 {
				edit.offset = fset.Position(existing[len(existing)-1].End()).Offset
			} else {
				edit.offset = fset.Position(clause.End()).Offset
			}
			lineEnd := bytes.IndexByte(source[edit.offset:], '\n')
			if lineEnd < 0 {
				lineEnd = len(source) - edit.offset
			}
			edit.offset += lineEnd
			if edit.offset < len(source) {
				edit.offset++
			}
			caseStart := fset.Position(clause.Case).Offset
			caseLineStart := bytes.LastIndexByte(source[:caseStart], '\n') + 1
			indent := source[caseLineStart:caseStart]
			edit.indent = string(indent) + "\t"
			edits = append(edits, edit)
		}
		return false
	})
	if len(edits) == 0 {
		return source, nil
	}
	for _, edit := range slices.Backward(edits) {
		var calls bytes.Buffer
		for _, field := range edit.fields {
			fmt.Fprintf(&calls, "%ssetSyntaxSpan(yyVAL.%s, yyVAL.span)\n", edit.indent, field)
		}
		source = append(source[:edit.offset], append(calls.Bytes(), source[edit.offset:]...)...)
	}
	return source, nil
}

func spanCall(statement ast.Stmt) (*ast.CallExpr, string, bool) {
	expression, ok := statement.(*ast.ExprStmt)
	if !ok {
		return nil, "", false
	}
	return spanCallFromNode(expression)
}

func spanCallFromNode(node ast.Node) (*ast.CallExpr, string, bool) {
	expression, ok := node.(*ast.ExprStmt)
	if !ok {
		return nil, "", false
	}
	call, ok := expression.X.(*ast.CallExpr)
	if !ok {
		return nil, "", false
	}
	fun, ok := call.Fun.(*ast.Ident)
	if !ok || fun.Name != "setSyntaxSpan" || len(call.Args) != 2 {
		return nil, "", false
	}
	field, ok := call.Args[0].(*ast.SelectorExpr)
	if !ok {
		return nil, "", false
	}
	return call, field.Sel.Name, true
}

func assignedField(node ast.Node) (string, bool) {
	assignment, ok := node.(*ast.AssignStmt)
	if !ok {
		return "", false
	}
	for _, lhs := range assignment.Lhs {
		field, ok := lhs.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		owner, ok := field.X.(*ast.Ident)
		if ok && owner.Name == "yyVAL" {
			return field.Sel.Name, true
		}
	}
	return "", false
}
