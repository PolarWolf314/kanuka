package revoke

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PolarWolf314/kanuka/cmd"
	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

func TestRevokeCommand_FilesystemEdgeCases(t *testing.T) {
	// Save original state
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings

	t.Run("RemoveWithOnlyPublicKeyFile", func(t *testing.T) {
		testRevokeWithOnlyPublicKeyFile(t, originalWd, originalUserSettings)
	})

	t.Run("RemoveWithOnlyKanukaKeyFile", func(t *testing.T) {
		testRevokeWithOnlyKanukaKeyFile(t, originalWd, originalUserSettings)
	})

	// Skip permission tests on Windows as they work differently
	if os.Getenv("SKIP_PERMISSION_TESTS") != "true" {
		t.Run("RemoveWithReadOnlyPublicKeyFile", func(t *testing.T) {
			testRevokeWithReadOnlyPublicKeyFile(t, originalWd, originalUserSettings)
		})
	}
}

func testRevokeWithOnlyPublicKeyFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
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

	// Create test user files - only public key (creating a .kanuka file to use --file)
	testUserUUID := "testuser2-uuid"
	publicKeyPath := filepath.Join(publicKeysDir, testUserUUID+".pub")
	kanukaKeyPath := filepath.Join(secretsDir, testUserUUID+".kanuka")

	// Create dummy public key file
	err = os.WriteFile(publicKeyPath, []byte("dummy public key"), 0600)
	if err != nil {
		t.Fatalf("Failed to create public key file: %v", err)
	}

	// Create dummy kanuka file to use with --file flag
	err = os.WriteFile(kanukaKeyPath, []byte("dummy kanuka key"), 0600)
	if err != nil {
		t.Fatalf("Failed to create kanuka key file: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		t.Fatal("Public key file should exist before removal")
	}

	// Remove the user using --file flag (use relative path)
	relativeKanukaKeyPath := filepath.Join(".kanuka", "secrets", testUserUUID+".kanuka")
	cmd.ResetGlobalState()
	testCmd := shared.CreateTestCLIWithArgs("revoke", []string{"--file", relativeKanukaKeyPath}, nil, nil, false, false)

	err = testCmd.Execute()
	if err != nil {
		t.Errorf("Remove command should succeed: %v", err)
	}

	// Verify file is removed
	if _, err := os.Stat(publicKeyPath); !os.IsNotExist(err) {
		t.Error("Public key file should be revoked")
	}
}

func testRevokeWithOnlyKanukaKeyFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
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

	// Create test user files - only kanuka key
	testUserUUID := "testuser2-uuid"
	kanukaKeyPath := filepath.Join(secretsDir, testUserUUID+".kanuka")

	// Create dummy kanuka key file
	err = os.WriteFile(kanukaKeyPath, []byte("dummy kanuka key"), 0600)
	if err != nil {
		t.Fatalf("Failed to create kanuka key file: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
		t.Fatal("Kanuka key file should exist before removal")
	}

	// Remove the user using --file flag (use relative path)
	relativeKanukaKeyPath := filepath.Join(".kanuka", "secrets", testUserUUID+".kanuka")
	cmd.ResetGlobalState()
	testCmd := shared.CreateTestCLIWithArgs("revoke", []string{"--file", relativeKanukaKeyPath}, nil, nil, false, false)

	err = testCmd.Execute()
	if err != nil {
		t.Errorf("Remove command should succeed: %v", err)
	}

	// Verify file is removed
	if _, err := os.Stat(kanukaKeyPath); !os.IsNotExist(err) {
		t.Error("Kanuka key file should be revoked")
	}
}

func testRevokeWithReadOnlyPublicKeyFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
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
	testUserUUID := "testuser2-uuid"
	publicKeyPath := filepath.Join(publicKeysDir, testUserUUID+".pub")
	kanukaKeyPath := filepath.Join(secretsDir, testUserUUID+".kanuka")

	// Create dummy files
	err = os.WriteFile(publicKeyPath, []byte("dummy public key"), 0600)
	if err != nil {
		t.Fatalf("Failed to create public key file: %v", err)
	}

	err = os.WriteFile(kanukaKeyPath, []byte("dummy kanuka key"), 0600)
	if err != nil {
		t.Fatalf("Failed to create kanuka key file: %v", err)
	}

	// Make public key file read-only
	if err := os.Chmod(publicKeyPath, 0444); err != nil {
		t.Fatalf("Failed to change file permissions: %v", err)
	}

	// Remove the user using --file flag (use relative path)
	relativeKanukaKeyPath := filepath.Join(".kanuka", "secrets", testUserUUID+".kanuka")
	cmd.ResetGlobalState()
	testCmd := shared.CreateTestCLIWithArgs("revoke", []string{"--file", relativeKanukaKeyPath}, nil, nil, false, false)

	err = testCmd.Execute()
	if err != nil {
		t.Errorf("Remove command should not return error even with permission issues: %v", err)
	}

	// Restore permissions to allow cleanup
	if err := os.Chmod(publicKeyPath, 0600); err != nil {
		t.Logf("Warning: Failed to restore permissions on public key file: %v", err)
	}
}
