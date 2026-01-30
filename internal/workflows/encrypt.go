package workflows

import (
	"context"
	"crypto/rsa"
	"fmt"

	"github.com/PolarWolf314/kanuka/internal/audit"
	"github.com/PolarWolf314/kanuka/internal/configs"
	kerrors "github.com/PolarWolf314/kanuka/internal/errors"
	"github.com/PolarWolf314/kanuka/internal/secrets"
)

// EncryptOptions configures the encrypt workflow.
type EncryptOptions struct {
	// FilePatterns specifies files to encrypt. If empty, all .env files are encrypted.
	FilePatterns []string

	// DryRun previews which files would be encrypted without making changes.
	DryRun bool

	// PrivateKeyData contains the private key bytes when reading from stdin.
	// If nil, the private key is loaded from disk.
	PrivateKeyData []byte
}

// EncryptResult contains the outcome of an encrypt operation.
type EncryptResult struct {
	// EncryptedFiles lists the .kanuka files that were created.
	EncryptedFiles []string

	// SourceFiles lists the .env files that were encrypted.
	SourceFiles []string

	// ProjectPath is the root path of the project.
	ProjectPath string

	// DryRun indicates whether this was a dry-run (no files modified).
	DryRun bool
}

// Encrypt encrypts environment files using the project's symmetric key.
//
// It loads the user's encrypted symmetric key from the project, decrypts it
// using the user's private key, then encrypts each .env file with NaCl
// secretbox. The encrypted files are written alongside the originals with
// a .kanuka extension.
//
// Returns ErrProjectNotInitialized if the project has no .kanuka directory.
// Returns ErrNoAccess if the user doesn't have a key file for this project.
// Returns ErrKeyDecryptFailed if the private key cannot decrypt the symmetric key.
// Returns ErrNoFilesFound if no .env files match the specified patterns.
func Encrypt(ctx context.Context, opts EncryptOptions) (*EncryptResult, error) {
	if err := configs.InitProjectSettings(); err != nil {
		return nil, fmt.Errorf("initializing project settings: %w", err)
	}

	projectPath := configs.ProjectKanukaSettings.ProjectPath
	if projectPath == "" {
		return nil, kerrors.ErrProjectNotInitialized
	}

	envFiles, err := resolveEnvFiles(opts.FilePatterns, projectPath)
	if err != nil {
		return nil, err
	}

	if len(envFiles) == 0 {
		return nil, kerrors.ErrNoFilesFound
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

	encryptedSymKey, err := secrets.GetProjectKanukaKey(userUUID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", kerrors.ErrNoAccess, err)
	}

	privateKey, err := loadPrivateKey(opts.PrivateKeyData, projectUUID)
	if err != nil {
		return nil, err
	}

	symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", kerrors.ErrKeyDecryptFailed, err)
	}

	result := &EncryptResult{
		SourceFiles: envFiles,
		ProjectPath: projectPath,
		DryRun:      opts.DryRun,
	}

	if opts.DryRun {
		result.EncryptedFiles = make([]string, len(envFiles))
		for i, f := range envFiles {
			result.EncryptedFiles[i] = f + ".kanuka"
		}
		return result, nil
	}

	if err := secrets.EncryptFiles(symKey, envFiles, false); err != nil {
		return nil, fmt.Errorf("%w: %v", kerrors.ErrEncryptFailed, err)
	}

	result.EncryptedFiles = make([]string, len(envFiles))
	for i, f := range envFiles {
		result.EncryptedFiles[i] = f + ".kanuka"
	}

	auditEntry := audit.LogWithUser("encrypt")
	auditEntry.Files = result.EncryptedFiles
	audit.Log(auditEntry)

	return result, nil
}

// resolveEnvFiles finds .env files based on patterns or defaults to all .env files.
func resolveEnvFiles(patterns []string, projectPath string) ([]string, error) {
	if len(patterns) > 0 {
		resolved, err := secrets.ResolveFiles(patterns, projectPath, true)
		if err != nil {
			return nil, fmt.Errorf("resolving file patterns: %w", err)
		}
		return resolved, nil
	}

	found, err := secrets.FindEnvOrKanukaFiles(projectPath, []string{}, false)
	if err != nil {
		return nil, fmt.Errorf("finding environment files: %w", err)
	}
	return found, nil
}

// loadPrivateKey loads the private key from bytes or from disk.
func loadPrivateKey(keyData []byte, projectUUID string) (*rsa.PrivateKey, error) {
	if len(keyData) > 0 {
		key, err := secrets.LoadPrivateKeyFromBytesWithTTYPrompt(keyData)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", kerrors.ErrInvalidPrivateKey, err)
		}
		return key, nil
	}

	privateKeyPath := configs.GetPrivateKeyPath(projectUUID)
	key, err := secrets.LoadPrivateKey(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", kerrors.ErrPrivateKeyNotFound, err)
	}

	return key, nil
}
