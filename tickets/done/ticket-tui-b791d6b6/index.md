---
id: b791d6b6-0bed-4dae-b502-1f8ffb09e950
title: 'Ticket TUI: Collapsible Comments and Scroll Preservation'
type: work
created: 2026-02-02T14:15:47.375647Z
updated: 2026-02-02T15:49:44.809366Z
---
## Summary

Two UX improvements for the ticket detail TUI:

1. Comments section takes too much space when not selected
2. Description scroll position resets when agent status updates

## Requirements

### Collapsible Comments Section
- When comments section (Row 2) is not focused, show only 10-15% height
- When focused/selected, expand to normal size
- Smooth transition if possible (or instant is fine)
- Should still show comment count or preview when collapsed

### Scroll Position Preservation
- Agent status changes trigger ticket refresh via SSE
- Currently resets description viewport scroll position
- Preserve scroll position across refreshes
- Apply to both description (Row 1) and comments (Row 2) viewports

## Technical Notes
- SSE updates come through `handleTicketUpdate()` 
- Viewport state likely reset when model rebuilds
- May need to store scroll offset and restore after update

## Acceptance Criteria
- [ ] Comments section is compact (10-15%) when not focused
- [ ] Comments expand when selected/focused
- [ ] Description scroll position preserved on status updates
- [ ] Comment list scroll position preserved on status updates