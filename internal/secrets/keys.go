package secrets

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"kanuka/internal/configs"
	"os"
	"path/filepath"
)

// LoadPrivateKey loads an RSA private key from disk.
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

// LoadPublicKey loads the user's public key from the project directory.
func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
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

// GenerateRSAKeyPair creates a new RSA key pair and saves them to disk.
func GenerateRSAKeyPair(privatePath string, publicPath string) error {
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

// CreateAndSaveRSAKeyPair generates a new RSA key pair for the project and saves them in the user's directory.
func CreateAndSaveRSAKeyPair(verbose bool) error {
	if err := configs.InitProjectSettings(); err != nil {
		return fmt.Errorf("failed to init project settings: %w", err)
	}
	projectName := configs.ProjectKanukaSettings.ProjectName

	// Create key paths
	keysDir := configs.UserKanukaSettings.UserKeysPath
	privateKeyPath := filepath.Join(keysDir, projectName)
	publicKeyPath := privateKeyPath + ".pub"

	// Ensure key directory exists
	if err := os.MkdirAll(keysDir, 0700); err != nil {
		return fmt.Errorf("failed to create keys directory at %s: %w", keysDir, err)
	}

	if err := GenerateRSAKeyPair(privateKeyPath, publicKeyPath); err != nil {
		return fmt.Errorf("failed to generate or save RSA key pair for project %s: %w", projectName, err)
	}

	return nil
}

// CopyUserPublicKeyToProject copies the user's public key to the project directory.
func CopyUserPublicKeyToProject() (string, error) {
	if err := configs.InitProjectSettings(); err != nil {
		return "", fmt.Errorf("failed to init project settings: %w", err)
	}

	username := configs.UserKanukaSettings.Username
	projectName := configs.ProjectKanukaSettings.ProjectName
	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath

	userKeysPath := configs.UserKanukaSettings.UserKeysPath
	sourceKeyPath := filepath.Join(userKeysPath, projectName+".pub")
	destKeyPath := filepath.Join(projectPublicKeyPath, username+".pub")

	// Check if source key exists
	if _, err := os.Stat(sourceKeyPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("public key for project %s not found at %s", projectName, sourceKeyPath)
		}
		return "", fmt.Errorf("failed to check for source key: %w", err)
	}

	keyData, err := os.ReadFile(sourceKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read source key file: %w", err)
	}

	// Write to destination file
	if err := os.WriteFile(destKeyPath, keyData, 0600); err != nil {
		return "", fmt.Errorf("failed to write key to project: %w", err)
	}

	return destKeyPath, nil
}

func SaveKanukaKeyToProject(username string, kanukaKey []byte) error {
	if err := configs.InitProjectSettings(); err != nil {
		return fmt.Errorf("failed to init project settings: %w", err)
	}

	projectPath := configs.ProjectKanukaSettings.ProjectPath
	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath

	destKeyPath := filepath.Join(projectSecretsPath, username+".kanuka")

	if projectPath == "" {
		return fmt.Errorf("failed to find project root because it doesn't exist")
	}

	if err := os.WriteFile(destKeyPath, kanukaKey, 0600); err != nil {
		return fmt.Errorf("failed to write key to project: %w", err)
	}

	return nil
}

// GetUserProjectKanukaKey retrieves the encrypted symmetric key for the current user and project.
func GetProjectKanukaKey(username string) ([]byte, error) {
	if err := configs.InitProjectSettings(); err != nil {
		return nil, fmt.Errorf("failed to init project settings: %w", err)
	}

	projectPath := configs.ProjectKanukaSettings.ProjectPath
	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath

	if projectPath == "" {
		return nil, fmt.Errorf("failed to find project root because it doesn't exist")
	}

	userKeyFile := filepath.Join(projectSecretsPath, username+".kanuka")
	if _, err := os.Stat(userKeyFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to get user's project encrypted symmetric key: %w", err)
	}

	encryptedSymmetricKey, err := os.ReadFile(userKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read user's project encrypted symmetric key: %w", err)
	}

	return encryptedSymmetricKey, nil
}
