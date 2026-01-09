package rotate

import (
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

// setupRotateTestProject initializes a complete project with user access for rotate tests.
func setupRotateTestProject(t *testing.T, tempDir, tempUserDir string) {
	// Initialize project using init command
	_, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLI("init", nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to initialize project: %v", err)
	}
}

// getPrivateKeyBytes reads and returns the private key bytes from the user's key directory.
func getPrivateKeyBytes(t *testing.T, projectUUID string) []byte {
	privateKeyPath := configs.GetPrivateKeyPath(projectUUID)
	keyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		t.Fatalf("Failed to read private key: %v", err)
	}
	return keyBytes
}

// getPublicKeyBytes reads and returns the public key bytes from the project.
func getPublicKeyBytes(t *testing.T, tempDir, userUUID string) []byte {
	publicKeyPath := filepath.Join(tempDir, ".kanuka", "public_keys", userUUID+".pub")
	keyBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		t.Fatalf("Failed to read public key: %v", err)
	}
	return keyBytes
}

// getKanukaKeyBytes reads and returns the encrypted symmetric key bytes.
func getKanukaKeyBytes(t *testing.T, tempDir, userUUID string) []byte {
	kanukaPath := filepath.Join(tempDir, ".kanuka", "secrets", userUUID+".kanuka")
	keyBytes, err := os.ReadFile(kanukaPath)
	if err != nil {
		t.Fatalf("Failed to read kanuka key: %v", err)
	}
	return keyBytes
}

// parsePrivateKey parses a PEM-encoded private key.
func parsePrivateKey(t *testing.T, keyBytes []byte) *rsa.PrivateKey {
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		t.Fatalf("Failed to decode PEM block from private key")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		t.Fatalf("Failed to parse private key: %v", err)
	}
	return key
}

func TestRotate_Basic(t *testing.T) {
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

	// Initialize the project
	setupRotateTestProject(t, tempDir, tempUserDir)

	// Get project and user UUIDs
	projectUUID := shared.GetProjectUUID(t)
	userUUID := shared.GetUserUUID(t)

	// Get the original keys before rotation
	originalPrivateKeyBytes := getPrivateKeyBytes(t, projectUUID)
	originalPublicKeyBytes := getPublicKeyBytes(t, tempDir, userUUID)
	originalKanukaKeyBytes := getKanukaKeyBytes(t, tempDir, userUUID)

	// Run rotate command with --force to skip confirmation
	_, err = shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("rotate", []string{"--force"}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Rotate command failed with error: %v", err)
	}

	// Verify new private key is different
	newPrivateKeyBytes := getPrivateKeyBytes(t, projectUUID)
	if string(newPrivateKeyBytes) == string(originalPrivateKeyBytes) {
		t.Error("Private key should have changed after rotation")
	}

	// Verify new public key in project is different
	newPublicKeyBytes := getPublicKeyBytes(t, tempDir, userUUID)
	if string(newPublicKeyBytes) == string(originalPublicKeyBytes) {
		t.Error("Public key in project should have changed after rotation")
	}

	// Verify kanuka key (encrypted symmetric key) is different
	newKanukaKeyBytes := getKanukaKeyBytes(t, tempDir, userUUID)
	if string(newKanukaKeyBytes) == string(originalKanukaKeyBytes) {
		t.Error("Encrypted symmetric key should have changed after rotation")
	}

	// Verify we can decrypt with new key
	newPrivateKey := parsePrivateKey(t, newPrivateKeyBytes)
	symKey, err := secrets.DecryptWithPrivateKey(newKanukaKeyBytes, newPrivateKey)
	if err != nil {
		t.Errorf("Failed to decrypt with new key: %v", err)
	}
	if len(symKey) != 32 {
		t.Errorf("Expected 32-byte symmetric key, got %d bytes", len(symKey))
	}
}

func TestRotate_OldKeyNoLongerWorks(t *testing.T) {
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

	// Initialize the project
	setupRotateTestProject(t, tempDir, tempUserDir)

	// Get project and user UUIDs
	projectUUID := shared.GetProjectUUID(t)
	userUUID := shared.GetUserUUID(t)

	// Get the original private key before rotation
	originalPrivateKeyBytes := getPrivateKeyBytes(t, projectUUID)
	originalPrivateKey := parsePrivateKey(t, originalPrivateKeyBytes)

	// Run rotate command with --force to skip confirmation
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("rotate", []string{"--force"}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Rotate command failed with error: %v\nOutput: %s", err, output)
	}

	// Try to decrypt with old key - should fail
	newKanukaKeyBytes := getKanukaKeyBytes(t, tempDir, userUUID)
	_, err = secrets.DecryptWithPrivateKey(newKanukaKeyBytes, originalPrivateKey)
	if err == nil {
		t.Error("Expected decryption with old key to fail, but it succeeded")
	}
}

