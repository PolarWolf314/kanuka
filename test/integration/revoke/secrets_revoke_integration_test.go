package revoke

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PolarWolf314/kanuka/cmd"
	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

func TestSecretsRemoveIntegration(t *testing.T) {
	// Save original state
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings

	t.Run("RemoveUserAfterRegistration", func(t *testing.T) {
		testRemoveUserAfterRegistration(t, originalWd, originalUserSettings)
	})
}

func testRemoveUserAfterRegistration(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
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
	initCmd := shared.CreateTestCLIWithArgs("init", []string{"--yes"}, nil, nil, false, false)
	if err := initCmd.Execute(); err != nil {
		t.Fatalf("Failed to initialize project: %v", err)
	}

	// Register a second user using --file flag
	secondUser := "seconduser"
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	// Generate key pair for second user, saving public key to project directory
	privateKeyPath := filepath.Join(tempUserDir, "private.key")
	projectPublicKeyPath := filepath.Join(publicKeysDir, secondUser+".pub")

	if err := shared.GenerateRSAKeyPair(privateKeyPath, projectPublicKeyPath); err != nil {
		t.Fatalf("Failed to generate RSA key pair: %v", err)
	}

	// Register the second user using --file flag
	cmd.ResetGlobalState()
	registerCmd := shared.CreateTestCLIWithArgs("register", []string{"--file", projectPublicKeyPath}, nil, nil, false, false)
	if err := registerCmd.Execute(); err != nil {
		t.Fatalf("Failed to register second user: %v", err)
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

	// Verify second user's kanuka file exists
	registeredKanukaKeyPath := filepath.Join(secretsDir, secondUser+".kanuka")

	var statErr error
	if _, statErr = os.Stat(registeredKanukaKeyPath); os.IsNotExist(statErr) {
		t.Fatal("Kanuka key file should exist after registration")
	}

	t.Logf("Found kanuka key file: %v", registeredKanukaKeyPath)

	// Remove the user using --file flag (use relative path)
	relativeKanukaKeyPath := filepath.Join(".kanuka", "secrets", secondUser+".kanuka")
	cmd.ResetGlobalState()
	revokeCmd := shared.CreateTestCLIWithArgs("revoke", []string{"--file", relativeKanukaKeyPath}, nil, nil, false, false)
	if err := revokeCmd.Execute(); err != nil {
		t.Errorf("Remove command should succeed: %v", err)
	}

	// Verify kanuka key file is revoked
	if _, statErr = os.Stat(registeredKanukaKeyPath); !os.IsNotExist(statErr) {
		t.Error("Kanuka key file should be revoked")
	}
}
