package register

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/cmd"
	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/secrets"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestRegisterDryRun_PreviewsWithoutCreating tests that --dry-run with --user shows preview without creating files.
func TestRegisterDryRun_PreviewsWithoutCreating(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-dry-*")
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
	_ = createDryRunTestUserKeyPair(t, tempDir, targetUserUUID)

	// Add the target user to the project config with UUID→email mapping.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectConfig.Users[targetUserUUID] = targetUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	// Note the expected .kanuka file path that would be created.
	targetKanukaFile := filepath.Join(tempDir, ".kanuka", "secrets", targetUserUUID+".kanuka")

	// Verify .kanuka file does NOT exist before dry-run.
	if _, err := os.Stat(targetKanukaFile); !os.IsNotExist(err) {
		t.Fatal(".kanuka file should not exist before dry-run")
	}

	// Run register with --dry-run.
	output, err := shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("register", []string{"--user", targetUserEmail, "--dry-run"}, nil, nil, true, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Dry-run command should not return error: %v", err)
	}

	// Verify output contains expected dry-run messages.
	if !strings.Contains(output, "[dry-run]") {
		t.Errorf("Output should contain '[dry-run]' prefix, got: %s", output)
	}
	if !strings.Contains(output, "Would register") {
		t.Errorf("Output should contain 'Would register', got: %s", output)
	}
	if !strings.Contains(output, targetUserEmail) {
		t.Errorf("Output should contain target email '%s', got: %s", targetUserEmail, output)
	}
	if !strings.Contains(output, "Files that would be created") {
		t.Errorf("Output should contain 'Files that would be created', got: %s", output)
	}
	if !strings.Contains(output, "No changes made") {
		t.Errorf("Output should contain 'No changes made', got: %s", output)
	}

	// Verify .kanuka file was NOT created.
	if _, err := os.Stat(targetKanukaFile); !os.IsNotExist(err) {
		t.Error(".kanuka file should NOT be created after dry-run")
	}
}

// TestRegisterDryRun_WithPubkeyFlag tests that --dry-run with --pubkey shows preview without creating files.
func TestRegisterDryRun_WithPubkeyFlag(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-dry-pubkey-*")
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

	// Generate a test key pair and convert public key to PEM format.
	targetUserUUID := shared.TestUser2UUID
	targetUserEmail := shared.TestUser2Email
	targetUserKeyPair := generateDryRunTestKeyPair(t)
	pubkeyText := convertDryRunPublicKeyToPEM(t, targetUserKeyPair.publicKey)

	// Add the target user to the project config with UUID→email mapping.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectConfig.Users[targetUserUUID] = targetUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	// Note the expected file paths that would be created.
	targetPubKeyFile := filepath.Join(tempDir, ".kanuka", "public_keys", targetUserUUID+".pub")
	targetKanukaFile := filepath.Join(tempDir, ".kanuka", "secrets", targetUserUUID+".kanuka")

	// Verify files do NOT exist before dry-run.
	if _, err := os.Stat(targetPubKeyFile); !os.IsNotExist(err) {
		t.Fatal("Public key file should not exist before dry-run")
	}
	if _, err := os.Stat(targetKanukaFile); !os.IsNotExist(err) {
		t.Fatal(".kanuka file should not exist before dry-run")
	}

	// Run register with --pubkey, --user, and --dry-run.
	output, err := shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("register", []string{"--pubkey", pubkeyText, "--user", targetUserEmail, "--dry-run"}, nil, nil, true, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Dry-run command should not return error: %v", err)
	}

	// Verify output contains expected dry-run messages.
	if !strings.Contains(output, "[dry-run]") {
		t.Errorf("Output should contain '[dry-run]' prefix, got: %s", output)
	}
	if !strings.Contains(output, "Would register") {
		t.Errorf("Output should contain 'Would register', got: %s", output)
	}
	if !strings.Contains(output, targetUserEmail) {
		t.Errorf("Output should contain target email '%s', got: %s", targetUserEmail, output)
	}
	if !strings.Contains(output, "No changes made") {
		t.Errorf("Output should contain 'No changes made', got: %s", output)
	}

	// Verify files were NOT created.
	if _, err := os.Stat(targetPubKeyFile); !os.IsNotExist(err) {
		t.Error("Public key file should NOT be created after dry-run")
	}
	if _, err := os.Stat(targetKanukaFile); !os.IsNotExist(err) {
		t.Error(".kanuka file should NOT be created after dry-run")
	}
}

