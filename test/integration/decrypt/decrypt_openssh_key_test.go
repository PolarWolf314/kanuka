package decrypt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestDecryptWithOpenSSHKey tests the decrypt command with OpenSSH format private keys.
func TestDecryptWithOpenSSHKey(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("DecryptWithOpenSSHFormatKey", func(t *testing.T) {
		testDecryptWithOpenSSHFormatKey(t, originalWd, originalUserSettings)
	})

	t.Run("DecryptWithConvertedOpenSSHKey", func(t *testing.T) {
		testDecryptWithConvertedOpenSSHKey(t, originalWd, originalUserSettings)
	})
}

// testDecryptWithOpenSSHFormatKey tests that decryption works when the private key
// is converted to OpenSSH format after the project is initialized.
func testDecryptWithOpenSSHFormatKey(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-openssh-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-openssh-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create and encrypt a .env file first
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\nAPI_KEY=secret123\n"
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Encrypt the file first (uses the original PKCS#1 key)
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

	// Now convert the private key to OpenSSH format
	projectUUID := shared.GetProjectUUID(t)
	keysDir := filepath.Join(tempUserDir, "keys")
	privateKeyPath := shared.GetPrivateKeyPath(keysDir, projectUUID)

	// Convert PKCS#1 to OpenSSH format
	if err := shared.ConvertPKCS1ToOpenSSH(privateKeyPath, privateKeyPath); err != nil {
		t.Fatalf("Failed to convert key to OpenSSH format: %v", err)
	}

	// Verify the key was converted to OpenSSH format
	keyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		t.Fatalf("Failed to read converted key: %v", err)
	}
	if !strings.Contains(string(keyData), "OPENSSH PRIVATE KEY") {
		t.Fatalf("Key was not converted to OpenSSH format. Got: %s", string(keyData)[:100])
	}

	// Decrypt using the OpenSSH format key
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
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

// testDecryptWithConvertedOpenSSHKey tests decrypt with multiple files using OpenSSH key.
func testDecryptWithConvertedOpenSSHKey(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-decrypt-openssh-multi-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-openssh-multi-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create multiple .env files
	envFiles := map[string]string{
		".env":       "DATABASE_URL=postgres://localhost:5432/mydb\n",
		".env.local": "API_KEY=secret123\n",
	}

	for filePath, content := range envFiles {
		fullPath := filepath.Join(tempDir, filePath)
		// #nosec G306 -- Writing a file that should be modifiable
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create .env file %s: %v", fullPath, err)
		}
	}

	// Encrypt all files first
	_, err = shared.CaptureOutput(func() error {
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

	// Convert the private key to OpenSSH format
	projectUUID := shared.GetProjectUUID(t)
	keysDir := filepath.Join(tempUserDir, "keys")
	privateKeyPath := shared.GetPrivateKeyPath(keysDir, projectUUID)

	if err := shared.ConvertPKCS1ToOpenSSH(privateKeyPath, privateKeyPath); err != nil {
		t.Fatalf("Failed to convert key to OpenSSH format: %v", err)
	}

	// Decrypt using the OpenSSH format key
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "Environment files decrypted successfully") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	// Verify all files were decrypted correctly
	for filePath, expectedContent := range envFiles {
		fullPath := filepath.Join(tempDir, filePath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf(".env file was not recreated at %s", fullPath)
			continue
		}

		decryptedContent, err := os.ReadFile(fullPath)
		if err != nil {
			t.Errorf("Failed to read decrypted .env file %s: %v", fullPath, err)
			continue
		}
		if string(decryptedContent) != expectedContent {
			t.Errorf("Decrypted content doesn't match original for %s. Expected: %s, Got: %s", filePath, expectedContent, string(decryptedContent))
		}
	}
}
