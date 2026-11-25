package ast

import (
	pkast "github.com/mia0x75/parserkit/ast"

	"github.com/mia0x75/pgparser/format"
)

// QualifiedName keeps the shared syntax-only name value.
type QualifiedName = pkast.QualifiedName

func restoreQualifiedName(ctx *format.RestoreCtx, name QualifiedName) error {
	for i, part := range name.Parts {
		if i != 0 {
			ctx.WritePlain(".")
		}
		ctx.WriteIdent(part)
	}
	return ctx.Err()
}
