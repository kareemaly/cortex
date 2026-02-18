---
id: e7d1e38a-8447-450e-aed3-4ea408198b86
title: Architect Session — 2026-02-12T13:05Z
tags:
    - architect
    - session-summary
created: 2026-02-12T13:05:00.917902Z
updated: 2026-02-12T13:05:00.917902Z
---
## Session Summary

### TUI Improvements (3 tickets)

1. **Remove tags from kanban TUI** (41e83133) — Removed tag rendering from ticket cards in the kanban board. Cleaned up unused `tagsStyle`. Commit `99d2a11`.

2. **Hide due date badges on done tickets** (e7868e7e) — Added `&& c.status == "backlog"` guard so OVERDUE/DUE SOON badges only show in the backlog column. Commit `5ebf962`.

3. **Add Config tab to companion TUI** (fcb37d2b) — Full new tab with prompt browsing, ejection status (○ default / ● ejected), section-grouped layout (ARCHITECT, META, TICKET › WORK, etc.), preview pane, `e` to eject/edit, `c` for cortex.yaml. Added 4 new API endpoints (`/prompts`, `/prompts/eject`, `/prompts/edit`, `/config/project/edit`) and SDK methods. Commit pushed to main.

4. **Sort dashboard sessions by start date** (97734ff6) — Sessions now sorted by `StartedAt` descending (most recent first). Added `SessionStartedAt` to `TicketSummary`. Duration display uses session start time. Active projects also sorted by newest session. Commit `5fb1a1b`.

### OpenCode Fixes & Configuration (3 tickets)

5. **Debug: OpenCode architect spawn** (3b6f61f9) — Root cause: opencode defaults had Claude-specific CLI flags (`--allow-dangerously-skip-permissions`, `--allowedTools`). Removed all invalid args from opencode defaults. Commit `8fa68a4`.

6. **Make OpenCode model configurable** (f6d9871a) — Confirmed model is not hardcoded in the command builder (already correct). Added 2 regression tests to prevent future hardcoding and verify args-based model override works. Commit `98b37db`.

7. **Research OpenCode plan mode** (313e998a, ephemeral) — Found that plan mode is agent-based (`--agent plan`), not permission-based. Critical limitation: headless mode locks agents into plan mode (can't transition to build). Recommended two-session approach or custom agent definition. Research doc created.

### Other Actions

- Committed all ticket/doc history (132 files) and pushed to main
- Updated radius project config to use opencode agent type for testing