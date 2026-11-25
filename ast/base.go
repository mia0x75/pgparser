package ast

import (
	pkast "github.com/mia0x75/parserkit/ast"

	"github.com/mia0x75/pgparser/types"
)

// node shares the common syntax data; outer nodes implement Accept and Restore.
// Node implementations should embed it in.
type node = pkast.BaseNode

// stmtNode shares the common statement marker; outer nodes implement Accept and Restore.
// Statement implementations should embed it in.
type stmtNode = pkast.BaseStmt

// ddlNode shares the common DDL marker; outer nodes implement Accept and Restore.
// DDL implementations should embed it in.
type ddlNode = pkast.BaseDDL

// dmlNode shares the common DML marker; outer nodes implement Accept and Restore.
// DML implementations should embed it in.
type dmlNode = pkast.BaseDML

// exprNode shares the common expression marker and keeps the local legacy
// type and flag cache.
// Expression implementations should embed it in.
type exprNode struct {
	pkast.BaseExpr
	Type types.FieldType
	flag uint64
}

// TexprNode preserves the legacy base type name during migration.
type TexprNode = exprNode

// SetType stores the local legacy field type.
func (en *exprNode) SetType(tp *types.FieldType) {
	en.Type = *tp
}

// GetType returns the local legacy field type.
func (en *exprNode) GetType() *types.FieldType {
	return &en.Type
}

// SetFlag stores the local legacy flag cache.
func (en *exprNode) SetFlag(flag uint64) {
	en.flag = flag
}

// GetFlag returns the local legacy flag cache.
func (en *exprNode) GetFlag() uint64 {
	return en.flag
}

type funcNode struct {
	exprNode
}

// functionExpression implements FunctionNode interface.
func (fn *funcNode) functionExpression() {}
