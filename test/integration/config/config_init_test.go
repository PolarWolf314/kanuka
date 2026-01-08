package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestConfigInit contains tests for the `kanuka config init` command.
func TestConfigInit(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("InitWithFlags", func(t *testing.T) {
		testConfigInitWithFlags(t, originalWd, originalUserSettings)
	})

	t.Run("InitShowsExistingConfig", func(t *testing.T) {
		testConfigInitShowsExistingConfig(t, originalWd, originalUserSettings)
	})

	t.Run("InitUpdateWithFlags", func(t *testing.T) {
		testConfigInitUpdateWithFlags(t, originalWd, originalUserSettings)
	})

	t.Run("InitInvalidEmail", func(t *testing.T) {
		testConfigInitInvalidEmail(t, originalWd, originalUserSettings)
	})

	t.Run("InitCreatesConfigFile", func(t *testing.T) {
		testConfigInitCreatesConfigFile(t, originalWd, originalUserSettings)
	})
}

// Tests config init with flags (non-interactive).
func testConfigInitWithFlags(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-config-init-flags-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Setup without user config (so we can test creating it).
	shared.SetupTestEnvironmentWithoutUserConfig(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Ensure the kanuka config directory exists.
	// UserConfigsPath is set to tempUserDir/config by SetupTestEnvironmentWithoutUserConfig.
	kanukaConfigDir := filepath.Join(tempUserDir, "config")
	if err := os.MkdirAll(kanukaConfigDir, 0755); err != nil {
		t.Fatalf("Failed to create kanuka config directory: %v", err)
	}

	// Run config init with flags (non-interactive).
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("init", nil, nil, false, false)
		cmd.SetArgs([]string{"config", "init", "--email", "test@example.com", "--name", "Test User", "--device", "test-device"})
		return cmd.Execute()
	})

	if err != nil {
		t.Fatalf("config init failed: %v\nOutput: %s", err, output)
	}

	// Verify output.
	if !strings.Contains(output, "User configuration updated") && !strings.Contains(output, "User configuration saved") {
		t.Errorf("Expected success message, got: %s", output)
	}
	if !strings.Contains(output, "test@example.com") {
		t.Errorf("Expected email in output, got: %s", output)
	}
	if !strings.Contains(output, "Test User") {
		t.Errorf("Expected name in output, got: %s", output)
	}
	if !strings.Contains(output, "test-device") {
		t.Errorf("Expected device name in output, got: %s", output)
	}

	// Verify config file was created.
	userConfig, err := configs.LoadUserConfig()
	if err != nil {
		t.Fatalf("Failed to load user config: %v", err)
	}

	if userConfig.User.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got: %s", userConfig.User.Email)
	}
	if userConfig.User.Name != "Test User" {
		t.Errorf("Expected name 'Test User', got: %s", userConfig.User.Name)
	}
	if userConfig.User.DefaultDeviceName != "test-device" {
		t.Errorf("Expected device 'test-device', got: %s", userConfig.User.DefaultDeviceName)
	}
	if userConfig.User.UUID == "" {
		t.Error("Expected user UUID to be generated")
	}
}

// Tests config init shows existing config when already configured.
func testConfigInitShowsExistingConfig(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-config-init-existing-*")
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

	// Update user config with additional fields.
	userConfig, err := configs.LoadUserConfig()
	if err != nil {
		t.Fatalf("Failed to load user config: %v", err)
	}
	userConfig.User.Name = "Existing User"
	userConfig.User.DefaultDeviceName = "existing-device"
	if err := configs.SaveUserConfig(userConfig); err != nil {
		t.Fatalf("Failed to save user config: %v", err)
	}

	// Run config init without flags - should show existing config.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("init", nil, nil, false, false)
		cmd.SetArgs([]string{"config", "init"})
		return cmd.Execute()
	})

	if err != nil {
		t.Fatalf("config init failed: %v\nOutput: %s", err, output)
	}

	// Verify output shows existing config.
	if !strings.Contains(output, "already exists") {
		t.Errorf("Expected 'already exists' message, got: %s", output)
	}
	if !strings.Contains(output, shared.TestUserEmail) {
		t.Errorf("Expected existing email in output, got: %s", output)
	}
	if !strings.Contains(output, "Existing User") {
		t.Errorf("Expected existing name in output, got: %s", output)
	}
}

