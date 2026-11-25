package ast

import (
	"fmt"
	"io"
	"strings"

	"github.com/mia0x75/pgparser/format"
)

// ExprFlag reads the legacy dialect cache without widening the shared ExprNode.
// shortcut: only unmigrated dialect nodes cache flags; remove after AST migration.
func ExprFlag(expr ExprNode) uint64 {
	if legacy, ok := expr.(interface{ GetFlag() uint64 }); ok {
		return legacy.GetFlag()
	}
	return 0
}

// FormatExpr preserves legacy rendering without widening the shared ExprNode.
// Use Restore when output errors must be reported.
func FormatExpr(expr ExprNode, w io.Writer) {
	if legacy, ok := expr.(interface{ Format(io.Writer) }); ok {
		legacy.Format(w)
		return
	}
	var out strings.Builder
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &out)
	if err := expr.Restore(ctx); err != nil {
		_, _ = fmt.Fprint(w, "<unsupported>")
		return
	}
	_, _ = io.WriteString(w, out.String())
}
