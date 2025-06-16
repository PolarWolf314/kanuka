# Integration Tests

This directory contains comprehensive integration tests for the Kanuka CLI application, organized by feature and test type for better maintainability.

## Directory Structure

```
test/
├── integration/
│   ├── init/           # Init command integration tests
│   ├── encrypt/        # Encrypt command integration tests  
│   ├── decrypt/        # Decrypt command integration tests
│   ├── shared/         # Shared testing utilities
│   └── README.md       # This file
└── README.md
```

## Integration Test Categories

### Init Command Tests (`test/integration/init/`)

- **`basic_test.go`** - Core functionality tests
  - `InitInEmptyFolder` - Successful initialization in empty folder
  - `InitInAlreadyInitializedFolder` - Behavior when already initialized
  - `InitWithVerboseFlag` - Initialization with verbose output
  - `InitWithDebugFlag` - Initialization with debug output

- **`permissions_test.go`** - Permission-related edge cases
  - `InitWithReadOnlyUserDirectory` - Read-only user directory handling

- **`environment_test.go`** - Environment variable edge cases
  - `InitWithInvalidXDGDataHome` - Invalid XDG_DATA_HOME values
  - `InitWithXDGDataHomeAsFile` - XDG_DATA_HOME pointing to file

- **`filesystem_edge_cases_test.go`** - File system conflicts
  - `InitWithKanukaAsRegularFile` - .kanuka exists as file
  - `InitWithKanukaAsSymlinkToFile` - .kanuka symlink to file
  - `InitWithKanukaAsSymlinkToNonExistentDir` - Broken symlinks

- **`cross_platform_test.go`** - Cross-platform compatibility
  - `InitWithSpecialCharactersInPath` - Special characters in paths
  - `InitWithUnicodeInPath` - Unicode characters in paths

- **`input_validation_test.go`** - Input validation edge cases
  - `InitWithVeryLongProjectName` - Very long project names
  - `InitWithSpecialCharactersInProjectName` - Special chars in names

- **`state_recovery_test.go`** - Recovery and cleanup scenarios
  - `InitWithPartialKanukaDirectory` - Partial .kanuka structure
  - `InitAfterPartialFailure` - Recovery after partial failure
  - `InitIdempotencyAfterFailure` - Multiple init attempts
  - `InitCleanupAfterUserKeyFailure` - Cleanup after key failure

### Shared Utilities (`test/integration/shared/`)

- **`testing_utils.go`** - Common testing functions
  - `SetupTestEnvironment()` - Test environment setup
  - `CaptureOutput()` - Output capture for testing
  - `CreateTestCLI()` - CLI instance creation
  - `VerifyProjectStructure()` - Project structure verification
  - `VerifyUserKeys()` - User key verification

## Running Tests

### Run All Integration Tests
```bash
go test ./test/...
```

### Run Specific Test Categories
```bash
# Init command tests
go test ./test/integration/init/...

# Specific test files
go test ./test/integration/init/basic_test.go
go test ./test/integration/init/permissions_test.go
```

### Run Specific Test Cases
```bash
# Basic functionality
go test -run "TestSecretsInitBasic" ./test/integration/init/

# Specific test scenario
go test -run "TestSecretsInitBasic/InitInEmptyFolder" ./test/integration/init/
```

### Run with Verbose Output
```bash
go test -v ./test/integration/init/...
```

## Test Design Principles

### 1. **Isolation**
- Each test runs in isolated temporary directories
- No shared state between tests
- Complete cleanup after each test

### 2. **Real Integration**
- Tests use actual application code (no mocking)
- Real file system operations
- Real cryptographic operations
- Complete command execution

### 3. **Cross-Platform**
- Tests work on Windows, macOS, and Linux
- Handle platform-specific path differences
- Unicode and special character support

### 4. **Fast and Reliable**
- Lightweight temporary directory usage
- Quick execution (seconds, not minutes)
- Deterministic results

### 5. **Comprehensive Coverage**
- Happy path scenarios
- Error conditions and edge cases
- Recovery and cleanup scenarios
- Input validation

## Benefits of This Organization

### ✅ **Separation of Concerns**
- Unit tests in `cmd/` package alongside source code
- Integration tests in dedicated `test/` directory
- Clear distinction between test types

### ✅ **Build Performance**
- `go build` and `go test ./...` exclude integration tests
- Faster regular development builds
- Integration tests run separately

### ✅ **CI/CD Flexibility**
```bash
# Fast unit tests for every commit
go test ./cmd/... ./internal/...

# Comprehensive integration tests for releases
go test ./test/...
```

### ✅ **Better Organization**
- Tests grouped by feature/command
- Easy to find relevant tests
- Logical structure for maintenance

### ✅ **Scalability**
- Easy to add new test categories
- Shared utilities prevent duplication
- Clear patterns for new tests

## Adding New Tests

### 1. **New Init Test**
Add to appropriate file in `test/integration/init/` or create new category file.

### 2. **New Command Tests**
Create new directory: `test/integration/[command]/`

### 3. **Shared Utilities**
Add common functions to `test/integration/shared/testing_utils.go`

## Test Categories and Expected Behavior

### ✅ **Success Scenarios**
Tests that verify correct functionality:
- Basic initialization
- Cross-platform compatibility  
- State recovery and idempotency
- Valid input handling

### ⚠️ **Error Detection Scenarios**
Tests that verify proper error handling:
- Permission issues
- File conflicts
- Invalid environment variables
- Broken symlinks

**Note**: Tests that verify error detection will show command failures in output, but this is expected behavior demonstrating proper error handling.