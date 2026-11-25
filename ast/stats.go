package ast

import "github.com/mia0x75/pgparser/format"

var _ StmtNode = &AnalyzeTableStmt{}

type AnalyzeTableStmt struct {
	stmtNode
	Options   *List
	Relations *List
}

func (n *AnalyzeTableStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("ANALYZE")
	if err := restoreUtilityOptions(ctx, n.Options); err != nil {
		return err
	}
	if n.Relations != nil {
		ctx.WritePlain(" ")
		return n.Relations.Restore(ctx)
	}
	return ctx.Err()
}

func (n *AnalyzeTableStmt) Accept(v Visitor) (Node, bool) {
	updated, skip := v.Enter(n)
	if skip {
		return v.Leave(updated)
	}
	n = updated.(*AnalyzeTableStmt)
	for _, list := range []**List{&n.Options, &n.Relations} {
		if *list != nil {
			child, ok := (*list).Accept(v)
			if !ok {
				return n, false
			}
			*list = child.(*List)
		}
	}
	return v.Leave(n)
}
