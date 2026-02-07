---
id: 38f6e893-f75a-460b-b4eb-95bf20731b2b
title: Display TUI Errors in Red
type: ""
created: 2026-01-26T17:32:19.844936Z
updated: 2026-01-27T09:25:37.304213Z
---
## Problem

Errors in the TUI are displayed as plain text, making them easy to miss or confuse with normal output.

## Solution

Style all error messages in the TUI with red text using lipgloss or the existing styling system.

## Acceptance Criteria

- [ ] All error messages in the TUI render in red
- [ ] Applies to inline errors, status bar errors, and any error flash messages