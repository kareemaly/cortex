---
id: f3f3e815-86cc-4c04-abe5-e4e6ed5bf864
author: claude
type: comment
created: 2026-02-11T08:39:26.174633Z
---
Completed deep-dive into OpenCode source. Traced the full system prompt assembly pipeline in session/llm.ts, session/system.ts, session/instruction.ts, and config/config.ts. Confirmed there is NO --system-prompt flag, but identified 6 distinct injection methods with important caveats about what gets replaced vs. appended. Creating research doc now.