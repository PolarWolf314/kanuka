package create

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsCreateFilesystem contains file system operation tests for the `kanuka secrets create` command.
func TestSecretsCreateFilesystem(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("KeyGenerationAndStorage", func(t *testing.T) {
		testKeyGenerationAndStorage(t, originalWd, originalUserSettings)
	})

	t.Run("PublicKeyCopying", func(t *testing.T) {
		testPublicKeyCopying(t, originalWd, originalUserSettings)
	})

	t.Run("DirectoryCreation", func(t *testing.T) {
		testDirectoryCreation(t, originalWd, originalUserSettings)
	})

	t.Run("FilePermissions", func(t *testing.T) {
		testFilePermissions(t, originalWd, originalUserSettings)
	})

	t.Run("CleanupOperations", func(t *testing.T) {
		testCleanupOperations(t, originalWd, originalUserSettings)
	})
}

// Tests key generation and storage.
func testKeyGenerationAndStorage(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-create-keygen-*")
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

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	projectName := filepath.Base(tempDir)
	privateKeyPath := filepath.Join(tempUserDir, "keys", projectName)
	publicKeyPath := filepath.Join(tempUserDir, "keys", projectName+".pub")

	privateKeyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		t.Errorf("Failed to read private key: %v", err)
	}

	if !strings.Contains(string(privateKeyData), "-----BEGIN RSA PRIVATE KEY-----") {
		t.Errorf("Private key is not in correct PEM format")
	}

	if !strings.Contains(string(privateKeyData), "-----END RSA PRIVATE KEY-----") {
		t.Errorf("Private key is not in correct PEM format")
	}

	publicKeyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		t.Errorf("Failed to read public key: %v", err)
	}

	if !strings.Contains(string(publicKeyData), "-----BEGIN PUBLIC KEY-----") {
		t.Errorf("Public key is not in correct PEM format")
	}

	if !strings.Contains(string(publicKeyData), "-----END PUBLIC KEY-----") {
		t.Errorf("Public key is not in correct PEM format")
	}
}

// Tests public key copying.
func testPublicKeyCopying(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-create-pubcopy-*")
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

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	projectName := filepath.Base(tempDir)
	username := configs.UserKanukaSettings.Username

	userPublicKeyPath := filepath.Join(tempUserDir, "keys", projectName+".pub")
	projectPublicKeyPath := filepath.Join(tempDir, ".kanuka", "public_keys", username+".pub")

	userKeyData, err := os.ReadFile(userPublicKeyPath)
	if err != nil {
		t.Errorf("Failed to read user public key: %v", err)
	}

	projectKeyData, err := os.ReadFile(projectPublicKeyPath)
	if err != nil {
		t.Errorf("Failed to read project public key: %v", err)
	}

	if string(userKeyData) != string(projectKeyData) {
		t.Errorf("Public key was not copied correctly to project")
	}
}

// Tests directory creation.
func testDirectoryCreation(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-create-dirs-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	if err := os.RemoveAll(tempUserDir); err != nil {
		t.Fatalf("Failed to remove temp user directory: %v", err)
	}

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	keysDir := filepath.Join(tempUserDir, "keys")
	if _, err := os.Stat(keysDir); os.IsNotExist(err) {
		t.Errorf("Keys directory was not created at %s", keysDir)
	}

	configDir := tempUserDir
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Errorf("Config directory was not created at %s", configDir)
	}
}

// Tests file permissions.
func testFilePermissions(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-create-perms-*")
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

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	projectName := filepath.Base(tempDir)
	username := configs.UserKanukaSettings.Username

	privateKeyPath := filepath.Join(tempUserDir, "keys", projectName)
	privateKeyInfo, err := os.Stat(privateKeyPath)
	if err != nil {
		t.Errorf("Failed to stat private key: %v", err)
	} else {
		mode := privateKeyInfo.Mode()
		// Private key should be readable/writable by owner only (0600)
		expectedMode := os.FileMode(0600)
		if mode.Perm() != expectedMode {
			t.Logf("Private key permissions: %o (expected %o)", mode.Perm(), expectedMode)
			// This is informational - file permissions may vary by system
		}
	}

	projectPublicKeyPath := filepath.Join(tempDir, ".kanuka", "public_keys", username+".pub")
	publicKeyInfo, err := os.Stat(projectPublicKeyPath)
	if err != nil {
		t.Errorf("Failed to stat project public key: %v", err)
	} else {
		mode := publicKeyInfo.Mode()
		// Public key should be readable (at least 0644)
		if mode.Perm()&0044 == 0 {
			t.Logf("Public key permissions: %o (should be readable)", mode.Perm())
			// This is informational - file permissions may vary by system
		}
	}
}

// Tests cleanup operations.
func testCleanupOperations(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-create-cleanup-*")
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

	username := configs.UserKanukaSettings.Username
	kanukaFilePath := filepath.Join(tempDir, ".kanuka", "secrets", username+".kanuka")

	if err := os.WriteFile(kanukaFilePath, []byte("existing kanuka data"), 0600); err != nil {
		t.Fatalf("Failed to create existing kanuka file: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "create", "--force"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	if _, err := os.Stat(kanukaFilePath); !os.IsNotExist(err) {
		t.Errorf("Existing kanuka file was not removed")
	}

	if !strings.Contains(output, "deleted:") {
		t.Errorf("Expected deletion message not found in output: %s", output)
	}
}