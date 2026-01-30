package workflows

import (
	"context"
	"crypto/rsa"
	"fmt"
	"os"
	"path/filepath"

	"github.com/PolarWolf314/kanuka/internal/audit"
	"github.com/PolarWolf314/kanuka/internal/configs"
	kerrors "github.com/PolarWolf314/kanuka/internal/errors"
	"github.com/PolarWolf314/kanuka/internal/secrets"
)

// RevokeOptions configures the revoke workflow.
type RevokeOptions struct {
	// UserEmail is the email of the user whose access is being revoked.
	UserEmail string

	// FilePath is an alternative way to specify revocation by .kanuka file path.
	FilePath string

	// DeviceName specifies a specific device to revoke (requires UserEmail).
	DeviceName string

	// DryRun previews revocation without making changes.
	DryRun bool

	// PrivateKeyData contains the private key bytes when reading from stdin.
	PrivateKeyData []byte

	// Verbose enables verbose output.
	Verbose bool

	// Debug enables debug output.
	Debug bool
}

// RevokeResult contains the outcome of a revoke operation.
type RevokeResult struct {
	// DisplayName is the user-friendly name of who was revoked.
	DisplayName string

	// RevokedFiles lists the files that were deleted.
	RevokedFiles []string

	// UUIDsRevoked lists the UUIDs that were removed from config.
	UUIDsRevoked []string

	// RemainingUsers is the count of users still in the project.
	RemainingUsers int

	// SecretsReEncrypted is the count of secrets re-encrypted.
	SecretsReEncrypted int

	// DryRun indicates whether this was a dry-run (no changes made).
	DryRun bool

	// FilesToDelete lists files that would be deleted (for dry-run).
	FilesToDelete []FileToRevoke

	// AllUsers lists all users currently in the project (for dry-run info).
	AllUsers []string

	// KanukaFilesCount is the number of .kanuka secret files (for dry-run info).
	KanukaFilesCount int
}

// FileToRevoke represents a file to be revoked.
type FileToRevoke struct {
	Path string
	Name string
}

// revokeContext holds context for the revocation operation.
type revokeContext struct {
	displayName  string
	files        []FileToRevoke
	uuidsRevoked []string
}

// Revoke revokes a user's access to project secrets.
//
// It removes the user's encrypted symmetric key and public key files,
// updates the project configuration, and re-encrypts all secrets with
// a new key so the revoked user cannot decrypt future secrets.
//
// Returns ErrProjectNotInitialized if the project has no .kanuka directory.
// Returns ErrUserNotFound if the specified user is not in the project.
// Returns ErrDeviceNotFound if the specified device is not found.
// Returns ErrSelfRevoke if attempting to revoke the current user.
func Revoke(ctx context.Context, opts RevokeOptions) (*RevokeResult, error) {
	if err := configs.InitProjectSettings(); err != nil {
		return nil, fmt.Errorf("initializing project settings: %w", err)
	}

	projectPath := configs.ProjectKanukaSettings.ProjectPath
	if projectPath == "" {
		return nil, kerrors.ErrProjectNotInitialized
	}

	exists, err := secrets.DoesProjectKanukaSettingsExist()
	if err != nil {
		return nil, fmt.Errorf("checking project existence: %w", err)
	}
	if !exists {
		return nil, kerrors.ErrProjectNotInitialized
	}

	revokeCtx, err := getFilesToRevokeForWorkflow(opts)
	if err != nil {
		return nil, err
	}

	if revokeCtx == nil || len(revokeCtx.files) == 0 {
		return nil, kerrors.ErrUserNotFound
	}

	if opts.DryRun {
		return buildDryRunResult(revokeCtx)
	}

	return executeRevoke(revokeCtx, opts)
}

// getFilesToRevokeForWorkflow determines which files to revoke based on options.
func getFilesToRevokeForWorkflow(opts RevokeOptions) (*revokeContext, error) {
	if opts.UserEmail != "" {
		return getFilesByUserEmailForWorkflow(opts)
	}
	return getFilesByPathForWorkflow(opts.FilePath)
}

