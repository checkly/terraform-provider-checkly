package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestCreatesAlertChannel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		configs []string
		want    bool
	}{
		{
			name:    "alert channel",
			configs: []string{`resource "checkly_alert_channel" "email" {}`},
			want:    true,
		},
		{
			name: "alert channel in a later step",
			configs: []string{
				`resource "checkly_api_check" "test" {}`,
				`
resource "checkly_api_check" "test" {}

  resource  "checkly_alert_channel"  "email" {}
`,
			},
			want: true,
		},
		{
			name:    "subscription only",
			configs: []string{`resource "checkly_alert_channel_subscription" "test" {}`},
			want:    false,
		},
		{
			name:    "reference to a channel",
			configs: []string{`alert_channel_id = checkly_alert_channel.email.id`},
			want:    false,
		},
		{
			name:    "no config",
			configs: []string{""},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var steps []resource.TestStep
			for _, config := range tt.configs {
				steps = append(steps, resource.TestStep{Config: config})
			}
			if got := createsAlertChannel(steps); got != tt.want {
				t.Errorf("createsAlertChannel() = %v, want %v", got, tt.want)
			}
		})
	}
}
