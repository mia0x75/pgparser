package parser

import (
	"github.com/mia0x75/parserkit/lex"

	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/auth"
)

// createUserStmt returns the create user statement of a role.
func createUserStmt(role bool, name lex.Ident, options *ast.List) ast.Node {
	spec := &ast.UserSpec{
		User:   &auth.UserIdentity{Username: name.Name},
		IsRole: role,
	}
	if opt := authOption(options); opt != nil {
		spec.AuthOpt = opt
	}
	return &ast.CreateUserStmt{
		IsCreateRole: role,
		Specs:        []*ast.UserSpec{spec},
		RoleOptions:  options,
		Role:         &ast.RoleSpec{Roletype: int(ast.ROLESPEC_CSTRING), Rolename: name},
	}
}

// alterUserStmt returns the alter user statement of a role.
func alterUserStmt(role bool, spec ast.Node, options *ast.List) ast.Node {
	stmt := &ast.AlterUserStmt{
		IsAlterRole: role,
		RoleOptions: options,
	}
	if s, ok := spec.(*ast.RoleSpec); ok {
		stmt.Role = s
		stmt.Specs = []*ast.UserSpec{{
			User:   &auth.UserIdentity{Username: s.Rolename.Name},
			IsRole: role,
		}}
	}
	if opt := authOption(options); opt != nil {
		stmt.CurrentAuth = opt
	}
	return stmt
}

// dropUserStmt returns the drop user statement of a role list.
func dropUserStmt(role bool, ifExists bool, roles *ast.List) ast.Node {
	stmt := &ast.DropUserStmt{
		IsDropRole: role,
		IfExists:   ifExists,
	}
	if roles != nil {
		for _, item := range roles.Items {
			switch v := item.(type) {
			case *ast.RoleSpec:
				stmt.UserList = append(stmt.UserList, &auth.UserIdentity{Username: v.Rolename.Name})
			case *ast.String:
				stmt.UserList = append(stmt.UserList, &auth.UserIdentity{Username: v.Str})
			}
		}
	}
	return stmt
}

// authOption returns the authentication option of a role option list.
func authOption(options *ast.List) *ast.AuthOption {
	if options == nil {
		return nil
	}
	for _, item := range options.Items {
		def, ok := item.(*ast.DefElem)
		if !ok {
			continue
		}
		switch def.Defname {
		case "password":
			return &ast.AuthOption{Password: true, ByAuthString: true, AuthString: defElemString(def)}
		case "encrypted":
			return &ast.AuthOption{ByAuthString: true, HashString: defElemString(def)}
		}
	}
	return nil
}

// defElemString returns the string value of a definition element.
func defElemString(def *ast.DefElem) string {
	switch v := def.Arg.(type) {
	case *ast.String:
		return v.Str
	case ast.ValueExpr:
		if s, ok := v.GetValue().(string); ok {
			return s
		}
	}
	return ""
}
