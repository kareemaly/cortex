---
id: 9161728f-d523-46ac-ac9c-4b06b6f79832
author: claude
type: comment
created: 2026-02-07T10:29:17.657787Z
---
## Audit Finding: CLEAN — API, SDK, and MCP Tools

### HTTP Endpoints (server.go)
All routes verified present and properly wired:
- **Global**: GET /health, GET /projects, DELETE /projects, POST /daemon/focus
- **Tickets**: Full CRUD + move, spawn, comments, reviews, focus, conclude, execute-action, due-date (set/clear)
- **Docs**: Full CRUD + move, comments (NEW: POST /{id}/comments)
- **Sessions**: GET / (list), DELETE /{id} (kill), POST /{id}/approve (NEW endpoint)
- **Architect**: GET / (state), POST /spawn, POST /focus
- **Agent**: POST /status
- **Events**: GET /events (SSE)

### SDK Client (client.go)
All endpoints have matching SDK methods: ListAllTickets, GetTicket, CreateTicket, UpdateTicket, DeleteTicket, MoveTicket, SpawnSession, AddComment, RequestReview, ConcludeSession, SetDueDate, ClearDueDate, ListDocs, GetDoc, CreateDoc, UpdateDoc, DeleteDoc, MoveDoc, AddDocComment, ListSessions, KillSession, ApproveSession, FindTicketByID, ListProjects, SpawnArchitect, ArchitectState, etc.

### MCP Architect Tools (tools_architect.go)
All 20+ tools registered and verified: listProjects, listTickets, readTicket, createTicket, updateTicket, deleteTicket, moveTicket, updateDueDate, clearDueDate, addTicketComment, spawnSession, getCortexConfigDocs, createDoc, readDoc, updateDoc, deleteDoc, moveDoc, listDocs, addDocComment (NEW), listSessions (NEW)

### MCP Ticket Tools (tools_ticket.go)
All 5 tools: readTicket, addComment, addBlocker, requestReview, concludeSession

### New tool verification:
- **addDocComment**: Calls SDK AddDocComment → POST /docs/{id}/comments → store.AddComment → writes comment file
- **listSessions**: Calls SDK ListSessions → GET /sessions/ → SessionManager → session.Store.List() → resolves ticket titles

**Severity: CLEAN**