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

func TestSecretsRemoveIntegration(t *testing.T) {
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

	secondUserUUID := "second-user-uuid-1234"
	secondUserEmail := "seconduser@example.com"

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
	pubKey := string(pem.EncodeToMemory(pubPem))

	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectConfig.Users[secondUserUUID] = secondUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	cmd.ResetGlobalState()
	registerCmd := shared.CreateTestCLIWithArgs("register", []string{"--pubkey", pubKey, "--user", secondUserEmail}, nil, nil, false, false)
	if err := registerCmd.Execute(); err != nil {
		t.Fatalf("Failed to register second user: %v", err)
	}

	kanukaDir := filepath.Join(tempDir, ".kanuka")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	secretFiles, err := os.ReadDir(secretsDir)
	if err != nil {
		t.Fatalf("Failed to read secrets directory: %v", err)
	}
	t.Logf("Files in secrets directory: %v", secretsDir)
	for _, file := range secretFiles {
		t.Logf("  - %s", file.Name())
	}

	registeredKanukaKeyPath := filepath.Join(secretsDir, secondUserUUID+".kanuka")

	if _, statErr := os.Stat(registeredKanukaKeyPath); os.IsNotExist(statErr) {
		t.Fatal("Kanuka key file should exist after registration")
	}

	t.Logf("Found kanuka key file: %v", registeredKanukaKeyPath)

	relativeKanukaKeyPath := filepath.Join(".kanuka", "secrets", secondUserUUID+".kanuka")
	cmd.ResetGlobalState()
	revokeCmd := shared.CreateTestCLIWithArgs("revoke", []string{"--file", relativeKanukaKeyPath}, nil, nil, false, false)
	if err := revokeCmd.Execute(); err != nil {
		t.Errorf("Remove command should succeed: %v", err)
	}

	if _, err := os.Stat(registeredKanukaKeyPath); !os.IsNotExist(err) {
		t.Error("Kanuka key file should be revoked")
	}
}
