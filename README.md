# requirecodeowner

A GitHub Action that validates specified directories have corresponding CODEOWNERS entries.

## Usage

```yaml
- uses: kpurdon/requirecodeowner@v1
  with:
    directories: |
      src/
      pkg/
      internal/
```

## Inputs

| Name | Required | Description |
|------|----------|-------------|
| `directories` | Yes | Newline-separated list of directories that must have CODEOWNERS entries |
| `codeowners-path` | No | Path to CODEOWNERS file (auto-detected from `.github/CODEOWNERS`, `CODEOWNERS`, or `docs/CODEOWNERS`) |
| `version` | No | Version of requirecodeowner to use (defaults to action version) |

## What it checks

1. Each specified directory exists
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
      - uses: kpurdon/requirecodeowner@v1
        with:
          directories: |
            src/
            pkg/
```

## CLI Usage

The underlying CLI can also be used directly:

```bash
requirecodeowner --directories="src,pkg,internal"
```
