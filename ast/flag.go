package ast

// HasAggFlag checks if the expr contains FlagHasAggregateFunc.
func HasAggFlag(expr ExprNode) bool {
	return ExprFlag(expr)&FlagHasAggregateFunc > 0
}

// HasWindowFlag checks if the expr contains FlagHasWindowFunc.
func HasWindowFlag(expr ExprNode) bool {
	return ExprFlag(expr)&FlagHasWindowFunc > 0
}

func expressionFlags(nodes ...Node) uint64 {
	var flag uint64
	for _, n := range nodes {
		if expr, ok := n.(ExprNode); ok && expr != nil {
			flag |= ExprFlag(expr)
		}
	}
	return flag
}

func functionOptionFlags(filter ExprNode, orderBy, withinGroup *OrderByClause) uint64 {
	var flag uint64
	if filter != nil {
		flag |= ExprFlag(filter)
	}
	for _, order := range []*OrderByClause{orderBy, withinGroup} {
		if order != nil {
			for _, item := range order.Items {
				if core := item.OrderItemCore(); core != nil && core.Expr != nil {
					flag |= ExprFlag(core.Expr)
				}
			}
		}
	}
	return flag
}
