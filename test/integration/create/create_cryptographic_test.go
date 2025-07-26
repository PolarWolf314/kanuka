package create

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/secrets"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsCreateCryptographic contains cryptographic validation tests for the `kanuka secrets create` command.
func TestSecretsCreateCryptographic(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("RSAKeyGeneration", func(t *testing.T) {
		testRSAKeyGeneration(t, originalWd, originalUserSettings)
	})

	t.Run("PEMFormatValidation", func(t *testing.T) {
		testPEMFormatValidation(t, originalWd, originalUserSettings)
	})

	t.Run("KeyPairMatching", func(t *testing.T) {
		testKeyPairMatching(t, originalWd, originalUserSettings)
	})

	t.Run("KeyUniqueness", func(t *testing.T) {
		testKeyUniqueness(t, originalWd, originalUserSettings)
	})
}

// Tests RSA key generation - verify 2048-bit RSA keys are generated.
func testRSAKeyGeneration(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-rsa-*")
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

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	projectName := filepath.Base(tempDir)
	privateKeyPath := filepath.Join(tempUserDir, "keys", projectName)

	// Load and validate the private key
	privateKey, err := secrets.LoadPrivateKey(privateKeyPath)
	if err != nil {
		t.Fatalf("Failed to load private key: %v", err)
	}

	// Verify it's an RSA key and not nil
	if privateKey == nil { //nolint:staticcheck // This check is intentional to ensure privateKey is not nil
		t.Fatalf("Private key is nil")
	}

	// Verify key size is 2048 bits (privateKey is guaranteed non-nil here)
	keySize := privateKey.N.BitLen() //nolint:staticcheck // privateKey is guaranteed non-nil after check above
	if keySize != 2048 {
		t.Errorf("Expected 2048-bit RSA key, got %d bits", keySize)
	}

	// Verify key can be used for encryption/decryption (privateKey is guaranteed non-nil here)
	testMessage := []byte("test message for encryption")
	encrypted, err := secrets.EncryptWithPublicKey(testMessage, &privateKey.PublicKey) //nolint:staticcheck // privateKey is guaranteed non-nil after check above
	if err != nil {
		t.Errorf("Failed to encrypt with generated public key: %v", err)
	}

	decrypted, err := secrets.DecryptWithPrivateKey(encrypted, privateKey)
	if err != nil {
		t.Errorf("Failed to decrypt with generated private key: %v", err)
	}

	if string(decrypted) != string(testMessage) {
		t.Errorf("Decrypted message doesn't match original. Expected: %s, Got: %s", testMessage, decrypted)
	}
}

// Tests PEM format validation - verify keys are in correct PEM format.
func testPEMFormatValidation(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-pem-*")
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

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	projectName := filepath.Base(tempDir)
	privateKeyPath := filepath.Join(tempUserDir, "keys", projectName)
	publicKeyPath := filepath.Join(tempUserDir, "keys", projectName+".pub")

	// Test private key PEM format
	privateKeyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		t.Fatalf("Failed to read private key: %v", err)
	}

	// Verify PEM structure
	if !strings.HasPrefix(string(privateKeyData), "-----BEGIN RSA PRIVATE KEY-----") {
		t.Errorf("Private key doesn't start with correct PEM header")
	}
	if !strings.HasSuffix(strings.TrimSpace(string(privateKeyData)), "-----END RSA PRIVATE KEY-----") {
		t.Errorf("Private key doesn't end with correct PEM footer")
	}

	// Verify PEM can be decoded
	block, _ := pem.Decode(privateKeyData)
	if block == nil {
		t.Errorf("Failed to decode private key PEM block")
	} else if block.Type != "RSA PRIVATE KEY" {
		t.Errorf("Expected RSA PRIVATE KEY block type, got: %s", block.Type)
	}

	// Verify private key can be parsed
	_, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		t.Errorf("Failed to parse private key: %v", err)
	}

	// Test public key PEM format
	publicKeyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		t.Fatalf("Failed to read public key: %v", err)
	}

	// Verify PEM structure
	if !strings.HasPrefix(string(publicKeyData), "-----BEGIN PUBLIC KEY-----") {
		t.Errorf("Public key doesn't start with correct PEM header")
	}
	if !strings.HasSuffix(strings.TrimSpace(string(publicKeyData)), "-----END PUBLIC KEY-----") {
		t.Errorf("Public key doesn't end with correct PEM footer")
	}

	// Verify PEM can be decoded
	block, _ = pem.Decode(publicKeyData)
	if block == nil {
		t.Errorf("Failed to decode public key PEM block")
	} else if block.Type != "PUBLIC KEY" {
		t.Errorf("Expected PUBLIC KEY block type, got: %s", block.Type)
	}

	// Verify public key can be parsed
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		t.Errorf("Failed to parse public key: %v", err)
	}

	// Verify it's an RSA public key
	_, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		t.Errorf("Parsed public key is not an RSA public key")
	}
}

