package register

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsRegisterUserEnvironment contains user environment tests for the `kanuka secrets register` command.
func TestSecretsRegisterUserEnvironment(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("RegisterWithDifferentUserDirectories", func(t *testing.T) {
		testRegisterWithDifferentUserDirectories(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithMissingUserDirectory", func(t *testing.T) {
		testRegisterWithMissingUserDirectory(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithCorruptedUserSettings", func(t *testing.T) {
		testRegisterWithCorruptedUserSettings(t, originalWd, originalUserSettings)
	})
}

// testRegisterWithDifferentUserDirectories tests with various user directory configurations.
func testRegisterWithDifferentUserDirectories(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-user-dirs-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test with custom user directory structure
	customUserDir, err := os.MkdirTemp("", "kanuka-custom-user-*")
	if err != nil {
		t.Fatalf("Failed to create custom user directory: %v", err)
	}
	defer os.RemoveAll(customUserDir)

	// Create nested directory structure
	nestedKeysDir := filepath.Join(customUserDir, "nested", "keys")
	nestedConfigDir := filepath.Join(customUserDir, "nested", "config")
	if err := os.MkdirAll(nestedKeysDir, 0755); err != nil {
		t.Fatalf("Failed to create nested keys directory: %v", err)
	}
	if err := os.MkdirAll(nestedConfigDir, 0755); err != nil {
		t.Fatalf("Failed to create nested config directory: %v", err)
	}

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Fatalf("Failed to change to original directory: %v", err)
		}
		configs.UserKanukaSettings = originalUserSettings
		configs.ProjectKanukaSettings = &configs.ProjectSettings{
			ProjectName:          "",
			ProjectPath:          "",
			ProjectPublicKeyPath: "",
			ProjectSecretsPath:   "",
		}
		configs.GlobalUserConfig = nil
	})

	// Override user settings to use custom nested directory
	configs.UserKanukaSettings = &configs.UserSettings{
		UserKeysPath:    nestedKeysDir,
		UserConfigsPath: nestedConfigDir,
		Username:        "testuser",
	}

	// Create user config with test UUID in the nested config directory
	userConfig := &configs.UserConfig{
		User: configs.User{
			UUID:  shared.TestUserUUID,
			Email: "testuser@example.com",
		},
		Projects: make(map[string]string),
	}
	if err := configs.SaveUserConfig(userConfig); err != nil {
		t.Fatalf("Failed to save user config: %v", err)
	}

	shared.InitializeProject(t, tempDir, customUserDir)

	// Create a target user's public key
	targetUser := "nesteduser"
	targetUserKeyPair := createTestUserKeyPair(t, tempDir, targetUser)

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--user", targetUser})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	// Verify the .kanuka file was created for the target user
	targetKanukaFile := filepath.Join(tempDir, ".kanuka", "secrets", targetUser+".kanuka")
	if _, err := os.Stat(targetKanukaFile); os.IsNotExist(err) {
		t.Errorf("Target user's .kanuka file was not created at %s", targetKanukaFile)
	}

	// Verify the target user can actually decrypt the symmetric key
	verifyUserCanDecrypt(t, targetUser, targetUserKeyPair.privateKey)
}

// testRegisterWithMissingUserDirectory tests handling missing user directory gracefully.
func testRegisterWithMissingUserDirectory(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-missing-user-*")
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
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Remove the user's keys directory after initialization
	userKeysDir := filepath.Join(tempUserDir, "keys")
	if err := os.RemoveAll(userKeysDir); err != nil {
		t.Fatalf("Failed to remove user keys directory: %v", err)
	}

	// Create a target user's public key
	targetUser := "missingdiruser"
	createTestUserKeyPair(t, tempDir, targetUser)

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--user", targetUser})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✗") {
		t.Errorf("Expected error symbol not found in output: %s", output)
	}

	if !strings.Contains(output, "private key") {
		t.Errorf("Expected private key error message not found in output: %s", output)
	}
}

// testRegisterWithCorruptedUserSettings tests handling corrupted user settings file.
func testRegisterWithCorruptedUserSettings(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-corrupted-settings-*")
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
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create a target user's public key
	targetUser := "corruptedsettingsuser"
	targetUserKeyPair := createTestUserKeyPair(t, tempDir, targetUser)

	// Corrupt user settings by setting invalid paths
	configs.UserKanukaSettings.UserKeysPath = "/invalid/nonexistent/path"
	configs.UserKanukaSettings.UserConfigsPath = "/invalid/nonexistent/path"
	// Clear cached user config so the command tries to create/save a new one
	configs.GlobalUserConfig = nil

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--user", targetUser})
		return cmd.Execute()
	})
	// We EXPECT the command to fail when given invalid paths
	// The error can be returned via err or shown in output
	hasError := err != nil || strings.Contains(output, "✗") || strings.Contains(output, "Error:")

	if !hasError {
		t.Errorf("Expected command to fail with invalid paths, but got success. Output: %s", output)
	}

	// Should contain some indication of path/permission/config issues
	if !strings.Contains(output, "private key") && !strings.Contains(output, "config") && !strings.Contains(output, "invalid") && !strings.Contains(output, "read-only") {
		t.Errorf("Expected error message about path/config issues not found in output: %s", output)
	}

	// Restore valid settings and verify registration works
	configs.UserKanukaSettings.UserKeysPath = filepath.Join(tempUserDir, "keys")
	configs.UserKanukaSettings.UserConfigsPath = filepath.Join(tempUserDir, "config")

	output, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--user", targetUser})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed after restoring settings: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success message not found after restoring settings: %s", output)
	}

	// Verify the target user can actually decrypt the symmetric key
	verifyUserCanDecrypt(t, targetUser, targetUserKeyPair.privateKey)
}
