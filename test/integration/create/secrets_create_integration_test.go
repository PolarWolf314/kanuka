package create

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsCreateIntegration contains basic functionality tests for the `kanuka secrets create` command.
func TestSecretsCreateIntegration(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("CreateInUninitializedProject", func(t *testing.T) {
		testCreateInUninitializedProject(t, originalWd, originalUserSettings)
	})

	t.Run("CreateInInitializedProject", func(t *testing.T) {
		testCreateInInitializedProject(t, originalWd, originalUserSettings)
	})

	t.Run("CreateWhenUserAlreadyHasKeys", func(t *testing.T) {
		testCreateWhenUserAlreadyHasKeys(t, originalWd, originalUserSettings)
	})

	t.Run("CreateWithForceFlag", func(t *testing.T) {
		testCreateWithForceFlag(t, originalWd, originalUserSettings)
	})
}

// Tests create in uninitialized project.
func testCreateInUninitializedProject(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-create-uninit-*")
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
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "Kānuka has not been initialized") {
		t.Errorf("Expected 'not initialized' message not found in output: %s", output)
	}

	if !strings.Contains(output, "kanuka secrets init") {
		t.Errorf("Expected init instruction not found in output: %s", output)
	}
}

// Tests create in initialized project (new user).
func testCreateInInitializedProject(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-create-init-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-init-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize project structure only (without creating user keys)
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to initialize project: %v", err)
	}

	// Verify no user keys exist yet
	projectUUID := shared.GetProjectUUID(t)
	userUUID := shared.GetUserUUID(t)
	keysDir := filepath.Join(tempUserDir, "keys")
	keyDir := shared.GetKeyDirPath(keysDir, projectUUID)
	privateKeyPath := shared.GetPrivateKeyPath(keysDir, projectUUID)
	publicKeyPath := shared.GetPublicKeyPath(keysDir, projectUUID)
	projectPublicKeyPath := filepath.Join(tempDir, ".kanuka", "public_keys", userUUID+".pub")

	// Remove any existing keys from init (if any)
	os.RemoveAll(keyDir)
	os.Remove(projectPublicKeyPath)

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "✓") || !strings.Contains(output, "Keys created for") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	if !strings.Contains(output, "created:") {
		t.Errorf("Expected file creation message not found in output: %s", output)
	}

	if !strings.Contains(output, "To gain access to the secrets") {
		t.Errorf("Expected instructions not found in output: %s", output)
	}

	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		t.Errorf("Private key was not created at %s", privateKeyPath)
	}

	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		t.Errorf("Public key was not created at %s", publicKeyPath)
	}
	if _, err := os.Stat(projectPublicKeyPath); os.IsNotExist(err) {
		t.Errorf("Public key was not copied to project at %s", projectPublicKeyPath)
	}
}

// Tests create when user already has keys.
func testCreateWhenUserAlreadyHasKeys(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-create-existing-*")
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

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("First create command failed: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false) // Use verbose to see the "already exists" message
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	userUUID := shared.GetUserUUID(t)
	// With the new implementation, the command should show "already exists" message
	// but may succeed if it can still complete the operation
	if !strings.Contains(output, userUUID+".pub already exists") && !strings.Contains(output, "already exists") {
		t.Errorf("Expected 'already exists' message not found in output: %s", output)
	}

	if !strings.Contains(output, "kanuka secrets create --force") {
		t.Errorf("Expected force flag instruction not found in output: %s", output)
	}
}

// Tests create with force flag.
func testCreateWithForceFlag(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-create-force-*")
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

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("First create command failed: %v", err)
	}

	projectUUID := shared.GetProjectUUID(t)
	privateKeyPath := shared.GetPrivateKeyPath(filepath.Join(tempUserDir, "keys"), projectUUID)

	originalKeyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		t.Fatalf("Failed to read original private key: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "create", "--force"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Force create command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "✓") || !strings.Contains(output, "Keys created for") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	newKeyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		t.Errorf("Failed to read new private key: %v", err)
	}

	if string(originalKeyData) == string(newKeyData) {
		t.Errorf("Private key was not replaced with force flag")
	}
}
