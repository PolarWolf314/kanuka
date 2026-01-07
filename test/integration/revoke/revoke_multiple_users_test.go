package revoke

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PolarWolf314/kanuka/cmd"
	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

func TestRevokeCommand_MultipleUsers(t *testing.T) {
	// Save original state
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings

	t.Run("RemoveOneUserFromMultipleUsers", func(t *testing.T) {
		testRevokeOneUserFromMultipleUsers(t, originalWd, originalUserSettings)
	})
}

func testRevokeOneUserFromMultipleUsers(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
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

	// Create user directories
	if err := os.MkdirAll(configs.UserKanukaSettings.UserKeysPath, 0755); err != nil {
		t.Fatalf("Failed to create user keys directory: %v", err)
	}
	if err := os.MkdirAll(configs.UserKanukaSettings.UserConfigsPath, 0755); err != nil {
		t.Fatalf("Failed to create user configs directory: %v", err)
	}

	// Create user config with UUID so init doesn't prompt for interactive setup.
	userConfig := &configs.UserConfig{
		User: configs.User{
			UUID:  shared.TestUserUUID,
			Email: shared.TestUserEmail,
		},
		Projects: make(map[string]configs.UserProjectEntry),
	}
	if err := configs.SaveUserConfig(userConfig); err != nil {
		t.Fatalf("Failed to save user config: %v", err)
	}

	// Initialize the project
	cmd.ResetGlobalState()
	secretsCmd := cmd.GetSecretsCmd()
	secretsCmd.SetArgs([]string{"init", "--yes"})
	if err := secretsCmd.Execute(); err != nil {
		t.Fatalf("Failed to initialize project: %v", err)
	}

	// Define multiple users (these will be used as filenames/identifiers)
	users := []string{"user1", "user2", "user3"}
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	// Create key pairs and register all users using --file flag
	for i, user := range users {
		// Generate key pair for user
		privateKeyPath := filepath.Join(tempUserDir, user+".key")
		// Save directly to project's public_keys directory so filename becomes user identifier
		projectPublicKeyPath := filepath.Join(publicKeysDir, user+".pub")

		if err := shared.GenerateRSAKeyPair(privateKeyPath, projectPublicKeyPath); err != nil {
			t.Fatalf("Failed to generate RSA key pair for user %s: %v", user, err)
		}

		// Register the user using --file flag (uses filename as user identifier)
		cmd.ResetGlobalState()
		secretsCmd = cmd.GetSecretsCmd()
		secretsCmd.SetArgs([]string{"register", "--file", projectPublicKeyPath})
		if err := secretsCmd.Execute(); err != nil {
			t.Fatalf("Failed to register user %s: %v", user, err)
		}

		t.Logf("Registered user %d: %s", i+1, user)
	}

	// List all files in the directories to debug
	publicKeyFiles, err := os.ReadDir(publicKeysDir)
	if err != nil {
		t.Fatalf("Failed to read public keys directory: %v", err)
	}
	t.Logf("Files in public keys directory: %v", publicKeysDir)
	for _, file := range publicKeyFiles {
		t.Logf("  - %s", file.Name())
	}

	secretFiles, err := os.ReadDir(secretsDir)
	if err != nil {
		t.Fatalf("Failed to read secrets directory: %v", err)
	}
	t.Logf("Files in secrets directory: %v", secretsDir)
	for _, file := range secretFiles {
		t.Logf("  - %s", file.Name())
	}

	// Remove the second user using --file flag (use relative path)
	userToRemove := users[1] // user2
	relativeKanukaKeyPath := filepath.Join(".kanuka", "secrets", userToRemove+".kanuka")
	cmd.ResetGlobalState()
	secretsCmd = cmd.GetSecretsCmd()
	secretsCmd.SetArgs([]string{"revoke", "--file", relativeKanukaKeyPath})
	if err := secretsCmd.Execute(); err != nil {
		t.Errorf("Remove command should succeed: %v", err)
	}

	// List files again after removal to see what changed
	publicKeyFilesAfter, err := os.ReadDir(publicKeysDir)
	if err != nil {
		t.Fatalf("Failed to read public keys directory after removal: %v", err)
	}
	t.Logf("Files in public keys directory after removal:")
	for _, file := range publicKeyFilesAfter {
		t.Logf("  - %s", file.Name())
	}

	secretFilesAfter, err := os.ReadDir(secretsDir)
	if err != nil {
		t.Fatalf("Failed to read secrets directory after removal: %v", err)
	}
	t.Logf("Files in secrets directory after removal:")
	for _, file := range secretFilesAfter {
		t.Logf("  - %s", file.Name())
	}

	// Verify that the kanuka key file for the removed user is gone
	removedUserKanukaKeyPath := filepath.Join(secretsDir, userToRemove+".kanuka")

	if _, err := os.Stat(removedUserKanukaKeyPath); !os.IsNotExist(err) {
		t.Errorf("Kanuka key file for removed user %s should be gone", userToRemove)
	}

	// Verify that the number of secret files has decreased
	if len(secretFilesAfter) >= len(secretFiles) {
		t.Errorf("Expected fewer secret files after removal, but got %d before and %d after",
			len(secretFiles), len(secretFilesAfter))
	}

	// Verify other users' kanuka key files still exist
	for _, user := range users {
		if user == userToRemove {
			continue // Skip the removed user
		}

		kanukaKeyPath := filepath.Join(secretsDir, user+".kanuka")

		if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
			t.Errorf("Kanuka key file for user %s should still exist", user)
		}
	}
}
