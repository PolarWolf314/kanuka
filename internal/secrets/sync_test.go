package secrets

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"

	"golang.org/x/crypto/nacl/secretbox"
)

// testUserUUID is a fixed UUID used for sync testing.
const testUserUUID = "test-user-uuid-sync-1234-abcdefghijkl"

// testUser2UUID is a second fixed UUID for multi-user testing.
const testUser2UUID = "test-user-2-uuid-sync-5678-abcdefghijkl"

// testProjectUUID is a fixed UUID used for sync testing.
const testProjectUUID = "test-proj-uuid-sync-1234-abcdefghijkl"

// setupSyncTestEnvironment creates a complete test environment for sync tests.
// Returns the project dir, user dir, private key, and cleanup function.
func setupSyncTestEnvironment(t *testing.T) (string, string, *rsa.PrivateKey, func()) {
	t.Helper()

	// Save original settings
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings
	originalProjectSettings := configs.ProjectKanukaSettings
	originalUserConfig := configs.GlobalUserConfig
	originalProjectConfig := configs.GlobalProjectConfig

	// Create temp directories
	tempDir := t.TempDir()
	tempUserDir := t.TempDir()

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create .kanuka directory structure
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	if err := os.MkdirAll(publicKeysDir, 0755); err != nil {
		t.Fatalf("Failed to create public keys directory: %v", err)
	}
	if err := os.MkdirAll(secretsDir, 0755); err != nil {
		t.Fatalf("Failed to create secrets directory: %v", err)
	}

	// Create user config directory
	userConfigsPath := filepath.Join(tempUserDir, "config")
	userKeysPath := filepath.Join(tempUserDir, "keys")
	if err := os.MkdirAll(userConfigsPath, 0755); err != nil {
		t.Fatalf("Failed to create user config directory: %v", err)
	}
	if err := os.MkdirAll(userKeysPath, 0755); err != nil {
		t.Fatalf("Failed to create user keys directory: %v", err)
	}

	// Override user settings
	configs.UserKanukaSettings = &configs.UserSettings{
		UserKeysPath:    userKeysPath,
		UserConfigsPath: userConfigsPath,
		Username:        "testuser",
	}

	// Create user config
	userConfig := &configs.UserConfig{
		User: configs.User{
			UUID:  testUserUUID,
			Email: "testuser@example.com",
		},
		Projects: make(map[string]configs.UserProjectEntry),
	}
	if err := configs.SaveUserConfig(userConfig); err != nil {
		t.Fatalf("Failed to save user config: %v", err)
	}

	// Set project settings
	configs.ProjectKanukaSettings = &configs.ProjectSettings{
		ProjectUUID:          testProjectUUID,
		ProjectName:          filepath.Base(tempDir),
		ProjectPath:          tempDir,
		ProjectPublicKeyPath: publicKeysDir,
		ProjectSecretsPath:   secretsDir,
	}

	// Create project config
	projectConfig := &configs.ProjectConfig{
		Project: configs.Project{
			UUID: testProjectUUID,
			Name: filepath.Base(tempDir),
		},
	}
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	// Generate RSA key pair for the user
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Save public key to project
	pubKeyPath := filepath.Join(publicKeysDir, testUserUUID+".pub")
	if err := savePublicKeyToFile(&privateKey.PublicKey, pubKeyPath); err != nil {
		t.Fatalf("Failed to save public key: %v", err)
	}

	// Create symmetric key and encrypt for user
	symKey := make([]byte, 32)
	if _, err := rand.Read(symKey); err != nil {
		t.Fatalf("Failed to generate symmetric key: %v", err)
	}

	encryptedSymKey, err := EncryptWithPublicKey(symKey, &privateKey.PublicKey)
	if err != nil {
		t.Fatalf("Failed to encrypt symmetric key: %v", err)
	}

	// Save encrypted symmetric key
	userKanukaPath := filepath.Join(secretsDir, testUserUUID+".kanuka")
	if err := os.WriteFile(userKanukaPath, encryptedSymKey, 0600); err != nil {
		t.Fatalf("Failed to save encrypted symmetric key: %v", err)
	}

	// Also save private key to user's key directory for the project
	keyDir := filepath.Join(userKeysPath, testProjectUUID)
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		t.Fatalf("Failed to create key directory: %v", err)
	}
	privKeyPath := filepath.Join(keyDir, "privkey")
	if err := savePrivateKeyToFile(privateKey, privKeyPath); err != nil {
		t.Fatalf("Failed to save private key: %v", err)
	}

	cleanup := func() {
		_ = os.Chdir(originalWd)
		configs.UserKanukaSettings = originalUserSettings
		configs.ProjectKanukaSettings = originalProjectSettings
		configs.GlobalUserConfig = originalUserConfig
		configs.GlobalProjectConfig = originalProjectConfig
	}

	return tempDir, tempUserDir, privateKey, cleanup
}

