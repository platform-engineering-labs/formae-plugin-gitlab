# GitLab Plugin for Formae

[![CI](https://github.com/platform-engineering-labs/formae-plugin-gitlab/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/platform-engineering-labs/formae-plugin-gitlab/actions/workflows/ci.yml)
[![Nightly](https://github.com/platform-engineering-labs/formae-plugin-gitlab/actions/workflows/nightly.yml/badge.svg?branch=main)](https://github.com/platform-engineering-labs/formae-plugin-gitlab/actions/workflows/nightly.yml)

Formae plugin for managing GitLab CI/CD resources.

## Supported Resources

| Resource Type | Description |
|---------------|-------------|
| `GitLab::Project::Variable` | CI/CD variables (with masked/protected support) |
| `GitLab::Project::File` | Repository files (`.gitlab-ci.yml`, etc.) |
| `GitLab::Project::Environment` | Deployment environments |
| `GitLab::Project::Pipeline` | `.gitlab-ci.yml` pipeline declared as a typed Pkl resource |

## Installation

```bash
make install
```

## Configuration

Configure a GitLab target in your Forma file:

```pkl
new formae.Target {
    label = "my-gitlab-target"
    namespace = "GitLab"
    config = new gitlab.Config {
        group = "my-group"
        project = "my-project"
    }
}
```

Authentication uses the following chain (in order):
- `GITLAB_TOKEN` environment variable
- `glab` CLI config file (`~/Library/Application Support/glab-cli/config.yml` or `~/.config/glab-cli/config.yml`)

## Examples

See [examples/](examples/) for usage patterns:

- `smoke-test.pkl` - Simple variable creation
- `infra-to-app/` - Full CI/CD pipeline with Azure credentials, environments, and deploy/destroy stages

## Development

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

### Clean Environment

```bash
GITLAB_TEST_GROUP=my-group GITLAB_TEST_PROJECT=my-test-project ./scripts/ci/clean-environment.sh
```

## License

FSL-1.1-ALv2
