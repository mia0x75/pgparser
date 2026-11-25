// Package ast is the abstract syntax tree parsed from a SQL statement by parser.
// It can be analysed and transformed by optimizer.
package ast

import (
	pkast "github.com/mia0x75/parserkit/ast"

	"github.com/mia0x75/pgparser/model"
)

// Node is the basic element of the AST.
// Interfaces embed Node should have 'Node' name suffix.
type Node = pkast.Node

// Flags indicates whether an expression contains certain types of expression.
const (
	FlagConstant       uint64 = 0
	FlagHasParamMarker uint64 = 1 << iota
	FlagHasFunc
	FlagHasReference
	FlagHasAggregateFunc
	FlagHasSubquery
	FlagHasVariable
	FlagHasDefault
	FlagPreEvaluated
	FlagHasWindowFunc
)

// ExprNode is a node that can be evaluated.
// Name of implementations should have 'Expr' suffix.
type ExprNode = pkast.ExprNode

// OptBinary is used for parser.
type OptBinary struct {
	IsBinary bool
	Charset  string
}

// FuncNode represents function call expression node.
type FuncNode interface {
	ExprNode
	functionExpression()
}

// StmtNode represents statement node.
// Name of implementations should have 'Stmt' suffix.
type StmtNode = pkast.StmtNode

// DDLNode represents DDL statement node.
type DDLNode = pkast.DDLNode

// DMLNode represents DML statement node.
type DMLNode = pkast.DMLNode

// ResultField represents a result field which can be a column from a table,
// or an expression in select field. It is a generated property during
// binding process. ResultField is the key element to evaluate a ColumnNameExpr.
// After resolving process, every ColumnNameExpr will be resolved to a ResultField.
// During execution, every row retrieved from table will set the row value to
// ResultFields of that table, so ColumnNameExpr resolved to that ResultField can be
// easily evaluated.
type ResultField struct {
	Column       *model.ColumnInfo
	ColumnAsName model.CIStr
	Table        *model.TableInfo
	TableAsName  model.CIStr
	DBName       model.CIStr

	// Expr represents the expression for the result field. If it is generated from a select field, it would
	// be the expression of that select field, otherwise the type would be ValueExpr and value
	// will be set for every retrieved row.
	Expr      ExprNode
	TableName *TableName
	// Referenced indicates the result field has been referenced or not.
	// If not, we don't need to get the values.
	Referenced bool
}

// ResultSetNode interface has a ResultFields property, represents a Node that returns result set.
// Implementations include SelectStmt, SubqueryExpr, TableSource, TableName and Join.
type ResultSetNode interface {
	Node
}

// SensitiveStmtNode overloads StmtNode and provides a SecureText method.
type SensitiveStmtNode interface {
	StmtNode
	// SecureText is different from Text that it hide password information.
	SecureText() string
}

// Visitor visits a Node.
type Visitor = pkast.Visitor
