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
	"github.com/PolarWolf314/kanuka/internal/secrets"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

func TestSecretsRegisterCrossPlatform(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings

	t.Run("RegisterWithWindowsLineSeparators", func(t *testing.T) {
		testRegisterWithWindowsLineSeparators(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithUnixLineSeparators", func(t *testing.T) {
		testRegisterWithUnixLineSeparators(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithMixedLineSeparators", func(t *testing.T) {
		testRegisterWithMixedLineSeparators(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithDifferentFilePermissions", func(t *testing.T) {
		testRegisterWithDifferentFilePermissions(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithValidEmailUsernames", func(t *testing.T) {
		testRegisterWithValidEmailUsernames(t, originalWd, originalUserSettings)
	})
}

func testRegisterWithWindowsLineSeparators(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-crlf-*")
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

	pemKey := generatePEMKeyCrossPlatform(t, &privateKey.PublicKey)
	windowsPemKey := strings.ReplaceAll(pemKey, "\n", "\r\n")

	targetUserUUID := "550e8400-e29b-41d4-a716-4466554401"
	targetUserEmail := "user1@example.com"

	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectConfig.Users[targetUserUUID] = targetUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	cmd.ResetGlobalState()

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", windowsPemKey, "--user", targetUserEmail})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success symbol not found in output: %s", output)
	}

	if !strings.Contains(output, targetUserEmail) {
		t.Errorf("Expected target user email not found in output: %s", output)
	}

	kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", targetUserUUID+".kanuka")
	if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
		t.Errorf("Kanuka key file was not created at %s", kanukaKeyPath)
	}

	encryptedSymKey, err := os.ReadFile(kanukaKeyPath)
	if err != nil {
		t.Fatalf("Failed to read kanuka key: %v", err)
	}

	_, err = secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		t.Errorf("Failed to decrypt symmetric key with private key: %v", err)
	}
}

func testRegisterWithUnixLineSeparators(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-lf-*")
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

	pemKey := generatePEMKeyCrossPlatform(t, &privateKey.PublicKey)

	targetUserUUID := "550e8400-e29b-41d4-a716-4466554402"
	targetUserEmail := "user2@example.com"

	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
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

	if !strings.Contains(output, targetUserEmail) {
		t.Errorf("Expected target user email not found in output: %s", output)
	}

	kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", targetUserUUID+".kanuka")
	if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
		t.Errorf("Kanuka key file was not created at %s", kanukaKeyPath)
	}

	encryptedSymKey, err := os.ReadFile(kanukaKeyPath)
	if err != nil {
		t.Fatalf("Failed to read kanuka key: %v", err)
	}

	_, err = secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		t.Errorf("Failed to decrypt symmetric key with private key: %v", err)
	}
}

func testRegisterWithMixedLineSeparators(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-mixed-*")
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

	pemKey := generatePEMKeyCrossPlatform(t, &privateKey.PublicKey)

	targetUserUUID := "550e8400-e29b-41d4-a716-4466554403"
	targetUserEmail := "user3@example.com"

	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectConfig.Users[targetUserUUID] = targetUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	var mixedLines []string
	for _, line := range strings.Split(pemKey, "\n") {
		if len(line) == 2 {
			mixedLines = append(mixedLines, line+"\r\n")
		} else {
			mixedLines = append(mixedLines, line)
		}
	}
	mixedPemKey := strings.Join(mixedLines, "\n")

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", mixedPemKey, "--user", targetUserEmail})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success symbol not found in output: %s", output)
	}

	if !strings.Contains(output, targetUserEmail) {
		t.Errorf("Expected email not found in output: %s", output)
	}

	kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", targetUserUUID+".kanuka")
	if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
		t.Errorf("Kanuka key file was not created at %s", kanukaKeyPath)
	}

	encryptedSymKey, err := os.ReadFile(kanukaKeyPath)
	if err != nil {
		t.Fatalf("Failed to read kanuka key: %v", err)
	}

	_, err = secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		t.Errorf("Failed to decrypt symmetric key with private key: %v", err)
	}
}

