---
id: f2238aee-c729-4d31-aa82-efef05ef5be8
author: claude
type: comment
created: 2026-02-09T16:27:39.981691Z
---
## Finding 5: Structural Improvements — XML Tags and Organization

### Current structure issues

The current prompt mixes concerns:
1. Role definition (line 3)
2. Behavioral constraints (XML tags, lines 5-11)
3. Tool reference (lines 17-19)
4. Workflow (lines 23-36)
5. Context awareness (line 40)
6. Communication (line 42-44)

The behavioral constraints are front-loaded but ticket quality (the core job!) isn't covered at all.

### Recommended structure

Based on Claude's prompting best practices:
- **Role first** — keep this strong (already good)
- **Core responsibility** — ticket quality guidelines immediately after role
- **Behavioral guardrails** — XML tags for constraints
- **Workflow** — how to use tools and manage state
- **Examples** — concrete good/bad examples last (Claude's attention is strong at beginning and end)

Proposed section order:
1. `# Role` — who you are
2. `## Writing Tickets` (with `<ticket_quality>` tag) — your core job, how to do it well
3. `## Cortex Workflow` — tools and state management
4. `## After Spawning` — post-spawn flow
5. `## Context Awareness` — compaction and persistence
6. `## Communication` — style guidelines
7. `## Examples` — good/bad ticket examples

### Best practice references
- Claude prompting guide: "Be explicit with your instructions" — the ticket quality section makes the implicit expectation explicit
- Claude prompting guide: "Add context to improve performance" — explaining WHY file paths are harmful gives Claude the reasoning to generalize
- Claude Code system prompt: "Avoid over-engineering" — directly applies to ticket authoring (don't over-specify)
- Claude prompting guide: "Use XML format indicators" — already used well, extend to ticket quality