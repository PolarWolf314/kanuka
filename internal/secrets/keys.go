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
	"time"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/utils"

	"github.com/fatih/color"
	"golang.org/x/crypto/ssh"
)

// ErrPassphraseRequired is returned when a private key is passphrase-protected
// but no passphrase was provided.
var ErrPassphraseRequired = errors.New("private key is passphrase-protected")

// parseOpenSSHPrivateKey parses an OpenSSH format private key and returns an RSA private key.
// If the key is passphrase-protected and no passphrase is provided, it returns ErrPassphraseRequired.
// Only RSA keys are supported; other key types (Ed25519, ECDSA) will return an error.
func parseOpenSSHPrivateKey(data []byte, passphrase []byte) (*rsa.PrivateKey, error) {
	var (
		rawKey interface{}
		err    error
	)

	if len(passphrase) > 0 {
		rawKey, err = ssh.ParseRawPrivateKeyWithPassphrase(data, passphrase)
	} else {
		rawKey, err = ssh.ParseRawPrivateKey(data)
	}

	if err != nil {
		// Check if the error indicates the key is passphrase-protected or wrong passphrase
		errMsg := err.Error()
		if strings.Contains(errMsg, "passphrase") ||
			strings.Contains(errMsg, "encrypted") ||
			strings.Contains(errMsg, "this private key is passphrase protected") ||
			strings.Contains(errMsg, "password incorrect") ||
			strings.Contains(errMsg, "decryption password") {
			return nil, ErrPassphraseRequired
		}
		return nil, fmt.Errorf("failed to parse OpenSSH private key: %w", err)
	}

	// Check if the key is an RSA key
	rsaKey, ok := rawKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("unsupported key type: only RSA keys are supported, got %T", rawKey)
	}

	return rsaKey, nil
}

// LoadPrivateKey loads an RSA private key from disk.
// If the key is passphrase-protected, prompts the user for the passphrase.
// Supports PEM (PKCS#1, PKCS#8) and OpenSSH formats.
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadPrivateKeyFromBytesWithPrompt(data)
}

// LoadPrivateKeyFromBytesWithPrompt parses a private key from bytes, prompting for passphrase if needed.
// If the key is passphrase-protected and stdin is a terminal, prompts up to 3 times for the passphrase.
// Returns an error if the key is encrypted but stdin is not a terminal.
func LoadPrivateKeyFromBytesWithPrompt(data []byte) (*rsa.PrivateKey, error) {
	// First attempt without passphrase
	key, err := ParsePrivateKeyBytes(data)
	if err == nil {
		return key, nil
	}

	// Check if passphrase is required
	if !errors.Is(err, ErrPassphraseRequired) {
		return nil, err
	}

	// Check if we can prompt for passphrase
	if !utils.IsTerminal() {
		return nil, fmt.Errorf("private key is passphrase-protected but stdin is not a terminal; cannot prompt for passphrase")
	}

	// Prompt for passphrase (up to 3 attempts)
	maxAttempts := 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		passphrase, promptErr := utils.ReadPassphrase("Enter passphrase for private key: ")
		if promptErr != nil {
			return nil, promptErr
		}

		key, err = ParsePrivateKeyBytesWithPassphrase(data, passphrase)
		if err == nil {
			return key, nil
		}

		// Check if it's still a passphrase error (wrong passphrase)
		if errors.Is(err, ErrPassphraseRequired) {
			if attempt < maxAttempts {
				fmt.Fprintln(os.Stderr, color.YellowString("✗")+" Incorrect passphrase. Please try again.")
			}
			continue
		}

		// Some other error
		return nil, err
	}

	return nil, fmt.Errorf("failed to decrypt private key after %d attempts", maxAttempts)
}

// LoadPrivateKeyFromBytesWithTTYPrompt parses a private key from bytes, prompting for passphrase via TTY if needed.
// This is used when stdin contains the private key data (e.g., piped from a secret manager),
// so passphrase prompting must happen via /dev/tty instead of stdin.
// Returns an error if the key is encrypted but TTY is not available.
func LoadPrivateKeyFromBytesWithTTYPrompt(data []byte) (*rsa.PrivateKey, error) {
	// First attempt without passphrase
	key, err := ParsePrivateKeyBytes(data)
	if err == nil {
		return key, nil
	}

	// Check if passphrase is required
	if !errors.Is(err, ErrPassphraseRequired) {
		return nil, err
	}

	// Check if we can prompt for passphrase via TTY
	if !utils.IsTTYAvailable() {
		return nil, fmt.Errorf("private key is passphrase-protected but no TTY available; cannot prompt for passphrase")
	}

	// Prompt for passphrase via TTY (up to 3 attempts)
	maxAttempts := 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		passphrase, promptErr := utils.ReadPassphraseFromTTY("Enter passphrase for private key: ")
		if promptErr != nil {
			return nil, promptErr
		}

		key, err = ParsePrivateKeyBytesWithPassphrase(data, passphrase)
		if err == nil {
			return key, nil
		}

		// Check if it's still a passphrase error (wrong passphrase)
		if errors.Is(err, ErrPassphraseRequired) {
			if attempt < maxAttempts {
				fmt.Fprintln(os.Stderr, color.YellowString("✗")+" Incorrect passphrase. Please try again.")
			}
			continue
		}

		// Some other error
		return nil, err
	}

	return nil, fmt.Errorf("failed to decrypt private key after %d attempts", maxAttempts)
}

