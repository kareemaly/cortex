---
id: a5123e95-9b28-48b0-9965-bccadd3313c3
author: claude
type: comment
created: 2026-02-09T16:28:54.357015Z
---
## Finding 8: Trade-offs and Design Decisions

### Trade-off 1: Prompt length vs. comprehensiveness
- Current: ~45 lines (very lean)
- Proposed: ~95 lines (doubled but still concise)
- CLAUDE.md best practices say "keep it concise" and "each line should pass: would removing this cause mistakes?"
- Every new line in the proposal addresses a documented behavioral problem. The examples section alone prevents the #1 issue (wrong assumptions in tickets).
- **Decision: Accept the length increase.** Every line earns its place.

### Trade-off 2: Examples in SYSTEM.md vs. separate file
- Could put examples in a separate file loaded via `@` import in CLAUDE.md
- But SYSTEM.md is the system prompt, not CLAUDE.md. It needs to be self-contained since it's injected at session start.
- Examples in-line ensures they're always present in context.
- **Decision: Keep examples inline in SYSTEM.md.**

### Trade-off 3: Strict "never include file paths" vs. nuanced
- Could allow file paths when the architect has verified them via explore agent
- But this creates a gray area — the architect might skip the explore step and include paths anyway
- The `<ticket_quality>` tag says "Speculative architecture unless verified by an explore agent" which already provides the escape hatch
- **Decision: Default to "never include", with explicit escape hatch for explore-verified facts.**

### Trade-off 4: XML tag naming style
- Current uses imperative action names: `<do_not_act_before_instructions>`, `<stay_high_level>`
- New tag `<ticket_quality>` is a noun/concept, not an action
- Both styles work — Claude's guide says XML tags help with "format indicators" and structure
- **Decision: Use concept names for content sections (`<ticket_quality>`) and imperative names for behavioral constraints (`<do_not_act_before_instructions>`).** This naturally distinguishes "what to know" from "what to do."

### Trade-off 5: Include cross-project guidance or not
- The architect has `listProjects` and cross-project `project_path` params
- But this is an advanced feature and most sessions are single-project
- Adding it to SYSTEM.md could encourage unnecessary cross-project operations
- **Decision: Omit for now.** The tools are self-describing, and the meta agent is the intended cross-project orchestrator.

### Items intentionally NOT included
- **No ticket body template/format** — forcing a template makes tickets rigid and formulaic. The good/bad examples show the pattern without mandating structure.
- **No dependency management guidance** — too complex for the system prompt, and the "prefer independent tickets" heuristic covers 90% of cases.
- **No specific tool call sequences** — the current workflow section is good enough, and over-specifying sequences limits flexibility.