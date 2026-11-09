package provider

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/checkly/checkly-go-sdk/v2"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func checkIntentObject(t *testing.T, goal string, requiredOutcomes, mustPreserve types.List) types.Object {
	t.Helper()

	obj, diags := types.ObjectValueFrom(context.Background(), checkIntentAttrTypes, CheckIntentModel{
		Goal:             types.StringValue(goal),
		RequiredOutcomes: requiredOutcomes,
		MustPreserve:     mustPreserve,
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	return obj
}

func stringList(t *testing.T, values ...string) types.List {
	t.Helper()

	var diags diag.Diagnostics
	list := listFromStrings(context.Background(), values, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}
	return list
}

func TestStringListFromAPINullIfUnset(t *testing.T) {
	ctx := context.Background()
	null := types.ListNull(types.StringType)
	empty := stringList(t)
	values := stringList(t, "a", "b")

	cases := map[string]struct {
		values []string
		prior  types.List
		want   types.List
	}{
		"empty with a null prior stays null":       {nil, null, null},
		"empty with an unknown prior becomes null": {[]string{}, types.ListUnknown(types.StringType), null},
		"empty with an empty prior stays empty":    {[]string{}, empty, empty},
		"values replace a null prior":              {[]string{"a", "b"}, null, values},
		"empty replaces a prior with values":       {nil, values, empty},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var diags diag.Diagnostics
			if got := stringListFromAPINullIfUnset(ctx, c.values, c.prior, &diags); !got.Equal(c.want) {
				t.Errorf("stringListFromAPINullIfUnset = %s, want %s", got, c.want)
			}
			if diags.HasError() {
				t.Fatal(diags)
			}
		})
	}
}

func TestCheckIntentToObject(t *testing.T) {
	ctx := context.Background()
	null := types.ListNull(types.StringType)
	empty := stringList(t)
	objectNull := types.ObjectNull(checkIntentAttrTypes)

	cases := map[string]struct {
		intent *checkly.CheckIntent
		prior  types.Object
		want   types.Object
	}{
		"no intent": {
			prior: checkIntentObject(t, "goal", null, null),
			want:  objectNull,
		},
		"a goal with a null prior": {
			intent: &checkly.CheckIntent{Goal: "goal", RequiredOutcomes: []string{}, MustPreserve: []string{}},
			prior:  objectNull,
			want:   checkIntentObject(t, "goal", null, null),
		},
		"empty lists stay as configured": {
			intent: &checkly.CheckIntent{Goal: "goal", RequiredOutcomes: []string{}, MustPreserve: []string{}},
			prior:  checkIntentObject(t, "goal", empty, null),
			want:   checkIntentObject(t, "goal", empty, null),
		},
		"statements keep their order": {
			intent: &checkly.CheckIntent{Goal: "goal", RequiredOutcomes: []string{"b", "a"}, MustPreserve: []string{"c"}},
			prior:  objectNull,
			want:   checkIntentObject(t, "goal", stringList(t, "b", "a"), stringList(t, "c")),
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var diags diag.Diagnostics
			if got := checkIntentToObject(ctx, c.intent, c.prior, &diags); !got.Equal(c.want) {
				t.Errorf("checkIntentToObject = %s, want %s", got, c.want)
			}
			if diags.HasError() {
				t.Fatal(diags)
			}
		})
	}
}

func TestCheckIntentFromObject(t *testing.T) {
	ctx := context.Background()
	null := types.ListNull(types.StringType)

	cases := map[string]struct {
		obj  types.Object
		want *checkly.CheckIntent
	}{
		"null":    {obj: types.ObjectNull(checkIntentAttrTypes)},
		"unknown": {obj: types.ObjectUnknown(checkIntentAttrTypes)},
		"a goal":  {obj: checkIntentObject(t, "goal", null, null), want: &checkly.CheckIntent{Goal: "goal"}},
		"statements": {
			obj:  checkIntentObject(t, "goal", stringList(t, "b", "a"), stringList(t, "c")),
			want: &checkly.CheckIntent{Goal: "goal", RequiredOutcomes: []string{"b", "a"}, MustPreserve: []string{"c"}},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var diags diag.Diagnostics
			got := checkIntentFromObject(ctx, c.obj, &diags)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if err := compareCheckIntent(got, c.want); err != nil {
				t.Error(err)
			}
		})
	}
}

