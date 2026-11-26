package ast

import (
	"fmt"

	pkast "github.com/mia0x75/parserkit/ast"

	"github.com/mia0x75/pgparser/format"
)

type TableSourceNode = pkast.TableSourceNode

// PostgreSQL adds ONLY, TABLESAMPLE and typed function columns to the shared source.
type TableSource struct {
	pkast.BaseTableSource
	ColDefs     []*ColumnDef
	TableSample *TableSample
	Only        bool
}

func (n *TableSource) TableSourceCore() *pkast.BaseTableSource {
	if n == nil {
		return nil
	}
	return &n.BaseTableSource
}

func (n *TableSource) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil || !nodePresent(n.Source) {
		return fmt.Errorf("restore table source: missing source")
	}
	if len(n.Columns) > 0 && !n.Alias.Present() {
		return fmt.Errorf("restore table source: column aliases require an alias")
	}
	if len(n.Columns) > 0 && len(n.ColDefs) > 0 {
		return fmt.Errorf("restore table source: both column aliases and definitions")
	}
	if n.Only || n.TableSample != nil {
		if _, ok := n.Source.(*TableName); !ok {
			return fmt.Errorf("restore table source: ONLY and TABLESAMPLE require a table name")
		}
	}
	if n.Lateral {
		ctx.WriteKeyword("LATERAL ")
	}
	if n.Only {
		ctx.WriteKeyword("ONLY ")
	}
	needParen := false
	switch source := n.Source.(type) {
	case *QueryTailStmt:
		// A parenthesized query restores its own brackets.
		needParen = !source.Parenthesized
	case QueryNode, *Join:
		needParen = true
	}
	if needParen {
		ctx.WritePlain("(")
	}
	level := ctx.JoinLevel
	if _, ok := n.Source.(*Join); ok {
		ctx.JoinLevel = 0
	}
	err := n.Source.Restore(ctx)
	ctx.JoinLevel = level
	if err != nil {
		return fmt.Errorf("restore table source: %w", err)
	}
	if needParen {
		ctx.WritePlain(")")
	}
	if n.Alias.Present() {
		ctx.WriteKeyword(" AS ")
		ctx.WriteIdent(n.Alias)
	}
	if len(n.ColDefs) > 0 {
		if !n.Alias.Present() {
			ctx.WriteKeyword(" AS ")
		}
		ctx.WritePlain("(")
		for i, col := range n.ColDefs {
			if col == nil {
				return fmt.Errorf("restore table source column %d: nil definition", i)
			}
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if err := col.Restore(ctx); err != nil {
				return fmt.Errorf("restore table source column %d: %w", i, err)
			}
		}
		ctx.WritePlain(")")
	}
	if len(n.Columns) > 0 {
		ctx.WritePlain("(")
		for i, col := range n.Columns {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			ctx.WriteIdent(col)
		}
		ctx.WritePlain(")")
	}
	if n.TableSample != nil {
		ctx.WritePlain(" ")
		if err := n.TableSample.Restore(ctx); err != nil {
			return fmt.Errorf("restore table source sample: %w", err)
		}
	}
	return ctx.Err()
}

func (n *TableSource) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	outer, ok := entered.(*TableSource)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	if outer.Source != nil {
		if !nodePresent(outer.Source) {
			return outer, false
		}
		updated, ok := outer.Source.Accept(v)
		if !ok {
			return outer, false
		}
		source, ok := updated.(ResultSetNode)
		if !ok || !nodePresent(source) {
			return outer, false
		}
		outer.Source = source
	}
	for i, col := range outer.ColDefs {
		if col == nil {
			return outer, false
		}
		updated, ok := col.Accept(v)
		if !ok {
			return outer, false
		}
		column, ok := updated.(*ColumnDef)
		if !ok || column == nil {
			return outer, false
		}
		outer.ColDefs[i] = column
	}
	if outer.TableSample != nil {
		updated, ok := outer.TableSample.Accept(v)
		if !ok {
			return outer, false
		}
		sample, ok := updated.(*TableSample)
		if !ok || sample == nil {
			return outer, false
		}
		outer.TableSample = sample
	}
	return v.Leave(outer)
}

var (
	_ TableSourceNode = (*TableSource)(nil)
	_ QueryNode       = (*TableStmt)(nil)
)
