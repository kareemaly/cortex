---
id: 9fa2e9b6-b64e-465d-85f5-317c6c617511
title: Project-Aware API
type: ""
created: 2026-01-21T16:26:53Z
updated: 2026-01-21T16:26:53Z
---
Make the HTTP API project-aware so a single daemon can serve multiple projects.

## Context

Currently `cortexd serve` uses a global `~/.cortex/tickets` directory. We need it to act as a proxy to project-local ticket stores, with each request specifying which project it's for.

**Architecture:**
- Single global daemon manages ALL projects
- Multiple architects can run simultaneously for different projects
- Multiple `cortex kanban` instances from different project directories
- Each API request specifies which project's tickets to access

## Changes Required

### 1. API Layer - Add Project Header

All endpoints (except `/health`) must require `X-Cortex-Project` header containing the absolute project path.

**File:** `internal/daemon/api/server.go`

Add middleware to extract and validate project path:
```go
func projectMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        projectPath := r.Header.Get("X-Cortex-Project")
        if projectPath == "" {
            http.Error(w, `{"error":"missing X-Cortex-Project header"}`, http.StatusBadRequest)
            return
        }

        // Validate project exists
        ticketsDir := filepath.Join(projectPath, ".cortex", "tickets")
        if _, err := os.Stat(ticketsDir); os.IsNotExist(err) {
            http.Error(w, `{"error":"project not found or not initialized"}`, http.StatusNotFound)
            return
        }

        // Add to context
        ctx := context.WithValue(r.Context(), projectPathKey, projectPath)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

Apply to all routes except `/health`.

### 2. Ticket Store Per Project

**File:** `internal/daemon/api/server.go` or new `internal/daemon/api/stores.go`

Replace single `TicketStore` in Dependencies with a store manager:
```go
type StoreManager struct {
    stores map[string]*ticket.Store
    mu     sync.RWMutex
}

func (m *StoreManager) GetStore(projectPath string) (*ticket.Store, error) {
    m.mu.RLock()
    if store, ok := m.stores[projectPath]; ok {
        m.mu.RUnlock()
        return store, nil
    }
    m.mu.RUnlock()

    // Create new store
    m.mu.Lock()
    defer m.mu.Unlock()

    // Double-check after acquiring write lock
    if store, ok := m.stores[projectPath]; ok {
        return store, nil
    }

    ticketsDir := filepath.Join(projectPath, ".cortex", "tickets")
    store, err := ticket.NewStore(ticketsDir)
    if err != nil {
        return nil, err
    }
    m.stores[projectPath] = store
    return store, nil
}
```

### 3. Update Handlers

All handlers need to get the ticket store from context:
```go
func (s *Server) handleListTickets(w http.ResponseWriter, r *http.Request) {
    projectPath := r.Context().Value(projectPathKey).(string)
    store, err := s.storeManager.GetStore(projectPath)
    if err != nil {
        // handle error
    }
    // use store...
}
```

### 4. SDK Client Changes

**File:** `internal/cli/sdk/client.go`

Add project path to client:
```go
type Client struct {
    baseURL     string
    projectPath string
    httpClient  *http.Client
}

func NewClient(baseURL, projectPath string) *Client {
    return &Client{
        baseURL:     baseURL,
        projectPath: projectPath,
        httpClient:  &http.Client{Timeout: 10 * time.Second},
    }
}

