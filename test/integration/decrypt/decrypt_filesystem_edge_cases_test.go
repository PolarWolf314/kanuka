package decrypt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// Test 19: Decrypt with corrupted encrypted file.
func TestDecryptWithCorruptedEncryptedFile(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	tempUserDir := t.TempDir()
	originalWd, _ := os.Getwd()
	originalUserSettings := configs.UserKanukaSettings

	// Setup test environment
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize project
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create and encrypt a .env file first
	envFile := filepath.Join(tempDir, ".env")
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\nAPI_KEY=secret123\n"
	if err := os.WriteFile(envFile, []byte(envContent), 0600); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Encrypt the file
	_, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt file for test setup: %v", err)
	}

	// Remove original .env file
	if err := os.Remove(envFile); err != nil {
		t.Fatalf("Failed to remove original .env file: %v", err)
	}

	// Corrupt the encrypted file (need at least 24 bytes for nonce + some data)
	encryptedFile := envFile + ".kanuka"
	corruptedData := make([]byte, 50) // 24 bytes for nonce + some corrupted ciphertext
	for i := range corruptedData {
		corruptedData[i] = byte(i % 256) // Fill with predictable but invalid data
	}
	if err := os.WriteFile(encryptedFile, corruptedData, 0600); err != nil {
		t.Fatalf("Failed to corrupt encrypted file: %v", err)
	}

	// Attempt to decrypt
	output, _ := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should fail due to corrupted file
	if !strings.Contains(output, "Failed to decrypt") || !strings.Contains(output, "failed to decrypt ciphertext") {
		t.Errorf("Expected decryption failure message, got: %s", output)
	}
}

// Test 20: Decrypt with read-only encrypted file.
func TestDecryptWithReadOnlyEncryptedFile(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	tempUserDir := t.TempDir()
	originalWd, _ := os.Getwd()
	originalUserSettings := configs.UserKanukaSettings

	// Setup test environment
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize project
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create and encrypt a .env file first
	envFile := filepath.Join(tempDir, ".env")
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\nAPI_KEY=secret123\n"
	if err := os.WriteFile(envFile, []byte(envContent), 0600); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Encrypt the file
	_, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt file for test setup: %v", err)
	}

	// Remove original .env file
	if err := os.Remove(envFile); err != nil {
		t.Fatalf("Failed to remove original .env file: %v", err)
	}

	// Make encrypted file read-only
	encryptedFile := envFile + ".kanuka"
	if err := os.Chmod(encryptedFile, 0400); err != nil {
		t.Fatalf("Failed to make encrypted file read-only: %v", err)
	}

	// Restore permissions for cleanup
	defer func() {
		if err := os.Chmod(encryptedFile, 0600); err != nil {
			t.Logf("Failed to restore permissions on %s: %v", encryptedFile, err)
		}
	}()

	// Attempt to decrypt
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should succeed - read-only encrypted file shouldn't prevent decryption
	if err != nil {
		t.Errorf("Expected command to succeed despite read-only encrypted file, but it failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Should contain success message
	if !strings.Contains(output, "Environment files decrypted successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// Verify .env file was created
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		t.Errorf(".env file was not created after decryption")
	}
}

// Test 21: Encrypted file path is a directory.
func TestDecryptWithEncryptedFileAsDirectory(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	tempUserDir := t.TempDir()
	originalWd, _ := os.Getwd()
	originalUserSettings := configs.UserKanukaSettings

	// Setup test environment
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize project
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create a directory where the encrypted file should be
	encryptedDir := filepath.Join(tempDir, ".env.kanuka")
	if err := os.MkdirAll(encryptedDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Attempt to decrypt
	output, _ := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should fail because there are no valid encrypted files
	if !strings.Contains(output, "No encrypted environment") || !strings.Contains(output, "files found") {
		t.Errorf("Expected 'no encrypted files found' message, got: %s", output)
	}
}

// Test 22: Specific encrypted file doesn't exist.
func TestDecryptWithMissingEncryptedFile(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	tempUserDir := t.TempDir()
	originalWd, _ := os.Getwd()
	originalUserSettings := configs.UserKanukaSettings

	// Setup test environment
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize project
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Don't create any encrypted files

	// Attempt to decrypt
	output, _ := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should fail because there are no encrypted files
	if !strings.Contains(output, "No encrypted environment") || !strings.Contains(output, "files found") {
		t.Errorf("Expected 'no encrypted files found' message, got: %s", output)
	}
}

// Test 23: Very large encrypted file.
func TestDecryptWithVeryLargeEncryptedFile(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	tempUserDir := t.TempDir()
	originalWd, _ := os.Getwd()
	originalUserSettings := configs.UserKanukaSettings

	// Setup test environment
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize project
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create a very large .env file (1MB+)
	envFile := filepath.Join(tempDir, ".env")
	var envContent strings.Builder
	for i := 0; i < 10000; i++ {
		envContent.WriteString("LARGE_VAR_")
		envContent.WriteString(strings.Repeat("X", 100))
		envContent.WriteString("=")
		envContent.WriteString(strings.Repeat("Y", 100))
		envContent.WriteString("\n")
	}

	if err := os.WriteFile(envFile, []byte(envContent.String()), 0600); err != nil {
		t.Fatalf("Failed to create large .env file: %v", err)
	}

	// Encrypt the file
	_, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt large file for test setup: %v", err)
	}

	// Remove original .env file
	if err := os.Remove(envFile); err != nil {
		t.Fatalf("Failed to remove original .env file: %v", err)
	}

	// Attempt to decrypt
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should succeed
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Should contain success message
	if !strings.Contains(output, "Environment files decrypted successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// Verify .env file was created and has correct size
	if fileInfo, err := os.Stat(envFile); err != nil {
		t.Errorf(".env file was not created after decryption")
	} else if fileInfo.Size() < 1000000 { // Should be over 1MB
		t.Errorf("Decrypted file is smaller than expected: %d bytes", fileInfo.Size())
	}
}

// Test 24: Empty encrypted file.
func TestDecryptWithEmptyEncryptedFile(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	tempUserDir := t.TempDir()
	originalWd, _ := os.Getwd()
	originalUserSettings := configs.UserKanukaSettings

	// Setup test environment
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize project
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create an encrypted file with exactly 24 bytes (valid nonce size) but invalid ciphertext
	encryptedFile := filepath.Join(tempDir, ".env.kanuka")
	invalidData := make([]byte, 24) // Exactly 24 bytes for nonce, but no actual ciphertext
	for i := range invalidData {
		invalidData[i] = byte(i) // Fill with invalid nonce data
	}
	if err := os.WriteFile(encryptedFile, invalidData, 0600); err != nil {
		t.Fatalf("Failed to create invalid encrypted file: %v", err)
	}

	// Attempt to decrypt
	output, _ := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should fail due to invalid encrypted data (no ciphertext after nonce)
	if !strings.Contains(output, "Failed to decrypt") || !strings.Contains(output, "failed to decrypt ciphertext") {
		t.Errorf("Expected decryption failure message, got: %s", output)
	}
}
