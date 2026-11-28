package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// Shared interfaces that can appear as concrete AST node field types.
// A nil pointer inside such an interface would type-assert successfully to
// nil and then panic in the child's Accept, so isNode must treat it as a
// node and emit a nodePresent guard.
var sharedNodeInterfaces = []string{
	"Node", "ExprNode", "StmtNode", "ResultSetNode",
	"QueryNode", "CTENode", "OrderByClauseNode",
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "parser" {
		if err := generateParserSpans(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := generate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate() error {
	files, err := filepath.Glob("ast/*.go")
	if err != nil {
		return err
	}
	nodes := make(map[string]bool)
	structures := make(map[string]*ast.StructType)
	restoreGuards := make(map[string][]sourceEdit)
	acceptGuards := make(map[string][]sourceEdit)
	acceptContracts := make(map[string][]sourceEdit)
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			return err
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					if t, ok := spec.(*ast.TypeSpec); ok {
						if s, ok := t.Type.(*ast.StructType); ok {
							structures[t.Name.Name] = s
						}
					}
				}
			case *ast.FuncDecl:
				if typeName, entered, pos, ok := missingAcceptContract(d, fset); ok {
					acceptContracts[name] = append(acceptContracts[name], sourceEdit{
						pos:  pos,
						text: fmt.Sprintf("\n\tenteredNode, enteredOK := %s.(*%s)\n\tif !enteredOK || enteredNode == nil {\n\t\treturn %s, false\n\t}\n", entered, typeName, entered),
					})
				}
				if d.Name.Name == "Accept" && needsAcceptGuard(d) {
					receiver := d.Recv.List[0].Names[0].Name
					pos := fset.Position(d.Body.Lbrace).Offset + 1
					acceptGuards[name] = append(acceptGuards[name], sourceEdit{
						pos:  pos,
						text: fmt.Sprintf("\n\tif %s == nil {\n\t\treturn nil, false\n\t}\n", receiver),
					})
				}
				if d.Name.Name == "Restore" && needsRestoreGuard(d) {
					receiver := d.Recv.List[0].Names[0].Name
					receiverType := d.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
					context := d.Type.Params.List[0].Names[0].Name
					pos := fset.Position(d.Body.Lbrace).Offset + 1
					restoreGuards[name] = append(restoreGuards[name], sourceEdit{
						pos:      pos,
						typeName: receiverType,
						text:     fmt.Sprintf("\n\tif %s == nil {\n\t\treturn restoreNode(%s, %s)\n\t}\n", receiver, context, receiver),
					})
				}
				if d.Name.Name == "Accept" && d.Recv != nil {
					if p, ok := d.Recv.List[0].Type.(*ast.StarExpr); ok {
						if id, ok := p.X.(*ast.Ident); ok {
							nodes[id.Name] = true
						}
					}
				}
			}
		}
	}
	guardEdits := make(map[string][]sourceEdit, len(acceptGuards)+len(acceptContracts)+len(restoreGuards))
	for name, edits := range acceptGuards {
		guardEdits[name] = append(guardEdits[name], edits...)
	}
	for name, edits := range acceptContracts {
		guardEdits[name] = append(guardEdits[name], edits...)
	}
	for name, edits := range restoreGuards {
		for _, edit := range edits {
			if nodes[edit.typeName] {
				guardEdits[name] = append(guardEdits[name], edit)
			}
		}
	}
	for name, edits := range guardEdits {
		if name == "ast/nodes_generated.go" {
			continue
		}
		source, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		sort.Slice(edits, func(i, j int) bool { return edits[i].pos > edits[j].pos })
		for _, edit := range edits {
			source = append(append(append([]byte(nil), source[:edit.pos]...), edit.text...), source[edit.pos:]...)
		}
		if err := os.WriteFile(name, source, 0o644); err != nil {
			return err
		}
	}
	for _, name := range []string{"ast/nodes_generated.go", "ast/collate.go", "ast/grouping_set.go", "ast/xml_table.go", "ast/json_table.go", "ast/rows_from.go", "ast/privilege_target.go"} {
		source, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		file, err := parser.ParseFile(fset, name, source, parser.ParseComments)
		if err != nil {
			return err
		}
		type edit struct {
			start, end int
			body       string
		}
		var edits []edit
		for _, decl := range file.Decls {
			d, ok := decl.(*ast.FuncDecl)
			if !ok || d.Recv == nil {
				continue
			}
			p, ok := d.Recv.List[0].Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			id, ok := p.X.(*ast.Ident)
			if !ok {
				continue
			}
			if d.Name.Name == "Restore" && name == "ast/nodes_generated.go" {
				edits = append(edits, edit{fset.Position(d.Body.Pos()).Offset, fset.Position(d.Body.End()).Offset, "{ return restoreNode(ctx, n) }"})
			}
			if d.Name.Name != "Accept" {
				continue
			}
			for i, statement := range d.Body.List {
				ifStmt, ok := statement.(*ast.IfStmt)
				if !ok || i == 0 || ifStmt.Body == nil {
					continue
				}
				condition, ok := ifStmt.Cond.(*ast.Ident)
				if !ok {
					continue
				}
				assign, ok := d.Body.List[i-1].(*ast.AssignStmt)
				if !ok || len(assign.Rhs) != 1 {
					continue
				}
				call, ok := assign.Rhs[0].(*ast.CallExpr)
				if !ok {
					continue
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != "Enter" || len(assign.Lhs) != 2 {
					continue
				}
				skipName, ok := assign.Lhs[1].(*ast.Ident)
				if !ok || skipName.Name != condition.Name {
					continue
				}
				for j := i + 1; j < len(d.Body.List); j++ {
					redundant, ok := d.Body.List[j].(*ast.AssignStmt)
					if !ok || len(redundant.Lhs) != 1 || len(redundant.Rhs) != 1 {
						continue
					}
					receiver, ok := redundant.Lhs[0].(*ast.Ident)
					rhs, rhsOK := redundant.Rhs[0].(*ast.TypeAssertExpr)
					if !ok || !rhsOK {
						break
					}
					asserted, assertedOK := rhs.Type.(*ast.StarExpr)
					if !assertedOK {
						break
					}
					assertedName, nameOK := asserted.X.(*ast.Ident)
					if nameOK && assertedName.Name == id.Name && receiver.Name == d.Recv.List[0].Names[0].Name {
						edits = append(edits, edit{fset.Position(redundant.Pos()).Offset, fset.Position(redundant.End()).Offset, ""})
					}
					break
				}
			}
			var b strings.Builder
			b.WriteString(acceptPrefix(id.Name))
			for _, field := range structures[id.Name].Fields.List {
				if len(field.Names) == 0 {
					if err := validateEmbeddedField(field, id.Name, fset, structures, nodes); err != nil {
						return err
					}
					continue
				}
				for _, f := range field.Names {
					if !f.IsExported() {
						continue
					}
					if err := visitField(&b, fset, field.Type, "n."+f.Name, nodes, 0, true); err != nil {
						return err
					}
				}
			}
			b.WriteString("return v.Leave(n) }")
			edits = append(edits, edit{fset.Position(d.Body.Pos()).Offset, fset.Position(d.Body.End()).Offset, b.String()})
		}
		for _, e := range slices.Backward(edits) {

			source = append(append(append([]byte(nil), source[:e.start]...), []byte(e.body)...), source[e.end:]...)
		}
		formatted, err := format.Source(source)
		if err != nil {
			return err
		}
		if err := os.WriteFile(name, formatted, 0o644); err != nil {
			return err
		}
	}
	return nil
}

