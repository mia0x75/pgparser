// PostgreSQL grammar for Go - translated from PostgreSQL's gram.y
// This grammar is compatible with goyacc (golang.org/x/tools/cmd/goyacc)
//
// To generate the parser:
//   go run golang.org/x/tools/cmd/goyacc@v0.38.0 -o parser.go -l -p yy parser.y

%{
package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mia0x75/pgparser/ast"
	"github.com/mia0x75/pgparser/model"
	"github.com/mia0x75/pgparser/opcode"
	"github.com/mia0x75/pgparser/types"
)

%}

// Union type for semantic values
%union {
	node      ast.Node
	expr      ast.ExprNode
	exprs     []ast.ExprNode
	fields    []*ast.SelectField
	froms     []ast.ResultSetNode
	list      *ast.List
	str       string
	ival      int64
	boolean   bool
	ivalp     *int64
	jtype     int        // for JoinType values
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
	ctes       []*ast.CommonTableExpression
	colnames   []*ast.ColumnName
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
%start stmtblock

%type <list>  stmtblock resource_group_options resource_group_option_list opt_index_name
%type <list>  NumericOnly_list
%type <node>  stmt SelectStmt simple_select select_clause
%type <node>  ResourceGroupStmt resource_group_option DropServerStmt partition_bound_spec partition_bound_expr
%type <node>  json_aggregate_func xmltable xmltable_column_el xmltable_column_option_el xml_namespace_el json_table json_table_column_definition json_table_column_path_clause_opt
%type <list>  xmltable_column_list xmltable_column_option_list xml_namespace_list json_table_column_definition_list json_array_aggregate_order_by_clause_opt
%type <str>   json_table_path_name_opt path_opt
%type <node>  select_with_parens select_no_parens
%type <expr>  a_expr b_expr c_expr columnref AexprConst func_expr func_application func_expr_common_subexpr
%type <node>  target_el
%type <expr>  where_clause where_or_current_clause
%type <list>  func_name
%type <froms>  from_clause
%type <node>  table_ref relation_expr joined_table join_qual
%type <node>  func_table tablesample_clause
%type <boolean>  opt_ordinality resource_group_type opt_force
%type <boolean>  opt_table
%type <expr>  opt_repeatable_clause
%type <ival>  join_type
%type <list>  stmtmulti
%type <fields>  opt_target_list
%type <fields>  target_list
%type <froms>  from_list
%type <grpclause> group_clause
%type <list>  group_by_list
%type <node>  group_by_item
%type <expr>  having_clause
%type <node>  empty_grouping_set cube_clause rollup_clause grouping_sets_clause
%type <node>  sort_clause opt_sort_clause
%type <list>  sortby_list
%type <exprs>  expr_list
%type <list>  opt_indirection indirection func_arg_list func_arg_list_opt
%type <exprs>  extract_list overlay_list position_list substr_list trim_list
%type <str>   extract_arg unicode_normal_form
%type <node>  sortby indirection_el
%type <expr>  func_arg_expr
%type <str>   name ColId ColLabel attr_name
%type <node>   opt_alias_clause alias_clause func_alias_clause
%type <str>   unreserved_keyword col_name_keyword type_func_name_keyword reserved_keyword
%type <list>  opt_name_list name_list any_name qualified_name
%type <ival>  opt_asc_desc
%type <ival>  Iconst
%type <str>   Sconst
%type <boolean> opt_all_clause
%type <ival>   set_quantifier
%type <exprs>  distinct_clause
%type <exprs>
%type <node>  with_clause opt_with_clause common_table_expr
%type <ival>  opt_materialized
%type <ctes>  cte_list
%type <list>  window_clause window_definition_list
%type <node>  window_definition window_specification over_clause opt_frame_clause
%type <frame>  frame_extent
%type <bound>  frame_bound
%type <str>   opt_existing_window_name
%type <exprs>  opt_partition_clause
%type <ival>  opt_window_exclusion_clause
%type <node>  within_group_clause
%type <expr>  filter_clause
%type <node>  func_expr_windowless
%type <slimit> select_limit limit_clause opt_select_limit
%type <list>  for_locking_clause opt_for_locking_clause for_locking_items
%type <node>  for_locking_item
%type <ival>  for_locking_strength opt_nowait_or_skip
%type <list>  locked_rels_list
%type <expr>   offset_clause select_limit_value select_offset_value select_fetch_first_value
%type <ival>   sub_type
%type <list>   subquery_Op
%type <expr>   case_expr case_default case_arg
%type <node>   when_clause
%type <list>   when_clause_list
%type <tp>  Typename SimpleTypename GenericType Numeric opt_float Character
%type <tp>  ConstDatetime ConstInterval Bit BitWithLength BitWithoutLength
%type <tp>  ConstTypename ConstBit ConstCharacter
%type <boolean>  opt_timezone
%type <list>  opt_interval
%type <list>  interval_second
%type <exprs>      opt_type_modifiers
%type <list>      opt_array_bounds
%type <str>       type_function_name
%type <boolean>   opt_varying
%type <expr>      array_expr opt_slice_bound
%type <exprs>      array_expr_list
%type <expr>      row explicit_row
%type <exprs>      implicit_row
%type <node>  InsertStmt insert_rest
%type <node>  insert_target
%type <colnames>  insert_column_list
%type <fields>  returning_clause
%type <node>  insert_column_item
%type <node>  opt_on_conflict
%type <node>  values_clause
%type <assignments>  set_clause_list
%type <list>
%type <node>  set_clause
%type <node>  into_clause OptTempTableName
%type <list>
%type <node>
%type <node>  opt_search_clause opt_cycle_clause
%type <node>
%type <node>  UpdateStmt
%type <node>  DeleteStmt
%type <froms>  using_clause
%type <node>  relation_expr_opt_alias
%type <node>  CreateStmt
%type <ival>  OptTemp
%type <list>  OptTableElementList TableElementList OptTypedTableElementList TypedTableElementList
%type <node>  TypedTableElement columnOptions
%type <node>  TableElement columnDef TableConstraint TableLikeClause
%type <ival>  TableLikeOptionList TableLikeOption
%type <colopts>  ColConstraintList opt_column_constraints
%type <node>  ColConstraint ColConstraintElem ConstraintAttr ConstraintElem
%type <node>  DomainConstraint DomainConstraintElem
%type <list>  opt_column_list columnList
%type <partspec>  OptPartitionSpec PartitionSpec_rule
%type <list>  OptWith
%type <ival>  OnCommitOption
%type <str>   OptTableSpace OptAccessMethod
%type <list>  part_params OptInherit opt_collate
%type <node>  part_elem
%type <partbound>  PartitionBoundSpec ForValues
%type <node>  AlterTableStmt RenameStmt DropStmt
%type <list>  alter_table_cmds
%type <node>  alter_table_cmd
%type <ival>  opt_drop_behavior
%type <ival>  object_type_any_name
%type <list>  any_name_list
%type <node>  IndexStmt
%type <boolean>  opt_unique opt_concurrently
%type <str>   opt_single_name access_method_clause
%type <str>   opt_name
%type <list>  index_params opt_include index_including_params
%type <node>  index_elem
%type <ival>  opt_nulls_order
%type <node>  ViewStmt
%type <ival>  opt_check_option
%type <node>  CreateFunctionStmt
%type <boolean>  opt_or_replace
%type <list>  func_args_with_defaults func_args_with_defaults_list
%type <node>  func_arg_with_default func_arg
%type <list>  table_func_column_list
%type <node>  table_func_column
%type <str>   param_name
%type <ival>  arg_class
%type <tp>  func_return func_type
%type <list>  createfunc_opt_list opt_createfunc_opt_list
%type <node>  createfunc_opt_item common_func_opt_item
%type <node>  opt_routine_body ReturnStmt routine_body_stmt
%type <list>  routine_body_stmt_list
%type <node>  TransactionStmt
%type <boolean>  opt_transaction_chain
%type <node>  ExplainStmt ExplainableStmt
%type <node>  CopyStmt
%type <boolean>  copy_from opt_program
%type <str>   copy_file_name
%type <list>  utility_option_list copy_opt_list copy_options transaction_mode_list transaction_mode_list_or_empty
%type <node>  copy_opt_item opt_binary copy_delimiter transaction_mode_item
%type <str>   iso_level
%type <node>  utility_option_elem
%type <str>   utility_option_name
%type <node>  utility_option_arg
%type <list>  utility_option_arg_list
%type <str>   opt_boolean_or_string
%type <node>  NumericOnly
%type <node>  GrantStmt RevokeStmt privilege_target
%type <list>  privileges privilege_list grantee_list
%type <node>  privilege grantee RoleSpec
%type <boolean> opt_grant_grant_option
%type <node>  CheckPointStmt DiscardStmt ListenStmt UnlistenStmt NotifyStmt LoadStmt ClosePortalStmt ConstraintsSetStmt
%type <list>  constraints_set_list
%type <boolean> constraints_set_mode
%type <str>   file_name cursor_name
%type <list>  qualified_name_list
%type <node>  VariableSetStmt VariableShowStmt VariableResetStmt
%type <node>  PrepareStmt ExecuteStmt DeallocateStmt
%type <node>  set_rest set_rest_more generic_set
%type <str>   var_name
%type <list>  var_list
%type <node>  var_value zone_value
%type <str>   NonReservedWord_or_Sconst NonReservedWord
%type <ival>  document_or_content
%type <tps>  prep_type_clause
%type <exprs>  execute_param_clause
%type <str>   opt_encoding
%type <node>  reset_rest generic_reset
%type <tps>  type_list
%type <ival>  SignedIconst
%type <node>  PreparableStmt
%type <node>  TruncateStmt LockStmt VacuumStmt AnalyzeStmt ClusterStmt ReindexStmt CommentStmt SecLabelStmt
%type <boolean> opt_restart_seqs opt_nowait opt_verbose opt_analyze opt_full opt_freeze
%type <list>  relation_expr_list opt_vacuum_relation_list vacuum_relation_list
%type <node>  vacuum_relation
%type <ival>  opt_lock lock_type reindex_target_type reindex_target_multitable OptNoLog
%type <str>   cluster_index_specification column_compression comment_text opt_provider
%type <ival>  object_type_name object_type_name_on_any_name
%type <list>  opt_reindex_option_list
%type <node>  DeclareCursorStmt FetchStmt MergeStmt CallStmt DoStmt
%type <ival>  cursor_options opt_hold
%type <node>  fetch_args
%type <node>  merge_when_clause
%type <list>  merge_when_list
%type <node>  merge_update merge_delete merge_insert
%type <expr>  opt_merge_when_condition
%type <exprs>  merge_values_clause
%type <ival>  merge_when_tgt_matched merge_when_tgt_not_matched
%type <ival>  override_kind
%type <list>  dostmt_opt_list
%type <node>  dostmt_opt_item
%type <node>  CreateRoleStmt AlterRoleStmt AlterRoleSetStmt DropRoleStmt
%type <node>  CreateUserStmt CreateGroupStmt AlterGroupStmt
%type <node>  GrantRoleStmt RevokeRoleStmt
%type <list>  OptRoleList AlterOptRoleList
%type <node>  CreateOptRoleElem AlterOptRoleElem
%type <str>   RoleId
%type <list>  role_list
%type <node>  SetResetClause
%type <node>  opt_granted_by
%type <list>  grant_role_opt_list
%type <node>  grant_role_opt
%type <node>  grant_role_opt_value
%type <ival>  add_drop
%type <node>  CreatedbStmt AlterDatabaseStmt AlterDatabaseSetStmt DropdbStmt AlterSystemStmt
%type <list>  createdb_opt_list createdb_opt_items
%type <node>  createdb_opt_item
%type <str>   createdb_opt_name
%type <list>  drop_option_list
%type <node>  drop_option
%type <node>  CreateSchemaStmt CreateSeqStmt AlterSeqStmt
%type <node>  CreateDomainStmt AlterDomainStmt AlterEnumStmt AlterCollationStmt AlterCompositeTypeStmt
%type <list>  OptSchemaEltList OptParenthesizedSeqOptList OptSeqOptList SeqOptList
%type <list>  alter_identity_column_option_list
%type <node>  alter_identity_column_option
%type <ival>  generated_when
%type <node>  SeqOptElem
%type <node>  schema_stmt
%type <list>  alter_type_cmds
%type <node>  alter_type_cmd
%type <node>  alter_column_default
%type <boolean>  opt_if_not_exists
%type <node>  TableFuncElement rowsfrom_item
%type <node>  opt_collate_clause
%type <node>  DefineStmt CompositeTypeStmt CreateEnumStmt CreateRangeStmt
%type <list>  definition def_list old_aggr_definition old_aggr_list
%type <node>  def_elem def_arg old_aggr_elem
%type <list>  OptTableFuncElementList TableFuncElementList opt_col_def_list
%type <list>  opt_enum_val_list enum_val_list
%type <list>  aggr_args aggr_args_list
%type <node>  aggr_arg
%type <list>  any_operator qual_all_Op qual_Op
%type <str>   all_Op MathOp
%type <node>  AlterFunctionStmt RemoveFuncStmt RemoveAggrStmt RemoveOperStmt
%type <node>  CreateTrigStmt CreateEventTrigStmt AlterEventTrigStmt
%type <node>  RuleStmt CreatePLangStmt
%type <list>  alterfunc_opt_list
%type <list>  function_with_argtypes_list func_args func_args_list rowsfrom_list
%type <node>  function_with_argtypes
%type <list>  aggregate_with_argtypes_list
%type <node>  aggregate_with_argtypes
%type <list>  operator_with_argtypes_list oper_argtypes
%type <node>  operator_with_argtypes
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
%type <list>  event_trigger_when_list
%type <node>  event_trigger_when_item
%type <list>  event_trigger_value_list
%type <ival>  enable_trigger
%type <list>  RuleActionList RuleActionMulti
%type <node>  RuleActionStmt RuleActionStmtOrEmpty
%type <ival>  event
%type <boolean>  opt_instead opt_trusted
%type <list>  handler_name opt_inline_handler opt_validator
%type <ival>  ConstraintAttributeSpec ConstraintAttributeElem
%type <ival>  key_match
%type <keyaction>  key_action
%type <keyaction>  key_update key_delete
%type <keyactions> key_actions
%type <list>  opt_c_include ExclusionConstraintList
%type <list>  ExclusionConstraintElem
%type <str>   OptConsTableSpace ExistingIndex
%type <list>  hash_partbound
%type <node>  hash_partbound_elem
%type <node>  OptConstrFromTable
%type <node>  xml_root_version opt_xml_root_standalone
%type <node>  xmlexists_argument
%type <node>
%type <list>  xml_attribute_list xml_attributes
%type <list>
%type <node> xml_attribute_el
%type <ival>  xml_whitespace_option xml_indent_option

%type <node>  json_value_expr json_format_clause json_format_clause_opt
%type <node>  json_returning_clause_opt
%type <node>  json_behavior json_behavior_clause_opt json_on_error_clause_opt
%type <node>
%type <list>  json_name_and_value_list json_value_expr_list
%type <list>  json_arguments json_passing_clause_opt
%type <list>
%type <list>
%type <node>  json_name_and_value json_argument
%type <node>
%type <ival>  json_behavior_type
%type <ival>  json_predicate_type_constraint
%type <ival>  json_wrapper_behavior json_quotes_clause_opt
%type <ival>  json_key_uniqueness_constraint_opt
%type <ival>  json_object_constructor_null_clause_opt json_array_constructor_null_clause_opt
%type <str>
%type <list>
%type <list>  attrs
%type <node>  CreateFdwStmt AlterFdwStmt
%type <node>  CreateForeignServerStmt AlterForeignServerStmt
%type <node>  CreateForeignTableStmt
%type <node>  CreateUserMappingStmt AlterUserMappingStmt DropUserMappingStmt
%type <node>  ImportForeignSchemaStmt
%type <list>  create_generic_options alter_generic_options
%type <list>  generic_option_list alter_generic_option_list
%type <node>  generic_option_elem alter_generic_option_elem
%type <str>   generic_option_name
%type <node>  generic_option_arg
%type <node>  fdw_option
%type <list>  fdw_options opt_fdw_options
%type <str>   opt_type foreign_server_version opt_foreign_server_version
%type <node>  auth_ident
%type <ival>  import_qualification_type
%type <iqual>  import_qualification
%type <node>  CreateExtensionStmt AlterExtensionStmt AlterExtensionContentsStmt
%type <list>  create_extension_opt_list alter_extension_opt_list
%type <node>  create_extension_opt_item alter_extension_opt_item
%type <node>  CreateTableSpaceStmt DropTableSpaceStmt
%type <node>  OptTableSpaceOwner
%type <node>  AlterTblSpcStmt
%type <node>  CreateAmStmt
%type <ival>  am_type
%type <node>  CreatePolicyStmt AlterPolicyStmt
%type <boolean>  RowSecurityDefaultPermissive
%type <str>   RowSecurityDefaultForCmd row_security_cmd
%type <list>  RowSecurityDefaultToRole RowSecurityOptionalToRole
%type <node>  RowSecurityOptionalExpr RowSecurityOptionalWithCheck
%type <node>  CreatePublicationStmt AlterPublicationStmt
%type <list>  opt_definition
%type <node>  PublicationObjSpec
%type <list>  pub_obj_list
%type <list>  reloptions opt_reloptions reloption_list
%type <node>  reloption_elem
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
%type <ival>  defacl_privilege_target
%type <list>  operator_def_list
%type <node>  operator_def_elem
%type <node>  operator_def_arg
%type <list>  stats_params
%type <node>  stats_param
%type <list>  opclass_item_list opclass_drop_list
%type <node>  opclass_item opclass_drop
%type <boolean>  opt_default
%type <list>  opt_opfamily opclass_purpose
%type <boolean>  opt_recheck
%type <ival>  cast_context
%type <boolean>  opt_if_exists
%type <list>  transform_element_list transform_type_list func_as
%type <boolean>  opt_no
%type <node>  create_as_target create_mv_target
%type <boolean> opt_with_data
%type <boolean> no_inherit
%type <boolean> opt_unique_null_treatment
%type <list>  opt_qualified_name
%type <node>  set_statistics_value
%type <list>  opt_stat_name_list

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
stmtblock:
	stmtmulti
		{
			setParseResult(yylex, $1)
		}
	;

stmtmulti:
	stmtmulti ';' stmt
		{
			if $3 != nil {
				$$ = appendList($1, $3)
			} else {
				$$ = $1
			}
		}
	| stmt
		{
			if $1 != nil {
				$$ = makeList($1)
			} else {
				$$ = nil
			}
		}
	;

stmt:
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
	| CreateStmt
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
	| ViewStmt
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
	| PrepareStmt
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
	| TruncateStmt
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
	| CreatedbStmt
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
	| DropdbStmt
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
	| CreateSeqStmt
		{
			$$ = $1
		}
	| AlterSeqStmt
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
	| CreateTrigStmt
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
	| CreateTableSpaceStmt
		{
			$$ = $1
		}
	| DropTableSpaceStmt
		{
			$$ = $1
		}
	| AlterTblSpcStmt
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
	opt_with_clause INSERT INTO insert_target insert_rest opt_on_conflict returning_clause
		{
			n := $5.(*ast.InsertStmt)
			n.Table = &ast.TableRefsClause{TableRefs: &ast.Join{Left: $4.(*ast.TableSource)}}
			if vals, ok := n.Select.(*ast.ValuesStmt); ok {
				for _, row := range vals.Rows {
					n.Lists = append(n.Lists, row.Values)
				}
				n.Select = nil
			}
			if $6 != nil {
				n.OnConflict = $6.(*ast.OnConflictClause)
			}
			n.Returning = returningFieldList($7)
			if $1 != nil {
				n.With = $1.(*ast.WithClause)
			}
			$$ = n
		}
	;

insert_target:
	qualified_name
		{
			$$ = makeTableSource($1)
		}
	| qualified_name AS ColId
		{
			ts := makeTableSource($1)
			ts.AsName = model.NewCIStr($3)
			$$ = ts
		}
	;

insert_rest:
	SelectStmt
		{
			$$ = &ast.InsertStmt{
				Select: $1.(ast.ResultSetNode),
			}
		}
	| OVERRIDING override_kind VALUE_P SelectStmt
		{
			$$ = &ast.InsertStmt{
				Select: $4.(ast.ResultSetNode),
			}
		}
	| '(' insert_column_list ')' SelectStmt
		{
			$$ = &ast.InsertStmt{
				Columns: $2,
				Select:  $4.(ast.ResultSetNode),
			}
		}
	| '(' insert_column_list ')' OVERRIDING override_kind VALUE_P SelectStmt
		{
			$$ = &ast.InsertStmt{
				Columns: $2,
				Select:  $7.(ast.ResultSetNode),
			}
		}
	| DEFAULT VALUES
		{
			$$ = &ast.InsertStmt{}
		}
	;

insert_column_list:
	insert_column_item
		{ $$ = []*ast.ColumnName{$1.(*ast.ColumnName)} }
	| insert_column_list ',' insert_column_item
		{ $$ = append($1, $3.(*ast.ColumnName)) }
	;

insert_column_item:
	ColId opt_indirection
		{
			$$ = &ast.ColumnName{Name: model.NewCIStr($1)}
		}
	;

opt_on_conflict:
	ON CONFLICT DO NOTHING
		{
			$$ = &ast.OnConflictClause{
				DoNothing: true,
			}
		}
	| ON CONFLICT DO UPDATE SET set_clause_list where_clause
		{
			$$ = &ast.OnConflictClause{
				UpdateList: $6,
				UpdateWhere: $7,
			}
		}
	| ON CONFLICT '(' index_params ')' DO NOTHING
		{
			$$ = &ast.OnConflictClause{
				IndexCols: indexColNamesFromElems($4),
				DoNothing: true,
			}
		}
	| ON CONFLICT '(' index_params ')' DO UPDATE SET set_clause_list where_clause
		{
			$$ = &ast.OnConflictClause{
				IndexCols:  indexColNamesFromElems($4),
				UpdateList: $9,
				UpdateWhere: $10,
			}
		}
	| ON CONFLICT ON CONSTRAINT name DO NOTHING
		{
			$$ = &ast.OnConflictClause{
				Constraint: $5,
				DoNothing:  true,
			}
		}
	| ON CONFLICT ON CONSTRAINT name DO UPDATE SET set_clause_list where_clause
		{
			$$ = &ast.OnConflictClause{
				Constraint: $5,
				UpdateList: $9,
				UpdateWhere: $10,
			}
		}
	| ON CONFLICT '(' index_params ')' where_clause DO NOTHING
		{
			$$ = &ast.OnConflictClause{
				IndexCols:  indexColNamesFromElems($4),
				TargetWhere: $6,
				DoNothing:  true,
			}
		}
	| ON CONFLICT '(' index_params ')' where_clause DO UPDATE SET set_clause_list where_clause
		{
			$$ = &ast.OnConflictClause{
				IndexCols:  indexColNamesFromElems($4),
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

returning_clause:
	RETURNING target_list
		{
			$$ = $2
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

set_clause_list:
	set_clause
		{
			$$ = []*ast.Assignment{$1.(*ast.Assignment)}
		}
	| set_clause_list ',' set_clause
		{
			$$ = append($1, $3.(*ast.Assignment))
		}
	;

set_clause:
	'(' columnList ')' '=' a_expr
		{
			$$ = &ast.Assignment{
				MultiColumns: columnNameListFromStrings($2),
				Expr:         $5,
			}
		}
	| ColId '=' a_expr
		{
			$$ = &ast.Assignment{
				Column: &ast.ColumnName{Name: model.NewCIStr($1)},
				Expr:   $3,
			}
		}
	| ColId indirection '=' a_expr
		{
			$$ = &ast.Assignment{
				Column: &ast.ColumnName{Name: model.NewCIStr($1)},
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
	opt_with_clause UPDATE relation_expr_opt_alias SET set_clause_list from_clause where_or_current_clause returning_clause
		{
			n := &ast.UpdateStmt{
				TableRefs: &ast.TableRefsClause{TableRefs: &ast.Join{Left: $3.(*ast.TableSource)}},
				List:      $5,
			}
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
	opt_with_clause DELETE_P FROM relation_expr_opt_alias using_clause where_or_current_clause returning_clause
		{
			n := &ast.DeleteStmt{
				TableRefs: &ast.TableRefsClause{TableRefs: &ast.Join{Left: $4.(*ast.TableSource)}},
			}
			if $5 != nil {
				n.Using = fromClauseOfNodes($5)
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

using_clause:
	USING from_list { $$ = $2 }
	| /* EMPTY */ { $$ = nil }
	;

relation_expr_opt_alias:
	relation_expr  %prec UMINUS
		{ $$ = $1 }
	| relation_expr ColId
		{
			$1.(*ast.TableSource).AsName = model.NewCIStr($2)
			$$ = $1
		}
	| relation_expr AS ColId
		{
			$1.(*ast.TableSource).AsName = model.NewCIStr($3)
			$$ = $1
		}
	;

/*****************************************************************************
 *
 *      CREATE TABLE statement
 *
 *****************************************************************************/

partition_bound_spec:
	FOR VALUES partition_bound_expr
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

partition_bound_expr:
	IN_P '(' expr_list ')'
		{
			$$ = &ast.PartitionBoundSpec{
				BoundType: "IN",
				In:        $3,
			}
		}
	| FROM '(' expr_list ')' TO '(' expr_list ')'
		{
			$$ = &ast.PartitionBoundSpec{
				BoundType: "FROM",
				From:      $3,
				To:        $7,
			}
		}
	| WITH '(' ColId Iconst ',' ColId Iconst ')'
		{
			$$ = &ast.PartitionBoundSpec{
				BoundType: "WITH",
				Options:   fmt.Sprintf("%s %d, %s %d", $3, $4, $6, $7),
			}
		}
	;

xmltable:
	XMLTABLE '(' c_expr xmlexists_argument COLUMNS xmltable_column_list ')'
		{
			$$ = &ast.XMLTableExpr{
				RowExpr: exprOrNil($3),
				DocExpr: exprOrNil($4),
				Columns: xmlTableColumns($6),
			}
		}
	| XMLTABLE '(' XMLNAMESPACES '(' xml_namespace_list ')' ','
		c_expr xmlexists_argument COLUMNS xmltable_column_list ')'
		{
			$$ = &ast.XMLTableExpr{
				RowExpr:    exprOrNil($8),
				DocExpr:    exprOrNil($9),
				Columns:    xmlTableColumns($11),
				Namespaces: xmlNamespaces($5),
			}
		}
	;

xmltable_column_list:
	xmltable_column_el
		{ $$ = makeList($1) }
	| xmltable_column_list ',' xmltable_column_el
		{ $$ = appendList($1, $3) }
	;

xmltable_column_el:
	ColId Typename
		{
			$$ = &ast.XMLTableColumn{
				Name:     $1,
				TypeName: $2,
			}
		}
	| ColId Typename xmltable_column_option_list
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

xmltable_column_option_list:
	xmltable_column_option_el
		{ $$ = makeList($1) }
	| xmltable_column_option_list xmltable_column_option_el
		{ $$ = appendList($1, $2) }
	;

xmltable_column_option_el:
	IDENT b_expr
		{
			$$ = makeDefElem($1, $2)
		}
	| DEFAULT b_expr
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
	| PATH b_expr
		{
			$$ = makeDefElem("path", $2)
		}
	;

xml_namespace_list:
	xml_namespace_el
		{ $$ = makeList($1) }
	| xml_namespace_list ',' xml_namespace_el
		{ $$ = appendList($1, $3) }
	;

xml_namespace_el:
	b_expr AS ColLabel
		{
			$$ = &ast.XMLTableNamespace{
				Expr: exprOrNil($1),
				Name: $3,
			}
		}
	| DEFAULT b_expr
		{
			$$ = &ast.XMLTableNamespace{
				Expr: exprOrNil($2),
			}
		}
	;

json_table:
	JSON_TABLE '('
		json_value_expr ',' a_expr json_table_path_name_opt
		json_passing_clause_opt
		COLUMNS '(' json_table_column_definition_list ')'
		json_on_error_clause_opt
	')'
		{
			$$ = &ast.JSONTableExpr{
				ContextItem: exprOrNil($3),
				PathSpec: &ast.JSONTablePathSpec{
					String: exprStringValue($5),
					Name:   $6,
				},
				Passing: exprsOfList($7),
				Columns: jsonTableColumns($10),
				OnError: jsonOnErrorName($12),
			}
		}
	;

json_table_path_name_opt:
	AS name
		{
			$$ = $2
		}
	| /* EMPTY */
		{
			$$ = ""
		}
	;

json_table_column_definition_list:
	json_table_column_definition
		{ $$ = makeList($1) }
	| json_table_column_definition_list ',' json_table_column_definition
		{ $$ = appendList($1, $3) }
	;

json_table_column_definition:
	ColId FOR ORDINALITY
		{
			$$ = &ast.JSONTableColumn{
				ColType: "ordinality",
				Name:    $1,
			}
		}
	| ColId Typename json_table_column_path_clause_opt
		json_wrapper_behavior json_quotes_clause_opt
		json_behavior_clause_opt
		{
			col := &ast.JSONTableColumn{
				ColType:  "regular",
				Name:     $1,
				TypeName: $2,
				PathSpec: jsonTablePathSpec($3),
				Wrapper:  jsonWrapperName($4),
				Quotes:   jsonQuotesName($5),
			}
			col.OnEmpty, col.OnError = jsonBehaviorNames($6)
			$$ = col
		}
	| ColId Typename json_format_clause json_table_column_path_clause_opt
		json_wrapper_behavior json_quotes_clause_opt
		json_behavior_clause_opt
		{
			col := &ast.JSONTableColumn{
				ColType:  "formatted",
				Name:     $1,
				TypeName: $2,
				PathSpec: jsonTablePathSpec($4),
				Wrapper:  jsonWrapperName($5),
				Quotes:   jsonQuotesName($6),
			}
			col.OnEmpty, col.OnError = jsonBehaviorNames($7)
			$$ = col
		}
	| ColId Typename EXISTS json_table_column_path_clause_opt
		json_on_error_clause_opt
		{
			$$ = &ast.JSONTableColumn{
				ColType:  "exists",
				Name:     $1,
				TypeName: $2,
				PathSpec: jsonTablePathSpec($4),
				OnError:  jsonOnErrorName($5),
			}
		}
	| NESTED path_opt Sconst json_table_path_name_opt
		COLUMNS '(' json_table_column_definition_list ')'
		{
			$$ = &ast.JSONTableColumn{
				ColType: "nested",
				PathSpec: &ast.JSONTablePathSpec{
					String: $3,
					Name:   $4,
				},
				Columns: jsonTableColumns($7),
			}
		}
	;

json_table_column_path_clause_opt:
	PATH Sconst
		{
			$$ = &ast.JSONTablePathSpec{
				String: $2,
			}
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

path_opt:
	PATH
		{
			$$ = ""
		}
	| /* EMPTY */
		{
			$$ = ""
		}
	;

CreateStmt:
	CREATE OptTemp TABLE qualified_name '(' OptTableElementList ')' OptInherit OptPartitionSpec OptAccessMethod OptWith OnCommitOption OptTableSpace
		{
			rv := makeRangeVar($4)
			relationName(rv).Relpersistence = relpersistenceForTemp($2)
			cols, constraints := tableElements($6)
			$$ = &ast.CreateTableStmt{
				Temporary:   $2 != 0,
				Table:       relationName(rv),
				Cols:        cols,
				Constraints: constraints,
				Inherits:    inheritTableNames($8),
				PartitionBy: partitionStrategy($9),
				WithOptions: relOptionString($11),
				Tablespace:  $13,
			}
		}
	| CREATE OptTemp TABLE IF_P NOT EXISTS qualified_name '(' OptTableElementList ')' OptInherit OptPartitionSpec OptAccessMethod OptWith OnCommitOption OptTableSpace
		{
			rv := makeRangeVar($7)
			relationName(rv).Relpersistence = relpersistenceForTemp($2)
			cols, constraints := tableElements($9)
			$$ = &ast.CreateTableStmt{
				Temporary:   $2 != 0,
				IfNotExists: true,
				Table:       relationName(rv),
				Cols:        cols,
				Constraints: constraints,
				Inherits:    inheritTableNames($11),
				PartitionBy: partitionStrategy($12),
				WithOptions: relOptionString($14),
				Tablespace:  $16,
			}
		}
	| CREATE OptTemp TABLE qualified_name '(' OptTableElementList ')' OptWith OptTableSpace OptInherit OptPartitionSpec
		{
			rv := makeRangeVar($4)
			relationName(rv).Relpersistence = relpersistenceForTemp($2)
			cols, constraints := tableElements($6)
			$$ = &ast.CreateTableStmt{
				Temporary:   $2 != 0,
				Table:       relationName(rv),
				Cols:        cols,
				Constraints: constraints,
				WithOptions: relOptionString($8),
				Tablespace:  $9,
				Inherits:    inheritTableNames($10),
				PartitionBy: partitionStrategy($11),
			}
		}
	| CREATE OptTemp TABLE qualified_name OF any_name OptTypedTableElementList OptPartitionSpec
		{
			rv := makeRangeVar($4)
			cols, constraints := tableElements($7)
			$$ = &ast.CreateTableStmt{
				Temporary:   $2 != 0,
				Table:       relationName(rv),
				ReferTable:  tableNameFromList($6),
				Cols:        cols,
				Constraints: constraints,
				PartitionBy: partitionStrategy($8),
			}
		}
	| CREATE OptTemp TABLE IF_P NOT EXISTS qualified_name OF any_name OptTypedTableElementList OptPartitionSpec
		{
			rv := makeRangeVar($7)
			cols, constraints := tableElements($10)
			$$ = &ast.CreateTableStmt{
				Temporary:   $2 != 0,
				IfNotExists: true,
				Table:       relationName(rv),
				ReferTable:  tableNameFromList($9),
				Cols:        cols,
				Constraints: constraints,
				PartitionBy: partitionStrategy($11),
			}
		}
	| CREATE OptTemp TABLE qualified_name PARTITION OF qualified_name OptTypedTableElementList partition_bound_spec OptPartitionSpec OptAccessMethod OptWith OnCommitOption OptTableSpace
		{
			rv := makeRangeVar($4)
			cols, constraints := tableElements($8)
			$$ = &ast.CreateTableStmt{
				Temporary:      $2 != 0,
				Table:          relationName(rv),
				ReferTable:     tableNameFromList($7),
				Cols:           cols,
				Constraints:    constraints,
				PartitionBound: $9.(*ast.PartitionBoundSpec),
				PartitionBy:    partitionStrategy($10),
				WithOptions:    relOptionString($12),
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
	INHERITS '(' qualified_name_list ')'	{ $$ = $3 }
	| /* EMPTY */							{ $$ = nil }
	;

OptPartitionSpec:
	PartitionSpec_rule	{ $$ = $1 }
	| /* EMPTY */		{ $$ = nil }
	;

OptAccessMethod:
	USING name  { $$ = $2 }
	| /* EMPTY */ { $$ = "" }
	;

OptWith:
	WITH reloptions				{ $$ = $2 }
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

OptTableSpace:
	TABLESPACE name				{ $$ = $2 }
	| /* EMPTY */				{ $$ = "" }
	;

PartitionSpec_rule:
	PARTITION BY ColId '(' part_params ')'
		{
			$$ = &ast.PartitionSpec{
				Strategy:   parsePartitionStrategy($3),
				PartParams: $5,
				Location:   -1,
			}
		}
	;

part_params:
	part_elem
		{ $$ = makeList($1) }
	| part_params ',' part_elem
		{ $$ = appendList($1, $3) }
	;

part_elem:
	ColId opt_collate opt_qualified_name
		{
			$$ = &ast.PartitionElem{
				Name:      $1,
				Collation: $2,
				Opclass:   $3,
				Location:  -1,
			}
		}
	| func_expr_windowless opt_collate opt_qualified_name
		{
			$$ = &ast.PartitionElem{
				Expr:      $1,
				Collation: $2,
				Opclass:   $3,
				Location:  -1,
			}
		}
	| '(' a_expr ')' opt_collate opt_qualified_name
		{
			$$ = &ast.PartitionElem{
				Expr:      $2,
				Collation: $4,
				Opclass:   $5,
				Location:  -1,
			}
		}
	;

opt_collate:
	COLLATE any_name	{ $$ = $2 }
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
	FOR VALUES IN_P '(' expr_list ')'
		{
			$$ = &ast.PartitionBoundSpec{
				BoundType: "IN",
				In:        $5,
			}
		}
	/* a RANGE partition */
	| FOR VALUES FROM '(' expr_list ')' TO '(' expr_list ')'
		{
			$$ = &ast.PartitionBoundSpec{
				BoundType: "FROM",
				From:      $5,
				To:        $9,
			}
		}
	/* a HASH partition */
	| FOR VALUES WITH '(' hash_partbound ')'
		{
			n := &ast.PartitionBoundSpec{
				BoundType: "MODULUS",
			}
			for _, item := range $5.Items {
				opt := item.(*ast.DefElem)
				switch opt.Defname {
				case "modulus":
					n.Options = "modulus " + strconv.FormatInt(opt.Arg.(*ast.Integer).Ival, 10)
				case "remainder":
					n.Options = "remainder " + strconv.FormatInt(opt.Arg.(*ast.Integer).Ival, 10)
				}
			}
			$$ = n
		}
	;

hash_partbound:
	hash_partbound_elem
		{
			$$ = makeList($1)
		}
	| hash_partbound ',' hash_partbound_elem
		{
			$$ = appendList($1, $3)
		}
	;

hash_partbound_elem:
	NonReservedWord Iconst
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
	columnOptions { $$ = $1 }
	| TableConstraint { $$ = $1 }
	;

columnOptions:
	ColId opt_column_constraints
		{
			$$ = &ast.ColumnDef{
				Name:    &ast.ColumnName{Name: model.NewCIStr($1)},
				Options: $2,
			}
		}
	| ColId WITH OPTIONS opt_column_constraints
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
	columnDef { $$ = $1 }
	| TableConstraint { $$ = $1 }
	| TableLikeClause { $$ = $1 }
	;

TableLikeClause:
	LIKE qualified_name
		{
			$$ = &ast.TableLikeClause{
				Relation: relationName(makeRangeVar($2)),
			}
		}
	| LIKE qualified_name TableLikeOptionList
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

columnDef:
	ColId Typename opt_column_constraints
		{
			$$ = &ast.ColumnDef{
				Name:    &ast.ColumnName{Name: model.NewCIStr($1)},
				Tp:      $2,
				Options: $3,
			}
		}
	;

opt_column_constraints:
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
	CONSTRAINT name ColConstraintElem
		{
			$$ = $3
		}
	| ColConstraintElem { $$ = $1 }
	| COLLATE any_name
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
	| OPTIONS '(' generic_option_list ')'
		{
			$$ = &ast.ColumnOption{
				Tp: ast.ColumnOptionNoOption,
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
	| UNIQUE opt_unique_null_treatment opt_definition OptConsTableSpace
		{
			$$ = &ast.ColumnOption{Tp: ast.ColumnOptionUniqKey}
		}
	| PRIMARY KEY opt_definition OptConsTableSpace
		{
			$$ = &ast.ColumnOption{Tp: ast.ColumnOptionPrimaryKey}
		}
	| CHECK '(' a_expr ')' no_inherit
		{
			$$ = &ast.ColumnOption{Tp: ast.ColumnOptionCheck, Expr: $3}
		}
	| DEFAULT b_expr
		{
			$$ = &ast.ColumnOption{Tp: ast.ColumnOptionDefaultValue, Expr: $2}
		}
	| REFERENCES qualified_name opt_column_list key_match key_actions ConstraintAttributeSpec
		{
			$$ = &ast.ColumnOption{
				Tp: ast.ColumnOptionReference,
				Refer: &ast.ReferenceDef{
					Table:         tableNameFromList($2),
					IndexColNames: makeIndexColNameList($3),
				},
			}
		}
	| GENERATED ALWAYS AS IDENTITY_P OptParenthesizedSeqOptList
		{
			$$ = &ast.ColumnOption{Tp: ast.ColumnOptionAutoIncrement}
		}
	| GENERATED BY DEFAULT AS IDENTITY_P OptParenthesizedSeqOptList
		{
			$$ = &ast.ColumnOption{Tp: ast.ColumnOptionAutoIncrement}
		}
	| GENERATED ALWAYS AS '(' a_expr ')' STORED
		{
			$$ = &ast.ColumnOption{Tp: ast.ColumnOptionGenerated, Expr: $5, Stored: true}
		}
	| GENERATED BY DEFAULT AS '(' a_expr ')' STORED
		{
			$$ = &ast.ColumnOption{Tp: ast.ColumnOptionGenerated, Expr: $6, Stored: true}
		}
	;

DomainConstraint:
	CONSTRAINT name DomainConstraintElem
		{
			n := $3.(*ast.Constraint)
			n.Name = $2
			$$ = n
		}
	| DomainConstraintElem
		{
			$$ = $1
		}
	;

DomainConstraintElem:
	CHECK '(' a_expr ')' ConstraintAttributeSpec
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
	CONSTRAINT name ConstraintElem
		{
			n := $3.(*ast.Constraint)
			n.Name = $2
			$$ = n
		}
	| ConstraintElem
		{
			$$ = $1
		}
	;

ConstraintElem:
	UNIQUE opt_unique_null_treatment '(' columnList ')' opt_c_include opt_definition OptConsTableSpace ConstraintAttributeSpec
		{
			n := &ast.Constraint{
				Tp:              ast.ConstraintUniq,
				Keys:            makeIndexColNameList($4),
				InitiallyValid:  true,
				NullsNotDistinct: $2,
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
	| PRIMARY KEY '(' columnList ')' opt_c_include opt_definition OptConsTableSpace ConstraintAttributeSpec
		{
			n := &ast.Constraint{
				Tp:             ast.ConstraintPrimaryKey,
				Keys:           makeIndexColNameList($4),
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
	| CHECK '(' a_expr ')' ConstraintAttributeSpec
		{
			n := &ast.Constraint{
				Tp:             ast.ConstraintCheck,
				Expr:           $3,
				InitiallyValid: true,
			}
			applyConstraintAttrs(n, $5)
			$$ = n
		}
	| FOREIGN KEY '(' columnList ')' REFERENCES qualified_name opt_column_list key_match key_actions ConstraintAttributeSpec
		{
			n := &ast.Constraint{
				Tp:             ast.ConstraintForeignKey,
				Keys:           makeIndexColNameList($4),
				InitiallyValid: true,
				Refer: &ast.ReferenceDef{
					Table:         tableNameFromList($7),
					IndexColNames: makeIndexColNameList($8),
				},
			}
			applyConstraintAttrs(n, $11)
			$$ = n
		}
	| EXCLUDE access_method_clause '(' ExclusionConstraintList ')' opt_c_include opt_definition OptConsTableSpace where_clause ConstraintAttributeSpec
		{
			n := &ast.Constraint{
				Tp:             ast.ConstraintUniq,
				InitiallyValid: true,
			}
			applyConstraintAttrs(n, $10)
			$$ = n
		}
	;

opt_column_list:
	'(' columnList ')' { $$ = $2 }
	| /* EMPTY */ { $$ = nil }
	;

columnList:
	name
		{ $$ = makeList(&ast.String{Str: $1}) }
	| columnList ',' name
		{ $$ = appendList($1, &ast.String{Str: $3}) }
	;

key_match:
	MATCH FULL     { $$ = int64('f') }
	| MATCH PARTIAL
		{
			/* PARTIAL is not implemented */
			$$ = int64('p')
		}
	| MATCH SIMPLE { $$ = int64('s') }
	| /* EMPTY */  { $$ = int64('s') }
	;

/*
 * key_actions packs update and delete actions into one int64:
 * high byte = update action, low byte = delete action
 */
key_actions:
	key_update
		{ $$ = &KeyActions{UpdateAction: $1, DeleteAction: &KeyAction{Action: 'a'}} }
	| key_delete
		{ $$ = &KeyActions{UpdateAction: &KeyAction{Action: 'a'}, DeleteAction: $1} }
	| key_update key_delete
		{ $$ = &KeyActions{UpdateAction: $1, DeleteAction: $2} }
	| key_delete key_update
		{ $$ = &KeyActions{UpdateAction: $2, DeleteAction: $1} }
	| /* EMPTY */
		{ $$ = &KeyActions{UpdateAction: &KeyAction{Action: 'a'}, DeleteAction: &KeyAction{Action: 'a'}} }
	;

key_update:
	ON UPDATE key_action { $$ = $3 }
	;

key_delete:
	ON DELETE_P key_action { $$ = $3 }
	;

key_action:
	NO ACTION                     { $$ = &KeyAction{Action: 'a'} }
	| RESTRICT                    { $$ = &KeyAction{Action: 'r'} }
	| CASCADE                     { $$ = &KeyAction{Action: 'c'} }
	| SET NULL_P opt_column_list  { $$ = &KeyAction{Action: 'n', Cols: $3} }
	| SET DEFAULT opt_column_list { $$ = &KeyAction{Action: 'd', Cols: $3} }
	;

opt_c_include:
	INCLUDE '(' columnList ')'  { $$ = $3 }
	| /* EMPTY */               { $$ = nil }
	;

OptConsTableSpace:
	USING INDEX TABLESPACE name { $$ = $4 }
	| /* EMPTY */               { $$ = "" }
	;

ExistingIndex:
	USING INDEX name { $$ = $3 }
	;

ExclusionConstraintList:
	ExclusionConstraintElem
		{ $$ = makeList($1) }
	| ExclusionConstraintList ',' ExclusionConstraintElem
		{ $$ = appendList($1, $3) }
	;

ExclusionConstraintElem:
	index_elem WITH any_operator
		{
			$$ = &ast.List{Items: []ast.Node{$1, $3}}
		}
	| index_elem WITH OPERATOR '(' any_operator ')'
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
	ALTER TABLE relation_expr alter_table_cmds
		{
			$$ = &ast.AlterTableStmt{
				Table: relationName($3),
				Specs: alterTableSpecs($4),
			}
		}
	| ALTER TABLE IF_P EXISTS relation_expr alter_table_cmds
		{
			$$ = &ast.AlterTableStmt{
				Table: relationName($5),
				Specs: alterTableSpecs($6),
				MissingOk: true,
			}
		}
	| ALTER INDEX qualified_name alter_table_cmds
		{
			rv := makeRangeVarFromAnyName($3)
			$$ = &ast.AlterTableStmt{
				Table: relationName(rv),
				Specs: alterTableSpecs($4),
			}
		}
	| ALTER INDEX IF_P EXISTS qualified_name alter_table_cmds
		{
			rv := makeRangeVarFromAnyName($5)
			$$ = &ast.AlterTableStmt{
				Table: relationName(rv),
				Specs: alterTableSpecs($6),
				MissingOk: true,
			}
		}
	| ALTER INDEX qualified_name ATTACH PARTITION qualified_name
		{
			rvIdx := makeRangeVarFromAnyName($3)
			rvPart := makeRangeVar($6)
			cmd := &ast.AlterTableCmd{
				Subtype: int(ast.AT_AttachPartition),
				Def:     &ast.PartitionCmd{
					Name: relationName(rvPart),
				},
			}
			$$ = &ast.AlterTableStmt{
				Table: relationName(rvIdx),
				Specs: alterTableSpecs(makeList(cmd)),
			}
		}
	| ALTER SEQUENCE qualified_name alter_table_cmds
		{
			rv := makeRangeVarFromAnyName($3)
			$$ = &ast.AlterTableStmt{
				Table: relationName(rv),
				Specs: alterTableSpecs($4),
			}
		}
	| ALTER SEQUENCE IF_P EXISTS qualified_name alter_table_cmds
		{
			rv := makeRangeVarFromAnyName($5)
			$$ = &ast.AlterTableStmt{
				Table: relationName(rv),
				Specs: alterTableSpecs($6),
				MissingOk: true,
			}
		}
	| ALTER VIEW qualified_name alter_table_cmds
		{
			rv := makeRangeVarFromAnyName($3)
			$$ = &ast.AlterTableStmt{
				Table: relationName(rv),
				Specs: alterTableSpecs($4),
			}
		}
	| ALTER VIEW IF_P EXISTS qualified_name alter_table_cmds
		{
			rv := makeRangeVarFromAnyName($5)
			$$ = &ast.AlterTableStmt{
				Table: relationName(rv),
				Specs: alterTableSpecs($6),
				MissingOk: true,
			}
		}
	| ALTER MATERIALIZED VIEW qualified_name alter_table_cmds
		{
			rv := makeRangeVarFromAnyName($4)
			$$ = &ast.AlterTableStmt{
				Table: relationName(rv),
				Specs: alterTableSpecs($5),
			}
		}
	| ALTER MATERIALIZED VIEW IF_P EXISTS qualified_name alter_table_cmds
		{
			rv := makeRangeVarFromAnyName($6)
			$$ = &ast.AlterTableStmt{
				Table: relationName(rv),
				Specs: alterTableSpecs($7),
				MissingOk: true,
			}
		}
	| ALTER FOREIGN TABLE relation_expr alter_table_cmds
		{
			$$ = &ast.AlterTableStmt{
				Table: relationName($4),
				Specs: alterTableSpecs($5),
			}
		}
	| ALTER FOREIGN TABLE IF_P EXISTS relation_expr alter_table_cmds
		{
			$$ = &ast.AlterTableStmt{
				Table: relationName($6),
				Specs: alterTableSpecs($7),
				MissingOk: true,
			}
		}
	/* ALTER TABLE ALL IN TABLESPACE ... SET TABLESPACE ... */
	| ALTER TABLE ALL IN_P TABLESPACE name SET TABLESPACE name opt_nowait
		{
			$$ = &ast.AlterTableMoveAllStmt{
				OrigTablespacename: $6,
				NewTablespacename:  $9,
				Nowait:             $10,
			}
		}
	| ALTER TABLE ALL IN_P TABLESPACE name OWNED BY role_list SET TABLESPACE name opt_nowait
		{
			$$ = &ast.AlterTableMoveAllStmt{
				OrigTablespacename: $6,
				Roles:              $9,
				NewTablespacename:  $12,
				Nowait:             $13,
			}
		}
	| ALTER INDEX ALL IN_P TABLESPACE name SET TABLESPACE name opt_nowait
		{
			$$ = &ast.AlterTableMoveAllStmt{
				OrigTablespacename: $6,
				NewTablespacename:  $9,
				Nowait:             $10,
			}
		}
	| ALTER INDEX ALL IN_P TABLESPACE name OWNED BY role_list SET TABLESPACE name opt_nowait
		{
			$$ = &ast.AlterTableMoveAllStmt{
				OrigTablespacename: $6,
				Roles:              $9,
				NewTablespacename:  $12,
				Nowait:             $13,
			}
		}
	| ALTER MATERIALIZED VIEW ALL IN_P TABLESPACE name SET TABLESPACE name opt_nowait
		{
			$$ = &ast.AlterTableMoveAllStmt{
				OrigTablespacename: $7,
				NewTablespacename:  $10,
				Nowait:             $11,
			}
		}
	| ALTER MATERIALIZED VIEW ALL IN_P TABLESPACE name OWNED BY role_list SET TABLESPACE name opt_nowait
		{
			$$ = &ast.AlterTableMoveAllStmt{
				OrigTablespacename: $7,
				Roles:              $10,
				NewTablespacename:  $13,
				Nowait:             $14,
			}
		}
	;

alter_table_cmds:
	alter_table_cmd
		{ $$ = makeList($1) }
	| alter_table_cmds ',' alter_table_cmd
		{ $$ = appendList($1, $3) }
	;

alter_table_cmd:
	ADD_P columnDef
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableAddColumns,
				NewColumns: []*ast.ColumnDef{$2.(*ast.ColumnDef)},
			}
		}
	| ADD_P COLUMN columnDef
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableAddColumns,
				NewColumns: []*ast.ColumnDef{$3.(*ast.ColumnDef)},
			}
		}
	| ADD_P COLUMN IF_P NOT EXISTS columnDef
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableAddColumns,
				NewColumns: []*ast.ColumnDef{$6.(*ast.ColumnDef)},
				MissingOk: true,
			}
		}
	| DROP COLUMN ColId opt_drop_behavior
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableDropColumn,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				Behavior: ast.DropBehavior(int($4)),
			}
		}
	| DROP COLUMN IF_P EXISTS ColId opt_drop_behavior
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableDropColumn,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($5)},
				Behavior: ast.DropBehavior(int($6)),
				MissingOk: true,
			}
		}
	| ALTER COLUMN ColId SET DEFAULT a_expr
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
	| ALTER COLUMN ColId TYPE_P Typename opt_collate_clause
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
	| DROP CONSTRAINT name opt_drop_behavior
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
				Name: $3,
				Behavior: ast.DropBehavior(int($4)),
			}
		}
	| DROP CONSTRAINT IF_P EXISTS name opt_drop_behavior
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
				Owner: roleSpecName($3),
			}
		}
	/* ALTER COLUMN ... TYPE ... USING */
	| ALTER COLUMN ColId SET DATA_P TYPE_P Typename opt_collate_clause
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
	| ALTER COLUMN ColId TYPE_P Typename opt_collate_clause USING a_expr
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
	| ALTER COLUMN ColId SET DATA_P TYPE_P Typename opt_collate_clause USING a_expr
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
	| ALTER ColId TYPE_P Typename opt_collate_clause
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
	| ALTER ColId SET DATA_P TYPE_P Typename opt_collate_clause
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
	| ALTER ColId SET DATA_P TYPE_P Typename opt_collate_clause USING a_expr
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
	| ALTER ColId SET DEFAULT a_expr
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
	| ALTER ColId TYPE_P Typename opt_collate_clause USING a_expr
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
	/* ALTER FOREIGN TABLE <name> ALTER [COLUMN] <colname> OPTIONS */
	| ALTER COLUMN ColId alter_generic_options
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableAlterColumn,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				Def: $4,
			}
		}
	| ALTER ColId alter_generic_options
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableAlterColumn,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($2)},
				Def: $3,
			}
		}
	/* VALIDATE CONSTRAINT */
	| VALIDATE CONSTRAINT name
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableValidateConstraint,
				Name: $3,
			}
		}
	/* ALTER CONSTRAINT */
	| ALTER CONSTRAINT name ConstraintAttributeSpec
		{
			$$ = &ast.AlterTableSpec{
				Tp:               ast.AlterTableAlterConstraint,
				Name:             $3,
				ConstraintOptions: constraintOptions($4),
			}
		}
	/* INHERIT / NO INHERIT */
	| INHERIT qualified_name
		{
			$$ = &ast.AlterTableSpec{
				Tp:      ast.AlterTableInherit,
				NewTable: tableNameFromList($2),
			}
		}
	| NO INHERIT qualified_name
		{
			$$ = &ast.AlterTableSpec{
				Tp:      ast.AlterTableNoInherit,
				NewTable: tableNameFromList($3),
			}
		}
	/* ATTACH / DETACH PARTITION */
	| ATTACH PARTITION qualified_name ForValues
		{
			$$ = &ast.AlterTableSpec{
				Tp:            ast.AlterTableAttachPartition,
				NewTable:      tableNameFromList($3),
				PartitionBound: $4,
			}
		}
	| DETACH PARTITION qualified_name
		{
			$$ = &ast.AlterTableSpec{
				Tp:       ast.AlterTableDetachPartition,
				NewTable: tableNameFromList($3),
			}
		}
	| DETACH PARTITION qualified_name CONCURRENTLY
		{
			$$ = &ast.AlterTableSpec{
				Tp:       ast.AlterTableDetachPartition,
				NewTable: tableNameFromList($3),
			}
		}
	| DETACH PARTITION qualified_name FINALIZE
		{
			$$ = &ast.AlterTableSpec{
				Tp:       ast.AlterTableDetachPartition,
				NewTable: tableNameFromList($3),
			}
		}
	/* ENABLE/DISABLE TRIGGER */
	| ENABLE_P TRIGGER name
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableEnableTrigger,
				Name: $3,
				TriggerMode: "DEFAULT",
			}
		}
	| ENABLE_P ALWAYS TRIGGER name
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableEnableTrigger,
				Name: $4,
				TriggerMode: "ALWAYS",
			}
		}
	| ENABLE_P REPLICA TRIGGER name
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableEnableTrigger,
				Name: $4,
				TriggerMode: "REPLICA",
			}
		}
	| DISABLE_P TRIGGER name
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
	| ENABLE_P RULE name
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableEnableRule,
				Name: $3,
				TriggerMode: "RULE",
			}
		}
	| ENABLE_P ALWAYS RULE name
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableEnableRule,
				Name: $4,
				TriggerMode: "RULE",
			}
		}
	| ENABLE_P REPLICA RULE name
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableEnableRule,
				Name: $4,
				TriggerMode: "RULE",
			}
		}
	| DISABLE_P RULE name
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
	| CLUSTER ON name
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
	| SET ACCESS METHOD name
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
	| SET reloptions
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableSetRelOptions,
				RelOptions: relOptionString($2),
			}
		}
	| RESET reloptions
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableResetRelOptions,
				RelOptions: relOptionString($2),
			}
		}
	/* ADD COLUMN IF NOT EXISTS (without COLUMN keyword) */
	| ADD_P IF_P NOT EXISTS columnDef
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableAddColumns,
				NewColumns: []*ast.ColumnDef{$5.(*ast.ColumnDef)},
				MissingOk: true,
			}
		}
	/* DROP (without COLUMN keyword) */
	| DROP ColId opt_drop_behavior
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableDropColumn,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($2)},
				Behavior: ast.DropBehavior(int($3)),
			}
		}
	| DROP IF_P EXISTS ColId opt_drop_behavior
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

				Tp: ast.AlterTableGeneric,
			}
		}
	| REPLICA IDENTITY_P FULL
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
			}
		}
	| REPLICA IDENTITY_P NOTHING
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
			}
		}
	| REPLICA IDENTITY_P USING INDEX name
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableReplicaIdentity,
				Name: $5,
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
				Def: &ast.String{Str: "default"},
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
				Def: &ast.String{Str: "default"},
			}
		}
	/* SET STATISTICS */
	| ALTER COLUMN ColId SET STATISTICS SignedIconst
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableSetStatistics,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				Num: uint64(int16($6)),
			}
		}
	| ALTER ColId SET STATISTICS SignedIconst
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableSetStatistics,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($2)},
				Num: uint64(int16($5)),
			}
		}
	| ALTER COLUMN Iconst SET STATISTICS SignedIconst
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableSetStatistics,
				Num: uint64(int16($3)),
				Def: makeIntConst($6),
			}
		}
	| ALTER Iconst SET STATISTICS SignedIconst
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableSetStatistics,
				Num: uint64(int16($2)),
				Def: makeIntConst($5),
			}
		}
	/* SET COMPRESSION */
	| ALTER COLUMN ColId SET column_compression
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				Def: &ast.String{Str: $5},
			}
		}
	| ALTER ColId SET column_compression
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($2)},
				Def: &ast.String{Str: $4},
			}
		}
	/* SET/DROP EXPRESSION */
	| ALTER COLUMN ColId SET EXPRESSION AS '(' a_expr ')'
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				Def: $8,
			}
		}
	| ALTER ColId SET EXPRESSION AS '(' a_expr ')'
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
	| ALTER COLUMN ColId ADD_P GENERATED ALWAYS AS IDENTITY_P '(' OptSeqOptList ')'
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
	| ALTER COLUMN ColId ADD_P GENERATED BY DEFAULT AS IDENTITY_P '(' OptSeqOptList ')'
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
	| ALTER COLUMN ColId ADD_P GENERATED generated_when AS IDENTITY_P OptParenthesizedSeqOptList
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
	| ALTER ColId ADD_P GENERATED generated_when AS IDENTITY_P OptParenthesizedSeqOptList
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
	| ALTER COLUMN ColId alter_identity_column_option_list
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				Def: $4,
			}
		}
	/* ALTER COLUMN ... SET (options) / RESET (options) */
	| ALTER COLUMN ColId SET '(' def_list ')'
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableSetRelOptions,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				Def: $6,
			}
		}
	| ALTER COLUMN ColId RESET '(' def_list ')'
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableResetRelOptions,
				OldColumnName: &ast.ColumnName{Name: model.NewCIStr($3)},
				RelOptions: relOptionString($6),
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
	| SET TABLESPACE name
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableSetTablespace,
				Name: $3,
			}
		}
	/* OPTIONS for foreign tables */
	| alter_generic_options
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableGeneric,
				Def: $1,
			}
		}
	/* OF typename */
	| OF any_name
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

