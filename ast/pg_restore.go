package ast

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mia0x75/pgparser/format"
)

var ErrUnsupportedRestore = errors.New("SQL restoration is unsupported")

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
			ctx.WriteKeyword("TRUE")
		} else {
			ctx.WriteKeyword("FALSE")
		}
	case *A_Star:
		ctx.WritePlain("*")
	case *A_Indirection:
		arg, ok := n.Arg.(ExprNode)
		if !ok || arg == nil {
			return fmt.Errorf("indirection has no expression")
		}
		if err := arg.Restore(ctx); err != nil {
			return err
		}
		if n.Indirection == nil {
			return ctx.Err()
		}
		for _, item := range n.Indirection.Items {
			switch v := item.(type) {
			case *String:
				ctx.WritePlain(".")
				ctx.WriteName(v.Str)
			case *A_Star:
				ctx.WritePlain(".*")
			case *A_Indices:
				if err := restoreIndices(ctx, v); err != nil {
					return err
				}
			default:
				return fmt.Errorf("%w: indirection element %T", ErrUnsupportedRestore, item)
			}
		}
	case *A_Const:
		if n.Isnull {
			ctx.WriteKeyword("NULL")
		} else {
			if n.Val == nil {
				return fmt.Errorf("constant has no value")
			}
			return n.Val.Restore(ctx)
		}
	case *DefElem:
		if n.Defname == "" || n.Defaction != int(DEFELEM_UNSPEC) {
			return fmt.Errorf("invalid relation option")
		}
		if n.Defnamespace != "" {
			ctx.WriteName(n.Defnamespace)
			ctx.WritePlain(".")
		}
		ctx.WriteName(n.Defname)
		if n.Arg != nil {
			ctx.WritePlain(" = ")
			return n.Arg.Restore(ctx)
		}
	case *RoleSpec:
		switch RoleSpecType(n.Roletype) {
		case ROLESPEC_CSTRING:
			ctx.WriteName(n.Rolename)
		case ROLESPEC_CURRENT_ROLE:
			ctx.WriteKeyword("CURRENT_ROLE")
		case ROLESPEC_CURRENT_USER:
			ctx.WriteKeyword("CURRENT_USER")
		case ROLESPEC_SESSION_USER:
			ctx.WriteKeyword("SESSION_USER")
		case ROLESPEC_PUBLIC:
			ctx.WriteKeyword("PUBLIC")
		default:
			return fmt.Errorf("unsupported role target %d", n.Roletype)
		}
	case *List:
		for i, child := range n.Items {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if child == nil {
				return fmt.Errorf("list has a nil child at index %d", i)
			}
			if err := child.Restore(ctx); err != nil {
				return err
			}
		}
	case *FunctionParameter:
		if n.ArgType == nil {
			return fmt.Errorf("routine parameter has no type")
		}
		switch n.Mode {
		case FUNC_PARAM_IN, FUNC_PARAM_TABLE:
		case FUNC_PARAM_OUT:
			ctx.WriteKeyword("OUT ")
		case FUNC_PARAM_INOUT:
			ctx.WriteKeyword("INOUT ")
		case FUNC_PARAM_VARIADIC:
			ctx.WriteKeyword("VARIADIC ")
		default:
			return fmt.Errorf("invalid parameter mode %d", n.Mode)
		}
		if n.Name != "" {
			ctx.WriteName(n.Name)
			ctx.WritePlain(" ")
		}
		if err := n.ArgType.Restore(ctx); err != nil {
			return err
		}
		if n.Defexpr != nil {
			ctx.WriteKeyword(" DEFAULT ")
			return n.Defexpr.Restore(ctx)
		}
	case *VacuumRelation:
		if n.Relation == nil {
			return fmt.Errorf("maintenance relation has no name")
		}
		if n.Oid != 0 {
			return fmt.Errorf("%w: maintenance relation OID", ErrUnsupportedRestore)
		}
		if err := n.Relation.Restore(ctx); err != nil {
			return err
		}
		if n.VaCols != nil {
			ctx.WritePlain(" (")
			for i, item := range n.VaCols.Items {
				name, ok := item.(*String)
				if !ok || name == nil {
					return fmt.Errorf("invalid maintenance column")
				}
				if i > 0 {
					ctx.WritePlain(", ")
				}
				ctx.WriteName(name.Str)
			}
			ctx.WritePlain(")")
		}
	case *PartitionSpec:
		switch strings.ToLower(n.Strategy) {
		case "r", "range":
			ctx.WriteKeyword("RANGE")
		case "l", "list":
			ctx.WriteKeyword("LIST")
		case "h", "hash":
			ctx.WriteKeyword("HASH")
		default:
			if n.Strategy == "" {
				return fmt.Errorf("partition strategy is missing")
			}
			ctx.WriteName(n.Strategy)
		}
		if n.PartParams == nil || len(n.PartParams.Items) == 0 {
			return fmt.Errorf("partition has no keys")
		}
		ctx.WritePlain(" (")
		if err := n.PartParams.Restore(ctx); err != nil {
			return err
		}
		ctx.WritePlain(")")
	case *PartitionElem:
		switch {
		case n.Name != "":
			ctx.WriteName(n.Name)
		case n.Expr != nil:
			ctx.WritePlain("(")
			if err := n.Expr.Restore(ctx); err != nil {
				return err
			}
			ctx.WritePlain(")")
		default:
			return fmt.Errorf("partition key has no expression")
		}
		if n.Collation != nil {
			ctx.WriteKeyword(" COLLATE ")
			if err := restoreNameList(ctx, n.Collation); err != nil {
				return err
			}
		}
		if n.Opclass != nil {
			ctx.WritePlain(" ")
			if err := restoreNameList(ctx, n.Opclass); err != nil {
				return err
			}
		}
	case *ObjectWithArgs:
		if err := restoreNameList(ctx, n.Objname); err != nil {
			return err
		}
		if !n.ArgsUnspecified {
			ctx.WritePlain("(")
			if n.Objargs != nil {
				if err := n.Objargs.Restore(ctx); err != nil {
					return err
				}
			}
			ctx.WritePlain(")")
		}
	case *VariableSetStmt:
		return restoreVariableSet(ctx, n)
	case *AlterSeqStmt:
		ctx.WriteKeyword("ALTER SEQUENCE ")
		if n.MissingOk {
			ctx.WriteKeyword("IF EXISTS ")
		}
		if n.Sequence == nil {
			return fmt.Errorf("ALTER SEQUENCE has no target")
		}
		if err := n.Sequence.Restore(ctx); err != nil {
			return err
		}
		return restoreSeqOptions(ctx, n.Options)
	case *CreateTrigStmt:
		return restoreTrigger(ctx, n)
	case *TriggerTransition:
		if n.Name == "" {
			return fmt.Errorf("trigger transition has no name")
		}
		if n.IsNew {
			ctx.WriteKeyword("NEW ")
		} else {
			ctx.WriteKeyword("OLD ")
		}
		if n.IsTable {
			ctx.WriteKeyword("TABLE AS ")
		} else {
			ctx.WriteKeyword("ROW AS ")
		}
		ctx.WriteName(n.Name)
	case *LockingClause:
		switch LockClauseStrength(n.Strength) {
		case LCS_FORKEYSHARE:
			ctx.WriteKeyword("FOR KEY SHARE")
		case LCS_FORSHARE:
			ctx.WriteKeyword("FOR SHARE")
		case LCS_FORNOKEYUPDATE:
			ctx.WriteKeyword("FOR NO KEY UPDATE")
		case LCS_FORUPDATE:
			ctx.WriteKeyword("FOR UPDATE")
		default:
			return fmt.Errorf("invalid lock strength %d", n.Strength)
		}
		if n.LockedRels != nil {
			ctx.WriteKeyword(" OF ")
			for i, rel := range n.LockedRels.Items {
				if i > 0 {
					ctx.WritePlain(", ")
				}
				if rel == nil {
					return fmt.Errorf("lock has a nil relation")
				}
				if err := rel.Restore(ctx); err != nil {
					return err
				}
			}
		}
		switch LockWaitPolicy(n.WaitPolicy) {
		case LockWaitBlock:
		case LockWaitError:
			ctx.WriteKeyword(" NOWAIT")
		case LockWaitSkip:
			ctx.WriteKeyword(" SKIP LOCKED")
		default:
			return fmt.Errorf("invalid lock wait policy %d", n.WaitPolicy)
		}
	case *RangeVar:
		if n.Catalogname != "" {
			ctx.WriteName(n.Catalogname)
			ctx.WritePlain(".")
		}
		if n.Schemaname != "" {
			ctx.WriteName(n.Schemaname)
			ctx.WritePlain(".")
		}
		if n.Relname == "" {
			return fmt.Errorf("relation has no name")
		}
		ctx.WriteName(n.Relname)
	case *IntoClause:
		if n.Rel == nil {
			return fmt.Errorf("SELECT INTO has no target")
		}
		if n.Rel.Relpersistence == 't' {
			ctx.WriteKeyword("TEMPORARY ")
		}
		if n.Rel.Relpersistence == 'u' {
			ctx.WriteKeyword("UNLOGGED ")
		}
		if err := n.Rel.Restore(ctx); err != nil {
			return err
		}
		return ctx.Err()
	case *SQLValueFunction:
		return restoreSQLValue(ctx, n)
	case *JsonScalarExpr:
		if n.Output != nil {
			return fmt.Errorf("%w: JSON_SCALAR output", ErrUnsupportedRestore)
		}
		ctx.WriteKeyword("JSON_SCALAR(")
		if err := n.Expr.Restore(ctx); err != nil {
			return err
		}
		ctx.WritePlain(")")
	case *JsonValueExpr:
		if n.FormatClause != nil {
			return fmt.Errorf("%w: JSON format", ErrUnsupportedRestore)
		}
		if n.RawExpr == nil {
			return fmt.Errorf("JSON value has no expression")
		}
		return n.RawExpr.Restore(ctx)
	case *JsonBehavior:
		switch n.Btype {
		case JSON_BEHAVIOR_ERROR:
			ctx.WriteKeyword("ERROR")
		case JSON_BEHAVIOR_NULL:
			ctx.WriteKeyword("NULL")
		case JSON_BEHAVIOR_TRUE:
			ctx.WriteKeyword("TRUE")
		case JSON_BEHAVIOR_FALSE:
			ctx.WriteKeyword("FALSE")
		case JSON_BEHAVIOR_UNKNOWN:
			ctx.WriteKeyword("UNKNOWN")
		case JSON_BEHAVIOR_EMPTY_ARRAY:
			ctx.WriteKeyword("EMPTY ARRAY")
		case JSON_BEHAVIOR_EMPTY_OBJECT:
			ctx.WriteKeyword("EMPTY OBJECT")
		case JSON_BEHAVIOR_DEFAULT:
			return fmt.Errorf("%w: JSON DEFAULT", ErrUnsupportedRestore)
		default:
			return fmt.Errorf("unsupported JSON behavior %d", n.Btype)
		}
	case *XmlExpr:
		return restoreXML(ctx, n)
	default:
		return fmt.Errorf("%w for %T", ErrUnsupportedRestore, node)
	}
	return ctx.Err()
}

