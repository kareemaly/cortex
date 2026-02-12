---
id: 1c8a25fa-5a52-4681-ac2e-1608f106221a
author: claude
type: comment
created: 2026-02-10T09:41:55.739214Z
---
## Investigation Findings

### Root cause confirmed
Tmux's `-t` flag does prefix matching on bare session names. `has-session -t cortex` would incorrectly match `cortex-meta`. The fix is to append a trailing colon (`:`) which forces exact session name resolution.

### Fix already applied
Commit `43778b0` ("fix: append trailing colon to all tmux session targets for exact matching") already fixed all session-level targets in `internal/tmux/`:

| Function | File | Line | Target |
|----------|------|------|--------|
| `SessionExists` | session.go:17 | `name+":"` | ✅ |
| `KillSession` | session.go:63 | `name+":"` | ✅ |
| `AttachSession` | session.go:88,90 | `name+":"` | ✅ |
| `SwitchClient` | session.go:96 | `session+":"` | ✅ |
| `ListWindows` | window.go:157 | `session+":"` | ✅ |
| `CreateWindow` | window.go:49 | `fmt.Sprintf("%s:", session)` | ✅ |
| `ListClients` | client.go:30 | `session+":"` | ✅ |
| `DisplayPopup` | popup.go:12 | `session+":"` | ✅ |

### Audit of non-colon targets
Window/pane targets using `session:index` format (e.g., `fmt.Sprintf("%s:%d", session, index)`) are **not affected** — the colon is inherently present as the separator between session name and window index, preventing prefix matching.

### Test coverage
Integration test `TestIntegrationSessionExistsExactMatch` (integration_test.go:707-744) was added in the same commit. It creates a `*-meta` session and verifies the prefix does NOT match.

### Conclusion
All acceptance criteria are already met. The branch `ticket/fix-tmux-session` was created from main after the fix was committed, so no additional code changes are needed.