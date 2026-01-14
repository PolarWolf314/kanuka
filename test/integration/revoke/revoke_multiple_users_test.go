package revoke

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/PolarWolf314/kanuka/cmd"
	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

func TestRevokeCommand_MultipleUsers(t *testing.T) {
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

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Fatalf("Failed to restore working directory: %v", err)
		}
	}()

	configs.UserKanukaSettings = &configs.UserSettings{
		UserKeysPath:    filepath.Join(tempUserDir, "keys"),
		UserConfigsPath: filepath.Join(tempUserDir, "config"),
		Username:        "testuser",
	}
	defer func() {
		configs.UserKanukaSettings = originalUserSettings
	}()

	if err := os.MkdirAll(configs.UserKanukaSettings.UserKeysPath, 0755); err != nil {
		t.Fatalf("Failed to create user keys directory: %v", err)
	}
	if err := os.MkdirAll(configs.UserKanukaSettings.UserConfigsPath, 0755); err != nil {
		t.Fatalf("Failed to create user configs directory: %v", err)
	}

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

	cmd.ResetGlobalState()
	initCmd := shared.CreateTestCLIWithArgs("init", []string{"--yes"}, nil, nil, false, false)
	if err := initCmd.Execute(); err != nil {
		t.Fatalf("Failed to initialize project: %v", err)
	}

	users := []struct {
		uuid   string
		email  string
		pubKey string
	}{
		{"user1-uuid-1234", "user1@example.com", ""},
		{"user2-uuid-1234", "user2@example.com", ""},
		{"user3-uuid-1234", "user3@example.com", ""},
	}

	kanukaDir := filepath.Join(tempDir, ".kanuka")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	for i, user := range users {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("Failed to generate RSA key: %v", err)
		}

		pubASN1, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
		if err != nil {
			t.Fatalf("Failed to marshal public key: %v", err)
		}

		pubPem := &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubASN1,
		}
		user.pubKey = string(pem.EncodeToMemory(pubPem))

		projectConfig, err := configs.LoadProjectConfig()
		if err != nil {
			t.Fatalf("Failed to load project config: %v", err)
		}
		projectConfig.Users[user.uuid] = user.email
		if err := configs.SaveProjectConfig(projectConfig); err != nil {
			t.Fatalf("Failed to save project config: %v", err)
		}

		cmd.ResetGlobalState()
		registerCmd := shared.CreateTestCLIWithArgs("register", []string{"--pubkey", user.pubKey, "--user", user.email}, nil, nil, false, false)
		if err := registerCmd.Execute(); err != nil {
			t.Fatalf("Failed to register user %s: %v", user.email, err)
		}

		t.Logf("Registered user %d: %s", i+1, user.email)
	}

	secretFiles, err := os.ReadDir(secretsDir)
	if err != nil {
		t.Fatalf("Failed to read secrets directory: %v", err)
	}
	t.Logf("Files in secrets directory: %v", secretsDir)
	for _, file := range secretFiles {
		t.Logf("  - %s", file.Name())
	}

	userToRemove := users[1]
	relativeKanukaKeyPath := filepath.Join(".kanuka", "secrets", userToRemove.uuid+".kanuka")
	cmd.ResetGlobalState()
	revokeCmd := shared.CreateTestCLIWithArgs("revoke", []string{"--file", relativeKanukaKeyPath}, nil, nil, false, false)
	if err := revokeCmd.Execute(); err != nil {
		t.Errorf("Remove command should succeed: %v", err)
	}

	secretFilesAfter, err := os.ReadDir(secretsDir)
	if err != nil {
		t.Fatalf("Failed to read secrets directory after removal: %v", err)
	}
	t.Logf("Files in secrets directory after removal:")
	for _, file := range secretFilesAfter {
		t.Logf("  - %s", file.Name())
	}

	removedUserKanukaKeyPath := filepath.Join(secretsDir, userToRemove.uuid+".kanuka")

	if _, err := os.Stat(removedUserKanukaKeyPath); !os.IsNotExist(err) {
		t.Errorf("Kanuka key file for removed user %s should be gone", userToRemove.email)
	}

	if len(secretFilesAfter) >= len(secretFiles) {
		t.Errorf("Expected fewer secret files after removal, but got %d before and %d after",
			len(secretFiles), len(secretFilesAfter))
	}

	for _, user := range users {
		if user.uuid == userToRemove.uuid {
			continue
		}

		kanukaKeyPath := filepath.Join(secretsDir, user.uuid+".kanuka")

		if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
			t.Errorf("Kanuka key file for user %s should still exist", user.email)
		}
	}
}
