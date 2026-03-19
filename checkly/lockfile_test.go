package checkly

import (
	"strings"
	"testing"
)

func TestExtractPackageVersionFromPackageLock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		packageName string
		want        string
		wantErr     bool
	}{
		{
			name:        "v3 lockfile",
			packageName: "@playwright/test",
			input: `{
				"lockfileVersion": 3,
				"packages": {
					"node_modules/@playwright/test": {
						"version": "1.48.0"
					}
				}
			}`,
			want: "1.48.0",
		},
		{
			name:        "v1 lockfile",
			packageName: "@playwright/test",
			input: `{
				"lockfileVersion": 1,
				"dependencies": {
					"@playwright/test": {
						"version": "1.40.0"
					}
				}
			}`,
			want: "1.40.0",
		},
		{
			name:        "workspace nested dependency",
			packageName: "@playwright/test",
			input: `{
				"lockfileVersion": 3,
				"packages": {
					"packages/e2e/node_modules/@playwright/test": {
						"version": "1.45.0"
					}
				}
			}`,
			want: "1.45.0",
		},
		{
			name:        "package not found",
			packageName: "@playwright/test",
			input:       `{"lockfileVersion": 3, "packages": {}}`,
			want:        "",
		},
		{
			name:        "different package",
			packageName: "typescript",
			input: `{
				"lockfileVersion": 3,
				"packages": {
					"node_modules/typescript": {
						"version": "5.3.2"
					}
				}
			}`,
			want: "5.3.2",
		},
		{
			name:        "invalid json",
			packageName: "@playwright/test",
			input:       `not json`,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := extractPackageVersionFromPackageLock(strings.NewReader(tt.input), tt.packageName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractPackageVersionFromPnpmLock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		packageName string
		want        string
		wantErr     bool
	}{
		{
			name:        "v9 format",
			packageName: "@playwright/test",
			input: `lockfileVersion: '9.0'
packages:
  '@playwright/test@1.48.0':
    resolution: {integrity: sha512-xxx}
`,
			want: "1.48.0",
		},
		{
			name:        "v6 format with leading slash",
			packageName: "@playwright/test",
			input: `lockfileVersion: '6.0'
packages:
  /@playwright/test@1.40.0:
    resolution: {integrity: sha512-xxx}
`,
			want: "1.40.0",
		},
		{
			name:        "v5 format with slash separator",
			packageName: "@playwright/test",
			input: `lockfileVersion: 5.4
packages:
  /@playwright/test/1.35.0:
    resolution: {integrity: sha512-xxx}
`,
			want: "1.35.0",
		},
		{
			name:        "with peer dependencies",
			packageName: "@playwright/test",
			input: `lockfileVersion: '6.0'
packages:
  /@playwright/test@1.40.0(@types/node@20.0.0):
    resolution: {integrity: sha512-xxx}
`,
			want: "1.40.0",
		},
		{
			name:        "importers fallback",
			packageName: "@playwright/test",
			input: `lockfileVersion: '9.0'
importers:
  packages/e2e:
    devDependencies:
      '@playwright/test':
        specifier: ^1.48.0
        version: 1.48.0
packages: {}
`,
			want: "1.48.0",
		},
		{
			name:        "importers dependencies section",
			packageName: "@playwright/test",
			input: `lockfileVersion: '9.0'
importers:
  .:
    dependencies:
      '@playwright/test':
        specifier: ^1.48.0
        version: 1.48.0
packages: {}
`,
			want: "1.48.0",
		},
		{
			name:        "different package",
			packageName: "typescript",
			input: `lockfileVersion: '9.0'
packages:
  typescript@5.3.2:
    resolution: {integrity: sha512-xxx}
`,
			want: "5.3.2",
		},
		{
			name:        "package not found",
			packageName: "@playwright/test",
			input: `lockfileVersion: '9.0'
packages: {}
`,
			want: "",
		},
		{
			name:        "invalid yaml",
			packageName: "@playwright/test",
			input:       `{not yaml`,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := extractPackageVersionFromPnpmLock(strings.NewReader(tt.input), tt.packageName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractVersionFromPnpmPackageKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key         string
		packageName string
		want        string
	}{
		{"/@playwright/test@1.40.0", "@playwright/test", "1.40.0"},
		{"/@playwright/test/1.35.0", "@playwright/test", "1.35.0"},
		{"@playwright/test@1.48.0", "@playwright/test", "1.48.0"},
		{"'@playwright/test@1.48.0'", "@playwright/test", "1.48.0"},
		{"/@playwright/test@1.40.0(@types/node@20.0.0)", "@playwright/test", "1.40.0"},
		{"typescript@5.3.2", "typescript", "5.3.2"},
		{"/typescript@5.3.2", "typescript", "5.3.2"},
		{"/@playwright/test", "@playwright/test", ""},
		{"other-package@1.0.0", "@playwright/test", ""},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			t.Parallel()
			got := extractVersionFromPnpmPackageKey(tt.key, tt.packageName)
			if got != tt.want {
				t.Errorf("extractVersionFromPnpmPackageKey(%q, %q) = %q, want %q", tt.key, tt.packageName, got, tt.want)
			}
		})
	}
}

func TestExtractPackageVersionFromYarnLock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		packageName string
		want        string
		wantErr     bool
	}{
		{
			name:        "classic format",
			packageName: "@playwright/test",
			input: `"@playwright/test@^1.48.0":
  version "1.48.0"
  resolved "https://registry.yarnpkg.com/@playwright/test/-/test-1.48.0.tgz"
  integrity sha512-xxx
`,
			want: "1.48.0",
		},
		{
			name:        "berry format",
			packageName: "@playwright/test",
			input: `"@playwright/test@npm:^1.48.0":
  version: 1.48.0
  resolution: "@playwright/test@npm:1.48.0"
`,
			want: "1.48.0",
		},
		{
			name:        "berry format with quoted version",
			packageName: "@playwright/test",
			input: `"@playwright/test@npm:^1.48.0":
  version: "1.48.0"
  resolution: "@playwright/test@npm:1.48.0"
`,
			want: "1.48.0",
		},
		{
			name:        "different package",
			packageName: "typescript",
			input: `typescript@^5.3.0:
  version "5.3.2"
  resolved "https://registry.yarnpkg.com/typescript/-/typescript-5.3.2.tgz"
`,
			want: "5.3.2",
		},
		{
			name:        "package not found",
			packageName: "@playwright/test",
			input: `typescript@^5.3.0:
  version "5.3.2"
`,
			want: "",
		},
		{
			name:        "multiple packages finds correct one",
			packageName: "@playwright/test",
			input: `typescript@^5.3.0:
  version "5.3.2"
  resolved "https://registry.yarnpkg.com/typescript/-/typescript-5.3.2.tgz"

"@playwright/test@^1.48.0":
  version "1.48.0"
  resolved "https://registry.yarnpkg.com/@playwright/test/-/test-1.48.0.tgz"
`,
			want: "1.48.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := extractPackageVersionFromYarnLock(strings.NewReader(tt.input), tt.packageName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
