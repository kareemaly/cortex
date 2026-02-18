---
id: 12cfb849-d059-4a7c-b76e-ee9788ba698e
author: claude
type: comment
created: 2026-02-14T11:45:46.607051Z
---
Investigation complete. Found the definitive answer: Cortex's SYSTEM.md REPLACES OpenCode's default provider prompt when injected via `agent.cortex.prompt` in `OPENCODE_CONFIG_CONTENT`. The key code path is in OpenCode's `session/llm.ts:71`. Creating detailed findings doc now.