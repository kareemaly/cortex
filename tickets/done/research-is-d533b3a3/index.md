---
id: d533b3a3-31a5-4b76-b52c-4eeb6aca947b
title: 'Research: Is OpenCode''s default prompt accessible from installed package'
type: research
tags:
    - research
    - opencode
    - agents
references:
    - doc:324cb419-80fe-4c49-929f-12e22bd0e9e6
created: 2026-02-14T11:53:05.917183Z
updated: 2026-02-14T11:59:01.895391Z
---
## Question

When OpenCode is installed (via npm), is the default provider prompt (`anthropic.txt`) accessible as a readable file on disk, or does it get bundled/compiled into the JS binary?

We need this to determine whether Cortex can read OpenCode's default prompt at runtime (to prepend it before Cortex's SYSTEM.md) instead of embedding a static copy that drifts from upstream.

## Investigation Scope

- Check the OpenCode source at `~/ephemeral/opencode` to understand how `anthropic.txt` is referenced in the build/bundle process
- Look at `packages/opencode/src/session/prompt/anthropic.txt` and how it's imported
- Check if the prompt is inlined at build time (e.g., bundled into JS) or shipped as a separate file
- Find where OpenCode gets installed globally (e.g., `which opencode`, then trace back to the package location)
- Check if the installed package contains the prompt file at a predictable path
- Look at how `system.ts` imports/reads the prompt — is it a filesystem read or a bundled import?

## Acceptance Criteria

- Clear answer: is the prompt file readable from the installed package at a known path?
- If yes, document the path pattern (e.g., `$(dirname $(which opencode))/../lib/node_modules/opencode/...`)
- If no (bundled), document how it's compiled in and confirm there's no runtime way to extract it
- Recommendation on best approach for Cortex to access it