---
id: 2c650880-f3ad-4ddf-a4de-d5a5e2e78b62
title: Fix architect session injecting wrong project's tickets/docs
type: debug
tags:
    - debug
    - meta-agent
    - architect
    - injection
created: 2026-02-10T09:42:59.686459Z
updated: 2026-02-10T09:59:06.021415Z
---
## Problem

When the meta agent spawns an architect for a specific project, the architect session receives tickets and docs from a different project. Observed: spawning architect for project "kesc" injected tickets/docs from "pcrf-e2e" instead.

## Reproduction

1. Have multiple projects registered (at least "kesc" and "pcrf-e2e")
2. From meta, spawn architect for "kesc"
3. Architect session shows tickets/docs belonging to "pcrf-e2e"

## Likely Cause

The project context (X-Cortex-Project header or CORTEX_PROJECT_PATH env) is not being correctly set or propagated when composing the architect's prompt injection. The prompt builder may be using a stale or wrong project reference.

## Acceptance Criteria

- Architect session always receives tickets/docs for the correct project it was spawned for
- Verified with multiple registered projects
- No cross-project data leakage in prompt injection