func validateCheckIntentGoal(value types.String) diag.Diagnostics {
	ctx := context.Background()
	req := validator.StringRequest{Path: path.Root("goal"), ConfigValue: value}
	var resp validator.StringResponse

	goal := makeCheckIntentSchema().Attributes["goal"].(schema.StringAttribute)
	for _, v := range goal.Validators {
		v.ValidateString(ctx, req, &resp)
	}
	return resp.Diagnostics
}

func TestCheckIntentGoalValidators(t *testing.T) {
	cases := map[string]struct {
		value types.String
		valid bool
	}{
		"a goal":                      {types.StringValue("The page loads"), true},
		"the longest goal":            {types.StringValue(strings.Repeat("a", 2000)), true},
		"an inner newline":            {types.StringValue("one\ntwo"), true},
		"unknown":                     {types.StringUnknown(), true},
		"the longest multi-byte goal": {types.StringValue(strings.Repeat("ä", 2000)), true},
		"a multi-byte goal too long":  {types.StringValue(strings.Repeat("ä", 2001)), false},
		"the longest emoji goal":      {types.StringValue(strings.Repeat(emoji, 1000)), true},
		"an emoji goal too long":      {types.StringValue(strings.Repeat(emoji, 1000) + "a"), false},
		"a trailing U+0085":           {types.StringValue("The page loads" + string(rune(0x85))), true},
		"a trailing U+2028":           {types.StringValue("The page loads" + string(rune(0x2028))), false},
		"a leading U+FEFF":            {types.StringValue("\uFEFFThe page loads"), false},
		"a trailing U+FEFF":           {types.StringValue("The page loads\uFEFF"), false},
		"an inner U+FEFF":             {types.StringValue("The page\uFEFF loads"), true},
		"empty":                       {types.StringValue(""), false},
		"too long":                    {types.StringValue(strings.Repeat("a", 2001)), false},
		"a heredoc's newline":         {types.StringValue("The page loads\n"), false},
		"a leading space":             {types.StringValue(" The page loads"), false},
		"only whitespace":             {types.StringValue("   "), false},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if diags := validateCheckIntentGoal(c.value); diags.HasError() == c.valid {
				t.Errorf("valid = %v, diagnostics: %v", c.valid, diags)
			}
		})
	}
}

