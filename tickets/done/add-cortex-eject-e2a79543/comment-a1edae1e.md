---
id: a1edae1e-d038-4111-ad56-8aff7f355d43
author: claude
type: done
created: 2026-02-03T08:39:25.752988Z
---
## Summary

Implemented the `cortex eject` command for copying prompts from global defaults to project-level for customization.

## Changes

**New file: `cmd/cortex/commands/eject.go`** (129 lines)

Added a CLI command that:
- Takes a prompt path argument (e.g., `ticket/work/SYSTEM.md`)
- Loads project config to get the resolved extend path
- Copies the source file from base prompts to `.cortex/prompts/`
- Supports `--force` flag to overwrite existing files
- Provides clear error messages for all edge cases

## Usage

```bash
cortex eject ticket/work/SYSTEM.md
cortex eject architect/KICKOFF.md --force
```

## Verification

- `make build` - Passes
- `make lint` - Passes
- Command help and error handling verified

## Git

- Committed: `17c4c9a` - feat(cli): add cortex eject command for prompt customization
- Merged to main and pushed to origin