opt_drop_behavior:
	CASCADE     { $$ = int64(ast.DROP_CASCADE) }
	| RESTRICT  { $$ = int64(ast.DROP_RESTRICT) }
	| /* EMPTY */ { $$ = int64(ast.DROP_RESTRICT) }
	;

column_compression:
	COMPRESSION ColId      { $$ = $2 }
	| COMPRESSION DEFAULT  { $$ = "default" }
	;

/*****************************************************************************
 *
 *      ALTER TABLE ... RENAME
 *
 *****************************************************************************/

RenameStmt:
	ALTER TABLE relation_expr RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_TABLE,
				Relation:   relationName($3),
				Newname:    $6,
			}
		}
	| ALTER TABLE IF_P EXISTS relation_expr RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_TABLE,
				Relation:   relationName($5),
				Newname:    $8,
				MissingOk:  true,
			}
		}
	| ALTER TABLE relation_expr RENAME COLUMN ColId TO name
		{
			$$ = &ast.RenameStmt{
				RenameType:   ast.OBJECT_COLUMN,
				RelationType: ast.OBJECT_TABLE,
				Relation:     relationName($3),
				Subname:      $6,
				Newname:      $8,
			}
		}
	| ALTER TABLE relation_expr RENAME ColId TO name
		{
			$$ = &ast.RenameStmt{
				RenameType:   ast.OBJECT_COLUMN,
				RelationType: ast.OBJECT_TABLE,
				Relation:     relationName($3),
				Subname:      $5,
				Newname:      $7,
			}
		}
	| ALTER TABLE relation_expr RENAME CONSTRAINT name TO name
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
	| ALTER TABLE IF_P EXISTS relation_expr RENAME COLUMN ColId TO name
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
	| ALTER TABLE IF_P EXISTS relation_expr RENAME ColId TO name
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
	| ALTER TABLE IF_P EXISTS relation_expr RENAME CONSTRAINT name TO name
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
	| ALTER INDEX qualified_name RENAME TO name
		{
			rv := makeRangeVarFromAnyName($3)
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_INDEX,
				Relation:   relationName(rv),
				Newname:    $6,
			}
		}
	| ALTER INDEX IF_P EXISTS qualified_name RENAME TO name
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
	| ALTER SEQUENCE qualified_name RENAME TO name
		{
			rv := makeRangeVarFromAnyName($3)
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_SEQUENCE,
				Relation:   relationName(rv),
				Newname:    $6,
			}
		}
	| ALTER SEQUENCE IF_P EXISTS qualified_name RENAME TO name
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
	| ALTER VIEW qualified_name RENAME TO name
		{
			rv := makeRangeVarFromAnyName($3)
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_VIEW,
				Relation:   relationName(rv),
				Newname:    $6,
			}
		}
	| ALTER VIEW IF_P EXISTS qualified_name RENAME TO name
		{
			rv := makeRangeVarFromAnyName($5)
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_VIEW,
				Relation:   relationName(rv),
				Newname:    $8,
				MissingOk:  true,
			}
		}
	| ALTER VIEW qualified_name RENAME COLUMN ColId TO name
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
	| ALTER VIEW qualified_name RENAME ColId TO name
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
	| ALTER MATERIALIZED VIEW qualified_name RENAME TO name
		{
			rv := makeRangeVarFromAnyName($4)
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_MATVIEW,
				Relation:   relationName(rv),
				Newname:    $7,
			}
		}
	| ALTER MATERIALIZED VIEW IF_P EXISTS qualified_name RENAME TO name
		{
			rv := makeRangeVarFromAnyName($6)
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_MATVIEW,
				Relation:   relationName(rv),
				Newname:    $9,
				MissingOk:  true,
			}
		}
	| ALTER MATERIALIZED VIEW qualified_name RENAME COLUMN ColId TO name
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
	| ALTER MATERIALIZED VIEW qualified_name RENAME ColId TO name
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
	| ALTER FUNCTION function_with_argtypes RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_FUNCTION,
				Object:     $3,
				Newname:    $6,
			}
		}
	| ALTER PROCEDURE function_with_argtypes RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_PROCEDURE,
				Object:     $3,
				Newname:    $6,
			}
		}
	| ALTER ROUTINE function_with_argtypes RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_ROUTINE,
				Object:     $3,
				Newname:    $6,
			}
		}
	/* RENAME AGGREGATE */
	| ALTER AGGREGATE aggregate_with_argtypes RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_AGGREGATE,
				Object:     $3,
				Newname:    $6,
			}
		}
	/* RENAME COLLATION */
	| ALTER COLLATION any_name RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_COLLATION,
				Object:     $3,
				Newname:    $6,
			}
		}
	/* RENAME CONVERSION */
	| ALTER CONVERSION_P any_name RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_CONVERSION,
				Object:     $3,
				Newname:    $6,
			}
		}
	/* RENAME DOMAIN */
	| ALTER DOMAIN_P any_name RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_DOMAIN,
				Object:     $3,
				Newname:    $6,
			}
		}
	| ALTER DOMAIN_P any_name RENAME CONSTRAINT name TO name
		{
			$$ = &ast.RenameStmt{
				RenameType:   ast.OBJECT_DOMCONSTRAINT,
				Object:       $3,
				Subname:      $6,
				Newname:      $8,
			}
		}
	/* RENAME SCHEMA */
	| ALTER SCHEMA name RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_SCHEMA,
				Subname:    $3,
				Newname:    $6,
			}
		}
	/* RENAME SERVER */
	| ALTER SERVER name RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_FOREIGN_SERVER,
				Object:     &ast.String{Str: $3},
				Newname:    $6,
			}
		}
	/* RENAME FOREIGN DATA WRAPPER */
	| ALTER FOREIGN DATA_P WRAPPER name RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_FDW,
				Object:     &ast.String{Str: $5},
				Newname:    $8,
			}
		}
	/* RENAME TYPE */
	| ALTER TYPE_P any_name RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_TYPE,
				Object:     $3,
				Newname:    $6,
			}
		}
	| ALTER TYPE_P any_name RENAME ATTRIBUTE name TO name opt_drop_behavior
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
	| ALTER FOREIGN TABLE relation_expr RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_FOREIGN_TABLE,
				Relation:   relationName($4),
				Newname:    $7,
			}
		}
	| ALTER FOREIGN TABLE IF_P EXISTS relation_expr RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_FOREIGN_TABLE,
				Relation:   relationName($6),
				Newname:    $9,
				MissingOk:  true,
			}
		}
	| ALTER FOREIGN TABLE relation_expr RENAME COLUMN ColId TO name
		{
			$$ = &ast.RenameStmt{
				RenameType:   ast.OBJECT_COLUMN,
				RelationType: ast.OBJECT_FOREIGN_TABLE,
				Relation:     relationName($4),
				Subname:      $7,
				Newname:      $9,
			}
		}
	| ALTER FOREIGN TABLE relation_expr RENAME ColId TO name
		{
			$$ = &ast.RenameStmt{
				RenameType:   ast.OBJECT_COLUMN,
				RelationType: ast.OBJECT_FOREIGN_TABLE,
				Relation:     relationName($4),
				Subname:      $6,
				Newname:      $8,
			}
		}
	| ALTER FOREIGN TABLE IF_P EXISTS relation_expr RENAME COLUMN ColId TO name
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
	| ALTER FOREIGN TABLE IF_P EXISTS relation_expr RENAME ColId TO name
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
	| ALTER OPERATOR CLASS any_name USING name RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_OPCLASS,
				Object:     prependList(&ast.String{Str: $6}, $4),
				Newname:    $9,
			}
		}
	| ALTER OPERATOR FAMILY any_name USING name RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_OPFAMILY,
				Object:     prependList(&ast.String{Str: $6}, $4),
				Newname:    $9,
			}
		}
	/* RENAME TEXT SEARCH */
	| ALTER TEXT_P SEARCH PARSER any_name RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_TSPARSER,
				Object:     $5,
				Newname:    $8,
			}
		}
	| ALTER TEXT_P SEARCH DICTIONARY any_name RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_TSDICTIONARY,
				Object:     $5,
				Newname:    $8,
			}
		}
	| ALTER TEXT_P SEARCH TEMPLATE any_name RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_TSTEMPLATE,
				Object:     $5,
				Newname:    $8,
			}
		}
	| ALTER TEXT_P SEARCH CONFIGURATION any_name RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_TSCONFIGURATION,
				Object:     $5,
				Newname:    $8,
			}
		}
	/* RENAME PUBLICATION / SUBSCRIPTION */
	| ALTER PUBLICATION name RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_PUBLICATION,
				Object:     &ast.String{Str: $3},
				Newname:    $6,
			}
		}
	| ALTER SUBSCRIPTION name RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_SUBSCRIPTION,
				Object:     &ast.String{Str: $3},
				Newname:    $6,
			}
		}
	/* RENAME RULE */
	| ALTER RULE name ON qualified_name RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_RULE,
				Relation:   relationName(makeRangeVarFromAnyName($5)),
				Subname:    $3,
				Newname:    $8,
			}
		}
	/* RENAME TRIGGER */
	| ALTER TRIGGER name ON qualified_name RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_TRIGGER,
				Relation:   relationName(makeRangeVarFromAnyName($5)),
				Subname:    $3,
				Newname:    $8,
			}
		}
	/* RENAME EVENT TRIGGER */
	| ALTER EVENT TRIGGER name RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_EVENT_TRIGGER,
				Object:     &ast.String{Str: $4},
				Newname:    $7,
			}
		}
	/* RENAME STATISTICS */
	| ALTER STATISTICS any_name RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_STATISTIC_EXT,
				Object:     $3,
				Newname:    $6,
			}
		}
	/* RENAME POLICY */
	| ALTER POLICY name ON qualified_name RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_POLICY,
				Relation:   relationName(makeRangeVarFromAnyName($5)),
				Subname:    $3,
				Newname:    $8,
			}
		}
	/* RENAME LANGUAGE / DATABASE / TABLESPACE / ROLE */
	| ALTER opt_procedural LANGUAGE name RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_LANGUAGE,
				Object:     &ast.String{Str: $4},
				Newname:    $7,
			}
		}
	| ALTER DATABASE name RENAME TO name
		{
			$$ = &ast.AlterDatabaseStmt{
				Name:     $3,
				RenameTo: $6,
			}
		}
	| ALTER TABLESPACE name RENAME TO name
		{
			$$ = &ast.AlterTablespaceStmt{
				Name:     $3,
				RenameTo: $6,
			}
		}
	| ALTER ROLE RoleSpec RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_ROLE,
				Subname:    $3.(*ast.RoleSpec).Rolename,
				Newname:    $6,
			}
		}
	| ALTER USER RoleSpec RENAME TO name
		{
			$$ = &ast.RenameStmt{
				RenameType: ast.OBJECT_ROLE,
				Subname:    $3.(*ast.RoleSpec).Rolename,
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
	DROP object_type_any_name IF_P EXISTS any_name_list opt_drop_behavior
		{
			$$ = dropObjectStmt(int($2), tableNamesFromNameList($5), true, int($6))
		}
	| DROP object_type_any_name any_name_list opt_drop_behavior
		{
			$$ = dropObjectStmt(int($2), tableNamesFromNameList($3), false, int($4))
		}
	/* DROP INDEX */
	| DROP INDEX CONCURRENTLY IF_P EXISTS name
		{
			$$ = &ast.DropIndexStmt{
				IndexName:    $6,
				Concurrently: true,
				IfExists:     true,
			}
		}
	| DROP INDEX CONCURRENTLY name
		{
			$$ = &ast.DropIndexStmt{
				IndexName:    $4,
				Concurrently: true,
			}
		}
	/* DROP TYPE / DROP DOMAIN */
	| DROP TYPE_P any_name_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($3),
				RemoveType: int(ast.OBJECT_TYPE),
				Behavior:   int($4),
			}
		}
	| DROP TYPE_P IF_P EXISTS any_name_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($5),
				RemoveType: int(ast.OBJECT_TYPE),
				Behavior:   int($6),
				Missing_ok: true,
			}
		}
	| DROP DOMAIN_P any_name_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($3),
				RemoveType: int(ast.OBJECT_DOMAIN),
				Behavior:   int($4),
			}
		}
	| DROP DOMAIN_P IF_P EXISTS any_name_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($5),
				RemoveType: int(ast.OBJECT_DOMAIN),
				Behavior:   int($6),
				Missing_ok: true,
			}
		}
	/* DROP SCHEMA / EXTENSION / LANGUAGE / PUBLICATION / SUBSCRIPTION */
	| DROP SCHEMA name_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($3),
				RemoveType: int(ast.OBJECT_SCHEMA),
				Behavior:   int($4),
			}
		}
	| DROP SCHEMA IF_P EXISTS name_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($5),
				RemoveType: int(ast.OBJECT_SCHEMA),
				Behavior:   int($6),
				Missing_ok: true,
			}
		}
	| DROP EXTENSION name_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($3),
				RemoveType: int(ast.OBJECT_EXTENSION),
				Behavior:   int($4),
			}
		}
	| DROP EXTENSION IF_P EXISTS name_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($5),
				RemoveType: int(ast.OBJECT_EXTENSION),
				Behavior:   int($6),
				Missing_ok: true,
			}
		}
	/* DROP FUNCTION / PROCEDURE / ROUTINE / AGGREGATE */
	| DROP FUNCTION any_name_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($3),
				RemoveType: int(ast.OBJECT_FUNCTION),
				Behavior:   int($4),
			}
		}
	| DROP FUNCTION IF_P EXISTS any_name_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($5),
				RemoveType: int(ast.OBJECT_FUNCTION),
				Behavior:   int($6),
				Missing_ok: true,
			}
		}
	| DROP PROCEDURE any_name_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($3),
				RemoveType: int(ast.OBJECT_PROCEDURE),
				Behavior:   int($4),
			}
		}
	| DROP PROCEDURE IF_P EXISTS any_name_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($5),
				RemoveType: int(ast.OBJECT_PROCEDURE),
				Behavior:   int($6),
				Missing_ok: true,
			}
		}
	| DROP ROUTINE any_name_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($3),
				RemoveType: int(ast.OBJECT_ROUTINE),
				Behavior:   int($4),
			}
		}
	| DROP ROUTINE IF_P EXISTS any_name_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($5),
				RemoveType: int(ast.OBJECT_ROUTINE),
				Behavior:   int($6),
				Missing_ok: true,
			}
		}
	| DROP AGGREGATE any_name_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($3),
				RemoveType: int(ast.OBJECT_AGGREGATE),
				Behavior:   int($4),
			}
		}
	| DROP AGGREGATE IF_P EXISTS any_name_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($5),
				RemoveType: int(ast.OBJECT_AGGREGATE),
				Behavior:   int($6),
				Missing_ok: true,
			}
		}
	/* DROP TRIGGER / POLICY / RULE ON relation */
	| DROP TRIGGER name ON any_name opt_drop_behavior
		{
			$$ = dropTriggerStmt(false, $3, $5)
		}
	| DROP TRIGGER IF_P EXISTS name ON any_name opt_drop_behavior
		{
			$$ = dropTriggerStmt(true, $5, $7)
		}
	| DROP POLICY name ON any_name opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList(&ast.List{Items: []ast.Node{$5}}),
				RemoveType: int(ast.OBJECT_POLICY),
				Behavior:   int($6),
			}
		}
	| DROP POLICY IF_P EXISTS name ON any_name opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList(&ast.List{Items: []ast.Node{$7}}),
				RemoveType: int(ast.OBJECT_POLICY),
				Behavior:   int($8),
				Missing_ok: true,
			}
		}
	| DROP RULE name ON any_name opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList(&ast.List{Items: []ast.Node{$5}}),
				RemoveType: int(ast.OBJECT_RULE),
				Behavior:   int($6),
			}
		}
	| DROP RULE IF_P EXISTS name ON any_name opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList(&ast.List{Items: []ast.Node{$7}}),
				RemoveType: int(ast.OBJECT_RULE),
				Behavior:   int($8),
				Missing_ok: true,
			}
		}
	/* DROP EVENT TRIGGER */
	| DROP EVENT TRIGGER name opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList(&ast.List{Items: []ast.Node{&ast.String{Str: $4}}}),
				RemoveType: int(ast.OBJECT_EVENT_TRIGGER),
				Behavior:   int($5),
			}
		}
	| DROP EVENT TRIGGER IF_P EXISTS name opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList(&ast.List{Items: []ast.Node{&ast.String{Str: $6}}}),
				RemoveType: int(ast.OBJECT_EVENT_TRIGGER),
				Behavior:   int($7),
				Missing_ok: true,
			}
		}
	/* DROP LANGUAGE / PUBLICATION / SUBSCRIPTION */
	| DROP LANGUAGE name opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList(&ast.List{Items: []ast.Node{&ast.String{Str: $3}}}),
				RemoveType: int(ast.OBJECT_LANGUAGE),
				Behavior:   int($4),
			}
		}
	| DROP LANGUAGE IF_P EXISTS name opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList(&ast.List{Items: []ast.Node{&ast.String{Str: $5}}}),
				RemoveType: int(ast.OBJECT_LANGUAGE),
				Behavior:   int($6),
				Missing_ok: true,
			}
		}
	| DROP PUBLICATION name_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($3),
				RemoveType: int(ast.OBJECT_PUBLICATION),
				Behavior:   int($4),
			}
		}
	| DROP PUBLICATION IF_P EXISTS name_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($5),
				RemoveType: int(ast.OBJECT_PUBLICATION),
				Behavior:   int($6),
				Missing_ok: true,
			}
		}
	| DROP SUBSCRIPTION name_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($3),
				RemoveType: int(ast.OBJECT_SUBSCRIPTION),
				Behavior:   int($4),
			}
		}
	| DROP SUBSCRIPTION IF_P EXISTS name_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($5),
				RemoveType: int(ast.OBJECT_SUBSCRIPTION),
				Behavior:   int($6),
				Missing_ok: true,
			}
		}
	/* DROP INDEX CONCURRENTLY */
	| DROP INDEX CONCURRENTLY any_name_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($4),
				RemoveType: int(ast.OBJECT_INDEX),
				Behavior:   int($5),
				Concurrent: true,
			}
		}
	| DROP INDEX CONCURRENTLY IF_P EXISTS any_name_list opt_drop_behavior
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
	| DROP FOREIGN DATA_P WRAPPER name_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($5),
				RemoveType: int(ast.OBJECT_FDW),
				Behavior:   int($6),
			}
		}
	| DROP FOREIGN DATA_P WRAPPER IF_P EXISTS name_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($7),
				RemoveType: int(ast.OBJECT_FDW),
				Behavior:   int($8),
				Missing_ok: true,
			}
		}
	/* DROP OWNED BY */
	| DROP OWNED BY name_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList($4),
				RemoveType: int(ast.OBJECT_ROLE),
				Behavior:   int($5),
			}
		}
	;

