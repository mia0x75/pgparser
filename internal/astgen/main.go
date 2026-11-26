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
	for _, name := range []string{"ast/pg_nodes_generated.go", "ast/pg_ext.go"} {
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
			if d.Name.Name == "Restore" && name == "ast/pg_nodes_generated.go" {
				edits = append(edits, edit{fset.Position(d.Body.Pos()).Offset, fset.Position(d.Body.End()).Offset, "{ return restorePGNode(ctx, n) }"})
			}
			if d.Name.Name != "Accept" {
				continue
			}
			var b strings.Builder
			fmt.Fprintf(&b, "{ if n == nil { return nil, false }; node, skip := v.Enter(n); if !nodePresent(node) { return n, false }; if skip { return v.Leave(node) }; outer, ok := node.(*%s); if !ok || outer == nil { return node, false }; n = outer;", id.Name)
			for _, field := range structures[id.Name].Fields.List {
				if len(field.Names) == 0 {
					continue
				}
				for _, f := range field.Names {
					if !f.IsExported() {
						continue
					}
					if err := visitField(&b, fset, field.Type, "n."+f.Name, nodes, 0); err != nil {
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

func visitField(b *strings.Builder, fset *token.FileSet, expr ast.Expr, access string, nodes map[string]bool, depth int) error {
	if a, ok := expr.(*ast.ArrayType); ok && a.Len == nil {
		var body strings.Builder
		index := fmt.Sprintf("i%d", depth)
		if err := visitField(&body, fset, a.Elt, access+"["+index+"]", nodes, depth+1); err != nil {
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
			fmt.Fprintf(b, "if %s != nil { if !nodePresent(%s) { return n, false }; updated, ok := %s.Accept(v); if !ok || !nodePresent(updated) { return n, false }; %s = updated };", access, access, access, access)
		} else {
			fmt.Fprintf(b, "if %s != nil { if !nodePresent(%s) { return n, false }; updated, ok := %s.Accept(v); if !ok { return n, false }; replacement, ok := updated.(%s); if !ok || !nodePresent(replacement) { return n, false }; %s = replacement };", access, access, access, typStr, access)
		}
	}
	return nil
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
