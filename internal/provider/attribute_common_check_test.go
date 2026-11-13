package provider

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// commonCheckAttributeNames lists every attribute the shared builder can
// produce.
var commonCheckAttributeNames = []string{
	activatedAttributeName,
	mutedAttributeName,
	shouldFailAttributeName,
	runParallelAttributeName,
	locationsAttributeName,
	privateLocationsAttributeName,
	tagsAttributeName,
	useGlobalAlertSettingsAttributeName,
	groupIDAttributeName,
	groupOrderAttributeName,
	runtimeIDAttributeName,
}

// attributeDefault returns the static default of a, or nil.
func attributeDefault(t *testing.T, ctx context.Context, a schema.Attribute) attr.Value {
	t.Helper()

	switch a := a.(type) {
	case schema.BoolAttribute:
		if a.Default == nil {
			return nil
		}
		var resp defaults.BoolResponse
		a.Default.DefaultBool(ctx, defaults.BoolRequest{}, &resp)
		return resp.PlanValue
	case schema.Int64Attribute:
		if a.Default == nil {
			return nil
		}
		var resp defaults.Int64Response
		a.Default.DefaultInt64(ctx, defaults.Int64Request{}, &resp)
		return resp.PlanValue
	case schema.SetAttribute:
		if a.Default == nil {
			return nil
		}
		var resp defaults.SetResponse
		a.Default.DefaultSet(ctx, defaults.SetRequest{}, &resp)
		return resp.PlanValue
	case schema.StringAttribute:
		if a.Default != nil {
			t.Errorf("string attribute has a default")
		}
		return nil
	}
	t.Fatalf("unexpected attribute type %T", a)
	return nil
}

func TestCommonCheckAttributeSchemas(t *testing.T) {
	ctx := context.Background()

	type shape struct {
		required, optional, computed bool
		def                          attr.Value
	}
	optionalDefault := func(v attr.Value) shape { return shape{optional: true, computed: true, def: v} }
	optional := shape{optional: true}
	emptyStringSet := types.SetValueMust(types.StringType, []attr.Value{})

	all := CommonCheckAttributeSchemaOptions{
		Subject:         subjectCheck,
		ShouldFail:      true,
		RunParallel:     true,
		Locations:       true,
		GroupMembership: true,
		RuntimeID:       true,
	}
	allShapes := map[string]shape{
		activatedAttributeName:        optionalDefault(types.BoolValue(true)),
		mutedAttributeName:            optionalDefault(types.BoolValue(false)),
		shouldFailAttributeName:       optionalDefault(types.BoolValue(false)),
		runParallelAttributeName:      optionalDefault(types.BoolValue(false)),
		locationsAttributeName:        optionalDefault(emptyStringSet),
		privateLocationsAttributeName: optionalDefault(emptyStringSet),
		tagsAttributeName:             optionalDefault(emptyStringSet),
		// use_global_alert_settings is derived by a plan modifier instead
		// of a static default.
		useGlobalAlertSettingsAttributeName: {optional: true, computed: true},
		groupIDAttributeName:                optional,
		groupOrderAttributeName:             optional,
		runtimeIDAttributeName:              optional,
	}

	cases := map[string]struct {
		options CommonCheckAttributeSchemaOptions
		want    map[string]shape
	}{
		"every attribute": {all, allShapes},
		"zero value": {
			CommonCheckAttributeSchemaOptions{Subject: subjectMonitor},
			map[string]shape{
				activatedAttributeName:              allShapes[activatedAttributeName],
				mutedAttributeName:                  allShapes[mutedAttributeName],
				tagsAttributeName:                   allShapes[tagsAttributeName],
				useGlobalAlertSettingsAttributeName: allShapes[useGlobalAlertSettingsAttributeName],
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got := makeCommonCheckAttributeSchemas(c.options)
			if len(got) != len(c.want) {
				t.Errorf("got %d attributes, want %d", len(got), len(c.want))
			}
			for attrName, want := range c.want {
				a, ok := got[attrName]
				if !ok {
					t.Errorf("%s: missing", attrName)
					continue
				}
				if a.IsRequired() != want.required || a.IsOptional() != want.optional || a.IsComputed() != want.computed {
					t.Errorf("%s: required=%v optional=%v computed=%v, want %v %v %v", attrName,
						a.IsRequired(), a.IsOptional(), a.IsComputed(), want.required, want.optional, want.computed)
				}
				def := attributeDefault(t, ctx, a)
				if (def == nil) != (want.def == nil) || (def != nil && !def.Equal(want.def)) {
					t.Errorf("%s: default %v, want %v", attrName, def, want.def)
				}
				if !hasDescription(a.GetDescription()) {
					t.Errorf("%s: no description", attrName)
				}
			}
		})
	}
}

