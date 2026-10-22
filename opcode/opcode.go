package opcode

import (
	"fmt"
	"io"

	"github.com/mia0x75/pgparser/errors"

	. "github.com/mia0x75/pgparser/format"
)

// Op is opcode type.
type Op int

// List operators.
const (
	LogicAnd Op = iota + 1
	LeftShift
	RightShift
	LogicOr
	GE
	LE
	EQ
	NE
	LT
	GT
	Plus
	Minus
	And
	Or
	Mod
	Xor
	Div
	Mul
	Not
	BitNeg
	IntDiv
	LogicXor
	NullEQ
	In
	Like
	Case
	Regexp
	IsNull
	IsTruth
	IsFalsity
	JSONExtract
	JSONExtractText
	JSONPathExtract
	JSONPathExtractText
	JSONContains
	JSONContained
	JSONExists
	JSONAllExists
	JSONAnyExists
)

// Ops maps opcode to string.
var Ops = map[Op]string{
	LogicAnd:            "and",
	LogicOr:             "or",
	LogicXor:            "xor",
	LeftShift:           "leftshift",
	RightShift:          "rightshift",
	GE:                  "ge",
	LE:                  "le",
	EQ:                  "eq",
	NE:                  "ne",
	LT:                  "lt",
	GT:                  "gt",
	Plus:                "plus",
	Minus:               "minus",
	And:                 "bitand",
	Or:                  "bitor",
	Mod:                 "mod",
	Xor:                 "bitxor",
	Div:                 "div",
	Mul:                 "mul",
	Not:                 "not",
	BitNeg:              "bitneg",
	IntDiv:              "intdiv",
	NullEQ:              "nulleq",
	In:                  "in",
	Like:                "like",
	Case:                "case",
	Regexp:              "regexp",
	IsNull:              "isnull",
	IsTruth:             "istrue",
	IsFalsity:           "isfalse",
	JSONExtract:         "json_extract",
	JSONExtractText:     "json_extract_text",
	JSONPathExtract:     "json_path_extract",
	JSONPathExtractText: "json_path_extract_text",
	JSONContains:        "json_contains",
	JSONContained:       "json_contained",
	JSONExists:          "json_exists",
	JSONAllExists:       "json_all_exists",
	JSONAnyExists:       "json_any_exists",
}

// String implements Stringer interface.
func (o Op) String() string {
	str, ok := Ops[o]
	if !ok {
		panic(fmt.Sprintf("%d", o))
	}

	return str
}

var opsLiteral = map[Op]string{
	LogicAnd:            " AND ",
	LogicOr:             " OR ",
	LogicXor:            " XOR ",
	LeftShift:           "<<",
	RightShift:          ">>",
	GE:                  ">=",
	LE:                  "<=",
	EQ:                  "=",
	NE:                  "!=",
	LT:                  "<",
	GT:                  ">",
	Plus:                "+",
	Minus:               "-",
	And:                 "&",
	Or:                  "|",
	Mod:                 "%",
	Xor:                 "^",
	Div:                 "/",
	Mul:                 "*",
	Not:                 "!",
	BitNeg:              "~",
	IntDiv:              "DIV",
	NullEQ:              "<=>",
	In:                  "IN",
	Like:                "LIKE",
	Case:                "CASE",
	Regexp:              "REGEXP",
	IsNull:              "IS NULL",
	IsTruth:             "IS TRUE",
	IsFalsity:           "IS FALSE",
	JSONExtract:         "->",
	JSONExtractText:     "->>",
	JSONPathExtract:     "#>",
	JSONPathExtractText: "#>>",
	JSONContains:        "@>",
	JSONContained:       "<@",
	JSONExists:          "?",
	JSONAllExists:       "?&",
	JSONAnyExists:       "?|",
}

// Format the ExprNode into a Writer.
func (o Op) Format(w io.Writer) {
	fmt.Fprintf(w, "%s", opsLiteral[o])
}

// Restore the Op into a Writer
func (o Op) Restore(ctx *RestoreCtx) error {
	if v, ok := opsLiteral[o]; ok {
		ctx.WriteKeyWord(v)
		return nil
	}
	return errors.Errorf("Invalid opcode type %d during restoring AST to SQL text", o)
}
