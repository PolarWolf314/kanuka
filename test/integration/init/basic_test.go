package init_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
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
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-empty-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Capture real stdout/stderr by redirecting them
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	shared.VerifyProjectStructure(t, tempDir)

	shared.VerifyUserKeys(t, tempUserDir)

	if !strings.Contains(output, "Warning: Remember: Never commit .env files") {
		t.Errorf("Expected warning message not found in output: %s", output)
	}
}

// testInitInAlreadyInitializedFolder tests behavior when running init in an already initialized folder.
func testInitInAlreadyInitializedFolder(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-existing-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Pre-create .kanuka directory to simulate already initialized project
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	if err := os.MkdirAll(kanukaDir, 0755); err != nil {
		t.Fatalf("Failed to create .kanuka directory: %v", err)
	}

	// Capture real stdout/stderr by redirecting them
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	if _, statErr := os.Stat(kanukaDir); os.IsNotExist(statErr) {
		t.Errorf(".kanuka directory should still exist after running init on already initialized project")
	}

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
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-verbose-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Capture real stdout/stderr by redirecting them
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "[info]") {
		t.Errorf("Expected verbose [info] messages not found in output: %s", output)
	}

	shared.VerifyProjectStructure(t, tempDir)
}

// testInitWithDebugFlag tests initialization with debug flag.
func testInitWithDebugFlag(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-debug-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Capture real stdout/stderr by redirecting them
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, false, true)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "[debug]") {
		t.Errorf("Expected debug [debug] messages not found in output: %s", output)
	}

	// Debug should also include info messages
	if !strings.Contains(output, "[info]") {
		t.Errorf("Expected [info] messages not found in debug output: %s", output)
	}

	shared.VerifyProjectStructure(t, tempDir)
}
