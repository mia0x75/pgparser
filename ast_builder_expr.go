package parser

import (
	"strconv"
	"strings"

	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/model"
	"github.com/mia0x75/pgparser/opcode"
)

// operatorName returns the operator name of a single element operator list.
func operatorName(l *ast.List) string {
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

// compareOpcode maps an operator name to its comparison opcode.
func compareOpcode(name string) opcode.Op {
	switch name {
	case "=":
		return opcode.EQ
	case "<>", "!=":
		return opcode.NE
	case "<":
		return opcode.LT
	case ">":
		return opcode.GT
	case "<=":
		return opcode.LE
	case ">=":
		return opcode.GE
	}
	return opcode.EQ
}

// subTypeName returns the name of a subquery type (ANY or ALL).
func subTypeName(t int64) string {
	if t == int64(ast.ALL_SUBLINK) {
		return "ALL"
	}
	return "ANY"
}

// escapeChar returns the escape character of a LIKE ... ESCAPE expression.
func escapeChar(e ast.ExprNode) byte {
	if v, ok := e.(ast.ValueExpr); ok {
		if s, ok := v.GetValue().(string); ok && len(s) > 0 {
			return s[0]
		}
	}
	return '\\'
}

// negateExpr returns the negation of an expression.
func negateExpr(e ast.ExprNode) ast.ExprNode {
	if v, ok := e.(ast.ValueExpr); ok {
		switch val := v.GetValue().(type) {
		case int64:
			return ast.NewValueExpr(-val)
		case float64:
			return ast.NewValueExpr(-val)
		}
	}
	return ast.NewUnaryOperationExpr(opcode.Minus, e)
}

// rowValues returns the values of a row expression.
func rowValues(n ast.ExprNode) []ast.ExprNode {
	if r, ok := n.(*ast.RowExpr); ok {
		return r.Values
	}
	return []ast.ExprNode{n}
}

// whenClauses returns the when clauses of a CASE expression.
func whenClauses(l *ast.List) []*ast.WhenClause {
	if l == nil {
		return nil
	}
	var result []*ast.WhenClause
	for _, item := range l.Items {
		switch v := item.(type) {
		case *ast.WhenClause:
			result = append(result, v)
		case *ast.CaseWhen:
			result = append(result, &ast.WhenClause{Expr: v.Expr.(ast.ExprNode), Result: v.Result.(ast.ExprNode)})
		}
	}
	return result
}

// floatValueExpr returns the value expression of a floating point constant.
func floatValueExpr(s string) ast.ExprNode {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return ast.NewValueExpr(s)
	}
	return ast.NewValueExpr(f)
}

// columnRefWithIndirection builds a column reference with an indirection list.
func columnRefWithIndirection(name string, ind *ast.List) ast.ExprNode {
	if ind == nil {
		return ast.NewColumnNameExpr("", "", name)
	}
	names := []string{name}
	var indices []ast.Node
	wildcard := false
	for _, item := range ind.Items {
		switch v := item.(type) {
		case *ast.String:
			names = append(names, v.Str)
		case *ast.A_Indices:
			indices = append(indices, v)
		case *ast.A_Star:
			wildcard = true
		}
	}
	if wildcard && len(indices) == 0 {
		wildcardField := &ast.WildCardField{}
		switch len(names) {
		case 1:
			wildcardField.Table = model.NewCIStr(names[0])
		case 2:
			wildcardField.Schema = model.NewCIStr(names[0])
			wildcardField.Table = model.NewCIStr(names[1])
		default:
			wildcardField.Schema = model.NewCIStr(names[len(names)-2])
			wildcardField.Table = model.NewCIStr(names[len(names)-1])
		}
		return wildcardField
	}
	if len(indices) > 0 {
		return &ast.A_Indirection{
			Arg:         ast.NewColumnNameExpr("", "", name),
			Indirection: &ast.List{Items: indices},
		}
	}
	schema, table, col := "", "", names[0]
	switch len(names) {
	case 1:
	case 2:
		table = names[0]
		col = names[1]
	default:
		schema = names[0]
		table = names[1]
		col = names[2]
	}
	return ast.NewColumnNameExpr(schema, table, col)
}

// exprsFromList returns the expressions of a list.
func exprsFromList(v interface{}) []ast.ExprNode {
	switch l := v.(type) {
	case nil:
		return nil
	case []ast.ExprNode:
		return l
	case *ast.List:
		var result []ast.ExprNode
		for _, item := range l.Items {
			if e, ok := item.(ast.ExprNode); ok {
				result = append(result, e)
			}
		}
		return result
	}
	return nil
}

