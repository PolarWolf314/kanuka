# Agent Guidelines for Kānuka

## Build, Lint, and Test Commands

### Building
```bash
go build -v ./...
```

### Linting
```bash
golangci-lint run
```
Uses golangci-lint v2.1.5 with config in `.golangci.yml`. Enabled linters:
- errcheck, godot, gosec, govet, ineffassign, staticcheck, unused
- goimports formatter (auto-runs on lint)

### Testing
```bash
# Run all tests
go test -v ./...

# Run tests for specific category
go test -v ./test/integration/init/...
go test -v ./test/integration/create/...
go test -v ./test/integration/register/...
go test -v ./test/integration/encrypt/...
go test -v ./test/integration/decrypt/...
go test -v ./test/integration/remove/...

# Run a single test
go test -v ./test/integration/init/... -run TestSecretsInitBasic/InitInEmptyFolder
```

## Code Style Guidelines

### Imports
- Standard library imports first, third-party imports second
- Separate groups with blank lines
- Use `goimports` to format (included in linter)
```go
import (
	"fmt"
	"os"

	"github.com/PolarWolf314/kanuka/internal/configs"

	"github.com/spf13/cobra"
)
```

### Formatting
- Use tabs for indentation
- No trailing whitespace
- Max line length ~100 characters (soft limit)
- Run `goimports` before committing (handled by linter)

### Naming Conventions
- **Packages**: lowercase, single word (`grove`, `secrets`, `cmd`)
- **Public functions/types**: PascalCase (`ParsePackageName`, `Package`)
- **Private functions**: PascalCase (`isInKanukaManagedSection`)
- **Variables**: camelCase (`packageName`, `tempDir`)
- **Constants**: PascalCase (`supportedLanguages`, `GroveCmd`)
- **Global package vars**: PascalCase (`GroveLogger`, `groveVerbose`)

### Types
- Use explicit types for function returns
- Prefer returning pointers to structs for large data
- Define structs with field comments:
```go
type Package struct {
	Name        string // Original name as provided by user.
	NixName     string // Nix package name (e.g., pkgs.nodejs_18)
	DisplayName string // Display name for user feedback.
	Version     string // Version if specified.
	Channel     string // Channel used (unstable, stable)
}
```

### Error Handling
- Always return errors with context using `fmt.Errorf("%w")` for wrapping
- Check errors immediately after operations
- Use descriptive error messages:
```go
if err != nil {
	return fmt.Errorf("failed to read devenv.nix: %w", err)
}
```

### File Operations
- Use `os.ReadFile` and `os.WriteFile` for simple file I/O
- Use `defer` for cleanup (`file.Close()`, `os.RemoveAll()`)
- Set restrictive permissions for sensitive files (0600 for secrets, 0755 for dirs)
- Use `#nosec G306` comment when intentionally using less restrictive permissions

### Testing
- Use subtests for grouping related test cases
- Create temp directories with `os.MkdirTemp("", "prefix-*")`
- Clean up temp directories with `defer os.RemoveAll(tempDir)`
- Use `test/integration/shared` package for shared test utilities
- Save and restore working directory in tests:
```go
originalWd, err := os.Getwd()
defer func() {
	os.Chdir(originalWd)
}()
```
- Use descriptive test names (`testInitInEmptyFolder`, `testWithVerboseFlag`)
- Output verification should check for expected strings in captured output

### Cobra Commands
- Define commands as package-level variables with descriptive names
- Use `PersistentPreRun` for initialization before subcommands
- Provide `Get*Cmd()` helper functions for testing
- Reset global state with `ResetGlobalState()` helper between tests
- Use `--verbose` and `--debug` flags consistently

### Comments
- Exported functions must have comments
- Use `// FunctionName does X.` format
- Keep comments concise and focused
- Avoid obvious comments ("this is a loop")

### Logging
- Use the internal logger from `internal/logging`
- Support `--verbose` (info) and `--debug` (debug) flags
- Log errors with context
- Prefix log messages with `[info]`, `[debug]`, or `[error]`

### Constants
- Define maps for static data (supported languages, validation info)
- Use maps for lookups rather than if/else chains
- Group related constants together

## Project Structure

```
cmd/              - CLI command definitions
internal/
  configs/        - Configuration management
  grove/          - Development environment logic
  logging/        - Logging utilities
  secrets/        - Secrets encryption/decryption
  utils/          - Shared utilities (filesystem, strings, system)
test/
  integration/    - Integration tests by feature
    shared/       - Shared test utilities
```

## Before Committing
1. Run `golangci-lint run` to ensure code quality
2. Run `go test -v ./...` to ensure all tests pass
3. Run `go build -v ./...` to ensure project compiles
4. Run `goimports` if not using the linter
