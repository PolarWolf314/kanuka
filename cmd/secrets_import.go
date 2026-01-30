package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	kerrors "github.com/PolarWolf314/kanuka/internal/errors"
	"github.com/PolarWolf314/kanuka/internal/ui"
	"github.com/PolarWolf314/kanuka/internal/workflows"

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
			finalMessage := ui.Error.Sprint("✗") + " Cannot use both --merge and --replace flags." +
				"\n\n" + ui.Info.Sprint("→") + " Use --merge to add new files while keeping existing files," +
				"\n   or use --replace to delete existing files and use only of backup."
			fmt.Print(finalMessage)
			return nil
		}

		spinner, cleanup := startSpinner("Importing secrets...", verbose)
		defer cleanup()

		// Pre-check the archive.
		preCheck, err := workflows.ImportPreCheck(context.Background(), archivePath)
		if err != nil {
			spinner.FinalMSG = formatImportError(err, archivePath)
			if isImportUnexpectedError(err) {
				return err
			}
			return nil
		}

		// Determine import mode.
		var mode workflows.ImportMode
		if importMergeFlag {
			mode = workflows.ImportModeMerge
		} else if importReplaceFlag {
			mode = workflows.ImportModeReplace
		} else if preCheck.KanukaExists && !importDryRunFlag {
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
			mode = workflows.ImportModeMerge
		}

		Logger.Debugf("Import mode: %v, dry-run: %v", mode, importDryRunFlag)

		// Perform import.
		opts := workflows.ImportOptions{
			ArchivePath: archivePath,
			ProjectPath: preCheck.ProjectPath,
			Mode:        mode,
			DryRun:      importDryRunFlag,
		}

		result, err := workflows.Import(context.Background(), opts)
		if err != nil {
			spinner.FinalMSG = formatImportError(err, archivePath)
			return err
		}

		// Build summary message.
		var finalMessage string
		if result.DryRun {
			finalMessage = ui.Info.Sprint("Dry run") + " - no changes made" +
				"\n\n"
		} else {
			finalMessage = ui.Success.Sprint("✓") + " Imported secrets from " + ui.Path.Sprint(archivePath) + "\n\n"
		}

		modeStr := "Merge"
		if result.Mode == workflows.ImportModeReplace {
			modeStr = "Replace"
		}
		finalMessage += fmt.Sprintf("Mode: %s", modeStr) + "\n"
		finalMessage += fmt.Sprintf("Total files in archive: %d", result.TotalFiles)

		if result.Mode == workflows.ImportModeMerge {
			finalMessage += fmt.Sprintf("  Added: %d", result.FilesAdded) + "\n"
			finalMessage += fmt.Sprintf("  Skipped (already exist): %d", result.FilesSkipped) + "\n"
		} else {
			finalMessage += fmt.Sprintf("  Extracted: %d", result.FilesReplaced) + "\n"
		}

		if !result.DryRun {
			finalMessage += "\n" + ui.Info.Sprint("Note:") + " You may need to run " + ui.Code.Sprint("kanuka secrets decrypt") + " to decrypt secrets."
		}

		spinner.FinalMSG = finalMessage
		return nil
	},
}

// formatImportError formats workflow errors into user-friendly messages.
func formatImportError(err error, archivePath string) string {
	switch {
	case errors.Is(err, kerrors.ErrFileNotFound):
		return ui.Error.Sprint("✗") + " Archive file not found: " + ui.Path.Sprint(archivePath)

	case errors.Is(err, kerrors.ErrInvalidFileType):
		return ui.Error.Sprint("✗") + " Invalid archive file: " + ui.Path.Sprint(archivePath) +
			"\n\n" + ui.Info.Sprint("→") + " The file is not a valid gzip archive. Ensure it was created with:" +
			"\n   " + ui.Code.Sprint("kanuka secrets export")

	case errors.Is(err, kerrors.ErrInvalidArchive):
		return ui.Error.Sprint("✗") + " Invalid archive structure" +
			"\n" + ui.Error.Sprint("Error: ") + err.Error()

	default:
		return ui.Error.Sprint("✗") + " Failed to import secrets" +
			"\n" + ui.Error.Sprint("Error: ") + err.Error()
	}
}

// isImportUnexpectedError returns true if the error is unexpected and should cause a non-zero exit.
func isImportUnexpectedError(err error) bool {
	expectedErrors := []error{
		kerrors.ErrFileNotFound,
		kerrors.ErrInvalidFileType,
		kerrors.ErrInvalidArchive,
	}

	for _, expected := range expectedErrors {
		if errors.Is(err, expected) {
			return false
		}
	}
	return true
}

// promptForImportMode asks the user how to handle existing .kanuka directory.
func promptForImportMode() (workflows.ImportMode, bool) {
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
		return workflows.ImportModeMerge, true
	case "r", "replace":
		return workflows.ImportModeReplace, true
	default:
		return 0, false
	}
}
