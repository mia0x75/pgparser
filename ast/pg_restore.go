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
			ctx.WriteKeyWord("TRUE")
		} else {
			ctx.WriteKeyWord("FALSE")
		}
	case *A_Star:
		ctx.WritePlain("*")
	case *A_Const:
		if n.Isnull {
			ctx.WriteKeyWord("NULL")
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
			ctx.WriteKeyWord("OUT ")
		case FUNC_PARAM_INOUT:
			ctx.WriteKeyWord("INOUT ")
		case FUNC_PARAM_VARIADIC:
			ctx.WriteKeyWord("VARIADIC ")
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
			ctx.WriteKeyWord(" DEFAULT ")
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
			ctx.WriteKeyWord("RANGE")
		case "l", "list":
			ctx.WriteKeyWord("LIST")
		case "h", "hash":
			ctx.WriteKeyWord("HASH")
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
		if n.Name != "" {
			ctx.WriteName(n.Name)
		} else if n.Expr != nil {
			ctx.WritePlain("(")
			if err := n.Expr.Restore(ctx); err != nil {
				return err
			}
			ctx.WritePlain(")")
		} else {
			return fmt.Errorf("partition key has no expression")
		}
		if n.Collation != nil {
			ctx.WriteKeyWord(" COLLATE ")
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
		ctx.WriteKeyWord("ALTER SEQUENCE ")
		if n.MissingOk {
			ctx.WriteKeyWord("IF EXISTS ")
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
			ctx.WriteKeyWord("NEW ")
		} else {
			ctx.WriteKeyWord("OLD ")
		}
		if n.IsTable {
			ctx.WriteKeyWord("TABLE AS ")
		} else {
			ctx.WriteKeyWord("ROW AS ")
		}
		ctx.WriteName(n.Name)
	case *LockingClause:
		switch LockClauseStrength(n.Strength) {
		case LCS_FORKEYSHARE:
			ctx.WriteKeyWord("FOR KEY SHARE")
		case LCS_FORSHARE:
			ctx.WriteKeyWord("FOR SHARE")
		case LCS_FORNOKEYUPDATE:
			ctx.WriteKeyWord("FOR NO KEY UPDATE")
		case LCS_FORUPDATE:
			ctx.WriteKeyWord("FOR UPDATE")
		default:
			return fmt.Errorf("invalid lock strength %d", n.Strength)
		}
		if n.LockedRels != nil {
			ctx.WriteKeyWord(" OF ")
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
			ctx.WriteKeyWord(" NOWAIT")
		case LockWaitSkip:
			ctx.WriteKeyWord(" SKIP LOCKED")
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
			return fmt.Errorf("%w: JSON_SCALAR output", ErrUnsupportedRestore)
		}
		ctx.WriteKeyWord("JSON_SCALAR(")
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
			return fmt.Errorf("%w: JSON DEFAULT", ErrUnsupportedRestore)
		default:
			return fmt.Errorf("unsupported JSON behavior %d", n.Btype)
		}
	case *XmlExpr:
		return restoreXML(ctx, n)
	default:
		return fmt.Errorf("%w for %T", ErrUnsupportedRestore, node)
	}
	return nil
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
	return nil
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
	return nil
}

func restoreAlterOptions(ctx *format.RestoreCtx, options *List) error {
	if options == nil || len(options.Items) == 0 {
		return fmt.Errorf("foreign options are missing")
	}
	ctx.WriteKeyWord("OPTIONS (")
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
			ctx.WriteKeyWord("ADD ")
		case DEFELEM_SET:
			ctx.WriteKeyWord("SET ")
		case DEFELEM_DROP:
			if option.Arg != nil {
				return fmt.Errorf("DROP option has a value")
			}
			ctx.WriteKeyWord("DROP ")
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
	return nil
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
	return nil
}

