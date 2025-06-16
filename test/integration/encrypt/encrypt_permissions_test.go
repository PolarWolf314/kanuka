package encrypt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsEncryptPermissions contains permission edge case tests for the `kanuka secrets encrypt` command.
func TestSecretsEncryptPermissions(t *testing.T) {
	// Save original working directory and settings
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings

	t.Run("EncryptWithReadOnlyKanukaDir", func(t *testing.T) {
		testEncryptWithReadOnlyKanukaDir(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptWithReadOnlySecretsDir", func(t *testing.T) {
		testEncryptWithReadOnlySecretsDir(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptWithNoWritePermissionToProject", func(t *testing.T) {
		testEncryptWithNoWritePermissionToProject(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptWithoutAccess", func(t *testing.T) {
		testEncryptWithoutAccess(t, originalWd, originalUserSettings)
	})
}

// Tests encrypt when .kanuka directory is read-only.
func testEncryptWithReadOnlyKanukaDir(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-readonly-kanuka-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create a .env file
	envFile := filepath.Join(tempDir, ".env")
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\nAPI_KEY=secret123\n"
	if err := os.WriteFile(envFile, []byte(envContent), 0600); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Make the .kanuka directory read-only
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	if err := os.Chmod(kanukaDir, 0555); err != nil {
		t.Fatalf("Failed to make .kanuka directory read-only: %v", err)
	}
	// Restore permissions for cleanup
	defer func() {
		if err := os.Chmod(kanukaDir, 0755); err != nil {
			t.Logf("Failed to restore permissions on %s: %v", kanukaDir, err)
		}
	}()

	// Capture output and expect failure
	output, _ := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	// The encrypt command doesn't write to .kanuka directory itself, only reads from it
	// and writes .env.kanuka files to the project root, so read-only .kanuka shouldn't fail
	if err != nil {
		t.Errorf("Expected command to succeed despite read-only .kanuka directory, but it failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Should contain success message
	if !strings.Contains(output, "✓") || !strings.Contains(output, "encrypted successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}
}

// Tests encrypt when .kanuka/secrets directory is read-only.
func testEncryptWithReadOnlySecretsDir(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-readonly-secrets-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create a .env file
	envFile := filepath.Join(tempDir, ".env")
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\nAPI_KEY=secret123\n"
	if err := os.WriteFile(envFile, []byte(envContent), 0600); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Make the secrets directory read-only
	secretsDir := filepath.Join(tempDir, ".kanuka", "secrets")
	if err := os.Chmod(secretsDir, 0555); err != nil {
		t.Fatalf("Failed to make secrets directory read-only: %v", err)
	}
	// Restore permissions for cleanup
	defer func() {
		if err := os.Chmod(secretsDir, 0755); err != nil {
			t.Logf("Failed to restore permissions on %s: %v", secretsDir, err)
		}
	}()

	// Capture output and expect failure
	output, _ := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	// The encrypt command only reads from secrets directory, doesn't write to it
	// so read-only secrets directory shouldn't fail
	if err != nil {
		t.Errorf("Expected command to succeed despite read-only secrets directory, but it failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Should contain success message
	if !strings.Contains(output, "✓") || !strings.Contains(output, "encrypted successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}
}

// Tests encrypt when project directory is not writable.
func testEncryptWithNoWritePermissionToProject(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-no-write-project-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create a .env file
	envFile := filepath.Join(tempDir, ".env")
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\nAPI_KEY=secret123\n"
	if err := os.WriteFile(envFile, []byte(envContent), 0600); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Make the entire project directory read-only
	if err := os.Chmod(tempDir, 0555); err != nil {
		t.Fatalf("Failed to make project directory read-only: %v", err)
	}
	// Restore permissions for cleanup
	defer func() {
		if err := os.Chmod(tempDir, 0755); err != nil {
			t.Logf("Failed to restore permissions on %s: %v", tempDir, err)
		}
	}()

	// Capture output and expect failure
	output, _ := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should fail due to no write permission to project directory
	// The CLI command may not return an error code, but should show failure in output
	if !strings.Contains(output, "Failed to encrypt") || !strings.Contains(output, "permission denied") {
		t.Errorf("Expected permission-related error message, got: %s", output)
	}
}

// testEncryptWithoutAccess tests encrypt when user doesn't have access (missing private key).
func testEncryptWithoutAccess(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-no-access-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory for kanuka settings
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Setup test environment and initialize project
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create a .env file
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\n"
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Remove the user's private key to simulate no access
	projectName := filepath.Base(tempDir)
	privateKeyPath := filepath.Join(tempUserDir, "keys", projectName)
	if err := os.Remove(privateKeyPath); err != nil {
		t.Fatalf("Failed to remove private key: %v", err)
	}

	// Capture output (run in verbose mode to capture final messages)
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	// Command should fail
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	// Verify error message about access
	if !strings.Contains(output, "Failed to get your private key file") {
		t.Errorf("Expected access error message not found in output: %s", output)
	}
}
