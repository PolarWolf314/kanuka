package encrypt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSelectiveEncryptIntegration contains integration tests for selective file encryption.
func TestSelectiveEncryptIntegration(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("EncryptSingleFile", func(t *testing.T) {
		testEncryptSingleFile(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptMultipleFiles", func(t *testing.T) {
		testEncryptMultipleFiles(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptDirectory", func(t *testing.T) {
		testEncryptDirectory(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptWithGlobPattern", func(t *testing.T) {
		testEncryptWithGlobPattern(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptWithDoubleStarGlob", func(t *testing.T) {
		testEncryptWithDoubleStarGlob(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptNonExistentFile", func(t *testing.T) {
		testEncryptNonExistentFile(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptNonEnvFile", func(t *testing.T) {
		testEncryptNonEnvFile(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptLeavesOtherFilesUntouched", func(t *testing.T) {
		testEncryptLeavesOtherFilesUntouched(t, originalWd, originalUserSettings)
	})
}

// testEncryptSingleFile tests encrypting a single specified file.
func testEncryptSingleFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-single-*")
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

	// Create multiple .env files.
	envFiles := map[string]string{
		".env":       "DATABASE_URL=postgres://localhost:5432/mydb\n",
		".env.local": "API_KEY=secret123\n",
		".env.prod":  "PROD_KEY=prod_secret\n",
	}

	for name, content := range envFiles {
		envPath := filepath.Join(tempDir, name)
		// #nosec G306 -- Writing a file that should be modifiable.
		if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s file: %v", name, err)
		}
	}

	// Encrypt only .env.local.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("encrypt", []string{".env.local"}, nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Check that only .env.local.kanuka was created.
	if _, err := os.Stat(filepath.Join(tempDir, ".env.local.kanuka")); os.IsNotExist(err) {
		t.Errorf(".env.local.kanuka was not created")
	}

	// Check that .env.kanuka was NOT created.
	if _, err := os.Stat(filepath.Join(tempDir, ".env.kanuka")); !os.IsNotExist(err) {
		t.Errorf(".env.kanuka should not have been created")
	}

	// Check that .env.prod.kanuka was NOT created.
	if _, err := os.Stat(filepath.Join(tempDir, ".env.prod.kanuka")); !os.IsNotExist(err) {
		t.Errorf(".env.prod.kanuka should not have been created")
	}
}

// testEncryptMultipleFiles tests encrypting multiple specified files.
func testEncryptMultipleFiles(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-multi-*")
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

	// Create multiple .env files.
	envFiles := map[string]string{
		".env":       "DATABASE_URL=postgres://localhost:5432/mydb\n",
		".env.local": "API_KEY=secret123\n",
		".env.prod":  "PROD_KEY=prod_secret\n",
	}

	for name, content := range envFiles {
		envPath := filepath.Join(tempDir, name)
		// #nosec G306 -- Writing a file that should be modifiable.
		if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s file: %v", name, err)
		}
	}

	// Encrypt .env and .env.local but not .env.prod.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("encrypt", []string{".env", ".env.local"}, nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Check that .env.kanuka was created.
	if _, err := os.Stat(filepath.Join(tempDir, ".env.kanuka")); os.IsNotExist(err) {
		t.Errorf(".env.kanuka was not created")
	}

	// Check that .env.local.kanuka was created.
	if _, err := os.Stat(filepath.Join(tempDir, ".env.local.kanuka")); os.IsNotExist(err) {
		t.Errorf(".env.local.kanuka was not created")
	}

	// Check that .env.prod.kanuka was NOT created.
	if _, err := os.Stat(filepath.Join(tempDir, ".env.prod.kanuka")); !os.IsNotExist(err) {
		t.Errorf(".env.prod.kanuka should not have been created")
	}
}

// testEncryptDirectory tests encrypting all .env files in a directory.
func testEncryptDirectory(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-dir-*")
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

	// Create .env file in root.
	rootEnvPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable.
	if err := os.WriteFile(rootEnvPath, []byte("ROOT_KEY=root\n"), 0644); err != nil {
		t.Fatalf("Failed to create root .env file: %v", err)
	}

	// Create subdirectory with .env files.
	subDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Create .env files in subdirectory.
	envFiles := map[string]string{
		".env":      "CONFIG_DB=db\n",
		".env.test": "TEST_KEY=test\n",
	}

	for name, content := range envFiles {
		envPath := filepath.Join(subDir, name)
		// #nosec G306 -- Writing a file that should be modifiable.
		if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s file: %v", name, err)
		}
	}

	// Encrypt only the config directory.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("encrypt", []string{"config"}, nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Check that config/.env.kanuka was created.
	if _, err := os.Stat(filepath.Join(subDir, ".env.kanuka")); os.IsNotExist(err) {
		t.Errorf("config/.env.kanuka was not created")
	}

	// Check that config/.env.test.kanuka was created.
	if _, err := os.Stat(filepath.Join(subDir, ".env.test.kanuka")); os.IsNotExist(err) {
		t.Errorf("config/.env.test.kanuka was not created")
	}

	// Check that root .env.kanuka was NOT created.
	if _, err := os.Stat(filepath.Join(tempDir, ".env.kanuka")); !os.IsNotExist(err) {
		t.Errorf(".env.kanuka in root should not have been created")
	}
}

// testEncryptWithGlobPattern tests encrypting with glob patterns.
func testEncryptWithGlobPattern(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-glob-*")
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

	// Create multiple .env files.
	envFiles := map[string]string{
		".env":       "DATABASE_URL=postgres://localhost:5432/mydb\n",
		".env.local": "API_KEY=secret123\n",
		".env.prod":  "PROD_KEY=prod_secret\n",
		".env.test":  "TEST_KEY=test\n",
	}

	for name, content := range envFiles {
		envPath := filepath.Join(tempDir, name)
		// #nosec G306 -- Writing a file that should be modifiable.
		if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s file: %v", name, err)
		}
	}

	// Encrypt with glob pattern matching .env.*.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("encrypt", []string{".env.*"}, nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Check that .env.local.kanuka, .env.prod.kanuka, .env.test.kanuka were created.
	for _, name := range []string{".env.local", ".env.prod", ".env.test"} {
		if _, err := os.Stat(filepath.Join(tempDir, name+".kanuka")); os.IsNotExist(err) {
			t.Errorf("%s.kanuka was not created", name)
		}
	}

	// Check that .env.kanuka was NOT created (doesn't match .env.*).
	if _, err := os.Stat(filepath.Join(tempDir, ".env.kanuka")); !os.IsNotExist(err) {
		t.Errorf(".env.kanuka should not have been created")
	}
}

// testEncryptWithDoubleStarGlob tests encrypting with ** glob patterns.
func testEncryptWithDoubleStarGlob(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-doublestar-*")
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

	// Create .env files at various levels.
	paths := []string{
		".env.prod",
		"services/api/.env.prod",
		"services/web/.env.prod",
	}

	for _, p := range paths {
		fullPath := filepath.Join(tempDir, p)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		// #nosec G306 -- Writing a file that should be modifiable.
		if err := os.WriteFile(fullPath, []byte("KEY=value\n"), 0644); err != nil {
			t.Fatalf("Failed to create %s file: %v", p, err)
		}
	}

	// Also create a .env.local that should NOT be matched.
	localEnvPath := filepath.Join(tempDir, ".env.local")
	// #nosec G306 -- Writing a file that should be modifiable.
	if err := os.WriteFile(localEnvPath, []byte("LOCAL=value\n"), 0644); err != nil {
		t.Fatalf("Failed to create .env.local file: %v", err)
	}

	// Encrypt with ** glob pattern matching all .env.prod files.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("encrypt", []string{"**/.env.prod"}, nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Check that all .env.prod.kanuka files were created.
	for _, p := range paths {
		kanukaPath := filepath.Join(tempDir, p+".kanuka")
		if _, err := os.Stat(kanukaPath); os.IsNotExist(err) {
			t.Errorf("%s.kanuka was not created", p)
		}
	}

	// Check that .env.local.kanuka was NOT created.
	if _, err := os.Stat(filepath.Join(tempDir, ".env.local.kanuka")); !os.IsNotExist(err) {
		t.Errorf(".env.local.kanuka should not have been created")
	}
}

// testEncryptNonExistentFile tests error handling for non-existent files.
func testEncryptNonExistentFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-nonexistent-*")
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

	// Try to encrypt a file that doesn't exist.
	output, _ := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("encrypt", []string{".env.nonexistent"}, nil, nil, true, false)
		return cmd.Execute()
	})

	// Should show an error message about file not found.
	// The command may return nil but displays error in spinner.FinalMSG.
	if !strings.Contains(output, "not found") && !strings.Contains(strings.ToLower(output), "no such file") {
		t.Errorf("Expected 'not found' error message, got: %s", output)
	}
}

