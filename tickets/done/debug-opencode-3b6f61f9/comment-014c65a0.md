---
id: 014c65a0-8ff9-4b71-8856-66f2a15a72de
author: claude
type: review_requested
created: 2026-02-12T11:42:23.763533Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
        commit: 8fa68a4
---
## Root Cause
The opencode defaults `cortex.yaml` was copied from claude-code defaults with only the `agent:` field changed. Claude Code-specific CLI flags (`--allow-dangerously-skip-permissions`, `--allowedTools`, `--permission-mode`) were left in place. OpenCode doesn't recognize these flags, so it prints help/usage and exits immediately.

## Changes
1. **`internal/install/defaults/opencode/cortex.yaml`** — Removed all `args:` entries from every role (architect, meta, ticket work/debug/research/chore). OpenCode permissions are already handled via `OPENCODE_CONFIG_CONTENT` env var.
2. **`internal/install/defaults/opencode/CONFIG_DOCS.md`** — Removed Claude-specific args from the example config and removed the "Restrict Agent Permissions" section which showed invalid `--allowedTools` usage.

## Verification
- `make test` — all tests pass
- `make lint` — 0 issues