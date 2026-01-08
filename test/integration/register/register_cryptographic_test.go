package register

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/PolarWolf314/kanuka/cmd"
	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/secrets"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// addUserToProjectConfig adds a user with the given UUID and email to the project config.
// This is needed before registering a user with --pubkey, as the register command
// looks up users by email to find their UUID.
func addUserToProjectConfig(t *testing.T, userUUID, email string) {
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}

	if projectConfig.Users == nil {
		projectConfig.Users = make(map[string]string)
	}

	projectConfig.Users[userUUID] = email

	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}
}

func TestSecretsRegisterCryptographic(t *testing.T) {
	// Save original state
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings

	t.Run("RegisterWithOpenSSHFormatKey", func(t *testing.T) {
		testRegisterWithOpenSSHFormatKey(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithPEMFormatKey", func(t *testing.T) {
		testRegisterWithPEMFormatKey(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterVerifyEncryptedKeyUniqueness", func(t *testing.T) {
		testRegisterVerifyEncryptedKeyUniqueness(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterVerifyDecryptionWorks", func(t *testing.T) {
		testRegisterVerifyDecryptionWorks(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithDifferentKeySizes", func(t *testing.T) {
		testRegisterWithDifferentKeySizes(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterCrossFormatCompatibility", func(t *testing.T) {
		testRegisterCrossFormatCompatibility(t, originalWd, originalUserSettings)
	})
}

// testRegisterWithOpenSSHFormatKey tests registering a user with an OpenSSH format public key.
func testRegisterWithOpenSSHFormatKey(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-openssh-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Generate a test RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Convert to OpenSSH format
	opensshKey := generateOpenSSHKey(t, &privateKey.PublicKey)
	targetUserEmail := "opensshuser@example.com"
	targetUserUUID := uuid.New().String()

	// Add user to project config so register can find them by email
	addUserToProjectConfig(t, targetUserUUID, targetUserEmail)

	// Reset register command state
	cmd.ResetGlobalState()

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", opensshKey, "--user", targetUserEmail})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success symbol not found in output: %s", output)
	}

	if !strings.Contains(output, targetUserEmail) {
		t.Errorf("Expected target user email not found in output: %s", output)
	}

	// Verify the public key was saved (using UUID, not email)
	pubKeyPath := filepath.Join(tempDir, ".kanuka", "public_keys", targetUserUUID+".pub")
	if _, err := os.Stat(pubKeyPath); os.IsNotExist(err) {
		t.Errorf("Public key file was not created at %s", pubKeyPath)
	}

	// Verify the kanuka key was created (using UUID, not email)
	kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", targetUserUUID+".kanuka")
	if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
		t.Errorf("Kanuka key file was not created at %s", kanukaKeyPath)
	}
}

// testRegisterWithPEMFormatKey tests registering a user with a PEM format public key.
func testRegisterWithPEMFormatKey(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-pem-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Generate a test RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Convert to PEM format
	pemKey := generatePEMKeyCrypto(t, &privateKey.PublicKey)
	targetUserEmail := "pemuser@example.com"
	targetUserUUID := uuid.New().String()

	// Add user to project config so register can find them by email
	addUserToProjectConfig(t, targetUserUUID, targetUserEmail)

	// Reset register command state
	cmd.ResetGlobalState()

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", pemKey, "--user", targetUserEmail})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success symbol not found in output: %s", output)
	}

	if !strings.Contains(output, targetUserEmail) {
		t.Errorf("Expected target user email not found in output: %s", output)
	}

	// Verify the public key was saved (using UUID, not email)
	pubKeyPath := filepath.Join(tempDir, ".kanuka", "public_keys", targetUserUUID+".pub")
	if _, err := os.Stat(pubKeyPath); os.IsNotExist(err) {
		t.Errorf("Public key file was not created at %s", pubKeyPath)
	}

	// Verify the kanuka key was created (using UUID, not email)
	kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", targetUserUUID+".kanuka")
	if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
		t.Errorf("Kanuka key file was not created at %s", kanukaKeyPath)
	}
}

