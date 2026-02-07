---
id: 7cc2a9bb-eb7f-446e-ae37-386b8d72e180
title: Split agent_args into Architect and Ticket Session Args
type: ""
created: 2026-01-27T09:24:08.453053Z
updated: 2026-01-27T09:32:49.241768Z
---
## Problem

Currently `agent_args` in `cortex.yaml` is a single list of extra CLI arguments passed to all agent sessions. There's no way to pass different arguments to the architect session vs ticket sessions.

For example, you might want a higher budget for the architect and a lower one for ticket agents, or pass different flags to each.

## Solution

Replace the single `agent_args` field with two separate fields:

```yaml
agent_args:
  architect:
    - "--budget=150000"
  ticket:
    - "--budget=50000"
```

## Scope

- Update the `Config` struct in `internal/project/config/config.go` to split `agent_args` into `architect` and `ticket` sub-fields
- Update YAML parsing and any validation logic
- Update all spawn paths that read `agent_args` to use the appropriate field based on session type
- Update the prompt templates if they reference agent args
- Ensure backward compatibility — if the old flat `agent_args` format is used, treat it as applying to both session types (or error with a clear message)

## Acceptance Criteria

- [ ] `cortex.yaml` supports `agent_args.architect` and `agent_args.ticket` as separate lists
- [ ] Architect sessions receive only `agent_args.architect` arguments
- [ ] Ticket sessions receive only `agent_args.ticket` arguments
- [ ] Existing flat `agent_args` format is handled gracefully (backward compat or clear error)
- [ ] Config tests cover the new structure