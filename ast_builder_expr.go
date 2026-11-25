package parser

import (
	"strings"

	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/model"
	"github.com/mia0x75/pgparser/opcode"
)

func operatorName(l *ast.List) string {
	if l == nil {
		return ""
	}
	if len(l.Items) > 1 {
		return qualifiedOperatorName(l)
	}
	for _, item := range l.Items {
		if s, ok := item.(*ast.String); ok {
			return s.Str
		}
	}
	return ""
}

func qualifiedOperatorName(l *ast.List) string {
	parts := funcNameParts(l)
	for i := 0; i < len(parts)-1; i++ {
		parts[i] = `"` + strings.ReplaceAll(parts[i], `"`, `""`) + `"`
	}
	return "OPERATOR(" + strings.Join(parts, ".") + ")"
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
	return 0
}

// subTypeName returns the name of a subquery type (ANY or ALL).
func subTypeName(t int64) string {
	if t == int64(ast.ALL_SUBLINK) {
		return "ALL"
	}
	return "ANY"
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
		return r.Items
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
	return &ast.NumericLiteral{Value: s}
}

// columnRefWithIndirection builds a column reference with an indirection list.
func columnRefWithIndirection(name string, ind *ast.List) ast.ExprNode {
	if ind == nil {
		return ast.NewColumnNameExpr("", "", name)
	}
	names := []string{name}
	wildcard := false
	for i, item := range ind.Items {
		switch v := item.(type) {
		case *ast.String:
			names = append(names, v.Str)
		case *ast.A_Indices:
			// Preserve the ordered suffix after the first subscript instead of flattening fields into the base name.
			base := columnRefWithIndirection(name, &ast.List{Items: ind.Items[:i]})
			return withIndirection(base, &ast.List{Items: ind.Items[i:]})
		case *ast.A_Star:
			wildcard = true
		}
	}
	if wildcard {
		wildcardField := &ast.WildCardField{}
		switch len(names) {
		case 1:
			wildcardField.Table = model.NewCIStr(names[0])
		case 2:
			wildcardField.Schema = model.NewCIStr(names[0])
			wildcardField.Table = model.NewCIStr(names[1])
		case 3:
			wildcardField.Catalog = model.NewCIStr(names[0])
			wildcardField.Schema = model.NewCIStr(names[1])
			wildcardField.Table = model.NewCIStr(names[2])
		default:
			return &ast.A_Indirection{Arg: ast.NewColumnNameExpr("", "", name), Indirection: ind}
		}
		return wildcardField
	}
	schema, table, col := "", "", names[0]
	switch len(names) {
	case 1:
	case 2:
		table = names[0]
		col = names[1]
	case 3:
		schema = names[0]
		table = names[1]
		col = names[2]
	case 4:
		expr := ast.NewColumnNameExpr(names[1], names[2], names[3])
		expr.Name.Catalog = model.NewCIStr(names[0])
		return expr
	default:
		return &ast.A_Indirection{Arg: ast.NewColumnNameExpr("", "", name), Indirection: ind}
	}
	return ast.NewColumnNameExpr(schema, table, col)
}

// exprsFromList returns the expressions of a list.
func exprsFromList(v any) []ast.ExprNode {
	switch l := v.(type) {
	case nil:
		return nil
	case []ast.ExprNode:
		return l
	case *ast.List:
		if l == nil {
			return nil
		}
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

// funcNameParts returns the name parts of a function name list.
func funcNameParts(l *ast.List) []string {
	if l == nil {
		return nil
	}
	parts := make([]string, 0, len(l.Items))
	for _, item := range l.Items {
		switch v := item.(type) {
		case *ast.String:
			parts = append(parts, v.Str)
		case *ast.DataType:
			// Name-only AST fields retain their legacy flattened representation.
			parts = append(parts, v.Type.String())
		}
	}
	return parts
}

// funcNameString returns the dotted name of a function name list.
func funcNameString(l *ast.List) string {
	return strings.Join(funcNameParts(l), ".")
}

// nameListString returns the dotted name of a name list.
func nameListString(l any) string {
	switch v := l.(type) {
	case string:
		return v
	case *ast.CollateClause:
		// opt_collate_clause clauses carry the collation name list.
		return nameListString(v.Collname)
	case ast.Node:
		// commentNameParts is the single node-to-parts helper; join for callers
		// that store a flattened dotted name.
		return strings.Join(commentNameParts(v), ".")
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
			if v.SortbyDir == ast.SORTBY_USING {
				bi.UseOp = operatorName(v.UseOp)
			}
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

// groupingSetsFromList returns the grouping sets of a group by list. Each
// element keeps its own slice so that an empty set restores as "()".
func groupingSetsFromList(l *ast.List) [][]ast.ExprNode {
	if l == nil {
		return nil
	}
	var sets [][]ast.ExprNode
	for _, item := range l.Items {
		switch v := item.(type) {
		case *ast.ByItem:
			if v.GroupingSet != nil {
				if v.GroupingSet.Type == ast.GroupingSetEmpty {
					sets = append(sets, nil)
					continue
				}
				// shortcut: nested ROLLUP/CUBE/SETS lose their kind and become
				// plain sets; upgrade if the grammar needs that fidelity.
				sets = append(sets, v.GroupingSet.Sets...)
				continue
			}
			sets = append(sets, groupingSetExprs(v.Expr))
		case *ast.GroupingSet:
			sets = append(sets, v.Sets...)
		}
	}
	return sets
}

// withIndirection attaches parsed subscripts and field selections to expr.
func withIndirection(expr ast.ExprNode, l *ast.List) ast.ExprNode {
	if l == nil || len(l.Items) == 0 {
		return expr
	}
	return &ast.A_Indirection{Arg: expr, Indirection: l}
}

// groupingSetExprs returns the expressions of one grouping set element. The
// parentheses are re-added by Restore, so unwrap them to stay stable.
func groupingSetExprs(e ast.ExprNode) []ast.ExprNode {
	for {
		paren, ok := e.(*ast.ParenthesesExpr)
		if !ok {
			break
		}
		e = paren.Expr
	}
	if row, ok := e.(*ast.RowExpr); ok {
		return row.Items
	}
	return []ast.ExprNode{e}
}

// windowFuncExpr returns the window expression of a function call.
func windowFuncExpr(fn *ast.FuncCallExpr, spec *ast.WindowSpec) ast.ExprNode {
	wf := &ast.WindowFuncExpr{F: fn.FnName.O, Args: fn.Args, Star: fn.Star,
		FnNameParts: fn.FnNameParts, Distinct: fn.Distinct, OrderBy: fn.OrderBy,
		Filter: fn.Filter, WithinGroup: fn.WithinGroup}
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
func typeModifiers(mods []ast.ExprNode) (int, int) {
	flen, dec := -1, -1
	for i, item := range mods {
		v, ok := item.(ast.ValueExpr)
		if !ok {
			continue
		}
		n, ok := v.GetValue().(int64)
		if !ok {
			continue
		}
		if i == 0 {
			flen = int(n)
		} else {
			dec = int(n)
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
