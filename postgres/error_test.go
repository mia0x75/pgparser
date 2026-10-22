package postgres

import (
	"testing"
)

func TestSQLError(t *testing.T) {
	e := NewErrf(ErrNoDB, "no db error")
	if !(len(e.Error()) > 0) {
		t.Fatalf("got %d, want > 0", len(e.Error()))
	}

	e = NewErrf(0, "customized error")
	if !(len(e.Error()) > 0) {
		t.Fatalf("got %d, want > 0", len(e.Error()))
	}

	e = NewErr(ErrNoDB)
	if !(len(e.Error()) > 0) {
		t.Fatalf("got %d, want > 0", len(e.Error()))
	}

	e = NewErr(0, "customized error")
	if !(len(e.Error()) > 0) {
		t.Fatalf("got %d, want > 0", len(e.Error()))
	}
}
