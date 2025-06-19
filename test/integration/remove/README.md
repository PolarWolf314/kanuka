# Remove Command Tests

This directory contains integration tests for the `kanuka secrets remove` command, which is used to remove a user's access to the secret store.

## Test Files

- `remove_basic_test.go`: Basic functionality tests for the remove command
- `remove_filesystem_edge_cases_test.go`: Tests for filesystem edge cases
- `remove_project_state_test.go`: Tests for project state requirements
- `secrets_remove_integration_test.go`: Integration tests with other commands
- `remove_multiple_users_test.go`: Tests for removing one user from multiple users
- `remove_permission_denied_test.go`: Tests for permission denied scenarios
- `remove_concurrent_access_test.go`: Tests for concurrent file access scenarios
- `remove_large_number_users_test.go`: Performance tests with a large number of users

## Running Tests

To run all tests for the remove command:

```bash
cd test/integration/remove
go test -v
```

To run a specific test:

```bash
cd test/integration/remove
go test -v -run TestRemoveCommand_SuccessfulRemoval
```

## Test Coverage

The tests in this directory cover:

1. Basic functionality:
   - Command flag requirements
   - User not found handling
   - Successful removal

2. Filesystem edge cases:
   - Only public key file exists
   - Only kanuka key file exists
   - Read-only files

3. Project state requirements:
   - Without initialization
   - Non-kanuka project

4. Integration with other commands:
   - Full workflow with init and register commands

5. Multiple users:
   - Removing one user from a project with multiple users
   - Verifying other users' files remain intact

6. Permission scenarios:
   - Handling directories with no write permissions
   - Graceful error reporting

7. Concurrent access:
   - Handling files being accessed by another process
   - Partial removal when some files are locked

8. Performance:
   - Handling a large number of users
   - Measuring removal time

## Test Plan

For a comprehensive test plan, see [TEST_PLAN.md](./TEST_PLAN.md) in this directory.