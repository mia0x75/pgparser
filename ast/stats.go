package ast

import (
	"github.com/mia0x75/pgparser/errors"

	"github.com/mia0x75/pgparser/format"
	"github.com/mia0x75/pgparser/model"
)

var (
	_ StmtNode = &AnalyzeTableStmt{}
	_ StmtNode = &DropStatsStmt{}
	_ StmtNode = &LoadStatsStmt{}
)

// AnalyzeTableStmt is used to create table statistics.
type AnalyzeTableStmt struct {
	stmtNode

	TableNames     []*TableName
	PartitionNames []model.CIStr
	IndexNames     []model.CIStr
	MaxNumBuckets  uint64

	// IndexFlag is true when we only analyze indices for a table.
	IndexFlag bool
}

// Restore implements Node interface.
func (n *AnalyzeTableStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("ANALYZE TABLE ")
	for i, table := range n.TableNames {
		if i != 0 {
			ctx.WritePlain(",")
		}
		if err := table.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore AnalyzeTableStmt.TableNames[%d]", i)
		}
	}
	if len(n.PartitionNames) != 0 {
		ctx.WriteKeyWord(" PARTITION ")
	}
	for i, partition := range n.PartitionNames {
		if i != 0 {
			ctx.WritePlain(",")
		}
		ctx.WriteName(partition.O)
	}
	if n.IndexFlag {
		ctx.WriteKeyWord(" INDEX")
	}
	for i, index := range n.IndexNames {
		if i != 0 {
			ctx.WritePlain(",")
		} else {
			ctx.WritePlain(" ")
		}
		ctx.WriteName(index.O)
	}
	if n.MaxNumBuckets != 0 {
		ctx.WriteKeyWord(" WITH ")
		ctx.WritePlainf("%d", n.MaxNumBuckets)
		ctx.WriteKeyWord(" BUCKETS")
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *AnalyzeTableStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*AnalyzeTableStmt)
	for i, val := range n.TableNames {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.TableNames[i] = node.(*TableName)
	}
	return v.Leave(n)
}

// DropStatsStmt is used to drop table statistics.
type DropStatsStmt struct {
	stmtNode

	Table *TableName
}

// Restore implements Node interface.
func (n *DropStatsStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("DROP STATS ")
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while add table")
	}

	return nil
}

// Accept implements Node Accept interface.
func (n *DropStatsStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*DropStatsStmt)
	node, ok := n.Table.Accept(v)
	if !ok {
		return n, false
	}
	n.Table = node.(*TableName)
	return v.Leave(n)
}

// LoadStatsStmt is the statement node for loading statistic.
type LoadStatsStmt struct {
	stmtNode

	Path string
}

// Restore implements Node interface.
func (n *LoadStatsStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("LOAD STATS ")
	ctx.WriteString(n.Path)
	return nil
}

// Accept implements Node Accept interface.
func (n *LoadStatsStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*LoadStatsStmt)
	return v.Leave(n)
}