func TestCheckIntentWhitespaceErrorNamesTrimspace(t *testing.T) {
	diags := validateCheckIntentGoal(types.StringValue("The page loads\n"))
	if diags.ErrorsCount() != 1 || !strings.Contains(diags.Errors()[0].Detail(), "trimspace()") {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestCheckIntentStatementValidators(t *testing.T) {
	ctx := context.Background()

	statements := func(n int) []attr.Value {
		values := make([]attr.Value, n)
		for i := range values {
			values[i] = types.StringValue(fmt.Sprintf("statement %d", i))
		}
		return values
	}
	list := func(values ...attr.Value) types.List {
		return types.ListValueMust(types.StringType, values)
	}

	cases := map[string]struct {
		value types.List
		valid bool
	}{
		"null":                   {types.ListNull(types.StringType), true},
		"empty":                  {list(), true},
		"the most statements":    {list(statements(20)...), true},
		"the longest":            {list(types.StringValue(strings.Repeat("a", 1000))), true},
		"the longest multi-byte": {list(types.StringValue(strings.Repeat("ä", 1000))), true},
		"multi-byte too long":    {list(types.StringValue(strings.Repeat("ä", 1001))), false},
		"the longest emoji":      {list(types.StringValue(strings.Repeat(emoji, 500))), true},
		"emoji too long":         {list(types.StringValue(strings.Repeat(emoji, 500) + "a")), false},
		"a trailing U+FEFF":      {list(types.StringValue("statement\uFEFF")), false},
		"too many statements":    {list(statements(21)...), false},
		"too long":               {list(types.StringValue(strings.Repeat("a", 1001))), false},
		"an empty statement":     {list(types.StringValue("")), false},
		"a null statement":       {list(types.StringNull()), false},
		"a trailing newline":     {list(types.StringValue("statement\n")), false},
	}

	for _, attribute := range []string{"required_outcomes", "must_preserve"} {
		statementList := makeCheckIntentSchema().Attributes[attribute].(schema.ListAttribute)

		for name, c := range cases {
			t.Run(attribute+"/"+name, func(t *testing.T) {
				req := validator.ListRequest{Path: path.Root(attribute), ConfigValue: c.value}
				var resp validator.ListResponse
				for _, v := range statementList.Validators {
					v.ValidateList(ctx, req, &resp)
				}
				if resp.Diagnostics.HasError() == c.valid {
					t.Errorf("valid = %v, diagnostics: %v", c.valid, resp.Diagnostics)
				}
			})
		}
	}
}

// A model derived from checkly_check carries its intent and automatic repair
// override, so a create, an update and a move keep them.
func TestCheckModelsFromCheckModelCarryIntent(t *testing.T) {
	intent := types.ObjectValueMust(checkIntentAttrTypes, map[string]attr.Value{
		"goal":              types.StringValue("stay up"),
		"required_outcomes": types.ListValueMust(types.StringType, []attr.Value{types.StringValue("it loads")}),
		"must_preserve":     types.ListNull(types.StringType),
	})

	sources := map[string]*CheckResourceModel{
		"set":   {Intent: intent, AIAutoRepairEnabled: types.BoolValue(false)},
		"unset": {Intent: types.ObjectNull(checkIntentAttrTypes), AIAutoRepairEnabled: types.BoolNull()},
	}

	for sourceName, source := range sources {
		api := apiCheckModelFromCheckModel(source)
		browser := browserCheckModelFromCheckModel(source)
		multistep := multistepCheckModelFromCheckModel(source)

		cases := map[string]struct {
			intent       types.Object
			aiAutoRepair types.Bool
		}{
			"api":        {api.Intent, api.AIAutoRepairEnabled},
			"browser":    {browser.Intent, browser.AIAutoRepairEnabled},
			"multi-step": {multistep.Intent, multistep.AIAutoRepairEnabled},
		}

		for name, c := range cases {
			t.Run(sourceName+"/"+name, func(t *testing.T) {
				if !c.intent.Equal(source.Intent) {
					t.Errorf("intent = %#v, want %#v", c.intent, source.Intent)
				}
				if !c.aiAutoRepair.Equal(source.AIAutoRepairEnabled) {
					t.Errorf("ai_auto_repair_enabled = %s, want %s", c.aiAutoRepair, source.AIAutoRepairEnabled)
				}
			})
		}
	}
}

// An update writes both values on every check type, so each type needs the
// attributes for them to be visible and removable.
func TestCheckResourcesHaveIntentAttributes(t *testing.T) {
	ctx := context.Background()

	cases := map[string]resource.Resource{
		"checkly_check":                  NewCheckResource(),
		"checkly_api_check":              NewAPICheckResource(),
		"checkly_browser_check":          NewBrowserCheckResource(),
		"checkly_multistep_check":        NewMultistepCheckResource(),
		"checkly_playwright_check_suite": NewPlaywrightCheckSuiteResource(),
	}

	for name, r := range cases {
		t.Run(name, func(t *testing.T) {
			var resp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &resp)
			for _, attribute := range []string{checkIntentAttributeName, aiAutoRepairEnabledAttributeName} {
				if _, ok := resp.Schema.Attributes[attribute]; !ok {
					t.Errorf("missing %s", attribute)
				}
			}
		})
	}
}

func compareCheckIntent(got, want *checkly.CheckIntent) error {
	if got == nil || want == nil {
		if got != want {
			return fmt.Errorf("intent = %+v, want %+v", got, want)
		}
		return nil
	}
	if got.Goal != want.Goal ||
		!slices.Equal(got.RequiredOutcomes, want.RequiredOutcomes) ||
		!slices.Equal(got.MustPreserve, want.MustPreserve) {
		return fmt.Errorf("intent = %+v, want %+v", got, want)
	}
	return nil
}

// checkIntentCase runs the intent attributes of one check resource against
// the API.
type checkIntentCase struct {
	resource string
	// config returns the configuration of a check with the given name and
	// further attributes.
	config       func(name, attributes string) string
	importIgnore []string
	set          func(ctx context.Context, client checkly.Client, id string, intent *checkly.CheckIntent, aiAutoRepair *bool) error
	read         func(ctx context.Context, client checkly.Client, id string) (*checkly.CheckIntent, *bool, error)
}

func testAccCheckIntent(t *testing.T, c checkIntentCase) {
	t.Helper()

	const goal = "tf-acc: the page keeps loading"
	const ignoreChanges = `
		lifecycle {
			ignore_changes = [intent, ai_auto_repair_enabled]
		}
	`
	name := fmt.Sprintf("tf-acc-intent-%d", acctest.RandInt())
	repairOn, repairOff := true, false

	var id string
	captureID := func(s *terraform.State) error {
		id = s.RootModule().Resources[c.resource].Primary.ID
		return nil
	}
	stored := func(intent *checkly.CheckIntent, aiAutoRepair *bool) tfresource.TestCheckFunc {
		return func(*terraform.State) error {
			gotIntent, gotAIAutoRepair, err := c.read(context.Background(), testAccClient(), id)
			if err != nil {
				return err
			}
			if err := compareCheckIntent(gotIntent, intent); err != nil {
				return err
			}
			if (gotAIAutoRepair == nil) != (aiAutoRepair == nil) ||
				(aiAutoRepair != nil && *gotAIAutoRepair != *aiAutoRepair) {
				return fmt.Errorf("aiAutoRepairEnabled = %v, want %v", gotAIAutoRepair, aiAutoRepair)
			}
			return nil
		}
	}
	setOutsideTerraform := func() {
		intent := &checkly.CheckIntent{Goal: goal, RequiredOutcomes: []string{"set outside Terraform"}}
		if err := c.set(context.Background(), testAccClient(), id, intent, &repairOff); err != nil {
			t.Fatal(err)
		}
	}

	accTestCase(t, []tfresource.TestStep{
		{
			Config: c.config(name, fmt.Sprintf(`
				intent = {
					goal              = %q
					required_outcomes = ["the page loads"]
					must_preserve     = ["the assertions stay"]
				}
				ai_auto_repair_enabled = false
			`, goal)),
			Check: tfresource.ComposeTestCheckFunc(
				captureID,
				tfresource.TestCheckResourceAttr(c.resource, "intent.goal", goal),
				tfresource.TestCheckResourceAttr(c.resource, "intent.required_outcomes.0", "the page loads"),
				tfresource.TestCheckResourceAttr(c.resource, "intent.must_preserve.0", "the assertions stay"),
				tfresource.TestCheckResourceAttr(c.resource, "ai_auto_repair_enabled", "false"),
				stored(&checkly.CheckIntent{
					Goal:             goal,
					RequiredOutcomes: []string{"the page loads"},
					MustPreserve:     []string{"the assertions stay"},
				}, &repairOff),
			),
		},
		{
			Config: c.config(name, fmt.Sprintf(`
				intent = {
					goal              = %q
					required_outcomes = ["the title shows", "the page loads"]
				}
				ai_auto_repair_enabled = true
			`, goal)),
			Check: tfresource.ComposeTestCheckFunc(
				tfresource.TestCheckResourceAttr(c.resource, "intent.required_outcomes.0", "the title shows"),
				tfresource.TestCheckResourceAttr(c.resource, "intent.required_outcomes.1", "the page loads"),
				tfresource.TestCheckNoResourceAttr(c.resource, "intent.must_preserve.#"),
				tfresource.TestCheckResourceAttr(c.resource, "ai_auto_repair_enabled", "true"),
				stored(&checkly.CheckIntent{
					Goal:             goal,
					RequiredOutcomes: []string{"the title shows", "the page loads"},
				}, &repairOn),
			),
		},
		{
			ResourceName:            c.resource,
			ImportState:             true,
			ImportStateVerify:       true,
			ImportStateVerifyIgnore: c.importIgnore,
		},
		{
			// The API accepts a goal without statements, and empty lists stay
			// empty in the state.
			Config: c.config(name, fmt.Sprintf(`
				intent = {
					goal              = %q
					required_outcomes = []
					must_preserve     = []
				}
			`, goal)),
			Check: tfresource.ComposeTestCheckFunc(
				tfresource.TestCheckResourceAttr(c.resource, "intent.required_outcomes.#", "0"),
				tfresource.TestCheckResourceAttr(c.resource, "intent.must_preserve.#", "0"),
				tfresource.TestCheckNoResourceAttr(c.resource, "ai_auto_repair_enabled"),
				stored(&checkly.CheckIntent{Goal: goal}, nil),
			),
		},
		{
			Config: c.config(name, ""),
			Check:  stored(nil, nil),
		},
		{
			// A value set outside Terraform shows up as a removal.
			PreConfig: setOutsideTerraform,
			Config:    c.config(name, ""),
			ConfigPlanChecks: tfresource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(c.resource, plancheck.ResourceActionUpdate),
					plancheck.ExpectKnownValue(c.resource, tfjsonpath.New("intent"), knownvalue.Null()),
					plancheck.ExpectKnownValue(c.resource, tfjsonpath.New("ai_auto_repair_enabled"), knownvalue.Null()),
				},
			},
			Check: stored(nil, nil),
		},
		{
			// With ignore_changes, updating another attribute keeps it.
			PreConfig: setOutsideTerraform,
			Config:    c.config(name+"-renamed", ignoreChanges),
			ConfigPlanChecks: tfresource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(c.resource, plancheck.ResourceActionUpdate),
				},
			},
			Check: stored(&checkly.CheckIntent{Goal: goal, RequiredOutcomes: []string{"set outside Terraform"}}, &repairOff),
		},
	})
}

