package checkly

import (
	_ "embed"
	"encoding/json"
	"fmt"
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
	Default string       `json:"default"`
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
