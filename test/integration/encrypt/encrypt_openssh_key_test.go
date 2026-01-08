package encrypt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestEncryptWithOpenSSHKey tests the encrypt command with OpenSSH format private keys.
func TestEncryptWithOpenSSHKey(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("EncryptWithOpenSSHFormatKey", func(t *testing.T) {
		testEncryptWithOpenSSHFormatKey(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptMultipleFilesWithOpenSSHKey", func(t *testing.T) {
		testEncryptMultipleFilesWithOpenSSHKey(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptAndDecryptRoundtripWithOpenSSHKey", func(t *testing.T) {
		testEncryptAndDecryptRoundtripWithOpenSSHKey(t, originalWd, originalUserSettings)
	})
}

// testEncryptWithOpenSSHFormatKey tests that encryption works when the private key
// is in OpenSSH format.
func testEncryptWithOpenSSHFormatKey(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-openssh-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-encrypt-openssh-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Convert the private key to OpenSSH format BEFORE encryption
	projectUUID := shared.GetProjectUUID(t)
	keysDir := filepath.Join(tempUserDir, "keys")
	privateKeyPath := shared.GetPrivateKeyPath(keysDir, projectUUID)

	if err := shared.ConvertPKCS1ToOpenSSH(privateKeyPath, privateKeyPath); err != nil {
		t.Fatalf("Failed to convert key to OpenSSH format: %v", err)
	}

	// Verify the key was converted to OpenSSH format
	keyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		t.Fatalf("Failed to read converted key: %v", err)
	}
	if !strings.Contains(string(keyData), "OPENSSH PRIVATE KEY") {
		t.Fatalf("Key was not converted to OpenSSH format")
	}

	// Create a .env file to encrypt
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\nAPI_KEY=secret123\n"
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Encrypt using the OpenSSH format key
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "Environment files encrypted successfully") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	// Verify the encrypted file was created
	kanukaPath := envPath + ".kanuka"
	if _, err := os.Stat(kanukaPath); os.IsNotExist(err) {
		t.Errorf(".env.kanuka file was not created at %s", kanukaPath)
	}

	// Verify the encrypted content is different from original (actually encrypted)
	kanukaContent, err := os.ReadFile(kanukaPath)
	if err != nil {
		t.Errorf("Failed to read .env.kanuka file: %v", err)
	}
	if len(kanukaContent) == 0 {
		t.Errorf(".env.kanuka file is empty")
	}
	if string(kanukaContent) == envContent {
		t.Errorf(".env.kanuka file content is the same as .env file (not encrypted)")
	}
}

// testEncryptMultipleFilesWithOpenSSHKey tests encrypting multiple files with OpenSSH key.
func testEncryptMultipleFilesWithOpenSSHKey(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-openssh-multi-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-encrypt-openssh-multi-*")
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

	// Encrypt using the OpenSSH format key
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "Environment files encrypted successfully") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	// Verify all encrypted files were created
	for filePath := range envFiles {
		fullPath := filepath.Join(tempDir, filePath)
		kanukaPath := fullPath + ".kanuka"
		if _, err := os.Stat(kanukaPath); os.IsNotExist(err) {
			t.Errorf(".kanuka file was not created at %s", kanukaPath)
		}
	}
}

// testEncryptAndDecryptRoundtripWithOpenSSHKey tests full roundtrip with OpenSSH key.
func testEncryptAndDecryptRoundtripWithOpenSSHKey(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-roundtrip-openssh-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-roundtrip-openssh-*")
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

	// Create a .env file with complex content
	// #nosec G101 -- These are fake test credentials, not real secrets
	envContent := `DATABASE_URL=postgres://user:password@localhost:5432/mydb
API_KEY=super_secret_key_12345
JWT_SECRET=jwt_token_secret_value
REDIS_URL=redis://localhost:6379
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
`
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Encrypt using the OpenSSH format key
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Encrypt command failed: %v", err)
	}

	// Verify encrypted file exists
	kanukaPath := envPath + ".kanuka"
	if _, err := os.Stat(kanukaPath); os.IsNotExist(err) {
		t.Fatalf(".env.kanuka file was not created")
	}

	// Remove the original .env file to test decryption (simulating typical workflow)
	if err := os.Remove(envPath); err != nil {
		t.Fatalf("Failed to remove .env file: %v", err)
	}

	// Decrypt using the same OpenSSH format key
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Decrypt command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "Environment files decrypted successfully") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	// Verify the decrypted content matches the original
	decryptedContent, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("Failed to read decrypted .env file: %v", err)
	}
	if string(decryptedContent) != envContent {
		t.Errorf("Decrypted content doesn't match original.\nExpected:\n%s\nGot:\n%s", envContent, string(decryptedContent))
	}
}
