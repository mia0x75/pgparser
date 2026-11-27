package parser

import (
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/pgparser/ast"
)

// makeSetOp returns the set operation of two query blocks.
func makeSetOp(op ast.SetOprType, all bool, larg, rarg ast.Node) ast.Node {
	operator := op
	if all {
		operator = allSetOprType(op)
	}
	return &ast.SetOprStmt{BaseSetOperation: pkast.BaseSetOperation{
		Left:     queryNodeOf(larg),
		Right:    queryNodeOf(rarg),
		Operator: operator,
	}}
}

// queryNodeOf returns the query node of a set operand.
func queryNodeOf(n ast.Node) ast.QueryNode {
	query, _ := n.(ast.QueryNode)
	return query
}

// allSetOprType returns the ALL variant of a set operation type.
func allSetOprType(op ast.SetOprType) ast.SetOprType {
	switch op {
	case ast.Union:
		return ast.UnionAll
	case ast.Except:
		return ast.ExceptAll
	case ast.Intersect:
		return ast.IntersectAll
	}
	return op
}

// setWithClause sets the with clause of a select statement.
func setWithClause(n ast.Node, with ast.Node) ast.Node {
	if w, ok := with.(*ast.WithClause); ok {
		n = selectClauseNode(n)
		switch v := n.(type) {
		case *ast.SelectStmt:
			v.With = w
		case *ast.SetOprStmt:
			v.With = w
		case *ast.QueryTailStmt:
			v.With = w
		}
	}
	return n
}

// currentLimit returns the dialect limit of a shared limit node.
func currentLimit(node ast.LimitNode) *ast.Limit {
	limit, _ := node.(*ast.Limit)
	return limit
}

// setSelectOptions sets the order by, locking and limit clauses of a select statement.
func setSelectOptions(n ast.Node, orderBy ast.Node, locking ast.Node, limit *SelectLimit) ast.Node {
	n = selectClauseNode(n)
	switch v := n.(type) {
	case *ast.SelectStmt:
		if orderBy != nil {
			v.OrderBy = orderBy.(*ast.OrderByClause)
		}
		if limit != nil {
			if updated := limitClause(limit, currentLimit(v.Limit)); updated != nil {
				v.Limit = updated
			}
		}
		applyLocking(v, locking)
	case *ast.SetOprStmt:
		if orderBy != nil {
			v.Tail.OrderBy = orderBy.(*ast.OrderByClause)
		}
		if limit != nil {
			if updated := limitClause(limit, currentLimit(v.Tail.Limit)); updated != nil {
				v.Tail.Limit = updated
			}
		}
		if list, ok := locking.(*ast.List); ok {
			v.Locking = concatLists(v.Locking, list)
		}
	case *ast.QueryTailStmt:
		if v.Parenthesized {
			v.Query = queryNodeOf(setSelectOptions(v.Query, orderBy, locking, limit))
			return v
		}
		if orderBy != nil {
			v.Tail.OrderBy = orderBy.(*ast.OrderByClause)
		}
		if limit != nil {
			if updated := limitClause(limit, currentLimit(v.Tail.Limit)); updated != nil {
				v.Tail.Limit = updated
			}
		}
	}
	return n
}

// selectClauseNode gives VALUES and TABLE operands a query tail container
// because ORDER, LIMIT and WITH are not fields of those statement nodes.
func selectClauseNode(n ast.Node) ast.Node {
	switch n.(type) {
	case *ast.ValuesStmt, *ast.TableStmt:
		return &ast.QueryTailStmt{BaseQueryTailStmt: pkast.BaseQueryTailStmt{Query: queryNodeOf(n)}}
	default:
		return n
	}
}

// applySelectClauses sets the from, where, group by, having and window clauses
// of a select statement.
func applySelectClauses(n *ast.SelectStmt, from []ast.ResultSetNode, where ast.ExprNode, group *GroupClause,
	having ast.ExprNode, window ast.Node) {
	if len(from) > 0 {
		n.From = fromClauseOfNodes(from)
	}
	if where != nil {
		n.Where = where
	}
	if group != nil && group.List != nil {
		n.GroupBy = &ast.GroupByClause{
			BaseGroupByClause: pkast.BaseGroupByClause{Items: groupItemsFromList(group.List)},
			Distinct:          group.Distinct,
		}
	}
	if having != nil {
		n.Having = having
	}
	if window != nil {
		n.WindowSpecs = windowSpecNodesFromList(window.(*ast.List))
	}
}

