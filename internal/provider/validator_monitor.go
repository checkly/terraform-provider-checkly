package provider

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// hostWithoutSchemeOrPortValidator rejects a host that includes a scheme,
// such as "https://", or a port, such as ":443". The API stores the host as
// given, so a probe would try to resolve the scheme or port as part of the
// host name. Most monitors that take a host have a separate port attribute;
// ICMP monitors have none. IPv6 addresses are accepted with or without brackets: a port is only
// detected after a closing bracket, or as the single colon of a host. The
// API does not validate this, so the rule exists only in the provider.
type hostWithoutSchemeOrPortValidator struct {
	// noPortAttribute is set for a monitor without a port attribute, whose
	// error then doesn't point to one.
	noPortAttribute bool
}

var _ validator.String = hostWithoutSchemeOrPortValidator{}

func (v hostWithoutSchemeOrPortValidator) Description(ctx context.Context) string {
	return "value must be a host name or IP address without a scheme or a port"
}

func (v hostWithoutSchemeOrPortValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v hostWithoutSchemeOrPortValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	if problem := hostSchemeOrPort(req.ConfigValue.ValueString()); problem != "" {
		detail := fmt.Sprintf("%s must be a host name or IP address without a scheme or a port, but it has %s: %q.", req.Path, problem, req.ConfigValue.ValueString())
		if !v.noPortAttribute {
			detail += " Set the port with the port attribute."
		}
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Host", detail)
	}
}

// hostSchemeOrPort describes the scheme or port that host includes, or
// returns "" when it includes neither.
func hostSchemeOrPort(host string) string {
	if strings.Contains(host, "://") {
		return "a scheme"
	}
	if strings.HasPrefix(host, "[") {
		if end := strings.Index(host, "]"); end >= 0 && strings.HasPrefix(host[end+1:], ":") {
			return "a port"
		}
		return ""
	}
	// A bare IPv6 address has at least two colons.
	if strings.Count(host, ":") == 1 {
		return "a port"
	}
	return ""
}

// numericAssertionTarget is the API's rule for the target of an assertion
// source that the runner compares as a number.
type numericAssertionTarget struct {
	pattern *regexp.Regexp
	// bounded is set when the number must lie in [min, max].
	bounded  bool
	min, max float64
	// label completes "must be ...".
	label string
}

var (
	nonNegativeNumberRegex  = regexp.MustCompile(`^\d+(\.\d+)?$`)
	nonNegativeIntegerRegex = regexp.MustCompile(`^\d+$`)
)

// assertionSourceRule is the API's rule for the assertions of one source.
type assertionSourceRule struct {
	// comparisons are the comparisons the source allows.
	comparisons []string
	// properties, when set, makes property required and one of these.
	properties []string
	// numericTarget, when set, makes target required and numeric.
	numericTarget *numericAssertionTarget
	// numericTargetComparisons, when set, limits numericTarget to these
	// comparisons, for a source that also allows comparisons that take no
	// value, such as IS_EMPTY.
	numericTargetComparisons []string
}

// assertionComparisonValues are every assertion comparison, for the sources
// that allow any of them.
var assertionComparisonValues = allowedValues[string]{
	{Value: "EQUALS"},
	{Value: "NOT_EQUALS"},
	{Value: "HAS_KEY"},
	{Value: "NOT_HAS_KEY"},
	{Value: "HAS_VALUE"},
	{Value: "NOT_HAS_VALUE"},
	{Value: "IS_EMPTY"},
	{Value: "NOT_EMPTY"},
	{Value: "GREATER_THAN"},
	{Value: "LESS_THAN"},
	{Value: "CONTAINS"},
	{Value: "NOT_CONTAINS"},
	{Value: "IS_NULL"},
	{Value: "NOT_NULL"},
}

// nonNegativeMillisecondsTarget is the target rule of a response time or
// latency assertion.
var nonNegativeMillisecondsTarget = &numericAssertionTarget{pattern: nonNegativeNumberRegex, label: "a non-negative number of milliseconds"}

// assertionRulesValidator validates an assertion set element against the
// rules of its source, as the API does: the comparisons each source allows,
// the property a source requires, and the format of a numeric target.
// Elements whose source is not known yet, or has no rule, are skipped; the
// source attribute has its own validator for the allowed sources.
type assertionRulesValidator struct {
	rules map[string]assertionSourceRule
}

var _ validator.Object = assertionRulesValidator{}

func (v assertionRulesValidator) Description(ctx context.Context) string {
	return "the comparison, property and target must be valid for the assertion source"
}

