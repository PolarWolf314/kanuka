package config

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestConfigRenameDevice contains tests for the `kanuka config rename-device` command.
func TestConfigRenameDevice(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("RenameDeviceSingleDevice", func(t *testing.T) {
		testRenameDeviceSingleDevice(t, originalWd, originalUserSettings)
	})

	t.Run("RenameDeviceMultipleDevicesWithOldName", func(t *testing.T) {
		testRenameDeviceMultipleDevicesWithOldName(t, originalWd, originalUserSettings)
	})

	t.Run("RenameDeviceMultipleDevicesNoOldName", func(t *testing.T) {
		testRenameDeviceMultipleDevicesNoOldName(t, originalWd, originalUserSettings)
	})

	t.Run("RenameDeviceNonExistentUser", func(t *testing.T) {
		testRenameDeviceNonExistentUser(t, originalWd, originalUserSettings)
	})

	t.Run("RenameDeviceNonExistentDevice", func(t *testing.T) {
		testRenameDeviceNonExistentDevice(t, originalWd, originalUserSettings)
	})

	t.Run("RenameDeviceDuplicateName", func(t *testing.T) {
		testRenameDeviceDuplicateName(t, originalWd, originalUserSettings)
	})

	t.Run("RenameDeviceSameName", func(t *testing.T) {
		testRenameDeviceSameName(t, originalWd, originalUserSettings)
	})

	t.Run("RenameDeviceInvalidNewName", func(t *testing.T) {
		testRenameDeviceInvalidNewName(t, originalWd, originalUserSettings)
	})

	t.Run("RenameDeviceOutsideProject", func(t *testing.T) {
		testRenameDeviceOutsideProject(t, originalWd, originalUserSettings)
	})

	t.Run("RenameOwnDeviceUpdatesUserConfig", func(t *testing.T) {
		testRenameOwnDeviceUpdatesUserConfig(t, originalWd, originalUserSettings)
	})

	t.Run("RenameOtherUserDeviceDoesNotUpdateUserConfig", func(t *testing.T) {
		testRenameOtherUserDeviceDoesNotUpdateUserConfig(t, originalWd, originalUserSettings)
	})
}

// Tests rename-device with single device (auto-infer old name).
func testRenameDeviceSingleDevice(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-rename-single-*")
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

	// Add a single device for the user.
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
			Name:      "old-laptop",
			CreatedAt: time.Now(),
		},
	}

	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("rename-device", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "rename-device", "--user", shared.TestUserEmail, "new-laptop"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "renamed to") {
		t.Errorf("Expected 'renamed to' message not found in output: %s", output)
	}
	if !strings.Contains(output, "old-laptop") {
		t.Errorf("Expected old name 'old-laptop' not found in output: %s", output)
	}
	if !strings.Contains(output, "new-laptop") {
		t.Errorf("Expected new name 'new-laptop' not found in output: %s", output)
	}

	// Verify the project config was updated.
	projectConfig, err = configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to reload project config: %v", err)
	}

	device, exists := projectConfig.Devices[shared.TestUserUUID]
	if !exists {
		t.Errorf("Device not found in project config after rename")
	}
	if device.Name != "new-laptop" {
		t.Errorf("Expected device name 'new-laptop', got '%s'", device.Name)
	}
}

// Tests rename-device with multiple devices using --old-name.
func testRenameDeviceMultipleDevicesWithOldName(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-rename-multi-*")
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

	// Add multiple devices for the user.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}

	device2UUID := "device-2-uuid-1234"
	projectConfig.Users = map[string]string{
		shared.TestUserUUID: shared.TestUserEmail,
		device2UUID:         shared.TestUserEmail,
	}
	projectConfig.Devices = map[string]configs.DeviceConfig{
		shared.TestUserUUID: {
			Email:     shared.TestUserEmail,
			Name:      "laptop",
			CreatedAt: time.Now(),
		},
		device2UUID: {
			Email:     shared.TestUserEmail,
			Name:      "desktop",
			CreatedAt: time.Now(),
		},
	}

	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("rename-device", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "rename-device", "--user", shared.TestUserEmail, "--old-name", "laptop", "macbook"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "renamed to") {
		t.Errorf("Expected 'renamed to' message not found in output: %s", output)
	}
	if !strings.Contains(output, "macbook") {
		t.Errorf("Expected new name 'macbook' not found in output: %s", output)
	}

	// Verify the correct device was renamed.
	projectConfig, err = configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to reload project config: %v", err)
	}

	device1 := projectConfig.Devices[shared.TestUserUUID]
	device2 := projectConfig.Devices[device2UUID]

	if device1.Name != "macbook" {
		t.Errorf("Expected device 1 name 'macbook', got '%s'", device1.Name)
	}
	if device2.Name != "desktop" {
		t.Errorf("Expected device 2 name 'desktop' (unchanged), got '%s'", device2.Name)
	}
}

