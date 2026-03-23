package checkly

import (
	"testing"
)

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
}
