package parser

import (
	"slices"
	"strconv"
	"strings"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"

	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/auth"
	"github.com/mia0x75/pgparser/format"
	"github.com/mia0x75/pgparser/model"
	"github.com/mia0x75/pgparser/postgres"
	"github.com/mia0x75/pgparser/types"
)

// defElemNames returns the names of the definition elements of a list.
func defElemNames(l any) []string {
	switch v := l.(type) {
	case nil:
		return nil
	case *ast.List:
		if v == nil {
			return nil
		}
		var result []string
		for _, item := range v.Items {
			if def, ok := item.(*ast.DefElem); ok {
				result = append(result, def.Defname)
			}
		}
		return result
	case string:
		return []string{v}
	}
	return nil
}

// hasDefElem reports whether the list holds a definition element with the name.
func hasDefElem(l *ast.List, name string) bool {
	return slices.Contains(defElemNames(l), name)
}

// reindexObjectName returns the object name of a reindex target type.
func reindexObjectName(code int64) string {
	switch ast.ReindexObjectType(code) {
	case ast.REINDEX_OBJECT_INDEX:
		return "INDEX"
	case ast.REINDEX_OBJECT_TABLE:
		return "TABLE"
	case ast.REINDEX_OBJECT_SCHEMA:
		return "SCHEMA"
	case ast.REINDEX_OBJECT_SYSTEM:
		return "SYSTEM"
	case ast.REINDEX_OBJECT_DATABASE:
		return "DATABASE"
	}
	return ""
}

// mergeWhenClauses returns the when clauses of a merge statement.
func mergeWhenClauses(l *ast.List) []*ast.MergeWhenClause {
	if l == nil {
		return nil
	}
	var result []*ast.MergeWhenClause
	for _, item := range l.Items {
		if when, ok := item.(*ast.MergeWhenClause); ok {
			result = append(result, when)
		}
	}
	return result
}

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

// alterTableSpecs returns the alter table specs of a command list.
func alterTableSpecs(l *ast.List) []*ast.AlterTableSpec {
	if l == nil {
		return nil
	}
	var result []*ast.AlterTableSpec
	for _, item := range l.Items {
		if spec, ok := item.(*ast.AlterTableSpec); ok {
			result = append(result, spec)
		}
	}
	return result
}

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