// savePublicKeyToFile saves an RSA public key to a file in PEM format.
func savePublicKeyToFile(publicKey *rsa.PublicKey, filePath string) error {
	pubASN1, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return err
	}
	pubPem := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubASN1,
	}
	// #nosec G306 -- Public keys are intended to be world-readable
	return os.WriteFile(filePath, pem.EncodeToMemory(pubPem), 0644)
}

// savePrivateKeyToFile saves an RSA private key to a file in PEM format.
func savePrivateKeyToFile(privateKey *rsa.PrivateKey, filePath string) error {
	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privPem := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}
	return os.WriteFile(filePath, pem.EncodeToMemory(privPem), 0600)
}

// createEncryptedSecretFile creates an encrypted .kanuka secret file.
func createEncryptedSecretFile(t *testing.T, path string, plaintext []byte, symKey []byte) {
	t.Helper()

	var key [32]byte
	copy(key[:], symKey)

	var nonce [24]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		t.Fatalf("Failed to generate nonce: %v", err)
	}

	ciphertext := secretbox.Seal(nonce[:], plaintext, &nonce, &key)

	if err := os.WriteFile(path, ciphertext, 0600); err != nil {
		t.Fatalf("Failed to write encrypted file: %v", err)
	}
}

// decryptSecretFile decrypts a .kanuka secret file and returns the plaintext.
func decryptSecretFile(t *testing.T, path string, symKey []byte) []byte {
	t.Helper()

	ciphertext, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read encrypted file: %v", err)
	}

	if len(ciphertext) < 24 {
		t.Fatalf("Ciphertext too short")
	}

	var key [32]byte
	copy(key[:], symKey)

	var nonce [24]byte
	copy(nonce[:], ciphertext[:24])

	plaintext, ok := secretbox.Open(nil, ciphertext[24:], &nonce, &key)
	if !ok {
		t.Fatalf("Failed to decrypt file")
	}

	return plaintext
}

// getSymmetricKeyForUser decrypts the symmetric key for a user.
func getSymmetricKeyForUser(t *testing.T, userUUID string, privateKey *rsa.PrivateKey) []byte {
	t.Helper()

	encryptedSymKey, err := GetProjectKanukaKey(userUUID)
	if err != nil {
		t.Fatalf("Failed to get encrypted symmetric key: %v", err)
	}

	symKey, err := DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		t.Fatalf("Failed to decrypt symmetric key: %v", err)
	}

	return symKey
}

func TestSyncSecrets_NoSecretFiles(t *testing.T) {
	tempDir, _, privateKey, cleanup := setupSyncTestEnvironment(t)
	defer cleanup()

	// Don't create any secret files - just the user's .kanuka key

	opts := SyncOptions{
		Verbose: false,
		Debug:   false,
	}

	result, err := SyncSecrets(privateKey, opts)
	if err != nil {
		t.Fatalf("SyncSecrets failed: %v", err)
	}

	// Should succeed with 0 secrets processed
	if result.SecretsProcessed != 0 {
		t.Errorf("Expected 0 secrets processed, got %d", result.SecretsProcessed)
	}

	if result.UsersProcessed != 1 {
		t.Errorf("Expected 1 user processed, got %d", result.UsersProcessed)
	}

	// Verify new symmetric key was created for the user
	userKanukaPath := filepath.Join(tempDir, ".kanuka", "secrets", testUserUUID+".kanuka")
	if _, err := os.Stat(userKanukaPath); os.IsNotExist(err) {
		t.Errorf("User .kanuka file should still exist after sync")
	}
}