// use_global_alert_settings is derived from alert_settings and warns when
// both are set; the builder must keep the plan modifier and validator that
// do this.
func TestCommonCheckUseGlobalAlertSettings(t *testing.T) {
	a, ok := makeCommonCheckAttributeSchemas(CommonCheckAttributeSchemaOptions{Subject: subjectCheck})[useGlobalAlertSettingsAttributeName].(schema.BoolAttribute)
	if !ok {
		t.Fatal("use_global_alert_settings is not a bool attribute")
	}
	if !slices.ContainsFunc(a.PlanModifiers, func(m planmodifier.Bool) bool {
		_, ok := m.(useGlobalAlertSettingsPlanModifier)
		return ok
	}) {
		t.Error("lacks useGlobalAlertSettingsPlanModifier")
	}
	if !slices.ContainsFunc(a.Validators, func(v validator.Bool) bool {
		_, ok := v.(useGlobalAlertSettingsValidator)
		return ok
	}) {
		t.Error("lacks useGlobalAlertSettingsValidator")
	}
}

func TestCommonCheckMutedWording(t *testing.T) {
	cases := map[string]struct {
		options CommonCheckAttributeSchemaOptions
		want    string
	}{
		"check": {
			CommonCheckAttributeSchemaOptions{Subject: subjectCheck},
			"Determines whether notifications are sent when the check fails, degrades or recovers. Defaults to `false`.",
		},
		"no degraded state": {
			CommonCheckAttributeSchemaOptions{Subject: subjectMonitor, NoDegradedState: true},
			"Determines whether notifications are sent when the monitor fails or recovers. Defaults to `false`.",
		},
		"check group": {
			CommonCheckAttributeSchemaOptions{Subject: subjectCheckGroup},
			"Determines whether notifications are sent when a check or monitor in the group fails or recovers. " +
				"Muting the group mutes all of them, regardless of their own `muted`. Defaults to `false`.",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := makeCommonCheckAttributeSchemas(c.options)[mutedAttributeName].GetDescription(); got != c.want {
				t.Errorf("muted description = %q, want %q", got, c.want)
			}
		})
	}
}

func TestCommonCheckRuntimeIDWording(t *testing.T) {
	cases := map[string]struct {
		options CommonCheckAttributeSchemaOptions
		want    string
	}{
		"check": {
			CommonCheckAttributeSchemaOptions{Subject: subjectCheck, RuntimeID: true, RuntimeIDNote: "The runtime must support multi-step checks."},
			"The ID of the runtime to use for the check. If not set, the account's default runtime is used. The runtime must support multi-step checks.",
		},
		"monitor": {
			CommonCheckAttributeSchemaOptions{Subject: subjectMonitor, RuntimeID: true},
			"The ID of the runtime to use for the monitor.",
		},
		"check group": {
			CommonCheckAttributeSchemaOptions{Subject: subjectCheckGroup, RuntimeID: true},
			"The ID of the runtime to use for the checks in the group. If not set, the account's default runtime is used.",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := makeCommonCheckAttributeSchemas(c.options)[runtimeIDAttributeName].GetDescription(); got != c.want {
				t.Errorf("runtime_id description = %q, want %q", got, c.want)
			}
		})
	}
}

func TestWithCommonCheckAttributesRejectsDuplicates(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected a panic")
		}
	}()

	withCommonCheckAttributes(map[string]schema.Attribute{
		tagsAttributeName: schema.SetAttribute{ElementType: types.StringType, Optional: true},
	}, CommonCheckAttributeSchemaOptions{Subject: subjectCheck})
}