func restoreLocking(ctx *format.RestoreCtx, locks *List) error {
	for _, item := range locks.Items {
		lock, ok := item.(*LockingClause)
		if !ok || lock == nil {
			return fmt.Errorf("invalid locking clause %T", item)
		}
		ctx.WritePlain(" ")
		if err := lock.Restore(ctx); err != nil {
			return err
		}
	}
	return ctx.Err()
}

func restoreRelOptions(ctx *format.RestoreCtx, options *List) error {
	if options == nil || len(options.Items) == 0 {
		return fmt.Errorf("relation options are missing")
	}
	ctx.WritePlain("(")
	if err := options.Restore(ctx); err != nil {
		return err
	}
	ctx.WritePlain(")")
	return ctx.Err()
}

func restoreAlterOptions(ctx *format.RestoreCtx, options *List) error {
	if options == nil || len(options.Items) == 0 {
		return fmt.Errorf("foreign options are missing")
	}
	ctx.WriteKeyword("OPTIONS (")
	for i, item := range options.Items {
		option, ok := item.(*DefElem)
		if !ok || option == nil || option.Defname == "" || option.Defnamespace != "" {
			return fmt.Errorf("invalid foreign option %d", i)
		}
		if i > 0 {
			ctx.WritePlain(", ")
		}
		switch DefElemAction(option.Defaction) {
		case DEFELEM_UNSPEC:
		case DEFELEM_ADD:
			ctx.WriteKeyword("ADD ")
		case DEFELEM_SET:
			ctx.WriteKeyword("SET ")
		case DEFELEM_DROP:
			if option.Arg != nil {
				return fmt.Errorf("DROP option has a value")
			}
			ctx.WriteKeyword("DROP ")
		default:
			return fmt.Errorf("invalid option action %d", option.Defaction)
		}
		ctx.WriteName(option.Defname)
		if option.Arg != nil {
			ctx.WritePlain(" ")
			if err := option.Arg.Restore(ctx); err != nil {
				return err
			}
		}
	}
	ctx.WritePlain(")")
	return ctx.Err()
}

