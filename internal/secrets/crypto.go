package secrets

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"kanuka/internal/utils"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/nacl/secretbox"
)

// DecryptWithPrivateKey decrypts data using an RSA private key.
func DecryptWithPrivateKey(ciphertext []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	return rsa.DecryptPKCS1v15(rand.Reader, privateKey, ciphertext)
}

func EncryptWithPublicKey(ciphertext []byte, publicKey *rsa.PublicKey) ([]byte, error) {
	return rsa.EncryptPKCS1v15(rand.Reader, publicKey, ciphertext)
}

// CreateSymmetricKey generates a new random symmetric key.
func CreateSymmetricKey() ([]byte, error) {
	symKey := make([]byte, 32) // AES-256
	if _, err := rand.Read(symKey); err != nil {
		return nil, err
	}

	return symKey, nil
}

// CreateAndSaveEncryptedSymmetricKey creates a symmetric key, encrypts it with the user's public key, and saves it.
func CreateAndSaveEncryptedSymmetricKey(verbose bool) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	username, err := utils.GetUsername()
	if err != nil {
		return fmt.Errorf("failed to get username: %w", err)
	}

	kanukaDir := filepath.Join(wd, ".kanuka")
	secretsDir := filepath.Join(kanukaDir, "secrets")
	pubKeyPath := filepath.Join(kanukaDir, "public_keys", username+".pub")

	// 1. create sym key in memory
	symKey, err := CreateSymmetricKey()
	if err != nil {
		return fmt.Errorf("failed to generate symmetric key: %w", err)
	}

	// 2. fetch user's public key from project
	pubKey, err := LoadPublicKey(pubKeyPath)
	if err != nil {
		return fmt.Errorf("failed to load project public key: %w", err)
	}

	// 3. encrypt sym key using public key
	encryptedSymKey, err := EncryptWithPublicKey(symKey, pubKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt symmetric key: %w", err)
	}

	// 4. save sym key to project
	encryptedSymPath := filepath.Join(secretsDir, fmt.Sprintf("%s.kanuka", username))

	if err := os.WriteFile(encryptedSymPath, encryptedSymKey, 0600); err != nil {
		return fmt.Errorf("failed to save encrypted symmetric key: %v", err)
	}

	return nil
}

// EncryptFiles encrypts files using a symmetric key.
func EncryptFiles(symKey []byte, inputPaths []string, verbose bool) error {
	if len(symKey) != 32 {
		return fmt.Errorf("invalid symmetric key length: expected 32 bytes, got %d bytes", len(symKey))
	}

	var key [32]byte
	copy(key[:], symKey)

	for _, inputPath := range inputPaths {
		plaintext, err := os.ReadFile(inputPath)
		if err != nil {
			return fmt.Errorf("failed to read .env file at %s: %w", inputPath, err)
		}

		var nonce [24]byte
		if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
			return fmt.Errorf("failed on ReadFull method: %w", err)
		}

		ciphertext := secretbox.Seal(nonce[:], plaintext, &nonce, &key)

		outputPath := inputPath + ".kanuka"

		if err := os.WriteFile(outputPath, ciphertext, 0600); err != nil {
			return fmt.Errorf("failed to write to %s: %w", outputPath, err)
		}
	}

	return nil
}

// DecryptFiles decrypts files using a symmetric key.
func DecryptFiles(symKey []byte, inputPaths []string, verbose bool) error {
	if len(symKey) != 32 {
		return fmt.Errorf("failed to decrypt files: symmetric key length must be exactly 32 bytes for secretbox")
	}
	var key [32]byte
	copy(key[:], symKey)
	for _, inputPath := range inputPaths {
		ciphertext, err := os.ReadFile(inputPath)
		if err != nil {
			return fmt.Errorf("failed to read .kanuka file at %s: %w", inputPath, err)
		}

		// Extract the nonce from the beginning of the ciphertext
		var decryptNonce [24]byte
		copy(decryptNonce[:], ciphertext[:24])

		// Decrypt using the extracted nonce and the rest of the ciphertext
		plaintext, ok := secretbox.Open(nil, ciphertext[24:], &decryptNonce, &key)
		if !ok {
			return fmt.Errorf("failed to decrypt ciphertext with secretbox")
		}

		outputPath := strings.TrimSuffix(inputPath, ".kanuka")
		// #nosec G306 -- We want the decrypted .env file to be editable by the user
		if err := os.WriteFile(outputPath, plaintext, 0644); err != nil {
			return fmt.Errorf("failed to write to %s: %w", outputPath, err)
		}
	}

	return nil
}
