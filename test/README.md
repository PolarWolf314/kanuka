# Tests

This directory contains integration tests for the Kanuka secrets management tool. The tests verify the functionality of all major commands and edge cases across different platforms and environments.

## Running Tests

```bash
# Run all tests
go test ./test/...

# Run tests with verbose output
go test -v ./test/...

# Run specific command categories
go test ./test/integration/init/...
go test ./test/integration/create/...
go test ./test/integration/register/...
go test ./test/integration/encrypt/...
go test ./test/integration/decrypt/...

# Run a specific test file
go test ./test/integration/init/basic_test.go
```

## Test Categories

### Init Tests
Project initialization functionality including basic setup, cross-platform behavior, environment handling, filesystem edge cases, input validation, permissions, and state recovery.

### Create Tests
Secret creation functionality covering project state management, cross-platform compatibility, integration workflows, error handling, output validation, filesystem operations, force flag behavior, cryptographic operations, and user environment scenarios.

### Register Tests
Secret registration functionality including cross-platform support, cryptographic operations, project state management, user environment handling, error scenarios, input/output validation, integration workflows, and filesystem edge cases.

### Encrypt Tests
Encryption command functionality covering core integration, filesystem edge cases, permission handling, and project state scenarios.

### Decrypt Tests
Decryption command functionality including core integration, content validation, cryptographic edge cases, filesystem scenarios, and project state handling.

## Test Structure

- `test/integration/shared/` - Common test utilities and helper functions
- `test/integration/init/` - Project initialization tests
- `test/integration/create/` - Secret creation tests  
- `test/integration/register/` - Secret registration tests
- `test/integration/encrypt/` - Encryption command tests
- `test/integration/decrypt/` - Decryption command tests

