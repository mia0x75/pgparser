package ast

import (
	"fmt"
	"io"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/pgparser/errors"
	"github.com/mia0x75/pgparser/format"
)

var (
	_ Node                            = &WindowSpec{}
	_ Node                            = &PartitionByClause{}
	_ pkast.WindowPartitionClauseNode = (*PartitionByClause)(nil)
	_ pkast.WindowPartitionItemNode   = (*PartitionItem)(nil)
	_ Node                            = &FrameClause{}
	_ Node                            = &FrameBound{}
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

func (n *WindowSpec) syncSharedCore() {
	if n == nil {
		return
	}
	if n.PartitionBy == nil {
		n.BaseWindowSpec.PartitionBy = nil
	} else {
		n.BaseWindowSpec.PartitionBy = n.PartitionBy
	}
	if n.OrderBy == nil {
		n.BaseWindowSpec.OrderBy = nil
	} else {
		n.BaseWindowSpec.OrderBy = n.OrderBy
	}
	if n.Frame == nil {
		n.BaseWindowSpec.Frame = nil
	} else {
		n.BaseWindowSpec.Frame = n.Frame
	}
}

func (n *WindowSpec) syncDialectFields() bool {
	if n.BaseWindowSpec.PartitionBy != nil {
		clause, ok := n.BaseWindowSpec.PartitionBy.(*PartitionByClause)
		if !ok || clause == nil {
			return false
		}
		n.PartitionBy = clause
	} else if n.PartitionBy != nil {
		n.BaseWindowSpec.PartitionBy = n.PartitionBy
	}
	if n.BaseWindowSpec.OrderBy != nil {
		clause, ok := n.BaseWindowSpec.OrderBy.(*OrderByClause)
		if !ok || clause == nil {
			return false
		}
		n.OrderBy = clause
	} else if n.OrderBy != nil {
		n.BaseWindowSpec.OrderBy = n.OrderBy
	}
	if n.BaseWindowSpec.Frame != nil {
		clause, ok := n.BaseWindowSpec.Frame.(*FrameClause)
		if !ok || clause == nil {
			return false
		}
		n.Frame = clause
	} else if n.Frame != nil {
		n.BaseWindowSpec.Frame = n.Frame
	}
	return true
}

func (n *WindowSpec) reconcileDialectFields(
	beforePartition *PartitionByClause,
	beforePartitionCore pkast.WindowPartitionClauseNode,
	beforeOrder *OrderByClause,
	beforeOrderCore pkast.OrderByClauseNode,
	beforeFrame *FrameClause,
	beforeFrameCore pkast.WindowFrameClauseNode,
) bool {
	switch {
	case n.PartitionBy != beforePartition && n.BaseWindowSpec.PartitionBy == beforePartitionCore:
		n.BaseWindowSpec.PartitionBy = n.PartitionBy
	case n.PartitionBy == beforePartition && n.BaseWindowSpec.PartitionBy != beforePartitionCore:
		partition, ok := n.BaseWindowSpec.PartitionBy.(*PartitionByClause)
		if !ok || partition == nil {
			return false
		}
		n.PartitionBy = partition
	case n.PartitionBy != nil && n.BaseWindowSpec.PartitionBy != n.PartitionBy:
		return false
	}

	switch {
	case n.OrderBy != beforeOrder && n.BaseWindowSpec.OrderBy == beforeOrderCore:
		n.BaseWindowSpec.OrderBy = n.OrderBy
	case n.OrderBy == beforeOrder && n.BaseWindowSpec.OrderBy != beforeOrderCore:
		order, ok := n.BaseWindowSpec.OrderBy.(*OrderByClause)
		if !ok || order == nil {
			return false
		}
		n.OrderBy = order
	case n.OrderBy != nil && n.BaseWindowSpec.OrderBy != n.OrderBy:
		return false
	}

	switch {
	case n.Frame != beforeFrame && n.BaseWindowSpec.Frame == beforeFrameCore:
		n.BaseWindowSpec.Frame = n.Frame
	case n.Frame == beforeFrame && n.BaseWindowSpec.Frame != beforeFrameCore:
		frame, ok := n.BaseWindowSpec.Frame.(*FrameClause)
		if !ok || frame == nil {
			return false
		}
		n.Frame = frame
	case n.Frame != nil && n.BaseWindowSpec.Frame != n.Frame:
		return false
	}
	return true
}

// Restore implements Node interface.
func (n *WindowSpec) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

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
	if n == nil {
		return nil, false
	}
	if !n.syncDialectFields() {
		return n, false
	}
	beforePartition := n.PartitionBy
	beforePartitionCore := n.BaseWindowSpec.PartitionBy
	beforeOrder := n.OrderBy
	beforeOrderCore := n.BaseWindowSpec.OrderBy
	beforeFrame := n.Frame
	beforeFrameCore := n.BaseWindowSpec.Frame
	newNode, skipChildren := v.Enter(n)
	spec, ok := newNode.(*WindowSpec)
	if !ok || spec == nil {
		return newNode, false
	}
	n = spec
	if !n.reconcileDialectFields(
		beforePartition,
		beforePartitionCore,
		beforeOrder,
		beforeOrderCore,
		beforeFrame,
		beforeFrameCore,
	) {
		return n, false
	}
	if skipChildren {
		n.syncSharedCore()
		return v.Leave(n)
	}
	if n.PartitionBy != nil {
		node, ok := n.PartitionBy.Accept(v)
		if !ok {
			return n, false
		}
		n.PartitionBy = node.(*PartitionByClause)
		n.PartitionBy.syncSharedCore()
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
	n.syncSharedCore()
	return v.Leave(n)
}

// PartitionByClause represents partition by clause.
type PartitionByClause struct {
	pkast.BaseWindowPartitionClause

	// Items preserves the PostgreSQL-specific AST view.
	Items []*OrderItem
}

// WindowPartitionClauseCore returns the shared partition data.
func (n *PartitionByClause) WindowPartitionClauseCore() *pkast.BaseWindowPartitionClause {
	if n == nil {
		return nil
	}
	return &n.BaseWindowPartitionClause
}

func (n *PartitionByClause) syncSharedCore() {
	if n == nil {
		return
	}
	if len(n.BaseWindowPartitionClause.Items) != len(n.Items) {
		n.BaseWindowPartitionClause.Items = make([]pkast.WindowPartitionItemNode, len(n.Items))
	}
	for i, item := range n.Items {
		if item == nil {
			n.BaseWindowPartitionClause.Items[i] = nil
			continue
		}
		shared, ok := n.BaseWindowPartitionClause.Items[i].(*PartitionItem)
		if !ok || shared == nil {
			shared = &PartitionItem{}
			n.BaseWindowPartitionClause.Items[i] = shared
		}
		shared.Expr = item.Expr
	}
}

func (n *PartitionByClause) syncDialectItems() bool {
	if len(n.BaseWindowPartitionClause.Items) == 0 {
		return true
	}
	if len(n.Items) != len(n.BaseWindowPartitionClause.Items) {
		n.Items = make([]*OrderItem, len(n.BaseWindowPartitionClause.Items))
	}
	for i, item := range n.BaseWindowPartitionClause.Items {
		shared, ok := item.(*PartitionItem)
		if !ok || shared == nil {
			return false
		}
		if n.Items[i] == nil {
			n.Items[i] = &OrderItem{}
		}
		n.Items[i].Expr = shared.Expr
	}
	return true
}

// Restore implements Node interface.
func (n *PartitionByClause) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

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
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	clause, ok := newNode.(*PartitionByClause)
	if !ok || clause == nil {
		return newNode, false
	}
	n = clause
	if !n.syncDialectItems() {
		return n, false
	}
	if skipChildren {
		n.syncSharedCore()
		return v.Leave(n)
	}
	for i, val := range n.Items {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Items[i] = node.(*OrderItem)
		shared := n.BaseWindowPartitionClause.Items[i]
		if shared == nil || shared.WindowPartitionItemCore() == nil {
			return n, false
		}
		shared.WindowPartitionItemCore().Expr = n.Items[i].Expr
	}
	return v.Leave(n)
}

