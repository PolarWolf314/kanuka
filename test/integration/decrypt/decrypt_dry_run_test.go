package decrypt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/cmd"
	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestDecryptDryRun_PreviewsWithoutCreating tests that --dry-run shows preview without creating files.
func TestDecryptDryRun_PreviewsWithoutCreating(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-dry-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize project.
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create and encrypt a .env file.
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\nAPI_KEY=secret123\n"
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Encrypt the file.
	_, err = shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt file for test setup: %v", err)
	}

	// Remove the original .env file to test decryption preview.
	if err := os.Remove(envPath); err != nil {
		t.Fatalf("Failed to remove .env file: %v", err)
	}

	// Verify .env does NOT exist before dry-run.
	if _, err := os.Stat(envPath); !os.IsNotExist(err) {
		t.Fatal(".env should not exist before dry-run")
	}

	// Run decrypt with --dry-run.
	output, err := shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("decrypt", []string{"--dry-run"}, nil, nil, true, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Dry-run command should not return error: %v", err)
	}

	// Verify output contains expected dry-run messages.
	if !strings.Contains(output, "[dry-run]") {
		t.Errorf("Output should contain '[dry-run]' prefix, got: %s", output)
	}
	if !strings.Contains(output, "Would decrypt") {
		t.Errorf("Output should contain 'Would decrypt', got: %s", output)
	}
	if !strings.Contains(output, "Files that would be created") {
		t.Errorf("Output should contain 'Files that would be created', got: %s", output)
	}
	if !strings.Contains(output, "No changes made") {
		t.Errorf("Output should contain 'No changes made', got: %s", output)
	}

	// Verify .env was NOT created.
	if _, err := os.Stat(envPath); !os.IsNotExist(err) {
		t.Error(".env file should NOT be created after dry-run")
	}
}

// TestDecryptDryRun_ShowsOverwriteWarning tests that --dry-run detects existing files that would be overwritten.
func TestDecryptDryRun_ShowsOverwriteWarning(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-dry-overwrite-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize project.
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create and encrypt a .env file.
	envContent := "API_KEY=original_secret\n"
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Encrypt the file.
	_, err = shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt file for test setup: %v", err)
	}

	// DO NOT remove the .env file - leave it in place to test overwrite detection.
	// Modify it to simulate local changes that would be overwritten.
	modifiedContent := "API_KEY=modified_local_value\n"
	// #nosec G306 -- Writing a file that should be modifiable
	if err := os.WriteFile(envPath, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to modify .env file: %v", err)
	}

	// Run decrypt with --dry-run.
	output, err := shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("decrypt", []string{"--dry-run"}, nil, nil, true, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Dry-run command should not return error: %v", err)
	}

	// Verify output shows overwrite warning.
	if !strings.Contains(output, "exists - would be overwritten") {
		t.Errorf("Output should contain 'exists - would be overwritten', got: %s", output)
	}
	if !strings.Contains(output, "Warning") {
		t.Errorf("Output should contain 'Warning' for overwrite, got: %s", output)
	}
	if !strings.Contains(output, "1 existing file(s) would be overwritten") {
		t.Errorf("Output should contain '1 existing file(s) would be overwritten', got: %s", output)
	}

	// Verify the .env file was NOT overwritten - still has modified content.
	actualContent, err := os.ReadFile(envPath)
	if err != nil {
		t.Errorf("Failed to read .env file: %v", err)
	}
	if string(actualContent) != modifiedContent {
		t.Errorf(".env file should still have modified content after dry-run, got: %s", string(actualContent))
	}
}

// TestDecryptDryRun_ShowsNewFileStatus tests that --dry-run shows "new file" for files that don't exist.
func TestDecryptDryRun_ShowsNewFileStatus(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-dry-new-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize project.
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create and encrypt a .env file.
	envContent := "API_KEY=secret\n"
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Encrypt the file.
	_, err = shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt file for test setup: %v", err)
	}

	// Remove the .env file so it shows as "new file".
	if err := os.Remove(envPath); err != nil {
		t.Fatalf("Failed to remove .env file: %v", err)
	}

	// Run decrypt with --dry-run.
	output, err := shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("decrypt", []string{"--dry-run"}, nil, nil, true, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Dry-run command should not return error: %v", err)
	}

	// Verify output shows "new file" status.
	if !strings.Contains(output, "new file") {
		t.Errorf("Output should contain 'new file', got: %s", output)
	}
	// Should NOT show overwrite warning since no files exist.
	if strings.Contains(output, "would be overwritten") {
		t.Errorf("Output should NOT contain overwrite warning when file doesn't exist, got: %s", output)
	}
}

