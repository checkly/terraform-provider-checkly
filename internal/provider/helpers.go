package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/checkly/checkly-go-sdk/v2"
)

// rateLimitAllowance is how much longer than a single request an API call
// may take, to wait out rate limits: the SDK re-sends a rate-limited (HTTP
// 429) request after the wait the API asks for, up to a minute, but only if
// the call's deadline leaves room for the wait and the re-sent request.
const rateLimitAllowance = 2 * time.Minute

// apiCallTimeout bounds one call to the Checkly API, including any time spent
// waiting out rate limits. Each HTTP request within it is bounded by
// apiRequestTimeout.
func apiCallTimeout() time.Duration {
	return apiRequestTimeout() + rateLimitAllowance
}

// defaultAPIRequestTimeout bounds a single HTTP request to the Checkly API
// when API_CALL_TIMEOUT is not set.
const defaultAPIRequestTimeout = 15 * time.Second

// apiRequestTimeout bounds a single HTTP request to the Checkly API:
// API_CALL_TIMEOUT seconds, 15 by default. The provider's Configure rejects an
// invalid API_CALL_TIMEOUT, so the default here only stands in for a value
// that has already been reported.
func apiRequestTimeout() time.Duration {
	timeout, err := parseAPIRequestTimeout()
	if err != nil {
		return defaultAPIRequestTimeout
	}
	return timeout
}

// parseAPIRequestTimeout reads API_CALL_TIMEOUT, a positive whole number of
// seconds.
func parseAPIRequestTimeout() (time.Duration, error) {
	timeout := os.Getenv("API_CALL_TIMEOUT")
	if timeout == "" {
		return defaultAPIRequestTimeout, nil
	}

	v, err := strconv.ParseInt(timeout, 10, 64)
	// The upper bound keeps the conversion to a Duration from overflowing.
	if err != nil || v < 1 || v > int64(math.MaxInt64/time.Second) {
		return 0, fmt.Errorf("invalid API_CALL_TIMEOUT value %q: must be a positive whole number of seconds", timeout)
	}
	return time.Duration(v) * time.Second, nil
}

// contextWithAPITimeout returns a child context with the standard API call
// timeout applied.
func contextWithAPITimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, apiCallTimeout())
}

// providerData is what the provider hands to resources and data sources in
// Configure: the API client, and the account it acts on, read only when a
// resource needs it.
type providerData struct {
	client  checkly.Client
	account *accountLookup
}

func newProviderData(client checkly.Client) *providerData {
	return &providerData{client: client, account: &accountLookup{client: client}}
}

// accountFromProviderData returns the account lookup of the provider data,
// or nil when Terraform has not configured the provider yet.
// clientFromProviderData reports provider data of an unexpected type.
func accountFromProviderData(data any) *accountLookup {
	d, ok := data.(*providerData)
	if !ok {
		return nil
	}

	return d.account
}

// clientFromProviderData extracts the Checkly API client from the provider
// data that the provider stores in ResourceData/DataSourceData during
// Configure. It returns nil (without diagnostics) when data is nil, which
// happens when Terraform has not configured the provider yet.
func clientFromProviderData(data any, diags *diag.Diagnostics) checkly.Client {
	if data == nil {
		return nil
	}

	d, ok := data.(*providerData)
	if !ok {
		diags.AddError(
			"Unexpected Provider Data Type",
			fmt.Sprintf("Expected *providerData, got: %T. Please report this issue to the provider developers.", data),
		)
		return nil
	}

	return d.client
}

// isNotFoundError reports whether an error returned by the Checkly API
// indicates that the resource no longer exists remotely: the API answered
// 404. It checks the status of the SDK's typed *checkly.APIError, also when
// wrapped, rather than the error text, so a "404" in a request path, an ID or
// a backend message never counts as a missing resource.
func isNotFoundError(err error) bool {
	return checkly.IsNotFound(err)
}

