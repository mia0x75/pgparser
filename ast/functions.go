package ast

import (
	"fmt"
	"io"
	"strings"

	"github.com/mia0x75/pgparser/errors"

	"github.com/mia0x75/pgparser/format"
	"github.com/mia0x75/pgparser/model"
	"github.com/mia0x75/pgparser/types"
)

var (
	_ FuncNode = &AggregateFuncExpr{}
	_ FuncNode = &FuncCallExpr{}
	_ FuncNode = &FuncCastExpr{}
	_ FuncNode = &WindowFuncExpr{}
)

// Scalar function names include compatibility entries, not a PostgreSQL support list.
const (
	LogicAnd   = "and"
	Cast       = "cast"
	LeftShift  = "leftshift"
	RightShift = "rightshift"
	LogicOr    = "or"
	GE         = "ge"
	LE         = "le"
	EQ         = "eq"
	NE         = "ne"
	LT         = "lt"
	GT         = "gt"
	Plus       = "plus"
	Minus      = "minus"
	And        = "bitand"
	Or         = "bitor"
	Mod        = "mod"
	Xor        = "bitxor"
	Div        = "div"
	Mul        = "mul"
	BitNeg     = "bitneg"
	IntDiv     = "intdiv"
	LogicXor   = "xor"
	NullEQ     = "nulleq"
	In         = "in"
	Like       = "like"
	Case       = "case"
	Regexp     = "regexp"
	IsNull     = "isnull"
	IsTruth    = "istrue"  // Avoid name conflict with IsTrue in github/mia0x75/check.
	IsFalsity  = "isfalse" // Avoid name conflict with IsFalse in github/mia0x75/check.
	SetVar     = "setvar"
	Values     = "values"

	// common functions

	// math functions
	Abs      = "abs"
	Ceiling  = "ceiling"
	Log      = "log"
	Round    = "round"
	Sign     = "sign"
	Truncate = "truncate"

	// time functions
	AddDate       = "adddate"
	CurrentTime   = "current_time"
	Date          = "date"
	DateFormat    = "date_format"
	DateDiff      = "datediff"
	Day           = "day"
	Extract       = "extract"
	Hour          = "hour"
	MicroSecond   = "microsecond"
	Minute        = "minute"
	Month         = "month"
	Now           = "now"
	Second        = "second"
	Time          = "time"
	TimeFormat    = "time_format"
	Timestamp     = "timestamp"
	TimestampDiff = "timestampdiff"
	Week          = "week"
	Weekday       = "weekday"
	Year          = "year"
	YearWeek      = "yearweek"

	// string functions
	ASCII    = "ascii"
	Convert  = "convert"
	Elt      = "elt"
	Field    = "field"
	Format   = "format"
	Left     = "left"
	Length   = "length"
	Lower    = "lower"
	Position = "position"
	Quote    = "quote"
	Repeat   = "repeat"
	Replace  = "replace"
	Right    = "right"
	Trim     = "trim"
	Hex      = "hex"

	// information functions
	Charset      = "charset"
	Collation    = "collation"
	ConnectionID = "connection_id"
	CurrentUser  = "current_user"
	Database     = "database"
	Schema       = "schema"
	User         = "user"
	Version      = "version"

	// control functions
	If = "if"

	// miscellaneous functions
	Sleep = "sleep"
	UUID  = "uuid"
	// Compatibility names do not imply lock execution by this syntax-only AST.

	// encryption and compression functions
	Decode = "decode"

	// json functions
	JSONExtract  = "json_extract"
	JSONContains = "json_contains"
)

// FuncCallExpr is for function expression.
type FuncCallExpr struct {
	funcNode
	// FnName is the dotted function name used by Format and legacy callers.
	FnName model.CIStr
	// FnNameParts keeps each name part unquoted so Restore can quote and case
	// them correctly; a flattened FnName cannot be split when a part contains a dot.
	FnNameParts []string
	// Args is the function args.
	Args []ExprNode
	// Filter is the optional FILTER (WHERE expr) clause.
	Filter ExprNode
	// WithinGroup is the optional WITHIN GROUP (ORDER BY ...) clause.
	WithinGroup *OrderByClause
	// Star is true when the call is written as `count(*)`.
	Star     bool
	Distinct bool
	OrderBy  *OrderByClause
}

