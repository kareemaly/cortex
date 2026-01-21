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
