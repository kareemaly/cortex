---
id: 482098f8-017d-48b9-8a71-fc2ce6b9b51a
author: claude
type: done
created: 2026-02-04T13:02:42.592796Z
---
## Summary

Implemented install script and fixed GoReleaser configuration to enable one-command installation of cortex.

### Files Changed

**`.goreleaser.yaml`**
- Fixed module path: `github.com/kareemaly/cortex1` → `github.com/kareemaly/cortex`
- Fixed ldflags variable: `.Date` → `.BuildDate` (matches `pkg/version/version.go:12`)
- Changed archive format from `tar.gz` to `binary` (raw binaries, not archives)
- Added OS/arch to binary names: `cortex-{{ .Os }}-{{ .Arch }}`
- Added `release.extra_files` to include `install.sh` in GitHub releases

**`install.sh`** (new file)
- OS detection (darwin/linux)
- Arch detection (amd64/arm64 including aliases x86_64/aarch64)
- Downloads binaries from GitHub releases
- SHA256 checksum verification using checksums.txt
- Installs to `/usr/local/bin` (with sudo) or `~/.local/bin` (without)
- macOS code signing
- Version verification via `cortex version`
- `-v/--version` flag for specific versions
- `-d/--dir` flag for custom install directory
- Colored progress output

**`Makefile`**
- Added `release-build` target for local cross-compilation
- Builds all 4 platform combinations (darwin/linux × amd64/arm64)
- Generates checksums.txt with SHA256 hashes

### Usage

```bash
# Latest version
curl -fsSL https://raw.githubusercontent.com/kareemaly/cortex/main/install.sh | bash

# Specific version
curl -fsSL https://raw.githubusercontent.com/kareemaly/cortex/main/install.sh | bash -s -- -v v1.0.0
```

### Release Artifacts

Each GitHub release will now contain:
- `cortex-darwin-amd64`, `cortex-darwin-arm64`, `cortex-linux-amd64`, `cortex-linux-arm64`
- `cortexd-darwin-amd64`, `cortexd-darwin-arm64`, `cortexd-linux-amd64`, `cortexd-linux-arm64`
- `checksums.txt`
- `install.sh`