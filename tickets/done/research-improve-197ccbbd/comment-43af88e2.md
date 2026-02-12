---
id: 43af88e2-4fd4-4514-8e98-e8ec2d820bf3
author: claude
type: comment
created: 2026-02-09T16:26:38.094857Z
---
## Finding 1: Current SYSTEM.md Analysis — Strengths

The current architect SYSTEM.md (45 lines) gets several things right:

**1. Clear role definition (line 3)**
> "You are a project architect orchestrating development through tickets and delegation. You do not write code or read source files."

This is excellent — concise, specific, with an explicit negative boundary. Aligns with Claude's prompting best practice of being explicit.

**2. Effective XML tags for behavioral constraints**
Three XML tags (`<do_not_act_before_instructions>`, `<stay_high_level>`, `<investigate_before_answering>`) use the directive-naming pattern recommended by Claude's prompting guide. The tag names themselves communicate intent even before reading the content.

**3. Explicit "don't read source files" with an alternative**
The `<stay_high_level>` tag tells the architect NOT to read source files AND gives it what to do instead (spawn an explore agent). This follows the best practice of "tell what to do instead of what not to do."

**4. Context compaction awareness**
Line 40: "Your context will compact as it fills. Persist important decisions in ticket comments." This is directly from Claude's long-horizon reasoning best practices. Good.

**5. Automatic state transitions documented**
Lines 25-29 prevent the architect from manually calling `moveTicket` for transitions that happen automatically. This prevents double-moves and confusion.

**6. Concise communication directive**
"Be direct and concise. Provide fact-based assessments. Do not give time estimates." — Clear, follows Claude Code's own system prompt style.