// restoreSeqOptions writes a sequence option list back in keyword form,
// because the grammar only accepts `INCREMENT`, `AS`, `OWNED BY`, ...
func restoreSeqOptions(ctx *format.RestoreCtx, options *List) error {
	if options == nil {
		return nil
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
			ctx.WriteKeyWord("AS ")
			ctx.WritePlain(value)
		case "cache":
			ctx.WriteKeyWord("CACHE ")
			ctx.WritePlain(value)
		case "cycle":
			if value == "true" {
				ctx.WriteKeyWord("CYCLE")
			} else {
				ctx.WriteKeyWord("NO CYCLE")
			}
		case "increment":
			ctx.WriteKeyWord("INCREMENT ")
			ctx.WritePlain(value)
		case "minvalue":
			if value == "" {
				ctx.WriteKeyWord("NO MINVALUE")
			} else {
				ctx.WriteKeyWord("MINVALUE ")
				ctx.WritePlain(value)
			}
		case "maxvalue":
			if value == "" {
				ctx.WriteKeyWord("NO MAXVALUE")
			} else {
				ctx.WriteKeyWord("MAXVALUE ")
				ctx.WritePlain(value)
			}
		case "owned_by":
			ctx.WriteKeyWord("OWNED BY ")
			ctx.WritePlain(value)
		case "sequence_name":
			ctx.WriteKeyWord("SEQUENCE NAME ")
			ctx.WritePlain(value)
		case "start":
			ctx.WriteKeyWord("START ")
			ctx.WritePlain(value)
		case "restart":
			ctx.WriteKeyWord("RESTART")
			if value != "" {
				ctx.WritePlain(" ")
				ctx.WritePlain(value)
			}
		case "logged":
			if value == "true" {
				ctx.WriteKeyWord("LOGGED")
			} else {
				ctx.WriteKeyWord("UNLOGGED")
			}
		default:
			return fmt.Errorf("%w: sequence option %q", ErrUnsupportedRestore, option.Defname)
		}
	}
	return nil
}

func restoreVariableSet(ctx *format.RestoreCtx, n *VariableSetStmt) error {
	if n.Kind == VAR_RESET_ALL {
		ctx.WriteKeyWord("RESET ALL")
		return nil
	}
	if n.Name == "" {
		return fmt.Errorf("SET has no parameter name")
	}
	switch n.Kind {
	case VAR_RESET:
		ctx.WriteKeyWord("RESET ")
		ctx.WriteName(n.Name)
		return nil
	case VAR_SET_VALUE, VAR_SET_DEFAULT, VAR_SET_CURRENT:
		ctx.WriteKeyWord("SET ")
		if n.IsLocal {
			ctx.WriteKeyWord("LOCAL ")
		}
		ctx.WriteName(n.Name)
	case VAR_SET_MULTI:
		if n.Args == nil || len(n.Args.Items) == 0 {
			return fmt.Errorf("SET has no transaction values")
		}
		ctx.WriteKeyWord("SET ")
		if n.IsLocal {
			ctx.WriteKeyWord("LOCAL ")
		}
		switch n.Name {
		case "TRANSACTION SNAPSHOT":
			if len(n.Args.Items) != 1 {
				return fmt.Errorf("SET TRANSACTION SNAPSHOT requires one value")
			}
			ctx.WriteKeyWord("TRANSACTION SNAPSHOT ")
			return n.Args.Restore(ctx)
		case "TRANSACTION":
			ctx.WriteKeyWord("TRANSACTION ")
		case "SESSION CHARACTERISTICS":
			ctx.WriteKeyWord("SESSION CHARACTERISTICS AS TRANSACTION ")
		default:
			return fmt.Errorf("invalid transaction SET name %q", n.Name)
		}
		return restoreTransactionModes(ctx, n.Args)
	default:
		return fmt.Errorf("%w: SET kind %d", ErrUnsupportedRestore, n.Kind)
	}
	switch n.Kind {
	case VAR_SET_CURRENT:
		ctx.WriteKeyWord(" FROM CURRENT")
	case VAR_SET_DEFAULT:
		ctx.WriteKeyWord(" TO DEFAULT")
	case VAR_SET_VALUE:
		if n.Args == nil || len(n.Args.Items) == 0 {
			return fmt.Errorf("SET has no values")
		}
		ctx.WriteKeyWord(" TO ")
		return n.Args.Restore(ctx)
	}
	return nil
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
				ctx.WriteKeyWord("ISOLATION LEVEL " + level)
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
				ctx.WriteKeyWord(on)
			} else {
				ctx.WriteKeyWord(off)
			}
		default:
			return fmt.Errorf("invalid transaction mode %q", option.Defname)
		}
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
		return fmt.Errorf("%w: XML operation %d", ErrUnsupportedRestore, n.Op)
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
