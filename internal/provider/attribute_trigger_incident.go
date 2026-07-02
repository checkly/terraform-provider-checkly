package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

const triggerIncidentAttributeName = "trigger_incident"

// TriggerIncidentModel describes the trigger_incident block data model.
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
// block.
func makeTriggerIncidentSchema() schema.SetNestedBlock {
	return schema.SetNestedBlock{
		Description: "Create and resolve an incident based on the alert configuration. Useful for status page automation.",
		Validators: []validator.Set{
			setvalidator.SizeAtMost(1),
		},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"service_id": schema.StringAttribute{
					Required:    true,
					Description: "The status page service that this incident will be associated with.",
				},
				"severity": schema.StringAttribute{
					Required:    true,
					Description: "The severity level of the incident. Possible values are `MINOR`, `MEDIUM`, `MAJOR`, and `CRITICAL`.",
					Validators: []validator.String{
						stringvalidator.OneOf("MINOR", "MEDIUM", "MAJOR", "CRITICAL"),
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
		},
	}
}

// triggerIncidentFromSet converts the trigger_incident block value to its
// API representation. An empty, null or unknown value converts to nil.
func triggerIncidentFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) *checkly.IncidentTrigger {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}

	var models []TriggerIncidentModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() || len(models) == 0 {
		return nil
	}

	model := models[0]

	return &checkly.IncidentTrigger{
		ServiceID:         model.ServiceID.ValueString(),
		Severity:          checkly.IncidentSeverity(model.Severity.ValueString()),
		Name:              model.Name.ValueString(),
		Description:       model.Description.ValueString(),
		NotifySubscribers: model.NotifySubscribers.ValueBool(),
	}
}

// triggerIncidentToSet converts the API representation of an incident
// trigger to the trigger_incident block value.
func triggerIncidentToSet(ctx context.Context, it *checkly.IncidentTrigger, diags *diag.Diagnostics) types.Set {
	elementType := types.ObjectType{AttrTypes: triggerIncidentAttrTypes}

	if it == nil {
		return types.SetValueMust(elementType, []attr.Value{})
	}

	set, d := types.SetValueFrom(ctx, elementType, []TriggerIncidentModel{
		{
			ServiceID:         types.StringValue(it.ServiceID),
			Severity:          types.StringValue(string(it.Severity)),
			Name:              types.StringValue(it.Name),
			Description:       types.StringValue(it.Description),
			NotifySubscribers: types.BoolValue(it.NotifySubscribers),
		},
	})
	diags.Append(d...)

	return set
}