func restoreNameList(ctx *format.RestoreCtx, names *List) error {
	if names == nil || len(names.Items) == 0 {
		return fmt.Errorf("qualified name is missing")
	}
	for i, item := range names.Items {
		name, ok := item.(*String)
		if !ok || name == nil || name.Str == "" {
			return fmt.Errorf("invalid name component %T", item)
		}
		if i > 0 {
			ctx.WritePlain(".")
		}
		ctx.WriteName(name.Str)
	}
	return ctx.Err()
}

// restoreSeqOptions writes a sequence option list back in keyword form,
// because the grammar only accepts `INCREMENT`, `AS`, `OWNED BY`, ...
func restoreSeqOptions(ctx *format.RestoreCtx, options *List) error {
	if options == nil {
		return ctx.Err()
	}
	for _, item := range options.Items {
		option, ok := item.(*DefElem)
		if !ok || option == nil || option.Defaction != int(DEFELEM_UNSPEC) {
			return fmt.Errorf("%w: sequence option %T", ErrUnsupportedRestore, item)
		}
		var value string
		switch arg := option.Arg.(type) {
		case nil:
		case *Float:
			value = arg.Fval
		case *Integer:
			value = fmt.Sprint(arg.Ival)
		case *Boolean:
			value = fmt.Sprint(arg.Boolval)
		case *DataType:
			var out strings.Builder
			if err := arg.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil {
				return err
			}
			value = out.String()
		case *List:
			var out strings.Builder
			if err := restoreNameList(format.NewRestoreCtx(format.DefaultRestoreFlags, &out), arg); err != nil {
				return err
			}
			value = out.String()
		default:
			return fmt.Errorf("%w: sequence option argument %T", ErrUnsupportedRestore, option.Arg)
		}
		ctx.WritePlain(" ")
		switch option.Defname {
		case "as":
			ctx.WriteKeyword("AS ")
			ctx.WritePlain(value)
		case "cache":
			ctx.WriteKeyword("CACHE ")
			ctx.WritePlain(value)
		case "cycle":
			if value == "true" {
				ctx.WriteKeyword("CYCLE")
			} else {
				ctx.WriteKeyword("NO CYCLE")
			}
		case "increment":
			ctx.WriteKeyword("INCREMENT ")
			ctx.WritePlain(value)
		case "minvalue":
			if value == "" {
				ctx.WriteKeyword("NO MINVALUE")
			} else {
				ctx.WriteKeyword("MINVALUE ")
				ctx.WritePlain(value)
			}
		case "maxvalue":
			if value == "" {
				ctx.WriteKeyword("NO MAXVALUE")
			} else {
				ctx.WriteKeyword("MAXVALUE ")
				ctx.WritePlain(value)
			}
		case "owned_by":
			ctx.WriteKeyword("OWNED BY ")
			ctx.WritePlain(value)
		case "sequence_name":
			ctx.WriteKeyword("SEQUENCE NAME ")
			ctx.WritePlain(value)
		case "start":
			ctx.WriteKeyword("START ")
			ctx.WritePlain(value)
		case "restart":
			ctx.WriteKeyword("RESTART")
			if value != "" {
				ctx.WritePlain(" ")
				ctx.WritePlain(value)
			}
		case "logged":
			if value == "true" {
				ctx.WriteKeyword("LOGGED")
			} else {
				ctx.WriteKeyword("UNLOGGED")
			}
		default:
			return fmt.Errorf("%w: sequence option %q", ErrUnsupportedRestore, option.Defname)
		}
	}
	return ctx.Err()
}

