---
id: 1aa5d854-bb04-4ee2-b286-9d44c64bd78b
author: claude
type: ticket_done
created: 2026-01-28T07:34:30.363564Z
---
## Summary

Replaced the architect agent's system prompt mechanism from `--append-system-prompt` (which appends to the default Claude Code engineer prompt) to `--system-prompt` (which fully replaces it). The architect prompt is now a self-contained system prompt that prevents the architect from behaving like an engineer — no file reading, no code editing, no implementation.

## Key Decisions

- **`--system-prompt` for architect only**: Ticket agent sessions continue using `--append-system-prompt` since the default Claude Code engineer prompt is appropriate for agents that write code. The architect needs a fundamentally different prompt that excludes engineer behaviors.
- **Boolean field on LauncherParams**: Added `ReplaceSystemPrompt bool` rather than encoding the flag name directly, keeping the launcher abstraction clean and testable.
- **Prompt content matches ticket spec exactly**: The full prompt includes behavioral guardrails (`do_not_act_before_instructions`, `stay_high_level`, `investigate_before_answering`), context awareness guidance, MCP tool documentation, workflow steps, ticket writing guidance, and communication style.

## Files Modified

1. **`internal/core/spawn/launcher.go`** — Added `ReplaceSystemPrompt bool` field to `LauncherParams`; `buildLauncherScript` selects `--system-prompt` or `--append-system-prompt` based on this flag.
2. **`internal/core/spawn/spawn.go`** — Sets `launcherParams.ReplaceSystemPrompt = true` for `AgentTypeArchitect` in `Spawn()`; updated `buildPrompt` comment to document both modes.
3. **`internal/install/install.go`** — Replaced 23-line `defaultArchitectPrompt` with full self-contained prompt (~80 lines) used during `cortex init`.
4. **`.cortex/prompts/architect.md`** — Replaced with identical full prompt content for the current project.
5. **`internal/core/spawn/spawn_test.go`** — `TestWriteLauncherScript_Architect`: added `ReplaceSystemPrompt: true` and assertions for `--system-prompt` (not `--append-system-prompt`). Ticket agent test unchanged.

## Verification

- `make test` — all tests pass
- `make lint` — 0 issues
- `make build` — builds successfully
- Branch merged to main