package workflows

import (
	"context"
	"crypto/rsa"
	"fmt"
	"os"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/audit"
	"github.com/PolarWolf314/kanuka/internal/configs"
	kerrors "github.com/PolarWolf314/kanuka/internal/errors"
	"github.com/PolarWolf314/kanuka/internal/secrets"
)

// DecryptOptions configures the decrypt workflow.
type DecryptOptions struct {
	// FilePatterns specifies files to decrypt. If empty, all .kanuka files are decrypted.
	FilePatterns []string

	// DryRun previews which files would be decrypted without making changes.
	DryRun bool

	// PrivateKeyData contains the private key bytes when reading from stdin.
	// If nil, the private key is loaded from disk.
	PrivateKeyData []byte
}

// DecryptResult contains the outcome of a decrypt operation.
type DecryptResult struct {
	// DecryptedFiles lists the .env files that were created.
	DecryptedFiles []string

	// SourceFiles lists the .kanuka files that were decrypted.
	SourceFiles []string

	// ProjectPath is the root path of the project.
	ProjectPath string

	// DryRun indicates whether this was a dry-run (no files modified).
	DryRun bool

	// ExistingFiles lists files that already exist and would be overwritten.
	ExistingFiles []string
}

// Decrypt decrypts .kanuka files back to .env files.
//
// It loads the user's encrypted symmetric key from the project, decrypts it
// using the user's private key, then decrypts each .kanuka file with NaCl
// secretbox. The decrypted files are written alongside the encrypted files
// with the .kanuka extension removed.
//
// Returns ErrProjectNotInitialized if the project has no .kanuka directory.
// Returns ErrNoAccess if the user doesn't have a key file for this project.
// Returns ErrKeyDecryptFailed if the private key cannot decrypt the symmetric key.
// Returns ErrNoFilesFound if no .kanuka files match the specified patterns.
func Decrypt(ctx context.Context, opts DecryptOptions) (*DecryptResult, error) {
	if err := configs.InitProjectSettings(); err != nil {
		return nil, fmt.Errorf("initializing project settings: %w", err)
	}

	projectPath := configs.ProjectKanukaSettings.ProjectPath
	if projectPath == "" {
		return nil, kerrors.ErrProjectNotInitialized
	}

	kanukaFiles, err := resolveKanukaFiles(opts.FilePatterns, projectPath)
	if err != nil {
		return nil, err
	}

	if len(kanukaFiles) == 0 {
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

	privateKey, err := loadPrivateKeyForDecrypt(opts.PrivateKeyData, projectUUID)
	if err != nil {
		return nil, err
	}

	symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", kerrors.ErrKeyDecryptFailed, err)
	}

	result := &DecryptResult{
		SourceFiles: kanukaFiles,
		ProjectPath: projectPath,
		DryRun:      opts.DryRun,
	}

	result.DecryptedFiles = make([]string, len(kanukaFiles))
	for i, f := range kanukaFiles {
		result.DecryptedFiles[i] = strings.TrimSuffix(f, ".kanuka")
	}

	if opts.DryRun {
		result.ExistingFiles = findExistingFiles(result.DecryptedFiles)
		return result, nil
	}

	if err := secrets.DecryptFiles(symKey, kanukaFiles, false); err != nil {
		return nil, fmt.Errorf("%w: %v", kerrors.ErrDecryptFailed, err)
	}

	auditEntry := audit.LogWithUser("decrypt")
	auditEntry.Files = kanukaFiles
	audit.Log(auditEntry)

	return result, nil
}

// resolveKanukaFiles finds .kanuka files based on patterns or defaults to all .kanuka files.
func resolveKanukaFiles(patterns []string, projectPath string) ([]string, error) {
	if len(patterns) > 0 {
		resolved, err := secrets.ResolveFiles(patterns, projectPath, false)
		if err != nil {
			return nil, fmt.Errorf("resolving file patterns: %w", err)
		}
		return resolved, nil
	}

	found, err := secrets.FindEnvOrKanukaFiles(projectPath, []string{}, true)
	if err != nil {
		return nil, fmt.Errorf("finding encrypted files: %w", err)
	}
	return found, nil
}

// loadPrivateKeyForDecrypt loads the private key from bytes or from disk.
func loadPrivateKeyForDecrypt(keyData []byte, projectUUID string) (*rsa.PrivateKey, error) {
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

// findExistingFiles returns which of the given paths already exist on disk.
func findExistingFiles(paths []string) []string {
	var existing []string
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			existing = append(existing, path)
		}
	}
	return existing
}
