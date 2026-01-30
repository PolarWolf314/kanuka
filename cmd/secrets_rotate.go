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

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
)

var (
	rotateForce bool
)

func init() {
	rotateCmd.Flags().BoolVar(&rotateForce, "force", false, "skip confirmation prompt")
}

// resetRotateCommandState resets the rotate command's global state for testing.
func resetRotateCommandState() {
	rotateForce = false
}

// confirmRotate prompts the user to confirm the keypair rotation.
// Returns true if the user confirms, false otherwise.
func confirmRotate(s *spinner.Spinner) bool {
	s.Stop()

	fmt.Printf("\n%s This will generate a new keypair and replace your current one.\n", ui.Warning.Sprint("Warning:"))
	fmt.Println("  Your old private key will no longer work for this project.")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Do you want to continue? [y/N]: ")
	response, err := reader.ReadString('\n')
	if err != nil {
		Logger.Errorf("Failed to read response: %v", err)
		s.Restart()
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))

	s.Restart()
	return response == "y" || response == "yes"
}

var rotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Rotate your keypair for this project",
	Long: `Generates a new keypair and replaces your current one.

This command is useful when you want to rotate your keys for security purposes,
such as when your private key may have been compromised.

The command will:
  1. Generate a new RSA keypair
  2. Decrypt the symmetric key with your old private key
  3. Re-encrypt the symmetric key with your new public key
  4. Update your public key in the project
  5. Save the new private key to your key directory

After running this command:
  - Your old private key will no longer work for this project
  - Other users do NOT need to take any action
  - You should commit the updated .kanuka/public_keys/<uuid>.pub file

Examples:
  # Rotate your keypair (with confirmation prompt)
  kanuka secrets rotate

  # Rotate without confirmation prompt
  kanuka secrets rotate --force`,
	RunE: func(cmd *cobra.Command, args []string) error {
		Logger.Infof("Starting rotate command")
		spinner, cleanup := startSpinner("Rotating keypair...", verbose)
		defer cleanup()

		// Confirmation prompt (unless --force) - must happen before workflow.
		if !rotateForce {
			if !confirmRotate(spinner) {
				spinner.FinalMSG = ui.Warning.Sprint("⚠") + " Keypair rotation cancelled."
				return nil
			}
		}

		opts := workflows.RotateOptions{
			Force: rotateForce,
		}

		result, err := workflows.Rotate(context.Background(), opts)
		if err != nil {
			spinner.FinalMSG = formatRotateError(err)
			if isUnexpectedError(err) {
				return err
			}
			return nil
		}

		finalMessage := ui.Success.Sprint("✓") + " Keypair rotated successfully\n\n" +
			"Your new public key has been added to the project.\n" +
			"Other users do not need to take any action.\n\n" +
			ui.Info.Sprint("→") + " Commit the updated " + ui.Path.Sprint(".kanuka/public_keys/"+result.UserUUID+".pub") + " file"
		spinner.FinalMSG = finalMessage
		return nil
	},
}

// formatRotateError formats workflow errors into user-friendly messages.
func formatRotateError(err error) string {
	switch {
	case errors.Is(err, kerrors.ErrProjectNotInitialized):
		return ui.Error.Sprint("✗") + " Kanuka has not been initialized" +
			"\n" + ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " instead"

	case errors.Is(err, kerrors.ErrNoAccess):
		return ui.Error.Sprint("✗") + " You don't have access to this project\n" +
			ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets create") + " and ask someone to register you"

	case errors.Is(err, kerrors.ErrPrivateKeyNotFound):
		return ui.Error.Sprint("✗") + " Couldn't load your private key\n" +
			ui.Error.Sprint("Error: ") + err.Error()

	case errors.Is(err, kerrors.ErrKeyDecryptFailed):
		return ui.Error.Sprint("✗") + " Failed to decrypt your Kanuka key\n" +
			ui.Error.Sprint("Error: ") + err.Error()

	default:
		return ui.Error.Sprint("✗") + " Failed to rotate keypair\n" +
			ui.Error.Sprint("Error: ") + err.Error()
	}
}

// isUnexpectedError returns true if the error is unexpected and should cause a non-zero exit.
func isUnexpectedError(err error) bool {
	// Expected errors (user-facing issues) that should exit cleanly.
	expectedErrors := []error{
		kerrors.ErrProjectNotInitialized,
		kerrors.ErrNoAccess,
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
