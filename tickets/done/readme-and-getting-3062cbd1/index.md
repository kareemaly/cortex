---
id: 3062cbd1-56c9-4181-902a-dbd52e5fd90d
title: README and getting started documentation
type: work
created: 2026-02-04T12:48:13.436708Z
updated: 2026-02-04T13:35:53.622558Z
---
## Goal

Create a compelling README that gets power users (tmux/vim users, CLI enthusiasts) from zero to running cortex in under 5 minutes.

## Target Audience

- Developers already comfortable with tmux, vim, CLI tools
- Users of AI coding assistants (Claude, Cursor, etc.) who want more control
- People who value keyboard-driven workflows and transparency

## What to Explore Before Writing

### 1. Commands and Workflow
- Explore `cmd/cortex/commands/` to document all available CLI commands
- Explore `cmd/cortexd/commands/` to understand daemon commands
- Map the core workflow: init → architect → ticket spawn → review → done

### 2. Installation
- Review `install.sh` for installation instructions
- Check `Makefile` for build targets
- Verify dependencies: tmux, git, claude CLI

### 3. Configuration
- Explore `internal/project/config/` for project config schema
- Explore `internal/daemon/config/` for global config schema
- Check `internal/install/defaults/claude-code/` for default templates
- Understand the extend/inheritance model

### 4. Architecture (for brief "How it works" section)
- Explore `internal/daemon/api/` for HTTP API structure
- Explore `internal/daemon/mcp/` for MCP tools
- Understand: daemon serves all projects, CLI/TUI are clients

### 5. Existing Docs
- Read `CLAUDE.md` for architecture details (don't duplicate, reference it)
- Read `CONTRIBUTING.md` for development setup
- Read `LINEAGE.md` for project history/context

## README Structure

Keep it SHORT and scannable. Target: under 300 lines.

```
# Cortex
[One-line description]

## What is Cortex?
[2-3 sentences max - value prop for power users]

## Quick Start
[3 commands: install, init, architect]

## Requirements
[Bullet list: tmux, git, claude CLI]

## Core Workflow
[Brief explanation with diagram or flow]
- Architect creates/manages tickets
- Agents work on tickets in isolated sessions
- Human reviews and approves

## Commands
[Table: command | description]

## Configuration
[Brief overview of .cortex/cortex.yaml and ~/.cortex/settings.yaml]
[Link to CONFIG_DOCS.md for details]

## Development
[Link to CONTRIBUTING.md]

## Architecture
[One paragraph + link to CLAUDE.md]
```

## Tone Guidelines

- Direct, no fluff ("Cortex is..." not "Cortex aims to be...")
- Show code, not prose
- Assume intelligence, don't over-explain
- No marketing speak or buzzwords

## Files to Create/Update

- `README.md` - main documentation (create new)
- Verify `CONTRIBUTING.md` is accurate and complete

## Out of Scope

- Detailed API documentation
- Tutorials or guides
- Configuration reference (already in CONFIG_DOCS.md)