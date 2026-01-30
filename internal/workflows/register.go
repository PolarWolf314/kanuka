package workflows

import (
	"context"
	"crypto/rsa"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/audit"
	"github.com/PolarWolf314/kanuka/internal/configs"
	kerrors "github.com/PolarWolf314/kanuka/internal/errors"
	"github.com/PolarWolf314/kanuka/internal/secrets"
)

// RegisterMode indicates how the user is being registered.
type RegisterMode string

const (
	// RegisterModeEmail registers a user by looking up their email in the project config.
	RegisterModeEmail RegisterMode = "email"
	// RegisterModePubkeyText registers a user with provided public key text.
	RegisterModePubkeyText RegisterMode = "pubkey_text"
	// RegisterModeFile registers a user from a public key file.
	RegisterModeFile RegisterMode = "file"
)

// RegisterOptions configures the register workflow.
type RegisterOptions struct {
	// Mode specifies how the user is being registered.
	Mode RegisterMode

	// UserEmail is the email of the user to register (required for email and pubkey_text modes).
	UserEmail string

	// PublicKeyText contains the public key content (for pubkey_text mode).
	PublicKeyText string

	// FilePath is the path to the public key file (for file mode).
	FilePath string

	// DryRun previews registration without making changes.
	DryRun bool

	// PrivateKeyData contains the private key bytes when reading from stdin.
	PrivateKeyData []byte

	// Force skips confirmation when updating existing user's access.
	Force bool

	// Verbose enables verbose output.
	Verbose bool

	// Debug enables debug output.
	Debug bool
}

// RegisterResult contains the outcome of a register operation.
type RegisterResult struct {
	// DisplayName is the user-friendly name of who was registered.
	DisplayName string

	// TargetUserUUID is the UUID of the registered user.
	TargetUserUUID string

	// FilesCreated lists files that were created.
	FilesCreated []RegisteredFile

	// FilesUpdated lists files that were updated.
	FilesUpdated []RegisteredFile

	// DryRun indicates whether this was a dry-run (no changes made).
	DryRun bool

	// UserAlreadyHadAccess indicates if user already had access before this registration.
	UserAlreadyHadAccess bool

	// PubKeyPath is the path where the public key is/would be stored.
	PubKeyPath string

	// KanukaFilePath is the path where the .kanuka key is/would be stored.
	KanukaFilePath string

	// Mode indicates which registration mode was used.
	Mode RegisterMode
}

// RegisteredFile represents a file that was created or updated.
type RegisteredFile struct {
	Type string // "public_key" or "encrypted_key"
	Path string
}

// Register grants a user access to the project's encrypted secrets.
//
// It encrypts the project's symmetric key with the target user's public key,
// allowing them to decrypt secrets. The caller must have access to the
// project's secrets before they can grant access to others.
//
// Returns ErrProjectNotInitialized if the project has no .kanuka directory.
// Returns ErrUserNotFound if the specified user is not in the project config.
// Returns ErrNoAccess if the current user doesn't have access to the project.
// Returns ErrPublicKeyNotFound if the target user's public key cannot be found.
func Register(ctx context.Context, opts RegisterOptions) (*RegisterResult, error) {
	if err := configs.InitProjectSettings(); err != nil {
		return nil, fmt.Errorf("initializing project settings: %w", err)
	}

	projectPath := configs.ProjectKanukaSettings.ProjectPath
	if projectPath == "" {
		return nil, kerrors.ErrProjectNotInitialized
	}

	switch opts.Mode {
	case RegisterModePubkeyText:
		return registerWithPubkeyText(ctx, opts)
	case RegisterModeFile:
		return registerWithFile(ctx, opts)
	default:
		return registerByEmail(ctx, opts)
	}
}