// LIMIT and OFFSET can arrive in separate reductions of a parenthesized query.
func limitClause(l *SelectLimit, current *ast.Limit) *ast.Limit {
	if l == nil || (l.LimitCount == nil && l.LimitOffset == nil) {
		return current
	}
	if current == nil {
		current = &ast.Limit{}
	}
	if l.LimitCount != nil {
		current.Count = l.LimitCount
		current.WithTies = l.LimitOption == ast.LIMIT_OPTION_WITH_TIES
	}
	if l.LimitOffset != nil {
		current.Offset = l.LimitOffset
	}
	return current
}

// limitValueClause returns the select limit of a limit value.
func limitValueClause(count ast.ExprNode) *SelectLimit {
	return &SelectLimit{
		LimitCount:  count,
		LimitOption: ast.LIMIT_OPTION_COUNT,
	}
}

// applyLocking sets the lock clauses of a select statement.
func applyLocking(n *ast.SelectStmt, locking ast.Node) {
	if locking == nil {
		return
	}
	list, ok := locking.(*ast.List)
	if !ok || list == nil {
		return
	}
	n.Locking = concatLists(n.Locking, list)
	for _, item := range list.Items {
		lock, ok := item.(*ast.LockingClause)
		if !ok || lock == nil {
			continue
		}
		switch ast.LockClauseStrength(lock.Strength) {
		case ast.LCS_FORKEYSHARE:
			n.LockTp = ast.SelectLockKeyShare
		case ast.LCS_FORSHARE:
			n.LockTp = ast.SelectLockForShare
		case ast.LCS_FORNOKEYUPDATE:
			n.LockTp = ast.SelectLockNoKeyUpdate
		case ast.LCS_FORUPDATE:
			n.LockTp = ast.SelectLockForUpdate
		}
		switch ast.LockWaitPolicy(lock.WaitPolicy) {
		case ast.LockWaitError:
			n.LockWaitPolicy = ast.SelectLockWaitNoWait
		case ast.LockWaitSkip:
			n.LockWaitPolicy = ast.SelectLockWaitSkipLocked
		}
		if lock.LockedRels != nil {
			for _, rel := range lock.LockedRels.Items {
				switch v := rel.(type) {
				case *ast.String:
					n.LockedTables = append(n.LockedTables, syntaxTableName(nameFromString(v.Str)))
				case *ast.List:
					for _, item := range v.Items {
						if s, ok := item.(*ast.String); ok {
							n.LockedTables = append(n.LockedTables, syntaxTableName(nameFromString(s.Str)))
						}
					}
				case *ast.RangeVar:
					n.LockedTables = append(n.LockedTables, syntaxTableName(nameFromParts(v.Schemaname, v.Relname)))
				}
			}
		}
	}
}

// makeTableSource returns the table source of a qualified name.
func makeTableSource(l yyLexer, names *ast.List, span loc.Span) *ast.TableSource {
	parts := make([]lex.Ident, 0, len(names.Items))
	for _, item := range names.Items {
		if name, ok := item.(*ast.String); ok {
			parts = append(parts, syntaxIdent(l, name.Str, name.Span()))
		}
	}
	return syntaxTableSource(syntaxTableName(syntaxName(parts...)), span)
}

// fromRelationOfNodes returns the relation tree of a from list.
func fromRelationOfNodes(list []ast.ResultSetNode) ast.ResultSetNode {
	var table ast.ResultSetNode
	for _, right := range list {
		if right == nil {
			continue
		}
		if table == nil {
			table = right
			continue
		}
		join := &ast.Join{BaseJoinExpr: pkast.BaseJoinExpr{
			Left:  table,
			Right: right,
			Type:  pkast.JoinTypeCross,
		}}
		join.SetSpan(loc.Cover(table.Span(), right.Span()))
		table = join
	}
	return table
}

// fromClauseOfNodes returns the table references clause of a from list.
func fromClauseOfNodes(list []ast.ResultSetNode) *ast.FromClause {
	table := fromRelationOfNodes(list)
	if table == nil {
		return nil
	}
	return syntaxFromClause(table)
}

