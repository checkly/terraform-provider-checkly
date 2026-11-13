package provider

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

// Response time limits, in milliseconds, of the checks and monitors whose
// runner timeout depends on the account. Without extended response time
// limits, TCP monitors time out after 5 seconds, and API checks and URL
// monitors after 30 seconds; with them, all of them wait up to 45 seconds.
const (
	tcpStandardResponseTimeLimit  = 5000
	httpStandardResponseTimeLimit = 30000
	extendedResponseTimeLimit     = 45000
)

// accountLookup reads the account the client acts on at most once per
// provider process, on first use, and keeps the result, an error included,
// for every later caller. Most plans never need the account, so it isn't
// read in Configure.
type accountLookup struct {
	client  checkly.Client
	once    sync.Once
	account *checkly.Account
	err     error
}

func (l *accountLookup) get(ctx context.Context) (*checkly.Account, error) {
	l.once.Do(func() {
		apiCtx, cancel := contextWithAPITimeout(ctx)
		defer cancel()

		l.account, l.err = l.client.GetAccount(apiCtx)
	})

	return l.account, l.err
}

// validateResponseTimeLimits fails the plan when the planned
// degraded_response_time or max_response_time is above standardLimit and the
// account doesn't have extended response time limits: the check times out
// at standardLimit, so the value couldn't work. The schema caps both at
// extendedResponseTimeLimit.
//
// The planned value is checked whether or not it changed, so a value above
// the limit that was set outside Terraform fails the plan too. Unknown and
// null values, a destroy plan and an unconfigured provider (a nil l) are
// skipped. When the account can't be read, a warning replaces the check.
func (l *accountLookup) validateResponseTimeLimits(ctx context.Context, plan tfsdk.Plan, standardLimit int64, diags *diag.Diagnostics) {
	if l == nil || plan.Raw.IsNull() {
		return
	}

	for _, name := range []string{"degraded_response_time", "max_response_time"} {
		var value types.Int64
		if getDiags := plan.GetAttribute(ctx, path.Root(name), &value); getDiags.HasError() {
			diags.Append(getDiags...)
			return
		}
		if value.IsNull() || value.IsUnknown() || value.ValueInt64() <= standardLimit {
			continue
		}

		account, err := l.get(ctx)
		if err != nil {
			diags.AddAttributeWarning(
				path.Root(name),
				"Unable to Verify Response Time Limit",
				fmt.Sprintf("`%s` is `%d`. Values above `%d` need extended limits to be approved for your account, "+
					"and the provider couldn't read the account to check this: %s", name, value.ValueInt64(), standardLimit, err),
			)
			continue
		}
		if !account.HasFeature(checkly.AccountFeatureExtendedResponseTimeLimits) {
			diags.AddAttributeError(
				path.Root(name),
				"Response Time Above Account Limit",
				fmt.Sprintf("`%s` is `%d`, but can be at most `%d` on this account. Values up to `%d` need extended limits "+
					"to be approved for your account; contact Checkly support to request them.",
					name, value.ValueInt64(), standardLimit, extendedResponseTimeLimit),
			)
		}
	}
}