// registerByEmail handles registration when only user email is provided.
func registerByEmail(ctx context.Context, opts RegisterOptions) (*RegisterResult, error) {
	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath
	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath

	userConfig, err := configs.EnsureUserConfig()
	if err != nil {
		return nil, fmt.Errorf("loading user config: %w", err)
	}
	currentUserUUID := userConfig.User.UUID

	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		return nil, fmt.Errorf("loading project config: %w", err)
	}
	projectUUID := projectConfig.Project.UUID

	// Look up user UUID by email.
	targetUserUUID, found := projectConfig.GetUserUUIDByEmail(opts.UserEmail)
	if !found {
		return nil, fmt.Errorf("%w: %s", kerrors.ErrUserNotFound, opts.UserEmail)
	}

	// Check if target user's public key exists.
	targetPubkeyPath := filepath.Join(projectPublicKeyPath, targetUserUUID+".pub")
	targetUserPublicKey, err := secrets.LoadPublicKey(targetPubkeyPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", kerrors.ErrPublicKeyNotFound, opts.UserEmail)
	}

	// Verify current user has access.
	encryptedSymKey, err := secrets.GetProjectKanukaKey(currentUserUUID)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot get kanuka key", kerrors.ErrNoAccess)
	}

	privateKey, err := loadPrivateKeyForRegister(opts.PrivateKeyData, projectUUID)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot load private key: %v", kerrors.ErrNoAccess, err)
	}

	symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot decrypt symmetric key", kerrors.ErrNoAccess)
	}

	// Compute paths.
	targetKanukaFilePath := filepath.Join(projectSecretsPath, targetUserUUID+".kanuka")

	// Check if files exist.
	pubkeyExisted := fileExistsForWorkflow(targetPubkeyPath)
	kanukaFileExisted := fileExistsForWorkflow(targetKanukaFilePath)
	userAlreadyHasAccess := pubkeyExisted && kanukaFileExisted

	result := &RegisterResult{
		DisplayName:          opts.UserEmail,
		TargetUserUUID:       targetUserUUID,
		DryRun:               opts.DryRun,
		UserAlreadyHadAccess: userAlreadyHasAccess,
		PubKeyPath:           targetPubkeyPath,
		KanukaFilePath:       targetKanukaFilePath,
		Mode:                 RegisterModeEmail,
	}

	if opts.DryRun {
		return result, nil
	}

	// Encrypt symmetric key with target user's public key.
	targetEncryptedSymKey, err := secrets.EncryptWithPublicKey(symKey, targetUserPublicKey)
	if err != nil {
		return nil, fmt.Errorf("encrypting symmetric key: %w", err)
	}

	// Save encrypted symmetric key for target user.
	if err := secrets.SaveKanukaKeyToProject(targetUserUUID, targetEncryptedSymKey); err != nil {
		return nil, fmt.Errorf("saving encrypted key: %w", err)
	}

	// Record which files were created/updated.
	if !kanukaFileExisted {
		result.FilesCreated = append(result.FilesCreated, RegisteredFile{Type: "encrypted_key", Path: targetKanukaFilePath})
	} else {
		result.FilesUpdated = append(result.FilesUpdated, RegisteredFile{Type: "encrypted_key", Path: targetKanukaFilePath})
	}

	// Log to audit trail.
	auditEntry := audit.LogWithUser("register")
	auditEntry.TargetUser = opts.UserEmail
	auditEntry.TargetUUID = targetUserUUID
	audit.Log(auditEntry)

	return result, nil
}

