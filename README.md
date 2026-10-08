# GitLab plugin for formae

[![CI](https://github.com/platform-engineering-labs/formae-plugin-gitlab/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/platform-engineering-labs/formae-plugin-gitlab/actions/workflows/ci.yml)
[![Monthly](https://github.com/platform-engineering-labs/formae-plugin-gitlab/actions/workflows/monthly.yml/badge.svg?branch=main)](https://github.com/platform-engineering-labs/formae-plugin-gitlab/actions/workflows/monthly.yml)

Manages GitLab CI/CD project variables, files, environments and pipelines as Infrastructure As Code with [formae](https://github.com/platform-engineering-labs/formae).

[formae](https://github.com/platform-engineering-labs/formae) · [Hub](https://hub.platform.engineering/platform.engineering/gitlab)

## Install

Requires the formae CLI: see the [quick start](https://docs.formae.ai/documentation/get-started/quickstart).

```bash
formae plugin install gitlab
```

Restart the formae agent afterwards so it loads the plugin.

**New project:** with the agent running, `formae project init --include gitlab my-project` creates `my-project` with a `PklProject` that declares the formae and gitlab schema packages, so `import "@gitlab/..."` resolves, and a starter `main.pkl`. Don't run it in an existing project: it overwrites both files.

**Existing project:** add the plugin to `dependencies` in your `PklProject`, with the current version from the [hub page](https://hub.platform.engineering/platform.engineering/gitlab), then run `pkl project resolve`:

```pkl
["gitlab"] {
  uri = "package://hub.platform.engineering/plugins/gitlab/schema/pkl/gitlab/gitlab@<version>"
}
```

Next: [write your first forma](https://docs.formae.ai/documentation/get-started/write-your-first-forma), then [`formae apply`](https://docs.formae.ai/documentation/reference/cli/apply) (see [apply modes](https://docs.formae.ai/documentation/concepts/apply-modes)).

With an AI coding assistant, use the [formae plugin](https://docs.formae.ai/documentation/guides/ai-coding-assistants) (formerly `formae-mcp`), which can search the hub and fetch plugin examples. The formae documentation is also available as [llms.txt](https://docs.formae.ai/llms.txt).

## Supported Resources

| Resource Type | Description |
|---------------|-------------|
| `GitLab::Project::Variable` | CI/CD variables (with masked/protected support) |
| `GitLab::Project::File` | Repository files (`.gitlab-ci.yml`, etc.) |
| `GitLab::Project::Environment` | Deployment environments |
| `GitLab::Project::Pipeline` | `.gitlab-ci.yml` pipeline declared as a typed Pkl resource |

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

### Authentication

The API token can come from the target config, which is the recommended form
because the field accepts a resolvable and can therefore be sourced from a
formae-managed secret. The agent resolves it live before every call, so
onboarding or rotating a token needs no agent restart:

```pkl
config = new gitlab.Config {
    group = "my-group"
    project = "my-project"
    token = gitlabToken.res.secretValue
}
```

A declared token is used as given: one that resolves to an empty value is an
error rather than a silent fall back, which would otherwise authenticate as
whoever the ambient chain happens to name.

Omit `token` to use the ambient chain instead, in order:

- `GITLAB_TOKEN` environment variable
- `glab` CLI config file (`~/Library/Application Support/glab-cli/config.yml` or `~/.config/glab-cli/config.yml`)

### Variable values are secrets

`GitLab::Project::Variable.value` is opaque. GitLab returns variable values on
read, so formae stores the value hashed and redacts it in output, whether it was
supplied as a literal, as `formae.value(...).opaque`, or resolved from another
resource. This applies to every variable, including ones holding ordinary
configuration: their values are not readable back out of formae, and an
out-of-band change is reported as drift but cannot be absorbed into code as a
readable value.

Other resources reference a variable's value with `myVariable.res.secretValue`,
which resolves live and is stored as a reference rather than a copy.

## Examples

See [examples/](examples/) for usage patterns:

- `smoke-test.pkl` - Simple variable creation
- `infra-to-app/` - Full CI/CD pipeline with Azure credentials, environments, and deploy/destroy stages

## License

FSL-1.1-ALv2