func TestSyncSecrets_SingleSecretFile(t *testing.T) {
	tempDir, _, privateKey, cleanup := setupSyncTestEnvironment(t)
	defer cleanup()

	// Get the current symmetric key
	originalSymKey := getSymmetricKeyForUser(t, testUserUUID, privateKey)

	// Create a secret file
	secretContent := []byte("API_KEY=secret123\nDB_PASSWORD=dbpass456")
	secretPath := filepath.Join(tempDir, ".env.kanuka")
	createEncryptedSecretFile(t, secretPath, secretContent, originalSymKey)

	opts := SyncOptions{
		Verbose: false,
		Debug:   false,
	}

	result, err := SyncSecrets(privateKey, opts)
	if err != nil {
		t.Fatalf("SyncSecrets failed: %v", err)
	}

	// Verify result
	if result.SecretsProcessed != 1 {
		t.Errorf("Expected 1 secret processed, got %d", result.SecretsProcessed)
	}

	if result.UsersProcessed != 1 {
		t.Errorf("Expected 1 user processed, got %d", result.UsersProcessed)
	}

	// Verify secret can be decrypted with new key
	newSymKey := getSymmetricKeyForUser(t, testUserUUID, privateKey)

	// New key should be different from original
	if string(newSymKey) == string(originalSymKey) {
		t.Errorf("New symmetric key should be different from original")
	}

	// Decrypt with new key and verify content
	decrypted := decryptSecretFile(t, secretPath, newSymKey)
	if string(decrypted) != string(secretContent) {
		t.Errorf("Decrypted content doesn't match original: got %q, want %q", decrypted, secretContent)
	}
}

