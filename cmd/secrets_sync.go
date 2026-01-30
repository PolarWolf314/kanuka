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

var syncDryRun bool

func init() {
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "preview sync without making changes")
}

func resetSyncCommandState() {
	syncDryRun = false
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Re-encrypt all secrets with a new symmetric key",
	Long: `Re-encrypts all secret files with a newly generated symmetric key.

This command is useful for:
  - Periodic security key rotation
  - After adding new team members
  - If you suspect a key may have been compromised

All users with access will receive the new symmetric key, encrypted
with their public key. The old symmetric key will no longer work.

Use --dry-run to preview what would happen without making changes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		Logger.Infof("Starting sync command")
		spinner, cleanup := startSpinner("Syncing secrets...", verbose)
		defer cleanup()

		opts := workflows.SyncOptions{
			DryRun: syncDryRun,
		}

		result, err := workflows.Sync(context.Background(), opts)
		if err != nil {
			spinner.FinalMSG = formatSyncError(err)
			if isSyncUnexpectedError(err) {
				return err
			}
			return nil
		}

		// Display results.
		if result.DryRun {
			spinner.Stop()
			printSyncDryRun(result)
			spinner.FinalMSG = ""
			return nil
		}

		// Handle case where no secrets needed processing.
		if result.SecretsProcessed == 0 {
			spinner.FinalMSG = ui.Success.Sprint("✓") + " No encrypted files found. Nothing to sync."
			return nil
		}

		finalMessage := ui.Success.Sprint("✓") + " Secrets synced successfully" +
			fmt.Sprintf("\n  Re-encrypted %d secret file(s) for %d user(s).", result.SecretsProcessed, result.UsersProcessed) +
			"\n  New encryption key generated and distributed to all users."
		spinner.FinalMSG = finalMessage
		return nil
	},
}

// formatSyncError formats workflow errors into user-friendly messages.
func formatSyncError(err error) string {
	switch {
	case errors.Is(err, kerrors.ErrProjectNotInitialized):
		return ui.Error.Sprint("✗") + " Kanuka has not been initialized" +
			"\n" + ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " first"

	case errors.Is(err, kerrors.ErrPrivateKeyNotFound):
		return ui.Error.Sprint("✗") + " Failed to load your private key. Are you sure you have access?" +
			"\n" + ui.Error.Sprint("Error: ") + err.Error()

	case errors.Is(err, kerrors.ErrKeyDecryptFailed):
		return ui.Error.Sprint("✗") + " Failed to decrypt the symmetric key" +
			"\n" + ui.Error.Sprint("Error: ") + err.Error()

	default:
		return ui.Error.Sprint("✗") + " Failed to sync secrets" +
			"\n" + ui.Error.Sprint("Error: ") + err.Error()
	}
}

// isSyncUnexpectedError returns true if the error is unexpected and should cause a non-zero exit.
func isSyncUnexpectedError(err error) bool {
	expectedErrors := []error{
		kerrors.ErrProjectNotInitialized,
		kerrors.ErrPrivateKeyNotFound,
		kerrors.ErrKeyDecryptFailed,
	}

	for _, expected := range expectedErrors {
		if errors.Is(err, expected) {
			return false
		}
	}
	return true
}

// printSyncDryRun displays what would happen during a sync operation.
func printSyncDryRun(result *workflows.SyncResult) {
	fmt.Println()
	fmt.Println(ui.Warning.Sprint("[dry-run]") + " Would sync secrets:")
	fmt.Println()

	if result.SecretsProcessed == 0 {
		fmt.Println("  No encrypted files found. Nothing to sync.")
		fmt.Println()
		fmt.Println(ui.Info.Sprint("No changes needed."))
		return
	}

	fmt.Printf("  - Decrypt %d secret file(s)\n", result.SecretsProcessed)
	fmt.Println("  - Generate new encryption key")
	fmt.Printf("  - Re-encrypt for %d user(s)\n", result.UsersProcessed)

	if result.UsersExcluded > 0 {
		fmt.Printf("  - Exclude %d user(s) from new key\n", result.UsersExcluded)
	}

	fmt.Printf("  - Re-encrypt %d secret file(s)\n", result.SecretsProcessed)
	fmt.Println()
	fmt.Println(ui.Info.Sprint("No changes made.") + " Run without --dry-run to execute.")
}