object_type_any_name:
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

any_name_list:
	any_name
		{
			$$ = &ast.List{Items: []ast.Node{$1}}
		}
	| any_name_list ',' any_name
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
	CREATE opt_unique INDEX opt_concurrently IF_P NOT EXISTS opt_index_name
	ON relation_expr access_method_clause '(' index_params ')' opt_unique_null_treatment opt_include opt_reloptions OptTableSpace where_clause
		{
			$$ = &ast.CreateIndexStmt{
				IndexName:     indexNameFromList($8),
				Table:         relationName($10),
				Unique:        $2,
				IfNotExists:   true,
				Concurrently:  $4,
				IndexColNames: indexElemColNames($13),
				Include:       indexIncludeNames($16),
				Using:         accessMethodName($11),
				WithOptions:   relOptionString($17),
				Tablespace:    $18,
				Where:         exprOrNil($19),
			}
		}
	| CREATE opt_unique INDEX opt_concurrently opt_index_name
	ON relation_expr access_method_clause '(' index_params ')' opt_unique_null_treatment opt_include opt_reloptions OptTableSpace where_clause
		{
			$$ = &ast.CreateIndexStmt{
				IndexName:     indexNameFromList($5),
				Table:         relationName($7),
				Unique:        $2,
				Concurrently:  $4,
				IndexColNames: indexElemColNames($10),
				Include:       indexIncludeNames($13),
				Using:         accessMethodName($8),
				WithOptions:   relOptionString($14),
				Tablespace:    $15,
				Where:         exprOrNil($16),
			}
		}
	;

opt_index_name:
	qualified_name
		{
			$$ = $1
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

opt_include:
	INCLUDE '(' index_including_params ')'
		{ $$ = $3 }
	| /* EMPTY */
		{ $$ = nil }
	;

index_including_params:
	index_elem
		{ $$ = makeList($1) }
	| index_including_params ',' index_elem
		{ $$ = appendList($1, $3) }
	;

opt_unique:
	UNIQUE      { $$ = true }
	| /* EMPTY */ { $$ = false }
	;

opt_concurrently:
	CONCURRENTLY { $$ = true }
	| /* EMPTY */  { $$ = false }
	;

opt_single_name:
	name        { $$ = $1 }
	| /* EMPTY */ { $$ = "" }
	;

access_method_clause:
	USING name  { $$ = $2 }
	| /* EMPTY */ { $$ = "" }
	;

index_params:
	index_elem
		{ $$ = makeList($1) }
	| index_params ',' index_elem
		{ $$ = appendList($1, $3) }
	;

index_elem:
	ColId opt_collate opt_qualified_name opt_asc_desc opt_nulls_order
		{
			$$ = &ast.IndexElem{
				Name:          $1,
				Collation:     $2,
				Opclass:       $3,
				Ordering:      ast.SortByDir($4),
				NullsOrdering: ast.SortByNulls($5),
			}
		}
	| ColId opt_collate any_name reloptions opt_asc_desc opt_nulls_order
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
	| func_expr_windowless opt_collate opt_qualified_name opt_asc_desc opt_nulls_order
		{
			$$ = &ast.IndexElem{
				Expr:          $1,
				Collation:     $2,
				Opclass:       $3,
				Ordering:      ast.SortByDir($4),
				NullsOrdering: ast.SortByNulls($5),
			}
		}
	| func_expr_windowless opt_collate any_name reloptions opt_asc_desc opt_nulls_order
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
	| '(' a_expr ')' opt_collate opt_qualified_name opt_asc_desc opt_nulls_order
		{
			$$ = &ast.IndexElem{
				Expr:          $2,
				Collation:     $4,
				Opclass:       $5,
				Ordering:      ast.SortByDir($6),
				NullsOrdering: ast.SortByNulls($7),
			}
		}
	| '(' a_expr ')' opt_collate any_name reloptions opt_asc_desc opt_nulls_order
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

opt_nulls_order:
	NULLS_LA FIRST_P  { $$ = int64(ast.SORTBY_NULLS_FIRST) }
	| NULLS_LA LAST_P { $$ = int64(ast.SORTBY_NULLS_LAST) }
	| /* EMPTY */      { $$ = int64(ast.SORTBY_NULLS_DEFAULT) }
	;

/*****************************************************************************
 *
 *      CREATE VIEW statement
 *
 *****************************************************************************/

ViewStmt:
	CREATE OptTemp VIEW qualified_name opt_column_list opt_reloptions
	AS SelectStmt opt_check_option
		{
			rv := relationName(makeRangeVar($4))
			rv.Relpersistence = relpersistenceForTemp($2)
			$$ = &ast.CreateViewStmt{
				Temporary:      $2 != 0,
				ViewName:       rv,
				Cols:           columnNameStrings($5),
				WithOptions:    relOptionsString($6),
				Select:         $8.(ast.StmtNode),
				HasCheckOption: $9 != 0,
				CheckOption:    viewCheckOption($9),
			}
		}
	| CREATE OR REPLACE OptTemp VIEW qualified_name opt_column_list opt_reloptions
	AS SelectStmt opt_check_option
		{
			rv := relationName(makeRangeVar($6))
			rv.Relpersistence = relpersistenceForTemp($4)
			$$ = &ast.CreateViewStmt{
				OrReplace:      true,
				Temporary:      $4 != 0,
				ViewName:       rv,
				Cols:           columnNameStrings($7),
				WithOptions:    relOptionsString($8),
				Select:         $10.(ast.StmtNode),
				HasCheckOption: $11 != 0,
				CheckOption:    viewCheckOption($11),
			}
		}
	| CREATE OptTemp RECURSIVE VIEW qualified_name '(' columnList ')' opt_reloptions
	AS SelectStmt opt_check_option
		{
			rv := relationName(makeRangeVar($5))
			rv.Relpersistence = relpersistenceForTemp($2)
			cte := &ast.CommonTableExpression{
				Name:     model.NewCIStr(rv.Name.O),
				ColNames: columnNameStrings($7),
				Query:    $11.(ast.ResultSetNode),
			}
			wc := &ast.WithClause{
				CTEs:        []*ast.CommonTableExpression{cte},
				IsRecursive: true,
			}
			sel := &ast.SelectStmt{
				SelectStmtOpts: &ast.SelectStmtOpts{},
				Fields: &ast.FieldList{Fields: []*ast.SelectField{
					{Expr: &ast.ColumnNameExpr{Name: &ast.ColumnName{Name: model.NewCIStr("*")}}},
				}},
				From: &ast.TableRefsClause{TableRefs: &ast.Join{
					Left: &ast.TableSource{Source: &ast.TableName{Name: model.NewCIStr(rv.Name.O)}},
				}},
				With: wc,
			}
			$$ = &ast.CreateViewStmt{
				Temporary:      $2 != 0,
				ViewName:       rv,
				Cols:           columnNameStrings($7),
				WithOptions:    relOptionsString($9),
				Select:         sel,
				HasCheckOption: $12 != 0,
				CheckOption:    viewCheckOption($12),
			}
		}
	| CREATE OR REPLACE OptTemp RECURSIVE VIEW qualified_name '(' columnList ')' opt_reloptions
	AS SelectStmt opt_check_option
		{
			rv := relationName(makeRangeVar($7))
			rv.Relpersistence = relpersistenceForTemp($4)
			cte := &ast.CommonTableExpression{
				Name:     model.NewCIStr(rv.Name.O),
				ColNames: columnNameStrings($9),
				Query:    $13.(ast.ResultSetNode),
			}
			wc := &ast.WithClause{
				CTEs:        []*ast.CommonTableExpression{cte},
				IsRecursive: true,
			}
			sel := &ast.SelectStmt{
				SelectStmtOpts: &ast.SelectStmtOpts{},
				Fields: &ast.FieldList{Fields: []*ast.SelectField{
					{Expr: &ast.ColumnNameExpr{Name: &ast.ColumnName{Name: model.NewCIStr("*")}}},
				}},
				From: &ast.TableRefsClause{TableRefs: &ast.Join{
					Left: &ast.TableSource{Source: &ast.TableName{Name: model.NewCIStr(rv.Name.O)}},
				}},
				With: wc,
			}
			$$ = &ast.CreateViewStmt{
				OrReplace:      true,
				Temporary:      $4 != 0,
				ViewName:       rv,
				Cols:           columnNameStrings($9),
				WithOptions:    relOptionsString($11),
				Select:         sel,
				HasCheckOption: $14 != 0,
				CheckOption:    viewCheckOption($14),
			}
		}
	;

opt_check_option:
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
	CREATE opt_or_replace FUNCTION func_name func_args_with_defaults
	RETURNS func_return opt_createfunc_opt_list opt_routine_body
		{
			$$ = &ast.CreateFunctionStmt{
				OrReplace:  $2,
				Name:       tableNameFromList($4),
				Params:     funcParams($5),
				Returns:    $7,
				Definition: routineDefinition($8),
				Body:       routineBodyStmt($9),
			}
		}
	| CREATE opt_or_replace FUNCTION func_name func_args_with_defaults
	RETURNS TABLE '(' table_func_column_list ')' opt_createfunc_opt_list opt_routine_body
		{
			$$ = &ast.CreateFunctionStmt{
				OrReplace:  $2,
				Name:       tableNameFromList($4),
				Params:     funcParams($5),
				Returns:    newDataType("record", -1, -1),
				Definition: routineDefinition($11),
				Body:       routineBodyStmt($12),
			}
		}
	| CREATE opt_or_replace FUNCTION func_name func_args_with_defaults
	opt_createfunc_opt_list opt_routine_body
		{
			$$ = &ast.CreateFunctionStmt{
				OrReplace:  $2,
				Name:       tableNameFromList($4),
				Params:     funcParams($5),
				Definition: routineDefinition($6),
				Body:       routineBodyStmt($7),
			}
		}
	| CREATE opt_or_replace PROCEDURE func_name func_args_with_defaults
	opt_createfunc_opt_list opt_routine_body
		{
			$$ = &ast.CreateFunctionStmt{
				OrReplace:  $2,
				Name:       tableNameFromList($4),
				Params:     funcParams($5),
				Definition: routineDefinition($6),
				Body:       routineBodyStmt($7),
			}
		}
	;

opt_or_replace:
	OR REPLACE   { $$ = true }
	| /* EMPTY */ { $$ = false }
	;

func_args_with_defaults:
	'(' func_args_with_defaults_list ')' { $$ = $2 }
	| '(' ')' { $$ = nil }
	;

func_args_with_defaults_list:
	func_arg_with_default
		{ $$ = makeList($1) }
	| func_args_with_defaults_list ',' func_arg_with_default
		{ $$ = appendList($1, $3) }
	;

func_arg_with_default:
	func_arg
		{ $$ = $1 }
	| func_arg DEFAULT a_expr
		{
			fp := $1.(*ast.FunctionParameter)
			fp.Defexpr = $3
			$$ = fp
		}
	| func_arg '=' a_expr
		{
			fp := $1.(*ast.FunctionParameter)
			fp.Defexpr = $3
			$$ = fp
		}
	;

func_arg:
	arg_class param_name func_type
		{
			$$ = &ast.FunctionParameter{
				Name:    $2,
				ArgType: $3,
				Mode:    ast.FunctionParameterMode($1),
			}
		}
	| param_name arg_class func_type
		{
			$$ = &ast.FunctionParameter{
				Name:    $1,
				ArgType: $3,
				Mode:    ast.FunctionParameterMode($2),
			}
		}
	| param_name func_type
		{
			$$ = &ast.FunctionParameter{
				Name:    $1,
				ArgType: $2,
				Mode:    ast.FUNC_PARAM_IN,
			}
		}
	| arg_class func_type
		{
			$$ = &ast.FunctionParameter{
				ArgType: $2,
				Mode:    ast.FunctionParameterMode($1),
			}
		}
	| func_type
		{
			$$ = &ast.FunctionParameter{
				ArgType: $1,
				Mode:    ast.FUNC_PARAM_IN,
			}
		}
	;

arg_class:
	IN_P       { $$ = int64(ast.FUNC_PARAM_IN) }
	| OUT_P    { $$ = int64(ast.FUNC_PARAM_OUT) }
	| INOUT    { $$ = int64(ast.FUNC_PARAM_INOUT) }
	| IN_P OUT_P { $$ = int64(ast.FUNC_PARAM_INOUT) }
	| VARIADIC { $$ = int64(ast.FUNC_PARAM_VARIADIC) }
	;

param_name:
	type_function_name { $$ = $1 }
	;

func_return:
	func_type { $$ = $1 }
	;

table_func_column_list:
	table_func_column
		{ $$ = makeList($1) }
	| table_func_column_list ',' table_func_column
		{ $$ = appendList($1, $3) }
	;

table_func_column:
	param_name func_type
		{
			$$ = &ast.FunctionParameter{
				Name:    $1,
				ArgType: $2,
				Mode:    ast.FUNC_PARAM_TABLE,
			}
		}
	;

func_type:
	Typename { $$ = $1 }
	| type_function_name attrs '%' TYPE_P
		{
			$$ = newDataType($1, -1, -1)
		}
	| SETOF type_function_name attrs '%' TYPE_P
		{
			$$ = newDataType($2, -1, -1)
		}
	;

opt_createfunc_opt_list:
	createfunc_opt_list { $$ = $1 }
	| /* EMPTY */ { $$ = nil }
	;

createfunc_opt_list:
	createfunc_opt_item
		{ $$ = makeList($1) }
	| createfunc_opt_list createfunc_opt_item
		{ $$ = appendList($1, $2) }
	;

opt_routine_body:
	ReturnStmt
		{
			$$ = $1
		}
	| BEGIN_P ATOMIC routine_body_stmt_list END_P
		{
			/* A compound statement stored as a single-item list containing the stmt list */
			$$ = makeList($3)
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

routine_body_stmt_list:
	routine_body_stmt_list routine_body_stmt ';'
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

routine_body_stmt:
	stmt
		{
			$$ = $1
		}
	| ReturnStmt
		{
			$$ = $1
		}
	;

ReturnStmt:
	RETURN a_expr
		{
			$$ = &ast.ReturnStmt{
				Expr: $2,
			}
		}
	;

createfunc_opt_item:
	AS func_as
		{
			$$ = &ast.DefElem{
				Defname: "as",
				Arg:     $2,
			}
		}
	| LANGUAGE NonReservedWord_or_Sconst
		{
			$$ = &ast.DefElem{
				Defname: "language",
				Arg:     &ast.String{Str: $2},
			}
		}
	| TRANSFORM transform_type_list
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
	| common_func_opt_item { $$ = $1 }
	;

func_as:
	Sconst
		{ $$ = makeList(&ast.String{Str: $1}) }
	| Sconst ',' Sconst
		{ $$ = makeList2(&ast.String{Str: $1}, &ast.String{Str: $3}) }
	;

transform_type_list:
	FOR TYPE_P Typename
		{ $$ = makeList(typeNameNode($3)) }
	| transform_type_list ',' FOR TYPE_P Typename
		{ $$ = appendList($1, typeNameNode($5)) }
	;

common_func_opt_item:
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
	| SET set_rest_more
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
	| SUPPORT any_name
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
	ABORT_P opt_transaction opt_transaction_chain
		{
			$$ = &ast.RollbackStmt{}
		}
	| BEGIN_P opt_transaction transaction_mode_list_or_empty
		{
			isolation, readOnly, deferrable := transactionModes($3)
			$$ = &ast.BeginStmt{
				IsolationLevel: isolation,
				ReadOnly:       readOnly,
				Deferrable:     deferrable,
			}
		}
	| START TRANSACTION transaction_mode_list_or_empty
		{
			isolation, readOnly, deferrable := transactionModes($3)
			$$ = &ast.BeginStmt{
				IsolationLevel: isolation,
				ReadOnly:       readOnly,
				Deferrable:     deferrable,
			}
		}
	| PREPARE TRANSACTION Sconst
		{
			$$ = &ast.TransactionStmt{
				Kind: ast.TRANS_STMT_PREPARE,
				Gid:  $3,
			}
		}
	| COMMIT PREPARED Sconst
		{
			$$ = &ast.TransactionStmt{
				Kind: ast.TRANS_STMT_COMMIT_PREPARED,
				Gid:  $3,
			}
		}
	| ROLLBACK PREPARED Sconst
		{
			$$ = &ast.TransactionStmt{
				Kind: ast.TRANS_STMT_ROLLBACK_PREPARED,
				Gid:  $3,
			}
		}
	| COMMIT opt_transaction opt_transaction_chain
		{
			$$ = &ast.CommitStmt{}
		}
	| END_P opt_transaction opt_transaction_chain
		{
			$$ = &ast.CommitStmt{}
		}
	| ROLLBACK opt_transaction opt_transaction_chain
		{
			$$ = &ast.RollbackStmt{}
		}
	| ROLLBACK opt_transaction TO SAVEPOINT ColId
		{
			$$ = &ast.RollbackStmt{SavepointName: $5}
		}
	| ROLLBACK opt_transaction TO ColId
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

opt_transaction:
	WORK        {}
	| TRANSACTION {}
	| /* EMPTY */ {}
	;

opt_transaction_chain:
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
	| EXPLAIN '(' utility_option_list ')' ExplainableStmt
		{
			$$ = &ast.ExplainStmt{
				Stmt:    $5.(ast.StmtNode),
				Options: explainOptionList($3),
				Format:  explainFormat(explainOptionList($3)),
			}
		}
	;

ExplainableStmt:
	SelectStmt          { $$ = $1 }
	| InsertStmt        { $$ = $1 }
	| UpdateStmt        { $$ = $1 }
	| DeleteStmt        { $$ = $1 }
	| MergeStmt         { $$ = $1 }
	| DeclareCursorStmt { $$ = $1 }
	| CreateAsStmt      { $$ = $1 }
	| CreateMatViewStmt { $$ = $1 }
	| RefreshMatViewStmt { $$ = $1 }
	| ExecuteStmt       { $$ = $1 }
	| CreateStmt        { $$ = $1 }
	;

/*****************************************************************************
 *
 *      COPY statement
 *
 *****************************************************************************/

CopyStmt:
	COPY opt_binary relation_expr opt_column_list
			copy_from opt_program copy_file_name copy_delimiter copy_opt_with
			copy_options where_clause
		{
			rv := relationName($3)
			stmt := &ast.CopyStmt{
				Table:       relationName(rv),
				Columns:  columnNamesFromList($4),
				IsFrom:   $5,
				Program:   $6,
				FileName:    $7,
				WhereClause: $11,
			}
			/* Concatenate user-supplied flags */
			var opts *ast.List
			if $2 != nil {
				opts = appendList(opts, $2)
			}
			if $8 != nil {
				opts = appendList(opts, $8)
			}
			opts = concatLists(opts, $10)
			stmt.Options = defElemNames(opts)
			$$ = stmt
		}
	| COPY '(' PreparableStmt ')' TO opt_program copy_file_name copy_opt_with copy_options
		{
			$$ = &ast.CopyStmt{
				Query:     $3,
				IsFrom:    false,
				Program: $6,
				FileName:  $7,
				Options:   defElemNames($9),
			}
		}
	;

copy_from:
	FROM { $$ = true }
	| TO { $$ = false }
	;

opt_program:
	PROGRAM     { $$ = true }
	| /* EMPTY */ { $$ = false }
	;

/*
 * copy_file_name NULL indicates stdio is used. Whether stdin or stdout is
 * used depends on the direction. (It really doesn't make sense to copy from
 * stdout. We silently correct the "typo".)		 - AY 9/94
 */
copy_file_name:
	Sconst      { $$ = $1 }
	| STDIN     { $$ = "" }
	| STDOUT    { $$ = "" }
	;

copy_opt_with:
	WITH {}
	| /* EMPTY */ {}
	;

copy_options:
	copy_opt_list                       { $$ = $1 }
	| '(' utility_option_list ')'       { $$ = $2 }
	;

/* old-style COPY option syntax */
copy_opt_list:
	copy_opt_list copy_opt_item
		{
			$$ = appendList($1, $2)
		}
	| /* EMPTY */ { $$ = nil }
	;

copy_opt_item:
	BINARY
		{
			$$ = &ast.DefElem{Defname: "format", Arg: &ast.String{Str: "binary"}}
		}
	| FREEZE
		{
			$$ = &ast.DefElem{Defname: "freeze", Arg: &ast.Boolean{Boolval: true}}
		}
	| DELIMITER opt_as Sconst
		{
			$$ = &ast.DefElem{Defname: "delimiter", Arg: &ast.String{Str: $3}}
		}
	| NULL_P opt_as Sconst
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
	| QUOTE opt_as Sconst
		{
			$$ = &ast.DefElem{Defname: "quote", Arg: &ast.String{Str: $3}}
		}
	| ESCAPE opt_as Sconst
		{
			$$ = &ast.DefElem{Defname: "escape", Arg: &ast.String{Str: $3}}
		}
	| FORCE QUOTE columnList
		{
			$$ = &ast.DefElem{Defname: "force_quote", Arg: $3}
		}
	| FORCE QUOTE '*'
		{
			$$ = &ast.DefElem{Defname: "force_quote", Arg: &ast.A_Star{}}
		}
	| FORCE NOT NULL_P columnList
		{
			$$ = &ast.DefElem{Defname: "force_not_null", Arg: $4}
		}
	| FORCE NOT NULL_P '*'
		{
			$$ = &ast.DefElem{Defname: "force_not_null", Arg: &ast.A_Star{}}
		}
	| FORCE NULL_P columnList
		{
			$$ = &ast.DefElem{Defname: "force_null", Arg: $3}
		}
	| FORCE NULL_P '*'
		{
			$$ = &ast.DefElem{Defname: "force_null", Arg: &ast.A_Star{}}
		}
	| ENCODING Sconst
		{
			$$ = &ast.DefElem{Defname: "encoding", Arg: &ast.String{Str: $2}}
		}
	;

/* The following exist for backward compatibility with very old versions */

opt_binary:
	BINARY
		{
			$$ = &ast.DefElem{Defname: "format", Arg: &ast.String{Str: "binary"}}
		}
	| /* EMPTY */ { $$ = nil }
	;

copy_delimiter:
	opt_using DELIMITERS Sconst
		{
			$$ = &ast.DefElem{Defname: "delimiter", Arg: &ast.String{Str: $3}}
		}
	| /* EMPTY */ { $$ = nil }
	;

opt_using:
	USING
	| /* EMPTY */
	;

utility_option_list:
	utility_option_elem
		{
			$$ = makeList($1)
		}
	| utility_option_list ',' utility_option_elem
		{
			$$ = appendList($1, $3)
		}
	;

utility_option_elem:
	utility_option_name utility_option_arg
		{
			$$ = &ast.DefElem{
				Defname: $1,
				Arg:     $2,
			}
		}
	;

utility_option_name:
	NonReservedWord  { $$ = $1 }
	| ANALYZE        { $$ = "analyze" }
	| FORMAT_LA      { $$ = "format" }
	| DEFAULT        { $$ = "default" }
	;

utility_option_arg:
	opt_boolean_or_string   { $$ = &ast.String{Str: $1} }
	| NumericOnly           { $$ = $1 }
	| '*'                   { $$ = &ast.A_Star{} }
	| DEFAULT               { $$ = &ast.String{Str: "default"} }
	| '(' utility_option_arg_list ')' { $$ = $2 }
	| /* EMPTY */           { $$ = nil }
	;

utility_option_arg_list:
	opt_boolean_or_string
		{ $$ = makeList(&ast.String{Str: $1}) }
	| utility_option_arg_list ',' opt_boolean_or_string
		{ $$ = appendList($1, &ast.String{Str: $3}) }
	;

opt_boolean_or_string:
	TRUE_P                      { $$ = "true" }
	| FALSE_P                   { $$ = "false" }
	| ON                        { $$ = "on" }
	| NonReservedWord_or_Sconst { $$ = $1 }
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

NumericOnly_list:
	NumericOnly
		{
			$$ = makeList($1)
		}
	| NumericOnly_list ',' NumericOnly
		{
			$$ = appendList($1, $3)
		}
	;

SignedIconst:
	Iconst      { $$ = $1 }
	| '+' Iconst { $$ = $2 }
	| '-' Iconst { $$ = -$2 }
	;

NonReservedWord_or_Sconst:
	NonReservedWord { $$ = $1 }
	| Sconst        { $$ = $1 }
	;

NonReservedWord:
	IDENT                    { $$ = $1 }
	| unreserved_keyword     { $$ = $1 }
	| col_name_keyword       { $$ = $1 }
	| type_func_name_keyword { $$ = $1 }
	;

/*****************************************************************************
 *
 *      GRANT / REVOKE statements
 *
 *****************************************************************************/

GrantStmt:
	GRANT privileges ON privilege_target TO grantee_list opt_grant_grant_option opt_granted_by
		{
			t := $4.(*ast.PrivilegeTarget)
			$$ = &ast.GrantStmt{
				ObjectType:  t.ObjectType,
				Objects:     t.Objects,
				Privs:       privElems($2),
				Users:       userSpecs($6),
				GrantOption: $7,
				Grantor:     roleSpecName($8),
			}
		}
	;

RevokeStmt:
	REVOKE privileges ON privilege_target FROM grantee_list opt_granted_by opt_drop_behavior
		{
			t := $4.(*ast.PrivilegeTarget)
			$$ = &ast.RevokeStmt{
				ObjectType: t.ObjectType,
				Objects:    t.Objects,
				Privs:      privElems($2),
				Users:      userSpecs($6),
				Behavior:   ast.DropBehavior($8),
			}
		}
	| REVOKE GRANT OPTION FOR privileges ON privilege_target FROM grantee_list opt_granted_by opt_drop_behavior
		{
			t := $7.(*ast.PrivilegeTarget)
			$$ = &ast.RevokeStmt{
				ObjectType: t.ObjectType,
				Objects:    t.Objects,
				Privs:      privElems($5),
				Users:      userSpecs($9),
				Behavior:   ast.DropBehavior($11),
			}
		}
	;

privilege_target:
	any_name_list
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeTable,
				Objects:    tableNamesFromNameList($1),
			}
		}
	| TABLE any_name_list
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeTable,
				Objects:    tableNamesFromNameList($2),
			}
		}
	| SEQUENCE any_name_list
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeSequence,
				Objects:    tableNamesFromNameList($2),
			}
		}
	| FOREIGN DATA_P WRAPPER name_list
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeForeignDataWrapper,
				Objects:    tableNamesFromNameList($4),
			}
		}
	| FOREIGN SERVER name_list
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeForeignServer,
				Objects:    tableNamesFromNameList($3),
			}
		}
	| FUNCTION function_with_argtypes_list
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeFunction,
				Objects:    objectWithArgsNames($2),
			}
		}
	| PROCEDURE function_with_argtypes_list
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeProcedure,
				Objects:    objectWithArgsNames($2),
			}
		}
	| ROUTINE function_with_argtypes_list
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeRoutine,
				Objects:    objectWithArgsNames($2),
			}
		}
	| TYPE_P any_name_list
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeUserType,
				Objects:    tableNamesFromNameList($2),
			}
		}
	| DOMAIN_P any_name_list
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeDomain,
				Objects:    tableNamesFromNameList($2),
			}
		}
	| LANGUAGE name_list
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeLanguage,
				Objects:    tableNamesFromNameList($2),
			}
		}
	| LARGE_P OBJECT_P NumericOnly_list
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeLargeObject,
			}
		}
	| SCHEMA name_list
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeSchema,
				Objects:    tableNamesFromNameList($2),
			}
		}
	| DATABASE name_list
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeDatabase,
				Objects:    tableNamesFromNameList($2),
			}
		}
	| TABLESPACE name_list
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeTablespace,
				Objects:    tableNamesFromNameList($2),
			}
		}
	| ALL TABLES IN_P SCHEMA name_list
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeAllTablesInSchema,
				Objects:    tableNamesFromNameList($5),
			}
		}
	| ALL SEQUENCES IN_P SCHEMA name_list
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeAllSequencesInSchema,
				Objects:    tableNamesFromNameList($5),
			}
		}
	| ALL FUNCTIONS IN_P SCHEMA name_list
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeAllFunctionsInSchema,
				Objects:    tableNamesFromNameList($5),
			}
		}
	| ALL PROCEDURES IN_P SCHEMA name_list
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeAllProceduresInSchema,
				Objects:    tableNamesFromNameList($5),
			}
		}
	| ALL ROUTINES IN_P SCHEMA name_list
		{
			$$ = &ast.PrivilegeTarget{
				ObjectType: ast.ObjectTypeAllRoutinesInSchema,
				Objects:    tableNamesFromNameList($5),
			}
		}
	;

privileges:
	ALL PRIVILEGES '(' columnList ')'
		{
			ap := &ast.AccessPriv{Cols: $4}
			$$ = makeList(ap)
		}
	| ALL '(' columnList ')'
		{
			ap := &ast.AccessPriv{Cols: $3}
			$$ = makeList(ap)
		}
	| ALL PRIVILEGES  { $$ = nil }
	| ALL           { $$ = nil }
	| privilege_list { $$ = $1 }
	;

privilege_list:
	privilege
		{ $$ = makeList($1) }
	| privilege_list ',' privilege
		{ $$ = appendList($1, $3) }
	;

privilege:
	SELECT opt_column_list
		{ $$ = &ast.AccessPriv{PrivName: "select", Cols: $2} }
	| REFERENCES opt_column_list
		{ $$ = &ast.AccessPriv{PrivName: "references", Cols: $2} }
	| CREATE opt_column_list
		{ $$ = &ast.AccessPriv{PrivName: "create", Cols: $2} }
	| ALTER SYSTEM_P
		{ $$ = &ast.AccessPriv{PrivName: "alter system"} }
	| ColId opt_column_list
		{ $$ = &ast.AccessPriv{PrivName: $1, Cols: $2} }
	;

grantee_list:
	grantee
		{ $$ = makeList($1) }
	| grantee_list ',' grantee
		{ $$ = appendList($1, $3) }
	;

grantee:
	RoleSpec { $$ = $1 }
	| GROUP_P RoleSpec { $$ = $2 }
	;