func TestAccAPICheckIntent(t *testing.T) {
	testAccCheckIntent(t, checkIntentCase{
		resource: "checkly_api_check.test",
		config: func(name, attributes string) string {
			return fmt.Sprintf(`
				resource "checkly_api_check" "test" {
					name      = %q
					activated = false
					locations = ["eu-central-1"]
					request = {
						url = "https://api.checklyhq.com/public-stats"
					}
					%s
				}
			`, name, attributes)
		},
		set: func(ctx context.Context, client checkly.Client, id string, intent *checkly.CheckIntent, aiAutoRepair *bool) error {
			check, err := client.GetAPICheck(ctx, id)
			if err != nil {
				return err
			}
			check.Intent, check.AIAutoRepairEnabled = intent, aiAutoRepair
			_, err = client.UpdateAPICheck(ctx, id, *check)
			return err
		},
		read: func(ctx context.Context, client checkly.Client, id string) (*checkly.CheckIntent, *bool, error) {
			check, err := client.GetAPICheck(ctx, id)
			if err != nil {
				return nil, nil, err
			}
			return check.Intent, check.AIAutoRepairEnabled, nil
		},
	})
}

func TestAccCheckIntent(t *testing.T) {
	testAccCheckIntent(t, checkIntentCase{
		resource: "checkly_check.test",
		config: func(name, attributes string) string {
			return fmt.Sprintf(`
				resource "checkly_check" "test" {
					name                      = %q
					type                      = "BROWSER"
					activated                 = false
					frequency                 = 10
					locations                 = ["eu-central-1"]
					use_global_alert_settings = true
					script                    = "console.log('test')"
					%s
				}
			`, name, attributes)
		},
		set: func(ctx context.Context, client checkly.Client, id string, intent *checkly.CheckIntent, aiAutoRepair *bool) error {
			check, err := client.GetBrowserCheck(ctx, id)
			if err != nil {
				return err
			}
			check.Intent, check.AIAutoRepairEnabled = intent, aiAutoRepair
			_, err = client.UpdateBrowserCheck(ctx, id, *check)
			return err
		},
		read: func(ctx context.Context, client checkly.Client, id string) (*checkly.CheckIntent, *bool, error) {
			check, err := client.GetBrowserCheck(ctx, id)
			if err != nil {
				return nil, nil, err
			}
			return check.Intent, check.AIAutoRepairEnabled, nil
		},
	})
}

