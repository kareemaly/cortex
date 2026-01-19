# CI/CD Pipeline

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
