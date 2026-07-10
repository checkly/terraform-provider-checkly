package checkly

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

// archiveLink describes a symbolic or hard link entry found in an archive.
type archiveLink struct {
	// name is the normalized path of the link entry itself.
	name string

	// linkname is the target exactly as stored in the tar header.
	linkname string

	// hard distinguishes hard links from symbolic links. A hard link's
	// target names another archive member relative to the archive root,
	// whereas a symbolic link's target is relative to the directory that
	// holds the link.
	hard bool
}

// normalizeArchivePath collapses the different ways tar writers spell the same
// path (a leading "./", a trailing "/", redundant separators) into one key.
func normalizeArchivePath(name string) string {
	return path.Clean(strings.TrimPrefix(name, "./"))
}

// newArchivePathSet returns a set that already contains the archive root.
func newArchivePathSet() map[string]bool {
	return map[string]bool{".": true}
}

// addArchivePath records name along with every one of its ancestor
// directories. Recording ancestors matters because tar archives frequently
// omit explicit directory entries, and a link may point at a directory.
func addArchivePath(names map[string]bool, name string) {
	names[name] = true

	for dir := path.Dir(name); dir != "." && dir != "/" && !names[dir]; dir = path.Dir(dir) {
		names[dir] = true
	}
}

// InspectArchivePaths checks that the archive can be extracted cleanly and is
// self-contained:
//
//   - no entry has an absolute path or a path that escapes the archive root,
//     either of which would write outside the extraction directory; and
//   - every symbolic and hard link resolves to another entry contained in the
//     archive.
//
// Archives that violate either rule fail to extract on the Checkly backend, so
// catching them here turns a confusing server-side failure into an actionable
// plan-time error.
//
// These path checks exist ONLY to improve the user experience by surfacing such
// problems early. They are not a security boundary: Checkly does not rely on
// them for security.
func (a *PlaywrightCodeBundlePrebuiltArchiveAttribute) InspectArchivePaths() error {
	file, err := os.Open(a.File)
	if err != nil {
		return fmt.Errorf("failed to open archive file %q: %w", a.File, err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader for %q: %w", a.File, err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	names := newArchivePathSet()
	var links []archiveLink
	var pathProblems []string

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read archive %q: %w", a.File, err)
		}

		if header.Typeflag == tar.TypeXGlobalHeader {
			// Carries archive-wide metadata rather than an entry.
			continue
		}

		name := normalizeArchivePath(header.Name)

		if reason := checkEntryPath(name); reason != "" {
			// The entry itself is unsafe regardless of its type, so it is
			// reported and not recorded as a resolution target or validated
			// as a link. The raw header name is shown so the user sees the
			// path exactly as stored in their archive.
			pathProblems = append(pathProblems, fmt.Sprintf("  - %s (%s)", header.Name, reason))
			continue
		}

		addArchivePath(names, name)

		switch header.Typeflag {
		case tar.TypeSymlink:
			links = append(links, archiveLink{
				name:     name,
				linkname: header.Linkname,
			})
		case tar.TypeLink:
			links = append(links, archiveLink{
				name:     name,
				linkname: header.Linkname,
				hard:     true,
			})
		}
	}

	problems := append(pathProblems, collectLinkProblems(names, links)...)

	return formatArchiveProblems(problems)
}

// maxReportedArchiveProblems bounds how many offending entries a single error
// lists.
const maxReportedArchiveProblems = 10

// escapesArchiveRoot reports whether a path points above the archive root. name
// must be cleaned and relative — the output of normalizeArchivePath or of
// path.Join, both of which collapse "." and ".." elements, so any leading ".."
// that survives can only mean the path leaves the root.
func escapesArchiveRoot(name string) bool {
	return name == ".." || strings.HasPrefix(name, "../")
}

// checkEntryPath returns a reason why an entry's own path would not extract
// safely within the archive root, or an empty string if it is safe. name must
// already be normalized.
func checkEntryPath(name string) string {
	if path.IsAbs(name) {
		return "the entry has an absolute path"
	}

	if escapesArchiveRoot(name) {
		return "the entry path escapes the archive root"
	}

	return ""
}

// collectLinkProblems returns a formatted problem line for every link that
// cannot be resolved within names, which must hold the normalized path of every
// archive entry and of every ancestor directory.
//
// Link chains need no transitive resolution. If a points at b and b points
// outside the archive, then b is itself a link entry and is rejected on its
// own. Entry order does not matter either, because names is complete before
// any link is checked.
func collectLinkProblems(names map[string]bool, links []archiveLink) []string {
	var problems []string

	for _, link := range links {
		reason := checkArchiveLink(names, link)
		if reason == "" {
			continue
		}

		problems = append(problems, fmt.Sprintf(
			"  - %s -> %s (%s)", link.name, link.linkname, reason,
		))
	}

	return problems
}

// formatArchiveProblems renders the collected problem lines into a single
// error, or returns nil when there are none. At most
// maxReportedArchiveProblems lines are listed; the header always reports the
// true total.
func formatArchiveProblems(problems []string) error {
	if len(problems) == 0 {
		return nil
	}

	var b strings.Builder

	noun := "entries"
	if len(problems) == 1 {
		noun = "entry"
	}

	fmt.Fprintf(&b, "the archive contains %d %s that cannot be safely extracted:\n", len(problems), noun)

	shown := problems
	if len(shown) > maxReportedArchiveProblems {
		shown = shown[:maxReportedArchiveProblems]
	}

	for _, problem := range shown {
		b.WriteString(problem)
		b.WriteString("\n")
	}

	if omitted := len(problems) - len(shown); omitted > 0 {
		fmt.Fprintf(&b, "  ... and %d more\n", omitted)
	}

	b.WriteString(
		"every entry must extract within the archive root, and every symbolic " +
			"or hard link must resolve to a file or directory that is also " +
			"included in the archive; rebuild the archive without absolute or " +
			"parent-escaping paths and with its symbolic links dereferenced",
	)

	return fmt.Errorf("%s", b.String())
}

// checkArchiveLink returns a human-readable reason why the link cannot be
// resolved within the archive, or an empty string if it can.
//
// The resolved target is matched against archive members directly. Symlinked
// directory components in the middle of a target path are not followed, so a
// link that would only resolve by traversing another symlink is rejected. This
// is deliberately conservative: such archives are rare, and rejecting them is
// safer than uploading one the backend cannot extract.
func checkArchiveLink(names map[string]bool, link archiveLink) string {
	if link.linkname == "" {
		return "the link has no target"
	}

	if path.IsAbs(link.linkname) {
		return "the target is an absolute path outside the archive"
	}

	base := "."
	if !link.hard {
		base = path.Dir(link.name)
	}

	// path.Join cleans the result, resolving any "." and ".." elements.
	resolved := path.Join(base, link.linkname)

	if escapesArchiveRoot(resolved) {
		return "the target escapes the archive root"
	}

	if !names[resolved] {
		return "the target is not present in the archive"
	}

	return ""
}
