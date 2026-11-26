package ast

import pkast "github.com/mia0x75/parserkit/ast"

// Aliases keep shared type identity without duplicate dialect state.
type (
	SelectItem   = pkast.SelectItem
	FieldList    = pkast.FieldList
	WildcardExpr = pkast.WildcardExpr
)

var (
	_ Node     = &SelectItem{}
	_ Node     = &FieldList{}
	_ ExprNode = &WildcardExpr{}
)
