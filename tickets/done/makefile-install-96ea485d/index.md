---
id: 96ea485d-b881-4329-a71d-f1765f3be52b
title: Makefile Install Target
type: ""
created: 2026-01-24T08:57:36Z
updated: 2026-01-24T08:57:36Z
---
## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

No single command to clean, build, install, and validate the installation.

## Requirements

Add `make install` target that:
1. Cleans existing installation (removes old binaries from system)
2. Cleans build artifacts
3. Builds fresh binaries
4. Installs to `~/.local/bin/` (with codesign on macOS)
5. Validates installation is correct
6. Outputs versions of installed binaries

## Implementation

### Commits

- `09b60da` feat: add make install target for clean build and validation

### Files Changed

- `Makefile` - Added `install` target and updated `.PHONY`

### Key Decisions

1. **Depends on `clean build`** - Ensures fresh build from clean state every time
2. **Creates `~/.local/bin/` if missing** - Safe for first-time installs
3. **Removes old binaries explicitly** - Clean installation before copy
4. **OS detection via `uname`** - Only codesigns on macOS (Darwin)
5. **Runs version commands** - Validates binaries execute correctly
6. **Uses `@` prefix** - Cleaner output (suppresses command echo)

### Scope Changes

None - implemented as specified