// joinQual holds the join condition of a joined table.
type joinQual struct {
	on    ast.ExprNode
	using []ast.QualifiedName
	alias lex.Ident
}

// cteSearchClause returns the SEARCH clause of a recursive CTE.
func cteSearchClause(n ast.Node) *ast.CTESearchClause {
	search, ok := n.(*ast.CTESearchClause)
	if !ok || search == nil {
		return nil
	}
	return search
}

// cteCycleClause returns the CYCLE clause of a recursive CTE.
func cteCycleClause(n ast.Node) *ast.CTECycleClause {
	cycle, ok := n.(*ast.CTECycleClause)
	if !ok || cycle == nil {
		return nil
	}
	return cycle
}

// makeJoin returns the join of two table references.
func makeJoin(tp pkast.JoinType, left, right ast.Node, qual *joinQual, natural bool) ast.Node {
	join := &ast.Join{BaseJoinExpr: pkast.BaseJoinExpr{
		Left:    resultSetNodeOf(left),
		Right:   resultSetNodeOf(right),
		Type:    tp,
		Natural: natural,
	}}
	if qual != nil {
		join.On = qual.on
		join.Using = qual.using
		join.UsingAlias = qual.alias
	}
	return join
}

// columnNamesFromList returns the column names of a name list.
func columnNamesFromList(l *ast.List) []*ast.ColumnName {
	if l == nil {
		return nil
	}
	var result []*ast.ColumnName
	for _, item := range l.Items {
		if s, ok := item.(*ast.String); ok {
			result = append(result, &ast.ColumnName{Name: s.Str})
		}
	}
	return result
}

// indexElemColNames returns the index columns of an index parameter list.
func indexElemColNames(l *ast.List) []*ast.IndexColName {
	if l == nil {
		return nil
	}
	var result []*ast.IndexColName
	for _, item := range l.Items {
		switch v := item.(type) {
		case *ast.IndexElem:
			col := &ast.IndexColName{Length: -1}
			if v.Name != "" {
				col.Column = &ast.ColumnName{Name: v.Name}
			} else if expr, ok := v.Expr.(ast.ExprNode); ok {
				col.Expr = expr
			}
			if v.Ordering == ast.SORTBY_DESC {
				col.Desc = true
			}
			switch v.NullsOrdering {
			case ast.SORTBY_NULLS_FIRST:
				col.NullsOrder = 1
			case ast.SORTBY_NULLS_LAST:
				col.NullsOrder = 2
			}
			col.Collation = v.Collation
			col.Opclass = v.Opclass
			col.OpclassOptions = v.Opclassopts
			result = append(result, col)
		case *ast.String:
			result = append(result, &ast.IndexColName{
				Column: &ast.ColumnName{Name: v.Str},
				Length: -1,
			})
		}
	}
	return result
}

// byItemsFromList returns the group by items of a group by list.
func byItemsFromList(l *ast.List) []*ast.GroupItem {
	if l == nil {
		return nil
	}
	var items []*ast.GroupItem
	for _, item := range l.Items {
		switch v := item.(type) {
		case *ast.GroupItem:
			items = append(items, v)
		case *ast.GroupingSet:
			items = append(items, &ast.GroupItem{GroupingSet: v})
		}
	}
	return items
}

// groupItemsFromList returns the grouping items of a group by list.
func groupItemsFromList(l *ast.List) []ast.GroupItemNode {
	items := byItemsFromList(l)
	if items == nil {
		return nil
	}
	result := make([]ast.GroupItemNode, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	return result
}

// windowSpecNodesFromList returns the window specifications of a window list.
func windowSpecNodesFromList(l *ast.List) []ast.WindowSpecNode {
	specs := windowSpecsFromList(l)
	if specs == nil {
		return nil
	}
	nodes := make([]ast.WindowSpecNode, 0, len(specs))
	for i := range specs {
		nodes = append(nodes, &specs[i])
	}
	return nodes
}

// returningFieldList returns the field list of a returning clause.
func returningFieldList(fields []*ast.SelectItem) *ast.FieldList {
	if fields == nil {
		return nil
	}
	list := &ast.FieldList{Items: fields}
	span := loc.InvalidSpan()
	for _, field := range fields {
		span = loc.Cover(span, field.Span())
	}
	list.SetSpan(span)
	return list
}
