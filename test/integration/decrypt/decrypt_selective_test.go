package decrypt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSelectiveDecryptIntegration contains integration tests for selective file decryption.
func TestSelectiveDecryptIntegration(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("DecryptSingleFile", func(t *testing.T) {
		testDecryptSingleFile(t, originalWd, originalUserSettings)
	})

	t.Run("DecryptMultipleFiles", func(t *testing.T) {
		testDecryptMultipleFiles(t, originalWd, originalUserSettings)
	})

	t.Run("DecryptDirectory", func(t *testing.T) {
		testDecryptDirectory(t, originalWd, originalUserSettings)
	})

	t.Run("DecryptWithGlobPattern", func(t *testing.T) {
		testDecryptWithGlobPattern(t, originalWd, originalUserSettings)
	})

	t.Run("DecryptWithDoubleStarGlob", func(t *testing.T) {
		testDecryptWithDoubleStarGlob(t, originalWd, originalUserSettings)
	})

	t.Run("DecryptNonExistentFile", func(t *testing.T) {
		testDecryptNonExistentFile(t, originalWd, originalUserSettings)
	})

	t.Run("DecryptNonKanukaFile", func(t *testing.T) {
		testDecryptNonKanukaFile(t, originalWd, originalUserSettings)
	})

	t.Run("DecryptLeavesOtherFilesUntouched", func(t *testing.T) {
		testDecryptLeavesOtherFilesUntouched(t, originalWd, originalUserSettings)
	})
}

// testDecryptSingleFile tests decrypting a single specified file.
func testDecryptSingleFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-single-*")
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

	// Create and encrypt multiple .env files.
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

	// Encrypt all files.
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// Remove all original .env files.
	for name := range envFiles {
		if err := os.Remove(filepath.Join(tempDir, name)); err != nil {
			t.Fatalf("Failed to remove %s: %v", name, err)
		}
	}

	// Decrypt only .env.local.kanuka.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("decrypt", []string{".env.local.kanuka"}, nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Check that only .env.local was recreated.
	if _, err := os.Stat(filepath.Join(tempDir, ".env.local")); os.IsNotExist(err) {
		t.Errorf(".env.local was not recreated")
	}

	// Check that .env was NOT recreated.
	if _, err := os.Stat(filepath.Join(tempDir, ".env")); !os.IsNotExist(err) {
		t.Errorf(".env should not have been recreated")
	}

	// Check that .env.prod was NOT recreated.
	if _, err := os.Stat(filepath.Join(tempDir, ".env.prod")); !os.IsNotExist(err) {
		t.Errorf(".env.prod should not have been recreated")
	}

	// Verify content.
	content, err := os.ReadFile(filepath.Join(tempDir, ".env.local"))
	if err != nil {
		t.Errorf("Failed to read .env.local: %v", err)
	}
	if string(content) != envFiles[".env.local"] {
		t.Errorf("Content mismatch. Expected: %s, Got: %s", envFiles[".env.local"], string(content))
	}
}

// testDecryptMultipleFiles tests decrypting multiple specified files.
func testDecryptMultipleFiles(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-multi-*")
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

	// Create and encrypt multiple .env files.
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

	// Encrypt all files.
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// Remove all original .env files.
	for name := range envFiles {
		if err := os.Remove(filepath.Join(tempDir, name)); err != nil {
			t.Fatalf("Failed to remove %s: %v", name, err)
		}
	}

	// Decrypt .env.kanuka and .env.local.kanuka but not .env.prod.kanuka.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("decrypt", []string{".env.kanuka", ".env.local.kanuka"}, nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Check that .env was recreated.
	if _, err := os.Stat(filepath.Join(tempDir, ".env")); os.IsNotExist(err) {
		t.Errorf(".env was not recreated")
	}

	// Check that .env.local was recreated.
	if _, err := os.Stat(filepath.Join(tempDir, ".env.local")); os.IsNotExist(err) {
		t.Errorf(".env.local was not recreated")
	}

	// Check that .env.prod was NOT recreated.
	if _, err := os.Stat(filepath.Join(tempDir, ".env.prod")); !os.IsNotExist(err) {
		t.Errorf(".env.prod should not have been recreated")
	}
}

// testDecryptDirectory tests decrypting all .kanuka files in a directory.
func testDecryptDirectory(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-dir-*")
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

	// Encrypt all files.
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// Remove all original .env files.
	if err := os.Remove(rootEnvPath); err != nil {
		t.Fatalf("Failed to remove root .env: %v", err)
	}
	for name := range envFiles {
		if err := os.Remove(filepath.Join(subDir, name)); err != nil {
			t.Fatalf("Failed to remove %s: %v", name, err)
		}
	}

	// Decrypt only the config directory.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("decrypt", []string{"config"}, nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Check that config/.env was recreated.
	if _, err := os.Stat(filepath.Join(subDir, ".env")); os.IsNotExist(err) {
		t.Errorf("config/.env was not recreated")
	}

	// Check that config/.env.test was recreated.
	if _, err := os.Stat(filepath.Join(subDir, ".env.test")); os.IsNotExist(err) {
		t.Errorf("config/.env.test was not recreated")
	}

	// Check that root .env was NOT recreated.
	if _, err := os.Stat(rootEnvPath); !os.IsNotExist(err) {
		t.Errorf(".env in root should not have been recreated")
	}
}

