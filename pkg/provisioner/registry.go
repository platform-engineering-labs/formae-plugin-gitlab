// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: Apache-2.0

package provisioner

import (
	"fmt"
	"sync"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/platform-engineering-labs/formae-plugin-gitlab/pkg/config"
)

// Factory creates a Provisioner given an authenticated GitLab client and config.
type Factory func(client *gitlab.Client, cfg *config.Config) Provisioner

var (
	mu        sync.RWMutex
	factories = make(map[string]Factory)
)

// Register associates a resource type with its provisioner factory.
// Called from init() functions in resource packages.
func Register(resourceType string, factory Factory) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := factories[resourceType]; exists {
		panic(fmt.Sprintf("provisioner already registered for %s", resourceType))
	}
	factories[resourceType] = factory
}

// Get returns the factory for a resource type, or false if not registered.
func Get(resourceType string) (Factory, bool) {
	mu.RLock()
	defer mu.RUnlock()
	f, ok := factories[resourceType]
	return f, ok
}
