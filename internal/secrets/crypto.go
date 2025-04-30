package secrets

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/nacl/secretbox"
)

func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, fmt.Errorf("failed to decode PEM block containing private key")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func DecryptWithPrivateKey(ciphertext []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	return rsa.DecryptPKCS1v15(rand.Reader, privateKey, ciphertext)
}

func GenerateRSAKeyPair(privatePath, publicPath string) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key pair: %w", err)
	}

	// Create directories if they don't exist
	privateDir := filepath.Dir(privatePath)
	if err := os.MkdirAll(privateDir, 0700); err != nil {
		return fmt.Errorf("failed to create directory for private key at %s: %w", privateDir, err)
	}
	publicDir := filepath.Dir(publicPath)
	if err := os.MkdirAll(publicDir, 0700); err != nil {
		return fmt.Errorf("failed to create directory for public key at %s: %w", publicDir, err)
	}

	// Save private key
	privFile, err := os.Create(privatePath)
	if err != nil {
		return fmt.Errorf("failed to create private key file at %s: %w", privatePath, err)
	}
	defer func() {
		if closeErr := privFile.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close private key file: %w", closeErr)
		}
	}()

	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privPem := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}
	if err := pem.Encode(privFile, privPem); err != nil {
		return fmt.Errorf("failed to PEM encode private key: %w", err)
	}

	// Save public key
	pubFile, err := os.Create(publicPath)
	if err != nil {
		return fmt.Errorf("failed to create public key file at %s: %w", publicPath, err)
	}
	defer func() {
		if closeErr := pubFile.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close public key file: %w", closeErr)
		}
	}()

	pubASN1, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}
	pubPem := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubASN1,
	}
	if err := pem.Encode(pubFile, pubPem); err != nil {
		return fmt.Errorf("failed to PEM encode public key: %w", err)
	}

	return nil
}

func CreateAndSaveRSAKeyPair() error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}
	projectName := filepath.Base(wd)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user's home directory: %w", err)
	}

	// Create key paths
	keysDir := filepath.Join(homeDir, ".kanuka", "keys")
	privateKeyPath := filepath.Join(keysDir, projectName)
	publicKeyPath := privateKeyPath + ".pub"

	// Ensure key directory exists
	if err := os.MkdirAll(keysDir, 0700); err != nil {
		return fmt.Errorf("failed to create keys directory at %s: %w", keysDir, err)
	}

	if err := GenerateRSAKeyPair(privateKeyPath, publicKeyPath); err != nil {
		return fmt.Errorf("failed to generate or save RSA key pair for project %s: %w", projectName, err)
	}

	log.Printf("✅ Successfully generated RSA keys at:\n  - Private: %s\n  - Public: %s", privateKeyPath, publicKeyPath)
	return nil
}

func CreateSymmetricKey() ([]byte, error) {
	symKey := make([]byte, 32) // AES-256
	if _, err := rand.Read(symKey); err != nil {
		return nil, err
	}

	return symKey, nil
}

func LoadPublicKey() (*rsa.PublicKey, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	username, err := GetUsername()
	if err != nil {
		return nil, fmt.Errorf("failed to get username: %w", err)
	}

	kanukaDir := filepath.Join(wd, ".kanuka")
	publicKeyPath := filepath.Join(kanukaDir, "public_keys", username+".pub")

	data, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("failed to decode PEM block containing public key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}
	return rsaPub, nil
}

func CreateAndSaveEncryptedSymmetricKey() error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	kanukaDir := filepath.Join(wd, ".kanuka")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	// 1. create sym key in memory
	symKey, err := CreateSymmetricKey()
	if err != nil {
		return fmt.Errorf("failed to generate symmetric key: %w", err)
	}
	log.Println("🔐 Symmetric key generated in memory")

	// 2. fetch user's public key from project
	pubKey, err := LoadPublicKey()
	if err != nil {
		return fmt.Errorf("failed to load project public key: %w", err)
	}

	// 3. encrypt sym key using public key
	encryptedSymKey, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, symKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt symmetric key: %w", err)
	}
	log.Println("🔒 Encrypted symmetric key with project public key")

	// 4. save sym key to project
	username, err := GetUsername()
	if err != nil {
		return fmt.Errorf("failed to get username: %w", err)
	}

	encryptedSymPath := filepath.Join(secretsDir, fmt.Sprintf("%s.kanuka", username))

	if err := os.WriteFile(encryptedSymPath, encryptedSymKey, 0600); err != nil {
		return fmt.Errorf("failed to save encrypted symmetric key: %v", err)
	}
	log.Println("✅ Saved encrypted symmetric key into project")

	return nil
}

func EncryptFiles(symKey []byte, inputPaths []string) error {
	if len(symKey) != 32 {
		return fmt.Errorf("failed as symmetric key must be 32 bytes for secretbox")
	}

	var key [32]byte
	copy(key[:], symKey)

	var outputPaths []string

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
		outputPaths = append(outputPaths, outputPath)

		if err := os.WriteFile(outputPath, ciphertext, 0600); err != nil {
			return fmt.Errorf("failed to write to %s: %w", outputPath, err)
		}
	}

	log.Println("✅ All environment files in the project have been encrypted 🎉")
	log.Printf("The following files were written: %s", FormatPaths(outputPaths))

	return nil
}

func DecryptFiles(symKey []byte, inputPaths []string) error {
	if len(symKey) != 32 {
		return fmt.Errorf("failed as symmetric key must be 32 bytes for secretbox")
	}
	var key [32]byte
	copy(key[:], symKey)
	var outputPaths []string
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
		outputPaths = append(outputPaths, outputPath)
		// #nosec G306 -- We want the decrypted .env file to be editable by the user
		if err := os.WriteFile(outputPath, plaintext, 0644); err != nil {
			return fmt.Errorf("failed to write to %s: %w", outputPath, err)
		}
	}
	log.Println("✅ All environment files in the project have been decrypted 🎉")
	log.Printf("The following files were written: %s", FormatPaths(outputPaths))
	return nil
}