// Tests rename-device with multiple devices but no --old-name (should fail).
func testRenameDeviceMultipleDevicesNoOldName(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-rename-multi-no-old-*")
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

	// Add multiple devices for the user.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}

	device2UUID := "device-2-uuid-5678"
	projectConfig.Users = map[string]string{
		shared.TestUserUUID: shared.TestUserEmail,
		device2UUID:         shared.TestUserEmail,
	}
	projectConfig.Devices = map[string]configs.DeviceConfig{
		shared.TestUserUUID: {
			Email:     shared.TestUserEmail,
			Name:      "laptop",
			CreatedAt: time.Now(),
		},
		device2UUID: {
			Email:     shared.TestUserEmail,
			Name:      "desktop",
			CreatedAt: time.Now(),
		},
	}

	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("rename-device", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "rename-device", "--user", shared.TestUserEmail, "new-name"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "multiple devices") {
		t.Errorf("Expected 'multiple devices' message not found in output: %s", output)
	}
	if !strings.Contains(output, "--old-name") {
		t.Errorf("Expected '--old-name' suggestion not found in output: %s", output)
	}
}

// Tests rename-device with non-existent user.
func testRenameDeviceNonExistentUser(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-rename-no-user-*")
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

	// Add a device for a different user.
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
		cmd := shared.CreateConfigTestCLI("rename-device", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "rename-device", "--user", "nonexistent@example.com", "new-name"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "not found") {
		t.Errorf("Expected 'not found' message for non-existent user in output: %s", output)
	}
}

// Tests rename-device with non-existent device name.
func testRenameDeviceNonExistentDevice(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-rename-no-device-*")
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
		cmd := shared.CreateConfigTestCLI("rename-device", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "rename-device", "--user", shared.TestUserEmail, "--old-name", "nonexistent-device", "new-name"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "not found") {
		t.Errorf("Expected 'not found' message for non-existent device in output: %s", output)
	}
}

// Tests rename-device with duplicate name (already taken by same user).
func testRenameDeviceDuplicateName(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-rename-dup-*")
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

	// Add two devices for the same user.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}

	device2UUID := "device-2-uuid-dup"
	projectConfig.Users = map[string]string{
		shared.TestUserUUID: shared.TestUserEmail,
		device2UUID:         shared.TestUserEmail,
	}
	projectConfig.Devices = map[string]configs.DeviceConfig{
		shared.TestUserUUID: {
			Email:     shared.TestUserEmail,
			Name:      "laptop",
			CreatedAt: time.Now(),
		},
		device2UUID: {
			Email:     shared.TestUserEmail,
			Name:      "desktop",
			CreatedAt: time.Now(),
		},
	}

	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	// Try to rename "laptop" to "desktop" (which already exists).
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("rename-device", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "rename-device", "--user", shared.TestUserEmail, "--old-name", "laptop", "desktop"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "already in use") {
		t.Errorf("Expected 'already in use' message not found in output: %s", output)
	}
}

// Tests rename-device to the same name.
func testRenameDeviceSameName(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-rename-same-*")
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

	// Rename to the same name.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("rename-device", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "rename-device", "--user", shared.TestUserEmail, "laptop"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "already named") {
		t.Errorf("Expected 'already named' message not found in output: %s", output)
	}
}

// Tests rename-device with invalid new name.
func testRenameDeviceInvalidNewName(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-rename-invalid-*")
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
		cmd := shared.CreateConfigTestCLI("rename-device", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "rename-device", "--user", shared.TestUserEmail, "invalid name!"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "Invalid device name") {
		t.Errorf("Expected 'Invalid device name' message not found in output: %s", output)
	}
}

// Tests rename-device outside a project directory.
func testRenameDeviceOutsideProject(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-rename-outside-*")
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
		cmd := shared.CreateConfigTestCLI("rename-device", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "rename-device", "--user", "test@example.com", "new-name"})
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

