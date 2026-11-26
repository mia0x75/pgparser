package ast

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mia0x75/parserkit/lex"

	"github.com/mia0x75/pgparser/errors"

	"github.com/mia0x75/pgparser/format"
	"github.com/mia0x75/pgparser/model"
	"github.com/mia0x75/pgparser/types"
)

var (
	_ DDLNode = &AlterTableStmt{}
	_ DDLNode = &AlterDatabaseStmt{}
	_ DDLNode = &CreateDatabaseStmt{}
	_ DDLNode = &CreateSchemaStmt{}
	_ DDLNode = &CreateIndexStmt{}
	_ DDLNode = &CreateTableStmt{}
	_ DDLNode = &CreateViewStmt{}
	_ DDLNode = &CreateFunctionStmt{}
	_ DDLNode = &AlterFunctionStmt{}
	_ DDLNode = &DropDatabaseStmt{}
	_ DDLNode = &DropIndexStmt{}
	_ DDLNode = &DropTableStmt{}
	_ DDLNode = &DropFunctionStmt{}
	_ DDLNode = &DropProcedureStmt{}
	_ DDLNode = &DropTriggerStmt{}
	_ DDLNode = &RenameTableStmt{}
	_ DDLNode = &TruncateTableStmt{}

	_ Node = &AlterTableSpec{}
	_ Node = &ColumnDef{}
	_ Node = &ColumnOption{}
	_ Node = &ColumnPosition{}
	_ Node = &Constraint{}
	_ Node = &IndexColName{}
	_ Node = &ReferenceDef{}
)

// CharsetOpt is used for parsing charset option from SQL.
type CharsetOpt struct {
	Chs string
	Col string
}

// DatabaseOptionType is the type for database options.
type DatabaseOptionType int

// Database option types.
const (
	DatabaseOptionNone DatabaseOptionType = iota
	DatabaseOptionOwner
	DatabaseOptionTemplate
	DatabaseOptionEncoding
	DatabaseOptionTablespace
	DatabaseOptionConnectionLimit
	DatabaseOptionLCCollate
	DatabaseOptionLCType
)

// DatabaseOption represents database option.
type DatabaseOption struct {
	Tp      DatabaseOptionType
	Name    string
	Value   string
	Default bool
}

func (n *DatabaseOption) validate() error {
	if n == nil {
		return errors.New("database has a nil option")
	}
	if n.Tp < DatabaseOptionNone || n.Tp > DatabaseOptionLCType {
		return errors.Errorf("invalid database option type: %d", n.Tp)
	}
	if n.Tp == DatabaseOptionNone && n.Name == "" {
		return errors.New("database option has no name")
	}
	if n.Default {
		if n.Value != "" {
			return errors.New("default database option also has a value")
		}
		return nil
	}
	if n.Tp == DatabaseOptionConnectionLimit {
		value, err := strconv.ParseInt(n.Value, 10, 32)
		if err != nil || value < -1 || strconv.FormatInt(value, 10) != n.Value {
			return errors.New("invalid database connection limit")
		}
	}
	return nil
}

// Restore implements Node interface.
func (n *DatabaseOption) Restore(ctx *format.RestoreCtx) error {
	if err := n.validate(); err != nil {
		return err
	}
	switch n.Tp {
	case DatabaseOptionLCCollate:
		ctx.WriteKeyword("LC_COLLATE")
	case DatabaseOptionLCType:
		ctx.WriteKeyword("LC_CTYPE")
	case DatabaseOptionOwner:
		ctx.WriteKeyword("OWNER")
	case DatabaseOptionTemplate:
		ctx.WriteKeyword("TEMPLATE")
	case DatabaseOptionEncoding:
		ctx.WriteKeyword("ENCODING")
	case DatabaseOptionTablespace:
		ctx.WriteKeyword("TABLESPACE")
	case DatabaseOptionConnectionLimit:
		ctx.WriteKeyword("CONNECTION LIMIT")
	case DatabaseOptionNone:
		ctx.WriteName(n.Name)
	}
	ctx.WritePlain(" = ")
	switch {
	case n.Default:
		ctx.WriteKeyword("DEFAULT")
	case n.Tp == DatabaseOptionConnectionLimit:
		ctx.WritePlain(n.Value)
	case n.Tp == DatabaseOptionOwner || n.Tp == DatabaseOptionTemplate || n.Tp == DatabaseOptionTablespace:
		ctx.WriteName(n.Value)
	default:
		ctx.WriteString(n.Value)
	}
	return ctx.Err()
}

// CreateSchemaStmt is a statement to create a schema.
// See https://www.postgresql.org/docs/current/sql-createschema.html
type CreateSchemaStmt struct {
	ddlNode

	Name          lex.Ident
	Authorization *RoleSpec
	IfNotExists   bool
}

// Restore implements Node interface.
func (n *CreateSchemaStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("CREATE SCHEMA ")
	if n.IfNotExists {
		ctx.WriteKeyword("IF NOT EXISTS ")
	}
	if n.Name.Present() {
		ctx.WriteIdent(n.Name)
	}
	if n.Authorization != nil {
		if n.Name.Present() {
			ctx.WritePlain(" ")
		}
		ctx.WriteKeyword("AUTHORIZATION ")
		return n.Authorization.Restore(ctx)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CreateSchemaStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*CreateSchemaStmt)
	if n.Authorization != nil {
		child, ok := n.Authorization.Accept(v)
		if !ok {
			return n, false
		}
		n.Authorization = child.(*RoleSpec)
	}
	return v.Leave(n)
}

// CreateDatabaseStmt is a statement to create a database.
type CreateDatabaseStmt struct {
	ddlNode

	IfNotExists bool
	Name        lex.Ident
	Options     []*DatabaseOption
}

// Restore implements Node interface.
func (n *CreateDatabaseStmt) Restore(ctx *format.RestoreCtx) error {
	if !n.Name.Present() {
		return errors.New("database has no name")
	}
	for i, option := range n.Options {
		if err := option.validate(); err != nil {
			return errors.Annotatef(err, "An error occurred while splicing CreateDatabaseStmt DatabaseOption: [%v]", i)
		}
	}
	ctx.WriteKeyword("CREATE DATABASE ")
	if n.IfNotExists {
		ctx.WriteKeyword("IF NOT EXISTS ")
	}
	ctx.WriteIdent(n.Name)
	for _, option := range n.Options {
		ctx.WritePlain(" ")
		if err := option.Restore(ctx); err != nil {
			return err
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CreateDatabaseStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*CreateDatabaseStmt)
	return v.Leave(n)
}

// AlterDatabaseStmt is a statement to alter a database.
type AlterDatabaseStmt struct {
	ddlNode

	Name                 lex.Ident
	AlterDefaultDatabase bool
	Options              []*DatabaseOption
	// RenameTo is the new database name of a rename statement.
	RenameTo   lex.Ident
	Tablespace lex.Ident
}

// Restore implements Node interface.
func (n *AlterDatabaseStmt) Restore(ctx *format.RestoreCtx) error {
	if !n.Name.Present() {
		return errors.New("database has no name")
	}
	if (n.RenameTo.Present() || n.Tablespace.Present()) && len(n.Options) > 0 {
		return errors.New("ALTER DATABASE has conflicting actions")
	}
	if n.RenameTo.Present() && n.Tablespace.Present() {
		return errors.New("ALTER DATABASE has conflicting actions")
	}
	for i, option := range n.Options {
		if err := option.validate(); err != nil {
			return errors.Annotatef(err, "An error occurred while splicing AlterDatabaseStmt DatabaseOption: [%v]", i)
		}
	}
	ctx.WriteKeyword("ALTER DATABASE ")
	ctx.WriteIdent(n.Name)
	for _, option := range n.Options {
		ctx.WritePlain(" ")
		if err := option.Restore(ctx); err != nil {
			return err
		}
	}
	if n.RenameTo.Present() {
		ctx.WriteKeyword(" RENAME TO ")
		ctx.WriteIdent(n.RenameTo)
	}
	if n.Tablespace.Present() {
		ctx.WriteKeyword(" SET TABLESPACE ")
		ctx.WriteIdent(n.Tablespace)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *AlterDatabaseStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*AlterDatabaseStmt)
	return v.Leave(n)
}

// CommentStmt is a PostgreSQL `COMMENT ON ... IS ...` statement.
type CommentStmt struct {
	ddlNode

	// ObjectType is the lowercase object kind, e.g. "table" or "column".
	ObjectType string
	// ObjectName holds the name parts of the commented object.
	ObjectName []string
	// ParentName holds the relation name for constraint, policy, rule and
	// trigger comments.
	ParentName []string
	// ExtraName holds the access method or language of operator class,
	// operator family and transform comments.
	ExtraName []string
	// Suffix holds the restored tail of the reference: an argument signature
	// for routines, a type for transforms and casts.
	Suffix string
	// Comment is nil when the comment is removed with `IS NULL`.
	Comment *string
}

// Restore implements Node interface.
func (n *CommentStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("COMMENT ON ")
	// Constraint, policy, rule and trigger comments write the object name
	// before the parent relation, so they cannot use the plain form.
	switch n.ObjectType {
	case "tabconstraint":
		ctx.WriteKeyword("CONSTRAINT ")
		writeNameParts(ctx, n.ObjectName)
		ctx.WriteKeyword(" ON ")
		writeNameParts(ctx, n.ParentName)
	case "domconstraint":
		ctx.WriteKeyword("CONSTRAINT ")
		writeNameParts(ctx, n.ObjectName)
		ctx.WriteKeyword(" ON DOMAIN ")
		writeNameParts(ctx, n.ParentName)
	case "policy", "rule", "trigger":
		ctx.WriteKeyword(strings.ToUpper(n.ObjectType) + " ")
		writeNameParts(ctx, n.ObjectName)
		ctx.WriteKeyword(" ON ")
		writeNameParts(ctx, n.ParentName)
	case "operator class", "operator family":
		ctx.WriteKeyword(strings.ToUpper(n.ObjectType) + " ")
		writeNameParts(ctx, n.ObjectName)
		ctx.WriteKeyword(" USING ")
		writeNameParts(ctx, n.ExtraName)
	case "transform":
		ctx.WriteKeyword("TRANSFORM FOR ")
		ctx.WritePlain(n.Suffix)
		ctx.WriteKeyword(" LANGUAGE ")
		writeNameParts(ctx, n.ExtraName)
	case "cast":
		ctx.WriteKeyword("CAST (")
		ctx.WritePlain(n.Suffix)
		ctx.WritePlain(")")
	case "large object":
		ctx.WriteKeyword("LARGE OBJECT ")
		ctx.WritePlain(strings.Join(n.ObjectName, "."))
	case "operator":
		if len(n.ObjectName) == 0 {
			return errors.New("operator comment has no name")
		}
		ctx.WriteKeyword("OPERATOR ")
		// A schema qualifies the operator; the operator itself is a symbol and
		// must stay unquoted, so it cannot go through a name writer.
		last := len(n.ObjectName) - 1
		if last > 0 {
			writeNameParts(ctx, n.ObjectName[:last])
			ctx.WritePlain(".")
		}
		ctx.WritePlain(n.ObjectName[last])
		ctx.WritePlain(n.Suffix)
	default:
		ctx.WriteKeyword(strings.ToUpper(n.ObjectType) + " ")
		writeNameParts(ctx, n.ObjectName)
		ctx.WritePlain(n.Suffix)
	}
	ctx.WriteKeyword(" IS ")
	if n.Comment == nil {
		ctx.WriteKeyword("NULL")
	} else {
		ctx.WriteString(*n.Comment)
	}
	return ctx.Err()
}

// writeNameParts writes a dotted name, quoting every part on its own.
func writeNameParts(ctx *format.RestoreCtx, parts []string) {
	for i, part := range parts {
		if i > 0 {
			ctx.WritePlain(".")
		}
		ctx.WriteName(part)
	}
}

// Accept implements Node Accept interface.
func (n *CommentStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	return v.Leave(newNode)
}

type DropDatabaseStmt struct {
	ddlNode

	IfExists bool
	Name     string
}

// Restore implements Node interface.
func (n *DropDatabaseStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("DROP DATABASE ")
	if n.IfExists {
		ctx.WriteKeyword("IF EXISTS ")
	}
	ctx.WriteName(n.Name)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DropDatabaseStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*DropDatabaseStmt)
	return v.Leave(n)
}

