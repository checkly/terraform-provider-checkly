package checkly

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccPlaywrightCodeBundleNonExistingFile(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_playwright_code_bundle" "test" {
					prebuilt_archive {
						file = "../fixtures/does-not-exist.tar.gz"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`non-existing file`),
		},
	})
}

func TestAccPlaywrightCodeBundleZipArchive(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_playwright_code_bundle" "test" {
					prebuilt_archive {
						file = "../fixtures/playwright-project.zip"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`appears to be a \.zip archive, but a \.tar\.gz archive is required`),
		},
	})
}

func TestAccPlaywrightCodeBundleInvalidArchiveFormat(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_playwright_code_bundle" "test" {
					prebuilt_archive {
						file = "../fixtures/not-a-gzip-archive.tar"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`is not a valid \.tar\.gz archive`),
		},
	})
}

func TestAccPlaywrightCodeBundleNoLockfile(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_playwright_code_bundle" "test" {
					prebuilt_archive {
						file = "../fixtures/playwright-project-no-lockfile.tar.gz"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`no lockfile found at the root of the archive`),
		},
	})
}

func TestAccPlaywrightCodeBundleNoPlaywrightInLockfile(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_playwright_code_bundle" "test" {
					prebuilt_archive {
						file = "../fixtures/playwright-project-no-playwright.tar.gz"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`the lockfile does not contain @playwright/test`),
		},
	})
}

func TestIsPlaywrightConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want bool
	}{
		{"playwright.config.ts", true},
		{"playwright.config.js", true},
		{"playwright.config.mjs", true},
		{"playwright.config.cjs", true},
		{"playwright.config.mts", true},
		{"playwright.config.cts", true},
		{"playwright-ct.config.ts", true},
		{"packages/e2e/playwright.config.ts", true},
		{"package.json", false},
		{"playwright.ts", false},
		{"playwright.config.yaml", false},
		{"notplaywright.config.ts", false},
		{"playwright.config.", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isPlaywrightConfig(tt.name); got != tt.want {
				t.Errorf("isPlaywrightConfig(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestDetectWorkingDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		file string
		want string
	}{
		{
			name: "flat project at root",
			file: "../fixtures/playwright-project-pnpm.tar.gz",
			want: ".",
		},
		{
			name: "monorepo with nested config",
			file: "../fixtures/playwright-project-monorepo-pnpm.tar.gz",
			want: "packages/e2e",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			attr := PlaywrightCodeBundlePrebuiltArchiveAttribute{
				File: tt.file,
			}

			got, err := attr.DetectWorkingDir()
			if err != nil {
				t.Fatalf("DetectWorkingDir failed: %v", err)
			}
			if got != tt.want {
				t.Errorf("DetectWorkingDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInspectLockfile(t *testing.T) {
	t.Parallel()

	fixtures := []struct {
		name           string
		file           string
		packageManager string
	}{
		{"npm", "../fixtures/playwright-project-npm.tar.gz", "npm"},
		{"pnpm", "../fixtures/playwright-project-pnpm.tar.gz", "pnpm"},
		{"yarn", "../fixtures/playwright-project-yarn.tar.gz", "yarn"},
	}

	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			t.Parallel()

			attr := PlaywrightCodeBundlePrebuiltArchiveAttribute{
				File: fixture.file,
			}

			info, err := attr.InspectLockfile("@playwright/test")
			if err != nil {
				t.Fatalf("InspectLockfile failed: %v", err)
			}
			if info == nil {
				t.Fatal("InspectLockfile returned nil")
			}

			if info.PackageManager != fixture.packageManager {
				t.Errorf("PackageManager = %q, want %q", info.PackageManager, fixture.packageManager)
			}

			if info.PackageVersion != "1.58.2" {
				t.Errorf("PackageVersion = %q, want %q", info.PackageVersion, "1.58.2")
			}

			if info.ChecksumSha256 == "" {
				t.Error("ChecksumSha256 is empty")
			}
		})
	}

	t.Run("lockfile without @playwright/test", func(t *testing.T) {
		t.Parallel()

		attr := PlaywrightCodeBundlePrebuiltArchiveAttribute{
			File: "../fixtures/playwright-project-no-playwright.tar.gz",
		}

		info, err := attr.InspectLockfile("@playwright/test")
		if err != nil {
			t.Fatalf("InspectLockfile returned unexpected error: %v", err)
		}
		if info == nil {
			t.Fatal("InspectLockfile returned nil, expected LockfileInfo with empty PackageVersion")
		}
		if info.PackageManager != "npm" {
			t.Errorf("PackageManager = %q, want %q", info.PackageManager, "npm")
		}
		if info.PackageVersion != "" {
			t.Errorf("PackageVersion = %q, want empty string", info.PackageVersion)
		}
	})

	t.Run("no lockfile in archive", func(t *testing.T) {
		t.Parallel()

		attr := PlaywrightCodeBundlePrebuiltArchiveAttribute{
			File: "../fixtures/playwright-project-no-lockfile.tar.gz",
		}

		info, err := attr.InspectLockfile("@playwright/test")
		if err != nil {
			t.Fatalf("InspectLockfile returned unexpected error: %v", err)
		}
		if info != nil {
			t.Fatalf("InspectLockfile returned %+v, want nil", info)
		}
	})

	// Verify all three fixtures produce the same detected values
	// (except checksum, which differs per lockfile format).
	t.Run("consistent values across fixtures", func(t *testing.T) {
		var versions []string
		for _, fixture := range fixtures {
			attr := PlaywrightCodeBundlePrebuiltArchiveAttribute{
				File: fixture.file,
			}

			info, err := attr.InspectLockfile("@playwright/test")
			if err != nil {
				t.Fatalf("InspectLockfile(%s) failed: %v", fixture.name, err)
			}

			versions = append(versions, info.PackageVersion)
		}

		for i := 1; i < len(versions); i++ {
			if versions[i] != versions[0] {
				t.Errorf(
					"PackageVersion mismatch: %s=%q vs %s=%q",
					fixtures[0].name, versions[0],
					fixtures[i].name, versions[i],
				)
			}
		}
	})

	t.Run("different archive produces different checksum", func(t *testing.T) {
		t.Parallel()

		pnpm := PlaywrightCodeBundlePrebuiltArchiveAttribute{
			File: "../fixtures/playwright-project-pnpm.tar.gz",
		}
		pnpmNext := PlaywrightCodeBundlePrebuiltArchiveAttribute{
			File: "../fixtures/playwright-project-pnpm-playwright-next.tar.gz",
		}

		infoPnpm, err := pnpm.InspectLockfile("@playwright/test")
		if err != nil {
			t.Fatalf("InspectLockfile(pnpm) failed: %v", err)
		}

		infoPnpmNext, err := pnpmNext.InspectLockfile("@playwright/test")
		if err != nil {
			t.Fatalf("InspectLockfile(pnpm-next) failed: %v", err)
		}

		if infoPnpm.ChecksumSha256 == infoPnpmNext.ChecksumSha256 {
			t.Errorf("expected different checksums, both are %q", infoPnpm.ChecksumSha256)
		}
	})
}
