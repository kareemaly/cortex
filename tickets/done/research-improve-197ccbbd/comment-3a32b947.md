---
id: 3a32b947-a350-4610-8853-4b6b89f4d315
author: claude
type: review_requested
created: 2026-02-09T16:29:15.945344Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/92f96e6d-c7c3-461e-9a7b-0dbbdb3c1a24
---
## Research Summary: Architect SYSTEM.md Improvements

### Sources Analyzed
1. Current `internal/install/defaults/claude-code/prompts/architect/SYSTEM.md` (45 lines)
2. Claude Code system prompt reference (`~/ephemeral/claude-code-best-practices/`)
3. Claude prompting best practices (platform.claude.com)
4. Claude Code guide (system prompts, agent behavior, multi-agent patterns)
5. Peer prompts: meta SYSTEM.md, ticket/work SYSTEM.md, ticket/research SYSTEM.md, KICKOFF.md

### Key Findings

**The current prompt is well-structured but missing its most important section: ticket quality guidelines.** The architect's core job is creating tickets, but there are zero instructions on what makes a good ticket. This is the root cause of the "wrong assumptions" problem.

### Three Critical Additions (Priority Order)

1. **`<ticket_quality>` section** — New XML-tagged section defining the "WHAT not HOW" principle with include/exclude lists and motivation ("you have not read the code"). This directly addresses the #1 behavioral issue.

2. **Good/bad ticket examples** — Claude's prompting guide emphasizes examples as the most effective behavioral shaping tool. A paired good/bad example shows the exact contrast between an assumption-laden ticket and a requirements-focused one.

3. **"No time estimates in tickets"** — Extends the existing communication guideline to explicitly cover ticket bodies, closing the loophole.

### Smaller Additions
- Ticket type guidance (work/debug/research/chore)
- Scoping heuristic ("each ticket completable by one agent in one session")
- Explore-first workflow for when technical details matter
- Docs workflow mention
- Minor wording improvement to `<do_not_act_before_instructions>` ("discuss scope and requirements first")

### Draft Available
Full draft SYSTEM.md is in Finding 7 comment (~95 lines, up from ~45). Every added line addresses a documented behavioral issue.

### Trade-offs Documented
- Prompt doubles in length but every line earns its place (Finding 8)
- Examples kept inline (SYSTEM.md is self-contained, not CLAUDE.md with imports)
- Strict "never include file paths" default with explore-agent escape hatch
- Cross-project guidance intentionally omitted (meta agent's job)
- No rigid ticket template (examples > templates for flexibility)

### Recommended Next Steps
1. Create an implementation ticket to update the SYSTEM.md
2. Use the draft from Finding 7 as the starting point
3. Test the updated prompt with a few architect sessions
4. Iterate based on observed behavior