// decodeNumericID parses a numeric resource ID from its string form.
func decodeNumericID(id string) (int64, error) {
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric ID %q: %w", id, err)
	}

	return n, nil
}

// encodeNumericID formats a numeric resource ID as a string.
func encodeNumericID(id int64) string {
	return strconv.FormatInt(id, 10)
}

// checksumSha256 computes the hex-encoded SHA-256 checksum of the data read
// from r.
func checksumSha256(r io.Reader) (string, error) {
	hash := sha256.New()

	if _, err := io.Copy(hash, r); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

type allowedValue[T any] struct {
	Value       T
	Description string
}

func (v *allowedValue[T]) String() string {
	if v.Description != "" {
		return fmt.Sprintf("`%v` (%s)", v.Value, v.Description)
	}

	return fmt.Sprintf("`%v`", v.Value)
}

type allowedValues[T any] []allowedValue[T]

func (v *allowedValues[T]) Values() []T {
	s := make([]T, 0, len(*v))

	for _, value := range *v {
		s = append(s, value.Value)
	}

	return s
}

func (v *allowedValues[T]) String() string {
	l := len(*v)
	switch l {
	case 0:
		return "There are no possible values."
	case 1:
		return fmt.Sprintf("The only possible value is %s.", (*v)[0].String())
	default:
		head := (*v)[:l-1]
		last := (*v)[l-1]

		var buf strings.Builder

		buf.WriteString("Possible values are ")

		for i, value := range head {
			if i > 0 {
				buf.WriteString(", ")
			}

			buf.WriteString(value.String())
		}

		buf.WriteString(" and ")
		buf.WriteString(last.String())
		buf.WriteString(".")

		return buf.String()
	}
}

// planNestedDefaults replaces the planned value at p with withDefaults
// applied to it. It implements ModifyPlan for resources whose nested
// optional+computed attributes cannot use schema defaults, typically because
// they are nested in sets: substituting static values instead keeps every set
// element fully known and stable across plans, which set element correlation
// requires.
//
// Destroy plans have no planned state to default values in and are left
// alone. So are plans whose configuration at p contains unknown values
// (references to attributes of resources that have not been created yet):
// null values cannot be distinguished from values that will resolve at apply
// time, and the apply-time re-plan applies the defaults once the
// configuration values are known.
func planNestedDefaults[T attr.Value](ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse, p path.Path, withDefaults func(context.Context, T, *diag.Diagnostics) T) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var config T
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, p, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if valueContainsUnknown(ctx, config) {
		return
	}

	var planned T
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, p, &planned)...)
	if resp.Diagnostics.HasError() {
		return
	}

	planned = withDefaults(ctx, planned, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, p, planned)...)
}

// setElementsWithDefaults applies defaults to the elements of a set value,
// for plan modification of optional+computed attributes nested in set
// elements, where schema defaults cannot be used. Each element is decoded
// into its model type T, passed to withDefaults to substitute default values
// for null and unknown attributes or to transform nested values in place, and
// encoded back into a set of the original element type.
//
// Null and unknown sets are returned unchanged. So is the set when an element
// cannot be decoded, for example because it is null or unknown, which is
// reported as an error, or when diags already holds an error.
func setElementsWithDefaults[T any](ctx context.Context, set types.Set, withDefaults func(*T), diags *diag.Diagnostics) types.Set {
	if set.IsNull() || set.IsUnknown() {
		return set
	}

	var models []T
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return set
	}

	for i := range models {
		withDefaults(&models[i])
	}

	updated, d := types.SetValueFrom(ctx, set.ElementType(ctx), models)
	diags.Append(d...)

	return updated
}

// objectAs decodes an object value into its model type T. It returns nil when
// the value is null or unknown, or when diags holds an error after decoding.
func objectAs[T any](ctx context.Context, obj types.Object, diags *diag.Diagnostics) *T {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}

	var model T
	diags.Append(obj.As(ctx, &model, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil
	}

	return &model
}