// All request methods add the header:
func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
    if c.projectPath != "" {
        req.Header.Set("X-Cortex-Project", c.projectPath)
    }
    return c.httpClient.Do(req)
}
```

### 5. CLI Commands

**File:** `cmd/cortex/commands/kanban.go`

Resolve project path and pass to SDK:
```go
func runKanban(cmd *cobra.Command, args []string) {
    cwd, err := os.Getwd()
    if err != nil {
        // handle error
    }

    _, projectRoot, err := projectconfig.LoadFromPath(cwd)
    if err != nil {
        if projectconfig.IsProjectNotFound(err) {
            fmt.Fprintf(os.Stderr, "Error: not in a cortex project\n")
            os.Exit(1)
        }
        // handle error
    }

    client := sdk.NewClient(sdk.DefaultBaseURL, projectRoot)
    // ...
}
```

### 6. Remove Global Tickets Directory

**File:** `cmd/cortexd/commands/serve.go`

Remove the global ticket store initialization (lines 74-83). The serve command no longer needs to know about any specific project at startup.

Also remove `~/.cortex/tickets` from the install process if it exists.

### 7. Update Integration Tests

**File:** `internal/daemon/api/integration_test.go`

Tests need to include `X-Cortex-Project` header in requests.

## Endpoints Affected

| Endpoint | Change |
|----------|--------|
| `GET /health` | No change |
| `GET /tickets` | Requires `X-Cortex-Project` header |
| `POST /tickets` | Requires `X-Cortex-Project` header |
| `GET /tickets/{status}` | Requires `X-Cortex-Project` header |
| `GET /tickets/{status}/{id}` | Requires `X-Cortex-Project` header |
| `PUT /tickets/{status}/{id}` | Requires `X-Cortex-Project` header |
| `DELETE /tickets/{status}/{id}` | Requires `X-Cortex-Project` header |
| `POST /tickets/{status}/{id}/move` | Requires `X-Cortex-Project` header |
| `POST /tickets/{status}/{id}/spawn` | Requires `X-Cortex-Project` header |
| `DELETE /sessions/{id}` | Requires `X-Cortex-Project` header |

## Verification

```bash
# Start daemon (no project context needed)
cortexd serve

# From project directory, kanban should show project-local tickets
cd ~/projects/test-cortex
cortex kanban

# From different project, should show different tickets
cd ~/projects/other-project
cortex kanban

# API without header should fail
curl http://localhost:4200/tickets
# Returns: {"error":"missing X-Cortex-Project header"}

# API with header should work
curl -H "X-Cortex-Project: /Users/me/projects/test-cortex" http://localhost:4200/tickets

# Run tests
make test
make test-integration
```

## Notes

- The daemon no longer needs a "current project" - it serves any project
- Store manager caches stores for performance (don't recreate on every request)
- Project path must be absolute (validate in middleware)
- Consider adding project path validation (must have `.cortex/tickets` directory)

## Implementation

### Commits Pushed

- `4ca33e0` feat: make HTTP API project-aware via X-Cortex-Project header

### Key Files Changed

| File | Change |
|------|--------|
| `internal/daemon/api/store_manager.go` | **NEW** - Thread-safe per-project store manager with lazy initialization |
| `internal/daemon/api/middleware.go` | Added `ProjectRequired()` middleware to validate header |
| `internal/daemon/api/deps.go` | Replaced `TicketStore`, `ProjectConfig`, `ProjectRoot` with `StoreManager` |
| `internal/daemon/api/server.go` | Applied middleware to project-scoped routes (all except `/health`) |
| `internal/daemon/api/tickets.go` | All handlers get store from context via `GetProjectPath()` + `StoreManager.GetStore()` |
| `internal/daemon/api/sessions.go` | Handler gets store from context, loads project config per-request |
| `cmd/cortexd/commands/serve.go` | Removed global ticket store init, creates `StoreManager` instead |
| `internal/cli/sdk/client.go` | Added `projectPath` field and `doRequest()` helper to add header |
| `cmd/cortex/commands/kanban.go` | Added `resolveProjectPath()` helper, passes project to client |
| `cmd/cortex/commands/list.go` | Uses `resolveProjectPath()` |
| `cmd/cortex/commands/spawn.go` | Uses `resolveProjectPath()` |
| `cmd/cortex/commands/session.go` | Uses `resolveProjectPath()` |
| `cmd/cortex/commands/version.go` | Passes empty string for health check (doesn't need project) |
| `internal/daemon/api/integration_test.go` | Updated to use `StoreManager`, adds header to requests, new validation tests |

### Important Decisions

1. **Middleware validates three things**: header presence, absolute path, and `.cortex/tickets` directory existence
2. **StoreManager uses double-check locking**: fast read path for cached stores, thread-safe creation for new ones
3. **Spawn handler loads project config per-request**: since config is now project-specific, it's loaded on demand
4. **Health endpoint excluded from middleware**: allows daemon health checks without project context
5. **`resolveProjectPath()` defined in kanban.go**: shared helper available to all CLI commands in the same package

### Scope Changes

None - implementation followed the original ticket specification.