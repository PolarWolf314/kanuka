# Integration Tests for Kanuka Secrets Init Command

This directory contains integration tests for the `kanuka secrets init` command.

## Test File: `secrets_init_integration_test.go`

### Overview
The integration tests verify the complete functionality of the `kanuka secrets init` command in various scenarios.

### Test Scenarios

#### 1. `InitInEmptyFolder`
- **Purpose**: Tests successful initialization in an empty folder
- **Verifies**:
  - Command executes successfully
  - `.kanuka` directory structure is created (`.kanuka/public_keys`, `.kanuka/secrets`)
  - RSA key pair is generated in user directory
  - Public key is copied to project directory
  - Encrypted symmetric key is created
  - Warning message about .env files is displayed

#### 2. `InitInAlreadyInitializedFolder`
- **Purpose**: Tests behavior when running init in an already initialized folder
- **Setup**: Pre-creates `.kanuka` directory to simulate existing initialization
- **Verifies**:
  - Command executes without error
  - `.kanuka` directory still exists
  - No additional files are created in `public_keys` or `secrets` directories
  - Command recognizes the project is already initialized

#### 3. `InitWithVerboseFlag`
- **Purpose**: Tests initialization with the `--verbose` flag
- **Verifies**:
  - Command executes successfully
  - Verbose output contains `[info]` log messages
  - Project structure is created correctly
  - All initialization steps are logged

#### 4. `InitWithDebugFlag`
- **Purpose**: Tests initialization with the `--debug` flag
- **Verifies**:
  - Command executes successfully
  - Debug output contains both `[debug]` and `[info]` log messages
  - Project structure is created correctly
  - Detailed debugging information is displayed

### Test Implementation Details

#### Environment Setup
- Each test creates temporary directories for:
  - Project directory (where `.kanuka` will be created)
  - User directory (where RSA keys will be stored)
- Tests override user settings to use temporary directories
- Original working directory and settings are restored after each test

#### Output Capture
- Tests use a custom `captureOutput` function that redirects `os.Stdout` and `os.Stderr`
- This captures all output including logger messages and spinner output
- Output is combined from both stdout and stderr for verification

#### Command Creation
- Tests create isolated command instances to avoid global state issues
- Commands are configured with appropriate flags (verbose/debug)
- Output streams are properly configured for testing

#### Verification Methods
- **Structure verification**: Checks that expected directories and files are created
- **Content verification**: Ensures RSA keys and encrypted symmetric keys are properly generated
- **Output verification**: Validates that appropriate log messages are displayed
- **Behavior verification**: Confirms correct handling of edge cases (already initialized)

### Running the Tests

```bash
# Run all integration tests
go test -v ./cmd -run TestSecretsInitIntegration

# Run a specific test scenario
go test -v ./cmd -run TestSecretsInitIntegration/InitInEmptyFolder
go test -v ./cmd -run TestSecretsInitIntegration/InitInAlreadyInitializedFolder
go test -v ./cmd -run TestSecretsInitIntegration/InitWithVerboseFlag
go test -v ./cmd -run TestSecretsInitIntegration/InitWithDebugFlag
```

### Test Dependencies
- Uses the actual application code (no mocking)
- Creates real files and directories in temporary locations
- Tests the complete integration including:
  - File system operations
  - Cryptographic key generation
  - Configuration management
  - Logging and output formatting

### Cleanup
- All temporary directories and files are automatically cleaned up after each test
- Original working directory and configuration settings are restored
- No persistent state is left behind after test execution