// mustQuoteFuncNames are the parser keywords that cannot be written as a bare
// function name: SELECT "<kw>"(1, 2) parses but SELECT <KW>(1, 2) does not, so
// if such a word appears as a function-name part it must stay quoted, or the
// restored SQL would read it back as the keyword.
// shortcut: the table is maintained by hand; regenerate it if keyword categories change.
var mustQuoteFuncNames = map[string]struct{}{
	"analyse":           {},
	"analyze":           {},
	"and":               {},
	"any":               {},
	"array":             {},
	"as":                {},
	"asc":               {},
	"asymmetric":        {},
	"between":           {},
	"bigint":            {},
	"bit":               {},
	"boolean":           {},
	"both":              {},
	"case":              {},
	"cast":              {},
	"char":              {},
	"character":         {},
	"check":             {},
	"collate":           {},
	"column":            {},
	"constraint":        {},
	"create":            {},
	"current_catalog":   {},
	"current_date":      {},
	"current_role":      {},
	"current_time":      {},
	"current_timestamp": {},
	"current_user":      {},
	"dec":               {},
	"decimal":           {},
	"default":           {},
	"deferrable":        {},
	"desc":              {},
	"do":                {},
	"else":              {},
	"end":               {},
	"except":            {},
	"extract":           {},
	"exists":            {},
	"false":             {},
	"fetch":             {},
	"float":             {},
	"for":               {},
	"foreign":           {},
	"from":              {},
	"grant":             {},
	"group":             {},
	"in":                {},
	"initially":         {},
	"inout":             {},
	"int":               {},
	"integer":           {},
	"intersect":         {},
	"interval":          {},
	"into":              {},
	"json":              {},
	"json_arrayagg":     {},
	"json_objectagg":    {},
	"json_scalar":       {},
	"json_serialize":    {},
	"json_table":        {},
	"lateral":           {},
	"leading":           {},
	"localtime":         {},
	"localtimestamp":    {},
	"merge_action":      {},
	"national":          {},
	"nchar":             {},
	"none":              {},
	"normalize":         {},
	"null":              {},
	"numeric":           {},
	"on":                {},
	"only":              {},
	"operator":          {},
	"or":                {},
	"order":             {},
	"out":               {},
	"placing":           {},
	"position":          {},
	"precision":         {},
	"primary":           {},
	"real":              {},
	"references":        {},
	"returning":         {},
	"select":            {},
	"session_user":      {},
	"setof":             {},
	"smallint":          {},
	"some":              {},
	"symmetric":         {},
	"system_user":       {},
	"table":             {},
	"then":              {},
	"time":              {},
	"timestamp":         {},
	"to":                {},
	"trailing":          {},
	"treat":             {},
	"true":              {},
	"trim":              {},
	"union":             {},
	"unique":            {},
	"user":              {},
	"using":             {},
	"values":            {},
	"varchar":           {},
	"variadic":          {},
	"when":              {},
	"window":            {},
	"with":              {},
	"xmlattributes":     {},
	"xmlelement":        {},
	"xmlexists":         {},
	"xmlnamespaces":     {},
	"xmlparse":          {},
	"xmlpi":             {},
	"xmlroot":           {},
	"xmlserialize":      {},
	"xmltable":          {},
}

// writeFuncName restores a dotted function name, quoting each part so that
// quoting, case and reserved words survive identifier folding. A single
// lowercase part that is not a keyword needing quotes is written as a keyword
// for readability; every other part is quoted.
func writeFuncName(ctx *format.RestoreCtx, parts []string, dotted string) {
	if len(parts) == 0 {
		if dotted == "" {
			return
		}
		parts = strings.Split(dotted, ".")
	}
	if len(parts) == 1 && !NeedsQuotedFuncName(parts[0]) {
		ctx.WriteKeyWord(parts[0])
		return
	}
	writeNameParts(ctx, parts)
}

