---
id: 197ccbbd-bbbc-4f3a-8727-86244ba2b129
title: 'Research: improve architect SYSTEM.md with prompting best practices'
type: research
tags:
    - architect-prompt
    - best-practices
    - research
created: 2026-02-09T16:22:01.488168Z
updated: 2026-02-09T16:36:52.183233Z
---
## Goal

Investigate the current architect SYSTEM.md prompt and brainstorm improvements based on Claude prompting best practices and our learnings from real usage.

## Research Inputs

1. **Current architect SYSTEM.md** — read `internal/install/defaults/claude-code/prompts/architect/SYSTEM.md` and analyze what's working and what's missing
2. **Claude Code best practices** — read files in `~/ephemeral/claude-code-best-practices` for agent-specific guidance
3. **Claude prompting best practices** — fetch and analyze https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/claude-prompting-best-practices
4. **Claude Code guide** — use the claude-code-guide skill to get best practices for system prompts and agent behavior

## Key Behavioral Changes to Address

### 1. No time estimates in tickets
The architect must never include time estimates, effort sizing, or complexity ratings in tickets. These are unreliable and misleading.

### 2. No assumptions — focus on WHAT, not HOW
This is the critical one. The architect currently tends to include implementation details in tickets that are often wrong assumptions. The rules should be:

- **Tickets should define WHAT needs to be done** — requirements, acceptance criteria, design decisions
- **Tickets should NOT define HOW to implement** — no assumed file paths, no guessed function signatures, no speculative code patterns
- **Wrong assumptions are worse than no details** — an agent picking up a ticket will explore the codebase anyway. Wrong assumptions actively mislead and waste time.
- **When the HOW matters for design** (e.g., "should we use a wrapper model or peer models?"), the architect must run an explore agent first to get accurate technical details before writing the ticket. Never guess.
- **Leave exploration to the agent** — the ticket agent will explore, plan, and figure out implementation details. They can ask the user if things are unclear. The architect should trust this.

### 3. General prompt quality
Review against Claude prompting best practices for:
- Clear role definition
- Effective use of XML tags for structure
- Appropriate constraints and guardrails
- Good examples vs. bad examples

## Expected Output

A doc with:
- Analysis of current SYSTEM.md strengths and weaknesses
- Specific recommended changes with rationale
- Draft updated SYSTEM.md content (or key sections)
- References to which best practices informed each change

This research will inform a follow-up implementation ticket to actually update the prompt.