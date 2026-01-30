package workflows

import (
	"context"
	"fmt"

	"github.com/PolarWolf314/kanuka/internal/audit"
	"github.com/PolarWolf314/kanuka/internal/configs"
	kerrors "github.com/PolarWolf314/kanuka/internal/errors"
	"github.com/PolarWolf314/kanuka/internal/secrets"
)

// SyncOptions configures the sync workflow.
type SyncOptions struct {
	// DryRun previews sync without making changes.
	DryRun bool

	// PrivateKeyData contains the private key bytes when reading from stdin.
	// If nil, the private key is loaded from disk.
	PrivateKeyData []byte
}

// SyncResult contains the outcome of a sync operation.
type SyncResult struct {
	// SecretsProcessed is the number of secret files re-encrypted.
	SecretsProcessed int

	// UsersProcessed is the number of users who received the new key.
	UsersProcessed int

	// UsersExcluded is the number of users excluded from the new key.
	UsersExcluded int

	// DryRun indicates whether this was a dry-run.
	DryRun bool
}

// Sync re-encrypts all secrets with a new symmetric key.
//
// This is useful for:
//   - Periodic security key rotation
//   - After adding new team members
//   - If you suspect a key may have been compromised
//
// All users with access will receive the new symmetric key, encrypted
// with their public key.
//
// Returns ErrProjectNotInitialized if the project has no .kanuka directory.
// Returns ErrPrivateKeyNotFound if the private key cannot be loaded.
// Returns ErrKeyDecryptFailed if the symmetric key cannot be decrypted.
func Sync(ctx context.Context, opts SyncOptions) (*SyncResult, error) {
	if err := configs.InitProjectSettings(); err != nil {
		return nil, fmt.Errorf("initializing project settings: %w", err)
	}

	projectPath := configs.ProjectKanukaSettings.ProjectPath
	if projectPath == "" {
		return nil, kerrors.ErrProjectNotInitialized
	}

	// Load project config for project UUID.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		return nil, fmt.Errorf("loading project config: %w", err)
	}
	projectUUID := projectConfig.Project.UUID

	// Load private key.
	privateKey, err := loadPrivateKey(opts.PrivateKeyData, projectUUID)
	if err != nil {
		return nil, err
	}

	// Build sync options.
	syncOpts := secrets.SyncOptions{
		DryRun:  opts.DryRun,
		Verbose: false, // Logging handled at cmd layer.
		Debug:   false,
	}

	// Call sync function.
	result, err := secrets.SyncSecrets(privateKey, syncOpts)
	if err != nil {
		return nil, fmt.Errorf("syncing secrets: %w", err)
	}

	// Log to audit trail (only if not dry-run and files were processed).
	if !opts.DryRun && result.SecretsProcessed > 0 {
		auditEntry := audit.LogWithUser("sync")
		auditEntry.UsersCount = result.UsersProcessed
		auditEntry.FilesCount = result.SecretsProcessed
		audit.Log(auditEntry)
	}

	return &SyncResult{
		SecretsProcessed: result.SecretsProcessed,
		UsersProcessed:   result.UsersProcessed,
		UsersExcluded:    result.UsersExcluded,
		DryRun:           opts.DryRun,
	}, nil
}
