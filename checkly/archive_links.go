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

// InspectLinks verifies that every symbolic and hard link in the archive
// resolves to another entry contained in the same archive. Extraction fails
// when a link points outside the bundle or at a file that was not included, so
// such archives are rejected before they are uploaded.
func (a *PlaywrightCodeBundlePrebuiltArchiveAttribute) InspectLinks() error {
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

	return validateArchiveLinks(names, links)
}

// maxReportedArchiveLinks bounds how many offending links a single error lists.
const maxReportedArchiveLinks = 10

// validateArchiveLinks checks that every link resolves to an entry present in
// names, which must hold the normalized path of every archive entry and of
// every ancestor directory.
//
// Link chains need no transitive resolution. If a points at b and b points
// outside the archive, then b is itself a link entry and is rejected on its
// own. Entry order does not matter either, because names is complete before
// any link is checked.
func validateArchiveLinks(names map[string]bool, links []archiveLink) error {
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

	if len(problems) == 0 {
		return nil
	}

	var b strings.Builder

	plural := "links"
	if len(problems) == 1 {
		plural = "link"
	}

	fmt.Fprintf(&b, "the archive contains %d %s that cannot be resolved within it:\n", len(problems), plural)

	shown := problems
	if len(shown) > maxReportedArchiveLinks {
		shown = shown[:maxReportedArchiveLinks]
	}

	for _, problem := range shown {
		b.WriteString(problem)
		b.WriteString("\n")
	}

	if omitted := len(problems) - len(shown); omitted > 0 {
		fmt.Fprintf(&b, "  ... and %d more\n", omitted)
	}

	b.WriteString(
		"every symbolic and hard link must point to a file or directory that is " +
			"also included in the archive; exclude node_modules from the archive " +
			"(Checkly installs dependencies from the lockfile), or rebuild the " +
			"archive with symbolic links dereferenced",
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

	if resolved == ".." || strings.HasPrefix(resolved, "../") {
		return "the target escapes the archive root"
	}

	if !names[resolved] {
		return "the target is not present in the archive"
	}

	return ""
}
