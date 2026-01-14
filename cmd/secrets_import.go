package cmd

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/PolarWolf314/kanuka/internal/audit"
	"github.com/PolarWolf314/kanuka/internal/configs"

	"github.com/PolarWolf314/kanuka/internal/ui"
	"github.com/spf13/cobra"
)

var (
	importMergeFlag   bool
	importReplaceFlag bool
	importDryRunFlag  bool
)

func init() {
	importCmd.Flags().BoolVar(&importMergeFlag, "merge", false, "merge with existing files (add new, keep existing)")
	importCmd.Flags().BoolVar(&importReplaceFlag, "replace", false, "replace existing .kanuka directory with backup")
	importCmd.Flags().BoolVar(&importDryRunFlag, "dry-run", false, "show what would be imported without making changes")
}

// resetImportCommandState resets the import command's global state for testing.
func resetImportCommandState() {
	importMergeFlag = false
	importReplaceFlag = false
	importDryRunFlag = false
}

// ImportMode represents the import strategy.
type ImportMode int

const (
	// MergeMode adds new files from archive, keeps existing files.
	MergeMode ImportMode = iota
	// ReplaceMode deletes existing .kanuka directory and extracts all from archive.
	ReplaceMode
)

// ImportResult contains summary information about the import operation.
type ImportResult struct {
	FilesAdded    int
	FilesSkipped  int
	FilesReplaced int
	TotalFiles    int
}

