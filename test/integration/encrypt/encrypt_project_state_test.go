package encrypt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsEncryptProjectState contains project state edge case tests for the `kanuka secrets encrypt` command.
func TestSecretsEncryptProjectState(t *testing.T) {
	// Save original working directory and settings
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings

	// Category 3: Project State Edge Cases
	t.Run("EncryptWithCorruptedKanukaDir", func(t *testing.T) {
		testEncryptWithCorruptedKanukaDir(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptWithMissingPublicKey", func(t *testing.T) {
		testEncryptWithMissingPublicKey(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptWithMissingSymmetricKey", func(t *testing.T) {
		testEncryptWithMissingSymmetricKey(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptWithCorruptedPublicKey", func(t *testing.T) {
		testEncryptWithCorruptedPublicKey(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptWithCorruptedSymmetricKey", func(t *testing.T) {
		testEncryptWithCorruptedSymmetricKey(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptWithWrongKeyFormat", func(t *testing.T) {
		testEncryptWithWrongKeyFormat(t, originalWd, originalUserSettings)
	})
}

// Test 9: .kanuka directory is corrupted/incomplete.
func testEncryptWithCorruptedKanukaDir(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-corrupted-kanuka-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create a .env file
	envFile := filepath.Join(tempDir, ".env")
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\nAPI_KEY=secret123\n"
	if err := os.WriteFile(envFile, []byte(envContent), 0600); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Corrupt the .kanuka directory by removing the secrets subdirectory
	secretsDir := filepath.Join(tempDir, ".kanuka", "secrets")
	if err := os.RemoveAll(secretsDir); err != nil {
		t.Fatalf("Failed to remove secrets directory: %v", err)
	}

	// Capture output and expect failure
	output, _ := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should fail due to corrupted .kanuka directory (missing secrets dir means no symmetric key)
	// The CLI command may not return an error code, but should show failure in output
	if !strings.Contains(output, "Failed to get your .kanuka file") || !strings.Contains(output, "no such file or directory") {
		t.Errorf("Expected error message about missing symmetric key file, got: %s", output)
	}
}

// Test 10: Public key file is missing.
func testEncryptWithMissingPublicKey(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-missing-public-key-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create a .env file
	envFile := filepath.Join(tempDir, ".env")
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\nAPI_KEY=secret123\n"
	if err := os.WriteFile(envFile, []byte(envContent), 0600); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Remove the public key file
	publicKeyFile := filepath.Join(tempDir, ".kanuka", "public_keys", "testuser.pub")
	if err := os.Remove(publicKeyFile); err != nil {
		t.Fatalf("Failed to remove public key file: %v", err)
	}

	// Capture output and expect failure
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	// The encrypt command doesn't use the public key directly - it only needs the symmetric key
	// and private key, so missing public key shouldn't cause failure
	if err != nil {
		t.Errorf("Expected command to succeed despite missing public key, but it failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Should contain success message
	if !strings.Contains(output, "✓") || !strings.Contains(output, "encrypted successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}
}

// Test 11: Symmetric key file is missing.
func testEncryptWithMissingSymmetricKey(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-missing-symmetric-key-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create a .env file
	envFile := filepath.Join(tempDir, ".env")
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\nAPI_KEY=secret123\n"
	if err := os.WriteFile(envFile, []byte(envContent), 0600); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Remove the symmetric key file
	symmetricKeyFile := filepath.Join(tempDir, ".kanuka", "secrets", "testuser.kanuka")
	if err := os.Remove(symmetricKeyFile); err != nil {
		t.Fatalf("Failed to remove symmetric key file: %v", err)
	}

	// Capture output and expect failure
	output, _ := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should fail due to missing symmetric key
	// The CLI command may not return an error code, but should show failure in output
	if !strings.Contains(output, "Failed to get your .kanuka file") || !strings.Contains(output, "no such file or directory") {
		t.Errorf("Expected error message about missing symmetric key file, got: %s", output)
	}
}

// Test 12: Public key file is corrupted.
func testEncryptWithCorruptedPublicKey(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-corrupted-public-key-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create a .env file
	envFile := filepath.Join(tempDir, ".env")
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\nAPI_KEY=secret123\n"
	if err := os.WriteFile(envFile, []byte(envContent), 0600); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Corrupt the public key file
	publicKeyFile := filepath.Join(tempDir, ".kanuka", "public_keys", "testuser.pub")
	corruptedContent := "this is not a valid public key"
	if err := os.WriteFile(publicKeyFile, []byte(corruptedContent), 0600); err != nil {
		t.Fatalf("Failed to corrupt public key file: %v", err)
	}

	// Capture output and expect failure
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	// The encrypt command doesn't use the public key directly, so corrupted public key shouldn't cause failure
	if err != nil {
		t.Errorf("Expected command to succeed despite corrupted public key, but it failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Should contain success message
	if !strings.Contains(output, "✓") || !strings.Contains(output, "encrypted successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}
}

// Test 13: Symmetric key file is corrupted.
func testEncryptWithCorruptedSymmetricKey(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-corrupted-symmetric-key-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create a .env file
	envFile := filepath.Join(tempDir, ".env")
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\nAPI_KEY=secret123\n"
	if err := os.WriteFile(envFile, []byte(envContent), 0600); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Corrupt the symmetric key file
	symmetricKeyFile := filepath.Join(tempDir, ".kanuka", "secrets", "testuser.kanuka")
	corruptedContent := "this is not a valid encrypted symmetric key"
	if err := os.WriteFile(symmetricKeyFile, []byte(corruptedContent), 0600); err != nil {
		t.Fatalf("Failed to corrupt symmetric key file: %v", err)
	}

	// Capture output and expect failure
	output, _ := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should fail due to corrupted symmetric key
	// The CLI command may not return an error code, but should show failure in output
	if !strings.Contains(output, "Failed to decrypt your .kanuka file") || !strings.Contains(output, "decryption error") {
		t.Errorf("Expected error message about decryption failure, got: %s", output)
	}
}

// Test 14: Key files have wrong format/content.
func testEncryptWithWrongKeyFormat(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-wrong-key-format-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create a .env file
	envFile := filepath.Join(tempDir, ".env")
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\nAPI_KEY=secret123\n"
	if err := os.WriteFile(envFile, []byte(envContent), 0600); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Replace public key with wrong format (valid PEM but wrong key type)
	publicKeyFile := filepath.Join(tempDir, ".kanuka", "public_keys", "testuser.pub")
	wrongFormatKey := `-----BEGIN CERTIFICATE-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA1234567890abcdef
-----END CERTIFICATE-----`
	if err := os.WriteFile(publicKeyFile, []byte(wrongFormatKey), 0600); err != nil {
		t.Fatalf("Failed to write wrong format key: %v", err)
	}

	// Capture output and expect failure
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	// The encrypt command doesn't validate public key format during encryption
	// It only uses the symmetric key and private key, so wrong public key format shouldn't cause failure
	if err != nil {
		t.Errorf("Expected command to succeed despite wrong public key format, but it failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Should contain success message
	if !strings.Contains(output, "✓") || !strings.Contains(output, "encrypted successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}
}