func TestAccBrowserCheckIntent(t *testing.T) {
	testAccCheckIntent(t, checkIntentCase{
		resource: "checkly_browser_check.test",
		config: func(name, attributes string) string {
			return fmt.Sprintf(`
				resource "checkly_browser_check" "test" {
					name      = %q
					activated = false
					locations = ["eu-central-1"]
					script    = "console.log('test')"
					%s
				}
			`, name, attributes)
		},
		importIgnore: []string{"dependencies"},
		set: func(ctx context.Context, client checkly.Client, id string, intent *checkly.CheckIntent, aiAutoRepair *bool) error {
			check, err := client.GetBrowserCheck(ctx, id)
			if err != nil {
				return err
			}
			check.Intent, check.AIAutoRepairEnabled = intent, aiAutoRepair
			_, err = client.UpdateBrowserCheck(ctx, id, *check)
			return err
		},
		read: func(ctx context.Context, client checkly.Client, id string) (*checkly.CheckIntent, *bool, error) {
			check, err := client.GetBrowserCheck(ctx, id)
			if err != nil {
				return nil, nil, err
			}
			return check.Intent, check.AIAutoRepairEnabled, nil
		},
	})
}

func TestAccMultistepCheckIntent(t *testing.T) {
	testAccCheckIntent(t, checkIntentCase{
		resource: "checkly_multistep_check.test",
		config: func(name, attributes string) string {
			return fmt.Sprintf(`
				resource "checkly_multistep_check" "test" {
					name      = %q
					activated = false
					locations = ["eu-central-1"]
					script    = "console.log('test')"
					%s
				}
			`, name, attributes)
		},
		importIgnore: []string{"dependencies"},
		set: func(ctx context.Context, client checkly.Client, id string, intent *checkly.CheckIntent, aiAutoRepair *bool) error {
			check, err := client.GetMultiStepCheck(ctx, id)
			if err != nil {
				return err
			}
			check.Intent, check.AIAutoRepairEnabled = intent, aiAutoRepair
			_, err = client.UpdateMultiStepCheck(ctx, id, *check)
			return err
		},
		read: func(ctx context.Context, client checkly.Client, id string) (*checkly.CheckIntent, *bool, error) {
			check, err := client.GetMultiStepCheck(ctx, id)
			if err != nil {
				return nil, nil, err
			}
			return check.Intent, check.AIAutoRepairEnabled, nil
		},
	})
}

