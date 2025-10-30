package checkly

import (
	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const alertSettingsAttributeName = "alert_settings"

var alertSettingsAttributeSchema = &schema.Schema{
	Description: "Determines the alert escalation policy for the monitor.",
	Type:        schema.TypeList,
	Optional:    true,
	Computed:    true,
	MaxItems:    1,
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			"escalation_type": {
				Description:  "Determines what type of escalation to use. Possible values are `RUN_BASED` and `TIME_BASED`.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateOneOf([]string{"RUN_BASED", "TIME_BASED"}),
			},
			"run_based_escalation": {
				Description: "Configuration for run-based escalation.",
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"failed_run_threshold": {
							Description:  "After how many failed consecutive check runs an alert notification should be sent. Possible values are between `1` and `5`. (Default `1`).",
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      1,
							ValidateFunc: validateBetween(1, 5),
						},
					},
				},
			},
			"time_based_escalation": {
				Description: "Configuration for time-based escalation.",
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"minutes_failing_threshold": {
							Description:  "After how many minutes after a monitor starts failing an alert should be sent. Possible values are `5`, `10`, `15`, and `30`. (Default `5`).",
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      5,
							ValidateFunc: validateOneOf([]int{5, 10, 15, 30}),
						},
					},
				},
			},
			"reminders": {
				Description: "Defines how often to send reminder notifications after initial alert.",
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"amount": {
							Description:  "Number of reminder notifications to send. Possible values are `0`, `1`, `2`, `3`, `4`, `5`, and `100000` (`0` to disable, `100000` for unlimited). (Default `0`).",
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      0,
							ValidateFunc: validateOneOf([]int{0, 1, 2, 3, 4, 5, 100000}),
						},
						"interval": {
							Description:  "Interval between reminder notifications in minutes. Possible values are `5`, `10`, `15`, and `30`. (Default `5`).",
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      5,
							ValidateFunc: validateOneOf([]int{5, 10, 15, 30}),
						},
					},
				},
			},
			"parallel_run_failure_threshold": {
				Description: "Configuration for parallel run failure threshold.",
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Description: "Whether parallel run failure threshold is enabled. Applicable only for monitors scheduled in parallel in multiple locations. (Default `false`).",
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
						},
						"percentage": {
							Description:  "Percentage of runs that must fail to trigger alert. Possible values are `10`, `20`, `30`, `40`, `50`, `60`, `70`, `80`, `90`, and `100`. (Default `10`).",
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      10,
							ValidateFunc: validateOneOf([]int{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}),
						},
					},
				},
			},
		},
	},
}

func alertSettingsFromSet(s []interface{}) checkly.AlertSettings {
	if len(s) == 0 {
		return checkly.AlertSettings{
			EscalationType: checkly.RunBased,
			RunBasedEscalation: checkly.RunBasedEscalation{
				FailedRunThreshold: 1,
			},
		}
	}
	res := s[0].(tfMap)
	alertSettings := checkly.AlertSettings{
		EscalationType:              res["escalation_type"].(string),
		Reminders:                   remindersFromSet(res["reminders"].([]interface{})),
		ParallelRunFailureThreshold: parallelRunFailureThresholdFromSet(res["parallel_run_failure_threshold"].([]interface{})),
	}

	if alertSettings.EscalationType == checkly.RunBased {
		alertSettings.RunBasedEscalation = runBasedEscalationFromSet(res["run_based_escalation"].([]interface{}))
	} else {
		alertSettings.TimeBasedEscalation = timeBasedEscalationFromSet(res["time_based_escalation"].([]interface{}))
	}

	return alertSettings
}

func setFromAlertSettings(as checkly.AlertSettings) []tfMap {
	if as.EscalationType == checkly.RunBased {
		return []tfMap{
			{
				"escalation_type": as.EscalationType,
				"run_based_escalation": []tfMap{
					{
						"failed_run_threshold": as.RunBasedEscalation.FailedRunThreshold,
					},
				},
				"reminders": []tfMap{
					{
						"amount":   as.Reminders.Amount,
						"interval": as.Reminders.Interval,
					},
				},
				"parallel_run_failure_threshold": []tfMap{
					{
						"enabled":    as.ParallelRunFailureThreshold.Enabled,
						"percentage": as.ParallelRunFailureThreshold.Percentage,
					},
				},
			},
		}
	} else {
		return []tfMap{
			{
				"escalation_type": as.EscalationType,
				"time_based_escalation": []tfMap{
					{
						"minutes_failing_threshold": as.TimeBasedEscalation.MinutesFailingThreshold,
					},
				},
				"reminders": []tfMap{
					{
						"amount":   as.Reminders.Amount,
						"interval": as.Reminders.Interval,
					},
				},
				"parallel_run_failure_threshold": []tfMap{
					{
						"enabled":    as.ParallelRunFailureThreshold.Enabled,
						"percentage": as.ParallelRunFailureThreshold.Percentage,
					},
				},
			},
		}
	}
}
