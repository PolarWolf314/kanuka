package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
)

// TestSecretsInitEnvironment contains environment variable edge case tests for the `kanuka secrets init` command.
func TestSecretsInitEnvironment(t *testing.T) {
	// Save original working directory and settings
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings

	// Category 6: Environment Variable Edge Cases
	t.Run("InitWithInvalidXDGDataHome", func(t *testing.T) {
		testInitWithInvalidXDGDataHome(t, originalWd, originalUserSettings)
	})

	t.Run("InitWithXDGDataHomeAsFile", func(t *testing.T) {
		testInitWithXDGDataHomeAsFile(t, originalWd, originalUserSettings)
	})
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
}