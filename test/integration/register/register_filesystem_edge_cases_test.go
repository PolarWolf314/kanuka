package register

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/cmd"
	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

func TestSecretsRegisterFilesystemEdgeCases(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings

	t.Run("RegisterWithReadOnlySecretsDirectory", func(t *testing.T) {
		testRegisterWithReadOnlySecretsDirectory(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithReadOnlyPublicKeysDirectory", func(t *testing.T) {
		testRegisterWithReadOnlyPublicKeysDirectory(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithMultipleKeys", func(t *testing.T) {
		testRegisterWithMultipleKeys(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterInDirectoryWithSpaces", func(t *testing.T) {
		testRegisterInDirectoryWithSpaces(t, originalWd, originalUserSettings)
	})
}

func testRegisterWithReadOnlySecretsDirectory(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping read-only directory test on Windows")
	}

	tempDir, err := os.MkdirTemp("", "kanuka-test-register-readonly-secrets-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	pemKey := generatePEMKeyFilesystem(t, &privateKey.PublicKey)
	targetUserEmail := "readonlyuser@example.com"
	targetUserUUID := "readonly-user-uuid-1234"

	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	if projectConfig.Users == nil {
		projectConfig.Users = make(map[string]string)
	}
	projectConfig.Users[targetUserUUID] = targetUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	secretsDir := filepath.Join(tempDir, ".kanuka", "secrets")
	if err := os.Chmod(secretsDir, 0444); err != nil {
		t.Fatalf("Failed to make secrets directory read-only: %v", err)
	}
	defer func() {
		_ = os.Chmod(secretsDir, 0755)
	}()

	cmd.ResetGlobalState()

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", pemKey, "--user", targetUserEmail})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✗") {
		t.Errorf("Expected error symbol not found in output: %s", output)
	}

	if !strings.Contains(output, "Error:") || !strings.Contains(output, "permission denied") {
		t.Errorf("Expected permission denied error not found in output: %s", output)
	}
}

func testRegisterWithReadOnlyPublicKeysDirectory(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping read-only directory test on Windows")
	}

	tempDir, err := os.MkdirTemp("", "kanuka-test-register-readonly-pubkeys-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	pemKey := generatePEMKeyFilesystem(t, &privateKey.PublicKey)
	targetUserEmail := "readonlypubuser@example.com"
	targetUserUUID := "readonly-pubuser-uuid-1234"

	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	if projectConfig.Users == nil {
		projectConfig.Users = make(map[string]string)
	}
	projectConfig.Users[targetUserUUID] = targetUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	publicKeysDir := filepath.Join(tempDir, ".kanuka", "public_keys")
	if err := os.Chmod(publicKeysDir, 0444); err != nil {
		t.Fatalf("Failed to make public_keys directory read-only: %v", err)
	}
	defer func() {
		_ = os.Chmod(publicKeysDir, 0755)
	}()

	cmd.ResetGlobalState()

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", pemKey, "--user", targetUserEmail})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✗") {
		t.Errorf("Expected error symbol not found in output: %s", output)
	}

	if !strings.Contains(output, "Error:") || !strings.Contains(output, "permission denied") {
		t.Errorf("Expected permission denied error not found in output: %s", output)
	}
}

func testRegisterWithMultipleKeys(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-multiple-keys-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	users := []struct {
		uuid  string
		email string
	}{
		{"user1-uuid-1234", "user1@example.com"},
		{"user2-uuid-1234", "user2@example.com"},
		{"user3-uuid-1234", "user3@example.com"},
	}

	for _, user := range users {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("Failed to generate RSA key: %v", err)
		}

		pemKey := generatePEMKeyFilesystem(t, &privateKey.PublicKey)

		projectConfig, err := configs.LoadProjectConfig()
		if err != nil {
			t.Fatalf("Failed to load project config: %v", err)
		}
		if projectConfig.Users == nil {
			projectConfig.Users = make(map[string]string)
		}
		projectConfig.Users[user.uuid] = user.email
		if err := configs.SaveProjectConfig(projectConfig); err != nil {
			t.Fatalf("Failed to save project config: %v", err)
		}

		cmd.ResetGlobalState()

		output, err := shared.CaptureOutput(func() error {
			cmd := shared.CreateTestCLI("register", nil, nil, true, false)
			cmd.SetArgs([]string{"secrets", "register", "--pubkey", pemKey, "--user", user.email})
			return cmd.Execute()
		})
		if err != nil {
			t.Errorf("Command failed unexpectedly for %s: %v", user.email, err)
		}

		if !strings.Contains(output, "✓") {
			t.Errorf("Expected success symbol not found in output for %s: %s", user.email, output)
		}

		if !strings.Contains(output, user.email) {
			t.Errorf("Expected email %s not found in output: %s", user.email, output)
		}

		kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", user.uuid+".kanuka")
		if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
			t.Errorf("Kanuka key file was not created for %s at %s", user.email, kanukaKeyPath)
		}
	}
}

func testRegisterInDirectoryWithSpaces(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka test register spaces *")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka user *")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	pemKey := generatePEMKeyFilesystem(t, &privateKey.PublicKey)
	targetUserEmail := "spaceuser@example.com"
	targetUserUUID := "space-user-uuid-1234"

	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	if projectConfig.Users == nil {
		projectConfig.Users = make(map[string]string)
	}
	projectConfig.Users[targetUserUUID] = targetUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	cmd.ResetGlobalState()

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", pemKey, "--user", targetUserEmail})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success symbol not found in output: %s", output)
	}

	kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", targetUserUUID+".kanuka")
	if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
		t.Errorf("Kanuka key file was not created at %s", kanukaKeyPath)
	}
}

func generatePEMKeyFilesystem(t *testing.T, publicKey *rsa.PublicKey) string {
	pubASN1, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		t.Fatalf("Failed to marshal public key: %v", err)
	}

	pubPem := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubASN1,
	}

	return string(pem.EncodeToMemory(pubPem))
}
