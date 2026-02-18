---
id: e4ed09ad-0e25-4033-9e37-25a3712e5685
title: 'Research: Audit Claude Code hooks for completeness and accuracy'
type: research
tags:
    - research
    - agents
    - hooks
created: 2026-02-13T13:06:42.179773Z
updated: 2026-02-13T13:20:09.534884Z
---
## Goal

Audit the Claude Code hook system to determine if we are using all available hooks and whether our current hook-to-status mappings are accurate. We may be missing hooks that would improve status tracking fidelity.

## Context

Cortex currently uses 3 Claude Code hooks:
- `PostToolUse` → `in_progress` (with tool name)
- `Stop` → `idle`
- `PermissionRequest` → `waiting_permission`

We need to verify:
1. What is the complete list of available hooks in Claude Code?
2. Are there hooks we're not using that would improve status accuracy? (e.g., hooks for when the agent starts thinking, encounters errors, or other state transitions)
3. Are our current hook-to-status mappings correct? For example, does `Stop` truly mean idle, or does it mean something else?
4. Is there a hook for when the agent starts working / receives a new message?
5. Are there hooks related to subagent/task spawning?

## Method

Use the Claude Code guide to look up the complete hooks API documentation and compare against our current implementation.

## Acceptance Criteria

- Complete list of available Claude Code hooks documented
- Gaps identified (hooks we should be using but aren't)
- Any incorrect mappings flagged
- Recommendations for improving hook coverage