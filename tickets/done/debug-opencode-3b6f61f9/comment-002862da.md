---
id: 002862da-9759-4110-ba2d-30d26e410967
author: claude
type: done
created: 2026-02-12T11:43:08.119182Z
---
## Root Cause

The opencode defaults `cortex.yaml` was copied from the claude-code defaults with only the `agent:` field changed to `opencode`. All Claude Code-specific CLI flags (`--allow-dangerously-skip-permissions`, `--allowedTools`, `--permission-mode`) were left in place. OpenCode doesn't recognize these flags, so it printed its help/usage text and exited immediately. This affected all opencode session types (architect, meta, ticket).

## Resolution

**Commit**: `8fa68a4` — `fix: remove Claude-specific CLI args from opencode defaults`

### Files Changed

1. **`internal/install/defaults/opencode/cortex.yaml`** — Removed all `args:` entries from every role (architect, meta, ticket work/debug/research/chore). OpenCode permissions are already configured via `OPENCODE_CONFIG_CONTENT` env var which sets `"permission": {"*": "allow"}`.

2. **`internal/install/defaults/opencode/CONFIG_DOCS.md`** — Removed Claude-specific args from the example config and removed the "Restrict Agent Permissions" section which showed invalid `--allowedTools` usage.

## Verification

- `make test` — all tests pass
- `make lint` — 0 issues
- `make install` + `cortex defaults upgrade --yes` — confirmed `~/.cortex/defaults/opencode/` files are correct with no Claude-specific args