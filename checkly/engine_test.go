package checkly

import (
	"testing"
)

func TestParseNodeVersionFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"full version", "22.14.0", "22.14.0"},
		{"with v prefix", "v22.14.0", "22.14.0"},
		{"major only", "22", "22"},
		{"with whitespace", "  22.14.0\n", "22.14.0"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseNodeVersionFile([]byte(tt.content))
			if got != tt.want {
				t.Errorf("parseNodeVersionFile(%q) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}

func TestParseNvmrcFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"full version", "22.14.0", "22.14.0"},
		{"with v prefix", "v22", "22"},
		{"lts wildcard", "lts/*", ""},
		{"lts named", "lts/iron", ""},
		{"lts bare", "lts", ""},
		{"node alias", "node", ""},
		{"stable alias", "stable", ""},
		{"latest alias", "latest", ""},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseNvmrcFile([]byte(tt.content))
			if got != tt.want {
				t.Errorf("parseNvmrcFile(%q) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}

func TestParseToolVersionsFile(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantNode   string
		wantBun    string
	}{
		{"nodejs entry", "nodejs 22.14.0", "22.14.0", ""},
		{"bun entry", "bun 1.3.11", "", "1.3.11"},
		{"multiple tools with nodejs", "python 3.12.0\nnodejs 24.1.0", "24.1.0", ""},
		{"comments and blanks", "# comment\n\nnodejs 22.14.0", "22.14.0", ""},
		{"both nodejs and bun", "nodejs 22.14.0\nbun 1.3.11", "22.14.0", "1.3.11"},
		{"no match", "python 3.12.0\nruby 3.3.0", "", ""},
		{"empty", "", "", ""},
		{"with v prefix", "nodejs v22.14.0", "22.14.0", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNode, gotBun := parseToolVersionsFile([]byte(tt.content))
			if gotNode != tt.wantNode || gotBun != tt.wantBun {
				t.Errorf("parseToolVersionsFile(%q) = (%q, %q), want (%q, %q)",
					tt.content, gotNode, gotBun, tt.wantNode, tt.wantBun)
			}
		})
	}
}

func TestParseBunVersionFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"full version", "1.3.11", "1.3.11"},
		{"major.minor", "1.3", "1.3"},
		{"with whitespace", "  1.3.11\n", "1.3.11"},
		{"with v prefix", "v1.3.11", "1.3.11"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseBunVersionFile([]byte(tt.content))
			if got != tt.want {
				t.Errorf("parseBunVersionFile(%q) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}

func TestParsePackageJSONEngines(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantNode string
		wantBun  string
	}{
		{"node engine", `{"engines":{"node":">=18"}}`, ">=18", ""},
		{"bun engine", `{"engines":{"bun":">=1.0"}}`, "", ">=1.0"},
		{"both", `{"engines":{"node":"^22","bun":">=1.3"}}`, "^22", ">=1.3"},
		{"no engines", `{"name":"test"}`, "", ""},
		{"malformed json", `{invalid`, "", ""},
		{"empty", ``, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNode, gotBun := parsePackageJSONEngines([]byte(tt.content))
			if gotNode != tt.wantNode || gotBun != tt.wantBun {
				t.Errorf("parsePackageJSONEngines(%q) = (%q, %q), want (%q, %q)",
					tt.content, gotNode, gotBun, tt.wantNode, tt.wantBun)
			}
		})
	}
}

func TestResolveNodeMajorVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"22.14.0", "22"},
		{"24.1.0", "24"},
		{"22", "22"},
		{"", ""},
		{"abc", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := resolveNodeMajorVersion(tt.input)
			if got != tt.want {
				t.Errorf("resolveNodeMajorVersion(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveBunVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1.3.11", "1.3"},
		{"1.3", "1.3"},
		{"1", "1"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := resolveBunVersion(tt.input)
			if got != tt.want {
				t.Errorf("resolveBunVersion(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMatchSemverConstraint(t *testing.T) {
	tests := []struct {
		name       string
		constraint string
		available  []string
		want       string
	}{
		{">=18 with node versions", ">=18", []string{"22", "24"}, "24"},
		{"^22 with node versions", "^22", []string{"22", "24"}, "22"},
		{">=25 no match", ">=25", []string{"22", "24"}, ""},
		{">=1.0 with bun", ">=1.0", []string{"1.3"}, "1.3"},
		{"invalid constraint", "not-valid", []string{"22"}, ""},
		{"22 || 24", "22 || 24", []string{"22", "24"}, "24"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchSemverConstraint(tt.constraint, tt.available)
			if got != tt.want {
				t.Errorf("matchSemverConstraint(%q, %v) = %q, want %q",
					tt.constraint, tt.available, got, tt.want)
			}
		})
	}
}

func TestDetectEngine(t *testing.T) {
	tests := []struct {
		name           string
		files          map[string][]byte
		packageManager string
		wantNil        bool
		wantName       string
		wantVersion    string
	}{
		{
			name:           "node-version file",
			files:          map[string][]byte{".node-version": []byte("22.14.0")},
			packageManager: "npm",
			wantName:       "node",
			wantVersion:    "22",
		},
		{
			name:           "node-version with v24",
			files:          map[string][]byte{".node-version": []byte("v24.1.0")},
			packageManager: "pnpm",
			wantName:       "node",
			wantVersion:    "24",
		},
		{
			name:           "bun-version file",
			files:          map[string][]byte{".bun-version": []byte("1.3.11")},
			packageManager: "bun",
			wantName:       "bun",
			wantVersion:    "1.3",
		},
		{
			name: "both node-version and bun-version, npm PM prefers node",
			files: map[string][]byte{
				".node-version": []byte("22"),
				".bun-version":  []byte("1.3.11"),
			},
			packageManager: "npm",
			wantName:       "node",
			wantVersion:    "22",
		},
		{
			name: "both node-version and bun-version, bun PM prefers bun",
			files: map[string][]byte{
				".node-version": []byte("22"),
				".bun-version":  []byte("1.3.11"),
			},
			packageManager: "bun",
			wantName:       "bun",
			wantVersion:    "1.3",
		},
		{
			name:           "nvmrc with lts skips, returns nil",
			files:          map[string][]byte{".nvmrc": []byte("lts/*")},
			packageManager: "pnpm",
			wantNil:        true,
		},
		{
			name:           "tool-versions with nodejs",
			files:          map[string][]byte{".tool-versions": []byte("nodejs 24.1.0")},
			packageManager: "yarn",
			wantName:       "node",
			wantVersion:    "24",
		},
		{
			name:           "tool-versions with bun",
			files:          map[string][]byte{".tool-versions": []byte("bun 1.3.11")},
			packageManager: "bun",
			wantName:       "bun",
			wantVersion:    "1.3",
		},
		{
			name:           "tool-versions with both nodejs and bun, npm PM prefers node",
			files:          map[string][]byte{".tool-versions": []byte("nodejs 24.1.0\nbun 1.3.11")},
			packageManager: "npm",
			wantName:       "node",
			wantVersion:    "24",
		},
		{
			name:           "tool-versions with both nodejs and bun, bun PM prefers bun",
			files:          map[string][]byte{".tool-versions": []byte("nodejs 24.1.0\nbun 1.3.11")},
			packageManager: "bun",
			wantName:       "bun",
			wantVersion:    "1.3",
		},
		{
			name:           "package.json engines node",
			files:          map[string][]byte{"package.json": []byte(`{"engines":{"node":">=22"}}`)},
			packageManager: "npm",
			wantName:       "node",
			wantVersion:    "24",
		},
		{
			name:           "package.json engines bun",
			files:          map[string][]byte{"package.json": []byte(`{"engines":{"bun":">=1.0"}}`)},
			packageManager: "bun",
			wantName:       "bun",
			wantVersion:    "1.3",
		},
		{
			name:           "no files, returns nil",
			files:          map[string][]byte{},
			packageManager: "bun",
			wantNil:        true,
		},
		{
			name:           "node-version with unavailable version returns unmatched",
			files:          map[string][]byte{".node-version": []byte("16.20.0")},
			packageManager: "npm",
			wantName:       "node",
			wantVersion:    "",
		},
		{
			name:           "nvmrc with unsupported version returns unmatched",
			files:          map[string][]byte{".nvmrc": []byte("25")},
			packageManager: "npm",
			wantName:       "node",
			wantVersion:    "",
		},
		{
			name:           "bun-version with unsupported version returns unmatched",
			files:          map[string][]byte{".bun-version": []byte("2.0.0")},
			packageManager: "bun",
			wantName:       "bun",
			wantVersion:    "",
		},
		{
			name: "nvmrc with 25 + package.json engines >=22, pinning file is authoritative",
			files: map[string][]byte{
				".nvmrc":       []byte("25"),
				"package.json": []byte(`{"engines":{"node":">=22"}}`),
			},
			packageManager: "npm",
			wantName:       "node",
			wantVersion:    "",
		},
		{
			name:           "nvmrc takes over when node-version absent",
			files:          map[string][]byte{".nvmrc": []byte("24")},
			packageManager: "npm",
			wantName:       "node",
			wantVersion:    "24",
		},
		{
			name:           "empty node-version file returns nil",
			files:          map[string][]byte{".node-version": []byte("")},
			packageManager: "npm",
			wantNil:        true,
		},
		{
			name: "unmatched node-version + matched bun-version, npm PM selects unmatched node",
			files: map[string][]byte{
				".node-version": []byte("25"),
				".bun-version":  []byte("1.3.11"),
			},
			packageManager: "npm",
			wantName:       "node",
			wantVersion:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectEngine(tt.files, tt.packageManager)
			if tt.wantNil {
				if got != nil {
					t.Errorf("detectEngine() = %+v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("detectEngine returned nil")
			}
			if got.Engine == nil {
				t.Fatal("detectEngine().Engine is nil")
			}
			if got.Engine.Name != tt.wantName {
				t.Errorf("detectEngine().Engine.Name = %q, want %q", got.Engine.Name, tt.wantName)
			}
			if got.Engine.Version != tt.wantVersion {
				t.Errorf("detectEngine().Engine.Version = %q, want %q", got.Engine.Version, tt.wantVersion)
			}
		})
	}
}
