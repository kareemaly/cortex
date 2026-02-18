---
id: b88a1f25-131c-47d2-8824-53b497733d3d
author: claude
type: comment
created: 2026-02-12T11:42:08.435479Z
---
**Root cause**: The opencode defaults `cortex.yaml` was copied from the claude-code defaults with only the `agent:` field changed to `opencode`. The Claude Code-specific CLI args (`--allow-dangerously-skip-permissions`, `--allowedTools`, `--permission-mode`) were left in place. OpenCode doesn't recognize these flags, so it prints help/usage text and exits immediately.

**Fix**: Removed all `args:` entries from every role in `internal/install/defaults/opencode/cortex.yaml`. OpenCode permissions are already configured via `OPENCODE_CONFIG_CONTENT` env var (which sets `"permission": {"*": "allow"}`), so these args were both invalid and redundant. Also cleaned up `CONFIG_DOCS.md` to remove references to Claude-specific args.