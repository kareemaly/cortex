---
id: 54449dfb-6876-4cdb-8062-521d65386132
author: claude
type: done
created: 2026-02-14T11:55:38.343594Z
---
## Summary

Reworked all three ticket agent SYSTEM.md prompts (work, debug, research) to fix issues with unconditional `readReference` calls, missing blocker/scope guidance, and duplicated coding advice.

### Changes made

**3 files modified:**
- `internal/install/defaults/main/prompts/ticket/work/SYSTEM.md` — Conditional `readReference`, removed "ask clarifying questions", added "When Things Go Wrong" section
- `internal/install/defaults/main/prompts/ticket/debug/SYSTEM.md` — Conditional `readReference`, added "When Things Go Wrong" with debug-specific examples
- `internal/install/defaults/main/prompts/ticket/research/SYSTEM.md` — Conditional `readReference`, removed "brainstorm with user", added "When Things Go Wrong" section

### Key fixes
| Issue | Fix |
|-------|-----|
| Unconditional `readReference` causing hallucinated IDs | Now says "If a References section is listed above" + reinforcing guard |
| No blocker guidance | "When Things Go Wrong" section with type-specific examples |
| No scope creep guidance | Explicit instruction to flag and stop |
| Duplicated general coding advice | Trimmed to role framing only |

### Verification
- `make build` ✓
- `make test` ✓  
- `make lint` ✓
- Pushed to origin/main as commit c1cf72a