RoleSpec:
	ColId
		{
			$$ = &ast.RoleSpec{
				Roletype: int(ast.ROLESPEC_CSTRING),
				Rolename: $1,
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

opt_grant_grant_option:
	WITH GRANT OPTION { $$ = true }
	| /* EMPTY */     { $$ = false }
	;

/*****************************************************************************
 *
 *      GRANT / REVOKE ROLE statements
 *
 *****************************************************************************/

GrantRoleStmt:
	GRANT privilege_list TO role_list opt_granted_by
		{
			$$ = &ast.GrantRoleStmt{
				Roles: roleIdentities($4),
			}
		}
	| GRANT privilege_list TO role_list WITH grant_role_opt_list opt_granted_by
		{
			$$ = &ast.GrantRoleStmt{
				Roles: roleIdentities($4),
			}
		}
	;

RevokeRoleStmt:
	REVOKE privilege_list FROM role_list opt_granted_by opt_drop_behavior
		{
			$$ = &ast.RevokeRoleStmt{
				Roles: roleIdentities($2),
			}
		}
	| REVOKE ColId OPTION FOR privilege_list FROM role_list opt_granted_by opt_drop_behavior
		{
			$$ = &ast.RevokeRoleStmt{
				Roles: roleIdentities($5),
			}
		}
	;

grant_role_opt_list:
	grant_role_opt_list ',' grant_role_opt
		{ $$ = appendList($1, $3) }
	| grant_role_opt
		{ $$ = makeList($1) }
	;

grant_role_opt:
	ColLabel grant_role_opt_value
		{
			$$ = makeDefElem($1, $2)
		}
	;

grant_role_opt_value:
	OPTION   { $$ = &ast.Boolean{Boolval: true} }
	| TRUE_P { $$ = &ast.Boolean{Boolval: true} }
	| FALSE_P { $$ = &ast.Boolean{Boolval: false} }
	;

opt_granted_by:
	GRANTED BY RoleSpec { $$ = $3 }
	| /* EMPTY */       { $$ = nil }
	;

/*****************************************************************************
 *
 *      CREATE ROLE / USER / GROUP
 *
 *****************************************************************************/

CreateRoleStmt:
	CREATE ROLE RoleId opt_with OptRoleList
		{
			$$ = createUserStmt(true, $3, $5)
		}
	;

CreateUserStmt:
	CREATE USER RoleId opt_with OptRoleList
		{
			$$ = createUserStmt(false, $3, $5)
		}
	;

CreateGroupStmt:
	CREATE GROUP_P RoleId opt_with OptRoleList
		{
			$$ = &ast.CreateRoleStmt{
				StmtType: ast.ROLESTMT_GROUP,
				Role:     $3,
				Options:  $5,
			}
		}
	;

opt_with:
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
	PASSWORD Sconst
		{
			$$ = makeDefElem("password", &ast.String{Str: $2})
		}
	| PASSWORD NULL_P
		{
			$$ = makeDefElem("password", nil)
		}
	| ENCRYPTED PASSWORD Sconst
		{
			$$ = makeDefElem("password", &ast.String{Str: $3})
		}
	| UNENCRYPTED PASSWORD Sconst
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
	| VALID UNTIL Sconst
		{
			$$ = makeDefElem("validUntil", &ast.String{Str: $3})
		}
	| USER role_list
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
	| SYSID Iconst
		{
			$$ = makeDefElem("sysid", &ast.Integer{Ival: int64($2)})
		}
	| ADMIN role_list
		{
			$$ = makeDefElem("adminmembers", $2)
		}
	| ROLE role_list
		{
			$$ = makeDefElem("rolemembers", $2)
		}
	| IN_P ROLE role_list
		{
			$$ = makeDefElem("addroleto", $3)
		}
	| IN_P GROUP_P role_list
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
	| ALTER ROLE RoleSpec IN_P DATABASE name SetResetClause
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
	| ALTER ROLE ALL IN_P DATABASE name SetResetClause
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
	| ALTER USER RoleSpec IN_P DATABASE name SetResetClause
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
	| ALTER USER ALL IN_P DATABASE name SetResetClause
		{
			$$ = &ast.AlterRoleSetStmt{
				Database: $6,
				Setstmt:  $7,
			}
		}
	;

SetResetClause:
	SET set_rest
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
	DROP ROLE role_list
		{
			$$ = dropUserStmt(true, false, $3)
		}
	| DROP ROLE IF_P EXISTS role_list
		{
			$$ = dropUserStmt(true, true, $5)
		}
	| DROP USER role_list
		{
			$$ = dropUserStmt(false, false, $3)
		}
	| DROP USER IF_P EXISTS role_list
		{
			$$ = dropUserStmt(false, true, $5)
		}
	| DROP GROUP_P role_list
		{
			$$ = dropUserStmt(true, false, $3)
		}
	| DROP GROUP_P IF_P EXISTS role_list
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
	ALTER GROUP_P RoleSpec add_drop USER role_list
		{
			$$ = &ast.AlterRoleStmt{
				Role:   $3.(*ast.RoleSpec),
				Action: int($4),
				Options: makeList(makeDefElem("rolemembers", $6)),
			}
		}
	;

add_drop:
	ADD_P  { $$ = 1 }
	| DROP { $$ = -1 }
	;

/*****************************************************************************
 *
 *      CREATE DATABASE
 *
 *****************************************************************************/

CreatedbStmt:
	CREATE DATABASE name opt_with createdb_opt_list
		{
			$$ = &ast.CreateDatabaseStmt{
				Name:    $3,
				Options: databaseOptions($5),
			}
		}
	| CREATE DATABASE IF_P NOT EXISTS name opt_with createdb_opt_list
		{
			$$ = &ast.CreateDatabaseStmt{
				Name:        $6,
				Options:     databaseOptions($8),
				IfNotExists: true,
			}
		}
	;

createdb_opt_list:
	createdb_opt_items
		{ $$ = $1 }
	| /* EMPTY */
		{ $$ = nil }
	;

createdb_opt_items:
	createdb_opt_item
		{ $$ = makeList($1) }
	| createdb_opt_items createdb_opt_item
		{ $$ = appendList($1, $2) }
	;

createdb_opt_item:
	createdb_opt_name opt_equal NumericOnly
		{
			$$ = makeDefElem($1, $3)
		}
	| createdb_opt_name opt_equal opt_boolean_or_string
		{
			$$ = makeDefElem($1, &ast.String{Str: $3})
		}
	| createdb_opt_name opt_equal DEFAULT
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
createdb_opt_name:
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
opt_equal:
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
	ALTER DATABASE name WITH createdb_opt_list
		{
			$$ = &ast.AlterDatabaseStmt{
				Name:  $3,
				Options: defElemsToDatabaseOptions($5),
			}
		}
	| ALTER DATABASE name createdb_opt_list
		{
			$$ = &ast.AlterDatabaseStmt{
				Name:  $3,
				Options: defElemsToDatabaseOptions($4),
			}
		}
	| ALTER DATABASE name SET TABLESPACE name
		{
			$$ = &ast.AlterDatabaseStmt{
				Name:  $3,
				Options: defElemsToDatabaseOptions(makeList(makeDefElem("tablespace", &ast.String{Str: $6}))),
			}
		}
	;

AlterDatabaseSetStmt:
	ALTER DATABASE name SetResetClause
		{
			$$ = &ast.AlterDatabaseSetStmt{
				Dbname:  $3,
				Setstmt: $4,
			}
		}
	;

/*****************************************************************************
 *
 *      DROP DATABASE [ IF EXISTS ] dbname [ [ WITH ] ( options ) ]
 *
 *****************************************************************************/

DropdbStmt:
	DROP DATABASE name
		{
			$$ = &ast.DropdbStmt{
				Dbname:    $3,
				MissingOk: false,
			}
		}
	| DROP DATABASE IF_P EXISTS name
		{
			$$ = &ast.DropdbStmt{
				Dbname:    $5,
				MissingOk: true,
			}
		}
	| DROP DATABASE name opt_with '(' drop_option_list ')'
		{
			$$ = &ast.DropdbStmt{
				Dbname:    $3,
				MissingOk: false,
				Options:   $6,
			}
		}
	| DROP DATABASE IF_P EXISTS name opt_with '(' drop_option_list ')'
		{
			$$ = &ast.DropdbStmt{
				Dbname:    $5,
				MissingOk: true,
				Options:   $8,
			}
		}
	;

drop_option_list:
	drop_option
		{ $$ = makeList($1) }
	| drop_option_list ',' drop_option
		{ $$ = appendList($1, $3) }
	;

drop_option:
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
	ALTER SYSTEM_P SET generic_set
		{
			$$ = &ast.AlterSystemStmt{
				Setstmt: $4,
			}
		}
	| ALTER SYSTEM_P RESET generic_reset
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
	CREATE SCHEMA opt_single_name AUTHORIZATION RoleSpec OptSchemaEltList
		{
			$$ = &ast.CreateSchemaStmt{
				Name:          $3,
				Authorization: roleSpecName($5),
			}
		}
	| CREATE SCHEMA ColId OptSchemaEltList
		{
			$$ = &ast.CreateSchemaStmt{
				Name: $3,
			}
		}
	| CREATE SCHEMA IF_P NOT EXISTS opt_single_name AUTHORIZATION RoleSpec OptSchemaEltList
		{
			$$ = &ast.CreateSchemaStmt{
				Name:          $6,
				Authorization: roleSpecName($8),
				IfNotExists:   true,
			}
		}
	| CREATE SCHEMA IF_P NOT EXISTS ColId OptSchemaEltList
		{
			$$ = &ast.CreateSchemaStmt{
				Name:        $6,
				IfNotExists: true,
			}
		}
	;

OptSchemaEltList:
	OptSchemaEltList schema_stmt
		{
			$$ = appendList($1, $2)
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

schema_stmt:
	CreateStmt        { $$ = $1 }
	| IndexStmt       { $$ = $1 }
	| CreateSeqStmt   { $$ = $1 }
	| CreateTrigStmt  { $$ = $1 }
	| ViewStmt        { $$ = $1 }
	| GrantStmt       { $$ = $1 }
	;

/*****************************************************************************
 *
 *      CREATE SEQUENCE / ALTER SEQUENCE
 *
 *****************************************************************************/

CreateSeqStmt:
	CREATE OptTemp SEQUENCE qualified_name OptSeqOptList
		{
			rv := makeRangeVar($4)
			relationName(rv).Relpersistence = relpersistenceForTemp($2)
			$$ = &ast.CreateSequenceStmt{
				Name:    relationName(rv),
				Options: sequenceOptions($5),
			}
		}
	| CREATE OptTemp SEQUENCE IF_P NOT EXISTS qualified_name OptSeqOptList
		{
			rv := makeRangeVar($7)
			relationName(rv).Relpersistence = relpersistenceForTemp($2)
			$$ = &ast.CreateSequenceStmt{
				Name:        relationName(rv),
				Options:     sequenceOptions($8),
				IfNotExists: true,
			}
		}
	;

AlterSeqStmt:
	ALTER SEQUENCE qualified_name SeqOptList
		{
			rv := makeRangeVar($3)
			$$ = &ast.AlterSeqStmt{
				Sequence: relationName(rv),
				Options:  $4,
			}
		}
	| ALTER SEQUENCE IF_P EXISTS qualified_name SeqOptList
		{
			rv := makeRangeVar($5)
			$$ = &ast.AlterSeqStmt{
				Sequence:  relationName(rv),
				Options:   $6,
				MissingOk: true,
			}
		}
	;

OptParenthesizedSeqOptList:
	'(' SeqOptList ')'	{ $$ = $2 }
	| /* EMPTY */		{ $$ = nil }
	;

OptSeqOptList:
	SeqOptList  { $$ = $1 }
	| /* EMPTY */ { $$ = nil }
	;

SeqOptList:
	SeqOptElem
		{ $$ = makeList($1) }
	| SeqOptList SeqOptElem
		{ $$ = appendList($1, $2) }
	;

SeqOptElem:
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
	| INCREMENT opt_by NumericOnly
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
	| OWNED BY any_name
		{
			$$ = makeDefElem("owned_by", $3)
		}
	| SEQUENCE NAME_P any_name
		{
			$$ = makeDefElem("sequence_name", $3)
		}
	| START opt_with NumericOnly
		{
			$$ = makeDefElem("start", $3)
		}
	| RESTART
		{
			$$ = makeDefElem("restart", nil)
		}
	| RESTART opt_with NumericOnly
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

opt_by:
	BY     { /* nothing */ }
	| /* EMPTY */ { /* nothing */ }
	;

generated_when:
	ALWAYS      { $$ = int64('a') }
	| BY DEFAULT { $$ = int64('d') }
	;

alter_identity_column_option_list:
	alter_identity_column_option
		{ $$ = makeList($1) }
	| alter_identity_column_option_list alter_identity_column_option
		{ $$ = appendList($1, $2) }
	;

alter_identity_column_option:
	RESTART
		{
			$$ = makeDefElem("restart", nil)
		}
	| RESTART opt_with NumericOnly
		{
			$$ = makeDefElem("restart", $3)
		}
	| SET SeqOptElem
		{
			$$ = $2
		}
	| SET GENERATED generated_when
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
	CREATE DOMAIN_P any_name opt_as Typename opt_column_constraints
		{
			$$ = &ast.CreateDomainStmt{
				Domainname:  $3,
				Typname:     $5,
				Constraints: $6,
			}
		}
	;

opt_as:
	AS     { /* nothing */ }
	| /* EMPTY */ { /* nothing */ }
	;

/*****************************************************************************
 *
 *      ALTER DOMAIN
 *
 *****************************************************************************/

AlterDomainStmt:
	ALTER DOMAIN_P any_name alter_column_default
		{
			n := $4.(*ast.AlterDomainStmt)
			n.Typname = $3
			$$ = n
		}
	| ALTER DOMAIN_P any_name DROP NOT NULL_P
		{
			$$ = &ast.AlterDomainStmt{
				Subtype: 'N',
				Typname: $3,
			}
		}
	| ALTER DOMAIN_P any_name SET NOT NULL_P
		{
			$$ = &ast.AlterDomainStmt{
				Subtype: 'O',
				Typname: $3,
			}
		}
	| ALTER DOMAIN_P any_name ADD_P DomainConstraint
		{
			$$ = &ast.AlterDomainStmt{
				Subtype: 'C',
				Typname: $3,
				Def:     $5,
			}
		}
	| ALTER DOMAIN_P any_name DROP CONSTRAINT name opt_drop_behavior
		{
			$$ = &ast.AlterDomainStmt{
				Subtype:  'X',
				Typname:  $3,
				Name:     $6,
				Behavior: ast.DropBehavior($7),
			}
		}
	| ALTER DOMAIN_P any_name DROP CONSTRAINT IF_P EXISTS name opt_drop_behavior
		{
			$$ = &ast.AlterDomainStmt{
				Subtype:   'X',
				Typname:   $3,
				Name:      $8,
				Behavior:  ast.DropBehavior($9),
				MissingOk: true,
			}
		}
	| ALTER DOMAIN_P any_name VALIDATE CONSTRAINT name
		{
			$$ = &ast.AlterDomainStmt{
				Subtype: 'V',
				Typname: $3,
				Name:    $6,
			}
		}
	;

alter_column_default:
	SET DEFAULT a_expr
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
	ALTER TYPE_P any_name ADD_P VALUE_P opt_if_not_exists Sconst
		{
			$$ = &ast.AlterEnumStmt{
				Typname:            $3,
				Newval:             $7,
				SkipIfNewvalExists: $6,
			}
		}
	| ALTER TYPE_P any_name ADD_P VALUE_P opt_if_not_exists Sconst BEFORE Sconst
		{
			$$ = &ast.AlterEnumStmt{
				Typname:            $3,
				Newval:             $7,
				NewvalNeighbor:     $9,
				NewvalIsAfter:      false,
				SkipIfNewvalExists: $6,
			}
		}
	| ALTER TYPE_P any_name ADD_P VALUE_P opt_if_not_exists Sconst AFTER Sconst
		{
			$$ = &ast.AlterEnumStmt{
				Typname:            $3,
				Newval:             $7,
				NewvalNeighbor:     $9,
				NewvalIsAfter:      true,
				SkipIfNewvalExists: $6,
			}
		}
	| ALTER TYPE_P any_name RENAME VALUE_P Sconst TO Sconst
		{
			$$ = &ast.AlterEnumStmt{
				Typname: $3,
				Oldval:  $6,
				Newval:  $8,
			}
		}
	;

opt_if_not_exists:
	IF_P NOT EXISTS  { $$ = true }
	| /* EMPTY */    { $$ = false }
	;

AlterCollationStmt:
	ALTER COLLATION any_name REFRESH VERSION_P
		{
			$$ = &ast.AlterCollationStmt{
				Collname: $3,
			}
		}
	;

AlterCompositeTypeStmt:
	ALTER TYPE_P any_name alter_type_cmds
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
				Table: relationName(rv),
				Specs:    alterTableSpecs($4),
			}
		}
	;

alter_type_cmds:
	alter_type_cmd
		{ $$ = makeList($1) }
	| alter_type_cmds ',' alter_type_cmd
		{ $$ = appendList($1, $3) }
	;

alter_type_cmd:
	ADD_P ATTRIBUTE TableFuncElement opt_drop_behavior
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableAddColumns,
				NewColumns: []*ast.ColumnDef{$3.(*ast.ColumnDef)},
				Behavior: ast.DropBehavior(int($4)),
			}
		}
	| DROP ATTRIBUTE ColId opt_drop_behavior
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableDropColumn,
				Name: $3,
				Behavior: ast.DropBehavior(int($4)),
			}
		}
	| DROP ATTRIBUTE IF_P EXISTS ColId opt_drop_behavior
		{
			$$ = &ast.AlterTableSpec{

				Tp: ast.AlterTableDropColumn,
				Name: $5,
				Behavior: ast.DropBehavior(int($6)),
				MissingOk: true,
			}
		}
	| ALTER ATTRIBUTE ColId SET DATA_P TYPE_P Typename opt_collate_clause opt_drop_behavior
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
				Name: $3,
				NewColumns: []*ast.ColumnDef{coldef},
				Behavior: ast.DropBehavior(int($9)),
			}
		}
	| ALTER ATTRIBUTE ColId TYPE_P Typename opt_collate_clause opt_drop_behavior
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
				Name: $3,
				NewColumns: []*ast.ColumnDef{coldef},
				Behavior: ast.DropBehavior(int($7)),
			}
		}
	;

TableFuncElement:
	ColId Typename opt_collate_clause
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

opt_collate_clause:
	COLLATE any_name
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
	CREATE opt_or_replace AGGREGATE func_name aggr_args definition
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
	| CREATE opt_or_replace AGGREGATE func_name old_aggr_definition
		{
			$$ = &ast.DefineStmt{
				Kind:       ast.OBJECT_AGGREGATE,
				Oldstyle:   true,
				Replace:    $2,
				Defnames:   $4,
				Definition: $5,
			}
		}
	| CREATE OPERATOR any_operator definition
		{
			$$ = &ast.DefineStmt{
				Kind:       ast.OBJECT_OPERATOR,
				Defnames:   $3,
				Definition: $4,
			}
		}
	| CREATE TYPE_P any_name definition
		{
			$$ = &ast.DefineStmt{
				Kind:       ast.OBJECT_TYPE,
				Defnames:   $3,
				Definition: $4,
			}
		}
	| CREATE TYPE_P any_name
		{
			/* Shell type (identified by lack of definition) */
			$$ = &ast.DefineStmt{
				Kind:     ast.OBJECT_TYPE,
				Defnames: $3,
			}
		}
	| CREATE TEXT_P SEARCH PARSER any_name definition
		{
			$$ = &ast.DefineStmt{
				Kind:       ast.OBJECT_TSPARSER,
				Defnames:   $5,
				Definition: $6,
			}
		}
	| CREATE TEXT_P SEARCH DICTIONARY any_name definition
		{
			$$ = &ast.DefineStmt{
				Kind:       ast.OBJECT_TSDICTIONARY,
				Defnames:   $5,
				Definition: $6,
			}
		}
	| CREATE TEXT_P SEARCH TEMPLATE any_name definition
		{
			$$ = &ast.DefineStmt{
				Kind:       ast.OBJECT_TSTEMPLATE,
				Defnames:   $5,
				Definition: $6,
			}
		}
	| CREATE TEXT_P SEARCH CONFIGURATION any_name definition
		{
			$$ = &ast.DefineStmt{
				Kind:       ast.OBJECT_TSCONFIGURATION,
				Defnames:   $5,
				Definition: $6,
			}
		}
	| CREATE COLLATION any_name definition
		{
			$$ = &ast.DefineStmt{
				Kind:       ast.OBJECT_COLLATION,
				Defnames:   $3,
				Definition: $4,
			}
		}
	| CREATE COLLATION IF_P NOT EXISTS any_name definition
		{
			$$ = &ast.DefineStmt{
				Kind:        ast.OBJECT_COLLATION,
				Defnames:    $6,
				Definition:  $7,
				IfNotExists: true,
			}
		}
	| CREATE COLLATION any_name FROM any_name
		{
			$$ = &ast.DefineStmt{
				Kind:       ast.OBJECT_COLLATION,
				Defnames:   $3,
				Definition: makeList(makeDefElem("from", $5)),
			}
		}
	| CREATE COLLATION IF_P NOT EXISTS any_name FROM any_name
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
	CREATE TYPE_P any_name AS '(' OptTableFuncElementList ')'
		{
			$$ = &ast.CompositeTypeStmt{
				Typevar:    makeRangeVarFromAnyName($3),
				Coldeflist: $6,
			}
		}
	;

CreateEnumStmt:
	CREATE TYPE_P any_name AS ENUM_P '(' opt_enum_val_list ')'
		{
			$$ = &ast.CreateEnumStmt{
				TypeName: tableNameFromList($3),
				Vals:     $7,
			}
		}
	;

CreateRangeStmt:
	CREATE TYPE_P any_name AS RANGE definition
		{
			$$ = &ast.CreateRangeStmt{
				TypeName: tableNameFromList($3),
				Params:   $6,
			}
		}
	;

definition:
	'(' def_list ')'
		{
			$$ = $2
		}
	;

def_list:
	def_elem
		{
			$$ = makeList($1)
		}
	| def_list ',' def_elem
		{
			$$ = appendList($1, $3)
		}
	;

def_elem:
	ColLabel '=' def_arg
		{
			$$ = makeDefElem($1, $3)
		}
	| ColLabel
		{
			$$ = makeDefElem($1, nil)
		}
	;

/* Note: any simple identifier will be returned as a type name! */
def_arg:
	func_type
		{
			$$ = typeNameNode($1)
		}
	| reserved_keyword
		{
			$$ = &ast.String{Str: $1}
		}
	| qual_all_Op
		{
			$$ = $1
		}
	| NumericOnly
		{
			$$ = $1
		}
	| Sconst
		{
			$$ = &ast.String{Str: $1}
		}
	| NONE
		{
			$$ = &ast.String{Str: "none"}
		}
	;

old_aggr_definition:
	'(' old_aggr_list ')'
		{
			$$ = $2
		}
	;

old_aggr_list:
	old_aggr_elem
		{
			$$ = makeList($1)
		}
	| old_aggr_list ',' old_aggr_elem
		{
			$$ = appendList($1, $3)
		}
	;

/*
 * Must use IDENT here to avoid reduce/reduce conflicts; fortunately none of
 * the item names needed in old aggregate definitions are likely to become
 * SQL keywords.
 */
old_aggr_elem:
	IDENT '=' def_arg
		{
			$$ = makeDefElem($1, $3)
		}
	;

opt_enum_val_list:
	enum_val_list
		{
			$$ = $1
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

enum_val_list:
	Sconst
		{
			$$ = makeList(&ast.String{Str: $1})
		}
	| enum_val_list ',' Sconst
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

aggr_args:
	'(' '*' ')'
		{
			/* agg(*) - returns 2-element list: [nil, Integer{-1}] */
			$$ = makeList2(nil, &ast.Integer{Ival: -1})
		}
	| '(' aggr_args_list ')'
		{
			/* normal args - returns 2-element list: [args, Integer{-1}] */
			$$ = makeList2($2, &ast.Integer{Ival: -1})
		}
	| '(' ORDER BY aggr_args_list ')'
		{
			/* ordered-set agg with no direct args - returns 2-element list: [args, Integer{0}] */
			$$ = makeList2($4, &ast.Integer{Ival: 0})
		}
	| '(' aggr_args_list ORDER BY aggr_args_list ')'
		{
			/* ordered-set agg with direct args and ordered args */
			$$ = makeOrderedSetArgs($2, $5)
		}
	;

aggr_args_list:
	aggr_arg
		{
			$$ = makeList($1)
		}
	| aggr_args_list ',' aggr_arg
		{
			$$ = appendList($1, $3)
		}
	;

aggr_arg:
	func_arg
		{
			$$ = $1
		}
	;

/*****************************************************************************
 *
 *      any_operator, all_Op, MathOp, qual_all_Op
 *
 *****************************************************************************/

any_operator:
	all_Op
		{
			$$ = makeList(&ast.String{Str: $1})
		}
	| ColId '.' any_operator
		{
			$$ = prependList(&ast.String{Str: $1}, $3)
		}
	;

all_Op:
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

qual_all_Op:
	all_Op
		{
			$$ = makeList(&ast.String{Str: $1})
		}
	| OPERATOR '(' any_operator ')'
		{
			$$ = $3
		}
	;

qual_Op:
	Op
		{
			$$ = makeList(&ast.String{Str: $1})
		}
	| OPERATOR '(' any_operator ')'
		{
			$$ = $3
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
			$$ = spc.Rolename
		}
	;

role_list:
	RoleSpec
		{ $$ = makeList($1) }
	| role_list ',' RoleSpec
		{ $$ = appendList($1, $3) }
	;

// SELECT statement
SelectStmt:
	select_no_parens %prec UMINUS
		{
			$$ = $1
		}
	| select_with_parens %prec UMINUS
		{
			$$ = $1
		}
	;

select_with_parens:
	'(' select_no_parens ')'
		{
			$$ = inBraces($2)
		}
	| '(' select_with_parens ')'
		{
			$$ = inBraces($2)
		}
	;

select_no_parens:
	simple_select
		{
			$$ = $1
		}
	| select_clause sort_clause
		{
			$$ = setSelectOptions($1, $2, nil, nil)
		}
	| select_clause opt_sort_clause for_locking_clause opt_select_limit
		{
			$$ = setSelectOptions($1, $2, $3, $4)
		}
	| select_clause opt_sort_clause select_limit opt_for_locking_clause
		{
			$$ = setSelectOptions($1, $2, $4, $3)
		}
	| with_clause select_clause
		{
			$$ = setWithClause($2, $1)
		}
	| with_clause select_clause sort_clause
		{
			$$ = setSelectOptions(setWithClause($2, $1), $3, nil, nil)
		}
	| with_clause select_clause opt_sort_clause for_locking_clause opt_select_limit
		{
			$$ = setSelectOptions(setWithClause($2, $1), $3, $4, $5)
		}
	| with_clause select_clause opt_sort_clause select_limit opt_for_locking_clause
		{
			$$ = setSelectOptions(setWithClause($2, $1), $3, $5, $4)
		}
	;

select_clause:
	simple_select
		{
			$$ = $1
		}
	| select_with_parens
		{
			$$ = $1
		}
	;

simple_select:
	SELECT opt_all_clause opt_target_list into_clause from_clause where_clause group_clause having_clause window_clause
		{
			n := &ast.SelectStmt{
				SelectStmtOpts: &ast.SelectStmtOpts{},
				Fields:         &ast.FieldList{Fields: $3},
			}
			applySelectClauses(n, $5, $6, $7, $8, $9)
			$$ = n
		}
	| SELECT distinct_clause target_list into_clause from_clause where_clause group_clause having_clause window_clause
		{
			n := &ast.SelectStmt{
				Distinct:      true,
				SelectStmtOpts: &ast.SelectStmtOpts{Distinct: true},
				Fields:         &ast.FieldList{Fields: $3},
			}
			n.DistinctOn = byItemsFromExprs($2)
			applySelectClauses(n, $5, $6, $7, $8, $9)
			$$ = n
		}
	| select_clause UNION set_quantifier select_clause
		{
			$$ = makeSetOp(ast.Union, $3 == SET_QUANTIFIER_ALL, $1, $4)
		}
	| select_clause INTERSECT set_quantifier select_clause
		{
			$$ = makeSetOp(ast.Intersect, $3 == SET_QUANTIFIER_ALL, $1, $4)
		}
	| select_clause EXCEPT set_quantifier select_clause
		{
			$$ = makeSetOp(ast.Except, $3 == SET_QUANTIFIER_ALL, $1, $4)
		}
	| values_clause
		{
			$$ = $1
		}
	| TABLE relation_expr
		{
			$$ = &ast.TableStmt{
				Table: tableNameFromSource($2),
			}
		}
	;

values_clause:
	VALUES '(' expr_list ')'
		{
			$$ = &ast.ValuesStmt{Rows: []*ast.RowExpr{{Values: $3}}}
		}
	| values_clause ',' '(' expr_list ')'
		{
			n := $1.(*ast.ValuesStmt)
			n.Rows = append(n.Rows, &ast.RowExpr{Values: $4})
			$$ = n
		}
	;

opt_all_clause:
	ALL { $$ = true }
	| /* EMPTY */ { $$ = false }
	;


distinct_clause:
	DISTINCT
		{
			/* We use (NIL) as a placeholder to indicate that all target expressions
			 * should be placed in the DISTINCT list during parsetree analysis.
			 */
			$$ = []ast.ExprNode{nil}
		}
	| DISTINCT ON '(' expr_list ')'
		{
			$$ = $4
		}
	;

set_quantifier:
	ALL { $$ = SET_QUANTIFIER_ALL }
	| DISTINCT { $$ = SET_QUANTIFIER_DISTINCT }
	| /* EMPTY */ { $$ = SET_QUANTIFIER_DEFAULT }
	;

/*****************************************************************************
 *
 *      WITH clause (Common Table Expressions)
 *
 *****************************************************************************/

opt_with_clause:
	with_clause { $$ = $1 }
	| /* EMPTY */ { $$ = nil }
	;

with_clause:
	WITH cte_list
		{
			$$ = &ast.WithClause{
				CTEs: $2,
			}
		}
	| WITH_LA cte_list
		{
			$$ = &ast.WithClause{
				CTEs: $2,
			}
		}
	| WITH RECURSIVE cte_list
		{
			$$ = &ast.WithClause{
				CTEs:        $3,
				IsRecursive: true,
			}
		}
	;

cte_list:
	common_table_expr
		{
			$$ = []*ast.CommonTableExpression{$1.(*ast.CommonTableExpression)}
		}
	| cte_list ',' common_table_expr
		{
			$$ = append($1, $3.(*ast.CommonTableExpression))
		}
	;

common_table_expr:
	name opt_name_list AS opt_materialized '(' PreparableStmt ')' opt_search_clause opt_cycle_clause
		{
			cte := &ast.CommonTableExpression{
				Name:         model.NewCIStr($1),
				ColNames:   columnNameStrings($2),
				Query:        $6.(ast.ResultSetNode),
			}
			if $8 != nil {
				_ = $8
			}
			if $9 != nil {
				_ = $9
			}
			$$ = cte
		}
	| name '(' name_list ')' AS opt_materialized '(' PreparableStmt ')' opt_search_clause opt_cycle_clause
		{
			cte := &ast.CommonTableExpression{
				Name:         model.NewCIStr($1),
				ColNames:   columnNameStrings($3),
				Query:        $8.(ast.ResultSetNode),
			}
			if $10 != nil {
				_ = $10
			}
			if $11 != nil {
				_ = $11
			}
			$$ = cte
		}
	;

opt_materialized:
	MATERIALIZED			{ $$ = int64(ast.CTEMaterializeAlways) }
	| NOT MATERIALIZED		{ $$ = int64(ast.CTEMaterializeNever) }
	| /* EMPTY */			{ $$ = int64(ast.CTEMaterializeDefault) }
	;

opt_search_clause:
	SEARCH DEPTH FIRST_P BY columnList SET ColId
		{
			$$ = &ast.CTESearchClause{
				SearchColList:      $5,
				SearchBreadthFirst: false,
				SearchSeqColumn:    $7,
				Location:           -1,
			}
		}
	| SEARCH BREADTH FIRST_P BY columnList SET ColId
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

opt_cycle_clause:
	CYCLE columnList SET ColId TO AexprConst DEFAULT AexprConst USING ColId
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
	| CYCLE columnList SET ColId USING ColId
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

opt_target_list:
	target_list { $$ = $1 }
	| /* EMPTY */ { $$ = nil }
	;

into_clause:
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
	TEMPORARY opt_table qualified_name  { rv := makeRangeVar($3); relationName(rv).Relpersistence = 't'; $$ = rv }
	| TEMP opt_table qualified_name     { rv := makeRangeVar($3); relationName(rv).Relpersistence = 't'; $$ = rv }
	| LOCAL TEMPORARY opt_table qualified_name  { rv := makeRangeVar($4); relationName(rv).Relpersistence = 't'; $$ = rv }
	| LOCAL TEMP opt_table qualified_name  { rv := makeRangeVar($4); relationName(rv).Relpersistence = 't'; $$ = rv }
	| GLOBAL TEMPORARY opt_table qualified_name  { rv := makeRangeVar($4); relationName(rv).Relpersistence = 't'; $$ = rv }
	| GLOBAL TEMP opt_table qualified_name  { rv := makeRangeVar($4); relationName(rv).Relpersistence = 't'; $$ = rv }
	| UNLOGGED opt_table qualified_name  { rv := makeRangeVar($3); relationName(rv).Relpersistence = 'u'; $$ = rv }
	| TABLE qualified_name  { $$ = makeRangeVar($2) }
	| qualified_name        { $$ = makeRangeVar($1) }
	;

target_list:
	target_el
		{
			$$ = []*ast.SelectField{$1.(*ast.SelectField)}
		}
	| target_list ',' target_el
		{
			$$ = append($1, $3.(*ast.SelectField))
		}
	;

target_el:
	a_expr AS ColLabel
		{
			$$ = &ast.SelectField{Expr: $1, AsName: model.NewCIStr($3)}
		}
	| a_expr IDENT
		{
			$$ = &ast.SelectField{Expr: $1, AsName: model.NewCIStr($2)}
		}
	| a_expr
		{
			if wc, ok := $1.(*ast.WildCardField); ok {
				$$ = &ast.SelectField{WildCard: wc}
			} else {
				$$ = &ast.SelectField{Expr: $1}
			}
		}
	| '*'
		{
			$$ = &ast.SelectField{WildCard: &ast.WildCardField{}}
		}
	;

// FROM clause
from_clause:
	FROM from_list { $$ = $2 }
	| /* EMPTY */ { $$ = nil }
	;

from_list:
	table_ref
		{
			$$ = []ast.ResultSetNode{$1.(ast.ResultSetNode)}
		}
	| from_list ',' table_ref
		{
			$$ = append($1, $3.(ast.ResultSetNode))
		}
	;

table_ref:
	relation_expr opt_alias_clause
		{
			ts := $1.(*ast.TableSource)
			if $2 != nil {
				ts.AsName = model.NewCIStr(aliasName($2))
			}
			$$ = ts
		}
	| select_with_parens opt_alias_clause
		{
			ts := &ast.TableSource{Source: $1.(ast.ResultSetNode)}
			if $2 != nil {
				ts.AsName = model.NewCIStr(aliasName($2))
			}
			$$ = ts
		}
	| joined_table
		{
			$$ = $1
		}
	| relation_expr opt_alias_clause tablesample_clause
		{
			ts := $1.(*ast.TableSource)
			if $2 != nil {
				ts.AsName = model.NewCIStr(aliasName($2))
			}
			ts.TableSample = $3.(*ast.TableSample)
			$$ = ts
		}
	| func_table func_alias_clause
		{
			ts := $1.(*ast.TableSource)
			if $2 != nil {
				ts.AsName = model.NewCIStr(aliasName($2))
				ts.ColumnAliases = aliasColumnNames($2)
			}
			$$ = ts
		}
	| LATERAL_P func_table func_alias_clause
		{
			ts := $2.(*ast.TableSource)
			ts.Lateral = true
			if $3 != nil {
				ts.AsName = model.NewCIStr(aliasName($3))
				ts.ColumnAliases = aliasColumnNames($3)
			}
			$$ = ts
		}
	| LATERAL_P select_with_parens opt_alias_clause
		{
			ts := &ast.TableSource{Source: $2.(ast.ResultSetNode), Lateral: true}
			if $3 != nil {
				ts.AsName = model.NewCIStr(aliasName($3))
			}
			$$ = ts
		}
	| '(' joined_table ')' opt_alias_clause
		{
			$$ = $2
		}
	| xmltable opt_alias_clause
		{
			ts := &ast.TableSource{Source: $1}
			if $2 != nil {
				ts.AsName = model.NewCIStr(aliasName($2))
			}
			$$ = ts
		}
	| LATERAL_P xmltable opt_alias_clause
		{
			ts := &ast.TableSource{Source: $2, Lateral: true}
			if $3 != nil {
				ts.AsName = model.NewCIStr(aliasName($3))
			}
			$$ = ts
		}
	| json_table opt_alias_clause
		{
			ts := &ast.TableSource{Source: $1}
			if $2 != nil {
				ts.AsName = model.NewCIStr(aliasName($2))
			}
			$$ = ts
		}
	| LATERAL_P json_table opt_alias_clause
		{
			ts := &ast.TableSource{Source: $2, Lateral: true}
			if $3 != nil {
				ts.AsName = model.NewCIStr(aliasName($3))
			}
			$$ = ts
		}
	;

joined_table:
	table_ref CROSS JOIN table_ref
		{
			$$ = makeJoin(ast.CrossJoin, $1, $4, nil, false)
		}
	| table_ref join_type JOIN table_ref join_qual
		{
			$$ = makeJoin(joinTypeOf($2), $1, $4, $5, false)
		}
	| table_ref JOIN table_ref join_qual
		{
			$$ = makeJoin(ast.CrossJoin, $1, $3, $4, false)
		}
	| table_ref NATURAL join_type JOIN table_ref
		{
			$$ = makeJoin(joinTypeOf($3), $1, $5, nil, true)
		}
	| table_ref NATURAL JOIN table_ref
		{
			$$ = makeJoin(ast.CrossJoin, $1, $4, nil, true)
		}
	;

join_type:
	FULL opt_outer      { $$ = int64(ast.JOIN_FULL) }
	| LEFT opt_outer    { $$ = int64(ast.JOIN_LEFT) }
	| RIGHT opt_outer   { $$ = int64(ast.JOIN_RIGHT) }
	| INNER_P           { $$ = int64(ast.JOIN_INNER) }
	;

opt_outer:
	OUTER_P
		{ }
	| /* EMPTY */
		{ }
	;

join_qual:
	USING '(' name_list ')' opt_alias_clause
		{
			cond := &ast.OnCondition{Using: columnNamesFromList($3)}
			if $5 != nil {
				cond.JoinUsingAlias = $5.(*ast.Alias)
			}
			$$ = cond
		}
	| ON a_expr
		{
			$$ = &ast.OnCondition{Expr: $2}
		}
	;


relation_expr:
	qualified_name
		{
			$$ = makeTableSource($1)
		}
	| qualified_name '*'
		{
			$$ = makeTableSource($1)
		}
	| ONLY qualified_name
		{
			ts := makeTableSource($2)
			ts.Only = true
			$$ = ts
		}
	| ONLY '(' qualified_name ')'
		{
			ts := makeTableSource($3)
			ts.Only = true
			$$ = ts
		}
	;

opt_alias_clause:
	alias_clause { $$ = $1 }
	| /* EMPTY */ { $$ = nil }
	;

func_alias_clause:
	alias_clause { $$ = $1 }
	| AS '(' TableFuncElementList ')'
		{
			$$ = &ast.List{Items: []ast.Node{nil, $3}}
		}
	| AS ColId '(' TableFuncElementList ')'
		{
			$$ = &ast.List{Items: []ast.Node{
				&ast.Alias{Aliasname: $2},
				$4,
			}}
		}
	| ColId '(' TableFuncElementList ')'
		{
			$$ = &ast.List{Items: []ast.Node{
				&ast.Alias{Aliasname: $1},
				$3,
			}}
		}
	| /* EMPTY */ { $$ = nil }
	;

alias_clause:
	AS ColId '(' name_list ')'
		{
			$$ = &ast.Alias{Aliasname: $2, Colnames: $4}
		}
	| AS ColId
		{
			$$ = &ast.Alias{Aliasname: $2}
		}
	| ColId '(' name_list ')'
		{
			$$ = &ast.Alias{Aliasname: $1, Colnames: $3}
		}
	| ColId
		{
			$$ = &ast.Alias{Aliasname: $1}
		}
	;

func_table:
	func_expr_windowless opt_ordinality
		{
			$$ = &ast.TableSource{Source: &ast.TableFunction{
				Expr:           $1.(ast.ExprNode),
				WithOrdinality: $2,
			}}
		}
	| ROWS FROM '(' rowsfrom_list ')' opt_ordinality
		{
			$$ = &ast.TableSource{Source: &ast.RowsFromExpr{
				Functions:      rowsFromFunctions($4),
				WithOrdinality: $6,
			}}
		}
	;

rowsfrom_list:
	rowsfrom_item
		{
			$$ = makeList($1)
		}
	| rowsfrom_list ',' rowsfrom_item
		{
			$$ = appendList($1, $3)
		}
	;

rowsfrom_item:
	func_expr_windowless opt_col_def_list
		{
			fn := &ast.TableFunction{Expr: $1.(ast.ExprNode)}
			if $2 != nil {
				fn.ColDefs = tableFuncElementDefs($2)
			}
			$$ = fn
		}
	;

opt_col_def_list:
	AS '(' TableFuncElementList ')'
		{
			$$ = $3
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;




opt_ordinality:
	WITH_LA ORDINALITY		{ $$ = true }
	| /* EMPTY */		{ $$ = false }
	;

tablesample_clause:
	TABLESAMPLE func_name '(' expr_list ')' opt_repeatable_clause
		{
			$$ = &ast.TableSample{
				Method: funcNameString($2),
				Args: exprsFromList($4),
				Repeatable: $6,
			}
		}
	;

opt_repeatable_clause:
	REPEATABLE '(' a_expr ')'	{ $$ = $3 }
	| /* EMPTY */				{ $$ = nil }
	;

// WHERE clause
where_clause:
	WHERE a_expr { $$ = $2 }
	| /* EMPTY */ { $$ = nil }
	;

where_or_current_clause:
	WHERE a_expr { $$ = $2 }
	| WHERE CURRENT_P OF cursor_name
		{
			$$ = &ast.CurrentOfExpr{
				CursorName: $4,
			}
		}
	| /* EMPTY */ { $$ = nil }
	;

// GROUP BY clause
group_clause:
	GROUP_P BY set_quantifier group_by_list
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

group_by_list:
	group_by_item
		{ $$ = makeList($1) }
	| group_by_list ',' group_by_item
		{ $$ = appendList($1, $3) }
	;

group_by_item:
	a_expr { $$ = &ast.ByItem{Expr: $1} }
	| empty_grouping_set { $$ = &ast.ByItem{GroupingSet: $1.(*ast.GroupingSet)} }
	| cube_clause { $$ = &ast.ByItem{GroupingSet: $1.(*ast.GroupingSet)} }
	| rollup_clause { $$ = &ast.ByItem{GroupingSet: $1.(*ast.GroupingSet)} }
	| grouping_sets_clause { $$ = &ast.ByItem{GroupingSet: $1.(*ast.GroupingSet)} }
	;

empty_grouping_set:
	'(' ')'
		{
			$$ = &ast.GroupingSet{Type: ast.GroupingSetEmpty}
		}
	;

cube_clause:
	CUBE '(' expr_list ')'
		{
			$$ = &ast.GroupingSet{Type: ast.GroupingSetCube, Sets: [][]ast.ExprNode{$3}}
		}
	;

rollup_clause:
	ROLLUP '(' expr_list ')'
		{
			$$ = &ast.GroupingSet{Type: ast.GroupingSetRollup, Sets: [][]ast.ExprNode{$3}}
		}
	;

grouping_sets_clause:
	GROUPING SETS '(' group_by_list ')'
		{
			$$ = &ast.GroupingSet{Type: ast.GroupingSetSets, Sets: groupingSetsFromList($4)}
		}
	;

// HAVING clause
having_clause:
	HAVING a_expr { $$ = $2 }
	| /* EMPTY */ { $$ = nil }
	;

// Sort clause
sort_clause:
	ORDER BY sortby_list { $$ = orderByFromList($3) }
	;

opt_sort_clause:
	sort_clause { $$ = $1 }
	| /* EMPTY */ { $$ = nil }
	;

sortby_list:
	sortby
		{
			$$ = makeList($1)
		}
	| sortby_list ',' sortby
		{
			$$ = appendList($1, $3)
		}
	;

sortby:
	a_expr USING qual_all_Op opt_nulls_order
		{
			$$ = &ast.ByItem{Expr: $1}
		}
	| a_expr opt_asc_desc opt_nulls_order
		{
			item := &ast.ByItem{Expr: $1}
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

opt_asc_desc:
	ASC { $$ = int64(ast.SORTBY_ASC) }
	| DESC { $$ = int64(ast.SORTBY_DESC) }
	| /* EMPTY */ { $$ = int64(ast.SORTBY_DEFAULT) }
	;

select_limit:
	limit_clause offset_clause
		{
			$$ = $1
			$$.LimitOffset = $2
		}
	| offset_clause limit_clause
		{
			$$ = $2
			$$.LimitOffset = $1
		}
	| limit_clause
		{
			$$ = $1
		}
	| offset_clause
		{
			$$ = &SelectLimit{
				LimitOffset: $1,
				LimitOption: ast.LIMIT_OPTION_COUNT,
			}
		}
	;

limit_clause:
	LIMIT select_limit_value
		{
			$$ = limitValueClause($2)
		}
	| LIMIT select_limit_value ',' select_offset_value
		{
			/* PostgreSQL disallows this syntax with an error, but we parse it.
			 * The LIMIT #,# syntax is deprecated. */
			$$ = &SelectLimit{
				LimitOffset: $2,
				LimitCount:  $4,
				LimitOption: ast.LIMIT_OPTION_COUNT,
			}
		}
	| FETCH first_or_next select_fetch_first_value row_or_rows ONLY
		{
			$$ = &SelectLimit{
				LimitCount:  $3,
				LimitOption: ast.LIMIT_OPTION_COUNT,
			}
		}
	| FETCH first_or_next select_fetch_first_value row_or_rows WITH TIES
		{
			$$ = &SelectLimit{
				LimitCount:  $3,
				LimitOption: ast.LIMIT_OPTION_WITH_TIES,
			}
		}
	| FETCH first_or_next row_or_rows ONLY
		{
			$$ = &SelectLimit{
				LimitCount:  ast.NewValueExpr(int64(1)),
				LimitOption: ast.LIMIT_OPTION_COUNT,
			}
		}
	| FETCH first_or_next row_or_rows WITH TIES
		{
			$$ = &SelectLimit{
				LimitCount:  ast.NewValueExpr(int64(1)),
				LimitOption: ast.LIMIT_OPTION_WITH_TIES,
			}
		}
	;

offset_clause:
	OFFSET select_offset_value
		{ $$ = $2 }
	| OFFSET select_fetch_first_value row_or_rows
		{ $$ = $2 }
	;

first_or_next:
	FIRST_P
	| NEXT
	;

row_or_rows:
	ROW
	| ROWS
	;

select_limit_value:
	a_expr { $$ = $1 }
	| ALL
		{
			/* LIMIT ALL is represented as a NULL constant */
			$$ = ast.NewValueExpr(nil)
		}
	;

select_offset_value:
	a_expr { $$ = $1 }
	;

select_fetch_first_value:
	c_expr { $$ = $1 }
	| '+' c_expr
		{
			$$ = $2
		}
	| '-' c_expr
		{
			$$ = negateExpr($2)
		}
	;

opt_select_limit:
	select_limit		{ $$ = $1 }
	| /* EMPTY */		{ $$ = nil }
	;

/*
 * Locking clause (FOR UPDATE/SHARE etc)
 */
for_locking_clause:
	for_locking_items		{ $$ = $1 }
	| FOR READ ONLY			{ $$ = nil }
	;

opt_for_locking_clause:
	for_locking_clause		{ $$ = $1 }
	| /* EMPTY */			{ $$ = nil }
	;

for_locking_items:
	for_locking_item						{ $$ = makeList($1) }
	| for_locking_items for_locking_item	{ $$ = appendList($1, $2) }
	;

for_locking_item:
	for_locking_strength locked_rels_list opt_nowait_or_skip
		{
			$$ = &ast.LockingClause{
				LockedRels: $2,
				Strength:   int($1),
				WaitPolicy: int($3),
			}
		}
	;

for_locking_strength:
	FOR UPDATE				{ $$ = int64(ast.LCS_FORUPDATE) }
	| FOR NO KEY UPDATE		{ $$ = int64(ast.LCS_FORNOKEYUPDATE) }
	| FOR SHARE				{ $$ = int64(ast.LCS_FORSHARE) }
	| FOR KEY SHARE			{ $$ = int64(ast.LCS_FORKEYSHARE) }
	;

locked_rels_list:
	OF qualified_name_list	{ $$ = $2 }
	| /* EMPTY */			{ $$ = nil }
	;

opt_nowait_or_skip:
	NOWAIT					{ $$ = int64(ast.LockWaitError) }
	| SKIP LOCKED			{ $$ = int64(ast.LockWaitSkip) }
	| /* EMPTY */			{ $$ = int64(ast.LockWaitBlock) }
	;

// Expressions
a_expr:
	c_expr { $$ = $1 }
	| a_expr '+' a_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.Plus, $1, $3)
		}
	| a_expr '-' a_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.Minus, $1, $3)
		}
	| a_expr '*' a_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.Mul, $1, $3)
		}
	| a_expr '/' a_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.Div, $1, $3)
		}
	| a_expr '%' a_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.Mod, $1, $3)
		}
	| a_expr '^' a_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.Xor, $1, $3)
		}
	| a_expr '<' a_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.LT, $1, $3)
		}
	| a_expr '>' a_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.GT, $1, $3)
		}
	| a_expr '=' a_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.EQ, $1, $3)
		}
	| a_expr LESS_EQUALS a_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.LE, $1, $3)
		}
	| a_expr GREATER_EQUALS a_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.GE, $1, $3)
		}
	| a_expr NOT_EQUALS a_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.NE, $1, $3)
		}
	| a_expr qual_Op a_expr				%prec Op
		{
			$$ = operatorExpr(operatorName($2), $1, $3)
		}
	| qual_Op a_expr					%prec Op
		{
			$$ = ast.NewUnaryOperationExprByName(operatorName($1), $2)
		}
	| a_expr AND a_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.LogicAnd, $1, $3)
		}
	| a_expr OR a_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.LogicOr, $1, $3)
		}
	| NOT a_expr
		{
			$$ = ast.NewUnaryOperationExpr(opcode.Not, $2)
		}
	| NOT_LA a_expr						%prec NOT
		{
			$$ = ast.NewUnaryOperationExpr(opcode.Not, $2)
		}
	| a_expr IS NULL_P
		{
			$$ = &ast.IsNullExpr{Expr: $1}
		}
	| a_expr IS NOT NULL_P
		{
			$$ = &ast.IsNullExpr{Expr: $1, Not: true}
		}
	| a_expr IS TRUE_P
		{
			$$ = &ast.IsTruthExpr{Expr: $1, True: 1}
		}
	| a_expr IS FALSE_P
		{
			$$ = &ast.IsTruthExpr{Expr: $1}
		}
	| a_expr IS NOT TRUE_P              %prec IS
		{
			$$ = &ast.IsTruthExpr{Expr: $1, Not: true, True: 1}
		}
	| a_expr IS NOT FALSE_P             %prec IS
		{
			$$ = &ast.IsTruthExpr{Expr: $1, Not: true}
		}
	| a_expr IS UNKNOWN                 %prec IS
		{
			$$ = &ast.IsTruthExpr{Expr: $1, Not: true}
		}
	| a_expr IS NOT UNKNOWN             %prec IS
		{
			$$ = &ast.IsTruthExpr{Expr: $1, True: 1}
		}
	| a_expr ISNULL
		{
			$$ = &ast.IsNullExpr{Expr: $1}
		}
	| a_expr NOTNULL
		{
			$$ = &ast.IsNullExpr{Expr: $1, Not: true}
		}
	| a_expr IS DISTINCT FROM a_expr    %prec IS
		{
			$$ = ast.NewBinaryOperationExprByName("IS DISTINCT FROM", $1, $5)
		}
	| a_expr IS NOT DISTINCT FROM a_expr %prec IS
		{
			$$ = ast.NewBinaryOperationExprByName("IS NOT DISTINCT FROM", $1, $6)
		}
	| row OVERLAPS row
		{
			args := append(rowValues($1), rowValues($3)...)
			$$ = ast.NewFuncCallExpr("overlaps", args...)
		}
	| a_expr IS DOCUMENT_P              %prec IS
		{
			$$ = &ast.XmlExpr{
				Op:       ast.IS_DOCUMENT,
				Args:     makeList($1),
				Location: -1,
			}
		}
	| a_expr IS NOT DOCUMENT_P          %prec IS
		{
			$$ = ast.NewUnaryOperationExpr(opcode.Not, &ast.XmlExpr{
				Op:       ast.IS_DOCUMENT,
				Args:     makeList($1),
				Location: -1,
			})
		}
	| a_expr IS json_predicate_type_constraint json_key_uniqueness_constraint_opt %prec IS
		{
			$$ = &ast.JsonIsPredicate{
				Expr:       $1,
				ItemType:   jsonValueType($3),
				UniqueKeys: $4 != 0,
				Location:   -1,
			}
		}
	| a_expr IS NOT json_predicate_type_constraint json_key_uniqueness_constraint_opt %prec IS
		{
			$$ = ast.NewUnaryOperationExpr(opcode.Not, &ast.JsonIsPredicate{
				Expr:       $1,
				ItemType:   jsonValueType($4),
				UniqueKeys: $5 != 0,
				Location:   -1,
			})
		}
	| a_expr IS unicode_normal_form NORMALIZED %prec IS
		{
			$$ = ast.NewFuncCallExpr("pg_catalog.is_normalized", $1, ast.NewValueExpr($3))
		}
	| a_expr IS NOT unicode_normal_form NORMALIZED %prec IS
		{
			$$ = ast.NewUnaryOperationExpr(opcode.Not,
				ast.NewFuncCallExpr("pg_catalog.is_normalized", $1, ast.NewValueExpr($3)))
		}
	| a_expr IS NORMALIZED              %prec IS
		{
			$$ = ast.NewFuncCallExpr("pg_catalog.is_normalized", $1, ast.NewValueExpr("NFC"))
		}
	| a_expr IS NOT NORMALIZED          %prec IS
		{
			$$ = ast.NewUnaryOperationExpr(opcode.Not,
				ast.NewFuncCallExpr("pg_catalog.is_normalized", $1, ast.NewValueExpr("NFC")))
		}
	| a_expr LIKE a_expr                %prec LIKE
		{
			$$ = &ast.PatternLikeExpr{Expr: $1, Pattern: $3}
		}
	| a_expr LIKE a_expr ESCAPE a_expr   %prec LIKE
		{
			$$ = &ast.PatternLikeExpr{Expr: $1, Pattern: $3, Escape: escapeChar($5)}
		}
	| a_expr NOT_LA LIKE a_expr          %prec NOT_LA
		{
			$$ = &ast.PatternLikeExpr{Expr: $1, Pattern: $4, Not: true}
		}
	| a_expr NOT_LA LIKE a_expr ESCAPE a_expr %prec NOT_LA
		{
			$$ = &ast.PatternLikeExpr{Expr: $1, Pattern: $4, Not: true, Escape: escapeChar($6)}
		}
	| a_expr ILIKE a_expr               %prec ILIKE
		{
			$$ = &ast.PatternLikeExpr{Expr: $1, Pattern: $3, ILike: true}
		}
	| a_expr ILIKE a_expr ESCAPE a_expr %prec ILIKE
		{
			$$ = &ast.PatternLikeExpr{Expr: $1, Pattern: $3, ILike: true, Escape: escapeChar($5)}
		}
	| a_expr NOT_LA ILIKE a_expr        %prec NOT_LA
		{
			$$ = &ast.PatternLikeExpr{Expr: $1, Pattern: $4, Not: true, ILike: true}
		}
	| a_expr NOT_LA ILIKE a_expr ESCAPE a_expr %prec NOT_LA
		{
			$$ = &ast.PatternLikeExpr{Expr: $1, Pattern: $4, Not: true, ILike: true, Escape: escapeChar($6)}
		}
	| a_expr SIMILAR TO a_expr          %prec SIMILAR
		{
			$$ = &ast.PatternRegexpExpr{Expr: $1, Pattern: $4}
		}
	| a_expr SIMILAR TO a_expr ESCAPE a_expr %prec SIMILAR
		{
			$$ = &ast.PatternRegexpExpr{Expr: $1, Pattern: $4, Escape: escapeChar($6)}
		}
	| a_expr NOT_LA SIMILAR TO a_expr   %prec NOT_LA
		{
			$$ = &ast.PatternRegexpExpr{Expr: $1, Pattern: $5, Not: true}
		}
	| a_expr NOT_LA SIMILAR TO a_expr ESCAPE a_expr %prec NOT_LA
		{
			$$ = &ast.PatternRegexpExpr{Expr: $1, Pattern: $5, Not: true, Escape: escapeChar($7)}
		}
	| a_expr BETWEEN opt_asymmetric b_expr AND a_expr %prec BETWEEN
		{
			$$ = &ast.BetweenExpr{Expr: $1, Left: $4, Right: $6}
		}
	| a_expr NOT_LA BETWEEN opt_asymmetric b_expr AND a_expr %prec NOT_LA
		{
			$$ = &ast.BetweenExpr{Expr: $1, Left: $5, Right: $7, Not: true}
		}
	| a_expr BETWEEN SYMMETRIC b_expr AND a_expr %prec BETWEEN
		{
			$$ = &ast.BetweenExpr{Expr: $1, Left: $4, Right: $6}
		}
	| a_expr NOT_LA BETWEEN SYMMETRIC b_expr AND a_expr %prec NOT_LA
		{
			$$ = &ast.BetweenExpr{Expr: $1, Left: $5, Right: $7, Not: true}
		}
	| a_expr IN_P '(' expr_list ')'
		{
			$$ = &ast.PatternInExpr{Expr: $1, List: $4}
		}
	| a_expr NOT_LA IN_P '(' expr_list ')' %prec NOT_LA
		{
			$$ = &ast.PatternInExpr{Expr: $1, List: $5, Not: true}
		}
	| a_expr IN_P select_with_parens
		{
			$$ = &ast.PatternInExpr{Expr: $1, Sel: &ast.SubqueryExpr{Query: $3.(ast.ResultSetNode)}}
		}
	| a_expr NOT_LA IN_P select_with_parens %prec NOT_LA
		{
			$$ = &ast.PatternInExpr{Expr: $1, Sel: &ast.SubqueryExpr{Query: $4.(ast.ResultSetNode)}, Not: true}
		}
	| a_expr subquery_Op sub_type select_with_parens %prec Op
		{
			$$ = &ast.CompareSubqueryExpr{
				L:   $1,
				Op:  compareOpcode(operatorName($2)),
				R:   &ast.SubqueryExpr{Query: $4.(ast.ResultSetNode)},
				All: $3 == int64(ast.ALL_SUBLINK),
			}
		}
	| a_expr subquery_Op sub_type '(' a_expr ')' %prec Op
		{
			$$ = ast.NewBinaryOperationExprByName(operatorName($2)+" "+subTypeName($3), $1, $5)
		}
	| UNIQUE opt_unique_null_treatment select_with_parens
		{
			yylex.Error("UNIQUE predicate is not yet implemented")
			$$ = nil
		}
	| a_expr COLLATE any_name
		{
			$$ = &ast.CollateExpr{Expr: $1, Collation: nameListString($3)}
		}
	| a_expr AT TIME ZONE a_expr        %prec AT
		{
			$$ = ast.NewFuncCallExpr("timezone", $5, $1)
		}
	| a_expr AT LOCAL                   %prec AT
		{
			$$ = ast.NewFuncCallExpr("timezone", $1)
		}
	| DEFAULT
		{
			$$ = &ast.DefaultExpr{}
		}
	| a_expr '[' a_expr ']'
		{
			$$ = &ast.A_Indirection{
				Arg:         $1,
				Indirection: makeList(&ast.A_Indices{Uidx: $3}),
			}
		}
	| a_expr '[' opt_slice_bound ':' opt_slice_bound ']'
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
	| a_expr TYPECAST Typename
		{
			$$ = ast.NewTypeCastExpr($1, $3)
		}
	| '+' a_expr %prec UMINUS
		{
			$$ = $2
		}
	| '-' a_expr %prec UMINUS
		{
			$$ = negateExpr($2)
		}
	;

