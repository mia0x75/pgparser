package ast

import (
	"fmt"

	"github.com/mia0x75/pgparser/format"
)

func restorePGNode(ctx *format.RestoreCtx, node Node) error {
	switch n := node.(type) {
	case *String:
		ctx.WriteString(n.Str)
	case *Integer:
		ctx.WritePlainf("%d", n.Ival)
	case *Float:
		ctx.WritePlain(n.Fval)
	case *Boolean:
		if n.Boolval {
			ctx.WriteKeyWord("TRUE")
		} else {
			ctx.WriteKeyWord("FALSE")
		}
	case *RoleSpec:
		switch RoleSpecType(n.Roletype) {
		case ROLESPEC_CSTRING:
			ctx.WriteName(n.Rolename)
		case ROLESPEC_CURRENT_ROLE:
			ctx.WriteKeyWord("CURRENT_ROLE")
		case ROLESPEC_CURRENT_USER:
			ctx.WriteKeyWord("CURRENT_USER")
		case ROLESPEC_SESSION_USER:
			ctx.WriteKeyWord("SESSION_USER")
		case ROLESPEC_PUBLIC:
			ctx.WriteKeyWord("PUBLIC")
		default:
			return fmt.Errorf("unsupported role target %d", n.Roletype)
		}
	case *List:
		for i, child := range n.Items {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if err := child.Restore(ctx); err != nil {
				return err
			}
		}
	case *IntoClause:
		if n.Rel == nil {
			return fmt.Errorf("SELECT INTO has no target")
		}
		if n.Rel.Relpersistence == 't' {
			ctx.WriteKeyWord("TEMPORARY ")
		}
		if n.Rel.Relpersistence == 'u' {
			ctx.WriteKeyWord("UNLOGGED ")
		}
		if err := n.Rel.Restore(ctx); err != nil {
			return err
		}
		return nil
	case *SQLValueFunction:
		return restoreSQLValue(ctx, n)
	case *JsonScalarExpr:
		if n.Output != nil {
			return fmt.Errorf("JSON_SCALAR output restoration is unsupported")
		}
		ctx.WriteKeyWord("JSON_SCALAR(")
		if err := n.Expr.Restore(ctx); err != nil {
			return err
		}
		ctx.WritePlain(")")
	case *JsonValueExpr:
		if n.FormatClause != nil {
			return fmt.Errorf("JSON format restoration is unsupported")
		}
		if n.RawExpr == nil {
			return fmt.Errorf("JSON value has no expression")
		}
		return n.RawExpr.Restore(ctx)
	case *JsonBehavior:
		switch n.Btype {
		case JSON_BEHAVIOR_ERROR:
			ctx.WriteKeyWord("ERROR")
		case JSON_BEHAVIOR_NULL:
			ctx.WriteKeyWord("NULL")
		case JSON_BEHAVIOR_TRUE:
			ctx.WriteKeyWord("TRUE")
		case JSON_BEHAVIOR_FALSE:
			ctx.WriteKeyWord("FALSE")
		case JSON_BEHAVIOR_UNKNOWN:
			ctx.WriteKeyWord("UNKNOWN")
		case JSON_BEHAVIOR_EMPTY_ARRAY:
			ctx.WriteKeyWord("EMPTY ARRAY")
		case JSON_BEHAVIOR_EMPTY_OBJECT:
			ctx.WriteKeyWord("EMPTY OBJECT")
		case JSON_BEHAVIOR_DEFAULT:
			return fmt.Errorf("JSON DEFAULT restoration is unsupported")
		default:
			return fmt.Errorf("unsupported JSON behavior %d", n.Btype)
		}
	case *XmlExpr:
		return restoreXML(ctx, n)
	default:
		return fmt.Errorf("PostgreSQL restoration is unsupported for %T", node)
	}
	return nil
}

func restoreSQLValue(ctx *format.RestoreCtx, n *SQLValueFunction) error {
	var name string
	switch n.Op {
	case SVFOP_CURRENT_DATE:
		name = "CURRENT_DATE"
	case SVFOP_CURRENT_TIME, SVFOP_CURRENT_TIME_N:
		name = "CURRENT_TIME"
	case SVFOP_CURRENT_TIMESTAMP, SVFOP_CURRENT_TIMESTAMP_N:
		name = "CURRENT_TIMESTAMP"
	case SVFOP_LOCALTIME, SVFOP_LOCALTIME_N:
		name = "LOCALTIME"
	case SVFOP_LOCALTIMESTAMP, SVFOP_LOCALTIMESTAMP_N:
		name = "LOCALTIMESTAMP"
	case SVFOP_CURRENT_ROLE:
		name = "CURRENT_ROLE"
	case SVFOP_CURRENT_USER:
		name = "CURRENT_USER"
	case SVFOP_SESSION_USER:
		name = "SESSION_USER"
	case SVFOP_USER:
		name = "USER"
	case SVFOP_CURRENT_CATALOG:
		name = "CURRENT_CATALOG"
	case SVFOP_CURRENT_SCHEMA:
		name = "CURRENT_SCHEMA"
	default:
		return fmt.Errorf("unsupported SQL value operation %d", n.Op)
	}
	ctx.WriteKeyWord(name)
	if n.Typmod >= 0 {
		ctx.WritePlainf("(%d)", n.Typmod)
	}
	return nil
}

func restoreXML(ctx *format.RestoreCtx, n *XmlExpr) error {
	if n.Op != IS_XMLPARSE || n.Args == nil || len(n.Args.Items) != 2 {
		return fmt.Errorf("XML operation %d restoration is unsupported", n.Op)
	}
	ctx.WriteKeyWord("XMLPARSE(")
	if n.Xmloption == XMLOPTION_DOCUMENT {
		ctx.WriteKeyWord("DOCUMENT ")
	} else {
		ctx.WriteKeyWord("CONTENT ")
	}
	if err := n.Args.Items[0].Restore(ctx); err != nil {
		return err
	}
	if value, ok := n.Args.Items[1].(ValueExpr); ok && (value.GetValue() == true || value.GetValue() == int64(1)) {
		ctx.WriteKeyWord(" PRESERVE WHITESPACE")
	} else {
		ctx.WriteKeyWord(" STRIP WHITESPACE")
	}
	ctx.WritePlain(")")
	return nil
}
