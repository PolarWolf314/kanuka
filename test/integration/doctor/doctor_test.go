package doctor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/cmd"
	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// mockExitCode stores the exit code from the doctor command.
var mockExitCode int

// mockExit is a mock exit function that captures the exit code instead of exiting.
func mockExit(code int) {
	mockExitCode = code
}

// setupMockExit sets up the mock exit function and returns a cleanup function.
func setupMockExit() func() {
	mockExitCode = 0
	cmd.SetDoctorExitFunc(mockExit)
	return func() {
		cmd.SetDoctorExitFunc(os.Exit)
	}
}

// DoctorResult mirrors the cmd.DoctorResult struct for JSON parsing.
type DoctorResult struct {
	Checks      []CheckResult `json:"checks"`
	Summary     DoctorSummary `json:"summary"`
	Suggestions []string      `json:"suggestions,omitempty"`
}

// CheckResult mirrors the cmd.CheckResult struct for JSON parsing.
type CheckResult struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// DoctorSummary mirrors the cmd.DoctorSummary struct for JSON parsing.
type DoctorSummary struct {
	Passed   int `json:"passed"`
	Warnings int `json:"warnings"`
	Errors   int `json:"errors"`
}

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

// createPrivateKey creates a private key file for testing.
func createPrivateKey(t *testing.T, tempUserDir string, permissions os.FileMode) {
	keysDir := filepath.Join(tempUserDir, "keys", shared.TestProjectUUID)
	if err := os.MkdirAll(keysDir, 0755); err != nil {
		t.Fatalf("Failed to create keys directory: %v", err)
	}

	privateKeyPath := filepath.Join(keysDir, "privkey")
	if err := os.WriteFile(privateKeyPath, []byte("fake-private-key"), permissions); err != nil {
		t.Fatalf("Failed to create private key: %v", err)
	}
}

// createPublicKey creates a public key file in the project.
func createPublicKey(t *testing.T, tempDir, uuid string) {
	publicKeysDir := filepath.Join(tempDir, ".kanuka", "public_keys")
	publicKeyPath := filepath.Join(publicKeysDir, uuid+".pub")
	// #nosec G306 -- Public keys are intended to be world-readable
	if err := os.WriteFile(publicKeyPath, []byte("fake-public-key"), 0644); err != nil {
		t.Fatalf("Failed to create public key: %v", err)
	}
}

// createKanukaFile creates an encrypted symmetric key file in the project.
func createKanukaFile(t *testing.T, tempDir, uuid string) {
	secretsDir := filepath.Join(tempDir, ".kanuka", "secrets")
	kanukaPath := filepath.Join(secretsDir, uuid+".kanuka")
	if err := os.WriteFile(kanukaPath, []byte("fake-encrypted-key"), 0600); err != nil {
		t.Fatalf("Failed to create kanuka file: %v", err)
	}
}

// createGitignore creates a .gitignore file with the given content.
func createGitignore(t *testing.T, tempDir, content string) {
	gitignorePath := filepath.Join(tempDir, ".gitignore")
	// #nosec G306 -- Gitignore files are intended to be world-readable
	if err := os.WriteFile(gitignorePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}
}

// createEnvFile creates a .env file with the given content.
func createEnvFile(t *testing.T, path, content string) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create env file %s: %v", path, err)
	}
}

// createEncryptedEnvFile creates an encrypted .env.kanuka file.
func createEncryptedEnvFile(t *testing.T, path, content string) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create encrypted file %s: %v", path, err)
	}
}

func TestDoctor_HealthyProject(t *testing.T) {
	cleanup := setupMockExit()
	defer cleanup()

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
	createPrivateKey(t, tempUserDir, 0600)
	createPublicKey(t, tempDir, shared.TestUserUUID)
	createKanukaFile(t, tempDir, shared.TestUserUUID)
	createGitignore(t, tempDir, ".env\n.env.*\n")

	// Run doctor command.
	output, _ := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("doctor", []string{}, nil, nil, false, false)
		cmd.SetDoctorExitFunc(mockExit) // Set mock after ResetGlobalState is called
		return testCmd.Execute()
	})

	// Verify output contains passing checks.
	if !strings.Contains(output, "Project configuration valid") {
		t.Errorf("Output should contain 'Project configuration valid', got: %s", output)
	}
	if !strings.Contains(output, "User configuration valid") {
		t.Errorf("Output should contain 'User configuration valid', got: %s", output)
	}
	if !strings.Contains(output, "Private key exists") {
		t.Errorf("Output should contain 'Private key exists', got: %s", output)
	}
	if !strings.Contains(output, "Private key has correct permissions") {
		t.Errorf("Output should contain 'Private key has correct permissions', got: %s", output)
	}
}

