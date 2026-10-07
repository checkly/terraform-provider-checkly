package checkly

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	checkly "github.com/checkly/checkly-go-sdk"
)

func resourceMaintenanceWindow() *schema.Resource {
	return &schema.Resource{
		Create: resourceMaintenanceWindowCreate,
		Read:   resourceMaintenanceWindowRead,
		Update: resourceMaintenanceWindowUpdate,
		Delete: resourceMaintenanceWindowDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The maintenance window name.",
			},
			"starts_at": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The start date of the maintenance window.",
			},
			"ends_at": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The end date of the maintenance window.",
			},
			"repeat_unit": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  nil,
				ValidateFunc: func(value interface{}, key string) (warns []string, errs []error) {
					v := value.(string)
					isValid := false
					options := []string{"DAY", "WEEK", "MONTH"}
					for _, option := range options {
						if v == option {
							isValid = true
						}
					}
					if !isValid {
						errs = append(errs, fmt.Errorf("%q must be one of %v, got %s", key, options, v))
					}
					return warns, errs
				},
				Description: "The repeat cadence for the maintenance window. Possible values `DAY`, `WEEK` and `MONTH`.",
			},
			"repeat_interval": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     nil,
				Description: "The repeat interval of the maintenance window from the first occurrence.",
			},
			"repeat_ends_at": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     nil,
				Description: "The date on which the maintenance window should stop repeating, interpreted as a calendar date in `timezone`.",
			},
			"tags": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				DefaultFunc: func() (interface{}, error) {
					return []tfMap{}, nil
				},
				Description: "The tags of the checks and groups that are paused during the maintenance window. Ignored when `pause_all_checks` is `true`.",
			},
			"timezone": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateFunc:     validateMaintenanceWindowTimezone,
				DiffSuppressFunc: suppressEquivalentTimezoneDiff,
				Description: "The named IANA time zone used to schedule recurring occurrences, e.g. `America/New_York`. " +
					"Occurrences keep the same local time across daylight-saving changes. " +
					"`starts_at` and `ends_at` remain absolute instants; the time zone does not reinterpret them. " +
					"Use the canonical IANA name (e.g. `America/New_York`, not `US/Eastern`): the API stores canonical names, " +
					"so an alias shows as a change on every plan. UTC offsets such as `+05:00` or `Etc/GMT+5` are not accepted. " +
					"Defaults to `UTC`. The time zone cannot be changed while a maintenance is active.",
			},
			"pause_all_checks": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Pause every check in the account during the maintenance window, regardless of `tags`. Defaults to `false`.",
			},
			"silence_alerts_tags": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "The tags of the checks and groups whose alerts are silenced during the maintenance window. Ignored when `silence_all_alerts` is `true`.",
			},
			"silence_all_alerts": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Silence alerts for every check in the account during the maintenance window, regardless of `silence_alerts_tags`. Defaults to `false`.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A description of the maintenance window. When the window is visible on status pages, the description is shown there too.",
			},
			"status_page_visibility": maintenanceWindowStatusPageVisibilitySchema(),
		},
	}
}

func maintenanceWindowsFromResourceData(d *schema.ResourceData) (checkly.MaintenanceWindow, error) {
	ID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		if d.Id() != "" {
			return checkly.MaintenanceWindow{}, err
		}
		ID = 0
	}
	a := checkly.MaintenanceWindow{
		ID:             ID,
		Name:           d.Get("name").(string),
		StartsAt:       d.Get("starts_at").(string),
		EndsAt:         d.Get("ends_at").(string),
		RepeatUnit:     d.Get("repeat_unit").(string),
		RepeatEndsAt:   d.Get("repeat_ends_at").(string),
		RepeatInterval: d.Get("repeat_interval").(int),
		Tags:           stringsFromSet(d.Get("tags").(*schema.Set)),
		// The SDK sends every field, and unset values as the API's defaults,
		// so removing an attribute from the configuration resets it.
		Timezone:             d.Get("timezone").(string),
		PauseAllChecks:       d.Get("pause_all_checks").(bool),
		SilenceAlertsTags:    stringsFromSet(d.Get("silence_alerts_tags").(*schema.Set)),
		SilenceAllAlerts:     d.Get("silence_all_alerts").(bool),
		Description:          d.Get("description").(string),
		StatusPageVisibility: maintenanceWindowStatusPageVisibilityFromList(d.Get("status_page_visibility").([]any)),
	}

	return a, nil
}

