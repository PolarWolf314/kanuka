package decrypt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// Tests decrypt when private key doesn't match.
func TestDecryptWithWrongPrivateKey(t *testing.T) {
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

	// Replace the private key with a wrong one
	projectName := filepath.Base(tempDir)
	privateKeyPath := filepath.Join(tempUserDir, "keys", projectName)
	// #nosec G101 -- This is a test with intentionally invalid key data
	wrongPrivateKey := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAwrongkey123456789abcdefghijklmnopqrstuvwxyz
-----END RSA PRIVATE KEY-----`
	if err := os.WriteFile(privateKeyPath, []byte(wrongPrivateKey), 0600); err != nil {
		t.Fatalf("Failed to write wrong private key: %v", err)
	}

	// Attempt to decrypt
	output, _ := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should fail due to wrong private key
	if !strings.Contains(output, "Failed to get your private key file") || !strings.Contains(output, "failed to decode PEM block") {
		t.Errorf("Expected private key error message, got: %s", output)
	}
}

// Tests decrypt when private key file is corrupted.
func TestDecryptWithCorruptedPrivateKey(t *testing.T) {
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

	// Corrupt the private key file
	projectName := filepath.Base(tempDir)
	privateKeyPath := filepath.Join(tempUserDir, "keys", projectName)
	if err := os.WriteFile(privateKeyPath, []byte("corrupted private key data"), 0600); err != nil {
		t.Fatalf("Failed to corrupt private key: %v", err)
	}

	// Attempt to decrypt
	output, _ := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should fail due to corrupted private key
	if !strings.Contains(output, "Failed to get your private key file") {
		t.Errorf("Expected private key error message, got: %s", output)
	}
}

// Tests decrypt when private key has wrong format.
func TestDecryptWithWrongKeyFormat(t *testing.T) {
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

	// Replace private key with wrong format (e.g., SSH key format)
	projectName := filepath.Base(tempDir)
	privateKeyPath := filepath.Join(tempUserDir, "keys", projectName)
	wrongFormatKey := `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAFwAAAAdzc2gtcn
-----END OPENSSH PRIVATE KEY-----`
	if err := os.WriteFile(privateKeyPath, []byte(wrongFormatKey), 0600); err != nil {
		t.Fatalf("Failed to write wrong format key: %v", err)
	}

	// Attempt to decrypt
	output, _ := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should fail due to wrong key format
	if !strings.Contains(output, "Failed to get your private key file") {
		t.Errorf("Expected private key error message, got: %s", output)
	}
}

// Tests decrypt when encrypted data has been modified/tampered.
func TestDecryptWithTamperedEncryptedData(t *testing.T) {
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

	// Read the encrypted file and tamper with it
	encryptedFile := envFile + ".kanuka"
	encryptedData, err := os.ReadFile(encryptedFile)
	if err != nil {
		t.Fatalf("Failed to read encrypted file: %v", err)
	}

	// Tamper with the data (flip some bits in the middle)
	if len(encryptedData) > 50 {
		encryptedData[25] ^= 0xFF // Flip bits
		encryptedData[26] ^= 0xFF // Flip bits
	}

	// Write back the tampered data
	if err := os.WriteFile(encryptedFile, encryptedData, 0600); err != nil {
		t.Fatalf("Failed to write tampered encrypted file: %v", err)
	}

	// Attempt to decrypt
	output, _ := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should fail due to tampered data
	if !strings.Contains(output, "Failed to decrypt") || !strings.Contains(output, "failed to decrypt ciphertext") {
		t.Errorf("Expected decryption failure message, got: %s", output)
	}
}

// Tests decrypt when file was encrypted with different algorithm.
func TestDecryptWithWrongEncryptionAlgorithm(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	tempUserDir := t.TempDir()
	originalWd, _ := os.Getwd()
	originalUserSettings := configs.UserKanukaSettings

	// Setup test environment
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize project
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create a fake encrypted file that looks like it was encrypted with a different algorithm
	encryptedFile := filepath.Join(tempDir, ".env.kanuka")
	// This simulates a file encrypted with a different algorithm/format (need 24+ bytes)
	fakeEncryptedData := make([]byte, 100)
	copy(fakeEncryptedData, []byte("DIFFERENT_ALGORITHM_HEADER"))
	for i := 26; i < len(fakeEncryptedData); i++ {
		fakeEncryptedData[i] = byte(i % 256)
	}
	if err := os.WriteFile(encryptedFile, fakeEncryptedData, 0600); err != nil {
		t.Fatalf("Failed to create fake encrypted file: %v", err)
	}

	// Attempt to decrypt
	output, _ := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should fail due to wrong encryption format
	if !strings.Contains(output, "Failed to decrypt") || !strings.Contains(output, "failed to decrypt ciphertext") {
		t.Errorf("Expected decryption failure message, got: %s", output)
	}
}
