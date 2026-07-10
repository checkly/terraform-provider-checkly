package checkly

import (
	"archive/tar"
	"fmt"
	"strings"
	"testing"
)

// archivePathSet builds the same names set that InspectArchivePaths derives
// from an archive: every entry plus every ancestor directory, plus the root.
func archivePathSet(entries ...string) map[string]bool {
	names := newArchivePathSet()
	for _, entry := range entries {
		addArchivePath(names, normalizeArchivePath(entry))
	}
	return names
}

// linkError runs the link half of the archive validation and renders its
// problems the same way InspectArchivePaths does.
func linkError(names map[string]bool, links []archiveLink) error {
	return formatArchiveProblems(collectLinkProblems(names, links))
}

func TestNormalizeArchivePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
	}{
		{"pkg/a.js", "pkg/a.js"},
		{"./pkg/a.js", "pkg/a.js"},
		{"pkg/sub/", "pkg/sub"},
		{"./", "."},
		{"pkg//a.js", "pkg/a.js"},
		{"pkg/./a.js", "pkg/a.js"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := normalizeArchivePath(tt.name); got != tt.want {
				t.Errorf("normalizeArchivePath(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestValidateArchiveLinks(t *testing.T) {
	t.Parallel()

	// "dir/nested" and "pkg/sub" exist only as ancestors of a file entry,
	// mirroring archives that omit explicit directory entries.
	names := archivePathSet("root.js", "pkg/a.js", "pkg/sub/b.js", "dir/nested/deep.js")

	tests := []struct {
		name string
		link archiveLink
		want string // substring of the expected reason, empty means valid
	}{
		{
			name: "symlink to sibling",
			link: archiveLink{name: "pkg/link.js", linkname: "a.js"},
		},
		{
			name: "symlink into parent directory",
			link: archiveLink{name: "pkg/sub/up.js", linkname: "../a.js"},
		},
		{
			name: "symlink to implicit directory",
			link: archiveLink{name: "shortcut", linkname: "dir/nested"},
		},
		{
			name: "symlink to archive root",
			link: archiveLink{name: "self", linkname: "."},
		},
		{
			name: "symlink to missing entry",
			link: archiveLink{name: "pkg/gone.js", linkname: "nope.js"},
			want: "not present in the archive",
		},
		{
			name: "symlink escaping the root",
			link: archiveLink{name: "pkg/escape.js", linkname: "../../outside.js"},
			want: "escapes the archive root",
		},
		{
			name: "symlink with absolute target",
			link: archiveLink{name: "abs", linkname: "/etc/passwd"},
			want: "absolute path outside the archive",
		},
		{
			name: "symlink with empty target",
			link: archiveLink{name: "empty", linkname: ""},
			want: "no target",
		},
		{
			name: "hard link to existing member",
			link: archiveLink{name: "hard", linkname: "pkg/a.js", hard: true},
		},
		{
			name: "hard link target normalized",
			link: archiveLink{name: "hard", linkname: "./pkg/a.js", hard: true},
		},
		{
			// A hard link's target is relative to the archive root, so this
			// resolves to "root.js" rather than to "pkg/root.js".
			name: "hard link target is root relative",
			link: archiveLink{name: "pkg/hard", linkname: "root.js", hard: true},
		},
		{
			name: "hard link to missing member",
			link: archiveLink{name: "hard", linkname: "pkg/a.js.bak", hard: true},
			want: "not present in the archive",
		},
		{
			// The same target as above, but a symlink resolves it relative to
			// the link's own directory, where it does not exist.
			name: "symlink target is not root relative",
			link: archiveLink{name: "pkg/link", linkname: "root.js"},
			want: "not present in the archive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := linkError(names, []archiveLink{tt.link})

			if tt.want == "" {
				if err != nil {
					t.Fatalf("linkError(%+v) = %v, want nil", tt.link, err)
				}
				return
			}

			if err == nil {
				t.Fatalf("linkError(%+v) = nil, want error containing %q", tt.link, tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.want)
			}
			if !strings.Contains(err.Error(), tt.link.name) {
				t.Errorf("error %q does not name the offending link %q", err.Error(), tt.link.name)
			}
		})
	}
}

func TestValidateArchiveLinksChain(t *testing.T) {
	t.Parallel()

	// "first" points at "second", which is itself a link that escapes. Only
	// "second" is at fault; each link is judged on its own target.
	names := archivePathSet("first", "second")

	err := linkError(names, []archiveLink{
		{name: "first", linkname: "second"},
		{name: "second", linkname: "../../outside"},
	})
	if err == nil {
		t.Fatal("linkError() = nil, want error")
	}

	if strings.Count(err.Error(), " -> ") != 1 {
		t.Errorf("expected exactly one offending link, got: %v", err)
	}
	if !strings.Contains(err.Error(), "second -> ../../outside") {
		t.Errorf("error does not blame the escaping link: %v", err)
	}
}

func TestValidateArchiveLinksOrderIndependent(t *testing.T) {
	t.Parallel()

	// A hard link may precede the member it targets in the archive.
	names := archivePathSet("a.js", "later.js")

	err := linkError(names, []archiveLink{
		{name: "a.js", linkname: "later.js", hard: true},
	})
	if err != nil {
		t.Fatalf("linkError() = %v, want nil", err)
	}
}

func TestValidateArchiveLinksTruncatesReport(t *testing.T) {
	t.Parallel()

	names := archivePathSet("root.js")

	var links []archiveLink
	for i := range maxReportedArchiveProblems + 2 {
		links = append(links, archiveLink{
			name:     fmt.Sprintf("link-%d", i),
			linkname: "../outside",
		})
	}

	err := linkError(names, links)
	if err == nil {
		t.Fatal("linkError() = nil, want error")
	}

	if got, want := strings.Count(err.Error(), " -> "), maxReportedArchiveProblems; got != want {
		t.Errorf("listed %d links, want %d", got, want)
	}
	if !strings.Contains(err.Error(), "... and 2 more") {
		t.Errorf("error does not mention the omitted links: %v", err)
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("contains %d entries", len(links))) {
		t.Errorf("error does not report the total count: %v", err)
	}
}

func TestCheckEntryPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		// name is the raw tar entry name, normalized before checking, exactly
		// as InspectArchivePaths does it.
		name string
		want string // substring of the expected reason, empty means safe
	}{
		{name: "package.json"},
		{name: "packages/e2e/spec.ts"},
		{name: "./tests/a.ts"},
		{name: "."},
		{name: "a/../b.ts"}, // stays within the root after cleaning
		{name: "/etc/passwd", want: "absolute path"},
		{name: "/", want: "absolute path"},
		{name: "../secrets.env", want: "escapes the archive root"},
		{name: "..", want: "escapes the archive root"},
		{name: "a/../../evil.sh", want: "escapes the archive root"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := checkEntryPath(normalizeArchivePath(tt.name))

			if tt.want == "" {
				if got != "" {
					t.Errorf("checkEntryPath(%q) = %q, want safe", tt.name, got)
				}
				return
			}
			if !strings.Contains(got, tt.want) {
				t.Errorf("checkEntryPath(%q) = %q, want a reason containing %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestInspectArchivePaths(t *testing.T) {
	t.Parallel()

	t.Run("archive without links", func(t *testing.T) {
		t.Parallel()

		file := buildTarGz(t, []tarEntry{
			{name: "package.json", content: []byte(`{"name":"example"}`)},
			{name: "tests/spec.ts", content: []byte("test")},
		})

		attr := PlaywrightCodeBundlePrebuiltArchiveAttribute{File: file}
		if err := attr.InspectArchivePaths(); err != nil {
			t.Fatalf("InspectArchivePaths() = %v, want nil", err)
		}
	})

	t.Run("archive with resolvable links", func(t *testing.T) {
		t.Parallel()

		file := buildTarGz(t, []tarEntry{
			{name: "package.json", content: []byte(`{"name":"example"}`)},
			{name: "packages/app/spec.ts", content: []byte("test")},
			{name: "packages/app/link.ts", typeflag: tar.TypeSymlink, linkname: "spec.ts"},
			{name: "shortcut", typeflag: tar.TypeSymlink, linkname: "packages/app"},
			{name: "manifest.json", typeflag: tar.TypeLink, linkname: "package.json"},
		})

		attr := PlaywrightCodeBundlePrebuiltArchiveAttribute{File: file}
		if err := attr.InspectArchivePaths(); err != nil {
			t.Fatalf("InspectArchivePaths() = %v, want nil", err)
		}
	})

	t.Run("archive with escaping symlink", func(t *testing.T) {
		t.Parallel()

		file := buildTarGz(t, []tarEntry{
			{name: "package.json", content: []byte(`{"name":"example"}`)},
			{
				name:     "node_modules/.bin/playwright",
				typeflag: tar.TypeSymlink,
				linkname: "../../../../elsewhere/playwright",
			},
		})

		attr := PlaywrightCodeBundlePrebuiltArchiveAttribute{File: file}
		err := attr.InspectArchivePaths()
		if err == nil {
			t.Fatal("InspectArchivePaths() = nil, want error")
		}

		for _, want := range []string{
			"node_modules/.bin/playwright",
			"../../../../elsewhere/playwright",
			"escapes the archive root",
		} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q does not contain %q", err.Error(), want)
			}
		}
	})

	t.Run("archive with dangling symlink", func(t *testing.T) {
		t.Parallel()

		file := buildTarGz(t, []tarEntry{
			{name: "package.json", content: []byte(`{"name":"example"}`)},
			{name: "link.ts", typeflag: tar.TypeSymlink, linkname: "missing.ts"},
		})

		attr := PlaywrightCodeBundlePrebuiltArchiveAttribute{File: file}
		err := attr.InspectArchivePaths()
		if err == nil {
			t.Fatal("InspectArchivePaths() = nil, want error")
		}
		if !strings.Contains(err.Error(), "not present in the archive") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("archive with absolute entry path", func(t *testing.T) {
		t.Parallel()

		file := buildTarGz(t, []tarEntry{
			{name: "package.json", content: []byte(`{"name":"example"}`)},
			{name: "/etc/cron.d/evil", content: []byte("x")},
		})

		attr := PlaywrightCodeBundlePrebuiltArchiveAttribute{File: file}
		err := attr.InspectArchivePaths()
		if err == nil {
			t.Fatal("InspectArchivePaths() = nil, want error")
		}
		for _, want := range []string{"/etc/cron.d/evil", "absolute path"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q does not contain %q", err.Error(), want)
			}
		}
	})

	t.Run("archive with parent-escaping entry path", func(t *testing.T) {
		t.Parallel()

		file := buildTarGz(t, []tarEntry{
			{name: "package.json", content: []byte(`{"name":"example"}`)},
			{name: "../../outside/evil.sh", content: []byte("x")},
		})

		attr := PlaywrightCodeBundlePrebuiltArchiveAttribute{File: file}
		err := attr.InspectArchivePaths()
		if err == nil {
			t.Fatal("InspectArchivePaths() = nil, want error")
		}
		for _, want := range []string{"../../outside/evil.sh", "escapes the archive root"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q does not contain %q", err.Error(), want)
			}
		}
	})

	t.Run("archive with both a bad entry path and a bad link", func(t *testing.T) {
		t.Parallel()

		file := buildTarGz(t, []tarEntry{
			{name: "package.json", content: []byte(`{"name":"example"}`)},
			{name: "../escape.txt", content: []byte("x")},
			{name: "link.ts", typeflag: tar.TypeSymlink, linkname: "gone.ts"},
		})

		attr := PlaywrightCodeBundlePrebuiltArchiveAttribute{File: file}
		err := attr.InspectArchivePaths()
		if err == nil {
			t.Fatal("InspectArchivePaths() = nil, want error")
		}
		if !strings.Contains(err.Error(), "contains 2 entries") {
			t.Errorf("error should report both problems: %v", err)
		}
		for _, want := range []string{"../escape.txt", "link.ts -> gone.ts"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q does not contain %q", err.Error(), want)
			}
		}
	})
}
