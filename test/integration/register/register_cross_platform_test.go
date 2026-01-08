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
	// Save original state
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

	t.Run("RegisterWithUnicodeUsernames", func(t *testing.T) {
		testRegisterWithUnicodeUsernames(t, originalWd, originalUserSettings)
	})
}

// testRegisterWithWindowsLineSeparators tests handling CRLF line endings in public keys.
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

	// Generate a test RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Generate PEM key with Windows line endings (CRLF)
	pemKey := generatePEMKeyCrossPlatform(t, &privateKey.PublicKey)
	windowsPemKey := strings.ReplaceAll(pemKey, "\n", "\r\n")

	// Write the key to the project's public_keys directory
	targetUser := "crlfuser"
	projectPubKeysDir := filepath.Join(tempDir, ".kanuka", "public_keys")
	keyPath := filepath.Join(projectPubKeysDir, targetUser+".pub")
	if err := os.WriteFile(keyPath, []byte(windowsPemKey), 0644); err != nil { //nolint:gosec // G306: public key file in test
		t.Fatalf("Failed to write key file: %v", err)
	}

	// Reset register command state
	cmd.ResetGlobalState()

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--file", keyPath})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success symbol not found in output: %s", output)
	}

	if !strings.Contains(output, targetUser) {
		t.Errorf("Expected target user name not found in output: %s", output)
	}

	// Verify the kanuka key was created
	kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", targetUser+".kanuka")
	if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
		t.Errorf("Kanuka key file was not created at %s", kanukaKeyPath)
	}

	// Verify the key can be decrypted
	encryptedSymKey, err := os.ReadFile(kanukaKeyPath)
	if err != nil {
		t.Fatalf("Failed to read kanuka key: %v", err)
	}

	_, err = secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		t.Errorf("Failed to decrypt symmetric key with private key: %v", err)
	}
}

// testRegisterWithUnixLineSeparators tests handling LF line endings in public keys.
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

	// Generate a test RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Generate PEM key with Unix line endings (LF) - this is the default
	pemKey := generatePEMKeyCrossPlatform(t, &privateKey.PublicKey)

	// Write the key to the project's public_keys directory
	targetUser := "lfuser"
	projectPubKeysDir := filepath.Join(tempDir, ".kanuka", "public_keys")
	keyPath := filepath.Join(projectPubKeysDir, targetUser+".pub")
	if err := os.WriteFile(keyPath, []byte(pemKey), 0644); err != nil { //nolint:gosec // G306: public key file in test
		t.Fatalf("Failed to write key file: %v", err)
	}

	// Reset register command state
	cmd.ResetGlobalState()

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--file", keyPath})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success symbol not found in output: %s", output)
	}

	if !strings.Contains(output, targetUser) {
		t.Errorf("Expected target user name not found in output: %s", output)
	}

	// Verify the kanuka key was created
	kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", targetUser+".kanuka")
	if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
		t.Errorf("Kanuka key file was not created at %s", kanukaKeyPath)
	}

	// Verify the key can be decrypted
	encryptedSymKey, err := os.ReadFile(kanukaKeyPath)
	if err != nil {
		t.Fatalf("Failed to read kanuka key: %v", err)
	}

	_, err = secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		t.Errorf("Failed to decrypt symmetric key with private key: %v", err)
	}
}

// testRegisterWithMixedLineSeparators tests handling mixed line endings.
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

	// Generate a test RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Generate PEM key with mixed line endings
	pemKey := generatePEMKeyCrossPlatform(t, &privateKey.PublicKey)
	lines := strings.Split(pemKey, "\n")

	// Mix CRLF and LF line endings
	var mixedLines []string
	for i, line := range lines {
		if i%2 == 0 {
			mixedLines = append(mixedLines, line+"\r") // CRLF
		} else {
			mixedLines = append(mixedLines, line) // LF
		}
	}
	mixedPemKey := strings.Join(mixedLines, "\n")

	// Write the key to the project's public_keys directory
	targetUser := "mixeduser"
	projectPubKeysDir := filepath.Join(tempDir, ".kanuka", "public_keys")
	keyPath := filepath.Join(projectPubKeysDir, targetUser+".pub")
	if err := os.WriteFile(keyPath, []byte(mixedPemKey), 0644); err != nil { //nolint:gosec // G306: public key file in test
		t.Fatalf("Failed to write key file: %v", err)
	}

	// Reset register command state
	cmd.ResetGlobalState()

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--file", keyPath})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success symbol not found in output: %s", output)
	}

	if !strings.Contains(output, targetUser) {
		t.Errorf("Expected target user name not found in output: %s", output)
	}

	// Verify the kanuka key was created
	kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", targetUser+".kanuka")
	if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
		t.Errorf("Kanuka key file was not created at %s", kanukaKeyPath)
	}

	// Verify the key can be decrypted
	encryptedSymKey, err := os.ReadFile(kanukaKeyPath)
	if err != nil {
		t.Fatalf("Failed to read kanuka key: %v", err)
	}

	_, err = secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		t.Errorf("Failed to decrypt symmetric key with private key: %v", err)
	}
}