// IndexColName is used for parsing index column name from SQL.
type IndexColName struct {
	node

	Column *ColumnName
	// Expr is the expression of an expression-based index element.
	Expr           ExprNode
	Length         int
	Desc           bool
	NullsOrder     int
	Opclass        *List
	OpclassOptions *List
	Collation      *List
}

// Restore implements Node interface.
func (n *IndexColName) Restore(ctx *format.RestoreCtx) error {
	if n.Column != nil && n.Expr != nil {
		return errors.New("index element has both a column and an expression")
	}
	switch {
	case n.Column != nil && n.Column.Name.O != "":
		if err := n.Column.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing IndexColName")
		}
	case n.Expr != nil:
		_, function := n.Expr.(*FuncCallExpr)
		if !function {
			ctx.WritePlain("(")
		}
		if err := n.Expr.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing IndexColName.Expr")
		}
		if !function {
			ctx.WritePlain(")")
		}
	default:
		return errors.New("IndexColName has neither Column nor Expr")
	}
	if n.Length > 0 {
		ctx.WritePlainf("(%d)", n.Length)
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
		if n.OpclassOptions != nil {
			if err := restoreRelOptions(ctx, n.OpclassOptions); err != nil {
				return err
			}
		}
	}
	if n.Desc {
		ctx.WriteKeyword(" DESC")
	}
	switch n.NullsOrder {
	case 1:
		ctx.WriteKeyword(" NULLS FIRST")
	case 2:
		ctx.WriteKeyword(" NULLS LAST")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *IndexColName) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*IndexColName)
	if n.Column != nil {
		node, ok := n.Column.Accept(v)
		if !ok {
			return n, false
		}
		n.Column = node.(*ColumnName)
	}
	if n.Expr != nil {
		node, ok := n.Expr.Accept(v)
		if !ok {
			return n, false
		}
		n.Expr = node.(ExprNode)
	}
	for _, list := range []**List{&n.Collation, &n.Opclass, &n.OpclassOptions} {
		if *list != nil {
			child, ok := (*list).Accept(v)
			if !ok {
				return n, false
			}
			*list = child.(*List)
		}
	}
	return v.Leave(n)
}

// ReferenceDef is used for parsing foreign key reference option from SQL.
type ReferenceDef struct {
	node

	Table         *TableName
	IndexColNames []*IndexColName
	OnDelete      *OnDeleteOpt
	OnUpdate      *OnUpdateOpt
	Match         byte
}

// Restore implements Node interface.
func (n *ReferenceDef) Restore(ctx *format.RestoreCtx) error {
	if n.Table == nil {
		return errors.New("foreign key has no referenced table")
	}
	ctx.WriteKeyword("REFERENCES ")
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "foreign key table")
	}
	if len(n.IndexColNames) > 0 {
		ctx.WritePlain("(")
		for i, col := range n.IndexColNames {
			if col == nil {
				return errors.New("foreign key has a nil referenced column")
			}
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if err := col.Restore(ctx); err != nil {
				return errors.Annotatef(err, "foreign key column %d", i)
			}
		}
		ctx.WritePlain(")")
	}
	switch n.Match {
	case 0, 's':
	case 'f':
		ctx.WriteKeyword(" MATCH FULL")
	case 'p':
		ctx.WriteKeyword(" MATCH PARTIAL")
	default:
		return errors.New("invalid foreign key match mode")
	}
	if n.OnDelete != nil && n.OnDelete.ReferOpt != ReferOptionNoOption {
		ctx.WritePlain(" ")
		if err := n.OnDelete.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing OnDelete")
		}
	}
	if n.OnUpdate != nil && n.OnUpdate.ReferOpt != ReferOptionNoOption {
		ctx.WritePlain(" ")
		if err := n.OnUpdate.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing OnUpdate")
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ReferenceDef) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*ReferenceDef)
	var node Node
	var ok bool
	if n.Table != nil {
		node, ok = n.Table.Accept(v)
		if !ok {
			return n, false
		}
		n.Table = node.(*TableName)
	}
	for i, val := range n.IndexColNames {
		if val == nil {
			continue
		}
		node, ok = val.Accept(v)
		if !ok {
			return n, false
		}
		n.IndexColNames[i] = node.(*IndexColName)
	}
	if n.OnDelete != nil {
		node, ok = n.OnDelete.Accept(v)
		if !ok {
			return n, false
		}
		n.OnDelete = node.(*OnDeleteOpt)
	}
	if n.OnUpdate != nil {
		node, ok = n.OnUpdate.Accept(v)
		if !ok {
			return n, false
		}
		n.OnUpdate = node.(*OnUpdateOpt)
	}
	return v.Leave(n)
}

// ReferOptionType is the type for refer options.
type ReferOptionType int

// Refer option types.
const (
	ReferOptionNoOption ReferOptionType = iota
	ReferOptionRestrict
	ReferOptionCascade
	ReferOptionSetNull
	ReferOptionNoAction
	ReferOptionSetDefault
)

// String implements fmt.Stringer interface.
func (r ReferOptionType) String() string {
	switch r {
	case ReferOptionRestrict:
		return "RESTRICT"
	case ReferOptionCascade:
		return "CASCADE"
	case ReferOptionSetNull:
		return "SET NULL"
	case ReferOptionNoAction:
		return "NO ACTION"
	case ReferOptionSetDefault:
		return "SET DEFAULT"
	}
	return ""
}

// OnDeleteOpt is used for optional on delete clause.
type OnDeleteOpt struct {
	node
	ReferOpt ReferOptionType
	Columns  []*ColumnName
}

// Restore implements Node interface.
func (n *OnDeleteOpt) Restore(ctx *format.RestoreCtx) error {
	if n.ReferOpt != ReferOptionNoOption && n.ReferOpt.String() == "" {
		return errors.New("invalid ON DELETE action")
	}
	if len(n.Columns) > 0 && n.ReferOpt != ReferOptionSetNull && n.ReferOpt != ReferOptionSetDefault {
		return errors.New("ON DELETE columns require SET NULL or SET DEFAULT")
	}
	if n.ReferOpt != ReferOptionNoOption {
		ctx.WriteKeyword("ON DELETE ")
		ctx.WriteKeyword(n.ReferOpt.String())
	}
	if len(n.Columns) > 0 {
		ctx.WritePlain(" (")
		for i, column := range n.Columns {
			if column == nil {
				return errors.New("ON DELETE has a nil column")
			}
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if err := column.Restore(ctx); err != nil {
				return err
			}
		}
		ctx.WritePlain(")")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *OnDeleteOpt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*OnDeleteOpt)
	for i, column := range n.Columns {
		if column == nil {
			continue
		}
		child, ok := column.Accept(v)
		if !ok {
			return n, false
		}
		n.Columns[i] = child.(*ColumnName)
	}
	return v.Leave(n)
}

// OnUpdateOpt is used for optional on update clause.
type OnUpdateOpt struct {
	node
	ReferOpt ReferOptionType
}

// Restore implements Node interface.
func (n *OnUpdateOpt) Restore(ctx *format.RestoreCtx) error {
	if n.ReferOpt != ReferOptionNoOption && n.ReferOpt.String() == "" {
		return errors.New("invalid ON UPDATE action")
	}
	if n.ReferOpt != ReferOptionNoOption {
		ctx.WriteKeyword("ON UPDATE ")
		ctx.WriteKeyword(n.ReferOpt.String())
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *OnUpdateOpt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*OnUpdateOpt)
	return v.Leave(n)
}

// ColumnOptionType is the type for ColumnOption.
type ColumnOptionType int

// ColumnOption types.
const (
	ColumnOptionNoOption ColumnOptionType = iota
	ColumnOptionPrimaryKey
	ColumnOptionNotNull
	ColumnOptionIdentity
	ColumnOptionDefaultValue
	ColumnOptionUniqKey
	ColumnOptionNull
	ColumnOptionOnUpdate // For Timestamp and Datetime only.
	ColumnOptionFulltext
	ColumnOptionComment
	ColumnOptionGenerated
	ColumnOptionReference
	ColumnOptionCollate
	ColumnOptionCheck
	// ColumnOptionStorage is the `STORAGE` option of a column.
	ColumnOptionStorage
	// ColumnOptionCompression is the `COMPRESSION` option of a column.
	ColumnOptionCompression
	ColumnOptionVisible
	ColumnOptionInvisible
	ColumnOptionForeignOptions
)

var (
	invalidOptionForGeneratedColumn = map[ColumnOptionType]struct{}{
		ColumnOptionIdentity:     {},
		ColumnOptionOnUpdate:     {},
		ColumnOptionDefaultValue: {},
	}
)

// ColumnOption is used for parsing column constraint info from SQL.
type ColumnOption struct {
	node

	Tp ColumnOptionType
	// Expr is used for ColumnOptionDefaultValue/ColumnOptionOnUpdateColumnOptionGenerated.
	// For ColumnOptionDefaultValue or ColumnOptionOnUpdate, it's the target value.
	// For ColumnOptionGenerated, it's the target expression.
	Expr ExprNode
	// Stored is only for ColumnOptionGenerated, default is false.
	Stored bool
	// Refer is used for foreign key.
	Refer    *ReferenceDef
	StrValue string
	// NullsNotDistinct is true for `UNIQUE NULLS NOT DISTINCT` columns.
	NullsNotDistinct bool
	// GeneratedWhen is 'a' for ALWAYS or 'd' for BY DEFAULT, for ColumnOptionIdentity.
	GeneratedWhen byte
	// SequenceOptions holds the parenthesized identity sequence options.
	SequenceOptions *List
	Options         *List
}