// PartitionItem is a shared-core partition expression.
type PartitionItem struct {
	pkast.BaseWindowPartitionItem
}

func (n *PartitionItem) WindowPartitionItemCore() *pkast.BaseWindowPartitionItem {
	if n == nil {
		return nil
	}
	return &n.BaseWindowPartitionItem
}

func (n *PartitionItem) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return restorePGNode(ctx, n)
	}

	if n == nil || n.Expr == nil {
		return errors.New("restore partition item: missing expression")
	}
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore partition item expression")
	}
	return ctx.Err()
}

func (n *PartitionItem) Format(w io.Writer) {
	if n != nil && n.Expr != nil {
		FormatExpr(n.Expr, w)
	}
}

func (n *PartitionItem) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	item, ok := newNode.(*PartitionItem)
	if !ok || item == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(item)
	}
	if item.Expr == nil {
		return item, false
	}
	updated, ok := item.Expr.Accept(v)
	if !ok {
		return item, false
	}
	expr, ok := updated.(ExprNode)
	if !ok || expr == nil {
		return item, false
	}
	item.Expr = expr
	return v.Leave(item)
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
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*FrameClause)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
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
	if n == nil {
		return nil, false
	}

	newNode, skipChildren := v.Enter(n)

	enteredNode, enteredOK := newNode.(*FrameBound)
	if !enteredOK || enteredNode == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(enteredNode)
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
