package encrypt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsEncryptIntegration contains integration tests for the `kanuka secrets encrypt` command.
func TestSecretsEncryptIntegration(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("EncryptInEmptyFolder", func(t *testing.T) {
		testEncryptInEmptyFolder(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptInInitializedFolderWithNoEnvFiles", func(t *testing.T) {
		testEncryptInInitializedFolderWithNoEnvFiles(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptInInitializedFolderWithOneEnvFile", func(t *testing.T) {
		testEncryptInInitializedFolderWithOneEnvFile(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptInInitializedFolderWithMultipleEnvFiles", func(t *testing.T) {
		testEncryptInInitializedFolderWithMultipleEnvFiles(t, originalWd, originalUserSettings)
	})
}

// testEncryptInEmptyFolder tests encrypt command in an empty folder (should fail).
func testEncryptInEmptyFolder(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-empty-*")
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

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "Kanuka has not been initialized") {
		t.Errorf("Expected 'not initialized' message not found in output: %s", output)
	}
}

// testEncryptInInitializedFolderWithNoEnvFiles tests encrypt in initialized folder with no .env files.
func testEncryptInInitializedFolderWithNoEnvFiles(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-no-env-*")
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

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "No environment files found") {
		t.Errorf("Expected 'no environment files found' message not found in output: %s", output)
	}
}

// testEncryptInInitializedFolderWithOneEnvFile tests encrypt with one .env file.
func testEncryptInInitializedFolderWithOneEnvFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-one-env-*")
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

	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\nAPI_KEY=secret123\n"
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "Environment files encrypted successfully") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	kanukaPath := envPath + ".kanuka"
	if _, err := os.Stat(kanukaPath); os.IsNotExist(err) {
		t.Errorf(".env.kanuka file was not created at %s", kanukaPath)
	}

	kanukaContent, err := os.ReadFile(kanukaPath)
	if err != nil {
		t.Errorf("Failed to read .env.kanuka file: %v", err)
	}
	if len(kanukaContent) == 0 {
		t.Errorf(".env.kanuka file is empty")
	}
	if string(kanukaContent) == envContent {
		t.Errorf(".env.kanuka file content is the same as .env file (not encrypted)")
	}
}

// testEncryptInInitializedFolderWithMultipleEnvFiles tests encrypt with multiple .env files.
func testEncryptInInitializedFolderWithMultipleEnvFiles(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-multi-env-*")
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

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "Environment files encrypted successfully") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	for filePath := range envFiles {
		kanukaPath := filepath.Join(tempDir, filePath+".kanuka")
		if _, err := os.Stat(kanukaPath); os.IsNotExist(err) {
			t.Errorf(".env.kanuka file was not created at %s", kanukaPath)
		}
	}
}