type sourceEdit struct {
	pos      int
	typeName string
	text     string
}

func needsRestoreGuard(d *ast.FuncDecl) bool {
	if d.Recv == nil || len(d.Recv.List) != 1 || len(d.Recv.List[0].Names) != 1 || d.Body == nil || len(d.Body.List) == 0 {
		return false
	}
	pointer, ok := d.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	if _, ok := pointer.X.(*ast.Ident); !ok {
		return false
	}
	if len(d.Type.Params.List) == 0 || len(d.Type.Params.List[0].Names) != 1 {
		return false
	}
	receiver := d.Recv.List[0].Names[0].Name
	if _, ok := d.Body.List[0].(*ast.IfStmt); ok && isNilGuard(d.Body.List[0], receiver) {
		return false
	}
	if ret, ok := d.Body.List[0].(*ast.ReturnStmt); ok && len(ret.Results) == 1 {
		if call, ok := ret.Results[0].(*ast.CallExpr); ok {
			if fn, ok := call.Fun.(*ast.Ident); ok && fn.Name == "restoreNode" {
				return false
			}
		}
	}
	return true
}

func needsAcceptGuard(d *ast.FuncDecl) bool {
	if d.Recv == nil || len(d.Recv.List) != 1 || len(d.Recv.List[0].Names) != 1 || d.Body == nil || len(d.Body.List) == 0 {
		return false
	}
	pointer, ok := d.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	if _, ok := pointer.X.(*ast.Ident); !ok {
		return false
	}
	ifStmt, ok := d.Body.List[0].(*ast.IfStmt)
	if !ok || !isReceiverNilCheck(ifStmt.Cond, d.Recv.List[0].Names[0].Name) {
		return true
	}
	var returnsNil bool
	ast.Inspect(ifStmt.Body, func(node ast.Node) bool {
		ret, ok := node.(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 2 {
			return true
		}
		first, firstOK := ret.Results[0].(*ast.Ident)
		second, secondOK := ret.Results[1].(*ast.Ident)
		returnsNil = returnsNil || firstOK && secondOK && first.Name == "nil" && second.Name == "false"
		return true
	})
	return !returnsNil
}

func missingAcceptContract(d *ast.FuncDecl, fset *token.FileSet) (string, string, int, bool) {
	if d.Name.Name != "Accept" || d.Recv == nil || len(d.Recv.List) != 1 || len(d.Recv.List[0].Names) != 1 || d.Body == nil {
		return "", "", 0, false
	}
	pointer, ok := d.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return "", "", 0, false
	}
	typeIdent, ok := pointer.X.(*ast.Ident)
	if !ok {
		return "", "", 0, false
	}
	enterIndex := -1
	entered := ""
	skipIndex := -1
	for i, statement := range d.Body.List {
		assign, ok := statement.(*ast.AssignStmt)
		if !ok || len(assign.Lhs) != 2 || len(assign.Rhs) != 1 {
			continue
		}
		call, ok := assign.Rhs[0].(*ast.CallExpr)
		selector, selectorOK := call.Fun.(*ast.SelectorExpr)
		if !ok || !selectorOK || selector.Sel.Name != "Enter" {
			continue
		}
		enteredIdent, enteredOK := assign.Lhs[0].(*ast.Ident)
		skipIdent, skipOK := assign.Lhs[1].(*ast.Ident)
		if !enteredOK || !skipOK {
			continue
		}
		enterIndex = i
		entered = enteredIdent.Name
		for j := i + 1; j < len(d.Body.List); j++ {
			ifStmt, ok := d.Body.List[j].(*ast.IfStmt)
			if !ok {
				continue
			}
			condition, conditionOK := ifStmt.Cond.(*ast.Ident)
			if conditionOK && condition.Name == skipIdent.Name {
				skipIndex = j
				break
			}
		}
		break
	}
	if enterIndex < 0 || skipIndex < 0 {
		return "", "", 0, false
	}
	for _, statement := range d.Body.List[enterIndex+1 : skipIndex] {
		found := false
		ast.Inspect(statement, func(node ast.Node) bool {
			assertion, ok := node.(*ast.TypeAssertExpr)
			if !ok {
				return true
			}
			assertedType, ok := assertion.Type.(*ast.StarExpr)
			if !ok {
				return true
			}
			assertedIdent, identOK := assertedType.X.(*ast.Ident)
			found = identOK && assertedIdent.Name == typeIdent.Name
			return !found
		})
		if found {
			return "", "", 0, false
		}
	}
	return typeIdent.Name, entered, fset.Position(d.Body.List[skipIndex].Pos()).Offset, true
}

