package encrypt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestEncryptWithPrivateKeyStdin tests the encrypt command with --private-key-stdin flag.
func TestEncryptWithPrivateKeyStdin(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("EncryptWithPKCS1KeyFromStdin", func(t *testing.T) {
		testEncryptWithPKCS1KeyFromStdin(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptWithOpenSSHKeyFromStdin", func(t *testing.T) {
		testEncryptWithOpenSSHKeyFromStdin(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptWithEmptyStdin", func(t *testing.T) {
		testEncryptWithEmptyStdin(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptWithInvalidKeyFromStdin", func(t *testing.T) {
		testEncryptWithInvalidKeyFromStdin(t, originalWd, originalUserSettings)
	})
}

// testEncryptWithPKCS1KeyFromStdin tests encryption with a PKCS#1 format private key from stdin.
func testEncryptWithPKCS1KeyFromStdin(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-stdin-pkcs1-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-encrypt-stdin-pkcs1-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create a .env file to encrypt
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\nAPI_KEY=secret123\n"
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Read the private key from disk
	projectUUID := shared.GetProjectUUID(t)
	keysDir := filepath.Join(tempUserDir, "keys")
	privateKeyPath := shared.GetPrivateKeyPath(keysDir, projectUUID)
	privateKeyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		t.Fatalf("Failed to read private key: %v", err)
	}

	// Encrypt using the private key from stdin
	output, err := shared.CaptureOutputWithStdin(privateKeyData, func() error {
		cmd := shared.CreateTestCLIWithArgs("encrypt", []string{"--private-key-stdin"}, nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
		return
	}

	// Check for any failure message in output (useful for debugging)
	if strings.Contains(output, "✗") || strings.Contains(output, "Failed") {
		t.Errorf("Command output indicates failure: %s", output)
		return
	}

	if !strings.Contains(output, "Environment files encrypted successfully") {
		t.Errorf("Expected success message not found in output: %s", output)
		return
	}

	// Verify the encrypted file was created (next to the original .env file)
	encryptedFile := envPath + ".kanuka"
	if _, err := os.Stat(encryptedFile); os.IsNotExist(err) {
		t.Errorf("Encrypted file was not created at %s", encryptedFile)
		return
	}

	// Remove the .env file and decrypt to verify the encryption worked
	if err := os.Remove(envPath); err != nil {
		t.Fatalf("Failed to remove .env file: %v", err)
	}

	// Decrypt to verify
	_, err = shared.CaptureOutputWithStdin(privateKeyData, func() error {
		cmd := shared.CreateTestCLIWithArgs("decrypt", []string{"--private-key-stdin"}, nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Decrypt verification failed: %v", err)
	}

	// Verify the .env file was recreated with correct content
	decryptedContent, err := os.ReadFile(envPath)
	if err != nil {
		t.Errorf("Failed to read decrypted .env file: %v", err)
		return
	}
	if string(decryptedContent) != envContent {
		t.Errorf("Decrypted content doesn't match original. Expected: %s, Got: %s", envContent, string(decryptedContent))
	}
}

// testEncryptWithOpenSSHKeyFromStdin tests encryption with an OpenSSH format private key from stdin.
func testEncryptWithOpenSSHKeyFromStdin(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-stdin-openssh-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-encrypt-stdin-openssh-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

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

	// Create a .env file to encrypt
	envContent := "SECRET_KEY=openssh-encrypt-test\nDEBUG=true\n"
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Encrypt using the OpenSSH private key from stdin
	output, err := shared.CaptureOutputWithStdin(privateKeyData, func() error {
		cmd := shared.CreateTestCLIWithArgs("encrypt", []string{"--private-key-stdin"}, nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "Environment files encrypted successfully") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	// Verify the encrypted file was created (next to the original .env file)
	encryptedFile := envPath + ".kanuka"
	if _, err := os.Stat(encryptedFile); os.IsNotExist(err) {
		t.Errorf("Encrypted file was not created at %s", encryptedFile)
	}

	// Remove the .env file and decrypt to verify the encryption worked
	if err := os.Remove(envPath); err != nil {
		t.Fatalf("Failed to remove .env file: %v", err)
	}

	// Decrypt to verify
	_, err = shared.CaptureOutputWithStdin(privateKeyData, func() error {
		cmd := shared.CreateTestCLIWithArgs("decrypt", []string{"--private-key-stdin"}, nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Decrypt verification failed: %v", err)
	}

	// Verify the .env file was recreated with correct content
	decryptedContent, err := os.ReadFile(envPath)
	if err != nil {
		t.Errorf("Failed to read decrypted .env file: %v", err)
		return
	}
	if string(decryptedContent) != envContent {
		t.Errorf("Decrypted content doesn't match original. Expected: %s, Got: %s", envContent, string(decryptedContent))
	}
}

// testEncryptWithEmptyStdin tests that encryption fails gracefully with empty stdin.
func testEncryptWithEmptyStdin(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-stdin-empty-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-encrypt-stdin-empty-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create a .env file to encrypt
	envContent := "TEST_VAR=value\n"
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Try to encrypt with empty stdin
	// Note: The command returns nil error but outputs a failure message
	// We use verbose mode to ensure the message is printed via fmt.Print
	output, _ := shared.CaptureOutputWithStdin([]byte{}, func() error {
		cmd := shared.CreateTestCLIWithArgs("encrypt", []string{"--private-key-stdin"}, nil, nil, true, false)
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

// testEncryptWithInvalidKeyFromStdin tests that encryption fails gracefully with invalid key data.
func testEncryptWithInvalidKeyFromStdin(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-stdin-invalid-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-encrypt-stdin-invalid-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create a .env file to encrypt
	envContent := "TEST_VAR=value\n"
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Try to encrypt with invalid key data
	// Note: The command returns nil error but outputs a failure message
	// We use verbose mode to ensure the message is printed via fmt.Print
	invalidKeyData := []byte("this is not a valid private key")
	output, _ := shared.CaptureOutputWithStdin(invalidKeyData, func() error {
		cmd := shared.CreateTestCLIWithArgs("encrypt", []string{"--private-key-stdin"}, nil, nil, true, false)
		return cmd.Execute()
	})

	// The command should output a message about failing to parse the key
	if !strings.Contains(output, "Failed to parse private key from stdin") {
		t.Errorf("Expected 'Failed to parse private key from stdin' message in output, got: %s", output)
	}
}
