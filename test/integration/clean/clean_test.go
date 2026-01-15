package clean

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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

// addActiveUser adds an active user (has both public key and .kanuka file).
func addActiveUser(t *testing.T, tempDir, uuid string) {
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	// Create public key file.
	publicKeyPath := filepath.Join(publicKeysDir, uuid+".pub")
	if err := os.WriteFile(publicKeyPath, []byte("dummy public key"), 0600); err != nil {
		t.Fatalf("Failed to create public key file: %v", err)
	}

	// Create .kanuka file.
	kanukaPath := filepath.Join(secretsDir, uuid+".kanuka")
	if err := os.WriteFile(kanukaPath, []byte("dummy encrypted key"), 0600); err != nil {
		t.Fatalf("Failed to create kanuka file: %v", err)
	}
}

// addOrphanUser adds an orphan user (has .kanuka file but NO public key).
func addOrphanUser(t *testing.T, tempDir, uuid string) {
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	// Create .kanuka file only.
	kanukaPath := filepath.Join(secretsDir, uuid+".kanuka")
	if err := os.WriteFile(kanukaPath, []byte("dummy encrypted key"), 0600); err != nil {
		t.Fatalf("Failed to create kanuka file: %v", err)
	}
}

// orphanExists checks if an orphan file exists.
func orphanExists(tempDir, uuid string) bool {
	kanukaPath := filepath.Join(tempDir, ".kanuka", "secrets", uuid+".kanuka")
	_, err := os.Stat(kanukaPath)
	return err == nil
}

func TestClean_NoOrphans(t *testing.T) {
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
	// Add an active user (not an orphan).
	addActiveUser(t, tempDir, "uuid-active")

	// Run clean command with --force.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("clean", []string{"--force"}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Clean command failed: %v", err)
	}

	// Verify output.
	if !strings.Contains(output, "No orphaned entries found") {
		t.Errorf("Output should indicate no orphans found, got: %s", output)
	}
}

func TestClean_SingleOrphan(t *testing.T) {
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
	addOrphanUser(t, tempDir, "uuid-orphan")

	// Verify orphan exists before clean.
	if !orphanExists(tempDir, "uuid-orphan") {
		t.Fatalf("Orphan file should exist before clean")
	}

	// Run clean command with --force.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("clean", []string{"--force"}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Clean command failed: %v", err)
	}

	// Verify output.
	if !strings.Contains(output, "✓ Removed 1 orphaned file") {
		t.Errorf("Output should indicate 1 file removed, got: %s", output)
	}

	// Verify orphan was removed.
	if orphanExists(tempDir, "uuid-orphan") {
		t.Errorf("Orphan file should have been removed")
	}
}

func TestClean_MultipleOrphans(t *testing.T) {
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
	addOrphanUser(t, tempDir, "uuid-orphan-1")
	addOrphanUser(t, tempDir, "uuid-orphan-2")
	addOrphanUser(t, tempDir, "uuid-orphan-3")

	// Run clean command with --force.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("clean", []string{"--force"}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Clean command failed: %v", err)
	}

	// Verify output.
	if !strings.Contains(output, "✓ Removed 3 orphaned file") {
		t.Errorf("Output should indicate 3 files removed, got: %s", output)
	}

	// Verify all orphans were removed.
	if orphanExists(tempDir, "uuid-orphan-1") {
		t.Errorf("Orphan 1 should have been removed")
	}
	if orphanExists(tempDir, "uuid-orphan-2") {
		t.Errorf("Orphan 2 should have been removed")
	}
	if orphanExists(tempDir, "uuid-orphan-3") {
		t.Errorf("Orphan 3 should have been removed")
	}
}

