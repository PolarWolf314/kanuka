package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
)

// TestSecretsInitInputValidation contains input validation edge case tests for the `kanuka secrets init` command.
func TestSecretsInitInputValidation(t *testing.T) {
	// Save original working directory and settings
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings

	// Category 13: Input Validation Edge Cases
	t.Run("InitWithVeryLongProjectName", func(t *testing.T) {
		testInitWithVeryLongProjectName(t, originalWd, originalUserSettings)
	})

	t.Run("InitWithSpecialCharactersInProjectName", func(t *testing.T) {
		testInitWithSpecialCharactersInProjectName(t, originalWd, originalUserSettings)
	})
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