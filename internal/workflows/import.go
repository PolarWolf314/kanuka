package workflows

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/PolarWolf314/kanuka/internal/audit"
	"github.com/PolarWolf314/kanuka/internal/configs"
	kerrors "github.com/PolarWolf314/kanuka/internal/errors"
)

// ImportMode represents the import strategy.
type ImportMode int

const (
	// ImportModeMerge adds new files from archive, keeps existing files.
	ImportModeMerge ImportMode = iota
	// ImportModeReplace deletes existing .kanuka directory and extracts all from archive.
	ImportModeReplace
)

// ImportOptions configures the import workflow.
type ImportOptions struct {
	// ArchivePath is the path to the tar.gz archive.
	ArchivePath string

	// ProjectPath is the path to the project directory.
	// If empty, uses the current working directory.
	ProjectPath string

	// Mode is the import strategy (merge or replace).
	Mode ImportMode

	// DryRun previews the import without making changes.
	DryRun bool
}

// ImportResult contains the outcome of an import operation.
type ImportResult struct {
	// FilesAdded is the count of new files added (merge mode).
	FilesAdded int

	// FilesSkipped is the count of files skipped because they exist (merge mode).
	FilesSkipped int

	// FilesReplaced is the count of files extracted (replace mode).
	FilesReplaced int

	// TotalFiles is the total number of files in the archive.
	TotalFiles int

	// DryRun indicates whether this was a dry-run.
	DryRun bool

	// Mode is the import mode used.
	Mode ImportMode
}

// ImportPreCheckResult contains information from validating the archive.
type ImportPreCheckResult struct {
	// ArchiveFiles is the list of files in the archive.
	ArchiveFiles []string

	// KanukaExists indicates whether a .kanuka directory already exists.
	KanukaExists bool

	// ProjectPath is the resolved project path.
	ProjectPath string
}

// ImportPreCheck validates the archive and checks the project state.
//
// Returns ErrFileNotFound if the archive doesn't exist.
// Returns ErrInvalidFileType if the archive is not a valid gzip file.
// Returns ErrInvalidArchive if the archive structure is invalid.
func ImportPreCheck(ctx context.Context, archivePath string) (*ImportPreCheckResult, error) {
	// Check archive exists.
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", kerrors.ErrFileNotFound, archivePath)
	}

	// Get current working directory as project path.
	projectPath, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting current directory: %w", err)
	}

	// Validate archive structure.
	archiveFiles, err := listArchiveContents(archivePath)
	if err != nil {
		if strings.Contains(err.Error(), "gzip") || strings.Contains(err.Error(), "invalid header") {
			return nil, fmt.Errorf("%w: not a valid gzip archive", kerrors.ErrInvalidFileType)
		}
		return nil, fmt.Errorf("reading archive: %w", err)
	}

	if err := validateArchiveStructure(archiveFiles); err != nil {
		return nil, fmt.Errorf("%w: %v", kerrors.ErrInvalidArchive, err)
	}

	kanukaDir := filepath.Join(projectPath, ".kanuka")
	kanukaExists := false
	if _, err := os.Stat(kanukaDir); err == nil {
		kanukaExists = true
	}

	return &ImportPreCheckResult{
		ArchiveFiles: archiveFiles,
		KanukaExists: kanukaExists,
		ProjectPath:  projectPath,
	}, nil
}

// Import restores secrets from a tar.gz archive.
//
// The archive should contain:
//   - .kanuka/config.toml (project configuration)
//   - .kanuka/public_keys/*.pub (user public keys)
//   - .kanuka/secrets/*.kanuka (encrypted symmetric keys)
//   - *.kanuka files (encrypted secret files)
//
// Returns ErrFileNotFound if the archive doesn't exist.
// Returns ErrInvalidFileType if the archive is not a valid gzip file.
// Returns ErrInvalidArchive if the archive structure is invalid.
func Import(ctx context.Context, opts ImportOptions) (*ImportResult, error) {
	projectPath := opts.ProjectPath
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting current directory: %w", err)
		}
	}

	// Check archive exists.
	if _, err := os.Stat(opts.ArchivePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", kerrors.ErrFileNotFound, opts.ArchivePath)
	}

	// Validate archive structure.
	archiveFiles, err := listArchiveContents(opts.ArchivePath)
	if err != nil {
		if strings.Contains(err.Error(), "gzip") || strings.Contains(err.Error(), "invalid header") {
			return nil, fmt.Errorf("%w: not a valid gzip archive", kerrors.ErrInvalidFileType)
		}
		return nil, fmt.Errorf("reading archive: %w", err)
	}

	if err := validateArchiveStructure(archiveFiles); err != nil {
		return nil, fmt.Errorf("%w: %v", kerrors.ErrInvalidArchive, err)
	}

	// Perform import.
	result, err := performImport(opts.ArchivePath, projectPath, archiveFiles, opts.Mode, opts.DryRun)
	if err != nil {
		return nil, err
	}

	// Log to audit trail (only if not dry-run).
	if !opts.DryRun {
		modeStr := "merge"
		if opts.Mode == ImportModeReplace {
			modeStr = "replace"
		}
		auditEntry := audit.LogWithUser("import")
		auditEntry.Mode = modeStr
		auditEntry.FilesCount = result.TotalFiles
		audit.Log(auditEntry)
	}

	return &ImportResult{
		FilesAdded:    result.FilesAdded,
		FilesSkipped:  result.FilesSkipped,
		FilesReplaced: result.FilesReplaced,
		TotalFiles:    result.TotalFiles,
		DryRun:        opts.DryRun,
		Mode:          opts.Mode,
	}, nil
}

