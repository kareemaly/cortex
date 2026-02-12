---
id: 48fb60f3-1d8a-4149-ba6f-464be3fb6a20
author: claude
type: comment
created: 2026-02-08T13:05:15.779416Z
---
## P1 - Test Coverage Gaps

### 9. Significant packages without unit tests
Test file distribution: 27 test files total (3 integration, 24 unit).

**Packages with NO tests:**
| Package | Code Files | Risk | 
|---------|-----------|------|
| `internal/daemon/autostart/` | 5 files | High — daemon lifecycle |
| `internal/daemon/logging/` | 1 file | Low |
| `internal/upgrade/` | 4 files | High — self-update, binary replacement |
| `internal/binpath/` | 1 file | Low |
| `internal/cli/sdk/` | 1 file (client.go) | **Critical — primary HTTP client** |
| `internal/cli/tui/dashboard/` | 3 files | Medium |
| `internal/cli/tui/kanban/` | 4 files | Medium |
| `internal/cli/tui/ticket/` | 3 files | Medium |
| `cmd/cortex/commands/` | 24 files | Medium — CLI entrypoints |
| `cmd/cortexd/commands/` | 5 files | Medium — daemon entrypoints |

**Packages with tests (but low coverage):**
| Package | Code/Test Ratio | Notes |
|---------|----------------|-------|
| `internal/daemon/api/` | 17 code, 1 test file (integration only) | Missing unit tests for 964-line `tickets.go` |
| `internal/daemon/mcp/` | 6 code, 3 test files | Decent coverage via integration tests |

**Well-tested packages:**
- `internal/storage/` — 5 code, 3 test (table-driven, comprehensive)
- `internal/session/` — 2 code, 1 test (355-line test for 212-line store)
- `internal/project/config/` — 4 code, 3 tests
- `internal/ticket/` — 2 code, 1 test
- `internal/prompt/` — 4 code, 2 tests

**Test quality observations:**
- Tests use table-driven patterns consistently (good)
- Proper `t.Run()` subtests
- `t.TempDir()` for cleanup
- Integration tests properly gated with `//go:build integration`
- No skipped tests (`t.Skip`) found

**Priority test additions:**
1. `internal/cli/sdk/client.go` — critical HTTP client, no tests at all
2. `internal/upgrade/` — risky binary replacement logic, no tests
3. `internal/daemon/api/tickets.go` — largest handler file (964 lines), unit tests needed