// funcNameString returns the dotted name of a function name list.
func funcNameString(l *ast.List) string {
	if l == nil {
		return ""
	}
	parts := make([]string, 0, len(l.Items))
	for _, item := range l.Items {
		if s, ok := item.(*ast.String); ok {
			parts = append(parts, s.Str)
		}
	}
	return strings.Join(parts, ".")
}

// nameListString returns the dotted name of a name list.
func nameListString(l interface{}) string {
	switch v := l.(type) {
	case *ast.List:
		return funcNameString(v)
	case string:
		return v
	case ast.Node:
		if s, ok := v.(*ast.String); ok {
			return s.Str
		}
	}
	return ""
}

// orderByFromList returns the order by clause of a sort clause list.
func orderByFromList(l *ast.List) *ast.OrderByClause {
	if l == nil || len(l.Items) == 0 {
		return nil
	}
	clause := &ast.OrderByClause{}
	for _, item := range l.Items {
		switch v := item.(type) {
		case *ast.ByItem:
			clause.Items = append(clause.Items, v)
		case *ast.SortBy:
			bi := &ast.ByItem{Expr: v.Node.(ast.ExprNode)}
			if v.SortbyDir == ast.SORTBY_DESC {
				bi.Desc = true
			}
			switch v.SortbyNulls {
			case ast.SORTBY_NULLS_FIRST:
				bi.NullOrder = 1
			case ast.SORTBY_NULLS_LAST:
				bi.NullOrder = 2
			}
			clause.Items = append(clause.Items, bi)
		}
	}
	return clause
}

// byItemsFromExprs returns the group by items of an expression list.
func byItemsFromExprs(exprs []ast.ExprNode) []*ast.ByItem {
	var items []*ast.ByItem
	for _, e := range exprs {
		items = append(items, &ast.ByItem{Expr: e})
	}
	return items
}

// groupingSetsFromList returns the grouping sets of a group by list.
func groupingSetsFromList(l *ast.List) [][]ast.ExprNode {
	if l == nil {
		return nil
	}
	var sets [][]ast.ExprNode
	for _, item := range l.Items {
		switch v := item.(type) {
		case *ast.ByItem:
			if v.GroupingSet != nil {
				sets = append(sets, v.GroupingSet.Sets...)
				continue
			}
			sets = append(sets, []ast.ExprNode{v.Expr})
		case *ast.GroupingSet:
			sets = append(sets, v.Sets...)
		}
	}
	return sets
}

// windowFuncExpr returns the window function expression of a function call.
func windowFuncExpr(fn *ast.FuncCallExpr, spec *ast.WindowSpec) ast.ExprNode {
	wf := &ast.WindowFuncExpr{F: fn.FnName.O, Args: fn.Args, Star: fn.Star}
	if spec != nil {
		wf.Spec = *spec
	}
	return wf
}

// windowSpecsFromList returns the window specifications of a window clause list.
func windowSpecsFromList(l *ast.List) []ast.WindowSpec {
	if l == nil {
		return nil
	}
	var specs []ast.WindowSpec
	for _, item := range l.Items {
		if spec, ok := item.(*ast.WindowSpec); ok {
			specs = append(specs, *spec)
		}
	}
	return specs
}

// typeModifiers returns the length and the scale of a type modifier list.
func typeModifiers(mods interface{}) (int, int) {
	flen, dec := -1, -1
	list, ok := mods.(*ast.List)
	if !ok {
		return flen, dec
	}
	for i, item := range list.Items {
		ac, ok := item.(*ast.A_Const)
		if !ok {
			continue
		}
		v, ok := ac.Val.(*ast.Integer)
		if !ok {
			continue
		}
		if i == 0 {
			flen = int(v.Ival)
		} else {
			dec = int(v.Ival)
		}
	}
	return flen, dec
}

// isAggregateName reports whether the function name is an aggregate function.
func isAggregateName(name string) bool {
	switch strings.ToLower(name) {
	case "count", "sum", "avg", "min", "max", "array_agg", "string_agg",
		"bool_and", "bool_or", "every", "json_agg", "jsonb_agg", "json_object_agg",
		"bit_and", "bit_or", "bit_xor", "stddev", "stddev_pop", "stddev_samp",
		"variance", "var_pop", "var_samp", "percentile_cont", "percentile_disc",
		"mode", "grouping", "corr", "covar_pop", "covar_samp", "regr_", "rank",
		"dense_rank", "row_number", "ntile", "cume_dist", "percent_rank":
		return true
	}
	return false
}
