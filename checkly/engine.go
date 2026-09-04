package checkly

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
)

type EngineInfo struct {
	Name    string
	Version string
}

type EngineDetectionResult struct {
	Engine     *EngineInfo
	RawVersion string
	Source     string
	Notices    []string
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

// parsePackageJSONVoltaNode returns the Node version pinned by Volta in
// package.json ("volta": {"node": "24.17.0"}), or "" when absent. Volta can
// also pin package managers (volta.npm, volta.yarn, volta.pnpm) but never
// Bun, so volta.node is the only key relevant to engine detection. A "volta.extends"
// reference to another manifest is not followed: the code bundle archive
// holds just the check's own directory, and an extends target normally
// points at a workspace-root manifest that is not part of the archive.
func parsePackageJSONVoltaNode(content []byte) string {
	var pkg struct {
		Volta struct {
			Node string `json:"node"`
		} `json:"volta"`
	}
	if err := json.Unmarshal(content, &pkg); err != nil {
		return ""
	}
	s := strings.TrimSpace(pkg.Volta.Node)
	return strings.TrimPrefix(s, "v")
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

var firstVersionRegex = regexp.MustCompile(`\d+(\.\d+)*`)

// extractVersionFromConstraint extracts the first version-like string from a
// semver constraint (e.g., ">=22" → "22", "^1.3" → "1.3").
func extractVersionFromConstraint(constraint string) string {
	m := firstVersionRegex.FindString(constraint)
	return m
}

// resolveVersionForEngine resolves a raw version string using the engine rules.
// Returns the resolved version (or empty if denied/unparseable) and any notices.
func resolveVersionForEngine(engineName string, rawVersion string) (string, []string) {
	config, ok := engineConfigs[engineName]
	if !ok {
		return rawVersion, nil
	}

	var version string
	if engineName == "node" {
		version = resolveNodeMajorVersion(rawVersion)
	} else {
		version = resolveBunVersion(rawVersion)
	}
	if version == "" {
		return "", nil
	}

	res := resolveEngineVersion(version, config)
	if res.Denied {
		return "", res.Notices
	}
	return res.Version, res.Notices
}

// resolveConstraintForEngine resolves a semver constraint (from package.json engines)
// through the engine rules. Extracts the base version, then applies rules.
func resolveConstraintForEngine(engineName string, constraint string) (string, []string) {
	// Try semver.minVersion first for accurate resolution
	c, err := semver.NewConstraint(constraint)
	if err == nil {
		// Test versions from the rules' targets to find the best match
		config := engineConfigs[engineName]
		var best string
		var bestVer *semver.Version
		targets := collectTargets(config)
		for _, t := range targets {
			sv, err := semver.NewVersion(t)
			if err != nil {
				continue
			}
			if c.Check(sv) {
				if bestVer == nil || sv.GreaterThan(bestVer) {
					best = t
					bestVer = sv
				}
			}
		}
		if best != "" {
			return resolveVersionForEngine(engineName, best)
		}
	}

	// Fallback: extract first version-like string from constraint
	extracted := extractVersionFromConstraint(constraint)
	if extracted == "" {
		return "", nil
	}
	return resolveVersionForEngine(engineName, extracted)
}

// collectTargets returns all unique target versions from the rules (plus default).
func collectTargets(config engineVersionConfig) []string {
	seen := make(map[string]bool)
	var targets []string
	if config.Default != "" {
		seen[config.Default] = true
		targets = append(targets, config.Default)
	}
	for _, r := range config.Rules {
		if r.Target != "" && !seen[r.Target] {
			seen[r.Target] = true
			targets = append(targets, r.Target)
		}
	}
	return targets
}

func detectEngine(files map[string][]byte, packageManager string) *EngineDetectionResult {
	preferBun := packageManager == "bun"

	type candidate struct {
		engine     string
		version    string
		rawVersion string
		source     string
		notices    []string
	}

	var nodeCandidate, bunCandidate *candidate

	tryNode := func(raw, source string) {
		if nodeCandidate != nil {
			return
		}
		resolved, notices := resolveVersionForEngine("node", raw)
		nodeCandidate = &candidate{engine: "node", version: resolved, rawVersion: raw, source: source, notices: notices}
	}

	tryBun := func(raw, source string) {
		if bunCandidate != nil {
			return
		}
		resolved, notices := resolveVersionForEngine("bun", raw)
		bunCandidate = &candidate{engine: "bun", version: resolved, rawVersion: raw, source: source, notices: notices}
	}

	// 1. .node-version (pinning file)
	if raw, ok := files[".node-version"]; ok {
		if parsed := parseNodeVersionFile(raw); parsed != "" {
			tryNode(parsed, ".node-version")
		}
	}

	// 2. .nvmrc (pinning file, only if no .node-version found)
	if nodeCandidate == nil {
		if raw, ok := files[".nvmrc"]; ok {
			if parsed := parseNvmrcFile(raw); parsed != "" {
				tryNode(parsed, ".nvmrc")
			}
		}
	}

	// 3. .tool-versions (pinning file)
	if raw, ok := files[".tool-versions"]; ok {
		tvNodeVersion, tvBunVersion := parseToolVersionsFile(raw)
		if tvNodeVersion != "" && nodeCandidate == nil {
			tryNode(tvNodeVersion, ".tool-versions")
		}
		if tvBunVersion != "" && bunCandidate == nil {
			tryBun(tvBunVersion, ".tool-versions")
		}
	}

	// 4. .bun-version (pinning file)
	if bunCandidate == nil {
		if raw, ok := files[".bun-version"]; ok {
			if parsed := parseBunVersionFile(raw); parsed != "" {
				tryBun(parsed, ".bun-version")
			}
		}
	}

	if raw, ok := files["package.json"]; ok {
		// 5. package.json volta.node (pin — outranks the engines range, but
		// loses to any dedicated version file). Like the other pinning
		// sources, a value that cannot be resolved still claims the node
		// candidate so the failure is reported rather than silently falling
		// through to engines.node. This is stricter than .nvmrc, where aliases
		// such as "lts/*" are skipped: Volta itself only ever writes exact
		// versions, so an alias here is a broken pin worth surfacing.
		if nodeCandidate == nil {
			if pin := parsePackageJSONVoltaNode(raw); pin != "" {
				resolved, notices := resolveConstraintForEngine("node", pin)
				nodeCandidate = &candidate{engine: "node", version: resolved, rawVersion: pin, source: "package.json volta.node", notices: notices}
			}
		}

		// 6. package.json engines (range file — only consulted when no pinning source was found for that engine)
		nodeRange, bunRange := parsePackageJSONEngines(raw)
		if nodeCandidate == nil && nodeRange != "" {
			if resolved, notices := resolveConstraintForEngine("node", nodeRange); resolved != "" {
				nodeCandidate = &candidate{engine: "node", version: resolved, rawVersion: nodeRange, source: "package.json engines.node", notices: notices}
			}
		}
		if bunCandidate == nil && bunRange != "" {
			if resolved, notices := resolveConstraintForEngine("bun", bunRange); resolved != "" {
				bunCandidate = &candidate{engine: "bun", version: resolved, rawVersion: bunRange, source: "package.json engines.bun", notices: notices}
			}
		}
	}

	toResult := func(c *candidate) *EngineDetectionResult {
		return &EngineDetectionResult{
			Engine:     &EngineInfo{Name: c.engine, Version: c.version},
			RawVersion: c.rawVersion,
			Source:     c.source,
			Notices:    c.notices,
		}
	}

	if preferBun {
		if bunCandidate != nil {
			return toResult(bunCandidate)
		}
		if nodeCandidate != nil {
			return toResult(nodeCandidate)
		}
	} else {
		if nodeCandidate != nil {
			return toResult(nodeCandidate)
		}
		if bunCandidate != nil {
			return toResult(bunCandidate)
		}
	}

	return nil
}