opt_asymmetric:
	ASYMMETRIC  {}
	| /* EMPTY */ {}
	;

sub_type:
	ANY  { $$ = int64(ast.ANY_SUBLINK) }
	| SOME { $$ = int64(ast.ANY_SUBLINK) }
	| ALL  { $$ = int64(ast.ALL_SUBLINK) }
	;

subquery_Op:
	all_Op
		{
			$$ = makeList(&ast.String{Str: $1})
		}
	| OPERATOR '(' any_operator ')'
		{
			$$ = $3
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

case_expr:
	CASE case_arg when_clause_list case_default END_P
		{
			$$ = &ast.CaseExpr{
				Value:      $2,
				WhenClauses: whenClauses($3),
				ElseClause: $4,
			}
		}
	;

when_clause_list:
	when_clause
		{
			$$ = makeList($1)
		}
	| when_clause_list when_clause
		{
			$$ = appendList($1, $2)
		}
	;

when_clause:
	WHEN a_expr THEN a_expr
		{
			$$ = &ast.WhenClause{
				Expr:   $2,
				Result: $4,
			}
		}
	;

case_default:
	ELSE a_expr { $$ = $2 }
	| /* EMPTY */ { $$ = nil }
	;

case_arg:
	a_expr { $$ = $1 }
	| /* EMPTY */ { $$ = nil }
	;

/*****************************************************************************
 *
 *      ARRAY expressions
 *
 *****************************************************************************/

array_expr:
	'[' expr_list ']'
		{
			$$ = ast.NewArrayExpr($2)
		}
	| '[' array_expr_list ']'
		{
			$$ = ast.NewArrayExpr($2)
		}
	| '[' ']'
		{
			$$ = ast.NewArrayExpr(nil)
		}
	;

array_expr_list:
	array_expr
		{
			$$ = []ast.ExprNode{$1}
		}
	| array_expr_list ',' array_expr
		{
			$$ = append($1, $3)
		}
	;

/*****************************************************************************
 *
 *      ROW expressions
 *
 *****************************************************************************/

row:
	explicit_row
		{
			$$ = $1
		}
	| implicit_row
		{
			$$ = &ast.RowExpr{Values: $1}
		}
	;

explicit_row:
	ROW '(' expr_list ')'
		{
			$$ = &ast.RowExpr{Values: $3}
		}
	| ROW '(' ')'
		{
			$$ = &ast.RowExpr{}
		}
	;

implicit_row:
	'(' expr_list ',' a_expr ')'
		{
			$$ = append($2, $4)
		}
	;

b_expr:
	c_expr { $$ = $1 }
	| b_expr '+' b_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.Plus, $1, $3)
		}
	| b_expr '-' b_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.Minus, $1, $3)
		}
	| b_expr '*' b_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.Mul, $1, $3)
		}
	| b_expr '/' b_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.Div, $1, $3)
		}
	| b_expr '%' b_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.Mod, $1, $3)
		}
	| b_expr '^' b_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.Xor, $1, $3)
		}
	| b_expr '<' b_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.LT, $1, $3)
		}
	| b_expr '>' b_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.GT, $1, $3)
		}
	| b_expr '=' b_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.EQ, $1, $3)
		}
	| b_expr LESS_EQUALS b_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.LE, $1, $3)
		}
	| b_expr GREATER_EQUALS b_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.GE, $1, $3)
		}
	| b_expr NOT_EQUALS b_expr
		{
			$$ = ast.NewBinaryOperationExpr(opcode.NE, $1, $3)
		}
	| b_expr qual_Op b_expr				%prec Op
		{
			$$ = ast.NewBinaryOperationExprByName(operatorName($2), $1, $3)
		}
	| qual_Op b_expr					%prec Op
		{
			$$ = ast.NewUnaryOperationExprByName(operatorName($1), $2)
		}
	| b_expr IS DISTINCT FROM b_expr		%prec IS
		{
			$$ = ast.NewBinaryOperationExprByName("IS DISTINCT FROM", $1, $5)
		}
	| b_expr IS NOT DISTINCT FROM b_expr	%prec IS
		{
			$$ = ast.NewBinaryOperationExprByName("IS NOT DISTINCT FROM", $1, $6)
		}
	| b_expr IS DOCUMENT_P				%prec IS
		{
			$$ = &ast.XmlExpr{
				Op:       ast.IS_DOCUMENT,
				Args:     makeList($1),
				Location: -1,
			}
		}
	| b_expr IS NOT DOCUMENT_P			%prec IS
		{
			$$ = ast.NewUnaryOperationExpr(opcode.Not, &ast.XmlExpr{
				Op:       ast.IS_DOCUMENT,
				Args:     makeList($1),
				Location: -1,
			})
		}
	| b_expr TYPECAST Typename
		{
			$$ = ast.NewTypeCastExpr($1, $3)
		}
	| '+' b_expr %prec UMINUS
		{
			$$ = $2
		}
	| '-' b_expr %prec UMINUS
		{
			$$ = negateExpr($2)
		}
	;

c_expr:
	columnref { $$ = $1 }
	| AexprConst { $$ = $1 }
	| PARAM opt_indirection
		{
			$$ = ast.NewParamMarkerExpr(int($1) - 1)
		}
	| '(' a_expr ')' opt_indirection
		{
			$$ = &ast.ParenthesesExpr{Expr: $2}
		}
	| func_expr { $$ = $1 }
	| select_with_parens %prec UMINUS
		{
			$$ = &ast.SubqueryExpr{Query: $1.(ast.ResultSetNode)}
		}
	| EXISTS select_with_parens
		{
			$$ = &ast.ExistsSubqueryExpr{Sel: &ast.SubqueryExpr{Query: $2.(ast.ResultSetNode)}}
		}
	| case_expr { $$ = $1 }
	| ARRAY select_with_parens
		{
			$$ = &ast.SubqueryExpr{Query: $2.(ast.ResultSetNode)}
		}
	| ARRAY array_expr
		{
			$$ = $2
		}
	| explicit_row
		{
			$$ = $1
		}
	| implicit_row
		{
			$$ = &ast.RowExpr{Values: $1}
		}
	;

// Function expressions
func_expr:
	func_application within_group_clause filter_clause over_clause
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
			} else if fn.WithinGroup != nil || fn.Filter != nil || isAggregateName(fn.FnName.O) {
				$$ = &ast.AggregateFuncExpr{
					F:           fn.FnName.O,
					Args:        fn.Args,
					Filter:      fn.Filter,
					WithinGroup: fn.WithinGroup,
				}
			} else {
				$$ = fn
			}
		}
	| func_expr_common_subexpr { $$ = $1 }
	| json_aggregate_func filter_clause over_clause
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

func_expr_windowless:
	func_application		{ $$ = $1 }
	| func_expr_common_subexpr	{ $$ = $1 }
	| json_aggregate_func	{ $$ = $1 }
	;

within_group_clause:
	WITHIN GROUP_P '(' sort_clause ')'	{ $$ = $4 }
	| /* EMPTY */						{ $$ = nil }
	;

filter_clause:
	FILTER '(' WHERE a_expr ')'		{ $$ = $4 }
	| /* EMPTY */						{ $$ = nil }
	;

over_clause:
	OVER window_specification	{ $$ = $2 }
	| OVER ColId
		{
			$$ = &ast.WindowSpec{Name: model.NewCIStr($2)}
		}
	| /* EMPTY */				{ $$ = nil }
	;

window_clause:
	WINDOW window_definition_list	{ $$ = $2 }
	| /* EMPTY */					{ $$ = nil }
	;

window_definition_list:
	window_definition							{ $$ = makeList($1) }
	| window_definition_list ',' window_definition	{ $$ = appendList($1, $3) }
	;

window_definition:
	ColId AS window_specification
		{
			n := $3.(*ast.WindowSpec)
			n.Name = model.NewCIStr($1)
			$$ = n
		}
	;

window_specification:
	'(' opt_existing_window_name opt_partition_clause opt_sort_clause opt_frame_clause ')'
		{
			spec := &ast.WindowSpec{Ref: model.NewCIStr($2)}
			if $3 != nil {
				spec.PartitionBy = &ast.PartitionByClause{Items: byItemsFromExprs($3)}
			}
			if $4 != nil {
				spec.OrderBy = $4.(*ast.OrderByClause)
			}
			if $5 != nil {
				spec.Frame = $5.(*ast.FrameClause)
			}
			$$ = spec
		}
	;

opt_existing_window_name:
	ColId			{ $$ = $1 }
	| /* EMPTY */	%prec Op	{ $$ = "" }
	;

opt_partition_clause:
	PARTITION BY expr_list	{ $$ = $3 }
	| /* EMPTY */			{ $$ = nil }
	;

opt_frame_clause:
	RANGE frame_extent opt_window_exclusion_clause
		{
			$$ = &ast.FrameClause{Type: ast.Ranges, Extent: $2}
		}
	| ROWS frame_extent opt_window_exclusion_clause
		{
			$$ = &ast.FrameClause{Type: ast.Rows, Extent: $2}
		}
	| GROUPS frame_extent opt_window_exclusion_clause
		{
			$$ = &ast.FrameClause{Type: ast.Groups, Extent: $2}
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

frame_extent:
	frame_bound
		{
			$$ = ast.FrameExtent{Start: *$1, End: ast.FrameBound{Type: ast.CurrentRow}}
		}
	| BETWEEN frame_bound AND frame_bound
		{
			$$ = ast.FrameExtent{Start: *$2, End: *$4}
		}
	;

frame_bound:
	UNBOUNDED PRECEDING
		{ $$ = &ast.FrameBound{Type: ast.Preceding, UnBounded: true} }
	| UNBOUNDED FOLLOWING
		{ $$ = &ast.FrameBound{Type: ast.Following, UnBounded: true} }
	| CURRENT_P ROW
		{ $$ = &ast.FrameBound{Type: ast.CurrentRow} }
	| a_expr PRECEDING
		{ $$ = &ast.FrameBound{Type: ast.Preceding, Expr: $1} }
	| a_expr FOLLOWING
		{ $$ = &ast.FrameBound{Type: ast.Following, Expr: $1} }
	;

opt_window_exclusion_clause:
	EXCLUDE CURRENT_P ROW		{ $$ = int64(ast.FRAMEOPTION_EXCLUDE_CURRENT_ROW) }
	| EXCLUDE GROUP_P			{ $$ = int64(ast.FRAMEOPTION_EXCLUDE_GROUP) }
	| EXCLUDE TIES				{ $$ = int64(ast.FRAMEOPTION_EXCLUDE_TIES) }
	| EXCLUDE NO OTHERS			{ $$ = 0 }
	| /* EMPTY */				{ $$ = 0 }
	;

func_application:
	func_name '(' ')'
		{
			$$ = ast.NewFuncCallExpr(funcNameString($1))
		}
	| func_name '(' func_arg_list opt_sort_clause ')'
		{
			$$ = ast.NewFuncCallExpr(funcNameString($1), exprsFromList($3)...)
		}
	| func_name '(' VARIADIC func_arg_expr opt_sort_clause ')'
		{
			$$ = ast.NewFuncCallExpr(funcNameString($1), $4)
		}
	| func_name '(' func_arg_list ',' VARIADIC func_arg_expr opt_sort_clause ')'
		{
			args := append(exprsFromList($3), $6)
			$$ = ast.NewFuncCallExpr(funcNameString($1), args...)
		}
	| func_name '(' '*' ')'
		{
			$$ = &ast.FuncCallExpr{FnName: model.NewCIStr(funcNameString($1)), Star: true}
		}
	| func_name '(' DISTINCT func_arg_list opt_sort_clause ')'
		{
			$$ = ast.NewFuncCallExpr(funcNameString($1), exprsFromList($4)...)
		}
	| func_name '(' ALL func_arg_list opt_sort_clause ')'
		{
			$$ = ast.NewFuncCallExpr(funcNameString($1), exprsFromList($4)...)
		}
	;

/*
 * func_expr_common_subexpr: SQL-standard special function expressions that
 * use reserved keywords as function names. These cannot be ordinary function
 * calls because the keywords would be consumed as part of the syntax.
 */
func_expr_common_subexpr:
	COLLATION FOR '(' a_expr ')'
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
			$$ = makeSQLValueFunction(ast.SVFOP_CURRENT_DATE, -1)
		}
	| CURRENT_TIME
		{
			$$ = makeSQLValueFunction(ast.SVFOP_CURRENT_TIME, -1)
		}
	| CURRENT_TIME '(' Iconst ')'
		{
			$$ = makeSQLValueFunction(ast.SVFOP_CURRENT_TIME_N, int($3))
		}
	| CURRENT_TIMESTAMP
		{
			$$ = makeSQLValueFunction(ast.SVFOP_CURRENT_TIMESTAMP, -1)
		}
	| CURRENT_TIMESTAMP '(' Iconst ')'
		{
			$$ = makeSQLValueFunction(ast.SVFOP_CURRENT_TIMESTAMP_N, int($3))
		}
	| LOCALTIME
		{
			$$ = makeSQLValueFunction(ast.SVFOP_LOCALTIME, -1)
		}
	| LOCALTIME '(' Iconst ')'
		{
			$$ = makeSQLValueFunction(ast.SVFOP_LOCALTIME_N, int($3))
		}
	| LOCALTIMESTAMP
		{
			$$ = makeSQLValueFunction(ast.SVFOP_LOCALTIMESTAMP, -1)
		}
	| LOCALTIMESTAMP '(' Iconst ')'
		{
			$$ = makeSQLValueFunction(ast.SVFOP_LOCALTIMESTAMP_N, int($3))
		}
	| CURRENT_ROLE
		{
			$$ = makeSQLValueFunction(ast.SVFOP_CURRENT_ROLE, -1)
		}
	| CURRENT_USER
		{
			$$ = makeSQLValueFunction(ast.SVFOP_CURRENT_USER, -1)
		}
	| SESSION_USER
		{
			$$ = makeSQLValueFunction(ast.SVFOP_SESSION_USER, -1)
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
			$$ = makeSQLValueFunction(ast.SVFOP_USER, -1)
		}
	| CURRENT_CATALOG
		{
			$$ = makeSQLValueFunction(ast.SVFOP_CURRENT_CATALOG, -1)
		}
	| CURRENT_SCHEMA
		{
			$$ = makeSQLValueFunction(ast.SVFOP_CURRENT_SCHEMA, -1)
		}
	| CAST '(' a_expr AS Typename ')'
		{
			$$ = makeTypeCast($3, $5)
		}
	| CAST '(' a_expr AS Typename opt_collate ')'
		{
			cast := makeTypeCast($3, $5)
			if c, ok := cast.(*ast.FuncCastExpr); ok {
				c.Collate = collationName($6)
			}
			$$ = cast
		}
	| NULLIF '(' a_expr ',' a_expr ')'
		{
			$$ = ast.NewFuncCallExpr("nullif", $3, $5)
		}
	| COALESCE '(' expr_list ')'
		{
			$$ = ast.NewFuncCallExpr("coalesce", $3...)
		}
	| GREATEST '(' expr_list ')'
		{
			$$ = ast.NewFuncCallExpr("greatest", $3...)
		}
	| LEAST '(' expr_list ')'
		{
			$$ = ast.NewFuncCallExpr("least", $3...)
		}
	| EXTRACT '(' extract_list ')'
		{
			$$ = &ast.FuncCallExpr{
				FnName: model.NewCIStr("extract"),
       Args: exprsFromList($3),
   }
		}
	| NORMALIZE '(' a_expr ')'
		{
			$$ = &ast.FuncCall{
				Funcname:   makeFuncName("pg_catalog", "normalize"),
				Args:       makeList($3),
				FuncFormat: int(ast.COERCE_SQL_SYNTAX),
				Location:   -1,
			}
		}
	| NORMALIZE '(' a_expr ',' unicode_normal_form ')'
		{
			$$ = &ast.FuncCall{
				Funcname:   makeFuncName("pg_catalog", "normalize"),
				Args:       makeList2($3, makeStringConst($5)),
				FuncFormat: int(ast.COERCE_SQL_SYNTAX),
				Location:   -1,
			}
		}
	| OVERLAY '(' overlay_list ')'
		{
			$$ = &ast.FuncCallExpr{
				FnName: model.NewCIStr("pg_catalog.overlay"),
       Args: exprsFromList($3),
   }
		}
	| OVERLAY '(' func_arg_list_opt ')'
		{
			$$ = &ast.FuncCallExpr{
				FnName: model.NewCIStr("overlay"),
       Args: exprsFromList($3),
   }
		}
	| POSITION '(' position_list ')'
		{
			$$ = &ast.FuncCallExpr{
				FnName: model.NewCIStr("pg_catalog.position"),
       Args: exprsFromList($3),
   }
		}
	| SUBSTRING '(' substr_list ')'
		{
			$$ = &ast.FuncCallExpr{
				FnName: model.NewCIStr("pg_catalog.substring"),
       Args: exprsFromList($3),
   }
		}
	| SUBSTRING '(' func_arg_list_opt ')'
		{
			$$ = &ast.FuncCallExpr{
				FnName: model.NewCIStr("substring"),
       Args: exprsFromList($3),
   }
		}
	| TREAT '(' a_expr AS Typename ')'
		{
			funcName := "treat"
			$$ = &ast.FuncCall{
				Funcname:   makeFuncName("pg_catalog", funcName),
				Args:       makeList($3),
				FuncFormat: int(ast.COERCE_EXPLICIT_CALL),
				Location:   -1,
			}
		}
	| TRIM '(' BOTH trim_list ')'
		{
			$$ = &ast.FuncCallExpr{
				FnName: model.NewCIStr("pg_catalog.btrim"),
       Args: exprsFromList($4),
   }
		}
	| TRIM '(' LEADING trim_list ')'
		{
			$$ = &ast.FuncCallExpr{
				FnName: model.NewCIStr("pg_catalog.ltrim"),
       Args: exprsFromList($4),
   }
		}
	| TRIM '(' TRAILING trim_list ')'
		{
			$$ = &ast.FuncCallExpr{
				FnName: model.NewCIStr("pg_catalog.rtrim"),
       Args: exprsFromList($4),
   }
		}
	| TRIM '(' trim_list ')'
		{
			$$ = &ast.FuncCallExpr{
				FnName: model.NewCIStr("pg_catalog.btrim"),
       Args: exprsFromList($3),
   }
		}
	| GROUPING '(' expr_list ')'
		{
			$$ = &ast.GroupingFunc{
				Args:     nodeList($3),
				Location: -1,
			}
		}
	| XMLCONCAT '(' expr_list ')'
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
	| XMLELEMENT '(' NAME_P ColLabel ',' xml_attributes ')'
		{
			$$ = &ast.XmlExpr{
				Op:        ast.IS_XMLELEMENT,
				Name:      $4,
				NamedArgs: nodeList($6),
				Location:  -1,
			}
		}
	| XMLELEMENT '(' NAME_P ColLabel ',' expr_list ')'
		{
			$$ = &ast.XmlExpr{
				Op:       ast.IS_XMLELEMENT,
				Name:     $4,
				Args:     nodeList($6),
				Location: -1,
			}
		}
	| XMLELEMENT '(' NAME_P ColLabel ',' xml_attributes ',' expr_list ')'
		{
			$$ = &ast.XmlExpr{
				Op:        ast.IS_XMLELEMENT,
				Name:      $4,
				NamedArgs: nodeList($6),
				Args:      nodeList($8),
				Location:  -1,
			}
		}
	| XMLEXISTS '(' c_expr xmlexists_argument ')'
		{
			/* xmlexists(A PASSING [BY REF] B [BY REF]) is converted to xmlexists(A, B) */
			$$ = &ast.FuncCall{
				Funcname:   makeFuncName("pg_catalog", "xmlexists"),
				Args:       makeList2($3, $4),
				FuncFormat: int(ast.COERCE_SQL_SYNTAX),
				Location:   -1,
			}
		}
	| XMLFOREST '(' xml_attribute_list ')'
		{
			$$ = &ast.XmlExpr{
				Op:        ast.IS_XMLFOREST,
				NamedArgs: nodeList($3),
				Location:  -1,
			}
		}
	| XMLPARSE '(' document_or_content a_expr xml_whitespace_option ')'
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
	| XMLPI '(' NAME_P ColLabel ',' a_expr ')'
		{
			$$ = &ast.XmlExpr{
				Op:       ast.IS_XMLPI,
				Name:     $4,
				Args:     makeList($6),
				Location: -1,
			}
		}
	| XMLROOT '(' a_expr ',' xml_root_version opt_xml_root_standalone ')'
		{
			$$ = &ast.XmlExpr{
				Op:       ast.IS_XMLROOT,
				Args:     &ast.List{Items: []ast.Node{$3, $5, $6}},
				Location: -1,
			}
		}
	| XMLSERIALIZE '(' document_or_content a_expr AS SimpleTypename xml_indent_option ')'
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
	| JSON_OBJECT '(' func_arg_list ')'
		{
			/* Legacy json_object() function call */
			$$ = &ast.FuncCallExpr{
				FnName: model.NewCIStr("pg_catalog.json_object"),
       Args: exprsFromList($3),
   }
		}
	| JSON_OBJECT '(' json_name_and_value_list json_object_constructor_null_clause_opt
		json_key_uniqueness_constraint_opt json_returning_clause_opt ')'
		{
			$$ = &ast.JsonObjectConstructor{
				Exprs:        $3,
				Output:       asJsonOutput($6),
				AbsentOnNull: $4 != 0,
				UniqueKeys:   $5 != 0,
				Location:     -1,
			}
		}
	| JSON_OBJECT '(' json_returning_clause_opt ')'
		{
			$$ = &ast.JsonObjectConstructor{
				Output:   asJsonOutput($3),
				Location: -1,
			}
		}
	| JSON_ARRAY '(' json_value_expr_list json_array_constructor_null_clause_opt
		json_returning_clause_opt ')'
		{
			$$ = &ast.JsonArrayConstructor{
				Exprs:        $3,
				AbsentOnNull: $4 != 0,
				Output:       asJsonOutput($5),
				Location:     -1,
			}
		}
	| JSON_ARRAY '(' select_no_parens json_format_clause_opt json_returning_clause_opt ')'
		{
			$$ = &ast.JsonArrayQueryConstructor{
				Query:  $3,
				Output: asJsonOutput($5),
				Location: -1,
			}
		}
	| JSON_ARRAY '(' json_returning_clause_opt ')'
		{
			$$ = &ast.JsonArrayConstructor{
				Output:   asJsonOutput($3),
				Location: -1,
			}
		}
	| JSON '(' json_value_expr json_key_uniqueness_constraint_opt ')'
		{
			$$ = &ast.JsonParseExpr{
				Expr:       $3.(*ast.JsonValueExpr),
				UniqueKeys: $4 != 0,
				Location:   -1,
			}
		}
	| JSON_SCALAR '(' a_expr ')'
		{
			$$ = &ast.JsonScalarExpr{
				Expr:     $3,
				Location: -1,
			}
		}
	| JSON_SERIALIZE '(' json_value_expr json_returning_clause_opt ')'
		{
			$$ = &ast.JsonSerializeExpr{
				Expr:   $3.(*ast.JsonValueExpr),
				Output: asJsonOutput($4),
				Location: -1,
			}
		}
	| JSON_QUERY '(' json_value_expr ',' a_expr json_passing_clause_opt
		json_returning_clause_opt json_wrapper_behavior json_quotes_clause_opt
		json_behavior_clause_opt ')'
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
	| JSON_EXISTS '(' json_value_expr ',' a_expr json_passing_clause_opt
		json_on_error_clause_opt ')'
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
	| JSON_VALUE '(' json_value_expr ',' a_expr json_passing_clause_opt
		json_returning_clause_opt json_behavior_clause_opt ')'
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
extract_list:
	extract_arg FROM a_expr
		{
			$$ = []ast.ExprNode{ast.NewValueExpr($1), $3}
		}
	;

