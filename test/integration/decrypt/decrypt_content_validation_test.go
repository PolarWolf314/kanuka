package decrypt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// Tests that decrypted content matches original.
func TestDecryptAndValidateContent(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	tempUserDir := t.TempDir()
	originalWd, _ := os.Getwd()
	originalUserSettings := configs.UserKanukaSettings

	// Setup test environment
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize project
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create a .env file with specific content
	envFile := filepath.Join(tempDir, ".env")
	originalContent := `DATABASE_URL=postgres://localhost:5432/mydb
API_KEY=secret123
DEBUG=true
PORT=3000
SECRET_TOKEN=abcdef123456
MULTI_LINE_VAR="line1
line2
line3"
SPECIAL_CHARS=!@#$%^&*()_+-={}[]|;':",./<>?
EMPTY_VAR=
SPACES_VAR=  value with spaces  `

	if err := os.WriteFile(envFile, []byte(originalContent), 0600); err != nil {
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

	// Decrypt the file
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

	// Verify .env file was created
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		t.Errorf(".env file was not created after decryption")
		return
	}

	// Read the decrypted content
	decryptedContent, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatalf("Failed to read decrypted .env file: %v", err)
	}

	// Verify content matches exactly
	if string(decryptedContent) != originalContent {
		t.Errorf("Decrypted content does not match original content")
		t.Errorf("Original:\n%s", originalContent)
		t.Errorf("Decrypted:\n%s", string(decryptedContent))
	}
}

// Tests decrypt when encrypted file has wrong format.
func TestDecryptWithInvalidEncryptedFormat(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	tempUserDir := t.TempDir()
	originalWd, _ := os.Getwd()
	originalUserSettings := configs.UserKanukaSettings

	// Setup test environment
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize project
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create an encrypted file with invalid format (valid nonce size but invalid ciphertext)
	encryptedFile := filepath.Join(tempDir, ".env.kanuka")
	invalidData := make([]byte, 30) // 24 bytes for nonce + 6 bytes of invalid ciphertext
	for i := range invalidData {
		invalidData[i] = byte(i % 256) // Fill with invalid data
	}
	if err := os.WriteFile(encryptedFile, invalidData, 0600); err != nil {
		t.Fatalf("Failed to create invalid encrypted file: %v", err)
	}

	// Attempt to decrypt
	output, _ := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should fail due to invalid format
	if !strings.Contains(output, "Failed to decrypt") || !strings.Contains(output, "failed to decrypt ciphertext") {
		t.Errorf("Expected decryption failure message, got: %s", output)
	}
}

// Test: Round-trip encryption/decryption with multiple files.
func TestDecryptMultipleFilesRoundTrip(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	tempUserDir := t.TempDir()
	originalWd, _ := os.Getwd()
	originalUserSettings := configs.UserKanukaSettings

	// Setup test environment
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize project
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create multiple .env files with different content
	envFiles := map[string]string{
		".env":        "DATABASE_URL=postgres://localhost:5432/mydb\nAPI_KEY=secret123\n",
		".env.local":  "DEBUG=true\nLOCAL_VAR=local_value\n",
		".env.prod":   "PRODUCTION=true\nAPI_URL=https://api.example.com\n",
		"config/.env": "CONFIG_VAR=config_value\nNESTED=true\n",
	}

	// Create directory structure and files
	if err := os.MkdirAll(filepath.Join(tempDir, "config"), 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	originalContents := make(map[string]string)
	for filePath, content := range envFiles {
		fullPath := filepath.Join(tempDir, filePath)
		if err := os.WriteFile(fullPath, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to create %s: %v", filePath, err)
		}
		originalContents[filePath] = content
	}

	// Encrypt all files
	_, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt files for test setup: %v", err)
	}

	// Remove all original .env files
	for filePath := range envFiles {
		fullPath := filepath.Join(tempDir, filePath)
		if err := os.Remove(fullPath); err != nil {
			t.Fatalf("Failed to remove .env file %s: %v", fullPath, err)
		}
	}

	// Decrypt all files
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

	// Verify all .env files were recreated with correct content
	for filePath, expectedContent := range originalContents {
		fullPath := filepath.Join(tempDir, filePath)

		// Check file exists
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf(".env file was not recreated at %s", fullPath)
			continue
		}

		// Check content matches
		actualContent, err := os.ReadFile(fullPath)
		if err != nil {
			t.Errorf("Failed to read decrypted file %s: %v", fullPath, err)
			continue
		}

		if string(actualContent) != expectedContent {
			t.Errorf("Content mismatch for %s", filePath)
			t.Errorf("Expected: %s", expectedContent)
			t.Errorf("Actual: %s", string(actualContent))
		}
	}
}