var importCmd = &cobra.Command{
	Use:   "import <archive>",
	Short: "Import secrets from a backup archive",
	Long: `Restores secrets from a tar.gz archive created by the export command.

Import modes:
  --merge    Add new files from archive, keep existing files
  --replace  Delete existing .kanuka directory, use backup files

If neither --merge nor --replace is specified and a .kanuka directory
already exists, you will be prompted to choose.

The archive should contain:
  - .kanuka/config.toml (project configuration)
  - .kanuka/public_keys/*.pub (user public keys)
  - .kanuka/secrets/*.kanuka (encrypted symmetric keys)
  - *.kanuka files (encrypted secret files)

Examples:
  # Import with interactive prompt (when .kanuka exists)
  kanuka secrets import kanuka-secrets-2024-01-15.tar.gz

  # Merge mode - add new files, keep existing
  kanuka secrets import backup.tar.gz --merge

  # Replace mode - delete existing, use backup
  kanuka secrets import backup.tar.gz --replace

  # Preview what would happen
  kanuka secrets import backup.tar.gz --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		Logger.Infof("Starting import command")
		archivePath := args[0]

		// Validate flags - can't use both merge and replace.
		if importMergeFlag && importReplaceFlag {
			return Logger.ErrorfAndReturn("cannot use both --merge and --replace flags")
		}

		// Check archive exists.
		if _, err := os.Stat(archivePath); os.IsNotExist(err) {
			return Logger.ErrorfAndReturn("archive file not found: %s", archivePath)
		}

		spinner, cleanup := startSpinner("Importing secrets...", verbose)
		defer cleanup()

		// Validate archive structure.
		Logger.Debugf("Validating archive structure")
		archiveFiles, err := listArchiveContents(archivePath)
		if err != nil {
			if strings.Contains(err.Error(), "gzip") || strings.Contains(err.Error(), "invalid header") {
				spinner.Stop()
				finalMessage := ui.Error.Sprint("✗") + " Invalid archive file: " + ui.Path.Sprint(archivePath) + "\n\n" +
					ui.Info.Sprint("→") + " The file is not a valid gzip archive. Ensure it was created with:\n" +
					"   " + ui.Code.Sprint("kanuka secrets export")
				fmt.Println(finalMessage)
				return fmt.Errorf("invalid archive")
			}
			return Logger.ErrorfAndReturn("failed to read archive: %v", err)
		}

		if err := validateArchiveStructure(archiveFiles); err != nil {
			return Logger.ErrorfAndReturn("invalid archive: %v", err)
		}

		// Get current working directory as project path.
		projectPath, err := os.Getwd()
		if err != nil {
			return Logger.ErrorfAndReturn("failed to get current directory: %v", err)
		}

		kanukaDir := filepath.Join(projectPath, ".kanuka")
		kanukaExists := false
		if _, err := os.Stat(kanukaDir); err == nil {
			kanukaExists = true
		}

		// Determine import mode.
		var mode ImportMode
		if importMergeFlag {
			mode = MergeMode
		} else if importReplaceFlag {
			mode = ReplaceMode
		} else if kanukaExists && !importDryRunFlag {
			// Interactive prompt needed - stop spinner first.
			spinner.Stop()
			var ok bool
			mode, ok = promptForImportMode()
			if !ok {
				fmt.Println(ui.Warning.Sprint("⚠") + " Import cancelled")
				return nil
			}
			// Restart spinner for the import operation.
			spinner, cleanup = startSpinner("Importing secrets...", verbose)
			defer cleanup()
		} else {
			// No existing .kanuka directory or dry-run, default to merge.
			mode = MergeMode
		}

		Logger.Debugf("Import mode: %v, dry-run: %v", mode, importDryRunFlag)

		// Perform import.
		result, err := performImport(archivePath, projectPath, archiveFiles, mode, importDryRunFlag)
		if err != nil {
			return Logger.ErrorfAndReturn("failed to import: %v", err)
		}

		// Build summary message.
		var finalMessage string
		if importDryRunFlag {
			finalMessage = ui.Info.Sprint("Dry run") + " - no changes made\n\n"
		} else {
			// Log to audit trail.
			modeStr := "merge"
			if mode == ReplaceMode {
				modeStr = "replace"
			}
			auditEntry := audit.LogWithUser("import")
			auditEntry.Mode = modeStr
			auditEntry.FilesCount = result.TotalFiles
			audit.Log(auditEntry)

			finalMessage = ui.Success.Sprint("✓") + " Imported secrets from " + ui.Path.Sprint(archivePath) + "\n\n"
		}

		modeStr := "Merge"
		if mode == ReplaceMode {
			modeStr = "Replace"
		}
		finalMessage += fmt.Sprintf("Mode: %s\n", modeStr)
		finalMessage += fmt.Sprintf("Total files in archive: %d\n", result.TotalFiles)

		if mode == MergeMode {
			finalMessage += fmt.Sprintf("  Added: %d\n", result.FilesAdded)
			finalMessage += fmt.Sprintf("  Skipped (already exist): %d\n", result.FilesSkipped)
		} else {
			finalMessage += fmt.Sprintf("  Extracted: %d\n", result.FilesReplaced)
		}

		if !importDryRunFlag {
			finalMessage += "\n" + ui.Info.Sprint("Note:") + " You may need to run " + ui.Code.Sprint("kanuka secrets decrypt") + " to decrypt secrets."
		}

		spinner.FinalMSG = finalMessage
		return nil
	},
}

// listArchiveContents returns a list of all file paths in the archive.
func listArchiveContents(archivePath string) ([]string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
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
			return nil, fmt.Errorf("failed to read tar header: %w", err)
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
		return fmt.Errorf("archive contains no encrypted content (public_keys, secrets, or .kanuka files)")
	}

	return nil
}

// validateExtractedConfig validates that the extracted config.toml is not empty and is valid TOML.
func validateExtractedConfig(projectPath string) error {
	configPath := filepath.Join(projectPath, ".kanuka", "config.toml")

	configContent, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config.toml: %w", err)
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

// promptForImportMode asks the user how to handle existing .kanuka directory.
func promptForImportMode() (ImportMode, bool) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Found existing .kanuka directory. How do you want to proceed?")
	fmt.Println("  [m] Merge - Add new files, keep existing")
	fmt.Println("  [r] Replace - Delete existing, use backup")
	fmt.Println("  [c] Cancel")
	fmt.Print("Choice: ")

	response, err := reader.ReadString('\n')
	if err != nil {
		return 0, false
	}
	response = strings.TrimSpace(strings.ToLower(response))

	switch response {
	case "m", "merge":
		return MergeMode, true
	case "r", "replace":
		return ReplaceMode, true
	default:
		return 0, false
	}
}

// performImport extracts files from the archive to the project directory.
func performImport(archivePath, projectPath string, archiveFiles []string, mode ImportMode, dryRun bool) (*ImportResult, error) {
	result := &ImportResult{
		TotalFiles: len(archiveFiles),
	}

	kanukaDir := filepath.Join(projectPath, ".kanuka")

	// For replace mode, delete existing .kanuka directory first.
	if mode == ReplaceMode && !dryRun {
		if _, err := os.Stat(kanukaDir); err == nil {
			if err := os.RemoveAll(kanukaDir); err != nil {
				return nil, fmt.Errorf("failed to remove existing .kanuka directory: %w", err)
			}
		}
	}

	// Open archive for extraction.
	file, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar header: %w", err)
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

		if mode == MergeMode && fileExists {
			result.FilesSkipped++
			Logger.Debugf("Skipping existing file: %s", header.Name)
			continue
		}

		if dryRun {
			if mode == MergeMode {
				if fileExists {
					result.FilesSkipped++
				} else {
					result.FilesAdded++
				}
			} else {
				result.FilesReplaced++
			}
			Logger.Debugf("Would extract: %s", header.Name)
			continue
		}

		// Create parent directories.
		parentDir := filepath.Dir(targetPath)
		// #nosec G301 -- Directories need to be accessible.
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", parentDir, err)
		}

		// Extract file.
		if err := extractFile(tarReader, targetPath, header.Mode); err != nil {
			return nil, fmt.Errorf("failed to extract %s: %w", header.Name, err)
		}

		if mode == MergeMode {
			result.FilesAdded++
		} else {
			result.FilesReplaced++
		}
		Logger.Debugf("Extracted: %s", header.Name)
	}

	// Validate extracted config.toml if not in dry-run mode.
	if !dryRun {
		if err := validateExtractedConfig(projectPath); err != nil {
			os.RemoveAll(kanukaDir)
			return nil, fmt.Errorf("invalid archive: %w\n\n"+
				"The archive contains an invalid .kanuka/config.toml file.\n"+
				"Ensure your backup was created with 'kanuka secrets export'\n\n"+
				"To fix this issue:\n"+
				"  1. Restore from a good backup\n"+
				"  2. Or re-export from a working project: kanuka secrets export", err)
		}

		if err := configs.InitProjectSettings(); err != nil {
			Logger.Debugf("Warning: failed to reinitialize project settings: %v", err)
		}
	}

	return result, nil
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
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	// Copy contents.
	// #nosec G110 -- We trust the archive since it was created by export command.
	if _, err := io.Copy(outFile, tr); err != nil {
		return fmt.Errorf("failed to write file contents: %w", err)
	}

	return nil
}
