package ast

import (
	"fmt"
	"strings"

	"github.com/mia0x75/pgparser/format"
	"github.com/mia0x75/pgparser/types"
)

type CreateFunctionStmt struct {
	ddlNode
	Name        *TableName
	Params      *List
	Returns     *types.FieldType
	ReturnTable *List
	Options     []*DefElem
	Body        Node
	OrReplace   bool
	IsProcedure bool
}

func (n *CreateFunctionStmt) Restore(ctx *format.RestoreCtx) error {
	if n.Name == nil {
		return fmt.Errorf("routine has no name")
	}
	ctx.WriteKeyWord("CREATE ")
	if n.OrReplace {
		ctx.WriteKeyWord("OR REPLACE ")
	}
	if n.IsProcedure {
		ctx.WriteKeyWord("PROCEDURE ")
	} else {
		ctx.WriteKeyWord("FUNCTION ")
	}
	if err := n.Name.Restore(ctx); err != nil {
		return err
	}
	ctx.WritePlain("(")
	if n.Params != nil {
		if err := n.Params.Restore(ctx); err != nil {
			return fmt.Errorf("routine parameters: %w", err)
		}
	}
	ctx.WritePlain(")")
	if n.ReturnTable != nil {
		ctx.WriteKeyWord(" RETURNS TABLE (")
		if err := n.ReturnTable.Restore(ctx); err != nil {
			return fmt.Errorf("routine return columns: %w", err)
		}
		ctx.WritePlain(")")
	} else if n.Returns != nil {
		ctx.WriteKeyWord(" RETURNS ")
		if err := n.Returns.Restore(ctx); err != nil {
			return fmt.Errorf("routine return type: %w", err)
		}
	}
	if err := restoreRoutineOptions(ctx, n.Options); err != nil {
		return err
	}
	if n.Body != nil {
		ctx.WritePlain(" ")
		if err := n.Body.Restore(ctx); err != nil {
			return fmt.Errorf("routine body: %w", err)
		}
	}
	return nil
}

func (n *CreateFunctionStmt) Accept(v Visitor) (Node, bool) {
	updated, skip := v.Enter(n)
	if skip {
		return v.Leave(updated)
	}
	n = updated.(*CreateFunctionStmt)
	if n.Name != nil {
		child, ok := n.Name.Accept(v)
		if !ok {
			return n, false
		}
		n.Name = child.(*TableName)
	}
	if n.Params != nil {
		child, ok := n.Params.Accept(v)
		if !ok {
			return n, false
		}
		n.Params = child.(*List)
	}
	if n.ReturnTable != nil {
		child, ok := n.ReturnTable.Accept(v)
		if !ok {
			return n, false
		}
		n.ReturnTable = child.(*List)
	}
	for i, option := range n.Options {
		if option == nil {
			continue
		}
		child, ok := option.Accept(v)
		if !ok {
			return n, false
		}
		n.Options[i] = child.(*DefElem)
	}
	if n.Body != nil {
		child, ok := n.Body.Accept(v)
		if !ok {
			return n, false
		}
		n.Body = child
	}
	return v.Leave(n)
}

type AlterFunctionStmt struct {
	ddlNode
	Func    Node
	Actions []*DefElem
	Objtype string
}

func (n *AlterFunctionStmt) Restore(ctx *format.RestoreCtx) error {
	if n.Func == nil {
		return fmt.Errorf("ALTER routine has no signature")
	}
	switch n.Objtype {
	case "FUNCTION", "PROCEDURE", "ROUTINE":
	default:
		return fmt.Errorf("invalid routine kind %q", n.Objtype)
	}
	ctx.WriteKeyWord("ALTER " + n.Objtype + " ")
	if err := n.Func.Restore(ctx); err != nil {
		return fmt.Errorf("routine signature: %w", err)
	}
	return restoreRoutineOptions(ctx, n.Actions)
}

func (n *AlterFunctionStmt) Accept(v Visitor) (Node, bool) {
	updated, skip := v.Enter(n)
	if skip {
		return v.Leave(updated)
	}
	n = updated.(*AlterFunctionStmt)
	if n.Func != nil {
		child, ok := n.Func.Accept(v)
		if !ok {
			return n, false
		}
		n.Func = child
	}
	for i, option := range n.Actions {
		if option == nil {
			continue
		}
		child, ok := option.Accept(v)
		if !ok {
			return n, false
		}
		n.Actions[i] = child.(*DefElem)
	}
	return v.Leave(n)
}

