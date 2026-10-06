package mcp

import (
	"context"
	"errors"
	"fmt"
)

var ErrDenied = errors.New("permission denied")

type Action struct {
	Tool        string
	Description string
	ReadOnly    bool
}

type Permission interface {
	Allow(ctx context.Context, a Action) error // nil = setuju
}

type PermissionFunc func(ctx context.Context, a Action) error

func (f PermissionFunc) Allow(ctx context.Context, a Action) error { return f(ctx, a) }

func AllowAll() Permission {
	return PermissionFunc(func(context.Context, Action) error { return nil })
}

func DenyAll() Permission {
	return PermissionFunc(func(_ context.Context, a Action) error {
		return fmt.Errorf("%w: tool %q tidak diizinkan", ErrDenied, a.Tool)
	})
}

// AllowReadOnly menyetujui tool ReadOnly otomatis, sisanya diteruskan ke next.
func AllowReadOnly(next Permission) Permission {
	return PermissionFunc(func(ctx context.Context, a Action) error {
		if a.ReadOnly {
			return nil
		}
		if next == nil {
			return fmt.Errorf("%w: tool non-readonly butuh permission", ErrDenied)
		}
		return next.Allow(ctx, a)
	})
}
