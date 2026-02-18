---
id: 6355e49c-a4a0-4feb-8de6-738a5bef9048
author: claude
type: review_requested
created: 2026-02-13T10:00:20.034736Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Fix: Architect tmux pane split 50/50 → 30/70

### Root Cause
Tmux's window resize distributes extra space **evenly** across panes, not proportionally. When `SpawnArchitect` creates a detached session at default 80x24, the 30/70 split is correct — but when the CLI attaches, the window resizes to the terminal width and the ratio drifts toward 50/50.

### Fix
Thread the CLI's terminal dimensions through the spawn pipeline so `tmux new-session` is created at the correct size from the start. When the session starts at the right width, the split produces a perfect 30/70 ratio and no resize drift occurs.

### Files Changed

**Layer 1: tmux package**
- `internal/tmux/session.go` — `CreateSession` accepts `width, height int`; passes `-x`/`-y` to `new-session` when non-zero
- `internal/tmux/command.go` — `SpawnArchitect` accepts and forwards `width, height`; `SpawnAgent` passes `0, 0`

**Layer 2: spawn package**
- `internal/core/spawn/spawn.go` — `TmuxManagerInterface.SpawnArchitect` updated; `SpawnRequest` and `ResumeRequest` gain `TermWidth`/`TermHeight` fields; `spawnInTmux` and `Resume` thread the values through

**Layer 3: API handlers**
- `internal/daemon/api/architect.go` — Parses `cols`/`rows` query params, passes to spawn request
- `internal/daemon/api/meta.go` — Same

**Layer 4: SDK client**
- `internal/cli/sdk/client.go` — `SpawnArchitect` and `SpawnMeta` accept `cols, rows int`; adds `buildTermSizeParams` helper

**Layer 5: CLI commands**
- `cmd/cortex/commands/architect.go` — Detects terminal size via `term.GetSize()`, passes to SDK
- `cmd/cortex/commands/meta.go` — Same

**Layer 6: Callers updated**
- `internal/cli/tui/dashboard/model.go` — Passes `0, 0` (TUI doesn't have direct terminal access for the target session)

**Tests**
- `internal/core/spawn/spawn_test.go` — Mock updated for new interface
- `internal/cli/sdk/client_test.go` — Test call updated
- `internal/tmux/integration_test.go` — All `CreateSession` calls updated

### Verification
- `make build` — passes
- `make lint` — 0 issues
- `make test` — all unit tests pass