package ast

import (
	"fmt"
	"io"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/pgparser/errors"
	"github.com/mia0x75/pgparser/format"
)

var (
	_ Node = &WindowSpec{}
	_ Node = &PartitionByClause{}
	_ Node = &FrameClause{}
	_ Node = &FrameBound{}
)

// WindowSpecNode preserves the concrete dialect window identity.
type WindowSpecNode = pkast.WindowSpecNode

// WindowSpec is the specification of a window.
type WindowSpec struct {
	pkast.BaseWindowSpec

	PartitionBy *PartitionByClause
	OrderBy     *OrderByClause
	Frame       *FrameClause
}

// WindowSpecCore returns the shared window data without replacing this outer.
func (n *WindowSpec) WindowSpecCore() *pkast.BaseWindowSpec {
	if n == nil {
		return nil
	}
	return &n.BaseWindowSpec
}

// Restore implements Node interface.
func (n *WindowSpec) Restore(ctx *format.RestoreCtx) error {
	if n.Name.Present() {
		ctx.WriteIdent(n.Name)
		if n.OnlyAlias {
			return ctx.Err()
		}
		ctx.WriteKeyword(" AS ")
	}
	ctx.WritePlain("(")
	sep := ""
	if n.Ref.Present() {
		ctx.WriteIdent(n.Ref)
		sep = " "
	}
	if n.PartitionBy != nil {
		ctx.WritePlain(sep)
		if err := n.PartitionBy.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore WindowSpec.PartitionBy")
		}
		sep = " "
	}
	if n.OrderBy != nil {
		ctx.WritePlain(sep)
		if err := n.OrderBy.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore WindowSpec.OrderBy")
		}
		sep = " "
	}
	if n.Frame != nil {
		ctx.WritePlain(sep)
		if err := n.Frame.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore WindowSpec.Frame")
		}
	}
	ctx.WritePlain(")")

	return ctx.Err()
}

// Format formats the WindowSpec into a Writer.
func (n *WindowSpec) Format(w io.Writer) {
	if n.Name.Present() {
		name := n.Name.Name
		_, _ = fmt.Fprint(w, name)
		if n.OnlyAlias {
			return
		}
		_, _ = fmt.Fprint(w, " AS ")
	}
	_, _ = fmt.Fprint(w, "(")
	sep := ""
	if n.Ref.Present() {
		refName := n.Ref.Name
		_, _ = fmt.Fprint(w, refName)
		sep = " "
	}
	if n.PartitionBy != nil {
		_, _ = fmt.Fprint(w, sep)
		n.PartitionBy.Format(w)
		sep = " "
	}
	if n.OrderBy != nil {
		_, _ = fmt.Fprint(w, sep)
		n.OrderBy.Format(w)
		sep = " "
	}
	if n.Frame != nil {
		_, _ = fmt.Fprint(w, sep)
		n.Frame.Format(w)
	}
	_, _ = fmt.Fprint(w, ")")
}

// Accept implements Node Accept interface.
func (n *WindowSpec) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*WindowSpec)
	if n.PartitionBy != nil {
		node, ok := n.PartitionBy.Accept(v)
		if !ok {
			return n, false
		}
		n.PartitionBy = node.(*PartitionByClause)
	}
	if n.OrderBy != nil {
		node, ok := n.OrderBy.Accept(v)
		if !ok {
			return n, false
		}
		n.OrderBy = node.(*OrderByClause)
	}
	if n.Frame != nil {
		node, ok := n.Frame.Accept(v)
		if !ok {
			return n, false
		}
		n.Frame = node.(*FrameClause)
	}
	return v.Leave(n)
}

// PartitionByClause represents partition by clause.
type PartitionByClause struct {
	node

	Items []*OrderItem
}

// Restore implements Node interface.
func (n *PartitionByClause) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("PARTITION BY ")
	for i, v := range n.Items {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore PartitionByClause.Items[%d]", i)
		}
	}
	return ctx.Err()
}

