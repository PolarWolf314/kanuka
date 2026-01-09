package status

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// setupTestProject creates a minimal kanuka project structure for testing.
func setupTestProject(t *testing.T, tempDir string) {
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	if err := os.MkdirAll(publicKeysDir, 0755); err != nil {
		t.Fatalf("Failed to create public keys directory: %v", err)
	}
	if err := os.MkdirAll(secretsDir, 0755); err != nil {
		t.Fatalf("Failed to create secrets directory: %v", err)
	}

	// Create project config.
	projectConfig := &configs.ProjectConfig{
		Project: configs.Project{
			UUID: shared.TestProjectUUID,
			Name: "test-project",
		},
		Users:   make(map[string]string),
		Devices: make(map[string]configs.DeviceConfig),
	}

	configs.ProjectKanukaSettings = &configs.ProjectSettings{
		ProjectName:          "test-project",
		ProjectPath:          tempDir,
		ProjectPublicKeyPath: publicKeysDir,
		ProjectSecretsPath:   secretsDir,
	}

	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}
}

// createEnvFile creates a .env file with the given content.
func createEnvFile(t *testing.T, path string, content string) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create env file %s: %v", path, err)
	}
}

// createKanukaFile creates a .kanuka (encrypted) file with the given content.
func createKanukaFile(t *testing.T, path string, content string) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create kanuka file %s: %v", path, err)
	}
}

// StatusResult mirrors the cmd.StatusResult struct for JSON parsing.
type StatusResult struct {
	ProjectName string           `json:"project"`
	Files       []FileStatusInfo `json:"files"`
	Summary     StatusSummary    `json:"summary"`
}

// FileStatusInfo mirrors the cmd.FileStatusInfo struct for JSON parsing.
type FileStatusInfo struct {
	Path           string `json:"path"`
	Status         string `json:"status"`
	PlaintextMtime string `json:"plaintext_mtime,omitempty"`
	EncryptedMtime string `json:"encrypted_mtime,omitempty"`
}

// StatusSummary mirrors the cmd.StatusSummary struct for JSON parsing.
type StatusSummary struct {
	Current       int `json:"current"`
	Stale         int `json:"stale"`
	Unencrypted   int `json:"unencrypted"`
	EncryptedOnly int `json:"encrypted_only"`
}

func TestStatus_AllCurrent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
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

	setupTestProject(t, tempDir)

	// Create .env file first.
	envPath := filepath.Join(tempDir, ".env")
	createEnvFile(t, envPath, "SECRET=value")

	// Wait a bit, then create .kanuka file (newer).
	time.Sleep(50 * time.Millisecond)
	kanukaPath := filepath.Join(tempDir, ".env.kanuka")
	createKanukaFile(t, kanukaPath, "encrypted-data")

	// Run status command.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("status", []string{}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Status command failed: %v", err)
	}

	// Verify output.
	if !strings.Contains(output, "encrypted (up to date)") {
		t.Errorf("Output should show 'encrypted (up to date)', got: %s", output)
	}
	if !strings.Contains(output, "1 file(s) up to date") {
		t.Errorf("Output should show '1 file(s) up to date' in summary, got: %s", output)
	}
}

func TestStatus_StaleFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
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

	setupTestProject(t, tempDir)

	// Create .kanuka file first (older).
	kanukaPath := filepath.Join(tempDir, ".env.kanuka")
	createKanukaFile(t, kanukaPath, "encrypted-data")

	// Wait a bit, then create .env file (newer - stale).
	time.Sleep(50 * time.Millisecond)
	envPath := filepath.Join(tempDir, ".env")
	createEnvFile(t, envPath, "SECRET=updated-value")

	// Run status command.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("status", []string{}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Status command failed: %v", err)
	}

	// Verify output.
	if !strings.Contains(output, "stale") {
		t.Errorf("Output should show 'stale', got: %s", output)
	}
	if !strings.Contains(output, "1 file(s) stale") {
		t.Errorf("Output should show '1 file(s) stale' in summary, got: %s", output)
	}
	if !strings.Contains(output, "kanuka secrets encrypt") {
		t.Errorf("Output should suggest running 'kanuka secrets encrypt', got: %s", output)
	}
}

func TestStatus_UnencryptedFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
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

	setupTestProject(t, tempDir)

	// Create only .env file (no .kanuka).
	envPath := filepath.Join(tempDir, ".env")
	createEnvFile(t, envPath, "SECRET=unencrypted")

	// Run status command.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("status", []string{}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Status command failed: %v", err)
	}

	// Verify output.
	if !strings.Contains(output, "not encrypted") {
		t.Errorf("Output should show 'not encrypted', got: %s", output)
	}
	if !strings.Contains(output, "1 file(s) not encrypted") {
		t.Errorf("Output should show '1 file(s) not encrypted' in summary, got: %s", output)
	}
}

