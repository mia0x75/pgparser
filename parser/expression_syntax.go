package parser

import (
	"strings"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/pgparser/ast"
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
	if literal, ok := e.(*ast.LiteralExpr); ok {
		switch literal.Kind {
		case pkast.LiteralInteger, pkast.LiteralFloat, pkast.LiteralDecimal:
			if !strings.HasPrefix(literal.Value, "-") {
				return ast.NewLiteralExpr(literal.Kind, "-"+literal.Value)
			}
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

// floatValueExpr keeps the source spelling of a floating point constant. The
// lexer leaves oversized PostgreSQL integer forms (0x, 0o, 0b) as strings.
func floatValueExpr(s string) ast.ExprNode {
	kind := pkast.LiteralFloat
	unsigned := strings.TrimPrefix(strings.TrimPrefix(s, "+"), "-")
	if strings.HasPrefix(unsigned, "0x") || strings.HasPrefix(unsigned, "0o") || strings.HasPrefix(unsigned, "0b") {
		kind = pkast.LiteralInteger
	}
	return ast.NewLiteralExpr(kind, s)
}

// bitStringLiteral strips the token's b prefix; the value is restored with it.
// shortcut: the lexer accepts invalid bit digits, so those tokens keep the
// legacy string form; drop the fallback when the lexer rejects them.
func bitStringLiteral(token string) ast.ExprNode {
	value := strings.TrimPrefix(strings.TrimPrefix(token, "b"), "B")
	if !validBinaryDigits(value, 2) {
		return ast.NewLiteralExpr(pkast.LiteralString, token)
	}
	return ast.NewLiteralExpr(pkast.LiteralBit, value)
}

// hexStringLiteral strips the token's x prefix; the value is restored with it.
// shortcut: see bitStringLiteral; malformed hex tokens keep the string form.
func hexStringLiteral(token string) ast.ExprNode {
	value := strings.TrimPrefix(strings.TrimPrefix(token, "x"), "X")
	if !validBinaryDigits(value, 16) {
		return ast.NewLiteralExpr(pkast.LiteralString, token)
	}
	return ast.NewLiteralExpr(pkast.LiteralHex, value)
}

func validBinaryDigits(value string, base int) bool {
	for _, digit := range value {
		var valid bool
		switch base {
		case 2:
			valid = digit == '0' || digit == '1'
		default:
			valid = digit >= '0' && digit <= '9' || digit >= 'a' && digit <= 'f' || digit >= 'A' && digit <= 'F'
		}
		if !valid {
			return false
		}
	}
	return true
}

const maxColumnRefParts = 4

// columnRefWithIndirection keeps suffixes that cannot be stored in a qualified name.
func columnRefWithIndirection(l yyLexer, name lex.Ident, ind *ast.List, span loc.Span) ast.ExprNode {
	parts := make([]lex.Ident, 1, len(ind.Items)+1)
	parts[0] = name
	wildcard := false
	for i, item := range ind.Items {
		switch v := item.(type) {
		case *ast.String:
			parts = append(parts, syntaxIdent(l, v.Str, v.Span()))
		case *ast.A_Indices:
			// Preserve the ordered suffix after the first subscript instead of flattening fields into the base name.
			prefixSpan := name.Span
			if i > 0 {
				prefixSpan = loc.Cover(prefixSpan, ind.Items[i-1].Span())
			}
			base := columnRefWithIndirection(l, name, &ast.List{Items: ind.Items[:i]}, prefixSpan)
			return withIndirection(base, &ast.List{Items: ind.Items[i:]})
		case *ast.A_Star:
			wildcard = true
		}
	}
	if wildcard {
		if len(parts) >= maxColumnRefParts {
			return &ast.A_Indirection{Arg: columnRef(syntaxName(name), name.Span), Indirection: ind}
		}
		return syntaxWildcard(syntaxName(parts...), span)
	}
	if len(parts) > maxColumnRefParts {
		return &ast.A_Indirection{Arg: columnRef(syntaxName(name), name.Span), Indirection: ind}
	}
	return columnRef(syntaxName(parts...), span)
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
		case *ast.OrderItem:
			clause.Items = append(clause.Items, v)
		case *ast.SortBy:
			bi := &ast.OrderItem{Expr: v.Node.(ast.ExprNode)}
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

// orderItemsFromExprs returns the sorting items of an expression list.
func orderItemsFromExprs(exprs []ast.ExprNode) []*ast.OrderItem {
	var items []*ast.OrderItem
	for _, e := range exprs {
		items = append(items, &ast.OrderItem{BaseOrderItem: pkast.BaseOrderItem{Expr: e}})
	}
	return items
}

func windowPartitionClauseFromExprs(exprs []ast.ExprNode) *ast.PartitionByClause {
	items := orderItemsFromExprs(exprs)
	coreItems := make([]pkast.WindowPartitionItemNode, len(items))
	for i, item := range items {
		coreItems[i] = &ast.PartitionItem{BaseWindowPartitionItem: pkast.BaseWindowPartitionItem{Expr: item.Expr}}
	}
	return &ast.PartitionByClause{
		BaseWindowPartitionClause: pkast.BaseWindowPartitionClause{Items: coreItems},
		Items:                     items,
	}
}

// byItemsFromExprs returns the group by items of an expression list.
func byItemsFromExprs(exprs []ast.ExprNode) []*ast.GroupItem {
	var items []*ast.GroupItem
	for _, e := range exprs {
		items = append(items, &ast.GroupItem{BaseGroupItem: pkast.BaseGroupItem{Expr: e}})
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
		case *ast.GroupItem:
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
	wf := &ast.WindowFuncExpr{F: nameString(fn.Name), Args: fn.Args, Star: fn.Star,
		FnNameParts: nameStrings(fn.Name), Distinct: fn.Distinct, OrderBy: fn.OrderBy,
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
