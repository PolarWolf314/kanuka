package init_test

import (
	"os"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsInitInputValidation contains input validation edge case tests for the `kanuka secrets init` command.
func TestSecretsInitInputValidation(t *testing.T) {
	// Save original working directory and settings
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings

	t.Run("InitWithVeryLongProjectName", func(t *testing.T) {
		testInitWithVeryLongProjectName(t, originalWd, originalUserSettings)
	})

	t.Run("InitWithSpecialCharactersInProjectName", func(t *testing.T) {
		testInitWithSpecialCharactersInProjectName(t, originalWd, originalUserSettings)
	})
}

// Tests init with invalid username input.
func testInitWithVeryLongProjectName(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory with very long name (but within filesystem limits)
	longName := strings.Repeat("a", 100) // 100 characters should be safe on most filesystems
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-long-"+longName)
	if err != nil {
		t.Fatalf("Failed to create temp directory with long name: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed with long project name: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "initialized successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}

	shared.VerifyProjectStructure(t, tempDir)
}

func testInitWithSpecialCharactersInProjectName(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory with special characters that are valid in filenames
	specialName := "kanuka-test-init-special-project-name_with-dashes.and.dots"
	tempDir, err := os.MkdirTemp("", specialName)
	if err != nil {
		t.Fatalf("Failed to create temp directory with special project name: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed with special characters in project name: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "initialized successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}

	shared.VerifyProjectStructure(t, tempDir)
}
