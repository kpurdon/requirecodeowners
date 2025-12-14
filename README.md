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

### Level explained

| Level | Behavior | Example |
|-------|----------|---------|
| `0` (default) | Check the directory itself | `libs/` must have an entry |
| `1` | Check immediate subdirectories | Each `services/*/` must have an entry |
| `2` | Check two levels deep | Each `services/*/*/` must have an entry |

### Full example

```yaml
directories:
  - path: services
    level: 1        # services/auth/, services/api/, etc.
  - path: libs
    level: 2        # libs/go/utils/, libs/js/common/, etc.
  - path: docs      # docs/ itself (level defaults to 0)
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
