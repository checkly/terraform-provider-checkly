package checkly

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
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
		{"bun", "../fixtures/playwright-project-bun.tar.gz", "bun"},
	}

	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			t.Parallel()

			attr := PlaywrightCodeBundlePrebuiltArchiveAttribute{
				File: fixture.file,
			}

			info, err := attr.InspectLockfile("@playwright/test", InspectLockfileOptions{})
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

		info, err := attr.InspectLockfile("@playwright/test", InspectLockfileOptions{})
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

	t.Run("bun.lockb only returns unsupported error", func(t *testing.T) {
		t.Parallel()

		attr := PlaywrightCodeBundlePrebuiltArchiveAttribute{
			File: "../fixtures/playwright-project-bun-lockb.tar.gz",
		}

		info, err := attr.InspectLockfile("@playwright/test", InspectLockfileOptions{})
		if !errors.Is(err, ErrUnsupportedBunLockb) {
			t.Fatalf("InspectLockfile error = %v, want ErrUnsupportedBunLockb", err)
		}
		if info != nil {
			t.Errorf("InspectLockfile returned %+v, want nil", info)
		}
	})

	t.Run("bun.lock takes precedence over bun.lockb", func(t *testing.T) {
		t.Parallel()

		attr := PlaywrightCodeBundlePrebuiltArchiveAttribute{
			File: "../fixtures/playwright-project-bun-with-lockb.tar.gz",
		}

		info, err := attr.InspectLockfile("@playwright/test", InspectLockfileOptions{})
		if err != nil {
			t.Fatalf("InspectLockfile failed: %v", err)
		}
		if info == nil {
			t.Fatal("InspectLockfile returned nil")
		}
		if info.PackageManager != "bun" {
			t.Errorf("PackageManager = %q, want %q", info.PackageManager, "bun")
		}
		if info.PackageVersion != "1.58.2" {
			t.Errorf("PackageVersion = %q, want %q", info.PackageVersion, "1.58.2")
		}
	})

	t.Run("no lockfile in archive", func(t *testing.T) {
		t.Parallel()

		attr := PlaywrightCodeBundlePrebuiltArchiveAttribute{
			File: "../fixtures/playwright-project-no-lockfile.tar.gz",
		}

		info, err := attr.InspectLockfile("@playwright/test", InspectLockfileOptions{})
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

			info, err := attr.InspectLockfile("@playwright/test", InspectLockfileOptions{})
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

		infoPnpm, err := pnpm.InspectLockfile("@playwright/test", InspectLockfileOptions{})
		if err != nil {
			t.Fatalf("InspectLockfile(pnpm) failed: %v", err)
		}

		infoPnpmNext, err := pnpmNext.InspectLockfile("@playwright/test", InspectLockfileOptions{})
		if err != nil {
			t.Fatalf("InspectLockfile(pnpm-next) failed: %v", err)
		}

		if infoPnpm.ChecksumSha256 == infoPnpmNext.ChecksumSha256 {
			t.Errorf("expected different checksums, both are %q", infoPnpm.ChecksumSha256)
		}
	})
}

type tarEntry struct {
	name    string
	content []byte
}