func TestSyncSecrets_MultipleSecretFiles(t *testing.T) {
	tempDir, _, privateKey, cleanup := setupSyncTestEnvironment(t)
	defer cleanup()

	// Get the current symmetric key
	originalSymKey := getSymmetricKeyForUser(t, testUserUUID, privateKey)

	// Create multiple secret files
	secrets := map[string][]byte{
		".env.kanuka":        []byte("API_KEY=secret123"),
		".env.local.kanuka":  []byte("LOCAL_VAR=localvalue"),
		"config/.env.kanuka": []byte("CONFIG_VAR=configvalue"),
	}

	// Create config subdirectory
	if err := os.MkdirAll(filepath.Join(tempDir, "config"), 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	for path, content := range secrets {
		fullPath := filepath.Join(tempDir, path)
		createEncryptedSecretFile(t, fullPath, content, originalSymKey)
	}

	opts := SyncOptions{
		Verbose: false,
		Debug:   false,
	}

	result, err := SyncSecrets(privateKey, opts)
	if err != nil {
		t.Fatalf("SyncSecrets failed: %v", err)
	}

	// Verify result
	if result.SecretsProcessed != 3 {
		t.Errorf("Expected 3 secrets processed, got %d", result.SecretsProcessed)
	}

	// Verify all secrets can be decrypted with new key
	newSymKey := getSymmetricKeyForUser(t, testUserUUID, privateKey)

	for path, expectedContent := range secrets {
		fullPath := filepath.Join(tempDir, path)
		decrypted := decryptSecretFile(t, fullPath, newSymKey)
		if string(decrypted) != string(expectedContent) {
			t.Errorf("Decrypted content for %s doesn't match: got %q, want %q", path, decrypted, expectedContent)
		}
	}
}

func TestSyncSecrets_DryRun(t *testing.T) {
	tempDir, _, privateKey, cleanup := setupSyncTestEnvironment(t)
	defer cleanup()

	// Get the current symmetric key
	originalSymKey := getSymmetricKeyForUser(t, testUserUUID, privateKey)

	// Create a secret file
	secretContent := []byte("API_KEY=secret123")
	secretPath := filepath.Join(tempDir, ".env.kanuka")
	createEncryptedSecretFile(t, secretPath, secretContent, originalSymKey)

	// Read original file contents
	originalCiphertext, err := os.ReadFile(secretPath)
	if err != nil {
		t.Fatalf("Failed to read original ciphertext: %v", err)
	}

	originalUserKanukaPath := filepath.Join(tempDir, ".kanuka", "secrets", testUserUUID+".kanuka")
	originalUserKanukaContent, err := os.ReadFile(originalUserKanukaPath)
	if err != nil {
		t.Fatalf("Failed to read original user .kanuka file: %v", err)
	}

	opts := SyncOptions{
		DryRun:  true,
		Verbose: false,
		Debug:   false,
	}

	result, err := SyncSecrets(privateKey, opts)
	if err != nil {
		t.Fatalf("SyncSecrets failed: %v", err)
	}

	// Verify result shows what would have been done
	if result.SecretsProcessed != 1 {
		t.Errorf("Expected 1 secret processed (dry-run), got %d", result.SecretsProcessed)
	}

	if result.UsersProcessed != 1 {
		t.Errorf("Expected 1 user processed (dry-run), got %d", result.UsersProcessed)
	}

	// Verify files were NOT modified
	currentCiphertext, err := os.ReadFile(secretPath)
	if err != nil {
		t.Fatalf("Failed to read current ciphertext: %v", err)
	}

	if string(currentCiphertext) != string(originalCiphertext) {
		t.Errorf("Secret file should not have been modified in dry-run mode")
	}

	currentUserKanukaContent, err := os.ReadFile(originalUserKanukaPath)
	if err != nil {
		t.Fatalf("Failed to read current user .kanuka file: %v", err)
	}

	if string(currentUserKanukaContent) != string(originalUserKanukaContent) {
		t.Errorf("User .kanuka file should not have been modified in dry-run mode")
	}
}

func TestSyncSecrets_MultipleUsers(t *testing.T) {
	tempDir, tempUserDir, privateKey, cleanup := setupSyncTestEnvironment(t)
	defer cleanup()

	// Add a second user
	privateKey2, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate second RSA key: %v", err)
	}

	publicKeysDir := configs.ProjectKanukaSettings.ProjectPublicKeyPath
	secretsDir := configs.ProjectKanukaSettings.ProjectSecretsPath

	// Save second user's public key
	pubKey2Path := filepath.Join(publicKeysDir, testUser2UUID+".pub")
	if err := savePublicKeyToFile(&privateKey2.PublicKey, pubKey2Path); err != nil {
		t.Fatalf("Failed to save second public key: %v", err)
	}

	// Get original symmetric key
	originalSymKey := getSymmetricKeyForUser(t, testUserUUID, privateKey)

	// Encrypt symmetric key for second user
	encryptedSymKey2, err := EncryptWithPublicKey(originalSymKey, &privateKey2.PublicKey)
	if err != nil {
		t.Fatalf("Failed to encrypt symmetric key for second user: %v", err)
	}

	// Save encrypted symmetric key for second user
	user2KanukaPath := filepath.Join(secretsDir, testUser2UUID+".kanuka")
	if err := os.WriteFile(user2KanukaPath, encryptedSymKey2, 0600); err != nil {
		t.Fatalf("Failed to save encrypted symmetric key for second user: %v", err)
	}

	// Also set up second user's private key directory
	keyDir2 := filepath.Join(tempUserDir, "keys", testProjectUUID+"_user2")
	if err := os.MkdirAll(keyDir2, 0700); err != nil {
		t.Fatalf("Failed to create second key directory: %v", err)
	}
	privKey2Path := filepath.Join(keyDir2, "privkey")
	if err := savePrivateKeyToFile(privateKey2, privKey2Path); err != nil {
		t.Fatalf("Failed to save second private key: %v", err)
	}

	// Create a secret file
	secretContent := []byte("API_KEY=secret123")
	secretPath := filepath.Join(tempDir, ".env.kanuka")
	createEncryptedSecretFile(t, secretPath, secretContent, originalSymKey)

	opts := SyncOptions{
		Verbose: false,
		Debug:   false,
	}

	result, err := SyncSecrets(privateKey, opts)
	if err != nil {
		t.Fatalf("SyncSecrets failed: %v", err)
	}

	// Verify result
	if result.UsersProcessed != 2 {
		t.Errorf("Expected 2 users processed, got %d", result.UsersProcessed)
	}

	// Verify both users can decrypt with new key
	newSymKey1 := getSymmetricKeyForUser(t, testUserUUID, privateKey)

	// Get new symmetric key for user 2
	encryptedSymKey2New, err := os.ReadFile(user2KanukaPath)
	if err != nil {
		t.Fatalf("Failed to read encrypted symmetric key for second user: %v", err)
	}
	newSymKey2, err := DecryptWithPrivateKey(encryptedSymKey2New, privateKey2)
	if err != nil {
		t.Fatalf("Failed to decrypt symmetric key for second user: %v", err)
	}

	// Both users should have the same new symmetric key
	if string(newSymKey1) != string(newSymKey2) {
		t.Errorf("Both users should have the same new symmetric key")
	}

	// Decrypt secret with new key
	decrypted := decryptSecretFile(t, secretPath, newSymKey1)
	if string(decrypted) != string(secretContent) {
		t.Errorf("Decrypted content doesn't match: got %q, want %q", decrypted, secretContent)
	}
}

