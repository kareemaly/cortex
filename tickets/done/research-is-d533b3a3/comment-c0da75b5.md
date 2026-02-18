---
id: c0da75b5-e469-44e1-8ac9-60db7a51b68d
author: claude
type: comment
created: 2026-02-14T11:57:28.728889Z
---
Investigation complete. Key finding: anthropic.txt is NOT accessible as a file on disk from the installed package. It's compiled into a Bun-native binary. The npm package ships only 6 files (wrapper script + platform binary). Will document full findings with alternative approaches.