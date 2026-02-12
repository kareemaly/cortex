---
id: 39bc8f28-b83a-4efd-9d50-36daefeb2427
author: claude
type: comment
created: 2026-02-11T08:00:08.332598Z
---
Completed investigation of both Codex CLI (v0.77.0) and all Cortex agent integration patterns. Key finding: Codex has full MCP support and a headless exec mode, making integration feasible. The main challenge is MCP config injection — Codex uses `codex mcp add/remove` instead of a `--mcp-config <file>` flag. Writing up full analysis now.