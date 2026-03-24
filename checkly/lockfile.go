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
