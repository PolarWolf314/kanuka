package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestConfigSetProjectDevice contains tests for the `kanuka config set-project-device` command.
func TestConfigSetProjectDevice(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("SetProjectDeviceInProject", func(t *testing.T) {
		testSetProjectDeviceInProject(t, originalWd, originalUserSettings)
	})

	t.Run("SetProjectDeviceWithProjectUUID", func(t *testing.T) {
		testSetProjectDeviceWithProjectUUID(t, originalWd, originalUserSettings)
	})

	t.Run("SetProjectDeviceInvalidName", func(t *testing.T) {
		testSetProjectDeviceInvalidName(t, originalWd, originalUserSettings)
	})

	t.Run("SetProjectDeviceUpdate", func(t *testing.T) {
		testSetProjectDeviceUpdate(t, originalWd, originalUserSettings)
	})

	t.Run("SetProjectDeviceSameValue", func(t *testing.T) {
		testSetProjectDeviceSameValue(t, originalWd, originalUserSettings)
	})

	t.Run("SetProjectDeviceOutsideProject", func(t *testing.T) {
		testSetProjectDeviceOutsideProject(t, originalWd, originalUserSettings)
	})

	t.Run("SetProjectDeviceUpdatesProjectConfig", func(t *testing.T) {
		testSetProjectDeviceUpdatesProjectConfig(t, originalWd, originalUserSettings)
	})
}

// Tests set-project-device in a project directory.
func testSetProjectDeviceInProject(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-set-project-device-*")
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
	shared.InitializeProjectStructureOnly(t, tempDir, tempUserDir)

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("set-project-device", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "set-project-device", "my-laptop"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "Device name set to") {
		t.Errorf("Expected success message not found in output: %s", output)
	}
	if !strings.Contains(output, "my-laptop") {
		t.Errorf("Expected device name 'my-laptop' not found in output: %s", output)
	}

	// Verify the user config was updated.
	userConfigPath := filepath.Join(tempUserDir, "config", "config.toml")
	content, err := os.ReadFile(userConfigPath)
	if err != nil {
		t.Fatalf("Failed to read user config: %v", err)
	}

	if !strings.Contains(string(content), "my-laptop") {
		t.Errorf("Expected device name 'my-laptop' not found in user config: %s", string(content))
	}
}

// Tests set-project-device with --project-uuid flag.
func testSetProjectDeviceWithProjectUUID(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-set-project-device-uuid-*")
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

	// Use a specific project UUID without being in a project.
	specificUUID := "specific-project-uuid-1234"

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("set-project-device", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "set-project-device", "--project-uuid", specificUUID, "workstation"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "Device name set to") {
		t.Errorf("Expected success message not found in output: %s", output)
	}
	if !strings.Contains(output, "workstation") {
		t.Errorf("Expected device name 'workstation' not found in output: %s", output)
	}

	// Verify the user config was updated with the specific UUID.
	userConfig, err := configs.LoadUserConfig()
	if err != nil {
		t.Fatalf("Failed to load user config: %v", err)
	}

	entry, exists := userConfig.Projects[specificUUID]
	if !exists {
		t.Errorf("Expected project UUID '%s' not found in user config projects", specificUUID)
	}
	if entry.DeviceName != "workstation" {
		t.Errorf("Expected device name 'workstation', got '%s'", entry.DeviceName)
	}
}

// Tests set-project-device with invalid device name.
func testSetProjectDeviceInvalidName(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-set-project-device-invalid-*")
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
	shared.InitializeProjectStructureOnly(t, tempDir, tempUserDir)

	// Test invalid names that don't start with a dash (to avoid flag parsing issues).
	invalidNames := []string{
		"my laptop", // Space.
		"my@laptop", // Special character.
		"laptop!",   // Exclamation mark.
	}

	for _, invalidName := range invalidNames {
		output, err := shared.CaptureOutput(func() error {
			cmd := shared.CreateConfigTestCLI("set-project-device", nil, nil, true, false)
			cmd.SetArgs([]string{"config", "set-project-device", invalidName})
			return cmd.Execute()
		})
		if err != nil {
			t.Errorf("Command failed unexpectedly for '%s': %v", invalidName, err)
		}

		if !strings.Contains(output, "Invalid device name") {
			t.Errorf("Expected 'Invalid device name' message for '%s', got: %s", invalidName, output)
		}
	}
}

