package init_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsInitEnvironment contains environment variable edge case tests for the `kanuka secrets init` command.
func TestSecretsInitEnvironment(t *testing.T) {
	// Save original working directory and settings
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings

	t.Run("InitWithInvalidXDGDataHome", func(t *testing.T) {
		testInitWithInvalidXDGDataHome(t, originalWd, originalUserSettings)
	})

	t.Run("InitWithXDGDataHomeAsFile", func(t *testing.T) {
		testInitWithXDGDataHomeAsFile(t, originalWd, originalUserSettings)
	})
}

func testInitWithInvalidXDGDataHome(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-invalid-xdg-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Save original XDG_DATA_HOME
	originalXDG := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", originalXDG)

	// Set XDG_DATA_HOME to non-existent path with invalid characters
	invalidPath := "/non/existent/path/with\x00null/chars"
	os.Setenv("XDG_DATA_HOME", invalidPath)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})

	// The behavior may vary by system, but it should either:
	// 1. Succeed by falling back to default paths, or
	// 2. Fail with a clear error message
	if err != nil {
		// If it fails, should have a clear error message
		if !strings.Contains(output, "failed") {
			t.Errorf("Expected clear error message for invalid XDG_DATA_HOME, got: %s", output)
		}
	} else {
		// If it succeeds, should have created the project structure
		if !strings.Contains(output, "initialized successfully") {
			t.Errorf("Expected success message, got: %s", output)
		}
	}
}

func testInitWithXDGDataHomeAsFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-xdg-file-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a file to use as XDG_DATA_HOME
	xdgFile := filepath.Join(tempDir, "xdg-data-file")
	// #nosec G306 -- Writing a file that should be modifiable
	if err := os.WriteFile(xdgFile, []byte("this is a file"), 0644); err != nil {
		t.Fatalf("Failed to create XDG file: %v", err)
	}

	// Save original XDG_DATA_HOME
	originalXDG := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", originalXDG)

	// Set XDG_DATA_HOME to the file
	os.Setenv("XDG_DATA_HOME", xdgFile)

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Cleanup function to restore original state
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
	})

	// Manually set the user settings to simulate what would happen if XDG_DATA_HOME pointed to a file
	// This simulates the behavior where the application tries to create kanuka/keys under a file path
	configs.UserKanukaSettings = &configs.UserSettings{
		UserKeysPath:    filepath.Join(xdgFile, "kanuka", "keys"), // This will fail because xdgFile is a file
		UserConfigsPath: filepath.Join(tempDir, "config"),
		Username:        "testuser",
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})

	if err == nil {
		t.Errorf("Expected command to fail due to XDG_DATA_HOME being a file, but it succeeded")
		t.Errorf("Output: %s", output)
		return
	}

	if !strings.Contains(output, "failed") && !strings.Contains(output, "not a directory") {
		t.Errorf("Expected error message about failed directory creation, got: %s", output)
	}
}
