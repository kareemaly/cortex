---
id: f6d9871a-13e7-488c-9419-109555da43e2
title: Make OpenCode model configurable instead of hardcoded
type: work
tags:
    - opencode
    - configuration
created: 2026-02-12T11:58:29.810659Z
updated: 2026-02-12T12:14:54.457639Z
---
## Problem

When spawning OpenCode sessions (architect or ticket), the model is hardcoded to `gpt-5.2` (openai) in the command builder. The model should not be hardcoded — it should either use OpenCode's own default or be configurable via the project's `cortex.yaml`.

## Expected Behavior

- If no model is specified in config, OpenCode uses its own default model
- Users can override the model via `args:` in `cortex.yaml`, e.g.:
  ```yaml
  ticket:
    work:
      agent: opencode
      args:
        - "-m"
        - "anthropic/claude-sonnet-4"
  ```

## Acceptance Criteria

- No hardcoded model in the OpenCode command builder
- Model can be configured via `args:` in cortex.yaml (both architect and ticket sessions)
- If no model arg is provided, OpenCode launches with its own default