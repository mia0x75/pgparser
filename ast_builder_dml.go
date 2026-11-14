package parser

import (
	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/model"
)

// makeSetOp returns the set operation statement of two select statements.
func makeSetOp(op ast.SetOprType, all bool, larg, rarg ast.Node) ast.Node {
	selects := make([]ast.Node, 0, 2)
	if set, ok := larg.(*ast.SetOprStmt); ok && set.SelectList != nil && !set.IsInBraces && set.With == nil && set.OrderBy == nil && set.Limit == nil && set.Locking == nil {
		// A chain of set operations is a flat list of selects.
		selects = append(selects, set.SelectList.Selects...)
	} else if larg != nil {
		selects = append(selects, larg)
	}
	if rarg != nil {
		selects = append(selects, rarg)
		t := op
		if all {
			t = allSetOprType(op)
		}
		switch sel := rarg.(type) {
		case *ast.SelectStmt:
			sel.AfterSetOperator = &t
		case *ast.SetOprStmt:
			sel.AfterSetOperator = &t
		case *ast.ValuesStmt:
			sel.AfterSetOperator = &t
		case *ast.TableStmt:
			sel.AfterSetOperator = &t
		}
	}
	return &ast.SetOprStmt{
		SelectList: &ast.SetOprSelectList{Selects: selects},
	}
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
		}
	}
	return n
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
			v.Limit = limitClause(limit, v.Limit)
		}
		applyLocking(v, locking)
	case *ast.SetOprStmt:
		if orderBy != nil {
			v.OrderBy = orderBy.(*ast.OrderByClause)
		}
		if limit != nil {
			v.Limit = limitClause(limit, v.Limit)
		}
		if list, ok := locking.(*ast.List); ok {
			v.Locking = concatLists(v.Locking, list)
		}
	}
	return n
}

// selectClauseNode gives VALUES and TABLE operands a SetOprStmt container
// because ORDER, LIMIT and WITH are not fields of those statement nodes.
func selectClauseNode(n ast.Node) ast.Node {
	switch n.(type) {
	case *ast.ValuesStmt, *ast.TableStmt:
		return &ast.SetOprStmt{SelectList: &ast.SetOprSelectList{Selects: []ast.Node{n}}}
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
		n.GroupBy = &ast.GroupByClause{Items: byItemsFromList(group.List)}
	}
	if having != nil {
		n.Having = &ast.HavingClause{Expr: having}
	}
	if window != nil {
		n.WindowSpecs = windowSpecsFromList(window.(*ast.List))
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
	if v, ok := count.(ast.ValueExpr); ok && v.GetValue() == nil {
		count = nil
	}
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
					n.LockedTables = append(n.LockedTables, &ast.TableName{Name: model.NewCIStr(v.Str)})
				case *ast.List:
					for _, item := range v.Items {
						if s, ok := item.(*ast.String); ok {
							n.LockedTables = append(n.LockedTables, &ast.TableName{Name: model.NewCIStr(s.Str)})
						}
					}
				case *ast.RangeVar:
					n.LockedTables = append(n.LockedTables, &ast.TableName{
						Schema: model.NewCIStr(v.Schemaname),
						Name:   model.NewCIStr(v.Relname),
					})
				}
			}
		}
	}
}

// makeTableSource returns the table source of a qualified name.
func makeTableSource(names *ast.List) *ast.TableSource {
	name := &ast.TableName{}
	if names != nil {
		switch len(names.Items) {
		case 1:
			if s, ok := names.Items[0].(*ast.String); ok {
				name.Name = model.NewCIStr(s.Str)
			}
		case 2:
			if s, ok := names.Items[0].(*ast.String); ok {
				name.Schema = model.NewCIStr(s.Str)
			}
			if s, ok := names.Items[1].(*ast.String); ok {
				name.Name = model.NewCIStr(s.Str)
			}
		case 3:
			if s, ok := names.Items[0].(*ast.String); ok {
				name.Schema = model.NewCIStr(s.Str)
			}
			if s, ok := names.Items[1].(*ast.String); ok {
				name.Name = model.NewCIStr(s.Str)
			}
		}
	}
	return &ast.TableSource{Source: name}
}

// fromClauseOfNodes returns the table references clause of a from list.
func fromClauseOfNodes(list []ast.ResultSetNode) *ast.TableRefsClause {
	var join *ast.Join
	for _, right := range list {
		if right == nil {
			continue
		}
		if join == nil {
			if j, ok := right.(*ast.Join); ok {
				join = j
				continue
			}
			join = &ast.Join{Left: right}
			continue
		}
		join = &ast.Join{Left: join, Right: right, Tp: ast.CrossJoin}
	}
	if join == nil {
		return nil
	}
	return &ast.TableRefsClause{TableRefs: join}
}

// joinTypeOf returns the join type of a join type code.
func joinTypeOf(code int64) ast.JoinType {
	switch ast.JoinType(code) {
	case ast.JOIN_LEFT:
		return ast.LeftJoin
	case ast.JOIN_RIGHT:
		return ast.RightJoin
	case ast.JOIN_FULL:
		return ast.FullJoin
	case ast.JOIN_INNER:
		return ast.InnerJoin
	}
	return ast.CrossJoin
}

// makeJoin returns the join of two table references.
func makeJoin(tp ast.JoinType, left, right ast.Node, qual ast.Node, natural bool) ast.Node {
	join := &ast.Join{
		Tp:          tp,
		NaturalJoin: natural,
	}
	if l, ok := left.(ast.ResultSetNode); ok {
		join.Left = l
	}
	if r, ok := right.(ast.ResultSetNode); ok {
		join.Right = r
	}
	if qual != nil {
		if on, ok := qual.(*ast.OnCondition); ok {
			join.On = on
			join.Using = on.Using
		}
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
			result = append(result, &ast.ColumnName{Name: model.NewCIStr(s.Str)})
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
				col.Column = &ast.ColumnName{Name: model.NewCIStr(v.Name)}
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
				Column: &ast.ColumnName{Name: model.NewCIStr(v.Str)},
				Length: -1,
			})
		}
	}
	return result
}

// byItemsFromList returns the group by items of a group by list.
func byItemsFromList(l *ast.List) []*ast.ByItem {
	if l == nil {
		return nil
	}
	var items []*ast.ByItem
	for _, item := range l.Items {
		switch v := item.(type) {
		case *ast.ByItem:
			items = append(items, v)
		case *ast.GroupingSet:
			items = append(items, &ast.ByItem{GroupingSet: v})
		}
	}
	return items
}

// returningFieldList returns the field list of a returning clause.
func returningFieldList(fields []*ast.SelectField) *ast.FieldList {
	if fields == nil {
		return nil
	}
	return &ast.FieldList{Fields: fields}
}
