---
id: 7dd2d881-6b09-426f-b0d4-47659f409dc8
title: Install script and GitHub releases workflow
type: work
created: 2026-02-04T12:47:58.978534Z
updated: 2026-02-04T13:02:42.593994Z
---
## Goal

Enable one-command installation of cortex via curl and set up automated GitHub releases.

## Requirements

### Install script (`install.sh`)

Create a bash script that:
- Detects OS (darwin/linux) and arch (amd64/arm64)
- Downloads correct binaries from GitHub releases
- Verifies SHA256 checksums
- Installs to `/usr/local/bin/` (with sudo) or `~/.local/bin/` (without)
- Makes binaries executable
- Runs `cortex version` to verify installation
- Provides clear progress and error messages

Usage:
```bash
# Latest version
curl -sSL https://github.com/kareemaly/cortex/releases/latest/download/install.sh | bash

# Specific version
curl -sSL https://github.com/kareemaly/cortex/releases/download/v1.0.0/install.sh | bash
```

### GitHub Actions release workflow

Create `.github/workflows/release.yml` that triggers on version tags (`v*`):
1. Build binaries for all platforms (darwin/linux × amd64/arm64)
2. Generate SHA256 checksums file
3. Create GitHub release with:
   - All binary artifacts
   - `checksums.txt`
   - `install.sh`
   - Release notes from tag or CHANGELOG

### Makefile updates

- Add `make release-build` target for cross-compilation
- Ensure version info is embedded at build time (ldflags)

## Artifacts per release

```
cortex-darwin-amd64
cortex-darwin-arm64
cortex-linux-amd64
cortex-linux-arm64
cortexd-darwin-amd64
cortexd-darwin-arm64
cortexd-linux-amd64
cortexd-linux-arm64
checksums.txt
install.sh
```

## Notes

- Script should be idempotent (safe to re-run)
- Consider adding `--version` flag to install.sh for specific versions
- install.sh should be included in the repo (not just releases) for development