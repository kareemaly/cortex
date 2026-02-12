---
id: b8e24380-cfc1-446d-bdc4-a61f93e93f5a
title: 'Integration test: OpenCode agent spawn lifecycle'
type: work
tags:
    - opencode
created: 2026-02-11T10:29:09.843444Z
updated: 2026-02-11T13:16:30.773516Z
---
## Objective

Verify the full OpenCode spawn lifecycle works end-to-end: spawning a ticket agent with OpenCode, MCP tools loading correctly, system prompt injection, and session lifecycle (addComment → requestReview → conclude).

## What to test

1. **Launcher script generation** — Verify `buildOpenCodeCommand()` produces a valid launcher script with correct `OPENCODE_CONFIG_CONTENT`, `--agent cortex`, and `--prompt` flag
2. **OPENCODE_CONFIG_CONTENT structure** — Verify the generated JSON contains the system prompt in the agent `prompt` field, MCP config in the `mcp` key, and permissions
3. **MCP config format** — Verify the MCP config uses OpenCode's format (`command` as array, `environment` not `env`, `type: "local"` not `stdio`)
4. **Agent type routing** — Verify `opencode` agent type routes to the OpenCode builder, not the Claude builder
5. **Resume handling** — Verify that resume mode is properly handled (OpenCode doesn't support resume, so it should always start fresh or return an appropriate error)

## Acceptance criteria
- Unit tests for `buildOpenCodeCommand()` pass
- Generated launcher script is syntactically valid bash
- OPENCODE_CONFIG_CONTENT JSON is valid and contains all required fields
- Agent routing correctly distinguishes claude, opencode, and copilot