package secrets

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/PolarWolf314/kanuka/internal/configs"
	logger "github.com/PolarWolf314/kanuka/internal/logging"

	"golang.org/x/crypto/nacl/secretbox"
)

// SyncOptions configures the sync operation.
type SyncOptions struct {
	// DryRun if true, simulates the operation without writing files.
	DryRun bool

	// ExcludeUsers is a list of user UUIDs to exclude from re-encryption.
	// Used by revoke to exclude the user being removed.
	ExcludeUsers []string

	// Verbose enables detailed logging.
	Verbose bool

	// Debug enables debug logging.
	Debug bool
}

// SyncResult contains the results of a sync operation.
type SyncResult struct {
	// SecretsProcessed is the number of secret files re-encrypted.
	SecretsProcessed int

	// UsersProcessed is the number of users who received the new key.
	UsersProcessed int

	// UsersExcluded is the number of users excluded from re-encryption.
	UsersExcluded int

	// UserEmails contains the emails of users who got the new key (if available).
	UserEmails []string

	// Errors contains any non-fatal errors encountered.
	Errors []error
}

// decryptedSecret holds a secret file's path and decrypted content.
type decryptedSecret struct {
	originalPath string
	plaintext    []byte
}

// userKeyData holds an encrypted symmetric key for a user.
type userKeyData struct {
	uuid         string
	encryptedKey []byte
}

