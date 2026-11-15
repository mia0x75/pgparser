package hack

import "unsafe"

// MutableString can be used as string via string(MutableString) without performance loss.
type MutableString string

// String converts slice to MutableString without copy.
// The MutableString can be converts to string without copy.
// Use it at your own risk.
func String(b []byte) (s MutableString) {
	if len(b) == 0 {
		return ""
	}
	return MutableString(unsafe.String(unsafe.SliceData(b), len(b)))
}

// Slice converts string to slice without copy.
// Use it at your own risk.
func Slice(s string) (b []byte) {
	if len(s) == 0 {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
