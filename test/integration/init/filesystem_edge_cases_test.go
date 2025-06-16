package init_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsInitFilesystemEdgeCases contains filesystem edge case tests for the `kanuka secrets init` command.
func TestSecretsInitFilesystemEdgeCases(t *testing.T) {
	// Save original working directory and settings
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings

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

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Create a regular file named .kanuka
	kanukaFile := filepath.Join(tempDir, ".kanuka")
	if err := os.WriteFile(kanukaFile, []byte("this is a file, not a directory"), 0644); err != nil {
		t.Fatalf("Failed to create .kanuka file: %v", err)
	}

	// Capture output and expect failure
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, true, false)
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

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

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
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, true, false)
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

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Create symlink to non-existent directory
	kanukaSymlink := filepath.Join(tempDir, ".kanuka")
	nonExistentTarget := filepath.Join(tempDir, "non-existent-dir")
	if err := os.Symlink(nonExistentTarget, kanukaSymlink); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Capture output and expect failure
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, true, false)
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