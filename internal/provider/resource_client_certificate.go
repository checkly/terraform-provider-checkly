package provider

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

var (
	_ resource.Resource                 = (*ClientCertificateResource)(nil)
	_ resource.ResourceWithConfigure    = (*ClientCertificateResource)(nil)
	_ resource.ResourceWithUpgradeState = (*ClientCertificateResource)(nil)
	_ resource.ResourceWithImportState  = (*ClientCertificateResource)(nil)
)

// NewClientCertificateResource returns a new checkly_client_certificate
// resource.
func NewClientCertificateResource() resource.Resource {
	return &ClientCertificateResource{}
}

// ClientCertificateResource implements the checkly_client_certificate
// resource.
type ClientCertificateResource struct {
	client checkly.Client
}

// ClientCertificateResourceModel describes the resource data model.
type ClientCertificateResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Host        types.String `tfsdk:"host"`
	Path        types.String `tfsdk:"path"`
	Certificate types.String `tfsdk:"certificate"`
	PrivateKey  types.String `tfsdk:"private_key"`
	Passphrase  types.String `tfsdk:"passphrase"`
	TrustedCA   types.String `tfsdk:"trusted_ca"`
}

func (r *ClientCertificateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_client_certificate"
}

func (r *ClientCertificateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: sdkv2SchemaVersion + 1,
		Description: "Use client certificates to authenticate your API " +
			"checks to APIs that require mutual TLS (mTLS) authentication, " +
			"or any other authentication scheme where the requester needs to " +
			"provide a certificate." +
			"\n\n" +
			"Each client certificate is specific to a host name, e.g. " +
			"`acme.com` or a wildcard such as `*.acme.com`, and will be used " +
			"automatically by any API checks targeting that host. Set `path` " +
			"to limit the certificate to requests under a URL path prefix." +
			"\n\n" +
			"Changing the value of any attribute forces a new resource to " +
			"be created, except for adding or removing a trailing `/` of " +
			"`path`, which the API ignores, and for the first apply after " +
			"an import. The API never returns `private_key` or " +
			"`passphrase`, so the provider cannot check an imported " +
			"certificate's key against the configured one: that apply " +
			"records the configured values in Terraform state only and " +
			"sends nothing to Checkly. To upload a different key, replace " +
			"the resource, for example with `terraform apply -replace`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the client certificate.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"host": schema.StringAttribute{
				Required:    true,
				Description: "The host domain that the certificate should be used for (at most 255 characters). Wildcards are supported, e.g. `*.acme.com`.",
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtMost(255),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"path": schema.StringAttribute{
				Optional: true,
				Description: "Optional URL path prefix that limits the certificate to requests under that path, " +
					"e.g. `/partner/api`. Matching is on whole path segments: `/partner` applies to `/partner` and " +
					"`/partner/orders` but not to `/partnership`. Must start with `/` and must not be a bare `/`; a " +
					"trailing `/` is ignored. The path must already be in the form the API stores: segments " +
					"separated by single slashes, with no `.` or `..` segments, no whitespace, control characters, " +
					"`?`, `#` or `\\`, and with non-ASCII characters and the characters `` \"<>^`{} `` " +
					"percent-encoded, e.g. `%7B` for `{` (at most " + strconv.Itoa(clientCertificatePathMaxLength) + " characters). " +
					"API checks and Multistep checks match on path; gRPC, SSL and TCP " +
					"monitors only use certificates without a path. Omit it to use the " +
					"certificate for every path on the host.",
				PlanModifiers: []planmodifier.String{
					// The API ignores a trailing slash, so changing only that
					// is an in-place update of the stored spelling. Releases
					// before v2.0.0 stored the path without it.
					stringplanmodifier.RequiresReplaceIf(
						func(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
							resp.RequiresReplace = req.PlanValue.IsUnknown() ||
								strings.TrimSuffix(req.StateValue.ValueString(), "/") !=
									strings.TrimSuffix(req.PlanValue.ValueString(), "/")
						},
						"Changing the path other than by a trailing slash forces a new resource to be created.",
						"Changing the path other than by a trailing slash forces a new resource to be created.",
					),
				},
				Validators: []validator.String{
					clientCertificatePathValidator{},
				},
			},
			"certificate": schema.StringAttribute{
				Required:    true,
				Description: "The client certificate in PEM format.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"private_key": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
				Description: "The private key for the certificate in PEM format. The API never returns it, so " +
					"the provider cannot check an imported certificate's key against the configured one: the " +
					"first apply after an import records the configured key and passphrase in Terraform state " +
					"only and sends nothing to Checkly. To upload a different key, replace the resource, for " +
					"example with `terraform apply -replace`.",
				// An empty key would create a certificate without one.
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIf(
						func(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
							resp.RequiresReplace = !isImportedPrivateKey(req.StateValue)
						},
						"Changing the private key forces a new resource to be created, except on the first apply after an import.",
						"Changing the private key forces a new resource to be created, except on the first apply after an import.",
					),
				},
			},
			"passphrase": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Passphrase for the private key (at most 95 bytes).",
				Validators: []validator.String{
					stringvalidator.LengthAtMost(95),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIf(
						func(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
							var privateKey types.String
							resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("private_key"), &privateKey)...)
							if resp.Diagnostics.HasError() || isImportedPrivateKey(privateKey) {
								return
							}
							requiresReplaceUnlessOnlyEmptiness(ctx, req, resp)
						},
						"Changing the passphrase forces a new resource to be created, except on the first apply after an import.",
						"Changing the passphrase forces a new resource to be created, except on the first apply after an import.",
					),
				},
			},
			"trusted_ca": schema.StringAttribute{
				Optional:    true,
				Description: "PEM formatted bundle of CA certificates that the client should trust. The bundle may contain many CA certificates.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIf(
						requiresReplaceUnlessOnlyEmptiness,
						"Changing the trusted CA bundle forces a new resource to be created.",
						"Changing the trusted CA bundle forces a new resource to be created.",
					),
				},
			},
		},
	}
}

