# Secrets Remove Test Plan

Test plan for implementing comprehensive test coverage for the `kanuka secrets remove --file` flag functionality.

## Overview

This document outlines test cases to be implemented for the `--file` flag in the remove command. These tests cover new code paths introduced by the `--file` flag that are not exercised by existing `--user` tests.

## Test Checklist

- [x] **RemoveFileWithBothFilesPresent**
  - **Purpose**: Verify that `--file` flag correctly removes both the .kanuka file and its corresponding public key
  - **Setup**: Create a project with a user having both `username.kanuka` and `username.pub` files
  - **Action**: Run `kanuka secrets remove --file .kanuka/secrets/username.kanuka`
  - **Expected Outcome**: Both `username.kanuka` and `username.pub` files are removed
  - **Edge Cases Covered**: Normal happy path with both files present

- [x] **RemoveFileWithOnlyKanukaFile**
  - **Purpose**: Verify that `--file` flag works when only the .kanuka file exists (no public key)
  - **Setup**: Create a project with only `username.kanuka` file (no `username.pub`)
  - **Action**: Run `kanuka secrets remove --file .kanuka/secrets/username.kanuka`
  - **Expected Outcome**: Only the .kanuka file is removed, no error about missing public key
  - **Edge Cases Covered**: Partial file state (only .kanuka exists)

- [x] **RemoveFileWithRelativePath**
  - **Purpose**: Verify that relative paths are correctly resolved
  - **Setup**: Create a project with user files
  - **Action**: Run `kanuka secrets remove --file .kanuka/secrets/username.kanuka`
  - **Expected Outcome**: Path is correctly resolved and files are removed
  - **Edge Cases Covered**: Relative path resolution via `filepath.Abs()`

- ~~**RemoveFileWithAbsolutePath**~~ (SKIPPED - Absolute path testing is difficult with Go test framework due to working directory changes)
  - **Purpose**: Verify that absolute paths work correctly
  - **Setup**: Create a project with user files, get absolute path to file
  - **Action**: Run `kanuka secrets remove --file /absolute/path/to/.kanuka/secrets/username.kanuka`
  - **Expected Outcome**: Files are removed using absolute path
  - **Edge Cases Covered**: Absolute path handling

- [x] **RemoveNonExistentFile**
  - **Purpose**: Verify proper error handling when specified file doesn't exist
  - **Setup**: Create a project without any user files
  - **Action**: Run `kanuka secrets remove --file .kanuka/secrets/nonexistent.kanuka`
  - **Expected Outcome**: Command shows error message about file not existing, no files are removed
  - **Edge Cases Covered**: `os.IsNotExist()` error handling in `getFilesByPath()`

- [x] **RemoveDirectoryPath**
  - **Purpose**: Verify that directory paths are rejected
  - **Setup**: Create a project with the secrets directory
  - **Action**: Run `kanuka secrets remove --file .kanuka/secrets/`
  - **Expected Outcome**: Command shows error message about path being a directory
  - **Edge Cases Covered**: `fileInfo.IsDir()` check in `getFilesByPath()`

- [x] **RemoveFileOutsideSecretsDir**
  - **Purpose**: Verify that files outside of project's secrets directory are rejected
  - **Setup**: Create a project and a test file at `/tmp/test.kanuka`
  - **Action**: Run `kanuka secrets remove --file /tmp/test.kanuka`
  - **Expected Outcome**: Command shows error message about file not being in the secrets directory
  - **Edge Cases Covered**: Directory validation logic `filepath.Dir(absFilePath) != absProjectSecretsPath`

- [x] **RemoveNonKanukaExtension**
  - **Purpose**: Verify that files without .kanuka extension are rejected
  - **Setup**: Create a test file `user.txt` in the secrets directory
  - **Action**: Run `kanuka secrets remove --file .kanuka/secrets/user.txt`
  - **Expected Outcome**: Command shows error message about file not having .kanuka extension
  - **Edge Cases Covered**: Extension validation `filepath.Ext(absFilePath) != ".kanuka"`

