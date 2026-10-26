package checkly

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"
)

//go:embed engines.json
var enginesJSON []byte

type engineRule struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Action string `json:"action"`
	Follow bool   `json:"follow"`
	Notice string `json:"notice"`
}

type engineVersionConfig struct {
	// Default applies to a version that semver cannot parse or that matches no
	// rule. Explicit versions are format-validated and always match a rule, so
	// only auto-detection reaches it; it must be a pinned version, not an alias.
	// This is why bun's default here is a pinned minor while checkly-cli's is
	// the "1" alias.
	Default string `json:"default"`
	// Aliases are version strings the API accepts as-is that semver would
	// misread, such as bun "1" (the newest Bun 1.x the runner provides), which
	// would otherwise parse as 1.0.0. They are accepted before any rule runs.
	Aliases []string     `json:"aliases"`
	Rules   []engineRule `json:"rules"`
}

type engineEntry struct {
	Name     string              `json:"name"`
	Versions engineVersionConfig `json:"versions"`
}

type enginesFile struct {
	Engines []engineEntry `json:"engines"`
}

var engineConfigs map[string]engineVersionConfig

func init() {
	var f enginesFile
	if err := json.Unmarshal(enginesJSON, &f); err != nil {
		panic(fmt.Sprintf("failed to parse embedded engines.json: %v", err))
	}
	engineConfigs = make(map[string]engineVersionConfig, len(f.Engines))
	for _, e := range f.Engines {
		engineConfigs[e.Name] = e.Versions
	}
}

// EngineResolution is the result of resolving an engine version through the rules.
type EngineResolution struct {
	Version string
	Notices []string
	Denied  bool
}

// resolveEngineVersion resolves a version string through the engine rules.
// It finds the first matching rule, applies the target, and follows the chain
// if follow is true. Returns the default version if no rule matches.
func resolveEngineVersion(version string, config engineVersionConfig) EngineResolution {
	var notices []string
	seen := make(map[string]bool)
	current := version

	for {
		if seen[current] {
			break
		}
		seen[current] = true

		if slices.Contains(config.Aliases, current) {
			return EngineResolution{Version: current, Notices: notices}
		}

		sv, err := semver.NewVersion(current)
		if err != nil {
			return EngineResolution{Version: config.Default, Notices: notices}
		}

		matched := false
		for _, rule := range config.Rules {
			c, err := semver.NewConstraint(rule.Source)
			if err != nil {
				continue
			}
			if !c.Check(sv) {
				continue
			}
			matched = true

			if rule.Notice != "" {
				notice := strings.ReplaceAll(rule.Notice, "${SOURCE}", current)
				notices = append(notices, notice)
			}

			if rule.Action == "deny" {
				return EngineResolution{Denied: true, Notices: notices}
			}

			next := current
			if rule.Target != "" {
				next = rule.Target
			}

			if !rule.Follow || next == current {
				return EngineResolution{Version: next, Notices: notices}
			}

			current = next
			break
		}

		if !matched {
			return EngineResolution{Version: config.Default, Notices: notices}
		}
	}

	return EngineResolution{Version: current, Notices: notices}
}
