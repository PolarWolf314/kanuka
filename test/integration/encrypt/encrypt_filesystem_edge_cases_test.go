package encrypt_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsEncryptFilesystemEdgeCases contains filesystem edge case tests for the `kanuka secrets encrypt` command.
func TestSecretsEncryptFilesystemEdgeCases(t *testing.T) {
	// Save original working directory and settings
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings

	t.Run("EncryptWithEmptyEnvFile", func(t *testing.T) {
		testEncryptWithEmptyEnvFile(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptWithReadOnlyEnvFile", func(t *testing.T) {
		testEncryptWithReadOnlyEnvFile(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptWithEnvFileAsDirectory", func(t *testing.T) {
		testEncryptWithEnvFileAsDirectory(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptWithEnvFileAsSymlink", func(t *testing.T) {
		testEncryptWithEnvFileAsSymlink(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptWithBrokenEnvSymlink", func(t *testing.T) {
		testEncryptWithBrokenEnvSymlink(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptWithVeryLargeEnvFile", func(t *testing.T) {
		testEncryptWithVeryLargeEnvFile(t, originalWd, originalUserSettings)
	})
}

// Tests encrypting an empty .env file.
func testEncryptWithEmptyEnvFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-empty-env-*")
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

	// Create an empty .env file
	envFile := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(envFile, []byte(""), 0600); err != nil {
		t.Fatalf("Failed to create empty .env file: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed with empty .env file: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "encrypted successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}

	userUUID := shared.GetUserUUID(t)
	encryptedFile := filepath.Join(tempDir, ".kanuka", "secrets", userUUID+".kanuka")
	if _, err := os.Stat(encryptedFile); os.IsNotExist(err) {
		t.Errorf("Encrypted file was not created at %s", encryptedFile)
	}
}

// Tests encrypting a read-only .env file.
func testEncryptWithReadOnlyEnvFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-readonly-env-*")
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

	// Create a .env file and make it read-only
	envFile := filepath.Join(tempDir, ".env")
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\nAPI_KEY=secret123\n"
	if err := os.WriteFile(envFile, []byte(envContent), 0600); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Make the file read-only
	if err := os.Chmod(envFile, 0444); err != nil {
		t.Fatalf("Failed to make .env file read-only: %v", err)
	}
	// Restore permissions for cleanup
	defer func() {
		if err := os.Chmod(envFile, 0644); err != nil {
			t.Logf("Failed to restore permissions on %s: %v", envFile, err)
		}
	}()

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed with read-only .env file: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "encrypted successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}
}

// Tests encrypting when .env exists as a directory instead of file.
func testEncryptWithEnvFileAsDirectory(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-env-as-dir-*")
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

	// Create a directory named .env
	envDir := filepath.Join(tempDir, ".env")
	if err := os.Mkdir(envDir, 0755); err != nil {
		t.Fatalf("Failed to create .env directory: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "No environment files found") {
		t.Errorf("Expected message about no environment files found, got: %s", output)
	}
}

// Tests encrypting when .env is a symlink to another file.
func testEncryptWithEnvFileAsSymlink(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-env-symlink-*")
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

	// Create a target file and symlink .env to it
	targetFile := filepath.Join(tempDir, "actual-env-file")
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\nAPI_KEY=secret123\n"
	if err := os.WriteFile(targetFile, []byte(envContent), 0600); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	envSymlink := filepath.Join(tempDir, ".env")
	if err := os.Symlink(targetFile, envSymlink); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "No environment files found") {
		t.Errorf("Expected message about no environment files found, got: %s", output)
	}
}

// Tests encrypting when .env is a broken symlink.
func testEncryptWithBrokenEnvSymlink(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-broken-symlink-*")
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

	// Create symlink to non-existent file
	envSymlink := filepath.Join(tempDir, ".env")
	nonExistentTarget := filepath.Join(tempDir, "non-existent-file")
	if err := os.Symlink(nonExistentTarget, envSymlink); err != nil {
		t.Fatalf("Failed to create broken symlink: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "No environment files found") {
		t.Errorf("Expected message about no environment files found, got: %s", output)
	}
}

// Tests encrypting a very large .env file (MB+ size).
func testEncryptWithVeryLargeEnvFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-large-env-*")
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

	// Create a very large .env file (1MB+)
	envFile := filepath.Join(tempDir, ".env")
	var envContent strings.Builder

	// Generate ~1MB of environment variables
	for i := 0; i < 10000; i++ {
		envContent.WriteString(fmt.Sprintf("LARGE_VAR_%d=%s\n", i, strings.Repeat("x", 100)))
	}

	if err := os.WriteFile(envFile, []byte(envContent.String()), 0600); err != nil {
		t.Fatalf("Failed to create large .env file: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed with large .env file: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "encrypted successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}

	userUUID := shared.GetUserUUID(t)
	encryptedFile := filepath.Join(tempDir, ".kanuka", "secrets", userUUID+".kanuka")
	if stat, err := os.Stat(encryptedFile); os.IsNotExist(err) {
		t.Errorf("Encrypted file was not created at %s", encryptedFile)
	} else if stat.Size() == 0 {
		t.Errorf("Encrypted file is empty, expected it to contain encrypted data")
	}
}