func (v assertionRulesValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v assertionRulesValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	attrs := req.ConfigValue.Attributes()
	source, _ := attrs["source"].(types.String)
	if source.IsNull() || source.IsUnknown() {
		return
	}
	rule, ok := v.rules[source.ValueString()]
	if !ok {
		return
	}

	if comparison, _ := attrs["comparison"].(types.String); rule.comparisons != nil && !comparison.IsNull() && !comparison.IsUnknown() && !slices.Contains(rule.comparisons, comparison.ValueString()) {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("comparison"),
			"Invalid Assertion Comparison",
			fmt.Sprintf("The comparison of a %s assertion must be one of %s, got: %q.", source.ValueString(), quotedList(rule.comparisons), comparison.ValueString()),
		)
	}

	// A null property or target counts as empty, which is what the
	// provider sends for it.
	if property, _ := attrs["property"].(types.String); rule.properties != nil && !property.IsUnknown() && !slices.Contains(rule.properties, property.ValueString()) {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("property"),
			"Invalid Assertion Property",
			fmt.Sprintf("A %s assertion requires a property, one of %s, got: %q.", source.ValueString(), quotedList(rule.properties), property.ValueString()),
		)
	}

	// A target limited to some comparisons is not checked while the
	// comparison is not known yet; it is checked once it is.
	comparison, _ := attrs["comparison"].(types.String)
	numericComparison := rule.numericTargetComparisons == nil || (!comparison.IsUnknown() && slices.Contains(rule.numericTargetComparisons, comparison.ValueString()))
	if target, _ := attrs["target"].(types.String); rule.numericTarget != nil && numericComparison && !target.IsUnknown() {
		if detail := rule.numericTarget.problem(source.ValueString(), target.ValueString()); detail != "" {
			resp.Diagnostics.AddAttributeError(req.Path.AtName("target"), "Invalid Assertion Target", detail)
		}
	}
}

// problem describes why target breaks the rule for an assertion of source,
// or returns "" when it doesn't.
func (t numericAssertionTarget) problem(source, target string) string {
	switch {
	case target == "":
		return fmt.Sprintf("A %s assertion requires a target, which must be %s.", source, t.label)
	case strings.TrimFunc(target, isAPIWhitespace) != target:
		return fmt.Sprintf("The target of a %s assertion must not have leading or trailing whitespace, which the API removes, got: %q.", source, target)
	case !t.pattern.MatchString(target):
		return fmt.Sprintf("The target of a %s assertion must be %s, got: %q.", source, t.label, target)
	}
	if t.bounded {
		if n, err := strconv.ParseFloat(target, 64); err != nil || n < t.min || n > t.max {
			return fmt.Sprintf("The target of a %s assertion must be %s, got: %q.", source, t.label, target)
		}
	}
	return ""
}

// quotedList formats values as a list of quoted strings for an error
// message, such as `"EQUALS", "NOT_EQUALS" or "LESS_THAN"`.
func quotedList(values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = strconv.Quote(v)
	}
	if len(quoted) < 2 {
		return strings.Join(quoted, "")
	}
	return strings.Join(quoted[:len(quoted)-1], ", ") + " or " + quoted[len(quoted)-1]
}

// orderedThresholdsConfigValidator rejects a configuration in which the
// value at upper is less than the value at lower, such as a failing response
// time below the degraded one, which the API rejects. An attribute that is
// not set counts as its default. An unset attribute without a static default
// (a null default) is derived by the API from the other one, so the pair is
// not checked then.
type orderedThresholdsConfigValidator struct {
	lower, upper path.Path
	// lowerDefault and upperDefault are the schema defaults of the
	// attributes, or null for an attribute that has none.
	lowerDefault, upperDefault types.Int64
	// zeroUpperUnbounded is set when an upper value of 0 means no limit,
	// which every lower value is within.
	zeroUpperUnbounded bool
}

var _ resource.ConfigValidator = orderedThresholdsConfigValidator{}

func (v orderedThresholdsConfigValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("%s must be greater than or equal to %s", v.upper, v.lower)
}

func (v orderedThresholdsConfigValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v orderedThresholdsConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var lower, upper types.Int64
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, v.lower, &lower)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, v.upper, &upper)...)
	if resp.Diagnostics.HasError() {
		return
	}

	lowerValue, lowerDescription, ok := thresholdValue(lower, v.lowerDefault)
	if !ok {
		return
	}
	upperValue, upperDescription, ok := thresholdValue(upper, v.upperDefault)
	if !ok {
		return
	}

	if v.zeroUpperUnbounded && upperValue == 0 {
		return
	}

	if upperValue < lowerValue {
		resp.Diagnostics.AddAttributeError(
			v.upper,
			"Invalid Attribute Combination",
			fmt.Sprintf("%s (%s) must be greater than or equal to %s (%s).", v.upper, upperDescription, v.lower, lowerDescription),
		)
	}
}

// thresholdValue returns the configured value of a threshold, or its default
// when it is not set, with a description of it for an error message. ok is
// false when the value is not known yet, or is not set and has no default.
func thresholdValue(v, def types.Int64) (value int64, description string, ok bool) {
	switch {
	case v.IsUnknown():
		return 0, "", false
	case v.IsNull() && def.IsNull():
		return 0, "", false
	case v.IsNull():
		return def.ValueInt64(), fmt.Sprintf("%d, the default", def.ValueInt64()), true
	}
	return v.ValueInt64(), strconv.FormatInt(v.ValueInt64(), 10), true
}

// uuidRegex matches a UUID in the lowercase, hyphenated form the API stores.
// The API stores a ping token as a Postgres uuid, which rejects other forms or
// rewrites them to this one, so the stored token would differ from the
// configured one.
var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// uuidValidator rejects a string that is not a lowercase, hyphenated UUID;
// see uuidRegex. The error doesn't
// repeat the value, which may be secret, such as a heartbeat ping token.
type uuidValidator struct{}

var _ validator.String = uuidValidator{}

func (v uuidValidator) Description(ctx context.Context) string {
	return "value must be a lowercase UUID"
}

func (v uuidValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v uuidValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	if !uuidRegex.MatchString(req.ConfigValue.ValueString()) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Attribute Value",
			fmt.Sprintf("%s must be a lowercase UUID with hyphens, such as 22868839-8450-4010-9241-1ea83a2e425f.", req.Path),
		)
	}
}
