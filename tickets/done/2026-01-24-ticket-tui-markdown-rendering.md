# Ticket TUI Markdown Rendering

## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

Ticket body and comments display raw markdown text instead of rendered formatting.

## Requirements

- Render ticket body as formatted markdown using github.com/charmbracelet/glamour
- Render comments as formatted markdown
- Style should fit terminal theme

## Implementation

### Commits pushed
- `ff447cf` feat: add markdown rendering to ticket TUI using glamour

### Key files changed
- `go.mod` - Added glamour v0.9.1 dependency
- `internal/cli/tui/ticket/model.go` - Integrated glamour renderer

### Changes made
- Added `mdRenderer *glamour.TermRenderer` field to Model struct
- Initialized renderer in `New()` with 80-character word wrap and auto-style
- Updated renderer width on window resize to match terminal width
- Added `renderMarkdown()` helper with fallback to raw text on error
- Modified `renderSection()` to render body as markdown
- Modified `renderComments()` to render comment content as markdown
