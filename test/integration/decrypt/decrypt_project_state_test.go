package decrypt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// Test 30: .kanuka directory is corrupted.
func TestDecryptWithCorruptedKanukaDir(t *testing.T) {
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

	// Corrupt the .kanuka directory by removing the secrets directory
	secretsDir := filepath.Join(tempDir, ".kanuka", "secrets")
	if err := os.RemoveAll(secretsDir); err != nil {
		t.Fatalf("Failed to remove secrets directory: %v", err)
	}

	// Attempt to decrypt
	output, _ := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should fail due to corrupted .kanuka directory (missing symmetric key)
	if !strings.Contains(output, "Failed to obtain your .kanuka file") || !strings.Contains(output, "no such file or directory") {
		t.Errorf("Expected missing symmetric key error message, got: %s", output)
	}
}

// Test 31: User key files are missing.
func TestDecryptWithMissingUserKeys(t *testing.T) {
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

	// Remove user key files
	userKeysDir := filepath.Join(tempUserDir, "keys")
	if err := os.RemoveAll(userKeysDir); err != nil {
		t.Fatalf("Failed to remove user keys directory: %v", err)
	}

	// Attempt to decrypt
	output, _ := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should fail due to missing user keys
	if !strings.Contains(output, "Failed to get your private key file") || !strings.Contains(output, "no such file or directory") {
		t.Errorf("Expected missing private key error message, got: %s", output)
	}
}
