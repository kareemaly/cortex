---
id: 1f4660e6-c2df-4358-8147-dc89ebd4cfda
author: claude
type: review_requested
created: 2026-02-14T11:54:47.143257Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Changes

Reworked all three ticket agent SYSTEM.md prompts to fix unconditional `readReference` hallucination, add blocker/scope guidance, and remove duplicated general coding advice.

### Files changed:

1. **`internal/install/defaults/main/prompts/ticket/work/SYSTEM.md`** — Made `readReference` conditional on References section existing; removed "ask clarifying questions" (agents use `addBlocker` instead); added "When Things Go Wrong" section with blocker and scope-creep guidance; trimmed role description to avoid duplicating base agent instructions.

2. **`internal/install/defaults/main/prompts/ticket/debug/SYSTEM.md`** — Made `readReference` conditional; added "When Things Go Wrong" section with debug-specific blocker examples (can't reproduce, missing access) and scope escalation guidance.

3. **`internal/install/defaults/main/prompts/ticket/research/SYSTEM.md`** — Made `readReference` conditional; removed "brainstorm with the user" (agents run in tmux, not conversationally); added "When Things Go Wrong" section with research-specific blocker examples and scope guidance; simplified role description.

### Verification:
- `make build` — embedded defaults compile ✓
- `make test` — all unit tests pass ✓
- `make lint` — 0 issues ✓