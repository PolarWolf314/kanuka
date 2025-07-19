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
	// Save original state
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

	t.Run("RegisterWithSymlinkedPublicKey", func(t *testing.T) {
		testRegisterWithSymlinkedPublicKey(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithRelativeFilePaths", func(t *testing.T) {
		testRegisterWithRelativeFilePaths(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithAbsoluteFilePaths", func(t *testing.T) {
		testRegisterWithAbsoluteFilePaths(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterInDirectoryWithSpaces", func(t *testing.T) {
		testRegisterInDirectoryWithSpaces(t, originalWd, originalUserSettings)
	})
}

// testRegisterWithReadOnlySecretsDirectory tests error when secrets directory is read-only.
func testRegisterWithReadOnlySecretsDirectory(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Skip on Windows as file permissions work differently
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

	// Make secrets directory read-only
	secretsDir := filepath.Join(tempDir, ".kanuka", "secrets")
	if err := os.Chmod(secretsDir, 0444); err != nil {
		t.Fatalf("Failed to make secrets directory read-only: %v", err)
	}
	defer func() {
		_ = os.Chmod(secretsDir, 0755) // Restore permissions for cleanup
	}()

	// Generate a test RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	pemKey := generatePEMKeyFilesystem(t, &privateKey.PublicKey)
	targetUser := "readonlyuser"

	// Reset register command state
	cmd.ResetGlobalState()

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", pemKey, "--user", targetUser})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	// Should fail due to read-only directory
	if !strings.Contains(output, "✗") {
		t.Errorf("Expected error symbol not found in output: %s", output)
	}

	if !strings.Contains(output, "Error:") || !strings.Contains(output, "permission denied") {
		t.Errorf("Expected permission denied error not found in output: %s", output)
	}
}

// testRegisterWithReadOnlyPublicKeysDirectory tests error when public_keys directory is read-only.
func testRegisterWithReadOnlyPublicKeysDirectory(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Skip on Windows as file permissions work differently
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

	// Make public_keys directory read-only
	publicKeysDir := filepath.Join(tempDir, ".kanuka", "public_keys")
	if err := os.Chmod(publicKeysDir, 0444); err != nil {
		t.Fatalf("Failed to make public_keys directory read-only: %v", err)
	}
	defer func() {
		_ = os.Chmod(publicKeysDir, 0755) // Restore permissions for cleanup
	}()

	// Generate a test RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	pemKey := generatePEMKeyFilesystem(t, &privateKey.PublicKey)
	targetUser := "readonlypubuser"

	// Reset register command state
	cmd.ResetGlobalState()

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", pemKey, "--user", targetUser})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	// Should fail due to read-only directory
	if !strings.Contains(output, "✗") {
		t.Errorf("Expected error symbol not found in output: %s", output)
	}

	if !strings.Contains(output, "Error:") || !strings.Contains(output, "permission denied") {
		t.Errorf("Expected permission denied error not found in output: %s", output)
	}
}

// testRegisterWithSymlinkedPublicKey tests handling symlinked public key files.
func testRegisterWithSymlinkedPublicKey(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Skip on Windows as symlinks require special permissions
	if runtime.GOOS == "windows" {
		t.Skip("Skipping symlink test on Windows")
	}

	tempDir, err := os.MkdirTemp("", "kanuka-test-register-symlink-*")
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

	// Create a real public key file
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	realKeyPath := filepath.Join(tempUserDir, "real_key.pub")
	pemKey := generatePEMKeyFilesystem(t, &privateKey.PublicKey)
	if err := os.WriteFile(realKeyPath, []byte(pemKey), 0600); err != nil {
		t.Fatalf("Failed to write real key file: %v", err)
	}

	// Create a symlink to the real key
	symlinkPath := filepath.Join(tempUserDir, "symlink_key.pub")
	if err := os.Symlink(realKeyPath, symlinkPath); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Reset register command state
	cmd.ResetGlobalState()

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--file", symlinkPath})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success symbol not found in output: %s", output)
	}

	// Verify the kanuka key was created for the symlink target
	targetUser := "symlink_key"
	kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", targetUser+".kanuka")
	if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
		t.Errorf("Kanuka key file was not created at %s", kanukaKeyPath)
	}
}

// testRegisterWithRelativeFilePaths tests handling relative paths in --file flag.
func testRegisterWithRelativeFilePaths(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-relative-*")
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

	// Create a subdirectory and key file
	subDir := filepath.Join(tempDir, "keys")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	keyPath := filepath.Join(subDir, "relative_user.pub")
	pemKey := generatePEMKeyFilesystem(t, &privateKey.PublicKey)
	if err := os.WriteFile(keyPath, []byte(pemKey), 0600); err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	// Use relative path
	relativePath := "./keys/relative_user.pub"

	// Reset register command state
	cmd.ResetGlobalState()

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--file", relativePath})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success symbol not found in output: %s", output)
	}

	// Verify the kanuka key was created
	targetUser := "relative_user"
	kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", targetUser+".kanuka")
	if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
		t.Errorf("Kanuka key file was not created at %s", kanukaKeyPath)
	}
}

// testRegisterWithAbsoluteFilePaths tests handling absolute paths in --file flag.
func testRegisterWithAbsoluteFilePaths(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-absolute-*")
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

	// Create a key file with absolute path
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	keyPath := filepath.Join(tempUserDir, "absolute_user.pub")
	pemKey := generatePEMKeyFilesystem(t, &privateKey.PublicKey)
	if err := os.WriteFile(keyPath, []byte(pemKey), 0600); err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	// Use absolute path
	absolutePath, err := filepath.Abs(keyPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Reset register command state
	cmd.ResetGlobalState()

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--file", absolutePath})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success symbol not found in output: %s", output)
	}

	// Verify the kanuka key was created
	targetUser := "absolute_user"
	kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", targetUser+".kanuka")
	if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
		t.Errorf("Kanuka key file was not created at %s", kanukaKeyPath)
	}
}

// testRegisterInDirectoryWithSpaces tests handling project paths containing spaces.
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

	// Generate a test RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	pemKey := generatePEMKeyFilesystem(t, &privateKey.PublicKey)
	targetUser := "spaceuser"

	// Reset register command state
	cmd.ResetGlobalState()

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", pemKey, "--user", targetUser})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success symbol not found in output: %s", output)
	}

	// Verify the kanuka key was created
	kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", targetUser+".kanuka")
	if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
		t.Errorf("Kanuka key file was not created at %s", kanukaKeyPath)
	}
}

// Helper function for generating PEM keys (filesystem edge cases).
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
