package ast_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/format"
	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/driver"
)

// sharedPureExpr is an expression that only embeds the shared BaseExpr: it
// carries no legacy type, flag or format metadata, so it can only travel
// through the shared Node contract.
type sharedPureExpr struct {
	pkast.BaseExpr
	restoreErr bool
}

// Accept implements the shared Node contract.
func (n *sharedPureExpr) Accept(v pkast.Visitor) (pkast.Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*sharedPureExpr)
	return v.Leave(n)
}

// Restore implements the shared Node contract. The text is written before the
// failure on purpose: a failing restore must not leak partial output.
func (n *sharedPureExpr) Restore(ctx *format.RestoreCtx) error {
	ctx.WritePlain("PURE")
	if n.restoreErr {
		return errors.New("shared pure expression restore failure")
	}
	return ctx.Err()
}

// sharedVisitor records the visited nodes and optionally replaces the legacy
// value expressions with a pure node.
type sharedVisitor struct {
	entered     []pkast.Node
	replacement pkast.Node
	skipRoot    bool
	stop        bool
}

func (v *sharedVisitor) Enter(n pkast.Node) (pkast.Node, bool) {
	v.entered = append(v.entered, n)
	if v.skipRoot && len(v.entered) == 1 {
		return n, true
	}
	return n, false
}

func (v *sharedVisitor) Leave(n pkast.Node) (pkast.Node, bool) {
	if v.replacement != nil {
		if _, ok := n.(*ast.LiteralExpr); ok {
			return v.replacement, true
		}
	}
	return n, !v.stop
}

// TestSharedNodeIdentity proves the dialect aliases are the shared types.
func TestSharedNodeIdentity(t *testing.T) {
	t.Parallel()

	if reflect.TypeFor[ast.Node]() != reflect.TypeFor[pkast.Node]() {
		t.Error("dialect Node is not the shared Node type")
	}
	if reflect.TypeFor[ast.ExprNode]() != reflect.TypeFor[pkast.ExprNode]() {
		t.Error("dialect ExprNode is not the shared ExprNode type")
	}
	if reflect.TypeFor[ast.StmtNode]() != reflect.TypeFor[pkast.StmtNode]() {
		t.Error("dialect StmtNode is not the shared StmtNode type")
	}
	if reflect.TypeFor[ast.DMLNode]() != reflect.TypeFor[pkast.DMLNode]() {
		t.Error("dialect DMLNode is not the shared DMLNode type")
	}
	if reflect.TypeFor[ast.DDLNode]() != reflect.TypeFor[pkast.DDLNode]() {
		t.Error("dialect DDLNode is not the shared DDLNode type")
	}
	if reflect.TypeFor[ast.Visitor]() != reflect.TypeFor[pkast.Visitor]() {
		t.Error("dialect Visitor is not the shared Visitor type")
	}
	if reflect.TypeFor[ast.ResultSetNode]() != reflect.TypeFor[pkast.ResultSetNode]() {
		t.Error("dialect ResultSetNode is not the shared ResultSetNode type")
	}
	if reflect.TypeFor[ast.QueryNode]() != reflect.TypeFor[pkast.QueryNode]() {
		t.Error("dialect QueryNode is not the shared QueryNode type")
	}
	if reflect.TypeFor[ast.TableRefNode]() != reflect.TypeFor[pkast.TableRefNode]() {
		t.Error("dialect TableRefNode is not the shared TableRefNode type")
	}
	// The pure expression satisfies the dialect expression contract.
	var _ ast.ExprNode = (*sharedPureExpr)(nil)
}

