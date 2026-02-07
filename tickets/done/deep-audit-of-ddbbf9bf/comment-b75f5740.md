---
id: b75f5740-b3f0-4eeb-94be-48bd5b42521f
author: claude
type: comment
created: 2026-02-07T10:29:45.817502Z
---
## Audit Finding: CLEAN — CLI/TUI, Config, Notifications, Types

### CLI Commands:
- **ticket list**: Renders ID, TYPE, TITLE, CREATED, ACTIVE columns correctly
- **ticket show**: Renders full detail with Type, Created, Updated, Body, Comments (with Type badge, Author)
- **ticket spawn**: --resume and --fresh flags with mutual exclusivity check, correct mode mapping

### TUI Components:
- **Kanban**: Type badges with color coding, due date indicators (OVERDUE red, DUE SOON orange), tags in metadata, agent status with icons, orphaned session detection
- **Ticket detail**: DETAILS section (Created, Updated, Due with color, Tags), SESSION section (Agent, Status, Tool), Comments with author and formatTimeAgo

### Config:
- TicketsPath() defaults to `{projectRoot}/tickets`, DocsPath() to `{projectRoot}/docs`
- Relative paths resolved to projectRoot, absolute paths respected
- cortex init creates correct structure

### Install:
- Creates .cortex/ directory, tickets/ with status subdirs, docs/ directory
- Auto-registers project in global config

### Notifications:
- Dispatcher uses SessionManager for session lookup
- Subscribes to event bus per project
- Event classification handles new storage events
- No TODOs/debug prints

### Types/Convert:
- TicketResponse: All fields including Tags, Type, Due
- TicketSummary: HasActiveSession, AgentStatus, AgentTool, IsOrphaned
- SessionResponse: All session fields
- ToTicketSummary() detects orphaned sessions by checking tmux window existence
- CommentResponse: Author field (not SessionID)

**Severity: CLEAN**