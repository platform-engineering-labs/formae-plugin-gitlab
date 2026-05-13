# Contributing

This document covers local development for plugin authors. For user-facing
plugin docs (configuration, supported resources, examples), see
[README.md](README.md).

## Prerequisites

- Go 1.25+
- [Pkl CLI](https://pkl-lang.org/main/current/pkl-cli/index.html) 0.30+
- GitLab access token (for integration/conformance testing)

## Local Installation

```bash
# Install the plugin
make install
```

## Building

```bash
make build      # Build plugin binary
make test-unit  # Run unit tests
make lint       # Run linter
make install    # Build + install locally
```

## Local Testing

```bash
# Install plugin locally
make install

# Start formae agent
formae agent start

# Apply example resources
formae apply --mode reconcile --watch examples/basic/main.pkl
```

## Credentials Setup for Testing

Integration and conformance tests require a GitLab access token plus a test
group/project:

```bash
export GITLAB_TOKEN=...           # PAT with api scope
export GITLAB_TEST_GROUP=...      # group used for test resources
export GITLAB_TEST_PROJECT=...    # project used for test resources

# Run conformance tests
make conformance-test
```

## Conformance Testing

Run the full CRUD lifecycle + discovery tests:

```bash
make conformance-test  # Latest formae version
```

The `scripts/ci/clean-environment.sh` script cleans up test resources. It runs
before and after conformance tests and is idempotent.