// TestSharedVisitorPureChild checks that the shared visitor sees the outer
// dialect node and the pure child, and that Leave can replace a legacy value
// expression in place.
func TestSharedVisitorPureChild(t *testing.T) {
	t.Parallel()

	pure := &sharedPureExpr{}
	pure.SetText("pure")
	between := &ast.BetweenExpr{
		Expr:  pure,
		Left:  ast.NewValueExpr(int64(1)),
		Right: ast.NewValueExpr(int64(2)),
	}

	replacement := &sharedPureExpr{}
	replacement.SetText("replacement")
	v := &sharedVisitor{replacement: replacement}
	node, ok := between.Accept(v)
	if !ok {
		t.Fatal("traversal stopped unexpectedly")
	}
	if node != between {
		t.Fatalf("outer dialect node identity lost: got %T", node)
	}
	if len(v.entered) != 4 {
		t.Fatalf("visited %d nodes, want 4", len(v.entered))
	}
	if _, ok := v.entered[0].(*ast.BetweenExpr); !ok {
		t.Fatalf("first visited node is %T, want *ast.BetweenExpr", v.entered[0])
	}
	if _, ok := v.entered[1].(*sharedPureExpr); !ok {
		t.Fatalf("second visited node is %T, want *sharedPureExpr", v.entered[1])
	}
	if between.Expr != pure {
		t.Fatal("pure child was replaced")
	}
	if between.Left != replacement || between.Right != replacement {
		t.Fatalf("replacement not written back: left=%T right=%T", between.Left, between.Right)
	}

	t.Run("skip children", func(t *testing.T) {
		skipped := &sharedVisitor{skipRoot: true}
		tree := &ast.BetweenExpr{
			Expr:  &sharedPureExpr{},
			Left:  ast.NewValueExpr(int64(1)),
			Right: ast.NewValueExpr(int64(2)),
		}
		if _, ok := tree.Accept(skipped); !ok {
			t.Fatal("traversal stopped unexpectedly")
		}
		if len(skipped.entered) != 1 {
			t.Fatalf("visited %d nodes, want 1", len(skipped.entered))
		}
		if _, ok := skipped.entered[0].(*ast.BetweenExpr); !ok {
			t.Fatalf("first visited node is %T, want *ast.BetweenExpr", skipped.entered[0])
		}
	})

	t.Run("stop on leave", func(t *testing.T) {
		stopper := &sharedVisitor{stop: true}
		tree := &ast.BetweenExpr{
			Expr:  &sharedPureExpr{},
			Left:  ast.NewValueExpr(int64(1)),
			Right: ast.NewValueExpr(int64(2)),
		}
		if _, ok := tree.Accept(stopper); ok {
			t.Fatal("visitor stop was ignored")
		}
		if len(stopper.entered) != 2 {
			t.Fatalf("visited %d nodes, want 2", len(stopper.entered))
		}
	})
}

// TestSharedParamMarkerAccept checks that the visitor receives the outer
// parameter marker instead of the embedded value expression.
func TestSharedParamMarkerAccept(t *testing.T) {
	t.Parallel()

	marker := ast.NewParamMarkerExpr(0)
	v := &sharedVisitor{}
	node, ok := marker.Accept(v)
	if !ok {
		t.Fatal("param marker traversal failed")
	}
	if node != marker {
		t.Fatalf("param marker identity lost: got %T", node)
	}
	if len(v.entered) != 1 {
		t.Fatalf("visited %d nodes, want 1", len(v.entered))
	}
	if _, ok := v.entered[0].(*ast.ParamMarkerExpr); !ok {
		t.Fatalf("visited node is %T, want *ast.ParamMarkerExpr", v.entered[0])
	}
}

// TestSharedSpanAndText checks the shared source range and text contract.
func TestSharedSpanAndText(t *testing.T) {
	t.Parallel()

	pure := &sharedPureExpr{}
	unknown := pure.Span()
	if unknown != loc.InvalidSpan() || unknown.Start != -1 || unknown.End != -1 {
		t.Fatalf("unknown span is %v, want {-1 -1}", unknown)
	}
	pure.SetSpan(loc.Span{Start: 0, End: 0})
	if got := pure.Span(); got != (loc.Span{Start: 0, End: 0}) {
		t.Fatalf("empty span is %v, want [0 0)", got)
	}
	pure.SetSpan(loc.Span{Start: -1, End: -1})
	if got := pure.Span(); got != (loc.Span{Start: -1, End: -1}) {
		t.Fatalf("invalid span is %v, want {-1 -1}", got)
	}
	pure.SetText("hello")
	if got := pure.Text(); got != "hello" {
		t.Fatalf("text is %q, want %q", got, "hello")
	}
}

