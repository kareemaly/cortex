# API Integration Tests

Add integration tests for HTTP API endpoints using `httptest`.

## Context

The daemon exposes REST endpoints for ticket management. We need integration tests that test the full HTTP flow (routing, handlers, JSON encoding) against a real file-based ticket store.

## Endpoints to Test

| Endpoint | Method | Test Cases |
|----------|--------|------------|
| `/health` | GET | Returns status ok |
| `/tickets` | GET | Empty list; list with tickets |
| `/tickets` | POST | Create ticket; invalid JSON |
| `/tickets/{status}` | GET | List by status; invalid status |
| `/tickets/{status}/{id}` | GET | Get ticket; not found; wrong status |
| `/tickets/{status}/{id}` | PUT | Update ticket; not found |
| `/tickets/{status}/{id}` | DELETE | Delete ticket; not found |
| `/tickets/{status}/{id}/move` | POST | Move ticket; invalid target status |

## Endpoints to Skip

These use tmux and should be skipped:
- `POST /tickets/{status}/{id}/spawn`
- `DELETE /sessions/{id}`

## Requirements

### 1. Expose Router for Testing

Update `internal/daemon/api/server.go` to export a `NewRouter(deps)` function:

```go
func NewRouter(deps *Dependencies) chi.Router {
    r := chi.NewRouter()
    // ... existing route setup ...
    return r
}

func NewServer(port int, logger *slog.Logger, deps *Dependencies) *Server {
    r := NewRouter(deps)
    // ... rest of server setup ...
}
```

### 2. Create Integration Test File

Create `internal/daemon/api/integration_test.go` with build tag:

```go
//go:build integration
```

### 3. Test Setup Helper

```go
func setupTestServer(t *testing.T) *httptest.Server {
    tmpDir := t.TempDir()

    // Create .cortex structure
    for _, dir := range []string{"backlog", "progress", "done"} {
        os.MkdirAll(filepath.Join(tmpDir, ".cortex/tickets", dir), 0755)
    }

    store := ticket.NewStore(filepath.Join(tmpDir, ".cortex/tickets"))
    deps := &Dependencies{
        TicketStore:   store,
        TmuxManager:   nil,
        ProjectConfig: &projectconfig.Config{Name: "test"},
        Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
        ProjectRoot:   tmpDir,
    }

    return httptest.NewServer(NewRouter(deps))
}
```

### 4. Test Cases to Implement

- `TestHealthEndpoint`
- `TestListAllTicketsEmpty`
- `TestListAllTicketsWithData`
- `TestCreateTicket`
- `TestCreateTicketInvalidJSON`
- `TestGetTicket`
- `TestGetTicketNotFound`
- `TestGetTicketWrongStatus`
- `TestUpdateTicket`
- `TestDeleteTicket`
- `TestMoveTicket`
- `TestMoveTicketInvalidStatus`
- `TestListByStatus`
- `TestInvalidStatus`

### 5. Update Makefile

Add to Makefile:

```makefile
test-integration:
	go test -tags=integration -v ./internal/daemon/api/... ./internal/daemon/mcp/...
```

## Verification

```bash
make test              # Unit tests only
make test-integration  # Integration tests
```

## Notes

- Use `t.TempDir()` for automatic cleanup
- Discard logger output in tests
- Test both success and error paths
- Verify response status codes and JSON bodies

## Implementation

### Commits Pushed

- `9949c59` feat: add API integration tests with httptest

### Key Files Changed

| File | Change |
|------|--------|
| `internal/daemon/api/server.go` | Extracted route setup into `NewRouter()` function |
| `internal/daemon/api/integration_test.go` | New file with 14 integration tests |
| `Makefile` | Added API tests to `test-integration` target |

### Important Decisions

- Used `httptest.NewServer` for in-process HTTP testing (no binary required)
- Tests use real `ticket.Store` with temporary directories for realistic integration
- Skipped spawn and session kill endpoints as they require tmux

### Scope Changes

None - implemented all 14 test cases as specified in the requirements.
