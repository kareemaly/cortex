---
id: e2a79543-839a-46b7-9f60-94e3023126ac
title: Add `cortex eject` command
type: work
created: 2026-02-03T08:28:06.75176Z
updated: 2026-02-03T08:39:25.754455Z
---
# Overview

Add a CLI command to copy prompts from global defaults to project-level for customization.

## Usage

```bash
cortex eject <prompt-path>
```

## Examples

```bash
cortex eject ticket/work/SYSTEM.md
# Copies: ~/.cortex/defaults/claude-code/prompts/ticket/work/SYSTEM.md
# To: .cortex/prompts/ticket/work/SYSTEM.md

cortex eject architect/SYSTEM.md
# Copies: ~/.cortex/defaults/claude-code/prompts/architect/SYSTEM.md
# To: .cortex/prompts/architect/SYSTEM.md
```

## Requirements

1. Resolve source path from `~/.cortex/defaults/<agent-type>/prompts/<path>` based on project config
2. Create destination directory structure in `.cortex/prompts/` if needed
3. Copy file to `.cortex/prompts/<path>`
4. Error if source doesn't exist
5. Error if destination already exists (unless `--force` flag)
6. Print success message with paths

## Flags

- `--force` - Overwrite existing file

## Notes

- Prompt auto-discovery already exists (`.cortex/prompts/` takes precedence)
- Agent type should come from project config (`architect.agent` or `ticket.<type>.agent`)