// Restore implements Node interface.
func (n *ColumnOption) Restore(ctx *format.RestoreCtx) error {
	switch n.Tp {
	case ColumnOptionNoOption:
		// NoOption still carries column constraint attributes.
		switch n.StrValue {
		case "deferrable":
			ctx.WriteKeyword("DEFERRABLE")
		case "not deferrable":
			ctx.WriteKeyword("NOT DEFERRABLE")
		case "initially deferred":
			ctx.WriteKeyword("INITIALLY DEFERRED")
		case "initially immediate":
			ctx.WriteKeyword("INITIALLY IMMEDIATE")
		default:
			return ctx.Err()
		}
	case ColumnOptionPrimaryKey:
		ctx.WriteKeyword("PRIMARY KEY")
	case ColumnOptionNotNull:
		ctx.WriteKeyword("NOT NULL")
	case ColumnOptionIdentity:
		ctx.WriteKeyword("GENERATED ")
		if n.GeneratedWhen == byte('d') {
			ctx.WriteKeyword("BY DEFAULT ")
		} else {
			ctx.WriteKeyword("ALWAYS ")
		}
		ctx.WriteKeyword("AS IDENTITY")
		if n.SequenceOptions != nil {
			var opts strings.Builder
			if err := restoreSeqOptions(format.NewRestoreCtx(format.DefaultRestoreFlags, &opts), n.SequenceOptions); err != nil {
				return errors.Annotate(err, "An error occurred while splicing ColumnOption Identity options")
			}
			ctx.WritePlain(" (")
			ctx.WritePlain(strings.TrimSpace(opts.String()))
			ctx.WritePlain(")")
		}
	case ColumnOptionDefaultValue:
		ctx.WriteKeyword("DEFAULT ")
		if err := n.Expr.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing ColumnOption DefaultValue Expr")
		}
	case ColumnOptionUniqKey:
		// PostgreSQL column constraints write plain UNIQUE.
		ctx.WriteKeyword("UNIQUE")
		if n.NullsNotDistinct {
			ctx.WriteKeyword(" NULLS NOT DISTINCT")
		}
	case ColumnOptionNull:
		ctx.WriteKeyword("NULL")
	case ColumnOptionOnUpdate:
		ctx.WriteKeyword("ON UPDATE ")
		if err := n.Expr.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing ColumnOption ON UPDATE Expr")
		}
	case ColumnOptionFulltext:
		return errors.New("Parser ignore the `ColumnOptionFulltext` type now")
	case ColumnOptionComment:
		ctx.WriteKeyword("COMMENT ")
		if err := n.Expr.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing ColumnOption COMMENT Expr")
		}
	case ColumnOptionGenerated:
		ctx.WriteKeyword("GENERATED ALWAYS AS")
		ctx.WritePlain("(")
		if err := n.Expr.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing ColumnOption GENERATED ALWAYS Expr")
		}
		ctx.WritePlain(")")
		if n.Stored {
			ctx.WriteKeyword(" STORED")
		} else {
			ctx.WriteKeyword(" VIRTUAL")
		}
	case ColumnOptionReference:
		if err := n.Refer.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing ColumnOption ReferenceDef")
		}
	case ColumnOptionCollate:
		if n.StrValue == "" {
			return errors.New("Empty ColumnOption COLLATE")
		}
		ctx.WriteKeyword("COLLATE ")
		ctx.WriteName(n.StrValue)
	case ColumnOptionStorage:
		if n.StrValue == "" {
			return errors.New("Empty ColumnOption STORAGE")
		}
		ctx.WriteKeyword("STORAGE ")
		ctx.WriteName(n.StrValue)
	case ColumnOptionCompression:
		if n.StrValue == "" {
			return errors.New("Empty ColumnOption COMPRESSION")
		}
		ctx.WriteKeyword("COMPRESSION ")
		ctx.WriteName(n.StrValue)
	case ColumnOptionCheck:
		ctx.WriteKeyword("CHECK (")
		if err := n.Expr.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing ColumnOption CHECK Expr")
		}
		ctx.WritePlain(")")
	case ColumnOptionVisible:
		ctx.WriteKeyword("VISIBLE")
	case ColumnOptionInvisible:
		ctx.WriteKeyword("INVISIBLE")
	case ColumnOptionForeignOptions:
		return restoreAlterOptions(ctx, n.Options)
	default:
		return errors.New("An error occurred while splicing ColumnOption")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ColumnOption) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*ColumnOption)
	if n.Expr != nil {
		node, ok := n.Expr.Accept(v)
		if !ok {
			return n, false
		}
		n.Expr = node.(ExprNode)
	}
	if n.Refer != nil {
		child, ok := n.Refer.Accept(v)
		if !ok {
			return n, false
		}
		n.Refer = child.(*ReferenceDef)
	}
	if n.Options != nil {
		child, ok := n.Options.Accept(v)
		if !ok {
			return n, false
		}
		n.Options = child.(*List)
	}
	return v.Leave(n)
}

// IndexOption is the index options.
//
//	  KEY_BLOCK_SIZE [=] value
//	| index_type
//	| WITH PARSER parser_name
//	| COMMENT 'string'
type IndexOption struct {
	node

	KeyBlockSize uint64
	Tp           model.IndexType
	Comment      string
}

// Restore implements Node interface.
func (n *IndexOption) Restore(ctx *format.RestoreCtx) error {
	hasPrevOption := false
	if n.KeyBlockSize > 0 {
		ctx.WriteKeyword("KEY_BLOCK_SIZE")
		ctx.WritePlainf("=%d", n.KeyBlockSize)
		hasPrevOption = true
	}

	if n.Tp != model.IndexTypeInvalid {
		if hasPrevOption {
			ctx.WritePlain(" ")
		}
		ctx.WriteKeyword("USING ")
		ctx.WritePlain(n.Tp.String())
		hasPrevOption = true
	}

	if n.Comment != "" {
		if hasPrevOption {
			ctx.WritePlain(" ")
		}
		ctx.WriteKeyword("COMMENT ")
		ctx.WriteString(n.Comment)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *IndexOption) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*IndexOption)
	return v.Leave(n)
}

// ConstraintType is the type for Constraint.
type ConstraintType int

// ConstraintTypes
const (
	ConstraintNoConstraint ConstraintType = iota
	ConstraintPrimaryKey
	ConstraintKey
	ConstraintIndex
	ConstraintUniq
	ConstraintUniqKey
	ConstraintUniqIndex
	ConstraintForeignKey
	ConstraintFulltext
	ConstraintCheck
)

// Constraint is constraint for table definition.
type Constraint struct {
	node

	Tp   ConstraintType
	Name lex.Ident

	Keys []*IndexColName // Used for PRIMARY KEY, UNIQUE, ......

	Refer *ReferenceDef // Used for foreign key.

	Expr ExprNode // Used for CHECK constraint.

	Option *IndexOption // Index Options

	// Deferrable is true for `DEFERRABLE` constraints.
	Deferrable bool
	// InitDeferred is true for `INITIALLY DEFERRED` constraints.
	InitDeferred bool
	// SkipValidation is true for `NOT VALID` constraints.
	SkipValidation bool
	// NoInherit is true for `NO INHERIT` check constraints.
	NoInherit bool
	// InitiallyValid is false for `NOT VALID` constraints.
	InitiallyValid bool
	// NullsNotDistinct is true for `UNIQUE NULLS NOT DISTINCT` constraints.
	NullsNotDistinct bool
	// IndexName is the name of an existing index of the constraint.
	IndexName lex.Ident
	// IndexSpace is the tablespace of the constraint.
	IndexSpace lex.Ident
	// Including holds the columns of the `INCLUDE` clause.
	Including []lex.Ident
	// Options holds the definition options of the constraint.
	Options []string
	// AccessMethod is the access method of an exclusion constraint.
	AccessMethod string
	// WhereClause is the predicate of a partial constraint.
	WhereClause ExprNode
	// GeneratedWhen is the generation mode of a generated column.
	GeneratedWhen byte
}

// Restore implements Node interface.
func (n *Constraint) Restore(ctx *format.RestoreCtx) error {
	if n.Tp != ConstraintCheck && !n.IndexName.Present() && len(n.Keys) == 0 {
		return errors.Annotate(ErrUnsupportedRestore, "constraint without modeled keys")
	}
	if n.Option != nil || n.AccessMethod != "" || n.WhereClause != nil || len(n.Options) > 0 {
		return errors.Annotate(ErrUnsupportedRestore, "constraint options")
	}
	if n.Name.Present() {
		ctx.WriteKeyword("CONSTRAINT ")
		ctx.WriteIdent(n.Name)
		ctx.WritePlain(" ")
	}
	switch n.Tp {
	case ConstraintPrimaryKey:
		ctx.WriteKeyword("PRIMARY KEY")
	case ConstraintUniq:
		ctx.WriteKeyword("UNIQUE")
		if n.NullsNotDistinct {
			ctx.WriteKeyword(" NULLS NOT DISTINCT")
		}
	case ConstraintForeignKey:
		ctx.WriteKeyword("FOREIGN KEY")
	case ConstraintCheck:
		ctx.WriteKeyword("CHECK")
	default:
		return errors.Annotate(ErrUnsupportedRestore, "constraint type")
	}
	if n.IndexName.Present() {
		ctx.WriteKeyword(" USING INDEX ")
		ctx.WriteIdent(n.IndexName)
	} else {
		ctx.WritePlain(" (")
		if n.Tp == ConstraintCheck {
			if n.Expr == nil {
				return errors.New("CHECK has no expression")
			}
			if err := n.Expr.Restore(ctx); err != nil {
				return err
			}
		} else {
			for i, key := range n.Keys {
				if key == nil {
					return errors.New("constraint has a nil key")
				}
				if i > 0 {
					ctx.WritePlain(", ")
				}
				if err := key.Restore(ctx); err != nil {
					return err
				}
			}
		}
		ctx.WritePlain(")")
	}
	if n.Refer != nil {
		ctx.WritePlain(" ")
		if err := n.Refer.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing Constraint Refer")
		}
	}

	if len(n.Including) > 0 {
		ctx.WriteKeyword(" INCLUDE (")
		for i, name := range n.Including {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			ctx.WriteIdent(name)
		}
		ctx.WritePlain(")")
	}
	if n.IndexSpace.Present() {
		ctx.WriteKeyword(" USING INDEX TABLESPACE ")
		ctx.WriteIdent(n.IndexSpace)
	}
	if n.Deferrable {
		ctx.WriteKeyword(" DEFERRABLE")
	}
	if n.InitDeferred {
		ctx.WriteKeyword(" INITIALLY DEFERRED")
	}
	if n.SkipValidation {
		ctx.WriteKeyword(" NOT VALID")
	}
	if n.NoInherit {
		ctx.WriteKeyword(" NO INHERIT")
	}

	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *Constraint) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*Constraint)
	for i, val := range n.Keys {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Keys[i] = node.(*IndexColName)
	}
	if n.Expr != nil {
		node, ok := n.Expr.Accept(v)
		if !ok {
			return n, false
		}
		n.Expr = node.(ExprNode)
	}
	if n.Refer != nil {
		node, ok := n.Refer.Accept(v)
		if !ok {
			return n, false
		}
		n.Refer = node.(*ReferenceDef)
	}
	if n.Option != nil {
		node, ok := n.Option.Accept(v)
		if !ok {
			return n, false
		}
		n.Option = node.(*IndexOption)
	}
	return v.Leave(n)
}

// ColumnDef is used for parsing column definition from SQL.
type ColumnDef struct {
	node

	Name    *ColumnName
	Tp      *types.FieldType
	Options []*ColumnOption
}