func TestDoctor_MissingPrivateKey(t *testing.T) {
	cleanup := setupMockExit()
	defer cleanup()

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
	// Note: Not creating private key
	createGitignore(t, tempDir, ".env\n")

	// Run doctor command.
	output, _ := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("doctor", []string{}, nil, nil, false, false)
		cmd.SetDoctorExitFunc(mockExit) // Set mock after ResetGlobalState is called
		return testCmd.Execute()
	})

	// Verify output shows private key error.
	if !strings.Contains(output, "Private key not found") {
		t.Errorf("Output should contain 'Private key not found', got: %s", output)
	}
}

func TestDoctor_BadPermissions(t *testing.T) {
	cleanup := setupMockExit()
	defer cleanup()

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
	createPrivateKey(t, tempUserDir, 0644) // Insecure permissions
	createGitignore(t, tempDir, ".env\n")

	// Run doctor command.
	output, _ := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("doctor", []string{}, nil, nil, false, false)
		cmd.SetDoctorExitFunc(mockExit) // Set mock after ResetGlobalState is called
		return testCmd.Execute()
	})

	// Verify output shows permission warning.
	if !strings.Contains(output, "insecure permissions") {
		t.Errorf("Output should contain 'insecure permissions', got: %s", output)
	}
}

func TestDoctor_PendingUsers(t *testing.T) {
	cleanup := setupMockExit()
	defer cleanup()

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
	createPrivateKey(t, tempUserDir, 0600)
	// Create public key but NOT kanuka file (pending user)
	createPublicKey(t, tempDir, shared.TestUserUUID)
	createGitignore(t, tempDir, ".env\n")

	// Run doctor command.
	output, _ := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("doctor", []string{}, nil, nil, false, false)
		cmd.SetDoctorExitFunc(mockExit) // Set mock after ResetGlobalState is called
		return testCmd.Execute()
	})

	// Verify output shows pending users warning.
	if !strings.Contains(output, "pending users") {
		t.Errorf("Output should contain 'pending users', got: %s", output)
	}
	if !strings.Contains(output, "kanuka secrets sync") {
		t.Errorf("Output should suggest 'kanuka secrets sync', got: %s", output)
	}
}

func TestDoctor_OrphanedKanukaFile(t *testing.T) {
	cleanup := setupMockExit()
	defer cleanup()

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
	createPrivateKey(t, tempUserDir, 0600)
	// Create kanuka file but NOT public key (orphaned)
	createKanukaFile(t, tempDir, shared.TestUserUUID)
	createGitignore(t, tempDir, ".env\n")

	// Run doctor command.
	output, _ := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("doctor", []string{}, nil, nil, false, false)
		cmd.SetDoctorExitFunc(mockExit) // Set mock after ResetGlobalState is called
		return testCmd.Execute()
	})

	// Verify output shows orphaned file error.
	if !strings.Contains(output, "orphaned") {
		t.Errorf("Output should contain 'orphaned', got: %s", output)
	}
	if !strings.Contains(output, "kanuka secrets clean") {
		t.Errorf("Output should suggest 'kanuka secrets clean', got: %s", output)
	}
}

func TestDoctor_MissingGitignore(t *testing.T) {
	cleanup := setupMockExit()
	defer cleanup()

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
	createPrivateKey(t, tempUserDir, 0600)
	// Note: Not creating .gitignore

	// Run doctor command.
	output, _ := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("doctor", []string{}, nil, nil, false, false)
		cmd.SetDoctorExitFunc(mockExit) // Set mock after ResetGlobalState is called
		return testCmd.Execute()
	})

	// Verify output shows gitignore warning.
	if !strings.Contains(output, "No .gitignore file found") {
		t.Errorf("Output should contain 'No .gitignore file found', got: %s", output)
	}
}

func TestDoctor_GitignoreMissingEnvPattern(t *testing.T) {
	cleanup := setupMockExit()
	defer cleanup()

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
	createPrivateKey(t, tempUserDir, 0600)
	// Create .gitignore without .env pattern
	createGitignore(t, tempDir, "node_modules/\n*.log\n")

	// Run doctor command.
	output, _ := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("doctor", []string{}, nil, nil, false, false)
		cmd.SetDoctorExitFunc(mockExit) // Set mock after ResetGlobalState is called
		return testCmd.Execute()
	})

	// Verify output shows gitignore warning.
	if !strings.Contains(output, ".env patterns not found") {
		t.Errorf("Output should contain '.env patterns not found', got: %s", output)
	}
}

