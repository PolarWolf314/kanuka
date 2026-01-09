package revoke

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PolarWolf314/kanuka/cmd"
	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// addTestDevice adds a device to the project config for testing purposes.
func addTestDevice(projectConfig *configs.ProjectConfig, uuid, email, deviceName string) {
	if projectConfig.Users == nil {
		projectConfig.Users = make(map[string]string)
	}
	if projectConfig.Devices == nil {
		projectConfig.Devices = make(map[string]configs.DeviceConfig)
	}
	projectConfig.Users[uuid] = email
	projectConfig.Devices[uuid] = configs.DeviceConfig{
		Email:     email,
		Name:      deviceName,
		CreatedAt: time.Now(),
	}
}

func TestRevokeDryRun_PreviewsWithoutDeleting(t *testing.T) {
	// Setup test environment.
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

	// Initialize project.
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Get the actual user UUID from config.
	userUUID := shared.GetUserUUID(t)

	// Get project paths.
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	// Create a second user to revoke.
	testUser2UUID := "test-user-2-uuid-for-revoke"
	testUser2Email := "revoke-test@example.com"
	publicKeyPath := filepath.Join(publicKeysDir, testUser2UUID+".pub")
	kanukaKeyPath := filepath.Join(secretsDir, testUser2UUID+".kanuka")

	// Create dummy files for the second user.
	if err := os.WriteFile(publicKeyPath, []byte("dummy public key"), 0600); err != nil {
		t.Fatalf("Failed to create public key file: %v", err)
	}
	if err := os.WriteFile(kanukaKeyPath, []byte("dummy kanuka key"), 0600); err != nil {
		t.Fatalf("Failed to create kanuka key file: %v", err)
	}

	// Add second user to project config.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	addTestDevice(projectConfig, testUser2UUID, testUser2Email, "test-device")
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	// Verify files exist before dry-run.
	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		t.Fatal("Public key file should exist before dry-run")
	}
	if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
		t.Fatal("Kanuka key file should exist before dry-run")
	}

	// Run revoke with --dry-run.
	output, err := shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("revoke", []string{"--user", testUser2Email, "--dry-run"}, nil, nil, false, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Dry-run command should not return error: %v", err)
	}

	// Verify output contains expected dry-run messages.
	if !strings.Contains(output, "[dry-run]") {
		t.Errorf("Output should contain '[dry-run]' prefix, got: %s", output)
	}
	if !strings.Contains(output, "Would revoke access for") {
		t.Errorf("Output should contain 'Would revoke access for', got: %s", output)
	}
	if !strings.Contains(output, "Files that would be deleted") {
		t.Errorf("Output should contain 'Files that would be deleted', got: %s", output)
	}
	if !strings.Contains(output, "No changes made") {
		t.Errorf("Output should contain 'No changes made', got: %s", output)
	}

	// Verify files still exist after dry-run (not deleted).
	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		t.Error("Public key file should still exist after dry-run")
	}
	if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
		t.Error("Kanuka key file should still exist after dry-run")
	}

	// Verify user still in project config.
	projectConfig, err = configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to reload project config: %v", err)
	}
	if _, exists := projectConfig.Users[testUser2UUID]; !exists {
		t.Error("User should still be in project config after dry-run")
	}

	// Verify current user's files are untouched.
	currentUserPublicKey := filepath.Join(publicKeysDir, userUUID+".pub")
	if _, err := os.Stat(currentUserPublicKey); os.IsNotExist(err) {
		t.Error("Current user's public key should still exist")
	}
}

func TestRevokeDryRun_ShowsConfigChanges(t *testing.T) {
	// Setup test environment.
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

	// Initialize project.
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Get project paths.
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	// Create a second user to revoke.
	testUser2UUID := "config-change-test-uuid"
	testUser2Email := "config-test@example.com"
	publicKeyPath := filepath.Join(publicKeysDir, testUser2UUID+".pub")
	kanukaKeyPath := filepath.Join(secretsDir, testUser2UUID+".kanuka")

	// Create dummy files.
	if err := os.WriteFile(publicKeyPath, []byte("dummy public key"), 0600); err != nil {
		t.Fatalf("Failed to create public key file: %v", err)
	}
	if err := os.WriteFile(kanukaKeyPath, []byte("dummy kanuka key"), 0600); err != nil {
		t.Fatalf("Failed to create kanuka key file: %v", err)
	}

	// Add user to project config.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	addTestDevice(projectConfig, testUser2UUID, testUser2Email, "test-device")
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	// Run revoke with --dry-run.
	output, err := shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("revoke", []string{"--user", testUser2Email, "--dry-run"}, nil, nil, false, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Dry-run command should not return error: %v", err)
	}

	// Verify output shows config changes.
	if !strings.Contains(output, "Config changes") {
		t.Errorf("Output should contain 'Config changes', got: %s", output)
	}
	if !strings.Contains(output, testUser2UUID) {
		t.Errorf("Output should contain the UUID being removed, got: %s", output)
	}
}

