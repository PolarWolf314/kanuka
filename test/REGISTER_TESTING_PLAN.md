# Kanuka Secrets Register - Testing Plan

## Overview

This document outlines the comprehensive testing plan for the `kanuka secrets register` command. The register command allows users with access to grant access to other users by encrypting the symmetric key with their public keys.

## Command Analysis

### Command Functionality

The `register` command has three main modes of operation:

1. **User Registration** (`--user`): Register a user whose public key already exists in the project
2. **Custom File Registration** (`--file`): Register using a custom public key file path
3. **Public Key Text Registration** (`--pubkey` + `--user`): Register using public key content directly

### Key Features

- Validates project initialization state
- Supports both OpenSSH and PEM format RSA keys
- Encrypts symmetric key with target user's public key
- Creates `.kanuka` file for target user
- Provides detailed error messages and success feedback
- Supports verbose and debug logging

### Dependencies

- Requires initialized project (`.kanuka` directory structure)
- Requires current user to have access (valid `.kanuka` file and private key)
- Target user must have public key available (for `--user` mode)

## Testing Categories

### 1. Basic Functionality Tests

**File**: `test/integration/register/secrets_register_integration_test.go`

#### Test Cases:

- **RegisterExistingUser**: Register user whose public key exists in project
- **RegisterWithCustomFile**: Register using `--file` flag with valid public key file
- **RegisterWithPubkeyText**: Register using `--pubkey` and `--user` flags
- **RegisterWithVerboseFlag**: Verify verbose output contains `[info]` messages
- **RegisterWithDebugFlag**: Verify debug output contains `[debug]` and `[info]` messages

### 2. Input Validation Tests

**File**: `test/integration/register/register_input_validation_test.go`

#### Test Cases:

- **RegisterWithNoFlags**: Error when no flags provided
- **RegisterWithPubkeyButNoUser**: Error when `--pubkey` provided without `--user`
- **RegisterWithInvalidPubkeyFormat**: Error with malformed public key content
- **RegisterWithNonExistentFile**: Error when `--file` points to non-existent file
- **RegisterWithInvalidFileExtension**: Error when `--file` doesn't end with `.pub`
- **RegisterWithEmptyPubkeyText**: Error when `--pubkey` is empty string
- **RegisterWithEmptyUsername**: Error when `--user` is empty string
- **RegisterWithSpecialCharactersInUsername**: Handle usernames with valid special characters
- **RegisterWithVeryLongUsername**: Handle very long usernames (within limits)

### 3. Project State Tests

**File**: `test/integration/register/register_project_state_test.go`

#### Test Cases:

- **RegisterInUninitializedProject**: Error when project not initialized
- **RegisterWhenCurrentUserHasNoAccess**: Error when current user's `.kanuka` file missing
- **RegisterWhenCurrentUserPrivateKeyMissing**: Error when current user's private key missing
- **RegisterWhenTargetUserAlreadyRegistered**: Handle re-registration of existing user
- **RegisterInCorruptedProjectStructure**: Handle missing directories gracefully
- **RegisterWithCorruptedKanukaFile**: Error when current user's `.kanuka` file corrupted
- **RegisterWithCorruptedPrivateKey**: Error when current user's private key corrupted

### 4. Cryptographic Tests

**File**: `test/integration/register/register_cryptographic_test.go`

#### Test Cases:

- **RegisterWithOpenSSHFormatKey**: Verify OpenSSH format keys work correctly
- **RegisterWithPEMFormatKey**: Verify PEM format keys work correctly
- **RegisterVerifyEncryptedKeyUniqueness**: Verify each user gets unique encrypted symmetric key
- **RegisterVerifyDecryptionWorks**: Verify registered user can decrypt with their private key
- **RegisterWithDifferentKeySizes**: Test with different RSA key sizes (2048, 4096)
- **RegisterCrossFormatCompatibility**: Mix OpenSSH and PEM formats in same project

### 5. Filesystem Edge Cases Tests

**File**: `test/integration/register/register_filesystem_edge_cases_test.go`

#### Test Cases:

- **RegisterWithReadOnlyProjectDirectory**: Error when project directory is read-only
- **RegisterWithReadOnlySecretsDirectory**: Error when secrets directory is read-only
- **RegisterWithReadOnlyPublicKeysDirectory**: Error when public_keys directory is read-only
- **RegisterWithSymlinkedPublicKey**: Handle symlinked public key files
- **RegisterWithRelativeFilePaths**: Handle relative paths in `--file` flag
- **RegisterWithAbsoluteFilePaths**: Handle absolute paths in `--file` flag
- **RegisterInDirectoryWithSpaces**: Handle project paths containing spaces
- **RegisterWithConcurrentAccess**: Handle concurrent register operations

### 6. Cross-Platform Tests

**File**: `test/integration/register/register_cross_platform_test.go`

#### Test Cases:

- **RegisterWithWindowsLineSeparators**: Handle CRLF line endings in public keys
- **RegisterWithUnixLineSeparators**: Handle LF line endings in public keys
- **RegisterWithMixedLineSeparators**: Handle mixed line endings
- **RegisterWithDifferentFilePermissions**: Test various file permission scenarios
- **RegisterWithUnicodeUsernames**: Handle Unicode characters in usernames (if supported)

### 7. Integration Workflow Tests

**File**: `test/integration/register/register_integration_workflow_test.go`

#### Test Cases:

