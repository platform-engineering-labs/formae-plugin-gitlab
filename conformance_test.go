// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: Apache-2.0

//go:build conformance

package main

import (
	"testing"

	conformance "github.com/platform-engineering-labs/formae/pkg/plugin-conformance-tests"
)

func TestPluginConformance(t *testing.T) {
	conformance.RunCRUDTests(t)
}

func TestPluginDiscovery(t *testing.T) {
	conformance.RunDiscoveryTests(t)
}
