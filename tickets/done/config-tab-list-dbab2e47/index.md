---
id: dbab2e47-0e8d-4663-b6ff-5062f663fb4d
title: 'Config tab: list prompts for all configured ticket types'
type: work
tags:
    - tui
    - configuration
    - agents
created: 2026-02-17T16:26:00.278991Z
updated: 2026-02-17T18:34:08.984421Z
---
## Problem

The config tab's prompt listing is filesystem-based — it walks the base prompts directory to discover `.md` files. When a project defines custom ticket types (e.g., `research`, `debug`, `frontend`) in `cortex.yaml`, their prompts don't appear in the config tab because no physical prompt files exist for those types in the defaults directory.

The prompt resolver already has correct fallback logic at runtime (custom type → work → embedded defaults), but the listing endpoint doesn't use it.

## Requirements

Rewrite the `GET /prompts` listing handler in `internal/daemon/api/prompts.go` to be **config-driven** instead of filesystem-driven:

1. Load the project config to get all ticket types from `cfg.Ticket`
2. For each configured ticket type, list the 3 stage prompts (SYSTEM.md, KICKOFF.md, APPROVE.md)
3. Use the prompt resolver's fallback chain to determine each prompt's actual source (ejected vs default, own type vs falling back to work)
4. Continue listing architect and meta prompts as before

## Expected Behavior

Given this config:
```yaml
ticket:
  work:
    agent: claude
  research:
    agent: claude
```

The config tab should show prompt groups for both `TICKET > WORK` and `TICKET > RESEARCH`, with research prompts showing as defaults (falling back to work). The user can eject any of them to customize.

## Acceptance Criteria
- All ticket types from `cortex.yaml` appear in the config tab with their 3 stage prompts
- Prompts that fall back to `work` are correctly shown as defaults (not ejected)
- Ejecting a prompt for a custom type copies the fallback content to the type-specific path
- Types removed from config no longer appear in the listing