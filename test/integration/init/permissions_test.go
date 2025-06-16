package init_test

import (
	"os"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsInitPermissions contains permission-related edge case tests for the `kanuka secrets init` command.
func TestSecretsInitPermissions(t *testing.T) {
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
}

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

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Capture output and expect failure
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, true, false)
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

