# Contributing

This document covers local development for plugin authors. For user-facing
plugin docs (configuration, supported resources, examples), see
[README.md](README.md).

## Local Installation

```bash
make install
```

## Building & Testing

```bash
make build          # Build plugin
make test           # Run tests
make lint           # Run golangci-lint
make install        # Install locally
make install-dev    # Install as v0.0.0 (for debug builds)
make gen-pkl        # Resolve PKL dependencies
make verify-schema  # Validate Pkl schemas
```

## Conformance Tests

Run against a real GitLab project:

```bash
export GITLAB_TOKEN=glpat-...
export GITLAB_TEST_GROUP=my-group
export GITLAB_TEST_PROJECT=my-test-project
make conformance-test
```

To run conformance against a local formae build (e.g. an unreleased version),
point the harness at the binary:

```bash
export FORMAE_BINARY=/path/to/formae
make conformance-test
```

## Clean Environment

```bash
GITLAB_TEST_GROUP=my-group GITLAB_TEST_PROJECT=my-test-project ./scripts/ci/clean-environment.sh
```
