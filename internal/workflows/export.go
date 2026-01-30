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
	"time"

	"github.com/BurntSushi/toml"
	"github.com/PolarWolf314/kanuka/internal/audit"
	"github.com/PolarWolf314/kanuka/internal/configs"
	kerrors "github.com/PolarWolf314/kanuka/internal/errors"
	"github.com/PolarWolf314/kanuka/internal/secrets"
	"github.com/PolarWolf314/kanuka/internal/utils"
)

// ExportOptions configures the export workflow.
type ExportOptions struct {
	// OutputPath is the path for the output archive.
	// If empty, defaults to kanuka-secrets-YYYY-MM-DD.tar.gz.
	OutputPath string
}

// ExportResult contains the outcome of an export operation.
type ExportResult struct {
	// ConfigIncluded indicates whether config.toml was included.
	ConfigIncluded bool

	// PublicKeyCount is the number of public keys included.
	PublicKeyCount int

	// UserKeyCount is the number of user .kanuka files included.
	UserKeyCount int

	// SecretFileCount is the number of encrypted secret files included.
	SecretFileCount int

	// TotalFilesCount is the total number of files in the archive.
	TotalFilesCount int

	// OutputPath is the path to the created archive.
	OutputPath string
}

// Export creates a tar.gz archive containing all encrypted secrets for backup.
//
// The archive includes:
//   - .kanuka/config.toml (project configuration)
//   - .kanuka/public_keys/*.pub (user public keys)
//   - .kanuka/secrets/*.kanuka (encrypted symmetric keys for users)
//   - All *.kanuka files in the project (encrypted secret files)
//
// The archive does NOT include:
//   - Private keys (these stay on each user's machine)
//   - Plaintext .env files (only encrypted versions are included)
//
// Returns ErrProjectNotInitialized if the project has no .kanuka directory.
// Returns ErrInvalidProjectConfig if the project config is malformed.
// Returns ErrNoFilesFound if no files are found to export.
func Export(ctx context.Context, opts ExportOptions) (*ExportResult, error) {
	projectPath, err := utils.FindProjectKanukaRoot()
	if err != nil {
		return nil, fmt.Errorf("finding project root: %w", err)
	}

	if projectPath == "" {
		return nil, kerrors.ErrProjectNotInitialized
	}

	configPath := filepath.Join(projectPath, ".kanuka", "config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, kerrors.ErrProjectNotInitialized
	}

	if err := validateExportConfig(configPath); err != nil {
		return nil, fmt.Errorf("%w: %v", kerrors.ErrInvalidProjectConfig, err)
	}

	if err := configs.InitProjectSettings(); err != nil {
		return nil, fmt.Errorf("initializing project settings: %w", err)
	}

	// Determine output path.
	outputPath := opts.OutputPath
	if outputPath == "" {
		outputPath = fmt.Sprintf("kanuka-secrets-%s.tar.gz", time.Now().Format("2006-01-02"))
	}

	// Collect files to archive.
	result, filesToArchive, err := collectFilesToExport(projectPath)
	if err != nil {
		return nil, fmt.Errorf("collecting files for export: %w", err)
	}
	result.OutputPath = outputPath

	if result.TotalFilesCount == 0 {
		return nil, kerrors.ErrNoFilesFound
	}

	// Create the archive.
	if err := createTarGzArchive(outputPath, projectPath, filesToArchive); err != nil {
		return nil, fmt.Errorf("creating archive: %w", err)
	}

	// Log to audit trail.
	auditEntry := audit.LogWithUser("export")
	auditEntry.OutputPath = outputPath
	audit.Log(auditEntry)

	return result, nil
}

// validateExportConfig validates that the config.toml is not empty and is valid TOML.
func validateExportConfig(configPath string) error {
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

// collectFilesToExport gathers all files that should be included in the export archive.
func collectFilesToExport(projectPath string) (*ExportResult, []string, error) {
	result := &ExportResult{}
	var files []string

	kanukaDir := filepath.Join(projectPath, ".kanuka")

	// 1. Include config.toml if it exists.
	configPath := filepath.Join(kanukaDir, "config.toml")
	if _, err := os.Stat(configPath); err == nil {
		files = append(files, configPath)
		result.ConfigIncluded = true
	}

	// 2. Include all public keys.
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	if entries, err := os.ReadDir(publicKeysDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".pub") {
				files = append(files, filepath.Join(publicKeysDir, entry.Name()))
				result.PublicKeyCount++
			}
		}
	}

	// 3. Include all user .kanuka files (encrypted symmetric keys).
	secretsDir := filepath.Join(kanukaDir, "secrets")
	if entries, err := os.ReadDir(secretsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".kanuka") {
				files = append(files, filepath.Join(secretsDir, entry.Name()))
				result.UserKeyCount++
			}
		}
	}

	// 4. Include all encrypted .kanuka secret files in the project.
	secretFiles, err := secrets.FindEnvOrKanukaFiles(projectPath, []string{}, true)
	if err != nil {
		return nil, nil, fmt.Errorf("finding secret files: %w", err)
	}
	files = append(files, secretFiles...)
	result.SecretFileCount = len(secretFiles)

	result.TotalFilesCount = len(files)
	return result, files, nil
}

// createTarGzArchive creates a gzip-compressed tar archive containing the specified files.
func createTarGzArchive(outputPath, projectPath string, files []string) error {
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer outFile.Close()

	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	for _, filePath := range files {
		if err := addFileToTar(tarWriter, projectPath, filePath); err != nil {
			return fmt.Errorf("adding file %s to archive: %w", filePath, err)
		}
	}

	return nil
}

// addFileToTar adds a single file to the tar archive with a path relative to projectPath.
func addFileToTar(tw *tar.Writer, projectPath, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return fmt.Errorf("creating tar header: %w", err)
	}

	relPath, err := filepath.Rel(projectPath, filePath)
	if err != nil {
		return fmt.Errorf("getting relative path: %w", err)
	}
	header.Name = relPath

	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("writing tar header: %w", err)
	}

	if _, err := io.Copy(tw, file); err != nil {
		return fmt.Errorf("writing file contents: %w", err)
	}

	return nil
}