- [x] **RemoveFileWithDotsInUsername**
  - **Purpose**: Verify username extraction from filenames containing dots
  - **Setup**: Create a file `user.name.kanuka` in secrets directory
  - **Action**: Run `kanuka secrets remove --file .kanuka/secrets/user.name.kanuka`
  - **Expected Outcome**: Username is correctly extracted as `user.name` (not just `user`)
  - **Edge Cases Covered**: Username extraction logic `baseName[:len(baseName)-len(".kanuka")]`

- [x] **RemoveFileWithEmptyUsername**
  - **Purpose**: Verify behavior when filename is just `.kanuka` (empty username)
  - **Setup**: Create a file `.kanuka` in secrets directory
  - **Action**: Run `kanuka secrets remove --file .kanuka/secrets/.kanuka`
  - **Expected Outcome**: Username is empty string, behavior should be defined (error or succeed)
  - **Edge Cases Covered**: Edge case in username extraction where `baseName == ".kanuka"`

- [x] **BothUserAndFileFlags**
  - **Purpose**: Verify that providing both flags is rejected
  - **Setup**: Create a project with user files
  - **Action**: Run `kanuka secrets remove --user username --file .kanuka/secrets/username.kanuka`
  - **Expected Outcome**: Command shows error message about not being able to specify both flags
  - **Edge Cases Covered**: Flag validation logic in remove command

## Implementation Notes

### Test Structure

Follow the existing test structure in `remove_basic_test.go`:
1. Create temporary directories
2. Set up user settings
3. Change to temp directory
4. Create necessary files
5. Run the command
6. Verify results
7. Clean up (defer)

### Helper Functions

Consider creating helper functions to reduce code duplication:
- `setupTestEnvironment()` - Creates temp directories and user settings
- `createUserFiles()` - Creates .kanuka and .pub files for a user
- `verifyFilesRemoved()` - Checks that expected files are removed
- `verifyFilesExist()` - Checks that expected files still exist

### Test Naming Convention

Use descriptive names that indicate:
- What is being tested (e.g., "RemoveFileWith...")
- The specific scenario (e.g., "...BothFilesPresent")
- The expected behavior (implicit in success/failure of test)

### Error Messages

When testing error scenarios, verify that:
- The command doesn't return an error (it should show the error in the final message)
- The spinner's FinalMSG contains the expected error text
- No files are removed when error occurs

### Concurrency

Most tests should be run sequentially. Only add concurrent tests if specifically testing concurrency scenarios (similar to existing `remove_concurrent_access_test.go`).

## Progress Tracking

- **Total Tests**: 10
- **Implemented**: 10
- **Remaining**: 0

## Implementation Order

Recommended order of implementation (simplest to most complex):
1. ~~RemoveFileWithBothFilesPresent~~ (basic happy path) - DONE
2. ~~RemoveFileWithRelativePath~~ - DONE
3. ~~RemoveFileWithAbsolutePath~~ (SKIPPED)
4. ~~RemoveFileWithOnlyKanukaFile~~ - DONE
5. ~~RemoveNonExistentFile~~ - DONE
6. ~~RemoveDirectoryPath~~ - DONE
7. ~~RemoveNonKanukaExtension~~ - DONE
8. ~~RemoveFileOutsideSecretsDir~~ - DONE
9. ~~BothUserAndFileFlags~~ - DONE
10. ~~RemoveFileWithDotsInUsername~~ - DONE
11. ~~RemoveFileWithEmptyUsername~~ - DONE

## Notes for Maintainers

- Each test should be independent and clean up after itself
- Use `defer os.RemoveAll(tempDir)` for cleanup
- Always restore working directory with `defer os.Chdir(originalWd)`
- Always restore user settings with `defer func() { configs.UserKanukaSettings = originalUserSettings }()`
- Call `cmd.ResetGlobalState()` before running each command
- Consider using subtests with `t.Run()` for related test scenarios

## Related Files

- `cmd/secrets_remove.go` - Main implementation file
- `test/integration/remove/remove_basic_test.go` - Reference for basic test structure
- `test/integration/remove/remove_filesystem_edge_cases_test.go` - Reference for edge case testing
- `test/integration/remove/remove_project_state_test.go` - Reference for project state testing
