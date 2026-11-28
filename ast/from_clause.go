package ast

import (
	"fmt"

	pkast "github.com/mia0x75/parserkit/ast"

	"github.com/mia0x75/pgparser/format"
)

type FromClauseNode = pkast.FromClauseNode

type FromClause struct {
	pkast.BaseFromClause
}

func (n *FromClause) FromClauseCore() *pkast.BaseFromClause {
	if n == nil {
		return nil
	}
	return &n.BaseFromClause
}

func (n *FromClause) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil || !nodePresent(n.Table) {
		return fmt.Errorf("restore from clause: missing relation")
	}
	// A derived query owns its join tree, independently of the surrounding JOIN.
	level := ctx.JoinLevel
	ctx.JoinLevel = 0
	defer func() { ctx.JoinLevel = level }()
	if err := n.Table.Restore(ctx); err != nil {
		return fmt.Errorf("restore from clause: %w", err)
	}
	return ctx.Err()
}

func (n *FromClause) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	clause, ok := entered.(*FromClause)
	if !ok || clause == nil {
		return entered, false
	}
	if skip {
		return v.Leave(clause)
	}
	if clause.Table != nil {
		if !nodePresent(clause.Table) {
			return clause, false
		}
		updated, ok := clause.Table.Accept(v)
		if !ok {
			return clause, false
		}
		table, ok := updated.(ResultSetNode)
		if !ok || !nodePresent(table) {
			return clause, false
		}
		clause.Table = table
	}
	return v.Leave(clause)
}

var _ FromClauseNode = (*FromClause)(nil)
