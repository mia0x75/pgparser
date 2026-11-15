package ast

import (
	"fmt"

	"github.com/mia0x75/pgparser/format"
)

func restoreTrigger(ctx *format.RestoreCtx, n *CreateTrigStmt) error {
	if n.Trigname == "" || n.Relation == nil || n.Funcname == nil || len(n.Funcname.Items) == 0 {
		return fmt.Errorf("trigger requires a name, relation and function")
	}
	const events = TRIGGER_TYPE_INSERT | TRIGGER_TYPE_UPDATE | TRIGGER_TYPE_DELETE | TRIGGER_TYPE_TRUNCATE
	if n.Events == 0 || n.Events & ^int16(events) != 0 {
		return fmt.Errorf("invalid trigger events %d", n.Events)
	}
	if n.Columns != nil && (len(n.Columns.Items) == 0 || n.Events&TRIGGER_TYPE_UPDATE == 0) {
		return fmt.Errorf("trigger columns require UPDATE")
	}
	if n.IsConstraint {
		if n.Timing != TRIGGER_TYPE_AFTER || !n.Row || n.TransitionRels != nil {
			return fmt.Errorf("constraint triggers require AFTER ROW without transition relations")
		}
	} else if n.Constrrel != nil || n.Deferrable || n.Initdeferred {
		return fmt.Errorf("non-constraint trigger has constraint options")
	}
	if n.Initdeferred && !n.Deferrable {
		return fmt.Errorf("INITIALLY DEFERRED requires DEFERRABLE")
	}
	var timing string
	switch n.Timing {
	case TRIGGER_TYPE_BEFORE:
		timing = "BEFORE"
	case TRIGGER_TYPE_AFTER:
		timing = "AFTER"
	case TRIGGER_TYPE_INSTEAD:
		timing = "INSTEAD OF"
	default:
		return fmt.Errorf("invalid trigger timing %d", n.Timing)
	}
	ctx.WriteKeyWord("CREATE ")
	if n.Replace {
		ctx.WriteKeyWord("OR REPLACE ")
	}
	if n.IsConstraint {
		ctx.WriteKeyWord("CONSTRAINT ")
	}
	ctx.WriteKeyWord("TRIGGER ")
	ctx.WriteName(n.Trigname)
	ctx.WriteKeyWord(" " + timing + " ")
	written := false
	for _, event := range []struct {
		mask int16
		name string
	}{
		{TRIGGER_TYPE_INSERT, "INSERT"},
		{TRIGGER_TYPE_UPDATE, "UPDATE"},
		{TRIGGER_TYPE_DELETE, "DELETE"},
		{TRIGGER_TYPE_TRUNCATE, "TRUNCATE"},
	} {
		if n.Events&event.mask == 0 {
			continue
		}
		if written {
			ctx.WriteKeyWord(" OR ")
		}
		written = true
		ctx.WriteKeyWord(event.name)
		if event.mask == TRIGGER_TYPE_UPDATE && n.Columns != nil {
			ctx.WriteKeyWord(" OF ")
			for i, item := range n.Columns.Items {
				column, ok := item.(*String)
				if !ok || column == nil || column.Str == "" {
					return fmt.Errorf("invalid trigger column %d", i)
				}
				if i > 0 {
					ctx.WritePlain(", ")
				}
				ctx.WriteName(column.Str)
			}
		}
	}
	ctx.WriteKeyWord(" ON ")
	if err := n.Relation.Restore(ctx); err != nil {
		return err
	}
	if n.Constrrel != nil {
		ctx.WriteKeyWord(" FROM ")
		if err := n.Constrrel.Restore(ctx); err != nil {
			return err
		}
	}
	if n.Deferrable {
		ctx.WriteKeyWord(" DEFERRABLE")
	}
	if n.Initdeferred {
		ctx.WriteKeyWord(" INITIALLY DEFERRED")
	}
	if n.TransitionRels != nil {
		ctx.WriteKeyWord(" REFERENCING")
		for _, item := range n.TransitionRels.Items {
			transition, ok := item.(*TriggerTransition)
			if !ok || transition == nil {
				return fmt.Errorf("invalid trigger transition")
			}
			ctx.WritePlain(" ")
			if err := transition.Restore(ctx); err != nil {
				return err
			}
		}
	}
	if n.Row {
		ctx.WriteKeyWord(" FOR EACH ROW")
	} else {
		ctx.WriteKeyWord(" FOR EACH STATEMENT")
	}
	if n.WhenClause != nil {
		ctx.WriteKeyWord(" WHEN (")
		if err := n.WhenClause.Restore(ctx); err != nil {
			return err
		}
		ctx.WritePlain(")")
	}
	ctx.WriteKeyWord(" EXECUTE FUNCTION ")
	if err := restoreNameList(ctx, n.Funcname); err != nil {
		return err
	}
	ctx.WritePlain("(")
	if n.Args != nil {
		if err := n.Args.Restore(ctx); err != nil {
			return err
		}
	}
	ctx.WritePlain(")")
	return ctx.Err()
}
