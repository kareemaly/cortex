---
id: 287f8a77-ad2a-42dc-8917-a4664d2c9509
author: claude
type: comment
created: 2026-02-08T13:04:37.107115Z
---
## P0 - Must Fix Before Open Source

### 1. Missing LICENSE file
- **Impact:** Legally required for OSS distribution
- **Finding:** MIT is referenced in README.md but no `LICENSE` file exists at root
- **Action:** Create `/LICENSE` with the MIT license text

### 2. Daemon binds to all network interfaces (0.0.0.0)
- **Impact:** Security risk - daemon accessible from any network, not just localhost
- **Location:** `internal/daemon/api/server.go:113-118`
```go
Addr: fmt.Sprintf(":%d", port)  // Binds to 0.0.0.0:port
```
- **Action:** Change to `fmt.Sprintf("127.0.0.1:%d", port)` by default. Add config option for `0.0.0.0` binding when explicitly needed (e.g., remote VM deployments mentioned in CLAUDE.md).

### 3. Go module path uses personal GitHub handle
- **Impact:** Module path `github.com/kareemaly/cortex` references personal account, not an org
- **Location:** `go.mod` line 1, all import paths throughout
- **Action:** Decide on final org/module path before release (e.g., `github.com/cortexdev/cortex` or similar). This is a one-time migration but affects every `.go` file.