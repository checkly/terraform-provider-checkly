package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
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

	checkly "github.com/checkly/checkly-go-sdk"
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

// apiRequestTimeout bounds a single HTTP request to the Checkly API:
// API_CALL_TIMEOUT seconds, 15 by default.
func apiRequestTimeout() time.Duration {
	timeout := os.Getenv("API_CALL_TIMEOUT")
	if timeout != "" {
		v, err := strconv.ParseInt(timeout, 10, 64)
		if err != nil || v < 1 {
			panic("Invalid API_CALL_TIMEOUT value, must be a positive number")
		}
		return time.Duration(v) * time.Second
	}
	return 15 * time.Second
}

// contextWithAPITimeout returns a child context with the standard API call
// timeout applied.
func contextWithAPITimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, apiCallTimeout())
}

// clientFromProviderData extracts the Checkly API client that the provider
// stores in ResourceData/DataSourceData during Configure. It returns nil
// (without diagnostics) when providerData is nil, which happens when
// Terraform has not configured the provider yet.
func clientFromProviderData(providerData any, diags *diag.Diagnostics) checkly.Client {
	if providerData == nil {
		return nil
	}

	client, ok := providerData.(checkly.Client)
	if !ok {
		diags.AddError(
			"Unexpected Provider Data Type",
			fmt.Sprintf("Expected checkly.Client, got: %T. Please report this issue to the provider developers.", providerData),
		)
		return nil
	}

	return client
}

// isNotFoundError reports whether an error returned by the Checkly API
// indicates that the resource no longer exists remotely. The SDK formats a
// 404 response in one of two ways. Matching only those prefixes keeps a "404"
// elsewhere in the message, such as in a request URL, an ID or an echoed
// response body, from being taken for a missing resource.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()
	return strings.HasPrefix(msg, "unexpected response status 404:") ||
		strings.HasPrefix(msg, "unexpected response status: 404,")
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
func checksumSha256(r io.Reader) string {
	hash := sha256.New()

	_, err := io.Copy(hash, r)
	if err != nil {
		panic("failed to calculate checksum: " + err.Error())
	}

	checksum := hex.EncodeToString(hash.Sum(nil))

	return checksum
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
		return "There are no allowed values."
	case 1:
		return fmt.Sprintf("The only allowed value is %s.", (*v)[0].String())
	default:
		head := (*v)[:l-1]
		last := (*v)[l-1]

		var buf strings.Builder

		buf.WriteString("The allowed values are ")

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
