package json

import (
	stderrors "errors"
	"testing"

	pgerrors "github.com/mia0x75/pgparser/errors"
	"github.com/mia0x75/pgparser/postgres"
)

func TestBinaryModifyAndRemoveErrorCodes(t *testing.T) {
	object := mustParseBinaryFromString(t, `{"a": [1]}`)
	value := mustParseBinaryFromString(t, `1`)
	wildcard, err := ParseJSONPathExpr("$.a[*]")
	if err != nil {
		t.Fatal(err)
	}
	empty, err := ParseJSONPathExpr("$")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		code postgres.ErrCode
		err  error
	}{
		{name: "modify parameter count", code: postgres.ErrWrongParamcountToNativeFct, err: func() error {
			_, err := object.Modify(nil, []BinaryJSON{value}, ModifySet)
			return err
		}()},
		{name: "modify wildcard", code: postgres.ErrInvalidJSONPathWildcard, err: func() error {
			_, err := object.Modify([]PathExpression{wildcard}, []BinaryJSON{value}, ModifySet)
			return err
		}()},
		{name: "remove wildcard", code: postgres.ErrInvalidJSONPathWildcard, err: func() error {
			_, err := object.Remove([]PathExpression{wildcard})
			return err
		}()},
		{name: "remove empty path", code: postgres.ErrInvalidJSONPathEmpty, err: func() error {
			_, err := object.Remove([]PathExpression{empty})
			return err
		}()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err == nil {
				t.Fatal("operation succeeded, want error")
			}
			root := pgerrors.Cause(tc.err)
			var coded *postgres.Error
			if !stderrors.As(root, &coded) {
				t.Fatalf("error cause = %T, want *postgres.Error", root)
			}
			if coded.Code() != tc.code {
				t.Fatalf("error code = %d, want %d", coded.Code(), tc.code)
			}
			if state := coded.ToSQLError().State; state != "42000" {
				t.Fatalf("SQLSTATE = %q, want 42000", state)
			}
		})
	}
}