// Tests config init updates config when flags are provided.
func testConfigInitUpdateWithFlags(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-config-init-update-*")
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

	// Update user config with additional fields.
	userConfig, err := configs.LoadUserConfig()
	if err != nil {
		t.Fatalf("Failed to load user config: %v", err)
	}
	oldUUID := userConfig.User.UUID
	userConfig.User.Name = "Old Name"
	if err := configs.SaveUserConfig(userConfig); err != nil {
		t.Fatalf("Failed to save user config: %v", err)
	}

	// Run config init with flags to update.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("init", nil, nil, false, false)
		cmd.SetArgs([]string{"config", "init", "--email", "new@example.com"})
		return cmd.Execute()
	})

	if err != nil {
		t.Fatalf("config init failed: %v\nOutput: %s", err, output)
	}

	// Verify output.
	if !strings.Contains(output, "updated") {
		t.Errorf("Expected 'updated' message, got: %s", output)
	}
	if !strings.Contains(output, "new@example.com") {
		t.Errorf("Expected new email in output, got: %s", output)
	}

	// Verify config was updated.
	updatedConfig, err := configs.LoadUserConfig()
	if err != nil {
		t.Fatalf("Failed to load user config: %v", err)
	}

	if updatedConfig.User.Email != "new@example.com" {
		t.Errorf("Expected email 'new@example.com', got: %s", updatedConfig.User.Email)
	}
	// Other fields should be preserved.
	if updatedConfig.User.Name != "Old Name" {
		t.Errorf("Expected name 'Old Name' to be preserved, got: %s", updatedConfig.User.Name)
	}
	if updatedConfig.User.UUID != oldUUID {
		t.Errorf("Expected UUID to be preserved, got: %s", updatedConfig.User.UUID)
	}
}

// Tests config init with invalid email.
func testConfigInitInvalidEmail(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-config-init-invalid-email-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironmentWithoutUserConfig(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Ensure the kanuka config directory exists.
	// UserConfigsPath is set to tempUserDir/config by SetupTestEnvironmentWithoutUserConfig.
	kanukaConfigDir := filepath.Join(tempUserDir, "config")
	if err := os.MkdirAll(kanukaConfigDir, 0755); err != nil {
		t.Fatalf("Failed to create kanuka config directory: %v", err)
	}

	// Run config init with invalid email.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("init", nil, nil, false, false)
		cmd.SetArgs([]string{"config", "init", "--email", "invalid-email"})
		return cmd.Execute()
	})

	if err != nil {
		t.Fatalf("config init returned error: %v\nOutput: %s", err, output)
	}

	// Verify output shows error.
	if !strings.Contains(output, "Invalid email format") {
		t.Errorf("Expected invalid email error, got: %s", output)
	}
}

// Tests config init creates config file.
func testConfigInitCreatesConfigFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-config-init-creates-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironmentWithoutUserConfig(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Ensure the kanuka config directory exists.
	// UserConfigsPath is set to tempUserDir/config by SetupTestEnvironmentWithoutUserConfig.
	kanukaConfigDir := filepath.Join(tempUserDir, "config")
	if err := os.MkdirAll(kanukaConfigDir, 0755); err != nil {
		t.Fatalf("Failed to create kanuka config directory: %v", err)
	}

	// Verify config file doesn't exist yet.
	// Config is saved to UserConfigsPath/config.toml.
	configPath := filepath.Join(tempUserDir, "config", "config.toml")
	if _, err := os.Stat(configPath); err == nil {
		t.Fatal("Config file should not exist before init")
	}

	// Run config init.
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("init", nil, nil, false, false)
		cmd.SetArgs([]string{"config", "init", "--email", "test@example.com"})
		return cmd.Execute()
	})

	if err != nil {
		t.Fatalf("config init failed: %v", err)
	}

	// Verify config file was created.
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file should exist after init")
	}
}
