// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package config

import (
	"testing"
)

// isolateAmbientCredentials points HOME at an empty directory and clears
// GITLAB_TOKEN, so a test observes only the sources it sets up itself.
func isolateAmbientCredentials(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GITLAB_TOKEN", "")
}

func TestTargetConfigTokenTakesPrecedenceOverEnvironment(t *testing.T) {
	isolateAmbientCredentials(t)
	t.Setenv("GITLAB_TOKEN", "glpat-from-environment")

	cfg, err := FromTargetConfig([]byte(`{"Group":"acme","Token":"glpat-from-target-config"}`))
	if err != nil {
		t.Fatalf("FromTargetConfig: %v", err)
	}

	if cfg.Token != "glpat-from-target-config" {
		t.Errorf("token = %q, want the target config value", cfg.Token)
	}
}

func TestTokenFallsBackToEnvironmentWhenTargetConfigOmitsIt(t *testing.T) {
	isolateAmbientCredentials(t)
	t.Setenv("GITLAB_TOKEN", "glpat-from-environment")

	cfg, err := FromTargetConfig([]byte(`{"Group":"acme"}`))
	if err != nil {
		t.Fatalf("FromTargetConfig: %v", err)
	}

	if cfg.Token != "glpat-from-environment" {
		t.Errorf("token = %q, want the environment value", cfg.Token)
	}
}

// A declared token that resolves to nothing is a misconfiguration, not an
// invitation to authenticate as whoever the ambient chain happens to name.
func TestDeclaredButEmptyTokenDoesNotFallBackToEnvironment(t *testing.T) {
	isolateAmbientCredentials(t)
	t.Setenv("GITLAB_TOKEN", "glpat-from-environment")

	cfg, err := FromTargetConfig([]byte(`{"Group":"acme","Token":""}`))
	if err != nil {
		t.Fatalf("FromTargetConfig: %v", err)
	}

	if cfg.Token != "" {
		t.Fatalf("token = %q, want no fall back to the environment", cfg.Token)
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate accepted a declared but empty token")
	}
}

func TestValidateRejectsMissingTokenFromEverySource(t *testing.T) {
	isolateAmbientCredentials(t)

	cfg, err := FromTargetConfig([]byte(`{"Group":"acme"}`))
	if err != nil {
		t.Fatalf("FromTargetConfig: %v", err)
	}

	if err := cfg.Validate(); err == nil {
		t.Error("Validate accepted a config with no token from any source")
	}
}

func TestValidateAcceptsATokenFromTheTargetConfig(t *testing.T) {
	isolateAmbientCredentials(t)

	cfg, err := FromTargetConfig([]byte(`{"Group":"acme","Token":"glpat-from-target-config"}`))
	if err != nil {
		t.Fatalf("FromTargetConfig: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}
