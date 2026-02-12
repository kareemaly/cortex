---
id: 44537ce1-0150-4dd4-9ca3-702da038a3de
title: Research OpenCode system prompt injection methods
type: research
tags:
    - research
    - opencode
references:
    - doc:b111544a-08d4-4153-a2f9-dddea1a025ea
created: 2026-02-11T08:33:59.584448Z
updated: 2026-02-11T08:44:03.866507Z
---
## Objective

The previous research (doc `b111544a`) concluded that OpenCode has no `--system-prompt` flag and relies on context files. This seems unlikely for a tool this customizable — investigate more thoroughly whether OpenCode supports direct system prompt injection.

## Context

- OpenCode source is cloned at `~/ephemeral/opencode`
- The npm package is `opencode-ai`, currently at v1.1.18 locally
- Previous research found agent markdown files (`.opencode/agents/*.md`) can define per-agent prompts, and `OPENCODE_CONFIG_CONTENT` can inject config inline
- But we need to know if there's a more direct way to pass a system prompt (e.g., a `--system-prompt` flag, a `SYSTEM.md` file convention, or a config field)

## Research Steps

1. **Search the OpenCode source** at `~/ephemeral/opencode` for system prompt handling — look for how the system prompt is assembled, what inputs feed into it, and whether there's a flag or config field for direct injection
2. **Look for SYSTEM.md or similar conventions** — search for any file-based system prompt injection patterns
3. **Check CLI flags thoroughly** — look at all flag definitions in the source, not just `--help` output
4. **Check the agent/prompt assembly pipeline** — trace how the final system prompt is built from all sources

## Deliverable

Update the existing research doc or create a new one clarifying exactly how to inject a full system prompt into OpenCode programmatically.