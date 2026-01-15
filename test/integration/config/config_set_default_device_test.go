package config

import (
	"os"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestConfigSetDefaultDevice contains tests for the `kanuka config set-default-device` command.
func TestConfigSetDefaultDevice(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("SetDefaultDevice", func(t *testing.T) {
		testSetDefaultDevice(t, originalWd, originalUserSettings)
	})

	t.Run("SetDefaultDeviceInvalidName", func(t *testing.T) {
		testSetDefaultDeviceInvalidName(t, originalWd, originalUserSettings)
	})

	t.Run("SetDefaultDeviceSameValue", func(t *testing.T) {
		testSetDefaultDeviceSameValue(t, originalWd, originalUserSettings)
	})

	t.Run("SetDefaultDeviceUpdate", func(t *testing.T) {
		testSetDefaultDeviceUpdate(t, originalWd, originalUserSettings)
	})
}

func testSetDefaultDevice(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-set-default-device-*")
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

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("set-default-device", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "set-default-device", "my-laptop"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "Default device name set to") {
		t.Errorf("Expected success message not found in output: %s", output)
	}
	if !strings.Contains(output, "my-laptop") {
		t.Errorf("Expected device name 'my-laptop' not found in output: %s", output)
	}

	userConfig, err := configs.LoadUserConfig()
	if err != nil {
		t.Fatalf("Failed to load user config: %v", err)
	}

	if userConfig.User.DefaultDeviceName != "my-laptop" {
		t.Errorf("Expected default device name 'my-laptop', got '%s'", userConfig.User.DefaultDeviceName)
	}
}

func testSetDefaultDeviceInvalidName(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-set-default-device-invalid-*")
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

	invalidNames := []string{
		"my laptop",
		"my@laptop",
		"laptop!",
	}

	for _, invalidName := range invalidNames {
		output, err := shared.CaptureOutput(func() error {
			cmd := shared.CreateConfigTestCLI("set-default-device", nil, nil, true, false)
			cmd.SetArgs([]string{"config", "set-default-device", invalidName})
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

func testSetDefaultDeviceSameValue(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-set-default-device-same-*")
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

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("set-default-device", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "set-default-device", "my-laptop"})
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to set initial default device name: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("set-default-device", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "set-default-device", "my-laptop"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "already set") {
		t.Errorf("Expected 'already set' message not found in output: %s", output)
	}
}

func testSetDefaultDeviceUpdate(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-set-default-device-update-*")
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

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("set-default-device", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "set-default-device", "old-laptop"})
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to set initial default device name: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("set-default-device", nil, nil, true, false)
		cmd.SetArgs([]string{"config", "set-default-device", "new-laptop"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "Default device name set to") {
		t.Errorf("Expected success message not found in output: %s", output)
	}
	if !strings.Contains(output, "new-laptop") {
		t.Errorf("Expected device name 'new-laptop' not found in output: %s", output)
	}

	userConfig, err := configs.LoadUserConfig()
	if err != nil {
		t.Fatalf("Failed to load user config: %v", err)
	}

	if userConfig.User.DefaultDeviceName != "new-laptop" {
		t.Errorf("Expected default device name 'new-laptop', got '%s'", userConfig.User.DefaultDeviceName)
	}
}