// testRegisterVerifyEncryptedKeyUniqueness verifies each user gets a unique encrypted symmetric key.
func testRegisterVerifyEncryptedKeyUniqueness(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-uniqueness-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Register two different users
	type testUser struct {
		email string
		uuid  string
	}
	users := []testUser{
		{email: "user1@example.com", uuid: uuid.New().String()},
		{email: "user2@example.com", uuid: uuid.New().String()},
	}
	var encryptedKeys [][]byte

	for _, user := range users {
		// Add user to project config so register can find them by email
		addUserToProjectConfig(t, user.uuid, user.email)

		// Generate a test RSA key pair for each user
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("Failed to generate RSA key for %s: %v", user.email, err)
		}

		opensshKey := generateOpenSSHKey(t, &privateKey.PublicKey)

		// Reset register command state
		cmd.ResetGlobalState()

		_, err = shared.CaptureOutput(func() error {
			cmd := shared.CreateTestCLI("register", nil, nil, true, false)
			cmd.SetArgs([]string{"secrets", "register", "--pubkey", opensshKey, "--user", user.email})
			return cmd.Execute()
		})
		if err != nil {
			t.Errorf("Command failed for user %s: %v", user.email, err)
		}

		// Read the encrypted key (using UUID, not email)
		kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", user.uuid+".kanuka")
		encryptedKey, err := os.ReadFile(kanukaKeyPath)
		if err != nil {
			t.Errorf("Failed to read kanuka key for %s: %v", user.email, err)
		}
		encryptedKeys = append(encryptedKeys, encryptedKey)
	}

	// Verify the encrypted keys are different (they should be since they're encrypted with different public keys)
	if len(encryptedKeys) >= 2 && string(encryptedKeys[0]) == string(encryptedKeys[1]) {
		t.Errorf("Encrypted keys for different users are identical, expected them to be different")
	}
}

// testRegisterVerifyDecryptionWorks verifies registered user can decrypt with their private key.
func testRegisterVerifyDecryptionWorks(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-decrypt-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Generate a test RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	opensshKey := generateOpenSSHKey(t, &privateKey.PublicKey)
	targetUserEmail := "decryptuser@example.com"
	targetUserUUID := uuid.New().String()

	// Add user to project config so register can find them by email
	addUserToProjectConfig(t, targetUserUUID, targetUserEmail)

	// Reset register command state
	cmd.ResetGlobalState()

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", opensshKey, "--user", targetUserEmail})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	// Read the encrypted symmetric key (using UUID, not email)
	kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", targetUserUUID+".kanuka")
	encryptedSymKey, err := os.ReadFile(kanukaKeyPath)
	if err != nil {
		t.Fatalf("Failed to read kanuka key: %v", err)
	}

	// Try to decrypt it with the private key
	decryptedSymKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		t.Errorf("Failed to decrypt symmetric key with private key: %v", err)
	}

	// Verify the decrypted key is 32 bytes (AES-256 key size)
	if len(decryptedSymKey) != 32 {
		t.Errorf("Expected decrypted symmetric key to be 32 bytes, got %d bytes", len(decryptedSymKey))
	}
}

// testRegisterWithDifferentKeySizes tests with different RSA key sizes.
func testRegisterWithDifferentKeySizes(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	keySizes := []int{2048, 4096}

	for _, keySize := range keySizes {
		t.Run(fmt.Sprintf("KeySize%d", keySize), func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", fmt.Sprintf("kanuka-test-register-keysize%d-*", keySize))
			if err != nil {
				t.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tempDir)

			tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
			if err != nil {
				t.Fatalf("Failed to create temp user directory: %v", err)
			}
			defer os.RemoveAll(tempUserDir)

			shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
			shared.InitializeProject(t, tempDir, tempUserDir)

			// Generate a test RSA key pair with specific size
			privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
			if err != nil {
				t.Fatalf("Failed to generate %d-bit RSA key: %v", keySize, err)
			}

			opensshKey := generateOpenSSHKey(t, &privateKey.PublicKey)
			targetUserEmail := fmt.Sprintf("keysize%duser@example.com", keySize)
			targetUserUUID := uuid.New().String()

			// Add user to project config so register can find them by email
			addUserToProjectConfig(t, targetUserUUID, targetUserEmail)

			// Reset register command state
			cmd.ResetGlobalState()

			output, err := shared.CaptureOutput(func() error {
				cmd := shared.CreateTestCLI("register", nil, nil, true, false)
				cmd.SetArgs([]string{"secrets", "register", "--pubkey", opensshKey, "--user", targetUserEmail})
				return cmd.Execute()
			})
			if err != nil {
				t.Errorf("Command failed for %d-bit key: %v", keySize, err)
			}

			if !strings.Contains(output, "✓") {
				t.Errorf("Expected success symbol not found for %d-bit key in output: %s", keySize, output)
			}

			// Verify the kanuka key was created and can be decrypted (using UUID, not email)
			kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", targetUserUUID+".kanuka")
			encryptedSymKey, err := os.ReadFile(kanukaKeyPath)
			if err != nil {
				t.Errorf("Failed to read kanuka key for %d-bit key: %v", keySize, err)
			}

			// Try to decrypt it
			_, err = secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
			if err != nil {
				t.Errorf("Failed to decrypt symmetric key with %d-bit private key: %v", keySize, err)
			}
		})
	}
}

