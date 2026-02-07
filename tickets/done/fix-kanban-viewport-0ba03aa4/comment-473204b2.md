---
id: 473204b2-8801-4165-b8f8-571c8f6a3668
author: claude
type: comment
created: 2026-02-06T10:19:54.389059Z
---
## Root Cause Analysis

Both bugs share the same root cause: a **title-wrapping width mismatch** between `ticketHeight()` and `renderAllTickets()`.

### The mismatch

In `renderAllTickets()` (column.go:113-140), tickets with type badges wrap their title at a **narrower width**:
```go
titleWidth := max(width-4, 10)
badgeWidth := len("[" + t.Type + "] ")  // e.g., 8 for "[debug] "
wrappedTitle := wrapText(t.Title, titleWidth-badgeWidth)  // ← narrower
```

But `ticketHeight()` (column.go:357-359) wraps at the **full width**:
```go
func ticketHeight(t sdk.TicketSummary, titleWidth int) int {
    return len(wrapText(t.Title, titleWidth)) + 1  // ← full width, ignores badge
}
```

Wrapping at a narrower width produces more lines. For example, a 39-char title with `titleWidth=40` and `badgeWidth=8`:
- `wrapText(title, 40)` → 1 line (fits)
- `wrapText(title, 32)` → 2 lines (wraps)

### Bug 1: Highlight disappears on scroll

`cursorYOffset()` calls `ticketHeight()` for each preceding ticket. Each ticket with a badge and a long-enough title causes the computed Y offset to be **too low** by 1+ lines. This error accumulates as you scroll past more tickets. The viewport's scroll-to-cursor logic scrolls to the wrong region, making the highlighted ticket invisible.

### Bug 2: Bottom chevron stays visible at end

When pressing "G", `cursorYOffset()` underestimates the last ticket's position. The viewport scrolls to `cursorY + cursorH - vpHeight`, which is too low. Since the actual content extends further than the computed position, `vp.YOffset + vpHeight < TotalLineCount()` remains true, keeping the chevron visible.