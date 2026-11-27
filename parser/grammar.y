// PostgreSQL grammar for Go - translated from PostgreSQL's gram.y
// This grammar is compatible with goyacc (golang.org/x/tools/cmd/goyacc)
//
// To generate the parser:
//   go run golang.org/x/tools/cmd/goyacc@v0.52.0 -o parser/parser_generated.go -l -p yy parser/grammar.y

%{
package parser

import (
	"fmt"
	"math"
	"strings"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/model"
	"github.com/mia0x75/pgparser/opcode"
	"github.com/mia0x75/pgparser/types"
)

%}

// Union type for semantic values
%union {
	span      loc.Span
	node      ast.Node
	expr      ast.ExprNode
	exprs     []ast.ExprNode
	fields    []*ast.SelectItem
	froms     []ast.ResultSetNode
	list      *ast.List
	str       string
	ident     lex.Ident
	strp      *string
	ival      int64
	boolean   bool
	ivalp     *int64
	jtype     pkast.JoinType // for join types
	joinqual  *joinQual
	setop     int        // for SetOperation values
	slimit    *SelectLimit  // for LIMIT/OFFSET
	tp        *types.FieldType // for Typename values
	partspec  *ast.PartitionSpec
	partbound *ast.PartitionBoundSpec
	grpclause  *GroupClause    // for GROUP BY clause
	keyaction  *KeyAction      // for FK key_action
	keyactions *KeyActions     // for FK key_actions
	iqual      *importQualification
	frame      ast.FrameExtent
	bound      *ast.FrameBound
	exclusion  pkast.WindowFrameExclusion
	ctes       []ast.CTENode
	colnames   []*ast.ColumnName
	alias      *sourceAlias
	colopts    []*ast.ColumnOption
	assignments []*ast.Assignment
	tps         []*types.FieldType
}

// Token types from the lexer
// Non-keyword tokens must be listed first
%token <str>   IDENT FCONST SCONST BCONST XCONST Op
%token <ival>  ICONST PARAM
%token         TYPECAST DOT_DOT COLON_EQUALS EQUALS_GREATER
%token         LESS_EQUALS GREATER_EQUALS NOT_EQUALS
%token         NOT_LA    /* lookahead token for NOT LIKE etc */
%token         WITH_LA   /* lookahead token for WITH TIME ZONE */
%token         NULLS_LA  /* lookahead token for NULLS FIRST/LAST */
%token         FORMAT_LA /* lookahead token for FORMAT JSON */
%token         WITHOUT_LA /* lookahead token for WITHOUT TIME ZONE */

// Keyword tokens (must match keywords.go)
%token <str> ABORT_P ABSENT ABSOLUTE_P ACCESS ACTION ADD_P ADMIN AFTER RESOURCE VCPU_P THREAD_PRIORITY_P
%token <str> AGGREGATE ALL ALSO ALTER ALWAYS ANALYSE ANALYZE AND ANY ARRAY AS ASC
%token <str> ASENSITIVE ASSERTION ASSIGNMENT ASYMMETRIC ATOMIC AT ATTACH ATTRIBUTE AUTHORIZATION
%token <str> BACKWARD BEFORE BEGIN_P BETWEEN BIGINT BINARY BIT
%token <str> BOOLEAN_P BOTH BREADTH BY
%token <str> CACHE CALL CALLED CASCADE CASCADED CASE CAST CATALOG_P CHAIN CHAR_P
%token <str> CHARACTER CHARACTERISTICS CHECK CHECKPOINT CLASS CLOSE
%token <str> CLUSTER COALESCE COLLATE COLLATION COLUMN COLUMNS COMMENT COMMENTS COMMIT
%token <str> COMMITTED COMPRESSION CONCURRENTLY CONDITIONAL CONFIGURATION CONFLICT
%token <str> CONNECTION CONSTRAINT CONSTRAINTS CONTENT_P CONTINUE_P CONVERSION_P COPY
%token <str> COST CREATE CROSS CSV CUBE CURRENT_P
%token <str> CURRENT_CATALOG CURRENT_DATE CURRENT_ROLE CURRENT_SCHEMA
%token <str> CURRENT_TIME CURRENT_TIMESTAMP CURRENT_USER CURSOR CYCLE
%token <str> DATA_P DATABASE DAY_P DEALLOCATE DEC DECIMAL_P DECLARE DEFAULT DEFAULTS
%token <str> DEFERRABLE DEFERRED DEFINER DELETE_P DELIMITER DELIMITERS DEPENDS DEPTH DESC
%token <str> DETACH DICTIONARY DISABLE_P DISCARD DISTINCT DO DOCUMENT_P DOMAIN_P
%token <str> DOUBLE_P DROP
%token <str> EACH ELSE EMPTY_P ENABLE_P ENCODING ENCRYPTED END_P ENUM_P ERROR_P ESCAPE
%token <str> EVENT EXCEPT EXCLUDE EXCLUDING EXCLUSIVE EXECUTE EXISTS EXPLAIN EXPRESSION
%token <str> EXTENSION EXTERNAL EXTRACT
%token <str> FALSE_P FAMILY FETCH FILTER FINALIZE FIRST_P FLOAT_P FOLLOWING FOR
%token <str> FORCE FOREIGN FORMAT FORWARD FREEZE FROM FULL FUNCTION FUNCTIONS
%token <str> GENERATED GLOBAL GRANT GRANTED GREATEST GROUP_P GROUPING GROUPS
%token <str> HANDLER HAVING HEADER_P HOLD HOUR_P
%token <str> IDENTITY_P IF_P ILIKE IMMEDIATE IMMUTABLE IMPLICIT_P IMPORT_P IN_P INCLUDE
%token <str> INCLUDING INCREMENT INDENT INDEX INDEXES INHERIT INHERITS INITIALLY INLINE_P
%token <str> INNER_P INOUT INPUT_P INSENSITIVE INSERT INSTEAD INT_P INTEGER
%token <str> INTERSECT INTERVAL INTO INVOKER IS ISNULL ISOLATION
%token <str> JOIN JSON JSON_ARRAY JSON_ARRAYAGG JSON_EXISTS JSON_OBJECT JSON_OBJECTAGG
%token <str> JSON_QUERY JSON_SCALAR JSON_SERIALIZE JSON_TABLE JSON_VALUE
%token <str> KEEP KEY KEYS
%token <str> LABEL LANGUAGE LARGE_P LAST_P LATERAL_P
%token <str> LEADING LEAKPROOF LEAST LEFT LEVEL LIKE LIMIT LISTEN LOAD LOCAL
%token <str> LOCALTIME LOCALTIMESTAMP LOCATION LOCK_P LOCKED LOGGED
%token <str> MAPPING MATCH MATCHED MATERIALIZED MAXVALUE MERGE MERGE_ACTION METHOD
%token <str> MINUTE_P MINVALUE MODE MONTH_P MOVE
%token <str> NAME_P NAMES NATIONAL NATURAL NCHAR NESTED NEW NEXT NFC NFD NFKC NFKD NO
%token <str> NONE NORMALIZE NORMALIZED
%token <str> NOT NOTHING NOTIFY NOTNULL NOWAIT NULL_P NULLIF
%token <str> NULLS_P NUMERIC
%token <str> OBJECT_P OF OFF OFFSET OIDS OLD OMIT ON ONLY OPERATOR OPTION OPTIONS OR
%token <str> ORDER ORDINALITY OTHERS OUT_P OUTER_P
%token <str> OVER OVERLAPS OVERLAY OVERRIDING OWNED OWNER
%token <str> PARALLEL PARAMETER PARSER PARTIAL PARTITION PASSING PASSWORD PATH
%token <str> PLACING PLAN PLANS POLICY
%token <str> POSITION PRECEDING PRECISION PRESERVE PREPARE PREPARED PRIMARY
%token <str> PRIOR PRIVILEGES PROCEDURAL PROCEDURE PROCEDURES PROGRAM PUBLICATION
%token <str> QUOTE QUOTES
%token <str> RANGE READ REAL REASSIGN RECHECK RECURSIVE REF_P REFERENCES REFERENCING
%token <str> REFRESH REINDEX RELATIVE_P RELEASE RENAME REPEATABLE REPLACE REPLICA
%token <str> RESET RESTART RESTRICT RETURN RETURNING RETURNS REVOKE RIGHT ROLE ROLLBACK ROLLUP
%token <str> ROUTINE ROUTINES ROW ROWS RULE
%token <str> SAVEPOINT SCALAR SCHEMA SCHEMAS SCROLL SEARCH SECOND_P SECURITY SELECT
%token <str> SEQUENCE SEQUENCES
%token <str> SERIALIZABLE SERVER SESSION SESSION_USER SET SETS SETOF SHARE SHOW
%token <str> SIMILAR SIMPLE SKIP SMALLINT SNAPSHOT SOME SOURCE SQL_P STABLE STANDALONE_P
%token <str> START STATEMENT STATISTICS STDIN STDOUT STORAGE STORED STRICT_P STRING_P STRIP_P
%token <str> SUBSCRIPTION SUBSTRING SUPPORT SYMMETRIC SYSID SYSTEM_P SYSTEM_USER
%token <str> TABLE TABLES TABLESAMPLE TABLESPACE TARGET TEMP TEMPLATE TEMPORARY TEXT_P THEN
%token <str> TIES TIME TIMESTAMP TO TRAILING TRANSACTION TRANSFORM
%token <str> TREAT TRIGGER TRIM TRUE_P
%token <str> TRUNCATE TRUSTED TYPE_P TYPES_P
%token <str> UESCAPE UNBOUNDED UNCONDITIONAL UNCOMMITTED UNENCRYPTED UNION UNIQUE UNKNOWN
%token <str> UNLISTEN UNLOGGED UNTIL UPDATE USER USING
%token <str> VACUUM VALID VALIDATE VALIDATOR VALUE_P VALUES VARCHAR VARIADIC VARYING
%token <str> VERBOSE VERSION_P VIEW VIEWS VOLATILE
%token <str> WHEN WHERE WHITESPACE_P WINDOW WITH WITHIN WITHOUT WORK WRAPPER WRITE
%token <str> XML_P XMLATTRIBUTES XMLCONCAT XMLELEMENT XMLEXISTS XMLFOREST XMLNAMESPACES
%token <str> XMLPARSE XMLPI XMLROOT XMLSERIALIZE XMLTABLE
%token <str> YEAR_P YES_P
%token <str> ZONE

// Non-terminals with types
%start Start

%type <list>  Start ResourceGroupOptions ResourceGroupOptionList OptIndexName
%type <list>  NumericOnlyList
%type <node>  Statement SelectStmt SimpleSelect SelectClause
%type <node>  ResourceGroupStmt ResourceGroupOption DropServerStmt PartitionBoundSpecClause PartitionBoundExpr
%type <node>  JSONAggregateFunc XMLTable XMLTableColumn XMLTableColumnOption XMLNamespace JSONTableExpr JSONTableColumn JSONTablePathOpt
%type <list>  XMLTableColumnList XMLTableColumnOptionList XMLNamespaceList JSONTableColumnList JSONArrayAggregateOrderByClauseOpt
%type <str>   JSONTablePathNameOpt PathOpt
%type <node>  SelectWithParens SelectNoParens
%type <expr>  AExpr BExpr CExpr ColumnRef AExprConst FuncExpr FuncApplication FuncExprCommonSubexpr
%type <node>  Field
%type <expr>  WhereClauseOpt WhereClause
%type <list>  FunctionName
%type <froms>  TableRefsClause
%type <node>  TableRef TableName JoinTable
%type <node>  FuncTable TablesampleClause
%type <boolean>  OptOrdinality ResourceGroupType OptForce
%type <boolean>  OptTable
%type <expr>  OptRepeatableClause
%type <jtype>  JoinType
%type <list>  StatementList
%type <fields>  FieldListOpt
%type <fields>  FieldList
%type <froms>  TableRefs
%type <grpclause> GroupByClause
%type <list>  GroupByList
%type <node>  GroupByItem
%type <expr>  HavingClause
%type <node>  EmptyGroupingSet CubeClause RollupClause GroupingSetsClause
%type <node>  OrderBy OrderByOpt
%type <list>  ByList
%type <exprs>  ExpressionList
%type <list>  OptIndirection Indirection FuncArgList FuncArgListOpt
%type <exprs>  ExtractList OverlayList PositionList SubstrList TrimList
%type <str>   ExtractArg UnicodeNormalForm
%type <node>  ByItem IndirectionElem
%type <expr>  FuncArgExpr
%type <str>   Name ColId ColLabel AttrName
%type <alias>  TableAsNameOpt AliasClause FuncAliasClause
%type <str>   UnReservedKeyword ColNameKeyword TypeFuncNameKeyword ReservedKeyword
%type <list>  OptNameList NameList AnyName QualifiedName
%type <ival>  Order
%type <ival>  IConst
%type <str>   SConst
%type <boolean> OptAllClause
%type <ival>   SetQuantifier
%type <exprs>  DistinctClause
%type <exprs>
%type <node>  WithClause OptWithClause WithClauseItem
%type <ival>  OptMaterialized
%type <ctes>  WithList
%type <list>  WindowClauseOpt WindowDefinitionList
%type <node>  WindowDefinition WindowSpec OverClause OptWindowFrameClause
%type <frame>  WindowFrameExtent
%type <bound>  WindowFrameBound
%type <str>   OptExistingWindowName
%type <exprs>  OptPartitionClause
%type <exclusion> OptWindowFrameExclusion
%type <node>  WithinGroupClause
%type <expr>  FilterClause
%type <node>  FuncExprWindowless
%type <slimit> SelectLimit LimitClause OptSelectLimit
%type <list>  ForLockingClause OptForLockingClause ForLockingItems
%type <node>  ForLockingItem
%type <ival>  ForLockingStrength OptNowaitOrSkip
%type <list>  LockedRelsList
%type <expr>   OffsetClause SelectLimitValue SelectOffsetValue SelectFetchFirstValue
%type <ival>   SubType
%type <list>   SubqueryOp
%type <expr>   CaseExpr CaseDefault CaseArg
%type <node>   WhenClause
%type <list>   WhenClauseList
%type <tp>  Typename SimpleTypename GenericType Numeric OptFloat Character
%type <tp>  ConstDatetime ConstInterval Bit BitWithLength BitWithoutLength
%type <tp>  ConstTypename ConstBit ConstCharacter
%type <boolean>  OptTimezone
%type <list>  OptInterval
%type <list>  IntervalSecond
%type <exprs>      OptTypeModifiers
%type <list>      OptArrayBounds
%type <str>       TypeFunctionName
%type <boolean>   OptVarying
%type <expr>      ArrayExpr OptSliceBound
%type <exprs>      ArrayExprList
%type <expr>      Row ExplicitRow
%type <exprs>      ImplicitRow
%type <node>  InsertStmt InsertRest
%type <node>  InsertTarget
%type <exprs>   InsertColumnList
%type <fields>  ReturningOpt
%type <node>  InsertColumnItem
%type <node>  OptOnConflict
%type <node>  ValuesClause
%type <assignments>  AssignmentList
%type <list>
%type <node>  Assignment
%type <node>  IntoClause OptTempTableName
%type <list>
%type <node>
%type <node>  OptSearchClause OptCycleClause
%type <joinqual> JoinQual
%type <node>
%type <node>  UpdateStmt
%type <node>  DeleteStmt
%type <froms>  UsingClause
%type <node>  RelationExprOptAlias
%type <node>  CreateTableStmt
%type <ival>  OptTemp
%type <list>  OptTableElementList TableElementList OptTypedTableElementList TypedTableElementList
%type <node>  TypedTableElement ColumnOptions
%type <node>  TableElement ColumnDef TableConstraint TableLikeClause
%type <ival>  TableLikeOptionList TableLikeOption
%type <colopts>  ColConstraintList OptColumnConstraints
%type <node>  ColConstraint ColConstraintElem ConstraintAttr ConstraintElem
%type <node>  DomainConstraint DomainConstraintElem
%type <list>  OptColumnList ColumnList
%type <partspec>  OptPartitionSpec PartitionSpecRule
%type <list>  OptWith
%type <ival>  OnCommitOption
%type <ident> OptTablespace
%type <str>   OptAccessMethod
%type <list>  PartParams OptInherit OptCollate
%type <node>  PartElem
%type <partbound>  PartitionBoundSpec ForValues
%type <node>  AlterTableStmt RenameStmt DropStmt
%type <list>  AlterTableCmds
%type <node>  AlterTableCmd
%type <ival>  OptDropBehavior
%type <ival>  ObjectTypeAnyName
%type <list>  AnyNameList
%type <node>  IndexStmt
%type <boolean>  OptUnique OptConcurrently
%type <str>   OptSingleName
%type <ident> AccessMethodClause
%type <str>   OptName
%type <list>  IndexParams OptInclude IndexIncludingParams
%type <node>  IndexElem
%type <ival>  OptNullsOrder
%type <node>  CreateViewStmt
%type <ival>  OptCheckOption
%type <node>  CreateFunctionStmt
%type <boolean>  OptOrReplace
%type <list>  FuncArgsWithDefaults FuncArgsWithDefaultsList
%type <node>  FuncArgWithDefault FuncArg
%type <list>  TableFuncColumnList
%type <node>  TableFuncColumn
%type <str>   ParamName
%type <ival>  ArgClass
%type <tp>  FuncReturn FuncType
%type <list>  CreateFuncOptList OptCreateFuncOptList
%type <node>  CreateFuncOptItem CommonFuncOptItem
%type <node>  OptRoutineBody ReturnStmt RoutineBodyStmt
%type <list>  RoutineBodyStmtList
%type <node>  TransactionStmt
%type <boolean>  OptTransactionChain
%type <node>  ExplainStmt ExplainableStmt
%type <node>  CopyStmt
%type <boolean>  CopyFrom OptProgram
%type <strp>  CopyFileName
%type <list>  UtilityOptionList CopyOptList CopyOptions TransactionModeList TransactionModeListOrEmpty
%type <node>  CopyOptItem OptBinary CopyDelimiter TransactionModeItem
%type <str>   IsoLevel
%type <node>  UtilityOptionElem
%type <str>   UtilityOptionName
%type <node>  UtilityOptionArg
%type <list>  UtilityOptionArgList
%type <str>   OptBooleanOrString
%type <node>  NumericOnly
%type <node>  GrantStmt RevokeStmt PrivilegeTarget
%type <list>  Privileges PrivilegeList GranteeList
%type <node>  Privilege Grantee RoleSpec
%type <boolean> OptGrantGrantOption
%type <node>  CheckPointStmt DiscardStmt ListenStmt UnlistenStmt NotifyStmt LoadStmt ClosePortalStmt ConstraintsSetStmt
%type <list>  ConstraintsSetList
%type <boolean> ConstraintsSetMode
%type <str>   FileName CursorName
%type <list>  QualifiedNameList
%type <node>  VariableSetStmt VariableShowStmt VariableResetStmt
%type <node>  PreparedStmt ExecuteStmt DeallocateStmt
%type <node>  SetRest SetRestMore GenericSet
%type <str>   VarName
%type <list>  VarList
%type <node>  VarValue ZoneValue
%type <str>   NonReservedWordOrSConst NonReservedWord
%type <ival>  DocumentOrContent
%type <tps>  PrepTypeClause
%type <exprs>  ExecuteParamClause
%type <str>   OptEncoding
%type <node>  ResetRest GenericReset
%type <tps>  TypeList
%type <ival>  SignedIconst
%type <node>  PreparableStmt
%type <node>  TruncateTableStmt LockStmt VacuumStmt AnalyzeStmt ClusterStmt ReindexStmt CommentStmt SecLabelStmt
%type <boolean> OptRestartSequences OptNowait OptVerbose OptAnalyze OptFull OptFreeze
%type <list>  RelationExprList OptVacuumRelationList VacuumRelationList
%type <node>  VacuumRelation
%type <ival>  OptLock LockType ReindexTargetType ReindexTargetMultitable OptNoLog
%type <str>   ClusterIndexSpecification ColumnCompression CommentText OptProvider
%type <ival>  ObjectTypeName ObjectTypeNameOnAnyName
%type <list>  OptReindexOptionList
%type <node>  DeclareCursorStmt FetchStmt MergeStmt CallStmt DoStmt
%type <ival>  CursorOptions OptHold
%type <node>  FetchArgs
%type <node>  MergeWhenClause
%type <list>  MergeWhenList
%type <node>  MergeUpdate MergeDelete MergeInsert
%type <expr>  OptMergeWhenCondition
%type <exprs>  MergeValuesClause
%type <ival>  MergeWhenTgtMatched MergeWhenTgtNotMatched
%type <ival>  OverrideKind
%type <list>  DoStmtOptList
%type <node>  DoStmtOptItem
%type <node>  CreateRoleStmt AlterRoleStmt AlterRoleSetStmt DropRoleStmt
%type <node>  CreateUserStmt CreateGroupStmt AlterGroupStmt
%type <node>  GrantRoleStmt RevokeRoleStmt
%type <list>  OptRoleList AlterOptRoleList
%type <node>  CreateOptRoleElem AlterOptRoleElem
%type <str>   RoleId
%type <list>  RoleList
%type <node>  SetResetClause
%type <node>  OptGrantedBy
%type <list>  GrantRoleOptList
%type <node>  GrantRoleOpt
%type <node>  GrantRoleOptValue
%type <ival>  AddDrop
%type <node>  CreateDatabaseStmt AlterDatabaseStmt AlterDatabaseSetStmt DropDatabaseStmt AlterSystemStmt
%type <list>  CreateDatabaseOptList CreateDatabaseOptItems
%type <node>  CreateDatabaseOptItem
%type <str>   CreateDatabaseOptName
%type <list>  DropOptionList
%type <node>  DropOption
%type <node>  CreateSchemaStmt CreateSequenceStmt AlterSequenceStmt
%type <node>  CreateDomainStmt AlterDomainStmt AlterEnumStmt AlterCollationStmt AlterCompositeTypeStmt
%type <list>  OptSchemaEltList OptParenthesizedSequenceOptList OptSequenceOptList SequenceOptList
%type <list>  AlterIdentityColumnOptionList
%type <node>  AlterIdentityColumnOption
%type <ival>  GeneratedWhen
%type <node>  SequenceOptElem
%type <node>  SchemaStmt
%type <list>  AlterTypeCmds
%type <node>  AlterTypeCmd
%type <node>  AlterColumnDefault
%type <boolean>  OptIfNotExists
%type <node>  TableFuncElement RowsFromItem
%type <node>  OptCollateClause
%type <node>  DefineStmt CompositeTypeStmt CreateEnumStmt CreateRangeStmt
%type <list>  Definition DefList OldAggrDefinition OldAggrList
%type <node>  DefElem DefArg OldAggrElem
%type <list>  OptTableFuncElementList TableFuncElementList OptColDefList
%type <list>  OptEnumValList EnumValList
%type <list>  AggrArgs AggrArgsList
%type <node>  AggrArg
%type <list>  AnyOperator QualAllOp QualOp
%type <str>   AllOp MathOp
%type <node>  AlterFunctionStmt RemoveFuncStmt RemoveAggrStmt RemoveOperStmt
%type <node>  CreateTriggerStmt CreateEventTrigStmt AlterEventTrigStmt
%type <node>  RuleStmt CreatePLangStmt
%type <list>  AlterFuncOptList
%type <list>  FunctionWithArgTypesList FuncArgs FuncArgsList RowsFromList
%type <node>  FunctionWithArgTypes
%type <list>  AggregateWithArgTypesList
%type <node>  AggregateWithArgTypes
%type <list>  OperatorWithArgTypesList OperArgTypes
%type <node>  OperatorWithArgTypes
%type <ival>  TriggerActionTime
%type <list>  TriggerEvents TriggerOneEvent
%type <boolean>  TriggerForSpec TriggerForType
%type <node>  TriggerWhen
%type <list>  TriggerFuncArgs
%type <node>  TriggerFuncArg
%type <list>  TriggerReferencing TriggerTransitions
%type <node>  TriggerTransition
%type <boolean>  TransitionOldOrNew TransitionRowOrTable
%type <str>   TransitionRelName
%type <list>  EventTriggerWhenList
%type <node>  EventTriggerWhenItem
%type <list>  EventTriggerValueList
%type <ival>  EnableTrigger
%type <list>  RuleActionList RuleActionMulti
%type <node>  RuleActionStmt RuleActionStmtOrEmpty
%type <ival>  Event
%type <boolean>  OptInstead OptTrusted
%type <list>  HandlerName OptInlineHandler OptValidator
%type <ival>  ConstraintAttributeSpec ConstraintAttributeElem
%type <ival>  KeyMatch
%type <keyaction>  KeyAction
%type <keyaction>  KeyUpdate KeyDelete
%type <keyactions> KeyActions
%type <list>  OptCInclude ExclusionConstraintList
%type <list>  ExclusionConstraintElem
%type <ident> OptConsTablespace ExistingIndex
%type <list>  HashPartbound
%type <node>  HashPartboundElem
%type <node>  OptConstrFromTable
%type <node>  XMLRootVersion OptXMLRootStandalone
%type <node>  XMLExistsArgument
%type <node>
%type <list>  XMLAttributeList XMLAttributes
%type <list>
%type <node> XMLAttributeElem
%type <ival>  XMLWhitespaceOption XMLIndentOption

%type <node>  JSONValueExpr JSONFormatClause JSONFormatClauseOpt
%type <node>  JSONReturningClauseOpt
%type <node>  JSONBehavior JSONBehaviorClauseOpt JSONTableOnErrorOpt
%type <node>
%type <list>  JSONNameAndValueList JSONValueExprList
%type <list>  JSONArguments JSONPassingClauseOpt
%type <list>
%type <list>
%type <node>  JSONNameAndValue JSONArgument
%type <node>
%type <ival>  JSONBehaviorType
%type <ival>  JSONPredicateTypeConstraint
%type <ival>  JSONWrapperBehavior JSONQuotesClauseOpt
%type <ival>  JSONKeyUniquenessConstraintOpt
%type <ival>  JSONObjectConstructorNullClauseOpt JSONArrayConstructorNullClauseOpt
%type <str>
%type <list>
%type <list>  Attrs
%type <node>  CreateFdwStmt AlterFdwStmt
%type <node>  CreateForeignServerStmt AlterForeignServerStmt
%type <node>  CreateForeignTableStmt
%type <node>  CreateUserMappingStmt AlterUserMappingStmt DropUserMappingStmt
%type <node>  ImportForeignSchemaStmt
%type <list>  CreateGenericOptions AlterGenericOptions
%type <list>  GenericOptionList AlterGenericOptionList
%type <node>  GenericOptionElem AlterGenericOptionElem
%type <str>   GenericOptionName
%type <node>  GenericOptionArg
%type <node>  FDWOption
%type <list>  FDWOptions OptFDWOptions
%type <str>   OptType ForeignServerVersion OptForeignServerVersion
%type <node>  AuthIdent
%type <ival>  ImportQualificationType
%type <iqual>  ImportQualification
%type <node>  CreateExtensionStmt AlterExtensionStmt AlterExtensionContentsStmt
%type <list>  CreateExtensionOptList AlterExtensionOptList
%type <node>  CreateExtensionOptItem AlterExtensionOptItem
%type <node>  CreateTablespaceStmt DropTablespaceStmt
%type <node>  OptTablespaceOwner
%type <node>  AlterTablespaceStmt
%type <node>  CreateAmStmt
%type <ival>  AMType
%type <node>  CreatePolicyStmt AlterPolicyStmt
%type <boolean>  RowSecurityDefaultPermissive
%type <str>   RowSecurityDefaultForCmd RowSecurityCmd
%type <list>  RowSecurityDefaultToRole RowSecurityOptToRole
%type <node>  RowSecurityOptExpr RowSecurityOptWithCheck
%type <node>  CreatePublicationStmt AlterPublicationStmt
%type <list>  OptDefinition
%type <node>  PublicationObjSpec
%type <list>  PubObjList
%type <list>  RelOptions OptRelOptions RelOptionList
%type <node>  RelOptionElem
%type <node>  CreateSubscriptionStmt AlterSubscriptionStmt DropSubscriptionStmt
%type <list>  OptWhereClause
%type <node>  AlterObjectSchemaStmt AlterOwnerStmt AlterObjectDependsStmt
%type <node>  AlterOperatorStmt AlterTypeStmt
%type <node>  AlterDefaultPrivilegesStmt
%type <node>  AlterTSConfigurationStmt AlterTSDictionaryStmt
%type <node>  CreateStatsStmt AlterStatsStmt
%type <node>  CreateOpClassStmt CreateOpFamilyStmt AlterOpFamilyStmt
%type <node>  DropOpClassStmt DropOpFamilyStmt
%type <node>  CreateCastStmt DropCastStmt
%type <node>  CreateTransformStmt DropTransformStmt
%type <node>  CreateConversionStmt
%type <node>  DropOwnedStmt ReassignOwnedStmt
%type <node>  CreateAsStmt CreateMatViewStmt RefreshMatViewStmt
%type <node>  DefACLAction
%type <list>  DefACLOptionList
%type <node>  DefACLOption
%type <ival>  DefaclPrivilegeTarget
%type <list>  OperatorDefList
%type <node>  OperatorDefElem
%type <node>  OperatorDefArg
%type <list>  StatsParams
%type <node>  StatsParam
%type <list>  OpClassItemList OpClassDropList
%type <node>  OpClassItem OpClassDrop
%type <boolean>  OptDefault
%type <list>  OptOpFamily OpClassPurpose
%type <boolean>  OptRecheck
%type <ival>  CastContext
%type <boolean>  OptIfExists
%type <list>  TransformElementList TransformTypeList FuncAs
%type <boolean>  OptNo
%type <node>  CreateAsTarget CreateMVTarget
%type <boolean> OptWithData
%type <boolean> NoInherit
%type <boolean> OptUniqueNullTreatment
%type <list>  OptQualifiedName
%type <node>  SetStatisticsValue
%type <list>  OptStatNameList

// Operator precedence - must match PostgreSQL's gram.y
// Lower precedence listed first

%left      UNION EXCEPT
%left      INTERSECT
%left      OR
%left      AND
%right     NOT
%nonassoc  IS ISNULL NOTNULL
%nonassoc  '<' '>' '=' LESS_EQUALS GREATER_EQUALS NOT_EQUALS
%nonassoc  BETWEEN IN_P LIKE ILIKE SIMILAR NOT_LA
%nonassoc  ESCAPE
%nonassoc  UNBOUNDED
%nonassoc  IDENT PARTITION RANGE ROWS GROUPS PRECEDING FOLLOWING CUBE ROLLUP
%nonassoc  SET KEYS OBJECT_P SCALAR VALUE_P WITH WITHOUT PATH
%left      Op OPERATOR
%left      '+' '-'
%left      '*' '/' '%'
%left      '^'
%left      AT
%left      COLLATE
%right     UMINUS
%left      '[' ']'
%left      '(' ')'
%left      TYPECAST
%left      '.'
%left      JOIN CROSS LEFT FULL RIGHT INNER_P NATURAL

%%

// Top-level rule
Start:
	StatementList
		{
			setParseResult(yylex, $1)
		}
	;

StatementList:
	StatementList ';' Statement
		{
			if $3 != nil {
				$3.SetSpan(yyS[yypt].span)
				$$ = appendList($1, $3)
			} else {
				$$ = $1
			}
		}
	| Statement
		{
			if $1 != nil {
				$1.SetSpan(yyS[yypt].span)
				$$ = makeList($1)
			} else {
				$$ = nil
			}
		}
	;

Statement:
	SelectStmt
		{
			$$ = $1
		}
	| InsertStmt
		{
			$$ = $1
		}
	| UpdateStmt
		{
			$$ = $1
		}
	| DeleteStmt
		{
			$$ = $1
		}
	| CreateTableStmt
		{
			$$ = $1
		}
	| AlterTableStmt
		{
			$$ = $1
		}
	| RenameStmt
		{
			$$ = $1
		}
	| DropStmt
		{
			$$ = $1
		}
	| IndexStmt
		{
			$$ = $1
		}
	| CreateViewStmt
		{
			$$ = $1
		}
	| CreateFunctionStmt
		{
			$$ = $1
		}
	| TransactionStmt
		{
			$$ = $1
		}
	| ExplainStmt
		{
			$$ = $1
		}
	| CopyStmt
		{
			$$ = $1
		}
	| GrantStmt
		{
			$$ = $1
		}
	| RevokeStmt
		{
			$$ = $1
		}
	| CheckPointStmt
		{
			$$ = $1
		}
	| DiscardStmt
		{
			$$ = $1
		}
	| ListenStmt
		{
			$$ = $1
		}
	| UnlistenStmt
		{
			$$ = $1
		}
	| NotifyStmt
		{
			$$ = $1
		}
	| LoadStmt
		{
			$$ = $1
		}
	| ClosePortalStmt
		{
			$$ = $1
		}
	| ConstraintsSetStmt
		{
			$$ = $1
		}
	| VariableSetStmt
		{
			$$ = $1
		}
	| VariableShowStmt
		{
			$$ = $1
		}
	| VariableResetStmt
		{
			$$ = $1
		}
	| PreparedStmt
		{
			$$ = $1
		}
	| ExecuteStmt
		{
			$$ = $1
		}
	| DeallocateStmt
		{
			$$ = $1
		}
	| TruncateTableStmt
		{
			$$ = $1
		}
	| LockStmt
		{
			$$ = $1
		}
	| VacuumStmt
		{
			$$ = $1
		}
	| AnalyzeStmt
		{
			$$ = $1
		}
	| ClusterStmt
		{
			$$ = $1
		}
	| ReindexStmt
		{
			$$ = $1
		}
	| CommentStmt
		{
			$$ = $1
		}
	| SecLabelStmt
		{
			$$ = $1
		}
	| DeclareCursorStmt
		{
			$$ = $1
		}
	| FetchStmt
		{
			$$ = $1
		}
	| MergeStmt
		{
			$$ = $1
		}
	| CallStmt
		{
			$$ = $1
		}
	| DoStmt
		{
			$$ = $1
		}
	| CreateRoleStmt
		{
			$$ = $1
		}
	| AlterRoleStmt
		{
			$$ = $1
		}
	| AlterRoleSetStmt
		{
			$$ = $1
		}
	| DropRoleStmt
		{
			$$ = $1
		}
	| CreateUserStmt
		{
			$$ = $1
		}
	| CreateGroupStmt
		{
			$$ = $1
		}
	| AlterGroupStmt
		{
			$$ = $1
		}
	| GrantRoleStmt
		{
			$$ = $1
		}
	| RevokeRoleStmt
		{
			$$ = $1
		}
	| CreateDatabaseStmt
		{
			$$ = $1
		}
	| AlterDatabaseStmt
		{
			$$ = $1
		}
	| AlterDatabaseSetStmt
		{
			$$ = $1
		}
	| DropDatabaseStmt
		{
			$$ = $1
		}
	| AlterSystemStmt
		{
			$$ = $1
		}
	| CreateSchemaStmt
		{
			$$ = $1
		}
	| CreateSequenceStmt
		{
			$$ = $1
		}
	| AlterSequenceStmt
		{
			$$ = $1
		}
	| CreateDomainStmt
		{
			$$ = $1
		}
	| AlterDomainStmt
		{
			$$ = $1
		}
	| AlterEnumStmt
		{
			$$ = $1
		}
	| AlterCollationStmt
		{
			$$ = $1
		}
	| AlterCompositeTypeStmt
		{
			$$ = $1
		}
	| DefineStmt
		{
			$$ = $1
		}
	| CompositeTypeStmt
		{
			$$ = $1
		}
	| CreateEnumStmt
		{
			$$ = $1
		}
	| CreateRangeStmt
		{
			$$ = $1
		}
	| AlterFunctionStmt
		{
			$$ = $1
		}
	| RemoveFuncStmt
		{
			$$ = $1
		}
	| RemoveAggrStmt
		{
			$$ = $1
		}
	| RemoveOperStmt
		{
			$$ = $1
		}
	| CreateTriggerStmt
		{
			$$ = $1
		}
	| CreateEventTrigStmt
		{
			$$ = $1
		}
	| AlterEventTrigStmt
		{
			$$ = $1
		}
	| RuleStmt
		{
			$$ = $1
		}
	| CreatePLangStmt
		{
			$$ = $1
		}
	| CreateFdwStmt
		{
			$$ = $1
		}
	| AlterFdwStmt
		{
			$$ = $1
		}
	| CreateForeignServerStmt
		{
			$$ = $1
		}
	| AlterForeignServerStmt
		{
			$$ = $1
		}
	| CreateForeignTableStmt
		{
			$$ = $1
		}
	| CreateUserMappingStmt
		{
			$$ = $1
		}
	| AlterUserMappingStmt
		{
			$$ = $1
		}
	| DropUserMappingStmt
		{
			$$ = $1
		}
	| ImportForeignSchemaStmt
		{
			$$ = $1
		}
	| CreateExtensionStmt
		{
			$$ = $1
		}
	| AlterExtensionStmt
		{
			$$ = $1
		}
	| AlterExtensionContentsStmt
		{
			$$ = $1
		}
	| CreateTablespaceStmt
		{
			$$ = $1
		}
	| DropTablespaceStmt
		{
			$$ = $1
		}
	| AlterTablespaceStmt
		{
			$$ = $1
		}
	| CreateAmStmt
		{
			$$ = $1
		}
	| CreatePolicyStmt
		{
			$$ = $1
		}
	| AlterPolicyStmt
		{
			$$ = $1
		}
	| CreatePublicationStmt
		{
			$$ = $1
		}
	| AlterPublicationStmt
		{
			$$ = $1
		}
	| CreateSubscriptionStmt
		{
			$$ = $1
		}
	| AlterSubscriptionStmt
		{
			$$ = $1
		}
	| DropSubscriptionStmt
		{
			$$ = $1
		}
	| AlterObjectSchemaStmt
		{
			$$ = $1
		}
	| AlterOwnerStmt
		{
			$$ = $1
		}
	| AlterObjectDependsStmt
		{
			$$ = $1
		}
	| AlterOperatorStmt
		{
			$$ = $1
		}
	| AlterTypeStmt
		{
			$$ = $1
		}
	| AlterDefaultPrivilegesStmt
		{
			$$ = $1
		}
	| AlterTSConfigurationStmt
		{
			$$ = $1
		}
	| AlterTSDictionaryStmt
		{
			$$ = $1
		}
	| CreateStatsStmt
		{
			$$ = $1
		}
	| AlterStatsStmt
		{
			$$ = $1
		}
	| CreateOpClassStmt
		{
			$$ = $1
		}
	| CreateOpFamilyStmt
		{
			$$ = $1
		}
	| AlterOpFamilyStmt
		{
			$$ = $1
		}
	| DropOpClassStmt
		{
			$$ = $1
		}
	| DropOpFamilyStmt
		{
			$$ = $1
		}
	| CreateCastStmt
		{
			$$ = $1
		}
	| DropCastStmt
		{
			$$ = $1
		}
	| CreateTransformStmt
		{
			$$ = $1
		}
	| DropTransformStmt
		{
			$$ = $1
		}
	| CreateConversionStmt
		{
			$$ = $1
		}
	| DropOwnedStmt
		{
			$$ = $1
		}
	| ReassignOwnedStmt
		{
			$$ = $1
		}
	| CreateAsStmt
		{
			$$ = $1
		}
	| CreateMatViewStmt
		{
			$$ = $1
		}
	| RefreshMatViewStmt
		{
			$$ = $1
		}
	| ResourceGroupStmt
		{
			$$ = $1
		}
	| DropServerStmt
		{
			$$ = $1
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

/*****************************************************************************
 *
 *      INSERT statement
 *
 *****************************************************************************/

InsertStmt:
	OptWithClause INSERT INTO InsertTarget InsertRest OptOnConflict ReturningOpt
		{
			n := $5.(*ast.InsertStmt)
			if target, ok := $4.(*ast.TableSource); ok {
				name, _ := target.Source.(*ast.TableName)
				n.Table = name
				n.TargetAlias = target.Alias
			}
			if vals, ok := n.Query.(*ast.ValuesStmt); ok {
				for _, row := range vals.Rows {
					n.Values = append(n.Values, row.Items)
				}
				n.Query = nil
			}
			if $6 != nil {
				n.OnConflict = $6.(*ast.OnConflictClause)
			}
			n.Returning = returningFieldList($7)
			if $1 != nil {
				n.With = $1.(*ast.WithClause)
			}
			n.SetSpan(yyVAL.span)
			$$ = n
		}
	;

InsertTarget:
	QualifiedName
		{
			$$ = makeTableSource(yylex, $1, yyVAL.span)
		}
	| QualifiedName AS ColId
		{
			ts := makeTableSource(yylex, $1, yyVAL.span)
			ts.Alias = syntaxIdent(yylex, $3, yyS[yypt].span)
			$$ = ts
		}
	;

InsertRest:
	SelectStmt
		{
			$$ = &ast.InsertStmt{BaseInsertStmt: pkast.BaseInsertStmt{Query: $1.(ast.QueryNode)}}
		}
	| OVERRIDING OverrideKind VALUE_P SelectStmt
		{
			$$ = &ast.InsertStmt{BaseInsertStmt: pkast.BaseInsertStmt{Query: $4.(ast.QueryNode)}}
		}
	| '(' InsertColumnList ')' SelectStmt
		{
			$$ = &ast.InsertStmt{BaseInsertStmt: pkast.BaseInsertStmt{
				Columns: $2,
				Query:   $4.(ast.QueryNode),
			}}
		}
	| '(' InsertColumnList ')' OVERRIDING OverrideKind VALUE_P SelectStmt
		{
			$$ = &ast.InsertStmt{BaseInsertStmt: pkast.BaseInsertStmt{
				Columns: $2,
				Query:   $7.(ast.QueryNode),
			}}
		}
	| DEFAULT VALUES
		{
			$$ = &ast.InsertStmt{}
		}
	;

InsertColumnList:
	InsertColumnItem
		{ $$ = []ast.ExprNode{$1.(ast.ExprNode)} }
	| InsertColumnList ',' InsertColumnItem
		{ $$ = append($1, $3.(ast.ExprNode)) }
	;

InsertColumnItem:
	ColId OptIndirection
		{
			if $2 == nil {
				$$ = ast.NewColumnNameExpr("", "", $1)
			} else {
				$$ = &ast.A_Indirection{Arg: ast.NewColumnNameExpr("", "", $1), Indirection: $2}
			}
		}
	;

OptOnConflict:
	ON CONFLICT DO NOTHING
		{
			$$ = &ast.OnConflictClause{
				DoNothing: true,
			}
		}
	| ON CONFLICT DO UPDATE SET AssignmentList WhereClauseOpt
		{
			$$ = &ast.OnConflictClause{
				UpdateList: $6,
				UpdateWhere: $7,
			}
		}
	| ON CONFLICT '(' IndexParams ')' DO NOTHING
		{
			$$ = &ast.OnConflictClause{
				IndexCols: indexElemColNames($4),
				DoNothing: true,
			}
		}
	| ON CONFLICT '(' IndexParams ')' DO UPDATE SET AssignmentList WhereClauseOpt
		{
			$$ = &ast.OnConflictClause{
				IndexCols:  indexElemColNames($4),
				UpdateList: $9,
				UpdateWhere: $10,
			}
		}
	| ON CONFLICT ON CONSTRAINT Name DO NOTHING
		{
			$$ = &ast.OnConflictClause{
				Constraint: $5,
				DoNothing:  true,
			}
		}
	| ON CONFLICT ON CONSTRAINT Name DO UPDATE SET AssignmentList WhereClauseOpt
		{
			$$ = &ast.OnConflictClause{
				Constraint: $5,
				UpdateList: $9,
				UpdateWhere: $10,
			}
		}
	| ON CONFLICT '(' IndexParams ')' WhereClauseOpt DO NOTHING
		{
			$$ = &ast.OnConflictClause{
				IndexCols:  indexElemColNames($4),
				TargetWhere: $6,
				DoNothing:  true,
			}
		}
	| ON CONFLICT '(' IndexParams ')' WhereClauseOpt DO UPDATE SET AssignmentList WhereClauseOpt
		{
			$$ = &ast.OnConflictClause{
				IndexCols:  indexElemColNames($4),
				TargetWhere: $6,
				UpdateList: $10,
				UpdateWhere: $11,
			}
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

ReturningOpt:
	RETURNING FieldList
		{
			$$ = $2
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

AssignmentList:
	Assignment
		{
			$$ = []*ast.Assignment{$1.(*ast.Assignment)}
		}
	| AssignmentList ',' Assignment
		{
			$$ = append($1, $3.(*ast.Assignment))
		}
	;

Assignment:
	'(' ColumnList ')' '=' AExpr
		{
			$$ = &ast.Assignment{
				Target: &ast.RowTarget{Columns: columnNameListFromStrings($2)},
				Expr:   $5,
			}
		}
	| ColId '=' AExpr
		{
			$$ = &ast.Assignment{
				Target: ast.NewColumnNameExpr("", "", $1),
				Expr:   $3,
			}
		}
	| ColId Indirection '=' AExpr
		{
			$$ = &ast.Assignment{
				Target: &ast.A_Indirection{Arg: ast.NewColumnNameExpr("", "", $1), Indirection: $2},
				Expr:   $4,
			}
		}
	;



/*****************************************************************************
 *
 *      UPDATE statement
 *
 *****************************************************************************/

UpdateStmt:
	OptWithClause UPDATE RelationExprOptAlias SET AssignmentList TableRefsClause WhereClause ReturningOpt
		{
			n := &ast.UpdateStmt{BaseUpdateStmt: pkast.BaseUpdateStmt{
				Table:       $3.(*ast.TableSource),
				Assignments: $5,
			}}
			if $6 != nil {
				n.From = fromClauseOfNodes($6)
			}
			if $7 != nil {
				n.Where = $7.(ast.ExprNode)
			}
			n.Returning = returningFieldList($8)
			if $1 != nil {
				n.With = $1.(*ast.WithClause)
			}
			$$ = n
		}
	;

/*****************************************************************************
 *
 *      DELETE statement
 *
 *****************************************************************************/

DeleteStmt:
	OptWithClause DELETE_P FROM RelationExprOptAlias UsingClause WhereClause ReturningOpt
		{
			n := &ast.DeleteStmt{BaseDeleteStmt: pkast.BaseDeleteStmt{
				Table: $4.(*ast.TableSource),
			}}
			if $5 != nil {
				n.Using = fromRelationOfNodes($5)
			}
			if $6 != nil {
				n.Where = $6.(ast.ExprNode)
			}
			n.Returning = returningFieldList($7)
			if $1 != nil {
				n.With = $1.(*ast.WithClause)
			}
			$$ = n
		}
	;

UsingClause:
	USING TableRefs { $$ = $2 }
	| /* EMPTY */ { $$ = nil }
	;

RelationExprOptAlias:
	TableName  %prec UMINUS
		{ $$ = $1 }
	| TableName ColId
		{
			$1.(*ast.TableSource).Alias = syntaxIdent(yylex, $2, yyS[yypt].span)
			$1.SetSpan(yyVAL.span)
			$$ = $1
		}
	| TableName AS ColId
		{
			$1.(*ast.TableSource).Alias = syntaxIdent(yylex, $3, yyS[yypt].span)
			$1.SetSpan(yyVAL.span)
			$$ = $1
		}
	;

/*****************************************************************************
 *
 *      CREATE TABLE statement
 *
 *****************************************************************************/

PartitionBoundSpecClause:
	FOR VALUES PartitionBoundExpr
		{
			$$ = $3
		}
	| DEFAULT
		{
			$$ = &ast.PartitionBoundSpec{
				BoundType: "DEFAULT",
			}
		}
	;

PartitionBoundExpr:
	IN_P '(' ExpressionList ')'
		{
			$$ = &ast.PartitionBoundSpec{
				BoundType: "IN",
				In:        $3,
			}
		}
	| FROM '(' ExpressionList ')' TO '(' ExpressionList ')'
		{
			$$ = &ast.PartitionBoundSpec{
				BoundType: "FROM",
				From:      $3,
				To:        $7,
			}
		}
	| WITH '(' ColId IConst ',' ColId IConst ')'
		{
			$$ = &ast.PartitionBoundSpec{
				BoundType: "WITH",
				Options: &ast.List{Items: []ast.Node{
					&ast.DefElem{Defname: $3, Arg: &ast.Integer{Ival: $4}},
					&ast.DefElem{Defname: $6, Arg: &ast.Integer{Ival: $7}},
				}},
			}
		}
	;

XMLTable:
	XMLTABLE '(' CExpr XMLExistsArgument COLUMNS XMLTableColumnList ')'
		{
			$$ = &ast.XMLTableExpr{
				RowExpr: exprOrNil($3),
				DocExpr: exprOrNil($4),
				Columns: xmlTableColumns($6),
			}
		}
	| XMLTABLE '(' XMLNAMESPACES '(' XMLNamespaceList ')' ','
		CExpr XMLExistsArgument COLUMNS XMLTableColumnList ')'
		{
			$$ = &ast.XMLTableExpr{
				RowExpr:    exprOrNil($8),
				DocExpr:    exprOrNil($9),
				Columns:    xmlTableColumns($11),
				Namespaces: xmlNamespaces($5),
			}
		}
	;

XMLTableColumnList:
	XMLTableColumn
		{ $$ = makeList($1) }
	| XMLTableColumnList ',' XMLTableColumn
		{ $$ = appendList($1, $3) }
	;

XMLTableColumn:
	ColId Typename
		{
			$$ = &ast.XMLTableColumn{
				Name:     $1,
				TypeName: $2,
			}
		}
	| ColId Typename XMLTableColumnOptionList
		{
			col := &ast.XMLTableColumn{
				Name:     $1,
				TypeName: $2,
			}
			applyXMLTableColumnOptions(col, $3)
			$$ = col
		}
	| ColId FOR ORDINALITY
		{
			$$ = &ast.XMLTableColumn{
				Name:         $1,
				ForOrdinality: true,
			}
		}
	;

XMLTableColumnOptionList:
	XMLTableColumnOption
		{ $$ = makeList($1) }
	| XMLTableColumnOptionList XMLTableColumnOption
		{ $$ = appendList($1, $2) }
	;

XMLTableColumnOption:
	IDENT BExpr
		{
			$$ = makeDefElem($1, $2)
		}
	| DEFAULT BExpr
		{
			$$ = makeDefElem("default", $2)
		}
	| NOT NULL_P
		{
			$$ = makeDefElem("__pg__is_not_null", ast.NewValueExpr(true))
		}
	| NULL_P
		{
			$$ = makeDefElem("__pg__is_not_null", ast.NewValueExpr(false))
		}
	| PATH BExpr
		{
			$$ = makeDefElem("path", $2)
		}
	;

XMLNamespaceList:
	XMLNamespace
		{ $$ = makeList($1) }
	| XMLNamespaceList ',' XMLNamespace
		{ $$ = appendList($1, $3) }
	;

XMLNamespace:
	BExpr AS ColLabel
		{
			$$ = &ast.XMLTableNamespace{
				Expr: exprOrNil($1),
				Name: $3,
			}
		}
	| DEFAULT BExpr
		{
			$$ = &ast.XMLTableNamespace{
				Expr: exprOrNil($2),
			}
		}
	;

JSONTableExpr:
	JSON_TABLE '('
		JSONValueExpr ',' AExpr JSONTablePathNameOpt
		JSONPassingClauseOpt
		COLUMNS '(' JSONTableColumnList ')'
		JSONTableOnErrorOpt
	')'
		{
			$$ = &ast.JSONTableExpr{
				ContextItem: exprOrNil($3),
				PathSpec: &ast.JSONTablePathSpec{
					Expr: $5,
					Name:   $6,
				},
				Passing: nodesOfList($7),
				Columns: jsonTableColumns($10),
				OnError: asJsonBehavior($12),
			}
		}
	;

JSONTablePathNameOpt:
	AS Name
		{
			$$ = $2
		}
	| /* EMPTY */
		{
			$$ = ""
		}
	;

JSONTableColumnList:
	JSONTableColumn
		{ $$ = makeList($1) }
	| JSONTableColumnList ',' JSONTableColumn
		{ $$ = appendList($1, $3) }
	;

JSONTableColumn:
	ColId FOR ORDINALITY
		{
			$$ = &ast.JSONTableColumn{
				ColType: "ordinality",
				Name:    $1,
			}
		}
	| ColId Typename JSONTablePathOpt
		JSONWrapperBehavior JSONQuotesClauseOpt
		JSONBehaviorClauseOpt
		{
			col := &ast.JSONTableColumn{
				ColType:  "regular",
				Name:     $1,
				TypeName: $2,
				PathSpec: jsonTablePathSpec($3),
				Wrapper:  jsonWrapperName($4),
				Quotes:   jsonQuotesName($5),
			}
			col.OnEmpty, col.OnError = splitJsonBehaviorClause($6)
			$$ = col
		}
	| ColId Typename JSONFormatClause JSONTablePathOpt
		JSONWrapperBehavior JSONQuotesClauseOpt
		JSONBehaviorClauseOpt
		{
			col := &ast.JSONTableColumn{
				ColType:  "formatted",
				Name:     $1,
				TypeName: $2,
				Format:   $3.(*ast.JsonFormat),
				PathSpec: jsonTablePathSpec($4),
				Wrapper:  jsonWrapperName($5),
				Quotes:   jsonQuotesName($6),
			}
			col.OnEmpty, col.OnError = splitJsonBehaviorClause($7)
			$$ = col
		}
	| ColId Typename EXISTS JSONTablePathOpt
		JSONTableOnErrorOpt
		{
			$$ = &ast.JSONTableColumn{
				ColType:  "exists",
				Name:     $1,
				TypeName: $2,
				PathSpec: jsonTablePathSpec($4),
				OnError:  asJsonBehavior($5),
			}
		}
	| NESTED PathOpt SConst JSONTablePathNameOpt
		COLUMNS '(' JSONTableColumnList ')'
		{
			$$ = &ast.JSONTableColumn{
				ColType: "nested",
				PathSpec: &ast.JSONTablePathSpec{
					Expr: ast.NewValueExpr($3),
					Name:   $4,
				},
				Columns: jsonTableColumns($7),
			}
		}
	;

JSONTablePathOpt:
	PATH SConst
		{
			$$ = &ast.JSONTablePathSpec{
				Expr: ast.NewValueExpr($2),
			}
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

PathOpt:
	PATH
		{
			$$ = ""
		}
	| /* EMPTY */
		{
			$$ = ""
		}
	;

CreateTableStmt:
	CREATE OptTemp TABLE QualifiedName '(' OptTableElementList ')' OptInherit OptPartitionSpec OptAccessMethod OptWith OnCommitOption OptTablespace
		{
			table := syntaxTableName(syntaxName(syntaxIdents(yylex, $4)...))
			table.Relpersistence = relpersistenceForTemp($2)
			cols, constraints, likes := tableElements($6)
			$$ = &ast.CreateTableStmt{
				Temporary:   $2 == 1,
				Table:       table,
				Cols:        cols,
				Constraints: constraints,
				LikeClauses: likes,
				Inherits:    inheritTableNames($8),
				PartitionBy: $9,
				AccessMethod: $10,
				WithOptions: $11,
				OnCommit: ast.OnCommitAction($12),
				Tablespace:  $13,
			}
		}
	| CREATE OptTemp TABLE IF_P NOT EXISTS QualifiedName '(' OptTableElementList ')' OptInherit OptPartitionSpec OptAccessMethod OptWith OnCommitOption OptTablespace
		{
			table := syntaxTableName(syntaxName(syntaxIdents(yylex, $7)...))
			table.Relpersistence = relpersistenceForTemp($2)
			cols, constraints, likes := tableElements($9)
			$$ = &ast.CreateTableStmt{
				Temporary:   $2 == 1,
				IfNotExists: true,
				Table:       table,
				Cols:        cols,
				Constraints: constraints,
				LikeClauses: likes,
				Inherits:    inheritTableNames($11),
				PartitionBy: $12,
				AccessMethod: $13,
				WithOptions: $14,
				OnCommit: ast.OnCommitAction($15),
				Tablespace:  $16,
			}
		}
	| CREATE OptTemp TABLE QualifiedName OF AnyName OptTypedTableElementList OptPartitionSpec
		{
			table := syntaxTableName(syntaxName(syntaxIdents(yylex, $4)...))
			table.Relpersistence = relpersistenceForTemp($2)
			cols, constraints, likes := tableElements($7)
			$$ = &ast.CreateTableStmt{
				Temporary:   $2 == 1,
				Table:       table,
				ReferTable:  syntaxTableName(syntaxName(syntaxIdents(yylex, $6)...)),
				Cols:        cols,
				Constraints: constraints,
				LikeClauses: likes,
				PartitionBy: $8,
			}
		}
	| CREATE OptTemp TABLE IF_P NOT EXISTS QualifiedName OF AnyName OptTypedTableElementList OptPartitionSpec
		{
			table := syntaxTableName(syntaxName(syntaxIdents(yylex, $7)...))
			table.Relpersistence = relpersistenceForTemp($2)
			cols, constraints, likes := tableElements($10)
			$$ = &ast.CreateTableStmt{
				Temporary:   $2 == 1,
				IfNotExists: true,
				Table:       table,
				ReferTable:  syntaxTableName(syntaxName(syntaxIdents(yylex, $9)...)),
				Cols:        cols,
				Constraints: constraints,
				LikeClauses: likes,
				PartitionBy: $11,
			}
		}
	| CREATE OptTemp TABLE QualifiedName PARTITION OF QualifiedName OptTypedTableElementList PartitionBoundSpecClause OptPartitionSpec OptAccessMethod OptWith OnCommitOption OptTablespace
		{
			table := syntaxTableName(syntaxName(syntaxIdents(yylex, $4)...))
			table.Relpersistence = relpersistenceForTemp($2)
			cols, constraints, likes := tableElements($8)
			$$ = &ast.CreateTableStmt{
				Temporary:      $2 == 1,
				Table:          table,
				ReferTable:     syntaxTableName(syntaxName(syntaxIdents(yylex, $7)...)),
				Cols:           cols,
				Constraints:    constraints,
				LikeClauses: likes,
				PartitionBound: $9.(*ast.PartitionBoundSpec),
				PartitionBy:    $10,
				AccessMethod: $11,
				WithOptions:    $12,
				OnCommit:       ast.OnCommitAction($13),
				Tablespace:     $14,
			}
		}

	;

OptTemp:
	TEMPORARY       { $$ = 1 }
	| TEMP          { $$ = 1 }
	| LOCAL TEMPORARY { $$ = 1 }
	| LOCAL TEMP    { $$ = 1 }
	| UNLOGGED      { $$ = 2 }
	| /* EMPTY */   { $$ = 0 }
	;

OptInherit:
	INHERITS '(' QualifiedNameList ')'	{ $$ = $3 }
	| /* EMPTY */							{ $$ = nil }
	;

OptPartitionSpec:
	PartitionSpecRule	{ $$ = $1 }
	| /* EMPTY */		{ $$ = nil }
	;

OptAccessMethod:
	USING Name  { $$ = $2 }
	| /* EMPTY */ { $$ = "" }
	;

OptWith:
	WITH RelOptions				{ $$ = $2 }
	| WITH OIDS					{ $$ = nil /* deprecated */ }
	| WITHOUT OIDS				{ $$ = nil }
	| /* EMPTY */				{ $$ = nil }
	;

OnCommitOption:
	ON COMMIT DROP				{ $$ = int64(ast.ONCOMMIT_DROP) }
	| ON COMMIT DELETE_P ROWS	{ $$ = int64(ast.ONCOMMIT_DELETE_ROWS) }
	| ON COMMIT PRESERVE ROWS	{ $$ = int64(ast.ONCOMMIT_PRESERVE_ROWS) }
	| /* EMPTY */				{ $$ = int64(ast.ONCOMMIT_NOOP) }
	;

OptTablespace:
	TABLESPACE Name				{ $$ = syntaxIdent(yylex, $2, yyS[yypt].span) }
	| /* EMPTY */				{ $$ = lex.Ident{Span: loc.InvalidSpan()} }
	;

PartitionSpecRule:
	PARTITION BY ColId '(' PartParams ')'
		{
			$$ = &ast.PartitionSpec{
				Strategy:   parsePartitionStrategy($3),
				PartParams: $5,
				Location:   -1,
			}
		}
	;

PartParams:
	PartElem
		{ $$ = makeList($1) }
	| PartParams ',' PartElem
		{ $$ = appendList($1, $3) }
	;

PartElem:
	ColId OptCollate OptQualifiedName
		{
			$$ = &ast.PartitionElem{
				Name:      $1,
				Collation: $2,
				Opclass:   $3,
				Location:  -1,
			}
		}
	| FuncExprWindowless OptCollate OptQualifiedName
		{
			$$ = &ast.PartitionElem{
				Expr:      $1,
				Collation: $2,
				Opclass:   $3,
				Location:  -1,
			}
		}
	| '(' AExpr ')' OptCollate OptQualifiedName
		{
			$$ = &ast.PartitionElem{
				Expr:      $2,
				Collation: $4,
				Opclass:   $5,
				Location:  -1,
			}
		}
	;

OptCollate:
	COLLATE AnyName	{ $$ = $2 }
	| /* EMPTY */		{ $$ = nil }
	;

ForValues:
	PartitionBoundSpec	{ $$ = $1 }
	| DEFAULT
		{
			$$ = &ast.PartitionBoundSpec{
				BoundType: "default",
			}
		}
	;

PartitionBoundSpec:
	/* a LIST partition */
	FOR VALUES IN_P '(' ExpressionList ')'
		{
			$$ = &ast.PartitionBoundSpec{
				BoundType: "IN",
				In:        $5,
			}
		}
	/* a RANGE partition */
	| FOR VALUES FROM '(' ExpressionList ')' TO '(' ExpressionList ')'
		{
			$$ = &ast.PartitionBoundSpec{
				BoundType: "FROM",
				From:      $5,
				To:        $9,
			}
		}
	/* a HASH partition */
	| FOR VALUES WITH '(' HashPartbound ')'
		{
			$$ = &ast.PartitionBoundSpec{BoundType: "WITH", Options: $5}
		}
	;

HashPartbound:
	HashPartboundElem
		{
			$$ = makeList($1)
		}
	| HashPartbound ',' HashPartboundElem
		{
			$$ = appendList($1, $3)
		}
	;

HashPartboundElem:
	NonReservedWord IConst
		{
			$$ = makeDefElem($1, &ast.Integer{Ival: $2})
		}
	;

OptTableElementList:
	TableElementList { $$ = $1 }
	| /* EMPTY */ { $$ = nil }
	;

OptTypedTableElementList:
	'(' TypedTableElementList ')' { $$ = $2 }
	| /* EMPTY */ { $$ = nil }
	;

TypedTableElementList:
	TypedTableElement
		{ $$ = makeList($1) }
	| TypedTableElementList ',' TypedTableElement
		{ $$ = appendList($1, $3) }
	;

TypedTableElement:
	ColumnOptions { $$ = $1 }
	| TableConstraint { $$ = $1 }
	;

ColumnOptions:
	ColId OptColumnConstraints
		{
			$$ = &ast.ColumnDef{
				Name:    &ast.ColumnName{Name: model.NewCIStr($1)},
				Options: $2,
			}
		}
	| ColId WITH OPTIONS OptColumnConstraints
		{
			$$ = &ast.ColumnDef{
				Name:    &ast.ColumnName{Name: model.NewCIStr($1)},
				Options: $4,
			}
		}
	;

TableElementList:
	TableElement
		{ $$ = makeList($1) }
	| TableElementList ',' TableElement
		{ $$ = appendList($1, $3) }
	;

TableElement:
	ColumnDef { $$ = $1 }
	| TableConstraint { $$ = $1 }
	| TableLikeClause { $$ = $1 }
	;

TableLikeClause:
	LIKE QualifiedName
		{
			$$ = &ast.TableLikeClause{
				Relation: relationName(makeRangeVar($2)),
			}
		}
	| LIKE QualifiedName TableLikeOptionList
		{
			$$ = &ast.TableLikeClause{
				Relation: relationName(makeRangeVar($2)),
				Options:  uint32($3),
			}
		}
	;

TableLikeOptionList:
	TableLikeOptionList INCLUDING TableLikeOption
		{ $$ = $1 | $3 }
	| TableLikeOptionList EXCLUDING TableLikeOption
		{ $$ = $1 &^ $3 }
	| INCLUDING TableLikeOption
		{ $$ = $2 }
	| EXCLUDING TableLikeOption
		{ $$ = 0 &^ $2 }
	;

TableLikeOption:
	ALL				{ $$ = 0xFFFFFFFF }
	| COMMENTS		{ $$ = 1 }
	| COMPRESSION	{ $$ = 2 }
	| CONSTRAINTS	{ $$ = 4 }
	| DEFAULTS		{ $$ = 8 }
	| GENERATED		{ $$ = 16 }
	| IDENTITY_P	{ $$ = 32 }
	| INDEXES		{ $$ = 64 }
	| STATISTICS	{ $$ = 128 }
	| STORAGE		{ $$ = 256 }
	;

ColumnDef:
	ColId Typename OptColumnConstraints
		{
			$$ = &ast.ColumnDef{
				Name:    &ast.ColumnName{Name: model.NewCIStr($1)},
				Tp:      $2,
				Options: $3,
			}
		}
	;

OptColumnConstraints:
	ColConstraintList { $$ = $1 }
	| /* EMPTY */ { $$ = nil }
	;

ColConstraintList:
	ColConstraint
		{ $$ = []*ast.ColumnOption{$1.(*ast.ColumnOption)} }
	| ColConstraintList ColConstraint
		{ $$ = append($1, $2.(*ast.ColumnOption)) }
	;

ColConstraint:
	CONSTRAINT Name ColConstraintElem
		{
			$$ = $3
		}
	| ColConstraintElem { $$ = $1 }
	| COLLATE AnyName
		{
			$$ = &ast.ColumnOption{
				Tp:       ast.ColumnOptionCollate,
				StrValue: nameListString($2),
			}
		}
	| COMPRESSION ColId
		{
			$$ = &ast.ColumnOption{
				Tp:       ast.ColumnOptionCompression,
				StrValue: $2,
			}
		}
	| STORAGE ColId
		{
			$$ = &ast.ColumnOption{
				Tp:       ast.ColumnOptionStorage,
				StrValue: $2,
			}
		}
	| OPTIONS '(' GenericOptionList ')'
		{
			$$ = &ast.ColumnOption{
				Tp:      ast.ColumnOptionForeignOptions,
				Options: $3,
			}
		}
	| ConstraintAttr
		{ $$ = $1 }
	;

ConstraintAttr:
	DEFERRABLE
		{
			$$ = &ast.ColumnOption{Tp: ast.ColumnOptionNoOption, StrValue: "deferrable"}
		}
	| NOT DEFERRABLE
		{
			$$ = &ast.ColumnOption{Tp: ast.ColumnOptionNoOption, StrValue: "not deferrable"}
		}
	| INITIALLY DEFERRED
		{
			$$ = &ast.ColumnOption{Tp: ast.ColumnOptionNoOption, StrValue: "initially deferred"}
		}
	| INITIALLY IMMEDIATE
		{
			$$ = &ast.ColumnOption{Tp: ast.ColumnOptionNoOption, StrValue: "initially immediate"}
		}
	;

ColConstraintElem:
	NOT NULL_P
		{
			$$ = &ast.ColumnOption{Tp: ast.ColumnOptionNotNull}
		}
	| NULL_P
		{
			$$ = &ast.ColumnOption{Tp: ast.ColumnOptionNull}
		}
	| UNIQUE OptUniqueNullTreatment OptDefinition OptConsTablespace
		{
			$$ = &ast.ColumnOption{Tp: ast.ColumnOptionUniqKey, NullsNotDistinct: $2}
		}
	| PRIMARY KEY OptDefinition OptConsTablespace
		{
			$$ = &ast.ColumnOption{Tp: ast.ColumnOptionPrimaryKey}
		}
	| CHECK '(' AExpr ')' NoInherit
		{
			$$ = &ast.ColumnOption{Tp: ast.ColumnOptionCheck, Expr: $3}
		}
	| DEFAULT BExpr
		{
			$$ = &ast.ColumnOption{Tp: ast.ColumnOptionDefaultValue, Expr: $2}
		}
	| REFERENCES QualifiedName OptColumnList KeyMatch KeyActions ConstraintAttributeSpec
		{
			$$ = &ast.ColumnOption{
				Tp: ast.ColumnOptionReference,
				Refer: makeReferenceDef($2, $3, $4, $5),
			}
		}
	| GENERATED ALWAYS AS IDENTITY_P OptParenthesizedSequenceOptList
		{
			$$ = &ast.ColumnOption{
				Tp:              ast.ColumnOptionIdentity,
				GeneratedWhen:   byte('a'),
				SequenceOptions: $5,
			}
		}
	| GENERATED BY DEFAULT AS IDENTITY_P OptParenthesizedSequenceOptList
		{
			$$ = &ast.ColumnOption{
				Tp:              ast.ColumnOptionIdentity,
				GeneratedWhen:   byte('d'),
				SequenceOptions: $6,
			}
		}
	| GENERATED ALWAYS AS '(' AExpr ')' STORED
		{
			$$ = &ast.ColumnOption{Tp: ast.ColumnOptionGenerated, Expr: $5, Stored: true}
		}
	| GENERATED BY DEFAULT AS '(' AExpr ')' STORED
		{
			$$ = &ast.ColumnOption{Tp: ast.ColumnOptionGenerated, Expr: $6, Stored: true}
		}
	;

DomainConstraint:
	CONSTRAINT Name DomainConstraintElem
		{
			n := $3.(*ast.Constraint)
			n.Name = syntaxIdent(yylex, $2, yyS[yypt-1].span)
			$$ = n
		}
	| DomainConstraintElem
		{
			$$ = $1
		}
	;

DomainConstraintElem:
	CHECK '(' AExpr ')' ConstraintAttributeSpec
		{
			n := &ast.Constraint{
				Tp:             ast.ConstraintCheck,
				Expr:           $3,
				InitiallyValid: true,
			}
			applyConstraintAttrs(n, $5)
			$$ = n
		}
	| NOT NULL_P ConstraintAttributeSpec
		{
			n := &ast.Constraint{
				Tp:             ast.ConstraintNoConstraint,
				InitiallyValid: true,
			}
			applyConstraintAttrs(n, $3)
			$$ = n
		}
	;

TableConstraint:
	CONSTRAINT Name ConstraintElem
		{
			n := $3.(*ast.Constraint)
			n.Name = syntaxIdent(yylex, $2, yyS[yypt-1].span)
			$$ = n
		}
	| ConstraintElem
		{
			$$ = $1
		}
	;

ConstraintElem:
	UNIQUE OptUniqueNullTreatment '(' ColumnList ')' OptCInclude OptDefinition OptConsTablespace ConstraintAttributeSpec
		{
			n := &ast.Constraint{
				Tp:              ast.ConstraintUniq,
				Keys:            makeIndexColNameList($4),
				InitiallyValid:  true,
				NullsNotDistinct: $2,
				Including: syntaxIdents(yylex, $6),
				IndexSpace: $8,
			}
			applyConstraintAttrs(n, $9)
			$$ = n
		}
	| UNIQUE ExistingIndex ConstraintAttributeSpec
		{
			n := &ast.Constraint{
				Tp:             ast.ConstraintUniq,
				IndexName:      $2,
				InitiallyValid: true,
			}
			applyConstraintAttrs(n, $3)
			$$ = n
		}
	| PRIMARY KEY '(' ColumnList ')' OptCInclude OptDefinition OptConsTablespace ConstraintAttributeSpec
		{
			n := &ast.Constraint{
				Tp:             ast.ConstraintPrimaryKey,
				Keys:           makeIndexColNameList($4),
				Including:      syntaxIdents(yylex, $6),
				IndexSpace:     $8,
				InitiallyValid: true,
			}
			applyConstraintAttrs(n, $9)
			$$ = n
		}
	| PRIMARY KEY ExistingIndex ConstraintAttributeSpec
		{
			n := &ast.Constraint{
				Tp:             ast.ConstraintPrimaryKey,
				IndexName:      $3,
				InitiallyValid: true,
			}
			applyConstraintAttrs(n, $4)
			$$ = n
		}
	| CHECK '(' AExpr ')' ConstraintAttributeSpec
		{
			n := &ast.Constraint{
				Tp:             ast.ConstraintCheck,
				Expr:           $3,
				InitiallyValid: true,
			}
			applyConstraintAttrs(n, $5)
			$$ = n
		}
	| FOREIGN KEY '(' ColumnList ')' REFERENCES QualifiedName OptColumnList KeyMatch KeyActions ConstraintAttributeSpec
		{
			n := &ast.Constraint{
				Tp:             ast.ConstraintForeignKey,
				Keys:           makeIndexColNameList($4),
				InitiallyValid: true,
				Refer: makeReferenceDef($7, $8, $9, $10),
			}
			applyConstraintAttrs(n, $11)
			$$ = n
		}
	| EXCLUDE AccessMethodClause '(' ExclusionConstraintList ')' OptCInclude OptDefinition OptConsTablespace WhereClauseOpt ConstraintAttributeSpec
		{
			n := &ast.Constraint{
				Tp:             ast.ConstraintUniq,
				InitiallyValid: true,
			}
			applyConstraintAttrs(n, $10)
			$$ = n
		}
	;

OptColumnList:
	'(' ColumnList ')' { $$ = $2 }
	| /* EMPTY */ { $$ = nil }
	;

ColumnList:
	Name
		{
			item := &ast.String{Str: $1}
			item.SetSpan(yyS[yypt].span)
			$$ = makeList(item)
		}
	| ColumnList ',' Name
		{
			item := &ast.String{Str: $3}
			item.SetSpan(yyS[yypt].span)
			$$ = appendList($1, item)
		}
	;

KeyMatch:
	MATCH FULL     { $$ = int64('f') }
	| MATCH PARTIAL
		{
			/* PARTIAL is not implemented */
			$$ = int64('p')
		}
	| MATCH SIMPLE { $$ = int64('s') }
	| /* EMPTY */  { $$ = int64('s') }
	;

KeyActions:
	KeyUpdate
		{ $$ = &KeyActions{UpdateAction: $1} }
	| KeyDelete
		{ $$ = &KeyActions{DeleteAction: $1} }
	| KeyUpdate KeyDelete
		{ $$ = &KeyActions{UpdateAction: $1, DeleteAction: $2} }
	| KeyDelete KeyUpdate
		{ $$ = &KeyActions{UpdateAction: $2, DeleteAction: $1} }
	| /* EMPTY */
		{ $$ = nil }
	;

KeyUpdate:
	ON UPDATE KeyAction {
		if $3.Cols != nil { yylex.Error("ON UPDATE cannot specify columns") }
		$$ = $3
	}
	;

KeyDelete:
	ON DELETE_P KeyAction { $$ = $3 }
	;

KeyAction:
	NO ACTION                     { $$ = &KeyAction{Action: 'a'} }
	| RESTRICT                    { $$ = &KeyAction{Action: 'r'} }
	| CASCADE                     { $$ = &KeyAction{Action: 'c'} }
	| SET NULL_P OptColumnList  { $$ = &KeyAction{Action: 'n', Cols: $3} }
	| SET DEFAULT OptColumnList { $$ = &KeyAction{Action: 'd', Cols: $3} }
	;

OptCInclude:
	INCLUDE '(' ColumnList ')'  { $$ = $3 }
	| /* EMPTY */               { $$ = nil }
	;

OptConsTablespace:
	USING INDEX TABLESPACE Name { $$ = syntaxIdent(yylex, $4, yyS[yypt].span) }
	| /* EMPTY */               { $$ = lex.Ident{Span: loc.InvalidSpan()} }
	;

ExistingIndex:
	USING INDEX Name { $$ = syntaxIdent(yylex, $3, yyS[yypt].span) }
	;

ExclusionConstraintList:
	ExclusionConstraintElem
		{ $$ = makeList($1) }
	| ExclusionConstraintList ',' ExclusionConstraintElem
		{ $$ = appendList($1, $3) }
	;

ExclusionConstraintElem:
	IndexElem WITH AnyOperator
		{
			$$ = &ast.List{Items: []ast.Node{$1, $3}}
		}
	| IndexElem WITH OPERATOR '(' AnyOperator ')'
		{
			$$ = &ast.List{Items: []ast.Node{$1, $5}}
		}
	;

/*****************************************************************************
 *
 *      ALTER TABLE statement
 *
 *****************************************************************************/

AlterTableStmt:
	ALTER TABLE TableName AlterTableCmds
		{
			$$ = &ast.AlterTableStmt{
				Table: relationName($3),
				Only: $3.(*ast.TableSource).Only,
				Specs: alterTableSpecs($4),
			}
		}
	| ALTER TABLE IF_P EXISTS TableName AlterTableCmds
		{
			$$ = &ast.AlterTableStmt{
				Table: relationName($5),
				Only: $5.(*ast.TableSource).Only,
				Specs: alterTableSpecs($6),
				MissingOk: true,
			}
		}
	| ALTER INDEX QualifiedName AlterTableCmds
		{
			rv := makeRangeVarFromAnyName($3)
			$$ = &ast.AlterTableStmt{
				Objtype: "INDEX",
				Table: relationName(rv),
				Specs: alterTableSpecs($4),
			}
		}
	| ALTER INDEX IF_P EXISTS QualifiedName AlterTableCmds
		{
			rv := makeRangeVarFromAnyName($5)
			$$ = &ast.AlterTableStmt{
				Objtype: "INDEX",
				Table: relationName(rv),
				Specs: alterTableSpecs($6),
				MissingOk: true,
			}
		}
	| ALTER INDEX QualifiedName ATTACH PARTITION QualifiedName
		{
			rvIdx := makeRangeVarFromAnyName($3)
			rvPart := makeRangeVar($6)
			$$ = &ast.AlterTableStmt{
				Objtype: "INDEX",
				Table: relationName(rvIdx),
				Specs: []*ast.AlterTableSpec{{
					Tp: ast.AlterTableAttachPartition,
					NewTable: relationName(rvPart),
				}},
			}
		}
	| ALTER SEQUENCE QualifiedName AlterTableCmds
		{
			rv := makeRangeVarFromAnyName($3)
			$$ = &ast.AlterTableStmt{
				Objtype: "SEQUENCE",
				Table: relationName(rv),
				Specs: alterTableSpecs($4),
			}
		}
	| ALTER SEQUENCE IF_P EXISTS QualifiedName AlterTableCmds
		{
			rv := makeRangeVarFromAnyName($5)
			$$ = &ast.AlterTableStmt{
				Objtype: "SEQUENCE",
				Table: relationName(rv),
				Specs: alterTableSpecs($6),
				MissingOk: true,
			}
		}
	| ALTER VIEW QualifiedName AlterTableCmds
		{
			rv := makeRangeVarFromAnyName($3)
			$$ = &ast.AlterTableStmt{
				Objtype: "VIEW",
				Table: relationName(rv),
				Specs: alterTableSpecs($4),
			}
		}
	| ALTER VIEW IF_P EXISTS QualifiedName AlterTableCmds
		{
			rv := makeRangeVarFromAnyName($5)
			$$ = &ast.AlterTableStmt{
				Objtype: "VIEW",
				Table: relationName(rv),
				Specs: alterTableSpecs($6),
				MissingOk: true,
			}
		}
	| ALTER MATERIALIZED VIEW QualifiedName AlterTableCmds
		{
			rv := makeRangeVarFromAnyName($4)
			$$ = &ast.AlterTableStmt{
				Objtype: "MATERIALIZED VIEW",
				Table: relationName(rv),
				Specs: alterTableSpecs($5),
			}
		}
	| ALTER MATERIALIZED VIEW IF_P EXISTS QualifiedName AlterTableCmds
		{
			rv := makeRangeVarFromAnyName($6)
			$$ = &ast.AlterTableStmt{
				Objtype: "MATERIALIZED VIEW",
				Table: relationName(rv),
				Specs: alterTableSpecs($7),
				MissingOk: true,
			}
		}
	| ALTER FOREIGN TABLE TableName AlterTableCmds
		{
			$$ = &ast.AlterTableStmt{
				Objtype: "FOREIGN TABLE",
				Table: relationName($4),
				Only: $4.(*ast.TableSource).Only,
				Specs: alterTableSpecs($5),
			}
		}
	| ALTER FOREIGN TABLE IF_P EXISTS TableName AlterTableCmds
		{
			$$ = &ast.AlterTableStmt{
				Objtype: "FOREIGN TABLE",
				Table: relationName($6),
				Only: $6.(*ast.TableSource).Only,
				Specs: alterTableSpecs($7),
				MissingOk: true,
			}
		}
	/* ALTER TABLE ALL IN TABLESPACE ... SET TABLESPACE ... */
	| ALTER TABLE ALL IN_P TABLESPACE Name SET TABLESPACE Name OptNowait
		{
			$$ = &ast.AlterTableMoveAllStmt{
				OrigTablespacename: $6,
				NewTablespacename:  $9,
				Nowait:             $10,
			}
		}
	| ALTER TABLE ALL IN_P TABLESPACE Name OWNED BY RoleList SET TABLESPACE Name OptNowait
		{
			$$ = &ast.AlterTableMoveAllStmt{
				OrigTablespacename: $6,
				Roles:              $9,
				NewTablespacename:  $12,
				Nowait:             $13,
			}
		}
	| ALTER INDEX ALL IN_P TABLESPACE Name SET TABLESPACE Name OptNowait
		{
			$$ = &ast.AlterTableMoveAllStmt{
				OrigTablespacename: $6,
				NewTablespacename:  $9,
				Nowait:             $10,
			}
		}
	| ALTER INDEX ALL IN_P TABLESPACE Name OWNED BY RoleList SET TABLESPACE Name OptNowait
		{
			$$ = &ast.AlterTableMoveAllStmt{
				OrigTablespacename: $6,
				Roles:              $9,
				NewTablespacename:  $12,
				Nowait:             $13,
			}
		}
	| ALTER MATERIALIZED VIEW ALL IN_P TABLESPACE Name SET TABLESPACE Name OptNowait
		{
			$$ = &ast.AlterTableMoveAllStmt{
				OrigTablespacename: $7,
				NewTablespacename:  $10,
				Nowait:             $11,
			}
		}
	| ALTER MATERIALIZED VIEW ALL IN_P TABLESPACE Name OWNED BY RoleList SET TABLESPACE Name OptNowait
		{
			$$ = &ast.AlterTableMoveAllStmt{
				OrigTablespacename: $7,
				Roles:              $10,
				NewTablespacename:  $13,
				Nowait:             $14,
			}
		}
	;

AlterTableCmds:
	AlterTableCmd
		{ $$ = makeList($1) }
	| AlterTableCmds ',' AlterTableCmd
		{ $$ = appendList($1, $3) }
	;

AlterTableCmd:
	ADD_P ColumnDef
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableAddColumns,
				NewColumns: []*ast.ColumnDef{$2.(*ast.ColumnDef)},
			}
		}
	| ADD_P COLUMN ColumnDef
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableAddColumns,
				NewColumns: []*ast.ColumnDef{$3.(*ast.ColumnDef)},
			}
		}
	| ADD_P COLUMN IF_P NOT EXISTS ColumnDef
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableAddColumns,
				NewColumns: []*ast.ColumnDef{$6.(*ast.ColumnDef)},
				MissingOk: true,
			}
		}
	| DROP COLUMN ColId OptDropBehavior
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableDropColumn,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				Behavior: ast.DropBehavior(int($4)),
			}
		}
	| DROP COLUMN IF_P EXISTS ColId OptDropBehavior
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableDropColumn,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($5)},
				Behavior: ast.DropBehavior(int($6)),
				MissingOk: true,
			}
		}
	| ALTER COLUMN ColId SET DEFAULT AExpr
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableModifyColumn,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				Def: $6,
			}
		}
	| ALTER COLUMN ColId DROP DEFAULT
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableDropDefault,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
			}
		}
	| ALTER COLUMN ColId SET NOT NULL_P
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableSetNotNull,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
			}
		}
	| ALTER COLUMN ColId DROP NOT NULL_P
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableDropNotNull,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
			}
		}
	| ALTER COLUMN ColId TYPE_P Typename OptCollateClause
		{
			coldef := &ast.ColumnDef{Tp: $5}
			if $6 != nil {
				coldef.Options = append(coldef.Options, &ast.ColumnOption{Tp: ast.ColumnOptionCollate, StrValue: nameListString($6)})
			}
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableAlterColumnType,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				NewColumns: []*ast.ColumnDef{coldef},
			}
		}
	| ADD_P TableConstraint
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableAddConstraint,
				Constraint: $2.(*ast.Constraint),
			}
		}
	| DROP CONSTRAINT Name OptDropBehavior
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
				Name: $3,
				Behavior: ast.DropBehavior(int($4)),
			}
		}
	| DROP CONSTRAINT IF_P EXISTS Name OptDropBehavior
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
				Name: $5,
				Behavior: ast.DropBehavior(int($6)),
				MissingOk: true,
			}
		}
	| OWNER TO RoleSpec
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableOwner,
				Owner: $3.(*ast.RoleSpec),
			}
		}
	/* ALTER COLUMN ... TYPE ... USING */
	| ALTER COLUMN ColId SET DATA_P TYPE_P Typename OptCollateClause
		{
			coldef := &ast.ColumnDef{Tp: $7}
			if $8 != nil {
				coldef.Options = append(coldef.Options, &ast.ColumnOption{Tp: ast.ColumnOptionCollate, StrValue: nameListString($8)})
			}
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableAlterColumnType,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				NewColumns: []*ast.ColumnDef{coldef},
			}
		}
	| ALTER COLUMN ColId TYPE_P Typename OptCollateClause USING AExpr
		{
			coldef := &ast.ColumnDef{Tp: $5}
			if $6 != nil {
				coldef.Options = append(coldef.Options, &ast.ColumnOption{Tp: ast.ColumnOptionCollate, StrValue: nameListString($6)})
			}
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableAlterColumnType,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				NewColumns: []*ast.ColumnDef{coldef},
				Def: $8,
			}
		}
	| ALTER COLUMN ColId SET DATA_P TYPE_P Typename OptCollateClause USING AExpr
		{
			coldef := &ast.ColumnDef{Tp: $7}
			if $8 != nil {
				coldef.Options = append(coldef.Options, &ast.ColumnOption{Tp: ast.ColumnOptionCollate, StrValue: nameListString($8)})
			}
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableAlterColumnType,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				NewColumns: []*ast.ColumnDef{coldef},
				Def: $10,
			}
		}
	| ALTER ColId TYPE_P Typename OptCollateClause
		{
			coldef := &ast.ColumnDef{Tp: $4}
			if $5 != nil {
				coldef.Options = append(coldef.Options, &ast.ColumnOption{Tp: ast.ColumnOptionCollate, StrValue: nameListString($5)})
			}
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableAlterColumnType,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($2)},
				NewColumns: []*ast.ColumnDef{coldef},
			}
		}
	| ALTER ColId SET DATA_P TYPE_P Typename OptCollateClause
		{
			coldef := &ast.ColumnDef{Tp: $6}
			if $7 != nil {
				coldef.Options = append(coldef.Options, &ast.ColumnOption{Tp: ast.ColumnOptionCollate, StrValue: nameListString($7)})
			}
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableAlterColumnType,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($2)},
				NewColumns: []*ast.ColumnDef{coldef},
			}
		}
	| ALTER ColId SET DATA_P TYPE_P Typename OptCollateClause USING AExpr
		{
			coldef := &ast.ColumnDef{Tp: $6}
			if $7 != nil {
				coldef.Options = append(coldef.Options, &ast.ColumnOption{Tp: ast.ColumnOptionCollate, StrValue: nameListString($7)})
			}
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableAlterColumnType,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($2)},
				NewColumns: []*ast.ColumnDef{coldef},
				Def: $9,
			}
		}
	| ALTER ColId SET DEFAULT AExpr
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableModifyColumn,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($2)},
				Def: $5,
			}
		}
	| ALTER ColId DROP DEFAULT
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableDropDefault,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($2)},
			}
		}
	| ALTER ColId SET NOT NULL_P
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableSetNotNull,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($2)},
			}
		}
	| ALTER ColId DROP NOT NULL_P
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableDropNotNull,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($2)},
			}
		}
	| ALTER ColId TYPE_P Typename OptCollateClause USING AExpr
		{
			coldef := &ast.ColumnDef{Tp: $4}
			if $5 != nil {
				coldef.Options = append(coldef.Options, &ast.ColumnOption{Tp: ast.ColumnOptionCollate, StrValue: nameListString($5)})
			}
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableAlterColumnType,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($2)},
				NewColumns: []*ast.ColumnDef{coldef},
				Def: $7,
			}
		}
	/* ALTER FOREIGN TABLE <name> ALTER [COLUMN] <colname> OPTIONS */
	| ALTER COLUMN ColId AlterGenericOptions
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableAlterColumn,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				Def: $4,
			}
		}
	| ALTER ColId AlterGenericOptions
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableAlterColumn,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($2)},
				Def: $3,
			}
		}
	/* VALIDATE CONSTRAINT */
	| VALIDATE CONSTRAINT Name
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableValidateConstraint,
				Name: $3,
			}
		}
	/* ALTER CONSTRAINT */
	| ALTER CONSTRAINT Name ConstraintAttributeSpec
		{
			$$ = &ast.AlterTableSpec{
				Tp:               ast.AlterTableAlterConstraint,
				Name:             $3,
				ConstraintOptions: constraintOptions($4),
			}
		}
	/* INHERIT / NO INHERIT */
	| INHERIT QualifiedName
		{
			$$ = &ast.AlterTableSpec{
				Tp:      ast.AlterTableInherit,
				NewTable: tableNameFromList($2),
			}
		}
	| NO INHERIT QualifiedName
		{
			$$ = &ast.AlterTableSpec{
				Tp:      ast.AlterTableNoInherit,
				NewTable: tableNameFromList($3),
			}
		}
	/* ATTACH / DETACH PARTITION */
	| ATTACH PARTITION QualifiedName ForValues
		{
			$$ = &ast.AlterTableSpec{
				Tp:            ast.AlterTableAttachPartition,
				NewTable:      tableNameFromList($3),
				PartitionBound: $4,
			}
		}
	| DETACH PARTITION QualifiedName
		{
			$$ = &ast.AlterTableSpec{
				Tp:       ast.AlterTableDetachPartition,
				NewTable: tableNameFromList($3),
			}
		}
	| DETACH PARTITION QualifiedName CONCURRENTLY
		{
			$$ = &ast.AlterTableSpec{
				Tp:       ast.AlterTableDetachPartition,
				NewTable: tableNameFromList($3),
				Concurrent: true,
			}
		}
	| DETACH PARTITION QualifiedName FINALIZE
		{
			$$ = &ast.AlterTableSpec{
				Tp:       ast.AlterTableDetachPartition,
				NewTable: tableNameFromList($3),
				Finalize: true,
			}
		}
	/* ENABLE/DISABLE TRIGGER */
	| ENABLE_P TRIGGER Name
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableEnableTrigger,
				Name: $3,
				TriggerMode: "DEFAULT",
			}
		}
	| ENABLE_P ALWAYS TRIGGER Name
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableEnableTrigger,
				Name: $4,
				TriggerMode: "ALWAYS",
			}
		}
	| ENABLE_P REPLICA TRIGGER Name
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableEnableTrigger,
				Name: $4,
				TriggerMode: "REPLICA",
			}
		}
	| DISABLE_P TRIGGER Name
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableDisableTrigger,
				Name: $3,
				TriggerMode: "DEFAULT",
			}
		}
	| ENABLE_P TRIGGER ALL
		{
			$$ = &ast.AlterTableSpec{
				Tp:         ast.AlterTableEnableTrigger,
				Name:       "ALL",
				TriggerMode: "ALL",
			}
		}
	| DISABLE_P TRIGGER ALL
		{
			$$ = &ast.AlterTableSpec{
				Tp:         ast.AlterTableDisableTrigger,
				Name:       "ALL",
				TriggerMode: "ALL",
			}
		}
	| ENABLE_P TRIGGER USER
		{
			$$ = &ast.AlterTableSpec{
				Tp:         ast.AlterTableEnableTrigger,
				Name:       "USER",
				TriggerMode: "USER",
			}
		}
	| DISABLE_P TRIGGER USER
		{
			$$ = &ast.AlterTableSpec{
				Tp:         ast.AlterTableDisableTrigger,
				Name:       "USER",
				TriggerMode: "USER",
			}
		}
	/* ENABLE/DISABLE RULE */
	| ENABLE_P RULE Name
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableEnableRule,
				Name: $3,
				TriggerMode: "RULE",
			}
		}
	| ENABLE_P ALWAYS RULE Name
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableEnableRule,
				Name: $4,
				TriggerMode: "ALWAYS",
			}
		}
	| ENABLE_P REPLICA RULE Name
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableEnableRule,
				Name: $4,
				TriggerMode: "REPLICA",
			}
		}
	| DISABLE_P RULE Name
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableDisableRule,
				Name: $3,
				TriggerMode: "RULE",
			}
		}
	/* ENABLE/DISABLE/FORCE ROW LEVEL SECURITY */
	| ENABLE_P ROW LEVEL SECURITY
		{
			$$ = &ast.AlterTableSpec{
				Tp:             ast.AlterTableRowLevelSecurity,
				RowLevelSecurity: "ENABLE",
			}
		}
	| DISABLE_P ROW LEVEL SECURITY
		{
			$$ = &ast.AlterTableSpec{
				Tp:             ast.AlterTableRowLevelSecurity,
				RowLevelSecurity: "DISABLE",
			}
		}
	| FORCE ROW LEVEL SECURITY
		{
			$$ = &ast.AlterTableSpec{
				Tp:             ast.AlterTableRowLevelSecurity,
				RowLevelSecurity: "FORCE",
			}
		}
	| NO FORCE ROW LEVEL SECURITY
		{
			$$ = &ast.AlterTableSpec{
				Tp:             ast.AlterTableRowLevelSecurity,
				RowLevelSecurity: "NO FORCE",
			}
		}
	/* CLUSTER ON / SET WITHOUT CLUSTER */
	| CLUSTER ON Name
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableSetCluster,
				Name: $3,
			}
		}
	| SET WITHOUT CLUSTER
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableSetWithoutCluster,
			}
		}
	/* SET LOGGED / SET UNLOGGED */
	| SET LOGGED
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableSetLogged,
			}
		}
	| SET UNLOGGED
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableSetUnlogged,
			}
		}
	/* SET ACCESS METHOD */
	| SET ACCESS METHOD Name
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
				Name: $4,
			}
		}
	| SET ACCESS METHOD DEFAULT
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
				Name: "",
			}
		}
	/* SET/RESET storage params */
	| SET RelOptions
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableSetRelOptions,
				RelOptions: $2,
			}
		}
	| RESET RelOptions
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableResetRelOptions,
				RelOptions: $2,
			}
		}
	/* ADD COLUMN IF NOT EXISTS (without COLUMN keyword) */
	| ADD_P IF_P NOT EXISTS ColumnDef
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableAddColumns,
				NewColumns: []*ast.ColumnDef{$5.(*ast.ColumnDef)},
				MissingOk: true,
			}
		}
	/* DROP (without COLUMN keyword) */
	| DROP ColId OptDropBehavior
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableDropColumn,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($2)},
				Behavior: ast.DropBehavior(int($3)),
			}
		}
	| DROP IF_P EXISTS ColId OptDropBehavior
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableDropColumn,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($4)},
				Behavior: ast.DropBehavior(int($5)),
				MissingOk: true,
			}
		}
	/* REPLICA IDENTITY */
	| REPLICA IDENTITY_P DEFAULT
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableReplicaIdentity,
				ReplicaIdentity: "DEFAULT",
			}
		}
	| REPLICA IDENTITY_P FULL
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableReplicaIdentity,
				ReplicaIdentity: "FULL",
			}
		}
	| REPLICA IDENTITY_P NOTHING
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableReplicaIdentity,
				ReplicaIdentity: "NOTHING",
			}
		}
	| REPLICA IDENTITY_P USING INDEX Name
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableReplicaIdentity,
				Name: $5,
				ReplicaIdentity: "USING INDEX",
			}
		}
	/* SET STORAGE */
	| ALTER COLUMN ColId SET STORAGE ColId
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableSetStorage,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				Storage: $6,
			}
		}
	| ALTER COLUMN ColId SET STORAGE DEFAULT
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableSetStorage,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				Storage: "default",
			}
		}
	| ALTER ColId SET STORAGE ColId
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableSetStorage,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($2)},
				Storage: $5,
			}
		}
	| ALTER ColId SET STORAGE DEFAULT
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableSetStorage,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($2)},
				Storage: "default",
			}
		}
	/* SET STATISTICS */
	| ALTER COLUMN ColId SET STATISTICS SignedIconst
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableSetStatistics,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				Num: $6,
			}
		}
	| ALTER ColId SET STATISTICS SignedIconst
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableSetStatistics,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($2)},
				Num: $5,
			}
		}
	| ALTER COLUMN IConst SET STATISTICS SignedIconst
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableSetStatistics,
				Num: $6,
				Def: makeIntConst($3),
			}
		}
	| ALTER IConst SET STATISTICS SignedIconst
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableSetStatistics,
				Num: $5,
				Def: makeIntConst($2),
			}
		}
	/* SET COMPRESSION */
	| ALTER COLUMN ColId SET ColumnCompression
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				Def: &ast.String{Str: $5},
			}
		}
	| ALTER ColId SET ColumnCompression
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($2)},
				Def: &ast.String{Str: $4},
			}
		}
	/* SET/DROP EXPRESSION */
	| ALTER COLUMN ColId SET EXPRESSION AS '(' AExpr ')'
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				Def: $8,
			}
		}
	| ALTER ColId SET EXPRESSION AS '(' AExpr ')'
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($2)},
				Def: $7,
			}
		}
	| ALTER COLUMN ColId DROP EXPRESSION
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
			}
		}
	| ALTER COLUMN ColId DROP EXPRESSION IF_P EXISTS
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				MissingOk: true,
			}
		}
	/* ADD GENERATED IDENTITY */
	| ALTER COLUMN ColId ADD_P GENERATED ALWAYS AS IDENTITY_P
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
			}
		}
	| ALTER COLUMN ColId ADD_P GENERATED ALWAYS AS IDENTITY_P '(' OptSequenceOptList ')'
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
			}
		}
	| ALTER COLUMN ColId ADD_P GENERATED BY DEFAULT AS IDENTITY_P
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
			}
		}
	| ALTER COLUMN ColId ADD_P GENERATED BY DEFAULT AS IDENTITY_P '(' OptSequenceOptList ')'
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
			}
		}
	/* DROP/SET IDENTITY */
	| ALTER COLUMN ColId DROP IDENTITY_P
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
			}
		}
	| ALTER COLUMN ColId DROP IDENTITY_P IF_P EXISTS
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				MissingOk: true,
			}
		}
	/* ADD GENERATED identity */
	| ALTER COLUMN ColId ADD_P GENERATED GeneratedWhen AS IDENTITY_P OptParenthesizedSequenceOptList
		{
			c := &ast.Constraint{
				Tp:             ast.ConstraintNoConstraint,
				GeneratedWhen: byte($6),
			}
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				Def: c,
			}
		}
	| ALTER ColId ADD_P GENERATED GeneratedWhen AS IDENTITY_P OptParenthesizedSequenceOptList
		{
			c := &ast.Constraint{
				Tp:             ast.ConstraintNoConstraint,
				GeneratedWhen: byte($5),
			}
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($2)},
				Def: c,
			}
		}
	/* SET GENERATED / identity options */
	| ALTER COLUMN ColId AlterIdentityColumnOptionList
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				Def: $4,
			}
		}
	/* ALTER COLUMN ... SET (options) / RESET (options) */
	| ALTER COLUMN ColId SET '(' DefList ')'
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableSetRelOptions,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				RelOptions: $6,
			}
		}
	| ALTER COLUMN ColId RESET '(' DefList ')'
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableResetRelOptions,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				RelOptions: $6,
			}
		}
	/* SET WITH/WITHOUT OIDS (deprecated but still parsed) */
	| SET WITH OIDS
		{
			$$ = &ast.AlterTableSpec{
				Tp:       ast.AlterTableSetOption,
				Name:     "with",
				SetValue: "oids",
			}
		}
	| SET WITHOUT OIDS
		{
			$$ = &ast.AlterTableSpec{
				Tp:       ast.AlterTableSetOption,
				Name:     "without",
				SetValue: "oids",
			}
		}
	/* SET TABLESPACE */
	| SET TABLESPACE Name
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableSetTablespace,
				Name: $3,
			}
		}
	/* OPTIONS for foreign tables */
	| AlterGenericOptions
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableForeignOptions,
				Def: $1,
			}
		}
	/* OF typename */
	| OF AnyName
		{
			$$ = &ast.AlterTableSpec{
				Tp:      ast.AlterTableOf,
				NewTable: tableNameFromList($2),
			}
		}
	/* NOT OF */
	| NOT OF
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableNotOf,
			}
		}
	;

OptDropBehavior:
	CASCADE     { $$ = int64(ast.DROP_CASCADE) }
	| RESTRICT  { $$ = int64(ast.DROP_RESTRICT) }
	| /* EMPTY */ { $$ = int64(ast.DROP_RESTRICT) }
	;

ColumnCompression:
	COMPRESSION ColId      { $$ = $2 }
	| COMPRESSION DEFAULT  { $$ = "default" }
	;

/*****************************************************************************
 *
 *      ALTER TABLE ... RENAME
 *
 *****************************************************************************/

RenameStmt:
	ALTER TABLE TableName RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_TABLE,
				Relation:   relationName($3),
				Newname:    $6,
			}
		}
	| ALTER TABLE IF_P EXISTS TableName RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_TABLE,
				Relation:   relationName($5),
				Newname:    $8,
				MissingOk:  true,
			}
		}
	| ALTER TABLE TableName RENAME COLUMN ColId TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType:   ast.OBJECT_COLUMN,
				RelationType: ast.OBJECT_TABLE,
				Relation:     relationName($3),
				Subname:      $6,
				Newname:      $8,
			}
		}
	| ALTER TABLE TableName RENAME ColId TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType:   ast.OBJECT_COLUMN,
				RelationType: ast.OBJECT_TABLE,
				Relation:     relationName($3),
				Subname:      $5,
				Newname:      $7,
			}
		}
	| ALTER TABLE TableName RENAME CONSTRAINT Name TO Name
		{
			$$ = &ast.AlterTableStmt{
				Table: relationName($3),
				Specs: []*ast.AlterTableSpec{{
					Tp:           ast.AlterTableRenameConstraint,
					Name:         $6,
					NewColumnName: &ast.ColumnName{Name: model.NewCIStr($8)},
				}},
			}
		}
	| ALTER TABLE IF_P EXISTS TableName RENAME COLUMN ColId TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType:   ast.OBJECT_COLUMN,
				RelationType: ast.OBJECT_TABLE,
				Relation:     relationName($5),
				Subname:      $8,
				Newname:      $10,
				MissingOk:    true,
			}
		}
	| ALTER TABLE IF_P EXISTS TableName RENAME ColId TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType:   ast.OBJECT_COLUMN,
				RelationType: ast.OBJECT_TABLE,
				Relation:     relationName($5),
				Subname:      $7,
				Newname:      $9,
				MissingOk:    true,
			}
		}
	| ALTER TABLE IF_P EXISTS TableName RENAME CONSTRAINT Name TO Name
		{
			$$ = &ast.AlterTableStmt{
				Table: relationName($5),
				Specs: []*ast.AlterTableSpec{{
					Tp:           ast.AlterTableRenameConstraint,
					Name:         $8,
					NewColumnName: &ast.ColumnName{Name: model.NewCIStr($10)},
				}},
				MissingOk: true,
			}
		}
	/* RENAME INDEX */
	| ALTER INDEX QualifiedName RENAME TO Name
		{
			rv := makeRangeVarFromAnyName($3)
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_INDEX,
				Relation:   relationName(rv),
				Newname:    $6,
			}
		}
	| ALTER INDEX IF_P EXISTS QualifiedName RENAME TO Name
		{
			rv := makeRangeVarFromAnyName($5)
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_INDEX,
				Relation:   relationName(rv),
				Newname:    $8,
				MissingOk:  true,
			}
		}
	/* RENAME SEQUENCE */
	| ALTER SEQUENCE QualifiedName RENAME TO Name
		{
			rv := makeRangeVarFromAnyName($3)
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_SEQUENCE,
				Relation:   relationName(rv),
				Newname:    $6,
			}
		}
	| ALTER SEQUENCE IF_P EXISTS QualifiedName RENAME TO Name
		{
			rv := makeRangeVarFromAnyName($5)
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_SEQUENCE,
				Relation:   relationName(rv),
				Newname:    $8,
				MissingOk:  true,
			}
		}
	/* RENAME VIEW */
	| ALTER VIEW QualifiedName RENAME TO Name
		{
			rv := makeRangeVarFromAnyName($3)
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_VIEW,
				Relation:   relationName(rv),
				Newname:    $6,
			}
		}
	| ALTER VIEW IF_P EXISTS QualifiedName RENAME TO Name
		{
			rv := makeRangeVarFromAnyName($5)
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_VIEW,
				Relation:   relationName(rv),
				Newname:    $8,
				MissingOk:  true,
			}
		}
	| ALTER VIEW QualifiedName RENAME COLUMN ColId TO Name
		{
			rv := makeRangeVarFromAnyName($3)
			$$ = &ast.RenameStmt{
				RenameType:   ast.OBJECT_COLUMN,
				RelationType: ast.OBJECT_VIEW,
				Relation:     relationName(rv),
				Subname:      $6,
				Newname:      $8,
			}
		}
	| ALTER VIEW QualifiedName RENAME ColId TO Name
		{
			rv := makeRangeVarFromAnyName($3)
			$$ = &ast.RenameStmt{
				RenameType:   ast.OBJECT_COLUMN,
				RelationType: ast.OBJECT_VIEW,
				Relation:     relationName(rv),
				Subname:      $5,
				Newname:      $7,
			}
		}
	/* RENAME MATERIALIZED VIEW */
	| ALTER MATERIALIZED VIEW QualifiedName RENAME TO Name
		{
			rv := makeRangeVarFromAnyName($4)
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_MATVIEW,
				Relation:   relationName(rv),
				Newname:    $7,
			}
		}
	| ALTER MATERIALIZED VIEW IF_P EXISTS QualifiedName RENAME TO Name
		{
			rv := makeRangeVarFromAnyName($6)
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_MATVIEW,
				Relation:   relationName(rv),
				Newname:    $9,
				MissingOk:  true,
			}
		}
	| ALTER MATERIALIZED VIEW QualifiedName RENAME COLUMN ColId TO Name
		{
			rv := makeRangeVarFromAnyName($4)
			$$ = &ast.RenameStmt{
				RenameType:   ast.OBJECT_COLUMN,
				RelationType: ast.OBJECT_MATVIEW,
				Relation:     relationName(rv),
				Subname:      $7,
				Newname:      $9,
			}
		}
	| ALTER MATERIALIZED VIEW QualifiedName RENAME ColId TO Name
		{
			rv := makeRangeVarFromAnyName($4)
			$$ = &ast.RenameStmt{
				RenameType:   ast.OBJECT_COLUMN,
				RelationType: ast.OBJECT_MATVIEW,
				Relation:     relationName(rv),
				Subname:      $6,
				Newname:      $8,
			}
		}
	/* RENAME FUNCTION / PROCEDURE / ROUTINE */
	| ALTER FUNCTION FunctionWithArgTypes RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_FUNCTION,
				Object:     $3,
				Newname:    $6,
			}
		}
	| ALTER PROCEDURE FunctionWithArgTypes RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_PROCEDURE,
				Object:     $3,
				Newname:    $6,
			}
		}
	| ALTER ROUTINE FunctionWithArgTypes RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_ROUTINE,
				Object:     $3,
				Newname:    $6,
			}
		}
	/* RENAME AGGREGATE */
	| ALTER AGGREGATE AggregateWithArgTypes RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_AGGREGATE,
				Object:     $3,
				Newname:    $6,
			}
		}
	/* RENAME COLLATION */
	| ALTER COLLATION AnyName RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_COLLATION,
				Object:     $3,
				Newname:    $6,
			}
		}
	/* RENAME CONVERSION */
	| ALTER CONVERSION_P AnyName RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_CONVERSION,
				Object:     $3,
				Newname:    $6,
			}
		}
	/* RENAME DOMAIN */
	| ALTER DOMAIN_P AnyName RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_DOMAIN,
				Object:     $3,
				Newname:    $6,
			}
		}
	| ALTER DOMAIN_P AnyName RENAME CONSTRAINT Name TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType:   ast.OBJECT_DOMCONSTRAINT,
				Object:       $3,
				Subname:      $6,
				Newname:      $8,
			}
		}
	/* RENAME SCHEMA */
	| ALTER SCHEMA Name RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_SCHEMA,
				Subname:    $3,
				Newname:    $6,
			}
		}
	/* RENAME SERVER */
	| ALTER SERVER Name RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_FOREIGN_SERVER,
				Object:     &ast.String{Str: $3},
				Newname:    $6,
			}
		}
	/* RENAME FOREIGN DATA WRAPPER */
	| ALTER FOREIGN DATA_P WRAPPER Name RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_FDW,
				Object:     &ast.String{Str: $5},
				Newname:    $8,
			}
		}
	/* RENAME TYPE */
	| ALTER TYPE_P AnyName RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_TYPE,
				Object:     $3,
				Newname:    $6,
			}
		}
	| ALTER TYPE_P AnyName RENAME ATTRIBUTE Name TO Name OptDropBehavior
		{
			$$ = &ast.RenameStmt{
				RenameType:   ast.OBJECT_ATTRIBUTE,
				RelationType: ast.OBJECT_TYPE,
				Relation:     relationName(makeRangeVarFromAnyName($3)),
				Subname:      $6,
				Newname:      $8,
				Behavior:     ast.DropBehavior($9),
			}
		}
	/* RENAME FOREIGN TABLE */
	| ALTER FOREIGN TABLE TableName RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_FOREIGN_TABLE,
				Relation:   relationName($4),
				Newname:    $7,
			}
		}
	| ALTER FOREIGN TABLE IF_P EXISTS TableName RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_FOREIGN_TABLE,
				Relation:   relationName($6),
				Newname:    $9,
				MissingOk:  true,
			}
		}
	| ALTER FOREIGN TABLE TableName RENAME COLUMN ColId TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType:   ast.OBJECT_COLUMN,
				RelationType: ast.OBJECT_FOREIGN_TABLE,
				Relation:     relationName($4),
				Subname:      $7,
				Newname:      $9,
			}
		}
	| ALTER FOREIGN TABLE TableName RENAME ColId TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType:   ast.OBJECT_COLUMN,
				RelationType: ast.OBJECT_FOREIGN_TABLE,
				Relation:     relationName($4),
				Subname:      $6,
				Newname:      $8,
			}
		}
	| ALTER FOREIGN TABLE IF_P EXISTS TableName RENAME COLUMN ColId TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType:   ast.OBJECT_COLUMN,
				RelationType: ast.OBJECT_FOREIGN_TABLE,
				Relation:     relationName($6),
				Subname:      $9,
				Newname:      $11,
				MissingOk:    true,
			}
		}
	| ALTER FOREIGN TABLE IF_P EXISTS TableName RENAME ColId TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType:   ast.OBJECT_COLUMN,
				RelationType: ast.OBJECT_FOREIGN_TABLE,
				Relation:     relationName($6),
				Subname:      $8,
				Newname:      $10,
				MissingOk:    true,
			}
		}
	/* RENAME OPERATOR CLASS / FAMILY */
	| ALTER OPERATOR CLASS AnyName USING Name RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_OPCLASS,
				Object:     prependList(&ast.String{Str: $6}, $4),
				Newname:    $9,
			}
		}
	| ALTER OPERATOR FAMILY AnyName USING Name RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_OPFAMILY,
				Object:     prependList(&ast.String{Str: $6}, $4),
				Newname:    $9,
			}
		}
	/* RENAME TEXT SEARCH */
	| ALTER TEXT_P SEARCH PARSER AnyName RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_TSPARSER,
				Object:     $5,
				Newname:    $8,
			}
		}
	| ALTER TEXT_P SEARCH DICTIONARY AnyName RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_TSDICTIONARY,
				Object:     $5,
				Newname:    $8,
			}
		}
	| ALTER TEXT_P SEARCH TEMPLATE AnyName RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_TSTEMPLATE,
				Object:     $5,
				Newname:    $8,
			}
		}
	| ALTER TEXT_P SEARCH CONFIGURATION AnyName RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_TSCONFIGURATION,
				Object:     $5,
				Newname:    $8,
			}
		}
	/* RENAME PUBLICATION / SUBSCRIPTION */
	| ALTER PUBLICATION Name RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_PUBLICATION,
				Object:     &ast.String{Str: $3},
				Newname:    $6,
			}
		}
	| ALTER SUBSCRIPTION Name RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_SUBSCRIPTION,
				Object:     &ast.String{Str: $3},
				Newname:    $6,
			}
		}
	/* RENAME RULE */
	| ALTER RULE Name ON QualifiedName RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_RULE,
				Relation:   relationName(makeRangeVarFromAnyName($5)),
				Subname:    $3,
				Newname:    $8,
			}
		}
	/* RENAME TRIGGER */
	| ALTER TRIGGER Name ON QualifiedName RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_TRIGGER,
				Relation:   relationName(makeRangeVarFromAnyName($5)),
				Subname:    $3,
				Newname:    $8,
			}
		}
	/* RENAME EVENT TRIGGER */
	| ALTER EVENT TRIGGER Name RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_EVENT_TRIGGER,
				Object:     &ast.String{Str: $4},
				Newname:    $7,
			}
		}
	/* RENAME STATISTICS */
	| ALTER STATISTICS AnyName RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_STATISTIC_EXT,
				Object:     $3,
				Newname:    $6,
			}
		}
	/* RENAME POLICY */
	| ALTER POLICY Name ON QualifiedName RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_POLICY,
				Relation:   relationName(makeRangeVarFromAnyName($5)),
				Subname:    $3,
				Newname:    $8,
			}
		}
	/* RENAME LANGUAGE / DATABASE / TABLESPACE / ROLE */
	| ALTER OptProcedural LANGUAGE Name RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_LANGUAGE,
				Object:     &ast.String{Str: $4},
				Newname:    $7,
			}
		}
	| ALTER DATABASE Name RENAME TO Name
		{
			$$ = &ast.AlterDatabaseStmt{
				Name:     syntaxIdent(yylex, $3, yyS[yypt-3].span),
				RenameTo: syntaxIdent(yylex, $6, yyS[yypt].span),
			}
		}
	| ALTER TABLESPACE Name RENAME TO Name
		{
			$$ = &ast.AlterTablespaceStmt{
				Name:     syntaxIdent(yylex, $3, yyS[yypt-3].span),
				RenameTo: syntaxIdent(yylex, $6, yyS[yypt].span),
			}
		}
	| ALTER ROLE RoleSpec RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_ROLE,
				Subname:    $3.(*ast.RoleSpec).Rolename.Name,
				Newname:    $6,
			}
		}
	| ALTER USER RoleSpec RENAME TO Name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_ROLE,
				Subname:    $3.(*ast.RoleSpec).Rolename.Name,
				Newname:    $6,
			}
		}
	;

/*****************************************************************************
 *
 *      DROP statement
 *
 *****************************************************************************/

DropStmt:
	DROP ObjectTypeAnyName IF_P EXISTS AnyNameList OptDropBehavior
		{
			$$ = dropObjectStmt(int($2), syntaxTableNames(yylex, $5), true, int($6))
		}
	| DROP ObjectTypeAnyName AnyNameList OptDropBehavior
		{
			$$ = dropObjectStmt(int($2), syntaxTableNames(yylex, $3), false, int($4))
		}
	/* DROP INDEX */
	| DROP INDEX CONCURRENTLY IF_P EXISTS Name
		{
			$$ = &ast.DropIndexStmt{
				IndexName:    syntaxName(syntaxIdent(yylex, $6, yyS[yypt].span)),
				Concurrently: true,
				IfExists:     true,
			}
		}
	| DROP INDEX CONCURRENTLY Name
		{
			$$ = &ast.DropIndexStmt{
				IndexName:    syntaxName(syntaxIdent(yylex, $4, yyS[yypt].span)),
				Concurrently: true,
			}
		}
	/* DROP TYPE / DROP DOMAIN */
	| DROP TYPE_P AnyNameList OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($3),
				RemoveType: int(ast.OBJECT_TYPE),
				Behavior:   int($4),
			}
		}
	| DROP TYPE_P IF_P EXISTS AnyNameList OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($5),
				RemoveType: int(ast.OBJECT_TYPE),
				Behavior:   int($6),
				Missing_ok: true,
			}
		}
	| DROP DOMAIN_P AnyNameList OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($3),
				RemoveType: int(ast.OBJECT_DOMAIN),
				Behavior:   int($4),
			}
		}
	| DROP DOMAIN_P IF_P EXISTS AnyNameList OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($5),
				RemoveType: int(ast.OBJECT_DOMAIN),
				Behavior:   int($6),
				Missing_ok: true,
			}
		}
	/* DROP SCHEMA / EXTENSION / LANGUAGE / PUBLICATION / SUBSCRIPTION */
	| DROP SCHEMA NameList OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($3),
				RemoveType: int(ast.OBJECT_SCHEMA),
				Behavior:   int($4),
			}
		}
	| DROP SCHEMA IF_P EXISTS NameList OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($5),
				RemoveType: int(ast.OBJECT_SCHEMA),
				Behavior:   int($6),
				Missing_ok: true,
			}
		}
	| DROP EXTENSION NameList OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($3),
				RemoveType: int(ast.OBJECT_EXTENSION),
				Behavior:   int($4),
			}
		}
	| DROP EXTENSION IF_P EXISTS NameList OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($5),
				RemoveType: int(ast.OBJECT_EXTENSION),
				Behavior:   int($6),
				Missing_ok: true,
			}
		}
	/* DROP FUNCTION / PROCEDURE / ROUTINE / AGGREGATE */
	| DROP FUNCTION AnyNameList OptDropBehavior
		{
			$$ = dropObjectStmt(int(ast.OBJECT_FUNCTION), tableNamesFromNameList($3), false, int($4))
		}
	| DROP FUNCTION IF_P EXISTS AnyNameList OptDropBehavior
		{
			$$ = dropObjectStmt(int(ast.OBJECT_FUNCTION), tableNamesFromNameList($5), true, int($6))
		}
	| DROP PROCEDURE AnyNameList OptDropBehavior
		{
			$$ = dropObjectStmt(int(ast.OBJECT_PROCEDURE), tableNamesFromNameList($3), false, int($4))
		}
	| DROP PROCEDURE IF_P EXISTS AnyNameList OptDropBehavior
		{
			$$ = dropObjectStmt(int(ast.OBJECT_PROCEDURE), tableNamesFromNameList($5), true, int($6))
		}
	| DROP ROUTINE AnyNameList OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($3),
				RemoveType: int(ast.OBJECT_ROUTINE),
				Behavior:   int($4),
			}
		}
	| DROP ROUTINE IF_P EXISTS AnyNameList OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($5),
				RemoveType: int(ast.OBJECT_ROUTINE),
				Behavior:   int($6),
				Missing_ok: true,
			}
		}
	| DROP AGGREGATE AnyNameList OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($3),
				RemoveType: int(ast.OBJECT_AGGREGATE),
				Behavior:   int($4),
			}
		}
	| DROP AGGREGATE IF_P EXISTS AnyNameList OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($5),
				RemoveType: int(ast.OBJECT_AGGREGATE),
				Behavior:   int($6),
				Missing_ok: true,
			}
		}
	/* DROP TRIGGER / POLICY / RULE ON relation */
	| DROP TRIGGER Name ON AnyName OptDropBehavior
		{
			$$ = dropTriggerStmt(false, $3, $5)
		}
	| DROP TRIGGER IF_P EXISTS Name ON AnyName OptDropBehavior
		{
			$$ = dropTriggerStmt(true, $5, $7)
		}
	| DROP POLICY Name ON AnyName OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList(&ast.List{Items: []ast.Node{$5}}),
				RemoveType: int(ast.OBJECT_POLICY),
				Behavior:   int($6),
			}
		}
	| DROP POLICY IF_P EXISTS Name ON AnyName OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList(&ast.List{Items: []ast.Node{$7}}),
				RemoveType: int(ast.OBJECT_POLICY),
				Behavior:   int($8),
				Missing_ok: true,
			}
		}
	| DROP RULE Name ON AnyName OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList(&ast.List{Items: []ast.Node{$5}}),
				RemoveType: int(ast.OBJECT_RULE),
				Behavior:   int($6),
			}
		}
	| DROP RULE IF_P EXISTS Name ON AnyName OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList(&ast.List{Items: []ast.Node{$7}}),
				RemoveType: int(ast.OBJECT_RULE),
				Behavior:   int($8),
				Missing_ok: true,
			}
		}
	/* DROP EVENT TRIGGER */
	| DROP EVENT TRIGGER Name OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList(&ast.List{Items: []ast.Node{&ast.String{Str: $4}}}),
				RemoveType: int(ast.OBJECT_EVENT_TRIGGER),
				Behavior:   int($5),
			}
		}
	| DROP EVENT TRIGGER IF_P EXISTS Name OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList(&ast.List{Items: []ast.Node{&ast.String{Str: $6}}}),
				RemoveType: int(ast.OBJECT_EVENT_TRIGGER),
				Behavior:   int($7),
				Missing_ok: true,
			}
		}
	/* DROP LANGUAGE / PUBLICATION / SUBSCRIPTION */
	| DROP LANGUAGE Name OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList(&ast.List{Items: []ast.Node{&ast.String{Str: $3}}}),
				RemoveType: int(ast.OBJECT_LANGUAGE),
				Behavior:   int($4),
			}
		}
	| DROP LANGUAGE IF_P EXISTS Name OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList(&ast.List{Items: []ast.Node{&ast.String{Str: $5}}}),
				RemoveType: int(ast.OBJECT_LANGUAGE),
				Behavior:   int($6),
				Missing_ok: true,
			}
		}
	| DROP PUBLICATION NameList OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($3),
				RemoveType: int(ast.OBJECT_PUBLICATION),
				Behavior:   int($4),
			}
		}
	| DROP PUBLICATION IF_P EXISTS NameList OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($5),
				RemoveType: int(ast.OBJECT_PUBLICATION),
				Behavior:   int($6),
				Missing_ok: true,
			}
		}
	| DROP SUBSCRIPTION NameList OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($3),
				RemoveType: int(ast.OBJECT_SUBSCRIPTION),
				Behavior:   int($4),
			}
		}
	| DROP SUBSCRIPTION IF_P EXISTS NameList OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($5),
				RemoveType: int(ast.OBJECT_SUBSCRIPTION),
				Behavior:   int($6),
				Missing_ok: true,
			}
		}
	/* DROP INDEX CONCURRENTLY */
	| DROP INDEX CONCURRENTLY AnyNameList OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($4),
				RemoveType: int(ast.OBJECT_INDEX),
				Behavior:   int($5),
				Concurrent: true,
			}
		}
	| DROP INDEX CONCURRENTLY IF_P EXISTS AnyNameList OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($6),
				RemoveType: int(ast.OBJECT_INDEX),
				Behavior:   int($7),
				Missing_ok: true,
				Concurrent: true,
			}
		}
	/* DROP FOREIGN DATA WRAPPER / DROP SERVER */
	| DROP FOREIGN DATA_P WRAPPER NameList OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($5),
				RemoveType: int(ast.OBJECT_FDW),
				Behavior:   int($6),
			}
		}
	| DROP FOREIGN DATA_P WRAPPER IF_P EXISTS NameList OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($7),
				RemoveType: int(ast.OBJECT_FDW),
				Behavior:   int($8),
				Missing_ok: true,
			}
		}
	/* DROP OWNED BY */
	| DROP OWNED BY NameList OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($4),
				RemoveType: int(ast.OBJECT_ROLE),
				Behavior:   int($5),
			}
		}
	;

ObjectTypeAnyName:
	TABLE               { $$ = int64(ast.OBJECT_TABLE) }
	| SEQUENCE          { $$ = int64(ast.OBJECT_SEQUENCE) }
	| VIEW              { $$ = int64(ast.OBJECT_VIEW) }
	| MATERIALIZED VIEW { $$ = int64(ast.OBJECT_MATVIEW) }
	| INDEX             { $$ = int64(ast.OBJECT_INDEX) }
	| FOREIGN TABLE     { $$ = int64(ast.OBJECT_FOREIGN_TABLE) }
	| COLLATION         { $$ = int64(ast.OBJECT_COLLATION) }
	| CONVERSION_P      { $$ = int64(ast.OBJECT_CONVERSION) }
	| STATISTICS        { $$ = int64(ast.OBJECT_STATISTIC_EXT) }
	| TEXT_P SEARCH PARSER { $$ = int64(ast.OBJECT_TSPARSER) }
	| TEXT_P SEARCH DICTIONARY { $$ = int64(ast.OBJECT_TSDICTIONARY) }
	| TEXT_P SEARCH TEMPLATE { $$ = int64(ast.OBJECT_TSTEMPLATE) }
	| TEXT_P SEARCH CONFIGURATION { $$ = int64(ast.OBJECT_TSCONFIGURATION) }
	| ACCESS METHOD     { $$ = int64(ast.OBJECT_ACCESS_METHOD) }
	;

AnyNameList:
	AnyName
		{
			$$ = &ast.List{Items: []ast.Node{$1}}
		}
	| AnyNameList ',' AnyName
		{
			$1.Items = append($1.Items, $3)
			$$ = $1
		}
	;

/*****************************************************************************
 *
 *      CREATE INDEX statement
 *
 *****************************************************************************/

IndexStmt:
	CREATE OptUnique INDEX OptConcurrently IF_P NOT EXISTS OptIndexName
	ON TableName AccessMethodClause '(' IndexParams ')' OptInclude OptUniqueNullTreatment OptRelOptions OptTablespace WhereClauseOpt
		{
			$$ = &ast.CreateIndexStmt{
				IndexName:     indexNameFromList(yylex, $8),
				Table:         relationName($10),
				Unique:        $2,
				NullsNotDistinct: $16,
				IfNotExists:   true,
				Concurrently:  $4,
				IndexColNames: indexElemColNames($13),
				Include:       indexIncludeNames($15),
				Using:         $11,
				WithOptions:   $17,
				Tablespace:    $18,
				Where:         exprOrNil($19),
			}
		}
	| CREATE OptUnique INDEX OptConcurrently OptIndexName
	ON TableName AccessMethodClause '(' IndexParams ')' OptInclude OptUniqueNullTreatment OptRelOptions OptTablespace WhereClauseOpt
		{
			$$ = &ast.CreateIndexStmt{
				IndexName:     indexNameFromList(yylex, $5),
				Table:         relationName($7),
				Unique:        $2,
				NullsNotDistinct: $13,
				Concurrently:  $4,
				IndexColNames: indexElemColNames($10),
				Include:       indexIncludeNames($12),
				Using:         $8,
				WithOptions:   $14,
				Tablespace:    $15,
				Where:         exprOrNil($16),
			}
		}
	;

OptIndexName:
	QualifiedName
		{
			$$ = $1
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

OptInclude:
	INCLUDE '(' IndexIncludingParams ')'
		{ $$ = $3 }
	| /* EMPTY */
		{ $$ = nil }
	;

IndexIncludingParams:
	IndexElem
		{ $$ = makeList($1) }
	| IndexIncludingParams ',' IndexElem
		{ $$ = appendList($1, $3) }
	;

OptUnique:
	UNIQUE      { $$ = true }
	| /* EMPTY */ { $$ = false }
	;

OptConcurrently:
	CONCURRENTLY { $$ = true }
	| /* EMPTY */  { $$ = false }
	;

OptSingleName:
	Name        { $$ = $1 }
	| /* EMPTY */ { $$ = "" }
	;

AccessMethodClause:
	USING Name  { $$ = syntaxIdent(yylex, $2, yyS[yypt].span) }
	| /* EMPTY */ { $$ = lex.Ident{Span: loc.InvalidSpan()} }
	;

IndexParams:
	IndexElem
		{ $$ = makeList($1) }
	| IndexParams ',' IndexElem
		{ $$ = appendList($1, $3) }
	;

IndexElem:
	ColId OptCollate OptQualifiedName Order OptNullsOrder
		{
			$$ = &ast.IndexElem{
				Name:          $1,
				Collation:     $2,
				Opclass:       $3,
				Ordering:      ast.SortByDir($4),
				NullsOrdering: ast.SortByNulls($5),
			}
		}
	| ColId OptCollate AnyName RelOptions Order OptNullsOrder
		{
			$$ = &ast.IndexElem{
				Name:          $1,
				Collation:     $2,
				Opclass:       $3,
				Opclassopts:   $4,
				Ordering:      ast.SortByDir($5),
				NullsOrdering: ast.SortByNulls($6),
			}
		}
	| FuncExprWindowless OptCollate OptQualifiedName Order OptNullsOrder
		{
			$$ = &ast.IndexElem{
				Expr:          $1,
				Collation:     $2,
				Opclass:       $3,
				Ordering:      ast.SortByDir($4),
				NullsOrdering: ast.SortByNulls($5),
			}
		}
	| FuncExprWindowless OptCollate AnyName RelOptions Order OptNullsOrder
		{
			$$ = &ast.IndexElem{
				Expr:          $1,
				Collation:     $2,
				Opclass:       $3,
				Opclassopts:   $4,
				Ordering:      ast.SortByDir($5),
				NullsOrdering: ast.SortByNulls($6),
			}
		}
	| '(' AExpr ')' OptCollate OptQualifiedName Order OptNullsOrder
		{
			$$ = &ast.IndexElem{
				Expr:          $2,
				Collation:     $4,
				Opclass:       $5,
				Ordering:      ast.SortByDir($6),
				NullsOrdering: ast.SortByNulls($7),
			}
		}
	| '(' AExpr ')' OptCollate AnyName RelOptions Order OptNullsOrder
		{
			$$ = &ast.IndexElem{
				Expr:          $2,
				Collation:     $4,
				Opclass:       $5,
				Opclassopts:   $6,
				Ordering:      ast.SortByDir($7),
				NullsOrdering: ast.SortByNulls($8),
			}
		}
	;

OptNullsOrder:
	NULLS_LA FIRST_P  { $$ = int64(ast.SORTBY_NULLS_FIRST) }
	| NULLS_LA LAST_P { $$ = int64(ast.SORTBY_NULLS_LAST) }
	| /* EMPTY */      { $$ = int64(ast.SORTBY_NULLS_DEFAULT) }
	;

/*****************************************************************************
 *
 *      CREATE VIEW statement
 *
 *****************************************************************************/

CreateViewStmt:
	CREATE OptTemp VIEW QualifiedName OptColumnList OptRelOptions
	AS SelectStmt OptCheckOption
		{
			rv := syntaxTableName(syntaxName(syntaxIdents(yylex, $4)...))
			rv.Relpersistence = relpersistenceForTemp($2)
			$$ = &ast.CreateViewStmt{
				Temporary:      $2 != 0,
				ViewName:       rv,
				Cols:           syntaxIdents(yylex, $5),
				WithOptions:    $6,
				Select:         $8.(ast.StmtNode),
				HasCheckOption: $9 != 0,
				CheckOption:    viewCheckOption($9),
			}
		}
	| CREATE OR REPLACE OptTemp VIEW QualifiedName OptColumnList OptRelOptions
	AS SelectStmt OptCheckOption
		{
			rv := syntaxTableName(syntaxName(syntaxIdents(yylex, $6)...))
			rv.Relpersistence = relpersistenceForTemp($4)
			$$ = &ast.CreateViewStmt{
				OrReplace:      true,
				Temporary:      $4 != 0,
				ViewName:       rv,
				Cols:           syntaxIdents(yylex, $7),
				WithOptions:    $8,
				Select:         $10.(ast.StmtNode),
				HasCheckOption: $11 != 0,
				CheckOption:    viewCheckOption($11),
			}
		}
	| CREATE OptTemp RECURSIVE VIEW QualifiedName '(' ColumnList ')' OptRelOptions
	AS SelectStmt OptCheckOption
		{
			rv := syntaxTableName(syntaxName(syntaxIdents(yylex, $5)...))
			rv.Relpersistence = relpersistenceForTemp($2)
			cte := &ast.CommonTableExpression{BaseCommonTableExpression: pkast.BaseCommonTableExpression{
				Name:    lastNameIdent(rv.Name),
				Columns: syntaxIdents(yylex, $7),
				Query:   $11.(ast.StmtNode),
			}}
			wc := &ast.WithClause{BaseWithClause: pkast.BaseWithClause{
				CTEs:        []ast.CTENode{cte},
				IsRecursive: true,
			}}
			sel := &ast.SelectStmt{
				SelectStmtOpts: &ast.SelectStmtOpts{},
				Fields: &ast.FieldList{Items: []*ast.SelectItem{
					syntaxSelectItem(syntaxWildcard(ast.QualifiedName{}, loc.InvalidSpan()), lex.Ident{Span: loc.InvalidSpan()}, loc.InvalidSpan()),
				}},
				From: syntaxFromClause(syntaxTableSource(syntaxTableName(rv.Name), loc.InvalidSpan())),
				With: wc,
			}
			$$ = &ast.CreateViewStmt{
				Temporary:      $2 != 0,
				ViewName:       rv,
				Cols:           syntaxIdents(yylex, $7),
				WithOptions:    $9,
				Select:         sel,
				HasCheckOption: $12 != 0,
				CheckOption:    viewCheckOption($12),
			}
		}
	| CREATE OR REPLACE OptTemp RECURSIVE VIEW QualifiedName '(' ColumnList ')' OptRelOptions
	AS SelectStmt OptCheckOption
		{
			rv := syntaxTableName(syntaxName(syntaxIdents(yylex, $7)...))
			rv.Relpersistence = relpersistenceForTemp($4)
			cte := &ast.CommonTableExpression{BaseCommonTableExpression: pkast.BaseCommonTableExpression{
				Name:    lastNameIdent(rv.Name),
				Columns: syntaxIdents(yylex, $9),
				Query:   $13.(ast.StmtNode),
			}}
			wc := &ast.WithClause{BaseWithClause: pkast.BaseWithClause{
				CTEs:        []ast.CTENode{cte},
				IsRecursive: true,
			}}
			sel := &ast.SelectStmt{
				SelectStmtOpts: &ast.SelectStmtOpts{},
				Fields: &ast.FieldList{Items: []*ast.SelectItem{
					syntaxSelectItem(syntaxWildcard(ast.QualifiedName{}, loc.InvalidSpan()), lex.Ident{Span: loc.InvalidSpan()}, loc.InvalidSpan()),
				}},
				From: syntaxFromClause(syntaxTableSource(syntaxTableName(rv.Name), loc.InvalidSpan())),
				With: wc,
			}
			$$ = &ast.CreateViewStmt{
				OrReplace:      true,
				Temporary:      $4 != 0,
				ViewName:       rv,
				Cols:           syntaxIdents(yylex, $9),
				WithOptions:    $11,
				Select:         sel,
				HasCheckOption: $14 != 0,
				CheckOption:    viewCheckOption($14),
			}
		}
	;

OptCheckOption:
	WITH CHECK OPTION              { $$ = int64(VIEW_CHECK_OPTION_LOCAL) }
	| WITH CASCADED CHECK OPTION   { $$ = int64(VIEW_CHECK_OPTION_CASCADED) }
	| WITH LOCAL CHECK OPTION      { $$ = int64(VIEW_CHECK_OPTION_LOCAL) }
	| /* EMPTY */                  { $$ = int64(VIEW_CHECK_OPTION_NONE) }
	;

/*****************************************************************************
 *
 *      CREATE FUNCTION / CREATE PROCEDURE
 *
 *****************************************************************************/

CreateFunctionStmt:
	CREATE OptOrReplace FUNCTION FunctionName FuncArgsWithDefaults
	RETURNS FuncReturn OptCreateFuncOptList OptRoutineBody
		{
			$$ = &ast.CreateFunctionStmt{
				OrReplace:  $2,
				Name:       tableNameFromList($4),
				Params:     $5,
				Returns:    $7,
				Options:    defElems($8),
				Body:       $9,
			}
		}
	| CREATE OptOrReplace FUNCTION FunctionName FuncArgsWithDefaults
	RETURNS TABLE '(' TableFuncColumnList ')' OptCreateFuncOptList OptRoutineBody
		{
			$$ = &ast.CreateFunctionStmt{
				OrReplace:  $2,
				Name:       tableNameFromList($4),
				Params:     $5,
				ReturnTable: $9,
				Options:    defElems($11),
				Body:       $12,
			}
		}
	| CREATE OptOrReplace FUNCTION FunctionName FuncArgsWithDefaults
	OptCreateFuncOptList OptRoutineBody
		{
			$$ = &ast.CreateFunctionStmt{
				OrReplace:  $2,
				Name:       tableNameFromList($4),
				Params:     $5,
				Options:    defElems($6),
				Body:       $7,
			}
		}
	| CREATE OptOrReplace PROCEDURE FunctionName FuncArgsWithDefaults
	OptCreateFuncOptList OptRoutineBody
		{
			$$ = &ast.CreateFunctionStmt{
				OrReplace:   $2,
				IsProcedure: true,
				Name:        tableNameFromList($4),
				Params:      $5,
				Options:     defElems($6),
				Body:        $7,
			}
		}
	;

OptOrReplace:
	OR REPLACE   { $$ = true }
	| /* EMPTY */ { $$ = false }
	;

FuncArgsWithDefaults:
	'(' FuncArgsWithDefaultsList ')' { $$ = $2 }
	| '(' ')' { $$ = nil }
	;

FuncArgsWithDefaultsList:
	FuncArgWithDefault
		{ $$ = makeList($1) }
	| FuncArgsWithDefaultsList ',' FuncArgWithDefault
		{ $$ = appendList($1, $3) }
	;

FuncArgWithDefault:
	FuncArg
		{ $$ = $1 }
	| FuncArg DEFAULT AExpr
		{
			fp := $1.(*ast.FunctionParameter)
			fp.Defexpr = $3
			$$ = fp
		}
	| FuncArg '=' AExpr
		{
			fp := $1.(*ast.FunctionParameter)
			fp.Defexpr = $3
			$$ = fp
		}
	;

FuncArg:
	ArgClass ParamName FuncType
		{
			$$ = &ast.FunctionParameter{
				Name:    $2,
				ArgType: $3,
				Mode:    ast.FunctionParameterMode($1),
			}
		}
	| ParamName ArgClass FuncType
		{
			$$ = &ast.FunctionParameter{
				Name:    $1,
				ArgType: $3,
				Mode:    ast.FunctionParameterMode($2),
			}
		}
	| ParamName FuncType
		{
			$$ = &ast.FunctionParameter{
				Name:    $1,
				ArgType: $2,
				Mode:    ast.FUNC_PARAM_IN,
			}
		}
	| ArgClass FuncType
		{
			$$ = &ast.FunctionParameter{
				ArgType: $2,
				Mode:    ast.FunctionParameterMode($1),
			}
		}
	| FuncType
		{
			$$ = &ast.FunctionParameter{
				ArgType: $1,
				Mode:    ast.FUNC_PARAM_IN,
			}
		}
	;

ArgClass:
	IN_P       { $$ = int64(ast.FUNC_PARAM_IN) }
	| OUT_P    { $$ = int64(ast.FUNC_PARAM_OUT) }
	| INOUT    { $$ = int64(ast.FUNC_PARAM_INOUT) }
	| IN_P OUT_P { $$ = int64(ast.FUNC_PARAM_INOUT) }
	| VARIADIC { $$ = int64(ast.FUNC_PARAM_VARIADIC) }
	;

ParamName:
	TypeFunctionName { $$ = $1 }
	;

FuncReturn:
	FuncType { $$ = $1 }
	;

TableFuncColumnList:
	TableFuncColumn
		{ $$ = makeList($1) }
	| TableFuncColumnList ',' TableFuncColumn
		{ $$ = appendList($1, $3) }
	;

TableFuncColumn:
	ParamName FuncType
		{
			$$ = &ast.FunctionParameter{
				Name:    $1,
				ArgType: $2,
				Mode:    ast.FUNC_PARAM_TABLE,
			}
		}
	;

FuncType:
	Typename { $$ = $1 }
	| TypeFunctionName Attrs '%' TYPE_P
		{
			$$ = typeNameFromList(prependList(&ast.String{Str: $1}, $2))
			$$.PGPercentType = true
		}
	| SETOF TypeFunctionName Attrs '%' TYPE_P
		{
			$$ = typeNameFromList(prependList(&ast.String{Str: $2}, $3))
			$$.PGPercentType = true
			$$.PGSetOf = true
		}
	;

OptCreateFuncOptList:
	CreateFuncOptList { $$ = $1 }
	| /* EMPTY */ { $$ = nil }
	;

CreateFuncOptList:
	CreateFuncOptItem
		{ $$ = makeList($1) }
	| CreateFuncOptList CreateFuncOptItem
		{ $$ = appendList($1, $2) }
	;

OptRoutineBody:
	ReturnStmt
		{
			$$ = $1
		}
	| BEGIN_P ATOMIC RoutineBodyStmtList END_P
		{
			$$ = &ast.BeginAtomicStmt{List: $3}
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

RoutineBodyStmtList:
	RoutineBodyStmtList RoutineBodyStmt ';'
		{
			if $2 != nil {
				$$ = appendList($1, $2)
			} else {
				$$ = $1
			}
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

RoutineBodyStmt:
	Statement
		{
			$$ = $1
		}
	| ReturnStmt
		{
			$$ = $1
		}
	;

ReturnStmt:
	RETURN AExpr
		{
			$$ = &ast.ReturnStmt{
				Expr: $2,
			}
		}
	;

CreateFuncOptItem:
	AS FuncAs
		{
			$$ = &ast.DefElem{
				Defname: "as",
				Arg:     $2,
			}
		}
	| LANGUAGE NonReservedWordOrSConst
		{
			$$ = &ast.DefElem{
				Defname: "language",
				Arg:     &ast.String{Str: $2},
			}
		}
	| TRANSFORM TransformTypeList
		{
			$$ = &ast.DefElem{
				Defname: "transform",
				Arg:     $2,
			}
		}
	| WINDOW
		{
			$$ = &ast.DefElem{
				Defname: "window",
				Arg:     &ast.Integer{Ival: 1},
			}
		}
	| CommonFuncOptItem { $$ = $1 }
	;

FuncAs:
	SConst
		{ $$ = makeList(&ast.String{Str: $1}) }
	| SConst ',' SConst
		{ $$ = makeList2(&ast.String{Str: $1}, &ast.String{Str: $3}) }
	;

TransformTypeList:
	FOR TYPE_P Typename
		{ $$ = makeList(typeNameNode($3)) }
	| TransformTypeList ',' FOR TYPE_P Typename
		{ $$ = appendList($1, typeNameNode($5)) }
	;

CommonFuncOptItem:
	IMMUTABLE
		{
			$$ = &ast.DefElem{
				Defname: "volatility",
				Arg:     &ast.String{Str: "immutable"},
			}
		}
	| STABLE
		{
			$$ = &ast.DefElem{
				Defname: "volatility",
				Arg:     &ast.String{Str: "stable"},
			}
		}
	| VOLATILE
		{
			$$ = &ast.DefElem{
				Defname: "volatility",
				Arg:     &ast.String{Str: "volatile"},
			}
		}
	| STRICT_P
		{
			$$ = &ast.DefElem{
				Defname: "strict",
				Arg:     &ast.Integer{Ival: 1},
			}
		}
	| CALLED ON NULL_P INPUT_P
		{
			$$ = &ast.DefElem{
				Defname: "strict",
				Arg:     &ast.Integer{Ival: 0},
			}
		}
	| RETURNS NULL_P ON NULL_P INPUT_P
		{
			$$ = &ast.DefElem{
				Defname: "strict",
				Arg:     &ast.Integer{Ival: 1},
			}
		}
	| SECURITY DEFINER
		{
			$$ = &ast.DefElem{
				Defname: "security",
				Arg:     &ast.Integer{Ival: 1},
			}
		}
	| SECURITY INVOKER
		{
			$$ = &ast.DefElem{
				Defname: "security",
				Arg:     &ast.Integer{Ival: 0},
			}
		}
	| LEAKPROOF
		{
			$$ = &ast.DefElem{
				Defname: "leakproof",
				Arg:     &ast.Integer{Ival: 1},
			}
		}
	| NOT LEAKPROOF
		{
			$$ = &ast.DefElem{
				Defname: "leakproof",
				Arg:     &ast.Integer{Ival: 0},
			}
		}
	| COST NumericOnly
		{
			$$ = &ast.DefElem{
				Defname: "cost",
				Arg:     $2,
			}
		}
	| ROWS NumericOnly
		{
			$$ = &ast.DefElem{
				Defname: "rows",
				Arg:     $2,
			}
		}
	| PARALLEL ColId
		{
			$$ = &ast.DefElem{
				Defname: "parallel",
				Arg:     &ast.String{Str: $2},
			}
		}
	| SET SetRestMore
		{
			$$ = &ast.DefElem{
				Defname: "set",
				Arg:     $2,
			}
		}
	| VariableResetStmt
		{
			$$ = &ast.DefElem{
				Defname: "set",
				Arg:     $1,
			}
		}
	| SUPPORT AnyName
		{
			$$ = &ast.DefElem{
				Defname: "support",
				Arg:     $2,
			}
		}
	;

/*****************************************************************************
 *
 *      Transaction statements
 *
 *****************************************************************************/

TransactionStmt:
	ABORT_P OptTransaction OptTransactionChain
		{
			$$ = &ast.RollbackStmt{}
		}
	| BEGIN_P OptTransaction TransactionModeListOrEmpty
		{
			isolation, readOnly, deferrable := transactionModes($3)
			$$ = &ast.BeginStmt{
				IsolationLevel: isolation,
				ReadOnly:       readOnly,
				Deferrable:     deferrable,
			}
		}
	| START TRANSACTION TransactionModeListOrEmpty
		{
			isolation, readOnly, deferrable := transactionModes($3)
			$$ = &ast.BeginStmt{
				IsolationLevel: isolation,
				ReadOnly:       readOnly,
				Deferrable:     deferrable,
			}
		}
	| PREPARE TRANSACTION SConst
		{
			$$ = &ast.TransactionStmt{
				Kind: ast.TRANS_STMT_PREPARE,
				Gid:  $3,
			}
		}
	| COMMIT PREPARED SConst
		{
			$$ = &ast.TransactionStmt{
				Kind: ast.TRANS_STMT_COMMIT_PREPARED,
				Gid:  $3,
			}
		}
	| ROLLBACK PREPARED SConst
		{
			$$ = &ast.TransactionStmt{
				Kind: ast.TRANS_STMT_ROLLBACK_PREPARED,
				Gid:  $3,
			}
		}
	| COMMIT OptTransaction OptTransactionChain
		{
			$$ = &ast.CommitStmt{}
		}
	| END_P OptTransaction OptTransactionChain
		{
			$$ = &ast.CommitStmt{}
		}
	| ROLLBACK OptTransaction OptTransactionChain
		{
			$$ = &ast.RollbackStmt{}
		}
	| ROLLBACK OptTransaction TO SAVEPOINT ColId
		{
			$$ = &ast.RollbackStmt{SavepointName: $5}
		}
	| ROLLBACK OptTransaction TO ColId
		{
			$$ = &ast.RollbackStmt{SavepointName: $4}
		}
	| SAVEPOINT ColId
		{
			$$ = &ast.SavepointStmt{Name: $2}
		}
	| RELEASE SAVEPOINT ColId
		{
			$$ = &ast.ReleaseSavepointStmt{Name: $3}
		}
	| RELEASE ColId
		{
			$$ = &ast.ReleaseSavepointStmt{Name: $2}
		}
	;

OptTransaction:
	WORK        {}
	| TRANSACTION {}
	| /* EMPTY */ {}
	;

OptTransactionChain:
	AND CHAIN      { $$ = true }
	| AND NO CHAIN { $$ = false }
	| /* EMPTY */  { $$ = false }
	;

/*****************************************************************************
 *
 *      EXPLAIN statement
 *
 *****************************************************************************/

ExplainStmt:
	EXPLAIN ExplainableStmt
		{
			$$ = &ast.ExplainStmt{
				Stmt: $2.(ast.StmtNode),
			}
		}
	| EXPLAIN ANALYZE ExplainableStmt
		{
			$$ = &ast.ExplainStmt{
				Stmt:    $3.(ast.StmtNode),
				Analyze: true,
				Options: []string{"analyze"},
			}
		}
	| EXPLAIN VERBOSE ExplainableStmt
		{
			$$ = &ast.ExplainStmt{
				Stmt:    $3.(ast.StmtNode),
				Options: []string{"verbose"},
			}
		}
	| EXPLAIN ANALYZE VERBOSE ExplainableStmt
		{
			$$ = &ast.ExplainStmt{
				Stmt:    $4.(ast.StmtNode),
				Analyze: true,
				Options: []string{"analyze", "verbose"},
			}
		}
	| EXPLAIN '(' UtilityOptionList ')' ExplainableStmt
		{
			options := explainOptionList($3)
			format, err := explainFormat(options)
			if err != nil {
				yylex.Error(err.Error())
			}
			$$ = &ast.ExplainStmt{
				Stmt:    $5.(ast.StmtNode),
				Options: options,
				Format:  format,
			}
		}
	;

ExplainableStmt:
	SelectStmt          { $$ = $1 }
	| InsertStmt        { $$ = $1 }
	| UpdateStmt        { $$ = $1 }
	| DeleteStmt    { $$ = $1 }
	| MergeStmt         { $$ = $1 }
	| DeclareCursorStmt { $$ = $1 }
	| CreateAsStmt      { $$ = $1 }
	| CreateMatViewStmt { $$ = $1 }
	| RefreshMatViewStmt { $$ = $1 }
	| ExecuteStmt       { $$ = $1 }
	| CreateTableStmt        { $$ = $1 }
	;

/*****************************************************************************
 *
 *      COPY statement
 *
 *****************************************************************************/

CopyStmt:
	COPY OptBinary TableName OptColumnList
			CopyFrom OptProgram CopyFileName CopyDelimiter CopyOptWith
			CopyOptions WhereClauseOpt
		{
			rv := relationName($3)
			stmt := &ast.CopyStmt{
				Table:       rv,
				Columns:  columnNamesFromList($4),
				IsFrom:   $5,
				Program:   $6,
				Std:         $7 == nil,
				WhereClause: $11,
			}
			if $7 != nil {
				stmt.FileName = *$7
			} else if $6 {
				yylex.Error("COPY PROGRAM requires a file name")
			}
			var opts *ast.List
			if $2 != nil {
				opts = appendList(opts, $2)
			}
			if $8 != nil {
				opts = appendList(opts, $8)
			}
			opts = concatLists(opts, $10)
			stmt.Options = defElems(opts)
			$$ = stmt
		}
	| COPY '(' PreparableStmt ')' TO OptProgram CopyFileName CopyOptWith CopyOptions
		{
			stmt := &ast.CopyStmt{
				Query:     $3,
				IsFrom:    false,
				Program: $6,
				Std:       $7 == nil,
				Options:   defElems($9),
			}
			if $7 != nil {
				stmt.FileName = *$7
			} else if $6 {
				yylex.Error("COPY PROGRAM requires a file name")
			}
			$$ = stmt
		}
	;

CopyFrom:
	FROM { $$ = true }
	| TO { $$ = false }
	;

OptProgram:
	PROGRAM     { $$ = true }
	| /* EMPTY */ { $$ = false }
	;

/*
 * copy_file_name NULL indicates stdio is used. Whether stdin or stdout is
 * used depends on the direction. (It really doesn't make sense to copy from
 * stdout. We silently correct the "typo".)		 - AY 9/94
 */
CopyFileName:
	SConst      { $$ = &$1 }
	| STDIN     { $$ = nil }
	| STDOUT    { $$ = nil }
	;

CopyOptWith:
	WITH {}
	| /* EMPTY */ {}
	;

CopyOptions:
	CopyOptList                       { $$ = $1 }
	| '(' UtilityOptionList ')'       { $$ = $2 }
	;

/* old-style COPY option syntax */
CopyOptList:
	CopyOptList CopyOptItem
		{
			$$ = appendList($1, $2)
		}
	| /* EMPTY */ { $$ = nil }
	;

CopyOptItem:
	BINARY
		{
			$$ = &ast.DefElem{Defname: "format", Arg: &ast.String{Str: "binary"}}
		}
	| FREEZE
		{
			$$ = &ast.DefElem{Defname: "freeze", Arg: &ast.Boolean{Boolval: true}}
		}
	| DELIMITER OptAs SConst
		{
			$$ = &ast.DefElem{Defname: "delimiter", Arg: &ast.String{Str: $3}}
		}
	| NULL_P OptAs SConst
		{
			$$ = &ast.DefElem{Defname: "null", Arg: &ast.String{Str: $3}}
		}
	| CSV
		{
			$$ = &ast.DefElem{Defname: "format", Arg: &ast.String{Str: "csv"}}
		}
	| HEADER_P
		{
			$$ = &ast.DefElem{Defname: "header", Arg: &ast.Boolean{Boolval: true}}
		}
	| QUOTE OptAs SConst
		{
			$$ = &ast.DefElem{Defname: "quote", Arg: &ast.String{Str: $3}}
		}
	| ESCAPE OptAs SConst
		{
			$$ = &ast.DefElem{Defname: "escape", Arg: &ast.String{Str: $3}}
		}
	| FORCE QUOTE ColumnList
		{
			$$ = &ast.DefElem{Defname: "force_quote", Arg: $3}
		}
	| FORCE QUOTE '*'
		{
			$$ = &ast.DefElem{Defname: "force_quote", Arg: &ast.A_Star{}}
		}
	| FORCE NOT NULL_P ColumnList
		{
			$$ = &ast.DefElem{Defname: "force_not_null", Arg: $4}
		}
	| FORCE NOT NULL_P '*'
		{
			$$ = &ast.DefElem{Defname: "force_not_null", Arg: &ast.A_Star{}}
		}
	| FORCE NULL_P ColumnList
		{
			$$ = &ast.DefElem{Defname: "force_null", Arg: $3}
		}
	| FORCE NULL_P '*'
		{
			$$ = &ast.DefElem{Defname: "force_null", Arg: &ast.A_Star{}}
		}
	| ENCODING SConst
		{
			$$ = &ast.DefElem{Defname: "encoding", Arg: &ast.String{Str: $2}}
		}
	;

/* The following exist for backward compatibility with very old versions */

OptBinary:
	BINARY
		{
			$$ = &ast.DefElem{Defname: "format", Arg: &ast.String{Str: "binary"}}
		}
	| /* EMPTY */ { $$ = nil }
	;

CopyDelimiter:
	OptUsing DELIMITERS SConst
		{
			$$ = &ast.DefElem{Defname: "delimiter", Arg: &ast.String{Str: $3}}
		}
	| /* EMPTY */ { $$ = nil }
	;

OptUsing:
	USING
	| /* EMPTY */
	;

UtilityOptionList:
	UtilityOptionElem
		{
			$$ = makeList($1)
		}
	| UtilityOptionList ',' UtilityOptionElem
		{
			$$ = appendList($1, $3)
		}
	;

UtilityOptionElem:
	UtilityOptionName UtilityOptionArg
		{
			$$ = &ast.DefElem{
				Defname: $1,
				Arg:     $2,
			}
		}
	;

UtilityOptionName:
	NonReservedWord  { $$ = $1 }
	| ANALYZE        { $$ = "analyze" }
	| FORMAT_LA      { $$ = "format" }
	| DEFAULT        { $$ = "default" }
	| NULL_P         { $$ = "null" }
	;

UtilityOptionArg:
	OptBooleanOrString   { $$ = &ast.String{Str: $1} }
	| NumericOnly           { $$ = $1 }
	| '*'                   { $$ = &ast.A_Star{} }
	| DEFAULT               { $$ = &ast.String{Str: "default"} }
	| '(' UtilityOptionArgList ')' { $$ = $2 }
	| /* EMPTY */           { $$ = nil }
	;

UtilityOptionArgList:
	OptBooleanOrString
		{ $$ = makeList(&ast.String{Str: $1}) }
	| UtilityOptionArgList ',' OptBooleanOrString
		{ $$ = appendList($1, &ast.String{Str: $3}) }
	;

OptBooleanOrString:
	TRUE_P                      { $$ = "true" }
	| FALSE_P                   { $$ = "false" }
	| ON                        { $$ = "on" }
	| NonReservedWordOrSConst { $$ = $1 }
	;

NumericOnly:
	FCONST
		{
			$$ = &ast.Float{Fval: $1}
		}
	| '+' FCONST
		{
			$$ = &ast.Float{Fval: $2}
		}
	| '-' FCONST
		{
			f := &ast.Float{Fval: $2}
			doNegateFloat(f)
			$$ = f
		}
	| SignedIconst
		{
			$$ = &ast.Integer{Ival: int64($1)}
		}
	;

NumericOnlyList:
	NumericOnly
		{
			$$ = makeList($1)
		}
	| NumericOnlyList ',' NumericOnly
		{
			$$ = appendList($1, $3)
		}
	;

SignedIconst:
	IConst      { $$ = $1 }
	| '+' IConst { $$ = $2 }
	| '-' IConst { $$ = -$2 }
	;

NonReservedWordOrSConst:
	NonReservedWord { $$ = $1 }
	| SConst        { $$ = $1 }
	;

NonReservedWord:
	IDENT                    { $$ = $1 }
	| UnReservedKeyword     { $$ = $1 }
	| ColNameKeyword       { $$ = $1 }
	| TypeFuncNameKeyword { $$ = $1 }
	;

/*****************************************************************************
 *
 *      GRANT / REVOKE statements
 *
 *****************************************************************************/

GrantStmt:
	GRANT Privileges ON PrivilegeTarget TO GranteeList OptGrantGrantOption OptGrantedBy
		{
			t := $4.(*ast.PrivilegeTarget)
			$$ = &ast.GrantStmt{
				ObjectType:  t.ObjectType,
				Objects:     t.Objects,
				TargetNodes: t.TargetNodes,
				Privs:       privElems($2),
				Users:       userSpecs($6),
				GrantOption: $7,
				Grantor:     roleSpecOrNil($8),
			}
		}
	;

RevokeStmt:
	REVOKE Privileges ON PrivilegeTarget FROM GranteeList OptGrantedBy OptDropBehavior
		{
			t := $4.(*ast.PrivilegeTarget)
			$$ = &ast.RevokeStmt{
				ObjectType: t.ObjectType,
				Objects:    t.Objects,
				TargetNodes: t.TargetNodes,
				Privs:      privElems($2),
				Users:      userSpecs($6),
				Behavior:   ast.DropBehavior($8),
				Grantor:    roleSpecOrNil($7),
			}
		}
	| REVOKE GRANT OPTION FOR Privileges ON PrivilegeTarget FROM GranteeList OptGrantedBy OptDropBehavior
		{
			t := $7.(*ast.PrivilegeTarget)
			$$ = &ast.RevokeStmt{
				ObjectType: t.ObjectType,
				Objects:    t.Objects,
				TargetNodes: t.TargetNodes,
				Privs:      privElems($5),
				Users:      userSpecs($9),
				Behavior:   ast.DropBehavior($11),
				GrantOption: true,
				Grantor:    roleSpecOrNil($10),
			}
		}
	;

PrivilegeTarget:
	AnyNameList
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeTable,
				Objects:    tableNamesFromNameList($1),
			}
		}
	| TABLE AnyNameList
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeTable,
				Objects:    tableNamesFromNameList($2),
			}
		}
	| SEQUENCE AnyNameList
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeSequence,
				Objects:    tableNamesFromNameList($2),
			}
		}
	| FOREIGN DATA_P WRAPPER NameList
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeForeignDataWrapper,
				Objects:    tableNamesFromNameList($4),
			}
		}
	| FOREIGN SERVER NameList
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeForeignServer,
				Objects:    tableNamesFromNameList($3),
			}
		}
	| FUNCTION FunctionWithArgTypesList
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeFunction,
				Objects:    objectWithArgsNames($2),
				TargetNodes: $2,
			}
		}
	| PROCEDURE FunctionWithArgTypesList
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeProcedure,
				Objects:    objectWithArgsNames($2),
				TargetNodes: $2,
			}
		}
	| ROUTINE FunctionWithArgTypesList
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeRoutine,
				Objects:    objectWithArgsNames($2),
				TargetNodes: $2,
			}
		}
	| TYPE_P AnyNameList
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeUserType,
				Objects:    tableNamesFromNameList($2),
			}
		}
	| DOMAIN_P AnyNameList
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeDomain,
				Objects:    tableNamesFromNameList($2),
			}
		}
	| LANGUAGE NameList
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeLanguage,
				Objects:    tableNamesFromNameList($2),
			}
		}
	| LARGE_P OBJECT_P NumericOnlyList
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeLargeObject,
				TargetNodes: $3,
			}
		}
	| SCHEMA NameList
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeSchema,
				Objects:    tableNamesFromNameList($2),
			}
		}
	| DATABASE NameList
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeDatabase,
				Objects:    tableNamesFromNameList($2),
			}
		}
	| TABLESPACE NameList
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeTablespace,
				Objects:    tableNamesFromNameList($2),
			}
		}
	| ALL TABLES IN_P SCHEMA NameList
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeAllTablesInSchema,
				Objects:    tableNamesFromNameList($5),
			}
		}
	| ALL SEQUENCES IN_P SCHEMA NameList
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeAllSequencesInSchema,
				Objects:    tableNamesFromNameList($5),
			}
		}
	| ALL FUNCTIONS IN_P SCHEMA NameList
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeAllFunctionsInSchema,
				Objects:    tableNamesFromNameList($5),
			}
		}
	| ALL PROCEDURES IN_P SCHEMA NameList
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeAllProceduresInSchema,
				Objects:    tableNamesFromNameList($5),
			}
		}
	| ALL ROUTINES IN_P SCHEMA NameList
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeAllRoutinesInSchema,
				Objects:    tableNamesFromNameList($5),
			}
		}
	;

Privileges:
	ALL PRIVILEGES '(' ColumnList ')'
		{
			ap := &ast.AccessPriv{Cols: $4}
			$$ = makeList(ap)
		}
	| ALL '(' ColumnList ')'
		{
			ap := &ast.AccessPriv{Cols: $3}
			$$ = makeList(ap)
		}
	| ALL PRIVILEGES  { $$ = nil }
	| ALL           { $$ = nil }
	| PrivilegeList { $$ = $1 }
	;

PrivilegeList:
	Privilege
		{ $$ = makeList($1) }
	| PrivilegeList ',' Privilege
		{ $$ = appendList($1, $3) }
	;

Privilege:
	SELECT OptColumnList
		{ $$ = &ast.AccessPriv{PrivName: "select", Cols: $2} }
	| REFERENCES OptColumnList
		{ $$ = &ast.AccessPriv{PrivName: "references", Cols: $2} }
	| CREATE OptColumnList
		{ $$ = &ast.AccessPriv{PrivName: "create", Cols: $2} }
	| ALTER SYSTEM_P
		{ $$ = &ast.AccessPriv{PrivName: "alter system"} }
	| ColId OptColumnList
		{ $$ = &ast.AccessPriv{PrivName: $1, Cols: $2} }
	;

GranteeList:
	Grantee
		{ $$ = makeList($1) }
	| GranteeList ',' Grantee
		{ $$ = appendList($1, $3) }
	;

Grantee:
	RoleSpec { $$ = $1 }
	| GROUP_P RoleSpec { $$ = $2 }
	;

RoleSpec:
	ColId
		{
			$$ = &ast.RoleSpec{
				Roletype: int(ast.ROLESPEC_CSTRING),
				Rolename: syntaxIdent(yylex, $1, yyS[yypt].span),
			}
			if $1 == "public" && $$.(*ast.RoleSpec).Rolename.Quote == lex.QuoteUnquoted {
				$$.(*ast.RoleSpec).Roletype = int(ast.ROLESPEC_PUBLIC)
			}
		}
	| CURRENT_ROLE
		{
			$$ = &ast.RoleSpec{
				Roletype: int(ast.ROLESPEC_CURRENT_ROLE),
			}
		}
	| CURRENT_USER
		{
			$$ = &ast.RoleSpec{
				Roletype: int(ast.ROLESPEC_CURRENT_USER),
			}
		}
	| SESSION_USER
		{
			$$ = &ast.RoleSpec{
				Roletype: int(ast.ROLESPEC_SESSION_USER),
			}
		}
	;

OptGrantGrantOption:
	WITH GRANT OPTION { $$ = true }
	| /* EMPTY */     { $$ = false }
	;

/*****************************************************************************
 *
 *      GRANT / REVOKE ROLE statements
 *
 *****************************************************************************/

GrantRoleStmt:
	GRANT PrivilegeList TO RoleList OptGrantedBy
		{
			$$ = &ast.GrantRoleStmt{
				Roles: grantRoleNames($2, yylex),
				Users: $4,
				Grantor: roleSpecOrNil($5),
			}
		}
	| GRANT PrivilegeList TO RoleList WITH GrantRoleOptList OptGrantedBy
		{
			$$ = &ast.GrantRoleStmt{
				Roles: grantRoleNames($2, yylex),
				Users: $4,
				Options: $6,
				Grantor: roleSpecOrNil($7),
			}
		}
	;

RevokeRoleStmt:
	REVOKE PrivilegeList FROM RoleList OptGrantedBy OptDropBehavior
		{
			$$ = &ast.RevokeRoleStmt{
				Roles: grantRoleNames($2, yylex),
				Users: $4,
				Grantor: roleSpecOrNil($5),
				Behavior: ast.DropBehavior($6),
			}
		}
	| REVOKE ColId OPTION FOR PrivilegeList FROM RoleList OptGrantedBy OptDropBehavior
		{
			$$ = &ast.RevokeRoleStmt{
				Roles: grantRoleNames($5, yylex),
				Users: $7,
				GrantOption: $2,
				Grantor: roleSpecOrNil($8),
				Behavior: ast.DropBehavior($9),
			}
		}
	;

GrantRoleOptList:
	GrantRoleOptList ',' GrantRoleOpt
		{ $$ = appendList($1, $3) }
	| GrantRoleOpt
		{ $$ = makeList($1) }
	;

GrantRoleOpt:
	ColLabel GrantRoleOptValue
		{
			$$ = makeDefElem($1, $2)
		}
	;

GrantRoleOptValue:
	OPTION   { $$ = &ast.Boolean{Boolval: true} }
	| TRUE_P { $$ = &ast.Boolean{Boolval: true} }
	| FALSE_P { $$ = &ast.Boolean{Boolval: false} }
	;

OptGrantedBy:
	GRANTED BY RoleSpec { $$ = $3 }
	| /* EMPTY */       { $$ = nil }
	;

/*****************************************************************************
 *
 *      CREATE ROLE / USER / GROUP
 *
 *****************************************************************************/

CreateRoleStmt:
	CREATE ROLE RoleId OptWithOptions OptRoleList
		{
			$$ = createUserStmt(true, syntaxIdent(yylex, $3, yyS[yypt-2].span), $5)
		}
	;

CreateUserStmt:
	CREATE USER RoleId OptWithOptions OptRoleList
		{
			$$ = createUserStmt(false, syntaxIdent(yylex, $3, yyS[yypt-2].span), $5)
		}
	;

CreateGroupStmt:
	CREATE GROUP_P RoleId OptWithOptions OptRoleList
		{
			$$ = &ast.CreateRoleStmt{
				StmtType: ast.ROLESTMT_GROUP,
				Role:     $3,
				Options:  $5,
			}
		}
	;

OptWithOptions:
	WITH      {}
	| WITH_LA {}
	| /* EMPTY */ {}
	;

OptRoleList:
	OptRoleList CreateOptRoleElem
		{
			$$ = appendList($1, $2)
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

AlterOptRoleList:
	AlterOptRoleList AlterOptRoleElem
		{
			$$ = appendList($1, $2)
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

AlterOptRoleElem:
	PASSWORD SConst
		{
			$$ = makeDefElem("password", &ast.String{Str: $2})
		}
	| PASSWORD NULL_P
		{
			$$ = makeDefElem("password", nil)
		}
	| ENCRYPTED PASSWORD SConst
		{
			$$ = makeDefElem("password", &ast.String{Str: $3})
		}
	| UNENCRYPTED PASSWORD SConst
		{
			yylex.Error("UNENCRYPTED PASSWORD is no longer supported")
			$$ = nil
		}
	| INHERIT
		{
			$$ = makeDefElem("inherit", &ast.Boolean{Boolval: true})
		}
	| CONNECTION LIMIT SignedIconst
		{
			$$ = makeDefElem("connectionlimit", &ast.Integer{Ival: int64($3)})
		}
	| VALID UNTIL SConst
		{
			$$ = makeDefElem("validUntil", &ast.String{Str: $3})
		}
	| USER RoleList
		{
			$$ = makeDefElem("rolemembers", $2)
		}
	| IDENT
		{
			switch $1 {
			case "superuser":
				$$ = makeDefElem("superuser", &ast.Boolean{Boolval: true})
			case "nosuperuser":
				$$ = makeDefElem("superuser", &ast.Boolean{Boolval: false})
			case "createrole":
				$$ = makeDefElem("createrole", &ast.Boolean{Boolval: true})
			case "nocreaterole":
				$$ = makeDefElem("createrole", &ast.Boolean{Boolval: false})
			case "replication":
				$$ = makeDefElem("isreplication", &ast.Boolean{Boolval: true})
			case "noreplication":
				$$ = makeDefElem("isreplication", &ast.Boolean{Boolval: false})
			case "createdb":
				$$ = makeDefElem("createdb", &ast.Boolean{Boolval: true})
			case "nocreatedb":
				$$ = makeDefElem("createdb", &ast.Boolean{Boolval: false})
			case "login":
				$$ = makeDefElem("canlogin", &ast.Boolean{Boolval: true})
			case "nologin":
				$$ = makeDefElem("canlogin", &ast.Boolean{Boolval: false})
			case "bypassrls":
				$$ = makeDefElem("bypassrls", &ast.Boolean{Boolval: true})
			case "nobypassrls":
				$$ = makeDefElem("bypassrls", &ast.Boolean{Boolval: false})
			case "noinherit":
				$$ = makeDefElem("inherit", &ast.Boolean{Boolval: false})
			default:
				yylex.Error("unrecognized role option \"" + $1 + "\"")
				$$ = nil
			}
		}
	;

CreateOptRoleElem:
	AlterOptRoleElem
		{
			$$ = $1
		}
	| SYSID IConst
		{
			$$ = makeDefElem("sysid", &ast.Integer{Ival: int64($2)})
		}
	| ADMIN RoleList
		{
			$$ = makeDefElem("adminmembers", $2)
		}
	| ROLE RoleList
		{
			$$ = makeDefElem("rolemembers", $2)
		}
	| IN_P ROLE RoleList
		{
			$$ = makeDefElem("addroleto", $3)
		}
	| IN_P GROUP_P RoleList
		{
			$$ = makeDefElem("addroleto", $3)
		}
	;

/*****************************************************************************
 *
 *      ALTER ROLE / USER (with inline opt_with to avoid reduce-reduce conflicts)
 *
 *****************************************************************************/

AlterRoleStmt:
	ALTER ROLE RoleSpec
		{
			$$ = alterUserStmt(true, $3, nil)
		}
	| ALTER ROLE RoleSpec WITH AlterOptRoleList
		{
			$$ = alterUserStmt(true, $3, $5)
		}
	| ALTER ROLE RoleSpec AlterOptRoleElem AlterOptRoleList
		{
			$$ = alterUserStmt(true, $3, prependList($4, $5))
		}
	| ALTER USER RoleSpec WITH AlterOptRoleList
		{
			$$ = alterUserStmt(false, $3, $5)
		}
	| ALTER USER RoleSpec AlterOptRoleElem AlterOptRoleList
		{
			$$ = alterUserStmt(false, $3, prependList($4, $5))
		}
	| ALTER GROUP_P RoleSpec WITH AlterOptRoleList
		{
			$$ = alterUserStmt(true, $3, $5)
		}
	| ALTER GROUP_P RoleSpec AlterOptRoleElem AlterOptRoleList
		{
			$$ = alterUserStmt(true, $3, prependList($4, $5))
		}
	;

/*****************************************************************************
 *
 *      ALTER ROLE SET/RESET (inlined opt_in_database to avoid reduce-reduce conflict)
 *
 *****************************************************************************/

AlterRoleSetStmt:
	ALTER ROLE RoleSpec SetResetClause
		{
			$$ = &ast.AlterRoleSetStmt{
				Role:    $3.(*ast.RoleSpec),
				Setstmt: $4,
			}
		}
	| ALTER ROLE RoleSpec IN_P DATABASE Name SetResetClause
		{
			$$ = &ast.AlterRoleSetStmt{
				Role:     $3.(*ast.RoleSpec),
				Database: $6,
				Setstmt:  $7,
			}
		}
	| ALTER ROLE ALL SetResetClause
		{
			$$ = &ast.AlterRoleSetStmt{
				Setstmt: $4,
			}
		}
	| ALTER ROLE ALL IN_P DATABASE Name SetResetClause
		{
			$$ = &ast.AlterRoleSetStmt{
				Database: $6,
				Setstmt:  $7,
			}
		}
	| ALTER USER RoleSpec SetResetClause
		{
			$$ = &ast.AlterRoleSetStmt{
				Role:    $3.(*ast.RoleSpec),
				Setstmt: $4,
			}
		}
	| ALTER USER RoleSpec IN_P DATABASE Name SetResetClause
		{
			$$ = &ast.AlterRoleSetStmt{
				Role:     $3.(*ast.RoleSpec),
				Database: $6,
				Setstmt:  $7,
			}
		}
	| ALTER USER ALL SetResetClause
		{
			$$ = &ast.AlterRoleSetStmt{
				Setstmt: $4,
			}
		}
	| ALTER USER ALL IN_P DATABASE Name SetResetClause
		{
			$$ = &ast.AlterRoleSetStmt{
				Database: $6,
				Setstmt:  $7,
			}
		}
	;

SetResetClause:
	SET SetRest
		{
			$$ = $2
		}
	| VariableResetStmt
		{
			$$ = $1
		}
	;

/*****************************************************************************
 *
 *      DROP ROLE / USER / GROUP
 *
 *****************************************************************************/

DropRoleStmt:
	DROP ROLE RoleList
		{
			$$ = dropUserStmt(true, false, $3)
		}
	| DROP ROLE IF_P EXISTS RoleList
		{
			$$ = dropUserStmt(true, true, $5)
		}
	| DROP USER RoleList
		{
			$$ = dropUserStmt(false, false, $3)
		}
	| DROP USER IF_P EXISTS RoleList
		{
			$$ = dropUserStmt(false, true, $5)
		}
	| DROP GROUP_P RoleList
		{
			$$ = dropUserStmt(true, false, $3)
		}
	| DROP GROUP_P IF_P EXISTS RoleList
		{
			$$ = dropUserStmt(true, true, $5)
		}
	;

/*****************************************************************************
 *
 *      ALTER GROUP
 *
 *****************************************************************************/

AlterGroupStmt:
	ALTER GROUP_P RoleSpec AddDrop USER RoleList
		{
			$$ = &ast.AlterRoleStmt{
				Role:   $3.(*ast.RoleSpec),
				Action: int($4),
				Options: makeList(makeDefElem("rolemembers", $6)),
			}
		}
	;

AddDrop:
	ADD_P  { $$ = 1 }
	| DROP { $$ = -1 }
	;

/*****************************************************************************
 *
 *      CREATE DATABASE statement
 *
 *****************************************************************************/

CreateDatabaseStmt:
	CREATE DATABASE Name OptWithOptions CreateDatabaseOptList
		{
			$$ = &ast.CreateDatabaseStmt{
				Name:    syntaxIdent(yylex, $3, yyS[yypt-2].span),
				Options: databaseOptions($5),
			}
		}
	| CREATE DATABASE IF_P NOT EXISTS Name OptWithOptions CreateDatabaseOptList
		{
			$$ = &ast.CreateDatabaseStmt{
				Name:        syntaxIdent(yylex, $6, yyS[yypt-2].span),
				Options:     databaseOptions($8),
				IfNotExists: true,
			}
		}
	;

CreateDatabaseOptList:
	CreateDatabaseOptItems
		{ $$ = $1 }
	| /* EMPTY */
		{ $$ = nil }
	;

CreateDatabaseOptItems:
	CreateDatabaseOptItem
		{ $$ = makeList($1) }
	| CreateDatabaseOptItems CreateDatabaseOptItem
		{ $$ = appendList($1, $2) }
	;

CreateDatabaseOptItem:
	CreateDatabaseOptName OptEqual NumericOnly
		{
			$$ = makeDefElem($1, $3)
		}
	| CreateDatabaseOptName OptEqual OptBooleanOrString
		{
			$$ = makeDefElem($1, &ast.String{Str: $3})
		}
	| CreateDatabaseOptName OptEqual DEFAULT
		{
			$$ = makeDefElem($1, nil)
		}
	;

/*
 * Ideally we'd use ColId here, but that causes shift/reduce conflicts against
 * the ALTER DATABASE SET/RESET syntaxes. Instead call out specific keywords
 * we need, and allow IDENT so that database option names don't have to be
 * parser keywords unless they are already keywords for other reasons.
 */
CreateDatabaseOptName:
	IDENT                  { $$ = $1 }
	| CONNECTION LIMIT     { $$ = "connection_limit" }
	| ENCODING             { $$ = "encoding" }
	| LOCATION             { $$ = "location" }
	| OWNER                { $$ = "owner" }
	| TABLESPACE           { $$ = "tablespace" }
	| TEMPLATE             { $$ = "template" }
	;

/*
 * Though the equals sign doesn't match other WITH options, pg_dump uses
 * equals for backward compatibility, and it doesn't seem worth removing it.
 */
OptEqual:
	'='
		{}
	| /* EMPTY */
		{}
	;

/*****************************************************************************
 *
 *      ALTER DATABASE
 *
 *****************************************************************************/

AlterDatabaseStmt:
	ALTER DATABASE Name WITH CreateDatabaseOptList
		{
			$$ = &ast.AlterDatabaseStmt{
				Name:  syntaxIdent(yylex, $3, yyS[yypt-2].span),
				Options: defElemsToDatabaseOptions($5),
			}
		}
	| ALTER DATABASE Name CreateDatabaseOptList
		{
			$$ = &ast.AlterDatabaseStmt{
				Name:  syntaxIdent(yylex, $3, yyS[yypt-1].span),
				Options: defElemsToDatabaseOptions($4),
			}
		}
	| ALTER DATABASE Name SET TABLESPACE Name
		{
			$$ = &ast.AlterDatabaseStmt{
				Name:  syntaxIdent(yylex, $3, yyS[yypt-3].span),
				Tablespace: syntaxIdent(yylex, $6, yyS[yypt].span),
			}
		}
	;

AlterDatabaseSetStmt:
	ALTER DATABASE Name SetResetClause
		{
			$$ = &ast.AlterDatabaseSetStmt{
				Dbname:  syntaxIdent(yylex, $3, yyS[yypt-1].span),
				Setstmt: $4,
			}
		}
	;

/*****************************************************************************
 *
 *      DROP DATABASE [ IF EXISTS ] dbname [ [ WITH ] ( options ) ]
 *
 *****************************************************************************/

DropDatabaseStmt:
	DROP DATABASE Name
		{
			$$ = &ast.DropdbStmt{
				Dbname:    syntaxIdent(yylex, $3, yyS[yypt].span),
				MissingOk: false,
			}
		}
	| DROP DATABASE IF_P EXISTS Name
		{
			$$ = &ast.DropdbStmt{
				Dbname:    syntaxIdent(yylex, $5, yyS[yypt].span),
				MissingOk: true,
			}
		}
	| DROP DATABASE Name OptWithOptions '(' DropOptionList ')'
		{
			$$ = &ast.DropdbStmt{
				Dbname:    syntaxIdent(yylex, $3, yyS[yypt-5].span),
				MissingOk: false,
				Options:   $6,
			}
		}
	| DROP DATABASE IF_P EXISTS Name OptWithOptions '(' DropOptionList ')'
		{
			$$ = &ast.DropdbStmt{
				Dbname:    syntaxIdent(yylex, $5, yyS[yypt-7].span),
				MissingOk: true,
				Options:   $8,
			}
		}
	;

DropOptionList:
	DropOption
		{ $$ = makeList($1) }
	| DropOptionList ',' DropOption
		{ $$ = appendList($1, $3) }
	;

DropOption:
	FORCE
		{
			$$ = makeDefElem("force", nil)
		}
	;

/*****************************************************************************
 *
 *      ALTER SYSTEM
 *
 *****************************************************************************/

AlterSystemStmt:
	ALTER SYSTEM_P SET GenericSet
		{
			$$ = &ast.AlterSystemStmt{
				Setstmt: $4,
			}
		}
	| ALTER SYSTEM_P RESET GenericReset
		{
			$$ = &ast.AlterSystemStmt{
				Setstmt: $4,
			}
		}
	;

/*****************************************************************************
 *
 *      CREATE SCHEMA
 *
 *****************************************************************************/

CreateSchemaStmt:
	CREATE SCHEMA OptSingleName AUTHORIZATION RoleSpec OptSchemaEltList
		{
			$$ = &ast.CreateSchemaStmt{
				Name:          syntaxIdent(yylex, $3, yyS[yypt-3].span),
				Authorization: $5.(*ast.RoleSpec),
			}
		}
	| CREATE SCHEMA ColId OptSchemaEltList
		{
			$$ = &ast.CreateSchemaStmt{
				Name: syntaxIdent(yylex, $3, yyS[yypt-1].span),
			}
		}
	| CREATE SCHEMA IF_P NOT EXISTS OptSingleName AUTHORIZATION RoleSpec OptSchemaEltList
		{
			$$ = &ast.CreateSchemaStmt{
				Name:          syntaxIdent(yylex, $6, yyS[yypt-3].span),
				Authorization: $8.(*ast.RoleSpec),
				IfNotExists:   true,
			}
		}
	| CREATE SCHEMA IF_P NOT EXISTS ColId OptSchemaEltList
		{
			$$ = &ast.CreateSchemaStmt{
				Name:        syntaxIdent(yylex, $6, yyS[yypt-1].span),
				IfNotExists: true,
			}
		}
	;

OptSchemaEltList:
	OptSchemaEltList SchemaStmt
		{
			$$ = appendList($1, $2)
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

SchemaStmt:
	CreateTableStmt        { $$ = $1 }
	| IndexStmt       { $$ = $1 }
	| CreateSequenceStmt   { $$ = $1 }
	| CreateTriggerStmt  { $$ = $1 }
	| CreateViewStmt        { $$ = $1 }
	| GrantStmt       { $$ = $1 }
	;

/*****************************************************************************
 *
 *      CREATE SEQUENCE / ALTER SEQUENCE
 *
 *****************************************************************************/

CreateSequenceStmt:
	CREATE OptTemp SEQUENCE QualifiedName OptSequenceOptList
		{
			rv := makeRangeVar($4)
			relationName(rv).Relpersistence = relpersistenceForTemp($2)
			$$ = &ast.CreateSequenceStmt{
				Name:    relationName(rv),
				Options: $5,
			}
		}
	| CREATE OptTemp SEQUENCE IF_P NOT EXISTS QualifiedName OptSequenceOptList
		{
			rv := makeRangeVar($7)
			relationName(rv).Relpersistence = relpersistenceForTemp($2)
			$$ = &ast.CreateSequenceStmt{
				Name:        relationName(rv),
				Options:     $8,
				IfNotExists: true,
			}
		}
	;

AlterSequenceStmt:
	ALTER SEQUENCE QualifiedName SequenceOptList
		{
			rv := makeRangeVar($3)
			$$ = &ast.AlterSeqStmt{
				Sequence: relationName(rv),
				Options:  $4,
			}
		}
	| ALTER SEQUENCE IF_P EXISTS QualifiedName SequenceOptList
		{
			rv := makeRangeVar($5)
			$$ = &ast.AlterSeqStmt{
				Sequence:  relationName(rv),
				Options:   $6,
				MissingOk: true,
			}
		}
	;

OptParenthesizedSequenceOptList:
	'(' SequenceOptList ')'	{ $$ = $2 }
	| /* EMPTY */		{ $$ = nil }
	;

OptSequenceOptList:
	SequenceOptList  { $$ = $1 }
	| /* EMPTY */ { $$ = nil }
	;

SequenceOptList:
	SequenceOptElem
		{ $$ = makeList($1) }
	| SequenceOptList SequenceOptElem
		{ $$ = appendList($1, $2) }
	;

SequenceOptElem:
	AS SimpleTypename
		{
			$$ = makeDefElem("as", typeNameNode($2))
		}
	| CACHE NumericOnly
		{
			$$ = makeDefElem("cache", $2)
		}
	| CYCLE
		{
			$$ = makeDefElem("cycle", &ast.Boolean{Boolval: true})
		}
	| NO CYCLE
		{
			$$ = makeDefElem("cycle", &ast.Boolean{Boolval: false})
		}
	| INCREMENT OptBy NumericOnly
		{
			$$ = makeDefElem("increment", $3)
		}
	| MAXVALUE NumericOnly
		{
			$$ = makeDefElem("maxvalue", $2)
		}
	| MINVALUE NumericOnly
		{
			$$ = makeDefElem("minvalue", $2)
		}
	| NO MAXVALUE
		{
			$$ = makeDefElem("maxvalue", nil)
		}
	| NO MINVALUE
		{
			$$ = makeDefElem("minvalue", nil)
		}
	| OWNED BY AnyName
		{
			$$ = makeDefElem("owned_by", $3)
		}
	| SEQUENCE NAME_P AnyName
		{
			$$ = makeDefElem("sequence_name", $3)
		}
	| START OptWithOptions NumericOnly
		{
			$$ = makeDefElem("start", $3)
		}
	| RESTART
		{
			$$ = makeDefElem("restart", nil)
		}
	| RESTART OptWithOptions NumericOnly
		{
			$$ = makeDefElem("restart", $3)
		}
	| LOGGED
		{
			$$ = makeDefElem("logged", &ast.Boolean{Boolval: true})
		}
	| UNLOGGED
		{
			$$ = makeDefElem("logged", &ast.Boolean{Boolval: false})
		}
	;

OptBy:
	BY     { /* nothing */ }
	| /* EMPTY */ { /* nothing */ }
	;

GeneratedWhen:
	ALWAYS      { $$ = int64('a') }
	| BY DEFAULT { $$ = int64('d') }
	;

AlterIdentityColumnOptionList:
	AlterIdentityColumnOption
		{ $$ = makeList($1) }
	| AlterIdentityColumnOptionList AlterIdentityColumnOption
		{ $$ = appendList($1, $2) }
	;

AlterIdentityColumnOption:
	RESTART
		{
			$$ = makeDefElem("restart", nil)
		}
	| RESTART OptWithOptions NumericOnly
		{
			$$ = makeDefElem("restart", $3)
		}
	| SET SequenceOptElem
		{
			$$ = $2
		}
	| SET GENERATED GeneratedWhen
		{
			$$ = makeDefElem("generated", makeIntConst($3))
		}
	;

/*****************************************************************************
 *
 *      CREATE DOMAIN
 *
 *****************************************************************************/

CreateDomainStmt:
	CREATE DOMAIN_P AnyName OptAs Typename OptColumnConstraints
		{
			$$ = &ast.CreateDomainStmt{
				Domainname:  $3,
				Typname:     $5,
				Constraints: $6,
			}
		}
	;

OptAs:
	AS     { /* nothing */ }
	| /* EMPTY */ { /* nothing */ }
	;

/*****************************************************************************
 *
 *      ALTER DOMAIN
 *
 *****************************************************************************/

AlterDomainStmt:
	ALTER DOMAIN_P AnyName AlterColumnDefault
		{
			n := $4.(*ast.AlterDomainStmt)
			n.Typname = $3
			$$ = n
		}
	| ALTER DOMAIN_P AnyName DROP NOT NULL_P
		{
			$$ = &ast.AlterDomainStmt{
				Subtype: 'N',
				Typname: $3,
			}
		}
	| ALTER DOMAIN_P AnyName SET NOT NULL_P
		{
			$$ = &ast.AlterDomainStmt{
				Subtype: 'O',
				Typname: $3,
			}
		}
	| ALTER DOMAIN_P AnyName ADD_P DomainConstraint
		{
			$$ = &ast.AlterDomainStmt{
				Subtype: 'C',
				Typname: $3,
				Def:     $5,
			}
		}
	| ALTER DOMAIN_P AnyName DROP CONSTRAINT Name OptDropBehavior
		{
			$$ = &ast.AlterDomainStmt{
				Subtype:  'X',
				Typname:  $3,
				Name:     $6,
				Behavior: ast.DropBehavior($7),
			}
		}
	| ALTER DOMAIN_P AnyName DROP CONSTRAINT IF_P EXISTS Name OptDropBehavior
		{
			$$ = &ast.AlterDomainStmt{
				Subtype:   'X',
				Typname:   $3,
				Name:      $8,
				Behavior:  ast.DropBehavior($9),
				MissingOk: true,
			}
		}
	| ALTER DOMAIN_P AnyName VALIDATE CONSTRAINT Name
		{
			$$ = &ast.AlterDomainStmt{
				Subtype: 'V',
				Typname: $3,
				Name:    $6,
			}
		}
	;

AlterColumnDefault:
	SET DEFAULT AExpr
		{
			$$ = &ast.AlterDomainStmt{
				Subtype: 'T',
				Def:     $3,
			}
		}
	| DROP DEFAULT
		{
			$$ = &ast.AlterDomainStmt{
				Subtype: 'T',
			}
		}
	;

/*****************************************************************************
 *
 *      ALTER TYPE ... ENUM / ALTER TYPE ... COMPOSITE / ALTER COLLATION
 *
 *****************************************************************************/

AlterEnumStmt:
	ALTER TYPE_P AnyName ADD_P VALUE_P OptIfNotExists SConst
		{
			$$ = &ast.AlterEnumStmt{
				Typname:            $3,
				Newval:             $7,
				SkipIfNewvalExists: $6,
			}
		}
	| ALTER TYPE_P AnyName ADD_P VALUE_P OptIfNotExists SConst BEFORE SConst
		{
			$$ = &ast.AlterEnumStmt{
				Typname:            $3,
				Newval:             $7,
				NewvalNeighbor:     $9,
				NewvalIsAfter:      false,
				SkipIfNewvalExists: $6,
			}
		}
	| ALTER TYPE_P AnyName ADD_P VALUE_P OptIfNotExists SConst AFTER SConst
		{
			$$ = &ast.AlterEnumStmt{
				Typname:            $3,
				Newval:             $7,
				NewvalNeighbor:     $9,
				NewvalIsAfter:      true,
				SkipIfNewvalExists: $6,
			}
		}
	| ALTER TYPE_P AnyName RENAME VALUE_P SConst TO SConst
		{
			$$ = &ast.AlterEnumStmt{
				Typname: $3,
				Oldval:  $6,
				Newval:  $8,
			}
		}
	;

OptIfNotExists:
	IF_P NOT EXISTS  { $$ = true }
	| /* EMPTY */    { $$ = false }
	;

AlterCollationStmt:
	ALTER COLLATION AnyName REFRESH VERSION_P
		{
			$$ = &ast.AlterCollationStmt{
				Collname: $3,
			}
		}
	;

AlterCompositeTypeStmt:
	ALTER TYPE_P AnyName AlterTypeCmds
		{
			rv := &ast.RangeVar{
				Inh:     true,
				Location: -1,
			}
			/* Convert any_name to schema.rel */
			names := $3
			if names != nil && len(names.Items) > 0 {
				switch len(names.Items) {
				case 1:
					rv.Relname = names.Items[0].(*ast.String).Str
				case 2:
					rv.Schemaname = names.Items[0].(*ast.String).Str
					rv.Relname = names.Items[1].(*ast.String).Str
				case 3:
					rv.Catalogname = names.Items[0].(*ast.String).Str
					rv.Schemaname = names.Items[1].(*ast.String).Str
					rv.Relname = names.Items[2].(*ast.String).Str
				}
			}
			$$ = &ast.AlterTableStmt{
				Objtype: "TYPE",
				Table: relationName(rv),
				Specs:    alterTableSpecs($4),
			}
		}
	;

AlterTypeCmds:
	AlterTypeCmd
		{ $$ = makeList($1) }
	| AlterTypeCmds ',' AlterTypeCmd
		{ $$ = appendList($1, $3) }
	;

AlterTypeCmd:
	ADD_P ATTRIBUTE TableFuncElement OptDropBehavior
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableAddColumns,
				Attribute: true,
				NewColumns: []*ast.ColumnDef{$3.(*ast.ColumnDef)},
				Behavior: ast.DropBehavior(int($4)),
			}
		}
	| DROP ATTRIBUTE ColId OptDropBehavior
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableDropColumn,
				Attribute: true,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				Behavior: ast.DropBehavior(int($4)),
			}
		}
	| DROP ATTRIBUTE IF_P EXISTS ColId OptDropBehavior
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableDropColumn,
				Attribute: true,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($5)},
				Behavior: ast.DropBehavior(int($6)),
				MissingOk: true,
			}
		}
	| ALTER ATTRIBUTE ColId SET DATA_P TYPE_P Typename OptCollateClause OptDropBehavior
		{
			coldef := &ast.ColumnDef{
				Name: &ast.ColumnName{Name: model.NewCIStr($3)},
				Tp: $7,
				}
			if $8 != nil {
				coldef.Options = append(coldef.Options, &ast.ColumnOption{Tp: ast.ColumnOptionCollate, StrValue: nameListString($8)})
			}
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableAlterColumnType,
				Attribute: true,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				NewColumns: []*ast.ColumnDef{coldef},
				Behavior: ast.DropBehavior(int($9)),
			}
		}
	| ALTER ATTRIBUTE ColId TYPE_P Typename OptCollateClause OptDropBehavior
		{
			coldef := &ast.ColumnDef{
				Name: &ast.ColumnName{Name: model.NewCIStr($3)},
				Tp: $5,
				}
			if $6 != nil {
				coldef.Options = append(coldef.Options, &ast.ColumnOption{Tp: ast.ColumnOptionCollate, StrValue: nameListString($6)})
			}
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableAlterColumnType,
				Attribute: true,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				NewColumns: []*ast.ColumnDef{coldef},
				Behavior: ast.DropBehavior(int($7)),
			}
		}
	;

TableFuncElement:
	ColId Typename OptCollateClause
		{
			coldef := &ast.ColumnDef{
				Name: &ast.ColumnName{Name: model.NewCIStr($1)},
				Tp: $2,
				}
			if $3 != nil {
				coldef.Options = append(coldef.Options, &ast.ColumnOption{Tp: ast.ColumnOptionCollate, StrValue: nameListString($3)})
			}
			$$ = coldef
		}
	;

OptCollateClause:
	COLLATE AnyName
		{
			$$ = &ast.CollateClause{
				Collname: $2,
				Location: -1,
			}
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

/*****************************************************************************
 *
 *      DefineStmt - CREATE AGGREGATE/OPERATOR/TYPE/TEXT SEARCH/COLLATION
 *
 *****************************************************************************/

DefineStmt:
	CREATE OptOrReplace AGGREGATE FunctionName AggrArgs Definition
		{
			$$ = &ast.DefineStmt{
				Kind:       ast.OBJECT_AGGREGATE,
				Oldstyle:   false,
				Replace:    $2,
				Defnames:   $4,
				Args:       nodeList($5),
				Definition: $6,
			}
		}
	| CREATE OptOrReplace AGGREGATE FunctionName OldAggrDefinition
		{
			$$ = &ast.DefineStmt{
				Kind:       ast.OBJECT_AGGREGATE,
				Oldstyle:   true,
				Replace:    $2,
				Defnames:   $4,
				Definition: $5,
			}
		}
	| CREATE OPERATOR AnyOperator Definition
		{
			$$ = &ast.DefineStmt{
				Kind:       ast.OBJECT_OPERATOR,
				Defnames:   $3,
				Definition: $4,
			}
		}
	| CREATE TYPE_P AnyName Definition
		{
			$$ = &ast.DefineStmt{
				Kind:       ast.OBJECT_TYPE,
				Defnames:   $3,
				Definition: $4,
			}
		}
	| CREATE TYPE_P AnyName
		{
			/* Shell type (identified by lack of definition) */
			$$ = &ast.DefineStmt{
				Kind:     ast.OBJECT_TYPE,
				Defnames: $3,
			}
		}
	| CREATE TEXT_P SEARCH PARSER AnyName Definition
		{
			$$ = &ast.DefineStmt{
				Kind:       ast.OBJECT_TSPARSER,
				Defnames:   $5,
				Definition: $6,
			}
		}
	| CREATE TEXT_P SEARCH DICTIONARY AnyName Definition
		{
			$$ = &ast.DefineStmt{
				Kind:       ast.OBJECT_TSDICTIONARY,
				Defnames:   $5,
				Definition: $6,
			}
		}
	| CREATE TEXT_P SEARCH TEMPLATE AnyName Definition
		{
			$$ = &ast.DefineStmt{
				Kind:       ast.OBJECT_TSTEMPLATE,
				Defnames:   $5,
				Definition: $6,
			}
		}
	| CREATE TEXT_P SEARCH CONFIGURATION AnyName Definition
		{
			$$ = &ast.DefineStmt{
				Kind:       ast.OBJECT_TSCONFIGURATION,
				Defnames:   $5,
				Definition: $6,
			}
		}
	| CREATE COLLATION AnyName Definition
		{
			$$ = &ast.DefineStmt{
				Kind:       ast.OBJECT_COLLATION,
				Defnames:   $3,
				Definition: $4,
			}
		}
	| CREATE COLLATION IF_P NOT EXISTS AnyName Definition
		{
			$$ = &ast.DefineStmt{
				Kind:        ast.OBJECT_COLLATION,
				Defnames:    $6,
				Definition:  $7,
				IfNotExists: true,
			}
		}
	| CREATE COLLATION AnyName FROM AnyName
		{
			$$ = &ast.DefineStmt{
				Kind:       ast.OBJECT_COLLATION,
				Defnames:   $3,
				Definition: makeList(makeDefElem("from", $5)),
			}
		}
	| CREATE COLLATION IF_P NOT EXISTS AnyName FROM AnyName
		{
			$$ = &ast.DefineStmt{
				Kind:        ast.OBJECT_COLLATION,
				Defnames:    $6,
				Definition:  makeList(makeDefElem("from", $8)),
				IfNotExists: true,
			}
		}
	;

CompositeTypeStmt:
	CREATE TYPE_P AnyName AS '(' OptTableFuncElementList ')'
		{
			$$ = &ast.CompositeTypeStmt{
				Typevar:    makeRangeVarFromAnyName($3),
				Coldeflist: $6,
			}
		}
	;

CreateEnumStmt:
	CREATE TYPE_P AnyName AS ENUM_P '(' OptEnumValList ')'
		{
			$$ = &ast.CreateEnumStmt{
				TypeName: tableNameFromList($3),
				Vals:     $7,
			}
		}
	;

CreateRangeStmt:
	CREATE TYPE_P AnyName AS RANGE Definition
		{
			$$ = &ast.CreateRangeStmt{
				TypeName: tableNameFromList($3),
				Params:   $6,
			}
		}
	;

Definition:
	'(' DefList ')'
		{
			$$ = $2
		}
	;

DefList:
	DefElem
		{
			$$ = makeList($1)
		}
	| DefList ',' DefElem
		{
			$$ = appendList($1, $3)
		}
	;

DefElem:
	ColLabel '=' DefArg
		{
			$$ = makeDefElem($1, $3)
		}
	| ColLabel
		{
			$$ = makeDefElem($1, nil)
		}
	;

/* Note: any simple identifier will be returned as a type name! */
DefArg:
	FuncType
		{
			$$ = typeNameNode($1)
		}
	| ReservedKeyword
		{
			$$ = &ast.String{Str: $1}
		}
	| QualAllOp
		{
			$$ = $1
		}
	| NumericOnly
		{
			$$ = $1
		}
	| SConst
		{
			$$ = &ast.String{Str: $1}
		}
	| NONE
		{
			$$ = &ast.String{Str: "none"}
		}
	;

OldAggrDefinition:
	'(' OldAggrList ')'
		{
			$$ = $2
		}
	;

OldAggrList:
	OldAggrElem
		{
			$$ = makeList($1)
		}
	| OldAggrList ',' OldAggrElem
		{
			$$ = appendList($1, $3)
		}
	;

/*
 * Must use IDENT here to avoid reduce/reduce conflicts; fortunately none of
 * the item names needed in old aggregate definitions are likely to become
 * SQL keywords.
 */
OldAggrElem:
	IDENT '=' DefArg
		{
			$$ = makeDefElem($1, $3)
		}
	;

OptEnumValList:
	EnumValList
		{
			$$ = $1
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

EnumValList:
	SConst
		{
			$$ = makeList(&ast.String{Str: $1})
		}
	| EnumValList ',' SConst
		{
			$$ = appendList($1, &ast.String{Str: $3})
		}
	;

OptTableFuncElementList:
	TableFuncElementList
		{
			$$ = $1
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

TableFuncElementList:
	TableFuncElement
		{
			$$ = makeList($1)
		}
	| TableFuncElementList ',' TableFuncElement
		{
			$$ = appendList($1, $3)
		}
	;

AggrArgs:
	'(' '*' ')'
		{
			/* agg(*) - returns 2-element list: [nil, Integer{-1}] */
			$$ = makeList2(nil, &ast.Integer{Ival: -1})
		}
	| '(' AggrArgsList ')'
		{
			/* normal args - returns 2-element list: [args, Integer{-1}] */
			$$ = makeList2($2, &ast.Integer{Ival: -1})
		}
	| '(' ORDER BY AggrArgsList ')'
		{
			/* ordered-set agg with no direct args - returns 2-element list: [args, Integer{0}] */
			$$ = makeList2($4, &ast.Integer{Ival: 0})
		}
	| '(' AggrArgsList ORDER BY AggrArgsList ')'
		{
			/* ordered-set agg with direct args and ordered args */
			$$ = makeOrderedSetArgs($2, $5)
		}
	;

AggrArgsList:
	AggrArg
		{
			$$ = makeList($1)
		}
	| AggrArgsList ',' AggrArg
		{
			$$ = appendList($1, $3)
		}
	;

AggrArg:
	FuncArg
		{
			$$ = $1
		}
	;

/*****************************************************************************
 *
 *      any_operator, all_Op, MathOp, qual_all_Op
 *
 *****************************************************************************/

AnyOperator:
	AllOp
		{
			$$ = makeList(&ast.String{Str: $1})
		}
	| ColId '.' AnyOperator
		{
			$$ = prependList(&ast.String{Str: $1}, $3)
		}
	;

AllOp:
	Op      { $$ = $1 }
	| MathOp { $$ = $1 }
	;

MathOp:
	'+'             { $$ = "+" }
	| '-'           { $$ = "-" }
	| '*'           { $$ = "*" }
	| '/'           { $$ = "/" }
	| '%'           { $$ = "%" }
	| '^'           { $$ = "^" }
	| '<'           { $$ = "<" }
	| '>'           { $$ = ">" }
	| '='           { $$ = "=" }
	| LESS_EQUALS   { $$ = "<=" }
	| GREATER_EQUALS { $$ = ">=" }
	| NOT_EQUALS    { $$ = "<>" }
	;

QualAllOp:
	AllOp
		{
			$$ = makeList(&ast.String{Str: $1})
		}
	| OPERATOR '(' AnyOperator ')'
		{
			$$ = makeList(&ast.String{Str: qualifiedOperatorName($3)})
		}
	;

QualOp:
	Op
		{
			$$ = makeList(&ast.String{Str: $1})
		}
	| OPERATOR '(' AnyOperator ')'
		{
			$$ = makeList(&ast.String{Str: qualifiedOperatorName($3)})
		}
	;

/*****************************************************************************
 *
 *      RoleId and role_list
 *
 *****************************************************************************/

RoleId:
	RoleSpec
		{
			spc := $1.(*ast.RoleSpec)
			if spc.Roletype != int(ast.ROLESPEC_CSTRING) {
				yylex.Error("role name cannot be a reserved keyword here")
			}
			$$ = spc.Rolename.Name
		}
	;

RoleList:
	RoleSpec
		{ $$ = makeList($1) }
	| RoleList ',' RoleSpec
		{ $$ = appendList($1, $3) }
	;

// SELECT statement
SelectStmt:
	SelectNoParens %prec UMINUS
		{
			$$ = $1
		}
	| SelectWithParens %prec UMINUS
		{
			$$ = $1
		}
	;

SelectWithParens:
	'(' SelectNoParens ')'
		{
			$$ = inBraces($2)
		}
	| '(' SelectWithParens ')'
		{
			$$ = inBraces($2)
		}
	;

SelectNoParens:
	SimpleSelect
		{
			$$ = $1
		}
	| SelectClause OrderBy
		{
			$$ = setSelectOptions($1, $2, nil, nil)
		}
	| SelectClause OrderByOpt ForLockingClause OptSelectLimit
		{
			$$ = setSelectOptions($1, $2, $3, $4)
		}
	| SelectClause OrderByOpt SelectLimit OptForLockingClause
		{
			$$ = setSelectOptions($1, $2, $4, $3)
		}
	| WithClause SelectClause
		{
			$$ = setWithClause($2, $1)
		}
	| WithClause SelectClause OrderBy
		{
			$$ = setSelectOptions(setWithClause($2, $1), $3, nil, nil)
		}
	| WithClause SelectClause OrderByOpt ForLockingClause OptSelectLimit
		{
			$$ = setSelectOptions(setWithClause($2, $1), $3, $4, $5)
		}
	| WithClause SelectClause OrderByOpt SelectLimit OptForLockingClause
		{
			$$ = setSelectOptions(setWithClause($2, $1), $3, $5, $4)
		}
	;

SelectClause:
	SimpleSelect
		{
			$$ = $1
		}
	| SelectWithParens
		{
			$$ = $1
		}
	;

SimpleSelect:
	SELECT OptAllClause FieldListOpt IntoClause TableRefsClause WhereClauseOpt GroupByClause HavingClause WindowClauseOpt
		{
			n := &ast.SelectStmt{
				SelectStmtOpts: &ast.SelectStmtOpts{},
				Fields:         &ast.FieldList{Items: $3},
			}
			n.Fields.SetSpan(yyS[yypt-6].span)
			applySelectClauses(n, $5, $6, $7, $8, $9)
			if n.From != nil { n.From.SetSpan(yyS[yypt-4].span) }
			if $4 != nil { n.IntoTable = $4.(*ast.IntoClause) }
			$$ = n
		}
	| SELECT DistinctClause FieldList IntoClause TableRefsClause WhereClauseOpt GroupByClause HavingClause WindowClauseOpt
		{
			n := &ast.SelectStmt{
				Distinct:      true,
				SelectStmtOpts: &ast.SelectStmtOpts{},
				Fields:         &ast.FieldList{Items: $3},
			}
			n.Fields.SetSpan(yyS[yypt-6].span)
			n.DistinctOn = byItemsFromExprs($2)
			applySelectClauses(n, $5, $6, $7, $8, $9)
			if n.From != nil { n.From.SetSpan(yyS[yypt-4].span) }
			if $4 != nil { n.IntoTable = $4.(*ast.IntoClause) }
			$$ = n
		}
	| SelectClause UNION SetQuantifier SelectClause
		{
			$$ = makeSetOp(ast.Union, $3 == SET_QUANTIFIER_ALL, $1, $4)
		}
	| SelectClause INTERSECT SetQuantifier SelectClause
		{
			$$ = makeSetOp(ast.Intersect, $3 == SET_QUANTIFIER_ALL, $1, $4)
		}
	| SelectClause EXCEPT SetQuantifier SelectClause
		{
			$$ = makeSetOp(ast.Except, $3 == SET_QUANTIFIER_ALL, $1, $4)
		}
	| ValuesClause
		{
			$$ = $1
		}
	| TABLE TableName
		{
			$$ = &ast.TableStmt{
				Table: tableNameFromSource($2),
			}
		}
	;

ValuesClause:
	VALUES '(' ExpressionList ')'
		{
			$$ = &ast.ValuesStmt{Rows: []*ast.RowExpr{{Items: $3}}}
		}
	| ValuesClause ',' '(' ExpressionList ')'
		{
			n := $1.(*ast.ValuesStmt)
			n.Rows = append(n.Rows, &ast.RowExpr{Items: $4})
			$$ = n
		}
	;

OptAllClause:
	ALL { $$ = true }
	| /* EMPTY */ { $$ = false }
	;


DistinctClause:
	DISTINCT
		{
			$$ = nil
		}
	| DISTINCT ON '(' ExpressionList ')'
		{
			$$ = $4
		}
	;

SetQuantifier:
	ALL { $$ = SET_QUANTIFIER_ALL }
	| DISTINCT { $$ = SET_QUANTIFIER_DISTINCT }
	| /* EMPTY */ { $$ = SET_QUANTIFIER_DEFAULT }
	;

/*****************************************************************************
 *
 *      WITH clause (Common Table Expressions)
 *
 *****************************************************************************/

OptWithClause:
	WithClause { $$ = $1 }
	| /* EMPTY */ { $$ = nil }
	;

WithClause:
	WITH WithList
		{
			$$ = &ast.WithClause{BaseWithClause: pkast.BaseWithClause{CTEs: $2}}
		}
	| WITH_LA WithList
		{
			$$ = &ast.WithClause{BaseWithClause: pkast.BaseWithClause{CTEs: $2}}
		}
	| WITH RECURSIVE WithList
		{
			$$ = &ast.WithClause{BaseWithClause: pkast.BaseWithClause{CTEs: $3, IsRecursive: true}}
		}
	;

WithList:
	WithClauseItem
		{
			$$ = []ast.CTENode{$1.(ast.CTENode)}
		}
	| WithList ',' WithClauseItem
		{
			$$ = append($1, $3.(ast.CTENode))
		}
	;

WithClauseItem:
	Name OptNameList AS OptMaterialized '(' PreparableStmt ')' OptSearchClause OptCycleClause
		{
			cte := &ast.CommonTableExpression{BaseCommonTableExpression: pkast.BaseCommonTableExpression{
				Name:    syntaxIdent(yylex, $1, yyS[yypt-8].span),
				Columns: syntaxIdents(yylex, $2),
				Query:   $6.(ast.StmtNode),
			}}
			cte.Materialized = ast.CTEMaterialize($4)
			cte.Search = cteSearchClause($8)
			cte.Cycle = cteCycleClause($9)
			$$ = cte
		}
	| Name '(' NameList ')' AS OptMaterialized '(' PreparableStmt ')' OptSearchClause OptCycleClause
		{
			cte := &ast.CommonTableExpression{BaseCommonTableExpression: pkast.BaseCommonTableExpression{
				Name:    syntaxIdent(yylex, $1, yyS[yypt-10].span),
				Columns: syntaxIdents(yylex, $3),
				Query:   $8.(ast.StmtNode),
			}}
			cte.Materialized = ast.CTEMaterialize($6)
			cte.Search = cteSearchClause($10)
			cte.Cycle = cteCycleClause($11)
			$$ = cte
		}
	;

OptMaterialized:
	MATERIALIZED			{ $$ = int64(ast.CTEMaterializeAlways) }
	| NOT MATERIALIZED		{ $$ = int64(ast.CTEMaterializeNever) }
	| /* EMPTY */			{ $$ = int64(ast.CTEMaterializeDefault) }
	;

OptSearchClause:
	SEARCH DEPTH FIRST_P BY ColumnList SET ColId
		{
			$$ = &ast.CTESearchClause{
				SearchColList:      $5,
				SearchBreadthFirst: false,
				SearchSeqColumn:    $7,
				Location:           -1,
			}
		}
	| SEARCH BREADTH FIRST_P BY ColumnList SET ColId
		{
			$$ = &ast.CTESearchClause{
				SearchColList:      $5,
				SearchBreadthFirst: true,
				SearchSeqColumn:    $7,
				Location:           -1,
			}
		}
	| /* EMPTY */ { $$ = nil }
	;

OptCycleClause:
	CYCLE ColumnList SET ColId TO AExprConst DEFAULT AExprConst USING ColId
		{
			$$ = &ast.CTECycleClause{
				CycleColList:     $2,
				CycleMarkColumn:  $4,
				CycleMarkValue:   $6,
				CycleMarkDefault: $8,
				CyclePathColumn:  $10,
				Location:         -1,
			}
		}
	| CYCLE ColumnList SET ColId USING ColId
		{
			$$ = &ast.CTECycleClause{
				CycleColList:     $2,
				CycleMarkColumn:  $4,
				CycleMarkValue:   makeBoolAConst(1),
				CycleMarkDefault: makeBoolAConst(0),
				CyclePathColumn:  $6,
				Location:         -1,
			}
		}
	| /* EMPTY */ { $$ = nil }
	;

FieldListOpt:
	FieldList { $$ = $1 }
	| /* EMPTY */ { $$ = nil }
	;

IntoClause:
	INTO OptTempTableName
		{
			$$ = &ast.IntoClause{
				Rel:            relationName($2),
				OnCommit:       ast.ONCOMMIT_NOOP,
			}
		}
	| /* EMPTY */ { $$ = nil }
	;

OptTempTableName:
	TEMPORARY OptTable QualifiedName  { rv := makeRangeVar($3); rv.(*ast.RangeVar).Relpersistence = 't'; $$ = rv }
	| TEMP OptTable QualifiedName     { rv := makeRangeVar($3); rv.(*ast.RangeVar).Relpersistence = 't'; $$ = rv }
	| LOCAL TEMPORARY OptTable QualifiedName  { rv := makeRangeVar($4); rv.(*ast.RangeVar).Relpersistence = 't'; $$ = rv }
	| LOCAL TEMP OptTable QualifiedName  { rv := makeRangeVar($4); rv.(*ast.RangeVar).Relpersistence = 't'; $$ = rv }
	| GLOBAL TEMPORARY OptTable QualifiedName  { rv := makeRangeVar($4); rv.(*ast.RangeVar).Relpersistence = 't'; $$ = rv }
	| GLOBAL TEMP OptTable QualifiedName  { rv := makeRangeVar($4); rv.(*ast.RangeVar).Relpersistence = 't'; $$ = rv }
	| UNLOGGED OptTable QualifiedName  { rv := makeRangeVar($3); rv.(*ast.RangeVar).Relpersistence = 'u'; $$ = rv }
	| TABLE QualifiedName  { $$ = makeRangeVar($2) }
	| QualifiedName        { $$ = makeRangeVar($1) }
	;

FieldList:
	Field
		{
			$$ = make([]*ast.SelectItem, 1, 4)
			$$[0] = $1.(*ast.SelectItem)
		}
	| FieldList ',' Field
		{
			$$ = append($1, $3.(*ast.SelectItem))
		}
	;

Field:
	AExpr AS ColLabel
		{
			$$ = syntaxSelectItem($1, syntaxIdent(yylex, $3, yyS[yypt].span), yyVAL.span)
		}
	| AExpr IDENT
		{
			$$ = syntaxSelectItem($1, syntaxIdent(yylex, $2, yyS[yypt].span), yyVAL.span)
		}
	| AExpr
		{
			$$ = syntaxSelectItem($1, lex.Ident{Span: loc.InvalidSpan()}, yyVAL.span)
		}
	| '*'
		{
			$$ = syntaxSelectItem(syntaxWildcard(ast.QualifiedName{}, yyVAL.span), lex.Ident{Span: loc.InvalidSpan()}, yyVAL.span)
		}
	;

// FROM clause
TableRefsClause:
	FROM TableRefs { $$ = $2 }
	| /* EMPTY */ { $$ = nil }
	;

TableRefs:
	TableRef
		{
			$$ = []ast.ResultSetNode{$1.(ast.ResultSetNode)}
		}
	| TableRefs ',' TableRef
		{
			$$ = append($1, $3.(ast.ResultSetNode))
		}
	;

TableRef:
	TableName TableAsNameOpt
		{
			ts := $1.(*ast.TableSource)
			applySourceAlias(ts, $2)
			ts.SetSpan(yyVAL.span)
			$$ = ts
		}
	| SelectWithParens TableAsNameOpt
		{
			ts := syntaxTableSource($1.(ast.ResultSetNode), yyVAL.span)
			applySourceAlias(ts, $2)
			$$ = ts
		}
	| JoinTable
		{
			$$ = $1
			$$.SetSpan(yyVAL.span)
		}
	| TableName TableAsNameOpt TablesampleClause
		{
			ts := $1.(*ast.TableSource)
			applySourceAlias(ts, $2)
			ts.TableSample = $3.(*ast.TableSample)
			ts.SetSpan(yyVAL.span)
			$$ = ts
		}
	| FuncTable FuncAliasClause
		{
			ts := $1.(*ast.TableSource)
			applySourceAlias(ts, $2)
			ts.SetSpan(yyVAL.span)
			$$ = ts
		}
	| LATERAL_P FuncTable FuncAliasClause
		{
			ts := $2.(*ast.TableSource)
			ts.Lateral = true
			applySourceAlias(ts, $3)
			ts.SetSpan(yyVAL.span)
			$$ = ts
		}
	| LATERAL_P SelectWithParens TableAsNameOpt
		{
			ts := syntaxTableSource($2.(ast.ResultSetNode), yyVAL.span)
			ts.Lateral = true
			applySourceAlias(ts, $3)
			$$ = ts
		}
	| '(' JoinTable ')' TableAsNameOpt
		{
			ts := syntaxTableSource($2.(ast.ResultSetNode), yyVAL.span)
			applySourceAlias(ts, $4)
			$$ = ts
		}
	| XMLTable TableAsNameOpt
		{
			ts := syntaxTableSource($1.(ast.ResultSetNode), yyVAL.span)
			applySourceAlias(ts, $2)
			$$ = ts
		}
	| LATERAL_P XMLTable TableAsNameOpt
		{
			ts := syntaxTableSource($2.(ast.ResultSetNode), yyVAL.span)
			ts.Lateral = true
			applySourceAlias(ts, $3)
			$$ = ts
		}
	| JSONTableExpr TableAsNameOpt
		{
			ts := syntaxTableSource($1.(ast.ResultSetNode), yyVAL.span)
			applySourceAlias(ts, $2)
			$$ = ts
		}
	| LATERAL_P JSONTableExpr TableAsNameOpt
		{
			ts := syntaxTableSource($2.(ast.ResultSetNode), yyVAL.span)
			ts.Lateral = true
			applySourceAlias(ts, $3)
			$$ = ts
		}
	;

JoinTable:
	TableRef CROSS JOIN TableRef
		{
			$$ = makeJoin(pkast.JoinTypeCross, $1, $4, nil, false)
		}
	| TableRef JoinType JOIN TableRef JoinQual
		{
			$$ = makeJoin($2, $1, $4, $5, false)
		}
	| TableRef JOIN TableRef JoinQual
		{
			$$ = makeJoin(pkast.JoinTypeInner, $1, $3, $4, false)
		}
	| TableRef NATURAL JoinType JOIN TableRef
		{
			$$ = makeJoin($3, $1, $5, nil, true)
		}
	| TableRef NATURAL JOIN TableRef
		{
			$$ = makeJoin(pkast.JoinTypeInner, $1, $4, nil, true)
		}
	;

JoinType:
	FULL OuterOpt      { $$ = pkast.JoinTypeFull }
	| LEFT OuterOpt    { $$ = pkast.JoinTypeLeft }
	| RIGHT OuterOpt   { $$ = pkast.JoinTypeRight }
	| INNER_P           { $$ = pkast.JoinTypeInner }
	;

OuterOpt:
	OUTER_P
		{ }
	| /* EMPTY */
		{ }
	;

JoinQual:
	USING '(' NameList ')' TableAsNameOpt
		{
			qual := &joinQual{using: syntaxQualifiedNames(yylex, $3)}
			if $5 != nil {
				if len($5.columns) > 0 {
					yylex.Error("USING alias cannot have column aliases")
				}
				qual.alias = $5.name
			}
			$$ = qual
		}
	| ON AExpr
		{
			$$ = &joinQual{on: $2}
		}
	;


TableName:
	QualifiedName
		{
			$$ = makeTableSource(yylex, $1, yyVAL.span)
		}
	| QualifiedName '*'
		{
			$$ = makeTableSource(yylex, $1, yyVAL.span)
		}
	| ONLY QualifiedName
		{
			ts := makeTableSource(yylex, $2, yyVAL.span)
			ts.Only = true
			$$ = ts
		}
	| ONLY '(' QualifiedName ')'
		{
			ts := makeTableSource(yylex, $3, yyVAL.span)
			ts.Only = true
			$$ = ts
		}
	;

TableAsNameOpt:
	AliasClause { $$ = $1 }
	| /* EMPTY */ { $$ = nil }
	;

FuncAliasClause:
	AliasClause { $$ = $1 }
	| AS '(' TableFuncElementList ')'
		{
			$$ = &sourceAlias{name: lex.Ident{Span: loc.InvalidSpan()}, defs: tableFuncElementDefs($3)}
		}
	| AS ColId '(' TableFuncElementList ')'
		{
			$$ = &sourceAlias{name: syntaxIdent(yylex, $2, yyS[yypt-3].span), defs: tableFuncElementDefs($4)}
		}
	| ColId '(' TableFuncElementList ')'
		{
			$$ = &sourceAlias{name: syntaxIdent(yylex, $1, yyS[yypt-3].span), defs: tableFuncElementDefs($3)}
		}
	| /* EMPTY */ { $$ = nil }
	;

AliasClause:
	AS ColId '(' NameList ')'
		{
			$$ = syntaxSourceAlias(yylex, $2, yyS[yypt-3].span, $4)
		}
	| AS ColId
		{
			$$ = syntaxSourceAlias(yylex, $2, yyS[yypt].span, nil)
		}
	| ColId '(' NameList ')'
		{
			$$ = syntaxSourceAlias(yylex, $1, yyS[yypt-3].span, $3)
		}
	| ColId
		{
			$$ = syntaxSourceAlias(yylex, $1, yyS[yypt].span, nil)
		}
	;

FuncTable:
	FuncExprWindowless OptOrdinality
		{
			$$ = syntaxTableSource(&ast.TableFunction{
				Expr:           $1.(ast.ExprNode),
				WithOrdinality: $2,
			}, yyVAL.span)
		}
	| ROWS FROM '(' RowsFromList ')' OptOrdinality
		{
			$$ = syntaxTableSource(&ast.RowsFromExpr{
				Functions:      rowsFromFunctions($4),
				WithOrdinality: $6,
			}, yyVAL.span)
		}
	;

RowsFromList:
	RowsFromItem
		{
			$$ = makeList($1)
		}
	| RowsFromList ',' RowsFromItem
		{
			$$ = appendList($1, $3)
		}
	;

RowsFromItem:
	FuncExprWindowless OptColDefList
		{
			fn := &ast.TableFunction{Expr: $1.(ast.ExprNode)}
			if $2 != nil {
				fn.ColDefs = tableFuncElementDefs($2)
			}
			$$ = fn
		}
	;

OptColDefList:
	AS '(' TableFuncElementList ')'
		{
			$$ = $3
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;




OptOrdinality:
	WITH_LA ORDINALITY		{ $$ = true }
	| /* EMPTY */		{ $$ = false }
	;

TablesampleClause:
	TABLESAMPLE FunctionName '(' ExpressionList ')' OptRepeatableClause
		{
			$$ = &ast.TableSample{
				Method:      funcNameString($2),
				MethodParts: funcNameParts($2),
				Args:        exprsFromList($4),
				Repeatable:  $6,
			}
		}
	;

OptRepeatableClause:
	REPEATABLE '(' AExpr ')'	{ $$ = $3 }
	| /* EMPTY */				{ $$ = nil }
	;

// optional WHERE clause
WhereClauseOpt:
	WHERE AExpr { $$ = $2 }
	| /* EMPTY */ { $$ = nil }
	;

WhereClause:
	WHERE AExpr { $$ = $2 }
	| WHERE CURRENT_P OF CursorName
		{
			$$ = &ast.CurrentOfExpr{
				CursorName: $4,
			}
		}
	| /* EMPTY */ { $$ = nil }
	;

// GROUP BY clause
GroupByClause:
	GROUP_P BY SetQuantifier GroupByList
		{
			$$ = &GroupClause{
				Distinct: $3 == SET_QUANTIFIER_DISTINCT,
				List:     $4,
			}
		}
	| /* EMPTY */
		{
			$$ = &GroupClause{}
		}
	;

GroupByList:
	GroupByItem
		{ $$ = makeList($1) }
	| GroupByList ',' GroupByItem
		{ $$ = appendList($1, $3) }
	;

GroupByItem:
	AExpr { $$ = &ast.GroupItem{BaseGroupItem: pkast.BaseGroupItem{Expr: $1}} }
	| EmptyGroupingSet { $$ = &ast.GroupItem{GroupingSet: $1.(*ast.GroupingSet)} }
	| CubeClause { $$ = &ast.GroupItem{GroupingSet: $1.(*ast.GroupingSet)} }
	| RollupClause { $$ = &ast.GroupItem{GroupingSet: $1.(*ast.GroupingSet)} }
	| GroupingSetsClause { $$ = &ast.GroupItem{GroupingSet: $1.(*ast.GroupingSet)} }
	;

EmptyGroupingSet:
	'(' ')'
		{
			$$ = &ast.GroupingSet{Type: ast.GroupingSetEmpty}
		}
	;

CubeClause:
	CUBE '(' ExpressionList ')'
		{
			$$ = &ast.GroupingSet{Type: ast.GroupingSetCube, Sets: [][]ast.ExprNode{$3}}
		}
	;

RollupClause:
	ROLLUP '(' ExpressionList ')'
		{
			$$ = &ast.GroupingSet{Type: ast.GroupingSetRollup, Sets: [][]ast.ExprNode{$3}}
		}
	;

GroupingSetsClause:
	GROUPING SETS '(' GroupByList ')'
		{
			$$ = &ast.GroupingSet{Type: ast.GroupingSetSets, Sets: groupingSetsFromList($4)}
		}
	;

// HAVING clause
HavingClause:
	HAVING AExpr { $$ = $2 }
	| /* EMPTY */ { $$ = nil }
	;

// Sort clause
OrderBy:
	ORDER BY ByList { $$ = orderByFromList($3) }
	;

// optional ORDER BY clause
OrderByOpt:
	OrderBy { $$ = $1 }
	| /* EMPTY */ { $$ = nil }
	;

ByList:
	ByItem
		{
			$$ = makeList($1)
		}
	| ByList ',' ByItem
		{
			$$ = appendList($1, $3)
		}
	;

ByItem:
	AExpr USING QualAllOp OptNullsOrder
		{
			item := &ast.OrderItem{Expr: $1, UseOp: operatorName($3)}
			switch $4 {
			case int64(ast.SORTBY_NULLS_FIRST):
				item.NullOrder = 1
			case int64(ast.SORTBY_NULLS_LAST):
				item.NullOrder = 2
			}
			$$ = item
		}
	| AExpr Order OptNullsOrder
		{
			item := &ast.OrderItem{Expr: $1}
			if $2 == int64(ast.SORTBY_DESC) {
				item.Desc = true
			}
			switch $3 {
			case int64(ast.SORTBY_NULLS_FIRST):
				item.NullOrder = 1
			case int64(ast.SORTBY_NULLS_LAST):
				item.NullOrder = 2
			}
			$$ = item
		}
	;

Order:
	ASC { $$ = int64(ast.SORTBY_ASC) }
	| DESC { $$ = int64(ast.SORTBY_DESC) }
	| /* EMPTY */ { $$ = int64(ast.SORTBY_DEFAULT) }
	;

SelectLimit:
	LimitClause OffsetClause
		{
			$$ = $1
			$$.LimitOffset = $2
		}
	| OffsetClause LimitClause
		{
			$$ = $2
			$$.LimitOffset = $1
		}
	| LimitClause
		{
			$$ = $1
		}
	| OffsetClause
		{
			$$ = &SelectLimit{
				LimitOffset: $1,
				LimitOption: ast.LIMIT_OPTION_COUNT,
			}
		}
	;

LimitClause:
	LIMIT SelectLimitValue
		{
			$$ = limitValueClause($2)
		}
	| LIMIT SelectLimitValue ',' SelectOffsetValue
		{
			/* PostgreSQL disallows this syntax with an error, but we parse it.
			 * The LIMIT #,# syntax is deprecated. */
			$$ = &SelectLimit{
				LimitOffset: $2,
				LimitCount:  $4,
				LimitOption: ast.LIMIT_OPTION_COUNT,
			}
		}
	| FETCH FirstOrNext SelectFetchFirstValue RowOrRows ONLY
		{
			$$ = &SelectLimit{
				LimitCount:  $3,
				LimitOption: ast.LIMIT_OPTION_COUNT,
			}
		}
	| FETCH FirstOrNext SelectFetchFirstValue RowOrRows WITH TIES
		{
			$$ = &SelectLimit{
				LimitCount:  $3,
				LimitOption: ast.LIMIT_OPTION_WITH_TIES,
			}
		}
	| FETCH FirstOrNext RowOrRows ONLY
		{
			$$ = &SelectLimit{
				LimitCount:  ast.NewValueExpr(int64(1)),
				LimitOption: ast.LIMIT_OPTION_COUNT,
			}
		}
	| FETCH FirstOrNext RowOrRows WITH TIES
		{
			$$ = &SelectLimit{
				LimitCount:  ast.NewValueExpr(int64(1)),
				LimitOption: ast.LIMIT_OPTION_WITH_TIES,
			}
		}
	;

OffsetClause:
	OFFSET SelectOffsetValue
		{ $$ = $2 }
	| OFFSET SelectFetchFirstValue RowOrRows
		{ $$ = $2 }
	;

FirstOrNext:
	FIRST_P
	| NEXT
	;

RowOrRows:
	ROW
	| ROWS
	;

SelectLimitValue:
	AExpr { $$ = $1 }
	| ALL
		{
			$$ = nil
		}
	;

SelectOffsetValue:
	AExpr { $$ = $1 }
	;

SelectFetchFirstValue:
	CExpr { $$ = $1 }
	| '+' CExpr
		{
			$$ = $2
		}
	| '-' CExpr
		{
			$$ = negateExpr($2)
		}
	;

OptSelectLimit:
	SelectLimit		{ $$ = $1 }
	| /* EMPTY */		{ $$ = nil }
	;

/*
 * Locking clause (FOR UPDATE/SHARE etc)
 */
ForLockingClause:
	ForLockingItems		{ $$ = $1 }
	| FOR READ ONLY			{ $$ = nil }
	;

OptForLockingClause:
	ForLockingClause		{ $$ = $1 }
	| /* EMPTY */			{ $$ = nil }
	;

ForLockingItems:
	ForLockingItem						{ $$ = makeList($1) }
	| ForLockingItems ForLockingItem	{ $$ = appendList($1, $2) }
	;

ForLockingItem:
	ForLockingStrength LockedRelsList OptNowaitOrSkip
		{
			$$ = &ast.LockingClause{
				LockedRels: $2,
				Strength:   int($1),
				WaitPolicy: int($3),
			}
		}
	;

ForLockingStrength:
	FOR UPDATE				{ $$ = int64(ast.LCS_FORUPDATE) }
	| FOR NO KEY UPDATE		{ $$ = int64(ast.LCS_FORNOKEYUPDATE) }
	| FOR SHARE				{ $$ = int64(ast.LCS_FORSHARE) }
	| FOR KEY SHARE			{ $$ = int64(ast.LCS_FORKEYSHARE) }
	;

LockedRelsList:
	OF QualifiedNameList	{ $$ = $2 }
	| /* EMPTY */			{ $$ = nil }
	;

OptNowaitOrSkip:
	NOWAIT					{ $$ = int64(ast.LockWaitError) }
	| SKIP LOCKED			{ $$ = int64(ast.LockWaitSkip) }
	| /* EMPTY */			{ $$ = int64(ast.LockWaitBlock) }
	;

// Expressions
AExpr:
	CExpr { $$ = $1 }
	| AExpr '+' AExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.Plus, $1, $3)
		}
	| AExpr '-' AExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.Minus, $1, $3)
		}
	| AExpr '*' AExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.Mul, $1, $3)
		}
	| AExpr '/' AExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.Div, $1, $3)
		}
	| AExpr '%' AExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.Mod, $1, $3)
		}
	| AExpr '^' AExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.Xor, $1, $3)
		}
	| AExpr '<' AExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.LT, $1, $3)
		}
	| AExpr '>' AExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.GT, $1, $3)
		}
	| AExpr '=' AExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.EQ, $1, $3)
		}
	| AExpr LESS_EQUALS AExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.LE, $1, $3)
		}
	| AExpr GREATER_EQUALS AExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.GE, $1, $3)
		}
	| AExpr NOT_EQUALS AExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.NE, $1, $3)
		}
	| AExpr QualOp AExpr				%prec Op
		{
			$$ = operatorExpr(operatorName($2), $1, $3)
		}
	| QualOp AExpr					%prec Op
		{
			$$ = ast.NewUnaryOperationExprByName(operatorName($1), $2)
		}
	| AExpr AND AExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.LogicAnd, $1, $3)
		}
	| AExpr OR AExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.LogicOr, $1, $3)
		}
	| NOT AExpr
		{
			$$ = ast.NewUnaryOperationExpr(opcode.Not, $2)
		}
	| NOT_LA AExpr						%prec NOT
		{
			$$ = ast.NewUnaryOperationExpr(opcode.Not, $2)
		}
	| AExpr IS NULL_P
		{
			$$ = &ast.IsNullExpr{Expr: $1}
		}
	| AExpr IS NOT NULL_P
		{
			$$ = &ast.IsNullExpr{Expr: $1, Not: true}
		}
	| AExpr IS TRUE_P
		{
			$$ = &ast.IsTruthExpr{Expr: $1, True: 1}
		}
	| AExpr IS FALSE_P
		{
			$$ = &ast.IsTruthExpr{Expr: $1}
		}
	| AExpr IS NOT TRUE_P              %prec IS
		{
			$$ = &ast.IsTruthExpr{Expr: $1, Not: true, True: 1}
		}
	| AExpr IS NOT FALSE_P             %prec IS
		{
			$$ = &ast.IsTruthExpr{Expr: $1, Not: true}
		}
	| AExpr IS UNKNOWN                 %prec IS
		{
			$$ = &ast.IsTruthExpr{Expr: $1, Not: true}
		}
	| AExpr IS NOT UNKNOWN             %prec IS
		{
			$$ = &ast.IsTruthExpr{Expr: $1, True: 1}
		}
	| AExpr ISNULL
		{
			$$ = &ast.IsNullExpr{Expr: $1}
		}
	| AExpr NOTNULL
		{
			$$ = &ast.IsNullExpr{Expr: $1, Not: true}
		}
	| AExpr IS DISTINCT FROM AExpr    %prec IS
		{
			$$ = ast.NewBinaryOperationExprByName("IS DISTINCT FROM", $1, $5)
		}
	| AExpr IS NOT DISTINCT FROM AExpr %prec IS
		{
			$$ = ast.NewBinaryOperationExprByName("IS NOT DISTINCT FROM", $1, $6)
		}
	| Row OVERLAPS Row
		{
			args := append(rowValues($1), rowValues($3)...)
			$$ = ast.NewFuncCallExpr("overlaps", args...)
		}
	| AExpr IS DOCUMENT_P              %prec IS
		{
			$$ = &ast.XmlExpr{
				Op:       ast.IS_DOCUMENT,
				Args:     makeList($1),
				Location: -1,
			}
		}
	| AExpr IS NOT DOCUMENT_P          %prec IS
		{
			$$ = ast.NewUnaryOperationExpr(opcode.Not, &ast.XmlExpr{
				Op:       ast.IS_DOCUMENT,
				Args:     makeList($1),
				Location: -1,
			})
		}
	| AExpr IS JSONPredicateTypeConstraint JSONKeyUniquenessConstraintOpt %prec IS
		{
			$$ = &ast.JsonIsPredicate{
				Expr:       $1,
				ItemType:   jsonValueType($3),
				UniqueKeys: $4 != 0,
				Location:   -1,
			}
		}
	| AExpr IS NOT JSONPredicateTypeConstraint JSONKeyUniquenessConstraintOpt %prec IS
		{
			$$ = ast.NewUnaryOperationExpr(opcode.Not, &ast.JsonIsPredicate{
				Expr:       $1,
				ItemType:   jsonValueType($4),
				UniqueKeys: $5 != 0,
				Location:   -1,
			})
		}
	| AExpr IS UnicodeNormalForm NORMALIZED %prec IS
		{
			$$ = ast.NewFuncCallExpr("pg_catalog.is_normalized", $1, ast.NewValueExpr($3))
		}
	| AExpr IS NOT UnicodeNormalForm NORMALIZED %prec IS
		{
			$$ = ast.NewUnaryOperationExpr(opcode.Not,
				ast.NewFuncCallExpr("pg_catalog.is_normalized", $1, ast.NewValueExpr($3)))
		}
	| AExpr IS NORMALIZED              %prec IS
		{
			$$ = ast.NewFuncCallExpr("pg_catalog.is_normalized", $1, ast.NewValueExpr("NFC"))
		}
	| AExpr IS NOT NORMALIZED          %prec IS
		{
			$$ = ast.NewUnaryOperationExpr(opcode.Not,
				ast.NewFuncCallExpr("pg_catalog.is_normalized", $1, ast.NewValueExpr("NFC")))
		}
	| AExpr LIKE AExpr                %prec LIKE
		{
			$$ = &ast.PatternLikeExpr{Expr: $1, Pattern: $3}
		}
	| AExpr LIKE AExpr ESCAPE AExpr   %prec LIKE
		{
			$$ = &ast.PatternLikeExpr{Expr: $1, Pattern: $3, Escape: $5}
		}
	| AExpr NOT_LA LIKE AExpr          %prec NOT_LA
		{
			$$ = &ast.PatternLikeExpr{Expr: $1, Pattern: $4, Not: true}
		}
	| AExpr NOT_LA LIKE AExpr ESCAPE AExpr %prec NOT_LA
		{
			$$ = &ast.PatternLikeExpr{Expr: $1, Pattern: $4, Not: true, Escape: $6}
		}
	| AExpr ILIKE AExpr               %prec ILIKE
		{
			$$ = &ast.PatternLikeExpr{Expr: $1, Pattern: $3, ILike: true}
		}
	| AExpr ILIKE AExpr ESCAPE AExpr %prec ILIKE
		{
			$$ = &ast.PatternLikeExpr{Expr: $1, Pattern: $3, ILike: true, Escape: $5}
		}
	| AExpr NOT_LA ILIKE AExpr        %prec NOT_LA
		{
			$$ = &ast.PatternLikeExpr{Expr: $1, Pattern: $4, Not: true, ILike: true}
		}
	| AExpr NOT_LA ILIKE AExpr ESCAPE AExpr %prec NOT_LA
		{
			$$ = &ast.PatternLikeExpr{Expr: $1, Pattern: $4, Not: true, ILike: true, Escape: $6}
		}
	| AExpr SIMILAR TO AExpr          %prec SIMILAR
		{
			$$ = &ast.PatternRegexpExpr{Expr: $1, Pattern: $4}
		}
	| AExpr SIMILAR TO AExpr ESCAPE AExpr %prec SIMILAR
		{
			$$ = &ast.PatternRegexpExpr{Expr: $1, Pattern: $4, Escape: $6}
		}
	| AExpr NOT_LA SIMILAR TO AExpr   %prec NOT_LA
		{
			$$ = &ast.PatternRegexpExpr{Expr: $1, Pattern: $5, Not: true}
		}
	| AExpr NOT_LA SIMILAR TO AExpr ESCAPE AExpr %prec NOT_LA
		{
			$$ = &ast.PatternRegexpExpr{Expr: $1, Pattern: $5, Not: true, Escape: $7}
		}
	| AExpr BETWEEN OptAsymmetric BExpr AND AExpr %prec BETWEEN
		{
			$$ = &ast.BetweenExpr{Expr: $1, Left: $4, Right: $6}
		}
	| AExpr NOT_LA BETWEEN OptAsymmetric BExpr AND AExpr %prec NOT_LA
		{
			$$ = &ast.BetweenExpr{Expr: $1, Left: $5, Right: $7, Not: true}
		}
	| AExpr BETWEEN SYMMETRIC BExpr AND AExpr %prec BETWEEN
		{
			$$ = &ast.BetweenExpr{Expr: $1, Left: $4, Right: $6, Symmetric: true}
		}
	| AExpr NOT_LA BETWEEN SYMMETRIC BExpr AND AExpr %prec NOT_LA
		{
			$$ = &ast.BetweenExpr{Expr: $1, Left: $5, Right: $7, Not: true, Symmetric: true}
		}
	| AExpr IN_P '(' ExpressionList ')'
		{
			$$ = &ast.PatternInExpr{Expr: $1, List: $4}
		}
	| AExpr NOT_LA IN_P '(' ExpressionList ')' %prec NOT_LA
		{
			$$ = &ast.PatternInExpr{Expr: $1, List: $5, Not: true}
		}
	| AExpr IN_P SelectWithParens
		{
			$$ = &ast.PatternInExpr{Expr: $1, Sel: &ast.SubqueryExpr{Query: $3.(ast.ResultSetNode)}}
		}
	| AExpr NOT_LA IN_P SelectWithParens %prec NOT_LA
		{
			$$ = &ast.PatternInExpr{Expr: $1, Sel: &ast.SubqueryExpr{Query: $4.(ast.ResultSetNode)}, Not: true}
		}
	| AExpr SubqueryOp SubType SelectWithParens %prec Op
		{
			name := operatorName($2)
			$$ = &ast.CompareSubqueryExpr{
				L:   $1,
				Op:  compareOpcode(name),
				OpName: name,
				R:   &ast.SubqueryExpr{Query: $4.(ast.ResultSetNode)},
				All: $3 == int64(ast.ALL_SUBLINK),
			}
		}
	| AExpr SubqueryOp SubType '(' AExpr ')' %prec Op
		{
			$$ = ast.NewBinaryOperationExprByName(operatorName($2)+" "+subTypeName($3), $1, $5)
		}
	| UNIQUE OptUniqueNullTreatment SelectWithParens
		{
			yylex.Error("UNIQUE predicate is not yet implemented")
			$$ = nil
		}
	| AExpr COLLATE AnyName
		{
			$$ = &ast.CollateExpr{Expr: $1, Collation: nameListString($3)}
		}
	| AExpr AT TIME ZONE AExpr        %prec AT
		{
			$$ = ast.NewFuncCallExpr("timezone", $5, $1)
		}
	| AExpr AT LOCAL                   %prec AT
		{
			$$ = ast.NewFuncCallExpr("timezone", $1)
		}
	| DEFAULT
		{
			$$ = &ast.DefaultExpr{}
		}
	| AExpr '[' AExpr ']'
		{
			$$ = &ast.A_Indirection{
				Arg:         $1,
				Indirection: makeList(&ast.A_Indices{Uidx: $3}),
			}
		}
	| AExpr '[' OptSliceBound ':' OptSliceBound ']'
		{
			$$ = &ast.A_Indirection{
				Arg: $1,
				Indirection: makeList(&ast.A_Indices{
					IsSlice: true,
					Lidx:    $3,
					Uidx:    $5,
				}),
			}
		}
	| AExpr TYPECAST Typename
		{
			$$ = ast.NewTypeCastExpr($1, $3)
		}
	| '+' AExpr %prec UMINUS
		{
			$$ = $2
		}
	| '-' AExpr %prec UMINUS
		{
			$$ = negateExpr($2)
		}
	;

OptAsymmetric:
	ASYMMETRIC  {}
	| /* EMPTY */ {}
	;

SubType:
	ANY  { $$ = int64(ast.ANY_SUBLINK) }
	| SOME { $$ = int64(ast.ANY_SUBLINK) }
	| ALL  { $$ = int64(ast.ALL_SUBLINK) }
	;

SubqueryOp:
	AllOp
		{
			$$ = makeList(&ast.String{Str: $1})
		}
	| OPERATOR '(' AnyOperator ')'
		{
			$$ = makeList(&ast.String{Str: qualifiedOperatorName($3)})
		}
	| LIKE
		{
			$$ = makeList(&ast.String{Str: "~~"})
		}
	| NOT_LA LIKE
		{
			$$ = makeList(&ast.String{Str: "!~~"})
		}
	| ILIKE
		{
			$$ = makeList(&ast.String{Str: "~~*"})
		}
	| NOT_LA ILIKE
		{
			$$ = makeList(&ast.String{Str: "!~~*"})
		}
	;

CaseExpr:
	CASE CaseArg WhenClauseList CaseDefault END_P
		{
			$$ = &ast.CaseExpr{
				Value:      $2,
				WhenClauses: whenClauses($3),
				ElseClause: $4,
			}
		}
	;

WhenClauseList:
	WhenClause
		{
			$$ = makeList($1)
		}
	| WhenClauseList WhenClause
		{
			$$ = appendList($1, $2)
		}
	;

WhenClause:
	WHEN AExpr THEN AExpr
		{
			$$ = &ast.WhenClause{
				Expr:   $2,
				Result: $4,
			}
		}
	;

CaseDefault:
	ELSE AExpr { $$ = $2 }
	| /* EMPTY */ { $$ = nil }
	;

CaseArg:
	AExpr { $$ = $1 }
	| /* EMPTY */ { $$ = nil }
	;

/*****************************************************************************
 *
 *      ARRAY expressions
 *
 *****************************************************************************/

ArrayExpr:
	'[' ExpressionList ']'
		{
			$$ = ast.NewArrayExpr($2)
		}
	| '[' ArrayExprList ']'
		{
			$$ = ast.NewArrayExpr($2)
		}
	| '[' ']'
		{
			$$ = ast.NewArrayExpr(nil)
		}
	;

ArrayExprList:
	ArrayExpr
		{
			$$ = []ast.ExprNode{$1}
		}
	| ArrayExprList ',' ArrayExpr
		{
			$$ = append($1, $3)
		}
	;

/*****************************************************************************
 *
 *      ROW expressions
 *
 *****************************************************************************/

Row:
	ExplicitRow
		{
			$$ = $1
		}
	| ImplicitRow
		{
			$$ = &ast.RowExpr{Items: $1}
		}
	;

ExplicitRow:
	ROW '(' ExpressionList ')'
		{
			$$ = &ast.RowExpr{Items: $3}
		}
	| ROW '(' ')'
		{
			$$ = &ast.RowExpr{}
		}
	;

ImplicitRow:
	'(' ExpressionList ',' AExpr ')'
		{
			$$ = append($2, $4)
		}
	;

BExpr:
	CExpr { $$ = $1 }
	| BExpr '+' BExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.Plus, $1, $3)
		}
	| BExpr '-' BExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.Minus, $1, $3)
		}
	| BExpr '*' BExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.Mul, $1, $3)
		}
	| BExpr '/' BExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.Div, $1, $3)
		}
	| BExpr '%' BExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.Mod, $1, $3)
		}
	| BExpr '^' BExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.Xor, $1, $3)
		}
	| BExpr '<' BExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.LT, $1, $3)
		}
	| BExpr '>' BExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.GT, $1, $3)
		}
	| BExpr '=' BExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.EQ, $1, $3)
		}
	| BExpr LESS_EQUALS BExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.LE, $1, $3)
		}
	| BExpr GREATER_EQUALS BExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.GE, $1, $3)
		}
	| BExpr NOT_EQUALS BExpr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.NE, $1, $3)
		}
	| BExpr QualOp BExpr				%prec Op
		{
			$$ = ast.NewBinaryOperationExprByName(operatorName($2), $1, $3)
		}
	| QualOp BExpr					%prec Op
		{
			$$ = ast.NewUnaryOperationExprByName(operatorName($1), $2)
		}
	| BExpr IS DISTINCT FROM BExpr		%prec IS
		{
			$$ = ast.NewBinaryOperationExprByName("IS DISTINCT FROM", $1, $5)
		}
	| BExpr IS NOT DISTINCT FROM BExpr	%prec IS
		{
			$$ = ast.NewBinaryOperationExprByName("IS NOT DISTINCT FROM", $1, $6)
		}
	| BExpr IS DOCUMENT_P				%prec IS
		{
			$$ = &ast.XmlExpr{
				Op:       ast.IS_DOCUMENT,
				Args:     makeList($1),
				Location: -1,
			}
		}
	| BExpr IS NOT DOCUMENT_P			%prec IS
		{
			$$ = ast.NewUnaryOperationExpr(opcode.Not, &ast.XmlExpr{
				Op:       ast.IS_DOCUMENT,
				Args:     makeList($1),
				Location: -1,
			})
		}
	| BExpr TYPECAST Typename
		{
			$$ = ast.NewTypeCastExpr($1, $3)
		}
	| '+' BExpr %prec UMINUS
		{
			$$ = $2
		}
	| '-' BExpr %prec UMINUS
		{
			$$ = negateExpr($2)
		}
	;

CExpr:
	ColumnRef { $$ = $1 }
	| AExprConst { $$ = $1 }
	| PARAM OptIndirection
		{
			$$ = withIndirection(ast.NewParamMarkerExpr(int($1)-1), $2)
		}
	| '(' AExpr ')' OptIndirection
		{
			$$ = withIndirection(&ast.ParenthesesExpr{Expr: $2}, $4)
		}
	| FuncExpr { $$ = $1 }
	| SelectWithParens %prec UMINUS
		{
			$$ = &ast.SubqueryExpr{Query: $1.(ast.ResultSetNode)}
		}
	| EXISTS SelectWithParens
		{
			$$ = &ast.ExistsSubqueryExpr{Sel: &ast.SubqueryExpr{Query: $2.(ast.ResultSetNode)}}
		}
	| CaseExpr { $$ = $1 }
	| ARRAY SelectWithParens
		{
			$$ = &ast.SubqueryExpr{Query: $2.(ast.ResultSetNode), Array: true}
		}
	| ARRAY ArrayExpr
		{
			$$ = $2
		}
	| ExplicitRow
		{
			$$ = $1
		}
	| ImplicitRow
		{
			$$ = &ast.RowExpr{Items: $1}
		}
	;

// Function expressions
FuncExpr:
	FuncApplication WithinGroupClause FilterClause OverClause
		{
			fn := $1.(*ast.FuncCallExpr)
			if $2 != nil {
				fn.WithinGroup = $2.(*ast.OrderByClause)
			}
			if $3 != nil {
				fn.Filter = $3.(ast.ExprNode)
			}
			if $4 != nil {
				$$ = windowFuncExpr(fn, $4.(*ast.WindowSpec))
			} else if fn.WithinGroup != nil || fn.Filter != nil || isAggregateName(nameString(fn.Name)) {
				$$ = &ast.AggregateFuncExpr{
					FnNameParts: nameStrings(fn.Name),
					F:           nameString(fn.Name),
					Args:        fn.Args,
					Filter:      fn.Filter,
					WithinGroup: fn.WithinGroup,
					OrderBy: fn.OrderBy,
					Distinct: fn.Distinct,
					Star: fn.Star,
				}
			} else {
				$$ = fn
			}
		}
	| FuncExprCommonSubexpr { $$ = $1 }
	| JSONAggregateFunc FilterClause OverClause
		{
			if c := jsonAggConstructor($1); c != nil {
				c.Agg_filter = $2
				if over, ok := $3.(*ast.WindowDef); ok {
					c.Over = over
				}
			}
			$$ = $1.(ast.ExprNode)
		}
	;

FuncExprWindowless:
	FuncApplication		{ $$ = $1 }
	| FuncExprCommonSubexpr	{ $$ = $1 }
	| JSONAggregateFunc	{ $$ = $1 }
	;

WithinGroupClause:
	WITHIN GROUP_P '(' OrderBy ')'	{ $$ = $4 }
	| /* EMPTY */						{ $$ = nil }
	;

FilterClause:
	FILTER '(' WHERE AExpr ')'		{ $$ = $4 }
	| /* EMPTY */						{ $$ = nil }
	;

OverClause:
	OVER WindowSpec	{ $$ = $2 }
	| OVER ColId
		{
			$$ = &ast.WindowSpec{Name: syntaxIdent(yylex, $2, yyS[yypt].span), OnlyAlias: true}
		}
	| /* EMPTY */				{ $$ = nil }
	;

// optional WINDOW clause
WindowClauseOpt:
	WINDOW WindowDefinitionList	{ $$ = $2 }
	| /* EMPTY */					{ $$ = nil }
	;

WindowDefinitionList:
	WindowDefinition							{ $$ = makeList($1) }
	| WindowDefinitionList ',' WindowDefinition	{ $$ = appendList($1, $3) }
	;

WindowDefinition:
	ColId AS WindowSpec
		{
			n := $3.(*ast.WindowSpec)
			n.Name = syntaxIdent(yylex, $1, yyS[yypt-2].span)
			$$ = n
		}
	;

WindowSpec:
	'(' OptExistingWindowName OptPartitionClause OrderByOpt OptWindowFrameClause ')'
		{
			spec := &ast.WindowSpec{Ref: syntaxIdent(yylex, $2, yyS[yypt-4].span)}
			if $3 != nil {
				spec.PartitionBy = windowPartitionClauseFromExprs($3)
				spec.BaseWindowSpec.PartitionBy = spec.PartitionBy
			}
			if $4 != nil {
				spec.OrderBy = $4.(*ast.OrderByClause)
				spec.BaseWindowSpec.OrderBy = spec.OrderBy
			}
			if $5 != nil {
				spec.Frame = $5.(*ast.FrameClause)
				spec.BaseWindowSpec.Frame = spec.Frame
			}
			$$ = spec
		}
	;

OptExistingWindowName:
	ColId			{ $$ = $1 }
	| /* EMPTY */	%prec Op	{ $$ = "" }
	;

OptPartitionClause:
	PARTITION BY ExpressionList	{ $$ = $3 }
	| /* EMPTY */			{ $$ = nil }
	;

OptWindowFrameClause:
	RANGE WindowFrameExtent OptWindowFrameExclusion
		{
			$$ = &ast.FrameClause{BaseWindowFrameClause: pkast.BaseWindowFrameClause{Type: ast.Ranges, Extent: $2, Exclusion: $3}}
		}
	| ROWS WindowFrameExtent OptWindowFrameExclusion
		{
			$$ = &ast.FrameClause{BaseWindowFrameClause: pkast.BaseWindowFrameClause{Type: ast.Rows, Extent: $2, Exclusion: $3}}
		}
	| GROUPS WindowFrameExtent OptWindowFrameExclusion
		{
			$$ = &ast.FrameClause{BaseWindowFrameClause: pkast.BaseWindowFrameClause{Type: ast.Groups, Extent: $2, Exclusion: $3}}
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

WindowFrameExtent:
	WindowFrameBound
		{
			$$ = ast.FrameExtent{Start: $1, End: &ast.FrameBound{Type: ast.CurrentRow}}
		}
	| BETWEEN WindowFrameBound AND WindowFrameBound
		{
			$$ = ast.FrameExtent{Start: $2, End: $4}
		}
	;

WindowFrameBound:
	UNBOUNDED PRECEDING
		{ $$ = &ast.FrameBound{Type: ast.Preceding, Unbounded: true} }
	| UNBOUNDED FOLLOWING
		{ $$ = &ast.FrameBound{Type: ast.Following, Unbounded: true} }
	| CURRENT_P ROW
		{ $$ = &ast.FrameBound{Type: ast.CurrentRow} }
	| AExpr PRECEDING
		{ $$ = &ast.FrameBound{Type: ast.Preceding, Expr: $1} }
	| AExpr FOLLOWING
		{ $$ = &ast.FrameBound{Type: ast.Following, Expr: $1} }
	;

OptWindowFrameExclusion:
	EXCLUDE CURRENT_P ROW		{ $$ = pkast.WindowFrameExcludeCurrentRow }
	| EXCLUDE GROUP_P			{ $$ = pkast.WindowFrameExcludeGroup }
	| EXCLUDE TIES				{ $$ = pkast.WindowFrameExcludeTies }
	| EXCLUDE NO OTHERS			{ $$ = pkast.WindowFrameExcludeNoOthers }
	| /* EMPTY */				{ $$ = pkast.WindowFrameExcludeNone }
	;

FuncApplication:
	FunctionName '(' ')'
		{
			$$ = ast.NewFuncCallExprParts(funcNameParts($1))
		}
	| FunctionName '(' FuncArgList OrderByOpt ')'
		{
			$$ = ast.NewFuncCallExprParts(funcNameParts($1), exprsFromList($3)...)
			if $4 != nil { $$.(*ast.FuncCallExpr).OrderBy = $4.(*ast.OrderByClause) }
		}
	| FunctionName '(' VARIADIC FuncArgExpr OrderByOpt ')'
		{
			$$ = ast.NewFuncCallExprParts(funcNameParts($1), $4)
		}
	| FunctionName '(' FuncArgList ',' VARIADIC FuncArgExpr OrderByOpt ')'
		{
			args := append(exprsFromList($3), $6)
			$$ = ast.NewFuncCallExprParts(funcNameParts($1), args...)
		}
	| FunctionName '(' '*' ')'
		{
			fn := ast.NewFuncCallExprParts(funcNameParts($1))
			fn.Star = true
			$$ = fn
		}
	| FunctionName '(' DISTINCT FuncArgList OrderByOpt ')'
		{
			$$ = ast.NewFuncCallExprParts(funcNameParts($1), exprsFromList($4)...)
			$$.(*ast.FuncCallExpr).Distinct = true
			if $5 != nil { $$.(*ast.FuncCallExpr).OrderBy = $5.(*ast.OrderByClause) }
		}
	| FunctionName '(' ALL FuncArgList OrderByOpt ')'
		{
			$$ = ast.NewFuncCallExprParts(funcNameParts($1), exprsFromList($4)...)
			if $5 != nil { $$.(*ast.FuncCallExpr).OrderBy = $5.(*ast.OrderByClause) }
		}
	;

/*
 * func_expr_common_subexpr: SQL-standard special function expressions that
 * use reserved keywords as function names. These cannot be ordinary function
 * calls because the keywords would be consumed as part of the syntax.
 */
FuncExprCommonSubexpr:
	COLLATION FOR '(' AExpr ')'
		{
			$$ = &ast.FuncCall{
				Funcname:   makeFuncName("pg_catalog", "pg_collation_for"),
				Args:       exprList1($4),
				FuncFormat: int(ast.COERCE_SQL_SYNTAX),
				Location:   -1,
			}
		}
	| CURRENT_DATE
		{
			$$ = makeSQLValueFunction(ast.SVFOP_CURRENT_DATE, -1, yylex)
		}
	| CURRENT_TIME
		{
			$$ = makeSQLValueFunction(ast.SVFOP_CURRENT_TIME, -1, yylex)
		}
	| CURRENT_TIME '(' IConst ')'
		{
			$$ = makeSQLValueFunction(ast.SVFOP_CURRENT_TIME_N, $3, yylex)
		}
	| CURRENT_TIMESTAMP
		{
			$$ = makeSQLValueFunction(ast.SVFOP_CURRENT_TIMESTAMP, -1, yylex)
		}
	| CURRENT_TIMESTAMP '(' IConst ')'
		{
			$$ = makeSQLValueFunction(ast.SVFOP_CURRENT_TIMESTAMP_N, $3, yylex)
		}
	| LOCALTIME
		{
			$$ = makeSQLValueFunction(ast.SVFOP_LOCALTIME, -1, yylex)
		}
	| LOCALTIME '(' IConst ')'
		{
			$$ = makeSQLValueFunction(ast.SVFOP_LOCALTIME_N, $3, yylex)
		}
	| LOCALTIMESTAMP
		{
			$$ = makeSQLValueFunction(ast.SVFOP_LOCALTIMESTAMP, -1, yylex)
		}
	| LOCALTIMESTAMP '(' IConst ')'
		{
			$$ = makeSQLValueFunction(ast.SVFOP_LOCALTIMESTAMP_N, $3, yylex)
		}
	| CURRENT_ROLE
		{
			$$ = makeSQLValueFunction(ast.SVFOP_CURRENT_ROLE, -1, yylex)
		}
	| CURRENT_USER
		{
			$$ = makeSQLValueFunction(ast.SVFOP_CURRENT_USER, -1, yylex)
		}
	| SESSION_USER
		{
			$$ = makeSQLValueFunction(ast.SVFOP_SESSION_USER, -1, yylex)
		}
	| SYSTEM_USER
		{
			$$ = &ast.FuncCall{
				Funcname:   makeFuncName("pg_catalog", "system_user"),
				FuncFormat: int(ast.COERCE_SQL_SYNTAX),
				Location:   -1,
			}
		}
	| USER
		{
			$$ = makeSQLValueFunction(ast.SVFOP_USER, -1, yylex)
		}
	| CURRENT_CATALOG
		{
			$$ = makeSQLValueFunction(ast.SVFOP_CURRENT_CATALOG, -1, yylex)
		}
	| CURRENT_SCHEMA
		{
			$$ = makeSQLValueFunction(ast.SVFOP_CURRENT_SCHEMA, -1, yylex)
		}
	| CAST '(' AExpr AS Typename ')'
		{
			$$ = makeTypeCast($3, $5)
		}
	| CAST '(' AExpr AS Typename OptCollate ')'
		{
			cast := makeTypeCast($3, $5)
			if c, ok := cast.(*ast.FuncCastExpr); ok {
				c.Collate = collationName($6)
			}
			$$ = cast
		}
	| NULLIF '(' AExpr ',' AExpr ')'
		{
			$$ = ast.NewFuncCallExpr("nullif", $3, $5)
		}
	| COALESCE '(' ExpressionList ')'
		{
			$$ = ast.NewFuncCallExpr("coalesce", $3...)
		}
	| GREATEST '(' ExpressionList ')'
		{
			$$ = ast.NewFuncCallExpr("greatest", $3...)
		}
	| LEAST '(' ExpressionList ')'
		{
			$$ = ast.NewFuncCallExpr("least", $3...)
		}
	| EXTRACT '(' ExtractList ')'
		{
			fn := ast.NewFuncCallExpr("extract", exprsFromList($3)...)
			fn.KeywordForm = true
			$$ = fn
		}
	| NORMALIZE '(' AExpr ')'
		{
			$$ = &ast.FuncCall{
				Funcname:   makeFuncName("pg_catalog", "normalize"),
				Args:       makeList($3),
				FuncFormat: int(ast.COERCE_SQL_SYNTAX),
				Location:   -1,
			}
		}
	| NORMALIZE '(' AExpr ',' UnicodeNormalForm ')'
		{
			$$ = &ast.FuncCall{
				Funcname:   makeFuncName("pg_catalog", "normalize"),
				Args:       makeList2($3, makeStringConst($5)),
				FuncFormat: int(ast.COERCE_SQL_SYNTAX),
				Location:   -1,
			}
		}
	| OVERLAY '(' OverlayList ')'
		{
			$$ = ast.NewFuncCallExpr("pg_catalog.overlay", exprsFromList($3)...)
		}
	| OVERLAY '(' FuncArgListOpt ')'
		{
			$$ = ast.NewFuncCallExpr("overlay", exprsFromList($3)...)
		}
	| POSITION '(' PositionList ')'
		{
			$$ = ast.NewFuncCallExpr("pg_catalog.position", exprsFromList($3)...)
		}
	| SUBSTRING '(' SubstrList ')'
		{
			$$ = ast.NewFuncCallExpr("pg_catalog.substring", exprsFromList($3)...)
		}
	| SUBSTRING '(' FuncArgListOpt ')'
		{
			$$ = ast.NewFuncCallExpr("substring", exprsFromList($3)...)
		}
	| TREAT '(' AExpr AS Typename ')'
		{
			funcName := "treat"
			$$ = &ast.FuncCall{
				Funcname:   makeFuncName("pg_catalog", funcName),
				Args:       makeList($3),
				FuncFormat: int(ast.COERCE_EXPLICIT_CALL),
				Location:   -1,
			}
		}
	| TRIM '(' BOTH TrimList ')'
		{
			$$ = ast.NewFuncCallExpr("pg_catalog.btrim", exprsFromList($4)...)
		}
	| TRIM '(' LEADING TrimList ')'
		{
			$$ = ast.NewFuncCallExpr("pg_catalog.ltrim", exprsFromList($4)...)
		}
	| TRIM '(' TRAILING TrimList ')'
		{
			$$ = ast.NewFuncCallExpr("pg_catalog.rtrim", exprsFromList($4)...)
		}
	| TRIM '(' TrimList ')'
		{
			$$ = ast.NewFuncCallExpr("pg_catalog.btrim", exprsFromList($3)...)
		}
	| GROUPING '(' ExpressionList ')'
		{
			$$ = &ast.GroupingFunc{
				Args:     nodeList($3),
				Location: -1,
			}
		}
	| XMLCONCAT '(' ExpressionList ')'
		{
			$$ = &ast.XmlExpr{
				Op:       ast.IS_XMLCONCAT,
				Args:     nodeList($3),
				Location: -1,
			}
		}
	| XMLELEMENT '(' NAME_P ColLabel ')'
		{
			$$ = &ast.XmlExpr{
				Op:       ast.IS_XMLELEMENT,
				Name:     $4,
				Location: -1,
			}
		}
	| XMLELEMENT '(' NAME_P ColLabel ',' XMLAttributes ')'
		{
			$$ = &ast.XmlExpr{
				Op:        ast.IS_XMLELEMENT,
				Name:      $4,
				NamedArgs: nodeList($6),
				Location:  -1,
			}
		}
	| XMLELEMENT '(' NAME_P ColLabel ',' ExpressionList ')'
		{
			$$ = &ast.XmlExpr{
				Op:       ast.IS_XMLELEMENT,
				Name:     $4,
				Args:     nodeList($6),
				Location: -1,
			}
		}
	| XMLELEMENT '(' NAME_P ColLabel ',' XMLAttributes ',' ExpressionList ')'
		{
			$$ = &ast.XmlExpr{
				Op:        ast.IS_XMLELEMENT,
				Name:      $4,
				NamedArgs: nodeList($6),
				Args:      nodeList($8),
				Location:  -1,
			}
		}
	| XMLEXISTS '(' CExpr XMLExistsArgument ')'
		{
			/* xmlexists(A PASSING [BY REF] B [BY REF]) is converted to xmlexists(A, B) */
			$$ = &ast.FuncCall{
				Funcname:   makeFuncName("pg_catalog", "xmlexists"),
				Args:       makeList2($3, $4),
				FuncFormat: int(ast.COERCE_SQL_SYNTAX),
				Location:   -1,
			}
		}
	| XMLFOREST '(' XMLAttributeList ')'
		{
			$$ = &ast.XmlExpr{
				Op:        ast.IS_XMLFOREST,
				NamedArgs: nodeList($3),
				Location:  -1,
			}
		}
	| XMLPARSE '(' DocumentOrContent AExpr XMLWhitespaceOption ')'
		{
			x := &ast.XmlExpr{
				Op:       ast.IS_XMLPARSE,
				Args:     makeList2($4, makeBoolAConst($5)),
				Xmloption: ast.XmlOptionType($3),
				Location: -1,
			}
			$$ = x
		}
	| XMLPI '(' NAME_P ColLabel ')'
		{
			$$ = &ast.XmlExpr{
				Op:       ast.IS_XMLPI,
				Name:     $4,
				Location: -1,
			}
		}
	| XMLPI '(' NAME_P ColLabel ',' AExpr ')'
		{
			$$ = &ast.XmlExpr{
				Op:       ast.IS_XMLPI,
				Name:     $4,
				Args:     makeList($6),
				Location: -1,
			}
		}
	| XMLROOT '(' AExpr ',' XMLRootVersion OptXMLRootStandalone ')'
		{
			$$ = &ast.XmlExpr{
				Op:       ast.IS_XMLROOT,
				Args:     &ast.List{Items: []ast.Node{$3, $5, $6}},
				Location: -1,
			}
		}
	| XMLSERIALIZE '(' DocumentOrContent AExpr AS SimpleTypename XMLIndentOption ')'
		{
			$$ = &ast.XmlSerialize{
				Xmloption: ast.XmlOptionType($3),
				Expr:      $4,
				TypeName:  $6,
				Indent:    $7 != 0,
				Location:  -1,
			}
		}
	/* SQL/JSON function expressions */
	| JSON_OBJECT '(' FuncArgList ')'
		{
			/* Legacy json_object() function call */
			$$ = ast.NewFuncCallExpr("pg_catalog.json_object", exprsFromList($3)...)
		}
	| JSON_OBJECT '(' JSONNameAndValueList JSONObjectConstructorNullClauseOpt
		JSONKeyUniquenessConstraintOpt JSONReturningClauseOpt ')'
		{
			$$ = &ast.JsonObjectConstructor{
				Exprs:        $3,
				Output:       asJsonOutput($6),
				AbsentOnNull: $4 != 0,
				UniqueKeys:   $5 != 0,
				Location:     -1,
			}
		}
	| JSON_OBJECT '(' JSONReturningClauseOpt ')'
		{
			$$ = &ast.JsonObjectConstructor{
				Output:   asJsonOutput($3),
				Location: -1,
			}
		}
	| JSON_ARRAY '(' JSONValueExprList JSONArrayConstructorNullClauseOpt
		JSONReturningClauseOpt ')'
		{
			$$ = &ast.JsonArrayConstructor{
				Exprs:        $3,
				AbsentOnNull: $4 != 0,
				Output:       asJsonOutput($5),
				Location:     -1,
			}
		}
	| JSON_ARRAY '(' SelectNoParens JSONFormatClauseOpt JSONReturningClauseOpt ')'
		{
			$$ = &ast.JsonArrayQueryConstructor{
				Query:  $3,
				Output: asJsonOutput($5),
				FormatClause: asJsonFormat($4),
				Location: -1,
			}
		}
	| JSON_ARRAY '(' JSONReturningClauseOpt ')'
		{
			$$ = &ast.JsonArrayConstructor{
				Output:   asJsonOutput($3),
				Location: -1,
			}
		}
	| JSON '(' JSONValueExpr JSONKeyUniquenessConstraintOpt ')'
		{
			$$ = &ast.JsonParseExpr{
				Expr:       $3.(*ast.JsonValueExpr),
				UniqueKeys: $4 != 0,
				Location:   -1,
			}
		}
	| JSON_SCALAR '(' AExpr ')'
		{
			$$ = &ast.JsonScalarExpr{
				Expr:     $3,
				Location: -1,
			}
		}
	| JSON_SERIALIZE '(' JSONValueExpr JSONReturningClauseOpt ')'
		{
			$$ = &ast.JsonSerializeExpr{
				Expr:   $3.(*ast.JsonValueExpr),
				Output: asJsonOutput($4),
				Location: -1,
			}
		}
	| JSON_QUERY '(' JSONValueExpr ',' AExpr JSONPassingClauseOpt
		JSONReturningClauseOpt JSONWrapperBehavior JSONQuotesClauseOpt
		JSONBehaviorClauseOpt ')'
		{
			onEmpty, onError := splitJsonBehaviorClause($10)
			$$ = &ast.JsonFuncExpr{
				Op:          ast.JSON_QUERY_OP,
				ContextItem: $3.(*ast.JsonValueExpr),
				Pathspec:    $5,
				Passing:     $6,
				Output:      asJsonOutput($7),
				Wrapper:     ast.JsonWrapper($8),
				Quotes:      ast.JsonQuotes($9),
				OnEmpty:     onEmpty,
				OnError:     onError,
				Location:    -1,
			}
		}
	| JSON_EXISTS '(' JSONValueExpr ',' AExpr JSONPassingClauseOpt
		JSONTableOnErrorOpt ')'
		{
			$$ = &ast.JsonFuncExpr{
				Op:          ast.JSON_EXISTS_OP,
				ContextItem: $3.(*ast.JsonValueExpr),
				Pathspec:    $5,
				Passing:     $6,
				OnError:     asJsonBehavior($7),
				Location:    -1,
			}
		}
	| JSON_VALUE '(' JSONValueExpr ',' AExpr JSONPassingClauseOpt
		JSONReturningClauseOpt JSONBehaviorClauseOpt ')'
		{
			onEmpty, onError := splitJsonBehaviorClause($8)
			$$ = &ast.JsonFuncExpr{
				Op:          ast.JSON_VALUE_OP,
				ContextItem: $3.(*ast.JsonValueExpr),
				Pathspec:    $5,
				Passing:     $6,
				Output:      asJsonOutput($7),
				OnEmpty:     onEmpty,
				OnError:     onError,
				Location:    -1,
			}
		}
	| MERGE_ACTION '(' ')'
		{
			$$ = &ast.FuncCall{
				Funcname:   makeFuncName("merge_action"),
				Location:   -1,
			}
		}
	;

/*
 * Helper rules for func_expr_common_subexpr
 */
ExtractList:
	ExtractArg FROM AExpr
		{
			$$ = []ast.ExprNode{ast.NewValueExpr($1), $3}
		}
	;

ExtractArg:
	IDENT       { $$ = $1 }
	| PARAM      { $$ = "?" }
	| YEAR_P    { $$ = "year" }
	| MONTH_P   { $$ = "month" }
	| DAY_P     { $$ = "day" }
	| HOUR_P    { $$ = "hour" }
	| MINUTE_P  { $$ = "minute" }
	| SECOND_P  { $$ = "second" }
	| SConst    { $$ = $1 }
	;

UnicodeNormalForm:
	NFC         { $$ = "NFC" }
	| NFD       { $$ = "NFD" }
	| NFKC      { $$ = "NFKC" }
	| NFKD      { $$ = "NFKD" }
	;

OverlayList:
	AExpr PLACING AExpr FROM AExpr FOR AExpr
		{
			$$ = []ast.ExprNode{$1, $3, $5, $7}
		}
	| AExpr PLACING AExpr FROM AExpr
		{
			$$ = []ast.ExprNode{$1, $3, $5}
		}
	;

PositionList:
	BExpr IN_P BExpr
		{
			/* note: arguments reversed per PG convention */
			$$ = []ast.ExprNode{$3, $1}
		}
	;

SubstrList:
	AExpr FROM AExpr FOR AExpr
		{
			$$ = []ast.ExprNode{$1, $3, $5}
		}
	| AExpr FOR AExpr FROM AExpr
		{
			$$ = []ast.ExprNode{$1, $5, $3}
		}
	| AExpr FROM AExpr
		{
			$$ = []ast.ExprNode{$1, $3}
		}
	| AExpr FOR AExpr
		{
			$$ = []ast.ExprNode{
				$1,
				ast.NewValueExpr(int64(1)),
				$3,
			}
		}
	| AExpr SIMILAR AExpr ESCAPE AExpr
		{
			$$ = []ast.ExprNode{$1, $3, $5}
		}
	| ExpressionList
		{
			/* comma-separated form: substring(x, 1, 3) */
			$$ = $1
		}
	;

TrimList:
	AExpr FROM ExpressionList
		{
			$$ = append($3, $1)
		}
	| FROM ExpressionList
		{
			$$ = $2
		}
	| ExpressionList
		{
			$$ = $1
		}
	;

/*****************************************************************************
 *
 * XML helper rules
 *
 *****************************************************************************/

XMLRootVersion:
	VERSION_P AExpr
		{ $$ = $2 }
	| VERSION_P NO VALUE_P
		{ $$ = makeNullAConst() }
	;

OptXMLRootStandalone:
	',' STANDALONE_P YES_P
		{ $$ = makeIntConst(int64(ast.XML_STANDALONE_YES)) }
	| ',' STANDALONE_P NO
		{ $$ = makeIntConst(int64(ast.XML_STANDALONE_NO)) }
	| ',' STANDALONE_P NO VALUE_P
		{ $$ = makeIntConst(int64(ast.XML_STANDALONE_NO_VALUE)) }
	| /* EMPTY */
		{ $$ = makeIntConst(int64(ast.XML_STANDALONE_OMITTED)) }
	;

XMLAttributes:
	XMLATTRIBUTES '(' XMLAttributeList ')'	{ $$ = $3 }
	;

XMLAttributeList:
	XMLAttributeElem
		{ $$ = makeList($1) }
	| XMLAttributeList ',' XMLAttributeElem
		{ $$ = appendList($1, $3) }
	;

XMLAttributeElem:
	AExpr AS ColLabel
		{
			$$ = &ast.ResTarget{
				Name:     $3,
				Val:      $1,
				Location: -1,
			}
		}
	| AExpr
		{
			$$ = &ast.ResTarget{
				Val:      $1,
				Location: -1,
			}
		}
	;

DocumentOrContent:
	DOCUMENT_P		{ $$ = int64(ast.XMLOPTION_DOCUMENT) }
	| CONTENT_P	{ $$ = int64(ast.XMLOPTION_CONTENT) }
	;

XMLIndentOption:
	INDENT			{ $$ = 1 }
	| NO INDENT		{ $$ = 0 }
	| /* EMPTY */	{ $$ = 0 }
	;

XMLWhitespaceOption:
	PRESERVE WHITESPACE_P		{ $$ = 1 }
	| STRIP_P WHITESPACE_P		{ $$ = 0 }
	| /* EMPTY */				{ $$ = 0 }
	;

XMLExistsArgument:
	PASSING CExpr
		{ $$ = $2 }
	| PASSING CExpr XMLPassingMech
		{ $$ = $2 }
	| PASSING XMLPassingMech CExpr
		{ $$ = $3 }
	| PASSING XMLPassingMech CExpr XMLPassingMech
		{ $$ = $3 }
	;

XMLPassingMech:
	BY REF_P
	| BY VALUE_P
	;

/*****************************************************************************
 *
 * XMLTABLE
 *
 *****************************************************************************/








/*****************************************************************************
 *
 * SQL/JSON helper rules
 *
 *****************************************************************************/

JSONValueExpr:
	AExpr JSONFormatClauseOpt
		{
			$$ = &ast.JsonValueExpr{
				RawExpr: $1,
				FormatClause: asJsonFormat($2),
			}
		}
	;

JSONFormatClause:
	FORMAT_LA JSON
		{
			$$ = &ast.JsonFormat{
				FormatType: ast.JS_FORMAT_JSON,
				Location:   -1,
			}
		}
	| FORMAT_LA JSON ENCODING Name
		{
			f := &ast.JsonFormat{
				FormatType: ast.JS_FORMAT_JSON,
				Location:   -1,
			}
			switch strings.ToUpper($4) {
			case "UTF8":
				f.Encoding = ast.JS_ENC_UTF8
			case "UTF16":
				f.Encoding = ast.JS_ENC_UTF16
			case "UTF32":
				f.Encoding = ast.JS_ENC_UTF32
			default:
				yylex.Error("unsupported JSON encoding")
			}
			$$ = f
		}
	;

JSONFormatClauseOpt:
	JSONFormatClause      { $$ = $1 }
	| /* EMPTY */           { $$ = nil }
	;

JSONReturningClauseOpt:
	RETURNING Typename JSONFormatClauseOpt
		{
			output := &ast.JsonOutput{
				TypeName: $2,
			}
			if $3 != nil {
				output.Returning = &ast.JsonReturning{FormatClause: asJsonFormat($3)}
			}
			$$ = output
		}
	| /* EMPTY */
		{ $$ = nil }
	;


JSONBehavior:
	DEFAULT AExpr
		{
			$$ = &ast.JsonBehavior{
				Btype: ast.JSON_BEHAVIOR_DEFAULT,
				Expr:  $2,
				Location: -1,
			}
		}
	| JSONBehaviorType
		{
			$$ = &ast.JsonBehavior{
				Btype: ast.JsonBehaviorType($1),
				Location: -1,
			}
		}
	;

JSONBehaviorType:
	ERROR_P         { $$ = int64(ast.JSON_BEHAVIOR_ERROR) }
	| NULL_P        { $$ = int64(ast.JSON_BEHAVIOR_NULL) }
	| TRUE_P        { $$ = int64(ast.JSON_BEHAVIOR_TRUE) }
	| FALSE_P       { $$ = int64(ast.JSON_BEHAVIOR_FALSE) }
	| UNKNOWN       { $$ = int64(ast.JSON_BEHAVIOR_UNKNOWN) }
	| EMPTY_P ARRAY { $$ = int64(ast.JSON_BEHAVIOR_EMPTY_ARRAY) }
	| EMPTY_P OBJECT_P { $$ = int64(ast.JSON_BEHAVIOR_EMPTY_OBJECT) }
	/* non-standard, for Oracle compatibility */
	| EMPTY_P          { $$ = int64(ast.JSON_BEHAVIOR_EMPTY_ARRAY) }
	;

JSONBehaviorClauseOpt:
	JSONBehavior ON EMPTY_P JSONBehavior ON ERROR_P
		{
			/* Returns a list of 2: [on_empty, on_error] */
			$$ = &ast.List{Items: []ast.Node{$1, $4}}
		}
	| JSONBehavior ON EMPTY_P
		{
			$$ = &ast.List{Items: []ast.Node{$1, nil}}
		}
	| JSONBehavior ON ERROR_P
		{
			$$ = &ast.List{Items: []ast.Node{nil, $1}}
		}
	| /* EMPTY */
		{ $$ = nil }
	;

JSONTableOnErrorOpt:
	JSONBehavior ON ERROR_P
		{ $$ = $1 }
	| /* EMPTY */
		{ $$ = nil }
	;

JSONWrapperBehavior:
	WITHOUT WRAPPER                 { $$ = int64(ast.JSW_NONE) }
	| WITHOUT ARRAY WRAPPER         { $$ = int64(ast.JSW_NONE) }
	| WITH WRAPPER                  { $$ = int64(ast.JSW_UNCONDITIONAL) }
	| WITH ARRAY WRAPPER            { $$ = int64(ast.JSW_UNCONDITIONAL) }
	| WITH CONDITIONAL ARRAY WRAPPER { $$ = int64(ast.JSW_CONDITIONAL) }
	| WITH CONDITIONAL WRAPPER      { $$ = int64(ast.JSW_CONDITIONAL) }
	| WITH UNCONDITIONAL ARRAY WRAPPER { $$ = int64(ast.JSW_UNCONDITIONAL) }
	| WITH UNCONDITIONAL WRAPPER    { $$ = int64(ast.JSW_UNCONDITIONAL) }
	| /* EMPTY */                   { $$ = int64(ast.JSW_UNSPEC) }
	;

JSONQuotesClauseOpt:
	KEEP QUOTES ON SCALAR STRING_P  { $$ = int64(ast.JS_QUOTES_KEEP) }
	| KEEP QUOTES                   { $$ = int64(ast.JS_QUOTES_KEEP) }
	| OMIT QUOTES ON SCALAR STRING_P { $$ = int64(ast.JS_QUOTES_OMIT) }
	| OMIT QUOTES                   { $$ = int64(ast.JS_QUOTES_OMIT) }
	| /* EMPTY */                   { $$ = int64(ast.JS_QUOTES_UNSPEC) }
	;

JSONPredicateTypeConstraint:
	JSON       %prec UNBOUNDED      { $$ = int64(ast.JS_TYPE_ANY) }
	| JSON VALUE_P                  { $$ = int64(ast.JS_TYPE_ANY) }
	| JSON ARRAY                    { $$ = int64(ast.JS_TYPE_ARRAY) }
	| JSON OBJECT_P                 { $$ = int64(ast.JS_TYPE_OBJECT) }
	| JSON SCALAR                   { $$ = int64(ast.JS_TYPE_SCALAR) }
	;

JSONKeyUniquenessConstraintOpt:
	WITH UNIQUE KEYS                    { $$ = 1 }
	| WITH UNIQUE      %prec UNBOUNDED  { $$ = 1 }
	| WITHOUT UNIQUE KEYS               { $$ = 0 }
	| WITHOUT UNIQUE   %prec UNBOUNDED  { $$ = 0 }
	| /* EMPTY */      %prec UNBOUNDED  { $$ = 0 }
	;

JSONAggregateFunc:
	JSON_OBJECTAGG '(' JSONNameAndValue JSONObjectConstructorNullClauseOpt
		JSONKeyUniquenessConstraintOpt JSONReturningClauseOpt ')'
		{
			$$ = &ast.JsonObjectAgg{
				Constructor: &ast.JsonAggConstructor{
					Output:   asJsonOutput($6),
					Location: -1,
				},
				Arg:          $3.(*ast.JsonKeyValue),
				AbsentOnNull: $4 != 0,
				UniqueKeys:   $5 != 0,
			}
		}
	| JSON_ARRAYAGG '(' JSONValueExpr JSONArrayAggregateOrderByClauseOpt
		JSONArrayConstructorNullClauseOpt JSONReturningClauseOpt ')'
		{
			$$ = &ast.JsonArrayAgg{
				Constructor: &ast.JsonAggConstructor{
					Output:    asJsonOutput($6),
					Agg_order: $4,
					Location:  -1,
				},
				Arg:          $3.(*ast.JsonValueExpr),
				AbsentOnNull: $5 != 0,
			}
		}
	;

JSONArrayAggregateOrderByClauseOpt:
	ORDER BY ByList
		{
			$$ = $3
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

JSONNameAndValueList:
	JSONNameAndValue
		{ $$ = makeList($1) }
	| JSONNameAndValueList ',' JSONNameAndValue
		{ $$ = appendList($1, $3) }
	;

JSONNameAndValue:
	CExpr VALUE_P JSONValueExpr
		{
			$$ = &ast.JsonKeyValue{
				Key:   $1,
				Value: $3.(*ast.JsonValueExpr),
			}
		}
	| AExpr ':' JSONValueExpr
		{
			$$ = &ast.JsonKeyValue{
				Key:   $1,
				Value: $3.(*ast.JsonValueExpr),
			}
		}
	;

JSONObjectConstructorNullClauseOpt:
	NULL_P ON NULL_P                { $$ = 0 }
	| ABSENT ON NULL_P              { $$ = 1 }
	| /* EMPTY */                   { $$ = 0 }
	;

JSONArrayConstructorNullClauseOpt:
	NULL_P ON NULL_P                { $$ = 0 }
	| ABSENT ON NULL_P              { $$ = 1 }
	| /* EMPTY */                   { $$ = 1 }
	;

JSONValueExprList:
	JSONValueExpr
		{ $$ = makeList($1) }
	| JSONValueExprList ',' JSONValueExpr
		{ $$ = appendList($1, $3) }
	;

JSONPassingClauseOpt:
	PASSING JSONArguments  { $$ = $2 }
	| /* EMPTY */           { $$ = nil }
	;

JSONArguments:
	JSONArgument
		{ $$ = makeList($1) }
	| JSONArguments ',' JSONArgument
		{ $$ = appendList($1, $3) }
	;

JSONArgument:
	JSONValueExpr AS ColLabel
		{
			$$ = &ast.JsonArgument{
				Val:  $1.(*ast.JsonValueExpr),
				Name: $3,
			}
		}
	;



/*****************************************************************************
 *
 * JSON_TABLE
 *
 *****************************************************************************/









FuncArgList:
	FuncArgExpr
		{
			$$ = makeList($1)
		}
	| FuncArgList ',' FuncArgExpr
		{
			$$ = appendList($1, $3)
		}
	;

FuncArgListOpt:
	FuncArgList
		{
			$$ = $1
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

FuncArgExpr:
	AExpr { $$ = $1 }
	| ParamName COLON_EQUALS AExpr
		{
			$$ = &ast.NamedArgExpr{
				Name:      $1,
				Arg:       $3,
				Argnumber: -1,
				Location:  -1,
			}
		}
	| ParamName EQUALS_GREATER AExpr
		{
			$$ = &ast.NamedArgExpr{
				Name:      $1,
				Arg:       $3,
				Argnumber: -1,
				Location:  -1,
			}
		}
	;

FunctionName:
	TypeFunctionName
		{
			$$ = makeList(&ast.String{Str: $1})
		}
	| ColId Indirection
		{
			/* schema.funcname or catalog.schema.funcname */
			l := &ast.List{Items: []ast.Node{&ast.String{Str: $1}}}
			if $2 != nil {
				for _, item := range $2.Items {
					l.Items = append(l.Items, item)
				}
			}
			$$ = l
		}
	;

ColumnRef:
	ColId
		{
			$$ = ast.NewColumnNameExpr("", "", $1)
		}
	| ColId Indirection
		{
			$$ = columnRefWithIndirection(yylex, syntaxIdent(yylex, $1, yyS[yypt-1].span), $2, yyVAL.span)
		}
	;

Indirection:
	IndirectionElem
		{
			$$ = makeList($1)
		}
	| Indirection IndirectionElem
		{
			$$ = appendList($1, $2)
		}
	;

IndirectionElem:
	'.' AttrName
		{
			$$ = &ast.String{Str: $2}
			$$.SetSpan(yyS[yypt].span)
		}
	| '.' '*'
		{
			$$ = &ast.A_Star{}
			$$.SetSpan(yyVAL.span)
		}
	| '[' AExpr ']'
		{
			$$ = &ast.A_Indices{Uidx: $2}
		}
	| '[' OptSliceBound ':' OptSliceBound ']'
		{
			$$ = &ast.A_Indices{IsSlice: true, Lidx: $2, Uidx: $4}
		}
	;

OptSliceBound:
	AExpr      { $$ = $1 }
	| /* EMPTY */ { $$ = nil }
	;

OptIndirection:
	Indirection { $$ = $1 }
	| /* EMPTY */ { $$ = nil }
	;

Attrs:
	'.' AttrName
		{
			$$ = makeList(&ast.String{Str: $2})
		}
	| Attrs '.' AttrName
		{
			$$ = appendList($1, &ast.String{Str: $3})
		}
	;

AExprConst:
	IConst
		{
			$$ = ast.NewValueExpr($1)
		}
	| FCONST
		{
			$$ = floatValueExpr($1)
		}
	| SConst
		{
			$$ = ast.NewValueExpr($1)
		}
	| BCONST
		{
			$$ = bitStringLiteral($1)
		}
	| XCONST
		{
			$$ = hexStringLiteral($1)
		}
	| TRUE_P
		{
			$$ = ast.NewValueExpr(true)
		}
	| FALSE_P
		{
			$$ = ast.NewValueExpr(false)
		}
	| NULL_P
		{
			$$ = ast.NewValueExpr(nil)
		}
	| FunctionName SConst
		{
			$$ = ast.NewTypeCastExpr(ast.NewValueExpr($2), typeNameFromList($1))
		}
	| FunctionName '(' FuncArgList OrderByOpt ')' SConst
		{
			tp := typeNameFromList($1)
			if $4 != nil { yylex.Error("type modifier cannot have ORDER BY") }
			if err := setTypeModifiers(tp, exprsFromList($3)); err != nil { yylex.Error(err.Error()) }
			$$ = ast.NewTypeCastExpr(ast.NewValueExpr($6), tp)
		}
	| ConstTypename SConst
		{
			$$ = ast.NewTypeCastExpr(ast.NewValueExpr($2), $1)
		}
	| ConstInterval SConst OptInterval
		{
			setIntervalFields($1, $3)
			$$ = ast.NewTypeCastExpr(ast.NewValueExpr($2), $1)
		}
	| ConstInterval '(' IConst ')' SConst
		{
			$1.PGModifiers = []string{fmt.Sprint($3)}
			$$ = ast.NewTypeCastExpr(ast.NewValueExpr($5), $1)
		}
	;

IConst:
	ICONST { $$ = $1 }
	;

SConst:
	SCONST { $$ = $1 }
	;

// Names and identifiers
ColId:
	IDENT { $$ = $1 }
	| UnReservedKeyword { $$ = $1 }
	| ColNameKeyword { $$ = $1 }
	;

ColLabel:
	IDENT { $$ = $1 }
	| UnReservedKeyword { $$ = $1 }
	| ColNameKeyword { $$ = $1 }
	| TypeFuncNameKeyword { $$ = $1 }
	| ReservedKeyword { $$ = $1 }
	;

AttrName:
	ColLabel { $$ = $1 }
	;

Name:
	ColId { $$ = $1 }
	;

QualifiedName:
	ColId
		{
			name := &ast.String{Str: $1}
			name.SetSpan(yyS[yypt].span)
			$$ = makeList(name)
		}
	| ColId Indirection
		{
			name := &ast.String{Str: $1}
			name.SetSpan(yyS[yypt-1].span)
			$$ = prependList(name, $2)
			if len($$.Items) > 3 { yylex.Error("too many dotted names in relation") }
			for _, item := range $2.Items {
				if _, ok := item.(*ast.String); !ok { yylex.Error("invalid indirection in relation name") }
			}
		}
	;

AnyName:
	ColId
		{
			part := &ast.String{Str: $1}
			part.SetSpan(yyS[yypt].span)
			$$ = makeList(part)
		}
	| ColId '.' AnyName
		{
			part := &ast.String{Str: $1}
			part.SetSpan(yyS[yypt-2].span)
			$$ = prependList(part, $3)
		}
	;



OptNameList:
	NameList { $$ = $1 }
	| /* EMPTY */ { $$ = nil }
	;

NameList:
	Name
		{
			name := &ast.String{Str: $1}
			name.SetSpan(yyS[yypt].span)
			$$ = makeList(name)
		}
	| NameList ',' Name
		{
			name := &ast.String{Str: $3}
			name.SetSpan(yyS[yypt].span)
			$$ = appendList($1, name)
		}
	;

ExpressionList:
	AExpr
		{
			$$ = []ast.ExprNode{$1}
		}
	| ExpressionList ',' AExpr
		{
			$$ = append($1, $3)
		}
	;

/*****************************************************************************
 *
 *      Type Name (Typename) rules
 *
 *****************************************************************************/

Typename:
	SimpleTypename OptArrayBounds
		{
			$$ = $1
			$$.PGArrayBounds = arrayBounds($2)
		}
	| SETOF SimpleTypename OptArrayBounds
		{
			$$ = $2
			$$.PGArrayBounds = arrayBounds($3)
			$$.PGSetOf = true
		}
	| SimpleTypename ARRAY '[' IConst ']'
		{
			$$ = $1
			$$.PGArrayBounds = []int64{$4}
		}
	| SETOF SimpleTypename ARRAY '[' IConst ']'
		{
			$$ = $2
			$$.PGArrayBounds = []int64{$5}
			$$.PGSetOf = true
		}
	| SimpleTypename ARRAY
		{
			$$ = $1
			$$.PGArrayBounds = []int64{-1}
		}
	| SETOF SimpleTypename ARRAY
		{
			$$ = $2
			$$.PGArrayBounds = []int64{-1}
			$$.PGSetOf = true
		}
	;

OptArrayBounds:
	OptArrayBounds '[' ']'
		{
			$$ = appendList($1, &ast.Integer{Ival: -1})
		}
	| OptArrayBounds '[' IConst ']'
		{
			$$ = appendList($1, &ast.Integer{Ival: $3})
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

SimpleTypename:
	GenericType       { $$ = $1 }
	| Numeric         { $$ = $1 }
	| Bit             { $$ = $1 }
	| Character       { $$ = $1 }
	| ConstDatetime   { $$ = $1 }
	| ConstInterval OptInterval
		{
			$$ = $1
			setIntervalFields($$, $2)
		}
	| ConstInterval '(' IConst ')'
		{
			$$ = $1
			$$.PGModifiers = []string{fmt.Sprint($3)}
		}
	| BOOLEAN_P       { $$ = newDataType("bool", -1, -1) }
	| JSON            { $$ = newDataType("json", -1, -1) }
	;

GenericType:
	TypeFunctionName OptTypeModifiers
		{
			flen, dec := typeModifiers($2)
			$$ = newDataType($1, flen, dec)
			if err := setTypeModifiers($$, $2); err != nil { yylex.Error(err.Error()) }
		}
	| TypeFunctionName '.' AttrName OptTypeModifiers
		{
			flen, dec := typeModifiers($4)
			$$ = newDataType($1+"."+$3, flen, dec)
			$$.PGName = []string{$1, $3}
			if err := setTypeModifiers($$, $4); err != nil { yylex.Error(err.Error()) }
		}
	;

OptTypeModifiers:
	'(' ExpressionList ')'  { $$ = $2 }
	| /* EMPTY */       { $$ = nil }
	;

Numeric:
	INT_P        { $$ = newDataType("int4", -1, -1) }
	| INTEGER    { $$ = newDataType("int4", -1, -1) }
	| SMALLINT   { $$ = newDataType("int2", -1, -1) }
	| BIGINT     { $$ = newDataType("int8", -1, -1) }
	| REAL       { $$ = newDataType("float4", -1, -1) }
	| FLOAT_P OptFloat
		{
			$$ = $2
		}
	| DOUBLE_P PRECISION  { $$ = newDataType("float8", -1, -1) }
	| DECIMAL_P OptTypeModifiers
		{
			flen, dec := typeModifiers($2)
			$$ = newDataType("numeric", flen, dec)
			if err := setTypeModifiers($$, $2); err != nil { yylex.Error(err.Error()) }
		}
	| DEC OptTypeModifiers
		{
			flen, dec := typeModifiers($2)
			$$ = newDataType("numeric", flen, dec)
			if err := setTypeModifiers($$, $2); err != nil { yylex.Error(err.Error()) }
		}
	| NUMERIC OptTypeModifiers
		{
			flen, dec := typeModifiers($2)
			$$ = newDataType("numeric", flen, dec)
			if err := setTypeModifiers($$, $2); err != nil { yylex.Error(err.Error()) }
		}
	;

OptFloat:
	'(' IConst ')'
		{
			if $2 <= 24 {
				$$ = newDataType("float4", -1, -1)
			} else {
				$$ = newDataType("float8", -1, -1)
			}
		}
	| /* EMPTY */
		{
			$$ = newDataType("float8", -1, -1)
		}
	;

Character:
CHARACTER OptVarying '(' IConst ')'
		{
			if $2 {
				$$ = newDataType("varchar", int($4), -1)
			} else {
				$$ = newDataType("bpchar", int($4), -1)
			}
		}
	| CHARACTER OptVarying
		{
			if $2 {
				$$ = newDataType("varchar", -1, -1)
			} else {
				$$ = newDataType("bpchar", -1, -1)
			}
		}
	| CHAR_P OptVarying '(' IConst ')'
		{
			if $2 {
				$$ = newDataType("varchar", int($4), -1)
			} else {
				$$ = newDataType("bpchar", int($4), -1)
			}
		}
	| CHAR_P OptVarying
		{
			if $2 {
				$$ = newDataType("varchar", -1, -1)
			} else {
				$$ = newDataType("bpchar", -1, -1)
			}
		}
	| NATIONAL CHARACTER OptVarying '(' IConst ')'
		{
			if $3 {
				$$ = newDataType("varchar", int($5), -1)
			} else {
				$$ = newDataType("bpchar", int($5), -1)
			}
		}
	| NATIONAL CHARACTER OptVarying
		{
			if $3 {
				$$ = newDataType("varchar", -1, -1)
			} else {
				$$ = newDataType("bpchar", -1, -1)
			}
		}
	| NATIONAL CHAR_P OptVarying '(' IConst ')'
		{
			if $3 {
				$$ = newDataType("varchar", int($5), -1)
			} else {
				$$ = newDataType("bpchar", int($5), -1)
			}
		}
	| NATIONAL CHAR_P OptVarying
		{
			if $3 {
				$$ = newDataType("varchar", -1, -1)
			} else {
				$$ = newDataType("bpchar", -1, -1)
			}
		}
	| NCHAR OptVarying '(' IConst ')'
		{
			if $2 {
				$$ = newDataType("varchar", int($4), -1)
			} else {
				$$ = newDataType("bpchar", int($4), -1)
			}
		}
	| NCHAR OptVarying
		{
			if $2 {
				$$ = newDataType("varchar", -1, -1)
			} else {
				$$ = newDataType("bpchar", -1, -1)
			}
		}
	| VARCHAR '(' IConst ')'
		{
			$$ = newDataType("varchar", int($3), -1)
		}
	| VARCHAR
		{
			$$ = newDataType("varchar", -1, -1)
		}
	;

Bit:
	BitWithLength		{ $$ = $1 }
	| BitWithoutLength	{ $$ = $1 }
	;

OptVarying:
	VARYING { $$ = true }
	| /* EMPTY */ { $$ = false }
	;

BitWithLength:
	BIT OptVarying '(' ExpressionList ')'
		{
			flen, _ := typeModifiers($4)
			if $2 {
				$$ = newDataType("varbit", flen, -1)
			} else {
				$$ = newDataType("bit", flen, -1)
			}
			if err := setTypeModifiers($$, $4); err != nil { yylex.Error(err.Error()) }
		}
	;

BitWithoutLength:
	BIT OptVarying
		{
			if $2 {
				$$ = newDataType("varbit", -1, -1)
			} else {
				$$ = newDataType("bit", 1, -1)
			}
		}
	;

ConstTypename:
	Numeric				{ $$ = $1 }
	| ConstBit			{ $$ = $1 }
	| ConstCharacter	{ $$ = $1 }
	| ConstDatetime		{ $$ = $1 }
	| JSON              { $$ = newDataType("json", -1, -1) }
	;

ConstBit:
	BitWithLength		{ $$ = $1 }
	| BitWithoutLength
		{
			$$ = $1
			/* ConstBit defaults to unspecified length for BIT */
		}
	;

ConstCharacter:
CHARACTER OptVarying '(' IConst ')'
		{
			if $2 {
				$$ = newDataType("varchar", int($4), -1)
			} else {
				$$ = newDataType("bpchar", int($4), -1)
			}
		}
	| CHARACTER OptVarying
		{
			if $2 {
				$$ = newDataType("varchar", -1, -1)
			} else {
				$$ = newDataType("bpchar", -1, -1)
			}
		}
	| CHAR_P OptVarying '(' IConst ')'
		{
			if $2 {
				$$ = newDataType("varchar", int($4), -1)
			} else {
				$$ = newDataType("bpchar", int($4), -1)
			}
		}
	| CHAR_P OptVarying
		{
			if $2 {
				$$ = newDataType("varchar", -1, -1)
			} else {
				$$ = newDataType("bpchar", -1, -1)
			}
		}
	| NATIONAL CHARACTER OptVarying '(' IConst ')'
		{
			if $3 {
				$$ = newDataType("varchar", int($5), -1)
			} else {
				$$ = newDataType("bpchar", int($5), -1)
			}
		}
	| NATIONAL CHARACTER OptVarying
		{
			if $3 {
				$$ = newDataType("varchar", -1, -1)
			} else {
				$$ = newDataType("bpchar", -1, -1)
			}
		}
	| NATIONAL CHAR_P OptVarying '(' IConst ')'
		{
			if $3 {
				$$ = newDataType("varchar", int($5), -1)
			} else {
				$$ = newDataType("bpchar", int($5), -1)
			}
		}
	| NATIONAL CHAR_P OptVarying
		{
			if $3 {
				$$ = newDataType("varchar", -1, -1)
			} else {
				$$ = newDataType("bpchar", -1, -1)
			}
		}
	| NCHAR OptVarying '(' IConst ')'
		{
			if $2 {
				$$ = newDataType("varchar", int($4), -1)
			} else {
				$$ = newDataType("bpchar", int($4), -1)
			}
		}
	| NCHAR OptVarying
		{
			if $2 {
				$$ = newDataType("varchar", -1, -1)
			} else {
				$$ = newDataType("bpchar", -1, -1)
			}
		}
	;
	| VARCHAR '(' IConst ')'
		{
			$$ = newDataType("varchar", int($3), -1)
		}
	| VARCHAR
		{
			$$ = newDataType("varchar", -1, -1)
		}
	;

ConstDatetime:
	TIMESTAMP '(' IConst ')' OptTimezone
		{
			if $5 {
				$$ = newDataType("timestamptz", int($3), -1)
			} else {
				$$ = newDataType("timestamp", int($3), -1)
			}
		}
	| TIMESTAMP OptTimezone
		{
			if $2 {
				$$ = newDataType("timestamptz", -1, -1)
			} else {
				$$ = newDataType("timestamp", -1, -1)
			}
		}
	| TIME '(' IConst ')' OptTimezone
		{
			if $5 {
				$$ = newDataType("timetz", int($3), -1)
			} else {
				$$ = newDataType("time", int($3), -1)
			}
		}
	| TIME OptTimezone
		{
			if $2 {
				$$ = newDataType("timetz", -1, -1)
			} else {
				$$ = newDataType("time", -1, -1)
			}
		}
	;

ConstInterval:
	INTERVAL
		{
			$$ = newDataType("interval", -1, -1)
		}
	;

OptTimezone:
	WITH_LA TIME ZONE		{ $$ = true }
	| WITHOUT_LA TIME ZONE		{ $$ = false }
	| /* EMPTY */			{ $$ = false }
	;

OptInterval:
	YEAR_P
		{ $$ = makeList(makeIntConst(int64(ast.INTERVAL_MASK_YEAR))) }
	| MONTH_P
		{ $$ = makeList(makeIntConst(int64(ast.INTERVAL_MASK_MONTH))) }
	| DAY_P
		{ $$ = makeList(makeIntConst(int64(ast.INTERVAL_MASK_DAY))) }
	| HOUR_P
		{ $$ = makeList(makeIntConst(int64(ast.INTERVAL_MASK_HOUR))) }
	| MINUTE_P
		{ $$ = makeList(makeIntConst(int64(ast.INTERVAL_MASK_MINUTE))) }
	| IntervalSecond
		{ $$ = $1 }
	| YEAR_P TO MONTH_P
		{
			$$ = makeList(makeIntConst(int64(ast.INTERVAL_MASK_YEAR | ast.INTERVAL_MASK_MONTH)))
		}
	| DAY_P TO HOUR_P
		{
			$$ = makeList(makeIntConst(int64(ast.INTERVAL_MASK_DAY | ast.INTERVAL_MASK_HOUR)))
		}
	| DAY_P TO MINUTE_P
		{
			$$ = makeList(makeIntConst(int64(ast.INTERVAL_MASK_DAY | ast.INTERVAL_MASK_HOUR | ast.INTERVAL_MASK_MINUTE)))
		}
	| DAY_P TO IntervalSecond
		{
			$$ = $3
			$$.Items[0] = makeIntConst(int64(ast.INTERVAL_MASK_DAY | ast.INTERVAL_MASK_HOUR | ast.INTERVAL_MASK_MINUTE | ast.INTERVAL_MASK_SECOND))
		}
	| HOUR_P TO MINUTE_P
		{
			$$ = makeList(makeIntConst(int64(ast.INTERVAL_MASK_HOUR | ast.INTERVAL_MASK_MINUTE)))
		}
	| HOUR_P TO IntervalSecond
		{
			$$ = $3
			$$.Items[0] = makeIntConst(int64(ast.INTERVAL_MASK_HOUR | ast.INTERVAL_MASK_MINUTE | ast.INTERVAL_MASK_SECOND))
		}
	| MINUTE_P TO IntervalSecond
		{
			$$ = $3
			$$.Items[0] = makeIntConst(int64(ast.INTERVAL_MASK_MINUTE | ast.INTERVAL_MASK_SECOND))
		}
	| /* EMPTY */
		{ $$ = nil }
	;

IntervalSecond:
	SECOND_P
		{
			$$ = makeList(makeIntConst(int64(ast.INTERVAL_MASK_SECOND)))
		}
	| SECOND_P '(' IConst ')'
		{
			$$ = makeList2(makeIntConst(int64(ast.INTERVAL_MASK_SECOND)), makeIntConst($3))
		}
	;

/*****************************************************************************
 *
 *      CHECKPOINT
 *
 *****************************************************************************/

CheckPointStmt:
	CHECKPOINT
		{
			$$ = &ast.CheckpointStmt{}
		}
	;

/*****************************************************************************
 *
 *      DISCARD
 *
 *****************************************************************************/

DiscardStmt:
	DISCARD ALL
		{
			$$ = &ast.DiscardStmt{Mode: "ALL"}
		}
	| DISCARD TEMP
		{
			$$ = &ast.DiscardStmt{Mode: "TEMP"}
		}
	| DISCARD TEMPORARY
		{
			$$ = &ast.DiscardStmt{Mode: "TEMP"}
		}
	| DISCARD PLANS
		{
			$$ = &ast.DiscardStmt{Mode: "PLANS"}
		}
	| DISCARD SEQUENCES
		{
			$$ = &ast.DiscardStmt{Mode: "SEQUENCES"}
		}
	;

/*****************************************************************************
 *
 *      LISTEN
 *
 *****************************************************************************/

ListenStmt:
	LISTEN ColId
		{
			$$ = &ast.ListenStmt{Channel: $2}
		}
	;

/*****************************************************************************
 *
 *      UNLISTEN
 *
 *****************************************************************************/

UnlistenStmt:
	UNLISTEN ColId
		{
			$$ = &ast.UnlistenStmt{Channel: $2}
		}
	| UNLISTEN '*'
		{
			$$ = &ast.UnlistenStmt{All: true}
		}
	;

/*****************************************************************************
 *
 *      NOTIFY
 *
 *****************************************************************************/

NotifyStmt:
	NOTIFY ColId
		{
			$$ = &ast.NotifyStmt{Channel: $2}
		}
	| NOTIFY ColId ',' SConst
		{
			$$ = &ast.NotifyStmt{Channel: $2, Payload: $4}
		}
	;

/*****************************************************************************
 *
 *      LOAD
 *
 *****************************************************************************/

LoadStmt:
	LOAD FileName
		{
			$$ = &ast.LoadStmt{Path: $2}
		}
	;

FileName:
	SConst { $$ = $1 }
	;

/*****************************************************************************
 *
 *      CLOSE cursor
 *
 *****************************************************************************/

ClosePortalStmt:
	CLOSE CursorName
		{
			$$ = &ast.CloseCursorStmt{Name: $2}
		}
	| CLOSE ALL
		{
			$$ = &ast.CloseCursorStmt{All: true}
		}
	;

CursorName:
	Name { $$ = $1 }
	;

/*****************************************************************************
 *
 *      SET CONSTRAINTS
 *
 *****************************************************************************/

ConstraintsSetStmt:
	SET CONSTRAINTS ConstraintsSetList ConstraintsSetMode
		{
			$$ = &ast.ConstraintsSetStmt{
				Constraints: columnOptions($3),
				Deferred:    $4,
			}
		}
	;

ConstraintsSetList:
	ALL
		{
			$$ = nil
		}
	| QualifiedNameList
		{
			$$ = $1
		}
	;

ConstraintsSetMode:
	DEFERRED  { $$ = true }
	| IMMEDIATE { $$ = false }
	;

QualifiedNameList:
	QualifiedName
		{
			$$ = makeList(makeRangeVar($1))
		}
	| QualifiedNameList ',' QualifiedName
		{
			$$ = appendList($1, makeRangeVar($3))
		}
	;

/*****************************************************************************
 *
 *      SET variable
 *
 *****************************************************************************/

VariableSetStmt:
	SET SetRest
		{
			$$ = $2
		}
	| SET LOCAL SetRest
		{
			$$ = setLocalStatement($3)
		}
	| SET SESSION SetRest
		{
			$$ = $3
		}
	;

SetRest:
	TRANSACTION TransactionModeList
		{
			$$ = &ast.VariableSetStmt{
				Kind: ast.VAR_SET_MULTI,
				Name: "TRANSACTION",
				Args: nodeList($2),
			}
		}
	| SESSION CHARACTERISTICS AS TRANSACTION TransactionModeList
		{
			$$ = &ast.VariableSetStmt{
				Kind: ast.VAR_SET_MULTI,
				Name: "SESSION CHARACTERISTICS",
				Args: nodeList($5),
			}
		}
	| SetRestMore
		{
			$$ = $1
		}
	;

GenericSet:
	VarName TO VarList
		{
			$$ = &ast.VariableSetStmt{
				Kind: ast.VAR_SET_VALUE,
				Name: $1,
				Args: $3,
			}
		}
	| VarName '=' VarList
		{
			$$ = &ast.VariableSetStmt{
				Kind: ast.VAR_SET_VALUE,
				Name: $1,
				Args: $3,
			}
		}
	| VarName TO DEFAULT
		{
			$$ = &ast.VariableSetStmt{
				Kind: ast.VAR_SET_DEFAULT,
				Name: $1,
			}
		}
	| VarName '=' DEFAULT
		{
			$$ = &ast.VariableSetStmt{
				Kind: ast.VAR_SET_DEFAULT,
				Name: $1,
			}
		}
	;

SetRestMore:
	GenericSet
		{
			$$ = $1
		}
	| VarName FROM CURRENT_P
		{
			$$ = &ast.VariableSetStmt{
				Kind: ast.VAR_SET_CURRENT,
				Name: $1,
			}
		}
	| TIME ZONE ZoneValue
		{
			n := &ast.VariableSetStmt{
				Kind: ast.VAR_SET_VALUE,
				Name: "timezone",
			}
			if $3 != nil {
				n.Args = makeList($3)
			} else {
				n.Kind = ast.VAR_SET_DEFAULT
			}
			$$ = n
		}
	| CATALOG_P SConst
		{
			yylex.Error("current database cannot be changed")
			$$ = nil
		}
	| SCHEMA SConst
		{
			$$ = &ast.VariableSetStmt{
				Kind: ast.VAR_SET_VALUE,
				Name: "search_path",
				Args: makeList(makeStringConst($2)),
			}
		}
	| NAMES OptEncoding
		{
			n := &ast.VariableSetStmt{
				Kind: ast.VAR_SET_VALUE,
				Name: "client_encoding",
			}
			if $2 != "" {
				n.Args = makeList(makeStringConst($2))
			} else {
				n.Kind = ast.VAR_SET_DEFAULT
			}
			$$ = n
		}
	| ROLE NonReservedWordOrSConst
		{
			$$ = &ast.VariableSetStmt{
				Kind: ast.VAR_SET_VALUE,
				Name: "role",
				Args: makeList(makeStringConst($2)),
			}
		}
	| ROLE DEFAULT
		{
			$$ = &ast.VariableSetStmt{Kind: ast.VAR_SET_DEFAULT, Name: "role"}
		}
	| SESSION AUTHORIZATION NonReservedWordOrSConst
		{
			$$ = &ast.VariableSetStmt{
				Kind: ast.VAR_SET_VALUE,
				Name: "session_authorization",
				Args: makeList(makeStringConst($3)),
			}
		}
	| SESSION AUTHORIZATION DEFAULT
		{
			$$ = &ast.VariableSetStmt{
				Kind: ast.VAR_SET_DEFAULT,
				Name: "session_authorization",
			}
		}
	| XML_P OPTION DocumentOrContent
		{
			var val string
			if $3 == int64(ast.XMLOPTION_DOCUMENT) {
				val = "DOCUMENT"
			} else {
				val = "CONTENT"
			}
			$$ = &ast.VariableSetStmt{
				Kind: ast.VAR_SET_VALUE,
				Name: "xmloption",
				Args: makeList(makeStringConst(val)),
			}
		}
	| TRANSACTION SNAPSHOT SConst
		{
			$$ = &ast.VariableSetStmt{
				Kind: ast.VAR_SET_MULTI,
				Name: "TRANSACTION SNAPSHOT",
				Args: makeList(makeStringConst($3)),
			}
		}
	;

VarName:
	ColId
		{
			$$ = $1
		}
	| VarName '.' ColId
		{
			$$ = $1 + "." + $3
		}
	;

VarList:
	VarValue
		{
			$$ = makeList($1)
		}
	| VarList ',' VarValue
		{
			$$ = appendList($1, $3)
		}
	;

VarValue:
	OptBooleanOrString
		{
			$$ = makeStringConst($1)
		}
	| NumericOnly
		{
			$$ = &ast.A_Const{Val: $1}
		}
	;

ZoneValue:
	SConst
		{
			$$ = makeStringConst($1)
		}
	| IDENT
		{
			$$ = makeStringConst($1)
		}
	| NumericOnly
		{
			$$ = &ast.A_Const{Val: $1}
		}
	| DEFAULT
		{
			$$ = nil
		}
	| LOCAL
		{
			$$ = nil
		}
	;

OptEncoding:
	SConst      { $$ = $1 }
	| DEFAULT   { $$ = "" }
	| /* EMPTY */ { $$ = "" }
	;

IsoLevel:
	READ UNCOMMITTED  { $$ = "read uncommitted" }
	| READ COMMITTED  { $$ = "read committed" }
	| REPEATABLE READ { $$ = "repeatable read" }
	| SERIALIZABLE    { $$ = "serializable" }
	;

TransactionModeItem:
	ISOLATION LEVEL IsoLevel
		{
			$$ = makeDefElem("transaction_isolation", makeStringConst($3))
		}
	| READ ONLY
		{
			$$ = makeDefElem("transaction_read_only", makeIntConst(1))
		}
	| READ WRITE
		{
			$$ = makeDefElem("transaction_read_only", makeIntConst(0))
		}
	| DEFERRABLE
		{
			$$ = makeDefElem("transaction_deferrable", makeIntConst(1))
		}
	| NOT DEFERRABLE
		{
			$$ = makeDefElem("transaction_deferrable", makeIntConst(0))
		}
	;

TransactionModeList:
	TransactionModeItem
		{
			$$ = makeList($1)
		}
	| TransactionModeList ',' TransactionModeItem
		{
			$$ = appendList($1, $3)
		}
	| TransactionModeList TransactionModeItem
		{
			$$ = appendList($1, $2)
		}
	;

TransactionModeListOrEmpty:
	TransactionModeList { $$ = $1 }
	| /* EMPTY */         { $$ = nil }
	;

/*****************************************************************************
 *
 *      SHOW variable
 *
 *****************************************************************************/

VariableShowStmt:
	SHOW VarName
		{
			$$ = &ast.ShowStmt{
				Tp:     showStmtType($2),
				DBName: $2,
			}
		}
	| SHOW TIME ZONE
		{
			$$ = &ast.ShowStmt{
				Tp:     ast.ShowVariables,
				DBName: "timezone",
			}
		}
	| SHOW TRANSACTION ISOLATION LEVEL
		{
			$$ = &ast.ShowStmt{
				Tp:     ast.ShowVariables,
				DBName: "transaction_isolation",
			}
		}
	| SHOW SESSION AUTHORIZATION
		{
			$$ = &ast.ShowStmt{
				Tp:     ast.ShowVariables,
				DBName: "session_authorization",
			}
		}
	| SHOW ALL
		{
			$$ = &ast.ShowStmt{
				Tp:     ast.ShowVariables,
				DBName: "all",
			}
		}
	;

/*****************************************************************************
 *
 *      RESET variable
 *
 *****************************************************************************/

VariableResetStmt:
	RESET ResetRest
		{
			$$ = $2
		}
	;

ResetRest:
	GenericReset
		{
			$$ = $1
		}
	| TIME ZONE
		{
			$$ = &ast.VariableSetStmt{
				Kind: ast.VAR_RESET,
				Name: "timezone",
			}
		}
	| TRANSACTION ISOLATION LEVEL
		{
			$$ = &ast.VariableSetStmt{
				Kind: ast.VAR_RESET,
				Name: "transaction_isolation",
			}
		}
	| SESSION AUTHORIZATION
		{
			$$ = &ast.VariableSetStmt{
				Kind: ast.VAR_RESET,
				Name: "session_authorization",
			}
		}
	;

GenericReset:
	VarName
		{
			$$ = &ast.VariableSetStmt{
				Kind: ast.VAR_RESET,
				Name: $1,
			}
		}
	| ALL
		{
			$$ = &ast.VariableSetStmt{
				Kind: ast.VAR_RESET_ALL,
			}
		}
	;

/*****************************************************************************
 *
 *      PREPARE / EXECUTE / DEALLOCATE
 *
 *****************************************************************************/

PreparedStmt:
	PREPARE Name PrepTypeClause AS PreparableStmt
		{
			$$ = &ast.PrepareStmt{
				Name:   syntaxIdent(yylex, $2, yyS[yypt-3].span),
				Params: $3,
				Stmt:   $5.(ast.StmtNode),
			}
		}
	;

PrepTypeClause:
	'(' TypeList ')'  { $$ = $2 }
	| /* EMPTY */       { $$ = nil }
	;

TypeList:
	Typename
		{
			$$ = []*types.FieldType{$1}
		}
	| TypeList ',' Typename
		{
			$$ = append($1, $3)
		}
	;

PreparableStmt:
	SelectStmt  { $$ = $1 }
	| InsertStmt { $$ = $1 }
	| UpdateStmt { $$ = $1 }
	| DeleteStmt { $$ = $1 }
	| MergeStmt  { $$ = $1 }
	;

ExecuteStmt:
	EXECUTE Name ExecuteParamClause
		{
			$$ = &ast.ExecuteStmt{
				Name: syntaxIdent(yylex, $2, yyS[yypt-1].span),
				Args: exprsFromList($3),
			}
		}
	;

ExecuteParamClause:
	'(' ExpressionList ')'  { $$ = $2 }
	| /* EMPTY */       { $$ = nil }
	;

DeallocateStmt:
	DEALLOCATE Name
		{
			$$ = &ast.DeallocateStmt{
				Name: syntaxIdent(yylex, $2, yyS[yypt].span),
			}
		}
	| DEALLOCATE PREPARE Name
		{
			$$ = &ast.DeallocateStmt{
				Name: syntaxIdent(yylex, $3, yyS[yypt].span),
			}
		}
	| DEALLOCATE ALL
		{
			$$ = &ast.DeallocateStmt{
				All: true,
			}
		}
	| DEALLOCATE PREPARE ALL
		{
			$$ = &ast.DeallocateStmt{
				All: true,
			}
		}
	;

/*****************************************************************************
 *
 *      TRUNCATE TABLE
 *
 *****************************************************************************/

TruncateTableStmt:
	TRUNCATE OptTable RelationExprList OptRestartSequences OptDropBehavior
		{
			tables := tableNamesFromList($3)
			stmt := &ast.TruncateTableStmt{
				Tables:   tables,
				Behavior: dropBehaviorName(int($5)),
			}
			stmt.Only = truncateOnly($3)
			if len(tables) > 0 {
				stmt.Table = tables[0]
			}
			$$ = stmt
		}
	;

OptRestartSequences:
	CONTINUE_P IDENTITY_P { $$ = false }
	| RESTART IDENTITY_P  { $$ = true }
	| /* EMPTY */          { $$ = false }
	;

OptTable:
	TABLE
		{ $$ = true }
	| /* EMPTY */
		{ $$ = false }
	;

RelationExprList:
	TableName
		{
			$$ = makeList($1)
		}
	| RelationExprList ',' TableName
		{
			$$ = appendList($1, $3)
		}
	;

/*****************************************************************************
 *
 *      LOCK TABLE
 *
 *****************************************************************************/

LockStmt:
	LOCK_P OptTable RelationExprList OptLock OptNowait
		{
			$$ = &ast.LockStmt{
				Relations: $3,
				Mode:      int($4),
				Nowait:    $5,
			}
		}
	;

OptLock:
	IN_P LockType MODE { $$ = $2 }
	| /* EMPTY */       { $$ = int64(ast.AccessExclusiveLock) }
	;

LockType:
	ACCESS SHARE                 { $$ = int64(ast.AccessShareLock) }
	| ROW SHARE                  { $$ = int64(ast.RowShareLock) }
	| ROW EXCLUSIVE              { $$ = int64(ast.RowExclusiveLock) }
	| SHARE UPDATE EXCLUSIVE     { $$ = int64(ast.ShareUpdateExclusiveLock) }
	| SHARE                      { $$ = int64(ast.ShareLock) }
	| SHARE ROW EXCLUSIVE        { $$ = int64(ast.ShareRowExclusiveLock) }
	| EXCLUSIVE                  { $$ = int64(ast.ExclusiveLock) }
	| ACCESS EXCLUSIVE           { $$ = int64(ast.AccessExclusiveLock) }
	;

OptNowait:
	NOWAIT      { $$ = true }
	| /* EMPTY */ { $$ = false }
	;

/*****************************************************************************
 *
 *      VACUUM / ANALYZE
 *
 *****************************************************************************/

VacuumStmt:
	VACUUM OptFull OptFreeze OptVerbose OptAnalyze OptVacuumRelationList
		{
			n := &ast.VacuumStmt{
				Relations: $6,
			}
			var opts *ast.List
			if $2 {
				opts = appendList(opts, makeDefElem("full", nil))
			}
			if $3 {
				opts = appendList(opts, makeDefElem("freeze", nil))
			}
			if $4 {
				opts = appendList(opts, makeDefElem("verbose", nil))
			}
			if $5 {
				opts = appendList(opts, makeDefElem("analyze", nil))
			}
			n.Options = opts
			$$ = n
		}
	| VACUUM '(' UtilityOptionList ')' OptVacuumRelationList
		{
			$$ = &ast.VacuumStmt{
				Options: $3,
				Relations: $5,
			}
		}
	;

AnalyzeStmt:
	AnalyzeKeyword OptVerbose OptVacuumRelationList
		{
			n := &ast.AnalyzeTableStmt{Relations: $3}
			if $2 { n.Options = makeList(makeDefElem("verbose", nil)) }
			$$ = n
		}
	| AnalyzeKeyword '(' UtilityOptionList ')' OptVacuumRelationList
		{
			$$ = &ast.AnalyzeTableStmt{
				Options: $3,
				Relations: $5,
			}
		}
	;

AnalyzeKeyword:
	ANALYZE
	| ANALYSE
	;

OptFull:
	FULL        { $$ = true }
	| /* EMPTY */ { $$ = false }
	;

OptFreeze:
	FREEZE      { $$ = true }
	| /* EMPTY */ { $$ = false }
	;

OptVerbose:
	VERBOSE     { $$ = true }
	| /* EMPTY */ { $$ = false }
	;

OptAnalyze:
	AnalyzeKeyword { $$ = true }
	| /* EMPTY */   { $$ = false }
	;

VacuumRelation:
	QualifiedName OptColumnList
		{
			$$ = &ast.VacuumRelation{Relation: tableNameFromList($1), VaCols: $2}
		}
	;

VacuumRelationList:
	VacuumRelation
		{
			$$ = makeList($1)
		}
	| VacuumRelationList ',' VacuumRelation
		{
			$$ = appendList($1, $3)
		}
	;

OptVacuumRelationList:
	VacuumRelationList { $$ = $1 }
	| /* EMPTY */        { $$ = nil }
	;

/*****************************************************************************
 *
 *      CLUSTER
 *
 *****************************************************************************/

ClusterStmt:
	CLUSTER '(' UtilityOptionList ')' QualifiedName ClusterIndexSpecification
		{
			$$ = &ast.ClusterStmt{
				Table:   tableNameFromList($5),
				Index:   indexName($6),
				Verbose: hasDefElem($3, "verbose"),
			}
		}
	| CLUSTER '(' UtilityOptionList ')'
		{
			$$ = &ast.ClusterStmt{
				Verbose: hasDefElem($3, "verbose"),
			}
		}
	| CLUSTER OptVerbose QualifiedName ClusterIndexSpecification
		{
			$$ = &ast.ClusterStmt{
				Table:   tableNameFromList($3),
				Index:   indexName($4),
				Verbose: $2,
			}
		}
	| CLUSTER OptVerbose
		{
			$$ = &ast.ClusterStmt{
				Verbose: $2,
			}
		}
	/* backwards compatible syntax from before 8.3 */
	| CLUSTER OptVerbose Name ON QualifiedName
		{
			$$ = &ast.ClusterStmt{
				Table:   tableNameFromList($5),
				Index:   indexName($3),
				Verbose: $2,
			}
		}
	;

ClusterIndexSpecification:
	USING Name   { $$ = $2 }
	| /* EMPTY */ { $$ = "" }
	;

/*****************************************************************************
 *
 *      REINDEX
 *
 *****************************************************************************/

ReindexStmt:
	REINDEX OptReindexOptionList ReindexTargetType OptConcurrently QualifiedName
		{
			$$ = &ast.ReindexStmt{
				ObjectType:   reindexObjectName($3),
				Name:         nameFromList($5),
				Concurrently: $4,
				Verbose:      hasDefElem($2, "verbose"),
			}
		}
	| REINDEX OptReindexOptionList SCHEMA OptConcurrently Name
		{
			$$ = &ast.ReindexStmt{
				ObjectType:   "SCHEMA",
				Name:         $5,
				Concurrently: $4,
				Verbose:      hasDefElem($2, "verbose"),
			}
		}
	| REINDEX OptReindexOptionList ReindexTargetMultitable OptConcurrently OptName
		{
			$$ = &ast.ReindexStmt{
				ObjectType:   reindexObjectName($3),
				Name:         $5,
				Concurrently: $4,
				Verbose:      hasDefElem($2, "verbose"),
			}
		}
	;

OptName:
	Name             { $$ = $1 }
	| /* EMPTY */    { $$ = "" }
	;

ReindexTargetType:
	INDEX { $$ = int64(ast.REINDEX_OBJECT_INDEX) }
	| TABLE { $$ = int64(ast.REINDEX_OBJECT_TABLE) }
	;

ReindexTargetMultitable:
	SYSTEM_P { $$ = int64(ast.REINDEX_OBJECT_SYSTEM) }
	| DATABASE { $$ = int64(ast.REINDEX_OBJECT_DATABASE) }
	;

OptReindexOptionList:
	'(' UtilityOptionList ')' { $$ = $2 }
	| /* EMPTY */               { $$ = nil }
	;

/*****************************************************************************
 *
 *      COMMENT ON
 *
 *****************************************************************************/


CommentStmt:
	COMMENT ON ObjectTypeAnyName AnyName IS CommentText
		{
			$$ = &ast.CommentStmt{
				ObjectType: strings.ToLower(objectTypeName($3)),
				ObjectName: commentNameParts($4),
				Comment:    commentPtr($6),
			}
		}
	| COMMENT ON COLUMN ColumnRef IS CommentText
		{
			$$ = &ast.CommentStmt{
				ObjectType: strings.ToLower(objectTypeName(int64(ast.OBJECT_COLUMN))),
				ObjectName: commentNameParts($4),
				Comment:    commentPtr($6),
			}
		}
	| COMMENT ON ObjectTypeName Name IS CommentText
		{
			$$ = &ast.CommentStmt{
				ObjectType: strings.ToLower(objectTypeName($3)),
				ObjectName: commentNameParts(&ast.String{Str: $4}),
				Comment:    commentPtr($6),
			}
		}
	| COMMENT ON TYPE_P AnyName IS CommentText
		{
			$$ = &ast.CommentStmt{
				ObjectType: strings.ToLower(objectTypeName(int64(ast.OBJECT_TYPE))),
				ObjectName: commentNameParts($4),
				Comment:    commentPtr($6),
			}
		}
	| COMMENT ON DOMAIN_P AnyName IS CommentText
		{
			$$ = &ast.CommentStmt{
				ObjectType: strings.ToLower(objectTypeName(int64(ast.OBJECT_DOMAIN))),
				ObjectName: commentNameParts($4),
				Comment:    commentPtr($6),
			}
		}
	| COMMENT ON AGGREGATE AggregateWithArgTypes IS CommentText
		{
			$$ = &ast.CommentStmt{
				ObjectType: strings.ToLower(objectTypeName(int64(ast.OBJECT_AGGREGATE))),
				ObjectName: commentNameParts($4),
				Suffix:     commentArgsSuffix($4),
				Comment:    commentPtr($6),
			}
		}
	| COMMENT ON FUNCTION FunctionWithArgTypes IS CommentText
		{
			$$ = &ast.CommentStmt{
				ObjectType: strings.ToLower(objectTypeName(int64(ast.OBJECT_FUNCTION))),
				ObjectName: commentNameParts($4),
				Suffix:     commentArgsSuffix($4),
				Comment:    commentPtr($6),
			}
		}
	| COMMENT ON PROCEDURE FunctionWithArgTypes IS CommentText
		{
			$$ = &ast.CommentStmt{
				ObjectType: strings.ToLower(objectTypeName(int64(ast.OBJECT_PROCEDURE))),
				ObjectName: commentNameParts($4),
				Suffix:     commentArgsSuffix($4),
				Comment:    commentPtr($6),
			}
		}
	| COMMENT ON ROUTINE FunctionWithArgTypes IS CommentText
		{
			$$ = &ast.CommentStmt{
				ObjectType: strings.ToLower(objectTypeName(int64(ast.OBJECT_ROUTINE))),
				ObjectName: commentNameParts($4),
				Suffix:     commentArgsSuffix($4),
				Comment:    commentPtr($6),
			}
		}
	| COMMENT ON OPERATOR OperatorWithArgTypes IS CommentText
		{
			$$ = &ast.CommentStmt{
				ObjectType: strings.ToLower(objectTypeName(int64(ast.OBJECT_OPERATOR))),
				ObjectName: commentNameParts($4),
				Suffix:     commentArgsSuffix($4),
				Comment:    commentPtr($6),
			}
		}
	| COMMENT ON CONSTRAINT Name ON AnyName IS CommentText
		{
			$$ = &ast.CommentStmt{
				ObjectType: "tabconstraint",
				ObjectName: []string{$4},
				ParentName: commentNameParts($6),
				Comment:    commentPtr($8),
			}
		}
	| COMMENT ON CONSTRAINT Name ON DOMAIN_P AnyName IS CommentText
		{
			$$ = &ast.CommentStmt{
				ObjectType: "domconstraint",
				ObjectName: []string{$4},
				ParentName: commentNameParts($7),
				Comment:    commentPtr($9),
			}
		}
	| COMMENT ON ObjectTypeNameOnAnyName Name ON AnyName IS CommentText
		{
			$$ = &ast.CommentStmt{
				ObjectType: strings.ToLower(objectTypeName($3)),
				ObjectName: []string{$4},
				ParentName: commentNameParts($6),
				Comment:    commentPtr($8),
			}
		}
	| COMMENT ON TRANSFORM FOR Typename LANGUAGE Name IS CommentText
		{
			$$ = &ast.CommentStmt{
				ObjectType: strings.ToLower(objectTypeName(int64(ast.OBJECT_TRANSFORM))),
				Suffix:     commentNodeString(typeNameNode($5)),
				ExtraName:  []string{$7},
				Comment:    commentPtr($9),
			}
		}
	| COMMENT ON OPERATOR CLASS AnyName USING Name IS CommentText
		{
			$$ = &ast.CommentStmt{
				ObjectType: strings.ToLower(objectTypeName(int64(ast.OBJECT_OPCLASS))),
				ObjectName: commentNameParts($5),
				ExtraName:  []string{$7},
				Comment:    commentPtr($9),
			}
		}
	| COMMENT ON OPERATOR FAMILY AnyName USING Name IS CommentText
		{
			$$ = &ast.CommentStmt{
				ObjectType: strings.ToLower(objectTypeName(int64(ast.OBJECT_OPFAMILY))),
				ObjectName: commentNameParts($5),
				ExtraName:  []string{$7},
				Comment:    commentPtr($9),
			}
		}
	| COMMENT ON LARGE_P OBJECT_P NumericOnly IS CommentText
		{
			$$ = &ast.CommentStmt{
				ObjectType: strings.ToLower(objectTypeName(int64(ast.OBJECT_LARGEOBJECT))),
				ObjectName: commentNameParts($5),
				Comment:    commentPtr($7),
			}
		}
	| COMMENT ON FOREIGN DATA_P WRAPPER Name IS CommentText
		{
			$$ = &ast.CommentStmt{
				ObjectType: strings.ToLower(objectTypeName(int64(ast.OBJECT_FDW))),
				ObjectName: commentNameParts(&ast.String{Str: $6}),
				Comment:    commentPtr($8),
			}
		}
	| COMMENT ON CAST '(' Typename AS Typename ')' IS CommentText
		{
			$$ = &ast.CommentStmt{
				ObjectType: strings.ToLower(objectTypeName(int64(ast.OBJECT_CAST))),
				Suffix:     commentNodeString(typeNameNode($5)) + " AS " + commentNodeString(typeNameNode($7)),
				Comment:    commentPtr($10),
			}
		}
	| COMMENT ON EVENT TRIGGER Name IS CommentText
		{
			$$ = &ast.CommentStmt{
				ObjectType: strings.ToLower(objectTypeName(int64(ast.OBJECT_EVENT_TRIGGER))),
				ObjectName: commentNameParts(&ast.String{Str: $5}),
				Comment:    commentPtr($7),
			}
		}
	;

CommentText:
	SConst      { $$ = $1 }
	| NULL_P    { $$ = "" }
	;

ObjectTypeName:
	SCHEMA       { $$ = int64(ast.OBJECT_SCHEMA) }
	| DATABASE   { $$ = int64(ast.OBJECT_DATABASE) }
	| ROLE       { $$ = int64(ast.OBJECT_ROLE) }
	| TABLESPACE { $$ = int64(ast.OBJECT_TABLESPACE) }
	| SUBSCRIPTION { $$ = int64(ast.OBJECT_SUBSCRIPTION) }
	| PUBLICATION  { $$ = int64(ast.OBJECT_PUBLICATION) }
	| SERVER       { $$ = int64(ast.OBJECT_FOREIGN_SERVER) }
	;

ObjectTypeNameOnAnyName:
	POLICY      { $$ = int64(ast.OBJECT_POLICY) }
	| RULE      { $$ = int64(ast.OBJECT_RULE) }
	| TRIGGER   { $$ = int64(ast.OBJECT_TRIGGER) }
	;

/*****************************************************************************
 *
 *      SECURITY LABEL
 *
 *****************************************************************************/

SecLabelStmt:
	SECURITY LABEL OptProvider ON ObjectTypeAnyName AnyName IS SConst
		{
			$$ = &ast.SecurityLabelStmt{
				Provider:   $3,
				ObjectType: objectTypeName($5),
				ObjectName: nameListString($6),
				Label:      $8,
			}
		}
	| SECURITY LABEL OptProvider ON COLUMN AnyName IS SConst
		{
			$$ = &ast.SecurityLabelStmt{
				Provider:   $3,
				ObjectType: "COLUMN",
				ObjectName: nameListString($6),
				Label:      $8,
			}
		}
	| SECURITY LABEL OptProvider ON ObjectTypeName Name IS SConst
		{
			$$ = &ast.SecurityLabelStmt{
				Provider:   $3,
				ObjectType: objectTypeName($5),
				ObjectName: $6,
				Label:      $8,
			}
		}
	;

/*****************************************************************************
 *
 *      DECLARE CURSOR
 *
 *****************************************************************************/

DeclareCursorStmt:
	DECLARE CursorName CursorOptions CURSOR OptHold FOR SelectStmt
		{
			opts := $3
			$$ = &ast.DeclareCursorStmt{
				Name:     $2,
				Binary:   opts&ast.CURSOR_OPT_BINARY != 0,
				Scroll:   opts&ast.CURSOR_OPT_SCROLL != 0,
				NoScroll: opts&ast.CURSOR_OPT_NO_SCROLL != 0,
				Hold:     $5&ast.CURSOR_OPT_HOLD != 0,
				Select:   $7.(ast.StmtNode),
			}
		}
	;

CursorOptions:
	/* EMPTY */
		{
			$$ = 0
		}
	| CursorOptions NO SCROLL
		{
			$$ = $1 | ast.CURSOR_OPT_NO_SCROLL
		}
	| CursorOptions SCROLL
		{
			$$ = $1 | ast.CURSOR_OPT_SCROLL
		}
	| CursorOptions BINARY
		{
			$$ = $1 | ast.CURSOR_OPT_BINARY
		}
	| CursorOptions ASENSITIVE
		{
			$$ = $1 | ast.CURSOR_OPT_ASENSITIVE
		}
	| CursorOptions INSENSITIVE
		{
			$$ = $1 | ast.CURSOR_OPT_INSENSITIVE
		}
	;

OptHold:
	/* EMPTY */
		{
			$$ = 0
		}
	| WITH HOLD
		{
			$$ = ast.CURSOR_OPT_HOLD
		}
	| WITHOUT HOLD
		{
			$$ = 0
		}
	;

/*****************************************************************************
 *
 *      FETCH / MOVE
 *
 *****************************************************************************/

FetchStmt:
	FETCH FetchArgs
		{
			n := $2.(*ast.FetchCursorStmt)
			n.IsMove = false
			$$ = n
		}
	| MOVE FetchArgs
		{
			n := $2.(*ast.FetchCursorStmt)
			n.IsMove = true
			$$ = n
		}
	;

FetchArgs:
	CursorName
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "FORWARD",
				Count:     1,
				Name:      $1,
			}
		}
	| FromIn CursorName
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "FORWARD",
				Count:     1,
				Name:      $2,
			}
		}
	| NEXT OptFromIn CursorName
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "NEXT",
				Count:     1,
				Name:      $3,
			}
		}
	| PRIOR OptFromIn CursorName
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "PRIOR",
				Count:     1,
				Name:      $3,
			}
		}
	| FIRST_P OptFromIn CursorName
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "FIRST",
				Count:     1,
				Name:      $3,
			}
		}
	| LAST_P OptFromIn CursorName
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "LAST",
				Count:     1,
				Name:      $3,
			}
		}
	| ABSOLUTE_P SignedIconst OptFromIn CursorName
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "ABSOLUTE",
				Count:     $2,
				Name:      $4,
			}
		}
	| RELATIVE_P SignedIconst OptFromIn CursorName
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "RELATIVE",
				Count:     $2,
				Name:      $4,
			}
		}
	| SignedIconst OptFromIn CursorName
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "FORWARD",
				Count:     $1,
				Name:      $3,
			}
		}
	| ALL OptFromIn CursorName
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "ALL",
				Count:     ast.FETCH_ALL,
				Name:      $3,
			}
		}
	| FORWARD OptFromIn CursorName
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "FORWARD",
				Count:     1,
				Name:      $3,
			}
		}
	| FORWARD ALL OptFromIn CursorName
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "FORWARD",
				Count:     ast.FETCH_ALL,
				Name:      $4,
			}
		}
	| FORWARD SignedIconst OptFromIn CursorName
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "FORWARD",
				Count:     $2,
				Name:      $4,
			}
		}
	| SignedIconst FORWARD OptFromIn CursorName
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "FORWARD",
				Count:     $1,
				Name:      $4,
			}
		}
	| BACKWARD OptFromIn CursorName
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "BACKWARD",
				Count:     1,
				Name:      $3,
			}
		}
	| BACKWARD SignedIconst OptFromIn CursorName
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "BACKWARD",
				Count:     $2,
				Name:      $4,
			}
		}
	| SignedIconst BACKWARD OptFromIn CursorName
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "BACKWARD",
				Count:     $1,
				Name:      $4,
			}
		}
	| BACKWARD ALL OptFromIn CursorName
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "BACKWARD",
				Count:     ast.FETCH_ALL,
				Name:      $4,
			}
		}
	;

FromIn:
	FROM
	| IN_P
	;

OptFromIn:
	FromIn
	| /* EMPTY */
	;

/*****************************************************************************
 *
 *      MERGE INTO
 *
 *****************************************************************************/

MergeStmt:
	OptWithClause MERGE INTO RelationExprOptAlias USING TableRef ON AExpr MergeWhenList ReturningOpt
		{
			m := &ast.MergeStmt{
				Table:  syntaxFromClause(resultSetNodeOf($4)),
				Source: syntaxFromClause(resultSetNodeOf($6)),
				On:     $8,
				Whens:  mergeWhenClauses($9),
			}
			m.Returning = returningFieldList($10)
			if $1 != nil {
				m.With = $1.(*ast.WithClause)
			}
			$$ = m
		}
	;

MergeWhenList:
	MergeWhenClause
		{
			$$ = makeList($1)
		}
	| MergeWhenList MergeWhenClause
		{
			$$ = appendList($1, $2)
		}
	;

MergeWhenClause:
	MergeWhenTgtMatched OptMergeWhenCondition THEN MergeUpdate
		{
			n := $4.(*ast.MergeWhenClause)
			n.Matched = true
			n.Condition = $2
			$$ = n
		}
	| MergeWhenTgtMatched OptMergeWhenCondition THEN MergeDelete
		{
			n := $4.(*ast.MergeWhenClause)
			n.Matched = true
			n.Condition = $2
			$$ = n
		}
	| MergeWhenTgtNotMatched OptMergeWhenCondition THEN MergeDelete
		{
			n := $4.(*ast.MergeWhenClause)
			n.Matched = false
			n.Condition = $2
			$$ = n
		}
	| MergeWhenTgtNotMatched OptMergeWhenCondition THEN MergeUpdate
		{
			n := $4.(*ast.MergeWhenClause)
			n.Matched = false
			n.Condition = $2
			$$ = n
		}
	| MergeWhenTgtMatched OptMergeWhenCondition THEN MergeInsert
		{
			n := $4.(*ast.MergeWhenClause)
			n.Matched = true
			n.Condition = $2
			$$ = n
		}
	| MergeWhenTgtNotMatched OptMergeWhenCondition THEN MergeInsert
		{
			n := $4.(*ast.MergeWhenClause)
			n.Matched = false
			n.Condition = $2
			$$ = n
		}
	| MergeWhenTgtMatched OptMergeWhenCondition THEN DO NOTHING
		{
			$$ = &ast.MergeWhenClause{
				Matched:   true,
				Condition: $2,
			}
		}
	| MergeWhenTgtNotMatched OptMergeWhenCondition THEN DO NOTHING
		{
			$$ = &ast.MergeWhenClause{
				Matched:   false,
				Condition: $2,
			}
		}
	;

MergeWhenTgtMatched:
	WHEN MATCHED
		{
			$$ = int64(ast.MERGE_WHEN_MATCHED)
		}
	;

MergeWhenTgtNotMatched:
	WHEN NOT MATCHED
		{
			$$ = int64(ast.MERGE_WHEN_NOT_MATCHED_BY_TARGET)
		}
	| WHEN NOT MATCHED BY TARGET
		{
			$$ = int64(ast.MERGE_WHEN_NOT_MATCHED_BY_TARGET)
		}
	| WHEN NOT MATCHED BY SOURCE
		{
			$$ = int64(ast.MERGE_WHEN_NOT_MATCHED_BY_SOURCE)
		}
	;

OptMergeWhenCondition:
	AND AExpr
		{
			$$ = $2
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

MergeUpdate:
	UPDATE SET AssignmentList
		{
			$$ = &ast.MergeWhenClause{
				Tp:         ast.MergeWhenMatchedUpdate,
				UpdateList: $3,
			}
		}
	;

MergeDelete:
	DELETE_P
		{
			$$ = &ast.MergeWhenClause{
				Tp: ast.MergeWhenMatchedDelete,
			}
		}
	;

MergeInsert:
	INSERT MergeValuesClause
		{
			$$ = &ast.MergeWhenClause{
				Tp:     ast.MergeWhenNotMatchedInsert,
				Values: $2,
			}
		}
	| INSERT OVERRIDING OverrideKind VALUE_P MergeValuesClause
		{
			$$ = &ast.MergeWhenClause{
				Tp:     ast.MergeWhenNotMatchedInsert,
				Values: $5,
			}
		}
	| INSERT '(' InsertColumnList ')' MergeValuesClause
		{
			$$ = &ast.MergeWhenClause{
				Tp:      ast.MergeWhenNotMatchedInsert,
				Columns: $3,
				Values:  $5,
			}
		}
	| INSERT '(' InsertColumnList ')' OVERRIDING OverrideKind VALUE_P MergeValuesClause
		{
			$$ = &ast.MergeWhenClause{
				Tp:      ast.MergeWhenNotMatchedInsert,
				Columns: $3,
				Values:  $8,
			}
		}
	| INSERT DEFAULT VALUES
		{
			$$ = &ast.MergeWhenClause{
				Tp: ast.MergeWhenNotMatchedInsert,
			}
		}
	;

MergeValuesClause:
	VALUES '(' ExpressionList ')'
		{
			$$ = $3
		}
	;

OverrideKind:
	USER
		{
			$$ = int64(ast.OVERRIDING_USER_VALUE)
		}
	| SYSTEM_P
		{
			$$ = int64(ast.OVERRIDING_SYSTEM_VALUE)
		}
	;

/*****************************************************************************
 *
 *      CALL
 *
 *****************************************************************************/

CallStmt:
	CALL FuncApplication
		{
			$$ = &ast.CallStmt{
				Proc: $2.(*ast.FuncCallExpr),
			}
		}
	;

/*****************************************************************************
 *
 *      DO (anonymous code block)
 *
 *****************************************************************************/

DoStmt:
	DO DoStmtOptList
		{
			if len(doExprs($2)) == 0 { yylex.Error("DO statement has no code") }
			$$ = &ast.DoStmt{
				Exprs:    doExprs($2),
				Language: doLanguage($2),
			}
		}
	;

DoStmtOptList:
	DoStmtOptItem
		{
			$$ = makeList($1)
		}
	| DoStmtOptList DoStmtOptItem
		{
			$$ = appendList($1, $2)
		}
	;

DoStmtOptItem:
	SConst
		{
			$$ = &ast.DefElem{
				Defname: "as",
				Arg:     &ast.String{Str: $1},
			}
		}
	| LANGUAGE NonReservedWordOrSConst
		{
			$$ = &ast.DefElem{
				Defname: "language",
				Arg:     &ast.String{Str: $2},
			}
		}
	;

OptProvider:
	FOR NonReservedWordOrSConst { $$ = $2 }
	| /* EMPTY */                  { $$ = "" }
	;


/*****************************************************************************
 *
 *      ALTER FUNCTION / ALTER PROCEDURE / ALTER ROUTINE
 *
 *****************************************************************************/

AlterFunctionStmt:
	ALTER FUNCTION FunctionWithArgTypes AlterFuncOptList OptRestrict
		{
			$$ = &ast.AlterFunctionStmt{
				Objtype: "FUNCTION",
				Func:    $3,
				Actions: defElems($4),
			}
		}
	| ALTER PROCEDURE FunctionWithArgTypes AlterFuncOptList OptRestrict
		{
			$$ = &ast.AlterFunctionStmt{
				Objtype: "PROCEDURE",
				Func:    $3,
				Actions: defElems($4),
			}
		}
	| ALTER ROUTINE FunctionWithArgTypes AlterFuncOptList OptRestrict
		{
			$$ = &ast.AlterFunctionStmt{
				Objtype: "ROUTINE",
				Func:    $3,
				Actions: defElems($4),
			}
		}
	;

AlterFuncOptList:
	CommonFuncOptItem
		{ $$ = makeList($1) }
	| AlterFuncOptList CommonFuncOptItem
		{ $$ = appendList($1, $2) }
	;

/* Ignored, merely for SQL compliance */
OptRestrict:
	RESTRICT
	| /* EMPTY */
	;

/*****************************************************************************
 *
 *      DROP FUNCTION / DROP PROCEDURE / DROP ROUTINE
 *
 *****************************************************************************/

RemoveFuncStmt:
	DROP FUNCTION FunctionWithArgTypesList OptDropBehavior
		{
			$$ = dropFunctionStmt(false, $3, int($4))
		}
	| DROP FUNCTION IF_P EXISTS FunctionWithArgTypesList OptDropBehavior
		{
			$$ = dropFunctionStmt(true, $5, int($6))
		}
	| DROP PROCEDURE FunctionWithArgTypesList OptDropBehavior
		{
			$$ = dropProcedureStmt(false, $3, int($4))
		}
	| DROP PROCEDURE IF_P EXISTS FunctionWithArgTypesList OptDropBehavior
		{
			$$ = dropProcedureStmt(true, $5, int($6))
		}
	| DROP ROUTINE FunctionWithArgTypesList OptDropBehavior
		{
			$$ = dropRoutineStmt(false, $3)
		}
	| DROP ROUTINE IF_P EXISTS FunctionWithArgTypesList OptDropBehavior
		{
			$$ = dropRoutineStmt(true, $5)
		}
	;

/*****************************************************************************
 *
 *      DROP AGGREGATE
 *
 *****************************************************************************/

RemoveAggrStmt:
	DROP AGGREGATE AggregateWithArgTypesList OptDropBehavior
		{
			$$ = &ast.DropStmt{
				RemoveType: int(ast.OBJECT_AGGREGATE),
				Objects:    tableNamesFromNameList($3),
				Behavior:   int($4),
				Missing_ok: false,
			}
		}
	| DROP AGGREGATE IF_P EXISTS AggregateWithArgTypesList OptDropBehavior
		{
			$$ = &ast.DropStmt{
				RemoveType: int(ast.OBJECT_AGGREGATE),
				Objects:    tableNamesFromNameList($5),
				Behavior:   int($6),
				Missing_ok: true,
			}
		}
	;

/*****************************************************************************
 *
 *      DROP OPERATOR
 *
 *****************************************************************************/

RemoveOperStmt:
	DROP OPERATOR OperatorWithArgTypesList OptDropBehavior
		{
			$$ = &ast.DropStmt{
				RemoveType: int(ast.OBJECT_OPERATOR),
				Objects:    tableNamesFromNameList($3),
				Behavior:   int($4),
				Missing_ok: false,
			}
		}
	| DROP OPERATOR IF_P EXISTS OperatorWithArgTypesList OptDropBehavior
		{
			$$ = &ast.DropStmt{
				RemoveType: int(ast.OBJECT_OPERATOR),
				Objects:    tableNamesFromNameList($5),
				Behavior:   int($6),
				Missing_ok: true,
			}
		}
	;

/*****************************************************************************
 *
 *      function_with_argtypes and supporting rules
 *
 *****************************************************************************/

FunctionWithArgTypesList:
	FunctionWithArgTypes
		{ $$ = makeList($1) }
	| FunctionWithArgTypesList ',' FunctionWithArgTypes
		{ $$ = appendList($1, $3) }
	;

FunctionWithArgTypes:
	FunctionName FuncArgs
		{
			$$ = &ast.ObjectWithArgs{
				Objname: $1,
				Objargs: extractArgTypes($2),
			}
		}
	| TypeFuncNameKeyword
		{
			$$ = &ast.ObjectWithArgs{
				Objname:        &ast.List{Items: []ast.Node{&ast.String{Str: $1}}},
				ArgsUnspecified: true,
			}
		}
	| ColId
		{
			$$ = &ast.ObjectWithArgs{
				Objname:        &ast.List{Items: []ast.Node{&ast.String{Str: $1}}},
				ArgsUnspecified: true,
			}
		}
	| ColId Indirection
		{
			$$ = &ast.ObjectWithArgs{
				Objname:        checkFuncName(prependList(&ast.String{Str: $1}, $2)),
				ArgsUnspecified: true,
			}
		}
	;

FuncArgs:
	'(' FuncArgsList ')'
		{ $$ = $2 }
	| '(' ')'
		{ $$ = nil }
	;

FuncArgsList:
	FuncArg
		{ $$ = makeList($1) }
	| FuncArgsList ',' FuncArg
		{ $$ = appendList($1, $3) }
	;

/*****************************************************************************
 *
 *      aggregate_with_argtypes and supporting rules
 *
 *****************************************************************************/

AggregateWithArgTypes:
	FunctionName AggrArgs
		{
			$$ = &ast.ObjectWithArgs{
				Objname: $1,
				Objargs: extractAggrArgTypes($2),
			}
		}
	;

AggregateWithArgTypesList:
	AggregateWithArgTypes
		{ $$ = makeList($1) }
	| AggregateWithArgTypesList ',' AggregateWithArgTypes
		{ $$ = appendList($1, $3) }
	;

/*****************************************************************************
 *
 *      operator_with_argtypes and supporting rules
 *
 *****************************************************************************/

OperatorWithArgTypesList:
	OperatorWithArgTypes
		{ $$ = makeList($1) }
	| OperatorWithArgTypesList ',' OperatorWithArgTypes
		{ $$ = appendList($1, $3) }
	;

OperatorWithArgTypes:
	AnyOperator OperArgTypes
		{
			$$ = &ast.ObjectWithArgs{
				Objname: $1,
				Objargs: $2,
			}
		}
	;

OperArgTypes:
	'(' Typename ')'
		{
			yylex.Error("missing argument, use NONE to denote the missing argument of a unary operator")
			$$ = nil
		}
	| '(' Typename ',' Typename ')'
		{
			$$ = &ast.List{Items: []ast.Node{typeNameNode($2), typeNameNode($4)}}
		}
	| '(' NONE ',' Typename ')'
		{
			/* left unary */
			$$ = &ast.List{Items: []ast.Node{nil, typeNameNode($4)}}
		}
	| '(' Typename ',' NONE ')'
		{
			/* right unary */
			$$ = &ast.List{Items: []ast.Node{typeNameNode($2), nil}}
		}
	;

/*****************************************************************************
 *
 *      CREATE TRIGGER
 *
 *****************************************************************************/

CreateTriggerStmt:
	CREATE OptOrReplace TRIGGER Name TriggerActionTime TriggerEvents ON
	QualifiedName TriggerReferencing TriggerForSpec TriggerWhen
	EXECUTE FunctionOrPROCEDURE FunctionName '(' TriggerFuncArgs ')'
		{
			eventsInt := $6.Items[0].(*ast.Integer).Ival
			var columns *ast.List
			if len($6.Items) > 1 && $6.Items[1] != nil {
				columns = $6.Items[1].(*ast.List)
			}
			$$ = &ast.CreateTrigStmt{
				Replace:        $2,
				IsConstraint:   false,
				Trigname:        $4,
				Relation:        relationName(makeRangeVarFromAnyName($8)),
				Funcname:        $14,
				Args:            nodeList($16),
				Row:             $10,
				Timing:          int16($5),
				Events:          int16(eventsInt),
				Columns:         columns,
				WhenClause:      $11,
				TransitionRels:  $9,
				Deferrable:      false,
				Initdeferred:    false,
			}
		}
	| CREATE OptOrReplace CONSTRAINT TRIGGER Name AFTER TriggerEvents ON
	QualifiedName OptConstrFromTable ConstraintAttributeSpec
	FOR EACH ROW TriggerWhen
	EXECUTE FunctionOrPROCEDURE FunctionName '(' TriggerFuncArgs ')'
		{
			eventsInt := $7.Items[0].(*ast.Integer).Ival
			var columns *ast.List
			if len($7.Items) > 1 && $7.Items[1] != nil {
				columns = $7.Items[1].(*ast.List)
			}
			casBits := $11
			initdeferred := (casBits & int64(ast.CAS_INITIALLY_DEFERRED)) != 0
			deferrable := (casBits & int64(ast.CAS_DEFERRABLE)) != 0 || initdeferred
			var constrrel *ast.RangeVar
			if $10 != nil {
				constrrel = $10.(*ast.RangeVar)
			}
			$$ = &ast.CreateTrigStmt{
				Replace:        $2,
				IsConstraint:   true,
				Trigname:        $5,
				Relation:        relationName(makeRangeVarFromAnyName($9)),
				Funcname:        $18,
				Args:            nodeList($20),
				Row:             true,
				Timing:          int16(ast.TRIGGER_TYPE_AFTER),
				Events:          int16(eventsInt),
				Columns:         columns,
				WhenClause:      $15,
				Deferrable:      deferrable,
				Initdeferred:    initdeferred,
				Constrrel:       constrrel,
			}
		}
	;

TriggerActionTime:
	BEFORE    { $$ = int64(ast.TRIGGER_TYPE_BEFORE) }
	| AFTER   { $$ = int64(ast.TRIGGER_TYPE_AFTER) }
	| INSTEAD OF { $$ = int64(ast.TRIGGER_TYPE_INSTEAD) }
	;

TriggerEvents:
	TriggerOneEvent
		{ $$ = $1 }
	| TriggerEvents OR TriggerOneEvent
		{
			events1 := $1.Items[0].(*ast.Integer).Ival
			events2 := $3.Items[0].(*ast.Integer).Ival
			var columns1, columns2 *ast.List
			if len($1.Items) > 1 && $1.Items[1] != nil {
				columns1 = $1.Items[1].(*ast.List)
			}
			if len($3.Items) > 1 && $3.Items[1] != nil {
				columns2 = $3.Items[1].(*ast.List)
			}
			mergedCols := concatLists(columns1, columns2)
			var mergedColsNode ast.Node
			if mergedCols != nil {
				mergedColsNode = mergedCols
			}
			$$ = &ast.List{Items: []ast.Node{
				&ast.Integer{Ival: events1 | events2},
				mergedColsNode,
			}}
		}
	;

TriggerOneEvent:
	INSERT
		{
			$$ = &ast.List{Items: []ast.Node{
				&ast.Integer{Ival: int64(ast.TRIGGER_TYPE_INSERT)},
				nil,
			}}
		}
	| DELETE_P
		{
			$$ = &ast.List{Items: []ast.Node{
				&ast.Integer{Ival: int64(ast.TRIGGER_TYPE_DELETE)},
				nil,
			}}
		}
	| UPDATE
		{
			$$ = &ast.List{Items: []ast.Node{
				&ast.Integer{Ival: int64(ast.TRIGGER_TYPE_UPDATE)},
				nil,
			}}
		}
	| UPDATE OF ColumnList
		{
			$$ = &ast.List{Items: []ast.Node{
				&ast.Integer{Ival: int64(ast.TRIGGER_TYPE_UPDATE)},
				$3,
			}}
		}
	| TRUNCATE
		{
			$$ = &ast.List{Items: []ast.Node{
				&ast.Integer{Ival: int64(ast.TRIGGER_TYPE_TRUNCATE)},
				nil,
			}}
		}
	;

TriggerReferencing:
	REFERENCING TriggerTransitions
		{ $$ = $2 }
	| /* EMPTY */
		{ $$ = nil }
	;

TriggerTransitions:
	TriggerTransition
		{ $$ = makeList($1) }
	| TriggerTransitions TriggerTransition
		{ $$ = appendList($1, $2) }
	;

TriggerTransition:
	TransitionOldOrNew TransitionRowOrTable OptAs TransitionRelName
		{
			$$ = &ast.TriggerTransition{
				Name:    $4,
				IsNew:   $1,
				IsTable: $2,
			}
		}
	;

TransitionOldOrNew:
	NEW   { $$ = true }
	| OLD { $$ = false }
	;

TransitionRowOrTable:
	TABLE { $$ = true }
	| ROW { $$ = false }
	;

TransitionRelName:
	ColId { $$ = $1 }
	;

TriggerForSpec:
	FOR TriggerForOptEach TriggerForType
		{ $$ = $3 }
	| /* EMPTY */
		{ $$ = false }
	;

TriggerForOptEach:
	EACH
	| /* EMPTY */
	;

TriggerForType:
	ROW       { $$ = true }
	| STATEMENT { $$ = false }
	;

TriggerWhen:
	WHEN '(' AExpr ')'  { $$ = $3 }
	| /* EMPTY */         { $$ = nil }
	;

FunctionOrPROCEDURE:
	FUNCTION
	| PROCEDURE
	;

TriggerFuncArgs:
	TriggerFuncArg
		{ $$ = makeList($1) }
	| TriggerFuncArgs ',' TriggerFuncArg
		{ $$ = appendList($1, $3) }
	| /* EMPTY */
		{ $$ = nil }
	;

TriggerFuncArg:
	IConst
		{ $$ = &ast.String{Str: intToString($1)} }
	| FCONST
		{ $$ = &ast.String{Str: $1} }
	| SConst
		{ $$ = &ast.String{Str: $1} }
	| ColLabel
		{ $$ = &ast.String{Str: $1} }
	;

OptConstrFromTable:
	FROM QualifiedName
		{ $$ = makeRangeVar($2) }
	| /* EMPTY */
		{ $$ = nil }
	;

ConstraintAttributeSpec:
	/* EMPTY */
		{ $$ = 0 }
	| ConstraintAttributeSpec ConstraintAttributeElem
		{ $$ = $1 | $2 }
	;

ConstraintAttributeElem:
	NOT DEFERRABLE        { $$ = int64(ast.CAS_NOT_DEFERRABLE) }
	| DEFERRABLE          { $$ = int64(ast.CAS_DEFERRABLE) }
	| INITIALLY IMMEDIATE { $$ = int64(ast.CAS_INITIALLY_IMMEDIATE) }
	| INITIALLY DEFERRED  { $$ = int64(ast.CAS_INITIALLY_DEFERRED) }
	| NOT VALID           { $$ = int64(ast.CAS_NOT_VALID) }
	| NO INHERIT          { $$ = int64(ast.CAS_NO_INHERIT) }
	;

NoInherit:
	NO INHERIT            { $$ = true }
	| /* EMPTY */         { $$ = false }
	;

OptUniqueNullTreatment:
	NULLS_LA DISTINCT      { $$ = false }
	| NULLS_LA NOT DISTINCT { $$ = true }
	| /* EMPTY */           { $$ = false }
	;

/*****************************************************************************
 *
 *      CREATE EVENT TRIGGER / ALTER EVENT TRIGGER
 *
 *****************************************************************************/

CreateEventTrigStmt:
	CREATE EVENT TRIGGER Name ON ColLabel
	EXECUTE FunctionOrPROCEDURE FunctionName '(' ')'
		{
			$$ = &ast.CreateEventTrigStmt{
				Trigname:  $4,
				Eventname: $6,
				Funcname:  $9,
			}
		}
	| CREATE EVENT TRIGGER Name ON ColLabel
	WHEN EventTriggerWhenList
	EXECUTE FunctionOrPROCEDURE FunctionName '(' ')'
		{
			$$ = &ast.CreateEventTrigStmt{
				Trigname:   $4,
				Eventname:  $6,
				Whenclause: $8,
				Funcname:   $11,
			}
		}
	;

EventTriggerWhenList:
	EventTriggerWhenItem
		{ $$ = makeList($1) }
	| EventTriggerWhenList AND EventTriggerWhenItem
		{ $$ = appendList($1, $3) }
	;

EventTriggerWhenItem:
	ColId IN_P '(' EventTriggerValueList ')'
		{
			$$ = &ast.DefElem{
				Defname: $1,
				Arg:     $4,
			}
		}
	;

EventTriggerValueList:
	SConst
		{ $$ = makeList(&ast.String{Str: $1}) }
	| EventTriggerValueList ',' SConst
		{ $$ = appendList($1, &ast.String{Str: $3}) }
	;

AlterEventTrigStmt:
	ALTER EVENT TRIGGER Name EnableTrigger
		{
			$$ = &ast.AlterEventTrigStmt{
				Trigname:  $4,
				Tgenabled: byte($5),
			}
		}
	;

EnableTrigger:
	ENABLE_P             { $$ = int64(ast.TRIGGER_FIRES_ON_ORIGIN) }
	| ENABLE_P REPLICA   { $$ = int64(ast.TRIGGER_FIRES_ON_REPLICA) }
	| ENABLE_P ALWAYS    { $$ = int64(ast.TRIGGER_FIRES_ALWAYS) }
	| DISABLE_P          { $$ = int64(ast.TRIGGER_DISABLED) }
	;

/*****************************************************************************
 *
 *      CREATE RULE
 *
 *****************************************************************************/

RuleStmt:
	CREATE OptOrReplace RULE Name AS
	ON Event TO QualifiedName WhereClauseOpt
	DO OptInstead RuleActionList
		{
			$$ = &ast.RuleStmt{
				Replace:     $2,
				Relation:    relationName(makeRangeVarFromAnyName($9)),
				Rulename:    $4,
				WhereClause: $10,
				Event:       ast.CmdType($7),
				Instead:     $12,
				Actions:     $13,
			}
		}
	;

RuleActionList:
	NOTHING
		{ $$ = nil }
	| RuleActionStmt
		{ $$ = makeList($1) }
	| '(' RuleActionMulti ')'
		{ $$ = $2 }
	;

RuleActionMulti:
	RuleActionMulti ';' RuleActionStmtOrEmpty
		{
			if $3 != nil {
				$$ = appendList($1, $3)
			} else {
				$$ = $1
			}
		}
	| RuleActionStmtOrEmpty
		{
			if $1 != nil {
				$$ = makeList($1)
			} else {
				$$ = nil
			}
		}
	;

RuleActionStmt:
	SelectStmt  { $$ = $1 }
	| InsertStmt { $$ = $1 }
	| UpdateStmt { $$ = $1 }
	| DeleteStmt { $$ = $1 }
	| NotifyStmt { $$ = $1 }
	;

RuleActionStmtOrEmpty:
	RuleActionStmt  { $$ = $1 }
	| /* EMPTY */   { $$ = nil }
	;

Event:
	SELECT    { $$ = int64(ast.CMD_SELECT) }
	| UPDATE  { $$ = int64(ast.CMD_UPDATE) }
	| DELETE_P { $$ = int64(ast.CMD_DELETE) }
	| INSERT  { $$ = int64(ast.CMD_INSERT) }
	;

OptInstead:
	INSTEAD   { $$ = true }
	| ALSO    { $$ = false }
	| /* EMPTY */ { $$ = false }
	;

/*****************************************************************************
 *
 *      CREATE LANGUAGE
 *
 *****************************************************************************/

CreatePLangStmt:
	CREATE OptOrReplace OptTrusted OptProcedural LANGUAGE Name
		{
			/* Parameterless form - creates extension */
			$$ = &ast.CreatePLangStmt{
				Replace:   $2,
				Plname:    $6,
				Pltrusted: $3,
			}
		}
	| CREATE OptOrReplace OptTrusted OptProcedural LANGUAGE Name
	  HANDLER HandlerName OptInlineHandler OptValidator
		{
			$$ = &ast.CreatePLangStmt{
				Replace:     $2,
				Plname:      $6,
				Plhandler:   $8,
				Plinline:    $9,
				Plvalidator: $10,
				Pltrusted:   $3,
			}
		}
	;

OptTrusted:
	TRUSTED    { $$ = true }
	| /* EMPTY */ { $$ = false }
	;

HandlerName:
	Name
		{ $$ = &ast.List{Items: []ast.Node{&ast.String{Str: $1}}} }
	| Name Attrs
		{ $$ = prependList(&ast.String{Str: $1}, $2) }
	;

OptInlineHandler:
	INLINE_P HandlerName { $$ = $2 }
	| /* EMPTY */         { $$ = nil }
	;

OptValidator:
	VALIDATOR HandlerName  { $$ = $2 }
	| NO VALIDATOR          { $$ = nil }
	| /* EMPTY */           { $$ = nil }
	;

OptProcedural:
	PROCEDURAL
	| /* EMPTY */
	;


/*****************************************************************************
 *
 *      FOREIGN DATA WRAPPER statements
 *
 *****************************************************************************/

CreateFdwStmt:
	CREATE FOREIGN DATA_P WRAPPER Name OptFDWOptions CreateGenericOptions
		{
			$$ = &ast.CreateFdwStmt{
				Fdwname:     $5,
				FuncOptions: $6,
				Options:     $7,
			}
		}
	;

FDWOption:
	HANDLER HandlerName
		{
			$$ = makeDefElem("handler", $2)
		}
	| NO HANDLER
		{
			$$ = makeDefElem("handler", nil)
		}
	| VALIDATOR HandlerName
		{
			$$ = makeDefElem("validator", $2)
		}
	| NO VALIDATOR
		{
			$$ = makeDefElem("validator", nil)
		}
	;

FDWOptions:
	FDWOption
		{ $$ = makeList($1) }
	| FDWOptions FDWOption
		{ $$ = appendList($1, $2) }
	;

OptFDWOptions:
	FDWOptions   { $$ = $1 }
	| /* EMPTY */ { $$ = nil }
	;

/*****************************************************************************
 *
 *      ALTER FOREIGN DATA WRAPPER
 *
 *****************************************************************************/

AlterFdwStmt:
	ALTER FOREIGN DATA_P WRAPPER Name OptFDWOptions AlterGenericOptions
		{
			$$ = &ast.AlterFdwStmt{
				Fdwname:     $5,
				FuncOptions: $6,
				Options:     $7,
			}
		}
	| ALTER FOREIGN DATA_P WRAPPER Name FDWOptions
		{
			$$ = &ast.AlterFdwStmt{
				Fdwname:     $5,
				FuncOptions: $6,
			}
		}
	;

/*****************************************************************************
 *
 *      Generic OPTIONS infrastructure for FDW/SERVER/USER MAPPING
 *
 *****************************************************************************/

CreateGenericOptions:
	OPTIONS '(' GenericOptionList ')'
		{ $$ = $3 }
	| /* EMPTY */
		{ $$ = nil }
	;

GenericOptionList:
	GenericOptionElem
		{ $$ = makeList($1) }
	| GenericOptionList ',' GenericOptionElem
		{ $$ = appendList($1, $3) }
	;

AlterGenericOptions:
	OPTIONS '(' AlterGenericOptionList ')'
		{ $$ = $3 }
	;

AlterGenericOptionList:
	AlterGenericOptionElem
		{ $$ = makeList($1) }
	| AlterGenericOptionList ',' AlterGenericOptionElem
		{ $$ = appendList($1, $3) }
	;

AlterGenericOptionElem:
	GenericOptionElem
		{
			$$ = $1
		}
	| SET GenericOptionElem
		{
			n := $2.(*ast.DefElem)
			n.Defaction = int(ast.DEFELEM_SET)
			$$ = n
		}
	| ADD_P GenericOptionElem
		{
			n := $2.(*ast.DefElem)
			n.Defaction = int(ast.DEFELEM_ADD)
			$$ = n
		}
	| DROP GenericOptionName
		{
			$$ = &ast.DefElem{
				Defname:   $2,
				Defaction: int(ast.DEFELEM_DROP),
				Location:  -1,
			}
		}
	;

GenericOptionElem:
	GenericOptionName
		{
			$$ = &ast.DefElem{Defname: $1, Location: -1}
		}
	| GenericOptionName GenericOptionArg
		{
			$$ = &ast.DefElem{
				Defname:  $1,
				Arg:      $2,
				Location: -1,
			}
		}
	;

GenericOptionName:
	ColLabel  { $$ = $1 }
	;

GenericOptionArg:
	SConst  { $$ = &ast.String{Str: $1} }
	;

/*****************************************************************************
 *
 *      CREATE SERVER / ALTER SERVER
 *
 *****************************************************************************/

CreateForeignServerStmt:
	CREATE SERVER Name OptType OptForeignServerVersion
	  FOREIGN DATA_P WRAPPER Name CreateGenericOptions
		{
			$$ = &ast.CreateServerStmt{
				Name:       $3,
				ServerType: $4,
				Version:    $5,
				Wrapper:    $9,
				Options:    serverOptions($10),
			}
		}
	| CREATE SERVER IF_P NOT EXISTS Name OptType OptForeignServerVersion
	  FOREIGN DATA_P WRAPPER Name CreateGenericOptions
		{
			$$ = &ast.CreateServerStmt{
				Name:       $6,
				ServerType: $7,
				Version:    $8,
				Wrapper:    $12,
				Options:    serverOptions($13),
			}
		}
	;

OptType:
	TYPE_P SConst   { $$ = $2 }
	| /* EMPTY */   { $$ = "" }
	;

ForeignServerVersion:
	VERSION_P SConst   { $$ = $2 }
	| VERSION_P NULL_P { $$ = "" }
	;

OptForeignServerVersion:
	ForeignServerVersion { $$ = $1 }
	| /* EMPTY */          { $$ = "" }
	;

/*****************************************************************************
 *
 *      ALTER SERVER
 *
 *****************************************************************************/

AlterForeignServerStmt:
	ALTER SERVER Name ForeignServerVersion AlterGenericOptions
		{
			$$ = &ast.AlterServerStmt{
				Name:    $3,
				Version: $4,
				Options: serverOptions($5),
			}
		}
	| ALTER SERVER Name ForeignServerVersion
		{
			$$ = &ast.AlterServerStmt{
				Name:    $3,
				Version: $4,
			}
		}
	| ALTER SERVER Name AlterGenericOptions
		{
			$$ = &ast.AlterServerStmt{
				Name:    $3,
				Options: serverOptions($4),
			}
		}
	;

/*****************************************************************************
 *
 *      CREATE FOREIGN TABLE
 *
 *****************************************************************************/

CreateForeignTableStmt:
	CREATE FOREIGN TABLE QualifiedName
	  '(' OptTableElementList ')'
	  OptInherit SERVER Name CreateGenericOptions
		{
			rv := makeRangeVar($4)
			relationName(rv).Relpersistence = 'p'
			$$ = &ast.CreateForeignTableStmt{
				Base: ast.CreateStmt{
					Relation:     relationName(rv),
					TableElts:    $6,
					InhRelations: $8,
					IfNotExists:  false,
				},
				Servername: $10,
				Options:    $11,
			}
		}
	| CREATE FOREIGN TABLE IF_P NOT EXISTS QualifiedName
	  '(' OptTableElementList ')'
	  OptInherit SERVER Name CreateGenericOptions
		{
			rv := makeRangeVar($7)
			relationName(rv).Relpersistence = 'p'
			$$ = &ast.CreateForeignTableStmt{
				Base: ast.CreateStmt{
					Relation:     relationName(rv),
					TableElts:    $9,
					InhRelations: $11,
					IfNotExists:  true,
				},
				Servername: $13,
				Options:    $14,
			}
		}
	| CREATE FOREIGN TABLE QualifiedName
	  PARTITION OF QualifiedName OptTypedTableElementList ForValues
	  SERVER Name CreateGenericOptions
		{
			rv := makeRangeVar($4)
			inh := makeRangeVar($7)
			relationName(rv).Relpersistence = 'p'
			$$ = &ast.CreateForeignTableStmt{
				Base: ast.CreateStmt{
					Relation:     relationName(rv),
					InhRelations: makeList(inh),
					TableElts:    $8,
					Partbound:    $9,
					IfNotExists:  false,
				},
				Servername: $11,
				Options:    $12,
			}
		}
	| CREATE FOREIGN TABLE IF_P NOT EXISTS QualifiedName
	  PARTITION OF QualifiedName OptTypedTableElementList ForValues
	  SERVER Name CreateGenericOptions
		{
			rv := makeRangeVar($7)
			inh := makeRangeVar($10)
			relationName(rv).Relpersistence = 'p'
			$$ = &ast.CreateForeignTableStmt{
				Base: ast.CreateStmt{
					Relation:     relationName(rv),
					InhRelations: makeList(inh),
					TableElts:    $11,
					Partbound:    $12,
					IfNotExists:  true,
				},
				Servername: $14,
				Options:    $15,
			}
		}
	;

/*****************************************************************************
 *
 *      IMPORT FOREIGN SCHEMA
 *
 *****************************************************************************/

ImportForeignSchemaStmt:
	IMPORT_P FOREIGN SCHEMA Name ImportQualification
	  FROM SERVER Name INTO Name CreateGenericOptions
		{
			$$ = &ast.ImportForeignSchemaStmt{
				RemoteSchema: $4,
				Server:       $8,
				Into:         $10,
				Options:      defElemNames($11),
				OptionDefs:   $11,
			}
			if $5 != nil {
				n := $$.(*ast.ImportForeignSchemaStmt)
				n.ListType = $5.listType
				n.TableList = $5.tableList
			}
		}
	;

ImportQualificationType:
	LIMIT TO    { $$ = int64(ast.FDW_IMPORT_SCHEMA_LIMIT_TO) }
	| EXCEPT    { $$ = int64(ast.FDW_IMPORT_SCHEMA_EXCEPT) }
	;

ImportQualification:
	ImportQualificationType '(' RelationExprList ')'
		{
			$$ = &importQualification{
				listType:  ast.ImportForeignSchemaType($1),
				tableList: $3,
			}
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

/*****************************************************************************
 *
 *      CREATE / ALTER / DROP USER MAPPING
 *
 *****************************************************************************/

CreateUserMappingStmt:
	CREATE USER MAPPING FOR AuthIdent SERVER Name CreateGenericOptions
		{
			$$ = &ast.CreateUserMappingStmt{
				User:        $5.(*ast.RoleSpec),
				Servername:  $7,
				Options:     $8,
				IfNotExists: false,
			}
		}
	| CREATE USER MAPPING IF_P NOT EXISTS FOR AuthIdent SERVER Name CreateGenericOptions
		{
			$$ = &ast.CreateUserMappingStmt{
				User:        $8.(*ast.RoleSpec),
				Servername:  $10,
				Options:     $11,
				IfNotExists: true,
			}
		}
	;

AuthIdent:
	RoleSpec   { $$ = $1 }
	| USER
		{
			$$ = &ast.RoleSpec{
				Roletype: int(ast.ROLESPEC_CURRENT_USER),
			}
		}
	;

DropUserMappingStmt:
	DROP USER MAPPING FOR AuthIdent SERVER Name
		{
			$$ = &ast.DropUserMappingStmt{
				User:       $5.(*ast.RoleSpec),
				Servername: $7,
				MissingOk:  false,
			}
		}
	| DROP USER MAPPING IF_P EXISTS FOR AuthIdent SERVER Name
		{
			$$ = &ast.DropUserMappingStmt{
				User:       $7.(*ast.RoleSpec),
				Servername: $9,
				MissingOk:  true,
			}
		}
	;

AlterUserMappingStmt:
	ALTER USER MAPPING FOR AuthIdent SERVER Name AlterGenericOptions
		{
			$$ = &ast.AlterUserMappingStmt{
				User:       $5.(*ast.RoleSpec),
				Servername: $7,
				Options:    $8,
			}
		}
	;


/*****************************************************************************
 *
 *      CREATE TABLESPACE / DROP TABLESPACE / ALTER TABLESPACE
 *
 *****************************************************************************/

CreateTablespaceStmt:
	CREATE TABLESPACE Name OptTablespaceOwner LOCATION SConst OptRelOptions
		{
			$$ = &ast.CreateTablespaceStmt{
				Name:     syntaxIdent(yylex, $3, yyS[yypt-4].span),
				Owner:    roleSpecOrNil($4),
				Location: $6,
				Options:  $7,
			}
		}
	;

OptTablespaceOwner:
	OWNER RoleSpec { $$ = $2 }
	| /* EMPTY */  { $$ = nil }
	;

DropServerStmt:
	DROP SERVER NameList OptDropBehavior
		{
			$$ = &ast.DropServerStmt{
				Name: firstName($3),
			}
		}
	| DROP SERVER IF_P EXISTS NameList OptDropBehavior
		{
			$$ = &ast.DropServerStmt{
				Name:     firstName($5),
				IfExists: true,
			}
		}
	;

DropTablespaceStmt:
	DROP TABLESPACE Name
		{
			$$ = &ast.DropTablespaceStmt{
				Name: syntaxIdent(yylex, $3, yyS[yypt].span),
			}
		}
	| DROP TABLESPACE IF_P EXISTS Name
		{
			$$ = &ast.DropTablespaceStmt{
				Name:     syntaxIdent(yylex, $5, yyS[yypt].span),
				IfExists: true,
			}
		}
	;

AlterTablespaceStmt:
	ALTER TABLESPACE Name SET RelOptions
		{
			$$ = &ast.AlterTablespaceStmt{
				Name:    syntaxIdent(yylex, $3, yyS[yypt-2].span),
				Action:  "SET",
				Options: $5,
			}
		}
	| ALTER TABLESPACE Name RESET RelOptions
		{
			$$ = &ast.AlterTablespaceStmt{
				Name:    syntaxIdent(yylex, $3, yyS[yypt-2].span),
				Action:  "RESET",
				Options: $5,
			}
		}
	;

RelOptions:
	'(' RelOptionList ')'    { $$ = $2 }
	;

OptRelOptions:
	WITH RelOptions   { $$ = $2 }
	| /* EMPTY */     { $$ = nil }
	;

RelOptionList:
	RelOptionElem
		{ $$ = makeList($1) }
	| RelOptionList ',' RelOptionElem
		{ $$ = appendList($1, $3) }
	;

RelOptionElem:
	ColLabel '=' DefArg
		{
			$$ = makeDefElem($1, $3)
		}
	| ColLabel
		{
			$$ = makeDefElem($1, nil)
		}
	| ColLabel '.' ColLabel '=' DefArg
		{
			$$ = &ast.DefElem{
				Defnamespace: $1,
				Defname:      $3,
				Arg:          $5,
				Location:     -1,
			}
		}
	| ColLabel '.' ColLabel
		{
			$$ = &ast.DefElem{
				Defnamespace: $1,
				Defname:      $3,
				Location:     -1,
			}
		}
	;

/*****************************************************************************
 *
 *      CREATE EXTENSION / ALTER EXTENSION
 *
 *****************************************************************************/

CreateExtensionStmt:
	CREATE EXTENSION Name OptWithOptions CreateExtensionOptList
		{
			$$ = &ast.GenericStmt{
				Kind: "CREATE",
				Raw:  "EXTENSION " + $3,
			}
		}
	| CREATE EXTENSION IF_P NOT EXISTS Name OptWithOptions CreateExtensionOptList
		{
			$$ = &ast.GenericStmt{
				Kind: "CREATE",
				Raw:  "EXTENSION " + $6,
			}
		}
	;

CreateExtensionOptList:
	CreateExtensionOptList CreateExtensionOptItem
		{ $$ = appendList($1, $2) }
	| /* EMPTY */
		{ $$ = nil }
	;

CreateExtensionOptItem:
	SCHEMA Name
		{
			$$ = makeDefElem("schema", &ast.String{Str: $2})
		}
	| VERSION_P NonReservedWordOrSConst
		{
			$$ = makeDefElem("new_version", &ast.String{Str: $2})
		}
	| CASCADE
		{
			$$ = makeDefElem("cascade", &ast.Boolean{Boolval: true})
		}
	;

/*****************************************************************************
 *
 *      ALTER EXTENSION name UPDATE [ TO version ]
 *
 *****************************************************************************/

AlterExtensionStmt:
	ALTER EXTENSION Name UPDATE AlterExtensionOptList
		{
			$$ = &ast.AlterExtensionStmt{
				Extname: $3,
				Options: $5,
			}
		}
	;

AlterExtensionOptList:
	AlterExtensionOptList AlterExtensionOptItem
		{ $$ = appendList($1, $2) }
	| /* EMPTY */
		{ $$ = nil }
	;

AlterExtensionOptItem:
	TO NonReservedWordOrSConst
		{
			$$ = makeDefElem("new_version", &ast.String{Str: $2})
		}
	;

/*****************************************************************************
 *
 *      ALTER EXTENSION name ADD/DROP object-identifier
 *
 *****************************************************************************/

AlterExtensionContentsStmt:
	ALTER EXTENSION Name AddDrop ObjectTypeName Name
		{
			$$ = &ast.AlterExtensionContentsStmt{
				Extname: $3,
				Action:  int($4),
				ObjectType: ast.ObjectTypeNone,
				Object:  &ast.String{Str: $6},
			}
		}
	| ALTER EXTENSION Name AddDrop ObjectTypeAnyName AnyName
		{
			$$ = &ast.AlterExtensionContentsStmt{
				Extname: $3,
				Action:  int($4),
				ObjectType: ast.ObjectTypeNone,
				Object:  &ast.String{Str: $2},
			}
		}
	| ALTER EXTENSION Name AddDrop AGGREGATE AggregateWithArgTypes
		{
			$$ = &ast.AlterExtensionContentsStmt{
				Extname: $3,
				Action:  int($4),
				Objtype: "OBJECT",
				Object:  &ast.String{Str: $2},
			}
		}
	| ALTER EXTENSION Name AddDrop FUNCTION FunctionWithArgTypes
		{
			$$ = &ast.AlterExtensionContentsStmt{
				Extname: $3,
				Action:  int($4),
				Objtype: "OBJECT",
				Object:  &ast.String{Str: $2},
			}
		}
	| ALTER EXTENSION Name AddDrop PROCEDURE FunctionWithArgTypes
		{
			$$ = &ast.AlterExtensionContentsStmt{
				Extname: $3,
				Action:  int($4),
				Objtype: "OBJECT",
				Object:  &ast.String{Str: $2},
			}
		}
	| ALTER EXTENSION Name AddDrop ROUTINE FunctionWithArgTypes
		{
			$$ = &ast.AlterExtensionContentsStmt{
				Extname: $3,
				Action:  int($4),
				Objtype: "OBJECT",
				Object:  &ast.String{Str: $2},
			}
		}
	| ALTER EXTENSION Name AddDrop OPERATOR OperatorWithArgTypes
		{
			$$ = &ast.AlterExtensionContentsStmt{
				Extname: $3,
				Action:  int($4),
				Objtype: "OBJECT",
				Object:  &ast.String{Str: $2},
			}
		}
	| ALTER EXTENSION Name AddDrop OPERATOR CLASS AnyName USING Name
		{
			$$ = &ast.AlterExtensionContentsStmt{
				Extname: $3,
				Action:  int($4),
				Objtype: "OBJECT",
				Object:  prependList(&ast.String{Str: $9}, $7),
			}
		}
	| ALTER EXTENSION Name AddDrop OPERATOR FAMILY AnyName USING Name
		{
			$$ = &ast.AlterExtensionContentsStmt{
				Extname: $3,
				Action:  int($4),
				Objtype: "OBJECT",
				Object:  prependList(&ast.String{Str: $9}, $7),
			}
		}
	| ALTER EXTENSION Name AddDrop DOMAIN_P Typename
		{
			$$ = &ast.AlterExtensionContentsStmt{
				Extname: $3,
				Action:  int($4),
				Objtype: "OBJECT",
				Object:  &ast.String{Str: $2},
			}
		}
	| ALTER EXTENSION Name AddDrop TYPE_P Typename
		{
			$$ = &ast.AlterExtensionContentsStmt{
				Extname: $3,
				Action:  int($4),
				Objtype: "OBJECT",
				Object:  &ast.String{Str: $2},
			}
		}
	;

/*****************************************************************************
 *
 *      CREATE ACCESS METHOD
 *
 *****************************************************************************/

CreateAmStmt:
	CREATE ACCESS METHOD Name TYPE_P AMType HANDLER HandlerName
		{
			$$ = &ast.CreateAmStmt{
				Amname:      $4,
				Amtype:      byte($6),
				HandlerName: $8,
			}
		}
	;

AMType:
	INDEX     { $$ = ast.AMTYPE_INDEX }
	| TABLE   { $$ = ast.AMTYPE_TABLE }
	;

/*****************************************************************************
 *
 *      CREATE POLICY / ALTER POLICY
 *
 *****************************************************************************/

CreatePolicyStmt:
	CREATE POLICY Name ON QualifiedName RowSecurityDefaultPermissive
		RowSecurityDefaultForCmd RowSecurityDefaultToRole
		RowSecurityOptExpr RowSecurityOptWithCheck
		{
			$$ = &ast.CreatePolicyStmt{
				PolicyName: $3,
				Table:      makeRangeVarFromAnyName($5),
				Permissive: $6,
				CmdName:    $7,
				Roles:      $8,
				Qual:       $9,
				WithCheck:  $10,
			}
		}
	;

AlterPolicyStmt:
	ALTER POLICY Name ON QualifiedName RowSecurityOptToRole
		RowSecurityOptExpr RowSecurityOptWithCheck
		{
			$$ = &ast.AlterPolicyStmt{
				PolicyName: $3,
				Table:      makeRangeVarFromAnyName($5),
				Roles:      $6,
				Qual:       $7,
				WithCheck:  $8,
			}
		}
	;

RowSecurityOptExpr:
	USING '(' AExpr ')'    { $$ = $3 }
	| /* EMPTY */            { $$ = nil }
	;

RowSecurityOptWithCheck:
	WITH CHECK '(' AExpr ')'    { $$ = $4 }
	| /* EMPTY */                 { $$ = nil }
	;

RowSecurityDefaultToRole:
	TO RoleList    { $$ = $2 }
	| /* EMPTY */
		{
			/* Default is PUBLIC */
			$$ = makeList(&ast.RoleSpec{
				Roletype: int(ast.ROLESPEC_PUBLIC),
				Location: -1,
			})
		}
	;

RowSecurityOptToRole:
	TO RoleList    { $$ = $2 }
	| /* EMPTY */   { $$ = nil }
	;

RowSecurityDefaultPermissive:
	AS IDENT
		{
			if $2 == "permissive" {
				$$ = true
			} else if $2 == "restrictive" {
				$$ = false
			} else {
				yylex.Error("only PERMISSIVE or RESTRICTIVE policies are supported")
				$$ = true
			}
		}
	| /* EMPTY */    { $$ = true }
	;

RowSecurityDefaultForCmd:
	FOR RowSecurityCmd    { $$ = $2 }
	| /* EMPTY */           { $$ = "all" }
	;

RowSecurityCmd:
	ALL          { $$ = "all" }
	| SELECT     { $$ = "select" }
	| INSERT     { $$ = "insert" }
	| UPDATE     { $$ = "update" }
	| DELETE_P   { $$ = "delete" }
	;

/*****************************************************************************
 *
 *      CREATE PUBLICATION / ALTER PUBLICATION
 *
 *****************************************************************************/

CreatePublicationStmt:
	CREATE PUBLICATION Name OptDefinition
		{
			$$ = &ast.CreatePublicationStmt{
				Pubname: $3,
				Options: $4,
			}
		}
	| CREATE PUBLICATION Name FOR ALL TABLES OptDefinition
		{
			$$ = &ast.CreatePublicationStmt{
				Pubname:      $3,
				Options:      $7,
				ForAllTables: true,
			}
		}
	| CREATE PUBLICATION Name FOR PubObjList OptDefinition
		{
			$$ = &ast.CreatePublicationStmt{
				Pubname:    $3,
				Options:    $6,
				Pubobjects: $5,
			}
		}
	;

AlterPublicationStmt:
	ALTER PUBLICATION Name SET Definition
		{
			$$ = &ast.AlterPublicationStmt{
				Pubname: $3,
				Options: $5,
			}
		}
	| ALTER PUBLICATION Name ADD_P PubObjList
		{
			$$ = &ast.AlterPublicationStmt{
				Pubname:    $3,
				Pubobjects: $5,
				Action:     ast.DEFELEM_ADD,
			}
		}
	| ALTER PUBLICATION Name SET PubObjList
		{
			$$ = &ast.AlterPublicationStmt{
				Pubname:    $3,
				Pubobjects: $5,
				Action:     ast.DEFELEM_SET,
			}
		}
	| ALTER PUBLICATION Name DROP PubObjList
		{
			$$ = &ast.AlterPublicationStmt{
				Pubname:    $3,
				Pubobjects: $5,
				Action:     ast.DEFELEM_DROP,
			}
		}
	;

OptDefinition:
	WITH Definition    { $$ = $2 }
	| /* EMPTY */      { $$ = nil }
	;

PublicationObjSpec:
	TABLE TableName OptColumnList OptWhereClause
		{
			var cols *ast.List
			if $3 != nil {
				cols = $3
			}
			pt := &ast.PublicationTable{
				Relation:    relationName($2),
				Columns:     cols,
			}
			if $4 != nil {
				pt.WhereClause = $4.Items[0]
			}
			$$ = &ast.PublicationObjSpec{
				Pubobjtype: ast.PUBLICATIONOBJ_TABLE,
				Pubtable:   pt,
			}
		}
	| TABLES IN_P SCHEMA ColId
		{
			$$ = &ast.PublicationObjSpec{
				Pubobjtype: ast.PUBLICATIONOBJ_TABLES_IN_SCHEMA,
				Name:       $4,
			}
		}
	| TABLES IN_P SCHEMA CURRENT_SCHEMA
		{
			$$ = &ast.PublicationObjSpec{
				Pubobjtype: ast.PUBLICATIONOBJ_TABLES_IN_CUR_SCHEMA,
			}
		}
	| TableName OptColumnList OptWhereClause
		{
			pt := &ast.PublicationTable{
				Relation: relationName($1),
				Columns:  $2,
			}
			if $3 != nil {
				pt.WhereClause = $3.Items[0]
			}
			$$ = &ast.PublicationObjSpec{
				Pubobjtype: ast.PUBLICATIONOBJ_CONTINUATION,
				Pubtable:   pt,
			}
		}
	| CURRENT_SCHEMA
		{
			$$ = &ast.PublicationObjSpec{
				Pubobjtype: ast.PUBLICATIONOBJ_CONTINUATION,
			}
		}
	;

OptWhereClause:
	WHERE '(' AExpr ')'   { $$ = makeList($3) }
	| /* EMPTY */           { $$ = nil }
	;

PubObjList:
	PublicationObjSpec
		{ $$ = makeList($1) }
	| PubObjList ',' PublicationObjSpec
		{ $$ = appendList($1, $3) }
	;

/*****************************************************************************
 *
 *      CREATE SUBSCRIPTION / ALTER SUBSCRIPTION / DROP SUBSCRIPTION
 *
 *****************************************************************************/

CreateSubscriptionStmt:
	CREATE SUBSCRIPTION Name CONNECTION SConst PUBLICATION NameList OptDefinition
		{
			$$ = &ast.CreateSubscriptionStmt{
				Subname:     $3,
				Conninfo:    $5,
				Publication: $7,
				Options:     $8,
			}
		}
	;

AlterSubscriptionStmt:
	ALTER SUBSCRIPTION Name SET Definition
		{
			$$ = &ast.AlterSubscriptionStmt{
				Kind:    ast.ALTER_SUBSCRIPTION_OPTIONS,
				Subname: $3,
				Options: $5,
			}
		}
	| ALTER SUBSCRIPTION Name CONNECTION SConst
		{
			$$ = &ast.AlterSubscriptionStmt{
				Kind:     ast.ALTER_SUBSCRIPTION_CONNECTION,
				Subname:  $3,
				Conninfo: $5,
			}
		}
	| ALTER SUBSCRIPTION Name REFRESH PUBLICATION OptDefinition
		{
			$$ = &ast.AlterSubscriptionStmt{
				Kind:    ast.ALTER_SUBSCRIPTION_REFRESH,
				Subname: $3,
				Options: $6,
			}
		}
	| ALTER SUBSCRIPTION Name ADD_P PUBLICATION NameList OptDefinition
		{
			$$ = &ast.AlterSubscriptionStmt{
				Kind:        ast.ALTER_SUBSCRIPTION_ADD_PUBLICATION,
				Subname:     $3,
				Publication: $6,
				Options:     $7,
			}
		}
	| ALTER SUBSCRIPTION Name DROP PUBLICATION NameList OptDefinition
		{
			$$ = &ast.AlterSubscriptionStmt{
				Kind:        ast.ALTER_SUBSCRIPTION_DROP_PUBLICATION,
				Subname:     $3,
				Publication: $6,
				Options:     $7,
			}
		}
	| ALTER SUBSCRIPTION Name SET PUBLICATION NameList OptDefinition
		{
			$$ = &ast.AlterSubscriptionStmt{
				Kind:        ast.ALTER_SUBSCRIPTION_SET_PUBLICATION,
				Subname:     $3,
				Publication: $6,
				Options:     $7,
			}
		}
	| ALTER SUBSCRIPTION Name ENABLE_P
		{
			$$ = &ast.AlterSubscriptionStmt{
				Kind:    ast.ALTER_SUBSCRIPTION_ENABLED,
				Subname: $3,
				Options: makeList(makeDefElem("enabled", &ast.Boolean{Boolval: true})),
			}
		}
	| ALTER SUBSCRIPTION Name DISABLE_P
		{
			$$ = &ast.AlterSubscriptionStmt{
				Kind:    ast.ALTER_SUBSCRIPTION_ENABLED,
				Subname: $3,
				Options: makeList(makeDefElem("enabled", &ast.Boolean{Boolval: false})),
			}
		}
	| ALTER SUBSCRIPTION Name SKIP Definition
		{
			$$ = &ast.AlterSubscriptionStmt{
				Kind:    ast.ALTER_SUBSCRIPTION_SKIP,
				Subname: $3,
				Options: $5,
			}
		}
	;

DropSubscriptionStmt:
	DROP SUBSCRIPTION Name OptDropBehavior
		{
			$$ = &ast.DropSubscriptionStmt{
				Subname:   $3,
				MissingOk: false,
				Behavior:  ast.DropBehavior($4),
			}
		}
	| DROP SUBSCRIPTION IF_P EXISTS Name OptDropBehavior
		{
			$$ = &ast.DropSubscriptionStmt{
				Subname:   $5,
				MissingOk: true,
				Behavior:  ast.DropBehavior($6),
			}
		}
	;


/*****************************************************************************
 *
 * ALTER ... DEPENDS ON EXTENSION
 *
 *****************************************************************************/

AlterObjectDependsStmt:
	ALTER FUNCTION FunctionWithArgTypes OptNo DEPENDS ON EXTENSION Name
		{
			$$ = &ast.AlterObjectDependsStmt{
				ObjectType: ast.OBJECT_FUNCTION,
				Object:     $3,
				Extname:    &ast.String{Str: $8},
				Remove:     $4,
			}
		}
	| ALTER PROCEDURE FunctionWithArgTypes OptNo DEPENDS ON EXTENSION Name
		{
			$$ = &ast.AlterObjectDependsStmt{
				ObjectType: ast.OBJECT_PROCEDURE,
				Object:     $3,
				Extname:    &ast.String{Str: $8},
				Remove:     $4,
			}
		}
	| ALTER ROUTINE FunctionWithArgTypes OptNo DEPENDS ON EXTENSION Name
		{
			$$ = &ast.AlterObjectDependsStmt{
				ObjectType: ast.OBJECT_ROUTINE,
				Object:     $3,
				Extname:    &ast.String{Str: $8},
				Remove:     $4,
			}
		}
	| ALTER TRIGGER Name ON QualifiedName OptNo DEPENDS ON EXTENSION Name
		{
			$$ = &ast.AlterObjectDependsStmt{
				ObjectType: ast.OBJECT_TRIGGER,
				Relation:   relationName(makeRangeVarFromAnyName($5)),
				Object:     makeList(&ast.String{Str: $3}),
				Extname:    &ast.String{Str: $10},
				Remove:     $6,
			}
		}
	| ALTER MATERIALIZED VIEW QualifiedName OptNo DEPENDS ON EXTENSION Name
		{
			$$ = &ast.AlterObjectDependsStmt{
				ObjectType: ast.OBJECT_MATVIEW,
				Relation:   relationName(makeRangeVarFromAnyName($4)),
				Extname:    &ast.String{Str: $9},
				Remove:     $5,
			}
		}
	| ALTER INDEX QualifiedName OptNo DEPENDS ON EXTENSION Name
		{
			$$ = &ast.AlterObjectDependsStmt{
				ObjectType: ast.OBJECT_INDEX,
				Relation:   relationName(makeRangeVarFromAnyName($3)),
				Extname:    &ast.String{Str: $8},
				Remove:     $4,
			}
		}
	;

OptNo:
	NO
		{ $$ = true }
	| /* EMPTY */
		{ $$ = false }
	;

/*****************************************************************************
 *
 * ALTER ... SET SCHEMA
 *
 *****************************************************************************/

AlterObjectSchemaStmt:
	ALTER AGGREGATE AggregateWithArgTypes SET SCHEMA Name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_AGGREGATE,
				Object:     $3,
				Newschema:  $6,
			}
		}
	| ALTER COLLATION AnyName SET SCHEMA Name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_COLLATION,
				Object:     $3,
				Newschema:  $6,
			}
		}
	| ALTER CONVERSION_P AnyName SET SCHEMA Name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_CONVERSION,
				Object:     $3,
				Newschema:  $6,
			}
		}
	| ALTER DOMAIN_P AnyName SET SCHEMA Name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_DOMAIN,
				Object:     $3,
				Newschema:  $6,
			}
		}
	| ALTER EXTENSION Name SET SCHEMA Name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_EXTENSION,
				Object:     &ast.String{Str: $3},
				Newschema:  $6,
			}
		}
	| ALTER FUNCTION FunctionWithArgTypes SET SCHEMA Name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_FUNCTION,
				Object:     $3,
				Newschema:  $6,
			}
		}
	| ALTER OPERATOR OperatorWithArgTypes SET SCHEMA Name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_OPERATOR,
				Object:     $3,
				Newschema:  $6,
			}
		}
	| ALTER OPERATOR CLASS AnyName USING Name SET SCHEMA Name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_OPCLASS,
				Object:     prependList(&ast.String{Str: $6}, $4),
				Newschema:  $9,
			}
		}
	| ALTER OPERATOR FAMILY AnyName USING Name SET SCHEMA Name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_OPFAMILY,
				Object:     prependList(&ast.String{Str: $6}, $4),
				Newschema:  $9,
			}
		}
	| ALTER PROCEDURE FunctionWithArgTypes SET SCHEMA Name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_PROCEDURE,
				Object:     $3,
				Newschema:  $6,
			}
		}
	| ALTER ROUTINE FunctionWithArgTypes SET SCHEMA Name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_ROUTINE,
				Object:     $3,
				Newschema:  $6,
			}
		}
	| ALTER TABLE TableName SET SCHEMA Name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_TABLE,
				Relation:   relationName($3),
				Newschema:  $6,
			}
		}
	| ALTER TABLE IF_P EXISTS TableName SET SCHEMA Name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_TABLE,
				Relation:   relationName($5),
				Newschema:  $8,
				MissingOk:  true,
			}
		}
	| ALTER STATISTICS AnyName SET SCHEMA Name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_STATISTIC_EXT,
				Object:     $3,
				Newschema:  $6,
			}
		}
	| ALTER TEXT_P SEARCH PARSER AnyName SET SCHEMA Name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_TSPARSER,
				Object:     $5,
				Newschema:  $8,
			}
		}
	| ALTER TEXT_P SEARCH DICTIONARY AnyName SET SCHEMA Name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_TSDICTIONARY,
				Object:     $5,
				Newschema:  $8,
			}
		}
	| ALTER TEXT_P SEARCH TEMPLATE AnyName SET SCHEMA Name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_TSTEMPLATE,
				Object:     $5,
				Newschema:  $8,
			}
		}
	| ALTER TEXT_P SEARCH CONFIGURATION AnyName SET SCHEMA Name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_TSCONFIGURATION,
				Object:     $5,
				Newschema:  $8,
			}
		}
	| ALTER SEQUENCE QualifiedName SET SCHEMA Name
		{
			rv := makeRangeVarFromAnyName($3)
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_SEQUENCE,
				Relation:   relationName(rv),
				Newschema:  $6,
			}
		}
	| ALTER SEQUENCE IF_P EXISTS QualifiedName SET SCHEMA Name
		{
			rv := makeRangeVarFromAnyName($5)
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_SEQUENCE,
				Relation:   relationName(rv),
				Newschema:  $8,
				MissingOk:  true,
			}
		}
	| ALTER VIEW QualifiedName SET SCHEMA Name
		{
			rv := makeRangeVarFromAnyName($3)
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_VIEW,
				Relation:   relationName(rv),
				Newschema:  $6,
			}
		}
	| ALTER VIEW IF_P EXISTS QualifiedName SET SCHEMA Name
		{
			rv := makeRangeVarFromAnyName($5)
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_VIEW,
				Relation:   relationName(rv),
				Newschema:  $8,
				MissingOk:  true,
			}
		}
	| ALTER MATERIALIZED VIEW QualifiedName SET SCHEMA Name
		{
			rv := makeRangeVarFromAnyName($4)
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_MATVIEW,
				Relation:   relationName(rv),
				Newschema:  $7,
			}
		}
	| ALTER MATERIALIZED VIEW IF_P EXISTS QualifiedName SET SCHEMA Name
		{
			rv := makeRangeVarFromAnyName($6)
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_MATVIEW,
				Relation:   relationName(rv),
				Newschema:  $9,
				MissingOk:  true,
			}
		}
	| ALTER FOREIGN TABLE TableName SET SCHEMA Name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_FOREIGN_TABLE,
				Relation:   relationName($4),
				Newschema:  $7,
			}
		}
	| ALTER FOREIGN TABLE IF_P EXISTS TableName SET SCHEMA Name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_FOREIGN_TABLE,
				Relation:   relationName($6),
				Newschema:  $9,
				MissingOk:  true,
			}
		}
	| ALTER TYPE_P AnyName SET SCHEMA Name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_TYPE,
				Object:     $3,
				Newschema:  $6,
			}
		}
	;

/*****************************************************************************
 *
 * ALTER OPERATOR ... SET (...)
 *
 *****************************************************************************/

AlterOperatorStmt:
	ALTER OPERATOR OperatorWithArgTypes SET '(' OperatorDefList ')'
		{
			$$ = &ast.AlterOperatorStmt{
				Opername: $3.(*ast.ObjectWithArgs),
				Options:  $6,
			}
		}
	;

OperatorDefList:
	OperatorDefElem
		{ $$ = makeList($1) }
	| OperatorDefList ',' OperatorDefElem
		{ $$ = appendList($1, $3) }
	;

OperatorDefElem:
	ColLabel '=' NONE
		{
			$$ = &ast.DefElem{
				Defname:  $1,
				Location: -1,
			}
		}
	| ColLabel '=' OperatorDefArg
		{
			$$ = &ast.DefElem{
				Defname:  $1,
				Arg:      $3,
				Location: -1,
			}
		}
	| ColLabel
		{
			$$ = &ast.DefElem{
				Defname:  $1,
				Location: -1,
			}
		}
	;

OperatorDefArg:
	FuncType
		{ $$ = typeNameNode($1) }
	| ReservedKeyword
		{ $$ = &ast.String{Str: $1} }
	| QualAllOp
		{ $$ = $1 }
	| NumericOnly
		{ $$ = $1 }
	| SConst
		{ $$ = &ast.String{Str: $1} }
	;

/*****************************************************************************
 *
 * ALTER TYPE ... SET (...)
 *
 *****************************************************************************/

AlterTypeStmt:
	ALTER TYPE_P AnyName SET '(' OperatorDefList ')'
		{
			$$ = &ast.AlterTypeStmt{
				TypeName: tableNameFromList($3),
				Options:  $6,
			}
		}
	;

/*****************************************************************************
 *
 * ALTER ... OWNER TO
 *
 *****************************************************************************/

AlterOwnerStmt:
	ALTER AGGREGATE AggregateWithArgTypes OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_AGGREGATE,
				Object:     $3,
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER COLLATION AnyName OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_COLLATION,
				Object:     $3,
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER CONVERSION_P AnyName OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_CONVERSION,
				Object:     $3,
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER DATABASE Name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_DATABASE,
				Object:     &ast.String{Str: $3},
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER DOMAIN_P AnyName OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_DOMAIN,
				Object:     $3,
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER FUNCTION FunctionWithArgTypes OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_FUNCTION,
				Object:     $3,
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER OptProcedural LANGUAGE Name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_LANGUAGE,
				Object:     &ast.String{Str: $4},
				Newowner:   $7.(*ast.RoleSpec),
			}
		}
	| ALTER LARGE_P OBJECT_P NumericOnly OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_LARGEOBJECT,
				Object:     $4,
				Newowner:   $7.(*ast.RoleSpec),
			}
		}
	| ALTER OPERATOR OperatorWithArgTypes OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_OPERATOR,
				Object:     $3,
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER OPERATOR CLASS AnyName USING Name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_OPCLASS,
				Object:     prependList(&ast.String{Str: $6}, $4),
				Newowner:   $9.(*ast.RoleSpec),
			}
		}
	| ALTER OPERATOR FAMILY AnyName USING Name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_OPFAMILY,
				Object:     prependList(&ast.String{Str: $6}, $4),
				Newowner:   $9.(*ast.RoleSpec),
			}
		}
	| ALTER PROCEDURE FunctionWithArgTypes OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_PROCEDURE,
				Object:     $3,
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER ROUTINE FunctionWithArgTypes OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_ROUTINE,
				Object:     $3,
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER SCHEMA Name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_SCHEMA,
				Object:     &ast.String{Str: $3},
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER TYPE_P AnyName OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_TYPE,
				Object:     $3,
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER TABLESPACE Name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_TABLESPACE,
				Object:     &ast.String{Str: $3},
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER STATISTICS AnyName OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_STATISTIC_EXT,
				Object:     $3,
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER TEXT_P SEARCH DICTIONARY AnyName OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_TSDICTIONARY,
				Object:     $5,
				Newowner:   $8.(*ast.RoleSpec),
			}
		}
	| ALTER TEXT_P SEARCH CONFIGURATION AnyName OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_TSCONFIGURATION,
				Object:     $5,
				Newowner:   $8.(*ast.RoleSpec),
			}
		}
	| ALTER FOREIGN DATA_P WRAPPER Name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_FDW,
				Object:     &ast.String{Str: $5},
				Newowner:   $8.(*ast.RoleSpec),
			}
		}
	| ALTER SERVER Name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_FOREIGN_SERVER,
				Object:     &ast.String{Str: $3},
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER EVENT TRIGGER Name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_EVENT_TRIGGER,
				Object:     &ast.String{Str: $4},
				Newowner:   $7.(*ast.RoleSpec),
			}
		}
	| ALTER PUBLICATION Name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_PUBLICATION,
				Object:     &ast.String{Str: $3},
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER SUBSCRIPTION Name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_SUBSCRIPTION,
				Object:     &ast.String{Str: $3},
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	;

/*****************************************************************************
 *
 * ALTER DEFAULT PRIVILEGES
 *
 *****************************************************************************/

AlterDefaultPrivilegesStmt:
	ALTER DEFAULT PRIVILEGES DefACLOptionList DefACLAction
		{
			$$ = &ast.AlterDefaultPrivilegesStmt{
				Options: $4,
				Action:  $5.(*ast.GrantStmt),
			}
		}
	;

DefACLOptionList:
	DefACLOptionList DefACLOption
		{ $$ = appendList($1, $2) }
	| /* EMPTY */
		{ $$ = nil }
	;

DefACLOption:
	IN_P SCHEMA NameList
		{
			$$ = &ast.DefElem{
				Defname: "schemas",
				Arg:     $3,
				Location: -1,
			}
		}
	| FOR ROLE RoleList
		{
			$$ = &ast.DefElem{
				Defname: "roles",
				Arg:     $3,
				Location: -1,
			}
		}
	| FOR USER RoleList
		{
			$$ = &ast.DefElem{
				Defname: "roles",
				Arg:     $3,
				Location: -1,
			}
		}
	;

DefACLAction:
	GRANT Privileges ON DefaclPrivilegeTarget TO GranteeList OptGrantGrantOption OptGrantedBy
		{
			$$ = &ast.GrantStmt{
				
				Privs: privElems($2),
				ObjectType:     ast.ObjectTypeNone,
				Users: userSpecs($6),
				GrantOption: $7,
				Grantor: roleSpecOrNil($8),
			}
		}
	| REVOKE Privileges ON DefaclPrivilegeTarget FROM GranteeList OptGrantedBy OptDropBehavior
		{
			$$ = &ast.GrantStmt{
				IsGrant:    false,
				Privs: privElems($2),
				ObjectType:    ast.ObjectTypeNone,
				Users: userSpecs($6),
				Grantor: roleSpecOrNil($7),
				Behavior:   ast.DropBehavior($8),
			}
		}
	| REVOKE GRANT OPTION FOR Privileges ON DefaclPrivilegeTarget FROM GranteeList OptGrantedBy OptDropBehavior
		{
			$$ = &ast.GrantStmt{
				IsGrant: false,
				GrantOption: true,
				Privs: privElems($5),
				ObjectType:     ast.ObjectTypeNone,
				Users: userSpecs($9),
				Grantor: roleSpecOrNil($10),
				Behavior:    ast.DropBehavior($11),
			}
		}
	;

DefaclPrivilegeTarget:
	TABLES
		{ $$ = int64(ast.OBJECT_TABLE) }
	| FUNCTIONS
		{ $$ = int64(ast.OBJECT_FUNCTION) }
	| ROUTINES
		{ $$ = int64(ast.OBJECT_FUNCTION) }
	| SEQUENCES
		{ $$ = int64(ast.OBJECT_SEQUENCE) }
	| TYPES_P
		{ $$ = int64(ast.OBJECT_TYPE) }
	| SCHEMAS
		{ $$ = int64(ast.OBJECT_SCHEMA) }
	;

/*****************************************************************************
 *
 * ALTER TEXT SEARCH DICTIONARY
 *
 *****************************************************************************/

AlterTSDictionaryStmt:
	ALTER TEXT_P SEARCH DICTIONARY AnyName Definition
		{
			$$ = &ast.AlterTSDictionaryStmt{
				Dictname: $5,
				Options:  $6,
			}
		}
	;

/*****************************************************************************
 *
 * ALTER TEXT SEARCH CONFIGURATION
 *
 *****************************************************************************/

AlterTSConfigurationStmt:
	ALTER TEXT_P SEARCH CONFIGURATION AnyName ADD_P MAPPING FOR NameList WITH AnyNameList
		{
			$$ = &ast.AlterTSConfigurationStmt{
				Kind:      ast.ALTER_TSCONFIG_ADD_MAPPING,
				Cfgname:   $5,
				Tokentype: $9,
				Dicts:     $11,
			}
		}
	| ALTER TEXT_P SEARCH CONFIGURATION AnyName ALTER MAPPING FOR NameList WITH AnyNameList
		{
			$$ = &ast.AlterTSConfigurationStmt{
				Kind:      ast.ALTER_TSCONFIG_ALTER_MAPPING_FOR_TOKEN,
				Cfgname:   $5,
				Tokentype: $9,
				Dicts:     $11,
				Override:  true,
			}
		}
	| ALTER TEXT_P SEARCH CONFIGURATION AnyName ALTER MAPPING REPLACE AnyName WITH AnyName
		{
			$$ = &ast.AlterTSConfigurationStmt{
				Kind:    ast.ALTER_TSCONFIG_REPLACE_DICT,
				Cfgname: $5,
				Dicts:   makeList2($9, $11),
				Replace: true,
			}
		}
	| ALTER TEXT_P SEARCH CONFIGURATION AnyName ALTER MAPPING FOR NameList REPLACE AnyName WITH AnyName
		{
			$$ = &ast.AlterTSConfigurationStmt{
				Kind:      ast.ALTER_TSCONFIG_REPLACE_DICT_FOR_TOKEN,
				Cfgname:   $5,
				Tokentype: $9,
				Dicts:     makeList2($11, $13),
				Replace:   true,
			}
		}
	| ALTER TEXT_P SEARCH CONFIGURATION AnyName DROP MAPPING FOR NameList
		{
			$$ = &ast.AlterTSConfigurationStmt{
				Kind:      ast.ALTER_TSCONFIG_DROP_MAPPING,
				Cfgname:   $5,
				Tokentype: $9,
			}
		}
	| ALTER TEXT_P SEARCH CONFIGURATION AnyName DROP MAPPING IF_P EXISTS FOR NameList
		{
			$$ = &ast.AlterTSConfigurationStmt{
				Kind:      ast.ALTER_TSCONFIG_DROP_MAPPING,
				Cfgname:   $5,
				Tokentype: $11,
				MissingOk: true,
			}
		}
	;

/*****************************************************************************
 *
 * CREATE STATISTICS
 *
 *****************************************************************************/

CreateStatsStmt:
	CREATE STATISTICS OptQualifiedName OptStatNameList ON StatsParams FROM TableRefs
		{
			$$ = &ast.CreateStatsStmt{
				Defnames:    $3,
				StatTypes:   $4,
				Exprs:       $6,
				Relations:   nodeList($8),
				IfNotExists: false,
			}
		}
	| CREATE STATISTICS IF_P NOT EXISTS AnyName OptStatNameList ON StatsParams FROM TableRefs
		{
			$$ = &ast.CreateStatsStmt{
				Defnames:    $6,
				StatTypes:   $7,
				Exprs:       $9,
				Relations:   nodeList($11),
				IfNotExists: true,
			}
		}
	;

OptQualifiedName:
	AnyName
		{ $$ = $1 }
	| /* EMPTY */
		{ $$ = nil }
	;

OptStatNameList:
	'(' NameList ')'
		{ $$ = $2 }
	| /* EMPTY */
		{ $$ = nil }
	;

StatsParams:
	StatsParam
		{ $$ = makeList($1) }
	| StatsParams ',' StatsParam
		{ $$ = appendList($1, $3) }
	;

StatsParam:
	ColId
		{
			$$ = &ast.StatsElem{
				Name: $1,
			}
		}
	| FuncExprWindowless
		{
			$$ = &ast.StatsElem{
				Expr: $1,
			}
		}
	| '(' AExpr ')'
		{
			$$ = &ast.StatsElem{
				Expr: $2,
			}
		}
	;

/*****************************************************************************
 *
 * ALTER STATISTICS
 *
 *****************************************************************************/

AlterStatsStmt:
	ALTER STATISTICS AnyName SET STATISTICS SetStatisticsValue
		{
			n := &ast.AlterStatsStmt{
				Defnames:  $3,
				MissingOk: false,
			}
			if iv, ok := $6.(*ast.Integer); ok {
				n.Stxstattarget = int(iv.Ival)
			}
			$$ = n
		}
	| ALTER STATISTICS IF_P EXISTS AnyName SET STATISTICS SetStatisticsValue
		{
			n := &ast.AlterStatsStmt{
				Defnames:  $5,
				MissingOk: true,
			}
			if iv, ok := $8.(*ast.Integer); ok {
				n.Stxstattarget = int(iv.Ival)
			}
			$$ = n
		}
	;

SetStatisticsValue:
	SignedIconst
		{ $$ = &ast.Integer{Ival: $1} }
	| DEFAULT
		{ $$ = nil }
	;

/*****************************************************************************
 *
 * CREATE OPERATOR CLASS
 *
 *****************************************************************************/

CreateOpClassStmt:
	CREATE OPERATOR CLASS AnyName OptDefault FOR TYPE_P Typename USING Name OptOpFamily AS OpClassItemList
		{
			$$ = &ast.CreateOpClassStmt{
				Opclassname:  $4,
				IsDefault:    $5,
				Datatype:     $8,
				Amname:       $10,
				Opfamilyname: $11,
				Items:        $13,
			}
		}
	;

OpClassItemList:
	OpClassItem
		{ $$ = makeList($1) }
	| OpClassItemList ',' OpClassItem
		{ $$ = appendList($1, $3) }
	;

OpClassItem:
	OPERATOR IConst AnyOperator OpClassPurpose OptRecheck
		{
			owa := &ast.ObjectWithArgs{
				Objname: $3,
			}
			$$ = &ast.CreateOpClassItem{
				Itemtype:    ast.OPCLASS_ITEM_OPERATOR,
				Name:        owa,
				Number:      int($2),
				OrderFamily: $4,
			}
		}
	| OPERATOR IConst OperatorWithArgTypes OpClassPurpose OptRecheck
		{
			$$ = &ast.CreateOpClassItem{
				Itemtype:    ast.OPCLASS_ITEM_OPERATOR,
				Name:        $3.(*ast.ObjectWithArgs),
				Number:      int($2),
				OrderFamily: $4,
			}
		}
	| FUNCTION IConst FunctionWithArgTypes
		{
			$$ = &ast.CreateOpClassItem{
				Itemtype: ast.OPCLASS_ITEM_FUNCTION,
				Name:     $3.(*ast.ObjectWithArgs),
				Number:   int($2),
			}
		}
	| FUNCTION IConst '(' TypeList ')' FunctionWithArgTypes
		{
			$$ = &ast.CreateOpClassItem{
				Itemtype:  ast.OPCLASS_ITEM_FUNCTION,
				Name:      $6.(*ast.ObjectWithArgs),
				Number:    int($2),
				ClassArgs: nodeList($4),
			}
		}
	| STORAGE Typename
		{
			$$ = &ast.CreateOpClassItem{
				Itemtype:   ast.OPCLASS_ITEM_STORAGETYPE,
				Storedtype: $2,
			}
		}
	;

OptDefault:
	DEFAULT
		{ $$ = true }
	| /* EMPTY */
		{ $$ = false }
	;

OptOpFamily:
	FAMILY AnyName
		{ $$ = $2 }
	| /* EMPTY */
		{ $$ = nil }
	;

OpClassPurpose:
	FOR SEARCH
		{ $$ = nil }
	| FOR ORDER BY AnyName
		{ $$ = $4 }
	| /* EMPTY */
		{ $$ = nil }
	;

OptRecheck:
	RECHECK
		{ $$ = true }
	| /* EMPTY */
		{ $$ = false }
	;

/*****************************************************************************
 *
 * CREATE OPERATOR FAMILY
 *
 *****************************************************************************/

CreateOpFamilyStmt:
	CREATE OPERATOR FAMILY AnyName USING Name
		{
			$$ = &ast.CreateOpFamilyStmt{
				Opfamilyname: $4,
				Amname:       $6,
			}
		}
	;

/*****************************************************************************
 *
 * ALTER OPERATOR FAMILY
 *
 *****************************************************************************/

AlterOpFamilyStmt:
	ALTER OPERATOR FAMILY AnyName USING Name ADD_P OpClassItemList
		{
			$$ = &ast.AlterOpFamilyStmt{
				Opfamilyname: $4,
				Amname:       $6,
				IsDrop:       false,
				Items:        $8,
			}
		}
	| ALTER OPERATOR FAMILY AnyName USING Name DROP OpClassDropList
		{
			$$ = &ast.AlterOpFamilyStmt{
				Opfamilyname: $4,
				Amname:       $6,
				IsDrop:       true,
				Items:        $8,
			}
		}
	;

OpClassDropList:
	OpClassDrop
		{ $$ = makeList($1) }
	| OpClassDropList ',' OpClassDrop
		{ $$ = appendList($1, $3) }
	;

OpClassDrop:
	OPERATOR IConst '(' TypeList ')'
		{
			$$ = &ast.CreateOpClassItem{
				Itemtype:  ast.OPCLASS_ITEM_OPERATOR,
				Number:    int($2),
				ClassArgs: nodeList($4),
			}
		}
	| FUNCTION IConst '(' TypeList ')'
		{
			$$ = &ast.CreateOpClassItem{
				Itemtype:  ast.OPCLASS_ITEM_FUNCTION,
				Number:    int($2),
				ClassArgs: nodeList($4),
			}
		}
	;

/*****************************************************************************
 *
 * DROP OPERATOR CLASS / FAMILY
 *
 *****************************************************************************/

DropOpClassStmt:
	DROP OPERATOR CLASS AnyName USING Name OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList(prependList(&ast.String{Str: $6}, $4)),
				RemoveType: int(ast.OBJECT_OPCLASS),
				Behavior:   int($7),
			}
		}
	| DROP OPERATOR CLASS IF_P EXISTS AnyName USING Name OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList(prependList(&ast.String{Str: $8}, $6)),
				RemoveType: int(ast.OBJECT_OPCLASS),
				Behavior:   int($9),
				Missing_ok: true,
			}
		}
	;

DropOpFamilyStmt:
	DROP OPERATOR FAMILY AnyName USING Name OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList(prependList(&ast.String{Str: $6}, $4)),
				RemoveType: int(ast.OBJECT_OPFAMILY),
				Behavior:   int($7),
			}
		}
	| DROP OPERATOR FAMILY IF_P EXISTS AnyName USING Name OptDropBehavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList(prependList(&ast.String{Str: $8}, $6)),
				RemoveType: int(ast.OBJECT_OPFAMILY),
				Behavior:   int($9),
				Missing_ok: true,
			}
		}
	;

/*****************************************************************************
 *
 * CREATE CAST / DROP CAST
 *
 *****************************************************************************/

CreateCastStmt:
	CREATE CAST '(' Typename AS Typename ')' WITH FUNCTION FunctionWithArgTypes CastContext
		{
			$$ = &ast.CreateCastStmt{
				Sourcetype: $4,
				Targettype: $6,
				Func:       $10.(*ast.ObjectWithArgs),
				Context:    ast.CoercionContext($11),
			}
		}
	| CREATE CAST '(' Typename AS Typename ')' WITHOUT FUNCTION CastContext
		{
			$$ = &ast.CreateCastStmt{
				Sourcetype: $4,
				Targettype: $6,
				Context:    ast.CoercionContext($10),
			}
		}
	| CREATE CAST '(' Typename AS Typename ')' WITH INOUT CastContext
		{
			$$ = &ast.CreateCastStmt{
				Sourcetype: $4,
				Targettype: $6,
				Context:    ast.CoercionContext($10),
				Inout:      true,
			}
		}
	;

CastContext:
	AS IMPLICIT_P
		{ $$ = int64(ast.COERCION_IMPLICIT) }
	| AS ASSIGNMENT
		{ $$ = int64(ast.COERCION_ASSIGNMENT) }
	| /* EMPTY */
		{ $$ = int64(ast.COERCION_EXPLICIT) }
	;

DropCastStmt:
	DROP CAST OptIfExists '(' Typename AS Typename ')' OptDropBehavior
		{
			$$ = &ast.DropStmt{
				RemoveType: int(ast.OBJECT_CAST),
				Objects:    tableNamesFromNameList(makeList2(typeNameNode($5), typeNameNode($7))),
				Behavior:   int($9),
				Missing_ok: $3,
			}
		}
	;

OptIfExists:
	IF_P EXISTS
		{ $$ = true }
	| /* EMPTY */
		{ $$ = false }
	;

/*****************************************************************************
 *
 * CREATE TRANSFORM / DROP TRANSFORM
 *
 *****************************************************************************/

CreateTransformStmt:
	CREATE OptOrReplace TRANSFORM FOR Typename LANGUAGE Name '(' TransformElementList ')'
		{
			items := $9.Items
			var fromsql *ast.ObjectWithArgs
			var tosql *ast.ObjectWithArgs
			if items[0] != nil {
				fromsql = items[0].(*ast.ObjectWithArgs)
			}
			if items[1] != nil {
				tosql = items[1].(*ast.ObjectWithArgs)
			}
			$$ = &ast.CreateTransformStmt{
				Replace:  $2,
				TypeName: newDataType($3, -1, -1),
				Lang:     $7,
				Fromsql:  fromsql,
				Tosql:    tosql,
			}
		}
	;

TransformElementList:
	FROM SQL_P WITH FUNCTION FunctionWithArgTypes ',' TO SQL_P WITH FUNCTION FunctionWithArgTypes
		{
			$$ = makeList2($5, $11)
		}
	| TO SQL_P WITH FUNCTION FunctionWithArgTypes ',' FROM SQL_P WITH FUNCTION FunctionWithArgTypes
		{
			$$ = makeList2($11, $5)
		}
	| FROM SQL_P WITH FUNCTION FunctionWithArgTypes
		{
			$$ = makeList2($5, nil)
		}
	| TO SQL_P WITH FUNCTION FunctionWithArgTypes
		{
			$$ = makeList2(nil, $5)
		}
	;

DropTransformStmt:
	DROP TRANSFORM OptIfExists FOR Typename LANGUAGE Name OptDropBehavior
		{
			$$ = &ast.DropStmt{
				RemoveType: int(ast.OBJECT_TRANSFORM),
				Objects:    tableNamesFromNameList(makeList2(typeNameNode($5), &ast.String{Str: $7})),
				Behavior:   int($8),
				Missing_ok: $3,
			}
		}
	;

/*****************************************************************************
 *
 * CREATE CONVERSION
 *
 *****************************************************************************/

CreateConversionStmt:
	CREATE OptDefault CONVERSION_P AnyName FOR SConst TO SConst FROM AnyName
		{
			$$ = &ast.CreateConversionStmt{
				ConversionName:  $4,
				ForEncodingName: $6,
				ToEncodingName:  $8,
				FuncName:        $10,
				Def:             $2,
			}
		}
	;

/*****************************************************************************
 *
 * DROP OWNED BY / REASSIGN OWNED BY
 *
 *****************************************************************************/

DropOwnedStmt:
	DROP OWNED BY RoleList OptDropBehavior
		{
			$$ = &ast.DropOwnedStmt{
				Roles:   roleIdentities($4),
				Cascade: ast.DropBehavior($5) == ast.DROP_CASCADE,
			}
		}
	;

ReassignOwnedStmt:
	REASSIGN OWNED BY RoleList TO RoleSpec
		{
			$$ = &ast.ReassignOwnedStmt{
				Roles:   roleIdentities($4),
				NewRole: roleIdentity($6),
			}
		}
	;

/*****************************************************************************
 *
 * CREATE TABLE AS / CREATE MATERIALIZED VIEW
 *
 *****************************************************************************/

CreateAsStmt:
	CREATE OptTemp TABLE CreateAsTarget AS SelectStmt OptWithData
		{
			into := $4.(*ast.IntoClause)
			into.Rel.Relpersistence = relpersistenceForTemp($2)
			$$ = &ast.CreateTableStmt{
				Temporary:  $2 != 0,
				Table:      relationName(into.Rel),
				Select:     $6.(ast.ResultSetNode),
			}
		}
	| CREATE OptTemp TABLE IF_P NOT EXISTS CreateAsTarget AS SelectStmt OptWithData
		{
			into := $7.(*ast.IntoClause)
			into.Rel.Relpersistence = relpersistenceForTemp($2)
			$$ = &ast.CreateTableStmt{
				Temporary:  $2 != 0,
				IfNotExists: true,
				Table:      relationName(into.Rel),
				Select:     $9.(ast.ResultSetNode),
			}
		}
	| CREATE OptTemp TABLE CreateAsTarget AS EXECUTE Name ExecuteParamClause OptWithData
		{
			into := $4.(*ast.IntoClause)
			into.Rel.Relpersistence = relpersistenceForTemp($2)
			$$ = &ast.CreateTableStmt{
				Temporary: $2 != 0,
				Table:     relationName(into.Rel),
				Execute:   $6,
			}
		}
	| CREATE OptTemp TABLE IF_P NOT EXISTS CreateAsTarget AS EXECUTE Name ExecuteParamClause OptWithData
		{
			into := $7.(*ast.IntoClause)
			into.Rel.Relpersistence = relpersistenceForTemp($2)
			$$ = &ast.CreateTableStmt{
				Temporary:  $2 != 0,
				IfNotExists: true,
				Table:      relationName(into.Rel),
				Execute:    $9,
			}
		}
	;

CreateAsTarget:
	QualifiedName OptColumnList OptAccessMethod OptWith OnCommitOption OptTablespace
		{
			rv := makeRangeVarFromAnyName($1)
			$$ = &ast.IntoClause{
				Rel:            relationName(rv),
				ColNames:       $2,
				AccessMethod:   $3,
				Options:        $4,
				OnCommit:       ast.OnCommitAction($5),
				TableSpaceName: $6,
			}
		}
	;

OptWithData:
	WITH DATA_P
		{ $$ = true }
	| WITH NO DATA_P
		{ $$ = false }
	| /* EMPTY */
		{ $$ = true }
	;

OptNoLog:
	UNLOGGED
		{ $$ = int64(ast.RELPERSISTENCE_UNLOGGED) }
	| /* EMPTY */
		{ $$ = int64(ast.RELPERSISTENCE_PERMANENT) }
	;

CreateMatViewStmt:
	CREATE OptNoLog MATERIALIZED VIEW CreateMVTarget AS SelectStmt OptWithData
		{
			into := $5.(*ast.IntoClause)
			into.Rel.Relpersistence = byte($2)
			into.SkipData = !$8
			$$ = &ast.CreateTableAsStmt{
				Query:       $7,
				Into:        into,
				Objtype:     "OBJECT",
				IfNotExists: false,
			}
		}
	| CREATE OptNoLog MATERIALIZED VIEW IF_P NOT EXISTS CreateMVTarget AS SelectStmt OptWithData
		{
			into := $8.(*ast.IntoClause)
			into.Rel.Relpersistence = byte($2)
			into.SkipData = !$11
			$$ = &ast.CreateTableAsStmt{
				Query:       $10,
				Into:        into,
				Objtype:     "OBJECT",
				IfNotExists: true,
			}
		}
	;

CreateMVTarget:
	QualifiedName OptColumnList OptAccessMethod OptRelOptions OptTablespace
		{
			rv := makeRangeVarFromAnyName($1)
			$$ = &ast.IntoClause{
				Rel:            relationName(rv),
				ColNames:       $2,
				AccessMethod:   $3,
				Options:        $4,
				TableSpaceName: $5,
			}
		}
	;

/*****************************************************************************
 *
 * REFRESH MATERIALIZED VIEW
 *
 *****************************************************************************/

RefreshMatViewStmt:
	REFRESH MATERIALIZED VIEW OptConcurrently QualifiedName OptWithData
		{
			$$ = &ast.RefreshMatViewStmt{
				Concurrently:  $4,
				Table:         tableNameFromList($5),
				NoData:        !$6,
				DataSpecified: !$6,
			}
		}
	;


TypeFunctionName:
	IDENT { $$ = $1 }
	| UnReservedKeyword { $$ = $1 }
	| TypeFuncNameKeyword { $$ = $1 }
	;

// Keyword categories
// These lists must match the keyword definitions in keywords.go

/*****************************************************************************
 *
 *      CREATE / ALTER / DROP RESOURCE GROUP
 *
 *****************************************************************************/

ResourceGroupStmt:
	CREATE RESOURCE GROUP_P Name ResourceGroupType ResourceGroupOptions
		{
			$$ = &ast.CreateResourceGroupStmt{
				Name:           $4,
				IsSystem:       $5,
				VCPU:           resourceGroupVCPU($6),
				ThreadPriority: resourceGroupThreadPriority($6),
				State:          resourceGroupState($6),
			}
		}
	| ALTER RESOURCE GROUP_P Name ResourceGroupOptions
		{
			$$ = &ast.AlterResourceGroupStmt{
				Name:           $4,
				VCPU:           resourceGroupVCPU($5),
				ThreadPriority: resourceGroupThreadPriority($5),
				State:          resourceGroupState($5),
			}
		}
	| DROP RESOURCE GROUP_P Name OptForce
		{
			$$ = &ast.DropResourceGroupStmt{
				Name:  $4,
				Force: $5,
			}
		}
	;

ResourceGroupType:
	TYPE_P '=' SYSTEM_P
		{
			$$ = true
		}
	| TYPE_P '=' USER
		{
			$$ = false
		}
	;

ResourceGroupOptions:
	/* EMPTY */
		{
			$$ = nil
		}
	| ResourceGroupOptionList
		{
			$$ = $1
		}
	;

ResourceGroupOptionList:
	ResourceGroupOption
		{
			$$ = makeList($1)
		}
	| ResourceGroupOptionList ResourceGroupOption
		{
			$$ = appendList($1, $2)
		}
	;

ResourceGroupOption:
	VCPU_P '=' IConst
		{
			$$ = makeDefElem("vcpu", ast.NewValueExpr($3))
		}
	| VCPU_P '=' IConst '-' IConst
		{
			$$ = makeDefElem("vcpu", &ast.List{Items: []ast.Node{
				ast.NewValueExpr($3),
				ast.NewValueExpr($5),
			}})
		}
	| THREAD_PRIORITY_P '=' IConst
		{
			$$ = makeDefElem("thread_priority", ast.NewValueExpr($3))
		}
	| ENABLE_P
		{
			$$ = makeDefElem("state", ast.NewValueExpr("enable"))
		}
	| DISABLE_P
		{
			$$ = makeDefElem("state", ast.NewValueExpr("disable"))
		}
	;

OptForce:
	FORCE
		{
			$$ = true
		}
	| /* EMPTY */
		{
			$$ = false
		}
	;

UnReservedKeyword:
	ABORT_P
	| ABSENT
	| RESOURCE
	| VCPU_P
	| THREAD_PRIORITY_P
	| ABSOLUTE_P
	| ACCESS
	| ACTION
	| ADD_P
	| ADMIN
	| AFTER
	| AGGREGATE
	| ALSO
	| ALTER
	| ALWAYS
	| ASENSITIVE
	| ASSERTION
	| ASSIGNMENT
	| AT
	| ATOMIC
	| ATTACH
	| ATTRIBUTE
	| BACKWARD
	| BEFORE
	| BEGIN_P
	| BREADTH
	| BY
	| CACHE
	| CALL
	| CALLED
	| CASCADE
	| CASCADED
	| CATALOG_P
	| CHAIN
	| CHARACTERISTICS
	| CHECKPOINT
	| CLASS
	| CLOSE
	| CLUSTER
	| COLUMNS
	| COMMENT
	| COMMENTS
	| COMMIT
	| COMMITTED
	| COMPRESSION
	| CONDITIONAL
	| CONFIGURATION
	| CONFLICT
	| CONNECTION
	| CONSTRAINTS
	| CONTENT_P
	| CONTINUE_P
	| CONVERSION_P
	| COPY
	| COST
	| CSV
	| CUBE
	| CURRENT_P
	| CURSOR
	| CYCLE
	| DATA_P
	| DATABASE
	| DAY_P
	| DEALLOCATE
	| DECLARE
	| DEFAULTS
	| DEFERRED
	| DEFINER
	| DELETE_P
	| DELIMITER
	| DELIMITERS
	| DEPENDS
	| DEPTH
	| DETACH
	| DICTIONARY
	| DISABLE_P
	| DISCARD
	| DOCUMENT_P
	| DOMAIN_P
	| DOUBLE_P
	| DROP
	| EACH
	| EMPTY_P
	| ENABLE_P
	| ENCODING
	| ENCRYPTED
	| ENUM_P
	| ERROR_P
	| ESCAPE
	| EVENT
	| EXCLUDE
	| EXCLUDING
	| EXCLUSIVE
	| EXECUTE
	| EXPLAIN
	| EXPRESSION
	| EXTENSION
	| EXTERNAL
	| FAMILY
	| FILTER
	| FINALIZE
	| FIRST_P
	| FOLLOWING
	| FORCE
	| FORMAT
	| FORWARD
	| FUNCTION
	| FUNCTIONS
	| GENERATED
	| GLOBAL
	| GRANTED
	| GROUPS
	| HANDLER
	| HEADER_P
	| HOLD
	| HOUR_P
	| IDENTITY_P
	| IF_P
	| IMMEDIATE
	| IMMUTABLE
	| IMPLICIT_P
	| IMPORT_P
	| INCLUDE
	| INCLUDING
	| INCREMENT
	| INDENT
	| INDEX
	| INDEXES
	| INHERIT
	| INHERITS
	| INLINE_P
	| INPUT_P
	| INSENSITIVE
	| INSERT
	| INSTEAD
	| INVOKER
	| ISOLATION
	| KEEP
	| KEY
	| KEYS
	| LABEL
	| LANGUAGE
	| LARGE_P
	| LAST_P
	| LEAKPROOF
	| LEVEL
	| LISTEN
	| LOAD
	| LOCAL
	| LOCATION
	| LOCK_P
	| LOCKED
	| LOGGED
	| MAPPING
	| MATCH
	| MATCHED
	| MATERIALIZED
	| MAXVALUE
	| MERGE
	| METHOD
	| MINUTE_P
	| MINVALUE
	| MODE
	| MONTH_P
	| MOVE
	| NAME_P
	| NAMES
	| NESTED
	| NEW
	| NEXT
	| NFC
	| NFD
	| NFKC
	| NFKD
	| NO
	| NORMALIZED
	| NOTHING
	| NOTIFY
	| NOWAIT
	| NULLS_P
	| OBJECT_P
	| OF
	| OFF
	| OIDS
	| OLD
	| OMIT
	| OPERATOR
	| OPTION
	| OPTIONS
	| ORDINALITY
	| OTHERS
	| OVER
	| OVERRIDING
	| OWNED
	| OWNER
	| PARALLEL
	| PARAMETER
	| PARSER
	| PARTIAL
	| PARTITION
	| PASSING
	| PASSWORD
	| PATH
	| PLAN
	| PLANS
	| POLICY
	| PRECEDING
	| PREPARE
	| PREPARED
	| PRESERVE
	| PRIOR
	| PRIVILEGES
	| PROCEDURAL
	| PROCEDURE
	| PROCEDURES
	| PROGRAM
	| PUBLICATION
	| QUOTE
	| QUOTES
	| RANGE
	| READ
	| REASSIGN
	| RECHECK
	| RECURSIVE
	| REF_P
	| REFERENCING
	| REFRESH
	| REINDEX
	| RELATIVE_P
	| RELEASE
	| RENAME
	| REPEATABLE
	| REPLACE
	| REPLICA
	| RESET
	| RESTART
	| RESTRICT
	| RETURN
	| RETURNS
	| REVOKE
	| ROLE
	| ROLLBACK
	| ROLLUP
	| ROUTINE
	| ROUTINES
	| ROWS
	| RULE
	| SAVEPOINT
	| SCALAR
	| SCHEMA
	| SCHEMAS
	| SCROLL
	| SEARCH
	| SECOND_P
	| SECURITY
	| SEQUENCE
	| SEQUENCES
	| SERIALIZABLE
	| SERVER
	| SESSION
	| SET
	| SETS
	| SHARE
	| SHOW
	| SIMPLE
	| SKIP
	| SNAPSHOT
	| SOURCE
	| SQL_P
	| STABLE
	| STANDALONE_P
	| START
	| STATEMENT
	| STATISTICS
	| STDIN
	| STDOUT
	| STORAGE
	| STORED
	| STRICT_P
	| STRING_P
	| STRIP_P
	| SUBSCRIPTION
	| SUPPORT
	| SYSID
	| SYSTEM_P
	| TABLES
	| TABLESPACE
	| TARGET
	| TEMP
	| TEMPLATE
	| TEMPORARY
	| TEXT_P
	| TIES
	| TRANSACTION
	| TRANSFORM
	| TRIGGER
	| TRUNCATE
	| TRUSTED
	| TYPE_P
	| TYPES_P
	| UESCAPE
	| UNBOUNDED
	| UNCOMMITTED
	| UNCONDITIONAL
	| UNENCRYPTED
	| UNKNOWN
	| UNLISTEN
	| UNLOGGED
	| UNTIL
	| UPDATE
	| VACUUM
	| VALID
	| VALIDATE
	| VALIDATOR
	| VALUE_P
	| VARYING
	| VERSION_P
	| VIEW
	| VIEWS
	| VOLATILE
	| WHITESPACE_P
	| WITHIN
	| WITHOUT
	| WORK
	| WRAPPER
	| WRITE
	| XML_P
	| YEAR_P
	| YES_P
	| ZONE
	;

ColNameKeyword:
	BETWEEN
	| BIGINT
	| BIT
	| BOOLEAN_P
	| CHAR_P
	| CHARACTER
	| COALESCE
	| DEC
	| DECIMAL_P
	| EXISTS
	| EXTRACT
	| FLOAT_P
	| GREATEST
	| GROUPING
	| INOUT
	| INT_P
	| INTEGER
	| INTERVAL
	| JSON
	| JSON_ARRAY
	| JSON_ARRAYAGG
	| JSON_EXISTS
	| JSON_OBJECT
	| JSON_OBJECTAGG
	| JSON_QUERY
	| JSON_SCALAR
	| JSON_SERIALIZE
	| JSON_TABLE
	| JSON_VALUE
	| LEAST
	| MERGE_ACTION
	| NATIONAL
	| NCHAR
	| NONE
	| NORMALIZE
	| NULLIF
	| NUMERIC
	| OUT_P
	| OVERLAY
	| POSITION
	| PRECISION
	| REAL
	| ROW
	| SETOF
	| SMALLINT
	| SUBSTRING
	| TIME
	| TIMESTAMP
	| TREAT
	| TRIM
	| VALUES
	| VARCHAR
	| XMLATTRIBUTES
	| XMLCONCAT
	| XMLELEMENT
	| XMLEXISTS
	| XMLFOREST
	| XMLNAMESPACES
	| XMLPARSE
	| XMLPI
	| XMLROOT
	| XMLSERIALIZE
	| XMLTABLE
	;

TypeFuncNameKeyword:
	AUTHORIZATION
	| BINARY
	| COLLATION
	| CONCURRENTLY
	| CROSS
	| CURRENT_SCHEMA
	| FREEZE
	| FULL
	| ILIKE
	| INNER_P
	| IS
	| ISNULL
	| JOIN
	| LEFT
	| LIKE
	| NATURAL
	| NOTNULL
	| OUTER_P
	| OVERLAPS
	| RIGHT
	| SIMILAR
	| TABLESAMPLE
	| VERBOSE
	;

ReservedKeyword:
	ALL
	| ANALYSE
	| ANALYZE
	| AND
	| ANY
	| ARRAY
	| AS
	| ASC
	| ASYMMETRIC
	| BOTH
	| CASE
	| CAST
	| CHECK
	| COLLATE
	| COLUMN
	| CONSTRAINT
	| CREATE
	| CURRENT_CATALOG
	| CURRENT_DATE
	| CURRENT_ROLE
	| CURRENT_TIME
	| CURRENT_TIMESTAMP
	| CURRENT_USER
	| DEFAULT
	| DEFERRABLE
	| DESC
	| DISTINCT
	| DO
	| ELSE
	| END_P
	| EXCEPT
	| FALSE_P
	| FETCH
	| FOR
	| FOREIGN
	| FROM
	| GRANT
	| GROUP_P
	| HAVING
	| IN_P
	| INITIALLY
	| INTERSECT
	| INTO
	| LATERAL_P
	| LEADING
	| LIMIT
	| LOCALTIME
	| LOCALTIMESTAMP
	| NOT
	| NULL_P
	| OFFSET
	| ON
	| ONLY
	| OR
	| ORDER
	| PLACING
	| PRIMARY
	| REFERENCES
	| RETURNING
	| SELECT
	| SESSION_USER
	| SOME
	| SYMMETRIC
	| SYSTEM_USER
	| TABLE
	| THEN
	| TO
	| TRAILING
	| TRUE_P
	| UNION
	| UNIQUE
	| USER
	| USING
	| VARIADIC
	| WHEN
	| WHERE
	| WINDOW
	| WITH
	;

%%

// OnConflict action constants
const (
	ONCONFLICT_NONE    = 0
	ONCONFLICT_NOTHING = 1
	ONCONFLICT_UPDATE  = 2
)

// ViewCheckOption constants
const (
	VIEW_CHECK_OPTION_NONE     = 0
	VIEW_CHECK_OPTION_LOCAL    = 1
	VIEW_CHECK_OPTION_CASCADED = 2
)

// Helper functions called from grammar actions

// setSyntaxSpan stamps a default production span without replacing a more
// precise span set by a grammar action.
func setSyntaxSpan(node ast.Node, span loc.Span) {
	if !ast.NodePresent(node) || !span.Valid() {
		return
	}
	if !node.Span().Valid() {
		node.SetSpan(span)
	}
}

// setParseResult stores the parse result in the lexer adapter.
func setParseResult(lex yyLexer, result *ast.List) {
	if s, ok := lex.(*Scanner); ok {
		s.result = result
	}
}

func makeList(n ast.Node) *ast.List {
	if n == nil {
		return &ast.List{}
	}
	return &ast.List{Items: []ast.Node{n}}
}

func appendList(l *ast.List, n ast.Node) *ast.List {
	if l == nil {
		return makeList(n)
	}
	if n != nil {
		l.Items = append(l.Items, n)
	}
	return l
}

func prependList(n ast.Node, l *ast.List) *ast.List {
	if l == nil {
		return makeList(n)
	}
	if n != nil {
		l.Items = append([]ast.Node{n}, l.Items...)
	}
	return l
}

func makeList2(a ast.Node, b ast.Node) *ast.List {
	return &ast.List{Items: []ast.Node{a, b}}
}

func makeListNode(l *ast.List) ast.Node {
	return l
}

func makeRangeVar(names *ast.List) ast.Node {
	rv := &ast.RangeVar{Inh: true, Relpersistence: 'p'}
	if names != nil && len(names.Items) > 0 {
		switch len(names.Items) {
		case 1:
			rv.Relname = names.Items[0].(*ast.String).Str
		case 2:
			rv.Schemaname = names.Items[0].(*ast.String).Str
			rv.Relname = names.Items[1].(*ast.String).Str
		case 3:
			rv.Catalogname = names.Items[0].(*ast.String).Str
			rv.Schemaname = names.Items[1].(*ast.String).Str
			rv.Relname = names.Items[2].(*ast.String).Str
		}
	}
	return rv
}

func makeRangeVarList(nameList *ast.List) *ast.List {
	if nameList == nil {
		return nil
	}
	result := &ast.List{}
	for _, item := range nameList.Items {
		rv := makeRangeVar(item.(*ast.List))
		result.Items = append(result.Items, rv)
	}
	return result
}

func makeAExpr(kind ast.A_Expr_Kind, op string, lexpr, rexpr ast.Node) ast.Node {
	return &ast.A_Expr{
		Kind:  kind,
		Name:  &ast.List{Items: []ast.Node{&ast.String{Str: op}}},
		Lexpr: lexpr,
		Rexpr: rexpr,
	}
}

// makeAExprFromList creates an A_Expr with an operator name list (from qual_Op).
// makeNameListAsAnyNameList wraps each name string into a single-element
// List (matching PG's any_name format for DropStmt.Objects).
func parsePartitionStrategy(strategy string) string {
	switch strategy {
	case "list", "LIST", "List":
		return "l"
	case "range", "RANGE", "Range":
		return "r"
	case "hash", "HASH", "Hash":
		return "h"
	default:
		return strategy
	}
}

func makeAExprFromList(kind ast.A_Expr_Kind, name *ast.List, lexpr, rexpr ast.Node) ast.Node {
	return &ast.A_Expr{
		Kind:  kind,
		Name:  name,
		Lexpr: lexpr,
		Rexpr: rexpr,
	}
}

func makeBoolExpr(boolop ast.BoolExprType, arg1, arg2 ast.Node) ast.Node {
	be := &ast.BoolExpr{
		Boolop: boolop,
		Args:   &ast.List{},
	}
	if arg1 != nil {
		be.Args.Items = append(be.Args.Items, arg1)
	}
	if arg2 != nil {
		be.Args.Items = append(be.Args.Items, arg2)
	}
	return be
}

func makeBetweenArgs(lower, upper ast.Node) ast.Node {
	return &ast.List{Items: []ast.Node{lower, upper}}
}

func doNegate(n ast.Node) ast.Node {
	// For numeric constants, negate in place
	if ac, ok := n.(*ast.A_Const); ok {
		if i, ok := ac.Val.(*ast.Integer); ok {
			i.Ival = -i.Ival
			return n
		}
		if f, ok := ac.Val.(*ast.Float); ok {
			if f.Fval[0] == '-' {
				f.Fval = f.Fval[1:]
			} else {
				f.Fval = "-" + f.Fval
			}
			return n
		}
	}
	// Otherwise, create unary minus expression
	return makeAExpr(ast.AEXPR_OP, "-", nil, n)
}

func concatLists(a, b *ast.List) *ast.List {
	if a == nil { return b }
	if b == nil { return a }
	result := &ast.List{Items: make([]ast.Node, 0, len(a.Items)+len(b.Items))}
	result.Items = append(result.Items, a.Items...)
	result.Items = append(result.Items, b.Items...)
	return result
}


func makeTypeName(typeName string) *ast.TypeName {
	return &ast.TypeName{
		Names: &ast.List{Items: []ast.Node{
			&ast.String{Str: "pg_catalog"},
			&ast.String{Str: typeName},
		}},
		Location: -1,
	}
}

func makeIntConst(val int64) ast.ExprNode {
	return ast.NewValueExpr(val)
}

func makeStringConst(str string) ast.ExprNode {
	return ast.NewValueExpr(str)
}

func makeBoolAConst(val int64) ast.ExprNode {
	return ast.NewValueExpr(val != 0)
}

func makeNullAConst() ast.ExprNode {
	return ast.NewValueExpr(nil)
}

func makeNotExpr(expr ast.Node) ast.Node {
	return &ast.BoolExpr{
		Boolop: ast.NOT_EXPR,
		Args:   &ast.List{Items: []ast.Node{expr}},
		Location: -1,
	}
}

func setFuncAlias(n *ast.RangeFunction, clause ast.Node) {
	if clause == nil {
		return
	}
	// If it's an Alias, just set the alias
	if alias, ok := clause.(*ast.Alias); ok {
		n.Alias = alias
		return
	}
	// If it's a List with [alias, coldeflist], it's a func_alias_clause with column defs
	if list, ok := clause.(*ast.List); ok && len(list.Items) == 2 {
		if alias, ok2 := list.Items[0].(*ast.Alias); ok2 {
			n.Alias = alias
		}
		if coldeflist, ok2 := list.Items[1].(*ast.List); ok2 {
			n.Coldeflist = coldeflist
		}
	}
}

func setJoinQual(n *ast.JoinExpr, qual ast.Node) {
	if qual == nil {
		return
	}
	if list, ok := qual.(*ast.List); ok {
		// Check if this is a USING clause with alias: [nameList, alias]
		if len(list.Items) == 2 {
			if innerList, ok2 := list.Items[0].(*ast.List); ok2 {
				if alias, ok3 := list.Items[1].(*ast.Alias); ok3 {
					n.UsingClause = innerList
					n.JoinUsing = alias
					return
				}
			}
		}
		n.UsingClause = list
	} else {
		n.Quals = qual
	}
}

func makeTypeNameFromNameList(names *ast.List) ast.Node {
	return typeNameNode(typeNameFromList(names))
}

func asJsonOutput(n ast.Node) *ast.JsonOutput {
	if n == nil {
		return nil
	}
	return n.(*ast.JsonOutput)
}

func asJsonFormat(n ast.Node) *ast.JsonFormat {
	if n == nil {
		return nil
	}
	return n.(*ast.JsonFormat)
}

func asJsonBehavior(n ast.Node) *ast.JsonBehavior {
	if n == nil {
		return nil
	}
	return n.(*ast.JsonBehavior)
}

func splitJsonBehaviorClause(n ast.Node) (*ast.JsonBehavior, *ast.JsonBehavior) {
	if n == nil {
		return nil, nil
	}
	list := n.(*ast.List)
	var onEmpty, onError *ast.JsonBehavior
	if list.Items[0] != nil {
		onEmpty = list.Items[0].(*ast.JsonBehavior)
	}
	if list.Items[1] != nil {
		onError = list.Items[1].(*ast.JsonBehavior)
	}
	return onEmpty, onError
}

func asJsonTablePathSpec(n ast.Node) *ast.JsonTablePathSpec {
	if n == nil {
		return nil
	}
	return n.(*ast.JsonTablePathSpec)
}

func doNegateFloat(f *ast.Float) {
	if len(f.Fval) > 0 && f.Fval[0] == '-' {
		f.Fval = f.Fval[1:]
	} else {
		f.Fval = "-" + f.Fval
	}
}

func makeDefElem(name string, arg ast.Node) ast.Node {
	return &ast.DefElem{
		Defname:  name,
		Arg:      arg,
		Location: -1,
	}
}

// makeFuncName converts a function name string to a *ast.List of String ast.
// With one argument, creates an unqualified name.
// With two arguments, creates a schema-qualified name (e.g. "pg_catalog", "func").
func makeFuncName(parts ...string) *ast.List {
	items := make([]ast.Node, len(parts))
	for i, p := range parts {
		items[i] = &ast.String{Str: p}
	}
	return &ast.List{Items: items}
}

// makeSQLValueFunction creates a SQLValueFunction node, rejecting precision
// values that would overflow the int32 Typmod field.
func makeSQLValueFunction(op ast.SVFOp, typmod int64, lexer yyLexer) ast.ExprNode {
	if typmod > math.MaxInt32 {
		lexer.Error("time precision is out of range")
		typmod = math.MaxInt32
	}
	return &ast.SQLValueFunction{Op: op, Typmod: int32(typmod), Location: -1}
}

// makeTypeCast creates a TypeCast node.
func makeTypeCast(arg ast.ExprNode, tp *types.FieldType) ast.ExprNode {
	return ast.NewTypeCastExpr(arg, tp)
}

// roleSpecOrNil safely casts a node to *ast.RoleSpec, returning nil if the node is nil.
func roleSpecOrNil(n ast.Node) *ast.RoleSpec {
	if n == nil {
		return nil
	}
	return n.(*ast.RoleSpec)
}

// SelectLimit is an internal helper struct for passing LIMIT/OFFSET through grammar rules.
// It is not a Node type - just used during parsing.
type SelectLimit struct {
	LimitOffset ast.ExprNode
	LimitCount  ast.ExprNode
	LimitOption ast.LimitOption
}

// GroupClause is an internal helper struct for passing GROUP BY clause
// distinct flag and list through grammar rules (matches PostgreSQL's GroupClause).
type GroupClause struct {
	Distinct bool
	List     *ast.List
}

// KeyAction is an internal helper struct for passing FK action and column list
// through grammar rules (matches PostgreSQL's KeyAction).
type KeyAction struct {
	Action byte
	Cols   *ast.List
}

// KeyActions is an internal helper struct for passing FK update/delete actions
// through grammar rules (matches PostgreSQL's KeyActions).
type KeyActions struct {
	UpdateAction *KeyAction
	DeleteAction *KeyAction
}

// SetQuantifier constants matching PostgreSQL's SetQuantifier enum.
const (
	SET_QUANTIFIER_DEFAULT  = int64(0)
	SET_QUANTIFIER_ALL      = int64(1)
	SET_QUANTIFIER_DISTINCT = int64(2)
)

// importQualification is an internal helper struct for passing IMPORT FOREIGN SCHEMA
// qualification type and table list through grammar rules.
// It implements the ast.Node interface so it can be passed through %union.
type importQualification struct {
	listType  ast.ImportForeignSchemaType
	tableList *ast.List
}


// splitColQualList separates a list of column qualifiers (constraints and COLLATE)
// into the ColumnDef's constraints and collClause fields.
// This matches PostgreSQL's SplitColQualList function.
func applyConstraintAttrs(n *ast.Constraint, attrs int64) {
	if attrs&int64(ast.CAS_DEFERRABLE) != 0 {
		n.Deferrable = true
	}
	if attrs&int64(ast.CAS_INITIALLY_DEFERRED) != 0 {
		n.InitDeferred = true
		n.Deferrable = true
	}
	if attrs&int64(ast.CAS_NOT_VALID) != 0 {
		n.SkipValidation = true
		n.InitiallyValid = false
	}
	if attrs&int64(ast.CAS_NO_INHERIT) != 0 {
		n.NoInherit = true
	}
}



// relpersistenceForTemp returns the relpersistence byte based on temp flag.
// In PostgreSQL: 'p' = permanent (default), 't' = temporary
func relpersistenceForTemp(tempFlag int64) byte {
	switch tempFlag {
	case 1:
		return 't'
	case 2:
		return 'u'
	default:
		return 'p'
	}
}

// extractArgTypes extracts the type names from a list of FunctionParameter ast.
func extractArgTypes(args *ast.List) *ast.List {
	if args == nil {
		return nil
	}
	result := &ast.List{}
	for _, item := range args.Items {
		fp, ok := item.(*ast.FunctionParameter)
		if ok {
			result.Items = append(result.Items, typeNameNode(fp.ArgType))
		}
	}
	return result
}

// extractAggrArgTypes extracts the type names from an aggregate argument list.
// aggr_args returns a 2-element list: [args_list, Integer_marker].
// The first element is either nil (for agg(*)) or a *List of FunctionParameter ast.
// This function extracts the ArgType from each FunctionParameter in the first element.
func extractAggrArgTypes(args *ast.List) *ast.List {
	if args == nil || len(args.Items) < 1 {
		return nil
	}
	// First element is the args list (may be nil for agg(*))
	argsList, ok := args.Items[0].(*ast.List)
	if !ok || argsList == nil {
		return makeList(&ast.A_Star{})
	}
	result := &ast.List{}
	for _, item := range argsList.Items {
		if fp, ok := item.(*ast.FunctionParameter); ok {
			result.Items = append(result.Items, typeNameNode(fp.ArgType))
		}
	}
	return result
}

// makeOrderedSetArgs builds the aggr_args result for ordered-set aggregates
// with both direct args and ORDER BY args.
// Returns a 2-element list: [combined_args_list, Integer{ndirectargs}].
// The combined list has the direct args followed by the ORDER BY args.
func makeOrderedSetArgs(directArgs *ast.List, orderedArgs *ast.List) *ast.List {
	combined := &ast.List{}
	ndirect := 0
	if directArgs != nil {
		combined.Items = append(combined.Items, directArgs.Items...)
		ndirect = len(directArgs.Items)
	}
	if orderedArgs != nil {
		combined.Items = append(combined.Items, orderedArgs.Items...)
	}
	return makeList2(combined, &ast.Integer{Ival: int64(ndirect)})
}

// checkFuncName validates that a qualified function name has at most 3 parts.
func checkFuncName(names *ast.List) *ast.List {
	// Just return as-is; error checking can be done later
	return names
}

// intToString converts an int64 to a string representation.
func intToString(val int64) string {
	return fmt.Sprintf("%d", val)
}

// makeRangeVarFromAnyName creates a RangeVar from a qualified name list (list of String nodes).
// It handles 1-part (name), 2-part (schema.name), and 3-part (catalog.schema.name) names.
func makeRangeVarFromAnyName(names *ast.List) *ast.RangeVar {
	rv := &ast.RangeVar{
		Inh:            true,
		Relpersistence: 'p',
		Location:       -1,
	}
	if names == nil {
		return rv
	}
	switch len(names.Items) {
	case 1:
		rv.Relname = names.Items[0].(*ast.String).Str
	case 2:
		rv.Schemaname = names.Items[0].(*ast.String).Str
		rv.Relname = names.Items[1].(*ast.String).Str
	case 3:
		rv.Catalogname = names.Items[0].(*ast.String).Str
		rv.Schemaname = names.Items[1].(*ast.String).Str
		rv.Relname = names.Items[2].(*ast.String).Str
	}
	return rv
}