// TestResourcesUseCommonCheckAttributes pins the shared attributes of every
// check, monitor and check group to the builder: each resource listed here
// must have exactly the shared attributes its options select, identical to
// the builder's, including defaults, plan modifiers and validators, which
// the framework's own attribute comparison ignores. A resource missing from
// the list must have none of them.
func TestResourcesUseCommonCheckAttributes(t *testing.T) {
	ctx := context.Background()

	tcpMonitor := CommonCheckAttributeSchemaOptions{Subject: subjectMonitor, ShouldFail: true, RunParallel: true, Locations: true, GroupMembership: true, RuntimeID: true}
	heartbeatMonitor := CommonCheckAttributeSchemaOptions{Subject: subjectMonitor, NoDegradedState: true}

	options := map[string]CommonCheckAttributeSchemaOptions{
		"checkly_api_check":              {Subject: subjectCheck, ShouldFail: true, RunParallel: true, Locations: true, GroupMembership: true, RuntimeID: true, ShouldFailNote: shouldFailErrorStatusNote},
		"checkly_browser_check":          {Subject: subjectCheck, RunParallel: true, Locations: true, GroupMembership: true, RuntimeID: true},
		"checkly_multistep_check":        {Subject: subjectCheck, RunParallel: true, Locations: true, GroupMembership: true, RuntimeID: true, RuntimeIDNote: "The runtime must support multi-step checks."},
		"checkly_check":                  {Subject: subjectCheck, ShouldFail: true, RunParallel: true, Locations: true, GroupMembership: true, RuntimeID: true, ShouldFailNote: checkShouldFailNote},
		"checkly_check_group":            {Subject: subjectCheckGroup, RunParallel: true, Locations: true, RuntimeID: true},
		"checkly_url_monitor":            {Subject: subjectMonitor, ShouldFail: true, RunParallel: true, Locations: true, GroupMembership: true, ShouldFailNote: shouldFailErrorStatusNote},
		"checkly_dns_monitor":            {Subject: subjectMonitor, RunParallel: true, Locations: true, GroupMembership: true, PrivateLocationsNote: "A private location runs DNS monitors with agent v7.0.0 or later."},
		"checkly_icmp_monitor":           {Subject: subjectMonitor, RunParallel: true, Locations: true, GroupMembership: true, PrivateLocationsNote: "A private location runs ICMP monitors with agent v7.0.0 or later and the `NET_RAW` capability."},
		"checkly_tcp_monitor":            tcpMonitor,
		"checkly_tcp_check":              tcpMonitor,
		"checkly_grpc_monitor":           {Subject: subjectMonitor, RunParallel: true, Locations: true, GroupMembership: true, RuntimeID: true},
		"checkly_ssl_monitor":            {Subject: subjectMonitor, RunParallel: true, Locations: true, GroupMembership: true, RuntimeID: true},
		"checkly_traceroute_monitor":     {Subject: subjectMonitor, RunParallel: true, Locations: true, GroupMembership: true, RuntimeID: true, PrivateLocationsNote: "A private location runs traceroute monitors with agent v8.4.0 or later and the `NET_RAW` capability."},
		"checkly_heartbeat_monitor":      heartbeatMonitor,
		"checkly_heartbeat":              heartbeatMonitor,
		"checkly_playwright_check_suite": {Subject: subjectCheck, RunParallel: true, Locations: true, GroupMembership: true, NoDegradedState: true},
	}

	p := New("test")()
	var providerMetadata provider.MetadataResponse
	p.Metadata(ctx, provider.MetadataRequest{}, &providerMetadata)

	var seen []string
	for _, newResource := range p.Resources(ctx) {
		r := newResource()

		var metadata resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: providerMetadata.TypeName}, &metadata)

		var schemaResp resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
		attrs := schemaResp.Schema.Attributes

		o, ok := options[metadata.TypeName]
		if !ok {
			// Other resources may have attributes with these names, such as
			// the tags of a dashboard, but not the shared
			// use_global_alert_settings.
			if _, ok := attrs[useGlobalAlertSettingsAttributeName]; ok {
				t.Errorf("%s: has %s but does not use the shared check attributes", metadata.TypeName, useGlobalAlertSettingsAttributeName)
			}
			continue
		}
		seen = append(seen, metadata.TypeName)

		// use_global_alert_settings is derived from alert_settings.
		if _, ok := attrs[alertSettingsAttributeName]; !ok {
			t.Errorf("%s: has %s but no %s", metadata.TypeName, useGlobalAlertSettingsAttributeName, alertSettingsAttributeName)
		}

		want := makeCommonCheckAttributeSchemas(o)
		for _, name := range commonCheckAttributeNames {
			got, has := attrs[name]
			wantAttr, wants := want[name]
			switch {
			case has && !wants:
				t.Errorf("%s: has %s, which its options do not select", metadata.TypeName, name)
			case !has && wants:
				t.Errorf("%s: lacks %s", metadata.TypeName, name)
			case has && !reflect.DeepEqual(got, wantAttr):
				t.Errorf("%s: %s differs from the shared attribute:\n got %#v\nwant %#v", metadata.TypeName, name, got, wantAttr)
			}
		}
	}

	var want []string
	for name := range options {
		want = append(want, name)
	}
	slices.Sort(want)
	slices.Sort(seen)
	if !slices.Equal(seen, want) {
		t.Errorf("resources with the shared check attributes = %v, want %v", seen, want)
	}
}

// checkly_check_group_v2 describes its top-level attributes in terms of v2
// group semantics and does not use the builder, but its tags follow the same
// convention: optional and computed, with an empty default.
func TestCheckGroupV2TagsShape(t *testing.T) {
	var resp resource.SchemaResponse
	NewCheckGroupV2Resource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	tags, ok := resp.Schema.Attributes[tagsAttributeName].(schema.SetAttribute)
	if !ok {
		t.Fatalf("tags is %T", resp.Schema.Attributes[tagsAttributeName])
	}
	if !tags.Optional || !tags.Computed || tags.Default == nil {
		t.Fatalf("tags: optional=%v computed=%v default=%v, want optional and computed with a default", tags.Optional, tags.Computed, tags.Default)
	}
	var def defaults.SetResponse
	tags.Default.DefaultSet(context.Background(), defaults.SetRequest{}, &def)
	if want := types.SetValueMust(types.StringType, []attr.Value{}); !def.PlanValue.Equal(want) {
		t.Errorf("tags default = %s, want %s", def.PlanValue, want)
	}
}
