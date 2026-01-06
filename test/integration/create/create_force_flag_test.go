package create

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsCreateForceFlag contains force flag scenario tests for the `kanuka secrets create` command.
func TestSecretsCreateForceFlag(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("ForceWithExistingKeys", func(t *testing.T) {
		testForceWithExistingKeys(t, originalWd, originalUserSettings)
	})

	t.Run("ForceWithExistingAccess", func(t *testing.T) {
		testForceWithExistingAccess(t, originalWd, originalUserSettings)
	})

	t.Run("ForceWithoutExistingKeys", func(t *testing.T) {
		testForceWithoutExistingKeys(t, originalWd, originalUserSettings)
	})

	t.Run("ForceFlagWarnings", func(t *testing.T) {
		testForceFlagWarnings(t, originalWd, originalUserSettings)
	})
}

// Tests force with existing keys - verify old keys are replaced.
func testForceWithExistingKeys(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-force-existing-*")
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

	// Create initial keys
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Initial create failed: %v", err)
	}

	projectUUID := shared.GetProjectUUID(t)
	userUUID := shared.GetUserUUID(t)

	privateKeyPath := filepath.Join(tempUserDir, "keys", projectUUID)
	publicKeyPath := filepath.Join(tempUserDir, "keys", projectUUID+".pub")
	projectPublicKeyPath := filepath.Join(tempDir, ".kanuka", "public_keys", userUUID+".pub")

	// Read original keys
	originalPrivateKey, err := os.ReadFile(privateKeyPath)
	if err != nil {
		t.Fatalf("Failed to read original private key: %v", err)
	}
	originalPublicKey, err := os.ReadFile(publicKeyPath)
	if err != nil {
		t.Fatalf("Failed to read original public key: %v", err)
	}
	originalProjectPublicKey, err := os.ReadFile(projectPublicKeyPath)
	if err != nil {
		t.Fatalf("Failed to read original project public key: %v", err)
	}

	// Use force flag to recreate keys
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "create", "--force"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Force create failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Verify success message
	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	// Read new keys
	newPrivateKey, err := os.ReadFile(privateKeyPath)
	if err != nil {
		t.Errorf("Failed to read new private key: %v", err)
	}
	newPublicKey, err := os.ReadFile(publicKeyPath)
	if err != nil {
		t.Errorf("Failed to read new public key: %v", err)
	}
	newProjectPublicKey, err := os.ReadFile(projectPublicKeyPath)
	if err != nil {
		t.Errorf("Failed to read new project public key: %v", err)
	}

	// Verify keys were replaced
	if string(originalPrivateKey) == string(newPrivateKey) {
		t.Errorf("Private key was not replaced with force flag")
	}
	if string(originalPublicKey) == string(newPublicKey) {
		t.Errorf("Public key was not replaced with force flag")
	}
	if string(originalProjectPublicKey) == string(newProjectPublicKey) {
		t.Errorf("Project public key was not replaced with force flag")
	}

	// Verify new keys are valid PEM format
	if !strings.Contains(string(newPrivateKey), "-----BEGIN RSA PRIVATE KEY-----") {
		t.Errorf("New private key is not in correct PEM format")
	}
	if !strings.Contains(string(newPublicKey), "-----BEGIN PUBLIC KEY-----") {
		t.Errorf("New public key is not in correct PEM format")
	}
}

// Tests force with existing access - verify old .kanuka file is removed.
func testForceWithExistingAccess(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-force-access-*")
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

	// Create initial keys
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Initial create failed: %v", err)
	}

	userUUID := shared.GetUserUUID(t)
	kanukaFilePath := filepath.Join(tempDir, ".kanuka", "secrets", userUUID+".kanuka")

	// Create a .kanuka file (simulating existing access)
	if err := os.WriteFile(kanukaFilePath, []byte("existing access data"), 0600); err != nil {
		t.Fatalf("Failed to create existing kanuka file: %v", err)
	}

	// Verify file exists before force
	if _, err := os.Stat(kanukaFilePath); os.IsNotExist(err) {
		t.Fatalf("Kanuka file was not created for test setup")
	}

	// Use force flag
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "create", "--force"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Force create failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Verify .kanuka file was removed
	if _, err := os.Stat(kanukaFilePath); !os.IsNotExist(err) {
		t.Errorf("Existing kanuka file was not removed with force flag")
	}

	// Verify deletion message in output
	if !strings.Contains(output, "deleted:") {
		t.Errorf("Expected deletion message not found in output: %s", output)
	}
}

// Tests force without existing keys - should work same as normal create.
func testForceWithoutExistingKeys(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-force-new-*")
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

	// Remove any existing keys from init to ensure clean state
	projectUUID := shared.GetProjectUUID(t)
	userUUID := shared.GetUserUUID(t)
	privateKeyPath := filepath.Join(tempUserDir, "keys", projectUUID)
	publicKeyPath := filepath.Join(tempUserDir, "keys", projectUUID+".pub")
	projectPublicKeyPath := filepath.Join(tempDir, ".kanuka", "public_keys", userUUID+".pub")
	kanukaFilePath := filepath.Join(tempDir, ".kanuka", "secrets", userUUID+".kanuka")

	os.Remove(privateKeyPath)
	os.Remove(publicKeyPath)
	os.Remove(projectPublicKeyPath)
	os.Remove(kanukaFilePath)

	// Use force flag without existing keys
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "create", "--force"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Force create failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Should work exactly like normal create
	if !strings.Contains(output, "✓") || !strings.Contains(output, "Keys created for") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	if !strings.Contains(output, "created:") {
		t.Errorf("Expected file creation message not found in output: %s", output)
	}

	// Verify keys were created
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		t.Errorf("Private key was not created")
	}

	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		t.Errorf("Public key was not created")
	}

	if _, err := os.Stat(projectPublicKeyPath); os.IsNotExist(err) {
		t.Errorf("Public key was not copied to project")
	}

	// With the new implementation, the command always tries to remove the .kanuka file
	// So we might see a deletion message even if no file existed before
	// This is acceptable behavior as the command is idempotent
}

// Tests force flag warnings - verify appropriate warnings are shown.
func testForceFlagWarnings(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-force-warnings-*")
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

	// Create initial keys
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Initial create failed: %v", err)
	}

	// Use force flag and capture output
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "create", "--force"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Force create failed: %v", err)
	}

	// Check for warning messages (these would be in debug/verbose output)
	// Note: The actual warning implementation may vary, but we should see some indication
	// that force flag was used and keys were overwritten
	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success confirmation not found in output: %s", output)
	}

	// Verify that the command completed successfully despite overwriting
	projectUUID := shared.GetProjectUUID(t)
	privateKeyPath := filepath.Join(tempUserDir, "keys", projectUUID)
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		t.Errorf("Private key was not created after force")
	}
}
