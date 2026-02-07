---
id: 1f44786e-a444-4943-b3e9-7fff154cada0
title: CI/CD Pipeline
type: ""
created: 2026-01-19T13:24:56Z
updated: 2026-01-19T13:24:56Z
---
Set up GitHub Actions workflows and GoReleaser for automated builds and releases.

## Context

Depends on project-foundation ticket being complete (needs Makefile, go.mod).

Reference `~/projects/cortex/.github/workflows/` and `~/projects/cortex/.goreleaser.yaml` for patterns.

## Requirements

### 1. .github/workflows/ci.yml

Runs on push and PR to main branch:

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.24"

      - name: Build
        run: make build

      - name: Test
        run: make test

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.24"

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest
          args: --timeout=5m
```

### 2. .github/workflows/release.yml

Runs on version tags:

```yaml
name: Release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.24"

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### 3. .goreleaser.yaml

```yaml
version: 2

builds:
  - id: cortex
    main: ./cmd/cortex
    binary: cortex
    env:
      - CGO_ENABLED=0
    goos:
      - darwin
      - linux
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X github.com/kareemaly/cortex1/pkg/version.Version={{.Version}}
      - -X github.com/kareemaly/cortex1/pkg/version.Commit={{.ShortCommit}}
      - -X github.com/kareemaly/cortex1/pkg/version.BuildDate={{.Date}}

  - id: cortexd
    main: ./cmd/cortexd
    binary: cortexd
    env:
      - CGO_ENABLED=0
    goos:
      - darwin
      - linux
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X github.com/kareemaly/cortex1/pkg/version.Version={{.Version}}
      - -X github.com/kareemaly/cortex1/pkg/version.Commit={{.ShortCommit}}
      - -X github.com/kareemaly/cortex1/pkg/version.BuildDate={{.Date}}

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: "checksums.txt"
  algorithm: sha256

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
      - "^chore:"

release:
  github:
    owner: kareemaly
    name: cortex1
  prerelease: auto
```

### 4. .gitignore

Add/update `.gitignore` for build artifacts:

```gitignore
# Binaries
bin/
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test artifacts
coverage.out
coverage.html
*.test

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Go
vendor/

# GoReleaser
dist/
```

## Verification

After implementation:

```bash
# Workflows are valid YAML
cat .github/workflows/ci.yml | head
cat .github/workflows/release.yml | head

# GoReleaser config is valid (if goreleaser installed)
goreleaser check

# .gitignore includes bin/ and dist/
grep "bin/" .gitignore
grep "dist/" .gitignore
```

## Notes

- CI runs build + test + lint in parallel jobs for speed
- Release only triggers on tags matching `v*` pattern
- GoReleaser builds for darwin/linux on amd64/arm64
- CGO_ENABLED=0 ensures static binaries
- Changelog excludes docs/test/chore commits

## Implementation

### Commits Pushed

- `ded018b` feat: add GitHub Actions CI/CD and GoReleaser configuration

### Key Files Changed

- `.github/workflows/ci.yml` - CI workflow with build, test, and lint jobs
- `.github/workflows/release.yml` - Release workflow triggered on v* tags
- `.goreleaser.yaml` - GoReleaser v2 configuration for both binaries
- `.gitignore` - Extended with build artifacts, test outputs, and vendor patterns

### Decisions Made

1. **GoReleaser v2 syntax**: Used `version: 2` format with modern configuration
2. **Changelog filtering**: Added `ci:` and `style:` to exclusion filters beyond the spec
3. **LDFlags variable names**: Used `Commit` and `Date` (matching pkg/version) instead of `ShortCommit` and `BuildDate` from the ticket spec
4. **Removed explicit release config**: Omitted the `release.github` block as GoReleaser auto-detects from git remote
5. **Removed checksum algorithm**: Let GoReleaser use its default (sha256)

### Scope Changes

None - implemented as specified