// clientCertificatePathMaxLength is the API's limit on the length of a
// stored client certificate path.
const clientCertificatePathMaxLength = 2048

// clientCertificatePathRejected matches the characters the API rejects in a
// client certificate path: whitespace, control characters, "?", "#" and
// "\".
var clientCertificatePathRejected = regexp.MustCompile(`[\s\x00-\x1f\x7f?#\\]`)

// clientCertificatePathEncoded matches the characters that a WHATWG URL
// parser percent-encodes in a path and the API doesn't reject: the rest of
// the path percent-encode set ("<>^`{}) and non-ASCII characters. "%" is not
// among them: the parser keeps it as it is, also when it isn't followed by
// two hex digits.
var clientCertificatePathEncoded = regexp.MustCompile("[\"<>^`{}]|[^\\x00-\\x7f]")

// clientCertificatePathValidator accepts a client certificate path only in
// the form the API stores, apart from a trailing slash, which the API drops
// and the provider ignores. The API rejects some paths, such as ones with
// dot segments, "//", whitespace, "?" or "#", and rewrites others: it
// percent-encodes what a WHATWG URL parser encodes in a path, such as
// non-ASCII characters and {}"<>. A rewritten path would read
// back different from the configuration and replace the certificate on every
// apply. An empty path means none and is accepted.
type clientCertificatePathValidator struct{}

var _ validator.String = clientCertificatePathValidator{}

func (v clientCertificatePathValidator) Description(ctx context.Context) string {
	return "value must be a URL path below \"/\" in the form the API stores"
}

func (v clientCertificatePathValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v clientCertificatePathValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	if problem := clientCertificatePathProblem(req.ConfigValue.ValueString()); problem != "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Client Certificate Path",
			fmt.Sprintf("%s %s, got: %q", req.Path, problem, req.ConfigValue.ValueString()),
		)
	}
}