func TestSyncSecrets_ExcludeUser(t *testing.T) {
	tempDir, _, privateKey, cleanup := setupSyncTestEnvironment(t)
	defer cleanup()

	// Add a second user
	privateKey2, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate second RSA key: %v", err)
	}

	publicKeysDir := configs.ProjectKanukaSettings.ProjectPublicKeyPath
	secretsDir := configs.ProjectKanukaSettings.ProjectSecretsPath

	// Save second user's public key
	pubKey2Path := filepath.Join(publicKeysDir, testUser2UUID+".pub")
	if err := savePublicKeyToFile(&privateKey2.PublicKey, pubKey2Path); err != nil {
		t.Fatalf("Failed to save second public key: %v", err)
	}

	// Get original symmetric key
	originalSymKey := getSymmetricKeyForUser(t, testUserUUID, privateKey)

	// Encrypt symmetric key for second user
	encryptedSymKey2, err := EncryptWithPublicKey(originalSymKey, &privateKey2.PublicKey)
	if err != nil {
		t.Fatalf("Failed to encrypt symmetric key for second user: %v", err)
	}

	// Save encrypted symmetric key for second user
	user2KanukaPath := filepath.Join(secretsDir, testUser2UUID+".kanuka")
	if err := os.WriteFile(user2KanukaPath, encryptedSymKey2, 0600); err != nil {
		t.Fatalf("Failed to save encrypted symmetric key for second user: %v", err)
	}

	// Create a secret file
	secretContent := []byte("API_KEY=secret123")
	secretPath := filepath.Join(tempDir, ".env.kanuka")
	createEncryptedSecretFile(t, secretPath, secretContent, originalSymKey)

	// Exclude the second user (simulating revoke)
	opts := SyncOptions{
		ExcludeUsers: []string{testUser2UUID},
		Verbose:      false,
		Debug:        false,
	}

	result, err := SyncSecrets(privateKey, opts)
	if err != nil {
		t.Fatalf("SyncSecrets failed: %v", err)
	}

	// Verify result
	if result.UsersProcessed != 1 {
		t.Errorf("Expected 1 user processed, got %d", result.UsersProcessed)
	}

	if result.UsersExcluded != 1 {
		t.Errorf("Expected 1 user excluded, got %d", result.UsersExcluded)
	}

	// Verify second user's .kanuka file was deleted
	if _, err := os.Stat(user2KanukaPath); !os.IsNotExist(err) {
		t.Errorf("Excluded user's .kanuka file should have been deleted")
	}

	// Verify first user can still decrypt
	newSymKey := getSymmetricKeyForUser(t, testUserUUID, privateKey)
	decrypted := decryptSecretFile(t, secretPath, newSymKey)
	if string(decrypted) != string(secretContent) {
		t.Errorf("Decrypted content doesn't match: got %q, want %q", decrypted, secretContent)
	}

	// Verify second user's old key cannot decrypt the new files
	// (This would fail because the file was re-encrypted with a new key)
	// We can verify this by trying to decrypt with the old symmetric key
	newCiphertext, err := os.ReadFile(secretPath)
	if err != nil {
		t.Fatalf("Failed to read new ciphertext: %v", err)
	}

	var oldKey [32]byte
	copy(oldKey[:], originalSymKey)

	var nonce [24]byte
	copy(nonce[:], newCiphertext[:24])

	_, ok := secretbox.Open(nil, newCiphertext[24:], &nonce, &oldKey)
	if ok {
		t.Errorf("Old symmetric key should NOT be able to decrypt the new files")
	}
}