// Format formats the PartitionByClause into a Writer.
func (n *PartitionByClause) Format(w io.Writer) {
	_, _ = fmt.Fprint(w, "PARTITION BY ")
	for i, v := range n.Items {
		if i != 0 {
			_, _ = fmt.Fprint(w, ", ")
		}
		v.Format(w)
	}
}

// Accept implements Node Accept interface.
func (n *PartitionByClause) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*PartitionByClause)
	for i, val := range n.Items {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Items[i] = node.(*OrderItem)
	}
	return v.Leave(n)
}

// FrameType is the type of window function frame.
type FrameType = pkast.WindowFrameType

// Window function frame types: ROWS, RANGE and GROUPS.
const (
	Rows   = pkast.WindowFrameRows
	Ranges = pkast.WindowFrameRange
	Groups = pkast.WindowFrameGroups
)

// FrameClause represents frame clause.
type FrameClause struct {
	pkast.BaseWindowFrameClause
}

// Restore implements Node interface.
func (n *FrameClause) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore FrameClause: nil frame")
	}
	if n.Extent.Start == nil || n.Extent.Start.WindowFrameBoundCore() == nil ||
		n.Extent.End == nil || n.Extent.End.WindowFrameBoundCore() == nil {
		return errors.New("window frame requires start and end bounds")
	}
	switch n.Type {
	case Rows:
		ctx.WriteKeyword("ROWS")
	case Ranges:
		ctx.WriteKeyword("RANGE")
	case Groups:
		ctx.WriteKeyword("GROUPS")
	default:
		return errors.New("invalid window frame type")
	}
	ctx.WriteKeyword(" BETWEEN ")
	if err := n.Extent.Start.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore FrameClause.Extent.Start")
	}
	ctx.WriteKeyword(" AND ")
	if err := n.Extent.End.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore FrameClause.Extent.End")
	}
	switch n.Exclusion {
	case pkast.WindowFrameExcludeNone:
	case pkast.WindowFrameExcludeCurrentRow:
		ctx.WriteKeyword(" EXCLUDE CURRENT ROW")
	case pkast.WindowFrameExcludeGroup:
		ctx.WriteKeyword(" EXCLUDE GROUP")
	case pkast.WindowFrameExcludeTies:
		ctx.WriteKeyword(" EXCLUDE TIES")
	case pkast.WindowFrameExcludeNoOthers:
		ctx.WriteKeyword(" EXCLUDE NO OTHERS")
	default:
		return errors.New("invalid window frame exclusion")
	}

	return ctx.Err()
}

// Format formats the FrameClause into a Writer.
func (n *FrameClause) Format(w io.Writer) {
	switch n.Type {
	case Rows:
		_, _ = fmt.Fprint(w, "ROWS")
	case Ranges:
		_, _ = fmt.Fprint(w, "RANGE")
	case Groups:
		_, _ = fmt.Fprint(w, "GROUPS")
	}
	_, _ = fmt.Fprint(w, " BETWEEN ")
	formatFrameBound(w, n.Extent.Start)
	_, _ = fmt.Fprint(w, " AND ")
	formatFrameBound(w, n.Extent.End)
	switch n.Exclusion {
	case pkast.WindowFrameExcludeNone:
	case pkast.WindowFrameExcludeCurrentRow:
		_, _ = fmt.Fprint(w, " EXCLUDE CURRENT ROW")
	case pkast.WindowFrameExcludeGroup:
		_, _ = fmt.Fprint(w, " EXCLUDE GROUP")
	case pkast.WindowFrameExcludeTies:
		_, _ = fmt.Fprint(w, " EXCLUDE TIES")
	case pkast.WindowFrameExcludeNoOthers:
		_, _ = fmt.Fprint(w, " EXCLUDE NO OTHERS")
	}
}