// getFilesByUserEmailForWorkflow finds files to revoke by user email.
func getFilesByUserEmailForWorkflow(opts RevokeOptions) (*revokeContext, error) {
	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath
	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath

	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		return nil, fmt.Errorf("loading project config: %w", err)
	}

	devices := projectConfig.GetDevicesByEmail(opts.UserEmail)
	if len(devices) == 0 {
		return nil, fmt.Errorf("%w: %s", kerrors.ErrUserNotFound, opts.UserEmail)
	}

	if opts.DeviceName != "" {
		targetUserUUID, found := projectConfig.GetUserUUIDByEmailAndDevice(opts.UserEmail, opts.DeviceName)
		if !found {
			return nil, fmt.Errorf("%w: %s for user %s", kerrors.ErrDeviceNotFound, opts.DeviceName, opts.UserEmail)
		}
		return getFilesForUUIDForWorkflow(targetUserUUID, opts.UserEmail+" ("+opts.DeviceName+")")
	}

	var allFiles []FileToRevoke
	var allUUIDs []string
	for userUUID := range devices {
		allUUIDs = append(allUUIDs, userUUID)
		publicKeyPath := filepath.Join(projectPublicKeyPath, userUUID+".pub")
		kanukaKeyPath := filepath.Join(projectSecretsPath, userUUID+".kanuka")

		if _, err := os.Stat(publicKeyPath); err == nil {
			allFiles = append(allFiles, FileToRevoke{Path: publicKeyPath, Name: userUUID + ".pub"})
		}
		if _, err := os.Stat(kanukaKeyPath); err == nil {
			allFiles = append(allFiles, FileToRevoke{Path: kanukaKeyPath, Name: userUUID + ".kanuka"})
		}
	}

	if len(allFiles) == 0 {
		return nil, fmt.Errorf("%w: no files found for %s", kerrors.ErrUserNotFound, opts.UserEmail)
	}

	return &revokeContext{
		displayName:  opts.UserEmail,
		files:        allFiles,
		uuidsRevoked: allUUIDs,
	}, nil
}

// getFilesForUUIDForWorkflow finds files for a specific UUID.
func getFilesForUUIDForWorkflow(userUUID, displayName string) (*revokeContext, error) {
	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath
	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath

	publicKeyPath := filepath.Join(projectPublicKeyPath, userUUID+".pub")
	kanukaKeyPath := filepath.Join(projectSecretsPath, userUUID+".kanuka")

	publicKeyExists := false
	kanukaKeyExists := false

	if _, err := os.Stat(publicKeyPath); err == nil {
		publicKeyExists = true
	}
	if _, err := os.Stat(kanukaKeyPath); err == nil {
		kanukaKeyExists = true
	}

	if !publicKeyExists && !kanukaKeyExists {
		return nil, fmt.Errorf("%w: %s", kerrors.ErrUserNotFound, displayName)
	}

	var files []FileToRevoke
	if publicKeyExists {
		files = append(files, FileToRevoke{Path: publicKeyPath, Name: userUUID + ".pub"})
	}
	if kanukaKeyExists {
		files = append(files, FileToRevoke{Path: kanukaKeyPath, Name: userUUID + ".kanuka"})
	}

	return &revokeContext{
		displayName:  displayName,
		files:        files,
		uuidsRevoked: []string{userUUID},
	}, nil
}

// getFilesByPathForWorkflow finds files to revoke by file path.
func getFilesByPathForWorkflow(filePath string) (*revokeContext, error) {
	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath
	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath

	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("resolving file path: %w", err)
	}

	fileInfo, err := os.Stat(absFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", kerrors.ErrFileNotFound, absFilePath)
		}
		return nil, fmt.Errorf("checking file: %w", err)
	}

	if fileInfo.IsDir() {
		return nil, fmt.Errorf("%w: path is a directory", kerrors.ErrInvalidFileType)
	}

	absProjectSecretsPath, err := filepath.Abs(projectSecretsPath)
	if err != nil {
		return nil, fmt.Errorf("resolving project secrets path: %w", err)
	}

	if filepath.Dir(absFilePath) != absProjectSecretsPath {
		return nil, fmt.Errorf("%w: file not in project secrets directory", kerrors.ErrInvalidFileType)
	}

	if filepath.Ext(absFilePath) != ".kanuka" {
		return nil, fmt.Errorf("%w: not a .kanuka file", kerrors.ErrInvalidFileType)
	}

	baseName := filepath.Base(absFilePath)
	userUUID := baseName[:len(baseName)-len(".kanuka")]

	projectConfig, err := configs.LoadProjectConfig()
	displayName := userUUID
	if err == nil {
		if email, exists := projectConfig.Users[userUUID]; exists && email != "" {
			displayName = email
		}
	}

	var files []FileToRevoke
	files = append(files, FileToRevoke{Path: absFilePath, Name: baseName})

	publicKeyPath := filepath.Join(projectPublicKeyPath, userUUID+".pub")
	if _, err := os.Stat(publicKeyPath); err == nil {
		files = append(files, FileToRevoke{Path: publicKeyPath, Name: userUUID + ".pub"})
	}

	return &revokeContext{
		displayName:  displayName,
		files:        files,
		uuidsRevoked: []string{userUUID},
	}, nil
}

