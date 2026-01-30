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
	force         bool
	createEmail   string
	createDevName string
)

func init() {
	createCmd.Flags().BoolVarP(&force, "force", "f", false, "force key creation")
	createCmd.Flags().StringVarP(&createEmail, "email", "e", "", "your email address for identification")
	createCmd.Flags().StringVar(&createDevName, "device-name", "", "custom device name (auto-generated from hostname if not specified)")
}

// resetCreateCommandState resets the create command's global state for testing.
func resetCreateCommandState() {
	force = false
	createEmail = ""
	createDevName = ""
}

// promptForEmail prompts the user for their email address.
func promptForEmail(reader *bufio.Reader) (string, error) {
	fmt.Print("Enter your email: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read email: %w", err)
	}
	return strings.TrimSpace(email), nil
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates and adds your public key, and gives instructions on how to gain access",
	Long: `Creates a new RSA key pair for accessing the project's encrypted secrets.

This command generates a unique cryptographic identity for you on this device,
identified by your email address. Each device you use gets its own key pair.

The command will:
  1. Generate an RSA key pair (stored locally in ~/.local/share/kanuka/keys/)
  2. Copy your public key to the project's .kanuka/public_keys/ directory
  3. Register your device in the project configuration

After running this command, you need to:
  1. Commit the new .kanuka/public_keys/<uuid>.pub file
  2. Ask someone with access to run: kanuka secrets register --user <your-email>

Examples:
  # Create keys with email prompt
  kanuka secrets create

  # Create keys with email specified
  kanuka secrets create --email alice@example.com

  # Create keys with custom device name
  kanuka secrets create --email alice@example.com --device-name macbook-pro

  # Force recreate keys (overwrites existing)
  kanuka secrets create --force`,
	RunE: func(cmd *cobra.Command, args []string) error {
		Logger.Infof("Starting create command")
		spinner, cleanup := startSpinner("Creating Kānuka file...", verbose)
		defer cleanup()

		// Pre-check to determine if we need to prompt for email.
		preCheck, err := workflows.CreatePreCheck(context.Background())
		if err != nil {
			spinner.FinalMSG = formatCreateError(err, "")
			if isCreateUnexpectedError(err) {
				return err
			}
			return nil
		}

		// Handle email: use flag, existing config, or prompt.
		userEmail := createEmail
		if userEmail == "" && preCheck.ExistingEmail != "" {
			userEmail = preCheck.ExistingEmail
		}

		// If still no email, prompt for it.
		if userEmail == "" && preCheck.NeedsEmail {
			spinner.Stop()
			reader := bufio.NewReader(os.Stdin)
			promptedEmail, promptErr := promptForEmail(reader)
			if promptErr != nil {
				return Logger.ErrorfAndReturn("Failed to read email: %v", promptErr)
			}
			userEmail = promptedEmail
			spinner.Restart()
		}

		// Warn if using force flag.
		if force {
			Logger.Infof("Force flag set, will override existing keys if present")
			spinner.Stop()
			Logger.WarnfUser("Using --force flag will overwrite existing keys - ensure you have backups")
			spinner.Restart()
		}

		opts := workflows.CreateOptions{
			Email:      userEmail,
			DeviceName: createDevName,
			Force:      force,
		}

		result, err := workflows.Create(context.Background(), opts)
		if err != nil {
			spinner.FinalMSG = formatCreateError(err, userEmail)
			if isCreateUnexpectedError(err) {
				return err
			}
			return nil
		}

		deletedMessage := ""
		if result.KanukaKeyDeleted {
			deletedMessage = "    deleted: " + ui.Error.Sprint(result.DeletedKanukaKeyPath) + "\n"
		}

		Logger.Infof("Create command completed successfully for user: %s (%s)", result.Email, result.UserUUID)

		finalMessage := ui.Success.Sprint("✓") + " Keys created for " + ui.Highlight.Sprint(result.Email) + " (device: " + ui.Highlight.Sprint(result.DeviceName) + ")" +
			"\n    created: " + ui.Path.Sprint(result.PublicKeyPath) + "\n" + deletedMessage +
			ui.Info.Sprint("To gain access to secrets in this project:") +
			"\n  1. Commit your " + ui.Path.Sprint(".kanuka/public_keys/"+result.UserUUID+".pub") + " file to your version control system" +
			"\n  2. Ask someone with permissions to grant you access with:" +
			"\n     " + ui.Code.Sprint("kanuka secrets register --user "+result.Email)

		spinner.FinalMSG = finalMessage
		return nil
	},
}

// formatCreateError formats workflow errors into user-friendly messages.
func formatCreateError(err error, email string) string {
	switch {
	case errors.Is(err, kerrors.ErrProjectNotInitialized):
		return ui.Error.Sprint("✗") + " Kānuka has not been initialized" +
			"\n" + ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " first to create a project"

	case errors.Is(err, kerrors.ErrInvalidProjectConfig):
		return ui.Error.Sprint("✗") + " Failed to load project config: .kanuka/config.toml is not valid TOML\n\n" +
			"To fix this issue:\n" +
			"  1. Restore the file from git: git checkout .kanuka/config.toml\n" +
			"  2. Or contact your project administrator for assistance"

	case errors.Is(err, kerrors.ErrInvalidEmail):
		return ui.Error.Sprint("✗") + " Invalid email format: " + ui.Highlight.Sprint(email) +
			"\n" + ui.Info.Sprint("→") + " Please provide a valid email address"

	case errors.Is(err, kerrors.ErrDeviceNameTaken):
		// Extract device name from error message.
		msg := err.Error()
		deviceName := strings.TrimPrefix(msg, "device name already in use: ")
		return ui.Error.Sprint("✗") + " Device name " + ui.Highlight.Sprint(deviceName) + " is already in use for " + ui.Highlight.Sprint(email) +
			"\n" + ui.Info.Sprint("→") + " Choose a different device name with " + ui.Flag.Sprint("--device-name")

	case errors.Is(err, kerrors.ErrPublicKeyExists):
		return ui.Error.Sprint("✗ ") + "Public key already exists" +
			"\nTo override, run: " + ui.Code.Sprint("kanuka secrets create --force")

	default:
		return ui.Error.Sprint("✗") + " Failed to create keys\n" +
			ui.Error.Sprint("Error: ") + err.Error()
	}
}

// isCreateUnexpectedError returns true if the error is unexpected and should cause a non-zero exit.
func isCreateUnexpectedError(err error) bool {
	expectedErrors := []error{
		kerrors.ErrProjectNotInitialized,
		kerrors.ErrInvalidProjectConfig,
		kerrors.ErrInvalidEmail,
		kerrors.ErrDeviceNameTaken,
		kerrors.ErrPublicKeyExists,
	}

	for _, expected := range expectedErrors {
		if errors.Is(err, expected) {
			return false
		}
	}
	return true
}