// Accept implements Node Accept interface.
func (n *FrameClause) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*FrameClause)
	if n.Extent.Start != nil {
		node, ok := n.Extent.Start.Accept(v)
		if !ok {
			return n, false
		}
		bound, ok := node.(pkast.WindowFrameBoundNode)
		if !ok || bound == nil || bound.WindowFrameBoundCore() == nil {
			return n, false
		}
		n.Extent.Start = bound
	}
	if n.Extent.End != nil {
		node, ok := n.Extent.End.Accept(v)
		if !ok {
			return n, false
		}
		bound, ok := node.(pkast.WindowFrameBoundNode)
		if !ok || bound == nil || bound.WindowFrameBoundCore() == nil {
			return n, false
		}
		n.Extent.End = bound
	}
	return v.Leave(n)
}

// FrameExtent represents frame extent.
type FrameExtent = pkast.WindowFrameExtent

func formatFrameBound(w io.Writer, bound pkast.WindowFrameBoundNode) {
	if b, ok := bound.(*FrameBound); ok {
		b.Format(w)
	}
}

// BoundType is the type of window function frame bound.
type BoundType = pkast.WindowFrameBoundType

// Frame bound types.
const (
	Following  = pkast.WindowFrameFollowing
	Preceding  = pkast.WindowFramePreceding
	CurrentRow = pkast.WindowFrameCurrentRow
)

// FrameBound represents frame bound.
type FrameBound struct {
	pkast.BaseWindowFrameBound
}

// WindowFrameBoundCore reports a missing bound before role-based dispatch.
func (n *FrameBound) WindowFrameBoundCore() *pkast.BaseWindowFrameBound {
	if n == nil {
		return nil
	}
	return &n.BaseWindowFrameBound
}

// Restore implements Node interface.
func (n *FrameBound) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore FrameBound: nil bound")
	}
	if n.Type != CurrentRow && n.Type != Preceding && n.Type != Following {
		return errors.New("invalid window frame bound type")
	}
	if n.Unbounded {
		ctx.WriteKeyword("UNBOUNDED")
	}
	switch n.Type {
	case CurrentRow:
		ctx.WriteKeyword("CURRENT ROW")
	case Preceding, Following:
		if n.Unit.Present() {
			ctx.WriteKeyword("INTERVAL ")
		}
		if n.Expr != nil {
			if err := n.Expr.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore FrameBound.Expr")
			}
		}
		if n.Unit.Present() {
			ctx.WritePlain(" ")
			ctx.WriteKeyword(n.Unit.Name)
		}
		if n.Type == Preceding {
			ctx.WriteKeyword(" PRECEDING")
		} else {
			ctx.WriteKeyword(" FOLLOWING")
		}
	}
	return ctx.Err()
}

// Format formats the FrameBound into a Writer.
func (n *FrameBound) Format(w io.Writer) {
	if n.Unbounded {
		_, _ = fmt.Fprint(w, "UNBOUNDED ")
	}
	switch n.Type {
	case CurrentRow:
		_, _ = fmt.Fprint(w, "CURRENT ROW")
	case Preceding, Following:
		if n.Unit.Present() {
			_, _ = fmt.Fprint(w, "INTERVAL ")
		}
		if n.Expr != nil {
			FormatExpr(n.Expr, w)
		}
		if n.Unit.Present() {
			_, _ = fmt.Fprint(w, " ")
			_, _ = fmt.Fprint(w, n.Unit.Name)
		}
		if n.Type == Preceding {
			_, _ = fmt.Fprint(w, " PRECEDING")
		} else {
			_, _ = fmt.Fprint(w, " FOLLOWING")
		}
	}
}

// Accept implements Node Accept interface.
func (n *FrameBound) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*FrameBound)
	if n.Expr != nil {
		node, ok := n.Expr.Accept(v)
		if !ok {
			return n, false
		}
		n.Expr = node.(ExprNode)
	}
	return v.Leave(n)
}
