# Integration Tests for Kanuka Secrets Init Command

This directory contains comprehensive integration tests for the `kanuka secrets init` command, organized by test category for better maintainability.

## Test File Organization

The integration tests have been split into focused test files based on the type of scenarios they test:

### Core Integration Tests

#### `secrets_init_basic_test.go`
- **Purpose**: Tests basic functionality and common usage scenarios
- **Test Function**: `TestSecretsInitBasic`
- **Scenarios**:
  - `InitInEmptyFolder`: Successful initialization in an empty folder
  - `InitInAlreadyInitializedFolder`: Behavior when running init in already initialized folder
  - `InitWithVerboseFlag`: Initialization with `--verbose` flag
  - `InitWithDebugFlag`: Initialization with `--debug` flag

### Edge Case Tests

#### `secrets_init_permissions_test.go`
- **Purpose**: Tests file system permission-related edge cases
- **Test Function**: `TestSecretsInitPermissions`
- **Scenarios**:
  - `InitWithReadOnlyUserDirectory`: Tests behavior when user directory is read-only

#### `secrets_init_filesystem_edge_cases_test.go`
- **Purpose**: Tests file system conflicts and edge cases
- **Test Function**: `TestSecretsInitFilesystemEdgeCases`
- **Scenarios**:
  - `InitWithKanukaAsRegularFile`: When `.kanuka` exists as a file instead of directory
  - `InitWithKanukaAsSymlinkToFile`: When `.kanuka` is a symlink pointing to a file
  - `InitWithKanukaAsSymlinkToNonExistentDir`: Tests broken symlinks

#### `secrets_init_state_recovery_test.go`
- **Purpose**: Tests recovery from corrupted/invalid states and cleanup scenarios
- **Test Function**: `TestSecretsInitStateRecovery`
- **Scenarios**:
  - `InitWithPartialKanukaDirectory`: Tests partial `.kanuka` directory structure
  - `InitAfterPartialFailure`: Tests recovery after partial failure
  - `InitIdempotencyAfterFailure`: Tests running init multiple times
  - `InitCleanupAfterUserKeyFailure`: Tests cleanup after key generation failure

#### `secrets_init_environment_test.go`
- **Purpose**: Tests environment variable edge cases
- **Test Function**: `TestSecretsInitEnvironment`
- **Scenarios**:
  - `InitWithInvalidXDGDataHome`: Tests invalid `XDG_DATA_HOME` values
  - `InitWithXDGDataHomeAsFile`: Tests when `XDG_DATA_HOME` points to a file

#### `secrets_init_cross_platform_test.go`
- **Purpose**: Tests cross-platform compatibility edge cases
- **Test Function**: `TestSecretsInitCrossPlatform`
- **Scenarios**:
  - `InitWithSpecialCharactersInPath`: Tests special characters in directory paths
  - `InitWithUnicodeInPath`: Tests Unicode characters in directory paths

#### `secrets_init_input_validation_test.go`
- **Purpose**: Tests input validation edge cases
- **Test Function**: `TestSecretsInitInputValidation`
- **Scenarios**:
  - `InitWithVeryLongProjectName`: Tests very long project names (100+ chars)
  - `InitWithSpecialCharactersInProjectName`: Tests special chars in project names


## Test Implementation Details

### Environment Setup
- Each test creates temporary directories for:
  - Project directory (where `.kanuka` will be created)
  - User directory (where RSA keys will be stored)
- Tests override user settings to use temporary directories
- Original working directory and settings are restored after each test

### Output Capture
- Tests use a custom `captureOutput` function that redirects `os.Stdout` and `os.Stderr`
- This captures all output including logger messages and spinner output
- Output is combined from both stdout and stderr for verification

### Command Creation
- Tests create isolated command instances to avoid global state issues
- Commands are configured with appropriate flags (verbose/debug)
- Output streams are properly configured for testing

### Verification Methods
- **Structure verification**: Checks that expected directories and files are created
- **Content verification**: Ensures RSA keys and encrypted symmetric keys are properly generated
- **Output verification**: Validates that appropriate log messages are displayed
- **Behavior verification**: Confirms correct handling of edge cases

## Running the Tests

### Run All Init Tests
```bash
# Run all init-related tests
go test -v ./cmd -run "TestSecretsInit"
```

### Run Specific Test Categories
```bash
# Basic functionality tests
go test -v ./cmd -run "TestSecretsInitBasic"

# Permission-related edge cases
go test -v ./cmd -run "TestSecretsInitPermissions"

# Filesystem edge cases
go test -v ./cmd -run "TestSecretsInitFilesystemEdgeCases"

# State recovery and cleanup tests
go test -v ./cmd -run "TestSecretsInitStateRecovery"

# Environment variable edge cases
go test -v ./cmd -run "TestSecretsInitEnvironment"

# Cross-platform compatibility tests
go test -v ./cmd -run "TestSecretsInitCrossPlatform"

# Input validation edge cases
go test -v ./cmd -run "TestSecretsInitInputValidation"
```

### Run Specific Test Scenarios
```bash
# Basic scenarios
go test -v ./cmd -run "TestSecretsInitBasic/InitInEmptyFolder"
go test -v ./cmd -run "TestSecretsInitBasic/InitWithVerboseFlag"

# Edge case scenarios
go test -v ./cmd -run "TestSecretsInitPermissions/InitWithReadOnlyUserDirectory"
go test -v ./cmd -run "TestSecretsInitFilesystemEdgeCases/InitWithKanukaAsRegularFile"
go test -v ./cmd -run "TestSecretsInitStateRecovery/InitIdempotencyAfterFailure"
```

## Test Categories and Expected Behavior

### ✅ Passing Tests (Success Scenarios)
These tests verify the application works correctly under edge conditions:
- Basic initialization scenarios
- Cross-platform compatibility (Unicode, special characters)
- State recovery and idempotency
- Input validation within reasonable limits

### ⚠️ Correctly Failing Tests (Error Detection)
These tests verify the application properly detects and handles error conditions:
- File system permission issues
- File/directory conflicts
- Invalid environment variables
- Broken symlinks

**Note**: Tests that "fail" are actually verifying that the application correctly detects error conditions and exits with appropriate error messages. This is the expected and desired behavior.

## Test Dependencies
- Uses the actual application code (no mocking)
- Creates real files and directories in temporary locations
- Tests the complete integration including:
  - File system operations
  - Cryptographic key generation
  - Configuration management
  - Logging and output formatting

## Cleanup
- All temporary directories and files are automatically cleaned up after each test
- Original working directory and configuration settings are restored
- No persistent state is left behind after test execution

## Benefits of This Organization

1. **Focused Testing**: Each file focuses on a specific category of tests
2. **Easier Maintenance**: Easier to locate and modify tests for specific scenarios
3. **Better Test Discovery**: Clear naming makes it easy to find relevant tests
4. **Parallel Development**: Different developers can work on different test categories
5. **Selective Testing**: Can run only the test categories relevant to changes being made