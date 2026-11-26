// Package format provides the PostgreSQL dialect facade over the shared
// github.com/mia0x75/parserkit/format restore primitives.
package format

import (
	"io"

	pkformat "github.com/mia0x75/parserkit/format"
)

// Formatter is an io.Writer extended formatter by a fmt.Printf like function Format.
type Formatter = pkformat.Formatter

// IndentFormatter returns a new Formatter which interprets %i and %u in the
// Format() formats string as indent and unindent commands. See pkformat.IndentFormatter
// for the full semantics.
var IndentFormatter = pkformat.IndentFormatter

// FlatFormatter returns a newly created Formatter which flattens nested
// structures to one line each. See pkformat.FlatFormatter.
var FlatFormatter = pkformat.FlatFormatter

// OutputFormat output escape character with backslash.
var OutputFormat = pkformat.OutputFormat

// RestoreFlag mark the Restore format
type RestoreFlags = pkformat.RestoreFlags

// Mutually exclusive group of `RestoreFlags`:
// [RestoreStringSingleQuotes, RestoreStringDoubleQuotes]
// [RestoreKeyWordUppercase, RestoreKeyWordLowercase]
// [RestoreNameUppercase, RestoreNameLowercase]
// [RestoreNameDoubleQuotes, RestoreNameBackQuotes]
// The flag with the left position in each group has a higher priority.
const (
	RestoreStringSingleQuotes          = pkformat.RestoreStringSingleQuotes
	RestoreStringDoubleQuotes          = pkformat.RestoreStringDoubleQuotes
	RestoreStringEscapeBackslash       = pkformat.RestoreStringEscapeBackslash
	RestoreKeyWordUppercase            = pkformat.RestoreKeywordUppercase
	RestoreKeyWordLowercase            = pkformat.RestoreKeywordLowercase
	RestoreNameUppercase               = pkformat.RestoreNameUppercase
	RestoreNameLowercase               = pkformat.RestoreNameLowercase
	RestoreNameDoubleQuotes            = pkformat.RestoreNameDoubleQuotes
	RestoreNameBackQuotes              = pkformat.RestoreNameBackQuotes
	RestoreSpacesAroundBinaryOperation = pkformat.RestoreSpacesAroundBinaryOperation
)

// DefaultRestoreFlags is the PostgreSQL default: single quoted strings, upper
// case keywords and double quoted identifiers.
const DefaultRestoreFlags = RestoreStringSingleQuotes | RestoreKeyWordUppercase | RestoreNameDoubleQuotes

// RestoreCtx is `Restore` context to hold flags and writer.
type RestoreCtx = pkformat.RestoreCtx

// NewRestoreCtx returns a new `RestoreCtx`.
//
// The shared context is built with a zero DialectConfig so that quotes are only
// emitted when the caller asks for them through the restore flags.
func NewRestoreCtx(flags RestoreFlags, in io.Writer) *RestoreCtx {
	return pkformat.NewRestoreCtx(in, pkformat.DialectConfig{}, flags)
}