// Tests key pair matching - verify private and public keys are mathematically related.
func testKeyPairMatching(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-keypair-*")
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

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	projectName := filepath.Base(tempDir)
	privateKeyPath := filepath.Join(tempUserDir, "keys", projectName)
	publicKeyPath := filepath.Join(tempUserDir, "keys", projectName+".pub")

	// Load both keys
	privateKey, err := secrets.LoadPrivateKey(privateKeyPath)
	if err != nil {
		t.Fatalf("Failed to load private key: %v", err)
	}

	publicKey, err := secrets.LoadPublicKey(publicKeyPath)
	if err != nil {
		t.Fatalf("Failed to load public key: %v", err)
	}

	// Verify the public key from the private key matches the loaded public key
	if privateKey.N.Cmp(publicKey.N) != 0 {
		t.Errorf("Public key modulus doesn't match private key's public key modulus")
	}

	if privateKey.E != publicKey.E {
		t.Errorf("Public key exponent doesn't match private key's public key exponent")
	}

	// Test encryption/decryption to verify mathematical relationship
	testMessage := []byte("test message for key pair validation")

	// Encrypt with public key
	encrypted, err := secrets.EncryptWithPublicKey(testMessage, publicKey)
	if err != nil {
		t.Errorf("Failed to encrypt with public key: %v", err)
	}

	// Decrypt with private key
	decrypted, err := secrets.DecryptWithPrivateKey(encrypted, privateKey)
	if err != nil {
		t.Errorf("Failed to decrypt with private key: %v", err)
	}

	if string(decrypted) != string(testMessage) {
		t.Errorf("Key pair validation failed. Expected: %s, Got: %s", testMessage, decrypted)
	}

	// Also verify the project public key matches
	username := configs.UserKanukaSettings.Username
	projectPublicKeyPath := filepath.Join(tempDir, ".kanuka", "public_keys", username+".pub")

	projectPublicKey, err := secrets.LoadPublicKey(projectPublicKeyPath)
	if err != nil {
		t.Errorf("Failed to load project public key: %v", err)
	} else {
		if publicKey.N.Cmp(projectPublicKey.N) != 0 || publicKey.E != projectPublicKey.E {
			t.Errorf("Project public key doesn't match user public key")
		}
	}
}

// Tests key uniqueness - verify each generation creates unique keys.
func testKeyUniqueness(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Generate multiple key pairs and verify they're all different
	var privateKeys []*rsa.PrivateKey
	var publicKeys []*rsa.PublicKey

	for i := 0; i < 3; i++ {
		tempDir, err := os.MkdirTemp("", "kanuka-test-unique-*")
		if err != nil {
			t.Fatalf("Failed to create temp directory %d: %v", i, err)
		}
		defer os.RemoveAll(tempDir)

		tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
		if err != nil {
			t.Fatalf("Failed to create temp user directory %d: %v", i, err)
		}
		defer os.RemoveAll(tempUserDir)

		shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
		shared.InitializeProject(t, tempDir, tempUserDir)

		_, err = shared.CaptureOutput(func() error {
			cmd := shared.CreateTestCLI("create", nil, nil, true, false)
			return cmd.Execute()
		})
		if err != nil {
			t.Errorf("Command failed for iteration %d: %v", i, err)
			continue
		}

		projectName := filepath.Base(tempDir)
		privateKeyPath := filepath.Join(tempUserDir, "keys", projectName)
		publicKeyPath := filepath.Join(tempUserDir, "keys", projectName+".pub")

		privateKey, err := secrets.LoadPrivateKey(privateKeyPath)
		if err != nil {
			t.Errorf("Failed to load private key %d: %v", i, err)
			continue
		}

		publicKey, err := secrets.LoadPublicKey(publicKeyPath)
		if err != nil {
			t.Errorf("Failed to load public key %d: %v", i, err)
			continue
		}

		privateKeys = append(privateKeys, privateKey)
		publicKeys = append(publicKeys, publicKey)
	}

	// Compare all keys to ensure they're unique
	for i := 0; i < len(privateKeys); i++ {
		for j := i + 1; j < len(privateKeys); j++ {
			if privateKeys[i].N.Cmp(privateKeys[j].N) == 0 {
				t.Errorf("Private keys %d and %d have the same modulus (not unique)", i, j)
			}
			if publicKeys[i].N.Cmp(publicKeys[j].N) == 0 {
				t.Errorf("Public keys %d and %d have the same modulus (not unique)", i, j)
			}
		}
	}

	// Verify we actually generated the expected number of unique keys
	if len(privateKeys) < 3 {
		t.Errorf("Expected to generate 3 unique key pairs, only got %d", len(privateKeys))
	}
}
