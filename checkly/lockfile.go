package checkly

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// extractPackageVersionFromPackageLock extracts the version of the given
// package from a package-lock.json file. Supports lockfile v1 (dependencies)
// and v2/v3 (packages), including workspace layouts where the package may
// be nested under a child package's node_modules.
func extractPackageVersionFromPackageLock(r io.Reader, packageName string) (string, error) {
	var lockfile struct {
		Packages map[string]struct {
			Version string `json:"version"`
		} `json:"packages"`
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}

	if err := json.NewDecoder(r).Decode(&lockfile); err != nil {
		return "", fmt.Errorf("failed to parse package-lock.json: %w", err)
	}

	// v2/v3: scan packages for any key ending with node_modules/PACKAGE
	// to handle both root and workspace-nested dependencies.
	suffix := "node_modules/" + packageName
	for key, pkg := range lockfile.Packages {
		if strings.HasSuffix(key, suffix) && pkg.Version != "" {
			return pkg.Version, nil
		}
	}

	// v1: check top-level dependencies
	if dep, ok := lockfile.Dependencies[packageName]; ok && dep.Version != "" {
		return dep.Version, nil
	}

	return "", nil
}

// extractPackageVersionFromPnpmLock extracts the version of the given package
// from a pnpm-lock.yaml file. Checks both the packages section (all versions)
// and the importers section (workspace-specific dependencies).
func extractPackageVersionFromPnpmLock(r io.Reader, packageName string) (string, error) {
	var lockfile struct {
		Packages  map[string]any `yaml:"packages"`
		Importers map[string]struct {
			Dependencies    map[string]pnpmImporterDep `yaml:"dependencies"`
			DevDependencies map[string]pnpmImporterDep `yaml:"devDependencies"`
		} `yaml:"importers"`
	}

	if err := yaml.NewDecoder(r).Decode(&lockfile); err != nil {
		return "", fmt.Errorf("failed to parse pnpm-lock.yaml: %w", err)
	}

	// Check packages section first — version is encoded in the key.
	for key := range lockfile.Packages {
		if version := extractVersionFromPnpmPackageKey(key, packageName); version != "" {
			return version, nil
		}
	}

	// Fallback: check importers (workspace packages) for the dependency.
	for _, importer := range lockfile.Importers {
		if dep, ok := importer.Dependencies[packageName]; ok && dep.Version != "" {
			return dep.Version, nil
		}
		if dep, ok := importer.DevDependencies[packageName]; ok && dep.Version != "" {
			return dep.Version, nil
		}
	}

	return "", nil
}

type pnpmImporterDep struct {
	Specifier string `yaml:"specifier"`
	Version   string `yaml:"version"`
}

// extractVersionFromPnpmPackageKey extracts the version of a given package
// from a pnpm lockfile package key. It handles the following formats:
//
//   - /package@version (v6-v8)
//   - /package@version(peer_deps) (v6-v8 with peers)
//   - /package/version (v5)
//   - package@version (v9+)
func extractVersionFromPnpmPackageKey(key string, packageName string) string {
	key = strings.Trim(key, "'\"")
	key = strings.TrimPrefix(key, "/")

	if !strings.HasPrefix(key, packageName) {
		return ""
	}

	rest := key[len(packageName):]
	if len(rest) == 0 {
		return ""
	}

	sep := rest[0]
	if sep != '@' && sep != '/' {
		return ""
	}

	version := rest[1:]

	// Remove trailing parenthetical peer dependencies
	if idx := strings.IndexByte(version, '('); idx >= 0 {
		version = version[:idx]
	}

	return version
}

// extractPackageVersionFromBunLock extracts the version of the given package
// from a bun.lock file. bun.lock is a JSONC-like format (JSON with optional
// comments and trailing commas). Each entry in the top-level "packages"
// object is an array whose first element is "name@version".
func extractPackageVersionFromBunLock(r io.Reader, packageName string) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("failed to read bun.lock: %w", err)
	}

	normalized := normalizeJSONC(data)

	var lockfile struct {
		Packages map[string]json.RawMessage `json:"packages"`
	}

	if err := json.Unmarshal(normalized, &lockfile); err != nil {
		return "", fmt.Errorf("failed to parse bun.lock: %w", err)
	}

	prefix := packageName + "@"
	for _, raw := range lockfile.Packages {
		var tuple []json.RawMessage
		if err := json.Unmarshal(raw, &tuple); err != nil || len(tuple) == 0 {
			continue
		}

		var first string
		if err := json.Unmarshal(tuple[0], &first); err != nil {
			continue
		}

		if !strings.HasPrefix(first, prefix) {
			continue
		}

		version := first[len(prefix):]
		version = strings.TrimPrefix(version, "npm:")

		// Non-registry protocols (workspace:, git+, file:, link:) aren't
		// resolvable to a concrete version — skip them.
		if strings.ContainsRune(version, ':') {
			continue
		}

		if version != "" {
			return version, nil
		}
	}

	return "", nil
}

// normalizeJSONC strips // line comments, /* */ block comments, and trailing
// commas from JSONC-like input so that encoding/json can parse it. It is not
// a fully conforming JSONC parser — it is just enough to handle bun.lock.
func normalizeJSONC(b []byte) []byte {
	out := make([]byte, 0, len(b))
	inString := false
	escape := false

	for i := 0; i < len(b); i++ {
		c := b[i]

		if inString {
			out = append(out, c)
			switch {
			case escape:
				escape = false
			case c == '\\':
				escape = true
			case c == '"':
				inString = false
			}
			continue
		}

		if c == '"' {
			inString = true
			out = append(out, c)
			continue
		}

		if c == '/' && i+1 < len(b) {
			if b[i+1] == '/' {
				i += 2
				for i < len(b) && b[i] != '\n' {
					i++
				}
				if i < len(b) {
					out = append(out, b[i])
				}
				continue
			}
			if b[i+1] == '*' {
				i += 2
				for i+1 < len(b) && !(b[i] == '*' && b[i+1] == '/') {
					i++
				}
				i++
				continue
			}
		}

		if c == ',' {
			j := i + 1
			for j < len(b) && (b[j] == ' ' || b[j] == '\t' || b[j] == '\n' || b[j] == '\r') {
				j++
			}
			if j < len(b) && (b[j] == '}' || b[j] == ']') {
				continue
			}
		}

		out = append(out, c)
	}

	return out
}

var yarnVersionRegexp = regexp.MustCompile(`^version[:\s]+["']?([^"'\s]+)["']?`)

// extractPackageVersionFromYarnLock extracts the version of the given package
// from a yarn.lock file. Handles both classic (v1) and berry (v2+) formats.
// Workspace layouts don't affect yarn.lock structure — all resolved packages
// are listed in the same flat structure.
func extractPackageVersionFromYarnLock(r io.Reader, packageName string) (string, error) {
	scanner := bufio.NewScanner(r)

	inBlock := false
	for scanner.Scan() {
		line := scanner.Text()

		if !inBlock {
			trimmed := strings.TrimLeft(line, "\"")
			if strings.HasPrefix(trimmed, packageName+"@") {
				inBlock = true
			}
			continue
		}

		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "version") {
			if matches := yarnVersionRegexp.FindStringSubmatch(trimmed); len(matches) >= 2 {
				return matches[1], nil
			}
		}

		// A non-indented, non-empty line means we've left the block
		if line != "" && line[0] != ' ' && line[0] != '\t' {
			inBlock = false
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read yarn.lock: %w", err)
	}

	return "", nil
}