// Restore implements Node interface.
func (n *ColumnDef) Restore(ctx *format.RestoreCtx) error {
	if err := n.Name.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while splicing ColumnDef Name")
	}
	if n.Tp != nil {
		ctx.WritePlain(" ")
		if err := n.Tp.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing ColumnDef Type")
		}
	}
	for i, options := range n.Options {
		ctx.WritePlain(" ")
		if err := options.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while splicing ColumnDef ColumnOption: [%v]", i)
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ColumnDef) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*ColumnDef)
	node, ok := n.Name.Accept(v)
	if !ok {
		return n, false
	}
	n.Name = node.(*ColumnName)
	for i, val := range n.Options {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Options[i] = node.(*ColumnOption)
	}
	return v.Leave(n)
}

// Validate checks if a column definition is legal.
// For example, generated column definitions that contain such
// column options as `ON UPDATE`, `AUTO_INCREMENT`, `DEFAULT`
// are illegal.
func (n *ColumnDef) Validate() bool {
	generatedCol := false
	illegalOpt4gc := false
	for _, opt := range n.Options {
		if opt.Tp == ColumnOptionGenerated {
			generatedCol = true
		}
		_, found := invalidOptionForGeneratedColumn[opt.Tp]
		illegalOpt4gc = illegalOpt4gc || found
	}
	return !generatedCol || !illegalOpt4gc
}

// CreateTableStmt is a statement to create a table.
type CreateTableStmt struct {
	ddlNode

	IfNotExists bool
	Temporary   bool
	OrReplace   bool
	Table       *TableName
	ReferTable  *TableName
	Cols        []*ColumnDef
	Constraints []*Constraint
	Options     []*TableOption
	Partition   *PartitionOptions
	// PartitionBound is the bound specification of a partition table.
	PartitionBound *PartitionBoundSpec
	WithOptions    *List
	Tablespace     lex.Ident
	Inherits       []*TableName
	PartitionBy    *PartitionSpec
	LikeClauses    []*TableLikeClause
	AccessMethod   string
	OnCommit       OnCommitAction
	OnDuplicate    OnDuplicateKeyHandlingType
	Select         ResultSetNode
	// Execute is the prepared statement of a CREATE TABLE AS EXECUTE statement.
	Execute string
}

// Restore implements Node interface.
func (n *CreateTableStmt) Restore(ctx *format.RestoreCtx) error {
	if len(n.LikeClauses) > 0 || n.Execute != "" {
		return errors.Annotate(ErrUnsupportedRestore, "CREATE TABLE LIKE or EXECUTE")
	}
	if n.Table == nil {
		return errors.New("CREATE TABLE has no name")
	}
	ctx.WriteKeyword("CREATE ")
	if n.OrReplace {
		ctx.WriteKeyword("OR REPLACE ")
	}
	if n.Table.Relpersistence == 'u' {
		ctx.WriteKeyword("UNLOGGED ")
	} else if n.Temporary {
		ctx.WriteKeyword("TEMPORARY ")
	}
	ctx.WriteKeyword("TABLE ")
	if n.IfNotExists {
		ctx.WriteKeyword("IF NOT EXISTS ")
	}
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while splicing CreateTableStmt Table")
	}
	ctx.WritePlain(" ")
	if n.ReferTable != nil {
		if n.PartitionBound != nil {
			ctx.WriteKeyword("PARTITION OF ")
		} else {
			ctx.WriteKeyword("OF ")
		}
		if err := n.ReferTable.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing CreateTableStmt ReferTable")
		}
	}

	lenCols := len(n.Cols)
	lenConstraints := len(n.Constraints)
	if lenCols+lenConstraints > 0 || n.ReferTable == nil && n.Select == nil {
		ctx.WritePlain("(")
		for i, col := range n.Cols {
			if i > 0 {
				ctx.WritePlain(",")
			}
			if err := col.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while splicing CreateTableStmt ColumnDef: [%v]", i)
			}
		}
		for i, constraint := range n.Constraints {
			if i > 0 || lenCols >= 1 {
				ctx.WritePlain(",")
			}
			if err := constraint.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while splicing CreateTableStmt Constraints: [%v]", i)
			}
		}
		ctx.WritePlain(")")
	}
	if n.PartitionBound != nil {
		ctx.WritePlain(" ")
		if err := n.PartitionBound.Restore(ctx); err != nil {
			return err
		}
	}

	for i, option := range n.Options {
		ctx.WritePlain(" ")
		if err := option.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while splicing CreateTableStmt TableOption: [%v]", i)
		}
	}

	if n.Partition != nil {
		ctx.WritePlain(" ")
		if err := n.Partition.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing CreateTableStmt Partition")
		}
	}
	if len(n.Inherits) > 0 {
		ctx.WriteKeyword(" INHERITS (")
		for i, table := range n.Inherits {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if err := table.Restore(ctx); err != nil {
				return err
			}
		}
		ctx.WritePlain(")")
	}
	if n.PartitionBy != nil {
		ctx.WriteKeyword(" PARTITION BY ")
		if err := n.PartitionBy.Restore(ctx); err != nil {
			return err
		}
	}
	if n.AccessMethod != "" {
		ctx.WriteKeyword(" USING ")
		ctx.WriteName(n.AccessMethod)
	}
	if n.WithOptions != nil {
		ctx.WriteKeyword(" WITH ")
		if err := restoreRelOptions(ctx, n.WithOptions); err != nil {
			return err
		}
	}
	switch n.OnCommit {
	case ONCOMMIT_PRESERVE_ROWS:
		ctx.WriteKeyword(" ON COMMIT PRESERVE ROWS")
	case ONCOMMIT_DELETE_ROWS:
		ctx.WriteKeyword(" ON COMMIT DELETE ROWS")
	case ONCOMMIT_DROP:
		ctx.WriteKeyword(" ON COMMIT DROP")
	}
	if n.Tablespace.Present() {
		ctx.WriteKeyword(" TABLESPACE ")
		ctx.WriteIdent(n.Tablespace)
	}

	if n.Select != nil {
		switch n.OnDuplicate {
		case OnDuplicateKeyHandlingError:
			ctx.WriteKeyword(" AS ")
		case OnDuplicateKeyHandlingIgnore:
			ctx.WriteKeyword(" IGNORE AS ")
		case OnDuplicateKeyHandlingReplace:
			ctx.WriteKeyword(" REPLACE AS ")
		}

		if err := n.Select.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing CreateTableStmt Select")
		}
	}

	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CreateTableStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*CreateTableStmt)
	node, ok := n.Table.Accept(v)
	if !ok {
		return n, false
	}
	n.Table = node.(*TableName)
	if n.ReferTable != nil {
		node, ok = n.ReferTable.Accept(v)
		if !ok {
			return n, false
		}
		n.ReferTable = node.(*TableName)
	}
	for i, val := range n.Cols {
		node, ok = val.Accept(v)
		if !ok {
			return n, false
		}
		n.Cols[i] = node.(*ColumnDef)
	}
	for i, val := range n.Constraints {
		node, ok = val.Accept(v)
		if !ok {
			return n, false
		}
		n.Constraints[i] = node.(*Constraint)
	}
	if n.Select != nil {
		node, ok := n.Select.Accept(v)
		if !ok {
			return n, false
		}
		n.Select = node.(ResultSetNode)
	}
	if n.WithOptions != nil {
		child, ok := n.WithOptions.Accept(v)
		if !ok {
			return n, false
		}
		n.WithOptions = child.(*List)
	}
	if n.PartitionBy != nil {
		child, ok := n.PartitionBy.Accept(v)
		if !ok {
			return n, false
		}
		n.PartitionBy = child.(*PartitionSpec)
	}
	if n.PartitionBound != nil {
		child, ok := n.PartitionBound.Accept(v)
		if !ok {
			return n, false
		}
		n.PartitionBound = child.(*PartitionBoundSpec)
	}
	for i, table := range n.Inherits {
		child, ok := table.Accept(v)
		if !ok {
			return n, false
		}
		n.Inherits[i] = child.(*TableName)
	}
	for i, like := range n.LikeClauses {
		child, ok := like.Accept(v)
		if !ok {
			return n, false
		}
		n.LikeClauses[i] = child.(*TableLikeClause)
	}

	return v.Leave(n)
}

// DropTableStmt is a statement to drop one or more tables.
type DropTableStmt struct {
	ddlNode

	IfExists bool
	Tables   []*TableName
	IsView   bool
}

// Restore implements Node interface.
func (n *DropTableStmt) Restore(ctx *format.RestoreCtx) error {
	if n.IsView {
		ctx.WriteKeyword("DROP VIEW ")
	} else {
		ctx.WriteKeyword("DROP TABLE ")
	}
	if n.IfExists {
		ctx.WriteKeyword("IF EXISTS ")
	}

	for index, table := range n.Tables {
		if index != 0 {
			ctx.WritePlain(", ")
		}
		if err := table.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore DropTableStmt.Tables "+strconv.Itoa(index))
		}
	}

	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DropTableStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*DropTableStmt)
	for i, val := range n.Tables {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Tables[i] = node.(*TableName)
	}
	return v.Leave(n)
}

// DropFunctionStmt is a statement to drop a stored function.
type DropFunctionStmt struct {
	ddlNode

	IfExists bool
	Name     *TableName
	// Names holds all dropped functions, the first one is Name.
	Names []*TableName
	// Nil entries keep omitted signatures distinct from explicit empty parentheses.
	Arguments []*List
	Behavior  DropBehavior
}

// Restore implements Node interface.
func (n *DropFunctionStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("DROP FUNCTION ")
	if n.IfExists {
		ctx.WriteKeyword("IF EXISTS ")
	}
	return restoreRoutineTargets(ctx, n.droppedNames(), n.Arguments, n.Behavior)
}

// droppedNames returns all dropped function names.
func (n *DropFunctionStmt) droppedNames() []*TableName {
	if len(n.Names) > 0 {
		return n.Names
	}
	if n.Name != nil {
		return []*TableName{n.Name}
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *DropFunctionStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*DropFunctionStmt)
	names, ok := acceptRoutineTargets(v, n.droppedNames(), n.Arguments)
	n.Names = names
	if len(names) > 0 {
		n.Name = names[0]
	}
	if !ok {
		return n, false
	}
	return v.Leave(n)
}

// DropProcedureStmt is a statement to drop a stored procedure.
type DropProcedureStmt struct {
	ddlNode

	IfExists bool
	Name     *TableName
	// Names holds all dropped procedures, the first one is Name.
	Names []*TableName
	// Nil entries keep omitted signatures distinct from explicit empty parentheses.
	Arguments []*List
	Behavior  DropBehavior
}

// Restore implements Node interface.
func (n *DropProcedureStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("DROP PROCEDURE ")
	if n.IfExists {
		ctx.WriteKeyword("IF EXISTS ")
	}
	return restoreRoutineTargets(ctx, n.droppedNames(), n.Arguments, n.Behavior)
}

// droppedNames returns all dropped procedure names.
func (n *DropProcedureStmt) droppedNames() []*TableName {
	if len(n.Names) > 0 {
		return n.Names
	}
	if n.Name != nil {
		return []*TableName{n.Name}
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *DropProcedureStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*DropProcedureStmt)
	names, ok := acceptRoutineTargets(v, n.droppedNames(), n.Arguments)
	n.Names = names
	if len(names) > 0 {
		n.Name = names[0]
	}
	if !ok {
		return n, false
	}
	return v.Leave(n)
}

