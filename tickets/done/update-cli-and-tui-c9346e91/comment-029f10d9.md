---
id: 029f10d9-fa80-46f2-b775-7e904b7c73b1
author: claude
type: comment
created: 2026-02-07T10:03:02.922926Z
---
Completed thorough exploration of CLI and TUI code. Key findings:

1. **No old patterns found** -- all dates are flat fields (Created/Updated/Due), Session is NOT embedded on ticket, Comments have no SessionID. The codebase has already been migrated to the new frontmatter storage patterns.

2. **Consumer code is fully decoupled** -- all TUI components (kanban, ticket detail, dashboard) and CLI commands consume `sdk.*Response` types via HTTP. They don't need changes for storage layer updates.

3. **Translation layer is clean** -- `internal/types/convert.go` has 4 converter functions (ToTicketResponse, ToTicketSummary, ToCommentResponse, ToSessionResponse) that bridge storage models to API responses. This is the only file that needs updating if storage models change.

4. **`cortex init`** creates `.cortex/tickets/{backlog,progress,review,done}/` directories -- already aligned with frontmatter directory-per-entity storage.

5. **Session info flows correctly** -- `TicketSummary.HasActiveSession/AgentStatus/AgentTool/IsOrphaned` populated by daemon from separate session store, not from ticket data. Ticket detail TUI uses status heuristic for session detection.

Fields consumed by TUI code from TicketSummary: ID, Type, Title, Tags, Status, Created, Updated, Due, HasActiveSession, AgentStatus, AgentTool, IsOrphaned
Fields consumed from TicketResponse: ID, Type, Title, Body, Tags, References, Status, Created, Updated, Due, Comments
Fields consumed from CommentResponse: ID, Author, Type, Content, Action, Created