// testDecryptWithGlobPattern tests decrypting with glob patterns.
func testDecryptWithGlobPattern(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-glob-*")
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

	// Encrypt all files.
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// Remove all original .env files.
	for name := range envFiles {
		if err := os.Remove(filepath.Join(tempDir, name)); err != nil {
			t.Fatalf("Failed to remove %s: %v", name, err)
		}
	}

	// Decrypt with glob pattern matching .env.*.kanuka.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("decrypt", []string{".env.*.kanuka"}, nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Check that .env.local, .env.prod, .env.test were recreated.
	for _, name := range []string{".env.local", ".env.prod", ".env.test"} {
		if _, err := os.Stat(filepath.Join(tempDir, name)); os.IsNotExist(err) {
			t.Errorf("%s was not recreated", name)
		}
	}

	// Check that .env was NOT recreated (doesn't match .env.*.kanuka).
	if _, err := os.Stat(filepath.Join(tempDir, ".env")); !os.IsNotExist(err) {
		t.Errorf(".env should not have been recreated")
	}
}

// testDecryptWithDoubleStarGlob tests decrypting with ** glob patterns.
func testDecryptWithDoubleStarGlob(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-doublestar-*")
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
	envFiles := map[string]string{
		".env.prod":              "ROOT_PROD=value\n",
		"services/api/.env.prod": "API_PROD=value\n",
		"services/web/.env.prod": "WEB_PROD=value\n",
		".env.local":             "LOCAL=value\n",
	}

	for p, content := range envFiles {
		fullPath := filepath.Join(tempDir, p)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		// #nosec G306 -- Writing a file that should be modifiable.
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s file: %v", p, err)
		}
	}

	// Encrypt all files.
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// Remove all original .env files.
	for p := range envFiles {
		if err := os.Remove(filepath.Join(tempDir, p)); err != nil {
			t.Fatalf("Failed to remove %s: %v", p, err)
		}
	}

	// Decrypt with ** glob pattern matching all .env.prod.kanuka files.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("decrypt", []string{"**/.env.prod.kanuka"}, nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Check that all .env.prod files were recreated.
	for _, p := range []string{".env.prod", "services/api/.env.prod", "services/web/.env.prod"} {
		if _, err := os.Stat(filepath.Join(tempDir, p)); os.IsNotExist(err) {
			t.Errorf("%s was not recreated", p)
		}
	}

	// Check that .env.local was NOT recreated.
	if _, err := os.Stat(filepath.Join(tempDir, ".env.local")); !os.IsNotExist(err) {
		t.Errorf(".env.local should not have been recreated")
	}
}

// testDecryptNonExistentFile tests error handling for non-existent files.
func testDecryptNonExistentFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-nonexistent-*")
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

	// Try to decrypt a file that doesn't exist.
	output, _ := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("decrypt", []string{".env.nonexistent.kanuka"}, nil, nil, true, false)
		return cmd.Execute()
	})

	// Should show an error message about file not found.
	// The command may return nil but displays error in spinner.FinalMSG.
	if !strings.Contains(output, "not found") && !strings.Contains(strings.ToLower(output), "no such file") {
		t.Errorf("Expected 'not found' error message, got: %s", output)
	}
}

// testDecryptNonKanukaFile tests error handling for non-.kanuka files.
func testDecryptNonKanukaFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-nonkanuka-*")
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

	// Create a non-.kanuka file.
	configPath := filepath.Join(tempDir, "config.json")
	// #nosec G306 -- Writing a file that should be modifiable.
	if err := os.WriteFile(configPath, []byte(`{"key": "value"}`), 0644); err != nil {
		t.Fatalf("Failed to create config.json file: %v", err)
	}

	// Try to decrypt the non-.kanuka file.
	output, _ := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("decrypt", []string{"config.json"}, nil, nil, true, false)
		return cmd.Execute()
	})

	// Should show an error message about not being a .kanuka file.
	// The command may return nil but displays error in spinner.FinalMSG.
	if !strings.Contains(output, "not a .kanuka file") {
		t.Errorf("Expected 'not a .kanuka file' error message, got: %s", output)
	}
}

// testDecryptLeavesOtherFilesUntouched tests that selective decryption doesn't affect other files.
func testDecryptLeavesOtherFilesUntouched(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-untouched-*")
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

	// Create and encrypt multiple .env files.
	envFiles := map[string]string{
		".env":       "KEY1=value1\n",
		".env.other": "KEY2=value2\n",
	}

	for name, content := range envFiles {
		envPath := filepath.Join(tempDir, name)
		// #nosec G306 -- Writing a file that should be modifiable.
		if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", name, err)
		}
	}

	// Encrypt all files.
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// Remove only .env (leave .env.other).
	if err := os.Remove(filepath.Join(tempDir, ".env")); err != nil {
		t.Fatalf("Failed to remove .env: %v", err)
	}

	// Decrypt only .env.kanuka.
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("decrypt", []string{".env.kanuka"}, nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Check that .env.other still exists with original content (was not modified).
	content, err := os.ReadFile(filepath.Join(tempDir, ".env.other"))
	if err != nil {
		t.Errorf("Failed to read .env.other: %v", err)
	}
	if string(content) != envFiles[".env.other"] {
		t.Errorf(".env.other content changed. Expected: %s, Got: %s", envFiles[".env.other"], string(content))
	}

	// Check that .env was recreated correctly.
	content, err = os.ReadFile(filepath.Join(tempDir, ".env"))
	if err != nil {
		t.Errorf("Failed to read .env: %v", err)
	}
	if string(content) != envFiles[".env"] {
		t.Errorf(".env content mismatch. Expected: %s, Got: %s", envFiles[".env"], string(content))
	}
}
