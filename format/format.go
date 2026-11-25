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
// Format() formats string as indent and unindent commands. The commands can
// nest. The Formatter writes to io.Writer 'w' and inserts one 'indent'
// string per current indent level value.
// Behaviour of commands reaching negative indent levels is undefined.
//
//	IndentFormatter(os.Stdout, "\t").Format("abc%d%%e%i\nx\ny\n%uz\n", 3)
//
// output:
//
//	abc3%e
//	    x
//	    y
//	z
//
// The Go quoted string literal form of the above is:
//
//	"abc%%e\n\tx\n\tx\nz\n"
//
// The commands can be scattered between separate invocations of Format(),
// i.e. the formatter keeps track of the indent level and knows if it is
// positioned on start of a line and should emit indentation(s).
// The same output as above can be produced by e.g.:
//
//	f := IndentFormatter(os.Stdout, " ")
//	f.Format("abc%d%%e%i\nx\n", 3)
//	f.Format("y\n%uz\n")
func IndentFormatter(w io.Writer, indent string) Formatter {
	return pkformat.IndentFormatter(w, indent)
}

// FlatFormatter returns a newly created Formatter with the same functionality as the one returned
// by IndentFormatter except it allows a newline in the 'format' string argument of Format
// to pass through if the indent level is current zero.
//
// If the indent level is non-zero then such new lines are changed to a space character.
// There is no indent string, the %i and %u format verbs are used solely to determine the indent level.
//
// The FlatFormatter is intended for flattening of normally nested structure textual representation to
// a one top level structure per line form.
//
//	FlatFormatter(os.Stdout, " ").Format("abc%d%%e%i\nx\ny\n%uz\n", 3)
//
// output in the form of a Go quoted string literal:
//
//	"abc3%%e x y z\n"
func FlatFormatter(w io.Writer) Formatter {
	return pkformat.FlatFormatter(w)
}

// OutputFormat output escape character with backslash.
func OutputFormat(s string) string {
	return pkformat.OutputFormat(s)
}

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