// clientCertificatePathProblem returns why the API would reject or rewrite
// path, phrased to follow the attribute name, or "" when it stores path as it
// is, apart from a trailing slash.
func clientCertificatePathProblem(path string) string {
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		return `must start with "/"`
	}

	if path == "/" {
		return `must name a path below "/"; omit it to match every path`
	}

	trimmed := strings.TrimSuffix(path[1:], "/")
	if len(trimmed)+1 > clientCertificatePathMaxLength {
		return fmt.Sprintf("must be at most %d characters long", clientCertificatePathMaxLength)
	}

	for _, segment := range strings.Split(trimmed, "/") {
		switch {
		case segment == "":
			return `must not contain empty segments ("//")`
		case isClientCertificatePathDotSegment(segment):
			return `must not contain "." or ".." segments`
		case clientCertificatePathRejected.MatchString(segment):
			return `must not contain whitespace, control characters, "?", "#" or "\"; write a space as %20`
		case clientCertificatePathEncoded.MatchString(segment):
			return `must have non-ASCII characters and the characters "<>^` + "`" + `{} percent-encoded, ` +
				`as the API stores them, for example %7B for {`
		}
	}

	return ""
}

// isClientCertificatePathDotSegment reports whether segment is a dot segment
// that a URL parser resolves: "." or "..", also with the dots
// percent-encoded as %2e in any case.
func isClientCertificatePathDotSegment(segment string) bool {
	decoded := strings.ReplaceAll(strings.ToLower(segment), "%2e", ".")
	return decoded == "." || decoded == ".."
}

// requiresReplaceUnlessOnlyEmptiness requires replacement for any change but
// one between "" and unset. The API treats an empty passphrase or trusted CA
// bundle as none, so that change leaves the certificate as it is.
func requiresReplaceUnlessOnlyEmptiness(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
	resp.RequiresReplace = req.PlanValue.IsUnknown() ||
		req.StateValue.ValueString() != req.PlanValue.ValueString()
}

// isImportedPrivateKey reports whether privateKey, read from prior state,
// holds no key. The attribute is required, so that happens only after an
// import, as the API never returns the key: null after a v2 import, "" after
// a v1.x import that was never applied; ValueString returns "" for both. The
// key and passphrase in the configuration then cannot be compared with the
// certificate's, so the first apply records them in state instead of
// replacing the certificate.
func isImportedPrivateKey(privateKey types.String) bool {
	return privateKey.ValueString() == ""
}

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *ClientCertificateResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

func (r *ClientCertificateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *ClientCertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ClientCertificateResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateClientCertificate(apiCtx, clientCertificateFromModel(&plan))
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to Create Client Certificate", err)
		return
	}

	plan.ID = types.StringValue(result.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ClientCertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ClientCertificateResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	clientCertificate, err := r.client.GetClientCertificate(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addAPIError(&resp.Diagnostics, "Unable to Read Client Certificate", err)
		return
	}

	state.Host = types.StringValue(clientCertificate.Host)
	state.Certificate = types.StringValue(clientCertificate.Certificate)
	// The API stores the path without a trailing slash. Keeping the
	// configured spelling when it only differs by that slash keeps config
	// and state equal, otherwise "/partner/api/" would force a replacement
	// on every apply.
	if strings.TrimSuffix(state.Path.ValueString(), "/") != clientCertificate.Path {
		state.Path = stringValueOrNull(clientCertificate.Path)
	}
	// The API does not return the private key, so the value already present
	// in state is kept as-is.
	// The API does not return the passphrase either, and even if it did, the
	// value would be encrypted. The value already present in state is kept
	// as-is.
	state.TrustedCA = stringValueOrPriorEmpty(clientCertificate.TrustedCA, state.TrustedCA)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update only stores changes the API ignores: a path that changed by a
// trailing slash, a passphrase or trusted CA bundle switched between "" and
// unset, or the private key and passphrase configured for an imported
// certificate, which the API cannot compare with the stored ones. Every other
// change requires replacement.
func (r *ClientCertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ClientCertificateResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ClientCertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ClientCertificateResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err := r.client.DeleteClientCertificate(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		addAPIError(&resp.Diagnostics, "Unable to Delete Client Certificate", err)
		return
	}
}

func (r *ClientCertificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func clientCertificateFromModel(model *ClientCertificateResourceModel) checkly.ClientCertificate {
	return checkly.ClientCertificate{
		ID:          model.ID.ValueString(),
		Host:        model.Host.ValueString(),
		Path:        strings.TrimSuffix(model.Path.ValueString(), "/"),
		Certificate: model.Certificate.ValueString(),
		PrivateKey:  model.PrivateKey.ValueString(),
		Passphrase:  model.Passphrase.ValueString(),
		TrustedCA:   model.TrustedCA.ValueString(),
	}
}