// testRegisterWithDifferentFilePermissions tests various file permission scenarios.
func testRegisterWithDifferentFilePermissions(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Skip on Windows as file permissions work differently
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

	// Test different file permissions
	permissions := []struct {
		name string
		perm os.FileMode
	}{
		{"ReadOnly", 0444},
		{"ReadWrite", 0644},
		{"ReadWriteExecute", 0755},
	}

	for _, perm := range permissions {
		t.Run(perm.name, func(t *testing.T) {
			// Generate a test RSA key pair
			privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				t.Fatalf("Failed to generate RSA key: %v", err)
			}

			// Create a key file with specific permissions
			keyPath := filepath.Join(tempUserDir, perm.name+"_user.pub")
			pemKey := generatePEMKeyCrossPlatform(t, &privateKey.PublicKey)
			if err := os.WriteFile(keyPath, []byte(pemKey), perm.perm); err != nil {
				t.Fatalf("Failed to write key file: %v", err)
			}

			// Reset register command state
			cmd.ResetGlobalState()

			output, err := shared.CaptureOutput(func() error {
				cmd := shared.CreateTestCLI("register", nil, nil, true, false)
				cmd.SetArgs([]string{"secrets", "register", "--file", keyPath})
				return cmd.Execute()
			})
			if err != nil {
				t.Errorf("Command failed unexpectedly for %s: %v", perm.name, err)
			}

			if !strings.Contains(output, "✓") {
				t.Errorf("Expected success symbol not found for %s in output: %s", perm.name, output)
			}

			// Verify the kanuka key was created
			targetUser := perm.name + "_user"
			kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", targetUser+".kanuka")
			if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
				t.Errorf("Kanuka key file was not created for %s at %s", perm.name, kanukaKeyPath)
			}
		})
	}
}

// testRegisterWithUnicodeUsernames tests handling Unicode characters in usernames via file names.
func testRegisterWithUnicodeUsernames(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-unicode-*")
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

	// Test various Unicode usernames via file names (--file approach)
	unicodeUsers := []struct {
		name     string
		username string
	}{
		{"Japanese", "田中太郎"},
		{"Emoji", "user😀test"},
		{"Accented", "josé_maría"},
		{"Cyrillic", "пользователь"},
		{"Mixed", "user_测试_123"},
	}

	projectPubKeysDir := filepath.Join(tempDir, ".kanuka", "public_keys")

	for _, user := range unicodeUsers {
		t.Run(user.name, func(t *testing.T) {
			// Generate a test RSA key pair
			privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				t.Fatalf("Failed to generate RSA key: %v", err)
			}

			pemKey := generatePEMKeyCrossPlatform(t, &privateKey.PublicKey)

			// Write the key to the project's public_keys directory
			keyPath := filepath.Join(projectPubKeysDir, user.username+".pub")
			if err := os.WriteFile(keyPath, []byte(pemKey), 0644); err != nil { //nolint:gosec // G306: public key file in test
				t.Fatalf("Failed to write key file for %s: %v", user.name, err)
			}

			// Reset register command state
			cmd.ResetGlobalState()

			output, err := shared.CaptureOutput(func() error {
				cmd := shared.CreateTestCLI("register", nil, nil, true, false)
				cmd.SetArgs([]string{"secrets", "register", "--file", keyPath})
				return cmd.Execute()
			})
			if err != nil {
				t.Errorf("Command failed unexpectedly for %s: %v", user.name, err)
			}

			if !strings.Contains(output, "✓") {
				t.Errorf("Expected success symbol not found for %s in output: %s", user.name, output)
			}

			if !strings.Contains(output, user.username) {
				t.Errorf("Expected username %s not found in output: %s", user.username, output)
			}

			// Verify the kanuka key was created
			kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", user.username+".kanuka")
			if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
				t.Errorf("Kanuka key file was not created for %s at %s", user.name, kanukaKeyPath)
			}

			// Verify the key can be decrypted
			encryptedSymKey, err := os.ReadFile(kanukaKeyPath)
			if err != nil {
				t.Errorf("Failed to read kanuka key for %s: %v", user.name, err)
				return
			}

			_, err = secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
			if err != nil {
				t.Errorf("Failed to decrypt symmetric key for %s: %v", user.name, err)
			}
		})
	}
}

// Helper function for generating PEM keys (cross-platform tests).
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