func restoreVariableSet(ctx *format.RestoreCtx, n *VariableSetStmt) error {
	if n.Kind == VAR_RESET_ALL {
		ctx.WriteKeyword("RESET ALL")
		return ctx.Err()
	}
	if n.Name == "" {
		return fmt.Errorf("SET has no parameter name")
	}
	switch n.Kind {
	case VAR_RESET:
		ctx.WriteKeyword("RESET ")
		ctx.WriteName(n.Name)
		return ctx.Err()
	case VAR_SET_VALUE, VAR_SET_DEFAULT, VAR_SET_CURRENT:
		ctx.WriteKeyword("SET ")
		if n.IsLocal {
			ctx.WriteKeyword("LOCAL ")
		}
		ctx.WriteName(n.Name)
	case VAR_SET_MULTI:
		if n.Args == nil || len(n.Args.Items) == 0 {
			return fmt.Errorf("SET has no transaction values")
		}
		ctx.WriteKeyword("SET ")
		if n.IsLocal {
			ctx.WriteKeyword("LOCAL ")
		}
		switch n.Name {
		case "TRANSACTION SNAPSHOT":
			if len(n.Args.Items) != 1 {
				return fmt.Errorf("SET TRANSACTION SNAPSHOT requires one value")
			}
			ctx.WriteKeyword("TRANSACTION SNAPSHOT ")
			return n.Args.Restore(ctx)
		case "TRANSACTION":
			ctx.WriteKeyword("TRANSACTION ")
		case "SESSION CHARACTERISTICS":
			ctx.WriteKeyword("SESSION CHARACTERISTICS AS TRANSACTION ")
		default:
			return fmt.Errorf("invalid transaction SET name %q", n.Name)
		}
		return restoreTransactionModes(ctx, n.Args)
	default:
		return fmt.Errorf("%w: SET kind %d", ErrUnsupportedRestore, n.Kind)
	}
	switch n.Kind {
	case VAR_SET_CURRENT:
		ctx.WriteKeyword(" FROM CURRENT")
	case VAR_SET_DEFAULT:
		ctx.WriteKeyword(" TO DEFAULT")
	case VAR_SET_VALUE:
		if n.Args == nil || len(n.Args.Items) == 0 {
			return fmt.Errorf("SET has no values")
		}
		ctx.WriteKeyword(" TO ")
		return n.Args.Restore(ctx)
	}
	return ctx.Err()
}

