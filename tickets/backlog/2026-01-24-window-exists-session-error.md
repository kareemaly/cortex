# Window Exists Session Error Handling

## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

`WindowExists` in `internal/tmux/window.go` returns an error when the tmux session doesn't exist. It only handles `WindowNotFoundError`, not `SessionNotFoundError`.

This causes "failed to detect architect state" when spawning architect after manually deleting the tmux session.

## Requirements

- `WindowExists` should return `false, nil` when the session doesn't exist (not just when the window doesn't exist)
