package interop

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

type RefreshFlags int

const (
	Created RefreshFlags = 1 << iota
	Updated

	Loaded RefreshFlags = 0
)

func (f RefreshFlags) Contains(other RefreshFlags) bool {
	return f&other == other
}

func (f RefreshFlags) Updated() bool {
	return f.Contains(Updated)
}

func (f RefreshFlags) Created() bool {
	return f.Contains(Created)
}

type Render[T any] interface {
	Render(ctx context.Context, into *T) diag.Diagnostics
}

type Refresh[T any] interface {
	Refresh(ctx context.Context, from *T, flags RefreshFlags) diag.Diagnostics
}

type Model[T any] interface {
	Render[T]
	Refresh[T]
}
