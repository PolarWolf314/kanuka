package decrypt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsDecryptProjectState contains project state edge case tests for the `kanuka secrets decrypt` command.
func TestSecretsDecryptProjectState(t *testing.T) {
	// Save original working directory and settings
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings

	t.Run("DecryptWithCorruptedKanukaDir", func(t *testing.T) {
		testDecryptWithCorruptedKanukaDir(t, originalWd, originalUserSettings)
	})

	t.Run("DecryptWithMissingUserKeys", func(t *testing.T) {
		testDecryptWithMissingUserKeys(t, originalWd, originalUserSettings)
	})

	t.Run("DecryptWithoutAccess", func(t *testing.T) {
		testDecryptWithoutAccess(t, originalWd, originalUserSettings)
	})

	t.Run("DecryptFromSubfolderWithOneKanukaFile", func(t *testing.T) {
		testDecryptFromSubfolderWithOneKanukaFile(t, originalWd, originalUserSettings)
	})

	t.Run("DecryptFromSubfolderWithMultipleKanukaFiles", func(t *testing.T) {
		testDecryptFromSubfolderWithMultipleKanukaFiles(t, originalWd, originalUserSettings)
	})
}

// Tests decrypt when .kanuka directory is corrupted.
func testDecryptWithCorruptedKanukaDir(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directories
	tempDir := t.TempDir()
	tempUserDir := t.TempDir()

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize project
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create and encrypt a .env file first
	envFile := filepath.Join(tempDir, ".env")
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\nAPI_KEY=secret123\n"
	if err := os.WriteFile(envFile, []byte(envContent), 0600); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Encrypt the file
	_, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt file for test setup: %v", err)
	}

	// Remove original .env file
	if err := os.Remove(envFile); err != nil {
		t.Fatalf("Failed to remove original .env file: %v", err)
	}

	// Corrupt the .kanuka directory by removing the secrets directory
	secretsDir := filepath.Join(tempDir, ".kanuka", "secrets")
	if err := os.RemoveAll(secretsDir); err != nil {
		t.Fatalf("Failed to remove secrets directory: %v", err)
	}

	// Attempt to decrypt
	output, _ := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})

	if !strings.Contains(output, "Failed to obtain your .kanuka file") || !strings.Contains(output, "no such file or directory") {
		t.Errorf("Expected missing symmetric key error message, got: %s", output)
	}
}

// Tests decrypt when user key files are missing.
func testDecryptWithMissingUserKeys(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directories
	tempDir := t.TempDir()
	tempUserDir := t.TempDir()

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize project
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create and encrypt a .env file first
	envFile := filepath.Join(tempDir, ".env")
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\nAPI_KEY=secret123\n"
	if err := os.WriteFile(envFile, []byte(envContent), 0600); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Encrypt the file
	_, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt file for test setup: %v", err)
	}

	// Remove original .env file
	if err := os.Remove(envFile); err != nil {
		t.Fatalf("Failed to remove original .env file: %v", err)
	}

	// Remove user key files
	userKeysDir := filepath.Join(tempUserDir, "keys")
	if err := os.RemoveAll(userKeysDir); err != nil {
		t.Fatalf("Failed to remove user keys directory: %v", err)
	}

	// Attempt to decrypt
	output, _ := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})

	if !strings.Contains(output, "Failed to get your private key file") || !strings.Contains(output, "no such file or directory") {
		t.Errorf("Expected missing private key error message, got: %s", output)
	}
}

// testDecryptWithoutAccess tests decrypt when user doesn't have access (missing private key).
func testDecryptWithoutAccess(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-no-access-*")
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

	// Create and encrypt a .env file first
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\n"
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Encrypt the file first
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt file for test setup: %v", err)
	}

	// Remove the original .env file
	if err := os.Remove(envPath); err != nil {
		t.Fatalf("Failed to remove .env file: %v", err)
	}

	// Remove the user's private key to simulate no access
	projectUUID := shared.GetProjectUUID(t)
	keyDir := shared.GetKeyDirPath(filepath.Join(tempUserDir, "keys"), projectUUID)
	if err := os.RemoveAll(keyDir); err != nil {
		t.Fatalf("Failed to remove key directory: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "Failed to get your private key file") {
		t.Errorf("Expected access error message not found in output: %s", output)
	}
}

// testDecryptFromSubfolderWithOneKanukaFile tests decrypt from subfolder with one .kanuka file.
func testDecryptFromSubfolderWithOneKanukaFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-subfolder-one-*")
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

	// Create and encrypt a .env file in root
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\n"
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Encrypt the file first
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt file for test setup: %v", err)
	}

	// Remove the original .env file
	if err := os.Remove(envPath); err != nil {
		t.Fatalf("Failed to remove .env file: %v", err)
	}

	// Create a subfolder and change to it
	subDir := filepath.Join(tempDir, "subfolder")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subfolder: %v", err)
	}
	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("Failed to change to subfolder: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "Environment files decrypted successfully") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		t.Errorf(".env file was not recreated at %s", envPath)
	}
}

// testDecryptFromSubfolderWithMultipleKanukaFiles tests decrypt from subfolder with multiple .kanuka files.
func testDecryptFromSubfolderWithMultipleKanukaFiles(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-subfolder-multi-*")
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

	// Create multiple .env files
	envFiles := map[string]string{
		".env":                   "DATABASE_URL=postgres://localhost:5432/mydb\n",
		".env.local":             "API_KEY=secret123\n",
		"config/.env.production": "PROD_API_KEY=prod_secret\n",
		"services/.env.test":     "TEST_DB=test_database\n",
	}

	for filePath, content := range envFiles {
		fullPath := filepath.Join(tempDir, filePath)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		// #nosec G306 -- Writing a file that should be modifiable
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create .env file %s: %v", fullPath, err)
		}
	}

	// Encrypt all files first
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt files for test setup: %v", err)
	}

	// Remove all original .env files
	for filePath := range envFiles {
		fullPath := filepath.Join(tempDir, filePath)
		if err := os.Remove(fullPath); err != nil {
			t.Fatalf("Failed to remove .env file %s: %v", fullPath, err)
		}
	}

	// Create a subfolder and change to it
	subDir := filepath.Join(tempDir, "subfolder")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subfolder: %v", err)
	}
	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("Failed to change to subfolder: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "Environment files decrypted successfully") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	for filePath := range envFiles {
		fullPath := filepath.Join(tempDir, filePath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf(".env file was not recreated at %s", fullPath)
		}
	}
}
