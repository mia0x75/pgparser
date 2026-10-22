package parser

import (
	"strconv"
	"strings"

	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/auth"
	"github.com/mia0x75/pgparser/model"
	"github.com/mia0x75/pgparser/postgres"
	"github.com/mia0x75/pgparser/types"
)

// defElemNames returns the names of the definition elements of a list.
func defElemNames(l interface{}) []string {
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
	for _, n := range defElemNames(l) {
		if n == name {
			return true
		}
	}
	return false
}

// tableNameFromList returns the table name of a qualified name list.
func tableNameFromList(l *ast.List) *ast.TableName {
	name := &ast.TableName{}
	if l == nil {
		return name
	}
	switch len(l.Items) {
	case 1:
		if s, ok := l.Items[0].(*ast.String); ok {
			name.Name = model.NewCIStr(s.Str)
		}
	case 2:
		if s, ok := l.Items[0].(*ast.String); ok {
			name.Schema = model.NewCIStr(s.Str)
		}
		if s, ok := l.Items[1].(*ast.String); ok {
			name.Name = model.NewCIStr(s.Str)
		}
	case 3:
		if s, ok := l.Items[0].(*ast.String); ok {
			name.Schema = model.NewCIStr(s.Str)
		}
		if s, ok := l.Items[1].(*ast.String); ok {
			name.Name = model.NewCIStr(s.Str)
		}
	}
	return name
}

// tableNamesFromList returns the table names of a list.
func tableNamesFromList(l *ast.List) []*ast.TableName {
	if l == nil {
		return nil
	}
	var result []*ast.TableName
	for _, item := range l.Items {
		switch v := item.(type) {
		case *ast.TableName:
			result = append(result, v)
		case *ast.String:
			result = append(result, &ast.TableName{Name: model.NewCIStr(v.Str)})
		case *ast.List:
			result = append(result, tableNameFromList(v))
		case *ast.RangeVar:
			result = append(result, &ast.TableName{
				Schema: model.NewCIStr(v.Schemaname),
				Name:   model.NewCIStr(v.Relname),
			})
		case *ast.TableSource:
			if name, ok := v.Source.(*ast.TableName); ok {
				result = append(result, name)
			}
		case *ast.VacuumRelation:
			result = append(result, tableNameFromList(nil))
		}
	}
	return result
}

// nameFromList returns the name of a qualified name list.
func nameFromList(l *ast.List) string {
	name := tableNameFromList(l)
	if name.Schema.String() != "" {
		return name.Schema.String() + "." + name.Name.String()
	}
	return name.Name.String()
}

