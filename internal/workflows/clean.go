package workflows

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/audit"
	"github.com/PolarWolf314/kanuka/internal/configs"
	kerrors "github.com/PolarWolf314/kanuka/internal/errors"
)

// OrphanEntry represents an orphaned .kanuka file with no corresponding public key.
type OrphanEntry struct {
	// UUID is the user UUID from the orphaned file.
	UUID string

	// FilePath is the absolute path to the orphaned file.
	FilePath string

	// RelativePath is the path relative to the project root.
	RelativePath string
}

// CleanOptions configures the clean workflow.
type CleanOptions struct {
	// DryRun previews what would be removed without making changes.
	DryRun bool

	// Force skips the confirmation prompt (handled by caller).
	Force bool
}

// CleanResult contains the outcome of a clean operation.
type CleanResult struct {
	// Orphans is the list of orphaned entries found.
	Orphans []OrphanEntry

	// RemovedCount is the number of files removed (0 if dry-run).
	RemovedCount int

	// DryRun indicates whether this was a dry-run.
	DryRun bool
}

// Clean removes orphaned keys and inconsistent state.
//
// An orphan is a .kanuka file in .kanuka/secrets/ that has no corresponding
// public key in .kanuka/public_keys/. This can happen if:
//   - A public key was manually deleted
//   - A revoke operation was interrupted
//   - Files were corrupted or partially restored
//
// Returns ErrProjectNotInitialized if the project has no .kanuka directory.
func Clean(ctx context.Context, opts CleanOptions) (*CleanResult, error) {
	if err := configs.InitProjectSettings(); err != nil {
		return nil, fmt.Errorf("initializing project settings: %w", err)
	}

	projectPath := configs.ProjectKanukaSettings.ProjectPath
	if projectPath == "" {
		return nil, kerrors.ErrProjectNotInitialized
	}

	// Find orphaned entries.
	orphans, err := findOrphanedEntries(projectPath)
	if err != nil {
		return nil, fmt.Errorf("finding orphaned entries: %w", err)
	}

	result := &CleanResult{
		Orphans: orphans,
		DryRun:  opts.DryRun,
	}

	// If no orphans found or dry-run, return early.
	if len(orphans) == 0 || opts.DryRun {
		return result, nil
	}

	// Remove orphaned files.
	for _, orphan := range orphans {
		if err := os.Remove(orphan.FilePath); err != nil {
			return nil, fmt.Errorf("removing %s: %w", orphan.FilePath, err)
		}
		result.RemovedCount++
	}

	// Log to audit trail.
	auditEntry := audit.LogWithUser("clean")
	auditEntry.RemovedCount = result.RemovedCount
	audit.Log(auditEntry)

	return result, nil
}

// findOrphanedEntries finds .kanuka files in secrets/ that have no corresponding public key.
func findOrphanedEntries(projectPath string) ([]OrphanEntry, error) {
	secretsDir := configs.ProjectKanukaSettings.ProjectSecretsPath
	publicKeysDir := configs.ProjectKanukaSettings.ProjectPublicKeyPath

	var orphans []OrphanEntry

	entries, err := os.ReadDir(secretsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return orphans, nil
		}
		return nil, fmt.Errorf("reading secrets directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".kanuka") {
			continue
		}

		uuid := strings.TrimSuffix(entry.Name(), ".kanuka")
		publicKeyPath := filepath.Join(publicKeysDir, uuid+".pub")

		if !fileExistsCheck(publicKeyPath) {
			orphanPath := filepath.Join(secretsDir, entry.Name())
			relPath, _ := filepath.Rel(projectPath, orphanPath)

			orphans = append(orphans, OrphanEntry{
				UUID:         uuid,
				FilePath:     orphanPath,
				RelativePath: relPath,
			})
		}
	}

	return orphans, nil
}