// NeedsQuotedFuncName reports whether a function-name part must be quoted on
// restore. A simple lowercase word that can be read back as a bare function name
// may stay unquoted; everything else (mixed case, spaces, reserved words) must
// be quoted or folding would change the name or make the call unparseable.
func NeedsQuotedFuncName(part string) bool {
	if !simpleLowerIdent(part) {
		return true
	}
	_, reserved := mustQuoteFuncNames[part]
	return reserved
}

// simpleLowerIdent reports whether s is a bare lowercase identifier that is safe
// to fold.
func simpleLowerIdent(s string) bool {
	if s == "" || (s[0] != '_' && (s[0] < 'a' || s[0] > 'z')) {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		isLower := c >= 'a' && c <= 'z'
		isDigit := c >= '0' && c <= '9'
		if !isLower && !isDigit && c != '_' {
			return false
		}
	}
	return true
}

// Restore implements Node interface.
func (n *FuncCallExpr) Restore(ctx *format.RestoreCtx) error {
	keywordForm := n.keywordFormName()
	if keywordForm {
		// The grammar's keyword forms (EXTRACT/POSITION/TRIM) must stay bare.
		ctx.WriteKeyWord(n.FnName.O)
	} else {
		writeFuncName(ctx, n.FnNameParts, n.FnName.O)
	}
	ctx.WritePlain("(")
	if n.Distinct {
		ctx.WriteKeyWord("DISTINCT ")
	}
	switch {
	case keywordForm:
		if err := n.restoreKeywordForm(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore FuncCallExpr")
		}
	case n.Star:
		ctx.WritePlain("*")
	default:
		for i, argv := range n.Args {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			if err := argv.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore FuncCallExpr.Args %d", i)
			}
		}
	}
	if n.OrderBy != nil {
		ctx.WritePlain(" ")
		if err := n.OrderBy.Restore(ctx); err != nil {
			return err
		}
	}
	ctx.WritePlain(")")
	if n.WithinGroup != nil {
		ctx.WriteKeyWord(" WITHIN GROUP (")
		if err := n.WithinGroup.Restore(ctx); err != nil {
			return err
		}
		ctx.WritePlain(")")
	}
	if n.Filter != nil {
		ctx.WriteKeyWord(" FILTER (WHERE ")
		if err := n.Filter.Restore(ctx); err != nil {
			return err
		}
		ctx.WritePlain(")")
	}
	return ctx.Err()
}

// keywordFormName reports whether the call has the argument shape of one of the
// grammar's keyword forms. A quoted user function like "extract"(a) does not
// match, so it keeps its quoted name instead of becoming the EXTRACT keyword.
func (n *FuncCallExpr) keywordFormName() bool {
	if len(n.FnNameParts) != 0 {
		return false
	}
	switch n.FnName.L {
	case "extract":
		if len(n.Args) != 2 {
			return false
		}
		_, ok := n.Args[0].(ValueExpr)
		return ok
	case "position":
		return len(n.Args) == 2
	case "trim":
		switch len(n.Args) {
		case 1, 2:
			return true
		case 3:
			_, ok2 := n.Args[2].(ValueExpr)
			_, ok1 := n.Args[1].(ValueExpr)
			return ok1 && ok2
		}
	}
	return false
}

