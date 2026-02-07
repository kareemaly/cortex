---
id: 9ed4e809-407f-4ef8-909b-1b754fdcc27c
title: Fix Config Deep Merge for Extended Bases
type: work
created: 2026-01-30T12:37:53.904218Z
updated: 2026-01-30T13:09:52.35916Z
---
## Summary

The `extend` configuration doesn't properly merge `cortex.yaml` values from the extended base into the project config. When a project extends `~/.cortex/defaults/claude-code`, fields like `agent_args` from the base are not applied.

**Reference case:** `~/projects/nau/radius` extends `~/.cortex/defaults/claude-code` but agent args from the base config are not being used when spawning agents.

## Requirements

- Investigate how config loading handles the `extend` directive
- Implement proper deep merge: base config values should be used as defaults, project config values should override
- Fields to merge include: `agent`, `agent_args`, `lifecycle`, and any other config fields
- Project-specific values always take precedence over extended base values

## Acceptance Criteria

- [ ] `agent_args` from extended base are applied when project doesn't specify them
- [ ] Project-level config values override base values
- [ ] Deep merge works for nested structures (e.g., `lifecycle` hooks)
- [ ] Test with `~/projects/nau/radius` extending `~/.cortex/defaults/claude-code`
- [ ] Do not modify the radius project — investigation only

## Investigation Notes

The agent should examine:
- How `internal/project/config/` loads and resolves extended configs
- Where config is consumed when spawning agents
- Whether the merge happens at load time or needs to happen at usage sites