package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
)

// TestSecretsInitIntegration contains integration tests for the `kanuka secrets init` command.
func TestSecretsInitIntegration(t *testing.T) {
	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	// Save original user settings to restore later
	originalUserSettings := configs.UserKanukaSettings

	t.Run("InitInEmptyFolder", func(t *testing.T) {
		testInitInEmptyFolder(t, originalWd, originalUserSettings)
	})

	t.Run("InitInAlreadyInitializedFolder", func(t *testing.T) {
		testInitInAlreadyInitializedFolder(t, originalWd, originalUserSettings)
	})

	t.Run("InitWithVerboseFlag", func(t *testing.T) {
		testInitWithVerboseFlag(t, originalWd, originalUserSettings)
	})

	t.Run("InitWithDebugFlag", func(t *testing.T) {
		testInitWithDebugFlag(t, originalWd, originalUserSettings)
	})
}

// TestSecretsInitEdgeCases contains edge case tests for the `kanuka secrets init` command.
func TestSecretsInitEdgeCases(t *testing.T) {
	// Save original working directory and settings
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings

	// Category 1: File System Permission Issues
	t.Run("InitWithReadOnlyUserDirectory", func(t *testing.T) {
		testInitWithReadOnlyUserDirectory(t, originalWd, originalUserSettings)
	})

	// Category 3: File System Edge Cases
	t.Run("InitWithKanukaAsRegularFile", func(t *testing.T) {
		testInitWithKanukaAsRegularFile(t, originalWd, originalUserSettings)
	})

	t.Run("InitWithKanukaAsSymlinkToFile", func(t *testing.T) {
		testInitWithKanukaAsSymlinkToFile(t, originalWd, originalUserSettings)
	})

	t.Run("InitWithKanukaAsSymlinkToNonExistentDir", func(t *testing.T) {
		testInitWithKanukaAsSymlinkToNonExistentDir(t, originalWd, originalUserSettings)
	})

	// Category 5: Corrupted/Invalid State Recovery
	t.Run("InitWithPartialKanukaDirectory", func(t *testing.T) {
		testInitWithPartialKanukaDirectory(t, originalWd, originalUserSettings)
	})

	t.Run("InitAfterPartialFailure", func(t *testing.T) {
		testInitAfterPartialFailure(t, originalWd, originalUserSettings)
	})

	// Category 6: Environment Variable Edge Cases
	t.Run("InitWithInvalidXDGDataHome", func(t *testing.T) {
		testInitWithInvalidXDGDataHome(t, originalWd, originalUserSettings)
	})

	t.Run("InitWithXDGDataHomeAsFile", func(t *testing.T) {
		testInitWithXDGDataHomeAsFile(t, originalWd, originalUserSettings)
	})

	// Category 10: Cross-Platform Edge Cases
	t.Run("InitWithSpecialCharactersInPath", func(t *testing.T) {
		testInitWithSpecialCharactersInPath(t, originalWd, originalUserSettings)
	})

	t.Run("InitWithUnicodeInPath", func(t *testing.T) {
		testInitWithUnicodeInPath(t, originalWd, originalUserSettings)
	})

	// Category 12: Recovery and Cleanup Scenarios
	t.Run("InitIdempotencyAfterFailure", func(t *testing.T) {
		testInitIdempotencyAfterFailure(t, originalWd, originalUserSettings)
	})

	t.Run("InitCleanupAfterUserKeyFailure", func(t *testing.T) {
		testInitCleanupAfterUserKeyFailure(t, originalWd, originalUserSettings)
	})

	// Category 13: Input Validation Edge Cases
	t.Run("InitWithVeryLongProjectName", func(t *testing.T) {
		testInitWithVeryLongProjectName(t, originalWd, originalUserSettings)
	})

	t.Run("InitWithSpecialCharactersInProjectName", func(t *testing.T) {
		testInitWithSpecialCharactersInProjectName(t, originalWd, originalUserSettings)
	})
}

