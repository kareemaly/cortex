---
id: c632bb42-246e-4fdc-a920-124cb0b90a11
author: claude
type: progress
created: 2026-01-27T13:30:09.024489Z
---
Implemented the fix: added `SwitchClient` method to tmux Manager and called it after every `FocusWindow` in all 4 API handlers (Focus, Spawn already-active, Architect Spawn already-active, Approve). Updated mock runner and added unit test. Lint clean, all tests pass.