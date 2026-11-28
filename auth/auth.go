package auth

import (
	"fmt"
	"strings"

	"github.com/mia0x75/pgparser/format"
)

// UserIdentity represents username and hostname.
type UserIdentity struct {
	Username     string
	Hostname     string
	CurrentUser  bool
	AuthUsername string // Username matched in privileges system
	AuthHostname string // Match in privs system (i.e. could be a wildcard)
}

// Restore implements Node interface.
func (user *UserIdentity) Restore(ctx *format.RestoreCtx) error {
	if user.CurrentUser {
		ctx.WriteKeyword("CURRENT_USER")
	} else {
		ctx.WriteName(user.Username)
		if user.Hostname != "" {
			ctx.WritePlain("@")
			ctx.WriteName(user.Hostname)
		}
	}
	return ctx.Err()
}

// String converts UserIdentity to the format user@host.
func (user *UserIdentity) String() string {
	if user == nil {
		return ""
	}
	return fmt.Sprintf("%s@%s", escapeIdentityPart(user.Username), escapeIdentityPart(user.Hostname))
}

// AuthIdentityString returns matched identity in user@host format
func (user *UserIdentity) AuthIdentityString() string {
	if user == nil {
		return ""
	}
	return fmt.Sprintf("%s@%s", escapeIdentityPart(user.AuthUsername), escapeIdentityPart(user.AuthHostname))
}

func escapeIdentityPart(value string) string {
	return strings.NewReplacer("\\", "\\\\", "@", "\\@").Replace(value)
}

type RoleIdentity struct {
	Username string
	Hostname string
}

func (role *RoleIdentity) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteName(role.Username)
	if role.Hostname != "" {
		ctx.WritePlain("@")
		ctx.WriteName(role.Hostname)
	}
	return ctx.Err()
}

// String converts UserIdentity to the format user@host.
func (role *RoleIdentity) String() string {
	if role == nil {
		return ""
	}
	return fmt.Sprintf("%s@%s", escapeIdentityPart(role.Username), escapeIdentityPart(role.Hostname))
}
