---
id: 26b5d5d0-eabf-4276-891c-171c930c23c8
author: claude
type: done
created: 2026-02-12T15:05:20.415366Z
---
## Root Cause

The kanban card type badges (`[work]`, `[debug]`, `[research]`, `[chore]`) and due date indicators (`[OVERDUE]`, `[DUE SOON]`) were pre-rendered with foreground-only ANSI styles via lipgloss. When the outer `selectedTicketStyle` applied a background color (color "62") to the full card line, it could not penetrate the already-encoded ANSI escape sequences from the badges — creating unstyled "islands" that broke the highlight.

## Resolution

**File:** `internal/cli/tui/kanban/column.go` (+15, -3)

In `renderAllTickets()`, conditionally added `.Background(lipgloss.Color("62")).Bold(true)` to the type badge and due date indicator styles when the card is selected (`i == c.cursor && isActive`). This ensures the badge/indicator ANSI codes already include the correct selection background before being composed into the card text.

**Commit:** `167b3f0` — fix: apply selection background to type badge and due date indicators on kanban cards

## Verification

- `make build` — compiles successfully
- `make lint` — 0 issues
- `make test` — all unit tests pass