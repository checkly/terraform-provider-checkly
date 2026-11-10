package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	activatedAttributeName        = "activated"
	mutedAttributeName            = "muted"
	shouldFailAttributeName       = "should_fail"
	runParallelAttributeName      = "run_parallel"
	locationsAttributeName        = "locations"
	privateLocationsAttributeName = "private_locations"
	tagsAttributeName             = "tags"
	groupIDAttributeName          = "group_id"
	groupOrderAttributeName       = "group_order"
	runtimeIDAttributeName        = "runtime_id"
)

// CheckSubject names what a resource manages, for the wording of the shared
// attribute descriptions.
type CheckSubject string

const (
	subjectCheck      CheckSubject = "check"
	subjectMonitor    CheckSubject = "monitor"
	subjectCheckGroup CheckSubject = "check group"
)

// shouldFailErrorStatusNote is the should_fail note of the resources that
// assert the status code of an HTTP response: API checks (including
// checkly_check, whose API type does) and URL monitors.
const shouldFailErrorStatusNote = "Use it to validate an error status such as 404."

// CommonCheckAttributeSchemaOptions selects and adapts the attributes that
// checks, monitors and check groups share. The zero value (apart from
// Subject) yields the attributes every such resource has: activated, muted,
// tags and use_global_alert_settings.
type CommonCheckAttributeSchemaOptions struct {
	Subject CheckSubject
	// ActivatedOptional makes activated optional with a default of true;
	// otherwise it is required.
	ActivatedOptional bool
	ShouldFail        bool // should_fail
	RunParallel       bool // run_parallel
	Locations         bool // locations and private_locations
	GroupMembership   bool // group_id and group_order
	RuntimeID         bool // runtime_id
	// NoDegradedState words muted for a subject that fails or recovers but
	// is never degraded, such as a heartbeat monitor.
	NoDegradedState bool
	// Notes are sentences added to the shared descriptions of should_fail,
	// private_locations and runtime_id.
	ShouldFailNote       string
	PrivateLocationsNote string
	RuntimeIDNote        string
}

// makeCommonCheckAttributeSchemas returns the shared attributes that o
// selects, by attribute name.
func makeCommonCheckAttributeSchemas(o CommonCheckAttributeSchemaOptions) map[string]schema.Attribute {
	attrs := map[string]schema.Attribute{
		activatedAttributeName:              activatedAttributeSchema(o),
		mutedAttributeName:                  mutedAttributeSchema(o),
		tagsAttributeName:                   tagsAttributeSchema(o.Subject),
		useGlobalAlertSettingsAttributeName: makeUseGlobalAlertSettingsAttributeSchema(string(o.Subject)),
	}
	if o.ShouldFail {
		attrs[shouldFailAttributeName] = shouldFailAttributeSchema(o.Subject, o.ShouldFailNote)
	}
	if o.RunParallel {
		attrs[runParallelAttributeName] = runParallelAttributeSchema(o.Subject)
	}
	if o.Locations {
		attrs[locationsAttributeName] = locationsAttributeSchema(o.Subject)
		attrs[privateLocationsAttributeName] = privateLocationsAttributeSchema(o.Subject, o.PrivateLocationsNote)
	}
	if o.GroupMembership {
		attrs[groupIDAttributeName] = groupIDAttributeSchema(o.Subject)
		attrs[groupOrderAttributeName] = groupOrderAttributeSchema(o.Subject)
	}
	if o.RuntimeID {
		attrs[runtimeIDAttributeName] = runtimeIDAttributeSchema(o.Subject, o.RuntimeIDNote)
	}

	return attrs
}

// withCommonCheckAttributes adds the shared attributes that o selects to
// attrs and returns attrs. It panics if attrs already defines one of them: a
// resource either takes an attribute from the builder or defines it itself,
// never both.
func withCommonCheckAttributes(attrs map[string]schema.Attribute, o CommonCheckAttributeSchemaOptions) map[string]schema.Attribute {
	for name, attr := range makeCommonCheckAttributeSchemas(o) {
		if _, ok := attrs[name]; ok {
			panic(fmt.Sprintf("attribute %q is defined both by the resource and by the shared check attributes", name))
		}
		attrs[name] = attr
	}

	return attrs
}

// describe joins the sentences of a description, leaving out empty ones. A
// default marker such as "(Default `false`)." goes last.
func describe(sentences ...string) string {
	var description string
	for _, sentence := range sentences {
		if sentence == "" {
			continue
		}
		if description != "" {
			description += " "
		}
		description += sentence
	}

	return description
}

func activatedAttributeSchema(o CommonCheckAttributeSchemaOptions) schema.BoolAttribute {
	description := fmt.Sprintf("Determines whether the %s runs on its schedule.", o.Subject)
	if o.Subject == subjectCheckGroup {
		description = "Determines whether the checks and monitors in the group run on their schedule."
	}

	if !o.ActivatedOptional {
		return schema.BoolAttribute{
			Required:    true,
			Description: description,
		}
	}

	return schema.BoolAttribute{
		Optional:    true,
		Computed:    true,
		Default:     booldefault.StaticBool(true),
		Description: describe(description, "(Default `true`)."),
	}
}

