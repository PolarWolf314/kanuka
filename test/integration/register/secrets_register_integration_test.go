package register

import (
	"crypto/rsa"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/secrets"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsRegisterIntegration contains basic functionality tests for the `kanuka secrets register` command.
func TestSecretsRegisterIntegration(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("RegisterExistingUser", func(t *testing.T) {
		testRegisterExistingUser(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithCustomFile", func(t *testing.T) {
		testRegisterWithCustomFile(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithPubkeyText", func(t *testing.T) {
		testRegisterWithPubkeyText(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithVerboseFlag", func(t *testing.T) {
		testRegisterWithVerboseFlag(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithDebugFlag", func(t *testing.T) {
		testRegisterWithDebugFlag(t, originalWd, originalUserSettings)
	})
}

// testRegisterExistingUser tests registering a user whose public key already exists in the project.
func testRegisterExistingUser(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-existing-*")
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

	// Create a second user's public key in the project
	targetUser := "targetuser"
	targetUserKeyPair := createTestUserKeyPair(t, tempDir, targetUser)

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, false, false)
		cmd.SetArgs([]string{"secrets", "register", "--user", targetUser})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	if !strings.Contains(output, targetUser+".pub has been registered successfully") {
		t.Errorf("Expected registration success message not found in output: %s", output)
	}

	if !strings.Contains(output, "They now have access to decrypt") {
		t.Errorf("Expected access message not found in output: %s", output)
	}

	// Verify the .kanuka file was created for the target user
	targetKanukaFile := filepath.Join(tempDir, ".kanuka", "secrets", targetUser+".kanuka")
	if _, err := os.Stat(targetKanukaFile); os.IsNotExist(err) {
		t.Errorf("Target user's .kanuka file was not created at %s", targetKanukaFile)
	}

	// Verify the target user can actually decrypt the symmetric key
	verifyUserCanDecrypt(t, targetUser, targetUserKeyPair.privateKey)
}

// testRegisterWithCustomFile tests registering using --file flag with a custom public key file.
func testRegisterWithCustomFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-file-*")
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

	// Create a custom public key file outside the project
	customKeyFile := filepath.Join(tempUserDir, "custom_user.pub")
	targetUser := "custom_user"
	targetUserKeyPair := generateTestKeyPair(t)

	if err := secrets.SavePublicKeyToFile(targetUserKeyPair.publicKey, customKeyFile); err != nil {
		t.Fatalf("Failed to save custom public key: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, false, false)
		cmd.SetArgs([]string{"secrets", "register", "--file", customKeyFile})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	if !strings.Contains(output, targetUser+".pub has been registered successfully") {
		t.Errorf("Expected registration success message not found in output: %s", output)
	}

	// Verify the .kanuka file was created for the target user
	targetKanukaFile := filepath.Join(tempDir, ".kanuka", "secrets", targetUser+".kanuka")
	if _, err := os.Stat(targetKanukaFile); os.IsNotExist(err) {
		t.Errorf("Target user's .kanuka file was not created at %s", targetKanukaFile)
	}

	// Verify the target user can actually decrypt the symmetric key
	verifyUserCanDecrypt(t, targetUser, targetUserKeyPair.privateKey)
}

// testRegisterWithPubkeyText tests registering using --pubkey and --user flags.
func testRegisterWithPubkeyText(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-pubkey-*")
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

	// Generate a test key pair and convert public key to PEM format
	targetUser := "pubkeyuser"
	targetUserKeyPair := generateTestKeyPair(t)
	pubkeyText := convertPublicKeyToPEM(t, targetUserKeyPair.publicKey)

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, false, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", pubkeyText, "--user", targetUser})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	if !strings.Contains(output, "Public key for "+targetUser+" has been saved and registered successfully") {
		t.Errorf("Expected pubkey registration success message not found in output: %s", output)
	}

	// Verify the public key file was created in the project
	targetPubKeyFile := filepath.Join(tempDir, ".kanuka", "public_keys", targetUser+".pub")
	if _, err := os.Stat(targetPubKeyFile); os.IsNotExist(err) {
		t.Errorf("Target user's public key file was not created at %s", targetPubKeyFile)
	}

	// Verify the .kanuka file was created for the target user
	targetKanukaFile := filepath.Join(tempDir, ".kanuka", "secrets", targetUser+".kanuka")
	if _, err := os.Stat(targetKanukaFile); os.IsNotExist(err) {
		t.Errorf("Target user's .kanuka file was not created at %s", targetKanukaFile)
	}

	// Verify the target user can actually decrypt the symmetric key
	verifyUserCanDecrypt(t, targetUser, targetUserKeyPair.privateKey)
}

// testRegisterWithVerboseFlag tests register command with verbose flag.
func testRegisterWithVerboseFlag(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-verbose-*")
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

	// Create a target user's public key
	targetUser := "verboseuser"
	createTestUserKeyPair(t, tempDir, targetUser)

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--user", targetUser})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "[info]") {
		t.Errorf("Expected verbose [info] messages not found in output: %s", output)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success message not found in output: %s", output)
	}
}

// testRegisterWithDebugFlag tests register command with debug flag.
func testRegisterWithDebugFlag(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-debug-*")
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

	// Create a target user's public key
	targetUser := "debuguser"
	createTestUserKeyPair(t, tempDir, targetUser)

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, false, true)
		cmd.SetArgs([]string{"secrets", "register", "--user", targetUser})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "[debug]") {
		t.Errorf("Expected debug [debug] messages not found in output: %s", output)
	}

	// Debug should also include info messages
	if !strings.Contains(output, "[info]") {
		t.Errorf("Expected [info] messages not found in debug output: %s", output)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success message not found in output: %s", output)
	}
}

// Helper types and functions

type testKeyPair struct {
	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
}

// generateTestKeyPair generates a test RSA key pair.
func generateTestKeyPair(t *testing.T) *testKeyPair {
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

	return &testKeyPair{
		publicKey:  publicKey,
		privateKey: privateKey,
	}
}

// createTestUserKeyPair creates a key pair for a test user and places the public key in the project.
func createTestUserKeyPair(t *testing.T, projectDir, username string) *testKeyPair {
	keyPair := generateTestKeyPair(t)

	// Save the public key to the project's public_keys directory
	publicKeyPath := filepath.Join(projectDir, ".kanuka", "public_keys", username+".pub")
	if err := secrets.SavePublicKeyToFile(keyPair.publicKey, publicKeyPath); err != nil {
		t.Fatalf("Failed to save test user's public key: %v", err)
	}

	return keyPair
}

// convertPublicKeyToPEM converts an RSA public key to PEM format string.
func convertPublicKeyToPEM(t *testing.T, publicKey *rsa.PublicKey) string {
	tempFile, err := os.CreateTemp("", "test-pubkey-*.pub")
	if err != nil {
		t.Fatalf("Failed to create temp file for public key: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if err := secrets.SavePublicKeyToFile(publicKey, tempFile.Name()); err != nil {
		t.Fatalf("Failed to save public key to temp file: %v", err)
	}

	pemData, err := os.ReadFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to read PEM data: %v", err)
	}

	return string(pemData)
}

// verifyUserCanDecrypt verifies that a user can decrypt the symmetric key with their private key.
func verifyUserCanDecrypt(t *testing.T, username string, privateKey *rsa.PrivateKey) {
	// Get the user's encrypted symmetric key
	encryptedSymKey, err := secrets.GetProjectKanukaKey(username)
	if err != nil {
		t.Errorf("Failed to get encrypted symmetric key for user %s: %v", username, err)
		return
	}

	// Try to decrypt it with the user's private key
	_, err = secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		t.Errorf("User %s cannot decrypt symmetric key with their private key: %v", username, err)
	}
}
