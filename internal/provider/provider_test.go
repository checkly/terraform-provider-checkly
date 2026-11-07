package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/checkly/checkly-go-sdk/v2"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// resourceSchemas returns the schema of every resource the provider
// registers, keyed by resource type name.
func resourceSchemas(t *testing.T) map[string]schema.Schema {
	t.Helper()

	ctx := context.Background()
	p := New("test")()

	var providerMetadata provider.MetadataResponse
	p.Metadata(ctx, provider.MetadataRequest{}, &providerMetadata)

	schemas := make(map[string]schema.Schema)
	for _, newResource := range p.Resources(ctx) {
		r := newResource()

		var metadata resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: providerMetadata.TypeName}, &metadata)

		var resp resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("%s: schema diagnostics: %v", metadata.TypeName, resp.Diagnostics)
		}

		schemas[metadata.TypeName] = resp.Schema
	}

	return schemas
}

func TestResourceSchemasAreValid(t *testing.T) {
	ctx := context.Background()

	for name, s := range resourceSchemas(t) {
		if diags := s.ValidateImplementation(ctx); diags.HasError() {
			t.Errorf("%s: invalid schema: %v", name, diags)
		}
	}
}

// Credentials must be marked Sensitive so Terraform redacts them in plan
// output. Attributes of nested objects and blocks are given as dotted paths.
func TestSensitiveAttributes(t *testing.T) {
	ctx := context.Background()

	var providerSchema provider.SchemaResponse
	New("test")().Schema(ctx, provider.SchemaRequest{}, &providerSchema)
	if !providerSchema.Schema.Attributes["api_key"].IsSensitive() {
		t.Errorf("provider attribute api_key is not Sensitive")
	}

	schemas := resourceSchemas(t)

	cases := map[string][]string{
		"checkly_client_certificate":   {"private_key", "passphrase"},
		"checkly_environment_variable": {"value"},
		"checkly_dashboard":            {"key"},
		"checkly_private_location":     {"keys"},
		"checkly_alert_channel": {
			"slack.url",
			"webhook.url",
			"webhook.webhook_secret",
			"webhook.headers",
			"webhook.query_parameters",
			"opsgenie.api_key",
			"pagerduty.service_key",
		},
		"checkly_api_check":              {"request.basic_auth.password"},
		"checkly_check":                  {"request.basic_auth.password", "environment_variable.value"},
		"checkly_check_group":            {"api_check_defaults.basic_auth.password", "environment_variable.value"},
		"checkly_check_group_v2":         {"api_check_defaults.basic_auth.password", "environment_variable.value"},
		"checkly_browser_check":          {"environment_variable.value"},
		"checkly_multistep_check":        {"environment_variable.value"},
		"checkly_playwright_check_suite": {"environment_variable.value"},
		"checkly_heartbeat_monitor":      {"heartbeat.ping_token"},
		"checkly_heartbeat":              {"heartbeat.ping_token"},
	}

	for resourceName, attrPaths := range cases {
		s, ok := schemas[resourceName]
		if !ok {
			t.Errorf("resource %s not found", resourceName)
			continue
		}
		for _, attrPath := range attrPaths {
			attr, ok := schemaAttributeAtPath(s, strings.Split(attrPath, "."))
			if !ok {
				t.Errorf("%s: attribute %s not found", resourceName, attrPath)
				continue
			}
			if !attr.IsSensitive() {
				t.Errorf("%s: attribute %s is not Sensitive", resourceName, attrPath)
			}
		}
	}
}

// schemaAttributeAtPath looks up an attribute by the names of the attributes
// and blocks that lead to it, skipping over list and set elements.
func schemaAttributeAtPath(s schema.Schema, names []string) (schema.Attribute, bool) {
	attributes, blocks := s.Attributes, s.Blocks
	for i, name := range names {
		if attr, ok := attributes[name]; ok {
			if i == len(names)-1 {
				return attr, true
			}
			switch attr := attr.(type) {
			case schema.SingleNestedAttribute:
				attributes, blocks = attr.Attributes, nil
			case schema.ListNestedAttribute:
				attributes, blocks = attr.NestedObject.Attributes, nil
			case schema.SetNestedAttribute:
				attributes, blocks = attr.NestedObject.Attributes, nil
			default:
				return nil, false
			}
			continue
		}

		switch block := blocks[name].(type) {
		case schema.ListNestedBlock:
			attributes, blocks = block.NestedObject.Attributes, block.NestedObject.Blocks
		case schema.SetNestedBlock:
			attributes, blocks = block.NestedObject.Attributes, block.NestedObject.Blocks
		default:
			return nil, false
		}
	}

	return nil, false
}

