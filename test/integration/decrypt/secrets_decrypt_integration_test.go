package decrypt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsDecryptIntegration contains integration tests for the `kanuka secrets decrypt` command.
func TestSecretsDecryptIntegration(t *testing.T) {
	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	// Save original user settings to restore later
	originalUserSettings := configs.UserKanukaSettings

	t.Run("DecryptInEmptyFolder", func(t *testing.T) {
		testDecryptInEmptyFolder(t, originalWd, originalUserSettings)
	})

	t.Run("DecryptInInitializedFolderWithNoKanukaFiles", func(t *testing.T) {
		testDecryptInInitializedFolderWithNoKanukaFiles(t, originalWd, originalUserSettings)
	})

	t.Run("DecryptInInitializedFolderWithOneKanukaFile", func(t *testing.T) {
		testDecryptInInitializedFolderWithOneKanukaFile(t, originalWd, originalUserSettings)
	})

	t.Run("DecryptInInitializedFolderWithMultipleKanukaFiles", func(t *testing.T) {
		testDecryptInInitializedFolderWithMultipleKanukaFiles(t, originalWd, originalUserSettings)
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

// testDecryptInEmptyFolder tests decrypt command in an empty folder (should fail).
func testDecryptInEmptyFolder(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-empty-*")
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

	// Setup test environment
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Capture output (run in verbose mode to capture final messages)
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	// Command should fail because kanuka is not initialized
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	// Verify error message about not being initialized
	if !strings.Contains(output, "Kanuka has not been initialized") {
		t.Errorf("Expected 'not initialized' message not found in output: %s", output)
	}
}

// testDecryptInInitializedFolderWithNoKanukaFiles tests decrypt in initialized folder with no .kanuka files.
func testDecryptInInitializedFolderWithNoKanukaFiles(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-no-kanuka-*")
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

	// Capture output (run in verbose mode to capture final messages)
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	// Command should succeed but report no files found
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	// Verify message about no kanuka files found
	if !strings.Contains(output, "No encrypted environment (.kanuka) files found") {
		t.Errorf("Expected 'no kanuka files found' message not found in output: %s", output)
	}
}

// testDecryptInInitializedFolderWithOneKanukaFile tests decrypt with one .kanuka file.
func testDecryptInInitializedFolderWithOneKanukaFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-one-kanuka-*")
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

	// Create and encrypt a .env file first
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\nAPI_KEY=secret123\n"
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

	// Remove the original .env file to test decryption
	if err := os.Remove(envPath); err != nil {
		t.Fatalf("Failed to remove .env file: %v", err)
	}

	// Verify .env.kanuka file exists
	kanukaPath := envPath + ".kanuka"
	if _, err := os.Stat(kanukaPath); os.IsNotExist(err) {
		t.Fatalf(".env.kanuka file was not created during setup")
	}

	// Capture output (run in verbose mode to capture final messages)
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	// Command should succeed
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Verify success message
	if !strings.Contains(output, "Environment files decrypted successfully") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	// Verify .env file was recreated
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		t.Errorf(".env file was not recreated at %s", envPath)
	}

	// Verify the decrypted content matches the original
	decryptedContent, err := os.ReadFile(envPath)
	if err != nil {
		t.Errorf("Failed to read decrypted .env file: %v", err)
	}
	if string(decryptedContent) != envContent {
		t.Errorf("Decrypted content doesn't match original. Expected: %s, Got: %s", envContent, string(decryptedContent))
	}
}

// testDecryptInInitializedFolderWithMultipleKanukaFiles tests decrypt with multiple .kanuka files.
func testDecryptInInitializedFolderWithMultipleKanukaFiles(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-multi-kanuka-*")
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

	// Remove all original .env files to test decryption
	for filePath := range envFiles {
		fullPath := filepath.Join(tempDir, filePath)
		if err := os.Remove(fullPath); err != nil {
			t.Fatalf("Failed to remove .env file %s: %v", fullPath, err)
		}
	}

	// Capture output (run in verbose mode to capture final messages)
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	// Command should succeed
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Verify success message
	if !strings.Contains(output, "Environment files decrypted successfully") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	// Verify all .env files were recreated with correct content
	for filePath, expectedContent := range envFiles {
		fullPath := filepath.Join(tempDir, filePath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf(".env file was not recreated at %s", fullPath)
			continue
		}

		decryptedContent, err := os.ReadFile(fullPath)
		if err != nil {
			t.Errorf("Failed to read decrypted .env file %s: %v", fullPath, err)
			continue
		}
		if string(decryptedContent) != expectedContent {
			t.Errorf("Decrypted content doesn't match original for %s. Expected: %s, Got: %s", filePath, expectedContent, string(decryptedContent))
		}
	}
}

// testDecryptWithoutAccess tests decrypt when user doesn't have access (missing private key).
func testDecryptWithoutAccess(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-no-access-*")
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
	projectName := filepath.Base(tempDir)
	privateKeyPath := filepath.Join(tempUserDir, "keys", projectName)
	if err := os.Remove(privateKeyPath); err != nil {
		t.Fatalf("Failed to remove private key: %v", err)
	}

	// Capture output (run in verbose mode to capture final messages)
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
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

// testDecryptFromSubfolderWithOneKanukaFile tests decrypt from subfolder with one .kanuka file.
func testDecryptFromSubfolderWithOneKanukaFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-subfolder-one-*")
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

	// Capture output (run in verbose mode to capture final messages)
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	// Command should succeed
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Verify success message
	if !strings.Contains(output, "Environment files decrypted successfully") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	// Verify .env file was recreated in the root
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		t.Errorf(".env file was not recreated at %s", envPath)
	}
}

// testDecryptFromSubfolderWithMultipleKanukaFiles tests decrypt from subfolder with multiple .kanuka files.
func testDecryptFromSubfolderWithMultipleKanukaFiles(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-subfolder-multi-*")
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

	// Capture output (run in verbose mode to capture final messages)
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	// Command should succeed
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Verify success message
	if !strings.Contains(output, "Environment files decrypted successfully") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	// Verify all .env files were recreated
	for filePath := range envFiles {
		fullPath := filepath.Join(tempDir, filePath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf(".env file was not recreated at %s", fullPath)
		}
	}
}
