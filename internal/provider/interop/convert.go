package interop

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func AttributeTypes(attributes map[string]schema.Attribute) map[string]attr.Type {
	result := make(map[string]attr.Type, len(attributes))

	for name, attribute := range attributes {
		result[name] = attribute.GetType()
	}

	return result
}

type Objecter[T any] struct {
	attrTypes map[string]attr.Type
}

func ObjecterForSingleNestedAttribute[T any](attribute schema.SingleNestedAttribute) Objecter[T] {
	return Objecter[T]{
		attrTypes: AttributeTypes(attribute.Attributes),
	}
}

func (o Objecter[T]) ObjectAsValue(ctx context.Context, from types.Object, into *T) diag.Diagnostics {
	return from.As(ctx, into, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	})
}

func (o Objecter[T]) ValueToObject(ctx context.Context, from *T) (types.Object, diag.Diagnostics) {
	return types.ObjectValueFrom(ctx, o.attrTypes, from)
}

func (o Objecter[T]) UnknownObject() types.Object {
	return types.ObjectUnknown(o.attrTypes)
}

func (o Objecter[T]) NullObject() types.Object {
	return types.ObjectNull(o.attrTypes)
}

type Lister[T any] struct {
	elemType attr.Type
}

func ListerForListNestedAttribute[T any](attribute schema.ListNestedAttribute) Lister[T] {
	return Lister[T]{
		elemType: attribute.NestedObject.Type(),
	}
}

func (l Lister[T]) ListAsValue(ctx context.Context, from types.List, into *[]T) diag.Diagnostics {
	if from.IsNull() {
		return nil
	}

	return from.ElementsAs(ctx, into, true)
}

func (l Lister[T]) ValueToList(ctx context.Context, from []T) (types.List, diag.Diagnostics) {
	return types.ListValueFrom(ctx, l.elemType, from)
}

func (l Lister[T]) UnknownList() types.List {
	return types.ListUnknown(l.elemType)
}

func (l Lister[T]) NullList() types.List {
	return types.ListNull(l.elemType)
}

type Setter[T any] struct {
	elemType attr.Type
}

func SetterForSetAttribute[T any](attribute schema.SetAttribute) Setter[T] {
	return Setter[T]{
		elemType: attribute.ElementType,
	}
}

func (l Setter[T]) SetAsValue(ctx context.Context, from types.Set, into *[]T) diag.Diagnostics {
	if from.IsNull() {
		return nil
	}

	return from.ElementsAs(ctx, into, true)
}

func (l Setter[T]) ValueToSet(ctx context.Context, from []T) (types.Set, diag.Diagnostics) {
	return types.SetValueFrom(ctx, l.elemType, from)
}

func (l Setter[T]) UnknownSet() types.Set {
	return types.SetUnknown(l.elemType)
}

func (l Setter[T]) NullSet() types.Set {
	return types.SetNull(l.elemType)
}

type Refresher[
	T any,
	R any,
	RP interface {
		Refresh[T]
		*R
	},
] struct {
	objecter Objecter[R]
	lister   Lister[R]
	setter   Setter[R]
}

func RefresherForSingleNestedAttribute[
	T any,
	R any,
	RP interface {
		Refresh[T]
		*R
	},
](
	attribute schema.SingleNestedAttribute,
) Refresher[T, R, RP] {
	return Refresher[T, R, RP]{
		objecter: ObjecterForSingleNestedAttribute[R](attribute),
	}
}

func RefresherForListNestedAttribute[
	T any,
	R any,
	RP interface {
		Refresh[T]
		*R
	},
](
	attribute schema.ListNestedAttribute,
) Refresher[T, R, RP] {
	return Refresher[T, R, RP]{
		lister: ListerForListNestedAttribute[R](attribute),
	}
}

func RefresherForSetAttribute[
	T any,
	R any,
	RP interface {
		Refresh[T]
		*R
	},
](
	attribute schema.SetAttribute,
) Refresher[T, R, RP] {
	return Refresher[T, R, RP]{
		setter: SetterForSetAttribute[R](attribute),
	}
}

func (r Refresher[T, R, RP]) RefreshToObject(
	ctx context.Context,
	source *T,
	flags RefreshFlags,
) (
	object types.Object,
	value R,
	diags diag.Diagnostics,
) {
	// TODO: Consider making this behavior configurable.
	if source == nil {
		object = r.objecter.NullObject()
		return object, value, diags
	}

	diags.Append(RP(&value).Refresh(ctx, source, flags)...)
	if diags.HasError() {
		return object, value, diags
	}

	object, diags = r.objecter.ValueToObject(ctx, &value)
	return object, value, diags
}

func (r Refresher[T, R, RP]) RefreshToList(
	ctx context.Context,
	sources *[]T,
	flags RefreshFlags,
) (
	list types.List,
	values []R,
	diags diag.Diagnostics,
) {
	// TODO: Consider making this behavior configurable.
	if sources == nil {
		list = r.lister.NullList()
		return list, values, diags
	}

	values = make([]R, 0, len(*sources))

	for _, source := range *sources {
		var value R

		diags.Append(RP(&value).Refresh(ctx, &source, flags)...)
		if diags.HasError() {
			return list, values, diags
		}

		values = append(values, value)
	}

	list, diags = r.lister.ValueToList(ctx, values)
	return list, values, diags
}