extract_arg:
	IDENT       { $$ = $1 }
	| PARAM      { $$ = "?" }
	| YEAR_P    { $$ = "year" }
	| MONTH_P   { $$ = "month" }
	| DAY_P     { $$ = "day" }
	| HOUR_P    { $$ = "hour" }
	| MINUTE_P  { $$ = "minute" }
	| SECOND_P  { $$ = "second" }
	| Sconst    { $$ = $1 }
	;

unicode_normal_form:
	NFC         { $$ = "NFC" }
	| NFD       { $$ = "NFD" }
	| NFKC      { $$ = "NFKC" }
	| NFKD      { $$ = "NFKD" }
	;

overlay_list:
	a_expr PLACING a_expr FROM a_expr FOR a_expr
		{
			$$ = []ast.ExprNode{$1, $3, $5, $7}
		}
	| a_expr PLACING a_expr FROM a_expr
		{
			$$ = []ast.ExprNode{$1, $3, $5}
		}
	;

position_list:
	b_expr IN_P b_expr
		{
			/* note: arguments reversed per PG convention */
			$$ = []ast.ExprNode{$3, $1}
		}
	;

substr_list:
	a_expr FROM a_expr FOR a_expr
		{
			$$ = []ast.ExprNode{$1, $3, $5}
		}
	| a_expr FOR a_expr FROM a_expr
		{
			$$ = []ast.ExprNode{$1, $5, $3}
		}
	| a_expr FROM a_expr
		{
			$$ = []ast.ExprNode{$1, $3}
		}
	| a_expr FOR a_expr
		{
			$$ = []ast.ExprNode{
				$1,
				ast.NewValueExpr(int64(1)),
				$3,
			}
		}
	| a_expr SIMILAR a_expr ESCAPE a_expr
		{
			$$ = []ast.ExprNode{$1, $3, $5}
		}
	| expr_list
		{
			/* comma-separated form: substring(x, 1, 3) */
			$$ = $1
		}
	;

trim_list:
	a_expr FROM expr_list
		{
			$$ = append([]ast.ExprNode{$1}, $3...)
		}
	| FROM expr_list
		{
			$$ = $2
		}
	| expr_list
		{
			$$ = $1
		}
	;

/*****************************************************************************
 *
 * XML helper rules
 *
 *****************************************************************************/

xml_root_version:
	VERSION_P a_expr
		{ $$ = $2 }
	| VERSION_P NO VALUE_P
		{ $$ = makeNullAConst() }
	;

opt_xml_root_standalone:
	',' STANDALONE_P YES_P
		{ $$ = makeIntConst(int64(ast.XML_STANDALONE_YES)) }
	| ',' STANDALONE_P NO
		{ $$ = makeIntConst(int64(ast.XML_STANDALONE_NO)) }
	| ',' STANDALONE_P NO VALUE_P
		{ $$ = makeIntConst(int64(ast.XML_STANDALONE_NO_VALUE)) }
	| /* EMPTY */
		{ $$ = makeIntConst(int64(ast.XML_STANDALONE_OMITTED)) }
	;

xml_attributes:
	XMLATTRIBUTES '(' xml_attribute_list ')'	{ $$ = $3 }
	;

xml_attribute_list:
	xml_attribute_el
		{ $$ = makeList($1) }
	| xml_attribute_list ',' xml_attribute_el
		{ $$ = appendList($1, $3) }
	;

xml_attribute_el:
	a_expr AS ColLabel
		{
			$$ = &ast.ResTarget{
				Name:     $3,
				Val:      $1,
				Location: -1,
			}
		}
	| a_expr
		{
			$$ = &ast.ResTarget{
				Val:      $1,
				Location: -1,
			}
		}
	;

document_or_content:
	DOCUMENT_P		{ $$ = int64(ast.XMLOPTION_DOCUMENT) }
	| CONTENT_P	{ $$ = int64(ast.XMLOPTION_CONTENT) }
	;

xml_indent_option:
	INDENT			{ $$ = 1 }
	| NO INDENT		{ $$ = 0 }
	| /* EMPTY */	{ $$ = 0 }
	;

xml_whitespace_option:
	PRESERVE WHITESPACE_P		{ $$ = 1 }
	| STRIP_P WHITESPACE_P		{ $$ = 0 }
	| /* EMPTY */				{ $$ = 0 }
	;

xmlexists_argument:
	PASSING c_expr
		{ $$ = $2 }
	| PASSING c_expr xml_passing_mech
		{ $$ = $2 }
	| PASSING xml_passing_mech c_expr
		{ $$ = $3 }
	| PASSING xml_passing_mech c_expr xml_passing_mech
		{ $$ = $3 }
	;

xml_passing_mech:
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

json_value_expr:
	a_expr json_format_clause_opt
		{
			$$ = &ast.JsonValueExpr{
				RawExpr: $1,
			}
		}
	;

json_format_clause:
	FORMAT_LA JSON
		{
			$$ = &ast.JsonFormat{
				FormatType: ast.JS_FORMAT_JSON,
				Location:   -1,
			}
		}
	| FORMAT_LA JSON ENCODING name
		{
			$$ = &ast.JsonFormat{
				FormatType: ast.JS_FORMAT_JSON,
				Location:   -1,
			}
		}
	;

json_format_clause_opt:
	json_format_clause      { $$ = $1 }
	| /* EMPTY */           { $$ = nil }
	;

json_returning_clause_opt:
	RETURNING Typename json_format_clause_opt
		{
			$$ = &ast.JsonOutput{
				TypeName: $2,
			}
		}
	| /* EMPTY */
		{ $$ = nil }
	;


json_behavior:
	DEFAULT a_expr
		{
			$$ = &ast.JsonBehavior{
				Btype: ast.JSON_BEHAVIOR_DEFAULT,
				Expr:  $2,
				Location: -1,
			}
		}
	| json_behavior_type
		{
			$$ = &ast.JsonBehavior{
				Btype: ast.JsonBehaviorType($1),
				Location: -1,
			}
		}
	;

json_behavior_type:
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

json_behavior_clause_opt:
	json_behavior ON EMPTY_P json_behavior ON ERROR_P
		{
			/* Returns a list of 2: [on_empty, on_error] */
			$$ = &ast.List{Items: []ast.Node{$1, $4}}
		}
	| json_behavior ON EMPTY_P
		{
			$$ = &ast.List{Items: []ast.Node{$1, nil}}
		}
	| json_behavior ON ERROR_P
		{
			$$ = &ast.List{Items: []ast.Node{nil, $1}}
		}
	| /* EMPTY */
		{ $$ = nil }
	;

json_on_error_clause_opt:
	json_behavior ON ERROR_P
		{ $$ = $1 }
	| /* EMPTY */
		{ $$ = nil }
	;

json_wrapper_behavior:
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

json_quotes_clause_opt:
	KEEP QUOTES ON SCALAR STRING_P  { $$ = int64(ast.JS_QUOTES_KEEP) }
	| KEEP QUOTES                   { $$ = int64(ast.JS_QUOTES_KEEP) }
	| OMIT QUOTES ON SCALAR STRING_P { $$ = int64(ast.JS_QUOTES_OMIT) }
	| OMIT QUOTES                   { $$ = int64(ast.JS_QUOTES_OMIT) }
	| /* EMPTY */                   { $$ = int64(ast.JS_QUOTES_UNSPEC) }
	;

json_predicate_type_constraint:
	JSON       %prec UNBOUNDED      { $$ = int64(ast.JS_TYPE_ANY) }
	| JSON VALUE_P                  { $$ = int64(ast.JS_TYPE_ANY) }
	| JSON ARRAY                    { $$ = int64(ast.JS_TYPE_ARRAY) }
	| JSON OBJECT_P                 { $$ = int64(ast.JS_TYPE_OBJECT) }
	| JSON SCALAR                   { $$ = int64(ast.JS_TYPE_SCALAR) }
	;

json_key_uniqueness_constraint_opt:
	WITH UNIQUE KEYS                    { $$ = 1 }
	| WITH UNIQUE      %prec UNBOUNDED  { $$ = 1 }
	| WITHOUT UNIQUE KEYS               { $$ = 0 }
	| WITHOUT UNIQUE   %prec UNBOUNDED  { $$ = 0 }
	| /* EMPTY */      %prec UNBOUNDED  { $$ = 0 }
	;

json_aggregate_func:
	JSON_OBJECTAGG '(' json_name_and_value json_object_constructor_null_clause_opt
		json_key_uniqueness_constraint_opt json_returning_clause_opt ')'
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
	| JSON_ARRAYAGG '(' json_value_expr json_array_aggregate_order_by_clause_opt
		json_array_constructor_null_clause_opt json_returning_clause_opt ')'
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

json_array_aggregate_order_by_clause_opt:
	ORDER BY sortby_list
		{
			$$ = $3
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

json_name_and_value_list:
	json_name_and_value
		{ $$ = makeList($1) }
	| json_name_and_value_list ',' json_name_and_value
		{ $$ = appendList($1, $3) }
	;

json_name_and_value:
	c_expr VALUE_P json_value_expr
		{
			$$ = &ast.JsonKeyValue{
				Key:   $1,
				Value: $3.(*ast.JsonValueExpr),
			}
		}
	| a_expr ':' json_value_expr
		{
			$$ = &ast.JsonKeyValue{
				Key:   $1,
				Value: $3.(*ast.JsonValueExpr),
			}
		}
	;

json_object_constructor_null_clause_opt:
	NULL_P ON NULL_P                { $$ = 0 }
	| ABSENT ON NULL_P              { $$ = 1 }
	| /* EMPTY */                   { $$ = 0 }
	;

json_array_constructor_null_clause_opt:
	NULL_P ON NULL_P                { $$ = 0 }
	| ABSENT ON NULL_P              { $$ = 1 }
	| /* EMPTY */                   { $$ = 1 }
	;

json_value_expr_list:
	json_value_expr
		{ $$ = makeList($1) }
	| json_value_expr_list ',' json_value_expr
		{ $$ = appendList($1, $3) }
	;

json_passing_clause_opt:
	PASSING json_arguments  { $$ = $2 }
	| /* EMPTY */           { $$ = nil }
	;

json_arguments:
	json_argument
		{ $$ = makeList($1) }
	| json_arguments ',' json_argument
		{ $$ = appendList($1, $3) }
	;

json_argument:
	json_value_expr AS ColLabel
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









func_arg_list:
	func_arg_expr
		{
			$$ = makeList($1)
		}
	| func_arg_list ',' func_arg_expr
		{
			$$ = appendList($1, $3)
		}
	;

func_arg_list_opt:
	func_arg_list
		{
			$$ = $1
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

func_arg_expr:
	a_expr { $$ = $1 }
	| param_name COLON_EQUALS a_expr
		{
			$$ = &ast.NamedArgExpr{
				Name:      $1,
				Arg:       $3,
				Argnumber: -1,
				Location:  -1,
			}
		}
	| param_name EQUALS_GREATER a_expr
		{
			$$ = &ast.NamedArgExpr{
				Name:      $1,
				Arg:       $3,
				Argnumber: -1,
				Location:  -1,
			}
		}
	;

func_name:
	type_function_name
		{
			$$ = makeList(&ast.String{Str: $1})
		}
	| ColId indirection
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

columnref:
	ColId
		{
			$$ = ast.NewColumnNameExpr("", "", $1)
		}
	| ColId indirection
		{
			$$ = columnRefWithIndirection($1, $2)
		}
	;

indirection:
	indirection_el
		{
			$$ = makeList($1)
		}
	| indirection indirection_el
		{
			$$ = appendList($1, $2)
		}
	;

indirection_el:
	'.' attr_name
		{
			$$ = &ast.String{Str: $2}
		}
	| '.' '*'
		{
			$$ = &ast.A_Star{}
		}
	| '[' a_expr ']'
		{
			$$ = &ast.A_Indices{Uidx: $2}
		}
	| '[' opt_slice_bound ':' opt_slice_bound ']'
		{
			$$ = &ast.A_Indices{IsSlice: true, Lidx: $2, Uidx: $4}
		}
	;

opt_slice_bound:
	a_expr      { $$ = $1 }
	| /* EMPTY */ { $$ = nil }
	;

opt_indirection:
	indirection { $$ = $1 }
	| /* EMPTY */ { $$ = nil }
	;

attrs:
	'.' attr_name
		{
			$$ = makeList(&ast.String{Str: $2})
		}
	| attrs '.' attr_name
		{
			$$ = appendList($1, &ast.String{Str: $3})
		}
	;

AexprConst:
	Iconst
		{
			$$ = ast.NewValueExpr($1)
		}
	| FCONST
		{
			$$ = floatValueExpr($1)
		}
	| Sconst
		{
			$$ = ast.NewValueExpr($1)
		}
	| BCONST
		{
			$$ = ast.NewValueExpr($1)
		}
	| XCONST
		{
			$$ = ast.NewValueExpr($1)
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
	| func_name Sconst
		{
			$$ = ast.NewTypeCastExpr(ast.NewValueExpr($2), typeNameFromList($1))
		}
	| func_name '(' func_arg_list opt_sort_clause ')' Sconst
		{
			$$ = ast.NewTypeCastExpr(ast.NewValueExpr($6), typeNameFromList($1))
		}
	| ConstTypename Sconst
		{
			$$ = ast.NewTypeCastExpr(ast.NewValueExpr($2), $1)
		}
	| ConstInterval Sconst opt_interval
		{
			$$ = ast.NewTypeCastExpr(ast.NewValueExpr($2), $1)
		}
	| ConstInterval '(' Iconst ')' Sconst
		{
			$$ = ast.NewTypeCastExpr(ast.NewValueExpr($5), $1)
		}
	;

Iconst:
	ICONST { $$ = $1 }
	;

Sconst:
	SCONST { $$ = $1 }
	;

// Names and identifiers
ColId:
	IDENT { $$ = $1 }
	| unreserved_keyword { $$ = $1 }
	| col_name_keyword { $$ = $1 }
	;

ColLabel:
	IDENT { $$ = $1 }
	| unreserved_keyword { $$ = $1 }
	| col_name_keyword { $$ = $1 }
	| type_func_name_keyword { $$ = $1 }
	| reserved_keyword { $$ = $1 }
	;

attr_name:
	ColLabel { $$ = $1 }
	;

name:
	ColId { $$ = $1 }
	;

qualified_name:
	ColId
		{
			$$ = makeList(&ast.String{Str: $1})
		}
	| ColId '.' attr_name
		{
			l := makeList(&ast.String{Str: $1})
			$$ = appendList(l, &ast.String{Str: $3})
		}
	| ColId '.' attr_name '.' attr_name
		{
			l := makeList(&ast.String{Str: $1})
			l = appendList(l, &ast.String{Str: $3})
			$$ = appendList(l, &ast.String{Str: $5})
		}
	;

any_name:
	ColId
		{
			$$ = makeList(&ast.String{Str: $1})
		}
	| ColId '.' any_name
		{
			$$ = prependList(&ast.String{Str: $1}, $3)
		}
	;



opt_name_list:
	name_list { $$ = $1 }
	| /* EMPTY */ { $$ = nil }
	;

name_list:
	name
		{
			$$ = makeList(&ast.String{Str: $1})
		}
	| name_list ',' name
		{
			$$ = appendList($1, &ast.String{Str: $3})
		}
	;

expr_list:
	a_expr
		{
			$$ = []ast.ExprNode{$1}
		}
	| expr_list ',' a_expr
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
	SimpleTypename opt_array_bounds
		{
			$$ = $1
		}
	| SETOF SimpleTypename opt_array_bounds
		{
			$$ = $2
		}
	| SimpleTypename ARRAY '[' Iconst ']'
		{
			$$ = $1
		}
	| SETOF SimpleTypename ARRAY '[' Iconst ']'
		{
			$$ = $2
		}
	| SimpleTypename ARRAY
		{
			$$ = $1
		}
	| SETOF SimpleTypename ARRAY
		{
			$$ = $2
		}
	;

opt_array_bounds:
	opt_array_bounds '[' ']'
		{
			$$ = appendList($1, &ast.Integer{Ival: -1})
		}
	| opt_array_bounds '[' Iconst ']'
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
	| ConstInterval opt_interval
		{
			$$ = $1
		}
	| ConstInterval '(' Iconst ')'
		{
			$$ = $1
		}
	| BOOLEAN_P       { $$ = newDataType("bool", -1, -1) }
	| JSON            { $$ = newDataType("json", -1, -1) }
	;

GenericType:
	type_function_name opt_type_modifiers
		{
			flen, dec := typeModifiers($2)
			$$ = newDataType($1, flen, dec)
		}
	| type_function_name '.' attr_name opt_type_modifiers
		{
			flen, dec := typeModifiers($4)
			$$ = newDataType($1+"."+$3, flen, dec)
		}
	;

opt_type_modifiers:
	'(' expr_list ')'  { $$ = $2 }
	| /* EMPTY */       { $$ = nil }
	;

Numeric:
	INT_P        { $$ = newDataType("int4", -1, -1) }
	| INTEGER    { $$ = newDataType("int4", -1, -1) }
	| SMALLINT   { $$ = newDataType("int2", -1, -1) }
	| BIGINT     { $$ = newDataType("int8", -1, -1) }
	| REAL       { $$ = newDataType("float4", -1, -1) }
	| FLOAT_P opt_float
		{
			$$ = $2
		}
	| DOUBLE_P PRECISION  { $$ = newDataType("float8", -1, -1) }
	| DECIMAL_P opt_type_modifiers
		{
			flen, dec := typeModifiers($2)
			$$ = newDataType("numeric", flen, dec)
		}
	| DEC opt_type_modifiers
		{
			flen, dec := typeModifiers($2)
			$$ = newDataType("numeric", flen, dec)
		}
	| NUMERIC opt_type_modifiers
		{
			flen, dec := typeModifiers($2)
			$$ = newDataType("numeric", flen, dec)
		}
	;

opt_float:
	'(' Iconst ')'
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
CHARACTER opt_varying '(' Iconst ')'
		{
			if $2 {
				$$ = newDataType("varchar", int($4), -1)
			} else {
				$$ = newDataType("bpchar", int($4), -1)
			}
		}
	| CHARACTER opt_varying
		{
			if $2 {
				$$ = newDataType("varchar", -1, -1)
			} else {
				$$ = newDataType("bpchar", -1, -1)
			}
		}
	| CHAR_P opt_varying '(' Iconst ')'
		{
			if $2 {
				$$ = newDataType("varchar", int($4), -1)
			} else {
				$$ = newDataType("bpchar", int($4), -1)
			}
		}
	| CHAR_P opt_varying
		{
			if $2 {
				$$ = newDataType("varchar", -1, -1)
			} else {
				$$ = newDataType("bpchar", -1, -1)
			}
		}
	| NATIONAL CHARACTER opt_varying '(' Iconst ')'
		{
			if $3 {
				$$ = newDataType("varchar", int($5), -1)
			} else {
				$$ = newDataType("bpchar", int($5), -1)
			}
		}
	| NATIONAL CHARACTER opt_varying
		{
			if $3 {
				$$ = newDataType("varchar", -1, -1)
			} else {
				$$ = newDataType("bpchar", -1, -1)
			}
		}
	| NATIONAL CHAR_P opt_varying '(' Iconst ')'
		{
			if $3 {
				$$ = newDataType("varchar", int($5), -1)
			} else {
				$$ = newDataType("bpchar", int($5), -1)
			}
		}
	| NATIONAL CHAR_P opt_varying
		{
			if $3 {
				$$ = newDataType("varchar", -1, -1)
			} else {
				$$ = newDataType("bpchar", -1, -1)
			}
		}
	| NCHAR opt_varying '(' Iconst ')'
		{
			if $2 {
				$$ = newDataType("varchar", int($4), -1)
			} else {
				$$ = newDataType("bpchar", int($4), -1)
			}
		}
	| NCHAR opt_varying
		{
			if $2 {
				$$ = newDataType("varchar", -1, -1)
			} else {
				$$ = newDataType("bpchar", -1, -1)
			}
		}
	| VARCHAR '(' Iconst ')'
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

opt_varying:
	VARYING { $$ = true }
	| /* EMPTY */ { $$ = false }
	;

BitWithLength:
	BIT opt_varying '(' expr_list ')'
		{
			flen, _ := typeModifiers($4)
			if $2 {
				$$ = newDataType("varbit", flen, -1)
			} else {
				$$ = newDataType("bit", flen, -1)
			}
		}
	;

BitWithoutLength:
	BIT opt_varying
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
CHARACTER opt_varying '(' Iconst ')'
		{
			if $2 {
				$$ = newDataType("varchar", int($4), -1)
			} else {
				$$ = newDataType("bpchar", int($4), -1)
			}
		}
	| CHARACTER opt_varying
		{
			if $2 {
				$$ = newDataType("varchar", -1, -1)
			} else {
				$$ = newDataType("bpchar", -1, -1)
			}
		}
	| CHAR_P opt_varying '(' Iconst ')'
		{
			if $2 {
				$$ = newDataType("varchar", int($4), -1)
			} else {
				$$ = newDataType("bpchar", int($4), -1)
			}
		}
	| CHAR_P opt_varying
		{
			if $2 {
				$$ = newDataType("varchar", -1, -1)
			} else {
				$$ = newDataType("bpchar", -1, -1)
			}
		}
	| NATIONAL CHARACTER opt_varying '(' Iconst ')'
		{
			if $3 {
				$$ = newDataType("varchar", int($5), -1)
			} else {
				$$ = newDataType("bpchar", int($5), -1)
			}
		}
	| NATIONAL CHARACTER opt_varying
		{
			if $3 {
				$$ = newDataType("varchar", -1, -1)
			} else {
				$$ = newDataType("bpchar", -1, -1)
			}
		}
	| NATIONAL CHAR_P opt_varying '(' Iconst ')'
		{
			if $3 {
				$$ = newDataType("varchar", int($5), -1)
			} else {
				$$ = newDataType("bpchar", int($5), -1)
			}
		}
	| NATIONAL CHAR_P opt_varying
		{
			if $3 {
				$$ = newDataType("varchar", -1, -1)
			} else {
				$$ = newDataType("bpchar", -1, -1)
			}
		}
	| NCHAR opt_varying '(' Iconst ')'
		{
			if $2 {
				$$ = newDataType("varchar", int($4), -1)
			} else {
				$$ = newDataType("bpchar", int($4), -1)
			}
		}
	| NCHAR opt_varying
		{
			if $2 {
				$$ = newDataType("varchar", -1, -1)
			} else {
				$$ = newDataType("bpchar", -1, -1)
			}
		}
	;
	| VARCHAR '(' Iconst ')'
		{
			$$ = newDataType("varchar", int($3), -1)
		}
	| VARCHAR
		{
			$$ = newDataType("varchar", -1, -1)
		}
	;

ConstDatetime:
	TIMESTAMP '(' Iconst ')' opt_timezone
		{
			if $5 {
				$$ = newDataType("timestamptz", int($3), -1)
			} else {
				$$ = newDataType("timestamp", int($3), -1)
			}
		}
	| TIMESTAMP opt_timezone
		{
			if $2 {
				$$ = newDataType("timestamptz", -1, -1)
			} else {
				$$ = newDataType("timestamp", -1, -1)
			}
		}
	| TIME '(' Iconst ')' opt_timezone
		{
			if $5 {
				$$ = newDataType("timetz", int($3), -1)
			} else {
				$$ = newDataType("time", int($3), -1)
			}
		}
	| TIME opt_timezone
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

opt_timezone:
	WITH_LA TIME ZONE		{ $$ = true }
	| WITHOUT_LA TIME ZONE		{ $$ = false }
	| /* EMPTY */			{ $$ = false }
	;

opt_interval:
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
	| interval_second
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
	| DAY_P TO interval_second
		{
			$$ = $3
			$$.Items[0] = makeIntConst(int64(ast.INTERVAL_MASK_DAY | ast.INTERVAL_MASK_HOUR | ast.INTERVAL_MASK_MINUTE | ast.INTERVAL_MASK_SECOND))
		}
	| HOUR_P TO MINUTE_P
		{
			$$ = makeList(makeIntConst(int64(ast.INTERVAL_MASK_HOUR | ast.INTERVAL_MASK_MINUTE)))
		}
	| HOUR_P TO interval_second
		{
			$$ = $3
			$$.Items[0] = makeIntConst(int64(ast.INTERVAL_MASK_HOUR | ast.INTERVAL_MASK_MINUTE | ast.INTERVAL_MASK_SECOND))
		}
	| MINUTE_P TO interval_second
		{
			$$ = $3
			$$.Items[0] = makeIntConst(int64(ast.INTERVAL_MASK_MINUTE | ast.INTERVAL_MASK_SECOND))
		}
	| /* EMPTY */
		{ $$ = nil }
	;

interval_second:
	SECOND_P
		{
			$$ = makeList(makeIntConst(int64(ast.INTERVAL_MASK_SECOND)))
		}
	| SECOND_P '(' Iconst ')'
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
	| NOTIFY ColId ',' Sconst
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
	LOAD file_name
		{
			$$ = &ast.LoadStmt{Path: $2}
		}
	;

file_name:
	Sconst { $$ = $1 }
	;

/*****************************************************************************
 *
 *      CLOSE cursor
 *
 *****************************************************************************/

ClosePortalStmt:
	CLOSE cursor_name
		{
			$$ = &ast.CloseCursorStmt{Name: $2}
		}
	| CLOSE ALL
		{
			$$ = &ast.CloseCursorStmt{All: true}
		}
	;

cursor_name:
	name { $$ = $1 }
	;

/*****************************************************************************
 *
 *      SET CONSTRAINTS
 *
 *****************************************************************************/

ConstraintsSetStmt:
	SET CONSTRAINTS constraints_set_list constraints_set_mode
		{
			$$ = &ast.ConstraintsSetStmt{
				Constraints: columnOptions($3),
				Deferred:    $4,
			}
		}
	;

constraints_set_list:
	ALL
		{
			$$ = nil
		}
	| qualified_name_list
		{
			$$ = $1
		}
	;

constraints_set_mode:
	DEFERRED  { $$ = true }
	| IMMEDIATE { $$ = false }
	;

qualified_name_list:
	qualified_name
		{
			$$ = makeList(makeRangeVar($1))
		}
	| qualified_name_list ',' qualified_name
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
	SET set_rest
		{
			$$ = $2
		}
	| SET LOCAL set_rest
		{
			$$ = setLocalStatement($3)
		}
	| SET SESSION set_rest
		{
			$$ = $3
		}
	;

set_rest:
	TRANSACTION transaction_mode_list
		{
			$$ = &ast.VariableSetStmt{
				Kind: ast.VAR_SET_MULTI,
				Name: "TRANSACTION",
				Args: nodeList($2),
			}
		}
	| SESSION CHARACTERISTICS AS TRANSACTION transaction_mode_list
		{
			$$ = &ast.VariableSetStmt{
				Kind: ast.VAR_SET_MULTI,
				Name: "SESSION CHARACTERISTICS",
				Args: nodeList($5),
			}
		}
	| set_rest_more
		{
			$$ = $1
		}
	;

generic_set:
	var_name TO var_list
		{
			if $1 == "role" {
				$$ = setRoleStmt($3)
			} else {
				$$ = &ast.SetStmt{
					Variables: variableAssignments($1, $3),
				}
			}
		}
	| var_name '=' var_list
		{
			$$ = &ast.SetStmt{
				Variables: variableAssignments($1, $3),
			}
		}
	| var_name TO DEFAULT
		{
			$$ = &ast.VariableSetStmt{
				Kind: ast.VAR_SET_DEFAULT,
				Name: $1,
			}
		}
	| var_name '=' DEFAULT
		{
			$$ = &ast.VariableSetStmt{
				Kind: ast.VAR_SET_DEFAULT,
				Name: $1,
			}
		}
	;

set_rest_more:
	generic_set
		{
			$$ = $1
		}
	| var_name FROM CURRENT_P
		{
			$$ = &ast.SetStmt{
				Variables: []*ast.VariableAssignment{{
					Name:  $1,
					Value: ast.NewValueExpr("CURRENT"),
				}},
			}
		}
	| TIME ZONE zone_value
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
	| CATALOG_P Sconst
		{
			yylex.Error("current database cannot be changed")
			$$ = nil
		}
	| SCHEMA Sconst
		{
			$$ = &ast.VariableSetStmt{
				Kind: ast.VAR_SET_VALUE,
				Name: "search_path",
				Args: makeList(makeStringConst($2)),
			}
		}
	| NAMES opt_encoding
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
	| ROLE NonReservedWord_or_Sconst
		{
			$$ = setRoleStmtFromName($2)
		}
	| ROLE DEFAULT
		{
			$$ = setRoleStmtFromName("default")
		}
	| SESSION AUTHORIZATION NonReservedWord_or_Sconst
		{
			$$ = &ast.SetStmt{
				Variables: []*ast.VariableAssignment{{
					Name:  "session_authorization",
					Value: ast.NewValueExpr($3),
				}},
			}
		}
	| SESSION AUTHORIZATION DEFAULT
		{
			$$ = &ast.VariableSetStmt{
				Kind: ast.VAR_SET_DEFAULT,
				Name: "session_authorization",
			}
		}
	| XML_P OPTION document_or_content
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
	| TRANSACTION SNAPSHOT Sconst
		{
			$$ = &ast.VariableSetStmt{
				Kind: ast.VAR_SET_MULTI,
				Name: "TRANSACTION SNAPSHOT",
				Args: makeList(makeStringConst($3)),
			}
		}
	;

var_name:
	ColId
		{
			$$ = $1
		}
	| var_name '.' ColId
		{
			$$ = $1 + "." + $3
		}
	;

var_list:
	var_value
		{
			$$ = makeList($1)
		}
	| var_list ',' var_value
		{
			$$ = appendList($1, $3)
		}
	;

var_value:
	opt_boolean_or_string
		{
			$$ = makeStringConst($1)
		}
	| NumericOnly
		{
			$$ = &ast.A_Const{Val: $1}
		}
	;

zone_value:
	Sconst
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

opt_encoding:
	Sconst      { $$ = $1 }
	| DEFAULT   { $$ = "" }
	| /* EMPTY */ { $$ = "" }
	;

iso_level:
	READ UNCOMMITTED  { $$ = "read uncommitted" }
	| READ COMMITTED  { $$ = "read committed" }
	| REPEATABLE READ { $$ = "repeatable read" }
	| SERIALIZABLE    { $$ = "serializable" }
	;

transaction_mode_item:
	ISOLATION LEVEL iso_level
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

transaction_mode_list:
	transaction_mode_item
		{
			$$ = makeList($1)
		}
	| transaction_mode_list ',' transaction_mode_item
		{
			$$ = appendList($1, $3)
		}
	| transaction_mode_list transaction_mode_item
		{
			$$ = appendList($1, $2)
		}
	;

transaction_mode_list_or_empty:
	transaction_mode_list { $$ = $1 }
	| /* EMPTY */         { $$ = nil }
	;

/*****************************************************************************
 *
 *      SHOW variable
 *
 *****************************************************************************/

VariableShowStmt:
	SHOW var_name
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
	RESET reset_rest
		{
			$$ = $2
		}
	;

reset_rest:
	generic_reset
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

generic_reset:
	var_name
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

PrepareStmt:
	PREPARE name prep_type_clause AS PreparableStmt
		{
			$$ = &ast.PrepareStmt{
				Name:   $2,
				Params: $3,
				Stmt:   $5.(ast.StmtNode),
			}
		}
	;

prep_type_clause:
	'(' type_list ')'  { $$ = $2 }
	| /* EMPTY */       { $$ = nil }
	;

type_list:
	Typename
		{
			$$ = []*types.FieldType{$1}
		}
	| type_list ',' Typename
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
	EXECUTE name execute_param_clause
		{
			$$ = &ast.ExecuteStmt{
				Name: $2,
				Args: exprsFromList($3),
			}
		}
	;

execute_param_clause:
	'(' expr_list ')'  { $$ = $2 }
	| /* EMPTY */       { $$ = nil }
	;

DeallocateStmt:
	DEALLOCATE name
		{
			$$ = &ast.DeallocateStmt{
				Name: $2,
			}
		}
	| DEALLOCATE PREPARE name
		{
			$$ = &ast.DeallocateStmt{
				Name: $3,
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

TruncateStmt:
	TRUNCATE opt_table relation_expr_list opt_restart_seqs opt_drop_behavior
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

opt_restart_seqs:
	CONTINUE_P IDENTITY_P { $$ = false }
	| RESTART IDENTITY_P  { $$ = true }
	| /* EMPTY */          { $$ = false }
	;

opt_table:
	TABLE
		{ $$ = true }
	| /* EMPTY */
		{ $$ = false }
	;

relation_expr_list:
	relation_expr
		{
			$$ = makeList($1)
		}
	| relation_expr_list ',' relation_expr
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
	LOCK_P opt_table relation_expr_list opt_lock opt_nowait
		{
			$$ = &ast.LockStmt{
				Relations: $3,
				Mode:      int($4),
				Nowait:    $5,
			}
		}
	;

opt_lock:
	IN_P lock_type MODE { $$ = $2 }
	| /* EMPTY */       { $$ = int64(ast.AccessExclusiveLock) }
	;

lock_type:
	ACCESS SHARE                 { $$ = int64(ast.AccessShareLock) }
	| ROW SHARE                  { $$ = int64(ast.RowShareLock) }
	| ROW EXCLUSIVE              { $$ = int64(ast.RowExclusiveLock) }
	| SHARE UPDATE EXCLUSIVE     { $$ = int64(ast.ShareUpdateExclusiveLock) }
	| SHARE                      { $$ = int64(ast.ShareLock) }
	| SHARE ROW EXCLUSIVE        { $$ = int64(ast.ShareRowExclusiveLock) }
	| EXCLUSIVE                  { $$ = int64(ast.ExclusiveLock) }
	| ACCESS EXCLUSIVE           { $$ = int64(ast.AccessExclusiveLock) }
	;

opt_nowait:
	NOWAIT      { $$ = true }
	| /* EMPTY */ { $$ = false }
	;

/*****************************************************************************
 *
 *      VACUUM / ANALYZE
 *
 *****************************************************************************/

VacuumStmt:
	VACUUM opt_full opt_freeze opt_verbose opt_analyze opt_vacuum_relation_list
		{
			n := &ast.VacuumStmt{
				Tables: tableNamesFromList($6),
			}
			var opts []string
			if $2 {
				opts = append(opts, "full")
			}
			if $3 {
				opts = append(opts, "freeze")
			}
			if $4 {
				opts = append(opts, "verbose")
			}
			if $5 {
				opts = append(opts, "analyze")
			}
			n.Options = opts
			$$ = n
		}
	| VACUUM '(' utility_option_list ')' opt_vacuum_relation_list
		{
			$$ = &ast.VacuumStmt{
				Options: defElemNames($3),
				Tables:  tableNamesFromList($5),
			}
		}
	;

AnalyzeStmt:
	analyze_keyword opt_verbose opt_vacuum_relation_list
		{
			$$ = &ast.AnalyzeTableStmt{
				TableNames: tableNamesFromList($3),
			}
		}
	| analyze_keyword '(' utility_option_list ')' opt_vacuum_relation_list
		{
			$$ = &ast.AnalyzeTableStmt{
				TableNames: tableNamesFromList($5),
			}
		}
	;

analyze_keyword:
	ANALYZE
	| ANALYSE
	;

opt_full:
	FULL        { $$ = true }
	| /* EMPTY */ { $$ = false }
	;

opt_freeze:
	FREEZE      { $$ = true }
	| /* EMPTY */ { $$ = false }
	;

opt_verbose:
	VERBOSE     { $$ = true }
	| /* EMPTY */ { $$ = false }
	;

opt_analyze:
	analyze_keyword { $$ = true }
	| /* EMPTY */   { $$ = false }
	;

vacuum_relation:
	qualified_name opt_column_list
		{
			$$ = tableNameFromList($1)
		}
	;

vacuum_relation_list:
	vacuum_relation
		{
			$$ = makeList($1)
		}
	| vacuum_relation_list ',' vacuum_relation
		{
			$$ = appendList($1, $3)
		}
	;

opt_vacuum_relation_list:
	vacuum_relation_list { $$ = $1 }
	| /* EMPTY */        { $$ = nil }
	;

/*****************************************************************************
 *
 *      CLUSTER
 *
 *****************************************************************************/

ClusterStmt:
	CLUSTER '(' utility_option_list ')' qualified_name cluster_index_specification
		{
			$$ = &ast.ClusterStmt{
				Table:   tableNameFromList($5),
				Index:   indexName($6),
				Verbose: hasDefElem($3, "verbose"),
			}
		}
	| CLUSTER '(' utility_option_list ')'
		{
			$$ = &ast.ClusterStmt{
				Verbose: hasDefElem($3, "verbose"),
			}
		}
	| CLUSTER opt_verbose qualified_name cluster_index_specification
		{
			$$ = &ast.ClusterStmt{
				Table:   tableNameFromList($3),
				Index:   indexName($4),
				Verbose: $2,
			}
		}
	| CLUSTER opt_verbose
		{
			$$ = &ast.ClusterStmt{
				Verbose: $2,
			}
		}
	/* backwards compatible syntax from before 8.3 */
	| CLUSTER opt_verbose name ON qualified_name
		{
			$$ = &ast.ClusterStmt{
				Table:   tableNameFromList($5),
				Index:   indexName($3),
				Verbose: $2,
			}
		}
	;

cluster_index_specification:
	USING name   { $$ = $2 }
	| /* EMPTY */ { $$ = "" }
	;

/*****************************************************************************
 *
 *      REINDEX
 *
 *****************************************************************************/

ReindexStmt:
	REINDEX opt_reindex_option_list reindex_target_type opt_concurrently qualified_name
		{
			$$ = &ast.ReindexStmt{
				ObjectType:   reindexObjectName($3),
				Name:         nameFromList($5),
				Concurrently: $4,
				Verbose:      hasDefElem($2, "verbose"),
			}
		}
	| REINDEX opt_reindex_option_list SCHEMA opt_concurrently name
		{
			$$ = &ast.ReindexStmt{
				ObjectType:   "SCHEMA",
				Name:         $5,
				Concurrently: $4,
				Verbose:      hasDefElem($2, "verbose"),
			}
		}
	| REINDEX opt_reindex_option_list reindex_target_multitable opt_concurrently opt_name
		{
			$$ = &ast.ReindexStmt{
				ObjectType:   reindexObjectName($3),
				Name:         $5,
				Concurrently: $4,
				Verbose:      hasDefElem($2, "verbose"),
			}
		}
	;

opt_name:
	name             { $$ = $1 }
	| /* EMPTY */    { $$ = "" }
	;

reindex_target_type:
	INDEX { $$ = int64(ast.REINDEX_OBJECT_INDEX) }
	| TABLE { $$ = int64(ast.REINDEX_OBJECT_TABLE) }
	;

reindex_target_multitable:
	SYSTEM_P { $$ = int64(ast.REINDEX_OBJECT_SYSTEM) }
	| DATABASE { $$ = int64(ast.REINDEX_OBJECT_DATABASE) }
	;

opt_reindex_option_list:
	'(' utility_option_list ')' { $$ = $2 }
	| /* EMPTY */               { $$ = nil }
	;

/*****************************************************************************
 *
 *      COMMENT ON
 *
 *****************************************************************************/


CommentStmt:
	COMMENT ON object_type_any_name any_name IS comment_text
		{
			$$ = &ast.CommentStmt{
				ObjectType: strings.ToLower(objectTypeName($3)),
				ObjectName: commentObjectName($4),
				Comment:    commentPtr($6),
			}
		}
	| COMMENT ON COLUMN columnref IS comment_text
		{
			$$ = &ast.CommentStmt{
				ObjectType: objectTypeNameOf("OBJECT_COLUMN"),
				ObjectName: commentObjectName($4),
				Comment: commentPtr($6),
			}
		}
	| COMMENT ON object_type_name name IS comment_text
		{
			$$ = &ast.CommentStmt{
				ObjectType: strings.ToLower(objectTypeName($3)),
				ObjectName: commentObjectName(&ast.String{Str: $4}),
				Comment:    commentPtr($6),
			}
		}
	| COMMENT ON TYPE_P any_name IS comment_text
		{
			$$ = &ast.CommentStmt{
				ObjectType: objectTypeNameOf("OBJECT_TYPE"),
				ObjectName: commentObjectName($4),
				Comment: commentPtr($6),
			}
		}
	| COMMENT ON DOMAIN_P any_name IS comment_text
		{
			$$ = &ast.CommentStmt{
				ObjectType: objectTypeNameOf("OBJECT_DOMAIN"),
				ObjectName: commentObjectName($4),
				Comment: commentPtr($6),
			}
		}
	| COMMENT ON AGGREGATE aggregate_with_argtypes IS comment_text
		{
			$$ = &ast.CommentStmt{
				ObjectType: objectTypeNameOf("OBJECT_AGGREGATE"),
				ObjectName: commentObjectName($4),
				Comment: commentPtr($6),
			}
		}
	| COMMENT ON FUNCTION function_with_argtypes IS comment_text
		{
			$$ = &ast.CommentStmt{
				ObjectType: objectTypeNameOf("OBJECT_FUNCTION"),
				ObjectName: functionObjectName($4),
				Comment:    commentPtr($6),
			}
		}
	| COMMENT ON PROCEDURE function_with_argtypes IS comment_text
		{
			$$ = &ast.CommentStmt{
				ObjectType: objectTypeNameOf("OBJECT_PROCEDURE"),
				ObjectName: commentObjectName($4),
				Comment: commentPtr($6),
			}
		}
	| COMMENT ON ROUTINE function_with_argtypes IS comment_text
		{
			$$ = &ast.CommentStmt{
				ObjectType: objectTypeNameOf("OBJECT_ROUTINE"),
				ObjectName: commentObjectName($4),
				Comment: commentPtr($6),
			}
		}
	| COMMENT ON OPERATOR operator_with_argtypes IS comment_text
		{
			$$ = &ast.CommentStmt{
				ObjectType: objectTypeNameOf("OBJECT_OPERATOR"),
				ObjectName: commentObjectName($4),
				Comment: commentPtr($6),
			}
		}
	| COMMENT ON CONSTRAINT name ON any_name IS comment_text
		{
			$$ = &ast.CommentStmt{
				ObjectType: objectTypeNameOf("OBJECT_TABCONSTRAINT"),
				ObjectName: commentObjectName(appendList($6, &ast.String{Str: $4})),
				Comment: commentPtr($8),
			}
		}
	| COMMENT ON CONSTRAINT name ON DOMAIN_P any_name IS comment_text
		{
			$$ = &ast.CommentStmt{
				ObjectType: objectTypeNameOf("OBJECT_DOMCONSTRAINT"),
				ObjectName: commentObjectName(&ast.List{Items: []ast.Node{typeNameNode(typeNameFromList($7)), &ast.String{Str: $4}}}),
				Comment: commentPtr($9),
			}
		}
	| COMMENT ON object_type_name_on_any_name name ON any_name IS comment_text
		{
			$$ = &ast.CommentStmt{
				ObjectType: objectTypeName($3),
				ObjectName: commentObjectName(appendList($6, &ast.String{Str: $4})),
				Comment: commentPtr($8),
			}
		}
	| COMMENT ON TRANSFORM FOR Typename LANGUAGE name IS comment_text
		{
			$$ = &ast.CommentStmt{
				ObjectType: objectTypeNameOf("OBJECT_TRANSFORM"),
				ObjectName: commentObjectName(&ast.List{Items: []ast.Node{typeNameNode($5), &ast.String{Str: $7}}}),
				Comment: commentPtr($9),
			}
		}
	| COMMENT ON OPERATOR CLASS any_name USING name IS comment_text
		{
			$$ = &ast.CommentStmt{
				ObjectType: objectTypeNameOf("OBJECT_OPCLASS"),
				ObjectName: commentObjectName(prependList(&ast.String{Str: $7}, $5)),
				Comment: commentPtr($9),
			}
		}
	| COMMENT ON OPERATOR FAMILY any_name USING name IS comment_text
		{
			$$ = &ast.CommentStmt{
				ObjectType: objectTypeNameOf("OBJECT_OPFAMILY"),
				ObjectName: commentObjectName(prependList(&ast.String{Str: $7}, $5)),
				Comment: commentPtr($9),
			}
		}
	| COMMENT ON LARGE_P OBJECT_P NumericOnly IS comment_text
		{
			$$ = &ast.CommentStmt{
				ObjectType: objectTypeNameOf("OBJECT_LARGEOBJECT"),
				ObjectName: commentObjectName($5),
				Comment: commentPtr($7),
			}
		}
	| COMMENT ON FOREIGN DATA_P WRAPPER name IS comment_text
		{
			$$ = &ast.CommentStmt{
				ObjectType: objectTypeNameOf("OBJECT_FDW"),
				ObjectName: commentObjectName(&ast.String{Str: $6}),
				Comment: commentPtr($8),
			}
		}
	| COMMENT ON CAST '(' Typename AS Typename ')' IS comment_text
		{
			$$ = &ast.CommentStmt{
				ObjectType: objectTypeNameOf("OBJECT_CAST"),
				ObjectName: commentObjectName(&ast.List{Items: []ast.Node{typeNameNode($5), typeNameNode($7)}}),
				Comment: commentPtr($10),
			}
		}
	| COMMENT ON EVENT TRIGGER name IS comment_text
		{
			$$ = &ast.CommentStmt{
				ObjectType: objectTypeNameOf("OBJECT_EVENT_TRIGGER"),
				ObjectName: commentObjectName(&ast.String{Str: $5}),
				Comment: commentPtr($7),
			}
		}
	;

comment_text:
	Sconst      { $$ = $1 }
	| NULL_P    { $$ = "" }
	;

object_type_name:
	SCHEMA       { $$ = int64(ast.OBJECT_SCHEMA) }
	| DATABASE   { $$ = int64(ast.OBJECT_DATABASE) }
	| ROLE       { $$ = int64(ast.OBJECT_ROLE) }
	| TABLESPACE { $$ = int64(ast.OBJECT_TABLESPACE) }
	| SUBSCRIPTION { $$ = int64(ast.OBJECT_SUBSCRIPTION) }
	| PUBLICATION  { $$ = int64(ast.OBJECT_PUBLICATION) }
	| SERVER       { $$ = int64(ast.OBJECT_FOREIGN_SERVER) }
	;

object_type_name_on_any_name:
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
	SECURITY LABEL opt_provider ON object_type_any_name any_name IS Sconst
		{
			$$ = &ast.SecurityLabelStmt{
				Provider:   $3,
				ObjectType: objectTypeName($5),
				ObjectName: nameListString($6),
				Label:      $8,
			}
		}
	| SECURITY LABEL opt_provider ON COLUMN any_name IS Sconst
		{
			$$ = &ast.SecurityLabelStmt{
				Provider:   $3,
				ObjectType: "COLUMN",
				ObjectName: nameListString($6),
				Label:      $8,
			}
		}
	| SECURITY LABEL opt_provider ON object_type_name name IS Sconst
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
	DECLARE cursor_name cursor_options CURSOR opt_hold FOR SelectStmt
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

cursor_options:
	/* EMPTY */
		{
			$$ = 0
		}
	| cursor_options NO SCROLL
		{
			$$ = $1 | ast.CURSOR_OPT_NO_SCROLL
		}
	| cursor_options SCROLL
		{
			$$ = $1 | ast.CURSOR_OPT_SCROLL
		}
	| cursor_options BINARY
		{
			$$ = $1 | ast.CURSOR_OPT_BINARY
		}
	| cursor_options ASENSITIVE
		{
			$$ = $1 | ast.CURSOR_OPT_ASENSITIVE
		}
	| cursor_options INSENSITIVE
		{
			$$ = $1 | ast.CURSOR_OPT_INSENSITIVE
		}
	;

opt_hold:
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
	FETCH fetch_args
		{
			n := $2.(*ast.FetchCursorStmt)
			n.IsMove = false
			$$ = n
		}
	| MOVE fetch_args
		{
			n := $2.(*ast.FetchCursorStmt)
			n.IsMove = true
			$$ = n
		}
	;

fetch_args:
	cursor_name
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "FORWARD",
				Count:     1,
				Name:      $1,
			}
		}
	| from_in cursor_name
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "FORWARD",
				Count:     1,
				Name:      $2,
			}
		}
	| NEXT opt_from_in cursor_name
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "NEXT",
				Count:     1,
				Name:      $3,
			}
		}
	| PRIOR opt_from_in cursor_name
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "PRIOR",
				Count:     1,
				Name:      $3,
			}
		}
	| FIRST_P opt_from_in cursor_name
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "FIRST",
				Count:     1,
				Name:      $3,
			}
		}
	| LAST_P opt_from_in cursor_name
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "LAST",
				Count:     1,
				Name:      $3,
			}
		}
	| ABSOLUTE_P SignedIconst opt_from_in cursor_name
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "ABSOLUTE",
				Count:     $2,
				Name:      $4,
			}
		}
	| RELATIVE_P SignedIconst opt_from_in cursor_name
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "RELATIVE",
				Count:     $2,
				Name:      $4,
			}
		}
	| SignedIconst opt_from_in cursor_name
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "FORWARD",
				Count:     $1,
				Name:      $3,
			}
		}
	| ALL opt_from_in cursor_name
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "ALL",
				Count:     ast.FETCH_ALL,
				Name:      $3,
			}
		}
	| FORWARD opt_from_in cursor_name
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "FORWARD",
				Count:     1,
				Name:      $3,
			}
		}
	| FORWARD ALL opt_from_in cursor_name
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "FORWARD",
				Count:     ast.FETCH_ALL,
				Name:      $4,
			}
		}
	| FORWARD SignedIconst opt_from_in cursor_name
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "FORWARD",
				Count:     $2,
				Name:      $4,
			}
		}
	| SignedIconst FORWARD opt_from_in cursor_name
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "FORWARD",
				Count:     $1,
				Name:      $4,
			}
		}
	| BACKWARD opt_from_in cursor_name
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "BACKWARD",
				Count:     1,
				Name:      $3,
			}
		}
	| BACKWARD SignedIconst opt_from_in cursor_name
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "BACKWARD",
				Count:     $2,
				Name:      $4,
			}
		}
	| SignedIconst BACKWARD opt_from_in cursor_name
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "BACKWARD",
				Count:     $1,
				Name:      $4,
			}
		}
	| BACKWARD ALL opt_from_in cursor_name
		{
			$$ = &ast.FetchCursorStmt{
				Direction: "BACKWARD",
				Count:     ast.FETCH_ALL,
				Name:      $4,
			}
		}
	;