// buildDryRunResult builds a result for dry-run mode.
func buildDryRunResult(revokeCtx *revokeContext) (*RevokeResult, error) {
	allUsers, _ := secrets.GetAllUsersInProject()

	kanukaFilesCount := 0
	projectPath := configs.ProjectKanukaSettings.ProjectPath
	if projectPath != "" {
		kanukaFiles, err := secrets.FindEnvOrKanukaFiles(projectPath, []string{}, true)
		if err == nil {
			kanukaFilesCount = len(kanukaFiles)
		}
	}

	return &RevokeResult{
		DisplayName:      revokeCtx.displayName,
		UUIDsRevoked:     revokeCtx.uuidsRevoked,
		FilesToDelete:    revokeCtx.files,
		DryRun:           true,
		AllUsers:         allUsers,
		RemainingUsers:   len(allUsers) - len(revokeCtx.uuidsRevoked),
		KanukaFilesCount: kanukaFilesCount,
	}, nil
}

// executeRevoke performs the actual revocation.
func executeRevoke(revokeCtx *revokeContext, opts RevokeOptions) (*RevokeResult, error) {
	userConfig, err := configs.EnsureUserConfig()
	if err != nil {
		return nil, fmt.Errorf("loading user config: %w", err)
	}

	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		return nil, fmt.Errorf("loading project config: %w", err)
	}
	projectUUID := projectConfig.Project.UUID

	var revokedFiles []string
	var revokeErrors []error

	for _, file := range revokeCtx.files {
		if err := os.Remove(file.Path); err != nil {
			revokeErrors = append(revokeErrors, fmt.Errorf("removing %s: %w", file.Name, err))
		} else {
			revokedFiles = append(revokedFiles, file.Name)
		}
	}

	if len(revokeErrors) > 0 {
		return nil, fmt.Errorf("failed to revoke files: %v", revokeErrors)
	}

	for _, uuid := range revokeCtx.uuidsRevoked {
		projectConfig.RemoveDevice(uuid)
	}

	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		return nil, fmt.Errorf("saving project config: %w", err)
	}

	allUsers, err := secrets.GetAllUsersInProject()
	if err != nil {
		return nil, fmt.Errorf("getting remaining users: %w", err)
	}

	result := &RevokeResult{
		DisplayName:    revokeCtx.displayName,
		RevokedFiles:   revokedFiles,
		UUIDsRevoked:   revokeCtx.uuidsRevoked,
		RemainingUsers: len(allUsers),
		DryRun:         false,
	}

	if len(allUsers) > 0 {
		privateKey, err := loadPrivateKeyForRevoke(opts.PrivateKeyData, projectUUID)
		if err != nil {
			return nil, fmt.Errorf("loading private key for re-encryption: %w", err)
		}

		syncOpts := secrets.SyncOptions{
			ExcludeUsers: revokeCtx.uuidsRevoked,
			Verbose:      opts.Verbose,
			Debug:        opts.Debug,
		}

		syncResult, err := secrets.SyncSecrets(privateKey, syncOpts)
		if err != nil {
			return nil, fmt.Errorf("re-encrypting secrets: %w", err)
		}

		result.SecretsReEncrypted = syncResult.SecretsProcessed
	}

	auditEntry := audit.LogWithUser("revoke")
	auditEntry.TargetUser = revokeCtx.displayName
	if len(revokeCtx.uuidsRevoked) > 0 {
		auditEntry.TargetUUID = revokeCtx.uuidsRevoked[0]
	}
	if opts.DeviceName != "" {
		auditEntry.Device = opts.DeviceName
	}
	audit.Log(auditEntry)

	// Check if user is revoking themselves.
	for _, uuid := range revokeCtx.uuidsRevoked {
		if uuid == userConfig.User.UUID {
			return result, kerrors.ErrSelfRevoke
		}
	}

	return result, nil
}

// loadPrivateKeyForRevoke loads the private key from bytes or disk.
func loadPrivateKeyForRevoke(keyData []byte, projectUUID string) (*rsa.PrivateKey, error) {
	if len(keyData) > 0 {
		return secrets.LoadPrivateKeyFromBytesWithTTYPrompt(keyData)
	}
	privateKeyPath := configs.GetPrivateKeyPath(projectUUID)
	return secrets.LoadPrivateKey(privateKeyPath)
}

// GetDevicesForUser returns devices for a user email (for interactive prompts).
func GetDevicesForUser(userEmail string) ([]configs.DeviceConfig, error) {
	if err := configs.InitProjectSettings(); err != nil {
		return nil, fmt.Errorf("initializing project settings: %w", err)
	}

	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		return nil, fmt.Errorf("loading project config: %w", err)
	}

	devicesMap := projectConfig.GetDevicesByEmail(userEmail)
	var devices []configs.DeviceConfig
	for _, device := range devicesMap {
		devices = append(devices, device)
	}

	return devices, nil
}
