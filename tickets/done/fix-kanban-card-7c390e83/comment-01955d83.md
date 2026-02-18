---
id: 01955d83-b974-49d4-8451-d171e962433a
author: claude
type: comment
created: 2026-02-13T08:32:08.36826Z
---
Root cause: Lipgloss's Render() appends \x1b[0m (full ANSI reset) after each rendered element. When badges are pre-rendered with lipgloss and concatenated into the title line, the badge's trailing reset cancels the outer selectedTicketStyle's background for all subsequent text. The previous fix (commit 167b3f0) added Background("62") to badge styles, but that only fixes the badge itself — the title text after the badge still loses styling due to the reset.

Fix: For selected cards only, bypass lipgloss and use raw ANSI escape sequences that change only the foreground color (no reset). The outer selectedTicketStyle.Render() sets the background once, and inline foreground-only changes preserve it throughout.