// TestSharedFormatExprAndFlag checks the legacy rendering helpers.
func TestSharedFormatExprAndFlag(t *testing.T) {
	t.Parallel()

	var sb strings.Builder
	ast.FormatExpr(ast.NewValueExpr("abc"), &sb)
	if got := sb.String(); got != `"abc"` {
		t.Fatalf("legacy string format is %q, want %q", got, `"abc"`)
	}
	sb.Reset()
	ast.FormatExpr(ast.NewValueExpr(int64(42)), &sb)
	if got := sb.String(); got != "42" {
		t.Fatalf("legacy int format is %q, want %q", got, "42")
	}

	// A node without the legacy Format method falls back to Restore.
	sb.Reset()
	ast.FormatExpr(&sharedPureExpr{}, &sb)
	if got := sb.String(); got != "PURE" {
		t.Fatalf("foreign format is %q, want %q", got, "PURE")
	}

	// A failing Restore is reported as a placeholder, partial output is dropped.
	sb.Reset()
	ast.FormatExpr(&sharedPureExpr{restoreErr: true}, &sb)
	if got := sb.String(); got != "<unsupported>" {
		t.Fatalf("failed foreign format is %q, want %q", got, "<unsupported>")
	}

	marker := ast.NewParamMarkerExpr(0)
	if got := ast.ExprFlag(marker); got != ast.FlagHasParamMarker {
		t.Fatalf("derived flag is %d, want %d", got, ast.FlagHasParamMarker)
	}
	if got := ast.ExprFlag(&sharedPureExpr{}); got != 0 {
		t.Fatalf("pure flag is %d, want 0", got)
	}
}

// TestSharedConstructors checks that the constructors work without any
// registration side effect and keep an existing value expression.
func TestSharedConstructors(t *testing.T) {
	t.Parallel()

	value := driver.NewValueExpr("abc")
	if got := driver.NewValueExpr(value); got != value {
		t.Fatal("driver.NewValueExpr did not keep the existing value expression")
	}
	// The driver value expression is an execution-layer bridge, not an AST node.
	built := ast.NewValueExpr("abc")
	literal, ok := built.(*ast.LiteralExpr)
	if !ok || literal.Kind != pkast.LiteralString || literal.Value != "abc" {
		t.Fatalf("dialect NewValueExpr returned %#v", built)
	}
	marker := ast.NewParamMarkerExpr(0)
	if marker.Index != 0 {
		t.Fatalf("dialect NewParamMarkerExpr returned index %d", marker.Index)
	}
}

// TestWindowFrameRestoreRejectsMissingBounds checks the interface bound slots
// that replaced the old value fields: a hand-built frame must fail cleanly
// instead of panicking during restore.
func TestWindowFrameRestoreRejectsMissingBounds(t *testing.T) {
	t.Parallel()

	var sb strings.Builder
	ctx := format.NewRestoreCtx(&sb, format.DialectConfig{}, 0)
	if err := (&ast.FrameClause{}).Restore(ctx); err == nil {
		t.Fatal("expected an error for a frame without bounds")
	}
}

// TestSharedBasesDoNotDispatch checks that shared bases never gain Node methods.
func TestSharedBasesDoNotDispatch(t *testing.T) {
	t.Parallel()

	nodeType := reflect.TypeFor[ast.Node]()
	tests := []struct {
		name string
		typ  reflect.Type
	}{
		{"BaseNode", reflect.TypeFor[*pkast.BaseNode]()},
		{"BaseExpr", reflect.TypeFor[*pkast.BaseExpr]()},
		{"BaseStmt", reflect.TypeFor[*pkast.BaseStmt]()},
		{"BaseDML", reflect.TypeFor[*pkast.BaseDML]()},
		{"BaseDDL", reflect.TypeFor[*pkast.BaseDDL]()},
		{"BaseResultSet", reflect.TypeFor[*pkast.BaseResultSet]()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.typ.Implements(nodeType) {
				t.Errorf("%s implements ast.Node; promoted dispatch would lose outer identity", tt.name)
			}
		})
	}
}
