package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
)

// TestSecretsInitCrossPlatform contains cross-platform edge case tests for the `kanuka secrets init` command.
func TestSecretsInitCrossPlatform(t *testing.T) {
	// Save original working directory and settings
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings

	t.Run("InitWithSpecialCharactersInPath", func(t *testing.T) {
		testInitWithSpecialCharactersInPath(t, originalWd, originalUserSettings)
	})

	t.Run("InitWithUnicodeInPath", func(t *testing.T) {
		testInitWithUnicodeInPath(t, originalWd, originalUserSettings)
	})
}

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
