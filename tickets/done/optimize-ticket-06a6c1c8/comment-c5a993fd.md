---
id: c5a993fd-a4ab-4ec1-9dd2-ea2ceb9aca36
author: claude
type: comment
created: 2026-02-10T08:03:10.059273Z
---
Completed thorough codebase exploration and detailed implementation plan covering all 5 areas:

1. **Remove `readTicket`** from ticket agents (tool registration, handler, EmptyInput struct, tests)
2. **Add `readReference`** tool with union output type (TicketOutput | DocOutput)
3. **Thread ticket type** through spawn -> CLI flag -> MCP Config -> Session -> conditional tool registration
4. **Add `createDoc`** for research tickets only (simplified handler using s.sdkClient directly)
5. **Update all prompts** (4 claude-code SYSTEM.md files, 4 copilot KICKOFF.md files, cortex.yaml defaults)

Key discovery: `cortex.yaml` defaults contain `--allowedTools` lists with `mcp__cortex__readTicket` that also need updating.

Plan is organized into 8 phases with explicit file paths, code snippets, and edge case analysis.