// restoreKeywordForm writes extract(), position() and trim() in their SQL forms.
// It is only called when keywordFormName reports a matching shape.
func (n *FuncCallExpr) restoreKeywordForm(ctx *format.RestoreCtx) error {
	switch n.FnName.L {
	case "extract":
		ctx.WriteKeyWord(n.Args[0].(ValueExpr).GetString())
		ctx.WriteKeyWord(" FROM ")
		return n.Args[1].Restore(ctx)
	case "position":
		if err := n.Args[0].Restore(ctx); err != nil {
			return err
		}
		ctx.WriteKeyWord(" IN ")
		return n.Args[1].Restore(ctx)
	case "trim":
		switch len(n.Args) {
		case 1:
			return n.Args[0].Restore(ctx)
		case 2:
			if err := n.Args[1].Restore(ctx); err != nil {
				return err
			}
			ctx.WriteKeyWord(" FROM ")
			return n.Args[0].Restore(ctx)
		case 3:
			direction := n.Args[2].(ValueExpr)
			from := n.Args[1].(ValueExpr)
			switch fmt.Sprint(direction.GetValue()) {
			case "3":
				ctx.WriteKeyWord("TRAILING ")
			case "2":
				ctx.WriteKeyWord("LEADING ")
			case "0", "1":
				ctx.WriteKeyWord("BOTH ")
			}
			if from.GetValue() != nil {
				if err := n.Args[1].Restore(ctx); err != nil {
					return err
				}
				ctx.WritePlain(" ")
			}
			ctx.WriteKeyWord("FROM ")
			return n.Args[0].Restore(ctx)
		}
	}
	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *FuncCallExpr) Format(w io.Writer) {
	_, _ = fmt.Fprintf(w, "%s(", n.FnName.L)
	if n.Star {
		_, _ = fmt.Fprint(w, "*")
	} else {
		for i, arg := range n.Args {
			arg.Format(w)
			if i != len(n.Args)-1 {
				_, _ = fmt.Fprint(w, ", ")
			}
		}
	}
	_, _ = fmt.Fprint(w, ")")
}

// Accept implements Node interface.
func (n *FuncCallExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*FuncCallExpr)
	if n.OrderBy != nil {
		node, ok := n.OrderBy.Accept(v)
		if !ok {
			return n, false
		}
		n.OrderBy = node.(*OrderByClause)
	}
	for i, val := range n.Args {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Args[i] = node.(ExprNode)
	}
	if n.Filter != nil {
		node, ok := n.Filter.Accept(v)
		if !ok {
			return n, false
		}
		n.Filter = node.(ExprNode)
	}
	if n.WithinGroup != nil {
		node, ok := n.WithinGroup.Accept(v)
		if !ok {
			return n, false
		}
		n.WithinGroup = node.(*OrderByClause)
	}
	return v.Leave(n)
}

// CastFunctionType is the type for cast function.
type CastFunctionType int

// CastFunction types
const (
	CastFunction CastFunctionType = iota + 1
	CastConvertFunction
	CastBinaryOperator
)

// FuncCastExpr retains CAST, CONVERT and BINARY forms for compatibility.
type FuncCastExpr struct {
	funcNode
	// Expr is the expression to be converted.
	Expr ExprNode
	// Tp is the conversion type.
	Tp *types.FieldType
	// FunctionType is either Cast, Convert or Binary.
	FunctionType CastFunctionType
	// Collate is the optional collation of a PostgreSQL cast.
	Collate string
}

// Restore implements Node interface.
func (n *FuncCastExpr) Restore(ctx *format.RestoreCtx) error {
	switch n.FunctionType {
	case CastFunction:
		ctx.WriteKeyWord("CAST")
		ctx.WritePlain("(")
		if err := n.Expr.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore FuncCastExpr.Expr")
		}
		ctx.WriteKeyWord(" AS ")
		n.Tp.RestoreAsCastType(ctx)
		ctx.WritePlain(")")
	case CastConvertFunction:
		ctx.WriteKeyWord("CONVERT")
		ctx.WritePlain("(")
		if err := n.Expr.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore FuncCastExpr.Expr")
		}
		ctx.WritePlain(", ")
		n.Tp.RestoreAsCastType(ctx)
		ctx.WritePlain(")")
	case CastBinaryOperator:
		ctx.WriteKeyWord("BINARY ")
		if err := n.Expr.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore FuncCastExpr.Expr")
		}
	}
	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *FuncCastExpr) Format(w io.Writer) {
	switch n.FunctionType {
	case CastFunction:
		_, _ = fmt.Fprint(w, "CAST(")
		n.Expr.Format(w)
		_, _ = fmt.Fprint(w, " AS ")
		n.Tp.FormatAsCastType(w)
		_, _ = fmt.Fprint(w, ")")
	case CastConvertFunction:
		_, _ = fmt.Fprint(w, "CONVERT(")
		n.Expr.Format(w)
		_, _ = fmt.Fprint(w, ", ")
		n.Tp.FormatAsCastType(w)
		_, _ = fmt.Fprint(w, ")")
	case CastBinaryOperator:
		_, _ = fmt.Fprint(w, "BINARY ")
		n.Expr.Format(w)
	}
}

