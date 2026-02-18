---
id: 5e233cc8-c362-4044-9f88-1bad7b2f250e
author: claude
type: comment
created: 2026-02-15T10:36:48.095478Z
---
Research complete. Found several issues:
1. Three event types defined but never emitted (SessionStarted, SessionEnded, ReviewRequested)
2. Non-blocking emit silently drops events when buffer is full (64 capacity)
3. No SSE reconnection logic in any TUI view - connection drop = permanent loss of live updates
4. Potential project path mismatch between SSE subscription (raw header) and event emission (filepath.Clean'd)
5. No event IDs or Last-Event-ID support for resume capability

Creating findings doc now.