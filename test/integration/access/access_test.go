package access

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

// addActiveUser adds an active user (has both public key and .kanuka file).
func addActiveUser(t *testing.T, tempDir, uuid, email, deviceName string) {
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

	// Add to project config.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectConfig.Users[uuid] = email
	projectConfig.Devices[uuid] = configs.DeviceConfig{
		Email:     email,
		Name:      deviceName,
		CreatedAt: time.Now(),
	}
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}
}

// addPendingUser adds a pending user (has public key but NO .kanuka file).
func addPendingUser(t *testing.T, tempDir, uuid, email, deviceName string) {
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")

	// Create public key file only.
	publicKeyPath := filepath.Join(publicKeysDir, uuid+".pub")
	if err := os.WriteFile(publicKeyPath, []byte("dummy public key"), 0600); err != nil {
		t.Fatalf("Failed to create public key file: %v", err)
	}

	// Add to project config.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectConfig.Users[uuid] = email
	projectConfig.Devices[uuid] = configs.DeviceConfig{
		Email:     email,
		Name:      deviceName,
		CreatedAt: time.Now(),
	}
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
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

	// Orphan users are NOT added to project config (they're inconsistent state).
}

func TestAccess_SingleActiveUser(t *testing.T) {
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
	addActiveUser(t, tempDir, shared.TestUserUUID, "alice@example.com", "laptop")

	// Run access command.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("access", []string{}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Access command failed: %v", err)
	}

	// Verify output.
	if !strings.Contains(output, "alice@example.com") {
		t.Errorf("Output should contain email, got: %s", output)
	}
	if !strings.Contains(output, "active") {
		t.Errorf("Output should contain 'active' status, got: %s", output)
	}
	if !strings.Contains(output, "Total: 1 user(s)") {
		t.Errorf("Output should show 1 user total, got: %s", output)
	}
}

func TestAccess_MultipleUsers(t *testing.T) {
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
	addActiveUser(t, tempDir, "uuid-alice", "alice@example.com", "laptop")
	addActiveUser(t, tempDir, "uuid-bob", "bob@example.com", "desktop")
	addPendingUser(t, tempDir, "uuid-charlie", "charlie@example.com", "tablet")
	addOrphanUser(t, tempDir, "uuid-orphan")

	// Run access command.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("access", []string{}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Access command failed: %v", err)
	}

	// Verify all users are shown.
	if !strings.Contains(output, "alice@example.com") {
		t.Errorf("Output should contain alice, got: %s", output)
	}
	if !strings.Contains(output, "bob@example.com") {
		t.Errorf("Output should contain bob, got: %s", output)
	}
	if !strings.Contains(output, "charlie@example.com") {
		t.Errorf("Output should contain charlie, got: %s", output)
	}
	if !strings.Contains(output, "uuid-orphan") {
		t.Errorf("Output should contain orphan UUID, got: %s", output)
	}

	// Verify status counts.
	if !strings.Contains(output, "2 active") {
		t.Errorf("Output should show 2 active users, got: %s", output)
	}
	if !strings.Contains(output, "1 pending") {
		t.Errorf("Output should show 1 pending user, got: %s", output)
	}
	if !strings.Contains(output, "1 orphan") {
		t.Errorf("Output should show 1 orphan user, got: %s", output)
	}

	// Verify clean tip is shown for orphans.
	if !strings.Contains(output, "kanuka secrets clean") {
		t.Errorf("Output should suggest 'kanuka secrets clean' for orphans, got: %s", output)
	}
}

func TestAccess_PendingUser(t *testing.T) {
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
	addPendingUser(t, tempDir, "uuid-pending", "pending@example.com", "device1")

	// Run access command.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("access", []string{}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Access command failed: %v", err)
	}

	// Verify pending status.
	if !strings.Contains(output, "pending") {
		t.Errorf("Output should contain 'pending' status, got: %s", output)
	}
	if !strings.Contains(output, "pending@example.com") {
		t.Errorf("Output should contain email, got: %s", output)
	}
}

func TestAccess_OrphanUser(t *testing.T) {
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
	addOrphanUser(t, tempDir, "uuid-orphan-only")

	// Run access command.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("access", []string{}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Access command failed: %v", err)
	}

	// Verify orphan status.
	if !strings.Contains(output, "orphan") {
		t.Errorf("Output should contain 'orphan' status, got: %s", output)
	}
	if !strings.Contains(output, "(unknown)") {
		t.Errorf("Output should show '(unknown)' for orphan without email, got: %s", output)
	}
	if !strings.Contains(output, "kanuka secrets clean") {
		t.Errorf("Output should suggest 'kanuka secrets clean', got: %s", output)
	}
}

func TestAccess_JSONOutput(t *testing.T) {
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
	addActiveUser(t, tempDir, "uuid-alice", "alice@example.com", "laptop")
	addPendingUser(t, tempDir, "uuid-bob", "bob@example.com", "desktop")

	// Run access command with --json.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("access", []string{"--json"}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Access command failed: %v", err)
	}

	// Verify output is valid JSON.
	var result struct {
		Project string `json:"project"`
		Users   []struct {
			UUID       string `json:"uuid"`
			Email      string `json:"email"`
			DeviceName string `json:"device_name"`
			Status     string `json:"status"`
		} `json:"users"`
		Summary struct {
			Active  int `json:"active"`
			Pending int `json:"pending"`
			Orphan  int `json:"orphan"`
		} `json:"summary"`
	}

	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, output)
	}

	// Verify JSON content.
	if result.Project != "test-project" {
		t.Errorf("Expected project 'test-project', got: %s", result.Project)
	}
	if len(result.Users) != 2 {
		t.Errorf("Expected 2 users, got: %d", len(result.Users))
	}
	if result.Summary.Active != 1 {
		t.Errorf("Expected 1 active user, got: %d", result.Summary.Active)
	}
	if result.Summary.Pending != 1 {
		t.Errorf("Expected 1 pending user, got: %d", result.Summary.Pending)
	}
}

func TestAccess_NotInitialized(t *testing.T) {
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

	// Run access command.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("access", []string{}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Access command failed: %v", err)
	}

	// Verify error message.
	if !strings.Contains(output, "not been initialized") {
		t.Errorf("Output should indicate project not initialized, got: %s", output)
	}
	if !strings.Contains(output, "kanuka secrets init") {
		t.Errorf("Output should suggest running 'kanuka secrets init', got: %s", output)
	}
}

func TestAccess_NoUsers(t *testing.T) {
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
	// Do NOT add any users.

	// Run access command.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("access", []string{}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Access command failed: %v", err)
	}

	// Verify empty output.
	if !strings.Contains(output, "No users found") {
		t.Errorf("Output should indicate no users found, got: %s", output)
	}
}
