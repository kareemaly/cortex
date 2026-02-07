---
id: c3eeed19-5d58-4d32-b567-b4b7169c41a7
author: claude
type: review_requested
created: 2026-02-04T13:00:56.315292Z
action:
    type: git_diff
    args:
        commit: 66f5ae2
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/f1d25f03-926a-4371-960f-06e4b0a6593f
---
## Summary

Implemented install script and fixed GoReleaser configuration for GitHub releases.

### Changes

**`.goreleaser.yaml`** - Fixed bugs and configured for raw binary releases:
- Fixed module path: `cortex1` → `cortex`
- Fixed ldflags variable: `.Date` → `.BuildDate` (matches `pkg/version/version.go`)
- Changed archive format from `tar.gz` to `binary` (raw binaries)
- Added OS/arch to binary names: `cortex-{{ .Os }}-{{ .Arch }}`
- Added `release.extra_files` to include `install.sh` in releases

**`install.sh`** - Created new installer script with:
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

**`Makefile`** - Added `release-build` target:
- Cross-compiles for all 4 platform combinations (darwin/linux × amd64/arm64)
- Outputs to `dist/` directory
- Generates `checksums.txt` with SHA256 hashes
- Uses `-s -w` ldflags for smaller binaries

### Testing

1. `./install.sh --help` - Works correctly
2. `make release-build` - Successfully builds all 8 binaries and generates checksums
3. `./dist/cortex-darwin-arm64 version` - Built binaries work correctly

### Release Artifacts

After this change, each GitHub release will contain:
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

### Usage

```bash
# Latest version
curl -fsSL https://raw.githubusercontent.com/kareemaly/cortex/main/install.sh | bash

# Specific version
curl -fsSL https://raw.githubusercontent.com/kareemaly/cortex/main/install.sh | bash -s -- -v v1.0.0
```