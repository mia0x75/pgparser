package ast

import (
	"strings"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"

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

// restoreFunctionName keeps a single bare name readable and quotes every other
// part so reserved words and case survive identifier folding.
func restoreFunctionName(ctx *format.RestoreCtx, name QualifiedName) error {
	if len(name.Parts) == 1 {
		part := name.Parts[0]
		if part.Quote == lex.QuoteUnquoted && !NeedsQuotedFuncName(part.Name) {
			ctx.WriteKeyword(part.Name)
			return ctx.Err()
		}
	}
	for i, part := range name.Parts {
		if i != 0 {
			ctx.WritePlain(".")
		}
		if part.Quote != lex.QuoteUnquoted {
			ctx.WriteIdent(part)
		} else {
			ctx.WriteName(part.Name)
		}
	}
	return ctx.Err()
}

func qualifiedNameString(name QualifiedName) string {
	parts := make([]string, 0, len(name.Parts))
	for _, part := range name.Parts {
		parts = append(parts, part.Name)
	}
	return strings.Join(parts, ".")
}

func nameFromDotted(name string) QualifiedName {
	return nameFromStrings(strings.Split(name, "."))
}

func nameFromStrings(parts []string) QualifiedName {
	idents := make([]lex.Ident, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			idents = append(idents, lex.Ident{Name: part})
		}
	}
	return QualifiedName{Parts: idents}
}
