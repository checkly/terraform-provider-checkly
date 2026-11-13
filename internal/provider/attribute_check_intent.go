package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

const (
	checkIntentAttributeName         = "intent"
	aiAutoRepairEnabledAttributeName = "ai_auto_repair_enabled"
)

// CheckIntentModel describes the intent attribute data model.
type CheckIntentModel struct {
	Goal             types.String `tfsdk:"goal"`
	RequiredOutcomes types.List   `tfsdk:"required_outcomes"`
	MustPreserve     types.List   `tfsdk:"must_preserve"`
}

var checkIntentAttrTypes = map[string]attr.Type{
	"goal":              types.StringType,
	"required_outcomes": types.ListType{ElemType: types.StringType},
	"must_preserve":     types.ListType{ElemType: types.StringType},
}

// makeCheckIntentStatementsSchema returns the schema for a list of intent
// statements.
func makeCheckIntentStatementsSchema(description string) schema.ListAttribute {
	return schema.ListAttribute{
		Optional:    true,
		Computed:    true,
		Default:     emptyListDefault(types.StringType),
		ElementType: types.StringType,
		Description: description,
		Validators: []validator.List{
			listvalidator.SizeAtMost(20),
			listvalidator.NoNullValues(),
			listvalidator.ValueStringsAre(
				utf16LengthBetween(1, 1000),
				trimmedStringValidator{},
			),
		},
	}
}

// makeCheckIntentSchema returns the schema for the intent attribute.
func makeCheckIntentSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Description: "Guidance for root cause analysis and automatic repair: what this check protects, and which changes keep it meaningful. " +
			"Automatic repair supports browser and multi-step checks only, and refuses an intent with no statements: a `goal` on its own is accepted here, but repair needs at least one statement in either `required_outcomes` or `must_preserve`. " +
			"An intent edited in the web app and absent from the configuration is removed by the next apply, every time. To leave it to the web app, add `intent` to `ignore_changes`.",
		Attributes: map[string]schema.Attribute{
			"goal": schema.StringAttribute{
				Required:    true,
				Description: "The user or system outcome the check protects. Must be between `1` and `2000` characters, without leading or trailing whitespace.",
				Validators: []validator.String{
					utf16LengthBetween(1, 2000),
					trimmedStringValidator{},
				},
			},
			"required_outcomes": makeCheckIntentStatementsSchema("Observable outcomes the check must continue to prove. Must contain at most `20` statements, each between `1` and `1000` characters, without leading or trailing whitespace."),
			"must_preserve":     makeCheckIntentStatementsSchema("Guardrails an automatic repair must not weaken or bypass. Must contain at most `20` statements, each between `1` and `1000` characters, without leading or trailing whitespace."),
		},
	}
}

// makeAIAutoRepairEnabledSchema returns the schema for the
// ai_auto_repair_enabled attribute.
func makeAIAutoRepairEnabledSchema() schema.BoolAttribute {
	return schema.BoolAttribute{
		Optional: true,
		Description: "Overrides the account's automatic repair setting for this check. Leave it out to inherit the account setting. " +
			"Automatic repair supports browser and multi-step checks only, so this setting has no effect on other check types today, and it needs an `intent` with at least one statement. " +
			"A repair is saved to the check, so it shows as a change to `script`, and the next apply restores the configured script unless you copy the repaired script into the configuration or add `script` to `ignore_changes`. " +
			"An override edited in the web app and absent from the configuration is removed by the next apply, every time. To leave it to the web app, add `ai_auto_repair_enabled` to `ignore_changes`.",
	}
}

// checkIntentFromObject converts the intent attribute value to its API
// representation. A null or unknown value converts to nil.
func checkIntentFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) *checkly.CheckIntent {
	model := objectAs[CheckIntentModel](ctx, obj, diags)
	if model == nil {
		return nil
	}

	return &checkly.CheckIntent{
		Goal:             model.Goal.ValueString(),
		RequiredOutcomes: stringsFromList(ctx, model.RequiredOutcomes, diags),
		MustPreserve:     stringsFromList(ctx, model.MustPreserve, diags),
	}
}

// checkIntentToObject converts the API representation of an intent to the
// intent attribute value. No intent converts to null, and an empty statement
// list to an empty list.
func checkIntentToObject(ctx context.Context, intent *checkly.CheckIntent, diags *diag.Diagnostics) types.Object {
	if intent == nil {
		return types.ObjectNull(checkIntentAttrTypes)
	}

	obj, d := types.ObjectValueFrom(ctx, checkIntentAttrTypes, CheckIntentModel{
		Goal:             types.StringValue(intent.Goal),
		RequiredOutcomes: listFromStrings(ctx, intent.RequiredOutcomes, diags),
		MustPreserve:     listFromStrings(ctx, intent.MustPreserve, diags),
	})
	diags.Append(d...)

	return obj
}