// indexName returns the index name of a cluster index specification.
func indexName(name string) *ast.TableName {
	if name == "" {
		return nil
	}
	return &ast.TableName{Name: model.NewCIStr(name)}
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

// roleSpecName returns the name of a role specification.
func roleSpecName(n ast.Node) string {
	if spec, ok := n.(*ast.RoleSpec); ok {
		return spec.Rolename
	}
	return ""
}

// roleIdentity returns the role identity of a role specification.
func roleIdentity(n ast.Node) *auth.RoleIdentity {
	if spec, ok := n.(*ast.RoleSpec); ok {
		return &auth.RoleIdentity{Username: spec.Rolename}
	}
	return nil
}

// roleIdentities returns the role identities of a role list.
func roleIdentities(l *ast.List) []*auth.RoleIdentity {
	if l == nil {
		return nil
	}
	var result []*auth.RoleIdentity
	for _, item := range l.Items {
		switch v := item.(type) {
		case *ast.RoleSpec:
			result = append(result, &auth.RoleIdentity{Username: v.Rolename})
		case *ast.String:
			result = append(result, &auth.RoleIdentity{Username: v.Str})
		}
	}
	return result
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

// relationName returns the table name of a relation expression.
func relationName(n ast.Node) *ast.TableName {
	switch v := n.(type) {
	case *ast.TableSource:
		if name, ok := v.Source.(*ast.TableName); ok {
			return name
		}
	case *ast.RangeVar:
		return &ast.TableName{
			Schema: model.NewCIStr(v.Schemaname),
			Name:   model.NewCIStr(v.Relname),
		}
	}
	return nil
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
				User:   &auth.UserIdentity{Username: v.Rolename},
				IsRole: true,
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

// privElems returns the privilege elements of a privilege list.
func privElems(l *ast.List) []*ast.PrivElem {
	if l == nil {
		return nil
	}
	var result []*ast.PrivElem
	for _, item := range l.Items {
		priv, ok := item.(*ast.AccessPriv)
		if !ok {
			continue
		}
		elem := &ast.PrivElem{Priv: privilegeType(priv.PrivName)}
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

// objectTypeNameOf returns the name of an object type constant.
// objectTypeNameOf returns the name of an object type.
func objectTypeNameOf(name string) string {
	return strings.ToLower(strings.TrimPrefix(name, "OBJECT_"))
}

// commentObjectName returns the name of a commented object.
func commentObjectName(n ast.Node) string {
	switch v := n.(type) {
	case *ast.List:
		return nameListString(v)
	case *ast.ColumnRef:
		return nameListString(v.Fields)
	case *ast.ColumnNameExpr:
		if v.Name == nil {
			return ""
		}
		if v.Name.Schema.String() != "" {
			return v.Name.Schema.String() + "." + v.Name.Table.String() + "." + v.Name.Name.String()
		}
		if v.Name.Table.String() != "" {
			return v.Name.Table.String() + "." + v.Name.Name.String()
		}
		return v.Name.Name.String()
	case *ast.String:
		return v.Str
	case *ast.TableName:
		if v.Schema.String() != "" {
			return v.Schema.String() + "." + v.Name.String()
		}
		return v.Name.String()
	}
	return ""
}

// tableNamesFromNameList returns the table names of a name list.
func tableNamesFromNameList(l *ast.List) []*ast.TableName {
	if l == nil {
		return nil
	}
	var result []*ast.TableName
	for _, item := range l.Items {
		switch v := item.(type) {
		case *ast.String:
			result = append(result, &ast.TableName{Name: model.NewCIStr(v.Str)})
		case *ast.List:
			result = append(result, tableNameFromList(v))
		}
	}
	return result
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

// columnNameStrings returns the column names of a column list.
func columnNameStrings(l *ast.List) []model.CIStr {
	if l == nil {
		return nil
	}
	var result []model.CIStr
	for _, item := range l.Items {
		if s, ok := item.(*ast.String); ok {
			result = append(result, model.NewCIStr(s.Str))
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
		switch strings.ToLower(def.Defname) {
		case "tablespace":
			tp = ast.DatabaseOptionTablespace
		case "encoding":
			tp = ast.DatabaseOptionEncoding
		case "owner":
			tp = ast.DatabaseOptionOwner
		case "template":
			tp = ast.DatabaseOptionTemplate
		case "connection_limit":
			tp = ast.DatabaseOptionConnectionLimit
		case "collate", "lc_collate":
			tp = ast.DatabaseOptionCollate
		}
		value := ""
		switch arg := def.Arg.(type) {
		case *ast.String:
			value = arg.Str
		case *ast.Integer:
			value = strconv.FormatInt(arg.Ival, 10)
		}
		result = append(result, &ast.DatabaseOption{Tp: tp, Value: value})
	}
	return result
}

// funcParams returns the function parameters of a parameter list.
func funcParams(l *ast.List) []*ast.FuncParam {
	if l == nil {
		return nil
	}
	var result []*ast.FuncParam
	for _, item := range l.Items {
		v, ok := item.(*ast.FunctionParameter)
		if ok {
			param := &ast.FuncParam{
				Name: v.Name,
				Type: v.ArgType,
				Dir:  ast.FuncParamDirIn,
			}
			switch v.Mode {
			case ast.FUNC_PARAM_OUT:
				param.Dir = ast.FuncParamDirOut
			case ast.FUNC_PARAM_INOUT:
				param.Dir = ast.FuncParamDirInOut
			}
			result = append(result, param)
		}
	}
	return result
}

// typeNameNode returns the node of a field type.
func typeNameNode(tp *types.FieldType) ast.Node {
	return &ast.String{Str: tp.String()}
}

// objectName returns the name of an object with arguments.
func objectName(n interface{}) string {
	switch v := n.(type) {
	case *ast.String:
		return v.Str
	case *ast.List:
		return funcNameString(v)
	case string:
		return v
	case ast.Node:
		if s, ok := v.(*ast.String); ok {
			return s.Str
		}
	}
	return ""
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
func nodeList(v interface{}) *ast.List {
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
	return newDataType(strings.Join(parts, "."), -1, -1)
}

// aliasName returns the name of an alias.
func aliasName(n ast.Node) string {
	if alias, ok := n.(*ast.Alias); ok {
		return alias.Aliasname
	}
	return ""
}

// relOptionsString returns the relation options of a list.
func relOptionsString(l *ast.List) string {
	if l == nil {
		return ""
	}
	var parts []string
	for _, item := range l.Items {
		if def, ok := item.(*ast.DefElem); ok {
			parts = append(parts, def.Defname)
		}
	}
	return strings.Join(parts, ", ")
}

// accessMethodName returns the name of an access method.
func accessMethodName(name string) string {
	return name
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

// relOptionString returns the relation options of a list with their values.
func relOptionString(l *ast.List) string {
	if l == nil {
		return ""
	}
	var parts []string
	for _, item := range l.Items {
		def, ok := item.(*ast.DefElem)
		if !ok {
			continue
		}
		if def.Arg == nil {
			parts = append(parts, def.Defname)
			continue
		}
		parts = append(parts, def.Defname+" = ?")
	}
	return strings.Join(parts, ", ")
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
func tableElements(l *ast.List) ([]*ast.ColumnDef, []*ast.Constraint) {
	if l == nil {
		return nil, nil
	}
	var cols []*ast.ColumnDef
	var constraints []*ast.Constraint
	for _, item := range l.Items {
		switch v := item.(type) {
		case *ast.ColumnDef:
			cols = append(cols, v)
		case *ast.Constraint:
			constraints = append(constraints, v)
		}
	}
	return cols, constraints
}

// partitionStrategy returns the partition strategy of a partition specification.
func partitionStrategy(n ast.Node) string {
	spec, ok := n.(*ast.PartitionSpec)
	if !ok || spec == nil {
		return ""
	}
	switch spec.Strategy {
	case "r":
		return "range"
	case "h":
		return "hash"
	case "l":
		return "list"
	}
	return spec.Strategy
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
			result = append(result, &ast.TableName{
				Schema: model.NewCIStr(v.Schemaname),
				Name:   model.NewCIStr(v.Relname),
			})
		case *ast.String:
			result = append(result, &ast.TableName{Name: model.NewCIStr(v.Str)})
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

// sequenceOptions returns the sequence options of a definition list.
func sequenceOptions(l *ast.List) []*ast.SequenceOption {
	if l == nil {
		return nil
	}
	var result []*ast.SequenceOption
	for _, item := range l.Items {
		def, ok := item.(*ast.DefElem)
		if !ok {
			continue
		}
		opt := &ast.SequenceOption{Val: defElemInt(def)}
		switch def.Defname {
		case "increment":
			opt.Tp = ast.SequenceOptionIncrement
		case "minvalue":
			opt.Tp = ast.SequenceOptionMinValue
		case "nomaxvalue":
			opt.Tp = ast.SequenceOptionNoMaxValue
		case "maxvalue":
			opt.Tp = ast.SequenceOptionMaxValue
		case "start":
			opt.Tp = ast.SequenceOptionStart
		case "cache":
			opt.Tp = ast.SequenceOptionCache
		case "cycle":
			if defElemInt(def) != 0 {
				opt.Tp = ast.SequenceOptionCycle
			} else {
				opt.Tp = ast.SequenceOptionNoCycle
			}
		default:
			continue
		}
		result = append(result, opt)
	}
	return result
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

// nameFromList returns the dotted name of a name list.

// setRoleStmt returns the set role statement of a role list.
func setRoleStmt(l *ast.List) ast.Node {
	stmt := &ast.SetRoleStmt{}
	if l == nil {
		return stmt
	}
	var roles []*auth.RoleIdentity
	for _, item := range l.Items {
		switch v := item.(type) {
		case *ast.String:
			switch strings.ToLower(v.Str) {
			case "none":
				stmt.SetRoleOpt = ast.SetRoleNone
				continue
			case "default":
				stmt.SetRoleOpt = ast.SetRoleDefault
				continue
			}
			roles = append(roles, &auth.RoleIdentity{Username: v.Str})
		case ast.ValueExpr:
			if s, ok := v.GetValue().(string); ok {
				switch strings.ToLower(s) {
				case "none":
					stmt.SetRoleOpt = ast.SetRoleNone
					continue
				case "default":
					stmt.SetRoleOpt = ast.SetRoleDefault
					continue
				}
				roles = append(roles, &auth.RoleIdentity{Username: s})
			}
		}
	}
	stmt.RoleList = roles
	return stmt
}

// setRoleStmtFromName returns the set role statement of a role name.
func setRoleStmtFromName(name string) ast.Node {
	stmt := &ast.SetRoleStmt{}
	switch strings.ToLower(name) {
	case "none":
		stmt.SetRoleOpt = ast.SetRoleNone
		return stmt
	case "default":
		stmt.SetRoleOpt = ast.SetRoleDefault
		return stmt
	}
	stmt.RoleList = []*auth.RoleIdentity{{Username: name}}
	return stmt
}

// createUserStmt returns the create user statement of a role.
func createUserStmt(role bool, name string, options *ast.List) ast.Node {
	spec := &ast.UserSpec{
		User:   &auth.UserIdentity{Username: name},
		IsRole: role,
	}
	if opt := authOption(options); opt != nil {
		spec.AuthOpt = opt
	}
	return &ast.CreateUserStmt{
		IsCreateRole: role,
		Specs:        []*ast.UserSpec{spec},
	}
}

// alterUserStmt returns the alter user statement of a role.
func alterUserStmt(role bool, spec ast.Node, options *ast.List) ast.Node {
	stmt := &ast.AlterUserStmt{
		IsAlterRole: role,
	}
	if s, ok := spec.(*ast.RoleSpec); ok {
		stmt.Specs = []*ast.UserSpec{{
			User:   &auth.UserIdentity{Username: s.Rolename},
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
				stmt.UserList = append(stmt.UserList, &auth.UserIdentity{Username: v.Rolename})
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

// inBraces marks a select statement as written in braces.
func inBraces(n ast.Node) ast.Node {
	switch v := n.(type) {
	case *ast.SelectStmt:
		v.IsInBraces = true
		return v
	case *ast.SetOprStmt:
		v.IsInBraces = true
		return v
	}
	return n
}

// databaseOptions returns the options of a database option list.
func databaseOptions(l *ast.List) []*ast.DatabaseOption {
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
		switch strings.ToLower(def.Defname) {
		case "owner":
			tp = ast.DatabaseOptionOwner
		case "template":
			tp = ast.DatabaseOptionTemplate
		case "connection_limit":
			tp = ast.DatabaseOptionConnectionLimit
		case "encoding":
			tp = ast.DatabaseOptionEncoding
		}
		result = append(result, &ast.DatabaseOption{Tp: tp, Value: defElemString(def)})
	}
	return result
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
				IndexName: objects[0].Name.O,
			}
		}
	}
	return &ast.DropStmt{
		Objects:    objects,
		RemoveType: removeType,
		Behavior:   behavior,
		Missing_ok: ifExists,
	}
}

// variableAssignments returns the variable assignments of a set clause.
func variableAssignments(name string, values *ast.List) []*ast.VariableAssignment {
	assignment := &ast.VariableAssignment{Name: name}
	if values != nil {
		for _, item := range values.Items {
			if e, ok := item.(ast.ExprNode); ok {
				assignment.Value = e
				break
			}
		}
	}
	return []*ast.VariableAssignment{assignment}
}

// functionObjectName returns the name of a function with arguments.
func functionObjectName(n ast.Node) string {
	switch v := n.(type) {
	case *ast.ObjectWithArgs:
		return nameListString(v.Objname)
	case *ast.List:
		return nameListString(v)
	}
	return ""
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

// aliasColumnNames returns the column names of an alias clause.
func aliasColumnNames(n ast.Node) []model.CIStr {
	switch v := n.(type) {
	case *ast.Alias:
		return columnNameStrings(v.Colnames)
	case *ast.List:
		if len(v.Items) > 1 {
			if cols, ok := v.Items[1].(*ast.List); ok {
				var names []model.CIStr
				for _, item := range cols.Items {
					if def, ok := item.(*ast.ColumnDef); ok && def.Name != nil {
						names = append(names, def.Name.Name)
					}
				}
				return names
			}
		}
		if len(v.Items) > 0 {
			if alias, ok := v.Items[0].(*ast.Alias); ok {
				return columnNameStrings(alias.Colnames)
			}
		}
	}
	return nil
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
		if s, ok := def.Arg.(*ast.String); ok {
			result = append(result, ast.NewValueExpr(s.Str))
		}
	}
	return result
}

// tableNameFromSource returns the table name of a table source.
func tableNameFromSource(n ast.Node) *ast.TableName {
	if src, ok := n.(*ast.TableSource); ok {
		if name, ok := src.Source.(*ast.TableName); ok {
			return name
		}
	}
	return nil
}

// dropFunctionStmt returns the drop function statement of a function list.
func dropFunctionStmt(ifExists bool, l *ast.List) ast.Node {
	names := objectWithArgsNames(l)
	if len(names) == 0 {
		return &ast.DropFunctionStmt{IfExists: ifExists}
	}
	return &ast.DropFunctionStmt{
		IfExists: ifExists,
		Name:     names[0],
		Names:    names,
	}
}

// dropProcedureStmt returns the drop procedure statement of a function list.
func dropProcedureStmt(ifExists bool, l *ast.List) ast.Node {
	names := objectWithArgsNames(l)
	if len(names) == 0 {
		return &ast.DropProcedureStmt{IfExists: ifExists}
	}
	return &ast.DropProcedureStmt{
		IfExists: ifExists,
		Name:     names[0],
		Names:    names,
	}
}

// objectWithArgsNames returns the names of an object list.
func objectWithArgsNames(l *ast.List) []*ast.TableName {
	if l == nil {
		return nil
	}
	var names []*ast.TableName
	for _, item := range l.Items {
		obj, ok := item.(*ast.ObjectWithArgs)
		if !ok {
			continue
		}
		var parts []string
		for _, name := range obj.Objname.Items {
			s, ok := name.(*ast.String)
			if !ok {
				continue
			}
			parts = append(parts, s.Str)
		}
		switch len(parts) {
		case 0:
			continue
		case 1:
			names = append(names, &ast.TableName{Name: model.NewCIStr(parts[0])})
		default:
			names = append(names, &ast.TableName{
				Schema: model.NewCIStr(parts[0]),
				Name:   model.NewCIStr(parts[1]),
			})
		}
	}
	return names
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

// tablespaceOptions returns the tablespace options of a location and reloptions.
func tablespaceOptions(location string, l *ast.List) []*ast.TablespaceOption {
	options := []*ast.TablespaceOption{
		{Tp: ast.TablespaceOptionLocation, Str: location},
	}
	return append(options, tablespaceOptionList(l)...)
}

// tablespaceOptionList returns the tablespace options of a reloption list.
func tablespaceOptionList(l *ast.List) []*ast.TablespaceOption {
	if l == nil {
		return nil
	}
	var result []*ast.TablespaceOption
	for _, item := range l.Items {
		def, ok := item.(*ast.DefElem)
		if !ok {
			continue
		}
		opt := &ast.TablespaceOption{Str: def.Defname}
		if s, ok := def.Arg.(*ast.String); ok {
			opt.Str = s.Str
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

// tableNamesFromObjectWithArgs returns the names of an object with args list.
func tableNamesFromObjectWithArgs(l *ast.List) []*ast.TableName {
	return objectWithArgsNames(l)
}

// dropTriggerStmt returns the drop trigger statement of a trigger name and table.
func dropTriggerStmt(ifExists bool, name string, table ast.Node) ast.Node {
	stmt := &ast.DropTriggerStmt{
		IfExists: ifExists,
	}
	if n, ok := table.(*ast.RangeVar); ok {
		stmt.Table = &ast.TableName{
			Schema: model.NewCIStr(n.Schemaname),
			Name:   model.NewCIStr(n.Relname),
		}
	}
	stmt.Name = &ast.TableName{Name: model.NewCIStr(name)}
	return stmt
}

// firstName returns the first name of a name list.
func firstName(l *ast.List) string {
	if l == nil {
		return ""
	}
	for _, item := range l.Items {
		if s, ok := item.(*ast.String); ok {
			return s.Str
		}
	}
	return ""
}

// indexNameFromList returns the index name of a name list.
func indexNameFromList(l *ast.List) string {
	if l == nil {
		return ""
	}
	return nameFromList(l)
}

// columnNameListFromStrings returns the column names of a string list.
func columnNameListFromStrings(l *ast.List) []*ast.ColumnName {
	if l == nil {
		return nil
	}
	var cols []*ast.ColumnName
	for _, item := range l.Items {
		s, ok := item.(*ast.String)
		if !ok {
			continue
		}
		cols = append(cols, &ast.ColumnName{Name: model.NewCIStr(s.Str)})
	}
	return cols
}

// routineBodyStmt returns the body statement of a routine body.
func routineBodyStmt(n ast.Node) ast.StmtNode {
	if n == nil {
		return nil
	}
	if stmt, ok := n.(ast.StmtNode); ok {
		return stmt
	}
	return nil
}

// routineDefinition returns the definition string of a routine option list.
func routineDefinition(l *ast.List) string {
	if l == nil {
		return ""
	}
	for _, item := range l.Items {
		def, ok := item.(*ast.DefElem)
		if !ok || def.Defname != "as" {
			continue
		}
		list, ok := def.Arg.(*ast.List)
		if !ok {
			continue
		}
		for _, name := range list.Items {
			if s, ok := name.(*ast.String); ok {
				return s.Str
			}
		}
	}
	return ""
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

// indexColNamesFromElems returns the index column names of an index element list.
func indexColNamesFromElems(l *ast.List) []*ast.IndexColName {
	if l == nil {
		return nil
	}
	var result []*ast.IndexColName
	for _, item := range l.Items {
		elem, ok := item.(*ast.IndexElem)
		if !ok {
			continue
		}
		col := &ast.IndexColName{
			Opclass: listFirstString(elem.Opclass),
		}
		if elem.Name != "" {
			col.Column = &ast.ColumnName{Name: model.NewCIStr(elem.Name)}
		}
		if expr, ok := elem.Expr.(ast.ExprNode); ok {
			col.Expr = expr
		}
		if elem.Ordering == ast.SORTBY_DESC {
			col.Desc = true
		}
		switch elem.NullsOrdering {
		case ast.SORTBY_NULLS_FIRST:
			col.NullsOrder = 1
		case ast.SORTBY_NULLS_LAST:
			col.NullsOrder = 2
		}
		result = append(result, col)
	}
	return result
}

// listFirstString returns the first string of a list.
func listFirstString(l *ast.List) string {
	if l == nil {
		return ""
	}
	for _, item := range l.Items {
		if s, ok := item.(*ast.String); ok {
			return s.Str
		}
	}
	return ""
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

// xmlTableColumns returns the column definitions of an XMLTABLE table function.
func xmlTableColumns(l *ast.List) []*ast.XMLTableColumn {
	if l == nil {
		return nil
	}
	var result []*ast.XMLTableColumn
	for _, item := range l.Items {
		col, ok := item.(*ast.XMLTableColumn)
		if !ok {
			continue
		}
		result = append(result, col)
	}
	return result
}

// xmlNamespaces returns the namespace definitions of an XMLTABLE table function.
func xmlNamespaces(l *ast.List) []*ast.XMLTableNamespace {
	if l == nil {
		return nil
	}
	var result []*ast.XMLTableNamespace
	for _, item := range l.Items {
		ns, ok := item.(*ast.XMLTableNamespace)
		if !ok {
			continue
		}
		result = append(result, ns)
	}
	return result
}

// applyXMLTableColumnOptions applies the column options of an XMLTABLE column.
func applyXMLTableColumnOptions(col *ast.XMLTableColumn, l *ast.List) {
	if col == nil || l == nil {
		return
	}
	for _, item := range l.Items {
		def, ok := item.(*ast.DefElem)
		if !ok {
			continue
		}
		switch def.Defname {
		case "default":
			if expr, ok := def.Arg.(ast.ExprNode); ok {
				col.DefaultExpr = expr
			}
		case "path":
			if expr, ok := def.Arg.(ast.ExprNode); ok {
				col.PathExpr = expr
			}
		case "__pg__is_not_null":
			if expr, ok := def.Arg.(ast.ExprNode); ok {
				if value, ok := expr.(ast.ValueExpr); ok {
					if b, ok := value.GetValue().(bool); ok {
						col.IsNotNull = b
					}
				}
			}
		}
	}
}

// jsonTableColumns returns the column definitions of a JSON_TABLE table function.
func jsonTableColumns(l *ast.List) []*ast.JSONTableColumn {
	if l == nil {
		return nil
	}
	var result []*ast.JSONTableColumn
	for _, item := range l.Items {
		col, ok := item.(*ast.JSONTableColumn)
		if !ok {
			continue
		}
		result = append(result, col)
	}
	return result
}

// jsonTablePathSpec returns the path specification of a JSON_TABLE column.
func jsonTablePathSpec(n ast.Node) *ast.JSONTablePathSpec {
	if n == nil {
		return nil
	}
	spec, ok := n.(*ast.JSONTablePathSpec)
	if !ok {
		return nil
	}
	return spec
}

// jsonWrapperName returns the wrapper behavior name of a JSON_TABLE column.
func jsonWrapperName(value int64) string {
	switch ast.JsonWrapper(value) {
	case ast.JSW_CONDITIONAL:
		return "conditional"
	case ast.JSW_UNCONDITIONAL:
		return "unconditional"
	case ast.JSW_NONE:
		return "none"
	}
	return ""
}

// jsonQuotesName returns the quotes behavior name of a JSON_TABLE column.
func jsonQuotesName(value int64) string {
	switch ast.JsonQuotes(value) {
	case ast.JS_QUOTES_KEEP:
		return "keep"
	case ast.JS_QUOTES_OMIT:
		return "omit"
	}
	return ""
}

// jsonBehaviorNames returns the empty and error behavior names of a JSON_TABLE column.
func jsonBehaviorNames(n ast.Node) (string, string) {
	if n == nil {
		return "", ""
	}
	list, ok := n.(*ast.List)
	if !ok || len(list.Items) != 2 {
		return "", ""
	}
	return jsonBehaviorName(list.Items[0]), jsonBehaviorName(list.Items[1])
}

// jsonOnErrorName returns the error behavior name of a JSON_TABLE call.
func jsonOnErrorName(n ast.Node) string {
	return jsonBehaviorName(n)
}

// jsonBehaviorName returns the name of a JSON behavior clause.
func jsonBehaviorName(n ast.Node) string {
	if n == nil {
		return ""
	}
	behavior, ok := n.(*ast.JsonBehavior)
	if !ok {
		return ""
	}
	switch behavior.Btype {
	case ast.JSON_BEHAVIOR_ERROR:
		return "error"
	case ast.JSON_BEHAVIOR_NULL:
		return "null"
	case ast.JSON_BEHAVIOR_DEFAULT:
		return "default"
	}
	return ""
}

// exprsOfList returns the expressions of an expression list.
func exprsOfList(l *ast.List) []ast.ExprNode {
	if l == nil {
		return nil
	}
	var result []ast.ExprNode
	for _, item := range l.Items {
		if expr, ok := item.(ast.ExprNode); ok {
			result = append(result, expr)
		}
	}
	return result
}

// exprStringValue returns the string value of an expression.
func exprStringValue(e ast.ExprNode) string {
	if e == nil {
		return ""
	}
	value, ok := e.(ast.ValueExpr)
	if !ok {
		return ""
	}
	switch v := value.GetValue().(type) {
	case string:
		return v
	case []byte:
		return string(v)
	}
	return ""
}

// rowsFromFunctions returns the functions of a ROWS FROM call.
func rowsFromFunctions(l *ast.List) []*ast.TableFunction {
	if l == nil {
		return nil
	}
	var result []*ast.TableFunction
	for _, item := range l.Items {
		expr, ok := item.(ast.ExprNode)
		if !ok {
			continue
		}
		result = append(result, &ast.TableFunction{Expr: expr})
	}
	return result
}

// tableFuncElementDefs returns the column definitions of a function result.
func tableFuncElementDefs(l *ast.List) []*ast.ColumnDef {
	if l == nil {
		return nil
	}
	var defs []*ast.ColumnDef
	for _, item := range l.Items {
		if def, ok := item.(*ast.ColumnDef); ok {
			defs = append(defs, def)
		}
	}
	return defs
}

// jsonAggConstructor returns the constructor of a JSON aggregate function.
func jsonAggConstructor(n ast.Node) *ast.JsonAggConstructor {
	switch v := n.(type) {
	case *ast.JsonObjectAgg:
		return v.Constructor
	case *ast.JsonArrayAgg:
		return v.Constructor
	}
	return nil
}
