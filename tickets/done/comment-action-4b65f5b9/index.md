---
id: 4b65f5b9-86f2-4f97-88bb-26dcf368d453
title: Comment Action Execution via Daemon with Tmux Popup
type: work
created: 2026-02-02T14:15:47.142901Z
updated: 2026-02-02T14:32:26.519155Z
---
## Summary

Wire up comment actions so TUI can request the daemon to execute them. Actions execute as tmux popups in the ticket's session.

## Requirements

### Global Config (`~/.cortex/settings.yaml`)
- Add `diff_tool` setting: `lazygit` or `git`
- Default based on availability detected at `cortex init`

### cortex init Enhancement
- Check if `lazygit` is installed (`which lazygit`)
- Set `diff_tool: lazygit` if available, otherwise `diff_tool: git`
- Only set on fresh init, don't override existing config

### API Endpoint
- `POST /tickets/{id}/comments/{comment_id}/execute`
- Reads `Comment.Action` from the comment
- Reads `diff_tool` from global config
- Requires active session for the ticket (return error if none)
- Opens tmux popup in the session's window

### Tmux Popup Execution
- For `git_diff` action type:
  - If `diff_tool: lazygit`: `tmux popup -E -d "<repo_path>" lazygit`
  - If `diff_tool: git`: `tmux popup -E -d "<repo_path>" git diff <commit>`
- Popup opens in the ticket's tmux session/window

### TUI Integration
- In comment detail modal, add keybinding `d` for diff action
- Calls SDK client to execute the action
- Only shown/enabled when `Comment.Action.Type == "git_diff"`
- Show error if no active session

## Acceptance Criteria
- [ ] Global config has `diff_tool` setting
- [ ] `cortex init` detects lazygit and sets config
- [ ] API endpoint executes comment actions
- [ ] Tmux popup opens with correct tool and repo path
- [ ] TUI `d` key triggers action execution
- [ ] Fails gracefully when no active session