package checkly

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
)

var availableNodeVersions = []string{"22", "24"}
var availableBunVersions = []string{"1.3"}

type EngineInfo struct {
	Name    string
	Version string
}

func parseNodeVersionFile(content []byte) string {
	s := strings.TrimSpace(string(content))
	s = strings.TrimPrefix(s, "v")
	return s
}

func parseNvmrcFile(content []byte) string {
	s := strings.TrimSpace(string(content))
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "lts/") || s == "lts" || s == "node" || s == "stable" || s == "latest" {
		return ""
	}
	s = strings.TrimPrefix(s, "v")
	return s
}

func parseToolVersionsFile(content []byte) (nodeVersion string, bunVersion string) {
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		switch parts[0] {
		case "nodejs":
			if nodeVersion == "" {
				nodeVersion = strings.TrimPrefix(parts[1], "v")
			}
		case "bun":
			if bunVersion == "" {
				bunVersion = strings.TrimPrefix(parts[1], "v")
			}
		}
	}
	return nodeVersion, bunVersion
}

func parseBunVersionFile(content []byte) string {
	s := strings.TrimSpace(string(content))
	s = strings.TrimPrefix(s, "v")
	return s
}

func parsePackageJSONEngines(content []byte) (nodeRange string, bunRange string) {
	var pkg struct {
		Engines struct {
			Node string `json:"node"`
			Bun  string `json:"bun"`
		} `json:"engines"`
	}
	if err := json.Unmarshal(content, &pkg); err != nil {
		return "", ""
	}
	return pkg.Engines.Node, pkg.Engines.Bun
}

func resolveNodeMajorVersion(raw string) string {
	if raw == "" {
		return ""
	}
	parts := strings.SplitN(raw, ".", 2)
	major := parts[0]
	if major == "" {
		return ""
	}
	for _, c := range major {
		if c < '0' || c > '9' {
			return ""
		}
	}
	return major
}

func resolveBunVersion(raw string) string {
	if raw == "" {
		return ""
	}
	parts := strings.SplitN(raw, ".", 3)
	if len(parts) < 2 {
		return parts[0]
	}
	return fmt.Sprintf("%s.%s", parts[0], parts[1])
}

func matchAvailableVersion(parsed string, available []string) string {
	for _, v := range available {
		if v == parsed {
			return v
		}
	}
	return ""
}

func matchSemverConstraint(constraint string, available []string) string {
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return ""
	}

	var best string
	var bestVer *semver.Version
	for _, v := range available {
		// Pad to valid semver: "22" → "22.0.0", "1.3" → "1.3.0"
		padded := v
		switch strings.Count(padded, ".") {
		case 0:
			padded += ".0.0"
		case 1:
			padded += ".0"
		}
		sv, err := semver.NewVersion(padded)
		if err != nil {
			continue
		}
		if c.Check(sv) {
			if bestVer == nil || sv.GreaterThan(bestVer) {
				best = v
				bestVer = sv
			}
		}
	}
	return best
}

func detectEngine(files map[string][]byte, packageManager string) *EngineInfo {
	preferBun := packageManager == "bun"

	type candidate struct {
		engine  string
		version string
	}

	var nodeCandidate, bunCandidate *candidate

	// 1. .node-version
	if raw, ok := files[".node-version"]; ok {
		if parsed := parseNodeVersionFile(raw); parsed != "" {
			major := resolveNodeMajorVersion(parsed)
			if matched := matchAvailableVersion(major, availableNodeVersions); matched != "" {
				nodeCandidate = &candidate{engine: "node", version: matched}
			}
		}
	}

	// 2. .nvmrc (only if no .node-version match yet)
	if nodeCandidate == nil {
		if raw, ok := files[".nvmrc"]; ok {
			if parsed := parseNvmrcFile(raw); parsed != "" {
				major := resolveNodeMajorVersion(parsed)
				if matched := matchAvailableVersion(major, availableNodeVersions); matched != "" {
					nodeCandidate = &candidate{engine: "node", version: matched}
				}
			}
		}
	}

	// 3. .tool-versions
	if raw, ok := files[".tool-versions"]; ok {
		tvNodeVersion, tvBunVersion := parseToolVersionsFile(raw)
		if tvNodeVersion != "" && nodeCandidate == nil {
			major := resolveNodeMajorVersion(tvNodeVersion)
			if matched := matchAvailableVersion(major, availableNodeVersions); matched != "" {
				nodeCandidate = &candidate{engine: "node", version: matched}
			}
		}
		if tvBunVersion != "" && bunCandidate == nil {
			minor := resolveBunVersion(tvBunVersion)
			if matched := matchAvailableVersion(minor, availableBunVersions); matched != "" {
				bunCandidate = &candidate{engine: "bun", version: matched}
			}
		}
	}

	// 4. .bun-version
	if bunCandidate == nil {
		if raw, ok := files[".bun-version"]; ok {
			if parsed := parseBunVersionFile(raw); parsed != "" {
				minor := resolveBunVersion(parsed)
				if matched := matchAvailableVersion(minor, availableBunVersions); matched != "" {
					bunCandidate = &candidate{engine: "bun", version: matched}
				}
			}
		}
	}

	// 5. package.json engines
	if raw, ok := files["package.json"]; ok {
		nodeRange, bunRange := parsePackageJSONEngines(raw)
		if nodeCandidate == nil && nodeRange != "" {
			if matched := matchSemverConstraint(nodeRange, availableNodeVersions); matched != "" {
				nodeCandidate = &candidate{engine: "node", version: matched}
			}
		}
		if bunCandidate == nil && bunRange != "" {
			if matched := matchSemverConstraint(bunRange, availableBunVersions); matched != "" {
				bunCandidate = &candidate{engine: "bun", version: matched}
			}
		}
	}

	// Select based on package manager tiebreaker
	if preferBun {
		if bunCandidate != nil {
			return &EngineInfo{Name: bunCandidate.engine, Version: bunCandidate.version}
		}
		if nodeCandidate != nil {
			return &EngineInfo{Name: nodeCandidate.engine, Version: nodeCandidate.version}
		}
	} else {
		if nodeCandidate != nil {
			return &EngineInfo{Name: nodeCandidate.engine, Version: nodeCandidate.version}
		}
		if bunCandidate != nil {
			return &EngineInfo{Name: bunCandidate.engine, Version: bunCandidate.version}
		}
	}

	// 6. Fallback: infer from package manager
	if preferBun {
		return &EngineInfo{Name: "bun", Version: "1.3"}
	}
	return &EngineInfo{Name: "node", Version: "22"}
}
