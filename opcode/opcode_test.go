package opcode

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mia0x75/pgparser/format"
)

func TestT(t *testing.T) {
	op := Plus
	if op.String() != "plus" {
		t.Fatalf("invalid op code")
	}

	if len(Ops) != len(opsLiteral) {
		t.Error("inconsistent count ops and opsliteral")
	}
	var buf bytes.Buffer
	for op := range Ops {
		op.Format(&buf)
		if buf.String() != opsLiteral[op] {
			t.Error("format op fail", op)
		}
		buf.Reset()
	}

	const invalid Op = 0
	if got := invalid.String(); got != "Op(0)" {
		t.Fatalf("unknown opcode String() = %q", got)
	}
	buf.Reset()
	invalid.Format(&buf)
	if got := buf.String(); got != "Op(0)" {
		t.Fatalf("unknown opcode Format() = %q", got)
	}
	var restored strings.Builder
	err := invalid.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &restored))
	if err == nil {
		t.Fatal("unknown opcode Restore() succeeded")
	}
}
