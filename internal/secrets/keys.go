package secrets

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/configs"
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

// ParsePublicKeyText parses a PEM-encoded or SSH format public key string and returns an RSA public key.
func ParsePublicKeyText(publicKeyText string) (*rsa.PublicKey, error) {
	// Ensure the text is trimmed of whitespace
	publicKeyText = strings.TrimSpace(publicKeyText)

	// Check if this is an SSH format key (starts with "ssh-rsa")
	if strings.HasPrefix(publicKeyText, "ssh-rsa") {
		return parseSSHPublicKey(publicKeyText)
	}

	// If not SSH format, try PEM format
	if !strings.HasPrefix(publicKeyText, "-----BEGIN") {
		return nil, errors.New("public key text does not appear to be in PEM or SSH format")
	}

	// Decode the PEM block
	block, _ := pem.Decode([]byte(publicKeyText))
	if block == nil {
		return nil, errors.New("failed to parse PEM block")
	}

	// Check that this is a public key
	if block.Type != "PUBLIC KEY" && block.Type != "RSA PUBLIC KEY" {
		return nil, errors.New("PEM block is not a public key")
	}

	// Parse the public key
	var publicKey interface{}
	var err error

	if block.Type == "RSA PUBLIC KEY" {
		// PKCS#1 format
		publicKey, err = x509.ParsePKCS1PublicKey(block.Bytes)
	} else {
		// PKCS#8 format (more common)
		publicKey, err = x509.ParsePKIXPublicKey(block.Bytes)
	}

	if err != nil {
		return nil, err
	}

	// Convert to RSA public key
	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}

	return rsaPublicKey, nil
}

// parseSSHPublicKey parses an SSH format RSA public key.
// Format: ssh-rsa BASE64DATA comment.
func parseSSHPublicKey(sshPublicKey string) (*rsa.PublicKey, error) {
	parts := strings.Fields(sshPublicKey)
	if len(parts) < 2 {
		return nil, errors.New("invalid SSH public key format")
	}

	if parts[0] != "ssh-rsa" {
		return nil, errors.New("unsupported key type, only RSA is supported")
	}

	// Decode the base64 encoded part
	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode SSH key: %v", err)
	}

	// SSH key format consists of:
	// - length of "ssh-rsa" (4 bytes)
	// - the string "ssh-rsa"
	// - length of exponent (4 bytes)
	// - exponent
	// - length of modulus (4 bytes)
	// - modulus

	// Let's parse this data structure
	var pos uint32 = 0

	// Skip the key type
	keyTypeLen := binary.BigEndian.Uint32(decoded[pos : pos+4])
	pos += 4 + keyTypeLen

	// Read the exponent
	expLen := binary.BigEndian.Uint32(decoded[pos : pos+4])
	pos += 4
	if int(pos)+int(expLen) > len(decoded) {
		return nil, errors.New("invalid SSH key: exponent length out of bounds")
	}
	e := int(0)
	for i := uint32(0); i < expLen; i++ {
		e = e*256 + int(decoded[int(pos)+int(i)])
	}
	pos += expLen

	// Read the modulus
	modLen := binary.BigEndian.Uint32(decoded[pos : pos+4])
	pos += 4
	if int(pos)+int(modLen) > len(decoded) {
		return nil, errors.New("invalid SSH key: modulus length out of bounds")
	}
	modBytes := decoded[int(pos) : int(pos)+int(modLen)]

	// Ensure the first byte isn't negative
	if modBytes[0] >= 0x80 {
		modBytes = append([]byte{0}, modBytes...)
	}

	n := new(big.Int).SetBytes(modBytes)

	// Create the RSA public key
	return &rsa.PublicKey{
		N: n,
		E: e,
	}, nil
}

// SavePublicKeyToFile saves an RSA public key to a file in PEM format.
func SavePublicKeyToFile(publicKey *rsa.PublicKey, filePath string) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Convert public key to DER format (PKIX)
	derBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return err
	}

	pemBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derBytes,
	}
	pemBytes := pem.EncodeToMemory(pemBlock)

	// #nosec G306 -- This is a pubkey
	return os.WriteFile(filePath, pemBytes, 0644)
}
