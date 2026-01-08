package config

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestConfigListDevices contains tests for the `kanuka config list-devices` command.
func TestConfigListDevices(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("ListDevicesNoDevices", func(t *testing.T) {
		testListDevicesNoDevices(t, originalWd, originalUserSettings)
	})

	t.Run("ListDevicesSingleDevice", func(t *testing.T) {
		testListDevicesSingleDevice(t, originalWd, originalUserSettings)
	})

	t.Run("ListDevicesMultipleUsersDevices", func(t *testing.T) {
		testListDevicesMultipleUsersDevices(t, originalWd, originalUserSettings)
	})

	t.Run("ListDevicesWithUserFilter", func(t *testing.T) {
		testListDevicesWithUserFilter(t, originalWd, originalUserSettings)
	})

	t.Run("ListDevicesWithNonExistentUserFilter", func(t *testing.T) {
		testListDevicesWithNonExistentUserFilter(t, originalWd, originalUserSettings)
	})

	t.Run("ListDevicesOutsideProject", func(t *testing.T) {
		testListDevicesOutsideProject(t, originalWd, originalUserSettings)
	})
}

// Tests list-devices with no devices in project.
func testListDevicesNoDevices(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-list-devices-empty-*")
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

	// Initialize project structure without devices.
	shared.InitializeProjectStructureOnly(t, tempDir, tempUserDir)

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("list-devices", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "No devices found") {
		t.Errorf("Expected 'No devices found' message not found in output: %s", output)
	}
}

// Tests list-devices with a single device.
func testListDevicesSingleDevice(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-list-devices-single-*")
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

	// Add a device to the project config.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}

	projectConfig.Users = map[string]string{
		shared.TestUserUUID: shared.TestUserEmail,
	}
	projectConfig.Devices = map[string]configs.DeviceConfig{
		shared.TestUserUUID: {
			Email:     shared.TestUserEmail,
			Name:      "test-laptop",
			CreatedAt: time.Now(),
		},
	}

	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("list-devices", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, shared.TestUserEmail) {
		t.Errorf("Expected user email '%s' not found in output: %s", shared.TestUserEmail, output)
	}

	if !strings.Contains(output, "test-laptop") {
		t.Errorf("Expected device name 'test-laptop' not found in output: %s", output)
	}
}

// Tests list-devices with multiple users and devices.
func testListDevicesMultipleUsersDevices(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-list-devices-multi-*")
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

	// Add multiple devices for multiple users.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}

	user1Device2UUID := "user1-device2-uuid"
	projectConfig.Users = map[string]string{
		shared.TestUserUUID:  shared.TestUserEmail,
		user1Device2UUID:     shared.TestUserEmail,
		shared.TestUser2UUID: shared.TestUser2Email,
	}
	projectConfig.Devices = map[string]configs.DeviceConfig{
		shared.TestUserUUID: {
			Email:     shared.TestUserEmail,
			Name:      "laptop",
			CreatedAt: time.Now(),
		},
		user1Device2UUID: {
			Email:     shared.TestUserEmail,
			Name:      "desktop",
			CreatedAt: time.Now(),
		},
		shared.TestUser2UUID: {
			Email:     shared.TestUser2Email,
			Name:      "workstation",
			CreatedAt: time.Now(),
		},
	}

	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("list-devices", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	// Verify both users appear.
	if !strings.Contains(output, shared.TestUserEmail) {
		t.Errorf("Expected user email '%s' not found in output: %s", shared.TestUserEmail, output)
	}
	if !strings.Contains(output, shared.TestUser2Email) {
		t.Errorf("Expected user email '%s' not found in output: %s", shared.TestUser2Email, output)
	}

	// Verify all devices appear.
	if !strings.Contains(output, "laptop") {
		t.Errorf("Expected device name 'laptop' not found in output: %s", output)
	}
	if !strings.Contains(output, "desktop") {
		t.Errorf("Expected device name 'desktop' not found in output: %s", output)
	}
	if !strings.Contains(output, "workstation") {
		t.Errorf("Expected device name 'workstation' not found in output: %s", output)
	}
}

// Tests list-devices with --user filter.
func testListDevicesWithUserFilter(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-list-devices-filter-*")
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

	// Add devices for multiple users.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}

	projectConfig.Users = map[string]string{
		shared.TestUserUUID:  shared.TestUserEmail,
		shared.TestUser2UUID: shared.TestUser2Email,
	}
	projectConfig.Devices = map[string]configs.DeviceConfig{
		shared.TestUserUUID: {
			Email:     shared.TestUserEmail,
			Name:      "laptop",
			CreatedAt: time.Now(),
		},
		shared.TestUser2UUID: {
			Email:     shared.TestUser2Email,
			Name:      "workstation",
			CreatedAt: time.Now(),
		},
	}

	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("list-devices", nil, nil, false, false)
		cmd.SetArgs([]string{"config", "list-devices", "--user", shared.TestUserEmail})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	// Should show the filtered user.
	if !strings.Contains(output, shared.TestUserEmail) {
		t.Errorf("Expected user email '%s' not found in output: %s", shared.TestUserEmail, output)
	}
	if !strings.Contains(output, "laptop") {
		t.Errorf("Expected device name 'laptop' not found in output: %s", output)
	}

	// Should NOT show the other user.
	if strings.Contains(output, shared.TestUser2Email) {
		t.Errorf("Did not expect user email '%s' in filtered output: %s", shared.TestUser2Email, output)
	}
	if strings.Contains(output, "workstation") {
		t.Errorf("Did not expect device name 'workstation' in filtered output: %s", output)
	}
}

// Tests list-devices with --user filter for non-existent user.
func testListDevicesWithNonExistentUserFilter(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-list-devices-bad-filter-*")
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

	// Add a device.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}

	projectConfig.Users = map[string]string{
		shared.TestUserUUID: shared.TestUserEmail,
	}
	projectConfig.Devices = map[string]configs.DeviceConfig{
		shared.TestUserUUID: {
			Email:     shared.TestUserEmail,
			Name:      "laptop",
			CreatedAt: time.Now(),
		},
	}

	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("list-devices", nil, nil, false, false)
		cmd.SetArgs([]string{"config", "list-devices", "--user", "nonexistent@example.com"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "not found") {
		t.Errorf("Expected 'not found' message for non-existent user in output: %s", output)
	}
}

// Tests list-devices outside a project directory.
func testListDevicesOutsideProject(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-list-devices-outside-*")
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
		cmd := shared.CreateConfigTestCLI("list-devices", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	// Should indicate not in a project directory.
	if !strings.Contains(output, "Not in a Kānuka project") && !strings.Contains(output, "Failed to initialize") {
		t.Errorf("Expected error message about not being in a project directory, got: %s", output)
	}
}