// registerWithPubkeyText handles registration with provided public key text.
func registerWithPubkeyText(ctx context.Context, opts RegisterOptions) (*RegisterResult, error) {
	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath
	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath

	userConfig, err := configs.EnsureUserConfig()
	if err != nil {
		return nil, fmt.Errorf("loading user config: %w", err)
	}
	currentUserUUID := userConfig.User.UUID

	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		return nil, fmt.Errorf("loading project config: %w", err)
	}
	projectUUID := projectConfig.Project.UUID

	// Look up user UUID by email.
	targetUserUUID, found := projectConfig.GetUserUUIDByEmail(opts.UserEmail)
	if !found {
		return nil, fmt.Errorf("%w: %s", kerrors.ErrUserNotFound, opts.UserEmail)
	}

	// Parse the public key text.
	publicKey, err := secrets.ParsePublicKeyText(opts.PublicKeyText)
	if err != nil {
		return nil, fmt.Errorf("invalid public key format: %w", err)
	}

	// Verify current user has access.
	encryptedSymKey, err := secrets.GetProjectKanukaKey(currentUserUUID)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot get kanuka key", kerrors.ErrNoAccess)
	}

	privateKey, err := loadPrivateKeyForRegister(opts.PrivateKeyData, projectUUID)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot load private key: %v", kerrors.ErrNoAccess, err)
	}

	symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot decrypt symmetric key", kerrors.ErrNoAccess)
	}

	// Compute paths.
	pubKeyFilePath := filepath.Join(projectPublicKeyPath, targetUserUUID+".pub")
	kanukaFilePath := filepath.Join(projectSecretsPath, targetUserUUID+".kanuka")

	// Check if files exist.
	pubkeyExisted := fileExistsForWorkflow(pubKeyFilePath)
	kanukaFileExisted := fileExistsForWorkflow(kanukaFilePath)
	userAlreadyHasAccess := pubkeyExisted && kanukaFileExisted

	result := &RegisterResult{
		DisplayName:          opts.UserEmail,
		TargetUserUUID:       targetUserUUID,
		DryRun:               opts.DryRun,
		UserAlreadyHadAccess: userAlreadyHasAccess,
		PubKeyPath:           pubKeyFilePath,
		KanukaFilePath:       kanukaFilePath,
		Mode:                 RegisterModePubkeyText,
	}

	if opts.DryRun {
		return result, nil
	}

	// Save the public key to a file.
	if err := secrets.SavePublicKeyToFile(publicKey, pubKeyFilePath); err != nil {
		return nil, fmt.Errorf("saving public key: %w", err)
	}

	if !pubkeyExisted {
		result.FilesCreated = append(result.FilesCreated, RegisteredFile{Type: "public_key", Path: pubKeyFilePath})
	} else {
		result.FilesUpdated = append(result.FilesUpdated, RegisteredFile{Type: "public_key", Path: pubKeyFilePath})
	}

	// Encrypt symmetric key with target user's public key.
	targetEncryptedSymKey, err := secrets.EncryptWithPublicKey(symKey, publicKey)
	if err != nil {
		return nil, fmt.Errorf("encrypting symmetric key: %w", err)
	}

	// Save encrypted symmetric key for target user.
	if err := secrets.SaveKanukaKeyToProject(targetUserUUID, targetEncryptedSymKey); err != nil {
		return nil, fmt.Errorf("saving encrypted key: %w", err)
	}

	if !kanukaFileExisted {
		result.FilesCreated = append(result.FilesCreated, RegisteredFile{Type: "encrypted_key", Path: kanukaFilePath})
	} else {
		result.FilesUpdated = append(result.FilesUpdated, RegisteredFile{Type: "encrypted_key", Path: kanukaFilePath})
	}

	// Log to audit trail.
	auditEntry := audit.LogWithUser("register")
	auditEntry.TargetUser = opts.UserEmail
	auditEntry.TargetUUID = targetUserUUID
	audit.Log(auditEntry)

	return result, nil
}

