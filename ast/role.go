package ast

import (
	"fmt"

	"github.com/mia0x75/pgparser/format"
)

func restoreRoleStatement(ctx *format.RestoreCtx, role, alter bool, target *RoleSpec, options *List) error {
	if target == nil {
		return fmt.Errorf("role statement has no target")
	}
	if alter {
		ctx.WriteKeyword("ALTER ")
	} else {
		ctx.WriteKeyword("CREATE ")
	}
	if role {
		ctx.WriteKeyword("ROLE ")
	} else {
		ctx.WriteKeyword("USER ")
	}
	if err := target.Restore(ctx); err != nil {
		return err
	}
	if options == nil {
		return ctx.Err()
	}
	for _, item := range options.Items {
		option, ok := item.(*DefElem)
		if !ok {
			return fmt.Errorf("invalid role option %T", item)
		}
		ctx.WritePlain(" ")
		if err := restoreRoleOption(ctx, option); err != nil {
			return err
		}
	}
	return ctx.Err()
}

func restoreRoleOption(ctx *format.RestoreCtx, option *DefElem) error {
	name := option.Defname
	switch name {
	case "superuser", "createrole", "createdb", "inherit", "bypassrls", "canlogin", "isreplication":
		if name == "canlogin" {
			name = "login"
		}
		if name == "isreplication" {
			name = "replication"
		}
		value, ok := option.Arg.(*Boolean)
		if !ok {
			return fmt.Errorf("invalid boolean role option %s", name)
		}
		if !value.Boolval {
			ctx.WriteKeyword("NO")
		}
		ctx.WriteKeyword(name)
	case "password", "validUntil", "connectionlimit", "sysid":
		switch name {
		case "validUntil":
			ctx.WriteKeyword("VALID UNTIL ")
		case "connectionlimit":
			ctx.WriteKeyword("CONNECTION LIMIT ")
		default:
			ctx.WriteKeyword(name + " ")
		}
		if option.Arg == nil {
			ctx.WriteKeyword("NULL")
			return ctx.Err()
		}
		return option.Arg.Restore(ctx)
	case "rolemembers", "adminmembers", "addroleto":
		switch name {
		case "rolemembers":
			ctx.WriteKeyword("ROLE ")
		case "adminmembers":
			ctx.WriteKeyword("ADMIN ")
		case "addroleto":
			ctx.WriteKeyword("IN ROLE ")
		}
		list, ok := option.Arg.(*List)
		if !ok {
			return fmt.Errorf("invalid role membership")
		}
		for i, item := range list.Items {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			role, ok := item.(*RoleSpec)
			if !ok {
				return fmt.Errorf("invalid role member")
			}
			if err := role.Restore(ctx); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported role option %s", name)
	}
	return ctx.Err()
}
