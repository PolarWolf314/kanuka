package register

import (
	"crypto/rand"
	"crypto/rsa"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/cmd"
	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/secrets"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestRegisterOverwrite tests the warning and force flag behavior when registering
// a user who already has access to the project.

// TestRegisterOverwrite_NewUserNoWarning tests that registering a new user
// shows "has been granted access" without any warning prompt.
func TestRegisterOverwrite_NewUserNoWarning(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-new-*")
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

	// Create a second user's public key in the project using their UUID.
	targetUserUUID := shared.TestUser2UUID
	targetUserEmail := shared.TestUser2Email
	_ = createOverwriteTestUserKeyPair(t, tempDir, targetUserUUID)

	// Add the target user to the project config with UUID->email mapping.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectConfig.Users[targetUserUUID] = targetUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	// Run register for a NEW user (no .kanuka file exists yet).
	output, err := shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("register", []string{"--user", targetUserEmail}, nil, nil, true, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Register command should not return error: %v", err)
	}

	// Verify output says "has been granted access" (not "updated").
	if !strings.Contains(output, "has been granted access") {
		t.Errorf("Output should contain 'has been granted access' for new user, got: %s", output)
	}

	// Verify output does NOT contain warning text.
	if strings.Contains(output, "already has access") {
		t.Errorf("Output should NOT contain 'already has access' for new user, got: %s", output)
	}

	// Verify output says "Files created" (not "Files updated").
	if !strings.Contains(output, "Files created") {
		t.Errorf("Output should contain 'Files created' for new user, got: %s", output)
	}

	// Verify the .kanuka file was created.
	targetKanukaFile := filepath.Join(tempDir, ".kanuka", "secrets", targetUserUUID+".kanuka")
	if _, err := os.Stat(targetKanukaFile); os.IsNotExist(err) {
		t.Errorf(".kanuka file should exist after registration")
	}
}

// TestRegisterOverwrite_ForceSkipsWarning tests that --force flag skips the
// confirmation prompt and shows "access has been updated".
func TestRegisterOverwrite_ForceSkipsWarning(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-force-*")
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

	// Create a second user's public key in the project using their UUID.
	targetUserUUID := shared.TestUser2UUID
	targetUserEmail := shared.TestUser2Email
	_ = createOverwriteTestUserKeyPair(t, tempDir, targetUserUUID)

	// Add the target user to the project config with UUID->email mapping.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectConfig.Users[targetUserUUID] = targetUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	// First, register the user so they have access.
	_, err = shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("register", []string{"--user", targetUserEmail}, nil, nil, true, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("First registration should succeed: %v", err)
	}

	// Verify .kanuka file exists after first registration.
	targetKanukaFile := filepath.Join(tempDir, ".kanuka", "secrets", targetUserUUID+".kanuka")
	if _, err := os.Stat(targetKanukaFile); os.IsNotExist(err) {
		t.Fatal(".kanuka file should exist after first registration")
	}

	// Get the file modification time before second registration.
	origInfo, err := os.Stat(targetKanukaFile)
	if err != nil {
		t.Fatalf("Failed to stat .kanuka file: %v", err)
	}
	origModTime := origInfo.ModTime()

	// Now register again with --force flag (user already has access).
	output, err := shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("register", []string{"--user", targetUserEmail, "--force"}, nil, nil, true, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Register command with --force should not return error: %v", err)
	}

	// Verify output says "access has been updated" (not "granted access").
	if !strings.Contains(output, "access has been updated") {
		t.Errorf("Output should contain 'access has been updated' for existing user with --force, got: %s", output)
	}

	// Verify output says "Files updated" (not "Files created").
	if !strings.Contains(output, "Files updated") {
		t.Errorf("Output should contain 'Files updated' for existing user, got: %s", output)
	}

	// Verify output does NOT contain warning prompt text (since --force skips it).
	if strings.Contains(output, "already has access") {
		t.Errorf("Output should NOT contain warning prompt with --force, got: %s", output)
	}
	if strings.Contains(output, "Do you want to continue") {
		t.Errorf("Output should NOT contain confirmation prompt with --force, got: %s", output)
	}

	// Verify the file was actually updated (modtime changed).
	newInfo, err := os.Stat(targetKanukaFile)
	if err != nil {
		t.Fatalf("Failed to stat .kanuka file after re-registration: %v", err)
	}
	if !newInfo.ModTime().After(origModTime) {
		t.Logf("Warning: file modification time did not change, may be too fast")
	}
}

// TestRegisterOverwrite_AbortOnDecline tests that declining the confirmation
// prompt cancels registration and makes no changes.
func TestRegisterOverwrite_AbortOnDecline(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-abort-*")
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

	// Create a second user's public key in the project using their UUID.
	targetUserUUID := shared.TestUser2UUID
	targetUserEmail := shared.TestUser2Email
	_ = createOverwriteTestUserKeyPair(t, tempDir, targetUserUUID)

	// Add the target user to the project config with UUID->email mapping.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectConfig.Users[targetUserUUID] = targetUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	// First, register the user so they have access.
	_, err = shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("register", []string{"--user", targetUserEmail}, nil, nil, true, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("First registration should succeed: %v", err)
	}

	// Verify .kanuka file exists and get its contents for comparison.
	targetKanukaFile := filepath.Join(tempDir, ".kanuka", "secrets", targetUserUUID+".kanuka")
	origContent, err := os.ReadFile(targetKanukaFile)
	if err != nil {
		t.Fatalf("Failed to read .kanuka file: %v", err)
	}

	// Now try to register again but decline the prompt (send "n").
	output, err := shared.CaptureOutputWithStdin([]byte("n\n"), func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("register", []string{"--user", targetUserEmail}, nil, nil, true, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Register command should not return error even when cancelled: %v", err)
	}

	// Verify output contains warning about existing access.
	if !strings.Contains(output, "already has access") {
		t.Errorf("Output should contain 'already has access' warning, got: %s", output)
	}

	// Verify output contains the cancellation message.
	if !strings.Contains(output, "Registration cancelled") {
		t.Errorf("Output should contain 'Registration cancelled', got: %s", output)
	}

	// Verify the file was NOT modified (contents unchanged).
	newContent, err := os.ReadFile(targetKanukaFile)
	if err != nil {
		t.Fatalf("Failed to read .kanuka file after cancelled registration: %v", err)
	}
	if string(origContent) != string(newContent) {
		t.Errorf(".kanuka file should not have been modified after declining confirmation")
	}
}

// TestRegisterOverwrite_ConfirmOnAccept tests that accepting the confirmation
// prompt proceeds with registration and updates the files.
func TestRegisterOverwrite_ConfirmOnAccept(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-accept-*")
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

	// Create a second user's public key in the project using their UUID.
	targetUserUUID := shared.TestUser2UUID
	targetUserEmail := shared.TestUser2Email
	_ = createOverwriteTestUserKeyPair(t, tempDir, targetUserUUID)

	// Add the target user to the project config with UUID->email mapping.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectConfig.Users[targetUserUUID] = targetUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	// First, register the user so they have access.
	_, err = shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("register", []string{"--user", targetUserEmail}, nil, nil, true, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("First registration should succeed: %v", err)
	}

	// Verify .kanuka file exists.
	targetKanukaFile := filepath.Join(tempDir, ".kanuka", "secrets", targetUserUUID+".kanuka")
	if _, err := os.Stat(targetKanukaFile); os.IsNotExist(err) {
		t.Fatal(".kanuka file should exist after first registration")
	}

	// Now register again and accept the prompt (send "y").
	output, err := shared.CaptureOutputWithStdin([]byte("y\n"), func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("register", []string{"--user", targetUserEmail}, nil, nil, true, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Register command should not return error: %v", err)
	}

	// Verify output contains warning about existing access.
	if !strings.Contains(output, "already has access") {
		t.Errorf("Output should contain 'already has access' warning, got: %s", output)
	}

	// Verify output says "access has been updated" after confirmation.
	if !strings.Contains(output, "access has been updated") {
		t.Errorf("Output should contain 'access has been updated' after confirmation, got: %s", output)
	}

	// Verify output does NOT contain cancellation message.
	if strings.Contains(output, "Registration cancelled") {
		t.Errorf("Output should NOT contain 'Registration cancelled' after accepting, got: %s", output)
	}
}

// TestRegisterOverwrite_DryRunNoWarning tests that --dry-run does not show
// the interactive warning prompt (it would hang otherwise).
func TestRegisterOverwrite_DryRunNoWarning(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-dryrun-*")
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

	// Create a second user's public key in the project using their UUID.
	targetUserUUID := shared.TestUser2UUID
	targetUserEmail := shared.TestUser2Email
	_ = createOverwriteTestUserKeyPair(t, tempDir, targetUserUUID)

	// Add the target user to the project config with UUID->email mapping.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectConfig.Users[targetUserUUID] = targetUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	// First, register the user so they have access.
	_, err = shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("register", []string{"--user", targetUserEmail}, nil, nil, true, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("First registration should succeed: %v", err)
	}

	// Now run with --dry-run for an existing user.
	// This should NOT prompt (would hang if it did), just show dry-run output.
	output, err := shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("register", []string{"--user", targetUserEmail, "--dry-run"}, nil, nil, true, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Dry-run command should not return error: %v", err)
	}

	// Verify output contains dry-run prefix.
	if !strings.Contains(output, "[dry-run]") {
		t.Errorf("Output should contain '[dry-run]', got: %s", output)
	}

	// Verify output does NOT contain warning prompt (since dry-run skips it).
	if strings.Contains(output, "already has access") {
		t.Errorf("Dry-run output should NOT contain warning prompt, got: %s", output)
	}
	if strings.Contains(output, "Do you want to continue") {
		t.Errorf("Dry-run output should NOT contain confirmation prompt, got: %s", output)
	}

	// Verify output says no changes made.
	if !strings.Contains(output, "No changes made") {
		t.Errorf("Output should contain 'No changes made', got: %s", output)
	}
}

// Helper types and functions for overwrite tests.

type overwriteTestKeyPair struct {
	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
}

// generateOverwriteTestKeyPair generates a test RSA key pair.
func generateOverwriteTestKeyPair(t *testing.T) *overwriteTestKeyPair {
	tempKeyDir, err := os.MkdirTemp("", "kanuka-test-keys-*")
	if err != nil {
		t.Fatalf("Failed to create temp key directory: %v", err)
	}
	defer os.RemoveAll(tempKeyDir)

	privateKeyPath := filepath.Join(tempKeyDir, "test_key")
	publicKeyPath := privateKeyPath + ".pub"

	if err := secrets.GenerateRSAKeyPair(privateKeyPath, publicKeyPath); err != nil {
		t.Fatalf("Failed to generate test key pair: %v", err)
	}

	privateKey, err := secrets.LoadPrivateKey(privateKeyPath)
	if err != nil {
		t.Fatalf("Failed to load test private key: %v", err)
	}

	publicKey, err := secrets.LoadPublicKey(publicKeyPath)
	if err != nil {
		t.Fatalf("Failed to load test public key: %v", err)
	}

	return &overwriteTestKeyPair{
		publicKey:  publicKey,
		privateKey: privateKey,
	}
}

// createOverwriteTestUserKeyPair creates a key pair for a test user and places the public key in the project.
func createOverwriteTestUserKeyPair(t *testing.T, projectDir, username string) *overwriteTestKeyPair {
	// Generate a key pair.
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	keyPair := &overwriteTestKeyPair{
		publicKey:  &privateKey.PublicKey,
		privateKey: privateKey,
	}

	// Save the public key to the project's public_keys directory.
	publicKeyPath := filepath.Join(projectDir, ".kanuka", "public_keys", username+".pub")
	if err := secrets.SavePublicKeyToFile(keyPair.publicKey, publicKeyPath); err != nil {
		t.Fatalf("Failed to save test user's public key: %v", err)
	}

	return keyPair
}