// importResultInternal is an internal struct for performImport.
type importResultInternal struct {
	FilesAdded    int
	FilesSkipped  int
	FilesReplaced int
	TotalFiles    int
}

// listArchiveContents returns a list of all file paths in the archive.
func listArchiveContents(archivePath string) ([]string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("opening archive: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	var files []string

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading tar header: %w", err)
		}
		files = append(files, header.Name)
	}

	return files, nil
}

// validateArchiveStructure checks that the archive contains required files.
func validateArchiveStructure(files []string) error {
	hasConfig := false
	hasContent := false

	for _, f := range files {
		if f == ".kanuka/config.toml" {
			hasConfig = true
		}
		// Check for any content in public_keys, secrets, or .kanuka files.
		if strings.HasPrefix(f, ".kanuka/public_keys/") ||
			strings.HasPrefix(f, ".kanuka/secrets/") ||
			strings.HasSuffix(f, ".kanuka") {
			hasContent = true
		}
	}

	if !hasConfig {
		return fmt.Errorf("archive missing .kanuka/config.toml")
	}

	if !hasContent {
		return fmt.Errorf("archive contains no encrypted content")
	}

	return nil
}

// performImport extracts files from the archive to the project directory.
func performImport(archivePath, projectPath string, archiveFiles []string, mode ImportMode, dryRun bool) (*importResultInternal, error) {
	result := &importResultInternal{
		TotalFiles: len(archiveFiles),
	}

	kanukaDir := filepath.Join(projectPath, ".kanuka")

	// For replace mode, delete existing .kanuka directory first.
	if mode == ImportModeReplace && !dryRun {
		if _, err := os.Stat(kanukaDir); err == nil {
			if err := os.RemoveAll(kanukaDir); err != nil {
				return nil, fmt.Errorf("removing existing .kanuka directory: %w", err)
			}
		}
	}

	// Open archive for extraction.
	file, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("opening archive: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading tar header: %w", err)
		}

		// Skip directories - we'll create them as needed.
		if header.Typeflag == tar.TypeDir {
			continue
		}

		// Validate path to prevent directory traversal attacks.
		// #nosec G305 -- We validate the path below before using it.
		targetPath := filepath.Join(projectPath, header.Name)

		// Ensure the target path is within the project directory.
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(projectPath)+string(os.PathSeparator)) &&
			filepath.Clean(targetPath) != filepath.Clean(projectPath) {
			return nil, fmt.Errorf("invalid file path in archive (path traversal attempt): %s", header.Name)
		}

		// Check if file already exists (for merge mode).
		fileExists := false
		if _, err := os.Stat(targetPath); err == nil {
			fileExists = true
		}

		if mode == ImportModeMerge && fileExists {
			result.FilesSkipped++
			continue
		}

		if dryRun {
			if mode == ImportModeMerge {
				if fileExists {
					result.FilesSkipped++
				} else {
					result.FilesAdded++
				}
			} else {
				result.FilesReplaced++
			}
			continue
		}

		// Create parent directories.
		parentDir := filepath.Dir(targetPath)
		// #nosec G301 -- Directories need to be accessible.
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return nil, fmt.Errorf("creating directory %s: %w", parentDir, err)
		}

		// Extract file.
		if err := extractFile(tarReader, targetPath, header.Mode); err != nil {
			return nil, fmt.Errorf("extracting %s: %w", header.Name, err)
		}

		if mode == ImportModeMerge {
			result.FilesAdded++
		} else {
			result.FilesReplaced++
		}
	}

	// Validate extracted config.toml if not in dry-run mode.
	if !dryRun {
		if err := validateExtractedConfig(projectPath); err != nil {
			_ = os.RemoveAll(kanukaDir)
			return nil, fmt.Errorf("invalid archive: %w", err)
		}

		if err := configs.InitProjectSettings(); err != nil {
			// Non-critical warning, continue anyway.
			_ = err
		}
	}

	return result, nil
}

// validateExtractedConfig validates that the extracted config.toml is not empty and is valid TOML.
func validateExtractedConfig(projectPath string) error {
	configPath := filepath.Join(projectPath, ".kanuka", "config.toml")

	configContent, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading config.toml: %w", err)
	}

	if len(configContent) == 0 {
		return fmt.Errorf("config.toml is empty")
	}

	var decoded map[string]interface{}
	if _, err := toml.Decode(string(configContent), &decoded); err != nil {
		return fmt.Errorf("config.toml is invalid: %w", err)
	}

	return nil
}

// extractFile extracts a single file from the tar reader to the target path.
func extractFile(tr *tar.Reader, targetPath string, mode int64) error {
	// Convert mode safely, defaulting to 0600 for invalid values.
	fileMode := os.FileMode(0600)
	if mode >= 0 && mode <= 0777 {
		fileMode = os.FileMode(mode) // #nosec G115 -- We validate mode is in valid range.
	}

	// Create the file.
	outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fileMode)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer outFile.Close()

	// Copy contents.
	// #nosec G110 -- We trust the archive since it was created by export command.
	if _, err := io.Copy(outFile, tr); err != nil {
		return fmt.Errorf("writing file contents: %w", err)
	}

	return nil
}