func TestStatus_EncryptedOnlyFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
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

	setupTestProject(t, tempDir)

	// Create only .kanuka file (no .env - encrypted only).
	kanukaPath := filepath.Join(tempDir, ".env.kanuka")
	createKanukaFile(t, kanukaPath, "encrypted-data")

	// Run status command.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("status", []string{}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Status command failed: %v", err)
	}

	// Verify output.
	if !strings.Contains(output, "encrypted only") {
		t.Errorf("Output should show 'encrypted only', got: %s", output)
	}
	if !strings.Contains(output, "1 file(s) encrypted only") {
		t.Errorf("Output should show '1 file(s) encrypted only' in summary, got: %s", output)
	}
}

func TestStatus_MixedStates(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
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

	setupTestProject(t, tempDir)

	// 1. Current file: .env first, then .kanuka (newer).
	envPath1 := filepath.Join(tempDir, ".env")
	createEnvFile(t, envPath1, "SECRET=value")
	time.Sleep(50 * time.Millisecond)
	kanukaPath1 := filepath.Join(tempDir, ".env.kanuka")
	createKanukaFile(t, kanukaPath1, "encrypted-data")

	// 2. Stale file: .kanuka first, then .env (newer).
	time.Sleep(50 * time.Millisecond)
	kanukaPath2 := filepath.Join(tempDir, ".env.local.kanuka")
	createKanukaFile(t, kanukaPath2, "encrypted-data")
	time.Sleep(50 * time.Millisecond)
	envPath2 := filepath.Join(tempDir, ".env.local")
	createEnvFile(t, envPath2, "SECRET=stale")

	// 3. Unencrypted file: only .env.
	time.Sleep(50 * time.Millisecond)
	envPath3 := filepath.Join(tempDir, ".env.production")
	createEnvFile(t, envPath3, "SECRET=unencrypted")

	// 4. Encrypted only file: only .kanuka.
	time.Sleep(50 * time.Millisecond)
	kanukaPath4 := filepath.Join(tempDir, ".env.backup.kanuka")
	createKanukaFile(t, kanukaPath4, "encrypted-only")

	// Run status command with --json for easier parsing.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("status", []string{"--json"}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Status command failed: %v", err)
	}

	// Parse JSON output.
	var result StatusResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, output)
	}

	// Verify summary counts.
	if result.Summary.Current != 1 {
		t.Errorf("Expected 1 current file, got %d", result.Summary.Current)
	}
	if result.Summary.Stale != 1 {
		t.Errorf("Expected 1 stale file, got %d", result.Summary.Stale)
	}
	if result.Summary.Unencrypted != 1 {
		t.Errorf("Expected 1 unencrypted file, got %d", result.Summary.Unencrypted)
	}
	if result.Summary.EncryptedOnly != 1 {
		t.Errorf("Expected 1 encrypted only file, got %d", result.Summary.EncryptedOnly)
	}

	// Verify total file count.
	if len(result.Files) != 4 {
		t.Errorf("Expected 4 files, got %d", len(result.Files))
	}
}

func TestStatus_Subdirectories(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
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

	setupTestProject(t, tempDir)

	// Create files in subdirectories.
	configDir := filepath.Join(tempDir, "config")
	nestedDir := filepath.Join(tempDir, "config", "nested")

	// Root level.
	envPath1 := filepath.Join(tempDir, ".env")
	createEnvFile(t, envPath1, "SECRET=root")
	time.Sleep(50 * time.Millisecond)
	kanukaPath1 := filepath.Join(tempDir, ".env.kanuka")
	createKanukaFile(t, kanukaPath1, "encrypted")

	// Config subdirectory.
	time.Sleep(50 * time.Millisecond)
	envPath2 := filepath.Join(configDir, ".env.production")
	createEnvFile(t, envPath2, "SECRET=config")

	// Nested subdirectory.
	time.Sleep(50 * time.Millisecond)
	kanukaPath3 := filepath.Join(nestedDir, ".env.test.kanuka")
	createKanukaFile(t, kanukaPath3, "encrypted-nested")

	// Run status command with --json.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("status", []string{"--json"}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Status command failed: %v", err)
	}

	// Parse JSON output.
	var result StatusResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, output)
	}

	// Verify we found all 3 files.
	if len(result.Files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(result.Files))
	}

	// Verify paths are relative.
	pathsFound := make(map[string]bool)
	for _, file := range result.Files {
		pathsFound[file.Path] = true
		// Paths should be relative, not absolute.
		if strings.HasPrefix(file.Path, "/") {
			t.Errorf("Path should be relative, got absolute: %s", file.Path)
		}
	}

	// Check expected paths exist.
	expectedPaths := []string{".env", "config/.env.production", "config/nested/.env.test"}
	for _, expected := range expectedPaths {
		if !pathsFound[expected] {
			t.Errorf("Expected path %s not found in result. Found: %v", expected, pathsFound)
		}
	}
}

