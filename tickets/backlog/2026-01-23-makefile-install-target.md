# Makefile Install Target

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
