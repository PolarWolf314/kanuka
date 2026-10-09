package create

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsCreateUsesConfiguredDeviceName verifies that create registers
// the device using the user's configured default_device_name (set via
// `kanuka config init --device`) instead of deriving one from the hostname.
func TestSecretsCreateUsesConfiguredDeviceName(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	tempDir, err := os.MkdirTemp("", "kanuka-test-create-device-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-device-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Set the user's configured default device name, as
	// `kanuka config init --email … --device recovery-key` would.
	userConfig, err := configs.LoadUserConfig()
	if err != nil {
		t.Fatalf("Failed to load user config: %v", err)
	}
	userConfig.User.DefaultDeviceName = "recovery-key"
	if err := configs.SaveUserConfig(userConfig); err != nil {
		t.Fatalf("Failed to save user config: %v", err)
	}

	shared.InitializeProjectStructureOnly(t, tempDir, tempUserDir)

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Keys created for") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	if !strings.Contains(output, "recovery-key") {
		t.Errorf("Expected configured device name 'recovery-key' in output: %s", output)
	}

	// The project config must register the device under the configured name.
	projectConfigPath := filepath.Join(tempDir, ".kanuka", "config.toml")
	content, err := os.ReadFile(projectConfigPath)
	if err != nil {
		t.Fatalf("Failed to read project config: %v", err)
	}
	if !strings.Contains(string(content), `name = "recovery-key"`) {
		t.Errorf("Expected device name 'recovery-key' in project config:\n%s", string(content))
	}
}