// Tests that renaming your own device also updates the user config's [projects] section.
func testRenameOwnDeviceUpdatesUserConfig(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-rename-own-*")
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

	// Get project UUID for verification.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectUUID := projectConfig.Project.UUID

	// Add the current user's device (matching TestUserUUID from user config).
	projectConfig.Users = map[string]string{
		shared.TestUserUUID: shared.TestUserEmail,
	}
	projectConfig.Devices = map[string]configs.DeviceConfig{
		shared.TestUserUUID: {
			Email:     shared.TestUserEmail,
			Name:      "old-device-name",
			CreatedAt: time.Now(),
		},
	}

	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	// Set up user config with the project entry.
	userConfig, err := configs.LoadUserConfig()
	if err != nil {
		t.Fatalf("Failed to load user config: %v", err)
	}
	userConfig.Projects[projectUUID] = "old-device-name"
	if err := configs.SaveUserConfig(userConfig); err != nil {
		t.Fatalf("Failed to save user config: %v", err)
	}

	// Rename the device.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("rename-device", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "rename-device", "--user", shared.TestUserEmail, "new-device-name"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "renamed to") {
		t.Errorf("Expected 'renamed to' message not found in output: %s", output)
	}

	// Verify project config was updated.
	projectConfig, err = configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to reload project config: %v", err)
	}
	device := projectConfig.Devices[shared.TestUserUUID]
	if device.Name != "new-device-name" {
		t.Errorf("Expected project config device name 'new-device-name', got '%s'", device.Name)
	}

	// Verify user config was also updated.
	userConfig, err = configs.LoadUserConfig()
	if err != nil {
		t.Fatalf("Failed to reload user config: %v", err)
	}
	deviceName, exists := userConfig.Projects[projectUUID]
	if !exists {
		t.Errorf("Expected project entry in user config, but not found")
	}
	if deviceName != "new-device-name" {
		t.Errorf("Expected user config device name 'new-device-name', got '%s'", deviceName)
	}
}

// Tests that renaming another user's device does NOT update the current user's config.
func testRenameOtherUserDeviceDoesNotUpdateUserConfig(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-rename-other-*")
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

	// Get project UUID for verification.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectUUID := projectConfig.Project.UUID

	// Add another user's device (different UUID from TestUserUUID).
	otherUserUUID := shared.TestUser2UUID
	projectConfig.Users = map[string]string{
		shared.TestUserUUID: shared.TestUserEmail,
		otherUserUUID:       shared.TestUser2Email,
	}
	projectConfig.Devices = map[string]configs.DeviceConfig{
		shared.TestUserUUID: {
			Email:     shared.TestUserEmail,
			Name:      "my-device",
			CreatedAt: time.Now(),
		},
		otherUserUUID: {
			Email:     shared.TestUser2Email,
			Name:      "other-old-device",
			CreatedAt: time.Now(),
		},
	}

	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	// Set up user config with the current user's project entry.
	userConfig, err := configs.LoadUserConfig()
	if err != nil {
		t.Fatalf("Failed to load user config: %v", err)
	}
	userConfig.Projects[projectUUID] = "my-device"
	if err := configs.SaveUserConfig(userConfig); err != nil {
		t.Fatalf("Failed to save user config: %v", err)
	}

	// Rename the OTHER user's device.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("rename-device", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "rename-device", "--user", shared.TestUser2Email, "other-new-device"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "renamed to") {
		t.Errorf("Expected 'renamed to' message not found in output: %s", output)
	}

	// Verify project config was updated for the other user.
	projectConfig, err = configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to reload project config: %v", err)
	}
	otherDevice := projectConfig.Devices[otherUserUUID]
	if otherDevice.Name != "other-new-device" {
		t.Errorf("Expected project config device name 'other-new-device', got '%s'", otherDevice.Name)
	}

	// Verify user config was NOT changed (should still be "my-device").
	userConfig, err = configs.LoadUserConfig()
	if err != nil {
		t.Fatalf("Failed to reload user config: %v", err)
	}
	deviceName, exists := userConfig.Projects[projectUUID]
	if !exists {
		t.Errorf("Expected project entry in user config, but not found")
	}
	if deviceName != "my-device" {
		t.Errorf("Expected user config device name to remain 'my-device', got '%s'", deviceName)
	}
}