// registerWithFile handles registration from a public key file.
func registerWithFile(ctx context.Context, opts RegisterOptions) (*RegisterResult, error) {
	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath
	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath

	userConfig, err := configs.EnsureUserConfig()
	if err != nil {
		return nil, fmt.Errorf("loading user config: %w", err)
	}
	currentUserUUID := userConfig.User.UUID

	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		return nil, fmt.Errorf("loading project config: %w", err)
	}
	projectUUID := projectConfig.Project.UUID

	// Validate file path.
	if !strings.HasSuffix(opts.FilePath, ".pub") {
		return nil, fmt.Errorf("%w: file must have .pub extension", kerrors.ErrInvalidFileType)
	}

	filename := filepath.Base(opts.FilePath)
	targetUserUUID := strings.TrimSuffix(filename, ".pub")

	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	if !uuidRegex.MatchString(targetUserUUID) {
		return nil, fmt.Errorf("%w: public key file must be named <uuid>.pub", kerrors.ErrInvalidFileType)
	}

	// Load the public key from file.
	targetUserPublicKey, err := secrets.LoadPublicKey(opts.FilePath)
	if err != nil {
		return nil, fmt.Errorf("loading public key from file: %w", err)
	}

	// Verify current user has access.
	encryptedSymKey, err := secrets.GetProjectKanukaKey(currentUserUUID)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot get kanuka key", kerrors.ErrNoAccess)
	}

	privateKey, err := loadPrivateKeyForRegister(opts.PrivateKeyData, projectUUID)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot load private key: %v", kerrors.ErrNoAccess, err)
	}

	symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot decrypt symmetric key", kerrors.ErrNoAccess)
	}

	// Try to find email for display purposes.
	targetEmail := projectConfig.Users[targetUserUUID]
	displayName := targetEmail
	if displayName == "" {
		if opts.UserEmail == "" {
			return nil, fmt.Errorf("%w: UUID %s not found in project, provide --user flag", kerrors.ErrUserNotFound, targetUserUUID)
		}
		displayName = opts.UserEmail
	}

	// Compute paths.
	targetPubkeyPath := filepath.Join(projectPublicKeyPath, targetUserUUID+".pub")
	targetKanukaFilePath := filepath.Join(projectSecretsPath, targetUserUUID+".kanuka")

	// Check if files exist.
	pubkeyExisted := fileExistsForWorkflow(targetPubkeyPath)
	kanukaFileExisted := fileExistsForWorkflow(targetKanukaFilePath)
	userAlreadyHasAccess := fileExistsForWorkflow(opts.FilePath) && kanukaFileExisted

	result := &RegisterResult{
		DisplayName:          displayName,
		TargetUserUUID:       targetUserUUID,
		DryRun:               opts.DryRun,
		UserAlreadyHadAccess: userAlreadyHasAccess,
		PubKeyPath:           targetPubkeyPath,
		KanukaFilePath:       targetKanukaFilePath,
		Mode:                 RegisterModeFile,
	}

	if opts.DryRun {
		return result, nil
	}

	// Copy public key to project if it doesn't exist there.
	if !pubkeyExisted {
		if err := secrets.SavePublicKeyToFile(targetUserPublicKey, targetPubkeyPath); err != nil {
			return nil, fmt.Errorf("saving public key to project: %w", err)
		}
		result.FilesCreated = append(result.FilesCreated, RegisteredFile{Type: "public_key", Path: targetPubkeyPath})

		// Add user to project config if email is provided.
		if opts.UserEmail != "" && projectConfig.Users[targetUserUUID] == "" {
			projectConfig.Users[targetUserUUID] = opts.UserEmail
			if err := configs.SaveProjectConfig(projectConfig); err != nil {
				return nil, fmt.Errorf("updating project config: %w", err)
			}
		}
	}

	// Encrypt symmetric key with target user's public key.
	targetEncryptedSymKey, err := secrets.EncryptWithPublicKey(symKey, targetUserPublicKey)
	if err != nil {
		return nil, fmt.Errorf("encrypting symmetric key: %w", err)
	}

	// Save encrypted symmetric key for target user.
	if err := secrets.SaveKanukaKeyToProject(targetUserUUID, targetEncryptedSymKey); err != nil {
		return nil, fmt.Errorf("saving encrypted key: %w", err)
	}

	if !kanukaFileExisted {
		result.FilesCreated = append(result.FilesCreated, RegisteredFile{Type: "encrypted_key", Path: targetKanukaFilePath})
	} else {
		result.FilesUpdated = append(result.FilesUpdated, RegisteredFile{Type: "encrypted_key", Path: targetKanukaFilePath})
	}

	// Log to audit trail.
	auditEntry := audit.LogWithUser("register")
	auditEntry.TargetUser = displayName
	auditEntry.TargetUUID = targetUserUUID
	audit.Log(auditEntry)

	return result, nil
}

// loadPrivateKeyForRegister loads the private key from bytes or disk.
func loadPrivateKeyForRegister(keyData []byte, projectUUID string) (*rsa.PrivateKey, error) {
	if len(keyData) > 0 {
		return secrets.LoadPrivateKeyFromBytesWithTTYPrompt(keyData)
	}
	privateKeyPath := configs.GetPrivateKeyPath(projectUUID)
	return secrets.LoadPrivateKey(privateKeyPath)
}

// fileExistsForWorkflow checks if a file exists and is not a directory.
func fileExistsForWorkflow(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil && !info.IsDir()
}

// CheckUserExistsForRegistration checks if a user can be registered (exists in project config).
// Returns the target user UUID and whether they already have access.
func CheckUserExistsForRegistration(userEmail string) (targetUUID string, alreadyHasAccess bool, err error) {
	if err := configs.InitProjectSettings(); err != nil {
		return "", false, fmt.Errorf("initializing project settings: %w", err)
	}

	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		return "", false, fmt.Errorf("loading project config: %w", err)
	}

	targetUserUUID, found := projectConfig.GetUserUUIDByEmail(userEmail)
	if !found {
		return "", false, fmt.Errorf("%w: %s", kerrors.ErrUserNotFound, userEmail)
	}

	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath
	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath

	pubkeyPath := filepath.Join(projectPublicKeyPath, targetUserUUID+".pub")
	kanukaPath := filepath.Join(projectSecretsPath, targetUserUUID+".kanuka")

	pubkeyExists := fileExistsForWorkflow(pubkeyPath)
	kanukaExists := fileExistsForWorkflow(kanukaPath)

	return targetUserUUID, pubkeyExists && kanukaExists, nil
}