// Accept implements Node Accept interface.
func (n *FuncCastExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*FuncCastExpr)
	node, ok := n.Expr.Accept(v)
	if !ok {
		return n, false
	}
	n.Expr = node.(ExprNode)
	return v.Leave(n)
}

// TrimDirectionType is the type for trim direction.
type TrimDirectionType int

const (
	// TrimBothDefault trims from both direction by default.
	TrimBothDefault TrimDirectionType = iota
	// TrimBoth trims from both direction with explicit notation.
	TrimBoth
	// TrimLeading trims from left.
	TrimLeading
	// TrimTrailing trims from right.
	TrimTrailing
)

// DateArithType is type for DateArith type.
type DateArithType byte

const (
	// DateArithAdd is to run adddate or date_add function option.
	DateArithAdd DateArithType = iota + 1
	// DateArithSub is to run subdate or date_sub function option.
	DateArithSub
)

const (
	// AggFuncCount is the name of Count function.
	AggFuncCount = "count"
	// AggFuncSum is the name of Sum function.
	AggFuncSum = "sum"
	// AggFuncAvg is the name of Avg function.
	AggFuncAvg = "avg"
	// AggFuncFirstRow is the name of FirstRowColumn function.
	AggFuncFirstRow = "firstrow"
	// AggFuncMax is the name of max function.
	AggFuncMax = "max"
	// AggFuncMin is the name of min function.
	AggFuncMin = "min"
	// AggFuncGroupConcat is the name of group_concat function.
	AggFuncGroupConcat = "group_concat"
	// AggFuncBitOr is the name of bit_or function.
	AggFuncBitOr = "bit_or"
	// AggFuncBitXor is the name of bit_xor function.
	AggFuncBitXor = "bit_xor"
	// AggFuncBitAnd is the name of bit_and function.
	AggFuncBitAnd = "bit_and"
	// AggFuncVarPop is the name of var_pop function
	AggFuncVarPop = "var_pop"
	// AggFuncVarSamp is the name of var_samp function
	AggFuncVarSamp = "var_samp"
	// AggFuncStddevPop is the name of stddev_pop function
	AggFuncStddevPop = "stddev_pop"
	// AggFuncStddevSamp is the name of stddev_samp function
	AggFuncStddevSamp = "stddev_samp"
)

// AggregateFuncExpr represents aggregate function expression.
type AggregateFuncExpr struct {
	funcNode
	FnNameParts []string
	// F is the function name.
	F string
	// Args is the function args.
	Args []ExprNode
	// Distinct is true, function hence only aggregate distinct values.
	// For example, column c1 values are "1", "2", "2",  "sum(c1)" is "5",
	// but "sum(distinct c1)" is "3".
	Distinct bool
	// Filter is the optional FILTER (WHERE expr) clause.
	Filter ExprNode
	// WithinGroup is the optional WITHIN GROUP (ORDER BY ...) clause.
	WithinGroup *OrderByClause
	OrderBy     *OrderByClause
	Star        bool
}

