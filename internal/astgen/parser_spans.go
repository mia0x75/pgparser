package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"slices"
)

var spanTargets = []string{"node", "expr", "list"}

func generateParserSpans() error {
	const path = "parser.go"
	source, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, source, 0)
	if err != nil {
		return err
	}
	type edit struct {
		start, end int
		fields     []string
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
			present := make(map[string]bool)
			var existing []ast.Stmt
			for _, statement := range clause.Body {
				ast.Inspect(statement, func(node ast.Node) bool {
					if _, field, ok := spanCallFromNode(node); ok {
						present[field] = true
					}
					if field, ok := assignedField(node); ok && slices.Contains(spanTargets, field) {
						present[field] = true
					}
					return true
				})
				if _, _, ok := spanCall(statement); ok {
					existing = append(existing, statement)
				}
			}
			var wanted []string
			for _, field := range spanTargets {
				if present[field] {
					wanted = append(wanted, field)
				}
			}
			if len(wanted) == 0 {
				continue
			}
			edit := edit{fields: wanted}
			if len(existing) > 0 {
				edit.start = fset.Position(existing[0].Pos()).Offset
				edit.end = fset.Position(existing[len(existing)-1].End()).Offset
			} else {
				offset := fset.Position(clause.End()).Offset
				edit.start = offset
				edit.end = offset
			}
			edits = append(edits, edit)
		}
		return false
	})
	for _, edit := range slices.Backward(edits) {
		var calls bytes.Buffer
		calls.WriteByte('\n')
		for _, field := range edit.fields {
			fmt.Fprintf(&calls, "setSyntaxSpan(yyVAL.%s, yyVAL.span)\n", field)
		}
		source = append(source[:edit.start], append(calls.Bytes(), source[edit.end:]...)...)
	}
	newFile, err := parser.ParseFile(token.NewFileSet(), path, source, parser.ParseComments)
	if err != nil {
		return err
	}
	var formatted bytes.Buffer
	if err := format.Node(&formatted, token.NewFileSet(), newFile); err != nil {
		return err
	}
	return os.WriteFile(path, formatted.Bytes(), 0o644)
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
	if !ok || fun.Name != "setSyntaxSpan" || len(call.Args) < 2 {
		return nil, "", false
	}
	field, ok := call.Args[1].(*ast.SelectorExpr)
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