func restoreTransactionModes(ctx *format.RestoreCtx, modes *List) error {
	for i, item := range modes.Items {
		option, ok := item.(*DefElem)
		if !ok || option == nil || option.Defnamespace != "" || option.Defaction != int(DEFELEM_UNSPEC) {
			return fmt.Errorf("invalid transaction mode")
		}
		value, ok := option.Arg.(ValueExpr)
		if !ok {
			return fmt.Errorf("transaction mode has no value")
		}
		if i > 0 {
			ctx.WritePlain(", ")
		}
		switch option.Defname {
		case "transaction_isolation":
			level, ok := value.GetValue().(string)
			if !ok {
				return fmt.Errorf("invalid isolation level")
			}
			switch strings.ToUpper(level) {
			case "READ UNCOMMITTED", "READ COMMITTED", "REPEATABLE READ", "SERIALIZABLE":
				ctx.WriteKeyword("ISOLATION LEVEL " + level)
			default:
				return fmt.Errorf("invalid isolation level")
			}
		case "transaction_read_only", "transaction_deferrable":
			flag, ok := value.GetValue().(int64)
			if !ok || (flag != 0 && flag != 1) {
				return fmt.Errorf("invalid transaction mode flag")
			}
			on, off := "READ ONLY", "READ WRITE"
			if option.Defname == "transaction_deferrable" {
				on, off = "DEFERRABLE", "NOT DEFERRABLE"
			}
			if flag == 1 {
				ctx.WriteKeyword(on)
			} else {
				ctx.WriteKeyword(off)
			}
		default:
			return fmt.Errorf("invalid transaction mode %q", option.Defname)
		}
	}
	return ctx.Err()
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
	ctx.WriteKeyword(name)
	if n.Typmod >= 0 {
		ctx.WritePlainf("(%d)", n.Typmod)
	}
	return ctx.Err()
}

// restoreIndices writes one array subscript, `[x]`, `[x:y]`, `[:y]` or `[x:]`.
func restoreIndices(ctx *format.RestoreCtx, n *A_Indices) error {
	ctx.WritePlain("[")
	if n.IsSlice {
		if n.Lidx != nil {
			if err := n.Lidx.Restore(ctx); err != nil {
				return err
			}
		}
		ctx.WritePlain(":")
		if n.Uidx != nil {
			if err := n.Uidx.Restore(ctx); err != nil {
				return err
			}
		}
	} else if n.Uidx != nil {
		if err := n.Uidx.Restore(ctx); err != nil {
			return err
		}
	}
	ctx.WritePlain("]")
	return ctx.Err()
}

func restoreXML(ctx *format.RestoreCtx, n *XmlExpr) error {
	if n.Op != IS_XMLPARSE || n.Args == nil || len(n.Args.Items) != 2 {
		return fmt.Errorf("%w: XML operation %d", ErrUnsupportedRestore, n.Op)
	}
	ctx.WriteKeyword("XMLPARSE(")
	if n.Xmloption == XMLOPTION_DOCUMENT {
		ctx.WriteKeyword("DOCUMENT ")
	} else {
		ctx.WriteKeyword("CONTENT ")
	}
	if err := n.Args.Items[0].Restore(ctx); err != nil {
		return err
	}
	if value, ok := n.Args.Items[1].(ValueExpr); ok && (value.GetValue() == true || value.GetValue() == int64(1)) {
		ctx.WriteKeyword(" PRESERVE WHITESPACE")
	} else {
		ctx.WriteKeyword(" STRIP WHITESPACE")
	}
	ctx.WritePlain(")")
	return ctx.Err()
}
