# Kanuka Secrets Create Command - Integration Test Plan

## Command Overview

The `kanuka secrets create` command creates and adds a user's public key to a project, enabling them to request access to encrypted secrets. It performs the following operations:

1. **Validates project state** - Ensures kanuka is initialized in the project
2. **Ensures user settings** - Creates user directories and gets username
3. **Checks for existing keys** - Prevents overwriting unless `--force` flag is used
4. **Generates RSA key pair** - Creates 2048-bit RSA private/public key pair
5. **Saves keys locally** - Stores private key in user's data directory (`~/.local/share/kanuka/keys/`)
6. **Copies public key to project** - Places public key in `.kanuka/public_keys/{username}.pub`
7. **Cleans up existing access** - Removes any existing `.kanuka/secrets/{username}.kanuka` file
8. **Provides next steps** - Shows instructions for gaining access

## Test Categories

### 1. Basic Functionality Tests

- **Create in uninitialized project** - Should fail with clear error message
- **Create in initialized project (new user)** - Should succeed and create all necessary files
- **Create when user already has keys** - Should fail without `--force` flag
- **Create with force flag** - Should overwrite existing keys and remove old access

### 2. File System Operations Tests

- **Key generation and storage** - Verify private/public key pair creation in correct locations
- **Public key copying** - Verify public key is copied to project directory with correct permissions
- **Directory creation** - Verify user directories are created if they don't exist
- **File permissions** - Verify correct permissions on created files (private key: 0600, public key: 0644)
- **Cleanup operations** - Verify existing `.kanuka` files are properly removed

### 3. Project State Tests

- **Multiple project support** - Create keys for different projects, verify isolation
- **Project name handling** - Test with various project directory names and structures
- **Existing project structure** - Test when `.kanuka` directories already exist
- **Corrupted project state** - Test behavior with malformed `.kanuka` directory

### 4. User Settings and Environment Tests

- **Username detection** - Test with different system usernames
- **Custom data directories** - Test with custom `XDG_DATA_HOME` settings
- **User directory permissions** - Test when user directories have restricted permissions
- **Concurrent access** - Test behavior when multiple processes access user directories

### 5. Force Flag Tests

- **Force with existing keys** - Verify old keys are replaced
- **Force with existing access** - Verify old `.kanuka` file is removed
- **Force without existing keys** - Should work same as normal create
- **Force flag warnings** - Verify appropriate warnings are shown

### 6. Error Handling Tests

- **Read-only project directory** - Should fail gracefully
- **Read-only user directory** - Should fail gracefully
- **Invalid project structure** - Should handle corrupted `.kanuka` directories
- **Permission denied scenarios** - Test various permission restriction scenarios

### 7. Key Format and Cryptographic Tests

- **RSA key generation** - Verify 2048-bit RSA keys are generated
- **PEM format validation** - Verify keys are in correct PEM format
- **Key pair matching** - Verify private and public keys are mathematically related
- **Key uniqueness** - Verify each generation creates unique keys

### 8. Cross-Platform Tests

- **Windows path handling** - Test on Windows with `%APPDATA%` paths
- **Unix path handling** - Test on Unix systems with XDG directories
- **Path separator handling** - Test with different path separators
- **Special characters in paths** - Test with spaces and special characters in project paths

### 9. Integration with Other Commands Tests

- **Create then register workflow** - Verify created keys work with register command
- **Create then encrypt workflow** - Verify workflow after gaining access
- **Multiple users workflow** - Test multiple users creating keys in same project

### 10. Output and User Experience Tests

- **Success messages** - Verify clear success messages with file paths
- **Error messages** - Verify clear, actionable error messages
- **Progress indicators** - Test spinner and progress feedback
- **Verbose mode** - Test detailed logging output
- **Instructions display** - Verify next steps are clearly communicated

## Test Implementation Structure

```
test/integration/create/
├── secrets_create_integration_test.go     # Basic functionality tests
├── create_filesystem_test.go              # File system operations
├── create_project_state_test.go           # Project state edge cases
├── create_user_environment_test.go        # User settings and environment
├── create_force_flag_test.go              # Force flag scenarios
├── create_error_handling_test.go          # Error scenarios
├── create_cryptographic_test.go           # Key generation and validation
├── create_cross_platform_test.go          # Platform-specific tests
├── create_integration_workflow_test.go    # Integration with other commands
└── create_output_validation_test.go       # Output and UX validation
```

## Key Test Scenarios

### High Priority

1. **Basic create in initialized project** - Core functionality
2. **Create with existing keys (should fail)** - Prevents accidental overwrites
3. **Force flag functionality** - Allows intentional overwrites
4. **Uninitialized project handling** - Clear error messaging
5. **File permissions and security** - Ensures keys are properly protected

### Medium Priority

6. **Multiple project isolation** - Ensures project separation
7. **Cross-platform compatibility** - Works on different operating systems
8. **Error handling** - Graceful failure modes
9. **User directory creation** - Handles first-time users
10. **Integration workflows** - Works with other commands

### Lower Priority

11. **Edge cases** - Unusual but valid scenarios
12. **Performance** - Large projects or many users
13. **Concurrent access** - Multiple simultaneous operations
14. **Recovery scenarios** - Handling partial failures

## Success Criteria

- All basic functionality works correctly
- Security is maintained (proper file permissions, key isolation)
- Clear error messages for all failure scenarios
- Cross-platform compatibility
- Integration with existing kanuka workflow
- Comprehensive test coverage for edge cases

