# Fix Module Path

The Go module path is `github.com/kareemaly/cortex1` but the actual repository is `github.com/kareemaly/cortex` (without the 1).

## Requirements

- Update `go.mod` module path to `github.com/kareemaly/cortex`
- Update all import statements across the codebase
- Ensure the project builds and all tests pass

## Verification

```bash
make build
make test
make test-integration
```

## Implementation

### Commits
- `5c0fd10` refactor: rename module path from cortex1 to cortex

### Key Files Changed
- `go.mod` - Updated module declaration
- `Makefile` - Updated ldflags references (3 occurrences)
- 29 Go files - Updated import statements

### Changes Made
- Changed module path from `github.com/kareemaly/cortex1` to `github.com/kareemaly/cortex`
- Updated all internal imports across cmd/, internal/ directories
- Ran `go mod tidy` to update dependencies
- Verified with `make build`, `make test`, and `make test-integration` (all passing)