// Tests set-project-device updating an existing name.
func testSetProjectDeviceUpdate(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-set-project-device-update-*")
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
	shared.InitializeProjectStructureOnly(t, tempDir, tempUserDir)

	// Set initial device name.
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("set-project-device", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "set-project-device", "old-name"})
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to set initial device name: %v", err)
	}

	// Update to new name.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("set-project-device", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "set-project-device", "new-name"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "updated from") {
		t.Errorf("Expected 'updated from' message not found in output: %s", output)
	}
	if !strings.Contains(output, "old-name") {
		t.Errorf("Expected old name 'old-name' not found in output: %s", output)
	}
	if !strings.Contains(output, "new-name") {
		t.Errorf("Expected new name 'new-name' not found in output: %s", output)
	}
}

// Tests set-project-device when name is already set to same value.
func testSetProjectDeviceSameValue(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-set-project-device-same-*")
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
	shared.InitializeProjectStructureOnly(t, tempDir, tempUserDir)

	// Set initial device name.
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("set-project-device", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "set-project-device", "my-laptop"})
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to set initial device name: %v", err)
	}

	// Set same name again.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("set-project-device", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "set-project-device", "my-laptop"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "already set") {
		t.Errorf("Expected 'already set' message not found in output: %s", output)
	}
}

// Tests set-project-device outside a project directory without --project-uuid.
func testSetProjectDeviceOutsideProject(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-set-project-device-outside-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Setup environment but don't create project structure.
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("set-project-device", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "set-project-device", "my-laptop"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	// Should suggest using --project-uuid.
	if !strings.Contains(output, "--project-uuid") {
		t.Errorf("Expected suggestion to use '--project-uuid' not found in output: %s", output)
	}
}

func testSetProjectDeviceUpdatesProjectConfig(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-set-project-device-project-config-*")
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

	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	if err := os.MkdirAll(publicKeysDir, 0755); err != nil {
		t.Fatalf("Failed to create public keys directory: %v", err)
	}
	if err := os.MkdirAll(secretsDir, 0755); err != nil {
		t.Fatalf("Failed to create secrets directory: %v", err)
	}

	projectConfig := &configs.ProjectConfig{
		Project: configs.Project{
			UUID: shared.TestProjectUUID,
			Name: "test-project",
		},
		Users: map[string]string{
			shared.TestUserUUID: shared.TestUserEmail,
		},
		Devices: map[string]configs.DeviceConfig{
			shared.TestUserUUID: {
				Email:     shared.TestUserEmail,
				Name:      "old-device-name",
				CreatedAt: time.Now().UTC(),
			},
		},
	}

	configs.ProjectKanukaSettings = &configs.ProjectSettings{
		ProjectName:          "test-project",
		ProjectPath:          tempDir,
		ProjectPublicKeyPath: publicKeysDir,
		ProjectSecretsPath:   secretsDir,
	}

	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("set-project-device", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "set-project-device", "new-device-name"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "Device name set to") {
		t.Errorf("Expected success message not found in output: %s", output)
	}
	if !strings.Contains(output, "new-device-name") {
		t.Errorf("Expected device name 'new-device-name' not found in output: %s", output)
	}

	userConfig, err := configs.LoadUserConfig()
	if err != nil {
		t.Fatalf("Failed to load user config: %v", err)
	}

	entry, exists := userConfig.Projects[shared.TestProjectUUID]
	if !exists {
		t.Errorf("Expected project UUID '%s' not found in user config projects", shared.TestProjectUUID)
	}
	if entry.DeviceName != "new-device-name" {
		t.Errorf("Expected user config device name 'new-device-name', got '%s'", entry.DeviceName)
	}

	updatedProjectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}

	deviceConfig, exists := updatedProjectConfig.Devices[shared.TestUserUUID]
	if !exists {
		t.Errorf("Expected user UUID '%s' not found in project config devices", shared.TestUserUUID)
	}
	if deviceConfig.Name != "new-device-name" {
		t.Errorf("Expected project config device name 'new-device-name', got '%s'", deviceConfig.Name)
	}

	if deviceConfig.Email != shared.TestUserEmail {
		t.Errorf("Expected email to be preserved, got '%s'", deviceConfig.Email)
	}
}