func mutedAttributeSchema(o CommonCheckAttributeSchemaOptions) schema.BoolAttribute {
	description := fmt.Sprintf("Determines whether notifications are sent when the %s fails, degrades or recovers.", o.Subject)
	if o.NoDegradedState {
		description = fmt.Sprintf("Determines whether notifications are sent when the %s fails or recovers.", o.Subject)
	}
	if o.Subject == subjectCheckGroup {
		description = "Determines whether notifications are sent when a check or monitor in the group fails or recovers."
	}

	return schema.BoolAttribute{
		Optional:    true,
		Computed:    true,
		Default:     booldefault.StaticBool(false),
		Description: describe(description, "(Default `false`)."),
	}
}

func shouldFailAttributeSchema(subject CheckSubject, note string) schema.BoolAttribute {
	return schema.BoolAttribute{
		Optional:    true,
		Computed:    true,
		Default:     booldefault.StaticBool(false),
		Description: describe(fmt.Sprintf("Inverts when the %s is considered to fail.", subject), note, "(Default `false`)."),
	}
}

func runParallelAttributeSchema(subject CheckSubject) schema.BoolAttribute {
	description := fmt.Sprintf("Determines whether the %s runs in all its locations in parallel instead of "+
		"in one location at a time, round-robin.", subject)
	if subject == subjectCheckGroup {
		description = "Determines whether the checks and monitors in the group run in all their locations in " +
			"parallel instead of round-robin."
	}

	return schema.BoolAttribute{
		Optional:    true,
		Computed:    true,
		Default:     booldefault.StaticBool(false),
		Description: describe(description, "(Default `false`)."),
	}
}

// The collection attributes have no default and are not computed: when they
// are not configured they are null, which the API treats as empty.

func locationsAttributeSchema(subject CheckSubject) schema.SetAttribute {
	description := fmt.Sprintf("The data center locations to run the %[1]s from, such as `us-east-1`. The %[1]s "+
		"needs at least one location in `locations` or `private_locations`, unless it is part of a check group.", subject)
	if subject == subjectCheckGroup {
		description = "The data center locations to run the checks and monitors in the group from, such as `us-east-1`."
	}

	return schema.SetAttribute{
		ElementType: types.StringType,
		Optional:    true,
		Description: description,
	}
}

func privateLocationsAttributeSchema(subject CheckSubject, note string) schema.SetAttribute {
	description := fmt.Sprintf("The slugs of the private locations to run the %s from.", subject)
	if subject == subjectCheckGroup {
		description = "The slugs of the private locations to run the checks and monitors in the group from."
	}

	return schema.SetAttribute{
		ElementType: types.StringType,
		Optional:    true,
		Description: describe(description, note),
	}
}

func tagsAttributeSchema(subject CheckSubject) schema.SetAttribute {
	description := "Tags for organizing and filtering checks and monitors."
	if subject == subjectCheckGroup {
		// The API adds a group's tags to the tags of each check and monitor
		// in it wherever it selects checks by tag.
		description = describe(description, "The checks and monitors in the group also carry these tags, for "+
			"example when maintenance windows and status page automation rules select checks by tag.")
	}

	return schema.SetAttribute{
		ElementType: types.StringType,
		Optional:    true,
		Description: description,
	}
}

func groupIDAttributeSchema(subject CheckSubject) schema.Int64Attribute {
	return schema.Int64Attribute{
		Optional:    true,
		Computed:    true,
		Default:     int64default.StaticInt64(0),
		Description: fmt.Sprintf("The ID of the check group the %s is part of. (Default `0`, no group).", subject),
	}
}

func groupOrderAttributeSchema(subject CheckSubject) schema.Int64Attribute {
	return schema.Int64Attribute{
		Optional: true,
		Computed: true,
		Default:  int64default.StaticInt64(0),
		Description: fmt.Sprintf("The position of the %s in its check group, which sets the order in which the "+
			"checks and monitors of the group run when the group is triggered from the API or CI/CD. "+
			"(Default `0`).", subject),
	}
}

func runtimeIDAttributeSchema(subject CheckSubject, note string) schema.StringAttribute {
	description := fmt.Sprintf("The ID of the runtime to use for the %s.", subject)
	if subject == subjectCheckGroup {
		description = "The ID of the runtime to use for the checks in the group."
	}

	// Checks and check groups fall back to the account's default runtime;
	// what an unset runtime means for a monitor is not documented.
	var fallback string
	if subject != subjectMonitor {
		fallback = "When unset, the account's default runtime is used."
	}

	return schema.StringAttribute{
		Optional:    true,
		Description: describe(description, fallback, note),
	}
}