func TestAccPlaywrightCheckSuiteIntent(t *testing.T) {
	testAccCheckIntent(t, checkIntentCase{
		resource: "checkly_playwright_check_suite.test",
		config: func(name, attributes string) string {
			return playwrightCheckSuiteBase + fmt.Sprintf(`
				resource "checkly_playwright_check_suite" "test" {
					name                      = %q
					activated                 = false
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						steps = {
							install = {
								command = "pnpm i"
							}

							test = {
								command = "pnpm playwright test"
							}
						}

						playwright = {
							version = "1.56.1"
							devices = [
								{
									type = "chromium"
								},
							]
						}
					}
					%s
				}
			`, name, attributes)
		},
		importIgnore: []string{"bundle"},
		set: func(ctx context.Context, client checkly.Client, id string, intent *checkly.CheckIntent, aiAutoRepair *bool) error {
			check, err := client.GetPlaywrightCheck(ctx, id)
			if err != nil {
				return err
			}
			check.Intent, check.AIAutoRepairEnabled = intent, aiAutoRepair
			_, err = client.UpdatePlaywrightCheck(ctx, id, *check)
			return err
		},
		read: func(ctx context.Context, client checkly.Client, id string) (*checkly.CheckIntent, *bool, error) {
			check, err := client.GetPlaywrightCheck(ctx, id)
			if err != nil {
				return nil, nil, err
			}
			return check.Intent, check.AIAutoRepairEnabled, nil
		},
	})
}