func isReceiverNilCheck(expr ast.Expr, receiver string) bool {
	comparison, ok := expr.(*ast.BinaryExpr)
	if !ok || comparison.Op != token.EQL {
		return false
	}
	left, leftOK := comparison.X.(*ast.Ident)
	right, rightOK := comparison.Y.(*ast.Ident)
	return leftOK && rightOK && (left.Name == receiver && right.Name == "nil" || right.Name == receiver && left.Name == "nil")
}

func isNilGuard(stmt ast.Stmt, receiver string) bool {
	ifStmt, ok := stmt.(*ast.IfStmt)
	if !ok {
		return false
	}
	if unary, ok := ifStmt.Cond.(*ast.UnaryExpr); ok && unary.Op == token.NOT {
		call, ok := unary.X.(*ast.CallExpr)
		if ok && len(call.Args) == 1 {
			fn, ok := call.Fun.(*ast.Ident)
			arg, argOK := call.Args[0].(*ast.Ident)
			return ok && argOK && fn.Name == "nodePresent" && arg.Name == receiver
		}
	}
	return isReceiverNilCheck(ifStmt.Cond, receiver)
}

func acceptPrefix(name string) string {
	return fmt.Sprintf("{ if n == nil { return nil, false }; node, skip := v.Enter(n); outer, ok := node.(*%s); if !ok || outer == nil { return node, false }; n = outer; if skip { return v.Leave(n) };", name)
}

