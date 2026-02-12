---
id: 7093e52e-4f5e-4ae8-ab45-19819213de09
title: Support OpenCode in cortex init and defaults management
type: work
tags:
    - opencode
created: 2026-02-11T10:29:02.890531Z
updated: 2026-02-11T10:51:53.153412Z
---
## Objective

Ensure `cortex init`, `cortex eject`, and `cortex defaults upgrade` properly handle the new OpenCode agent type and its defaults.

## What to build

### `cortex init`
- When a user runs `cortex init`, they should be able to select `opencode` as their agent type (alongside `claude` and `copilot`)
- Selecting `opencode` should set `extend: ~/.cortex/defaults/opencode` and configure the appropriate `cortex.yaml` defaults
- Look at how the existing init flow handles agent type selection and replicate for opencode

### `cortex defaults upgrade`
- The embedded opencode defaults should be extractable to `~/.cortex/defaults/opencode/` via the defaults upgrade command
- Look at how claude-code and copilot defaults are handled in this flow

### `cortex eject`
- Users should be able to eject/customize individual opencode prompts to their project's `.cortex/prompts/` directory
- This should work the same as it does for claude-code prompts

## Acceptance criteria
- `cortex init` offers opencode as an agent choice
- Selecting opencode creates proper config with `extend: ~/.cortex/defaults/opencode`
- `cortex defaults upgrade` extracts opencode defaults to `~/.cortex/defaults/opencode/`
- `cortex eject` works for opencode prompt files
- Existing claude and copilot init paths are unaffected