---
id: c13a181b-ce96-4222-867c-e1e45c526200
author: claude
type: comment
created: 2026-02-06T09:44:08.113971Z
---
Starting implementation: replacing manual scroll system with bubbles/viewport in column.go. The viewport will properly clip content within fixed height bounds, fixing the overflow issue.