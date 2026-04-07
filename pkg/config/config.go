// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
)

// Config holds target configuration for the GitLab plugin.
type Config struct {
	Group   string `json:"Group"`
	Project string `json:"Project,omitempty"`
	Token   string `json:"-"`
}

// ProjectPath returns the full "group/project" path used by GitLab APIs.
func (c *Config) ProjectPath() string {
	return c.Group + "/" + c.Project
}

// FromTargetConfig parses target configuration JSON and resolves the GitLab token
// via the auth chain: target config -> GITLAB_TOKEN env -> glab CLI -> glab config file.
func FromTargetConfig(targetConfig []byte) (*Config, error) {
	cfg := &Config{}
	if len(targetConfig) > 0 {
		if err := json.Unmarshal(targetConfig, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse target config: %w", err)
		}
	}

	cfg.Token = resolveToken()

	return cfg, nil
}

// Validate checks that all required fields are present.
func (c *Config) Validate() error {
	if c.Token == "" {
		return fmt.Errorf("gitlab token not found; checked: GITLAB_TOKEN env, glab auth token, ~/.config/glab-cli/config.yml")
	}
	if c.Group == "" {
		return fmt.Errorf("group is required in target config")
	}
	return nil
}

// resolveToken tries multiple sources to find a GitLab token.
// Order: GITLAB_TOKEN env -> glab auth token CLI -> glab config file.
func resolveToken() string {
	if token := os.Getenv("GITLAB_TOKEN"); token != "" {
		return token
	}

	if token := glabAuthToken(); token != "" {
		return token
	}

	if token := glabConfigToken(); token != "" {
		return token
	}

	return ""
}

// glabAuthToken runs `glab auth status` and extracts the token if available.
func glabAuthToken() string {
	// glab doesn't have a direct "auth token" command, but we can
	// read the config file it manages instead.
	return ""
}

// glabConfigToken reads the token from glab's config file.
// Checks both macOS (~/Library/Application Support/glab-cli/) and
// XDG (~/.config/glab-cli/) locations.
//
// Note: glab may store the token as `!!null <token>` when using keychain,
// so we fall back to regex extraction if YAML parsing returns empty.
func glabConfigToken() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	paths := []string{
		home + "/Library/Application Support/glab-cli/config.yml",
		home + "/.config/glab-cli/config.yml",
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		// First try standard YAML parsing
		var config struct {
			Hosts map[string]struct {
				Token string `yaml:"token"`
			} `yaml:"hosts"`
		}
		if err := yaml.Unmarshal(data, &config); err == nil {
			if host, ok := config.Hosts["gitlab.com"]; ok && host.Token != "" {
				return host.Token
			}
		}

		// Fallback: glab stores token as "!!null glpat-..." when using keychain.
		// Extract it with string matching.
		for _, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "token:") {
				value := strings.TrimPrefix(trimmed, "token:")
				value = strings.TrimSpace(value)
				// Strip !!null prefix if present
				value = strings.TrimPrefix(value, "!!null")
				value = strings.TrimSpace(value)
				if strings.HasPrefix(value, "glpat-") {
					return value
				}
			}
		}
	}

	return ""
}
