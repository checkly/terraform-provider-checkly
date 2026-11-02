package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

var (
	_ resource.Resource                = (*ClientCertificateResource)(nil)
	_ resource.ResourceWithConfigure   = (*ClientCertificateResource)(nil)
	_ resource.ResourceWithImportState = (*ClientCertificateResource)(nil)
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
			"be created.",
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
				Description: "The host domain that the certificate should be used for. Wildcards are supported, e.g. `*.acme.com`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"path": schema.StringAttribute{
				Optional: true,
				Description: "Optional URL path prefix that limits the certificate to requests under that path, " +
					"e.g. `/partner/api`. Matching is on whole path segments: `/partner` applies to `/partner` and " +
					"`/partner/orders` but not to `/partnership`. Must start with `/` and must not be a bare `/`; a " +
					"trailing `/` is ignored. API checks and Multistep checks match on path; gRPC, SSL and TCP " +
					"monitors only use certificates without a path. Omit it to use the " +
					"certificate for every path on the host.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				// The API refuses these too; failing at plan time is clearer
				// than an apply error, and a bare "/" would otherwise be
				// trimmed to an empty path and silently sent as "no path".
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^(/|$)`), `must start with "/"`),
					stringvalidator.RegexMatches(regexp.MustCompile(`^$|[^/]`), `must name a path below "/"; omit it to match every path`),
				},
			},
			"certificate": schema.StringAttribute{
				Required:    true,
				Description: "The client certificate in PEM format.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"private_key": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "The private key for the certificate in PEM format.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"passphrase": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Passphrase for the private key.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"trusted_ca": schema.StringAttribute{
				Optional:    true,
				Description: "PEM formatted bundle of CA certificates that the client should trust. The bundle may contain many CA certificates.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
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
		resp.Diagnostics.AddError(
			"Unable to Create Client Certificate",
			fmt.Sprintf("Checkly API error: %s", err),
		)
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
		resp.Diagnostics.AddError(
			"Unable to Read Client Certificate",
			fmt.Sprintf("Checkly API error: %s", err),
		)
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
	state.TrustedCA = stringValueOrNull(clientCertificate.TrustedCA)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is never called because every attribute requires replacement, but
// the framework requires an implementation.
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
		resp.Diagnostics.AddError(
			"Unable to Delete Client Certificate",
			fmt.Sprintf("Checkly API error: %s", err),
		)
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