// TestRegisterDryRun_ShowsPrerequisites tests that --dry-run shows prerequisites verified.
func TestRegisterDryRun_ShowsPrerequisites(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-dry-prereq-*")
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
	_ = createDryRunTestUserKeyPair(t, tempDir, targetUserUUID)

	// Add the target user to the project config with UUID→email mapping.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectConfig.Users[targetUserUUID] = targetUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	// Run register with --dry-run.
	output, err := shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("register", []string{"--user", targetUserEmail, "--dry-run"}, nil, nil, true, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Dry-run command should not return error: %v", err)
	}

	// Verify output contains prerequisites section.
	if !strings.Contains(output, "Prerequisites verified") {
		t.Errorf("Output should contain 'Prerequisites verified', got: %s", output)
	}
	if !strings.Contains(output, "User exists in project config") {
		t.Errorf("Output should contain 'User exists in project config', got: %s", output)
	}
	if !strings.Contains(output, "Current user has access to decrypt symmetric key") {
		t.Errorf("Output should contain 'Current user has access to decrypt symmetric key', got: %s", output)
	}
}

// TestRegisterDryRun_ValidationStillRuns tests that validation errors occur with --dry-run.
func TestRegisterDryRun_ValidationStillRuns(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-dry-validate-*")
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

	// Try to register a user that doesn't exist in the project.
	nonExistentEmail := "nonexistent@example.com"

	output, err := shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("register", []string{"--user", nonExistentEmail, "--dry-run"}, nil, nil, true, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Command should not return error: %v", err)
	}

	// Should show "user not found" error, not dry-run output.
	if !strings.Contains(output, "not found") {
		t.Errorf("Output should contain 'not found' error message, got: %s", output)
	}
	if strings.Contains(output, "[dry-run]") {
		t.Errorf("Output should NOT contain '[dry-run]' when validation fails, got: %s", output)
	}
}

// TestRegisterDryRun_NotInitialized tests that validation errors occur when project not initialized.
func TestRegisterDryRun_NotInitialized(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-dry-uninit-*")
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
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("register", []string{"--user", "test@example.com", "--dry-run"}, nil, nil, true, false)
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

// TestRegisterDryRun_SymmetricKeyValidation tests that symmetric key decryption is validated with --dry-run.
func TestRegisterDryRun_SymmetricKeyValidation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-dry-symkey-*")
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

	// Create a second user's public key.
	targetUserUUID := shared.TestUser2UUID
	targetUserEmail := shared.TestUser2Email
	_ = createDryRunTestUserKeyPair(t, tempDir, targetUserUUID)

	// Add the target user to the project config.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectConfig.Users[targetUserUUID] = targetUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	// Corrupt the current user's kanuka key file.
	userUUID := shared.GetUserUUID(t)
	kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", userUUID+".kanuka")
	if err := os.WriteFile(kanukaKeyPath, []byte("corrupted key data"), 0600); err != nil {
		t.Fatalf("Failed to corrupt kanuka key file: %v", err)
	}

	// Run register with --dry-run - should fail due to key validation.
	output, err := shared.CaptureOutput(func() error {
		cmd.ResetGlobalState()
		testCmd := shared.CreateTestCLIWithArgs("register", []string{"--user", targetUserEmail, "--dry-run"}, nil, nil, true, false)
		return testCmd.Execute()
	})

	if err != nil {
		t.Errorf("Command should not return error: %v", err)
	}

	// Should show error about decrypting the kanuka file, not dry-run output.
	if !strings.Contains(output, "Failed to decrypt") {
		t.Errorf("Output should contain 'Failed to decrypt', got: %s", output)
	}
}

// Helper types and functions for dry-run tests.

type dryRunTestKeyPair struct {
	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
}

// generateDryRunTestKeyPair generates a test RSA key pair.
func generateDryRunTestKeyPair(t *testing.T) *dryRunTestKeyPair {
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

	return &dryRunTestKeyPair{
		publicKey:  publicKey,
		privateKey: privateKey,
	}
}

// convertDryRunPublicKeyToPEM converts an RSA public key to PEM format string.
func convertDryRunPublicKeyToPEM(t *testing.T, publicKey *rsa.PublicKey) string {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		t.Fatalf("Failed to marshal public key: %v", err)
	}

	pemBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	return string(pem.EncodeToMemory(pemBlock))
}

// createDryRunTestUserKeyPair creates a key pair for a test user and places the public key in the project.
func createDryRunTestUserKeyPair(t *testing.T, projectDir, username string) *dryRunTestKeyPair {
	// Generate a key pair.
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	keyPair := &dryRunTestKeyPair{
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
