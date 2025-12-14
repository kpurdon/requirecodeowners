# requirecodeowners

A GitHub Action that validates specified directories have corresponding CODEOWNERS entries.

## Usage

```yaml
- uses: kpurdon/requirecodeowners@v1
  with:
    directories: |
      src/
      pkg/
      internal/
```

### Check subdirectories

Use `level` to check subdirectories instead of the directory itself:

```yaml
# Given services/foo/, services/bar/, services/baz/
# This ensures each subdirectory has a CODEOWNERS entry
- uses: kpurdon/requirecodeowners@v1
  with:
    directories: services
    level: 1
```

## Inputs

| Name | Required | Default | Description |
|------|----------|---------|-------------|
| `directories` | Yes | | Newline-separated list of directories to validate |
| `level` | No | `0` | Directory depth to check (0=directory itself, 1=immediate subdirs, etc.) |
| `codeowners-path` | No | | Path to CODEOWNERS file (auto-detected from `.github/CODEOWNERS`, `CODEOWNERS`, or `docs/CODEOWNERS`) |
| `version` | No | | Version of requirecodeowner to use (defaults to action version) |

## What it checks

1. Each specified directory (or subdirectory at the given level) exists
2. Each directory has a CODEOWNERS entry that covers it

## Example workflow

```yaml
name: Validate CODEOWNERS

on:
  pull_request:
    paths:
      - "CODEOWNERS"
      - ".github/CODEOWNERS"

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: kpurdon/requirecodeowners@v1
        with:
          directories: services
          level: 1
```

## CLI Usage

```bash
# Check directory itself
requirecodeowners --directories="src,pkg"

# Check immediate subdirectories
requirecodeowners --directories="services" --level=1
```