// objectTypeName returns the name of an object type.
func objectTypeName(code int64) string {
	switch ast.ObjectType(code) {
	case ast.OBJECT_TABLE:
		return "TABLE"
	case ast.OBJECT_COLUMN:
		return "COLUMN"
	case ast.OBJECT_SEQUENCE:
		return "SEQUENCE"
	case ast.OBJECT_VIEW:
		return "VIEW"
	case ast.OBJECT_MATVIEW:
		return "MATERIALIZED VIEW"
	case ast.OBJECT_INDEX:
		return "INDEX"
	case ast.OBJECT_TYPE:
		return "TYPE"
	case ast.OBJECT_DOMAIN:
		return "DOMAIN"
	case ast.OBJECT_AGGREGATE:
		return "AGGREGATE"
	case ast.OBJECT_FUNCTION:
		return "FUNCTION"
	case ast.OBJECT_PROCEDURE:
		return "PROCEDURE"
	case ast.OBJECT_ROUTINE:
		return "ROUTINE"
	case ast.OBJECT_OPERATOR:
		return "OPERATOR"
	case ast.OBJECT_SCHEMA:
		return "SCHEMA"
	case ast.OBJECT_DATABASE:
		return "DATABASE"
	case ast.OBJECT_ROLE:
		return "ROLE"
	case ast.OBJECT_TABLESPACE:
		return "TABLESPACE"
	case ast.OBJECT_SUBSCRIPTION:
		return "SUBSCRIPTION"
	case ast.OBJECT_PUBLICATION:
		return "PUBLICATION"
	case ast.OBJECT_FOREIGN_SERVER:
		return "SERVER"
	case ast.OBJECT_TABCONSTRAINT:
		return "CONSTRAINT"
	case ast.OBJECT_DOMCONSTRAINT:
		return "CONSTRAINT"
	case ast.OBJECT_TRANSFORM:
		return "TRANSFORM"
	case ast.OBJECT_OPCLASS:
		return "OPERATOR CLASS"
	case ast.OBJECT_OPFAMILY:
		return "OPERATOR FAMILY"
	case ast.OBJECT_LARGEOBJECT:
		return "LARGE OBJECT"
	case ast.OBJECT_FDW:
		return "FOREIGN DATA WRAPPER"
	case ast.OBJECT_CAST:
		return "CAST"
	case ast.OBJECT_EVENT_TRIGGER:
		return "EVENT TRIGGER"
	case ast.OBJECT_FOREIGN_TABLE:
		return "FOREIGN TABLE"
	case ast.OBJECT_COLLATION:
		return "COLLATION"
	case ast.OBJECT_CONVERSION:
		return "CONVERSION"
	case ast.OBJECT_STATISTIC_EXT:
		return "STATISTICS"
	case ast.OBJECT_TSPARSER:
		return "TEXT SEARCH PARSER"
	case ast.OBJECT_TSDICTIONARY:
		return "TEXT SEARCH DICTIONARY"
	case ast.OBJECT_TSTEMPLATE:
		return "TEXT SEARCH TEMPLATE"
	case ast.OBJECT_TSCONFIGURATION:
		return "TEXT SEARCH CONFIGURATION"
	case ast.OBJECT_ACCESS_METHOD:
		return "ACCESS METHOD"
	case ast.OBJECT_POLICY:
		return "POLICY"
	case ast.OBJECT_RULE:
		return "RULE"
	case ast.OBJECT_TRIGGER:
		return "TRIGGER"
	}
	return "OBJECT"
}

// commentPtr returns the pointer of a comment text.
func commentPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// makeIndexColNameList returns the index column names of a column list.
func makeIndexColNameList(l *ast.List) []*ast.IndexColName {
	if l == nil {
		return nil
	}
	result := make([]*ast.IndexColName, 0, len(l.Items))
	for _, item := range l.Items {
		switch v := item.(type) {
		case *ast.String:
			result = append(result, &ast.IndexColName{
				Column: &ast.ColumnName{Name: model.NewCIStr(v.Str)},
				Length: -1,
			})
		case *ast.IndexElem:
			result = append(result, &ast.IndexColName{
				Column: &ast.ColumnName{Name: model.NewCIStr(v.Name)},
				Length: -1,
			})
		}
	}
	return result
}

// commentNameParts returns the unquoted name parts of a commented object.
func commentNameParts(n ast.Node) []string {
	switch v := n.(type) {
	case *ast.List:
		parts := make([]string, 0, len(v.Items))
		for _, item := range v.Items {
			parts = append(parts, commentNameParts(item)...)
		}
		return parts
	case *ast.ColumnRef:
		return commentNameParts(v.Fields)
	case *ast.ColumnNameExpr:
		if v.Name == nil {
			return nil
		}
		parts := []string{v.Name.Name.String()}
		if v.Name.Table.String() != "" {
			parts = append([]string{v.Name.Table.String()}, parts...)
		}
		if v.Name.Schema.String() != "" {
			parts = append([]string{v.Name.Schema.String()}, parts...)
		}
		return parts
	case *ast.String:
		return []string{v.Str}
	case *ast.TableName:
		parts := []string{}
		for _, part := range v.Name.Parts {
			parts = append(parts, part.Name)
		}
		return parts
	case *ast.Integer:
		return []string{strconv.FormatInt(v.Ival, 10)}
	case *ast.Float:
		return []string{v.Fval}
	case *ast.ObjectWithArgs:
		return commentNameParts(v.Objname)
	}
	return nil
}

