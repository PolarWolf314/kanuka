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

	t.Run("InitUpdatesUserConfigWithProject", func(t *testing.T) {
		testInitUpdatesUserConfigWithProject(t, originalWd, originalUserSettings)
	})

	t.Run("InitWithValidConfigSkipsSetup", func(t *testing.T) {
		testInitWithValidConfigSkipsSetup(t, originalWd, originalUserSettings)
	})

	t.Run("InitWithYesFlagAndNoConfigFails", func(t *testing.T) {
		testInitWithYesFlagAndNoConfigFails(t, originalWd, originalUserSettings)
	})

	t.Run("InitWithNameFlag", func(t *testing.T) {
		testInitWithNameFlag(t, originalWd, originalUserSettings)
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

	if !strings.Contains(output, "Warning: Remember to never commit .env files") {
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

// testInitUpdatesUserConfigWithProject tests that init updates user config with project entry.
func testInitUpdatesUserConfigWithProject(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-user-config-*")
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

	// Verify user config has empty projects before init.
	userConfigBefore, err := configs.LoadUserConfig()
	if err != nil {
		t.Fatalf("Failed to load user config before init: %v", err)
	}
	if len(userConfigBefore.Projects) != 0 {
		t.Errorf("Expected empty projects before init, got: %v", userConfigBefore.Projects)
	}

	// Run init command.
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Verify project structure.
	shared.VerifyProjectStructure(t, tempDir)

	// Load project config to get the project UUID.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectUUID := projectConfig.Project.UUID
	if projectUUID == "" {
		t.Fatal("Project UUID should not be empty")
	}

	// Verify user config was updated with project entry.
	userConfigAfter, err := configs.LoadUserConfig()
	if err != nil {
		t.Fatalf("Failed to load user config after init: %v", err)
	}

	entry, exists := userConfigAfter.Projects[projectUUID]
	if !exists {
		t.Errorf("Expected project UUID %s in user config projects, got: %v", projectUUID, userConfigAfter.Projects)
	}
	if entry.DeviceName == "" {
		t.Error("Expected device name to be set in user config projects")
	}
}

// testInitWithValidConfigSkipsSetup tests that init skips config setup when user config is complete.
func testInitWithValidConfigSkipsSetup(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-valid-config-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// SetupTestEnvironment creates a complete user config with email and UUID.
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Run init command.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Should NOT see the "User configuration not found" message.
	if strings.Contains(output, "User configuration not found") {
		t.Errorf("Expected to skip config init setup but saw 'User configuration not found' message")
	}

	// Should NOT see the "Welcome to Kanuka" message (from config init).
	if strings.Contains(output, "Welcome to Kanuka") {
		t.Errorf("Expected to skip config init setup but saw 'Welcome to Kanuka' message")
	}

	// Verify project was created.
	shared.VerifyProjectStructure(t, tempDir)
}

// testInitWithYesFlagAndNoConfigFails tests that init with --yes flag fails when user config is incomplete.
func testInitWithYesFlagAndNoConfigFails(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-yes-no-config-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// SetupTestEnvironmentWithoutUserConfig does NOT create user config.
	shared.SetupTestEnvironmentWithoutUserConfig(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Run init command with --yes flag.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("init", []string{"--yes"}, nil, nil, false, false)
		return cmd.Execute()
	})

	// Should fail with an error.
	if err == nil {
		t.Errorf("Expected command to fail with --yes flag when user config is incomplete, but it succeeded")
		t.Errorf("Output: %s", output)
	}

	// Error message should mention running config init.
	if !strings.Contains(output, "User configuration is incomplete") && !strings.Contains(err.Error(), "user configuration required") {
		t.Errorf("Expected error about incomplete user config, got output: %s, error: %v", output, err)
	}

	// Project should NOT be created.
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	if _, statErr := os.Stat(filepath.Join(kanukaDir, "config.toml")); statErr == nil {
		t.Errorf("Project config should not exist when init fails due to missing user config")
	}
}

// testInitWithNameFlag tests that init with --name flag uses the specified project name.
func testInitWithNameFlag(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-name-flag-*")
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

	customProjectName := "My Custom Project Name"

	// Run init command with --name flag.
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("init", []string{"--name", customProjectName}, nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Verify project structure.
	shared.VerifyProjectStructure(t, tempDir)

	// Load project config and verify name.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}

	if projectConfig.Project.Name != customProjectName {
		t.Errorf("Expected project name to be %q, got %q", customProjectName, projectConfig.Project.Name)
	}
}
