package encrypt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/cmd"
	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestEncryptDryRun_PreviewsWithoutCreating tests that --dry-run shows preview without creating files.
func TestEncryptDryRun_PreviewsWithoutCreating(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-dry-*")
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

	// Create a .env file.
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\nAPI_KEY=secret123\n"
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Verify .env.kanuka does NOT exist before dry-run.
	kanukaPath := envPath + ".kanuka"
	if _, err := os.Stat(kanukaPath); !os.IsNotExist(err) {
		t.Fatal(".env.kanuka should not exist before dry-run")
	}

	// Run encrypt with --dry-run.
	output, err := shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("encrypt", []string{"--dry-run"}, nil, nil, false, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Dry-run command should not return error: %v", err)
	}

	// Verify output contains expected dry-run messages.
	if !strings.Contains(output, "[dry-run]") {
		t.Errorf("Output should contain '[dry-run]' prefix, got: %s", output)
	}
	if !strings.Contains(output, "Would encrypt") {
		t.Errorf("Output should contain 'Would encrypt', got: %s", output)
	}
	if !strings.Contains(output, "Files that would be created") {
		t.Errorf("Output should contain 'Files that would be created', got: %s", output)
	}
	if !strings.Contains(output, "No changes made") {
		t.Errorf("Output should contain 'No changes made', got: %s", output)
	}

	// Verify .env.kanuka was NOT created.
	if _, err := os.Stat(kanukaPath); !os.IsNotExist(err) {
		t.Error(".env.kanuka file should NOT be created after dry-run")
	}
}

// TestEncryptDryRun_ShowsCorrectFileMapping tests that --dry-run shows source to destination mapping.
func TestEncryptDryRun_ShowsCorrectFileMapping(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-dry-map-*")
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

	// Create .env file.
	envContent := "API_KEY=secret\n"
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Run encrypt with --dry-run.
	output, err := shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("encrypt", []string{"--dry-run"}, nil, nil, false, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Dry-run command should not return error: %v", err)
	}

	// Verify output shows correct file mapping (source -> destination).
	if !strings.Contains(output, ".env") {
		t.Errorf("Output should contain '.env', got: %s", output)
	}
	if !strings.Contains(output, ".env.kanuka") {
		t.Errorf("Output should contain '.env.kanuka', got: %s", output)
	}
	// Check for the arrow showing the mapping.
	if !strings.Contains(output, "→") {
		t.Errorf("Output should contain '→' showing mapping, got: %s", output)
	}
}

// TestEncryptDryRun_ShowsCorrectFileCount tests that --dry-run shows the correct file count.
func TestEncryptDryRun_ShowsCorrectFileCount(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-dry-count-*")
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
		".env":                   "DATABASE_URL=postgres://localhost:5432/mydb\n",
		".env.local":             "API_KEY=secret123\n",
		"config/.env.production": "PROD_API_KEY=prod_secret\n",
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

	// Run encrypt with --dry-run.
	output, err := shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("encrypt", []string{"--dry-run"}, nil, nil, false, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Dry-run command should not return error: %v", err)
	}

	// Verify output shows correct count (3 files).
	if !strings.Contains(output, "3 environment file(s)") {
		t.Errorf("Output should contain '3 environment file(s)', got: %s", output)
	}

	// Verify none of the .kanuka files were created.
	for filePath := range envFiles {
		kanukaPath := filepath.Join(tempDir, filePath+".kanuka")
		if _, err := os.Stat(kanukaPath); !os.IsNotExist(err) {
			t.Errorf("%s.kanuka should NOT be created after dry-run", filePath)
		}
	}
}

// TestEncryptDryRun_NotInitialized tests that validation errors occur with --dry-run when project not initialized.
func TestEncryptDryRun_NotInitialized(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-dry-uninit-*")
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
		testCmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Command should not return error: %v", err)
	}

	// Should show "not initialized" message, not dry-run output.
	if !strings.Contains(output, "not been initialized") {
		t.Errorf("Output should contain 'not been initialized', got: %s", output)
	}
}

// TestEncryptDryRun_NoEnvFiles tests that validation errors occur with --dry-run when no env files exist.
func TestEncryptDryRun_NoEnvFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-dry-noenv-*")
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

	// Initialize project but don't create any .env files.
	shared.InitializeProject(t, tempDir, tempUserDir)

	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Command should not return error: %v", err)
	}

	// Should show "no environment files" message.
	if !strings.Contains(output, "No environment files found") {
		t.Errorf("Output should contain 'No environment files found', got: %s", output)
	}
}

// TestEncryptDryRun_WithMultipleSubdirectories tests that --dry-run works with .env files in subdirectories.
func TestEncryptDryRun_WithMultipleSubdirectories(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-dry-subdirs-*")
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
		".env":                         "ROOT=value\n",
		"services/api/.env":            "API=value\n",
		"services/worker/.env":         "WORKER=value\n",
		"config/.env.production":       "PROD=value\n",
		"deep/nested/path/.env.secret": "SECRET=value\n",
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

	// Run encrypt with --dry-run.
	output, err := shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("encrypt", []string{"--dry-run"}, nil, nil, false, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Dry-run command should not return error: %v", err)
	}

	// Verify output shows all files would be encrypted.
	if !strings.Contains(output, "5 environment file(s)") {
		t.Errorf("Output should contain '5 environment file(s)', got: %s", output)
	}

	// Verify output contains some of the subdirectory paths.
	if !strings.Contains(output, "services/api/.env") {
		t.Errorf("Output should contain 'services/api/.env', got: %s", output)
	}
	if !strings.Contains(output, "deep/nested/path/.env.secret") {
		t.Errorf("Output should contain 'deep/nested/path/.env.secret', got: %s", output)
	}

	// Verify none of the .kanuka files were created.
	for filePath := range envFiles {
		kanukaPath := filepath.Join(tempDir, filePath+".kanuka")
		if _, err := os.Stat(kanukaPath); !os.IsNotExist(err) {
			t.Errorf("%s.kanuka should NOT be created after dry-run", filePath)
		}
	}
}

// TestEncryptDryRun_SymmetricKeyValidation tests that symmetric key decryption is still validated with --dry-run.
func TestEncryptDryRun_SymmetricKeyValidation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-dry-symkey-*")
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

	// Create a .env file.
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable
	if err := os.WriteFile(envPath, []byte("KEY=value\n"), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Get user UUID to corrupt their .kanuka key file.
	userUUID := shared.GetUserUUID(t)
	kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", userUUID+".kanuka")

	// Corrupt the kanuka key file to simulate user without access.
	if err := os.WriteFile(kanukaKeyPath, []byte("corrupted key data"), 0600); err != nil {
		t.Fatalf("Failed to corrupt kanuka key file: %v", err)
	}

	// Run encrypt with --dry-run - should fail due to key validation.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("encrypt", []string{"--dry-run"}, nil, nil, true, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Command should not return error: %v", err)
	}

	// Should show error about decrypting the kanuka file, not dry-run output.
	if !strings.Contains(output, "Failed to decrypt") {
		t.Errorf("Output should contain 'Failed to decrypt', got: %s", output)
	}

	// Verify .env.kanuka was NOT created.
	envKanukaPath := envPath + ".kanuka"
	if _, err := os.Stat(envKanukaPath); !os.IsNotExist(err) {
		t.Error(".env.kanuka file should NOT be created when key validation fails")
	}
}
