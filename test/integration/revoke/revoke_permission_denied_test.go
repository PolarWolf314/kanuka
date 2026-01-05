package revoke

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PolarWolf314/kanuka/cmd"
	"github.com/PolarWolf314/kanuka/internal/configs"
)

func TestRevokeCommand_PermissionDenied(t *testing.T) {
	// Skip on Windows as permissions work differently
	if os.Getenv("SKIP_PERMISSION_TESTS") == "true" {
		t.Skip("Skipping permission tests")
	}

	// Save original state
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings

	t.Run("RemoveWithNoWritePermissionOnDirectory", func(t *testing.T) {
		testRevokeWithNoWritePermissionOnDirectory(t, originalWd, originalUserSettings)
	})
}

func testRevokeWithNoWritePermissionOnDirectory(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
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
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Fatalf("Failed to restore working directory: %v", err)
		}
	}()

	// Setup user settings
	configs.UserKanukaSettings = &configs.UserSettings{
		UserKeysPath:    filepath.Join(tempUserDir, "keys"),
		UserConfigsPath: filepath.Join(tempUserDir, "config"),
		Username:        "testuser",
	}
	defer func() {
		configs.UserKanukaSettings = originalUserSettings
	}()

	// Create kanuka directories
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	if err := os.MkdirAll(publicKeysDir, 0755); err != nil {
		t.Fatalf("Failed to create public keys directory: %v", err)
	}
	if err := os.MkdirAll(secretsDir, 0755); err != nil {
		t.Fatalf("Failed to create secrets directory: %v", err)
	}

	// Create test user files
	testUser := "testuser2"
	publicKeyPath := filepath.Join(publicKeysDir, testUser+".pub")
	kanukaKeyPath := filepath.Join(secretsDir, testUser+".kanuka")

	// Create dummy files
	err = os.WriteFile(publicKeyPath, []byte("dummy public key"), 0600)
	if err != nil {
		t.Fatalf("Failed to create public key file: %v", err)
	}

	err = os.WriteFile(kanukaKeyPath, []byte("dummy kanuka key"), 0600)
	if err != nil {
		t.Fatalf("Failed to create kanuka key file: %v", err)
	}

	// Make directories read-only
	if err := os.Chmod(publicKeysDir, 0555); err != nil {
		t.Fatalf("Failed to change directory permissions: %v", err)
	}
	if err := os.Chmod(secretsDir, 0555); err != nil {
		t.Fatalf("Failed to change directory permissions: %v", err)
	}

	// Ensure permissions are restored for cleanup
	defer func() {
		if err := os.Chmod(publicKeysDir, 0755); err != nil {
			t.Logf("Warning: Failed to restore permissions on public keys directory: %v", err)
		}
		if err := os.Chmod(secretsDir, 0755); err != nil {
			t.Logf("Warning: Failed to restore permissions on secrets directory: %v", err)
		}
	}()

	// Remove the user
	cmd.ResetGlobalState()
	secretsCmd := cmd.GetSecretsCmd()
	secretsCmd.SetArgs([]string{"revoke", "--user", testUser})

	err = secretsCmd.Execute()
	if err != nil {
		t.Errorf("Remove command should not return error even with permission issues: %v", err)
	}

	// Verify files still exist (removal should have failed due to permissions)
	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		t.Error("Public key file should still exist due to permission issues")
	}

	if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
		t.Error("Kanuka key file should still exist due to permission issues")
	}
}
