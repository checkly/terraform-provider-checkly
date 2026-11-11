package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// deprecateAliasSchema marks a schema copied from target as belonging to the
// deprecated alias resource type, replacing its description and setting a
// deprecation message that points users to target. reason completes the
// sentence "The `<alias>` resource has been renamed to `<target>`" and
// therefore starts with its own separator, e.g. " to ..." or ", because ...".
func deprecateAliasSchema(resp *resource.SchemaResponse, alias, target, reason string) {
	resp.Schema.Description = "" +
		"The `" + alias + "` resource has been renamed to `" + target + "`" + reason + ". " +
		"The `" + alias + "` resource type is deprecated and will be removed in a future version of the provider." +
		"\n\n" +
		"The schemas of both resource types are identical, and existing `" + alias + "` resources can be " +
		"migrated to `" + target + "` using a `moved` block." +
		"\n\n" +
		"We recommend using the `" + target + "` resource type for any new resources."

	resp.Schema.DeprecationMessage = "" +
		"Use the `" + target + "` resource instead. The schemas are identical, and existing resources can be " +
		"migrated using a `moved` block."
}
