package cmd

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/secrets"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	exportOutputPath string
)

func init() {
	exportCmd.Flags().StringVarP(&exportOutputPath, "output", "o", "", "output path for the archive (default: kanuka-secrets-YYYY-MM-DD.tar.gz)")
}

// resetExportCommandState resets the export command's global state for testing.
func resetExportCommandState() {
	exportOutputPath = ""
}

// ExportResult contains summary information about the export operation.
type ExportResult struct {
	ConfigIncluded  bool
	PublicKeyCount  int
	UserKeyCount    int
	SecretFileCount int
	TotalFilesCount int
	OutputPath      string
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export encrypted secrets to a backup archive",
	Long: `Creates a tar.gz archive containing all encrypted secrets for backup.

The archive includes:
  - .kanuka/config.toml (project configuration)
  - .kanuka/public_keys/*.pub (user public keys)
  - .kanuka/secrets/*.kanuka (encrypted symmetric keys for users)
  - All *.kanuka files in the project (encrypted secret files)

The archive does NOT include:
  - Private keys (these stay on each user's machine)
  - Plaintext .env files (only encrypted versions are included)

Use -o/--output to specify a custom output path.
Default filename includes today's date: kanuka-secrets-YYYY-MM-DD.tar.gz

Examples:
  # Export to default filename
  kanuka secrets export

  # Export to custom path
  kanuka secrets export -o /backups/project-secrets.tar.gz

  # Export with verbose output
  kanuka secrets export --verbose`,
	RunE: func(cmd *cobra.Command, args []string) error {
		Logger.Infof("Starting export command")
		spinner, cleanup := startSpinner("Exporting secrets...", verbose)
		defer cleanup()

		Logger.Debugf("Initializing project settings")
		if err := configs.InitProjectSettings(); err != nil {
			return Logger.ErrorfAndReturn("failed to init project settings: %v", err)
		}

		projectPath := configs.ProjectKanukaSettings.ProjectPath
		Logger.Debugf("Project path: %s", projectPath)

		if projectPath == "" {
			finalMessage := color.RedString("✗") + " Kanuka has not been initialized\n" +
				color.CyanString("→") + " Run " + color.YellowString("kanuka secrets init") + " instead"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Determine output path.
		outputPath := exportOutputPath
		if outputPath == "" {
			outputPath = fmt.Sprintf("kanuka-secrets-%s.tar.gz", time.Now().Format("2006-01-02"))
		}
		Logger.Debugf("Output path: %s", outputPath)

		// Collect files to archive.
		result, filesToArchive, err := collectFilesToExport(projectPath)
		if err != nil {
			return Logger.ErrorfAndReturn("failed to collect files for export: %v", err)
		}
		result.OutputPath = outputPath

		if result.TotalFilesCount == 0 {
			finalMessage := color.YellowString("⚠") + " No files found to export"
			spinner.FinalMSG = finalMessage
			return nil
		}

		Logger.Infof("Found %d files to export", result.TotalFilesCount)

		// Create the archive.
		if err := createTarGzArchive(outputPath, projectPath, filesToArchive); err != nil {
			return Logger.ErrorfAndReturn("failed to create archive: %v", err)
		}

		Logger.Infof("Archive created successfully at %s", outputPath)

		// Build summary message.
		finalMessage := color.GreenString("✓") + " Exported secrets to " + color.YellowString(outputPath) + "\n\n" +
			"Archive contents:\n"

		if result.ConfigIncluded {
			finalMessage += "  .kanuka/config.toml\n"
		}
		if result.PublicKeyCount > 0 {
			finalMessage += fmt.Sprintf("  .kanuka/public_keys/ (%d file(s))\n", result.PublicKeyCount)
		}
		if result.UserKeyCount > 0 {
			finalMessage += fmt.Sprintf("  .kanuka/secrets/ (%d user key(s))\n", result.UserKeyCount)
		}
		if result.SecretFileCount > 0 {
			finalMessage += fmt.Sprintf("  %d encrypted secret file(s)\n", result.SecretFileCount)
		}

		finalMessage += "\n" + color.CyanString("Note:") + " This archive contains encrypted data only.\n" +
			"      Private keys are NOT included."

		spinner.FinalMSG = finalMessage
		return nil
	},
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
	// These are the actual encrypted .env files (e.g., .env.kanuka).
	secretFiles, err := secrets.FindEnvOrKanukaFiles(projectPath, []string{}, true)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find secret files: %w", err)
	}
	files = append(files, secretFiles...)
	result.SecretFileCount = len(secretFiles)

	result.TotalFilesCount = len(files)
	return result, files, nil
}

// createTarGzArchive creates a gzip-compressed tar archive containing the specified files.
func createTarGzArchive(outputPath, projectPath string, files []string) error {
	// Create the output file.
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Create gzip writer.
	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	// Create tar writer.
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Add each file to the archive.
	for _, filePath := range files {
		if err := addFileToTar(tarWriter, projectPath, filePath); err != nil {
			return fmt.Errorf("failed to add file %s to archive: %w", filePath, err)
		}
	}

	return nil
}

// addFileToTar adds a single file to the tar archive with a path relative to projectPath.
func addFileToTar(tw *tar.Writer, projectPath, filePath string) error {
	// Open the file.
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info.
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Create tar header.
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return fmt.Errorf("failed to create tar header: %w", err)
	}

	// Use relative path from project root.
	relPath, err := filepath.Rel(projectPath, filePath)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}
	header.Name = relPath

	// Write header.
	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write tar header: %w", err)
	}

	// Copy file contents.
	if _, err := io.Copy(tw, file); err != nil {
		return fmt.Errorf("failed to write file contents: %w", err)
	}

	return nil
}
