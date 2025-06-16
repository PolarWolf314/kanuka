package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
)

// TestSecretsInitBasic contains basic integration tests for the `kanuka secrets init` command.
func TestSecretsInitBasic(t *testing.T) {
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