- **InitCreateRegisterWorkflow**: Full workflow from init → create → register
- **MultipleUserRegistrationWorkflow**: Register multiple users sequentially
- **RegisterThenEncryptDecryptWorkflow**: Register user, then verify encrypt/decrypt works
- **RegisterThenRemoveWorkflow**: Register user, then remove access
- **ChainedRegistrationWorkflow**: User A registers User B, User B registers User C
- **RegisterAfterPurgeWorkflow**: Register user after project purge

### 8. Error Handling Tests

**File**: `test/integration/register/register_error_handling_test.go`

#### Test Cases:

- **RegisterWithNetworkInterruption**: Simulate filesystem errors during operation
- **RegisterWithPermissionDenied**: Handle permission denied errors
- **RegisterRecoveryFromPartialFailure**: Verify clean state after partial failures

### 9. User Environment Tests

**File**: `test/integration/register/register_user_environment_test.go`

#### Test Cases:

- **RegisterWithDifferentUserDirectories**: Test with various user directory configurations
- **RegisterWithMissingUserDirectory**: Handle missing user directory gracefully
- **RegisterWithCorruptedUserSettings**: Handle corrupted user settings file

### 10. Output Validation Tests

**File**: `test/integration/register/register_output_validation_test.go`

#### Test Cases:

- **RegisterSuccessMessageFormat**: Verify success message format and content
- **RegisterErrorMessageFormat**: Verify error message format and content
- **RegisterSpinnerBehavior**: Verify spinner shows and stops correctly
- **RegisterColoredOutput**: Verify colored output in success/error messages
- **RegisterQuietMode**: Verify minimal output in quiet mode (if supported)

## Test Structure and Conventions

### Directory Structure

```
test/integration/register/
├── secrets_register_integration_test.go      # Basic functionality
├── register_input_validation_test.go         # Input validation
├── register_project_state_test.go           # Project state edge cases
├── register_cryptographic_test.go           # Cryptographic scenarios
├── register_filesystem_edge_cases_test.go   # Filesystem edge cases
├── register_cross_platform_test.go          # Cross-platform compatibility
├── register_integration_workflow_test.go    # Integration workflows
├── register_error_handling_test.go          # Error handling
├── register_user_environment_test.go        # User environment scenarios
└── register_output_validation_test.go       # Output validation
```

### Testing Conventions

Following the established patterns in the codebase:

1. **Test Function Naming**: `TestSecretsRegister[Category]`
2. **Helper Function Naming**: `test[SpecificScenario]`
3. **Setup Pattern**: Save original working directory and user settings
4. **Cleanup Pattern**: Use `defer os.RemoveAll()` for temporary directories
5. **Environment Setup**: Use `shared.SetupTestEnvironment()`
6. **Output Capture**: Use `shared.CaptureOutput()` for command execution
7. **CLI Creation**: Use `shared.CreateTestCLI()` with appropriate flags
8. **Verification**: Use `shared.VerifyProjectStructure()` where applicable

### Test Utilities Needed

Based on the register command's specific needs:

1. **`shared.CreateTestPublicKey()`**: Generate test RSA public keys
2. **`shared.CreateTestKeyPair()`**: Generate test RSA key pairs
3. **`shared.SetupMultiUserEnvironment()`**: Setup multiple user directories
4. **`shared.VerifyUserRegistration()`**: Verify user was registered correctly
5. **`shared.CreateCorruptedKanukaFile()`**: Create corrupted .kanuka files for testing
6. **`shared.SimulateUserWithAccess()`**: Setup user with existing access

### Key Test Scenarios Priority

#### High Priority (Core Functionality)

1. Basic user registration with existing public key
2. Registration with custom file
3. Registration with public key text
4. Input validation (missing flags, invalid formats)
5. Project state validation (uninitialized, no access)

#### Medium Priority (Edge Cases)

1. Cryptographic format compatibility
2. Filesystem permission issues
3. Error handling and recovery
4. Integration workflows

#### Lower Priority (Advanced Scenarios)

1. Cross-platform compatibility
2. Concurrent access handling
3. Performance under load
4. Unicode and special character handling

## Implementation Notes

### Test Data Management

- Use temporary directories for all test operations
- Generate test keys dynamically rather than using static test data
- Clean up all test artifacts after each test
- Use realistic usernames and project names in tests

### Error Testing Strategy

- Test both expected errors (validation failures) and unexpected errors (filesystem issues)
- Verify error messages are user-friendly and actionable
- Ensure partial failures leave the system in a consistent state

### Integration with Existing Tests

- Leverage existing test utilities from `test/integration/shared/`
- Follow the same patterns as init, create, encrypt, and decrypt tests
- Ensure register tests can be run independently and as part of the full suite

### Performance Considerations

- Keep individual tests fast (< 1 second each when possible)
- Use parallel test execution where safe
- Avoid unnecessary file I/O operations in tests

## Success Criteria

A test implementation will be considered complete when:

1. All test categories have comprehensive coverage
2. Tests follow established conventions and patterns
3. Tests can be run independently and as part of the full suite
4. Tests provide clear, actionable failure messages
5. Tests cover both happy path and error scenarios
6. Tests verify the actual cryptographic functionality works end-to-end
7. Tests are maintainable and well-documented

## Future Considerations

### Potential Enhancements to Test

- Support for different key types (ed25519, ECDSA) when implemented
- Performance testing with large numbers of users
- Security testing for timing attacks or information leakage
- Integration with CI/CD pipeline for automated testing

### Test Maintenance

- Regular review of test coverage as command evolves
- Update tests when new features or flags are added
- Ensure tests remain compatible with dependency updates
- Monitor test execution time and optimize as needed