// TestDecryptDryRun_MixedNewAndExisting tests --dry-run with both new files and files that would be overwritten.
func TestDecryptDryRun_MixedNewAndExisting(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-dry-mixed-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize project.
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create multiple .env files.
	envFiles := map[string]string{
		".env":       "ROOT=value\n",
		".env.local": "LOCAL=value\n",
	}

	for filePath, content := range envFiles {
		fullPath := filepath.Join(tempDir, filePath)
		// #nosec G306 -- Writing a file that should be modifiable
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create .env file %s: %v", fullPath, err)
		}
	}

	// Encrypt all files.
	_, err = shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt files for test setup: %v", err)
	}

	// Remove only one of the .env files - keep the other to test mixed scenario.
	if err := os.Remove(filepath.Join(tempDir, ".env")); err != nil {
		t.Fatalf("Failed to remove .env file: %v", err)
	}
	// Keep .env.local to simulate an existing file that would be overwritten.

	// Run decrypt with --dry-run.
	output, err := shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("decrypt", []string{"--dry-run"}, nil, nil, true, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Dry-run command should not return error: %v", err)
	}

	// Verify output shows count of 2 files.
	if !strings.Contains(output, "2 encrypted file(s)") {
		t.Errorf("Output should contain '2 encrypted file(s)', got: %s", output)
	}

	// Verify output shows both statuses.
	if !strings.Contains(output, "new file") {
		t.Errorf("Output should contain 'new file' for .env, got: %s", output)
	}
	if !strings.Contains(output, "exists - would be overwritten") {
		t.Errorf("Output should contain 'exists - would be overwritten' for .env.local, got: %s", output)
	}

	// Verify warning shows 1 file would be overwritten.
	if !strings.Contains(output, "1 existing file(s) would be overwritten") {
		t.Errorf("Output should contain '1 existing file(s) would be overwritten', got: %s", output)
	}
}

// TestDecryptDryRun_NotInitialized tests that validation errors occur with --dry-run when project not initialized.
func TestDecryptDryRun_NotInitialized(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-dry-uninit-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Do NOT initialize project - should fail validation.

	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Command should not return error: %v", err)
	}

	// Should show "not initialized" message.
	if !strings.Contains(output, "not been initialized") {
		t.Errorf("Output should contain 'not been initialized', got: %s", output)
	}
}

// TestDecryptDryRun_NoKanukaFiles tests that validation errors occur with --dry-run when no .kanuka files exist.
func TestDecryptDryRun_NoKanukaFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-dry-nofiles-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize project but don't create any .kanuka files.
	shared.InitializeProject(t, tempDir, tempUserDir)

	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Command should not return error: %v", err)
	}

	// Should show "no .kanuka files found" message.
	if !strings.Contains(output, "No encrypted environment") {
		t.Errorf("Output should contain 'No encrypted environment', got: %s", output)
	}
}

// TestDecryptDryRun_SymmetricKeyValidation tests that symmetric key decryption is still validated with --dry-run.
func TestDecryptDryRun_SymmetricKeyValidation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-dry-symkey-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize project.
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create and encrypt a .env file.
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable
	if err := os.WriteFile(envPath, []byte("KEY=value\n"), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Encrypt the file.
	_, err = shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt file for test setup: %v", err)
	}

	// Remove the .env file.
	if err := os.Remove(envPath); err != nil {
		t.Fatalf("Failed to remove .env file: %v", err)
	}

	// Get user UUID to corrupt their .kanuka key file.
	userUUID := shared.GetUserUUID(t)
	kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", userUUID+".kanuka")

	// Corrupt the kanuka key file to simulate user without access.
	if err := os.WriteFile(kanukaKeyPath, []byte("corrupted key data"), 0600); err != nil {
		t.Fatalf("Failed to corrupt kanuka key file: %v", err)
	}

	// Run decrypt with --dry-run - should fail due to key validation.
	output, err := shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("decrypt", []string{"--dry-run"}, nil, nil, true, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Command should not return error: %v", err)
	}

	// Should show error about decrypting the kanuka file, not dry-run output.
	if !strings.Contains(output, "Failed to decrypt") {
		t.Errorf("Output should contain 'Failed to decrypt', got: %s", output)
	}

	// Verify .env was NOT created.
	if _, err := os.Stat(envPath); !os.IsNotExist(err) {
		t.Error(".env file should NOT be created when key validation fails")
	}
}

// TestDecryptDryRun_WithSubdirectories tests --dry-run with .kanuka files in subdirectories.
func TestDecryptDryRun_WithSubdirectories(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-dry-subdirs-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize project.
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create .env files in various subdirectories.
	envFiles := map[string]string{
		".env":                   "ROOT=value\n",
		"services/api/.env":      "API=value\n",
		"config/.env.production": "PROD=value\n",
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

	// Encrypt all files.
	_, err = shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt files for test setup: %v", err)
	}

	// Remove all .env files.
	for filePath := range envFiles {
		fullPath := filepath.Join(tempDir, filePath)
		if err := os.Remove(fullPath); err != nil {
			t.Fatalf("Failed to remove .env file %s: %v", fullPath, err)
		}
	}

	// Run decrypt with --dry-run.
	output, err := shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("decrypt", []string{"--dry-run"}, nil, nil, true, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Dry-run command should not return error: %v", err)
	}

	// Verify output shows correct count.
	if !strings.Contains(output, "3 encrypted file(s)") {
		t.Errorf("Output should contain '3 encrypted file(s)', got: %s", output)
	}

	// Verify output contains subdirectory paths.
	if !strings.Contains(output, "services/api/.env") {
		t.Errorf("Output should contain 'services/api/.env', got: %s", output)
	}
	if !strings.Contains(output, "config/.env.production") {
		t.Errorf("Output should contain 'config/.env.production', got: %s", output)
	}

	// Verify no .env files were created.
	for filePath := range envFiles {
		fullPath := filepath.Join(tempDir, filePath)
		if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
			t.Errorf("%s should NOT be created after dry-run", filePath)
		}
	}
}