func restoreRoutineOptions(ctx *format.RestoreCtx, options []*DefElem) error {
	for _, option := range options {
		if option == nil {
			return fmt.Errorf("routine has a nil option")
		}
		ctx.WritePlain(" ")
		if err := restoreRoutineOption(ctx, option); err != nil {
			return fmt.Errorf("routine option %q: %w", option.Defname, err)
		}
	}
	return nil
}

func restoreRoutineOption(ctx *format.RestoreCtx, option *DefElem) error {
	name := strings.ToLower(option.Defname)
	if option.Defnamespace != "" || option.Defaction != int(DEFELEM_UNSPEC) {
		return fmt.Errorf("invalid routine option metadata")
	}
	switch name {
	case "as", "transform":
		list, ok := option.Arg.(*List)
		if !ok || list == nil || len(list.Items) == 0 {
			return fmt.Errorf("option has no arguments")
		}
		ctx.WriteKeyWord(option.Defname + " ")
		for i, item := range list.Items {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if item == nil {
				return fmt.Errorf("option has a nil argument")
			}
			if name == "transform" {
				ctx.WriteKeyWord("FOR TYPE ")
			}
			if err := item.Restore(ctx); err != nil {
				return err
			}
		}
	case "language", "volatility", "parallel":
		arg, ok := option.Arg.(*String)
		if !ok || arg == nil || arg.Str == "" {
			return fmt.Errorf("option has no name")
		}
		switch name {
		case "language":
			ctx.WriteKeyWord("LANGUAGE ")
			ctx.WriteName(arg.Str)
		case "volatility":
			switch strings.ToLower(arg.Str) {
			case "immutable", "stable", "volatile":
				ctx.WriteKeyWord(arg.Str)
			default:
				return fmt.Errorf("invalid volatility %q", arg.Str)
			}
		case "parallel":
			switch strings.ToLower(arg.Str) {
			case "safe", "restricted", "unsafe":
				ctx.WriteKeyWord("PARALLEL " + arg.Str)
			default:
				return fmt.Errorf("invalid parallel mode %q", arg.Str)
			}
		}
	case "strict", "security", "leakproof":
		arg, ok := option.Arg.(*Integer)
		if !ok || arg == nil || (arg.Ival != 0 && arg.Ival != 1) {
			return fmt.Errorf("option is not boolean")
		}
		on, off := "STRICT", "CALLED ON NULL INPUT"
		if name == "security" {
			on, off = "SECURITY DEFINER", "SECURITY INVOKER"
		}
		if name == "leakproof" {
			on, off = "LEAKPROOF", "NOT LEAKPROOF"
		}
		if arg.Ival == 1 {
			ctx.WriteKeyWord(on)
		} else {
			ctx.WriteKeyWord(off)
		}
	case "cost", "rows":
		if option.Arg == nil {
			return fmt.Errorf("option has no value")
		}
		ctx.WriteKeyWord(option.Defname + " ")
		return option.Arg.Restore(ctx)
	case "window":
		arg, ok := option.Arg.(*Integer)
		if !ok || arg == nil || arg.Ival != 1 {
			return fmt.Errorf("WINDOW option is not enabled")
		}
		ctx.WriteKeyWord("WINDOW")
	case "support":
		list, ok := option.Arg.(*List)
		if !ok {
			return fmt.Errorf("SUPPORT has no function name")
		}
		ctx.WriteKeyWord("SUPPORT ")
		return restoreNameList(ctx, list)
	case "set":
		stmt, ok := option.Arg.(*VariableSetStmt)
		if !ok || stmt == nil {
			return fmt.Errorf("SET has no statement")
		}
		return stmt.Restore(ctx)
	default:
		return fmt.Errorf("%w: routine option %q", ErrUnsupportedRestore, option.Defname)
	}
	return nil
}

type ReturnStmt struct {
	stmtNode
	Expr ExprNode
}

func (n *ReturnStmt) Restore(ctx *format.RestoreCtx) error {
	if n.Expr == nil {
		return fmt.Errorf("RETURN has no expression")
	}
	ctx.WriteKeyWord("RETURN ")
	return n.Expr.Restore(ctx)
}

func (n *ReturnStmt) Accept(v Visitor) (Node, bool) {
	updated, skip := v.Enter(n)
	if skip {
		return v.Leave(updated)
	}
	n = updated.(*ReturnStmt)
	if n.Expr != nil {
		child, ok := n.Expr.Accept(v)
		if !ok {
			return n, false
		}
		n.Expr = child.(ExprNode)
	}
	return v.Leave(n)
}