// testEncryptNonEnvFile tests error handling for non-.env files.
func testEncryptNonEnvFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-nonenv-*")
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

	// Create a non-.env file.
	configPath := filepath.Join(tempDir, "config.json")
	// #nosec G306 -- Writing a file that should be modifiable.
	if err := os.WriteFile(configPath, []byte(`{"key": "value"}`), 0644); err != nil {
		t.Fatalf("Failed to create config.json file: %v", err)
	}

	// Try to encrypt the non-.env file.
	output, _ := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("encrypt", []string{"config.json"}, nil, nil, true, false)
		return cmd.Execute()
	})

	// Should show an error message about not being a .env file.
	// The command may return nil but displays error in spinner.FinalMSG.
	if !strings.Contains(output, "not a .env file") {
		t.Errorf("Expected 'not a .env file' error message, got: %s", output)
	}
}

// testEncryptLeavesOtherFilesUntouched tests that selective encryption doesn't affect other files.
func testEncryptLeavesOtherFilesUntouched(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-untouched-*")
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

	// Create multiple .env files.
	envPath1 := filepath.Join(tempDir, ".env")
	envPath2 := filepath.Join(tempDir, ".env.other")
	content1 := "KEY1=value1\n"
	content2 := "KEY2=value2\n"

	// #nosec G306 -- Writing a file that should be modifiable.
	if err := os.WriteFile(envPath1, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}
	// #nosec G306 -- Writing a file that should be modifiable.
	if err := os.WriteFile(envPath2, []byte(content2), 0644); err != nil {
		t.Fatalf("Failed to create .env.other file: %v", err)
	}

	// Encrypt only .env.
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("encrypt", []string{".env"}, nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Check that .env.other still exists with original content.
	content, err := os.ReadFile(envPath2)
	if err != nil {
		t.Errorf("Failed to read .env.other: %v", err)
	}
	if string(content) != content2 {
		t.Errorf(".env.other content changed. Expected: %s, Got: %s", content2, string(content))
	}

	// Check that .env.other.kanuka was NOT created.
	if _, err := os.Stat(envPath2 + ".kanuka"); !os.IsNotExist(err) {
		t.Errorf(".env.other.kanuka should not have been created")
	}
}