func TestRevokeDryRun_ShowsKeyRotationImpact(t *testing.T) {
	// Setup test environment.
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

	// Initialize project (creates first user).
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Get project paths.
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	// Create a second user to revoke.
	testUser2UUID := "key-rotation-test-uuid"
	testUser2Email := "rotation-test@example.com"
	publicKeyPath := filepath.Join(publicKeysDir, testUser2UUID+".pub")
	kanukaKeyPath := filepath.Join(secretsDir, testUser2UUID+".kanuka")

	// Create dummy files.
	if err := os.WriteFile(publicKeyPath, []byte("dummy public key"), 0600); err != nil {
		t.Fatalf("Failed to create public key file: %v", err)
	}
	if err := os.WriteFile(kanukaKeyPath, []byte("dummy kanuka key"), 0600); err != nil {
		t.Fatalf("Failed to create kanuka key file: %v", err)
	}

	// Add user to project config.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	addTestDevice(projectConfig, testUser2UUID, testUser2Email, "test-device")
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	// Run revoke with --dry-run.
	output, err := shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("revoke", []string{"--user", testUser2Email, "--dry-run"}, nil, nil, false, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Dry-run command should not return error: %v", err)
	}

	// Verify output shows key rotation impact (should show 1 remaining user).
	if !strings.Contains(output, "Post-revocation actions") {
		t.Errorf("Output should contain 'Post-revocation actions', got: %s", output)
	}
	if !strings.Contains(output, "Re-encrypt symmetric key") {
		t.Errorf("Output should contain re-encryption message, got: %s", output)
	}
	if !strings.Contains(output, "1 remaining user") {
		t.Errorf("Output should mention 1 remaining user, got: %s", output)
	}
}

func TestRevokeDryRun_WorksWithFileFlag(t *testing.T) {
	// Setup test environment.
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

	// Initialize project.
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Get project paths.
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	// Create a second user to revoke.
	testUser2UUID := "file-flag-test-uuid"
	publicKeyPath := filepath.Join(publicKeysDir, testUser2UUID+".pub")
	kanukaKeyPath := filepath.Join(secretsDir, testUser2UUID+".kanuka")

	// Create dummy files.
	if err := os.WriteFile(publicKeyPath, []byte("dummy public key"), 0600); err != nil {
		t.Fatalf("Failed to create public key file: %v", err)
	}
	if err := os.WriteFile(kanukaKeyPath, []byte("dummy kanuka key"), 0600); err != nil {
		t.Fatalf("Failed to create kanuka key file: %v", err)
	}

	// Run revoke with --file and --dry-run.
	relativeKanukaKeyPath := filepath.Join(".kanuka", "secrets", testUser2UUID+".kanuka")
	output, err := shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("revoke", []string{"--file", relativeKanukaKeyPath, "--dry-run"}, nil, nil, false, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Dry-run command with --file should not return error: %v", err)
	}

	// Verify output contains dry-run messages.
	if !strings.Contains(output, "[dry-run]") {
		t.Errorf("Output should contain '[dry-run]' prefix, got: %s", output)
	}
	if !strings.Contains(output, "No changes made") {
		t.Errorf("Output should contain 'No changes made', got: %s", output)
	}

	// Verify files still exist after dry-run.
	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		t.Error("Public key file should still exist after dry-run")
	}
	if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
		t.Error("Kanuka key file should still exist after dry-run")
	}
}