func (r Refresher[T, R, RP]) RefreshToSet(
	ctx context.Context,
	sources *[]T,
	flags RefreshFlags,
) (
	set types.Set,
	values []R,
	diags diag.Diagnostics,
) {
	// TODO: Consider making this behavior configurable.
	if sources == nil {
		set = r.setter.NullSet()
		return set, values, diags
	}

	values = make([]R, 0, len(*sources))

	for _, source := range *sources {
		var value R

		diags.Append(RP(&value).Refresh(ctx, &source, flags)...)
		if diags.HasError() {
			return set, values, diags
		}

		values = append(values, value)
	}

	set, diags = r.setter.ValueToSet(ctx, values)
	return set, values, diags
}

type Renderer[
	T any,
	R any,
	RP interface {
		Render[T]
		*R
	},
] struct {
	objecter Objecter[R]
	lister   Lister[R]
	setter   Setter[R]
}

func RendererForSingleNestedAttribute[
	T any,
	R any,
	RP interface {
		Render[T]
		*R
	},
](
	attribute schema.SingleNestedAttribute,
) Renderer[T, R, RP] {
	return Renderer[T, R, RP]{
		objecter: ObjecterForSingleNestedAttribute[R](attribute),
	}
}

func RendererForListNestedAttribute[
	T any,
	R any,
	RP interface {
		Render[T]
		*R
	},
](
	attribute schema.ListNestedAttribute,
) Renderer[T, R, RP] {
	return Renderer[T, R, RP]{
		lister: ListerForListNestedAttribute[R](attribute),
	}
}

func RendererForSetAttribute[
	T any,
	R any,
	RP interface {
		Render[T]
		*R
	},
](
	attribute schema.SetAttribute,
) Renderer[T, R, RP] {
	return Renderer[T, R, RP]{
		setter: SetterForSetAttribute[R](attribute),
	}
}

func (r Renderer[T, R, RP]) RenderFromObject(
	ctx context.Context,
	object types.Object,
) (
	result T,
	value R,
	diags diag.Diagnostics,
) {
	diags = r.objecter.ObjectAsValue(ctx, object, &value)
	if diags.HasError() {
		return result, value, diags
	}

	diags.Append(RP(&value).Render(ctx, &result)...)
	return result, value, diags
}

func (r Renderer[T, R, RP]) RenderFromList(
	ctx context.Context,
	list types.List,
) (
	results []T,
	values []R,
	diags diag.Diagnostics,
) {
	diags = r.lister.ListAsValue(ctx, list, &values)
	if diags.HasError() {
		return results, values, diags
	}

	results = make([]T, 0, len(values))

	for _, value := range values {
		var result T

		diags.Append(RP(&value).Render(ctx, &result)...)
		if diags.HasError() {
			return results, values, diags
		}

		results = append(results, result)
	}

	return results, values, diags
}

func (r Renderer[T, R, RP]) RenderFromSet(
	ctx context.Context,
	set types.Set,
) (
	results []T,
	values []R,
	diags diag.Diagnostics,
) {
	diags = r.setter.SetAsValue(ctx, set, &values)
	if diags.HasError() {
		return results, values, diags
	}

	results = make([]T, 0, len(values))

	for _, value := range values {
		var result T

		diags.Append(RP(&value).Render(ctx, &result)...)
		if diags.HasError() {
			return results, values, diags
		}

		results = append(results, result)
	}

	return results, values, diags
}

type Gluer[
	T any,
	R any,
	RP interface {
		Refresh[T]
		Render[T]
		*R
	},
] struct {
	Refresher[T, R, RP]
	Renderer[T, R, RP]
}

func GluerForSingleNestedAttribute[
	T any,
	R any,
	RP interface {
		Refresh[T]
		Render[T]
		*R
	},
](
	attribute schema.SingleNestedAttribute,
) Gluer[T, R, RP] {
	return Gluer[T, R, RP]{
		Refresher: RefresherForSingleNestedAttribute[T, R, RP](attribute),
		Renderer:  RendererForSingleNestedAttribute[T, R, RP](attribute),
	}
}

func GluerForListNestedAttribute[
	T any,
	R any,
	RP interface {
		Refresh[T]
		Render[T]
		*R
	},
](
	attribute schema.ListNestedAttribute,
) Gluer[T, R, RP] {
	return Gluer[T, R, RP]{
		Refresher: RefresherForListNestedAttribute[T, R, RP](attribute),
		Renderer:  RendererForListNestedAttribute[T, R, RP](attribute),
	}
}

func GluerForSetAttribute[
	T any,
	R any,
	RP interface {
		Refresh[T]
		Render[T]
		*R
	},
](
	attribute schema.SetAttribute,
) Gluer[T, R, RP] {
	return Gluer[T, R, RP]{
		Refresher: RefresherForSetAttribute[T, R, RP](attribute),
		Renderer:  RendererForSetAttribute[T, R, RP](attribute),
	}
}
