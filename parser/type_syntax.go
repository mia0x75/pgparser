package parser

import (
	"fmt"
	"strings"

	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/format"
	"github.com/mia0x75/pgparser/types"
)

// typeNameNode returns the node of a field type.
func typeNameNode(tp *types.FieldType) ast.Node {
	return &ast.DataType{Type: tp}
}

// typeNameFromList returns the field type of a name list.
func typeNameFromList(l *ast.List) *types.FieldType {
	if l == nil {
		return newDataType("text", -1, -1)
	}
	var parts []string
	for _, item := range l.Items {
		if s, ok := item.(*ast.String); ok {
			parts = append(parts, s.Str)
		}
	}
	tp := newDataType(strings.Join(parts, "."), -1, -1)
	tp.PGName = parts
	return tp
}

// collationName returns the name of a collation clause.
func collationName(n ast.Node) string {
	if n == nil {
		return ""
	}
	list, ok := n.(*ast.List)
	if !ok {
		return ""
	}
	var parts []string
	for _, item := range list.Items {
		s, ok := item.(*ast.String)
		if !ok {
			continue
		}
		parts = append(parts, s.Str)
	}
	return strings.Join(parts, ".")
}

func setTypeModifiers(tp *types.FieldType, mods []ast.ExprNode) error {
	tp.PGModifiers = nil
	for _, mod := range mods {
		var out strings.Builder
		if err := mod.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil {
			return err
		}
		tp.PGModifiers = append(tp.PGModifiers, out.String())
	}
	return nil
}

func setIntervalFields(tp *types.FieldType, fields *ast.List) {
	if fields == nil || len(fields.Items) == 0 {
		return
	}
	value, ok := fields.Items[0].(ast.ValueExpr)
	if !ok {
		return
	}
	mask, ok := value.GetValue().(int64)
	if !ok {
		return
	}
	units := []struct {
		mask int64
		name string
	}{
		{int64(ast.INTERVAL_MASK_YEAR), "YEAR"}, {int64(ast.INTERVAL_MASK_MONTH), "MONTH"},
		{int64(ast.INTERVAL_MASK_DAY), "DAY"}, {int64(ast.INTERVAL_MASK_HOUR), "HOUR"},
		{int64(ast.INTERVAL_MASK_MINUTE), "MINUTE"}, {int64(ast.INTERVAL_MASK_SECOND), "SECOND"},
	}
	var first, last string
	for _, unit := range units {
		if mask&unit.mask != 0 {
			if first == "" {
				first = unit.name
			}
			last = unit.name
		}
	}
	tp.PGInterval = first
	if first != last {
		tp.PGInterval += " TO " + last
	}
	if len(fields.Items) > 1 {
		if precision, ok := fields.Items[1].(ast.ValueExpr); ok {
			tp.PGInterval += fmt.Sprintf("(%v)", precision.GetValue())
		}
	}
}

func arrayBounds(list *ast.List) []int64 {
	if list == nil {
		return nil
	}
	bounds := make([]int64, 0, len(list.Items))
	for _, item := range list.Items {
		if bound, ok := item.(*ast.Integer); ok {
			bounds = append(bounds, bound.Ival)
		}
	}
	return bounds
}