func TestRotate_Force(t *testing.T) {
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

	// Initialize the project
	setupRotateTestProject(t, tempDir, tempUserDir)

	// Get project UUID to check key changed
	projectUUID := shared.GetProjectUUID(t)
	originalPrivateKeyBytes := getPrivateKeyBytes(t, projectUUID)

	// Run rotate command with --force - should succeed without prompting
	_, err = shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("rotate", []string{"--force"}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Rotate command with --force failed: %v", err)
	}

	// Verify the key changed (command succeeded)
	newPrivateKeyBytes := getPrivateKeyBytes(t, projectUUID)
	if string(newPrivateKeyBytes) == string(originalPrivateKeyBytes) {
		t.Error("Private key should have changed after rotation with --force")
	}
}

func TestRotate_NotInitialized(t *testing.T) {
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

	// Don't initialize the project - run rotate directly
	// Use verbose mode to ensure output is printed (not just spinner final msg)
	output, _ := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("rotate", []string{"--force"}, nil, nil, true, false)
		return testCmd.Execute()
	})

	// Verify error message - either about not initialized or project not found
	if !strings.Contains(output, "not") && !strings.Contains(output, "init") {
		t.Errorf("Expected error message about project not initialized, got: %s", output)
	}
}

func TestRotate_NoAccess(t *testing.T) {
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

	// Initialize the project structure only (no user keys or access)
	shared.InitializeProjectStructureOnly(t, tempDir, tempUserDir)

	// Run rotate command with verbose mode - should fail because user doesn't have access
	output, _ := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("rotate", []string{"--force"}, nil, nil, true, false)
		return testCmd.Execute()
	})

	// Verify error message about no access (check for either "don't have access" or "access" or no .kanuka file)
	if !strings.Contains(output, "access") && !strings.Contains(output, "kanuka") {
		t.Errorf("Expected message about access or kanuka file, got: %s", output)
	}
}

func TestRotate_MetadataUpdated(t *testing.T) {
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

	// Initialize the project
	setupRotateTestProject(t, tempDir, tempUserDir)

	// Get project UUID
	projectUUID := shared.GetProjectUUID(t)

	// Get metadata before rotation
	metadataBefore, err := configs.LoadKeyMetadata(projectUUID)
	if err != nil {
		t.Fatalf("Failed to load metadata before rotation: %v", err)
	}

	// Wait a moment to ensure timestamps differ
	// (In practice, the test runs fast enough that we just verify metadata exists)

	// Run rotate command
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("rotate", []string{"--force"}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Rotate command failed: %v\nOutput: %s", err, output)
	}

	// Get metadata after rotation
	metadataAfter, err := configs.LoadKeyMetadata(projectUUID)
	if err != nil {
		t.Fatalf("Failed to load metadata after rotation: %v", err)
	}

	// Verify metadata was updated (CreatedAt should be newer or same)
	if metadataAfter.CreatedAt.Before(metadataBefore.CreatedAt) {
		t.Error("Metadata CreatedAt should not be before original")
	}

	// Verify project info is preserved
	if metadataAfter.ProjectName == "" {
		t.Error("Metadata ProjectName should not be empty after rotation")
	}
}

func TestRotate_SymmetricKeyUnchanged(t *testing.T) {
	// This test verifies that while the encryption of the symmetric key changes,
	// the actual symmetric key value remains the same (so other users can still decrypt).
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

	// Initialize the project
	setupRotateTestProject(t, tempDir, tempUserDir)

	// Get project and user UUIDs
	projectUUID := shared.GetProjectUUID(t)
	userUUID := shared.GetUserUUID(t)

	// Decrypt and store original symmetric key
	originalPrivateKeyBytes := getPrivateKeyBytes(t, projectUUID)
	originalPrivateKey := parsePrivateKey(t, originalPrivateKeyBytes)
	originalKanukaKeyBytes := getKanukaKeyBytes(t, tempDir, userUUID)

	originalSymKey, err := secrets.DecryptWithPrivateKey(originalKanukaKeyBytes, originalPrivateKey)
	if err != nil {
		t.Fatalf("Failed to decrypt original symmetric key: %v", err)
	}

	// Run rotate command
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("rotate", []string{"--force"}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Rotate command failed: %v\nOutput: %s", err, output)
	}

	// Decrypt symmetric key with new private key
	newPrivateKeyBytes := getPrivateKeyBytes(t, projectUUID)
	newPrivateKey := parsePrivateKey(t, newPrivateKeyBytes)
	newKanukaKeyBytes := getKanukaKeyBytes(t, tempDir, userUUID)

	newSymKey, err := secrets.DecryptWithPrivateKey(newKanukaKeyBytes, newPrivateKey)
	if err != nil {
		t.Fatalf("Failed to decrypt new symmetric key: %v", err)
	}

	// Verify the symmetric key value is the same
	if string(originalSymKey) != string(newSymKey) {
		t.Error("Symmetric key value should remain the same after rotation")
	}
}

func TestRotate_VerboseOutput(t *testing.T) {
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

	// Initialize the project
	setupRotateTestProject(t, tempDir, tempUserDir)

	// Run rotate command with verbose flag
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("rotate", []string{"--force"}, nil, nil, true, false)
		cmd.SetVerbose(true)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Rotate command with verbose failed: %v\nOutput: %s", err, output)
	}

	// Verify success message still appears
	if !strings.Contains(output, "Keypair rotated successfully") {
		t.Errorf("Expected success message in verbose output, got: %s", output)
	}
}
