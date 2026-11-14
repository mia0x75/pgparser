// Package charset exposes the binary charset markers used by the type layer.
package charset

import "github.com/mia0x75/pgparser/postgres"

const (
	// CharsetBin marks a binary charset.
	CharsetBin = "binary"
	// CollationBin is the binary collation.
	CollationBin = "binary"
)

// GetDefaultCharsetAndCollate returns the default charset and collation.
func GetDefaultCharsetAndCollate() (string, string) {
	return postgres.DefaultCharset, postgres.DefaultCollationName
}
