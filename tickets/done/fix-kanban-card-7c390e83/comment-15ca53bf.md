---
id: 15ca53bf-6d62-45ba-bc3d-fd6049cf5535
author: claude
type: comment
created: 2026-02-12T15:04:44.870124Z
---
Root cause: The type badge (`[work]`, `[debug]`, etc.) and due date indicators (`[OVERDUE]`, `[DUE SOON]`) are pre-rendered with their own foreground-only ANSI styles before being composed into the card text. When `selectedTicketStyle` applies a background color to the full card line, it cannot penetrate the already-rendered ANSI escape sequences from the badges — creating unstyled "islands" that break the highlight.

Fix: In `renderAllTickets()`, conditionally add `Background(lipgloss.Color("62")).Bold(true)` to the badge/indicator styles when the card is selected (`i == c.cursor && isActive`). This ensures the badge's own ANSI codes already include the correct background, so there's no conflict with the outer style.