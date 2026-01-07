package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestConfigSetDeviceName contains tests for the `kanuka config set-device-name` command.
func TestConfigSetDeviceName(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("SetDeviceNameInProject", func(t *testing.T) {
		testSetDeviceNameInProject(t, originalWd, originalUserSettings)
	})

	t.Run("SetDeviceNameWithProjectUUID", func(t *testing.T) {
		testSetDeviceNameWithProjectUUID(t, originalWd, originalUserSettings)
	})

	t.Run("SetDeviceNameInvalidName", func(t *testing.T) {
		testSetDeviceNameInvalidName(t, originalWd, originalUserSettings)
	})

	t.Run("SetDeviceNameUpdate", func(t *testing.T) {
		testSetDeviceNameUpdate(t, originalWd, originalUserSettings)
	})

	t.Run("SetDeviceNameSameValue", func(t *testing.T) {
		testSetDeviceNameSameValue(t, originalWd, originalUserSettings)
	})

	t.Run("SetDeviceNameOutsideProject", func(t *testing.T) {
		testSetDeviceNameOutsideProject(t, originalWd, originalUserSettings)
	})
}

// Tests set-device-name in a project directory.
func testSetDeviceNameInProject(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-set-device-name-*")
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
		cmd := shared.CreateConfigTestCLI("set-device-name", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "set-device-name", "my-laptop"})
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

// Tests set-device-name with --project-uuid flag.
func testSetDeviceNameWithProjectUUID(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-set-device-uuid-*")
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
		cmd := shared.CreateConfigTestCLI("set-device-name", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "set-device-name", "--project-uuid", specificUUID, "workstation"})
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

	deviceName, exists := userConfig.Projects[specificUUID]
	if !exists {
		t.Errorf("Expected project UUID '%s' not found in user config projects", specificUUID)
	}
	if deviceName != "workstation" {
		t.Errorf("Expected device name 'workstation', got '%s'", deviceName)
	}
}

// Tests set-device-name with invalid device name.
func testSetDeviceNameInvalidName(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-set-device-invalid-*")
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
			cmd := shared.CreateConfigTestCLI("set-device-name", nil, nil, true, false)
			cmd.SetArgs([]string{"config", "set-device-name", invalidName})
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

// Tests set-device-name updating an existing name.
func testSetDeviceNameUpdate(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-set-device-update-*")
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
		cmd := shared.CreateConfigTestCLI("set-device-name", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "set-device-name", "old-name"})
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to set initial device name: %v", err)
	}

	// Update to new name.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("set-device-name", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "set-device-name", "new-name"})
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

// Tests set-device-name when name is already set to same value.
func testSetDeviceNameSameValue(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-set-device-same-*")
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
		cmd := shared.CreateConfigTestCLI("set-device-name", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "set-device-name", "my-laptop"})
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to set initial device name: %v", err)
	}

	// Set same name again.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("set-device-name", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "set-device-name", "my-laptop"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "already set") {
		t.Errorf("Expected 'already set' message not found in output: %s", output)
	}
}

// Tests set-device-name outside a project directory without --project-uuid.
func testSetDeviceNameOutsideProject(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-set-device-outside-*")
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
		cmd := shared.CreateConfigTestCLI("set-device-name", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "set-device-name", "my-laptop"})
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
