package parser

import (
	"strings"

	"github.com/mia0x75/pgparser/ast"
)

func makeReferenceDef(names, columns *ast.List, match int64, actions *KeyActions) *ast.ReferenceDef {
	ref := &ast.ReferenceDef{
		Table:         tableNameFromList(names),
		IndexColNames: makeIndexColNameList(columns),
		Match:         byte(match),
	}
	if actions != nil {
		if action := actions.DeleteAction; action != nil {
			ref.OnDelete = &ast.OnDeleteOpt{
				ReferOpt: referenceAction(action.Action),
				Columns:  columnNamesFromList(action.Cols),
			}
		}
		if action := actions.UpdateAction; action != nil {
			ref.OnUpdate = &ast.OnUpdateOpt{ReferOpt: referenceAction(action.Action)}
		}
	}
	return ref
}

func referenceAction(action byte) ast.ReferOptionType {
	switch action {
	case 'r':
		return ast.ReferOptionRestrict
	case 'c':
		return ast.ReferOptionCascade
	case 'n':
		return ast.ReferOptionSetNull
	case 'd':
		return ast.ReferOptionSetDefault
	default:
		return ast.ReferOptionNoAction
	}
}

// columnOptions returns the column options of a list.
func columnOptions(l *ast.List) []*ast.ColumnOption {
	if l == nil {
		return nil
	}
	var result []*ast.ColumnOption
	for _, item := range l.Items {
		if opt, ok := item.(*ast.ColumnOption); ok {
			result = append(result, opt)
		}
	}
	return result
}

// constraintOptions returns the options of a constraint attribute list.
func constraintOptions(bits int64) string {
	var parts []string
	if bits&int64(ast.CAS_DEFERRABLE) != 0 {
		parts = append(parts, "DEFERRABLE")
	} else {
		parts = append(parts, "NOT DEFERRABLE")
	}
	if bits&int64(ast.CAS_INITIALLY_DEFERRED) != 0 {
		parts = append(parts, "initially deferred")
	} else {
		parts = append(parts, "initially immediate")
	}
	return strings.Join(parts, " ")
}