func testRegisterWithDifferentFilePermissions(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping file permissions test on Windows")
	}

	tempDir, err := os.MkdirTemp("", "kanuka-test-register-perms-*")
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

	permissions := []struct {
		name string
		perm os.FileMode
		uuid string
	}{
		{"ReadOnly", 0444, "550e8400-e29b-41d4-a716-4466554401"},
		{"ReadWrite", 0644, "550e8400-e29b-41d4-a716-4466554402"},
		{"ReadWriteExecute", 0755, "550e8400-e29b-41d4-a716-4466554403"},
	}

	for _, perm := range permissions {
		t.Run(perm.name, func(t *testing.T) {
			privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				t.Fatalf("Failed to generate RSA key: %v", err)
			}

			pemKey := generatePEMKeyCrossPlatform(t, &privateKey.PublicKey)
			cmd.ResetGlobalState()

			targetUserEmail := perm.uuid + "@example.com"

			projectConfig, err := configs.LoadProjectConfig()
			if err != nil {
				t.Fatalf("Failed to load project config: %v", err)
			}
			projectConfig.Users[perm.uuid] = targetUserEmail
			if err := configs.SaveProjectConfig(projectConfig); err != nil {
				t.Fatalf("Failed to save project config: %v", err)
			}

			email := targetUserEmail
			output, err := shared.CaptureOutput(func() error {
				cmd := shared.CreateTestCLI("register", nil, nil, true, false)
				cmd.SetArgs([]string{"secrets", "register", "--pubkey", pemKey, "--user", email})
				return cmd.Execute()
			})
			if err != nil {
				t.Errorf("Command failed unexpectedly: %v", err)
			}

			if !strings.Contains(output, "✓") {
				t.Errorf("Expected success symbol not found in output for %s: %s", perm.name, output)
			}

			if !strings.Contains(output, email) {
				t.Errorf("Expected email %s not found in output for %s: %s", email, perm.name, output)
			}

			kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", perm.uuid+".kanuka")
			if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
				t.Errorf("Kanuka key file was not created for %s at %s", perm.name, kanukaKeyPath)
			}

			encryptedSymKey, err := os.ReadFile(kanukaKeyPath)
			if err != nil {
				t.Fatalf("Failed to read kanuka key: %v", err)
			}

			_, err = secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
			if err != nil {
				t.Errorf("Failed to decrypt symmetric key with private key: %v", err)
			}
		})
	}
}

func testRegisterWithValidEmailUsernames(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-email-*")
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

	projectPubKeysDir := filepath.Join(tempDir, ".kanuka", "public_keys")

	emailUsers := []struct {
		name     string
		username string
		uuid     string
	}{
		{"SimpleEmail", "simple", "550e8400-e29b-41d4-a716-4466554401"},
		{"EmailWithDots", "user.test", "550e8400-e29b-41d4-a716-4466554402"},
		{"EmailWithPlus", "user+test", "550e8400-e29b-41d4-a716-4466554403"},
		{"EmailWithNumbers", "user123", "550e8400-e29b-41d4-a716-4466554404"},
		{"EmailMixed", "user_test_123", "550e8400-e29b-41d4-a716-4466554405"},
	}

	for _, user := range emailUsers {
		t.Run(user.name, func(t *testing.T) {
			privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				t.Fatalf("Failed to generate RSA key: %v", err)
			}

			pemKey := generatePEMKeyCrossPlatform(t, &privateKey.PublicKey)

			projectConfig, err := configs.LoadProjectConfig()
			if err != nil {
				t.Fatalf("Failed to load project config: %v", err)
			}
			projectConfig.Users[user.uuid] = user.username + "@example.com"
			if err := configs.SaveProjectConfig(projectConfig); err != nil {
				t.Fatalf("Failed to save project config: %v", err)
			}

			keyPath := filepath.Join(projectPubKeysDir, user.uuid+".pub")
			if err := os.WriteFile(keyPath, []byte(pemKey), 0600); err != nil {
				t.Fatalf("Failed to write key file for %s: %v", user.name, err)
			}

			cmd.ResetGlobalState()

			email := user.username + "@example.com"
			output, err := shared.CaptureOutput(func() error {
				cmd := shared.CreateTestCLI("register", nil, nil, true, false)
				cmd.SetArgs([]string{"secrets", "register", "--pubkey", pemKey, "--user", email})
				return cmd.Execute()
			})
			if err != nil {
				t.Errorf("Command failed unexpectedly for %s: %v", user.name, err)
			}

			if !strings.Contains(output, "✓") {
				t.Errorf("Expected success symbol not found in output for %s: %s", user.name, output)
			}

			if !strings.Contains(output, email) {
				t.Errorf("Expected email %s not found in output for %s: %s", email, user.name, output)
			}

			kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", user.uuid+".kanuka")
			if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
				t.Errorf("Kanuka key file was not created for %s at %s", user.name, kanukaKeyPath)
			}

			os.Remove(kanukaKeyPath)
		})
	}
}

func generatePEMKeyCrossPlatform(t *testing.T, publicKey *rsa.PublicKey) string {
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
