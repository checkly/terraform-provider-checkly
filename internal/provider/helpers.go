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

	"github.com/hashicorp/terraform-plugin-framework/diag"

	checkly "github.com/checkly/checkly-go-sdk"
)

// apiCallTimeout returns the timeout used for individual Checkly API calls.
// It can be overridden using the API_CALL_TIMEOUT environment variable
// (in seconds).
func apiCallTimeout() time.Duration {
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
// indicates that the resource no longer exists remotely.
func isNotFoundError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "404")
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
