# requirecodeowners

A GitHub Action and CLI tool that ensures directories have CODEOWNERS coverage.

## Why?

As codebases grow, it's easy to add new directories without updating CODEOWNERS. This tool catches those gaps in CI, ensuring every important directory has clear ownership.

## Quick Start

1. Create `.requirecodeowners.yml` in your repository:

```yaml
directories:
  - path: services
    level: 1    # Check each subdirectory of services/
  - path: libs  # Check libs/ itself (level: 0 is default)
```

2. Add the GitHub Action:

```yaml
- uses: kpurdon/requirecodeowners@v1
```

That's it! The action will fail if any configured directories lack CODEOWNERS entries.

## Configuration

### Match mode

Controls how CODEOWNERS coverage is checked for each directory:

| Mode | Behavior |
|------|----------|
| `exact` (default) | Directory must have its own CODEOWNERS rule. Inheritance from a parent rule is not sufficient. |
| `coverage` | Directory only needs to be covered by any CODEOWNERS rule, including inherited parent rules. |

```yaml
directories:
  - path: services
    level: 1
    # Default: each services/* must have its own CODEOWNERS entry
  - path: internal
    level: 1
    match: coverage  # Opt out: parent coverage is acceptable
```

For example, if CODEOWNERS contains `/apps/ @team-apps`, a subdirectory `apps/my-service/` is covered by inheritance. With the default `exact` mode, this would fail because `apps/my-service/` lacks its own entry. With `coverage` mode, it would pass.

### Level explained

| Level | Behavior | Example |
|-------|----------|---------|
| `0` (default) | Check the directory itself | `libs/` must have an entry |
| `1` | Check immediate subdirectories | Each `services/*/` must have an entry |
| `2` | Check two levels deep | Each `services/*/*/` must have an entry |

### Glob patterns

Paths support glob patterns using `*`:

```yaml
directories:
  - path: applications/*/services
    level: 1
```

This matches `applications/a/services`, `applications/b/services`, etc., and checks that each of their subdirectories has CODEOWNERS coverage.

### Excluding directories (opt-out)

Sometimes a directory intentionally has no owner — deprecated code, vendored dependencies, generated output. Add it to a top-level `exclude:` list to opt it out of the check. Each exclude needs a `path`, a `reason`, and an `owner`:

```yaml
directories:
  - path: services
    level: 1
exclude:
  - path: services/legacy
    reason: "Deprecated, removal tracked in JIRA-123"
    owner: "@team-platform"
  - path: vendor
    reason: "Third-party code, no internal ownership"
    owner: "@alice"
```

`reason` and `owner` are both required. They keep opt-outs honest: every exclusion is documented and attributed in the config, visible in review, and printed in the CI logs. The `owner` is any non-empty string — a GitHub user or team is the natural choice. When a `path` is a glob, the log also lists the concrete directories it resolved to, so a too-broad pattern is visible at a glance.

Exclusion is recursive — opting out `services/legacy` also skips everything beneath it. Paths accept the same glob syntax as `directories`. An exclude that matches no directory fails the check, so stale opt-outs don't pile up after the directory is gone.

### Full example

```yaml
directories:
  - path: services
    level: 1           # services/auth/, services/api/, etc. must each have an exact CODEOWNERS entry
  - path: libs
    level: 2           # libs/go/utils/, libs/js/common/, etc.
  - path: internal
    level: 1
    match: coverage    # parent CODEOWNERS coverage is acceptable
  - path: docs         # docs/ itself (level defaults to 0)
```

## GitHub Action

### Basic usage

```yaml
name: Require CODEOWNERS

on:
  pull_request:

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: kpurdon/requirecodeowners@v1
```

### Inputs

| Name | Required | Default | Description |
|------|----------|---------|-------------|
| `config` | No | `.requirecodeowners.yml` | Path to config file |
| `codeowners-path` | No | auto-detected | Path to CODEOWNERS file |
| `version` | No | `latest` | CLI version to use |

### Output

When directories are missing CODEOWNERS coverage, you'll see clear error messages:

```
  ✗ services/new-api
    Not covered by CODEOWNERS. Add: /services/new-api/ @your-team

✗ 1 directory failed CODEOWNERS check
```

GitHub Actions also displays a summary table for easy scanning.

## CLI Usage

Install from [releases](https://github.com/kpurdon/requirecodeowners/releases) or use `go install`:

```bash
go install github.com/kpurdon/requirecodeowners@latest
```

Run in any repository with a `.requirecodeowners.yml`:

```bash
requirecodeowners
requirecodeowners --config path/to/config.yml
requirecodeowners --codeowners-path .github/CODEOWNERS
```

## Example Repository

See [kpurdon/requirecodeowners-example](https://github.com/kpurdon/requirecodeowners-example) for a complete working example demonstrating various failure modes.