from_in:
	FROM
	| IN_P
	;

opt_from_in:
	from_in
	| /* EMPTY */
	;

/*****************************************************************************
 *
 *      MERGE INTO
 *
 *****************************************************************************/

MergeStmt:
	opt_with_clause MERGE INTO relation_expr_opt_alias USING table_ref ON a_expr merge_when_list returning_clause
		{
			m := &ast.MergeStmt{
				Table:  &ast.TableRefsClause{TableRefs: &ast.Join{Left: resultSetNodeOf($4)}},
				Source: &ast.TableRefsClause{TableRefs: &ast.Join{Left: resultSetNodeOf($6)}},
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

merge_when_list:
	merge_when_clause
		{
			$$ = makeList($1)
		}
	| merge_when_list merge_when_clause
		{
			$$ = appendList($1, $2)
		}
	;

merge_when_clause:
	merge_when_tgt_matched opt_merge_when_condition THEN merge_update
		{
			n := $4.(*ast.MergeWhenClause)
			n.Matched = true
			n.Condition = $2
			$$ = n
		}
	| merge_when_tgt_matched opt_merge_when_condition THEN merge_delete
		{
			n := $4.(*ast.MergeWhenClause)
			n.Matched = true
			n.Condition = $2
			$$ = n
		}
	| merge_when_tgt_not_matched opt_merge_when_condition THEN merge_delete
		{
			n := $4.(*ast.MergeWhenClause)
			n.Matched = false
			n.Condition = $2
			$$ = n
		}
	| merge_when_tgt_not_matched opt_merge_when_condition THEN merge_update
		{
			n := $4.(*ast.MergeWhenClause)
			n.Matched = false
			n.Condition = $2
			$$ = n
		}
	| merge_when_tgt_matched opt_merge_when_condition THEN merge_insert
		{
			n := $4.(*ast.MergeWhenClause)
			n.Matched = true
			n.Condition = $2
			$$ = n
		}
	| merge_when_tgt_not_matched opt_merge_when_condition THEN merge_insert
		{
			n := $4.(*ast.MergeWhenClause)
			n.Matched = false
			n.Condition = $2
			$$ = n
		}
	| merge_when_tgt_matched opt_merge_when_condition THEN DO NOTHING
		{
			$$ = &ast.MergeWhenClause{
				Matched:   true,
				Condition: $2,
			}
		}
	| merge_when_tgt_not_matched opt_merge_when_condition THEN DO NOTHING
		{
			$$ = &ast.MergeWhenClause{
				Matched:   false,
				Condition: $2,
			}
		}
	;

merge_when_tgt_matched:
	WHEN MATCHED
		{
			$$ = int64(ast.MERGE_WHEN_MATCHED)
		}
	;

merge_when_tgt_not_matched:
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

opt_merge_when_condition:
	AND a_expr
		{
			$$ = $2
		}
	| /* EMPTY */
		{
			$$ = nil
		}
	;

merge_update:
	UPDATE SET set_clause_list
		{
			$$ = &ast.MergeWhenClause{
				Tp:         ast.MergeWhenMatchedUpdate,
				UpdateList: $3,
			}
		}
	;

merge_delete:
	DELETE_P
		{
			$$ = &ast.MergeWhenClause{
				Tp: ast.MergeWhenMatchedDelete,
			}
		}
	;

merge_insert:
	INSERT merge_values_clause
		{
			$$ = &ast.MergeWhenClause{
				Tp:     ast.MergeWhenNotMatchedInsert,
				Values: $2,
			}
		}
	| INSERT OVERRIDING override_kind VALUE_P merge_values_clause
		{
			$$ = &ast.MergeWhenClause{
				Tp:     ast.MergeWhenNotMatchedInsert,
				Values: $5,
			}
		}
	| INSERT '(' insert_column_list ')' merge_values_clause
		{
			$$ = &ast.MergeWhenClause{
				Tp:      ast.MergeWhenNotMatchedInsert,
				Columns: $3,
				Values:  $5,
			}
		}
	| INSERT '(' insert_column_list ')' OVERRIDING override_kind VALUE_P merge_values_clause
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

merge_values_clause:
	VALUES '(' expr_list ')'
		{
			$$ = $3
		}
	;

override_kind:
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
	CALL func_application
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
	DO dostmt_opt_list
		{
			$$ = &ast.DoStmt{
				Exprs: doExprs($2),
			}
		}
	;

dostmt_opt_list:
	dostmt_opt_item
		{
			$$ = makeList($1)
		}
	| dostmt_opt_list dostmt_opt_item
		{
			$$ = appendList($1, $2)
		}
	;

dostmt_opt_item:
	Sconst
		{
			$$ = &ast.DefElem{
				Defname: "as",
				Arg:     &ast.String{Str: $1},
			}
		}
	| LANGUAGE NonReservedWord_or_Sconst
		{
			$$ = &ast.DefElem{
				Defname: "language",
				Arg:     &ast.String{Str: $2},
			}
		}
	;

opt_provider:
	FOR NonReservedWord_or_Sconst { $$ = $2 }
	| /* EMPTY */                  { $$ = "" }
	;


/*****************************************************************************
 *
 *      ALTER FUNCTION / ALTER PROCEDURE / ALTER ROUTINE
 *
 *****************************************************************************/

AlterFunctionStmt:
	ALTER FUNCTION function_with_argtypes alterfunc_opt_list opt_restrict
		{
			$$ = &ast.AlterFunctionStmt{
				Objtype: "FUNCTION",
				Func:    objectName($2),
				Actions: defElemNames($2),
			}
		}
	| ALTER PROCEDURE function_with_argtypes alterfunc_opt_list opt_restrict
		{
			$$ = &ast.AlterFunctionStmt{
				Objtype: "PROCEDURE",
				Func:    objectName($2),
				Actions: defElemNames($2),
			}
		}
	| ALTER ROUTINE function_with_argtypes alterfunc_opt_list opt_restrict
		{
			$$ = &ast.AlterFunctionStmt{
				Objtype: "ROUTINE",
				Func:    objectName($2),
				Actions: defElemNames($2),
			}
		}
	;

alterfunc_opt_list:
	common_func_opt_item
		{ $$ = makeList($1) }
	| alterfunc_opt_list common_func_opt_item
		{ $$ = appendList($1, $2) }
	;

/* Ignored, merely for SQL compliance */
opt_restrict:
	RESTRICT
	| /* EMPTY */
	;

/*****************************************************************************
 *
 *      DROP FUNCTION / DROP PROCEDURE / DROP ROUTINE
 *
 *****************************************************************************/

RemoveFuncStmt:
	DROP FUNCTION function_with_argtypes_list opt_drop_behavior
		{
			$$ = dropFunctionStmt(false, $3)
		}
	| DROP FUNCTION IF_P EXISTS function_with_argtypes_list opt_drop_behavior
		{
			$$ = dropFunctionStmt(true, $5)
		}
	| DROP PROCEDURE function_with_argtypes_list opt_drop_behavior
		{
			$$ = dropProcedureStmt(false, $3)
		}
	| DROP PROCEDURE IF_P EXISTS function_with_argtypes_list opt_drop_behavior
		{
			$$ = dropProcedureStmt(true, $5)
		}
	| DROP ROUTINE function_with_argtypes_list opt_drop_behavior
		{
			$$ = dropRoutineStmt(false, $3)
		}
	| DROP ROUTINE IF_P EXISTS function_with_argtypes_list opt_drop_behavior
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
	DROP AGGREGATE aggregate_with_argtypes_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				RemoveType: int(ast.OBJECT_AGGREGATE),
				Objects:    tableNamesFromNameList($3),
				Behavior:   int($4),
				Missing_ok: false,
			}
		}
	| DROP AGGREGATE IF_P EXISTS aggregate_with_argtypes_list opt_drop_behavior
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
	DROP OPERATOR operator_with_argtypes_list opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				RemoveType: int(ast.OBJECT_OPERATOR),
				Objects:    tableNamesFromNameList($3),
				Behavior:   int($4),
				Missing_ok: false,
			}
		}
	| DROP OPERATOR IF_P EXISTS operator_with_argtypes_list opt_drop_behavior
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

function_with_argtypes_list:
	function_with_argtypes
		{ $$ = makeList($1) }
	| function_with_argtypes_list ',' function_with_argtypes
		{ $$ = appendList($1, $3) }
	;

function_with_argtypes:
	func_name func_args
		{
			$$ = &ast.ObjectWithArgs{
				Objname: $1,
				Objargs: extractArgTypes($2),
			}
		}
	| type_func_name_keyword
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
	| ColId indirection
		{
			$$ = &ast.ObjectWithArgs{
				Objname:        checkFuncName(prependList(&ast.String{Str: $1}, $2)),
				ArgsUnspecified: true,
			}
		}
	;

func_args:
	'(' func_args_list ')'
		{ $$ = $2 }
	| '(' ')'
		{ $$ = nil }
	;

func_args_list:
	func_arg
		{ $$ = makeList($1) }
	| func_args_list ',' func_arg
		{ $$ = appendList($1, $3) }
	;

/*****************************************************************************
 *
 *      aggregate_with_argtypes and supporting rules
 *
 *****************************************************************************/

aggregate_with_argtypes:
	func_name aggr_args
		{
			$$ = &ast.ObjectWithArgs{
				Objname: $1,
				Objargs: extractAggrArgTypes($2),
			}
		}
	;

aggregate_with_argtypes_list:
	aggregate_with_argtypes
		{ $$ = makeList($1) }
	| aggregate_with_argtypes_list ',' aggregate_with_argtypes
		{ $$ = appendList($1, $3) }
	;

/*****************************************************************************
 *
 *      operator_with_argtypes and supporting rules
 *
 *****************************************************************************/

operator_with_argtypes_list:
	operator_with_argtypes
		{ $$ = makeList($1) }
	| operator_with_argtypes_list ',' operator_with_argtypes
		{ $$ = appendList($1, $3) }
	;

operator_with_argtypes:
	any_operator oper_argtypes
		{
			$$ = &ast.ObjectWithArgs{
				Objname: $1,
				Objargs: $2,
			}
		}
	;

oper_argtypes:
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