// SyncSecrets re-encrypts all secrets with a new symmetric key.
// The privateKey is used to decrypt the current symmetric key.
// Returns a SyncResult with details of the operation.
func SyncSecrets(privateKey *rsa.PrivateKey, opts SyncOptions) (*SyncResult, error) {
	log := logger.Logger{Verbose: opts.Verbose, Debug: opts.Debug}

	result := &SyncResult{
		Errors: []error{},
	}

	// Initialize project settings.
	if err := configs.InitProjectSettings(); err != nil {
		return nil, fmt.Errorf("failed to init project settings: %w", err)
	}

	projectPath := configs.ProjectKanukaSettings.ProjectPath
	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath
	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath

	// Load user config to get current user's UUID.
	userConfig, err := configs.LoadUserConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load user config: %w", err)
	}

	currentUserUUID := userConfig.User.UUID
	if currentUserUUID == "" {
		return nil, fmt.Errorf("user UUID not found in user config")
	}

	log.Debugf("Starting sync for user %s", currentUserUUID)

	// Get all user UUIDs in the project.
	allUserUUIDs, err := GetAllUsersInProject()
	if err != nil {
		return nil, fmt.Errorf("failed to get list of users: %w", err)
	}

	if len(allUserUUIDs) == 0 {
		return nil, fmt.Errorf("no users found in project")
	}

	log.Debugf("Found %d users in project", len(allUserUUIDs))

	// Filter out excluded users.
	excludeMap := make(map[string]bool)
	for _, uuid := range opts.ExcludeUsers {
		excludeMap[uuid] = true
	}

	var activeUserUUIDs []string
	for _, uuid := range allUserUUIDs {
		if excludeMap[uuid] {
			result.UsersExcluded++
			log.Debugf("Excluding user %s from re-encryption", uuid)
		} else {
			activeUserUUIDs = append(activeUserUUIDs, uuid)
		}
	}

	if len(activeUserUUIDs) == 0 {
		return nil, fmt.Errorf("no active users remaining after exclusions")
	}

	// Get current encrypted symmetric key.
	currentEncryptedSymKey, err := GetProjectKanukaKey(currentUserUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current symmetric key for user %s: %w", currentUserUUID, err)
	}

	// Decrypt current symmetric key.
	currentSymKey, err := DecryptWithPrivateKey(currentEncryptedSymKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt current symmetric key: %w", err)
	}

	// Zero out the current symmetric key when we're done (defense in depth).
	defer func() {
		for i := range currentSymKey {
			currentSymKey[i] = 0
		}
	}()

	log.Infof("Decrypted current symmetric key")

	// Find all .kanuka secret files in project (excluding .kanuka/secrets/ which has user keys).
	kanukaFiles, err := FindEnvOrKanukaFiles(projectPath, []string{}, true)
	if err != nil {
		return nil, fmt.Errorf("failed to find .kanuka files: %w", err)
	}

	log.Infof("Found %d secret files to process", len(kanukaFiles))

	// Decrypt all files to memory.
	var decryptedSecrets []decryptedSecret

	var key [32]byte
	copy(key[:], currentSymKey)

	for _, kanukaFile := range kanukaFiles {
		ciphertext, err := os.ReadFile(kanukaFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read .kanuka file %s: %w", kanukaFile, err)
		}

		if len(ciphertext) < 24 {
			return nil, fmt.Errorf("invalid .kanuka file %s: too short", kanukaFile)
		}

		var decryptNonce [24]byte
		copy(decryptNonce[:], ciphertext[:24])

		plaintext, ok := secretbox.Open(nil, ciphertext[24:], &decryptNonce, &key)
		if !ok {
			return nil, fmt.Errorf("failed to decrypt file %s", kanukaFile)
		}

		decryptedSecrets = append(decryptedSecrets, decryptedSecret{
			originalPath: kanukaFile,
			plaintext:    plaintext,
		})

		log.Debugf("Decrypted %s", kanukaFile)
	}

	// Generate new symmetric key.
	newSymKey, err := CreateSymmetricKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate new symmetric key: %w", err)
	}

	// Zero out the new symmetric key when we're done (defense in depth).
	defer func() {
		for i := range newSymKey {
			newSymKey[i] = 0
		}
	}()

	log.Infof("Generated new symmetric key")

	// Encrypt new symmetric key for each active user.
	var userKeys []userKeyData

	for _, userUUID := range activeUserUUIDs {
		publicKeyPath := filepath.Join(projectPublicKeyPath, userUUID+".pub")
		publicKey, err := LoadPublicKey(publicKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load public key for user %s: %w", userUUID, err)
		}

		encryptedSymKey, err := EncryptWithPublicKey(newSymKey, publicKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt symmetric key for user %s: %w", userUUID, err)
		}

		userKeys = append(userKeys, userKeyData{
			uuid:         userUUID,
			encryptedKey: encryptedSymKey,
		})

		log.Debugf("Encrypted symmetric key for user %s", userUUID)
	}

	result.UsersProcessed = len(userKeys)

	// Re-encrypt all secret files with new symmetric key.
	var newKey [32]byte
	copy(newKey[:], newSymKey)

	reencryptedSecrets := make(map[string][]byte)

	for _, ds := range decryptedSecrets {
		var nonce [24]byte
		if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
			return nil, fmt.Errorf("failed to generate nonce: %w", err)
		}

		ciphertext := secretbox.Seal(nonce[:], ds.plaintext, &nonce, &newKey)
		reencryptedSecrets[ds.originalPath] = ciphertext

		log.Debugf("Re-encrypted %s", ds.originalPath)
	}

	result.SecretsProcessed = len(reencryptedSecrets)

	// If dry-run, stop here before writing anything.
	if opts.DryRun {
		log.Infof("Dry-run mode: would write %d user keys and %d secret files", len(userKeys), len(reencryptedSecrets))
		return result, nil
	}

	// Write everything to disk atomically.
	// First, write all user .kanuka files.
	for _, uk := range userKeys {
		kanukaPath := filepath.Join(projectSecretsPath, uk.uuid+".kanuka")
		if err := os.WriteFile(kanukaPath, uk.encryptedKey, 0600); err != nil {
			return nil, fmt.Errorf("failed to save symmetric key for user %s: %w", uk.uuid, err)
		}
		log.Debugf("Wrote user key file %s", kanukaPath)
	}

	// Then, write all re-encrypted secret files.
	for path, ciphertext := range reencryptedSecrets {
		if err := os.WriteFile(path, ciphertext, 0600); err != nil {
			return nil, fmt.Errorf("failed to write re-encrypted file %s: %w", path, err)
		}
		log.Debugf("Wrote secret file %s", path)
	}

	// Delete .kanuka files for excluded users (they should no longer have access).
	for _, excludedUUID := range opts.ExcludeUsers {
		kanukaPath := filepath.Join(projectSecretsPath, excludedUUID+".kanuka")
		if _, err := os.Stat(kanukaPath); err == nil {
			if err := os.Remove(kanukaPath); err != nil {
				// Non-fatal error - record it but continue.
				result.Errors = append(result.Errors, fmt.Errorf("failed to remove .kanuka file for excluded user %s: %w", excludedUUID, err))
				log.Warnf("Failed to remove .kanuka file for excluded user %s: %v", excludedUUID, err)
			} else {
				log.Debugf("Removed .kanuka file for excluded user %s", excludedUUID)
			}
		}
	}

	log.Infof("Sync completed: %d secrets re-encrypted for %d users", result.SecretsProcessed, result.UsersProcessed)

	return result, nil
}

// SyncSecretsSimple is a simplified version of SyncSecrets for backward compatibility.
// It wraps the existing RotateSymmetricKey functionality.
func SyncSecretsSimple(currentUserUUID string, privateKey *rsa.PrivateKey, verbose bool) error {
	opts := SyncOptions{
		Verbose: verbose,
	}

	_, err := SyncSecrets(privateKey, opts)
	return err
}