func TestClean_DryRun(t *testing.T) {
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
	addOrphanUser(t, tempDir, "uuid-orphan")

	// Run clean command with --dry-run.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("clean", []string{"--dry-run"}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Clean command failed: %v", err)
	}

	// Verify output indicates dry run.
	if !strings.Contains(output, "[dry-run]") {
		t.Errorf("Output should indicate dry-run, got: %s", output)
	}
	if !strings.Contains(output, "Would remove 1 orphaned file") {
		t.Errorf("Output should indicate would remove 1 file, got: %s", output)
	}
	if !strings.Contains(output, "No changes made") {
		t.Errorf("Output should indicate no changes made, got: %s", output)
	}

	// Verify orphan still exists (not deleted).
	if !orphanExists(tempDir, "uuid-orphan") {
		t.Errorf("Orphan file should NOT have been removed in dry-run mode")
	}
}

func TestClean_MixedUsersOnlyRemovesOrphans(t *testing.T) {
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
	addActiveUser(t, tempDir, "uuid-active")
	addOrphanUser(t, tempDir, "uuid-orphan")

	// Run clean command with --force.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("clean", []string{"--force"}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Clean command failed: %v", err)
	}

	// Verify output.
	if !strings.Contains(output, "✓ Removed 1 orphaned file") {
		t.Errorf("Output should indicate 1 file removed, got: %s", output)
	}

	// Verify only orphan was removed, active user remains.
	if orphanExists(tempDir, "uuid-orphan") {
		t.Errorf("Orphan file should have been removed")
	}

	// Check active user's .kanuka file still exists.
	activeKanukaPath := filepath.Join(tempDir, ".kanuka", "secrets", "uuid-active.kanuka")
	if _, err := os.Stat(activeKanukaPath); os.IsNotExist(err) {
		t.Errorf("Active user's .kanuka file should NOT have been removed")
	}
}

func TestClean_NotInitialized(t *testing.T) {
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

	// Run clean command.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("clean", []string{"--force"}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Clean command failed: %v", err)
	}

	// Verify error message.
	if !strings.Contains(output, "✗ Kanuka has not been initialized") {
		t.Errorf("Output should indicate project not initialized, got: %s", output)
	}
	if !strings.Contains(output, "kanuka secrets init") {
		t.Errorf("Output should suggest running 'kanuka secrets init', got: %s", output)
	}
}

func TestClean_InteractiveConfirmYes(t *testing.T) {
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
	addOrphanUser(t, tempDir, "uuid-orphan")

	// Run clean command without --force, provide "y" on stdin.
	output, err := shared.CaptureOutputWithStdin([]byte("y\n"), func() error {
		testCmd := shared.CreateTestCLIWithArgs("clean", []string{}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Clean command failed: %v", err)
	}

	// Verify prompt was shown.
	if !strings.Contains(output, "Do you want to continue?") {
		t.Errorf("Output should contain confirmation prompt, got: %s", output)
	}

	// Verify orphan was removed.
	if orphanExists(tempDir, "uuid-orphan") {
		t.Errorf("Orphan file should have been removed after confirming 'y'")
	}
}

func TestClean_InteractiveConfirmNo(t *testing.T) {
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
	addOrphanUser(t, tempDir, "uuid-orphan")

	// Run clean command without --force, provide "n" on stdin.
	output, err := shared.CaptureOutputWithStdin([]byte("n\n"), func() error {
		testCmd := shared.CreateTestCLIWithArgs("clean", []string{}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Clean command failed: %v", err)
	}

	// Verify abort message.
	if !strings.Contains(output, "Aborted") {
		t.Errorf("Output should indicate aborted, got: %s", output)
	}

	// Verify orphan was NOT removed.
	if !orphanExists(tempDir, "uuid-orphan") {
		t.Errorf("Orphan file should NOT have been removed after declining")
	}
}

func TestClean_ShowsRelativePaths(t *testing.T) {
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
	addOrphanUser(t, tempDir, "uuid-orphan")

	// Run clean command with --dry-run.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("clean", []string{"--dry-run"}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Clean command failed: %v", err)
	}

	// Verify relative path is shown (not absolute path).
	if !strings.Contains(output, ".kanuka/secrets/uuid-orphan.kanuka") {
		t.Errorf("Output should show relative path, got: %s", output)
	}
	// Should NOT contain the temp directory absolute path in the file column.
	if strings.Contains(output, tempDir+"/.kanuka") {
		t.Errorf("Output should NOT show absolute path in file column, got: %s", output)
	}
}
