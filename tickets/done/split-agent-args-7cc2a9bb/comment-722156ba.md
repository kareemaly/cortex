---
id: 722156ba-a6ac-4c13-bb1a-385ad7453e40
author: claude
type: ticket_done
created: 2026-01-27T09:32:49.239853Z
---
## Summary

Split the flat `agent_args` config field into a structured `AgentArgsConfig` with separate `architect` and `ticket` sub-fields, allowing different CLI arguments to be passed to architect vs ticket agent sessions.

## Changes Made

### 1. `internal/project/config/config.go`
- Added `AgentArgsConfig` struct with `Architect []string` and `Ticket []string` fields
- Implemented custom `UnmarshalYAML` using `yaml.Node` to detect YAML kind:
  - **MappingNode** → new structured format, decoded into respective fields
  - **SequenceNode** → old flat format, copied to both `Architect` and `Ticket` for backward compatibility
- Changed `Config.AgentArgs` field type from `[]string` to `AgentArgsConfig`

### 2. `internal/core/spawn/orchestrate.go` (line 145)
- Changed `projectCfg.AgentArgs` → `projectCfg.AgentArgs.Ticket` so ticket sessions only receive ticket-specific args

### 3. `internal/daemon/api/architect.go` (line 124)
- Added `AgentArgs: projectCfg.AgentArgs.Architect` to the architect `SpawnRequest`, threading architect-specific args through to the spawner

### 4. `internal/project/config/config_test.go`
- Added 5 test cases:
  - `TestAgentArgs_NewNestedFormat` — verifies independent architect/ticket parsing
  - `TestAgentArgs_OldFlatFormat` — verifies backward compat (flat list populates both)
  - `TestAgentArgs_Absent` — verifies both fields are empty when agent_args is absent
  - `TestAgentArgs_PartialArchitectOnly` — only architect specified
  - `TestAgentArgs_PartialTicketOnly` — only ticket specified

## Key Decisions

- **Backward compatibility via custom unmarshaler**: Rather than requiring a migration or erroring on the old format, the custom `UnmarshalYAML` transparently handles both YAML sequence (old) and mapping (new) formats. Old flat lists are copied to both `Architect` and `Ticket`, preserving existing behavior.
- **No changes to SpawnRequest/launcher**: The `SpawnRequest.AgentArgs` field stays `[]string` — only the callers (orchestrate.go and architect.go) select the appropriate sub-field. This minimizes the blast radius.
- **Type alias trick for unmarshaling**: Used `type plain AgentArgsConfig` to avoid infinite recursion in the custom unmarshaler.

## Verification

- `make test` — all tests pass (including 5 new config tests)
- `make lint` — 0 issues
- `make build` — clean compilation

## New YAML Format

```yaml
agent_args:
  architect:
    - "--budget=150000"
  ticket:
    - "--budget=50000"
```

Old format still works:
```yaml
agent_args:
  - "--budget=100000"
```