func restoreRoutineTargets(ctx *format.RestoreCtx, names []*TableName, args []*List, behavior DropBehavior) error {
	if len(names) == 0 {
		return errors.New("routine drop has no targets")
	}
	if len(args) > 0 && len(args) != len(names) {
		return errors.New("routine signatures do not match the targets")
	}
	if behavior != DROP_RESTRICT && behavior != DROP_CASCADE {
		return errors.New("invalid routine drop behavior")
	}
	for i, name := range names {
		if name == nil {
			return errors.New("routine drop has a nil target")
		}
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if err := name.Restore(ctx); err != nil {
			return err
		}
		if len(args) > 0 && args[i] != nil {
			ctx.WritePlain("(")
			if err := args[i].Restore(ctx); err != nil {
				return err
			}
			ctx.WritePlain(")")
		}
	}
	if behavior == DROP_CASCADE {
		ctx.WriteKeyword(" CASCADE")
	}
	return ctx.Err()
}

func acceptRoutineTargets(v Visitor, names []*TableName, args []*List) ([]*TableName, bool) {
	for i, name := range names {
		if name == nil {
			return names, false
		}
		updated, ok := name.Accept(v)
		if !ok {
			return names, false
		}
		names[i] = updated.(*TableName)
	}
	for i, arg := range args {
		if arg == nil {
			continue
		}
		updated, ok := arg.Accept(v)
		if !ok {
			return names, false
		}
		args[i] = updated.(*List)
	}
	return names, true
}

// DropTriggerStmt is a statement to drop a trigger.
type DropTriggerStmt struct {
	ddlNode

	IfExists bool
	Name     *TableName
	// Table is the table the trigger belongs to.
	Table *TableName
}

// Restore implements Node interface.
func (n *DropTriggerStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("DROP TRIGGER ")
	if n.IfExists {
		ctx.WriteKeyword("IF EXISTS ")
	}
	if err := n.Name.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore DropTriggerStmt.Name")
	}
	if n.Table != nil {
		ctx.WriteKeyword(" ON ")
		if err := n.Table.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore DropTriggerStmt.Table")
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DropTriggerStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*DropTriggerStmt)
	node, ok := n.Name.Accept(v)
	if !ok {
		return n, false
	}
	n.Name = node.(*TableName)
	return v.Leave(n)
}

// RenameTableStmt is a statement to rename a table.
type RenameTableStmt struct {
	ddlNode

	OldTable *TableName
	NewTable *TableName

	// TableToTables is only useful for tools that depend heavily on this parser.
	// TODO: Refactor this when you are going to add full support for multiple schema changes.
	TableToTables []*TableToTable
}

// Restore implements Node interface.
func (n *RenameTableStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("RENAME TABLE ")
	for index, table2table := range n.TableToTables {
		if index != 0 {
			ctx.WritePlain(", ")
		}
		if err := table2table.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore RenameTableStmt.TableToTables")
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *RenameTableStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*RenameTableStmt)
	node, ok := n.OldTable.Accept(v)
	if !ok {
		return n, false
	}
	n.OldTable = node.(*TableName)
	node, ok = n.NewTable.Accept(v)
	if !ok {
		return n, false
	}
	n.NewTable = node.(*TableName)

	for i, t := range n.TableToTables {
		node, ok := t.Accept(v)
		if !ok {
			return n, false
		}
		n.TableToTables[i] = node.(*TableToTable)
	}

	return v.Leave(n)
}

// TableToTable represents renaming old table to new table used in RenameTableStmt.
type TableToTable struct {
	node
	OldTable *TableName
	NewTable *TableName
}

// Restore implements Node interface.
func (n *TableToTable) Restore(ctx *format.RestoreCtx) error {
	if err := n.OldTable.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore TableToTable.OldTable")
	}
	ctx.WriteKeyword(" TO ")
	if err := n.NewTable.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore TableToTable.NewTable")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *TableToTable) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*TableToTable)
	node, ok := n.OldTable.Accept(v)
	if !ok {
		return n, false
	}
	n.OldTable = node.(*TableName)
	node, ok = n.NewTable.Accept(v)
	if !ok {
		return n, false
	}
	n.NewTable = node.(*TableName)
	return v.Leave(n)
}

// CreateViewStmt is a statement to create a View.
type CreateViewStmt struct {
	ddlNode

	OrReplace      bool
	Temporary      bool
	ViewName       *TableName
	Cols           []lex.Ident
	Select         StmtNode
	WithOptions    *List
	HasCheckOption bool
	CheckOption    model.ViewCheckOption
	Algorithm      model.ViewAlgorithm
	Security       model.ViewSecurity
}

// Restore implements Node interface.
func (n *CreateViewStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("CREATE ")
	if n.OrReplace {
		ctx.WriteKeyword("OR REPLACE ")
	}
	if n.Temporary {
		ctx.WriteKeyword("TEMPORARY ")
	}
	ctx.WriteKeyword("VIEW ")
	if err := n.ViewName.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while create CreateViewStmt.ViewName")
	}
	if len(n.Cols) > 0 {
		ctx.WritePlain(" (")
		for i, col := range n.Cols {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			ctx.WriteIdent(col)
		}
		ctx.WritePlain(")")
	}
	if n.WithOptions != nil {
		ctx.WriteKeyword(" WITH ")
		if err := restoreRelOptions(ctx, n.WithOptions); err != nil {
			return err
		}
	}
	ctx.WriteKeyword(" AS ")
	if err := n.Select.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while create CreateViewStmt.Select")
	}
	if n.HasCheckOption {
		ctx.WriteKeyword(" WITH ")
		ctx.WriteKeyword(n.CheckOption.String())
		ctx.WriteKeyword(" CHECK OPTION")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CreateViewStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*CreateViewStmt)
	node, ok := n.ViewName.Accept(v)
	if !ok {
		return n, false
	}
	n.ViewName = node.(*TableName)
	selnode, ok := n.Select.Accept(v)
	if !ok {
		return n, false
	}
	if sel, ok := selnode.(StmtNode); ok {
		n.Select = sel
	}
	if n.WithOptions != nil {
		child, ok := n.WithOptions.Accept(v)
		if !ok {
			return n, false
		}
		n.WithOptions = child.(*List)
	}
	return v.Leave(n)
}

// CreateIndexStmt is a statement to create an index.
type CreateIndexStmt struct {
	ddlNode

	IndexName QualifiedName
	Table     *TableName
	Unique    bool
	// NullsNotDistinct is true for `UNIQUE ... NULLS NOT DISTINCT` indexes.
	NullsNotDistinct bool
	IfNotExists      bool
	Only             bool
	Concurrently     bool
	IndexColNames    []*IndexColName
	IndexOption      *IndexOption
	// Include is the optional INCLUDE column list of a covering index.
	Include []*ColumnName
	// Where is the optional predicate of a partial index.
	Where ExprNode
	// Using is the optional index access method.
	Using lex.Ident
	// Tablespace is the optional index tablespace.
	Tablespace lex.Ident
	// WithOptions is the optional storage parameter list.
	WithOptions *List
}

// Restore implements Node interface.
func (n *CreateIndexStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("CREATE ")
	if n.Unique {
		ctx.WriteKeyword("UNIQUE ")
	}
	ctx.WriteKeyword("INDEX ")
	if n.Concurrently {
		ctx.WriteKeyword("CONCURRENTLY ")
	}
	if n.IfNotExists {
		ctx.WriteKeyword("IF NOT EXISTS ")
	}
	if len(n.IndexName.Parts) > 0 {
		if err := restoreQualifiedName(ctx, n.IndexName); err != nil {
			return err
		}
		ctx.WritePlain(" ")
	}
	ctx.WriteKeyword("ON ")
	if n.Only {
		ctx.WriteKeyword("ONLY ")
	}
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CreateIndexStmt.Table")
	}
	if n.Using.Present() {
		ctx.WriteKeyword(" USING ")
		ctx.WriteIdent(n.Using)
	}

	ctx.WritePlain(" (")
	for i, indexColName := range n.IndexColNames {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := indexColName.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore CreateIndexStmt.IndexColNames: [%v]", i)
		}
	}
	ctx.WritePlain(")")
	if len(n.Include) > 0 {
		ctx.WriteKeyword(" INCLUDE (")
		for i, col := range n.Include {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			ctx.WriteName(col.Name.O)
		}
		ctx.WritePlain(")")
	}
	if n.NullsNotDistinct {
		ctx.WriteKeyword(" NULLS NOT DISTINCT")
	}
	if n.WithOptions != nil {
		ctx.WriteKeyword(" WITH ")
		if err := restoreRelOptions(ctx, n.WithOptions); err != nil {
			return err
		}
	}
	if n.Tablespace.Present() {
		ctx.WriteKeyword(" TABLESPACE ")
		ctx.WriteIdent(n.Tablespace)
	}
	if n.Where != nil {
		ctx.WriteKeyword(" WHERE ")
		if err := n.Where.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore CreateIndexStmt.Where")
		}
	}
	if n.IndexOption != nil && (n.IndexOption.Tp != model.IndexTypeInvalid || n.IndexOption.KeyBlockSize > 0 || n.IndexOption.Comment != "") {
		ctx.WritePlain(" ")
		if err := n.IndexOption.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore CreateIndexStmt.IndexOption")
		}
	}

	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CreateIndexStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*CreateIndexStmt)
	node, ok := n.Table.Accept(v)
	if !ok {
		return n, false
	}
	n.Table = node.(*TableName)
	for i, val := range n.IndexColNames {
		node, ok = val.Accept(v)
		if !ok {
			return n, false
		}
		n.IndexColNames[i] = node.(*IndexColName)
	}
	if n.IndexOption != nil {
		node, ok := n.IndexOption.Accept(v)
		if !ok {
			return n, false
		}
		n.IndexOption = node.(*IndexOption)
	}
	for i, col := range n.Include {
		child, ok := col.Accept(v)
		if !ok {
			return n, false
		}
		n.Include[i] = child.(*ColumnName)
	}
	if n.Where != nil {
		child, ok := n.Where.Accept(v)
		if !ok {
			return n, false
		}
		n.Where = child.(ExprNode)
	}
	if n.WithOptions != nil {
		child, ok := n.WithOptions.Accept(v)
		if !ok {
			return n, false
		}
		n.WithOptions = child.(*List)
	}
	return v.Leave(n)
}

// DropIndexStmt is a statement to drop the index.
type DropIndexStmt struct {
	ddlNode

	IfExists     bool
	Concurrently bool
	IndexName    QualifiedName
	Table        *TableName
}

// Restore implements Node interface.
func (n *DropIndexStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("DROP INDEX ")
	if n.Concurrently {
		ctx.WriteKeyword("CONCURRENTLY ")
	}
	if n.IfExists {
		ctx.WriteKeyword("IF EXISTS ")
	}
	if err := restoreQualifiedName(ctx, n.IndexName); err != nil {
		return err
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DropIndexStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*DropIndexStmt)
	if n.Table != nil {
		node, ok := n.Table.Accept(v)
		if !ok {
			return n, false
		}
		n.Table = node.(*TableName)
	}
	return v.Leave(n)
}

// TableOptionType is the type for TableOption
type TableOptionType int