// Every API key needs a valid account ID, an empty configured one falls back
// to the environment, and no diagnostic repeats the credentials.
func TestConfigureValidatesAccountID(t *testing.T) {
	const (
		apiKey         = "cu_test-api-key"
		validAccountID = "00000000-0000-4000-8000-000000000000"
	)

	cases := map[string]struct {
		envAccountID string
		accountID    *string
		apiURL       *string
		want         string
	}{
		"missing":             {want: "Missing Checkly Account ID"},
		"empty without env":   {accountID: new(""), want: "Missing Checkly Account ID"},
		"empty falls to env":  {envAccountID: validAccountID, accountID: new("")},
		"null falls to env":   {envAccountID: validAccountID},
		"configured over env": {envAccountID: "not-a-uuid", accountID: new(validAccountID)},
		"invalid env":         {envAccountID: "not-a-uuid", want: "Invalid Checkly Account ID"},
		"invalid":             {accountID: new("not-a-uuid"), want: "Invalid Checkly Account ID"},
		"API key in its place": {
			accountID: new("cu_0123456789abcdef0123456789abcdef"),
			want:      "Invalid Checkly Account ID",
		},
		"invalid API URL": {
			accountID: new(validAccountID),
			apiURL:    new("ftp://example.com"),
			want:      "Unable to Create the Checkly API Client",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv("CHECKLY_API_KEY", apiKey)
			t.Setenv("CHECKLY_ACCOUNT_ID", c.envAccountID)
			t.Setenv("CHECKLY_API_URL", "")

			ctx := context.Background()
			p := New("test")()

			var schemaResp provider.SchemaResponse
			p.Schema(ctx, provider.SchemaRequest{}, &schemaResp)
			s := schemaResp.Schema

			optionalString := func(value *string) tftypes.Value {
				if value == nil {
					return tftypes.NewValue(tftypes.String, nil)
				}
				return tftypes.NewValue(tftypes.String, *value)
			}

			var resp provider.ConfigureResponse
			p.Configure(ctx, provider.ConfigureRequest{
				Config: tfsdk.Config{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), map[string]tftypes.Value{
					"api_key":    tftypes.NewValue(tftypes.String, nil),
					"api_url":    optionalString(c.apiURL),
					"account_id": optionalString(c.accountID),
				})},
			}, &resp)

			if c.want == "" {
				if resp.Diagnostics.HasError() {
					t.Fatal(resp.Diagnostics)
				}
				if resp.ResourceData == nil {
					t.Error("no client in the resource data")
				}
				return
			}

			if !resp.Diagnostics.HasError() {
				t.Fatal("expected an error diagnostic")
			}
			got := resp.Diagnostics.Errors()[0]
			if got.Summary() != c.want {
				t.Errorf("summary = %q, want %q", got.Summary(), c.want)
			}
			text := got.Summary() + " " + got.Detail()
			if strings.Contains(text, apiKey) {
				t.Errorf("diagnostic contains the API key: %q", text)
			}
			for _, accountID := range []string{c.envAccountID, valueOrZero(c.accountID)} {
				if accountID != "" && strings.Contains(text, accountID) {
					t.Errorf("diagnostic contains the account ID: %q", text)
				}
			}
		})
	}
}

// An empty configured value counts as not set for every setting, and the
// client sends the resulting credentials and source with every request.
func TestConfigureBuildsTheClientFromEnvironment(t *testing.T) {
	const (
		apiKey    = "cu_test-api-key"
		accountID = "00000000-0000-4000-8000-000000000000"
	)

	cases := map[string]struct {
		configured tftypes.Value
		source     string
		wantSource string
	}{
		"null values and the default source": {
			configured: tftypes.NewValue(tftypes.String, nil),
			wantSource: "TF",
		},
		"empty values and a configured source": {
			configured: tftypes.NewValue(tftypes.String, ""),
			source:     "my-pipeline",
			wantSource: "my-pipeline",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var headers http.Header
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				headers = r.Header.Clone()
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			t.Setenv("CHECKLY_API_KEY", apiKey)
			t.Setenv("CHECKLY_ACCOUNT_ID", accountID)
			t.Setenv("CHECKLY_API_URL", server.URL)
			t.Setenv("CHECKLY_API_SOURCE", c.source)

			ctx := context.Background()
			p := New("test")()

			var schemaResp provider.SchemaResponse
			p.Schema(ctx, provider.SchemaRequest{}, &schemaResp)
			s := schemaResp.Schema

			var resp provider.ConfigureResponse
			p.Configure(ctx, provider.ConfigureRequest{
				Config: tfsdk.Config{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), map[string]tftypes.Value{
					"api_key":    c.configured,
					"api_url":    c.configured,
					"account_id": c.configured,
				})},
			}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}

			client, ok := resp.ResourceData.(checkly.Client)
			if !ok {
				t.Fatalf("resource data is a %T", resp.ResourceData)
			}
			if _, err := client.GetEnvironmentVariable(ctx, "MISSING"); !isNotFoundError(err) {
				t.Fatalf("GetEnvironmentVariable error = %v, want a 404 from the test server", err)
			}

			if got := headers.Get("x-checkly-source"); got != c.wantSource {
				t.Errorf("x-checkly-source = %q, want %q", got, c.wantSource)
			}
			if got := headers.Get("x-checkly-account"); got != accountID {
				t.Errorf("x-checkly-account = %q, want %q", got, accountID)
			}
			if got := headers.Get("Authorization"); got != "Bearer "+apiKey {
				t.Errorf("Authorization header does not carry the API key")
			}
		})
	}
}