func buildTarGz(t *testing.T, entries []tarEntry) string {
	t.Helper()

	p := filepath.Join(t.TempDir(), "archive.tar.gz")
	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)

	for _, e := range entries {
		hdr := &tar.Header{
			Name:     e.name,
			Mode:     0644,
			Size:     int64(len(e.content)),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write header %q: %v", e.name, err)
		}
		if _, err := tw.Write(e.content); err != nil {
			t.Fatalf("write body %q: %v", e.name, err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close file: %v", err)
	}
	return p
}

const syntheticPackageLock = `{
  "name": "example",
  "version": "1.0.0",
  "lockfileVersion": 3,
  "packages": {
    "": {"name": "example", "version": "1.0.0", "dependencies": {"@playwright/test": "1.58.2"}},
    "node_modules/@playwright/test": {"version": "1.58.2"}
  }
}`

func inspectWithExcludedVersion(t *testing.T, file string) *LockfileInfo {
	t.Helper()
	attr := PlaywrightCodeBundlePrebuiltArchiveAttribute{File: file}
	info, err := attr.InspectLockfile("@playwright/test", InspectLockfileOptions{
		PackageJSONExcludedFields: []string{"version"},
	})
	if err != nil {
		t.Fatalf("InspectLockfile failed: %v", err)
	}
	if info == nil {
		t.Fatalf("InspectLockfile returned nil")
	}
	return info
}

func TestInspectLockfileChecksumIncludesPackageJSON(t *testing.T) {
	t.Parallel()

	t.Run("excluded top-level field ignored", func(t *testing.T) {
		t.Parallel()

		a := buildTarGz(t, []tarEntry{
			{"package-lock.json", []byte(syntheticPackageLock)},
			{"package.json", []byte(`{"name":"example","version":"1.0.0"}`)},
		})
		b := buildTarGz(t, []tarEntry{
			{"package-lock.json", []byte(syntheticPackageLock)},
			{"package.json", []byte(`{"name":"example","version":"2.0.0"}`)},
		})

		if inspectWithExcludedVersion(t, a).ChecksumSha256 != inspectWithExcludedVersion(t, b).ChecksumSha256 {
			t.Error("checksum should be stable when only an excluded field changes")
		}
	})

	t.Run("non-excluded field affects checksum", func(t *testing.T) {
		t.Parallel()

		a := buildTarGz(t, []tarEntry{
			{"package-lock.json", []byte(syntheticPackageLock)},
			{"package.json", []byte(`{"name":"example","version":"1.0.0"}`)},
		})
		b := buildTarGz(t, []tarEntry{
			{"package-lock.json", []byte(syntheticPackageLock)},
			{"package.json", []byte(`{"name":"renamed","version":"1.0.0"}`)},
		})

		if inspectWithExcludedVersion(t, a).ChecksumSha256 == inspectWithExcludedVersion(t, b).ChecksumSha256 {
			t.Error("checksum should change when a non-excluded field changes")
		}
	})

	t.Run("package.json inside node_modules is ignored", func(t *testing.T) {
		t.Parallel()

		a := buildTarGz(t, []tarEntry{
			{"package-lock.json", []byte(syntheticPackageLock)},
			{"package.json", []byte(`{"name":"example","version":"1.0.0"}`)},
			{"node_modules/@playwright/test/package.json", []byte(`{"name":"@playwright/test","main":"index.js"}`)},
		})
		b := buildTarGz(t, []tarEntry{
			{"package-lock.json", []byte(syntheticPackageLock)},
			{"package.json", []byte(`{"name":"example","version":"1.0.0"}`)},
			{"node_modules/@playwright/test/package.json", []byte(`{"name":"different","main":"other.js"}`)},
		})

		if inspectWithExcludedVersion(t, a).ChecksumSha256 != inspectWithExcludedVersion(t, b).ChecksumSha256 {
			t.Error("package.json inside node_modules should not contribute to checksum")
		}
	})

	t.Run("nested package.json outside node_modules is included", func(t *testing.T) {
		t.Parallel()

		a := buildTarGz(t, []tarEntry{
			{"package-lock.json", []byte(syntheticPackageLock)},
			{"package.json", []byte(`{"name":"root"}`)},
		})
		b := buildTarGz(t, []tarEntry{
			{"package-lock.json", []byte(syntheticPackageLock)},
			{"package.json", []byte(`{"name":"root"}`)},
			{"packages/e2e/package.json", []byte(`{"name":"e2e"}`)},
		})

		if inspectWithExcludedVersion(t, a).ChecksumSha256 == inspectWithExcludedVersion(t, b).ChecksumSha256 {
			t.Error("adding a nested package.json should change the checksum")
		}
	})

	t.Run("whitespace and key order are canonicalized", func(t *testing.T) {
		t.Parallel()

		a := buildTarGz(t, []tarEntry{
			{"package-lock.json", []byte(syntheticPackageLock)},
			{"package.json", []byte(`{"name":"example","scripts":{"test":"playwright test"}}`)},
		})
		b := buildTarGz(t, []tarEntry{
			{"package-lock.json", []byte(syntheticPackageLock)},
			{"package.json", []byte("{\n  \"scripts\": { \"test\": \"playwright test\" },\n  \"name\": \"example\"\n}\n")},
		})

		if inspectWithExcludedVersion(t, a).ChecksumSha256 != inspectWithExcludedVersion(t, b).ChecksumSha256 {
			t.Error("checksum should match regardless of whitespace or key order")
		}
	})

	t.Run("tar entry order does not affect checksum", func(t *testing.T) {
		t.Parallel()

		rootPkg := []byte(`{"name":"root"}`)
		nestedPkg := []byte(`{"name":"nested"}`)

		a := buildTarGz(t, []tarEntry{
			{"package-lock.json", []byte(syntheticPackageLock)},
			{"package.json", rootPkg},
			{"packages/e2e/package.json", nestedPkg},
		})
		b := buildTarGz(t, []tarEntry{
			{"packages/e2e/package.json", nestedPkg},
			{"package.json", rootPkg},
			{"package-lock.json", []byte(syntheticPackageLock)},
		})

		if inspectWithExcludedVersion(t, a).ChecksumSha256 != inspectWithExcludedVersion(t, b).ChecksumSha256 {
			t.Error("checksum should be stable regardless of tar entry order")
		}
	})
}

func TestInspectLockfileDetectsEngine(t *testing.T) {
	t.Parallel()

	t.Run("node-version file selects node", func(t *testing.T) {
		t.Parallel()
		archive := buildTarGz(t, []tarEntry{
			{name: "package-lock.json", content: []byte(syntheticPackageLock)},
			{name: "package.json", content: []byte(`{"name":"test","dependencies":{"@playwright/test":"1.58.2"}}`)},
			{name: ".node-version", content: []byte("24.1.0")},
		})
		attr := PlaywrightCodeBundlePrebuiltArchiveAttribute{File: archive}
		info, err := attr.InspectLockfile("@playwright/test", InspectLockfileOptions{})
		if err != nil {
			t.Fatalf("InspectLockfile failed: %v", err)
		}
		if info.Engine != "node" {
			t.Errorf("Engine = %q, want %q", info.Engine, "node")
		}
		if info.EngineVersion != "24" {
			t.Errorf("EngineVersion = %q, want %q", info.EngineVersion, "24")
		}
	})

	t.Run("bun-version file selects bun", func(t *testing.T) {
		t.Parallel()
		archive := buildTarGz(t, []tarEntry{
			{name: "bun.lock", content: []byte(`{"packages":{"":{},"@playwright/test@1.58.2":["@playwright/test@1.58.2","",{},""]}}`)},
			{name: "package.json", content: []byte(`{"name":"test","dependencies":{"@playwright/test":"1.58.2"}}`)},
			{name: ".bun-version", content: []byte("1.3.11")},
		})
		attr := PlaywrightCodeBundlePrebuiltArchiveAttribute{File: archive}
		info, err := attr.InspectLockfile("@playwright/test", InspectLockfileOptions{})
		if err != nil {
			t.Fatalf("InspectLockfile failed: %v", err)
		}
		if info.Engine != "bun" {
			t.Errorf("Engine = %q, want %q", info.Engine, "bun")
		}
		if info.EngineVersion != "1.3" {
			t.Errorf("EngineVersion = %q, want %q", info.EngineVersion, "1.3")
		}
	})

	t.Run("no version file falls back to package manager", func(t *testing.T) {
		t.Parallel()
		archive := buildTarGz(t, []tarEntry{
			{name: "package-lock.json", content: []byte(syntheticPackageLock)},
			{name: "package.json", content: []byte(`{"name":"test","dependencies":{"@playwright/test":"1.58.2"}}`)},
		})
		attr := PlaywrightCodeBundlePrebuiltArchiveAttribute{File: archive}
		info, err := attr.InspectLockfile("@playwright/test", InspectLockfileOptions{})
		if err != nil {
			t.Fatalf("InspectLockfile failed: %v", err)
		}
		if info.Engine != "node" {
			t.Errorf("Engine = %q, want %q", info.Engine, "node")
		}
		if info.EngineVersion != "22" {
			t.Errorf("EngineVersion = %q, want %q", info.EngineVersion, "22")
		}
	})

	t.Run("tool-versions with nodejs", func(t *testing.T) {
		t.Parallel()
		archive := buildTarGz(t, []tarEntry{
			{name: "package-lock.json", content: []byte(syntheticPackageLock)},
			{name: "package.json", content: []byte(`{"name":"test","dependencies":{"@playwright/test":"1.58.2"}}`)},
			{name: ".tool-versions", content: []byte("nodejs 24.1.0\npython 3.12.0")},
		})
		attr := PlaywrightCodeBundlePrebuiltArchiveAttribute{File: archive}
		info, err := attr.InspectLockfile("@playwright/test", InspectLockfileOptions{})
		if err != nil {
			t.Fatalf("InspectLockfile failed: %v", err)
		}
		if info.Engine != "node" {
			t.Errorf("Engine = %q, want %q", info.Engine, "node")
		}
		if info.EngineVersion != "24" {
			t.Errorf("EngineVersion = %q, want %q", info.EngineVersion, "24")
		}
	})

	t.Run("package.json engines.node", func(t *testing.T) {
		t.Parallel()
		archive := buildTarGz(t, []tarEntry{
			{name: "package-lock.json", content: []byte(syntheticPackageLock)},
			{name: "package.json", content: []byte(`{"name":"test","dependencies":{"@playwright/test":"1.58.2"},"engines":{"node":">=22"}}`)},
		})
		attr := PlaywrightCodeBundlePrebuiltArchiveAttribute{File: archive}
		info, err := attr.InspectLockfile("@playwright/test", InspectLockfileOptions{})
		if err != nil {
			t.Fatalf("InspectLockfile failed: %v", err)
		}
		if info.Engine != "node" {
			t.Errorf("Engine = %q, want %q", info.Engine, "node")
		}
		if info.EngineVersion != "24" {
			t.Errorf("EngineVersion = %q, want %q", info.EngineVersion, "24")
		}
	})
}