// commentArgsSuffix returns the restored argument list of a routine reference.
func commentArgsSuffix(n ast.Node) string {
	obj, ok := n.(*ast.ObjectWithArgs)
	if !ok || obj.ArgsUnspecified {
		return ""
	}
	var sb strings.Builder
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)
	ctx.WritePlain("(")
	if obj.Objargs != nil {
		for i, arg := range obj.Objargs.Items {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if arg == nil {
				ctx.WriteKeyword("NONE")
			} else if err := arg.Restore(ctx); err != nil {
				return ""
			}
		}
	}
	ctx.WritePlain(")")
	return sb.String()
}

// commentNodeString rewrites a node as plain SQL, for the type parts of
// transform and cast comments.
func commentNodeString(n ast.Node) string {
	var sb strings.Builder
	if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err != nil {
		return ""
	}
	return sb.String()
}

// exprList1 returns the list of a single expression.
func exprList1(e ast.ExprNode) *ast.List {
	if e == nil {
		return nil
	}
	return &ast.List{Items: []ast.Node{e}}
}

// exprList returns the list of the given expressions.
func exprList(exprs []ast.ExprNode) *ast.List {
	if exprs == nil {
		return nil
	}
	items := make([]ast.Node, 0, len(exprs))
	for _, e := range exprs {
		items = append(items, e)
	}
	return &ast.List{Items: items}
}

// defElems returns the definition elements of a list.
func defElems(l *ast.List) []*ast.DefElem {
	if l == nil {
		return nil
	}
	var result []*ast.DefElem
	for _, item := range l.Items {
		if def, ok := item.(*ast.DefElem); ok {
			result = append(result, def)
		}
	}
	return result
}

// defElemsToDatabaseOptions returns the database options of a definition list.
func defElemsToDatabaseOptions(l *ast.List) []*ast.DatabaseOption {
	if l == nil {
		return nil
	}
	var result []*ast.DatabaseOption
	for _, item := range l.Items {
		def, ok := item.(*ast.DefElem)
		if !ok {
			continue
		}
		tp := ast.DatabaseOptionNone
		name := def.Defname
		switch strings.ToLower(def.Defname) {
		case "lc_collate":
			tp, name = ast.DatabaseOptionLCCollate, ""
		case "lc_ctype":
			tp, name = ast.DatabaseOptionLCType, ""
		case "tablespace":
			tp, name = ast.DatabaseOptionTablespace, ""
		case "encoding":
			tp, name = ast.DatabaseOptionEncoding, ""
		case "owner":
			tp, name = ast.DatabaseOptionOwner, ""
		case "template":
			tp, name = ast.DatabaseOptionTemplate, ""
		case "connection_limit":
			tp, name = ast.DatabaseOptionConnectionLimit, ""
		}
		value := ""
		switch arg := def.Arg.(type) {
		case *ast.String:
			value = arg.Str
		case *ast.Integer:
			value = strconv.FormatInt(arg.Ival, 10)
		case *ast.Float:
			value = arg.Fval
		}
		result = append(result, &ast.DatabaseOption{Tp: tp, Name: name, Value: value, Default: def.Arg == nil})
	}
	return result
}