func TestRevokeDryRun_WithYesFlagStillShowsPreview(t *testing.T) {
	// Setup test environment.
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

	// Initialize project.
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Get project paths.
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	// Create a second user to revoke.
	testUser2UUID := "yes-flag-test-uuid"
	testUser2Email := "yes-test@example.com"
	publicKeyPath := filepath.Join(publicKeysDir, testUser2UUID+".pub")
	kanukaKeyPath := filepath.Join(secretsDir, testUser2UUID+".kanuka")

	// Create dummy files.
	if err := os.WriteFile(publicKeyPath, []byte("dummy public key"), 0600); err != nil {
		t.Fatalf("Failed to create public key file: %v", err)
	}
	if err := os.WriteFile(kanukaKeyPath, []byte("dummy kanuka key"), 0600); err != nil {
		t.Fatalf("Failed to create kanuka key file: %v", err)
	}

	// Add user to project config.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	addTestDevice(projectConfig, testUser2UUID, testUser2Email, "test-device")
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	// Run revoke with --dry-run AND --yes.
	output, err := shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("revoke", []string{"--user", testUser2Email, "--dry-run", "--yes"}, nil, nil, false, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Dry-run command with --yes should not return error: %v", err)
	}

	// Verify output still shows dry-run preview (--yes doesn't skip dry-run).
	if !strings.Contains(output, "[dry-run]") {
		t.Errorf("Output should contain '[dry-run]' prefix even with --yes, got: %s", output)
	}
	if !strings.Contains(output, "No changes made") {
		t.Errorf("Output should contain 'No changes made', got: %s", output)
	}

	// Verify files still exist (--yes should not cause deletion in dry-run mode).
	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		t.Error("Public key file should still exist after dry-run with --yes")
	}
	if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
		t.Error("Kanuka key file should still exist after dry-run with --yes")
	}
}

func TestRevokeDryRun_ValidationStillRuns(t *testing.T) {
	// Setup test environment.
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

	// Initialize project.
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Get project paths.
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	// Create a test user that should NOT be revoked (to verify validation prevents action).
	testUserUUID := "validation-test-uuid"
	testUserEmail := "validation-test@example.com"
	publicKeyPath := filepath.Join(publicKeysDir, testUserUUID+".pub")
	kanukaKeyPath := filepath.Join(secretsDir, testUserUUID+".kanuka")

	// Create dummy files.
	if err := os.WriteFile(publicKeyPath, []byte("dummy public key"), 0600); err != nil {
		t.Fatalf("Failed to create public key file: %v", err)
	}
	if err := os.WriteFile(kanukaKeyPath, []byte("dummy kanuka key"), 0600); err != nil {
		t.Fatalf("Failed to create kanuka key file: %v", err)
	}

	// Add user to project config.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	addTestDevice(projectConfig, testUserUUID, testUserEmail, "test-device")
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	// Test 1: Invalid email format should still be rejected with --dry-run.
	// Command completes without error but doesn't show dry-run output because validation failed.
	_, err = shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("revoke", []string{"--user", "invalid-email", "--dry-run"}, nil, nil, false, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Command should not return error: %v", err)
	}

	// Verify original user's files are NOT touched (validation prevented action).
	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		t.Error("Files should still exist after validation failure with --dry-run")
	}

	// Test 2: User not found should also prevent any action with --dry-run.
	_, err = shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("revoke", []string{"--user", "nonexistent@example.com", "--dry-run"}, nil, nil, false, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Command should not return error: %v", err)
	}

	// Verify original user's files are still NOT touched.
	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		t.Error("Files should still exist after user-not-found with --dry-run")
	}
	if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
		t.Error("Kanuka key file should still exist after user-not-found with --dry-run")
	}
}

func TestRevokeDryRun_ShowsGitHistoryWarning(t *testing.T) {
	// Setup test environment.
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

	// Initialize project.
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Get project paths.
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	// Create a second user to revoke.
	testUser2UUID := "git-warning-test-uuid"
	testUser2Email := "git-warning@example.com"
	publicKeyPath := filepath.Join(publicKeysDir, testUser2UUID+".pub")
	kanukaKeyPath := filepath.Join(secretsDir, testUser2UUID+".kanuka")

	// Create dummy files.
	if err := os.WriteFile(publicKeyPath, []byte("dummy public key"), 0600); err != nil {
		t.Fatalf("Failed to create public key file: %v", err)
	}
	if err := os.WriteFile(kanukaKeyPath, []byte("dummy kanuka key"), 0600); err != nil {
		t.Fatalf("Failed to create kanuka key file: %v", err)
	}

	// Add user to project config.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	addTestDevice(projectConfig, testUser2UUID, testUser2Email, "test-device")
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	// Run revoke with --dry-run.
	output, err := shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("revoke", []string{"--user", testUser2Email, "--dry-run"}, nil, nil, false, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Dry-run command should not return error: %v", err)
	}

	// Verify output contains git history warning.
	if !strings.Contains(output, "git history") {
		t.Errorf("Output should contain git history warning, got: %s", output)
	}
}
