---
id: b406bcf6-4072-4752-a1e4-e3bdcf378060
author: claude
type: comment
created: 2026-02-13T13:14:14.780911Z
---
Completed deep-dive into OpenCode source code. Found two viable injection mechanisms: OPENCODE_CONFIG_DIR (temp directory with plugin/ subfolder) and OPENCODE_CONFIG_CONTENT plugin array with file:// URLs. Now writing up findings doc.