// ParsePrivateKeyBytes parses an RSA private key from bytes.
// Supports PEM (PKCS#1, PKCS#8) and OpenSSH formats.
// For passphrase-protected keys, use ParsePrivateKeyBytesWithPassphrase.
func ParsePrivateKeyBytes(data []byte) (*rsa.PrivateKey, error) {
	return ParsePrivateKeyBytesWithPassphrase(data, nil)
}

// ParsePrivateKeyBytesWithPassphrase parses an RSA private key from bytes with an optional passphrase.
// Supports PEM (PKCS#1, PKCS#8) and OpenSSH formats.
// If passphrase is nil and the key is encrypted, returns ErrPassphraseRequired.
func ParsePrivateKeyBytesWithPassphrase(data []byte, passphrase []byte) (*rsa.PrivateKey, error) {
	// Try to decode as PEM
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from private key data")
	}

	// Check the PEM block type to determine format
	switch block.Type {
	case "RSA PRIVATE KEY":
		// Traditional PEM format (PKCS#1)
		// Note: PKCS#1 encrypted keys have headers like "Proc-Type" and "DEK-Info"
		if x509.IsEncryptedPEMBlock(block) { //nolint:staticcheck // deprecated but still needed for legacy PEM
			if len(passphrase) == 0 {
				return nil, ErrPassphraseRequired
			}
			decryptedBytes, err := x509.DecryptPEMBlock(block, passphrase) //nolint:staticcheck // deprecated but still needed for legacy PEM
			if err != nil {
				return nil, ErrPassphraseRequired // Likely wrong passphrase
			}
			return x509.ParsePKCS1PrivateKey(decryptedBytes)
		}
		return x509.ParsePKCS1PrivateKey(block.Bytes)

	case "PRIVATE KEY":
		// PKCS#8 format (unencrypted)
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKCS#8 private key: %w", err)
		}
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("PKCS#8 key is not an RSA key, got %T", key)
		}
		return rsaKey, nil

	case "ENCRYPTED PRIVATE KEY":
		// PKCS#8 encrypted format - not commonly used, return helpful error
		return nil, fmt.Errorf("encrypted PKCS#8 keys are not supported; please convert to OpenSSH format")

	case "OPENSSH PRIVATE KEY":
		// OpenSSH format - pass the full data (including PEM wrapper)
		return parseOpenSSHPrivateKey(data, passphrase)

	default:
		return nil, fmt.Errorf("unsupported private key format: %s", block.Type)
	}
}

