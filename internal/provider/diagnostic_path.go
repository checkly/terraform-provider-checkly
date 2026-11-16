package provider

import (
	"context"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// displayPath renders p for a diagnostic message the way p.String() does,
// except that each set element step renders as [*]. The framework identifies
// a set element by its whole value and prints that value in the step, such
// as request.metadata[Value({"key":"authorization","value":"Bearer ..."})],
// so the path of an attribute inside a set element would print the element's
// other attributes, including sensitive ones. Attribute names, list indexes
// and map keys render unchanged. Provider code formats paths into strings
// through displayPath only: TestNoRawPathFormatting fails on a path passed to
// fmt.Sprintf, Errorf, Sprint or Sprintln, used as a + operand, or converted
// with String(), outside displayPath and its helpers.
func displayPath(p path.Path) string {
	var b strings.Builder
	for i, step := range p.Steps() {
		switch step := step.(type) {
		case path.PathStepAttributeName:
			if i != 0 {
				b.WriteString(".")
			}
			b.WriteString(string(step))
		case path.PathStepElementKeyValue:
			b.WriteString("[*]")
		default:
			b.WriteString(step.String())
		}
	}
	return b.String()
}

// displayPathText rewrites the set elements of p in text as displayPath
// renders them. It replaces the rendering of every prefix of p that ends in
// a set element, longest first, so a sibling path in the same element, such
// as the other attribute an AlsoRequires validator names, is rewritten too.
// Each prefix is replaced both as p.String() renders it and as %q quotes it,
// the two ways terraform-plugin-framework-validators format paths.
func displayPathText(text string, p path.Path) string {
	for prefix := p; len(prefix.Steps()) > 0; prefix = prefix.ParentPath() {
		steps := prefix.Steps()
		if _, ok := steps[len(steps)-1].(path.PathStepElementKeyValue); !ok {
			continue
		}
		raw, shown := prefix.String(), displayPath(prefix)
		text = strings.ReplaceAll(text, raw, shown)
		text = strings.ReplaceAll(text, quotedInner(raw), quotedInner(shown))
	}
	return text
}

// quotedInner returns s as %q formats it, without the surrounding quotes.
func quotedInner(s string) string {
	q := strconv.Quote(s)
	return q[1 : len(q)-1]
}

// displayPathDiagnostics returns diags with the summary and detail of each
// diagnostic that has a path rewritten by displayPathText for that path,
// keeping its severity and path. Diagnostics without a path, or with nothing
// to rewrite, are returned as they are.
func displayPathDiagnostics(diags diag.Diagnostics) diag.Diagnostics {
	out := make(diag.Diagnostics, 0, len(diags))
	for _, d := range diags {
		withPath, ok := d.(diag.DiagnosticWithPath)
		if !ok {
			out = append(out, d)
			continue
		}
		p := withPath.Path()
		summary, detail := displayPathText(d.Summary(), p), displayPathText(d.Detail(), p)
		switch {
		case summary == d.Summary() && detail == d.Detail():
			out = append(out, d)
		case d.Severity() == diag.SeverityWarning:
			out = append(out, diag.NewAttributeWarningDiagnostic(p, summary, detail))
		default:
			out = append(out, diag.NewAttributeErrorDiagnostic(p, summary, detail))
		}
	}
	return out
}

// The safe* functions wrap a validator from terraform-plugin-framework-
// validators, whose messages format paths with p.String(), so that its
// diagnostics render set elements as displayPath does. Description and
// MarkdownDescription are the wrapped validator's, unchanged. Wrapping an
// element validator, such as setvalidator.ValueStringsAre, also covers the
// validators it applies to each element, because each diagnostic is
// rewritten for its own path. The framework has no hook that rewrites
// validator diagnostics centrally, so each such validator in a schema is
// wrapped; TestSchemaStockValidatorsWrapped fails on one attached unwrapped.

// safeString wraps a stock validator.String; see displayPathDiagnostics.
func safeString(v validator.String) validator.String { return safeStringValidator{v} }

// safeInt64 wraps a stock validator.Int64; see displayPathDiagnostics.
func safeInt64(v validator.Int64) validator.Int64 { return safeInt64Validator{v} }

// safeBool wraps a stock validator.Bool; see displayPathDiagnostics.
func safeBool(v validator.Bool) validator.Bool { return safeBoolValidator{v} }

// safeList wraps a stock validator.List; see displayPathDiagnostics.
func safeList(v validator.List) validator.List { return safeListValidator{v} }

// safeSet wraps a stock validator.Set; see displayPathDiagnostics.
func safeSet(v validator.Set) validator.Set { return safeSetValidator{v} }

// safeResourceConfig wraps a stock resource.ConfigValidator; see
// displayPathDiagnostics.
func safeResourceConfig(v resource.ConfigValidator) resource.ConfigValidator {
	return safeResourceConfigValidator{v}
}

type safeStringValidator struct{ validator.String }

func (v safeStringValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	inner := &validator.StringResponse{}
	v.String.ValidateString(ctx, req, inner)
	resp.Diagnostics.Append(displayPathDiagnostics(inner.Diagnostics)...)
}

type safeInt64Validator struct{ validator.Int64 }

func (v safeInt64Validator) ValidateInt64(ctx context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	inner := &validator.Int64Response{}
	v.Int64.ValidateInt64(ctx, req, inner)
	resp.Diagnostics.Append(displayPathDiagnostics(inner.Diagnostics)...)
}

type safeBoolValidator struct{ validator.Bool }

func (v safeBoolValidator) ValidateBool(ctx context.Context, req validator.BoolRequest, resp *validator.BoolResponse) {
	inner := &validator.BoolResponse{}
	v.Bool.ValidateBool(ctx, req, inner)
	resp.Diagnostics.Append(displayPathDiagnostics(inner.Diagnostics)...)
}

type safeListValidator struct{ validator.List }

func (v safeListValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	inner := &validator.ListResponse{}
	v.List.ValidateList(ctx, req, inner)
	resp.Diagnostics.Append(displayPathDiagnostics(inner.Diagnostics)...)
}

type safeSetValidator struct{ validator.Set }

func (v safeSetValidator) ValidateSet(ctx context.Context, req validator.SetRequest, resp *validator.SetResponse) {
	inner := &validator.SetResponse{}
	v.Set.ValidateSet(ctx, req, inner)
	resp.Diagnostics.Append(displayPathDiagnostics(inner.Diagnostics)...)
}

type safeResourceConfigValidator struct{ resource.ConfigValidator }

func (v safeResourceConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	inner := &resource.ValidateConfigResponse{}
	v.ConfigValidator.ValidateResource(ctx, req, inner)
	resp.Diagnostics.Append(displayPathDiagnostics(inner.Diagnostics)...)
}