CreateTrigStmt:
	CREATE opt_or_replace TRIGGER name TriggerActionTime TriggerEvents ON
	qualified_name TriggerReferencing TriggerForSpec TriggerWhen
	EXECUTE FUNCTION_or_PROCEDURE func_name '(' TriggerFuncArgs ')'
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
	| CREATE opt_or_replace CONSTRAINT TRIGGER name AFTER TriggerEvents ON
	qualified_name OptConstrFromTable ConstraintAttributeSpec
	FOR EACH ROW TriggerWhen
	EXECUTE FUNCTION_or_PROCEDURE func_name '(' TriggerFuncArgs ')'
		{
			eventsInt := $7.Items[0].(*ast.Integer).Ival
			var columns *ast.List
			if len($7.Items) > 1 && $7.Items[1] != nil {
				columns = $7.Items[1].(*ast.List)
			}
			casBits := $11
			deferrable := (casBits & int64(ast.CAS_DEFERRABLE)) != 0
			initdeferred := (casBits & int64(ast.CAS_INITIALLY_DEFERRED)) != 0
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
	| UPDATE OF columnList
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
	TransitionOldOrNew TransitionRowOrTable opt_as TransitionRelName
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
	WHEN '(' a_expr ')'  { $$ = $3 }
	| /* EMPTY */         { $$ = nil }
	;

FUNCTION_or_PROCEDURE:
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
	Iconst
		{ $$ = &ast.String{Str: intToString($1)} }
	| FCONST
		{ $$ = &ast.String{Str: $1} }
	| Sconst
		{ $$ = &ast.String{Str: $1} }
	| ColLabel
		{ $$ = &ast.String{Str: $1} }
	;

OptConstrFromTable:
	FROM qualified_name
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

no_inherit:
	NO INHERIT            { $$ = true }
	| /* EMPTY */         { $$ = false }
	;

opt_unique_null_treatment:
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
	CREATE EVENT TRIGGER name ON ColLabel
	EXECUTE FUNCTION_or_PROCEDURE func_name '(' ')'
		{
			$$ = &ast.CreateEventTrigStmt{
				Trigname:  $4,
				Eventname: $6,
				Funcname:  $9,
			}
		}
	| CREATE EVENT TRIGGER name ON ColLabel
	WHEN event_trigger_when_list
	EXECUTE FUNCTION_or_PROCEDURE func_name '(' ')'
		{
			$$ = &ast.CreateEventTrigStmt{
				Trigname:   $4,
				Eventname:  $6,
				Whenclause: $8,
				Funcname:   $11,
			}
		}
	;

event_trigger_when_list:
	event_trigger_when_item
		{ $$ = makeList($1) }
	| event_trigger_when_list AND event_trigger_when_item
		{ $$ = appendList($1, $3) }
	;

event_trigger_when_item:
	ColId IN_P '(' event_trigger_value_list ')'
		{
			$$ = &ast.DefElem{
				Defname: $1,
				Arg:     $4,
			}
		}
	;

event_trigger_value_list:
	Sconst
		{ $$ = makeList(&ast.String{Str: $1}) }
	| event_trigger_value_list ',' Sconst
		{ $$ = appendList($1, &ast.String{Str: $3}) }
	;

AlterEventTrigStmt:
	ALTER EVENT TRIGGER name enable_trigger
		{
			$$ = &ast.AlterEventTrigStmt{
				Trigname:  $4,
				Tgenabled: byte($5),
			}
		}
	;

enable_trigger:
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
	CREATE opt_or_replace RULE name AS
	ON event TO qualified_name where_clause
	DO opt_instead RuleActionList
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

event:
	SELECT    { $$ = int64(ast.CMD_SELECT) }
	| UPDATE  { $$ = int64(ast.CMD_UPDATE) }
	| DELETE_P { $$ = int64(ast.CMD_DELETE) }
	| INSERT  { $$ = int64(ast.CMD_INSERT) }
	;

opt_instead:
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
	CREATE opt_or_replace opt_trusted opt_procedural LANGUAGE name
		{
			/* Parameterless form - creates extension */
			$$ = &ast.CreatePLangStmt{
				Replace:   $2,
				Plname:    $6,
				Pltrusted: $3,
			}
		}
	| CREATE opt_or_replace opt_trusted opt_procedural LANGUAGE name
	  HANDLER handler_name opt_inline_handler opt_validator
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

opt_trusted:
	TRUSTED    { $$ = true }
	| /* EMPTY */ { $$ = false }
	;

handler_name:
	name
		{ $$ = &ast.List{Items: []ast.Node{&ast.String{Str: $1}}} }
	| name attrs
		{ $$ = prependList(&ast.String{Str: $1}, $2) }
	;

opt_inline_handler:
	INLINE_P handler_name { $$ = $2 }
	| /* EMPTY */         { $$ = nil }
	;

opt_validator:
	VALIDATOR handler_name  { $$ = $2 }
	| NO VALIDATOR          { $$ = nil }
	| /* EMPTY */           { $$ = nil }
	;

opt_procedural:
	PROCEDURAL
	| /* EMPTY */
	;


/*****************************************************************************
 *
 *      FOREIGN DATA WRAPPER statements
 *
 *****************************************************************************/

CreateFdwStmt:
	CREATE FOREIGN DATA_P WRAPPER name opt_fdw_options create_generic_options
		{
			$$ = &ast.CreateFdwStmt{
				Fdwname:     $5,
				FuncOptions: $6,
				Options:     $7,
			}
		}
	;

fdw_option:
	HANDLER handler_name
		{
			$$ = makeDefElem("handler", $2)
		}
	| NO HANDLER
		{
			$$ = makeDefElem("handler", nil)
		}
	| VALIDATOR handler_name
		{
			$$ = makeDefElem("validator", $2)
		}
	| NO VALIDATOR
		{
			$$ = makeDefElem("validator", nil)
		}
	;

fdw_options:
	fdw_option
		{ $$ = makeList($1) }
	| fdw_options fdw_option
		{ $$ = appendList($1, $2) }
	;

opt_fdw_options:
	fdw_options   { $$ = $1 }
	| /* EMPTY */ { $$ = nil }
	;

/*****************************************************************************
 *
 *      ALTER FOREIGN DATA WRAPPER
 *
 *****************************************************************************/

AlterFdwStmt:
	ALTER FOREIGN DATA_P WRAPPER name opt_fdw_options alter_generic_options
		{
			$$ = &ast.AlterFdwStmt{
				Fdwname:     $5,
				FuncOptions: $6,
				Options:     $7,
			}
		}
	| ALTER FOREIGN DATA_P WRAPPER name fdw_options
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

create_generic_options:
	OPTIONS '(' generic_option_list ')'
		{ $$ = $3 }
	| /* EMPTY */
		{ $$ = nil }
	;

generic_option_list:
	generic_option_elem
		{ $$ = makeList($1) }
	| generic_option_list ',' generic_option_elem
		{ $$ = appendList($1, $3) }
	;

alter_generic_options:
	OPTIONS '(' alter_generic_option_list ')'
		{ $$ = $3 }
	;

alter_generic_option_list:
	alter_generic_option_elem
		{ $$ = makeList($1) }
	| alter_generic_option_list ',' alter_generic_option_elem
		{ $$ = appendList($1, $3) }
	;

alter_generic_option_elem:
	generic_option_elem
		{
			$$ = $1
		}
	| SET generic_option_elem
		{
			n := $2.(*ast.DefElem)
			n.Defaction = int(ast.DEFELEM_SET)
			$$ = n
		}
	| ADD_P generic_option_elem
		{
			n := $2.(*ast.DefElem)
			n.Defaction = int(ast.DEFELEM_ADD)
			$$ = n
		}
	| DROP generic_option_name
		{
			$$ = &ast.DefElem{
				Defname:   $2,
				Defaction: int(ast.DEFELEM_DROP),
				Location:  -1,
			}
		}
	;

generic_option_elem:
	generic_option_name
		{
			$$ = &ast.DefElem{Defname: $1, Location: -1}
		}
	| generic_option_name generic_option_arg
		{
			$$ = &ast.DefElem{
				Defname:  $1,
				Arg:      $2,
				Location: -1,
			}
		}
	;

generic_option_name:
	ColLabel  { $$ = $1 }
	;

generic_option_arg:
	Sconst  { $$ = &ast.String{Str: $1} }
	;

/*****************************************************************************
 *
 *      CREATE SERVER / ALTER SERVER
 *
 *****************************************************************************/

CreateForeignServerStmt:
	CREATE SERVER name opt_type opt_foreign_server_version
	  FOREIGN DATA_P WRAPPER name create_generic_options
		{
			$$ = &ast.CreateServerStmt{
				Name:    $3,
				Wrapper: $9,
				Options: serverOptions($10),
			}
		}
	| CREATE SERVER IF_P NOT EXISTS name opt_type opt_foreign_server_version
	  FOREIGN DATA_P WRAPPER name create_generic_options
		{
			$$ = &ast.CreateServerStmt{
				Name:    $6,
				Wrapper: $12,
				Options: serverOptions($13),
			}
		}
	;

opt_type:
	TYPE_P Sconst   { $$ = $2 }
	| /* EMPTY */   { $$ = "" }
	;

foreign_server_version:
	VERSION_P Sconst   { $$ = $2 }
	| VERSION_P NULL_P { $$ = "" }
	;

opt_foreign_server_version:
	foreign_server_version { $$ = $1 }
	| /* EMPTY */          { $$ = "" }
	;

/*****************************************************************************
 *
 *      ALTER SERVER
 *
 *****************************************************************************/

AlterForeignServerStmt:
	ALTER SERVER name foreign_server_version alter_generic_options
		{
			$$ = &ast.AlterServerStmt{
				Name:    $3,
				Options: serverOptions($5),
			}
		}
	| ALTER SERVER name foreign_server_version
		{
			$$ = &ast.AlterServerStmt{
				Name: $3,
			}
		}
	| ALTER SERVER name alter_generic_options
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
	CREATE FOREIGN TABLE qualified_name
	  '(' OptTableElementList ')'
	  OptInherit SERVER name create_generic_options
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
	| CREATE FOREIGN TABLE IF_P NOT EXISTS qualified_name
	  '(' OptTableElementList ')'
	  OptInherit SERVER name create_generic_options
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
	| CREATE FOREIGN TABLE qualified_name
	  PARTITION OF qualified_name OptTypedTableElementList ForValues
	  SERVER name create_generic_options
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
	| CREATE FOREIGN TABLE IF_P NOT EXISTS qualified_name
	  PARTITION OF qualified_name OptTypedTableElementList ForValues
	  SERVER name create_generic_options
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
	IMPORT_P FOREIGN SCHEMA name import_qualification
	  FROM SERVER name INTO name create_generic_options
		{
			$$ = &ast.ImportForeignSchemaStmt{
				RemoteSchema: $4,
				Server:       $8,
				Into:         $10,
				Options:      defElemNames($11),
			}
		}
	;

import_qualification_type:
	LIMIT TO    { $$ = int64(ast.FDW_IMPORT_SCHEMA_LIMIT_TO) }
	| EXCEPT    { $$ = int64(ast.FDW_IMPORT_SCHEMA_EXCEPT) }
	;

import_qualification:
	import_qualification_type '(' relation_expr_list ')'
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
	CREATE USER MAPPING FOR auth_ident SERVER name create_generic_options
		{
			$$ = &ast.CreateUserMappingStmt{
				User:        $5.(*ast.RoleSpec),
				Servername:  $7,
				Options:     $8,
				IfNotExists: false,
			}
		}
	| CREATE USER MAPPING IF_P NOT EXISTS FOR auth_ident SERVER name create_generic_options
		{
			$$ = &ast.CreateUserMappingStmt{
				User:        $8.(*ast.RoleSpec),
				Servername:  $10,
				Options:     $11,
				IfNotExists: true,
			}
		}
	;

auth_ident:
	RoleSpec   { $$ = $1 }
	| USER
		{
			$$ = &ast.RoleSpec{
				Roletype: int(ast.ROLESPEC_CURRENT_USER),
			}
		}
	;

DropUserMappingStmt:
	DROP USER MAPPING FOR auth_ident SERVER name
		{
			$$ = &ast.DropUserMappingStmt{
				User:       $5.(*ast.RoleSpec),
				Servername: $7,
				MissingOk:  false,
			}
		}
	| DROP USER MAPPING IF_P EXISTS FOR auth_ident SERVER name
		{
			$$ = &ast.DropUserMappingStmt{
				User:       $7.(*ast.RoleSpec),
				Servername: $9,
				MissingOk:  true,
			}
		}
	;

AlterUserMappingStmt:
	ALTER USER MAPPING FOR auth_ident SERVER name alter_generic_options
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

CreateTableSpaceStmt:
	CREATE TABLESPACE name OptTableSpaceOwner LOCATION Sconst opt_reloptions
		{
			$$ = &ast.CreateTablespaceStmt{
				Name:    $3,
				Options: tablespaceOptions($6, $7),
			}
		}
	;

OptTableSpaceOwner:
	OWNER RoleSpec { $$ = $2 }
	| /* EMPTY */  { $$ = nil }
	;

DropServerStmt:
	DROP SERVER name_list opt_drop_behavior
		{
			$$ = &ast.DropServerStmt{
				Name: firstName($3),
			}
		}
	| DROP SERVER IF_P EXISTS name_list opt_drop_behavior
		{
			$$ = &ast.DropServerStmt{
				Name:     firstName($5),
				IfExists: true,
			}
		}
	;

DropTableSpaceStmt:
	DROP TABLESPACE name
		{
			$$ = &ast.DropTablespaceStmt{
				Name: $3,
			}
		}
	| DROP TABLESPACE IF_P EXISTS name
		{
			$$ = &ast.DropTablespaceStmt{
				Name:     $5,
				IfExists: true,
			}
		}
	;

AlterTblSpcStmt:
	ALTER TABLESPACE name SET reloptions
		{
			$$ = &ast.AlterTablespaceStmt{
				Name:    $3,
				Options: tablespaceOptionList($5),
			}
		}
	| ALTER TABLESPACE name RESET reloptions
		{
			$$ = &ast.AlterTablespaceStmt{
				Name:    $3,
				Options: tablespaceOptionList($5),
			}
		}
	;

reloptions:
	'(' reloption_list ')'    { $$ = $2 }
	;

opt_reloptions:
	WITH reloptions   { $$ = $2 }
	| /* EMPTY */     { $$ = nil }
	;

reloption_list:
	reloption_elem
		{ $$ = makeList($1) }
	| reloption_list ',' reloption_elem
		{ $$ = appendList($1, $3) }
	;

reloption_elem:
	ColLabel '=' def_arg
		{
			$$ = makeDefElem($1, $3)
		}
	| ColLabel
		{
			$$ = makeDefElem($1, nil)
		}
	| ColLabel '.' ColLabel '=' def_arg
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
	CREATE EXTENSION name opt_with create_extension_opt_list
		{
			$$ = &ast.GenericStmt{
				Kind: "CREATE",
				Raw:  "EXTENSION " + $3,
			}
		}
	| CREATE EXTENSION IF_P NOT EXISTS name opt_with create_extension_opt_list
		{
			$$ = &ast.GenericStmt{
				Kind: "CREATE",
				Raw:  "EXTENSION " + $6,
			}
		}
	;

create_extension_opt_list:
	create_extension_opt_list create_extension_opt_item
		{ $$ = appendList($1, $2) }
	| /* EMPTY */
		{ $$ = nil }
	;

create_extension_opt_item:
	SCHEMA name
		{
			$$ = makeDefElem("schema", &ast.String{Str: $2})
		}
	| VERSION_P NonReservedWord_or_Sconst
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
	ALTER EXTENSION name UPDATE alter_extension_opt_list
		{
			$$ = &ast.AlterExtensionStmt{
				Extname: $3,
				Options: $5,
			}
		}
	;

alter_extension_opt_list:
	alter_extension_opt_list alter_extension_opt_item
		{ $$ = appendList($1, $2) }
	| /* EMPTY */
		{ $$ = nil }
	;

alter_extension_opt_item:
	TO NonReservedWord_or_Sconst
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
	ALTER EXTENSION name add_drop object_type_name name
		{
			$$ = &ast.AlterExtensionContentsStmt{
				Extname: $3,
				Action:  int($4),
				ObjectType: ast.ObjectTypeNone,
				Object:  &ast.String{Str: $6},
			}
		}
	| ALTER EXTENSION name add_drop object_type_any_name any_name
		{
			$$ = &ast.AlterExtensionContentsStmt{
				Extname: $3,
				Action:  int($4),
				ObjectType: ast.ObjectTypeNone,
				Object:  &ast.String{Str: $2},
			}
		}
	| ALTER EXTENSION name add_drop AGGREGATE aggregate_with_argtypes
		{
			$$ = &ast.AlterExtensionContentsStmt{
				Extname: $3,
				Action:  int($4),
				Objtype: "OBJECT",
				Object:  &ast.String{Str: $2},
			}
		}
	| ALTER EXTENSION name add_drop FUNCTION function_with_argtypes
		{
			$$ = &ast.AlterExtensionContentsStmt{
				Extname: $3,
				Action:  int($4),
				Objtype: "OBJECT",
				Object:  &ast.String{Str: $2},
			}
		}
	| ALTER EXTENSION name add_drop PROCEDURE function_with_argtypes
		{
			$$ = &ast.AlterExtensionContentsStmt{
				Extname: $3,
				Action:  int($4),
				Objtype: "OBJECT",
				Object:  &ast.String{Str: $2},
			}
		}
	| ALTER EXTENSION name add_drop ROUTINE function_with_argtypes
		{
			$$ = &ast.AlterExtensionContentsStmt{
				Extname: $3,
				Action:  int($4),
				Objtype: "OBJECT",
				Object:  &ast.String{Str: $2},
			}
		}
	| ALTER EXTENSION name add_drop OPERATOR operator_with_argtypes
		{
			$$ = &ast.AlterExtensionContentsStmt{
				Extname: $3,
				Action:  int($4),
				Objtype: "OBJECT",
				Object:  &ast.String{Str: $2},
			}
		}
	| ALTER EXTENSION name add_drop OPERATOR CLASS any_name USING name
		{
			$$ = &ast.AlterExtensionContentsStmt{
				Extname: $3,
				Action:  int($4),
				Objtype: "OBJECT",
				Object:  prependList(&ast.String{Str: $9}, $7),
			}
		}
	| ALTER EXTENSION name add_drop OPERATOR FAMILY any_name USING name
		{
			$$ = &ast.AlterExtensionContentsStmt{
				Extname: $3,
				Action:  int($4),
				Objtype: "OBJECT",
				Object:  prependList(&ast.String{Str: $9}, $7),
			}
		}
	| ALTER EXTENSION name add_drop DOMAIN_P Typename
		{
			$$ = &ast.AlterExtensionContentsStmt{
				Extname: $3,
				Action:  int($4),
				Objtype: "OBJECT",
				Object:  &ast.String{Str: $2},
			}
		}
	| ALTER EXTENSION name add_drop TYPE_P Typename
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
	CREATE ACCESS METHOD name TYPE_P am_type HANDLER handler_name
		{
			$$ = &ast.CreateAmStmt{
				Amname:      $4,
				Amtype:      byte($6),
				HandlerName: $8,
			}
		}
	;

am_type:
	INDEX     { $$ = ast.AMTYPE_INDEX }
	| TABLE   { $$ = ast.AMTYPE_TABLE }
	;

/*****************************************************************************
 *
 *      CREATE POLICY / ALTER POLICY
 *
 *****************************************************************************/

CreatePolicyStmt:
	CREATE POLICY name ON qualified_name RowSecurityDefaultPermissive
		RowSecurityDefaultForCmd RowSecurityDefaultToRole
		RowSecurityOptionalExpr RowSecurityOptionalWithCheck
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
	ALTER POLICY name ON qualified_name RowSecurityOptionalToRole
		RowSecurityOptionalExpr RowSecurityOptionalWithCheck
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

RowSecurityOptionalExpr:
	USING '(' a_expr ')'    { $$ = $3 }
	| /* EMPTY */            { $$ = nil }
	;

RowSecurityOptionalWithCheck:
	WITH CHECK '(' a_expr ')'    { $$ = $4 }
	| /* EMPTY */                 { $$ = nil }
	;

RowSecurityDefaultToRole:
	TO role_list    { $$ = $2 }
	| /* EMPTY */
		{
			/* Default is PUBLIC */
			$$ = makeList(&ast.RoleSpec{
				Roletype: int(ast.ROLESPEC_PUBLIC),
				Location: -1,
			})
		}
	;

RowSecurityOptionalToRole:
	TO role_list    { $$ = $2 }
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
	FOR row_security_cmd    { $$ = $2 }
	| /* EMPTY */           { $$ = "all" }
	;

row_security_cmd:
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
	CREATE PUBLICATION name opt_definition
		{
			$$ = &ast.CreatePublicationStmt{
				Pubname: $3,
				Options: $4,
			}
		}
	| CREATE PUBLICATION name FOR ALL TABLES opt_definition
		{
			$$ = &ast.CreatePublicationStmt{
				Pubname:      $3,
				Options:      $7,
				ForAllTables: true,
			}
		}
	| CREATE PUBLICATION name FOR pub_obj_list opt_definition
		{
			$$ = &ast.CreatePublicationStmt{
				Pubname:    $3,
				Options:    $6,
				Pubobjects: $5,
			}
		}
	;

AlterPublicationStmt:
	ALTER PUBLICATION name SET definition
		{
			$$ = &ast.AlterPublicationStmt{
				Pubname: $3,
				Options: $5,
			}
		}
	| ALTER PUBLICATION name ADD_P pub_obj_list
		{
			$$ = &ast.AlterPublicationStmt{
				Pubname:    $3,
				Pubobjects: $5,
				Action:     ast.DEFELEM_ADD,
			}
		}
	| ALTER PUBLICATION name SET pub_obj_list
		{
			$$ = &ast.AlterPublicationStmt{
				Pubname:    $3,
				Pubobjects: $5,
				Action:     ast.DEFELEM_SET,
			}
		}
	| ALTER PUBLICATION name DROP pub_obj_list
		{
			$$ = &ast.AlterPublicationStmt{
				Pubname:    $3,
				Pubobjects: $5,
				Action:     ast.DEFELEM_DROP,
			}
		}
	;

opt_definition:
	WITH definition    { $$ = $2 }
	| /* EMPTY */      { $$ = nil }
	;

PublicationObjSpec:
	TABLE relation_expr opt_column_list OptWhereClause
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
	| relation_expr opt_column_list OptWhereClause
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
	WHERE '(' a_expr ')'   { $$ = makeList($3) }
	| /* EMPTY */           { $$ = nil }
	;

pub_obj_list:
	PublicationObjSpec
		{ $$ = makeList($1) }
	| pub_obj_list ',' PublicationObjSpec
		{ $$ = appendList($1, $3) }
	;

/*****************************************************************************
 *
 *      CREATE SUBSCRIPTION / ALTER SUBSCRIPTION / DROP SUBSCRIPTION
 *
 *****************************************************************************/

CreateSubscriptionStmt:
	CREATE SUBSCRIPTION name CONNECTION Sconst PUBLICATION name_list opt_definition
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
	ALTER SUBSCRIPTION name SET definition
		{
			$$ = &ast.AlterSubscriptionStmt{
				Kind:    ast.ALTER_SUBSCRIPTION_OPTIONS,
				Subname: $3,
				Options: $5,
			}
		}
	| ALTER SUBSCRIPTION name CONNECTION Sconst
		{
			$$ = &ast.AlterSubscriptionStmt{
				Kind:     ast.ALTER_SUBSCRIPTION_CONNECTION,
				Subname:  $3,
				Conninfo: $5,
			}
		}
	| ALTER SUBSCRIPTION name REFRESH PUBLICATION opt_definition
		{
			$$ = &ast.AlterSubscriptionStmt{
				Kind:    ast.ALTER_SUBSCRIPTION_REFRESH,
				Subname: $3,
				Options: $6,
			}
		}
	| ALTER SUBSCRIPTION name ADD_P PUBLICATION name_list opt_definition
		{
			$$ = &ast.AlterSubscriptionStmt{
				Kind:        ast.ALTER_SUBSCRIPTION_ADD_PUBLICATION,
				Subname:     $3,
				Publication: $6,
				Options:     $7,
			}
		}
	| ALTER SUBSCRIPTION name DROP PUBLICATION name_list opt_definition
		{
			$$ = &ast.AlterSubscriptionStmt{
				Kind:        ast.ALTER_SUBSCRIPTION_DROP_PUBLICATION,
				Subname:     $3,
				Publication: $6,
				Options:     $7,
			}
		}
	| ALTER SUBSCRIPTION name SET PUBLICATION name_list opt_definition
		{
			$$ = &ast.AlterSubscriptionStmt{
				Kind:        ast.ALTER_SUBSCRIPTION_SET_PUBLICATION,
				Subname:     $3,
				Publication: $6,
				Options:     $7,
			}
		}
	| ALTER SUBSCRIPTION name ENABLE_P
		{
			$$ = &ast.AlterSubscriptionStmt{
				Kind:    ast.ALTER_SUBSCRIPTION_ENABLED,
				Subname: $3,
				Options: makeList(makeDefElem("enabled", &ast.Boolean{Boolval: true})),
			}
		}
	| ALTER SUBSCRIPTION name DISABLE_P
		{
			$$ = &ast.AlterSubscriptionStmt{
				Kind:    ast.ALTER_SUBSCRIPTION_ENABLED,
				Subname: $3,
				Options: makeList(makeDefElem("enabled", &ast.Boolean{Boolval: false})),
			}
		}
	| ALTER SUBSCRIPTION name SKIP definition
		{
			$$ = &ast.AlterSubscriptionStmt{
				Kind:    ast.ALTER_SUBSCRIPTION_SKIP,
				Subname: $3,
				Options: $5,
			}
		}
	;

DropSubscriptionStmt:
	DROP SUBSCRIPTION name opt_drop_behavior
		{
			$$ = &ast.DropSubscriptionStmt{
				Subname:   $3,
				MissingOk: false,
				Behavior:  ast.DropBehavior($4),
			}
		}
	| DROP SUBSCRIPTION IF_P EXISTS name opt_drop_behavior
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
	ALTER FUNCTION function_with_argtypes opt_no DEPENDS ON EXTENSION name
		{
			$$ = &ast.AlterObjectDependsStmt{
				ObjectType: ast.OBJECT_FUNCTION,
				Object:     $3,
				Extname:    &ast.String{Str: $8},
				Remove:     $4,
			}
		}
	| ALTER PROCEDURE function_with_argtypes opt_no DEPENDS ON EXTENSION name
		{
			$$ = &ast.AlterObjectDependsStmt{
				ObjectType: ast.OBJECT_PROCEDURE,
				Object:     $3,
				Extname:    &ast.String{Str: $8},
				Remove:     $4,
			}
		}
	| ALTER ROUTINE function_with_argtypes opt_no DEPENDS ON EXTENSION name
		{
			$$ = &ast.AlterObjectDependsStmt{
				ObjectType: ast.OBJECT_ROUTINE,
				Object:     $3,
				Extname:    &ast.String{Str: $8},
				Remove:     $4,
			}
		}
	| ALTER TRIGGER name ON qualified_name opt_no DEPENDS ON EXTENSION name
		{
			$$ = &ast.AlterObjectDependsStmt{
				ObjectType: ast.OBJECT_TRIGGER,
				Relation:   relationName(makeRangeVarFromAnyName($5)),
				Object:     makeList(&ast.String{Str: $3}),
				Extname:    &ast.String{Str: $10},
				Remove:     $6,
			}
		}
	| ALTER MATERIALIZED VIEW qualified_name opt_no DEPENDS ON EXTENSION name
		{
			$$ = &ast.AlterObjectDependsStmt{
				ObjectType: ast.OBJECT_MATVIEW,
				Relation:   relationName(makeRangeVarFromAnyName($4)),
				Extname:    &ast.String{Str: $9},
				Remove:     $5,
			}
		}
	| ALTER INDEX qualified_name opt_no DEPENDS ON EXTENSION name
		{
			$$ = &ast.AlterObjectDependsStmt{
				ObjectType: ast.OBJECT_INDEX,
				Relation:   relationName(makeRangeVarFromAnyName($3)),
				Extname:    &ast.String{Str: $8},
				Remove:     $4,
			}
		}
	;

opt_no:
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
	ALTER AGGREGATE aggregate_with_argtypes SET SCHEMA name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_AGGREGATE,
				Object:     $3,
				Newschema:  $6,
			}
		}
	| ALTER COLLATION any_name SET SCHEMA name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_COLLATION,
				Object:     $3,
				Newschema:  $6,
			}
		}
	| ALTER CONVERSION_P any_name SET SCHEMA name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_CONVERSION,
				Object:     $3,
				Newschema:  $6,
			}
		}
	| ALTER DOMAIN_P any_name SET SCHEMA name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_DOMAIN,
				Object:     $3,
				Newschema:  $6,
			}
		}
	| ALTER EXTENSION name SET SCHEMA name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_EXTENSION,
				Object:     &ast.String{Str: $3},
				Newschema:  $6,
			}
		}
	| ALTER FUNCTION function_with_argtypes SET SCHEMA name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_FUNCTION,
				Object:     $3,
				Newschema:  $6,
			}
		}
	| ALTER OPERATOR operator_with_argtypes SET SCHEMA name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_OPERATOR,
				Object:     $3,
				Newschema:  $6,
			}
		}
	| ALTER OPERATOR CLASS any_name USING name SET SCHEMA name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_OPCLASS,
				Object:     prependList(&ast.String{Str: $6}, $4),
				Newschema:  $9,
			}
		}
	| ALTER OPERATOR FAMILY any_name USING name SET SCHEMA name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_OPFAMILY,
				Object:     prependList(&ast.String{Str: $6}, $4),
				Newschema:  $9,
			}
		}
	| ALTER PROCEDURE function_with_argtypes SET SCHEMA name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_PROCEDURE,
				Object:     $3,
				Newschema:  $6,
			}
		}
	| ALTER ROUTINE function_with_argtypes SET SCHEMA name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_ROUTINE,
				Object:     $3,
				Newschema:  $6,
			}
		}
	| ALTER TABLE relation_expr SET SCHEMA name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_TABLE,
				Relation:   relationName($3),
				Newschema:  $6,
			}
		}
	| ALTER TABLE IF_P EXISTS relation_expr SET SCHEMA name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_TABLE,
				Relation:   relationName($5),
				Newschema:  $8,
				MissingOk:  true,
			}
		}
	| ALTER STATISTICS any_name SET SCHEMA name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_STATISTIC_EXT,
				Object:     $3,
				Newschema:  $6,
			}
		}
	| ALTER TEXT_P SEARCH PARSER any_name SET SCHEMA name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_TSPARSER,
				Object:     $5,
				Newschema:  $8,
			}
		}
	| ALTER TEXT_P SEARCH DICTIONARY any_name SET SCHEMA name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_TSDICTIONARY,
				Object:     $5,
				Newschema:  $8,
			}
		}
	| ALTER TEXT_P SEARCH TEMPLATE any_name SET SCHEMA name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_TSTEMPLATE,
				Object:     $5,
				Newschema:  $8,
			}
		}
	| ALTER TEXT_P SEARCH CONFIGURATION any_name SET SCHEMA name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_TSCONFIGURATION,
				Object:     $5,
				Newschema:  $8,
			}
		}
	| ALTER SEQUENCE qualified_name SET SCHEMA name
		{
			rv := makeRangeVarFromAnyName($3)
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_SEQUENCE,
				Relation:   relationName(rv),
				Newschema:  $6,
			}
		}
	| ALTER SEQUENCE IF_P EXISTS qualified_name SET SCHEMA name
		{
			rv := makeRangeVarFromAnyName($5)
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_SEQUENCE,
				Relation:   relationName(rv),
				Newschema:  $8,
				MissingOk:  true,
			}
		}
	| ALTER VIEW qualified_name SET SCHEMA name
		{
			rv := makeRangeVarFromAnyName($3)
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_VIEW,
				Relation:   relationName(rv),
				Newschema:  $6,
			}
		}
	| ALTER VIEW IF_P EXISTS qualified_name SET SCHEMA name
		{
			rv := makeRangeVarFromAnyName($5)
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_VIEW,
				Relation:   relationName(rv),
				Newschema:  $8,
				MissingOk:  true,
			}
		}
	| ALTER MATERIALIZED VIEW qualified_name SET SCHEMA name
		{
			rv := makeRangeVarFromAnyName($4)
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_MATVIEW,
				Relation:   relationName(rv),
				Newschema:  $7,
			}
		}
	| ALTER MATERIALIZED VIEW IF_P EXISTS qualified_name SET SCHEMA name
		{
			rv := makeRangeVarFromAnyName($6)
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_MATVIEW,
				Relation:   relationName(rv),
				Newschema:  $9,
				MissingOk:  true,
			}
		}
	| ALTER FOREIGN TABLE relation_expr SET SCHEMA name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_FOREIGN_TABLE,
				Relation:   relationName($4),
				Newschema:  $7,
			}
		}
	| ALTER FOREIGN TABLE IF_P EXISTS relation_expr SET SCHEMA name
		{
			$$ = &ast.AlterObjectSchemaStmt{
				ObjectType: ast.OBJECT_FOREIGN_TABLE,
				Relation:   relationName($6),
				Newschema:  $9,
				MissingOk:  true,
			}
		}
	| ALTER TYPE_P any_name SET SCHEMA name
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
	ALTER OPERATOR operator_with_argtypes SET '(' operator_def_list ')'
		{
			$$ = &ast.AlterOperatorStmt{
				Opername: $3.(*ast.ObjectWithArgs),
				Options:  $6,
			}
		}
	;

operator_def_list:
	operator_def_elem
		{ $$ = makeList($1) }
	| operator_def_list ',' operator_def_elem
		{ $$ = appendList($1, $3) }
	;

operator_def_elem:
	ColLabel '=' NONE
		{
			$$ = &ast.DefElem{
				Defname:  $1,
				Location: -1,
			}
		}
	| ColLabel '=' operator_def_arg
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

operator_def_arg:
	func_type
		{ $$ = typeNameNode($1) }
	| reserved_keyword
		{ $$ = &ast.String{Str: $1} }
	| qual_all_Op
		{ $$ = $1 }
	| NumericOnly
		{ $$ = $1 }
	| Sconst
		{ $$ = &ast.String{Str: $1} }
	;

/*****************************************************************************
 *
 * ALTER TYPE ... SET (...)
 *
 *****************************************************************************/

AlterTypeStmt:
	ALTER TYPE_P any_name SET '(' operator_def_list ')'
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
	ALTER AGGREGATE aggregate_with_argtypes OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_AGGREGATE,
				Object:     $3,
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER COLLATION any_name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_COLLATION,
				Object:     $3,
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER CONVERSION_P any_name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_CONVERSION,
				Object:     $3,
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER DATABASE name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_DATABASE,
				Object:     &ast.String{Str: $3},
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER DOMAIN_P any_name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_DOMAIN,
				Object:     $3,
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER FUNCTION function_with_argtypes OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_FUNCTION,
				Object:     $3,
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER opt_procedural LANGUAGE name OWNER TO RoleSpec
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
	| ALTER OPERATOR operator_with_argtypes OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_OPERATOR,
				Object:     $3,
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER OPERATOR CLASS any_name USING name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_OPCLASS,
				Object:     prependList(&ast.String{Str: $6}, $4),
				Newowner:   $9.(*ast.RoleSpec),
			}
		}
	| ALTER OPERATOR FAMILY any_name USING name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_OPFAMILY,
				Object:     prependList(&ast.String{Str: $6}, $4),
				Newowner:   $9.(*ast.RoleSpec),
			}
		}
	| ALTER PROCEDURE function_with_argtypes OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_PROCEDURE,
				Object:     $3,
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER ROUTINE function_with_argtypes OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_ROUTINE,
				Object:     $3,
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER SCHEMA name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_SCHEMA,
				Object:     &ast.String{Str: $3},
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER TYPE_P any_name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_TYPE,
				Object:     $3,
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER TABLESPACE name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_TABLESPACE,
				Object:     &ast.String{Str: $3},
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER STATISTICS any_name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_STATISTIC_EXT,
				Object:     $3,
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER TEXT_P SEARCH DICTIONARY any_name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_TSDICTIONARY,
				Object:     $5,
				Newowner:   $8.(*ast.RoleSpec),
			}
		}
	| ALTER TEXT_P SEARCH CONFIGURATION any_name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_TSCONFIGURATION,
				Object:     $5,
				Newowner:   $8.(*ast.RoleSpec),
			}
		}
	| ALTER FOREIGN DATA_P WRAPPER name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_FDW,
				Object:     &ast.String{Str: $5},
				Newowner:   $8.(*ast.RoleSpec),
			}
		}
	| ALTER SERVER name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_FOREIGN_SERVER,
				Object:     &ast.String{Str: $3},
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER EVENT TRIGGER name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_EVENT_TRIGGER,
				Object:     &ast.String{Str: $4},
				Newowner:   $7.(*ast.RoleSpec),
			}
		}
	| ALTER PUBLICATION name OWNER TO RoleSpec
		{
			$$ = &ast.AlterOwnerStmt{
				ObjectType: ast.OBJECT_PUBLICATION,
				Object:     &ast.String{Str: $3},
				Newowner:   $6.(*ast.RoleSpec),
			}
		}
	| ALTER SUBSCRIPTION name OWNER TO RoleSpec
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
	IN_P SCHEMA name_list
		{
			$$ = &ast.DefElem{
				Defname: "schemas",
				Arg:     $3,
				Location: -1,
			}
		}
	| FOR ROLE role_list
		{
			$$ = &ast.DefElem{
				Defname: "roles",
				Arg:     $3,
				Location: -1,
			}
		}
	| FOR USER role_list
		{
			$$ = &ast.DefElem{
				Defname: "roles",
				Arg:     $3,
				Location: -1,
			}
		}
	;

DefACLAction:
	GRANT privileges ON defacl_privilege_target TO grantee_list opt_grant_grant_option opt_granted_by
		{
			$$ = &ast.GrantStmt{
				
				Privs: privElems($2),
				ObjectType:     ast.ObjectTypeNone,
				Users: userSpecs($6),
				GrantOption: $7,
				Grantor: roleSpecName($8),
			}
		}
	| REVOKE privileges ON defacl_privilege_target FROM grantee_list opt_granted_by opt_drop_behavior
		{
			$$ = &ast.GrantStmt{
				IsGrant:    false,
				Privs: privElems($2),
				ObjectType:    ast.ObjectTypeNone,
				Users: userSpecs($6),
				Grantor: roleSpecName($7),
				Behavior:   ast.DropBehavior($8),
			}
		}
	| REVOKE GRANT OPTION FOR privileges ON defacl_privilege_target FROM grantee_list opt_granted_by opt_drop_behavior
		{
			$$ = &ast.GrantStmt{
				IsGrant: false,
				GrantOption: true,
				Privs: privElems($5),
				ObjectType:     ast.ObjectTypeNone,
				Users: userSpecs($9),
				Grantor: roleSpecName($10),
				Behavior:    ast.DropBehavior($11),
			}
		}
	;

defacl_privilege_target:
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
	ALTER TEXT_P SEARCH DICTIONARY any_name definition
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
	ALTER TEXT_P SEARCH CONFIGURATION any_name ADD_P MAPPING FOR name_list WITH any_name_list
		{
			$$ = &ast.AlterTSConfigurationStmt{
				Kind:      ast.ALTER_TSCONFIG_ADD_MAPPING,
				Cfgname:   $5,
				Tokentype: $9,
				Dicts:     $11,
			}
		}
	| ALTER TEXT_P SEARCH CONFIGURATION any_name ALTER MAPPING FOR name_list WITH any_name_list
		{
			$$ = &ast.AlterTSConfigurationStmt{
				Kind:      ast.ALTER_TSCONFIG_ALTER_MAPPING_FOR_TOKEN,
				Cfgname:   $5,
				Tokentype: $9,
				Dicts:     $11,
				Override:  true,
			}
		}
	| ALTER TEXT_P SEARCH CONFIGURATION any_name ALTER MAPPING REPLACE any_name WITH any_name
		{
			$$ = &ast.AlterTSConfigurationStmt{
				Kind:    ast.ALTER_TSCONFIG_REPLACE_DICT,
				Cfgname: $5,
				Dicts:   makeList2($9, $11),
				Replace: true,
			}
		}
	| ALTER TEXT_P SEARCH CONFIGURATION any_name ALTER MAPPING FOR name_list REPLACE any_name WITH any_name
		{
			$$ = &ast.AlterTSConfigurationStmt{
				Kind:      ast.ALTER_TSCONFIG_REPLACE_DICT_FOR_TOKEN,
				Cfgname:   $5,
				Tokentype: $9,
				Dicts:     makeList2($11, $13),
				Replace:   true,
			}
		}
	| ALTER TEXT_P SEARCH CONFIGURATION any_name DROP MAPPING FOR name_list
		{
			$$ = &ast.AlterTSConfigurationStmt{
				Kind:      ast.ALTER_TSCONFIG_DROP_MAPPING,
				Cfgname:   $5,
				Tokentype: $9,
			}
		}
	| ALTER TEXT_P SEARCH CONFIGURATION any_name DROP MAPPING IF_P EXISTS FOR name_list
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
	CREATE STATISTICS opt_qualified_name opt_stat_name_list ON stats_params FROM from_list
		{
			$$ = &ast.CreateStatsStmt{
				Defnames:    $3,
				StatTypes:   $4,
				Exprs:       $6,
				Relations:   nodeList($8),
				IfNotExists: false,
			}
		}
	| CREATE STATISTICS IF_P NOT EXISTS any_name opt_stat_name_list ON stats_params FROM from_list
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

opt_qualified_name:
	any_name
		{ $$ = $1 }
	| /* EMPTY */
		{ $$ = nil }
	;

opt_stat_name_list:
	'(' name_list ')'
		{ $$ = $2 }
	| /* EMPTY */
		{ $$ = nil }
	;

stats_params:
	stats_param
		{ $$ = makeList($1) }
	| stats_params ',' stats_param
		{ $$ = appendList($1, $3) }
	;

stats_param:
	ColId
		{
			$$ = &ast.StatsElem{
				Name: $1,
			}
		}
	| func_expr_windowless
		{
			$$ = &ast.StatsElem{
				Expr: $1,
			}
		}
	| '(' a_expr ')'
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
	ALTER STATISTICS any_name SET STATISTICS set_statistics_value
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
	| ALTER STATISTICS IF_P EXISTS any_name SET STATISTICS set_statistics_value
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

set_statistics_value:
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
	CREATE OPERATOR CLASS any_name opt_default FOR TYPE_P Typename USING name opt_opfamily AS opclass_item_list
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

opclass_item_list:
	opclass_item
		{ $$ = makeList($1) }
	| opclass_item_list ',' opclass_item
		{ $$ = appendList($1, $3) }
	;

opclass_item:
	OPERATOR Iconst any_operator opclass_purpose opt_recheck
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
	| OPERATOR Iconst operator_with_argtypes opclass_purpose opt_recheck
		{
			$$ = &ast.CreateOpClassItem{
				Itemtype:    ast.OPCLASS_ITEM_OPERATOR,
				Name:        $3.(*ast.ObjectWithArgs),
				Number:      int($2),
				OrderFamily: $4,
			}
		}
	| FUNCTION Iconst function_with_argtypes
		{
			$$ = &ast.CreateOpClassItem{
				Itemtype: ast.OPCLASS_ITEM_FUNCTION,
				Name:     $3.(*ast.ObjectWithArgs),
				Number:   int($2),
			}
		}
	| FUNCTION Iconst '(' type_list ')' function_with_argtypes
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

opt_default:
	DEFAULT
		{ $$ = true }
	| /* EMPTY */
		{ $$ = false }
	;

opt_opfamily:
	FAMILY any_name
		{ $$ = $2 }
	| /* EMPTY */
		{ $$ = nil }
	;

opclass_purpose:
	FOR SEARCH
		{ $$ = nil }
	| FOR ORDER BY any_name
		{ $$ = $4 }
	| /* EMPTY */
		{ $$ = nil }
	;

opt_recheck:
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
	CREATE OPERATOR FAMILY any_name USING name
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
	ALTER OPERATOR FAMILY any_name USING name ADD_P opclass_item_list
		{
			$$ = &ast.AlterOpFamilyStmt{
				Opfamilyname: $4,
				Amname:       $6,
				IsDrop:       false,
				Items:        $8,
			}
		}
	| ALTER OPERATOR FAMILY any_name USING name DROP opclass_drop_list
		{
			$$ = &ast.AlterOpFamilyStmt{
				Opfamilyname: $4,
				Amname:       $6,
				IsDrop:       true,
				Items:        $8,
			}
		}
	;

opclass_drop_list:
	opclass_drop
		{ $$ = makeList($1) }
	| opclass_drop_list ',' opclass_drop
		{ $$ = appendList($1, $3) }
	;

opclass_drop:
	OPERATOR Iconst '(' type_list ')'
		{
			$$ = &ast.CreateOpClassItem{
				Itemtype:  ast.OPCLASS_ITEM_OPERATOR,
				Number:    int($2),
				ClassArgs: nodeList($4),
			}
		}
	| FUNCTION Iconst '(' type_list ')'
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
	DROP OPERATOR CLASS any_name USING name opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList(prependList(&ast.String{Str: $6}, $4)),
				RemoveType: int(ast.OBJECT_OPCLASS),
				Behavior:   int($7),
			}
		}
	| DROP OPERATOR CLASS IF_P EXISTS any_name USING name opt_drop_behavior
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
	DROP OPERATOR FAMILY any_name USING name opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				Objects:    tableNamesFromNameList(prependList(&ast.String{Str: $6}, $4)),
				RemoveType: int(ast.OBJECT_OPFAMILY),
				Behavior:   int($7),
			}
		}
	| DROP OPERATOR FAMILY IF_P EXISTS any_name USING name opt_drop_behavior
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
	CREATE CAST '(' Typename AS Typename ')' WITH FUNCTION function_with_argtypes cast_context
		{
			$$ = &ast.CreateCastStmt{
				Sourcetype: $4,
				Targettype: $6,
				Func:       $10.(*ast.ObjectWithArgs),
				Context:    ast.CoercionContext($11),
			}
		}
	| CREATE CAST '(' Typename AS Typename ')' WITHOUT FUNCTION cast_context
		{
			$$ = &ast.CreateCastStmt{
				Sourcetype: $4,
				Targettype: $6,
				Context:    ast.CoercionContext($10),
			}
		}
	| CREATE CAST '(' Typename AS Typename ')' WITH INOUT cast_context
		{
			$$ = &ast.CreateCastStmt{
				Sourcetype: $4,
				Targettype: $6,
				Context:    ast.CoercionContext($10),
				Inout:      true,
			}
		}
	;

cast_context:
	AS IMPLICIT_P
		{ $$ = int64(ast.COERCION_IMPLICIT) }
	| AS ASSIGNMENT
		{ $$ = int64(ast.COERCION_ASSIGNMENT) }
	| /* EMPTY */
		{ $$ = int64(ast.COERCION_EXPLICIT) }
	;

DropCastStmt:
	DROP CAST opt_if_exists '(' Typename AS Typename ')' opt_drop_behavior
		{
			$$ = &ast.DropStmt{
				RemoveType: int(ast.OBJECT_CAST),
				Objects:    tableNamesFromNameList(makeList2(typeNameNode($5), typeNameNode($7))),
				Behavior:   int($9),
				Missing_ok: $3,
			}
		}
	;

opt_if_exists:
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
	CREATE opt_or_replace TRANSFORM FOR Typename LANGUAGE name '(' transform_element_list ')'
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

transform_element_list:
	FROM SQL_P WITH FUNCTION function_with_argtypes ',' TO SQL_P WITH FUNCTION function_with_argtypes
		{
			$$ = makeList2($5, $11)
		}
	| TO SQL_P WITH FUNCTION function_with_argtypes ',' FROM SQL_P WITH FUNCTION function_with_argtypes
		{
			$$ = makeList2($11, $5)
		}
	| FROM SQL_P WITH FUNCTION function_with_argtypes
		{
			$$ = makeList2($5, nil)
		}
	| TO SQL_P WITH FUNCTION function_with_argtypes
		{
			$$ = makeList2(nil, $5)
		}
	;

DropTransformStmt:
	DROP TRANSFORM opt_if_exists FOR Typename LANGUAGE name opt_drop_behavior
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
	CREATE opt_default CONVERSION_P any_name FOR Sconst TO Sconst FROM any_name
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
	DROP OWNED BY role_list opt_drop_behavior
		{
			$$ = &ast.DropOwnedStmt{
				Roles:   roleIdentities($4),
				Cascade: ast.DropBehavior($5) == ast.DROP_CASCADE,
			}
		}
	;

ReassignOwnedStmt:
	REASSIGN OWNED BY role_list TO RoleSpec
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
	CREATE OptTemp TABLE create_as_target AS SelectStmt opt_with_data
		{
			into := $4.(*ast.IntoClause)
			into.Rel.Relpersistence = relpersistenceForTemp($2)
			$$ = &ast.CreateTableStmt{
				Temporary:  $2 != 0,
				Table:      relationName(into.Rel),
				Select:     $6.(ast.ResultSetNode),
				WithOptions: "",
			}
		}
	| CREATE OptTemp TABLE IF_P NOT EXISTS create_as_target AS SelectStmt opt_with_data
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
	| CREATE OptTemp TABLE create_as_target AS EXECUTE name execute_param_clause opt_with_data
		{
			into := $4.(*ast.IntoClause)
			into.Rel.Relpersistence = relpersistenceForTemp($2)
			$$ = &ast.CreateTableStmt{
				Temporary: $2 != 0,
				Table:     relationName(into.Rel),
				Execute:   $6,
			}
		}
	| CREATE OptTemp TABLE IF_P NOT EXISTS create_as_target AS EXECUTE name execute_param_clause opt_with_data
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

create_as_target:
	qualified_name opt_column_list OptAccessMethod OptWith OnCommitOption OptTableSpace
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

opt_with_data:
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
	CREATE OptNoLog MATERIALIZED VIEW create_mv_target AS SelectStmt opt_with_data
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
	| CREATE OptNoLog MATERIALIZED VIEW IF_P NOT EXISTS create_mv_target AS SelectStmt opt_with_data
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

create_mv_target:
	qualified_name opt_column_list OptAccessMethod opt_reloptions OptTableSpace
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
	REFRESH MATERIALIZED VIEW opt_concurrently qualified_name opt_with_data
		{
			$$ = &ast.RefreshMatViewStmt{
				Concurrently:  $4,
				Table:         tableNameFromList($5),
				NoData:        !$6,
				DataSpecified: !$6,
			}
		}
	;


type_function_name:
	IDENT { $$ = $1 }
	| unreserved_keyword { $$ = $1 }
	| type_func_name_keyword { $$ = $1 }
	;

// Keyword categories
// These lists must match the keyword definitions in keywords.go

/*****************************************************************************
 *
 *      CREATE / ALTER / DROP RESOURCE GROUP
 *
 *****************************************************************************/

ResourceGroupStmt:
	CREATE RESOURCE GROUP_P name resource_group_type resource_group_options
		{
			$$ = &ast.CreateResourceGroupStmt{
				Name:           $4,
				IsSystem:       $5,
				VCPU:           resourceGroupVCPU($6),
				ThreadPriority: resourceGroupThreadPriority($6),
				State:          resourceGroupState($6),
			}
		}
	| ALTER RESOURCE GROUP_P name resource_group_options
		{
			$$ = &ast.AlterResourceGroupStmt{
				Name:           $4,
				VCPU:           resourceGroupVCPU($5),
				ThreadPriority: resourceGroupThreadPriority($5),
				State:          resourceGroupState($5),
			}
		}
	| DROP RESOURCE GROUP_P name opt_force
		{
			$$ = &ast.DropResourceGroupStmt{
				Name:  $4,
				Force: $5,
			}
		}
	;

resource_group_type:
	TYPE_P '=' SYSTEM_P
		{
			$$ = true
		}
	| TYPE_P '=' USER
		{
			$$ = false
		}
	;

resource_group_options:
	/* EMPTY */
		{
			$$ = nil
		}
	| resource_group_option_list
		{
			$$ = $1
		}
	;

resource_group_option_list:
	resource_group_option
		{
			$$ = makeList($1)
		}
	| resource_group_option_list resource_group_option
		{
			$$ = appendList($1, $2)
		}
	;

resource_group_option:
	VCPU_P '=' Iconst
		{
			$$ = makeDefElem("vcpu", ast.NewValueExpr($3))
		}
	| VCPU_P '=' Iconst '-' Iconst
		{
			$$ = makeDefElem("vcpu", &ast.List{Items: []ast.Node{
				ast.NewValueExpr($3),
				ast.NewValueExpr($5),
			}})
		}
	| THREAD_PRIORITY_P '=' Iconst
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

opt_force:
	FORCE
		{
			$$ = true
		}
	| /* EMPTY */
		{
			$$ = false
		}
	;

unreserved_keyword:
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

col_name_keyword:
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

type_func_name_keyword:
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

reserved_keyword:
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

// makeSQLValueFunction creates a SQLValueFunction node.
func makeSQLValueFunction(op ast.SVFOp, typmod int) ast.ExprNode {
	switch op {
	case ast.SVFOP_CURRENT_DATE:
		return ast.NewFuncCallExpr("current_date")
	case ast.SVFOP_CURRENT_TIME:
		return ast.NewFuncCallExpr("current_time")
	case ast.SVFOP_CURRENT_TIMESTAMP:
		return ast.NewFuncCallExpr("current_timestamp")
	case ast.SVFOP_CURRENT_USER:
		return ast.NewFuncCallExpr("current_user")
	case ast.SVFOP_SESSION_USER:
		return ast.NewFuncCallExpr("session_user")
	case ast.SVFOP_CURRENT_CATALOG:
		return ast.NewFuncCallExpr("current_catalog")
	case ast.SVFOP_CURRENT_SCHEMA:
		return ast.NewFuncCallExpr("current_schema")
	case ast.SVFOP_LOCALTIME:
		return ast.NewFuncCallExpr("localtime")
	case ast.SVFOP_LOCALTIMESTAMP:
		return ast.NewFuncCallExpr("localtimestamp")
	}
	return ast.NewFuncCallExpr("current_timestamp")
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
		return nil
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
