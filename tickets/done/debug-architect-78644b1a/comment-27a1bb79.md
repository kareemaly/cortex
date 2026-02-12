---
id: 27a1bb79-e407-4d83-8683-f67efc4686e3
author: claude
type: comment
created: 2026-02-10T14:41:19.373427Z
---
Root cause: tmux resolves `-t name:` using a cascade of exact match → fnmatch → substring matching. The trailing colon only separates session:window — it does NOT disable substring matching for the session part. So `tmux has-session -t cortex:` resolves "cortex" as a substring of "cortex-meta" and returns success, causing false "already spawned" detection.

Fix: Replace all `name+":"` target patterns with `sessionTarget(name)+":"` where `sessionTarget()` prepends tmux's `=` prefix for exact name matching (supported since tmux 2.1, 2015). `tmux has-session -t =cortex:` only matches a session named exactly "cortex".