func isNode(expr ast.Expr, nodes map[string]bool) bool {
	switch t := expr.(type) {
	case *ast.StarExpr:
		id, ok := t.X.(*ast.Ident)
		return ok && nodes[id.Name]
	case *ast.Ident:
		return slices.Contains(sharedNodeInterfaces, t.Name)
	}
	return false
}

func isValueNode(expr ast.Expr, nodes map[string]bool) bool {
	id, ok := expr.(*ast.Ident)
	return ok && nodes[id.Name]
}

func hasNodeChildren(structure *ast.StructType, nodes map[string]bool) bool {
	for _, field := range structure.Fields.List {
		if len(field.Names) == 0 || !field.Names[0].IsExported() {
			continue
		}
		if fieldHasNode(field.Type, nodes) {
			return true
		}
	}
	return false
}

func validateEmbeddedField(field *ast.Field, owner string, fset *token.FileSet, structures map[string]*ast.StructType, nodes map[string]bool) error {
	embedded := fieldTypeName(field.Type, fset)
	if structures[embedded] != nil && hasNodeChildren(structures[embedded], nodes) {
		return fmt.Errorf("unsupported child-bearing embedded field %s in %s", embedded, owner)
	}
	return nil
}

func fieldHasNode(expr ast.Expr, nodes map[string]bool) bool {
	if isNode(expr, nodes) || isValueNode(expr, nodes) {
		return true
	}
	if list, ok := expr.(*ast.ArrayType); ok && list.Len == nil {
		return fieldHasNode(list.Elt, nodes)
	}
	return false
}

func visitField(b *strings.Builder, fset *token.FileSet, expr ast.Expr, access string, nodes map[string]bool, depth int, optional bool) error {
	if a, ok := expr.(*ast.ArrayType); ok && a.Len == nil {
		var body strings.Builder
		index := fmt.Sprintf("i%d", depth)
		if err := visitField(&body, fset, a.Elt, access+"["+index+"]", nodes, depth+1, false); err != nil {
			return err
		}
		if body.Len() > 0 {
			fmt.Fprintf(b, "for %s := range %s { %s };", index, access, body.String())
		}
	} else if isNode(expr, nodes) {
		var typ bytes.Buffer
		if err := format.Node(&typ, fset, expr); err != nil {
			return err
		}
		typStr := typ.String()
		if typStr == "Node" {
			visitNode(b, access, typStr, optional, true)
		} else {
			visitNode(b, access, typStr, optional, false)
		}
	} else if isValueNode(expr, nodes) {
		typ := fieldTypeName(expr, fset)
		fmt.Fprintf(b, "updated, ok := (&%s).Accept(v); if !ok || !nodePresent(updated) { return n, false }; replacement, ok := updated.(*%s); if !ok || replacement == nil { return n, false }; %s = *replacement;", access, typ, access)
	}
	return nil
}

func visitNode(b *strings.Builder, access, typ string, optional, interfaceType bool) {
	if optional {
		fmt.Fprintf(b, "if %s != nil {", access)
	}
	fmt.Fprintf(b, "if !nodePresent(%s) { return n, false }; updated, ok := %s.Accept(v); if !ok || !nodePresent(updated) { return n, false };", access, access)
	if interfaceType {
		fmt.Fprintf(b, "%s = updated;", access)
	} else {
		fmt.Fprintf(b, "replacement, ok := updated.(%s); if !ok { return n, false }; %s = replacement;", typ, access)
	}
	if optional {
		b.WriteByte('}')
	}
	b.WriteByte(';')
}

func fieldTypeName(expr ast.Expr, fset *token.FileSet) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		id, ok := t.X.(*ast.Ident)
		if ok {
			return id.Name
		}
	case *ast.Ident:
		return t.Name
	}
	var buf strings.Builder
	_ = format.Node(&buf, fset, expr)
	return buf.String()
}
