---
id: fc552093-04ff-4998-bbf9-11913410558b
author: claude
type: done
created: 2026-02-11T10:42:53.520679Z
---
## Summary

Added OpenCode agent defaults so that `cortex init` with OpenCode produces a working setup out of the box.

### Changes

**Created 16 files** under `internal/install/defaults/opencode/`:

- **`cortex.yaml`** — Uses `agent: opencode` for all roles (architect, meta, ticket/work/debug/research/chore) with empty args. OpenCode handles permissions via `OPENCODE_CONFIG_CONTENT` env var rather than CLI flags.
- **`CONFIG_DOCS.md`** — OpenCode-specific configuration guide explaining the empty-args approach and prompt customization.
- **14 prompt files** — Copied verbatim from `claude-code` defaults (architect, meta, and all 4 ticket types with SYSTEM/KICKOFF/APPROVE prompts).

**Updated documentation:**
- `README.md` — Added `opencode` to the agent options list
- `CLAUDE.md` — Added `opencode` to the agent defaults path listing

### No code changes required
The existing `//go:embed defaults/*` directive, `AgentOpenCode` constant, and launcher logic already support opencode — only the default files were missing.

### Verification
- `make build` — succeeds
- `make test` — all tests pass
- `make lint` — 0 issues (via pre-push hook)