// TableOption types.
const (
	TableOptionNone TableOptionType = iota
	TableOptionEngine
	TableOptionCharset
	TableOptionCollate
	TableOptionAutoIncrement
	TableOptionComment
	TableOptionAvgRowLength
	TableOptionCheckSum
	TableOptionCompression
	TableOptionConnection
	TableOptionPassword
	TableOptionKeyBlockSize
	TableOptionMaxRows
	TableOptionMinRows
	TableOptionDelayKeyWrite
	TableOptionRowFormat
	TableOptionStatsPersistent
	TableOptionShardRowID
	TableOptionPackKeys
)

// RowFormat types
const (
	RowFormatDefault uint64 = iota + 1
	RowFormatDynamic
	RowFormatFixed
	RowFormatCompressed
	RowFormatRedundant
	RowFormatCompact
	TokuDBRowFormatDefault
	TokuDBRowFormatFast
	TokuDBRowFormatSmall
	TokuDBRowFormatZlib
	TokuDBRowFormatQuickLZ
	TokuDBRowFormatLzma
	TokuDBRowFormatSnappy
	TokuDBRowFormatUncompressed
)

// OnDuplicateKeyHandlingType is the option that handle unique key values in 'CREATE TABLE ... SELECT' or `LOAD DATA`.
type OnDuplicateKeyHandlingType int

// OnDuplicateKeyHandling types
const (
	OnDuplicateKeyHandlingError OnDuplicateKeyHandlingType = iota
	OnDuplicateKeyHandlingIgnore
	OnDuplicateKeyHandlingReplace
)

// TableOption is used for parsing table option from SQL.
type TableOption struct {
	Tp        TableOptionType
	StrValue  string
	UintValue uint64
}

// Restore implements Node interface.
func (n *TableOption) Restore(ctx *format.RestoreCtx) error {
	switch n.Tp {
	case TableOptionEngine:
		ctx.WriteKeyword("ENGINE ")
		ctx.WritePlain("= ")
		if n.StrValue != "" {
			ctx.WritePlain(n.StrValue)
		} else {
			ctx.WritePlain("''")
		}
	case TableOptionCharset:
		ctx.WriteKeyword("DEFAULT CHARACTER SET ")
		ctx.WritePlain("= ")
		ctx.WritePlain(n.StrValue)
	case TableOptionCollate:
		ctx.WriteKeyword("DEFAULT COLLATE ")
		ctx.WritePlain("= ")
		ctx.WritePlain(n.StrValue)
	case TableOptionAutoIncrement:
		ctx.WriteKeyword("AUTO_INCREMENT ")
		ctx.WritePlain("= ")
		ctx.WritePlainf("%d", n.UintValue)
	case TableOptionComment:
		ctx.WriteKeyword("COMMENT ")
		ctx.WritePlain("= ")
		ctx.WriteString(n.StrValue)
	case TableOptionAvgRowLength:
		ctx.WriteKeyword("AVG_ROW_LENGTH ")
		ctx.WritePlain("= ")
		ctx.WritePlainf("%d", n.UintValue)
	case TableOptionCheckSum:
		ctx.WriteKeyword("CHECKSUM ")
		ctx.WritePlain("= ")
		ctx.WritePlainf("%d", n.UintValue)
	case TableOptionCompression:
		ctx.WriteKeyword("COMPRESSION ")
		ctx.WritePlain("= ")
		ctx.WriteString(n.StrValue)
	case TableOptionConnection:
		ctx.WriteKeyword("CONNECTION ")
		ctx.WritePlain("= ")
		ctx.WriteString(n.StrValue)
	case TableOptionPassword:
		ctx.WriteKeyword("PASSWORD ")
		ctx.WritePlain("= ")
		ctx.WriteString(n.StrValue)
	case TableOptionKeyBlockSize:
		ctx.WriteKeyword("KEY_BLOCK_SIZE ")
		ctx.WritePlain("= ")
		ctx.WritePlainf("%d", n.UintValue)
	case TableOptionMaxRows:
		ctx.WriteKeyword("MAX_ROWS ")
		ctx.WritePlain("= ")
		ctx.WritePlainf("%d", n.UintValue)
	case TableOptionMinRows:
		ctx.WriteKeyword("MIN_ROWS ")
		ctx.WritePlain("= ")
		ctx.WritePlainf("%d", n.UintValue)
	case TableOptionDelayKeyWrite:
		ctx.WriteKeyword("DELAY_KEY_WRITE ")
		ctx.WritePlain("= ")
		ctx.WritePlainf("%d", n.UintValue)
	case TableOptionRowFormat:
		ctx.WriteKeyword("ROW_FORMAT ")
		ctx.WritePlain("= ")
		switch n.UintValue {
		case RowFormatDefault:
			ctx.WriteKeyword("DEFAULT")
		case RowFormatDynamic:
			ctx.WriteKeyword("DYNAMIC")
		case RowFormatFixed:
			ctx.WriteKeyword("FIXED")
		case RowFormatCompressed:
			ctx.WriteKeyword("COMPRESSED")
		case RowFormatRedundant:
			ctx.WriteKeyword("REDUNDANT")
		case RowFormatCompact:
			ctx.WriteKeyword("COMPACT")
		case TokuDBRowFormatDefault:
			ctx.WriteKeyword("TOKUDB_DEFAULT")
		case TokuDBRowFormatFast:
			ctx.WriteKeyword("TOKUDB_FAST")
		case TokuDBRowFormatSmall:
			ctx.WriteKeyword("TOKUDB_SMALL")
		case TokuDBRowFormatZlib:
			ctx.WriteKeyword("TOKUDB_ZLIB")
		case TokuDBRowFormatQuickLZ:
			ctx.WriteKeyword("TOKUDB_QUICKLZ")
		case TokuDBRowFormatLzma:
			ctx.WriteKeyword("TOKUDB_LZMA")
		case TokuDBRowFormatSnappy:
			ctx.WriteKeyword("TOKUDB_SNAPPY")
		case TokuDBRowFormatUncompressed:
			ctx.WriteKeyword("TOKUDB_UNCOMPRESSED")
		default:
			return errors.Errorf("invalid TableOption: TableOptionRowFormat: %d", n.UintValue)
		}
	case TableOptionStatsPersistent:
		// TODO: not support
		ctx.WriteKeyword("STATS_PERSISTENT ")
		ctx.WritePlain("= ")
		ctx.WriteKeyword("DEFAULT")
		ctx.WritePlain(" /* TableOptionStatsPersistent is not supported */ ")
	case TableOptionShardRowID:
		ctx.WriteKeyword("SHARD_ROW_ID_BITS ")
		ctx.WritePlain("= ")
		ctx.WritePlainf("%d", n.UintValue)
	case TableOptionPackKeys:
		// TODO: not support
		ctx.WriteKeyword("PACK_KEYS ")
		ctx.WritePlain("= ")
		ctx.WriteKeyword("DEFAULT")
		ctx.WritePlain(" /* TableOptionPackKeys is not supported */ ")
	default:
		return errors.Errorf("invalid TableOption: %d", n.Tp)
	}
	return ctx.Err()
}

// ColumnPositionType is the type for ColumnPosition.
type ColumnPositionType int

// ColumnPosition Types
const (
	ColumnPositionNone ColumnPositionType = iota
	ColumnPositionFirst
	ColumnPositionAfter
)

// ColumnPosition represent the position of the newly added column
type ColumnPosition struct {
	node
	// Tp is either ColumnPositionNone, ColumnPositionFirst or ColumnPositionAfter.
	Tp ColumnPositionType
	// RelativeColumn is the column the newly added column after if type is ColumnPositionAfter
	RelativeColumn *ColumnName
}

// Restore implements Node interface.
func (n *ColumnPosition) Restore(ctx *format.RestoreCtx) error {
	switch n.Tp {
	case ColumnPositionNone:
		// do nothing
	case ColumnPositionFirst:
		ctx.WriteKeyword("FIRST")
	case ColumnPositionAfter:
		ctx.WriteKeyword("AFTER ")
		if err := n.RelativeColumn.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore ColumnPosition.RelativeColumn")
		}
	default:
		return errors.Errorf("invalid ColumnPositionType: %d", n.Tp)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ColumnPosition) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*ColumnPosition)
	if n.RelativeColumn != nil {
		node, ok := n.RelativeColumn.Accept(v)
		if !ok {
			return n, false
		}
		n.RelativeColumn = node.(*ColumnName)
	}
	return v.Leave(n)
}

// AlterTableType is the type for AlterTableSpec.
type AlterTableType int

// AlterTable types.
const (
	AlterTableOption AlterTableType = iota + 1
	AlterTableAddColumns
	AlterTableAddConstraint
	AlterTableDropColumn
	AlterTableDropPrimaryKey
	AlterTableDropIndex
	AlterTableDropForeignKey
	AlterTableModifyColumn
	AlterTableChangeColumn
	AlterTableRenameTable
	AlterTableAlterColumn
	AlterTableLock
	AlterTableAlgorithm
	AlterTableRenameIndex
	AlterTableForce
	AlterTableAddPartitions
	AlterTableCoalescePartitions
	AlterTableDropPartition
	AlterTableTruncatePartition
	AlterTableRenameColumn
	AlterTableExchangePartition
	AlterTableReorganizePartition
	AlterTableRemovePartitioning
	AlterTableDiscardTablespace
	AlterTableImportTablespace
	AlterTableGeneric
	AlterTableAlterColumnType
	AlterTableDropDefault
	AlterTableSetNotNull
	AlterTableDropNotNull
	AlterTableSetStatistics
	AlterTableSetStorage
	AlterTableOwner
	AlterTableSetTablespace
	AlterTableSetLogged
	AlterTableSetUnlogged
	AlterTableEnableTrigger
	AlterTableDisableTrigger
	// AlterTableEnableRule enables a rule of a table.
	AlterTableEnableRule
	AlterTableSetRelOptions
	AlterTableResetRelOptions
	AlterTableSetCluster
	AlterTableSetWithoutCluster
	AlterTableInherit
	AlterTableNoInherit
	AlterTableOf
	AlterTableNotOf
	AlterTableReplicaIdentity
	AlterTableRowLevelSecurity
	AlterTableRenameConstraint
	AlterTableValidateConstraint
	AlterTableAlterConstraint
	AlterTableAttachPartition
	AlterTableDetachPartition
	AlterTableDisableRule
	AlterTableSetOption
	AlterTableSetOn
	AlterTableForeignOptions

	// TODO: Add more actions
)

// LockType is the type for AlterTableSpec.
type LockType byte

func (n LockType) String() string {
	switch n {
	case LockTypeNone:
		return "NONE"
	case LockTypeDefault:
		return "DEFAULT"
	case LockTypeShared:
		return "SHARED"
	case LockTypeExclusive:
		return "EXCLUSIVE"
	}
	return ""
}

// Lock Types.
const (
	LockTypeNone LockType = iota + 1
	LockTypeDefault
	LockTypeShared
	LockTypeExclusive
)

// AlterAlgorithm is the algorithm of the DDL operations.
type AlterAlgorithm byte

// DDL alter algorithms.
// For now, only supported inplace and instance algorithms. If the user specify `copy`,
// will get an error.
const (
	AlterAlgorithmDefault AlterAlgorithm = iota
	AlterAlgorithmCopy
	AlterAlgorithmInplace
	AlterAlgorithmInstant
)

