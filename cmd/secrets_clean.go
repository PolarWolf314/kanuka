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
	cleanForce  bool
	cleanDryRun bool
)

func init() {
	cleanCmd.Flags().BoolVar(&cleanForce, "force", false, "skip confirmation prompt")
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "show what would be removed without making changes")
}

func resetCleanCommandState() {
	cleanForce = false
	cleanDryRun = false
}

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove orphaned keys and inconsistent state",
	Long: `Removes orphaned entries detected by 'kanuka secrets access'.

An orphan is a .kanuka file in .kanuka/secrets/ that has no corresponding
public key in .kanuka/public_keys/. This can happen if:
  - A public key was manually deleted
  - A revoke operation was interrupted
  - Files were corrupted or partially restored

Use --dry-run to preview what would be removed.
Use --force to skip the confirmation prompt.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		Logger.Infof("Starting clean command")

		spinner, cleanup := startSpinner("Scanning for orphaned entries...", verbose)
		defer cleanup()

		// First, do a dry-run to find orphans (regardless of user's dry-run flag).
		// This lets us display the orphans and prompt for confirmation.
		previewOpts := workflows.CleanOptions{
			DryRun: true,
			Force:  cleanForce,
		}

		previewResult, err := workflows.Clean(context.Background(), previewOpts)
		if err != nil {
			spinner.FinalMSG = formatCleanError(err)
			if isCleanUnexpectedError(err) {
				return err
			}
			return nil
		}

		if len(previewResult.Orphans) == 0 {
			spinner.FinalMSG = ui.Success.Sprint("✓") + " No orphaned entries found. Nothing to clean."
			return nil
		}

		// Display orphans.
		spinner.Stop()
		if cleanDryRun {
			fmt.Printf("[dry-run] Would remove %d orphaned file(s):\n", len(previewResult.Orphans))
		} else {
			fmt.Printf("Found %d orphaned entry(ies):\n\n", len(previewResult.Orphans))
		}

		printOrphanTable(previewResult.Orphans)

		// Dry run - don't delete anything.
		if cleanDryRun {
			fmt.Println("\nNo changes made.")
			spinner.FinalMSG = ""
			return nil
		}

		// Confirm deletion (if not --force).
		if !cleanForce {
			fmt.Println("\nThis will permanently delete the orphaned files listed above.")
			fmt.Println("These files cannot be recovered.")
			fmt.Println()

			if !confirmCleanAction() {
				fmt.Println("Aborted.")
				spinner.FinalMSG = ""
				return nil
			}
		}

		spinner.Restart()

		// Now actually clean.
		cleanOpts := workflows.CleanOptions{
			DryRun: false,
			Force:  true, // We already confirmed.
		}

		result, err := workflows.Clean(context.Background(), cleanOpts)
		if err != nil {
			spinner.FinalMSG = formatCleanError(err)
			return err
		}

		spinner.FinalMSG = ui.Success.Sprint("✓") + fmt.Sprintf(" Removed %d orphaned file(s)", result.RemovedCount)
		return nil
	},
}

// formatCleanError formats workflow errors into user-friendly messages.
func formatCleanError(err error) string {
	switch {
	case errors.Is(err, kerrors.ErrProjectNotInitialized):
		return ui.Error.Sprint("✗") + " Kanuka has not been initialized.\n" +
			ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " first"

	default:
		return ui.Error.Sprint("✗") + " Failed to clean\n" +
			ui.Error.Sprint("Error: ") + err.Error()
	}
}

// isCleanUnexpectedError returns true if the error is unexpected and should cause a non-zero exit.
func isCleanUnexpectedError(err error) bool {
	expectedErrors := []error{
		kerrors.ErrProjectNotInitialized,
	}

	for _, expected := range expectedErrors {
		if errors.Is(err, expected) {
			return false
		}
	}
	return true
}

// printOrphanTable prints a formatted table of orphaned entries.
func printOrphanTable(orphans []workflows.OrphanEntry) {
	// Calculate column widths.
	uuidWidth := 36 // Standard UUID length.

	fmt.Printf("  %-*s  %s\n", uuidWidth, "UUID", "FILE")

	for _, orphan := range orphans {
		fmt.Printf("  %-*s  %s\n", uuidWidth, orphan.UUID, orphan.RelativePath)
	}
}

// confirmCleanAction prompts the user to confirm the clean operation.
func confirmCleanAction() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Do you want to continue? [y/N]: ")
	response, err := reader.ReadString('\n')
	if err != nil {
		Logger.Errorf("Failed to read response: %v", err)
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
