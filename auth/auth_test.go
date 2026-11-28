package auth

import "testing"

func TestIdentityStringsEscapeSeparators(t *testing.T) {
	user := &UserIdentity{
		Username:     `user\name@domain`,
		Hostname:     `host\name@domain`,
		AuthUsername: `matched\user@domain`,
		AuthHostname: `matched\host@domain`,
	}
	if got, want := user.String(), `user\\name\@domain@host\\name\@domain`; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
	if got, want := user.AuthIdentityString(), `matched\\user\@domain@matched\\host\@domain`; got != want {
		t.Fatalf("AuthIdentityString() = %q, want %q", got, want)
	}
	role := &RoleIdentity{Username: `role@name`, Hostname: `host\name`}
	if got, want := role.String(), `role\@name@host\\name`; got != want {
		t.Fatalf("RoleIdentity.String() = %q, want %q", got, want)
	}
	if got := (*UserIdentity)(nil).String(); got != "" {
		t.Fatalf("nil UserIdentity String() = %q", got)
	}
	if got := (*RoleIdentity)(nil).String(); got != "" {
		t.Fatalf("nil RoleIdentity String() = %q", got)
	}
}