func TestDoctor_UnencryptedFiles(t *testing.T) {
	cleanup := setupMockExit()
	defer cleanup()

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
	createPrivateKey(t, tempUserDir, 0600)
	createGitignore(t, tempDir, ".env\n")

	// Create .env file without corresponding .env.kanuka
	createEnvFile(t, filepath.Join(tempDir, ".env"), "SECRET=value")

	// Run doctor command.
	output, _ := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("doctor", []string{}, nil, nil, false, false)
		cmd.SetDoctorExitFunc(mockExit) // Set mock after ResetGlobalState is called
		return testCmd.Execute()
	})

	// Verify output shows unencrypted files warning.
	if !strings.Contains(output, "unencrypted .env file") {
		t.Errorf("Output should contain 'unencrypted .env file', got: %s", output)
	}
	if !strings.Contains(output, "kanuka secrets encrypt") {
		t.Errorf("Output should suggest 'kanuka secrets encrypt', got: %s", output)
	}
}

func TestDoctor_AllEncrypted(t *testing.T) {
	cleanup := setupMockExit()
	defer cleanup()

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
	createPrivateKey(t, tempUserDir, 0600)
	createGitignore(t, tempDir, ".env\n")

	// Create .env file WITH corresponding .env.kanuka
	createEnvFile(t, filepath.Join(tempDir, ".env"), "SECRET=value")
	createEncryptedEnvFile(t, filepath.Join(tempDir, ".env.kanuka"), "encrypted-data")

	// Run doctor command.
	output, _ := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("doctor", []string{}, nil, nil, false, false)
		cmd.SetDoctorExitFunc(mockExit) // Set mock after ResetGlobalState is called
		return testCmd.Execute()
	})

	// Verify output shows files are encrypted.
	if !strings.Contains(output, "All .env files have encrypted versions") {
		t.Errorf("Output should contain 'All .env files have encrypted versions', got: %s", output)
	}
}

func TestDoctor_JSONOutput(t *testing.T) {
	cleanup := setupMockExit()
	defer cleanup()

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
	createPrivateKey(t, tempUserDir, 0600)
	createPublicKey(t, tempDir, shared.TestUserUUID)
	createKanukaFile(t, tempDir, shared.TestUserUUID)
	createGitignore(t, tempDir, ".env\n")

	// Run doctor command with --json flag.
	output, _ := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("doctor", []string{"--json"}, nil, nil, false, false)
		cmd.SetDoctorExitFunc(mockExit) // Set mock after ResetGlobalState is called
		return testCmd.Execute()
	})

	// Parse JSON output.
	var result DoctorResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput was: %s", err, output)
	}

	// Verify structure.
	if len(result.Checks) == 0 {
		t.Error("Expected at least one check in JSON output")
	}

	// Verify summary.
	if result.Summary.Passed == 0 {
		t.Error("Expected at least one passed check")
	}

	// Verify check structure.
	for _, check := range result.Checks {
		if check.Name == "" {
			t.Error("Check name should not be empty")
		}
		if check.Status != "pass" && check.Status != "warning" && check.Status != "error" {
			t.Errorf("Invalid check status: %s", check.Status)
		}
		if check.Message == "" {
			t.Error("Check message should not be empty")
		}
	}
}

func TestDoctor_NotInitialized(t *testing.T) {
	cleanup := setupMockExit()
	defer cleanup()

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

	// Note: Not setting up project (not initialized)

	// Run doctor command.
	output, _ := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("doctor", []string{}, nil, nil, false, false)
		cmd.SetDoctorExitFunc(mockExit) // Set mock after ResetGlobalState is called
		return testCmd.Execute()
	})

	// Verify output shows project not found.
	if !strings.Contains(output, "not found") || !strings.Contains(output, "kanuka secrets init") {
		t.Errorf("Output should indicate project not found and suggest init, got: %s", output)
	}
}

func TestDoctor_Summary(t *testing.T) {
	cleanup := setupMockExit()
	defer cleanup()

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
	// Missing private key (error) and missing gitignore (warning)
	// This should give us both errors and warnings

	// Run doctor command.
	output, _ := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("doctor", []string{}, nil, nil, false, false)
		cmd.SetDoctorExitFunc(mockExit) // Set mock after ResetGlobalState is called
		return testCmd.Execute()
	})

	// Verify summary is present.
	if !strings.Contains(output, "Summary:") {
		t.Errorf("Output should contain 'Summary:', got: %s", output)
	}
	if !strings.Contains(output, "passed") {
		t.Errorf("Output should contain 'passed' count, got: %s", output)
	}
}
