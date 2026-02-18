---
id: fa5fe1c9-9fd0-46ac-8f90-faa1283e20ad
title: Render ticket references in kickoff prompt
type: work
tags:
    - agents
    - tmux
created: 2026-02-14T11:29:44.271112Z
updated: 2026-02-14T11:33:58.815595Z
---
## Problem

Tickets support a `references` field (e.g., `ticket:abc123`, `doc:xyz789`) stored in YAML frontmatter, but when an agent session spawns, the kickoff prompt doesn't include them. Agents have no way to know references exist unless the ticket body happens to mention them by text.

## Requirements

- Add a `References` field to the `TicketVars` struct used for kickoff template rendering
- Pass the ticket's references into the template vars during spawn assembly
- Update the kickoff templates (all four ticket types: work, research, debug, chore) to render a references section when references are present

## Design Constraints

- **Lightweight rendering only** — list the reference IDs (e.g., `ticket:abc123`, `doc:xyz789`), do not resolve or inline their content
- Agents already have the `readReference` MCP tool to follow up on any reference they see
- Only render the section when references are non-empty

## Acceptance Criteria

- Kickoff prompt includes a references section when the ticket has references
- References section is absent when there are no references
- All four ticket type templates (work, research, debug, chore) are updated
- Existing tests pass