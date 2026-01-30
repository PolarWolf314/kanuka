package cmd

import (
	"context"
	"errors"
	"fmt"

	kerrors "github.com/PolarWolf314/kanuka/internal/errors"
	"github.com/PolarWolf314/kanuka/internal/ui"
	"github.com/PolarWolf314/kanuka/internal/workflows"

	"github.com/spf13/cobra"
)

var exportOutputPath string

func init() {
	exportCmd.Flags().StringVarP(&exportOutputPath, "output", "o", "", "output path for the archive (default: kanuka-secrets-YYYY-MM-DD.tar.gz)")
}

// resetExportCommandState resets the export command's global state for testing.
func resetExportCommandState() {
	exportOutputPath = ""
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
	RunE: runExport,
}

func runExport(cmd *cobra.Command, args []string) error {
	Logger.Infof("Starting export command")
	spinner, cleanup := startSpinner("Exporting secrets...", verbose)
	defer cleanup()

	opts := workflows.ExportOptions{
		OutputPath: exportOutputPath,
	}

	result, err := workflows.Export(context.Background(), opts)
	if err != nil {
		spinner.FinalMSG = formatExportError(err)
		if isExportUnexpectedError(err) {
			return err
		}
		return nil
	}

	Logger.Infof("Archive created successfully at %s", result.OutputPath)
	spinner.FinalMSG = formatExportSuccess(result)
	return nil
}

// formatExportError formats an export error for display to the user.
func formatExportError(err error) string {
	switch {
	case errors.Is(err, kerrors.ErrProjectNotInitialized):
		return ui.Error.Sprint("✗") + " Kanuka has not been initialized" +
			"\n" + ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " instead"

	case errors.Is(err, kerrors.ErrInvalidProjectConfig):
		return ui.Error.Sprint("✗") + " Failed to load project configuration." +
			"\n\n" + ui.Info.Sprint("→") + " " + ui.Code.Sprint(err.Error()) +
			"\n\n" + ui.Info.Sprint("→") + " To fix this issue:" +
			"\n   1. Restore from git: " + ui.Code.Sprint("git checkout .kanuka/config.toml") +
			"\n   2. Or contact your project administrator for assistance"

	case errors.Is(err, kerrors.ErrNoFilesFound):
		return ui.Warning.Sprint("⚠") + " No files found to export."

	default:
		return ui.Error.Sprint("✗") + " Export failed: " + err.Error()
	}
}

// isExportUnexpectedError returns true if the error is unexpected and should cause a non-zero exit.
func isExportUnexpectedError(err error) bool {
	switch {
	case errors.Is(err, kerrors.ErrProjectNotInitialized),
		errors.Is(err, kerrors.ErrInvalidProjectConfig),
		errors.Is(err, kerrors.ErrNoFilesFound):
		return false
	default:
		return true
	}
}

// formatExportSuccess formats a successful export result for display to the user.
func formatExportSuccess(result *workflows.ExportResult) string {
	message := ui.Success.Sprint("✓") + " Exported secrets to " + ui.Path.Sprint(result.OutputPath) +
		"\n\nArchive contents:\n"

	if result.ConfigIncluded {
		message += "  .kanuka/config.toml"
	}
	if result.PublicKeyCount > 0 {
		message += fmt.Sprintf("\n  .kanuka/public_keys/ (%d file(s))", result.PublicKeyCount)
	}
	if result.UserKeyCount > 0 {
		message += fmt.Sprintf("\n  .kanuka/secrets/ (%d user key(s))", result.UserKeyCount)
	}
	if result.SecretFileCount > 0 {
		message += fmt.Sprintf("\n  %d encrypted secret file(s)", result.SecretFileCount)
	}

	message += "\n\n" + ui.Info.Sprint("Note:") + " This archive contains encrypted data only." +
		"\n      Private keys are NOT included."

	return message
}
