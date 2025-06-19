package remove

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PolarWolf314/kanuka/cmd"
	"github.com/PolarWolf314/kanuka/internal/configs"
)

func TestRemoveCommand_ProjectStateRequirements(t *testing.T) {
	// Save original state
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings

	t.Run("RemoveWithoutInitialization", func(t *testing.T) {
		testRemoveWithoutInitialization(t, originalWd, originalUserSettings)
	})

	t.Run("RemoveInNonKanukaProject", func(t *testing.T) {
		testRemoveInNonKanukaProject(t, originalWd, originalUserSettings)
	})
}

func testRemoveWithoutInitialization(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Setup test environment
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer os.Chdir(originalWd)

	// Setup user settings
	configs.UserKanukaSettings = &configs.UserSettings{
		UserKeysPath:    filepath.Join(tempUserDir, "keys"),
		UserConfigsPath: filepath.Join(tempUserDir, "config"),
		Username:        "testuser",
	}
	defer func() {
		configs.UserKanukaSettings = originalUserSettings
	}()

	// Test remove command without initialization
	cmd.ResetGlobalState()
	secretsCmd := cmd.GetSecretsCmd()
	secretsCmd.SetArgs([]string{"remove", "--user", "testuser2"})

	err = secretsCmd.Execute()
	if err != nil {
		t.Errorf("Command should not return error, but should show initialization required message: %v", err)
	}
}

func testRemoveInNonKanukaProject(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Setup test environment
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer os.Chdir(originalWd)

	// Setup user settings
	configs.UserKanukaSettings = &configs.UserSettings{
		UserKeysPath:    filepath.Join(tempUserDir, "keys"),
		UserConfigsPath: filepath.Join(tempUserDir, "config"),
		Username:        "testuser",
	}
	defer func() {
		configs.UserKanukaSettings = originalUserSettings
	}()

	// Create a .kanuka directory but without proper structure
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	if err := os.MkdirAll(kanukaDir, 0755); err != nil {
		t.Fatalf("Failed to create kanuka directory: %v", err)
	}

	// Test remove command in non-kanuka project
	cmd.ResetGlobalState()
	secretsCmd := cmd.GetSecretsCmd()
	secretsCmd.SetArgs([]string{"remove", "--user", "testuser2"})

	err = secretsCmd.Execute()
	if err != nil {
		t.Errorf("Command should not return error, but should show project not found message: %v", err)
	}
}