// testRegisterCrossFormatCompatibility tests mixing OpenSSH and PEM formats in same project.
func testRegisterCrossFormatCompatibility(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-crossformat-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Define users with their emails and UUIDs
	opensshUserEmail := "opensshuser@example.com"
	opensshUserUUID := uuid.New().String()
	pemUserEmail := "pemuser@example.com"
	pemUserUUID := uuid.New().String()

	// Add both users to project config so register can find them by email
	addUserToProjectConfig(t, opensshUserUUID, opensshUserEmail)
	addUserToProjectConfig(t, pemUserUUID, pemUserEmail)

	// Register one user with OpenSSH format
	privateKey1, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key 1: %v", err)
	}
	opensshKey := generateOpenSSHKey(t, &privateKey1.PublicKey)

	// Reset register command state
	cmd.ResetGlobalState()

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", opensshKey, "--user", opensshUserEmail})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed for OpenSSH user: %v", err)
	}

	// Register another user with PEM format
	privateKey2, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key 2: %v", err)
	}
	pemKey := generatePEMKeyCrypto(t, &privateKey2.PublicKey)

	// Reset register command state
	cmd.ResetGlobalState()

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", pemKey, "--user", pemUserEmail})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed for PEM user: %v", err)
	}

	// Verify both users can decrypt their keys (using UUIDs, not emails)
	users := []struct {
		uuid       string
		email      string
		privateKey *rsa.PrivateKey
	}{
		{opensshUserUUID, opensshUserEmail, privateKey1},
		{pemUserUUID, pemUserEmail, privateKey2},
	}

	for _, user := range users {
		kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", user.uuid+".kanuka")
		encryptedSymKey, err := os.ReadFile(kanukaKeyPath)
		if err != nil {
			t.Errorf("Failed to read kanuka key for %s: %v", user.email, err)
			continue
		}

		_, err = secrets.DecryptWithPrivateKey(encryptedSymKey, user.privateKey)
		if err != nil {
			t.Errorf("Failed to decrypt symmetric key for %s: %v", user.email, err)
		}
	}
}

// Helper functions for cryptographic tests

func generateOpenSSHKey(t *testing.T, publicKey *rsa.PublicKey) string {
	// Generate proper OpenSSH format key
	// SSH key format consists of:
	// - length of "ssh-rsa" (4 bytes)
	// - the string "ssh-rsa"
	// - length of exponent (4 bytes)
	// - exponent
	// - length of modulus (4 bytes)
	// - modulus

	keyType := "ssh-rsa"

	// Convert exponent to bytes
	e := big.NewInt(int64(publicKey.E))
	eBytes := e.Bytes()

	// Get modulus bytes
	nBytes := publicKey.N.Bytes()

	// Build the SSH key data structure
	var keyData []byte

	// Add key type
	keyData = append(keyData, make([]byte, 4)...)
	keyTypeLen := len(keyType)
	if keyTypeLen > 0xFFFFFFFF {
		t.Fatalf("keyType length exceeds uint32 maximum")
	}
	binary.BigEndian.PutUint32(keyData[len(keyData)-4:], uint32(keyTypeLen)) // #nosec G115
	keyData = append(keyData, []byte(keyType)...)

	// Add exponent
	keyData = append(keyData, make([]byte, 4)...)
	eBytesLen := len(eBytes)
	if eBytesLen > 0xFFFFFFFF {
		t.Fatalf("eBytes length exceeds uint32 maximum")
	}
	binary.BigEndian.PutUint32(keyData[len(keyData)-4:], uint32(eBytesLen)) // #nosec G115
	keyData = append(keyData, eBytes...)

	// Add modulus
	keyData = append(keyData, make([]byte, 4)...)
	nBytesLen := len(nBytes)
	if nBytesLen > 0xFFFFFFFF {
		t.Fatalf("nBytes length exceeds uint32 maximum")
	}
	binary.BigEndian.PutUint32(keyData[len(keyData)-4:], uint32(nBytesLen)) // #nosec G115
	keyData = append(keyData, nBytes...)

	// Encode to base64
	encoded := base64.StdEncoding.EncodeToString(keyData)

	return fmt.Sprintf("ssh-rsa %s test@example.com", encoded)
}

func generatePEMKeyCrypto(t *testing.T, publicKey *rsa.PublicKey) string {
	pubASN1, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		t.Fatalf("Failed to marshal public key: %v", err)
	}

	pubPem := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubASN1,
	}

	return string(pem.EncodeToMemory(pubPem))
}
