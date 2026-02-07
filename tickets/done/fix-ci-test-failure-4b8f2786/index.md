---
id: 4b8f2786-ee16-4d4f-9c3a-5478b74b1a0c
title: Fix CI test failure
type: work
created: 2026-02-04T12:04:14.261005Z
updated: 2026-02-04T12:16:28.366462Z
---
CI build is failing on `make test`. The output shows all visible tests passing but exits with FAIL status and error code 2.

**Symptoms:**
- All logged tests show PASS
- Final result is FAIL
- Exit code 2

**Investigation needed:**
1. Run `make test` locally to reproduce
2. Check for compilation errors or test failures in packages not shown in the truncated output
3. Address `go mod tidy` diagnostic (charmbracelet/x/ansi dependency)
4. Ensure all tests pass cleanly