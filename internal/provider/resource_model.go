package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

type RefreshFlags int

const (
	ModelCreated RefreshFlags = 1 << iota
	ModelUpdated

	ModelLoaded RefreshFlags = 0
)

func (f RefreshFlags) Contains(other RefreshFlags) bool {
	return f&other == other
}

func (f RefreshFlags) Updated() bool {
	return f.Contains(ModelUpdated)
}

func (f RefreshFlags) Created() bool {
	return f.Contains(ModelCreated)
}

type Render[SDKModel any] interface {
	Render(ctx context.Context, into *SDKModel) diag.Diagnostics
}

func RenderMany[
	SDKModel any,
	R any,
	RPtr interface {
		Render[SDKModel]
		*R
	},
	Sources ~[]R,
	Results ~[]SDKModel,
](
	ctx context.Context,
	sources Sources,
	results Results,
) (
	diags diag.Diagnostics,
) {
	for _, source := range sources {
		var result SDKModel

		diags.Append(RPtr(&source).Render(ctx, &result)...)
		if diags.HasError() {
			return diags
		}

		results = append(results, result)
	}

	return diags
}

type Refresh[SDKModel any] interface {
	Refresh(ctx context.Context, from *SDKModel, flags RefreshFlags) diag.Diagnostics
}

func RefreshMany[
	SDKModel any,
	R any,
	RPtr interface {
		Refresh[SDKModel]
		*R
	},
	Sources ~[]SDKModel,
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

type ResourceModel[SDKModel any] interface {
	Render[SDKModel]
	Refresh[SDKModel]
}
