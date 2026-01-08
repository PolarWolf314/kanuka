package decrypt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestDecryptWithPrivateKeyStdin tests the decrypt command with --private-key-stdin flag.
func TestDecryptWithPrivateKeyStdin(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("DecryptWithPKCS1KeyFromStdin", func(t *testing.T) {
		testDecryptWithPKCS1KeyFromStdin(t, originalWd, originalUserSettings)
	})

	t.Run("DecryptWithOpenSSHKeyFromStdin", func(t *testing.T) {
		testDecryptWithOpenSSHKeyFromStdin(t, originalWd, originalUserSettings)
	})

	t.Run("DecryptWithEmptyStdin", func(t *testing.T) {
		testDecryptWithEmptyStdin(t, originalWd, originalUserSettings)
	})

	t.Run("DecryptWithInvalidKeyFromStdin", func(t *testing.T) {
		testDecryptWithInvalidKeyFromStdin(t, originalWd, originalUserSettings)
	})
}

// testDecryptWithPKCS1KeyFromStdin tests decryption with a PKCS#1 format private key from stdin.
func testDecryptWithPKCS1KeyFromStdin(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-stdin-pkcs1-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-stdin-pkcs1-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create and encrypt a .env file
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\nAPI_KEY=secret123\n"
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Encrypt the file first
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt file for test setup: %v", err)
	}

	// Remove the original .env file to test decryption
	if err := os.Remove(envPath); err != nil {
		t.Fatalf("Failed to remove .env file: %v", err)
	}

	// Read the private key from disk
	projectUUID := shared.GetProjectUUID(t)
	keysDir := filepath.Join(tempUserDir, "keys")
	privateKeyPath := shared.GetPrivateKeyPath(keysDir, projectUUID)
	privateKeyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		t.Fatalf("Failed to read private key: %v", err)
	}

	// Decrypt using the private key from stdin
	output, err := shared.CaptureOutputWithStdin(privateKeyData, func() error {
		cmd := shared.CreateTestCLIWithArgs("decrypt", []string{"--private-key-stdin"}, nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "Environment files decrypted successfully") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	// Verify the .env file was recreated with correct content
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		t.Errorf(".env file was not recreated at %s", envPath)
		return
	}

	decryptedContent, err := os.ReadFile(envPath)
	if err != nil {
		t.Errorf("Failed to read decrypted .env file: %v", err)
		return
	}
	if string(decryptedContent) != envContent {
		t.Errorf("Decrypted content doesn't match original. Expected: %s, Got: %s", envContent, string(decryptedContent))
	}
}

// testDecryptWithOpenSSHKeyFromStdin tests decryption with an OpenSSH format private key from stdin.
func testDecryptWithOpenSSHKeyFromStdin(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-stdin-openssh-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-stdin-openssh-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create and encrypt a .env file
	envContent := "SECRET_KEY=openssh-test-value\nDEBUG=true\n"
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Encrypt the file first (uses PKCS#1 key from init)
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt file for test setup: %v", err)
	}

	// Remove the original .env file
	if err := os.Remove(envPath); err != nil {
		t.Fatalf("Failed to remove .env file: %v", err)
	}

	// Convert the private key to OpenSSH format
	projectUUID := shared.GetProjectUUID(t)
	keysDir := filepath.Join(tempUserDir, "keys")
	privateKeyPath := shared.GetPrivateKeyPath(keysDir, projectUUID)

	if err := shared.ConvertPKCS1ToOpenSSH(privateKeyPath, privateKeyPath); err != nil {
		t.Fatalf("Failed to convert key to OpenSSH format: %v", err)
	}

	// Read the OpenSSH format private key
	privateKeyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		t.Fatalf("Failed to read private key: %v", err)
	}

	// Verify it's in OpenSSH format
	if !strings.Contains(string(privateKeyData), "OPENSSH PRIVATE KEY") {
		t.Fatalf("Key was not in OpenSSH format")
	}

	// Decrypt using the OpenSSH private key from stdin
	output, err := shared.CaptureOutputWithStdin(privateKeyData, func() error {
		cmd := shared.CreateTestCLIWithArgs("decrypt", []string{"--private-key-stdin"}, nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "Environment files decrypted successfully") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	// Verify the .env file was recreated with correct content
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		t.Errorf(".env file was not recreated at %s", envPath)
		return
	}

	decryptedContent, err := os.ReadFile(envPath)
	if err != nil {
		t.Errorf("Failed to read decrypted .env file: %v", err)
		return
	}
	if string(decryptedContent) != envContent {
		t.Errorf("Decrypted content doesn't match original. Expected: %s, Got: %s", envContent, string(decryptedContent))
	}
}

// testDecryptWithEmptyStdin tests that decryption fails gracefully with empty stdin.
func testDecryptWithEmptyStdin(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-stdin-empty-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-stdin-empty-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create and encrypt a .env file
	envContent := "TEST_VAR=value\n"
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Encrypt the file
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt file for test setup: %v", err)
	}

	// Remove the original .env file
	if err := os.Remove(envPath); err != nil {
		t.Fatalf("Failed to remove .env file: %v", err)
	}

	// Try to decrypt with empty stdin
	// Note: The command returns nil error but outputs a failure message
	// We use verbose mode to ensure the message is printed via fmt.Print
	output, _ := shared.CaptureOutputWithStdin([]byte{}, func() error {
		cmd := shared.CreateTestCLIWithArgs("decrypt", []string{"--private-key-stdin"}, nil, nil, true, false)
		return cmd.Execute()
	})

	// The command should output a message about stdin being empty
	if !strings.Contains(output, "stdin is empty") {
		t.Errorf("Expected 'stdin is empty' message in output, got: %s", output)
	}

	// Verify the command indicates failure
	if !strings.Contains(output, "Failed to read private key from stdin") {
		t.Errorf("Expected failure message about reading private key from stdin, got: %s", output)
	}
}

// testDecryptWithInvalidKeyFromStdin tests that decryption fails gracefully with invalid key data.
func testDecryptWithInvalidKeyFromStdin(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-stdin-invalid-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-stdin-invalid-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create and encrypt a .env file
	envContent := "TEST_VAR=value\n"
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Encrypt the file
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt file for test setup: %v", err)
	}

	// Remove the original .env file
	if err := os.Remove(envPath); err != nil {
		t.Fatalf("Failed to remove .env file: %v", err)
	}

	// Try to decrypt with invalid key data
	// Note: The command returns nil error but outputs a failure message
	// We use verbose mode to ensure the message is printed via fmt.Print
	invalidKeyData := []byte("this is not a valid private key")
	output, _ := shared.CaptureOutputWithStdin(invalidKeyData, func() error {
		cmd := shared.CreateTestCLIWithArgs("decrypt", []string{"--private-key-stdin"}, nil, nil, true, false)
		return cmd.Execute()
	})

	// The command should output a message about failing to parse the key
	if !strings.Contains(output, "Failed to parse private key from stdin") {
		t.Errorf("Expected 'Failed to parse private key from stdin' message in output, got: %s", output)
	}
}