func (a AlterAlgorithm) String() string {
	switch a {
	case AlterAlgorithmDefault:
		return "DEFAULT"
	case AlterAlgorithmCopy:
		return "COPY"
	case AlterAlgorithmInplace:
		return "INPLACE"
	case AlterAlgorithmInstant:
		return "INSTANT"
	default:
		return "DEFAULT"
	}
}

// PartitionBoundSpec represents a partition bound specification.
type PartitionBoundSpec struct {
	node

	BoundType string
	From      []ExprNode
	To        []ExprNode
	In        []ExprNode
	Options   *List
}

// Restore implements Node interface.
func (n *PartitionBoundSpec) Restore(ctx *format.RestoreCtx) error {
	switch n.BoundType {
	case "FROM":
		ctx.WriteKeyword("FOR VALUES FROM (")
		for i, expr := range n.From {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			if err := expr.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore PartitionBoundSpec.From[%d]", i)
			}
		}
		ctx.WriteKeyword(") TO (")
		for i, expr := range n.To {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			if err := expr.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore PartitionBoundSpec.To[%d]", i)
			}
		}
		ctx.WritePlain(")")
	case "IN":
		ctx.WriteKeyword("FOR VALUES IN (")
		for i, expr := range n.In {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			if err := expr.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore PartitionBoundSpec.In[%d]", i)
			}
		}
		ctx.WritePlain(")")
	case "WITH", "MODULUS":
		ctx.WriteKeyword("FOR VALUES WITH (")
		if n.Options == nil {
			return errors.New("hash partition has no bounds")
		}
		for i, item := range n.Options.Items {
			option, ok := item.(*DefElem)
			if !ok || option == nil || option.Arg == nil {
				return errors.New("invalid hash partition bound")
			}
			if i > 0 {
				ctx.WritePlain(", ")
			}
			ctx.WriteKeyword(option.Defname)
			ctx.WritePlain(" ")
			if err := option.Arg.Restore(ctx); err != nil {
				return err
			}
		}
		ctx.WritePlain(")")
	case "DEFAULT", "default":
		ctx.WriteKeyword("DEFAULT")
	default:
		return errors.New("invalid partition bound type")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *PartitionBoundSpec) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*PartitionBoundSpec)
	for i, expr := range n.From {
		node, ok := expr.Accept(v)
		if !ok {
			return n, false
		}
		n.From[i] = node.(ExprNode)
	}
	for i, expr := range n.To {
		node, ok := expr.Accept(v)
		if !ok {
			return n, false
		}
		n.To[i] = node.(ExprNode)
	}
	for i, expr := range n.In {
		node, ok := expr.Accept(v)
		if !ok {
			return n, false
		}
		n.In[i] = node.(ExprNode)
	}
	if n.Options != nil {
		child, ok := n.Options.Accept(v)
		if !ok {
			return n, false
		}
		n.Options = child.(*List)
	}
	return v.Leave(n)
}

// AlterTableSpec represents alter table specification.
type AlterTableSpec struct {
	node

	Tp                AlterTableType
	Name              string
	Constraint        *Constraint
	Options           []*TableOption
	NewTable          *TableName
	NewColumns        []*ColumnDef
	OldColumnName     *ColumnName
	NewColumnName     *ColumnName
	Position          *ColumnPosition
	LockType          LockType
	Algorithm         AlterAlgorithm
	Comment           string
	FromKey           model.CIStr
	ToKey             model.CIStr
	PartDefinitions   []*PartitionDefinition
	PartitionNames    []model.CIStr
	WithoutValidation bool
	Num               int64
	Raw               string
	Owner             *RoleSpec
	Storage           string
	RelOptions        *List
	ConstraintOptions string
	TriggerMode       string
	// Def holds the definition of the alter table command.
	Def Node
	// Behavior is the drop behavior of the alter table command.
	Behavior DropBehavior
	// MissingOk is true when the object of the command may be missing.
	MissingOk bool
	// Concurrent is true for a concurrent partition operation.
	Concurrent       bool
	Finalize         bool
	Attribute        bool
	ReplicaIdentity  string
	RowLevelSecurity string
	SetValue         string
	PartitionBound   *PartitionBoundSpec
}

// Restore implements Node interface.
func (n *AlterTableSpec) Restore(ctx *format.RestoreCtx) error {
	column := "COLUMN"
	if n.Attribute {
		column = "ATTRIBUTE"
	}
	switch n.Tp {
	case AlterTableDropColumn, AlterTableModifyColumn, AlterTableAlterColumn,
		AlterTableAlterColumnType, AlterTableDropDefault, AlterTableSetNotNull,
		AlterTableDropNotNull, AlterTableSetStorage, AlterTableRenameColumn:
		if n.OldColumnName == nil {
			return errors.New("ALTER command has no column name")
		}
	}
	switch n.Tp {
	case AlterTableAddColumns:
		if len(n.NewColumns) != 1 || n.NewColumns[0] == nil {
			return errors.New("ADD requires one column definition")
		}
		ctx.WriteKeyword("ADD " + column + " ")
		if n.MissingOk {
			ctx.WriteKeyword("IF NOT EXISTS ")
		}
		if err := n.NewColumns[0].Restore(ctx); err != nil {
			return err
		}
	case AlterTableAddConstraint:
		if n.Constraint == nil {
			return errors.New("ADD has no constraint")
		}
		ctx.WriteKeyword("ADD ")
		if err := n.Constraint.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore AlterTableSpec.Constraint")
		}
	case AlterTableDropColumn:
		ctx.WriteKeyword("DROP " + column + " ")
		if n.MissingOk {
			ctx.WriteKeyword("IF EXISTS ")
		}
		if err := n.OldColumnName.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore AlterTableSpec.OldColumnName")
		}
	case AlterTableModifyColumn:
		if n.Def == nil {
			return errors.New("SET DEFAULT has no expression")
		}
		ctx.WriteKeyword("ALTER COLUMN ")
		if err := n.OldColumnName.Restore(ctx); err != nil {
			return err
		}
		ctx.WriteKeyword(" SET DEFAULT ")
		return n.Def.Restore(ctx)
	case AlterTableAlterColumn:
		options, ok := n.Def.(*List)
		if !ok || options == nil {
			return errors.New("ALTER COLUMN OPTIONS has no options")
		}
		ctx.WriteKeyword("ALTER COLUMN ")
		if err := n.OldColumnName.Restore(ctx); err != nil {
			return err
		}
		ctx.WritePlain(" ")
		return restoreAlterOptions(ctx, options)
	case AlterTableRenameColumn:
		if n.NewColumnName == nil {
			return errors.New("RENAME COLUMN has no new name")
		}
		ctx.WriteKeyword("RENAME COLUMN ")
		if err := n.OldColumnName.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore AlterTableSpec.OldColumnName")
		}
		ctx.WriteKeyword(" TO ")
		if err := n.NewColumnName.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore AlterTableSpec.NewColumnName")
		}
	case AlterTableGeneric:
		return fmt.Errorf("%w: ALTER operation", ErrUnsupportedRestore)
	case AlterTableAlterColumnType:
		if len(n.NewColumns) != 1 || n.NewColumns[0] == nil || n.NewColumns[0].Tp == nil {
			return errors.New("ALTER TYPE has no data type")
		}
		ctx.WriteKeyword("ALTER " + column + " ")
		ctx.WriteName(n.OldColumnName.Name.O)
		ctx.WriteKeyword(" TYPE ")
		definition := n.NewColumns[0]
		if err := definition.Tp.Restore(ctx); err != nil {
			return err
		}
		for _, option := range definition.Options {
			if option == nil {
				return errors.New("ALTER TYPE has a nil option")
			}
			ctx.WritePlain(" ")
			if err := option.Restore(ctx); err != nil {
				return err
			}
		}
		if n.Def != nil {
			ctx.WriteKeyword(" USING ")
			if err := n.Def.Restore(ctx); err != nil {
				return err
			}
		}
	case AlterTableDropDefault:
		ctx.WriteKeyword("ALTER COLUMN ")
		ctx.WriteName(n.OldColumnName.Name.O)
		ctx.WriteKeyword(" DROP DEFAULT")
	case AlterTableSetNotNull:
		ctx.WriteKeyword("ALTER COLUMN ")
		ctx.WriteName(n.OldColumnName.Name.O)
		ctx.WriteKeyword(" SET NOT NULL")
	case AlterTableDropNotNull:
		ctx.WriteKeyword("ALTER COLUMN ")
		ctx.WriteName(n.OldColumnName.Name.O)
		ctx.WriteKeyword(" DROP NOT NULL")
	case AlterTableSetStatistics:
		ctx.WriteKeyword("ALTER COLUMN ")
		if n.OldColumnName != nil {
			ctx.WriteName(n.OldColumnName.Name.O)
		} else {
			if n.Def == nil {
				return errors.New("SET STATISTICS has no column number")
			}
			if err := n.Def.Restore(ctx); err != nil {
				return err
			}
		}
		ctx.WriteKeyword(" SET STATISTICS ")
		ctx.WritePlainf("%d", n.Num)
	case AlterTableSetStorage:
		ctx.WriteKeyword("ALTER COLUMN ")
		ctx.WriteName(n.OldColumnName.Name.O)
		ctx.WriteKeyword(" SET STORAGE ")
		ctx.WriteKeyword(n.Storage)
	case AlterTableOwner:
		if n.Owner == nil {
			return errors.New("ALTER OWNER has no role")
		}
		ctx.WriteKeyword("OWNER TO ")
		return n.Owner.Restore(ctx)
	case AlterTableSetTablespace:
		ctx.WriteKeyword("SET TABLESPACE ")
		ctx.WriteName(n.Name)
	case AlterTableSetLogged:
		ctx.WriteKeyword("SET LOGGED")
	case AlterTableSetUnlogged:
		ctx.WriteKeyword("SET UNLOGGED")
	case AlterTableEnableTrigger, AlterTableDisableTrigger, AlterTableEnableRule, AlterTableDisableRule:
		if n.Tp == AlterTableEnableTrigger || n.Tp == AlterTableEnableRule {
			ctx.WriteKeyword("ENABLE ")
		} else {
			ctx.WriteKeyword("DISABLE ")
		}
		if n.TriggerMode == "ALWAYS" || n.TriggerMode == "REPLICA" {
			ctx.WriteKeyword(n.TriggerMode)
			ctx.WritePlain(" ")
		}
		if n.Tp == AlterTableEnableRule || n.Tp == AlterTableDisableRule {
			ctx.WriteKeyword("RULE ")
		} else {
			ctx.WriteKeyword("TRIGGER ")
		}
		if n.TriggerMode == "ALL" || n.TriggerMode == "USER" {
			ctx.WriteKeyword(n.Name)
		} else {
			ctx.WriteName(n.Name)
		}
	case AlterTableSetRelOptions, AlterTableResetRelOptions:
		if n.OldColumnName != nil {
			ctx.WriteKeyword("ALTER COLUMN ")
			ctx.WriteName(n.OldColumnName.Name.O)
			ctx.WritePlain(" ")
		}
		if n.Tp == AlterTableSetRelOptions {
			ctx.WriteKeyword("SET ")
		} else {
			ctx.WriteKeyword("RESET ")
		}
		return restoreRelOptions(ctx, n.RelOptions)
	case AlterTableSetCluster:
		ctx.WriteKeyword("CLUSTER ON ")
		ctx.WriteName(n.Name)
	case AlterTableSetWithoutCluster:
		ctx.WriteKeyword("SET WITHOUT CLUSTER")
	case AlterTableInherit, AlterTableNoInherit:
		if n.NewTable == nil {
			return errors.New("INHERIT has no relation")
		}
		if n.Tp == AlterTableInherit {
			ctx.WriteKeyword("INHERIT ")
		} else {
			ctx.WriteKeyword("NO INHERIT ")
		}
		if err := n.NewTable.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore AlterTableSpec.NewTable")
		}
	case AlterTableOf, AlterTableNotOf:
		if n.Tp == AlterTableOf {
			if n.NewTable == nil {
				return errors.New("OF has no type")
			}
			ctx.WriteKeyword("OF ")
			if err := n.NewTable.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore AlterTableSpec.NewTable")
			}
		} else {
			ctx.WriteKeyword("NOT OF")
		}
	case AlterTableReplicaIdentity:
		ctx.WriteKeyword("REPLICA IDENTITY ")
		if n.ReplicaIdentity == "USING INDEX" {
			ctx.WriteKeyword("USING INDEX ")
			ctx.WriteName(n.Name)
		} else {
			ctx.WriteKeyword(n.ReplicaIdentity)
		}
	case AlterTableRowLevelSecurity:
		ctx.WriteKeyword(n.RowLevelSecurity)
		ctx.WriteKeyword(" ROW LEVEL SECURITY")
	case AlterTableRenameConstraint:
		if n.NewColumnName == nil {
			return errors.New("RENAME CONSTRAINT has no new name")
		}
		ctx.WriteKeyword("RENAME CONSTRAINT ")
		ctx.WriteName(n.Name)
		ctx.WriteKeyword(" TO ")
		ctx.WriteName(n.NewColumnName.Name.O)
	case AlterTableValidateConstraint:
		ctx.WriteKeyword("VALIDATE CONSTRAINT ")
		ctx.WriteName(n.Name)
	case AlterTableAlterConstraint:
		ctx.WriteKeyword("ALTER CONSTRAINT ")
		ctx.WriteName(n.Name)
		if n.ConstraintOptions != "" {
			ctx.WritePlain(" ")
			ctx.WritePlain(n.ConstraintOptions)
		}
	case AlterTableAttachPartition, AlterTableDetachPartition:
		if n.NewTable == nil {
			return errors.New("partition operation has no relation")
		}
		if n.Tp == AlterTableAttachPartition {
			ctx.WriteKeyword("ATTACH PARTITION ")
		} else {
			ctx.WriteKeyword("DETACH PARTITION ")
		}
		if err := n.NewTable.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore AlterTableSpec.NewTable")
		}
		if n.PartitionBound != nil {
			ctx.WritePlain(" ")
			if err := n.PartitionBound.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore AlterTableSpec.PartitionBound")
			}
		}
		if n.Concurrent {
			ctx.WriteKeyword(" CONCURRENTLY")
		}
		if n.Finalize {
			ctx.WriteKeyword(" FINALIZE")
		}
	case AlterTableForeignOptions:
		options, ok := n.Def.(*List)
		if !ok || options == nil {
			return errors.New("OPTIONS has no options")
		}
		return restoreAlterOptions(ctx, options)
	case AlterTableSetOption:
		ctx.WriteKeyword("SET ")
		ctx.WriteKeyword(n.Name)
		ctx.WritePlain(" ")
		ctx.WriteKeyword(n.SetValue)
	case AlterTableSetOn:
		ctx.WriteKeyword(n.Name)
		ctx.WriteKeyword(" ON ")
		ctx.WriteKeyword(n.SetValue)
	default:
		return fmt.Errorf("%w: ALTER operation %d", ErrUnsupportedRestore, n.Tp)
	}
	if n.Behavior == DROP_CASCADE {
		ctx.WriteKeyword(" CASCADE")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *AlterTableSpec) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*AlterTableSpec)
	if n.Constraint != nil {
		node, ok := n.Constraint.Accept(v)
		if !ok {
			return n, false
		}
		n.Constraint = node.(*Constraint)
	}
	if n.NewTable != nil {
		node, ok := n.NewTable.Accept(v)
		if !ok {
			return n, false
		}
		n.NewTable = node.(*TableName)
	}
	for i, col := range n.NewColumns {
		if col == nil {
			continue
		}
		node, ok := col.Accept(v)
		if !ok {
			return n, false
		}
		n.NewColumns[i] = node.(*ColumnDef)
	}
	if n.OldColumnName != nil {
		node, ok := n.OldColumnName.Accept(v)
		if !ok {
			return n, false
		}
		n.OldColumnName = node.(*ColumnName)
	}
	if n.Position != nil {
		node, ok := n.Position.Accept(v)
		if !ok {
			return n, false
		}
		n.Position = node.(*ColumnPosition)
	}
	if n.NewColumnName != nil {
		node, ok := n.NewColumnName.Accept(v)
		if !ok {
			return n, false
		}
		n.NewColumnName = node.(*ColumnName)
	}
	if n.PartitionBound != nil {
		node, ok := n.PartitionBound.Accept(v)
		if !ok {
			return n, false
		}
		n.PartitionBound = node.(*PartitionBoundSpec)
	}
	if n.Def != nil {
		child, ok := n.Def.Accept(v)
		if !ok {
			return n, false
		}
		n.Def = child
	}
	if n.RelOptions != nil {
		child, ok := n.RelOptions.Accept(v)
		if !ok {
			return n, false
		}
		n.RelOptions = child.(*List)
	}
	if n.Owner != nil {
		child, ok := n.Owner.Accept(v)
		if !ok {
			return n, false
		}
		n.Owner = child.(*RoleSpec)
	}
	return v.Leave(n)
}

