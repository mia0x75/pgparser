package parser

import (
	"strings"

	"github.com/mia0x75/parserkit/lex"

	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/auth"
	"github.com/mia0x75/pgparser/postgres"
)

// userSpecs returns the user specifications of a grantee list.
func userSpecs(l *ast.List) []*ast.UserSpec {
	if l == nil {
		return nil
	}
	var result []*ast.UserSpec
	for _, item := range l.Items {
		switch v := item.(type) {
		case *ast.RoleSpec:
			result = append(result, &ast.UserSpec{
				User:   &auth.UserIdentity{Username: v.Rolename.Name},
				IsRole: true,
				Role:   v,
			})
		case *ast.String:
			result = append(result, &ast.UserSpec{
				User:   &auth.UserIdentity{Username: v.Str},
				IsRole: true,
			})
		}
	}
	return result
}

// Role membership uses privilege_list grammar, but the names are identifiers.
func grantRoleNames(l *ast.List, lexer yyLexer) *ast.List {
	if l == nil {
		return nil
	}
	for i, item := range l.Items {
		priv, ok := item.(*ast.AccessPriv)
		if !ok || priv.Cols != nil {
			lexer.Error("invalid role name list")
			return nil
		}
		l.Items[i] = &ast.RoleSpec{Roletype: int(ast.ROLESPEC_CSTRING), Rolename: syntaxIdent(lexer, priv.PrivName, priv.Span())}
	}
	return l
}

// PostgreSQL's grammar uses nil or an unnamed privilege for ALL.
func privElems(l *ast.List) []*ast.PrivElem {
	if l == nil {
		return []*ast.PrivElem{{Priv: postgres.AllPriv, Name: "all"}}
	}
	var result []*ast.PrivElem
	for _, item := range l.Items {
		priv, ok := item.(*ast.AccessPriv)
		if !ok {
			continue
		}
		name := priv.PrivName
		if name == "" {
			name = "all"
		}
		elem := &ast.PrivElem{Priv: privilegeType(name), Name: name}
		if priv.Cols != nil {
			elem.Cols = columnNamesFromList(priv.Cols)
		}
		result = append(result, elem)
	}
	return result
}

// privilegeType returns the privilege type of a privilege name.
func privilegeType(name string) postgres.PrivilegeType {
	switch strings.ToUpper(name) {
	case "SELECT":
		return postgres.SelectPriv
	case "INSERT":
		return postgres.InsertPriv
	case "UPDATE":
		return postgres.UpdatePriv
	case "DELETE":
		return postgres.DeletePriv
	case "ALL":
		return postgres.AllPriv
	case "CREATE":
		return postgres.CreatePriv
	case "DROP":
		return postgres.DropPriv
	case "INDEX":
		return postgres.IndexPriv
	case "ALTER":
		return postgres.AlterPriv
	}
	return postgres.PrivilegeType(0)
}

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