// testInitInEmptyFolder tests successful initialization in an empty folder.
func testInitInEmptyFolder(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-empty-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory for kanuka settings
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Setup test environment
	setupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Capture real stdout/stderr by redirecting them
	output, err := captureOutput(func() error {
		cmd := createInitCommand(nil, nil)
		return cmd.Execute()
	})
	// Verify command succeeded
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Verify project structure was created
	verifyProjectStructure(t, tempDir)

	// Verify user keys were created
	verifyUserKeys(t, tempUserDir)

	// Verify that the command ran (check for warning message which is always shown)
	if !strings.Contains(output, "Warning: Remember: Never commit .env files") {
		t.Errorf("Expected warning message not found in output: %s", output)
	}
}

// testInitInAlreadyInitializedFolder tests behavior when running init in an already initialized folder.
func testInitInAlreadyInitializedFolder(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-existing-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory for kanuka settings
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Setup test environment
	setupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Pre-create .kanuka directory to simulate already initialized project
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	if err := os.MkdirAll(kanukaDir, 0755); err != nil {
		t.Fatalf("Failed to create .kanuka directory: %v", err)
	}

	// Capture real stdout/stderr by redirecting them
	_, err = captureOutput(func() error {
		cmd := createInitCommand(nil, nil)
		return cmd.Execute()
	})
	// Command should succeed but show already initialized message
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	// Verify that the .kanuka directory still exists and no new files were created
	if _, statErr := os.Stat(kanukaDir); os.IsNotExist(statErr) {
		t.Errorf(".kanuka directory should still exist after running init on already initialized project")
	}

	// Verify no additional files were created (public_keys and secrets dirs should be empty)
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	if publicKeysEntries, readErr := os.ReadDir(publicKeysDir); readErr == nil && len(publicKeysEntries) > 0 {
		t.Errorf("public_keys directory should be empty but contains: %v", publicKeysEntries)
	}

	if secretsEntries, readErr := os.ReadDir(secretsDir); readErr == nil && len(secretsEntries) > 0 {
		t.Errorf("secrets directory should be empty but contains: %v", secretsEntries)
	}
}

// testInitWithVerboseFlag tests initialization with verbose flag.
func testInitWithVerboseFlag(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-verbose-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory for kanuka settings
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Setup test environment
	setupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Capture real stdout/stderr by redirecting them
	output, err := captureOutput(func() error {
		cmd := createTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})
	// Verify command succeeded
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Verify verbose output contains info messages
	if !strings.Contains(output, "[info]") {
		t.Errorf("Expected verbose [info] messages not found in output: %s", output)
	}

	// Verify project structure was created
	verifyProjectStructure(t, tempDir)
}

// testInitWithDebugFlag tests initialization with debug flag.
func testInitWithDebugFlag(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-debug-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory for kanuka settings
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Setup test environment
	setupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Capture real stdout/stderr by redirecting them
	output, err := captureOutput(func() error {
		cmd := createTestCLI("init", nil, nil, false, true)
		return cmd.Execute()
	})
	// Verify command succeeded
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Verify debug output contains debug messages
	if !strings.Contains(output, "[debug]") {
		t.Errorf("Expected debug [debug] messages not found in output: %s", output)
	}

	// Debug should also include info messages
	if !strings.Contains(output, "[info]") {
		t.Errorf("Expected [info] messages not found in debug output: %s", output)
	}

	// Verify project structure was created
	verifyProjectStructure(t, tempDir)
}

// ============================================================================
// Edge Case Test Implementations
// ============================================================================

