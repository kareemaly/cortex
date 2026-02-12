---
id: 67f40f62-3024-4bef-b8c3-c09750d5dd7b
author: claude
type: review_requested
created: 2026-02-10T10:22:03.54688Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/5cafc359-6a34-41d9-92ec-2cb2ab1a1ff9
        commit: "6826581"
---
Two changes to clarify that `concludeSession` auto-creates a session summary doc:

1. **`internal/daemon/mcp/tools_architect.go`** — Updated `concludeSession` tool description to: "Automatically creates a session summary doc and cleans up session state. Do not manually create a session doc before calling this."

2. **`internal/install/defaults/claude-code/prompts/architect/SYSTEM.md`** — Added explicit warning under Documentation section: "Do not create a session summary doc manually — `concludeSession` automatically creates one as part of cleanup. Creating one before concluding will result in duplicates." Also removed "session context" from the `createDoc` guidance to avoid implying the architect should capture session summaries manually.