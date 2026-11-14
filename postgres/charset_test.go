package postgres

import "testing"

func TestCompatibilityCollationMappings(t *testing.T) {
	for id, name := range Collations {
		if id == 159 || id == 223 {
			t.Fatalf("removed compatibility collation ID %d is still supported", id)
		}
		if got, ok := CollationNames[name]; !ok || got != id {
			t.Fatalf("collation %q: got ID %d, present %t; want %d", name, got, ok, id)
		}
	}
	for name, id := range CollationNames {
		if got, ok := Collations[id]; !ok || got != name {
			t.Fatalf("collation ID %d: got name %q, present %t; want %q", id, got, ok, name)
		}
	}
	if got := Collations[DefaultCollationID]; got != DefaultCollationName {
		t.Fatalf("got default collation %q, want %q", got, DefaultCollationName)
	}
	for name, id := range CharsetIDs {
		if got := Collations[id]; got != Charsets[name] {
			t.Fatalf("charset %q: got default collation %q, want %q", name, got, Charsets[name])
		}
	}
}
