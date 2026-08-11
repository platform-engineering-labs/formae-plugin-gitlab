# Changelog

All notable changes to the formae GitLab plugin are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Install with `sudo formae plugin install gitlab` on the host that runs the
formae agent.

## [0.1.2]

### Added

- The GitLab API token can now be set in the target config, and the field
  accepts a resolvable, so it can be sourced from a formae-managed secret. The
  agent resolves it live before every call, so onboarding or rotating a token
  needs no agent restart. A declared token is used as given; omit it to keep
  using `GITLAB_TOKEN` or the glab CLI configuration.
- `GitLab::Project::Variable` now exposes its value as a first-class secret.
  Other resources reference it with `myVariable.res.secretValue`, which resolves
  live and is stored as a reference rather than a copy of the value.

### Changed

- `GitLab::Project::Variable.value` is now opaque: it is stored hashed and
  redacted in output, closing a leak where the value GitLab returns on read was
  persisted in cleartext. This applies to every variable, including ones holding
  ordinary configuration. Their values are no longer readable back out of
  formae, and an out-of-band change is still reported as drift but can no longer
  be absorbed into code as a readable value.
- Requires formae 0.89.0 or later, and the schema is built against the
  `formae@0.89.0` package.

## [0.1.1]

### Changed

- Now available on the platform.engineering Hub. Install with
  `sudo formae plugin install gitlab` on the host that runs the formae agent.

## [0.1.0]

### Added

- Initial release of the GitLab plugin as a standalone package built on the
  formae Plugin SDK.