func resourceDataFromMaintenanceWindows(s *checkly.MaintenanceWindow, d *schema.ResourceData) error {
	d.Set("name", s.Name)
	d.Set("starts_at", s.StartsAt)
	d.Set("ends_at", s.EndsAt)
	d.Set("repeat_unit", s.RepeatUnit)
	d.Set("repeat_ends_at", s.RepeatEndsAt)
	d.Set("repeat_interval", s.RepeatInterval)
	d.Set("tags", s.Tags)
	// The API returns null for a window created without a time zone and
	// "UTC" for one reset to UTC; both are stored as "" so the state doesn't
	// depend on the window's history.
	timezone := s.Timezone
	if timezonesEquivalent(timezone, "") {
		timezone = ""
	}
	d.Set("timezone", timezone)
	d.Set("pause_all_checks", s.PauseAllChecks)
	d.Set("silence_alerts_tags", s.SilenceAlertsTags)
	d.Set("silence_all_alerts", s.SilenceAllAlerts)
	d.Set("description", s.Description)
	d.Set("status_page_visibility", maintenanceWindowStatusPageVisibilityToList(
		s.StatusPageVisibility,
		len(d.Get("status_page_visibility").([]any)) == 0,
	))
	return nil
}

// fixedOffsetTimezoneRegex matches UTC offsets, which the API rejects because
// they are not named IANA zones: "+05:00" style identifiers and the
// fixed-offset Etc/GMT±N zones. Etc/GMT+0 and Etc/GMT-0 are aliases of UTC and
// are accepted.
var fixedOffsetTimezoneRegex = regexp.MustCompile(`(?i)^[+-]|^Etc/GMT[+-]0*[1-9]\d?$`)

func validateMaintenanceWindowTimezone(value interface{}, key string) (warns []string, errs []error) {
	v := value.(string)
	if fixedOffsetTimezoneRegex.MatchString(v) {
		errs = append(errs, fmt.Errorf("%q must be a named IANA time zone such as \"America/New_York\", got %q; UTC offsets are not supported", key, v))
	}
	return warns, errs
}

// utcTimezoneAliases are the names the API resolves to UTC. The API may report
// a UTC window as "UTC" or as unset, so all of them count as the same value.
var utcTimezoneAliases = map[string]bool{
	"":              true,
	"utc":           true,
	"etc/utc":       true,
	"etc/uct":       true,
	"uct":           true,
	"gmt":           true,
	"etc/gmt":       true,
	"gmt0":          true,
	"etc/gmt0":      true,
	"zulu":          true,
	"etc/zulu":      true,
	"universal":     true,
	"etc/universal": true,
	"greenwich":     true,
	"etc/greenwich": true,
	"etc/gmt+0":     true,
	"etc/gmt-0":     true,
	"gmt+0":         true,
	"gmt-0":         true,
}

// timezonesEquivalent reports whether two timezone values schedule the same
// way. The API canonicalizes the casing of stored names, so names are compared
// case-insensitively.
func timezonesEquivalent(a, b string) bool {
	a, b = strings.ToLower(a), strings.ToLower(b)
	if utcTimezoneAliases[a] && utcTimezoneAliases[b] {
		return true
	}
	return a == b
}

func suppressEquivalentTimezoneDiff(_, old, new string, _ *schema.ResourceData) bool {
	return timezonesEquivalent(old, new)
}

func resourceMaintenanceWindowCreate(d *schema.ResourceData, client interface{}) error {
	mw, err := maintenanceWindowsFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceMaintenanceWindowCreate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	result, err := client.(checkly.Client).CreateMaintenanceWindow(ctx, mw)

	if err != nil {
		return fmt.Errorf("CreateMaintenanceWindows: API error: %w", err)
	}

	d.SetId(fmt.Sprintf("%d", result.ID))
	return resourceMaintenanceWindowRead(d, client)
}

func resourceMaintenanceWindowUpdate(d *schema.ResourceData, client interface{}) error {
	mw, err := maintenanceWindowsFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceMaintenanceWindowUpdate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err = client.(checkly.Client).UpdateMaintenanceWindow(ctx, mw.ID, mw)
	if err != nil {
		return fmt.Errorf("resourceMaintenanceWindowUpdate: API error: %w", err)
	}
	d.SetId(fmt.Sprintf("%d", mw.ID))
	return resourceMaintenanceWindowRead(d, client)
}

func resourceMaintenanceWindowDelete(d *schema.ResourceData, client interface{}) error {
	ID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return fmt.Errorf("resourceMaintenanceWindowDelete: ID %s is not numeric: %w", d.Id(), err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	err = client.(checkly.Client).DeleteMaintenanceWindow(ctx, ID)
	if err != nil {
		return fmt.Errorf("resourceMaintenanceWindowDelete: API error: %w", err)
	}
	return nil
}

func resourceMaintenanceWindowRead(d *schema.ResourceData, client interface{}) error {
	ID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return fmt.Errorf("resourceMaintenanceWindowRead: ID %s is not numeric: %w", d.Id(), err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	mw, err := client.(checkly.Client).GetMaintenanceWindow(ctx, ID)
	defer cancel()
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("resourceMaintenanceWindowRead: API error: %w", err)
	}
	return resourceDataFromMaintenanceWindows(mw, d)
}
