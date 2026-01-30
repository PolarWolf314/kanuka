package workflows

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/configs"
	kerrors "github.com/PolarWolf314/kanuka/internal/errors"
	"github.com/PolarWolf314/kanuka/internal/secrets"
)

// FileStatus represents the encryption status of a secret file.
type FileStatus string

const (
	// StatusCurrent means the encrypted file is newer than the plaintext.
	StatusCurrent FileStatus = "current"
	// StatusStale means the plaintext was modified after encryption.
	StatusStale FileStatus = "stale"
	// StatusUnencrypted means plaintext exists with no encrypted version.
	StatusUnencrypted FileStatus = "unencrypted"
	// StatusEncryptedOnly means encrypted exists with no plaintext.
	StatusEncryptedOnly FileStatus = "encrypted_only"
)

// FileStatusInfo holds information about a file's encryption status.
type FileStatusInfo struct {
	// Path is the relative path of the file.
	Path string

	// Status is the encryption status of the file.
	Status FileStatus

	// PlaintextMtime is the modification time of the plaintext file (if any).
	PlaintextMtime string

	// EncryptedMtime is the modification time of the encrypted file (if any).
	EncryptedMtime string
}

// StatusSummary holds counts of files by status.
type StatusSummary struct {
	// Current is the count of files that are up to date.
	Current int

	// Stale is the count of files where plaintext was modified after encryption.
	Stale int

	// Unencrypted is the count of files that have no encrypted version.
	Unencrypted int

	// EncryptedOnly is the count of files that only have an encrypted version.
	EncryptedOnly int
}

// StatusOptions configures the status workflow.
type StatusOptions struct {
	// No options currently needed - included for consistency.
}

// StatusResult contains the outcome of a status operation.
type StatusResult struct {
	// ProjectName is the name of the project.
	ProjectName string

	// Files contains the status of each discovered file.
	Files []FileStatusInfo

	// Summary contains counts of files by status.
	Summary StatusSummary
}

// Status checks the encryption status of all secret files in the project.
//
// It discovers all .env and .kanuka files and determines their status:
//   - current: encrypted file is newer than plaintext (up to date)
//   - stale: plaintext modified after encryption (needs re-encryption)
//   - unencrypted: plaintext exists with no encrypted version
//   - encrypted_only: encrypted exists with no plaintext
//
// Returns ErrProjectNotInitialized if the project has no .kanuka directory.
// Returns ErrInvalidProjectConfig if the project config is malformed.
func Status(ctx context.Context, opts StatusOptions) (*StatusResult, error) {
	if err := configs.InitProjectSettings(); err != nil {
		return nil, fmt.Errorf("initializing project settings: %w", err)
	}

	projectPath := configs.ProjectKanukaSettings.ProjectPath
	if projectPath == "" {
		return nil, kerrors.ErrProjectNotInitialized
	}

	// Load project config for project name.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		if strings.Contains(err.Error(), "toml:") {
			return nil, fmt.Errorf("%w: .kanuka/config.toml is not valid TOML", kerrors.ErrInvalidProjectConfig)
		}
		return nil, fmt.Errorf("loading project config: %w", err)
	}

	projectName := projectConfig.Project.Name
	if projectName == "" {
		projectName = configs.ProjectKanukaSettings.ProjectName
	}

	// Discover all files and their statuses.
	files, err := discoverFileStatuses(projectPath)
	if err != nil {
		return nil, fmt.Errorf("discovering file statuses: %w", err)
	}

	// Sort files by path for consistent output.
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return &StatusResult{
		ProjectName: projectName,
		Files:       files,
		Summary:     calculateStatusSummary(files),
	}, nil
}

// discoverFileStatuses finds all .env and .kanuka files and determines their status.
func discoverFileStatuses(projectPath string) ([]FileStatusInfo, error) {
	// Find all plaintext .env files (excluding .kanuka directory).
	envFiles, err := secrets.FindEnvOrKanukaFiles(projectPath, []string{}, false)
	if err != nil {
		return nil, fmt.Errorf("finding env files: %w", err)
	}

	// Find all encrypted .kanuka files.
	kanukaFiles, err := secrets.FindEnvOrKanukaFiles(projectPath, []string{}, true)
	if err != nil {
		return nil, fmt.Errorf("finding kanuka files: %w", err)
	}

	// Build a set of all base paths (without .kanuka extension).
	basePaths := make(map[string]bool)
	for _, f := range envFiles {
		basePaths[f] = true
	}
	for _, f := range kanukaFiles {
		basePath := strings.TrimSuffix(f, ".kanuka")
		basePaths[basePath] = true
	}

	// Determine status for each base path.
	var files []FileStatusInfo
	for basePath := range basePaths {
		status, envMtime, kanukaMtime := determineFileStatus(basePath)

		// Convert to relative path for display.
		relPath, err := filepath.Rel(projectPath, basePath)
		if err != nil {
			relPath = basePath
		}

		files = append(files, FileStatusInfo{
			Path:           relPath,
			Status:         status,
			PlaintextMtime: envMtime,
			EncryptedMtime: kanukaMtime,
		})
	}

	return files, nil
}

// determineFileStatus determines the encryption status of a file.
func determineFileStatus(basePath string) (FileStatus, string, string) {
	kanukaPath := basePath + ".kanuka"

	envInfo, envErr := os.Stat(basePath)
	kanukaInfo, kanukaErr := os.Stat(kanukaPath)

	envExists := envErr == nil
	kanukaExists := kanukaErr == nil

	var envMtime, kanukaMtime string
	if envExists {
		envMtime = envInfo.ModTime().Format("2006-01-02T15:04:05Z07:00")
	}
	if kanukaExists {
		kanukaMtime = kanukaInfo.ModTime().Format("2006-01-02T15:04:05Z07:00")
	}

	switch {
	case envExists && kanukaExists:
		// Both exist - check modification times.
		if kanukaInfo.ModTime().After(envInfo.ModTime()) {
			return StatusCurrent, envMtime, kanukaMtime
		}
		return StatusStale, envMtime, kanukaMtime

	case envExists && !kanukaExists:
		return StatusUnencrypted, envMtime, ""

	case !envExists && kanukaExists:
		return StatusEncryptedOnly, "", kanukaMtime

	default:
		// Neither exists - shouldn't happen.
		return StatusUnencrypted, "", ""
	}
}

// calculateStatusSummary calculates the counts of files by status.
func calculateStatusSummary(files []FileStatusInfo) StatusSummary {
	var summary StatusSummary
	for _, file := range files {
		switch file.Status {
		case StatusCurrent:
			summary.Current++
		case StatusStale:
			summary.Stale++
		case StatusUnencrypted:
			summary.Unencrypted++
		case StatusEncryptedOnly:
			summary.EncryptedOnly++
		}
	}
	return summary
}
