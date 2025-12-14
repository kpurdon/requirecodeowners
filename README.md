# requirecodeowners

A GitHub Action that validates specified directories have corresponding CODEOWNERS entries.

## Configuration

Create `.requirecodeowners.yml` in your repository root:

```yaml
directories:
  - path: services
    level: 1        # check immediate subdirectories
  - path: libs
    level: 2        # check two levels deep
  - path: src       # level defaults to 0 (check directory itself)
```

### Level explained

- `level: 0` (default) - check that the directory itself has a CODEOWNERS entry
- `level: 1` - check that each immediate subdirectory has a CODEOWNERS entry
- `level: 2` - check subdirectories two levels deep, etc.

## Usage

```yaml
- uses: kpurdon/requirecodeowners@v1
```

The action reads `.requirecodeowners.yml` from your repository root.

### Action inputs

| Name | Required | Default | Description |
|------|----------|---------|-------------|
| `config` | No | `.requirecodeowners.yml` | Path to config file |
| `codeowners-path` | No | auto-detected | Path to CODEOWNERS file |
| `version` | No | action version | Version of requirecodeowners to use |

## Example workflow

```yaml
name: Validate CODEOWNERS

on:
  pull_request:
    paths:
      - "CODEOWNERS"
      - ".github/CODEOWNERS"
      - ".requirecodeowners.yml"

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: kpurdon/requirecodeowners@v1
```

## CLI Usage

```bash
# Uses .requirecodeowners.yml in current directory
requirecodeowners

# Use custom config location
requirecodeowners --config path/to/config.yml
```