// AlterTableStmt is a statement to change the structure of a table.
type AlterTableStmt struct {
	ddlNode

	Table   *TableName
	Specs   []*AlterTableSpec
	Objtype string
	Only    bool
	// MissingOk is true when the table of the statement may be missing.
	MissingOk bool
}

// Restore implements Node interface.
func (n *AlterTableStmt) Restore(ctx *format.RestoreCtx) error {
	if n.Table == nil || len(n.Specs) == 0 {
		return errors.New("ALTER has no relation or commands")
	}
	kind := n.Objtype
	if kind == "" {
		kind = "TABLE"
	}
	switch kind {
	case "TABLE", "INDEX", "SEQUENCE", "VIEW", "MATERIALIZED VIEW", "FOREIGN TABLE", "TYPE":
	default:
		return errors.Errorf("invalid ALTER object kind %q", kind)
	}
	ctx.WriteKeyword("ALTER " + kind + " ")
	if n.MissingOk {
		ctx.WriteKeyword("IF EXISTS ")
	}
	if n.Only {
		ctx.WriteKeyword("ONLY ")
	}
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore AlterTableStmt.Table")
	}
	for i, spec := range n.Specs {
		if spec == nil {
			return errors.New("ALTER has a nil command")
		}
		if i == 0 {
			ctx.WritePlain(" ")
		} else {
			ctx.WritePlain(", ")
		}
		if err := spec.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore AlterTableStmt.Specs[%d]", i)
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *AlterTableStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*AlterTableStmt)
	if n.Table != nil {
		child, ok := n.Table.Accept(v)
		if !ok {
			return n, false
		}
		n.Table = child.(*TableName)
	}
	for i, val := range n.Specs {
		if val == nil {
			continue
		}
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Specs[i] = node.(*AlterTableSpec)
	}
	return v.Leave(n)
}

// TruncateTableStmt is a statement to empty a table completely.
type TruncateTableStmt struct {
	ddlNode

	Table *TableName
	// Tables holds the full list of tables when more than one is truncated.
	Tables []*TableName
	// Only is true when ONLY was specified.
	Only bool
	// Behavior is "CASCADE", "RESTRICT", or empty.
	Behavior string
}

// Restore implements Node interface.
func (n *TruncateTableStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("TRUNCATE ")
	// PostgreSQL wants TRUNCATE [TABLE] [ONLY] name, so TABLE comes first.
	ctx.WriteKeyword("TABLE ")
	if n.Only {
		ctx.WriteKeyword("ONLY ")
	}
	tables := n.Tables
	if len(tables) == 0 && n.Table != nil {
		tables = []*TableName{n.Table}
	}
	for i, tbl := range tables {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := tbl.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore TruncateTableStmt.Table")
		}
	}
	if n.Behavior != "" {
		ctx.WritePlain(" ")
		ctx.WriteKeyword(n.Behavior)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *TruncateTableStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*TruncateTableStmt)
	node, ok := n.Table.Accept(v)
	if !ok {
		return n, false
	}
	n.Table = node.(*TableName)
	return v.Leave(n)
}

// PartitionDefinition defines a single partition.
type PartitionDefinition struct {
	Name     model.CIStr
	LessThan []ExprNode
	MaxValue bool
	Comment  string
}

// Restore implements Node interface.
func (n *PartitionDefinition) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("PARTITION ")
	ctx.WriteName(n.Name.O)
	if n.LessThan != nil {
		ctx.WriteKeyword(" VALUES LESS THAN ")
		ctx.WritePlain("(")
		for k, less := range n.LessThan {
			if k != 0 {
				ctx.WritePlain(", ")
			}
			if err := less.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore PartitionDefinition.LessThan[%d]", k)
			}
		}
		ctx.WritePlain(")")
	}
	if n.Comment != "" {
		ctx.WriteKeyword(" COMMENT ")
		ctx.WritePlain("= ")
		ctx.WriteString(n.Comment)
	}
	return ctx.Err()
}

// PartitionOptions specifies the partition options.
type PartitionOptions struct {
	Tp          model.PartitionType
	Expr        ExprNode
	ColumnNames []*ColumnName
	Definitions []*PartitionDefinition
	Num         uint64
}

// Restore implements Node interface.
func (n *PartitionOptions) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("PARTITION BY ")
	switch n.Tp {
	case model.PartitionTypeRange:
		ctx.WriteKeyword("RANGE ")
	case model.PartitionTypeHash:
		ctx.WriteKeyword("HASH ")
	case model.PartitionTypeList:
		return errors.New("Parser ignore the `PartitionTypeList` type now")
	default:
		return errors.Errorf("invalid model.PartitionType: %d", n.Tp)
	}

	if n.Expr != nil {
		ctx.WritePlain("(")
		if err := n.Expr.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore PartitionOptions Expr")
		}
		ctx.WritePlain(") ")
	}
	if len(n.ColumnNames) > 0 {
		ctx.WriteKeyword("COLUMNS")
		ctx.WritePlain("(")
		for i, col := range n.ColumnNames {
			if i > 0 {
				ctx.WritePlain(",")
			}
			if err := col.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while splicing PartitionOptions ColumnName: [%v]", i)
			}
		}
		ctx.WritePlain(") ")
	}
	if n.Num > 0 {
		ctx.WriteKeyword("PARTITIONS ")
		ctx.WritePlainf("%d", n.Num)
	}

	if len(n.Definitions) > 0 {
		ctx.WritePlain("(")
		for i, def := range n.Definitions {
			if i > 0 {
				ctx.WritePlain(",")
			}
			if err := def.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while splicing PartitionOptions Definitions: [%v]", i)
			}
		}
		ctx.WritePlain(")")
	}

	return ctx.Err()
}