// typeNameNode returns the node of a field type.
func typeNameNode(tp *types.FieldType) ast.Node {
	return &ast.DataType{Type: tp}
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

// nodeList returns the list of nodes of an expression list or a node list.
func nodeList(v any) *ast.List {
	switch x := v.(type) {
	case []ast.ExprNode:
		return exprList(x)
	case *ast.List:
		return x
	case ast.Node:
		return &ast.List{Items: []ast.Node{x}}
	}
	return nil
}

// jsonValueType returns the JSON value type of a JSON type code.
func jsonValueType(code int64) ast.JsonValueType {
	return ast.JsonValueType(code)
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

// exprOrNil returns the expression of a node, or nil when there is none.
func exprOrNil(n ast.Node) ast.ExprNode {
	if n == nil {
		return nil
	}
	if e, ok := n.(ast.ExprNode); ok {
		return e
	}
	return nil
}

// tableElements returns the columns and the constraints of a table element list.
func tableElements(l *ast.List) ([]*ast.ColumnDef, []*ast.Constraint, []*ast.TableLikeClause) {
	if l == nil {
		return nil, nil, nil
	}
	var cols []*ast.ColumnDef
	var constraints []*ast.Constraint
	var likes []*ast.TableLikeClause
	for _, item := range l.Items {
		switch v := item.(type) {
		case *ast.ColumnDef:
			cols = append(cols, v)
		case *ast.Constraint:
			constraints = append(constraints, v)
		case *ast.TableLikeClause:
			likes = append(likes, v)
		}
	}
	return cols, constraints, likes
}

// inheritTableNames returns the table names of an inherit list.
func inheritTableNames(l *ast.List) []*ast.TableName {
	if l == nil {
		return nil
	}
	var result []*ast.TableName
	for _, item := range l.Items {
		switch v := item.(type) {
		case *ast.RangeVar:
			result = append(result, syntaxTableName(nameFromParts(v.Schemaname, v.Relname)))
		case *ast.String:
			result = append(result, syntaxTableName(nameFromString(v.Str)))
		}
	}
	return result
}

// viewCheckOption returns the check option of a view.
func viewCheckOption(code int64) model.ViewCheckOption {
	if code == VIEW_CHECK_OPTION_CASCADED {
		return model.CheckOptionCascaded
	}
	return model.CheckOptionLocal
}

// defElemInt returns the integer value of a definition element.
func defElemInt(def *ast.DefElem) int64 {
	switch v := def.Arg.(type) {
	case *ast.Integer:
		return v.Ival
	case ast.ValueExpr:
		if i, ok := v.GetValue().(int64); ok {
			return i
		}
	}
	return 0
}

// transactionModes returns the transaction modes of a mode list.
func transactionModes(l *ast.List) (string, *bool, *bool) {
	if l == nil {
		return "", nil, nil
	}
	var isolation string
	var readOnly, deferrable *bool
	for _, item := range l.Items {
		def, ok := item.(*ast.DefElem)
		if !ok {
			continue
		}
		switch def.Defname {
		case "transaction_isolation":
			if v, ok := def.Arg.(ast.ValueExpr); ok {
				if str, ok := v.GetValue().(string); ok {
					isolation = strings.ToUpper(str)
				}
			}
		case "transaction_read_only":
			v := defElemInt(def) != 0
			readOnly = &v
		case "transaction_deferrable":
			v := defElemInt(def) != 0
			deferrable = &v
		}
	}
	return isolation, readOnly, deferrable
}

// explainOptionList returns the explain options with their values.
func explainOptionList(l *ast.List) []string {
	if l == nil {
		return nil
	}
	var result []string
	for _, item := range l.Items {
		def, ok := item.(*ast.DefElem)
		if !ok {
			continue
		}
		option := def.Defname
		switch v := def.Arg.(type) {
		case *ast.String:
			option += " " + v.Str
		case *ast.Integer:
			option += " " + strconv.FormatInt(v.Ival, 10)
		}
		result = append(result, option)
	}
	return result
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

// inBraces marks a query as written in braces.
func inBraces(n ast.Node) ast.Node {
	switch v := n.(type) {
	case *ast.SelectStmt:
		v.IsInBraces = true
		return v
	case *ast.QueryTailStmt:
		if v.Parenthesized {
			return v
		}
		return &ast.QueryTailStmt{BaseQueryTailStmt: pkast.BaseQueryTailStmt{Query: v, Parenthesized: true}}
	case *ast.SetOprStmt:
		return &ast.QueryTailStmt{BaseQueryTailStmt: pkast.BaseQueryTailStmt{Query: v, Parenthesized: true}}
	case *ast.ValuesStmt, *ast.TableStmt:
		return &ast.QueryTailStmt{BaseQueryTailStmt: pkast.BaseQueryTailStmt{Query: queryNodeOf(v), Parenthesized: true}}
	default:
		return n
	}
}

// databaseOptions returns the options of a database option list.
func databaseOptions(l *ast.List) []*ast.DatabaseOption {
	return defElemsToDatabaseOptions(l)
}

// dropObjectStmt returns the drop statement of an object type.
func dropObjectStmt(removeType int, objects []*ast.TableName, ifExists bool, behavior int) ast.Node {
	switch ast.ObjectType(removeType) {
	case ast.OBJECT_TABLE:
		return &ast.DropTableStmt{
			IfExists: ifExists,
			Tables:   objects,
		}
	case ast.OBJECT_SEQUENCE:
		return &ast.DropSequenceStmt{
			IfExists: ifExists,
			Names:    objects,
		}
	case ast.OBJECT_VIEW:
		return &ast.DropTableStmt{
			Tables: objects,
			IsView: true,
		}
	case ast.OBJECT_INDEX:
		if len(objects) > 0 {
			return &ast.DropIndexStmt{
				IfExists:  ifExists,
				IndexName: indexNameFromTableName(objects[0]),
			}
		}
	case ast.OBJECT_FUNCTION:
		stmt := &ast.DropFunctionStmt{IfExists: ifExists, Names: objects, Behavior: ast.DropBehavior(behavior)}
		if len(objects) > 0 {
			stmt.Name = objects[0]
		}
		return stmt
	case ast.OBJECT_PROCEDURE:
		stmt := &ast.DropProcedureStmt{IfExists: ifExists, Names: objects, Behavior: ast.DropBehavior(behavior)}
		if len(objects) > 0 {
			stmt.Name = objects[0]
		}
		return stmt
	}
	return &ast.DropStmt{
		Objects:    objects,
		RemoveType: removeType,
		Behavior:   behavior,
		Missing_ok: ifExists,
	}
}

// truncateOnly returns the ONLY marker of a truncate statement.
func truncateOnly(list *ast.List) bool {
	if list == nil || len(list.Items) == 0 {
		return false
	}
	src, ok := list.Items[0].(*ast.TableSource)
	return ok && src.Only
}

// dropBehaviorName returns the name of a drop behavior.
func dropBehaviorName(behavior int) string {
	if ast.DropBehavior(behavior) == ast.DROP_CASCADE {
		return "CASCADE"
	}
	return "RESTRICT"
}

// showStmtType returns the show statement type of a variable name.
func showStmtType(name string) ast.ShowStmtType {
	switch strings.ToLower(name) {
	case "databases", "schemas":
		return ast.ShowDatabases
	case "tables":
		return ast.ShowTables
	case "columns", "fields":
		return ast.ShowColumns
	case "index", "indexes", "keys":
		return ast.ShowIndex
	case "create table":
		return ast.ShowCreateTable
	case "create view":
		return ast.ShowCreateView
	case "grants":
		return ast.ShowGrants
	case "warnings":
		return ast.ShowWarnings
	case "status":
		return ast.ShowStatus
	case "engines":
		return ast.ShowEngines
	case "charset":
		return ast.ShowCharset
	case "collation":
		return ast.ShowCollation
	case "variables":
		return ast.ShowVariables
	}
	return ast.ShowVariables
}

// operatorExpr returns the expression of a binary operator.
func operatorExpr(name string, l, r ast.ExprNode) ast.ExprNode {
	switch name {
	case "~~":
		return &ast.PatternLikeExpr{Expr: l, Pattern: r}
	case "!~~":
		return &ast.PatternLikeExpr{Expr: l, Pattern: r, Not: true}
	case "~~*":
		return &ast.PatternLikeExpr{Expr: l, Pattern: r, ILike: true}
	case "!~~*":
		return &ast.PatternLikeExpr{Expr: l, Pattern: r, Not: true, ILike: true}
	}
	return ast.NewBinaryOperationExprByName(name, l, r)
}

// indexIncludeNames returns the column names of an index include list.
func indexIncludeNames(l *ast.List) []*ast.ColumnName {
	if l == nil {
		return nil
	}
	var result []*ast.ColumnName
	for _, item := range l.Items {
		switch v := item.(type) {
		case *ast.IndexElem:
			result = append(result, &ast.ColumnName{Name: model.NewCIStr(v.Name)})
		case *ast.String:
			result = append(result, &ast.ColumnName{Name: model.NewCIStr(v.Str)})
		}
	}
	return result
}

// setLocalStatement returns the local set statement of a set clause.
func setLocalStatement(n ast.Node) ast.Node {
	if v, ok := n.(*ast.VariableSetStmt); ok {
		v.IsLocal = true
	}
	return n
}

// doExprs returns the body expressions of a do statement.
func doExprs(l *ast.List) []ast.ExprNode {
	if l == nil {
		return nil
	}
	var result []ast.ExprNode
	for _, item := range l.Items {
		def, ok := item.(*ast.DefElem)
		if !ok {
			continue
		}
		if def.Defname != "as" {
			continue
		}
		if s, ok := def.Arg.(*ast.String); ok {
			result = append(result, ast.NewValueExpr(s.Str))
		}
	}
	return result
}

// doLanguage returns the LANGUAGE option of a DO statement.
func doLanguage(l *ast.List) string {
	if l == nil {
		return ""
	}
	for _, item := range l.Items {
		def, ok := item.(*ast.DefElem)
		if !ok || def.Defname != "language" {
			continue
		}
		if s, ok := def.Arg.(*ast.String); ok {
			return s.Str
		}
	}
	return ""
}

// dropFunctionStmt returns the drop function statement of a function list.
func dropFunctionStmt(ifExists bool, l *ast.List, behavior int) ast.Node {
	names := objectWithArgsNames(l)
	if len(names) == 0 {
		return &ast.DropFunctionStmt{IfExists: ifExists}
	}
	return &ast.DropFunctionStmt{
		IfExists:  ifExists,
		Name:      names[0],
		Names:     names,
		Arguments: routineArguments(l),
		Behavior:  ast.DropBehavior(behavior),
	}
}

// dropProcedureStmt returns the drop procedure statement of a function list.
func dropProcedureStmt(ifExists bool, l *ast.List, behavior int) ast.Node {
	names := objectWithArgsNames(l)
	if len(names) == 0 {
		return &ast.DropProcedureStmt{IfExists: ifExists}
	}
	return &ast.DropProcedureStmt{
		IfExists:  ifExists,
		Name:      names[0],
		Names:     names,
		Arguments: routineArguments(l),
		Behavior:  ast.DropBehavior(behavior),
	}
}

func routineArguments(l *ast.List) []*ast.List {
	args := make([]*ast.List, 0, len(l.Items))
	for _, item := range l.Items {
		obj, ok := item.(*ast.ObjectWithArgs)
		if !ok {
			continue
		}
		var signature *ast.List
		if !obj.ArgsUnspecified {
			signature = obj.Objargs
			if signature == nil {
				signature = &ast.List{}
			}
		}
		args = append(args, signature)
	}
	return args
}

// serverOptions returns the server options of a generic option list.
func serverOptions(l *ast.List) []*ast.ServerOption {
	if l == nil {
		return nil
	}
	var result []*ast.ServerOption
	for _, item := range l.Items {
		def, ok := item.(*ast.DefElem)
		if !ok {
			continue
		}
		opt := &ast.ServerOption{Name: def.Defname}
		if s, ok := def.Arg.(*ast.String); ok {
			opt.Value = s.Str
		}
		result = append(result, opt)
	}
	return result
}

// resourceGroupVCPU returns the vcpu ranges of a resource group option list.
func resourceGroupVCPU(l *ast.List) []*ast.ResourceGroupVCPU {
	if l == nil {
		return nil
	}
	var result []*ast.ResourceGroupVCPU
	for _, item := range l.Items {
		def, ok := item.(*ast.DefElem)
		if !ok || def.Defname != "vcpu" {
			continue
		}
		switch v := def.Arg.(type) {
		case *ast.List:
			if len(v.Items) < 2 {
				continue
			}
			from, ok1 := v.Items[0].(ast.ExprNode)
			to, ok2 := v.Items[1].(ast.ExprNode)
			if !ok1 || !ok2 {
				continue
			}
			result = append(result, &ast.ResourceGroupVCPU{
				From: uint64(exprIntValue(from)),
				To:   uint64(exprIntValue(to)),
			})
		case ast.ExprNode:
			result = append(result, &ast.ResourceGroupVCPU{
				From: uint64(exprIntValue(v)),
				To:   uint64(exprIntValue(v)),
			})
		}
	}
	return result
}

// resourceGroupThreadPriority returns the thread priority of a resource group option list.
func resourceGroupThreadPriority(l *ast.List) *int {
	if l == nil {
		return nil
	}
	for _, item := range l.Items {
		def, ok := item.(*ast.DefElem)
		if !ok || def.Defname != "thread_priority" {
			continue
		}
		value, ok := def.Arg.(ast.ExprNode)
		if !ok {
			continue
		}
		priority := int(exprIntValue(value))
		return &priority
	}
	return nil
}

// resourceGroupState returns the state of a resource group option list.
func resourceGroupState(l *ast.List) int {
	if l == nil {
		return ast.ResourceGroupStateNone
	}
	for _, item := range l.Items {
		def, ok := item.(*ast.DefElem)
		if !ok || def.Defname != "state" {
			continue
		}
		value, ok := def.Arg.(ast.ExprNode)
		if !ok {
			continue
		}
		if s, ok := value.(ast.ValueExpr); ok {
			if v, ok := s.GetValue().(string); ok && v == "disable" {
				return ast.ResourceGroupStateDisable
			}
		}
		return ast.ResourceGroupStateEnable
	}
	return ast.ResourceGroupStateNone
}

// exprIntValue returns the integer value of an expression.
func exprIntValue(e ast.ExprNode) int64 {
	value, ok := e.(ast.ValueExpr)
	if !ok {
		return 0
	}
	switch v := value.GetValue().(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case uint64:
		return int64(v)
	case uint:
		return int64(v)
	}
	return 0
}

// dropRoutineStmt returns the drop routine statement of a function list.
func dropRoutineStmt(ifExists bool, l *ast.List) ast.Node {
	return &ast.DropStmt{
		RemoveType: int(ast.OBJECT_ROUTINE),
		Objects:    tableNamesFromObjectWithArgs(l),
		Missing_ok: ifExists,
	}
}

// dropTriggerStmt returns the drop trigger statement of a trigger name and table.
func dropTriggerStmt(ifExists bool, name string, table ast.Node) ast.Node {
	stmt := &ast.DropTriggerStmt{
		IfExists: ifExists,
	}
	switch v := table.(type) {
	case *ast.RangeVar:
		stmt.Table = syntaxTableName(nameFromParts(v.Schemaname, v.Relname))
	case *ast.List:
		stmt.Table = tableNameFromList(v)
	}
	stmt.Name = syntaxTableName(nameFromString(name))
	return stmt
}

// resultSetNodeOf returns the result set node of a table reference.
func resultSetNodeOf(n ast.Node) ast.ResultSetNode {
	switch v := n.(type) {
	case *ast.TableSource:
		return v
	case *ast.Join:
		return v
	}
	return nil
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