// Restore implements Node interface.
func (n *AggregateFuncExpr) Restore(ctx *format.RestoreCtx) error {
	writeFuncName(ctx, n.FnNameParts, n.F)
	ctx.WritePlain("(")
	if n.Distinct {
		ctx.WriteKeyWord("DISTINCT ")
	}
	if n.Star {
		ctx.WritePlain("*")
	}
	for i, argv := range n.Args {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := argv.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore AggregateFuncExpr.Args[%d]", i)
		}
	}
	if n.OrderBy != nil {
		ctx.WritePlain(" ")
		if err := n.OrderBy.Restore(ctx); err != nil {
			return err
		}
	}
	ctx.WritePlain(")")
	if n.WithinGroup != nil {
		ctx.WriteKeyWord(" WITHIN GROUP (")
		if err := n.WithinGroup.Restore(ctx); err != nil {
			return err
		}
		ctx.WritePlain(")")
	}
	if n.Filter != nil {
		ctx.WriteKeyWord(" FILTER (WHERE ")
		if err := n.Filter.Restore(ctx); err != nil {
			return err
		}
		ctx.WritePlain(")")
	}
	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *AggregateFuncExpr) Format(w io.Writer) {
	_, _ = fmt.Fprintf(w, "%s(", n.F)
	if n.Distinct {
		_, _ = fmt.Fprint(w, "DISTINCT ")
	}
	if n.Star {
		_, _ = fmt.Fprint(w, "*")
	} else {
		for i, arg := range n.Args {
			if i != 0 {
				_, _ = fmt.Fprint(w, ", ")
			}
			arg.Format(w)
		}
	}
	if n.OrderBy != nil {
		_, _ = fmt.Fprint(w, " ")
		n.OrderBy.Format(w)
	}
	_, _ = fmt.Fprint(w, ")")
	if n.WithinGroup != nil {
		_, _ = fmt.Fprint(w, " WITHIN GROUP (")
		n.WithinGroup.Format(w)
		_, _ = fmt.Fprint(w, ")")
	}
	if n.Filter != nil {
		_, _ = fmt.Fprint(w, " FILTER (WHERE ")
		n.Filter.Format(w)
		_, _ = fmt.Fprint(w, ")")
	}
}

// Accept implements Node Accept interface.
func (n *AggregateFuncExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*AggregateFuncExpr)
	if n.OrderBy != nil {
		node, ok := n.OrderBy.Accept(v)
		if !ok {
			return n, false
		}
		n.OrderBy = node.(*OrderByClause)
	}
	for i, val := range n.Args {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Args[i] = node.(ExprNode)
	}
	if n.Filter != nil {
		node, ok := n.Filter.Accept(v)
		if !ok {
			return n, false
		}
		n.Filter = node.(ExprNode)
	}
	if n.WithinGroup != nil {
		node, ok := n.WithinGroup.Accept(v)
		if !ok {
			return n, false
		}
		n.WithinGroup = node.(*OrderByClause)
	}
	return v.Leave(n)
}

const (
	// WindowFuncRowNumber is the name of row_number function.
	WindowFuncRowNumber = "row_number"
	// WindowFuncRank is the name of rank function.
	WindowFuncRank = "rank"
	// WindowFuncDenseRank is the name of dense_rank function.
	WindowFuncDenseRank = "dense_rank"
	// WindowFuncCumeDist is the name of cume_dist function.
	WindowFuncCumeDist = "cume_dist"
	// WindowFuncPercentRank is the name of percent_rank function.
	WindowFuncPercentRank = "percent_rank"
	// WindowFuncNtile is the name of ntile function.
	WindowFuncNtile = "ntile"
	// WindowFuncLead is the name of lead function.
	WindowFuncLead = "lead"
	// WindowFuncLag is the name of lag function.
	WindowFuncLag = "lag"
	// WindowFuncFirstValue is the name of first_value function.
	WindowFuncFirstValue = "first_value"
	// WindowFuncLastValue is the name of last_value function.
	WindowFuncLastValue = "last_value"
	// WindowFuncNthValue is the name of nth_value function.
	WindowFuncNthValue = "nth_value"
)