// ParsePrivateKeyText parses a PEM-encoded or OpenSSH format private key string
// and returns an RSA private key.
func ParsePrivateKeyText(privateKeyText string) (*rsa.PrivateKey, error) {
	// Ensure the text is trimmed of whitespace
	privateKeyText = strings.TrimSpace(privateKeyText)

	if privateKeyText == "" {
		return nil, errors.New("private key text is empty")
	}

	// Check that it looks like a PEM-encoded key
	if !strings.HasPrefix(privateKeyText, "-----BEGIN") {
		return nil, errors.New("private key text does not appear to be in PEM format")
	}

	return ParsePrivateKeyBytes([]byte(privateKeyText))
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
	privFile, err := os.OpenFile(privatePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
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
	pubFile, err := os.OpenFile(publicPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
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
// It uses the project UUID from the project config to create a subdirectory for the key files.
// The new structure is: ~/.local/share/kanuka/keys/{project_uuid}/privkey, pubkey.pub, metadata.toml.
func CreateAndSaveRSAKeyPair(verbose bool) error {
	if err := configs.InitProjectSettings(); err != nil {
		return fmt.Errorf("failed to init project settings: %w", err)
	}

	// Load project config to get project UUID and name
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		return fmt.Errorf("failed to load project config: %w", err)
	}

	projectUUID := projectConfig.Project.UUID
	if projectUUID == "" {
		return fmt.Errorf("project UUID not found in project config")
	}

	// Create key directory for this project
	keyDir := configs.GetKeyDirPath(projectUUID)
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return fmt.Errorf("failed to create key directory at %s: %w", keyDir, err)
	}

	// Create key paths using new structure
	privateKeyPath := configs.GetPrivateKeyPath(projectUUID)
	publicKeyPath := configs.GetPublicKeyPath(projectUUID)

	if err := GenerateRSAKeyPair(privateKeyPath, publicKeyPath); err != nil {
		return fmt.Errorf("failed to generate or save RSA key pair for project %s: %w", projectUUID, err)
	}

	// Create metadata.toml with project information
	metadata := &configs.KeyMetadata{
		ProjectName:    projectConfig.Project.Name,
		ProjectPath:    configs.ProjectKanukaSettings.ProjectPath,
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
	}

	if err := configs.SaveKeyMetadata(projectUUID, metadata); err != nil {
		return fmt.Errorf("failed to save key metadata for project %s: %w", projectUUID, err)
	}

	return nil
}

// CopyUserPublicKeyToProject copies the user's public key to the project directory.
// It uses the project UUID for the source key and user UUID for the destination.
func CopyUserPublicKeyToProject() (string, error) {
	if err := configs.InitProjectSettings(); err != nil {
		return "", fmt.Errorf("failed to init project settings: %w", err)
	}

	// Load project config to get project UUID
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load project config: %w", err)
	}

	projectUUID := projectConfig.Project.UUID
	if projectUUID == "" {
		return "", fmt.Errorf("project UUID not found in project config")
	}

	// Ensure user config has UUID
	userConfig, err := configs.EnsureUserConfig()
	if err != nil {
		return "", fmt.Errorf("failed to ensure user config: %w", err)
	}

	userUUID := userConfig.User.UUID
	if userUUID == "" {
		return "", fmt.Errorf("user UUID not found in user config")
	}

	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath

	// Source key is in the project's key directory
	sourceKeyPath := configs.GetPublicKeyPath(projectUUID)
	// Destination key is named with user UUID
	destKeyPath := filepath.Join(projectPublicKeyPath, userUUID+".pub")

	// Check if source key exists
	if _, err := os.Stat(sourceKeyPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("public key for project %s not found at %s", projectUUID, sourceKeyPath)
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

// SaveKanukaKeyToProject saves an encrypted symmetric key for a user identified by their UUID.
func SaveKanukaKeyToProject(userUUID string, kanukaKey []byte) error {
	if err := configs.InitProjectSettings(); err != nil {
		return fmt.Errorf("failed to init project settings: %w", err)
	}

	projectPath := configs.ProjectKanukaSettings.ProjectPath
	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath

	if projectPath == "" {
		return fmt.Errorf("failed to find project root because it doesn't exist")
	}

	// Use user UUID for the file name
	destKeyPath := filepath.Join(projectSecretsPath, userUUID+".kanuka")

	if err := os.WriteFile(destKeyPath, kanukaKey, 0600); err != nil {
		return fmt.Errorf("failed to write key to project: %w", err)
	}

	return nil
}

// GetProjectKanukaKey retrieves the encrypted symmetric key for a user identified by their UUID.
func GetProjectKanukaKey(userUUID string) ([]byte, error) {
	if err := configs.InitProjectSettings(); err != nil {
		return nil, fmt.Errorf("failed to init project settings: %w", err)
	}

	projectPath := configs.ProjectKanukaSettings.ProjectPath
	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath

	if projectPath == "" {
		return nil, fmt.Errorf("failed to find project root because it doesn't exist")
	}

	// Use user UUID for the file name
	userKeyFile := filepath.Join(projectSecretsPath, userUUID+".kanuka")
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

// GetAllUsersInProject returns a list of all user UUIDs with access to the project.
// Files in the public_keys directory are named with user UUIDs.
func GetAllUsersInProject() ([]string, error) {
	if err := configs.InitProjectSettings(); err != nil {
		return nil, fmt.Errorf("failed to init project settings: %w", err)
	}

	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath

	entries, err := os.ReadDir(projectPublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public keys directory: %w", err)
	}

	var userUUIDs []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".pub" {
			// Extract UUID from filename (e.g., "uuid.pub" -> "uuid")
			userUUID := entry.Name()[:len(entry.Name())-len(".pub")]
			userUUIDs = append(userUUIDs, userUUID)
		}
	}

	return userUUIDs, nil
}
