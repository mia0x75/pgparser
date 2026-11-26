package parser

import "github.com/mia0x75/pgparser/ast"

// xmlTableColumns returns the column definitions of an XMLTABLE table function.
func xmlTableColumns(l *ast.List) []*ast.XMLTableColumn {
	if l == nil {
		return nil
	}
	var result []*ast.XMLTableColumn
	for _, item := range l.Items {
		col, ok := item.(*ast.XMLTableColumn)
		if !ok {
			continue
		}
		result = append(result, col)
	}
	return result
}

// xmlNamespaces returns the namespace definitions of an XMLTABLE table function.
func xmlNamespaces(l *ast.List) []*ast.XMLTableNamespace {
	if l == nil {
		return nil
	}
	var result []*ast.XMLTableNamespace
	for _, item := range l.Items {
		ns, ok := item.(*ast.XMLTableNamespace)
		if !ok {
			continue
		}
		result = append(result, ns)
	}
	return result
}

// applyXMLTableColumnOptions applies the column options of an XMLTABLE column.
func applyXMLTableColumnOptions(col *ast.XMLTableColumn, l *ast.List) {
	if col == nil || l == nil {
		return
	}
	for _, item := range l.Items {
		def, ok := item.(*ast.DefElem)
		if !ok {
			continue
		}
		switch def.Defname {
		case "default":
			if expr, ok := def.Arg.(ast.ExprNode); ok {
				col.DefaultExpr = expr
			}
		case "path":
			if expr, ok := def.Arg.(ast.ExprNode); ok {
				col.PathExpr = expr
			}
		case "__pg__is_not_null":
			if expr, ok := def.Arg.(ast.ExprNode); ok {
				if value, ok := expr.(ast.ValueExpr); ok {
					col.IsNotNull = value.GetValue() == int64(1) || value.GetValue() == true
				}
			}
		}
	}
}

// jsonTableColumns returns the column definitions of a JSON_TABLE table function.
func jsonTableColumns(l *ast.List) []*ast.JSONTableColumn {
	if l == nil {
		return nil
	}
	var result []*ast.JSONTableColumn
	for _, item := range l.Items {
		col, ok := item.(*ast.JSONTableColumn)
		if !ok {
			continue
		}
		result = append(result, col)
	}
	return result
}

// jsonTablePathSpec returns the path specification of a JSON_TABLE column.
func jsonTablePathSpec(n ast.Node) *ast.JSONTablePathSpec {
	if n == nil {
		return nil
	}
	spec, ok := n.(*ast.JSONTablePathSpec)
	if !ok {
		return nil
	}
	return spec
}

// jsonWrapperName returns the wrapper behavior name of a JSON_TABLE column.
func jsonWrapperName(value int64) string {
	switch ast.JsonWrapper(value) {
	case ast.JSW_CONDITIONAL:
		return "conditional"
	case ast.JSW_UNCONDITIONAL:
		return "unconditional"
	case ast.JSW_NONE:
		return "none"
	}
	return ""
}

// jsonQuotesName returns the quotes behavior name of a JSON_TABLE column.
func jsonQuotesName(value int64) string {
	switch ast.JsonQuotes(value) {
	case ast.JS_QUOTES_KEEP:
		return "keep"
	case ast.JS_QUOTES_OMIT:
		return "omit"
	}
	return ""
}

func nodesOfList(l *ast.List) []ast.Node {
	if l == nil {
		return nil
	}
	return l.Items
}

// rowsFromFunctions returns the functions of a ROWS FROM call.
func rowsFromFunctions(l *ast.List) []*ast.TableFunction {
	if l == nil {
		return nil
	}
	var result []*ast.TableFunction
	for _, item := range l.Items {
		fn, ok := item.(*ast.TableFunction)
		if !ok {
			continue
		}
		result = append(result, fn)
	}
	return result
}

// tableFuncElementDefs returns the column definitions of a function result.
func tableFuncElementDefs(l *ast.List) []*ast.ColumnDef {
	if l == nil {
		return nil
	}
	var defs []*ast.ColumnDef
	for _, item := range l.Items {
		if def, ok := item.(*ast.ColumnDef); ok {
			defs = append(defs, def)
		}
	}
	return defs
}

// jsonAggConstructor returns the constructor of a JSON aggregate function.
func jsonAggConstructor(n ast.Node) *ast.JsonAggConstructor {
	switch v := n.(type) {
	case *ast.JsonObjectAgg:
		return v.Constructor
	case *ast.JsonArrayAgg:
		return v.Constructor
	}
	return nil
}