func TestStatus_JsonOutput(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
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

	setupTestProject(t, tempDir)

	// Create a test file.
	envPath := filepath.Join(tempDir, ".env")
	createEnvFile(t, envPath, "SECRET=value")
	time.Sleep(50 * time.Millisecond)
	kanukaPath := filepath.Join(tempDir, ".env.kanuka")
	createKanukaFile(t, kanukaPath, "encrypted-data")

	// Run status command with --json.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("status", []string{"--json"}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Status command failed: %v", err)
	}

	// Parse JSON output.
	var result StatusResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, output)
	}

	// Verify JSON structure.
	if result.ProjectName != "test-project" {
		t.Errorf("Expected project name 'test-project', got: %s", result.ProjectName)
	}

	if len(result.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(result.Files))
	}

	if result.Files[0].Path != ".env" {
		t.Errorf("Expected file path '.env', got: %s", result.Files[0].Path)
	}

	if result.Files[0].Status != "current" {
		t.Errorf("Expected status 'current', got: %s", result.Files[0].Status)
	}

	// Verify timestamps are present.
	if result.Files[0].PlaintextMtime == "" {
		t.Errorf("Expected plaintext_mtime to be set")
	}
	if result.Files[0].EncryptedMtime == "" {
		t.Errorf("Expected encrypted_mtime to be set")
	}

	// Verify summary.
	if result.Summary.Current != 1 {
		t.Errorf("Expected summary.current = 1, got: %d", result.Summary.Current)
	}
}

func TestStatus_NoFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
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

	setupTestProject(t, tempDir)
	// Do NOT create any .env or .kanuka files.

	// Run status command.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("status", []string{}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Status command failed: %v", err)
	}

	// Verify output indicates no files.
	if !strings.Contains(output, "No secret files found") {
		t.Errorf("Output should indicate no secret files found, got: %s", output)
	}
}

func TestStatus_NotInitialized(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
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

	// Do NOT set up project - leave it uninitialized.

	// Run status command.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("status", []string{}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Status command failed: %v", err)
	}

	// Verify error message.
	if !strings.Contains(output, "not been initialized") {
		t.Errorf("Output should indicate project not initialized, got: %s", output)
	}
	if !strings.Contains(output, "kanuka secrets init") {
		t.Errorf("Output should suggest running 'kanuka secrets init', got: %s", output)
	}
}

func TestStatus_NotInitializedJsonOutput(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
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

	// Do NOT set up project - leave it uninitialized.

	// Run status command with --json.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("status", []string{"--json"}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Status command failed: %v", err)
	}

	// Verify JSON error output.
	if !strings.Contains(output, `"error"`) {
		t.Errorf("Output should contain JSON error, got: %s", output)
	}
	if !strings.Contains(output, "not been initialized") {
		t.Errorf("Output should indicate project not initialized, got: %s", output)
	}
}

func TestStatus_ShowsRelativePaths(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
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

	setupTestProject(t, tempDir)

	// Create file in subdirectory.
	configDir := filepath.Join(tempDir, "config")
	envPath := filepath.Join(configDir, ".env.production")
	createEnvFile(t, envPath, "SECRET=value")

	// Run status command.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("status", []string{}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Status command failed: %v", err)
	}

	// Verify relative path is shown.
	if !strings.Contains(output, "config/.env.production") {
		t.Errorf("Output should show relative path 'config/.env.production', got: %s", output)
	}
	// Should NOT contain the temp directory absolute path.
	if strings.Contains(output, tempDir) {
		t.Errorf("Output should NOT show absolute path, got: %s", output)
	}
}

func TestStatus_ProjectName(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
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

	setupTestProject(t, tempDir)

	// Create a file to have something to show.
	envPath := filepath.Join(tempDir, ".env")
	createEnvFile(t, envPath, "SECRET=value")

	// Run status command.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("status", []string{}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Status command failed: %v", err)
	}

	// Verify project name is shown (may have quotes in NO_COLOR mode).
	if !strings.Contains(output, "Project: test-project") && !strings.Contains(output, "Project: 'test-project'") {
		t.Errorf("Output should show 'Project: test-project', got: %s", output)
	}
}
