package ast

// HasAggFlag checks if the expr contains FlagHasAggregateFunc.
func HasAggFlag(expr ExprNode) bool {
	return ExprFlag(expr)&FlagHasAggregateFunc > 0
}

// HasWindowFlag TODO:
func HasWindowFlag(expr ExprNode) bool {
	return ExprFlag(expr)&FlagHasWindowFunc > 0
}

// SetFlag sets flag for expression.
func SetFlag(n Node) {
	var setter flagSetter
	n.Accept(&setter)
}

type flagSetter struct {
}

// Enter TODO:
func (f *flagSetter) Enter(in Node) (Node, bool) {
	return in, false
}

// Leave TODO:
func (f *flagSetter) Leave(in Node) (Node, bool) {
	if x, ok := in.(ParamMarkerExpr); ok {
		x.SetFlag(FlagHasParamMarker)
	}
	switch x := in.(type) {
	case *A_Indirection:
		flag := expressionFlags(x.Arg)
		if x.Indirection != nil {
			for _, item := range x.Indirection.Items {
				if index, ok := item.(*A_Indices); ok {
					flag |= expressionFlags(index.Lidx, index.Uidx)
				}
			}
		}
		x.SetFlag(flag)
	case *AggregateFuncExpr:
		f.aggregateFunc(x)
	case *WindowFuncExpr:
		f.windowFunc(x)
	case *BetweenExpr:
		x.SetFlag(ExprFlag(x.Expr) | ExprFlag(x.Left) | ExprFlag(x.Right))
	case *BinaryOperationExpr:
		x.SetFlag(ExprFlag(x.L) | ExprFlag(x.R))
	case *CaseExpr:
		f.caseExpr(x)
	case *ColumnNameExpr:
		x.SetFlag(FlagHasReference)
	case *CompareSubqueryExpr:
		x.SetFlag(ExprFlag(x.L) | ExprFlag(x.R))
	case *DefaultExpr:
		x.SetFlag(FlagHasDefault)
	case *ExistsSubqueryExpr:
		x.SetFlag(ExprFlag(x.Sel))
	case *FuncCallExpr:
		f.funcCall(x)
	case *SQLValueFunction:
		x.SetFlag(FlagHasFunc)
	case *FuncCastExpr:
		x.SetFlag(FlagHasFunc | ExprFlag(x.Expr))
	case *IsNullExpr:
		x.SetFlag(ExprFlag(x.Expr))
	case *IsTruthExpr:
		x.SetFlag(ExprFlag(x.Expr))
	case *ParenthesesExpr:
		x.SetFlag(ExprFlag(x.Expr))
	case *PatternInExpr:
		f.patternIn(x)
	case *PatternLikeExpr:
		f.patternLike(x)
	case *PatternRegexpExpr:
		f.patternRegexp(x)
	case *PositionExpr:
		x.SetFlag(FlagHasReference)
	case *RowExpr:
		f.row(x)
	case *SubqueryExpr:
		x.SetFlag(FlagHasSubquery)
	case *UnaryOperationExpr:
		x.SetFlag(ExprFlag(x.V))
	case *ValuesExpr:
		x.SetFlag(FlagHasReference)
	case *VariableExpr:
		if x.Value == nil {
			x.SetFlag(FlagHasVariable)
		} else {
			x.SetFlag(FlagHasVariable | ExprFlag(x.Value))
		}
	}

	return in, true
}

func (f *flagSetter) caseExpr(x *CaseExpr) {
	var flag uint64
	if x.Value != nil {
		flag |= ExprFlag(x.Value)
	}
	for _, val := range x.WhenClauses {
		flag |= ExprFlag(val.Expr)
		flag |= ExprFlag(val.Result)
	}
	if x.ElseClause != nil {
		flag |= ExprFlag(x.ElseClause)
	}
	x.SetFlag(flag)
}

func (f *flagSetter) patternIn(x *PatternInExpr) {
	flag := ExprFlag(x.Expr)
	for _, val := range x.List {
		flag |= ExprFlag(val)
	}
	if x.Sel != nil {
		flag |= ExprFlag(x.Sel)
	}
	x.SetFlag(flag)
}

func (f *flagSetter) patternLike(x *PatternLikeExpr) {
	flag := ExprFlag(x.Pattern)
	if x.Expr != nil {
		flag |= ExprFlag(x.Expr)
	}
	x.SetFlag(flag)
}

func (f *flagSetter) patternRegexp(x *PatternRegexpExpr) {
	flag := ExprFlag(x.Pattern)
	if x.Expr != nil {
		flag |= ExprFlag(x.Expr)
	}
	x.SetFlag(flag)
}

func (f *flagSetter) row(x *RowExpr) {
	var flag uint64
	for _, val := range x.Values {
		flag |= ExprFlag(val)
	}
	x.SetFlag(flag)
}

func (f *flagSetter) funcCall(x *FuncCallExpr) {
	flag := FlagHasFunc
	for _, val := range x.Args {
		flag |= ExprFlag(val)
	}
	x.SetFlag(flag | functionOptionFlags(x.Filter, x.OrderBy, x.WithinGroup))
}

func (f *flagSetter) aggregateFunc(x *AggregateFuncExpr) {
	flag := FlagHasAggregateFunc
	for _, val := range x.Args {
		flag |= ExprFlag(val)
	}
	x.SetFlag(flag | functionOptionFlags(x.Filter, x.OrderBy, x.WithinGroup))
}

func (f *flagSetter) windowFunc(x *WindowFuncExpr) {
	flag := FlagHasWindowFunc
	for _, val := range x.Args {
		flag |= ExprFlag(val)
	}
	if x.Spec.PartitionBy != nil {
		for _, item := range x.Spec.PartitionBy.Items {
			flag |= ExprFlag(item.Expr)
		}
	}
	flag |= functionOptionFlags(nil, x.Spec.OrderBy, nil)
	if x.Spec.Frame != nil {
		for _, bound := range []*FrameBound{&x.Spec.Frame.Extent.Start, &x.Spec.Frame.Extent.End} {
			flag |= expressionFlags(bound.Expr, bound.Unit)
		}
	}
	x.SetFlag(flag | functionOptionFlags(x.Filter, x.OrderBy, x.WithinGroup))
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
				flag |= ExprFlag(item.Expr)
			}
		}
	}
	return flag
}
