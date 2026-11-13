package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

// incidentSeverityValues are the severities of an incident, also shown on
// status pages during maintenance.
var incidentSeverityValues = allowedValues[string]{
	{Value: "MINOR"},
	{Value: "MEDIUM"},
	{Value: "MAJOR"},
	{Value: "CRITICAL"},
}

const triggerIncidentAttributeName = "trigger_incident"

// TriggerIncidentModel describes the trigger_incident attribute data model.
type TriggerIncidentModel struct {
	ServiceID         types.String `tfsdk:"service_id"`
	Severity          types.String `tfsdk:"severity"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	NotifySubscribers types.Bool   `tfsdk:"notify_subscribers"`
}

var triggerIncidentAttrTypes = map[string]attr.Type{
	"service_id":         types.StringType,
	"severity":           types.StringType,
	"name":               types.StringType,
	"description":        types.StringType,
	"notify_subscribers": types.BoolType,
}

// makeTriggerIncidentSchema returns the schema for the trigger_incident
// attribute.
func makeTriggerIncidentSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Description: "Create and resolve an incident based on the alert configuration. Useful for status page automation.",
		Attributes: map[string]schema.Attribute{
			"service_id": schema.StringAttribute{
				Required:    true,
				Description: "The status page service that this incident will be associated with.",
			},
			"severity": schema.StringAttribute{
				Required:    true,
				Description: "The severity level of the incident. " + incidentSeverityValues.String(),
				Validators: []validator.String{
					stringvalidator.OneOf(incidentSeverityValues.Values()...),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the incident.",
			},
			"description": schema.StringAttribute{
				Required:    true,
				Description: "A detailed description of the incident.",
			},
			"notify_subscribers": schema.BoolAttribute{
				Required:    true,
				Description: "Whether to notify subscribers when the incident is triggered.",
			},
		},
	}
}

// triggerIncidentFromObject converts the trigger_incident attribute value to
// its API representation. A null or unknown value converts to nil.
func triggerIncidentFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) *checkly.IncidentTrigger {
	model := objectAs[TriggerIncidentModel](ctx, obj, diags)
	if model == nil {
		return nil
	}

	return &checkly.IncidentTrigger{
		ServiceID:         model.ServiceID.ValueString(),
		Severity:          checkly.IncidentSeverity(model.Severity.ValueString()),
		Name:              model.Name.ValueString(),
		Description:       model.Description.ValueString(),
		NotifySubscribers: model.NotifySubscribers.ValueBool(),
	}
}

// triggerIncidentToObject converts the API representation of an incident
// trigger to the trigger_incident attribute value. No incident trigger
// converts to null.
func triggerIncidentToObject(ctx context.Context, it *checkly.IncidentTrigger, diags *diag.Diagnostics) types.Object {
	if it == nil {
		return types.ObjectNull(triggerIncidentAttrTypes)
	}

	obj, d := types.ObjectValueFrom(ctx, triggerIncidentAttrTypes, TriggerIncidentModel{
		ServiceID:         types.StringValue(it.ServiceID),
		Severity:          types.StringValue(string(it.Severity)),
		Name:              types.StringValue(it.Name),
		Description:       types.StringValue(it.Description),
		NotifySubscribers: types.BoolValue(it.NotifySubscribers),
	})
	diags.Append(d...)

	return obj
}