// Category 1: File System Permission Issues
func testInitWithReadOnlyUserDirectory(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-readonly-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory that we'll make read-only
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-readonly-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Make the user directory read-only
	if err := os.Chmod(tempUserDir, 0444); err != nil {
		t.Fatalf("Failed to make user directory read-only: %v", err)
	}
	// Restore permissions for cleanup
	defer os.Chmod(tempUserDir, 0755)

	setupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Capture output and expect failure
	output, err := captureOutput(func() error {
		cmd := createTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should fail due to permission issues
	if err == nil {
		t.Errorf("Expected command to fail due to read-only user directory, but it succeeded")
		t.Errorf("Output: %s", output)
	}

	// Should contain error message about permissions or directory creation
	if !strings.Contains(output, "failed") && !strings.Contains(output, "permission") {
		t.Errorf("Expected permission-related error message, got: %s", output)
	}
}

// Category 3: File System Edge Cases
func testInitWithKanukaAsRegularFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-file-conflict-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	setupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Create a regular file named .kanuka
	kanukaFile := filepath.Join(tempDir, ".kanuka")
	if err := os.WriteFile(kanukaFile, []byte("this is a file, not a directory"), 0644); err != nil {
		t.Fatalf("Failed to create .kanuka file: %v", err)
	}

	// Capture output and expect failure
	output, err := captureOutput(func() error {
		cmd := createTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should fail due to file conflict
	if err == nil {
		t.Errorf("Expected command to fail due to .kanuka file conflict, but it succeeded")
		t.Errorf("Output: %s", output)
	}

	// Should contain error message about .kanuka not being a directory
	if !strings.Contains(output, "not a directory") && !strings.Contains(output, "exists") {
		t.Errorf("Expected error about .kanuka not being a directory, got: %s", output)
	}
}

func testInitWithKanukaAsSymlinkToFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-symlink-file-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	setupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Create a regular file and symlink .kanuka to it
	targetFile := filepath.Join(tempDir, "target-file")
	if err := os.WriteFile(targetFile, []byte("target file content"), 0644); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	kanukaSymlink := filepath.Join(tempDir, ".kanuka")
	if err := os.Symlink(targetFile, kanukaSymlink); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Capture output and expect failure
	output, err := captureOutput(func() error {
		cmd := createTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should fail due to symlink pointing to file
	if err == nil {
		t.Errorf("Expected command to fail due to .kanuka symlink to file, but it succeeded")
		t.Errorf("Output: %s", output)
	}

	// Should contain error message about .kanuka not being a directory
	if !strings.Contains(output, "not a directory") && !strings.Contains(output, "exists") {
		t.Errorf("Expected error about .kanuka not being a directory, got: %s", output)
	}
}

func testInitWithKanukaAsSymlinkToNonExistentDir(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-symlink-nonexistent-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	setupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Create symlink to non-existent directory
	kanukaSymlink := filepath.Join(tempDir, ".kanuka")
	nonExistentTarget := filepath.Join(tempDir, "non-existent-dir")
	if err := os.Symlink(nonExistentTarget, kanukaSymlink); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Capture output and expect failure
	output, err := captureOutput(func() error {
		cmd := createTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should fail due to broken symlink
	if err == nil {
		t.Errorf("Expected command to fail due to broken .kanuka symlink, but it succeeded")
		t.Errorf("Output: %s", output)
	}

	// Should contain error message about checking directory or not existing
	if !strings.Contains(output, "failed") {
		t.Errorf("Expected error about failed directory check, got: %s", output)
	}
}
// Category 5: Corrupted/Invalid State Recovery
func testInitWithPartialKanukaDirectory(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-partial-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	setupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Create partial .kanuka directory structure (missing some subdirectories)
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	if err := os.Mkdir(kanukaDir, 0755); err != nil {
		t.Fatalf("Failed to create .kanuka directory: %v", err)
	}

	// Create only one of the required subdirectories
	secretsDir := filepath.Join(kanukaDir, "secrets")
	if err := os.Mkdir(secretsDir, 0755); err != nil {
		t.Fatalf("Failed to create secrets directory: %v", err)
	}
	// Deliberately omit public_keys directory

	// Capture output - should detect existing .kanuka and report already initialized
	output, err := captureOutput(func() error {
		cmd := createTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should detect existing .kanuka directory
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Should contain message about already being initialized
	if !strings.Contains(output, "already been initialized") {
		t.Errorf("Expected message about already initialized, got: %s", output)
	}
}

func testInitAfterPartialFailure(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-after-failure-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	setupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Simulate partial failure by creating .kanuka directory but making it read-only
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	if err := os.Mkdir(kanukaDir, 0755); err != nil {
		t.Fatalf("Failed to create .kanuka directory: %v", err)
	}

	// First attempt should detect existing .kanuka
	output1, err1 := captureOutput(func() error {
		cmd := createTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})

	// Should report already initialized
	if err1 != nil {
		t.Errorf("First init attempt failed unexpectedly: %v", err1)
	}
	if !strings.Contains(output1, "already been initialized") {
		t.Errorf("Expected already initialized message, got: %s", output1)
	}

	// Remove the .kanuka directory to simulate cleanup
	if err := os.RemoveAll(kanukaDir); err != nil {
		t.Fatalf("Failed to remove .kanuka directory: %v", err)
	}

	// Second attempt should succeed
	output2, err2 := captureOutput(func() error {
		cmd := createTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})

	if err2 != nil {
		t.Errorf("Second init attempt failed: %v", err2)
		t.Errorf("Output: %s", output2)
	}

	// Should contain success message
	if !strings.Contains(output2, "initialized successfully") {
		t.Errorf("Expected success message, got: %s", output2)
	}

	// Verify project structure was created
	verifyProjectStructure(t, tempDir)
}

// Category 6: Environment Variable Edge Cases
func testInitWithInvalidXDGDataHome(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-invalid-xdg-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Save original XDG_DATA_HOME
	originalXDG := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", originalXDG)

	// Set XDG_DATA_HOME to non-existent path with invalid characters
	invalidPath := "/non/existent/path/with\x00null/chars"
	os.Setenv("XDG_DATA_HOME", invalidPath)

	// Create temporary user directory (will be overridden by configs)
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	setupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Capture output - may succeed or fail depending on how the system handles invalid paths
	output, err := captureOutput(func() error {
		cmd := createTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})

	// The behavior may vary by system, but it should either:
	// 1. Succeed by falling back to default paths, or
	// 2. Fail with a clear error message
	if err != nil {
		// If it fails, should have a clear error message
		if !strings.Contains(output, "failed") {
			t.Errorf("Expected clear error message for invalid XDG_DATA_HOME, got: %s", output)
		}
	} else {
		// If it succeeds, should have created the project structure
		if !strings.Contains(output, "initialized successfully") {
			t.Errorf("Expected success message, got: %s", output)
		}
	}
}

func testInitWithXDGDataHomeAsFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-xdg-file-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a file to use as XDG_DATA_HOME
	xdgFile := filepath.Join(tempDir, "xdg-data-file")
	if err := os.WriteFile(xdgFile, []byte("this is a file"), 0644); err != nil {
		t.Fatalf("Failed to create XDG file: %v", err)
	}

	// Save original XDG_DATA_HOME
	originalXDG := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", originalXDG)

	// Set XDG_DATA_HOME to the file
	os.Setenv("XDG_DATA_HOME", xdgFile)

	// Create temporary user directory (will be overridden by configs)
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	setupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Capture output and expect failure
	output, err := captureOutput(func() error {
		cmd := createTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should fail due to XDG_DATA_HOME being a file
	if err == nil {
		t.Errorf("Expected command to fail due to XDG_DATA_HOME being a file, but it succeeded")
		t.Errorf("Output: %s", output)
	}

	// Should contain error message about directory creation or path issues
	if !strings.Contains(output, "failed") {
		t.Errorf("Expected error message about failed directory creation, got: %s", output)
	}
}// Category 10: Cross-Platform Edge Cases
func testInitWithSpecialCharactersInPath(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory with special characters (but valid for filesystem)
	specialName := "kanuka-test-init-special-chars-!@#$%^&()_+-=[]{}|;',."
	tempDir, err := os.MkdirTemp("", specialName)
	if err != nil {
		t.Fatalf("Failed to create temp directory with special chars: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	setupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Capture output
	output, err := captureOutput(func() error {
		cmd := createTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should succeed
	if err != nil {
		t.Errorf("Command failed with special characters in path: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Should contain success message
	if !strings.Contains(output, "initialized successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// Verify project structure was created
	verifyProjectStructure(t, tempDir)
}

func testInitWithUnicodeInPath(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory with Unicode characters
	unicodeName := "kanuka-test-init-unicode-测试-🔐-café"
	tempDir, err := os.MkdirTemp("", unicodeName)
	if err != nil {
		t.Fatalf("Failed to create temp directory with Unicode: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	setupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Capture output
	output, err := captureOutput(func() error {
		cmd := createTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should succeed
	if err != nil {
		t.Errorf("Command failed with Unicode in path: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Should contain success message
	if !strings.Contains(output, "initialized successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// Verify project structure was created
	verifyProjectStructure(t, tempDir)
}

// Category 12: Recovery and Cleanup Scenarios
func testInitIdempotencyAfterFailure(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-idempotency-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	setupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// First init should succeed
	output1, err1 := captureOutput(func() error {
		cmd := createTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})

	if err1 != nil {
		t.Errorf("First init failed: %v", err1)
		t.Errorf("Output: %s", output1)
	}

	// Second init should detect existing initialization
	output2, err2 := captureOutput(func() error {
		cmd := createTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})

	if err2 != nil {
		t.Errorf("Second init failed: %v", err2)
	}

	// Should contain message about already being initialized
	if !strings.Contains(output2, "already been initialized") {
		t.Errorf("Expected already initialized message, got: %s", output2)
	}

	// Project structure should still be intact
	verifyProjectStructure(t, tempDir)
}

func testInitCleanupAfterUserKeyFailure(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-cleanup-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	setupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Create a file where the keys directory should be to cause failure
	keysPath := filepath.Join(tempUserDir, "keys")
	if err := os.WriteFile(keysPath, []byte("blocking file"), 0644); err != nil {
		t.Fatalf("Failed to create blocking file: %v", err)
	}

	// Capture output and expect failure
	output, err := captureOutput(func() error {
		cmd := createTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should fail
	if err == nil {
		t.Errorf("Expected command to fail due to blocked keys directory, but it succeeded")
		t.Errorf("Output: %s", output)
	}

	// Should contain error message
	if !strings.Contains(output, "failed") {
		t.Errorf("Expected error message, got: %s", output)
	}

	// .kanuka directory should not exist (cleanup should have occurred)
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	if _, err := os.Stat(kanukaDir); !os.IsNotExist(err) {
		t.Errorf("Expected .kanuka directory to not exist after failure, but it does")
	}
}

// Category 13: Input Validation Edge Cases
func testInitWithVeryLongProjectName(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory with very long name (but within filesystem limits)
	longName := strings.Repeat("a", 100) // 100 characters should be safe on most filesystems
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-long-"+longName)
	if err != nil {
		t.Fatalf("Failed to create temp directory with long name: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	setupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Capture output
	output, err := captureOutput(func() error {
		cmd := createTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should succeed (100 chars is reasonable)
	if err != nil {
		t.Errorf("Command failed with long project name: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Should contain success message
	if !strings.Contains(output, "initialized successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// Verify project structure was created
	verifyProjectStructure(t, tempDir)
}

func testInitWithSpecialCharactersInProjectName(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory with special characters that are valid in filenames
	specialName := "kanuka-test-init-special-project-name_with-dashes.and.dots"
	tempDir, err := os.MkdirTemp("", specialName)
	if err != nil {
		t.Fatalf("Failed to create temp directory with special project name: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	setupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Capture output
	output, err := captureOutput(func() error {
		cmd := createTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should succeed
	if err != nil {
		t.Errorf("Command failed with special characters in project name: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Should contain success message
	if !strings.Contains(output, "initialized successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// Verify project structure was created
	verifyProjectStructure(t, tempDir)
}