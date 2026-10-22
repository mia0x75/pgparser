package types

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/mia0x75/pgparser/errors"
)

func TestStrToInt(t *testing.T) {
	tests := []struct {
		input  string
		output string
		err    error
	}{
		{"9223372036854775806", "9223372036854775806", nil},
		{"9223372036854775807", "9223372036854775807", nil},
		{"9223372036854775808", "9223372036854775807", ErrBadNumber},
		{"-9223372036854775807", "-9223372036854775807", nil},
		{"-9223372036854775808", "-9223372036854775808", nil},
		{"-9223372036854775809", "-9223372036854775808", ErrBadNumber},
	}
	for _, tt := range tests {
		output, err := strToInt(tt.input)
		if !reflect.DeepEqual(errors.Cause(err), tt.err) {
			t.Fatalf("got %#v, want %#v", errors.Cause(err), tt.err)
		}
		if strconv.FormatInt(output, 10) != tt.output {
			t.Errorf("got %#v, want %#v", strconv.FormatInt(output, 10), tt.output)
		}
	}
}