// WindowFuncExpr represents window function expression.
type WindowFuncExpr struct {
	funcNode
	FnNameParts []string

	// F is the function name.
	F string
	// Args is the function args.
	Args []ExprNode
	// Distinct is retained for compatibility; PostgreSQL does not support DISTINCT in window calls.
	Distinct bool
	// IgnoreNull is retained for compatibility; PostgreSQL uses RESPECT NULLS behavior only.
	IgnoreNull bool
	// FromLast is retained for compatibility; PostgreSQL implements nth_value FROM FIRST behavior only.
	FromLast bool
	// Spec is the specification of this window.
	Spec WindowSpec
	// Star is true when the call is written as `count(*)`.
	Star        bool
	Filter      ExprNode
	OrderBy     *OrderByClause
	WithinGroup *OrderByClause
}

// Restore implements Node interface.
func (n *WindowFuncExpr) Restore(ctx *format.RestoreCtx) error {
	writeFuncName(ctx, n.FnNameParts, n.F)
	ctx.WritePlain("(")
	if n.Star {
		ctx.WritePlain("*")
	}
	for i, v := range n.Args {
		if i != 0 {
			ctx.WritePlain(", ")
		} else if n.Distinct {
			ctx.WriteKeyWord("DISTINCT ")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore WindowFuncExpr.Args[%d]", i)
		}
	}
	if n.OrderBy != nil {
		ctx.WritePlain(" ")
		if err := n.OrderBy.Restore(ctx); err != nil {
			return err
		}
	}
	ctx.WritePlain(")")
	if n.WithinGroup != nil {
		ctx.WriteKeyWord(" WITHIN GROUP (")
		if err := n.WithinGroup.Restore(ctx); err != nil {
			return err
		}
		ctx.WritePlain(")")
	}
	if n.Filter != nil {
		ctx.WriteKeyWord(" FILTER (WHERE ")
		if err := n.Filter.Restore(ctx); err != nil {
			return err
		}
		ctx.WritePlain(")")
	}
	if n.FromLast {
		ctx.WriteKeyWord(" FROM LAST")
	}
	if n.IgnoreNull {
		ctx.WriteKeyWord(" IGNORE NULLS")
	}
	ctx.WriteKeyWord(" OVER ")
	if err := n.Spec.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore WindowFuncExpr.Spec")
	}

	return ctx.Err()
}

// Format formats the window function expression into a Writer.
func (n *WindowFuncExpr) Format(w io.Writer) {
	_, _ = fmt.Fprintf(w, "%s(", n.F)
	if n.Distinct {
		_, _ = fmt.Fprint(w, "DISTINCT ")
	}
	for i, arg := range n.Args {
		if i != 0 {
			_, _ = fmt.Fprint(w, ", ")
		}
		arg.Format(w)
	}
	_, _ = fmt.Fprint(w, ")")
	if n.Filter != nil {
		_, _ = fmt.Fprint(w, " FILTER (WHERE ")
		n.Filter.Format(w)
		_, _ = fmt.Fprint(w, ")")
	}
	if n.WithinGroup != nil {
		_, _ = fmt.Fprint(w, " WITHIN GROUP (")
		n.WithinGroup.Format(w)
		_, _ = fmt.Fprint(w, ")")
	}
	if n.IgnoreNull {
		_, _ = fmt.Fprint(w, " IGNORE NULLS")
	}
	_, _ = fmt.Fprint(w, " OVER ")
	if n.Spec != (WindowSpec{}) {
		n.Spec.Format(w)
	}
}

// Accept implements Node Accept interface.
func (n *WindowFuncExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*WindowFuncExpr)
	if n.Filter != nil {
		node, ok := n.Filter.Accept(v)
		if !ok {
			return n, false
		}
		n.Filter = node.(ExprNode)
	}
	if n.OrderBy != nil {
		node, ok := n.OrderBy.Accept(v)
		if !ok {
			return n, false
		}
		n.OrderBy = node.(*OrderByClause)
	}
	if n.WithinGroup != nil {
		node, ok := n.WithinGroup.Accept(v)
		if !ok {
			return n, false
		}
		n.WithinGroup = node.(*OrderByClause)
	}
	for i, val := range n.Args {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Args[i] = node.(ExprNode)
	}
	node, ok := n.Spec.Accept(v)
	if !ok {
		return n, false
	}
	n.Spec = *node.(*WindowSpec)
	return v.Leave(n)
}