func TestSyncSecrets_DecryptionFailure(t *testing.T) {
	_, _, privateKey, cleanup := setupSyncTestEnvironment(t)
	defer cleanup()

	// Generate a different key to cause decryption failure
	wrongKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate wrong RSA key: %v", err)
	}

	opts := SyncOptions{
		Verbose: false,
		Debug:   false,
	}

	// This should fail because we're using the wrong private key
	_, err = SyncSecrets(wrongKey, opts)
	if err == nil {
		t.Fatalf("SyncSecrets should have failed with wrong private key")
	}

	// We only need to verify the function failed appropriately
	// The private key passed doesn't match what's stored
	_ = privateKey // Acknowledge we have the correct key but didn't use it
}

func TestSyncSecrets_NoUsersAfterExclusion(t *testing.T) {
	_, _, privateKey, cleanup := setupSyncTestEnvironment(t)
	defer cleanup()

	// Exclude the only user
	opts := SyncOptions{
		ExcludeUsers: []string{testUserUUID},
		Verbose:      false,
		Debug:        false,
	}

	_, err := SyncSecrets(privateKey, opts)
	if err == nil {
		t.Fatalf("SyncSecrets should have failed when all users are excluded")
	}

	// Verify error message mentions no active users
	if err.Error() != "no active users remaining after exclusions" {
		t.Errorf("Expected 'no active users remaining' error, got: %v", err)
	}
}

func TestSyncSecretsSimple(t *testing.T) {
	tempDir, _, privateKey, cleanup := setupSyncTestEnvironment(t)
	defer cleanup()

	// Get the current symmetric key
	originalSymKey := getSymmetricKeyForUser(t, testUserUUID, privateKey)

	// Create a secret file
	secretContent := []byte("API_KEY=secret123")
	secretPath := filepath.Join(tempDir, ".env.kanuka")
	createEncryptedSecretFile(t, secretPath, secretContent, originalSymKey)

	// Use the simple wrapper function
	err := SyncSecretsSimple(testUserUUID, privateKey, false)
	if err != nil {
		t.Fatalf("SyncSecretsSimple failed: %v", err)
	}

	// Verify secret can be decrypted with new key
	newSymKey := getSymmetricKeyForUser(t, testUserUUID, privateKey)

	// New key should be different from original
	if string(newSymKey) == string(originalSymKey) {
		t.Errorf("New symmetric key should be different from original")
	}

	// Decrypt with new key and verify content
	decrypted := decryptSecretFile(t, secretPath, newSymKey)
	if string(decrypted) != string(secretContent) {
		t.Errorf("Decrypted content doesn't match original: got %q, want %q", decrypted, secretContent)
	}
}
