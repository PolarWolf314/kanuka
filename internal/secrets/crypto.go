package secrets

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/configs"

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
// Uses user UUID for file naming.
func CreateAndSaveEncryptedSymmetricKey(verbose bool) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Ensure user config has UUID
	userConfig, err := configs.EnsureUserConfig()
	if err != nil {
		return fmt.Errorf("failed to ensure user config: %w", err)
	}

	userUUID := userConfig.User.UUID
	if userUUID == "" {
		return fmt.Errorf("user UUID not found in user config")
	}

	// Project hasn't been made at this point yet, so do it relative to working directory.
	kanukaDir := filepath.Join(wd, ".kanuka")
	secretsDir := filepath.Join(kanukaDir, "secrets")
	// Public key is named with user UUID
	pubKeyPath := filepath.Join(kanukaDir, "public_keys", userUUID+".pub")

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

	// 4. save sym key to project using user UUID
	encryptedSymPath := filepath.Join(secretsDir, fmt.Sprintf("%s.kanuka", userUUID))

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

// RotateSymmetricKey rotates the symmetric key for all users in the project.
// It generates a new symmetric key, encrypts it for all users, and re-encrypts all files.
// currentUserUUID is the UUID of the user performing the rotation.
func RotateSymmetricKey(currentUserUUID string, privateKey *rsa.PrivateKey, verbose bool) error {
	if err := configs.InitProjectSettings(); err != nil {
		return fmt.Errorf("failed to init project settings: %w", err)
	}

	projectPath := configs.ProjectKanukaSettings.ProjectPath
	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath

	// Get all user UUIDs in the project
	userUUIDs, err := GetAllUsersInProject()
	if err != nil {
		return fmt.Errorf("failed to get list of users: %w", err)
	}

	if len(userUUIDs) == 0 {
		return fmt.Errorf("no users found in project")
	}

	// Get current encrypted symmetric key
	currentEncryptedSymKey, err := GetProjectKanukaKey(currentUserUUID)
	if err != nil {
		return fmt.Errorf("failed to get current symmetric key for user %s: %w", currentUserUUID, err)
	}

	// Decrypt current symmetric key
	currentSymKey, err := DecryptWithPrivateKey(currentEncryptedSymKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt current symmetric key: %w", err)
	}

	// Decrypt all .kanuka files to get plaintext
	kanukaFiles, err := FindEnvOrKanukaFiles(projectPath, []string{}, true)
	if err != nil {
		return fmt.Errorf("failed to find .kanuka files: %w", err)
	}

	var plaintexts []struct {
		Path    string
		Content []byte
		NewPath string
	}

	for _, kanukaFile := range kanukaFiles {
		var key [32]byte
		copy(key[:], currentSymKey)

		ciphertext, err := os.ReadFile(kanukaFile)
		if err != nil {
			return fmt.Errorf("failed to read .kanuka file %s: %w", kanukaFile, err)
		}

		var decryptNonce [24]byte
		copy(decryptNonce[:], ciphertext[:24])

		plaintext, ok := secretbox.Open(nil, ciphertext[24:], &decryptNonce, &key)
		if !ok {
			return fmt.Errorf("failed to decrypt file %s", kanukaFile)
		}

		plaintexts = append(plaintexts, struct {
			Path    string
			Content []byte
			NewPath string
		}{
			Path:    kanukaFile,
			Content: plaintext,
			NewPath: strings.TrimSuffix(kanukaFile, ".kanuka"),
		})
	}

	// Generate new symmetric key
	newSymKey, err := CreateSymmetricKey()
	if err != nil {
		return fmt.Errorf("failed to generate new symmetric key: %w", err)
	}

	// Encrypt new symmetric key for each user UUID
	for _, userUUID := range userUUIDs {
		publicKeyPath := filepath.Join(projectPublicKeyPath, userUUID+".pub")
		publicKey, err := LoadPublicKey(publicKeyPath)
		if err != nil {
			return fmt.Errorf("failed to load public key for user %s: %w", userUUID, err)
		}

		encryptedSymKey, err := EncryptWithPublicKey(newSymKey, publicKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt symmetric key for user %s: %w", userUUID, err)
		}

		if err := SaveKanukaKeyToProject(userUUID, encryptedSymKey); err != nil {
			return fmt.Errorf("failed to save symmetric key for user %s: %w", userUUID, err)
		}
	}

	// Re-encrypt all files with new symmetric key
	for _, fileData := range plaintexts {
		inputPaths := []string{fileData.NewPath}
		if err := EncryptFiles(newSymKey, inputPaths, verbose); err != nil {
			return fmt.Errorf("failed to re-encrypt file %s: %w", fileData.NewPath, err)
		}
	}

	return nil
}
