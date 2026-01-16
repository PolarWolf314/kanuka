package config

import (
	"os"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestConfigShow contains tests for the `kanuka config show` command.
func TestConfigShow(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("ShowUserConfig", func(t *testing.T) {
		testConfigShowUserConfig(t, originalWd, originalUserSettings)
	})

	t.Run("ShowUserConfigJSON", func(t *testing.T) {
		testConfigShowUserConfigJSON(t, originalWd, originalUserSettings)
	})

	t.Run("ShowNoUserConfig", func(t *testing.T) {
		testConfigShowNoUserConfig(t, originalWd, originalUserSettings)
	})

	t.Run("ShowProjectConfig", func(t *testing.T) {
		testConfigShowProjectConfig(t, originalWd, originalUserSettings)
	})

	t.Run("ShowProjectConfigJSON", func(t *testing.T) {
		testConfigShowProjectConfigJSON(t, originalWd, originalUserSettings)
	})

	t.Run("ShowProjectConfigNotInProject", func(t *testing.T) {
		testConfigShowProjectConfigNotInProject(t, originalWd, originalUserSettings)
	})
}

// testConfigShowUserConfig tests showing user configuration.
func testConfigShowUserConfig(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-config-show-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Setup with user config.
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Run config show command.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("show", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Verify output contains user config info.
	if !strings.Contains(output, "User Configuration") {
		t.Errorf("Expected 'User Configuration' in output, got: %s", output)
	}
	if !strings.Contains(output, shared.TestUserEmail) {
		t.Errorf("Expected email '%s' in output, got: %s", shared.TestUserEmail, output)
	}
	if !strings.Contains(output, shared.TestUserUUID[:8]) {
		t.Errorf("Expected user UUID prefix in output, got: %s", output)
	}
}

// testConfigShowUserConfigJSON tests showing user configuration in JSON format.
func testConfigShowUserConfigJSON(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-config-show-json-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Setup with user config.
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Run config show --json command.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLIWithArgs("show", []string{"--json"}, nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Verify output is valid JSON with expected fields.
	if !strings.Contains(output, `"Email"`) {
		t.Errorf("Expected JSON with 'Email' field, got: %s", output)
	}
	if !strings.Contains(output, `"UUID"`) {
		t.Errorf("Expected JSON with 'UUID' field, got: %s", output)
	}
	if !strings.Contains(output, shared.TestUserEmail) {
		t.Errorf("Expected email '%s' in JSON output, got: %s", shared.TestUserEmail, output)
	}
}

// testConfigShowNoUserConfig tests showing config when no user config exists.
func testConfigShowNoUserConfig(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-config-show-none-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Setup without user config.
	shared.SetupTestEnvironmentWithoutUserConfig(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Run config show command.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLI("show", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Verify output indicates no config found.
	if !strings.Contains(output, "⚠ No user configuration found") {
		t.Errorf("Expected 'No user configuration found' in output, got: %s", output)
	}
	if !strings.Contains(output, "config init") {
		t.Errorf("Expected suggestion to run 'config init', got: %s", output)
	}
}

// testConfigShowProjectConfig tests showing project configuration.
func testConfigShowProjectConfig(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-config-show-project-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Setup with user config.
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize a project first.
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Run config show --project command.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLIWithArgs("show", []string{"--project"}, nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Verify output contains project config info.
	if !strings.Contains(output, "Project Configuration") {
		t.Errorf("Expected 'Project Configuration' in output, got: %s", output)
	}
	if !strings.Contains(output, "Project ID:") {
		t.Errorf("Expected 'Project ID:' in output, got: %s", output)
	}
	if !strings.Contains(output, "Project Name:") {
		t.Errorf("Expected 'Project Name:' in output, got: %s", output)
	}
	if !strings.Contains(output, shared.TestUserEmail) {
		t.Errorf("Expected user email '%s' in output, got: %s", shared.TestUserEmail, output)
	}
}

// testConfigShowProjectConfigJSON tests showing project configuration in JSON format.
func testConfigShowProjectConfigJSON(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-config-show-project-json-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Setup with user config.
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize a project first.
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Run config show --project --json command.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLIWithArgs("show", []string{"--project", "--json"}, nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Verify output is valid JSON with expected fields.
	if !strings.Contains(output, `"Project"`) {
		t.Errorf("Expected JSON with 'Project' field, got: %s", output)
	}
	if !strings.Contains(output, `"Name"`) {
		t.Errorf("Expected JSON with 'Name' field, got: %s", output)
	}
	if !strings.Contains(output, `"Devices"`) {
		t.Errorf("Expected JSON with 'Devices' field, got: %s", output)
	}
}

// testConfigShowProjectConfigNotInProject tests showing project config when not in a project.
func testConfigShowProjectConfigNotInProject(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-config-show-no-project-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Setup with user config but no project.
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Run config show --project command (should fail gracefully).
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateConfigTestCLIWithArgs("show", []string{"--project"}, nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Verify output indicates not in a project.
	if !strings.Contains(output, "✗ Not in a Kanuka project directory") {
		t.Errorf("Expected 'Not in a Kanuka project directory' in output, got: %s", output)
	}
	if !strings.Contains(output, "secrets init") {
		t.Errorf("Expected suggestion to run 'secrets init', got: %s", output)
	}
}
