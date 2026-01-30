package workflows

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/PolarWolf314/kanuka/internal/audit"
	"github.com/PolarWolf314/kanuka/internal/configs"
	kerrors "github.com/PolarWolf314/kanuka/internal/errors"
	"github.com/PolarWolf314/kanuka/internal/secrets"
)

// RotateOptions configures the rotate workflow.
type RotateOptions struct {
	// Force skips the confirmation prompt (handled by caller).
	// This field is informational; the actual prompting is done in the cmd layer.
	Force bool

	// PrivateKeyData contains the private key bytes when reading from stdin.
	// If nil, the private key is loaded from disk.
	PrivateKeyData []byte
}

// RotateResult contains the outcome of a rotate operation.
type RotateResult struct {
	// UserUUID is the UUID of the user whose keys were rotated.
	UserUUID string

	// ProjectUUID is the UUID of the project.
	ProjectUUID string

	// PrivateKeyPath is where the new private key was saved.
	PrivateKeyPath string

	// PublicKeyPath is where the new public key was saved (user directory).
	PublicKeyPath string

	// ProjectPublicKeyPath is where the new public key was copied (project directory).
	ProjectPublicKeyPath string
}

// Rotate generates a new keypair and replaces the user's current keys for this project.
//
// This command is useful for key rotation when a private key may have been compromised.
// The workflow:
//  1. Loads the user's current private key
//  2. Decrypts the symmetric key with the old private key
//  3. Generates a new RSA keypair
//  4. Re-encrypts the symmetric key with the new public key
//  5. Saves the new private key and updates the public key in both locations
//
// Returns ErrProjectNotInitialized if the project has no .kanuka directory.
// Returns ErrNoAccess if the user doesn't have a key file for this project.
// Returns ErrPrivateKeyNotFound if the old private key cannot be loaded.
// Returns ErrKeyDecryptFailed if the private key cannot decrypt the symmetric key.
func Rotate(ctx context.Context, opts RotateOptions) (*RotateResult, error) {
	if err := configs.InitProjectSettings(); err != nil {
		return nil, fmt.Errorf("initializing project settings: %w", err)
	}

	projectPath := configs.ProjectKanukaSettings.ProjectPath
	if projectPath == "" {
		return nil, kerrors.ErrProjectNotInitialized
	}

	userConfig, err := configs.EnsureUserConfig()
	if err != nil {
		return nil, fmt.Errorf("loading user config: %w", err)
	}
	userUUID := userConfig.User.UUID

	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		return nil, fmt.Errorf("loading project config: %w", err)
	}
	projectUUID := projectConfig.Project.UUID

	// Check if user has access to this project.
	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath
	userKanukaKeyPath := filepath.Join(projectSecretsPath, userUUID+".kanuka")
	if _, err := os.Stat(userKanukaKeyPath); os.IsNotExist(err) {
		return nil, kerrors.ErrNoAccess
	}

	// Load current private key.
	oldPrivateKey, err := loadPrivateKey(opts.PrivateKeyData, projectUUID)
	if err != nil {
		return nil, err
	}

	// Get and decrypt symmetric key.
	encryptedSymKey, err := secrets.GetProjectKanukaKey(userUUID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", kerrors.ErrNoAccess, err)
	}

	symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, oldPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", kerrors.ErrKeyDecryptFailed, err)
	}

	// Generate new keypair.
	newPrivateKey, newPublicKey, err := generateNewKeypair()
	if err != nil {
		return nil, fmt.Errorf("generating new keypair: %w", err)
	}

	// Re-encrypt symmetric key with new public key.
	newEncryptedSymKey, err := secrets.EncryptWithPublicKey(symKey, newPublicKey)
	if err != nil {
		return nil, fmt.Errorf("encrypting symmetric key with new public key: %w", err)
	}

	// Save new private key.
	privateKeyPath := configs.GetPrivateKeyPath(projectUUID)
	if err := savePrivateKey(newPrivateKey, privateKeyPath); err != nil {
		return nil, fmt.Errorf("saving new private key: %w", err)
	}

	// Save new public key to user's key directory.
	publicKeyPath := configs.GetPublicKeyPath(projectUUID)
	if err := secrets.SavePublicKeyToFile(newPublicKey, publicKeyPath); err != nil {
		return nil, fmt.Errorf("saving new public key to user directory: %w", err)
	}

	// Copy new public key to project.
	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath
	projectPubKeyPath := filepath.Join(projectPublicKeyPath, userUUID+".pub")
	if err := secrets.SavePublicKeyToFile(newPublicKey, projectPubKeyPath); err != nil {
		return nil, fmt.Errorf("copying public key to project: %w", err)
	}

	// Save new encrypted symmetric key.
	if err := secrets.SaveKanukaKeyToProject(userUUID, newEncryptedSymKey); err != nil {
		return nil, fmt.Errorf("saving new encrypted symmetric key: %w", err)
	}

	// Update key metadata.
	metadata := &configs.KeyMetadata{
		ProjectName:    projectConfig.Project.Name,
		ProjectPath:    projectPath,
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
	}
	// Non-critical - just ignore errors.
	_ = configs.SaveKeyMetadata(projectUUID, metadata)

	// Log to audit trail.
	auditEntry := audit.LogWithUser("rotate")
	audit.Log(auditEntry)

	return &RotateResult{
		UserUUID:             userUUID,
		ProjectUUID:          projectUUID,
		PrivateKeyPath:       privateKeyPath,
		PublicKeyPath:        publicKeyPath,
		ProjectPublicKeyPath: projectPubKeyPath,
	}, nil
}

// generateNewKeypair generates a new RSA keypair.
func generateNewKeypair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("generating RSA key: %w", err)
	}
	return privateKey, &privateKey.PublicKey, nil
}

// savePrivateKey saves an RSA private key to a file in PEM format.
func savePrivateKey(privateKey *rsa.PrivateKey, filePath string) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privPem := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}
	pemBytes := pem.EncodeToMemory(privPem)

	if err := os.WriteFile(filePath, pemBytes, 0600); err != nil {
		return fmt.Errorf("writing private key: %w", err)
	}

	return nil
}
