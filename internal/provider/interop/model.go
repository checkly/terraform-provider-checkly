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

func RenderMany[
	T any,
	R any,
	RPtr interface {
		Render[T]
		*R
	},
	Sources ~[]R,
	Results ~[]T,
](
	ctx context.Context,
	sources Sources,
	results Results,
) (
	diags diag.Diagnostics,
) {
	for _, source := range sources {
		var result T

		diags.Append(RPtr(&source).Render(ctx, &result)...)
		if diags.HasError() {
			return diags
		}

		results = append(results, result)
	}

	return diags
}

type Refresh[T any] interface {
	Refresh(ctx context.Context, from *T, flags RefreshFlags) diag.Diagnostics
}

func RefreshMany[
	T any,
	R any,
	RPtr interface {
		Refresh[T]
		*R
	},
	Sources ~[]T,
	Results ~[]R,
](
	ctx context.Context,
	sources Sources,
	results Results,
	flags RefreshFlags,
) (
	diags diag.Diagnostics,
) {
	for _, source := range sources {
		var r R

		diags.Append(RPtr(&r).Refresh(ctx, &source, flags)...)
		if diags.HasError() {
			return diags
		}

		results = append(results, r)
	}

	return diags
}

type Model[T any] interface {
	Render[T]
	Refresh[T]
}
