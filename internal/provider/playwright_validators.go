package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var (
	_ validator.String = (*playwrightFileExistsValidator)(nil)
	_ validator.String = (*playwrightGzipArchiveValidator)(nil)
	_ validator.String = (*playwrightVersionFormatValidator)(nil)
	_ validator.String = (*playwrightWorkingDirValidator)(nil)
)

// playwrightFileExistsValidator validates that the value refers to an
// existing file on the local filesystem.
type playwrightFileExistsValidator struct{}

func (v playwrightFileExistsValidator) Description(ctx context.Context) string {
	return "value must refer to an existing file"
}

func (v playwrightFileExistsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v playwrightFileExistsValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	_, err := os.Stat(value)
	if os.IsNotExist(err) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Attribute Value",
			fmt.Sprintf("Attribute %s refers to a non-existing file %q: %s", req.Path, value, err),
		)
	}
}

// playwrightGzipArchiveValidator checks that the file at the given path is a
// gzip archive by inspecting the first two bytes (magic number 0x1f 0x8b). If
// the file appears to be a zip archive instead, the error message says so.
type playwrightGzipArchiveValidator struct{}

func (v playwrightGzipArchiveValidator) Description(ctx context.Context) string {
	return "value must refer to a .tar.gz archive"
}

func (v playwrightGzipArchiveValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v playwrightGzipArchiveValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	f, err := os.Open(value)
	if err != nil {
		// Let playwrightFileExistsValidator handle this.
		return
	}
	defer f.Close()

	var magic [2]byte
	if _, err := f.Read(magic[:]); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Attribute Value",
			fmt.Sprintf("Attribute %s could not be read: %s", req.Path, err),
		)
		return
	}

	if magic[0] == 0x50 && magic[1] == 0x4b {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Attribute Value",
			fmt.Sprintf("Attribute %s appears to be a .zip archive, but a .tar.gz archive is required", req.Path),
		)
		return
	}

	if magic[0] != 0x1f || magic[1] != 0x8b {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Attribute Value",
			fmt.Sprintf("Attribute %s is not a valid .tar.gz archive", req.Path),
		)
	}
}

var playwrightVersionFormatRegex = regexp.MustCompile(`^\d+(\.\d+){0,2}$`)

// playwrightVersionFormatValidator validates that the value looks like a
// version number consisting of up to three dot-separated numeric components.
type playwrightVersionFormatValidator struct{}

func (v playwrightVersionFormatValidator) Description(ctx context.Context) string {
	return `value must be a version number (e.g. "22", "24", "1.3")`
}

func (v playwrightVersionFormatValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v playwrightVersionFormatValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	if !playwrightVersionFormatRegex.MatchString(value) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Attribute Value",
			fmt.Sprintf("Attribute %s must be a version number (e.g. \"22\", \"24\", \"1.3\"), got: %s", req.Path, value),
		)
	}
}

// playwrightWorkingDirValidator validates that a working directory value is
// not the empty string. "." must be used to refer to the root directory.
type playwrightWorkingDirValidator struct{}

func (v playwrightWorkingDirValidator) Description(ctx context.Context) string {
	return `value must not be empty; use "." for the root directory`
}

func (v playwrightWorkingDirValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v playwrightWorkingDirValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	if req.ConfigValue.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Attribute Value",
			fmt.Sprintf("Attribute %s must not be empty; use \".\" for the root directory", req.Path),
		)
	}
}
