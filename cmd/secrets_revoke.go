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
	"github.com/PolarWolf314/kanuka/internal/utils"
	"github.com/PolarWolf314/kanuka/internal/workflows"
	"github.com/spf13/cobra"
)

var (
	revokeUserEmail       string
	revokeFilePath        string
	revokeDevice          string
	revokeYes             bool
	revokeDryRun          bool
	revokePrivateKeyStdin bool
	revokePrivateKeyData  []byte
)

// resetRevokeCommandState resets all revoke command global variables to their default values for testing.
func resetRevokeCommandState() {
	revokeUserEmail = ""
	revokeFilePath = ""
	revokeDevice = ""
	revokeYes = false
	revokeDryRun = false
	revokePrivateKeyStdin = false
	revokePrivateKeyData = nil
}

func init() {
	revokeCmd.Flags().StringVarP(&revokeUserEmail, "user", "u", "", "user email to revoke access from the secret store")
	revokeCmd.Flags().StringVarP(&revokeFilePath, "file", "f", "", "path to a .kanuka file to revoke along with its corresponding public key")
	revokeCmd.Flags().StringVar(&revokeDevice, "device", "", "specific device name to revoke (requires --user)")
	revokeCmd.Flags().BoolVarP(&revokeYes, "yes", "y", false, "skip confirmation prompts (for automation)")
	revokeCmd.Flags().BoolVar(&revokeDryRun, "dry-run", false, "preview revocation without making changes")
	revokeCmd.Flags().BoolVar(&revokePrivateKeyStdin, "private-key-stdin", false, "read private key from stdin instead of from disk")
}

var revokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Revokes access to the secret store",
	Long: `Revokes a user's access to the project's encrypted secrets.

This command removes the user's encrypted symmetric key and public key,
preventing them from decrypting secrets. It also automatically rotates the
symmetric key for all remaining users to ensure the revoked user cannot
decrypt any future secrets.

You can revoke access by:
  1. User email: --user <email> (revokes all devices for that user)
  2. Specific device: --user <email> --device <device-name>
  3. File path: --file <path-to-.kanuka-file>

When revoking a user with multiple devices, you will be prompted to confirm
unless --yes is specified. Use --device to revoke only a specific device.

Use --dry-run to preview what would be revoked without making any changes.
This shows which files would be deleted, config changes, and key rotation impact.

Warning: After revocation, the revoked user may still have access to old
secret values from their local git history. Consider rotating your actual
secret values after this revocation if the user was compromised.

Private Key Input:
  By default, your private key is loaded from disk based on the project UUID.
  Use --private-key-stdin to read the private key from stdin instead (useful
  for CI/CD pipelines or when the key is stored in a secrets manager).

  When using --private-key-stdin with a passphrase-protected key, the
  passphrase prompt will be read from /dev/tty (or CON on Windows), allowing
  you to pipe the key while still entering the passphrase interactively.

Examples:
  # Revoke all devices for a user (prompts for confirmation if multiple)
  kanuka secrets revoke --user alice@example.com

  # Revoke a specific device
  kanuka secrets revoke --user alice@example.com --device macbook-pro

  # Revoke without confirmation (for CI/CD automation)
  kanuka secrets revoke --user alice@example.com --yes

  # Preview revocation without making changes
  kanuka secrets revoke --user alice@example.com --dry-run

  # Revoke by file path
  kanuka secrets revoke --file .kanuka/secrets/abc123.kanuka

  # Revoke with private key from stdin
  cat ~/.ssh/id_rsa | kanuka secrets revoke --user alice@example.com --private-key-stdin

  # Use with a secrets manager
  vault kv get -field=private_key secret/kanuka | kanuka secrets revoke --user alice@example.com --private-key-stdin`,
	RunE: runRevoke,
}

func runRevoke(cmd *cobra.Command, args []string) error {
	Logger.Infof("Starting revoke command")
	spinner, cleanup := startSpinner("Revoking access...", verbose)
	defer cleanup()

	// Validate flags early.
	if revokeDevice != "" && revokeUserEmail == "" {
		finalMessage := ui.Error.Sprint("✗") + " The " + ui.Flag.Sprint("--device") + " flag requires " + ui.Flag.Sprint("--user") + " flag." +
			"\nRun " + ui.Code.Sprint("kanuka secrets revoke --help") + " to see the available commands."
		spinner.FinalMSG = finalMessage
		return nil
	}

	if revokeUserEmail == "" && revokeFilePath == "" {
		finalMessage := ui.Error.Sprint("✗") + " Either " + ui.Flag.Sprint("--user") + " or " + ui.Flag.Sprint("--file") + " flag is required." +
			"\nRun " + ui.Code.Sprint("kanuka secrets revoke --help") + " to see the available commands."
		spinner.FinalMSG = finalMessage
		return nil
	}

	if revokeUserEmail != "" && revokeFilePath != "" {
		finalMessage := ui.Error.Sprint("✗") + " Cannot specify both " + ui.Flag.Sprint("--user") + " and " + ui.Flag.Sprint("--file") + " flags.\n" +
			"Run " + ui.Code.Sprint("kanuka secrets revoke --help") + " to see the available commands.\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Validate email format if provided.
	if revokeUserEmail != "" && !utils.IsValidEmail(revokeUserEmail) {
		finalMessage := ui.Error.Sprint("✗") + " Invalid email format: " + ui.Highlight.Sprint(revokeUserEmail) +
			"\n" + ui.Info.Sprint("→") + " Please provide a valid email address"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Read private key from stdin early, before any other code can consume stdin.
	if revokePrivateKeyStdin {
		Logger.Debugf("Reading private key from stdin")
		keyData, err := utils.ReadStdin()
		if err != nil {
			return Logger.ErrorfAndReturn("failed to read private key from stdin: %v", err)
		}
		revokePrivateKeyData = keyData
		Logger.Infof("Read %d bytes of private key data from stdin", len(keyData))
	}

	// Handle multi-device confirmation prompt (interactive - must stay in cmd layer).
	if revokeUserEmail != "" && revokeDevice == "" && !revokeYes && !revokeDryRun {
		devices, err := workflows.GetDevicesForUser(revokeUserEmail)
		if err == nil && len(devices) > 1 {
			spinner.Stop()

			fmt.Printf("\n%s Warning: %s has %d devices:\n", ui.Warning.Sprint("⚠"), revokeUserEmail, len(devices))
			for _, device := range devices {
				fmt.Printf("  - %s (created: %s)\n", device.Name, device.CreatedAt.Format("Jan 2, 2006"))
			}
			fmt.Println("\nThis will revoke ALL devices for this user.")

			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Proceed? [y/N]: ")
			response, err := reader.ReadString('\n')
			if err != nil {
				return Logger.ErrorfAndReturn("Failed to read response: %v", err)
			}
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				finalMessage := ui.Warning.Sprint("⚠") + " Revocation cancelled."
				spinner.FinalMSG = finalMessage
				return nil
			}

			spinner.Restart()
		}
	}

	ctx := context.Background()
	opts := workflows.RevokeOptions{
		UserEmail:      revokeUserEmail,
		FilePath:       revokeFilePath,
		DeviceName:     revokeDevice,
		DryRun:         revokeDryRun,
		PrivateKeyData: revokePrivateKeyData,
		Verbose:        verbose,
		Debug:          debug,
	}

	result, err := workflows.Revoke(ctx, opts)
	if err != nil {
		spinner.FinalMSG = formatRevokeError(err)
		// Return nil for expected errors, return error for unexpected ones.
		if errors.Is(err, kerrors.ErrProjectNotInitialized) ||
			errors.Is(err, kerrors.ErrUserNotFound) ||
			errors.Is(err, kerrors.ErrDeviceNotFound) ||
			errors.Is(err, kerrors.ErrFileNotFound) ||
			errors.Is(err, kerrors.ErrInvalidFileType) {
			return nil
		}
		return err
	}

	// Handle self-revoke warning (returned as result + error).
	if result != nil && errors.Is(err, kerrors.ErrSelfRevoke) {
		spinner.FinalMSG = formatRevokeSuccess(result) + "\n" +
			ui.Warning.Sprint("⚠") + " Note: You revoked your own access to this project"
		return nil
	}

	if result.DryRun {
		spinner.FinalMSG = ""
		spinner.Stop()
		printRevokeDryRunResult(result)
		return nil
	}

	spinner.FinalMSG = formatRevokeSuccess(result)
	return nil
}

func formatRevokeError(err error) string {
	switch {
	case errors.Is(err, kerrors.ErrProjectNotInitialized):
		return ui.Error.Sprint("✗") + " Kānuka has not been initialized" +
			"\n" + ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " first"

	case errors.Is(err, kerrors.ErrUserNotFound):
		// Extract email from error message if available.
		msg := ui.Error.Sprint("✗") + " User not found in this project"
		if strings.Contains(err.Error(), ":") {
			parts := strings.SplitN(err.Error(), ":", 2)
			if len(parts) == 2 {
				email := strings.TrimSpace(parts[1])
				msg = ui.Error.Sprint("✗") + " User " + ui.Highlight.Sprint(email) + " not found in this project"
			}
		}
		return msg + "\n" + ui.Info.Sprint("→") + " No devices found for this user"

	case errors.Is(err, kerrors.ErrDeviceNotFound):
		return ui.Error.Sprint("✗") + " Device not found" +
			"\n" + ui.Info.Sprint("→") + " " + err.Error()

	case errors.Is(err, kerrors.ErrFileNotFound):
		return ui.Error.Sprint("✗") + " File does not exist" +
			"\n" + ui.Info.Sprint("→") + " " + err.Error()

	case errors.Is(err, kerrors.ErrInvalidFileType):
		return ui.Error.Sprint("✗") + " Invalid file type" +
			"\n" + ui.Info.Sprint("→") + " " + err.Error()

	case strings.Contains(err.Error(), "toml:"):
		return ui.Error.Sprint("✗") + " Failed to load project configuration." +
			"\n\n" + ui.Info.Sprint("→") + " The .kanuka/config.toml file is not valid TOML." +
			"\n   " + ui.Code.Sprint(err.Error()) +
			"\n\n   To fix this issue:" +
			"\n   1. Restore the file from git: " + ui.Code.Sprint("git checkout .kanuka/config.toml") +
			"\n   2. Or contact your project administrator for assistance"

	default:
		return ui.Error.Sprint("✗") + " Revoke failed: " + err.Error()
	}
}

func formatRevokeSuccess(result *workflows.RevokeResult) string {
	finalMessage := ui.Success.Sprint("✓") + " Access for " + ui.Highlight.Sprint(result.DisplayName) + " has been revoked successfully!" +
		"\n" + ui.Info.Sprint("→") + " Revoked: "

	for i, file := range result.RevokedFiles {
		if i > 0 {
			finalMessage += ", "
		}
		finalMessage += ui.Highlight.Sprint(file)
	}

	if result.RemainingUsers > 0 {
		finalMessage += "\n" + ui.Info.Sprint("→") + " All secrets have been re-encrypted with a new key"
	}

	finalMessage += "\n" + ui.Warning.Sprint("⚠") + ui.Error.Sprint(" Warning: ") + ui.Highlight.Sprint(result.DisplayName) + " may still have access to old secrets from their local git history." +
		"\n" + ui.Info.Sprint("→") + " If necessary, rotate your actual secret values after this revocation."

	return finalMessage
}

func printRevokeDryRunResult(result *workflows.RevokeResult) {
	fmt.Println()
	fmt.Println(ui.Warning.Sprint("[dry-run]") + " Would revoke access for " + ui.Highlight.Sprint(result.DisplayName))
	fmt.Println()

	// List files that would be deleted.
	fmt.Println("Files that would be deleted:")
	for _, file := range result.FilesToDelete {
		fmt.Println("  - " + ui.Error.Sprint(file.Path))
	}
	fmt.Println()

	// Show config changes.
	fmt.Println("Config changes:")
	for _, uuid := range result.UUIDsRevoked {
		fmt.Println("  - Remove user " + ui.Highlight.Sprint(uuid) + " from project")
	}
	fmt.Println()

	// Show re-encryption impact.
	if len(result.AllUsers) > len(result.UUIDsRevoked) {
		fmt.Println("Post-revocation actions:")
		fmt.Printf("  - Generate new encryption key\n")
		fmt.Printf("  - Re-encrypt symmetric key for %d remaining user(s)\n", result.RemainingUsers)

		if result.KanukaFilesCount > 0 {
			fmt.Printf("  - Re-encrypt %d secret file(s) with new key\n", result.KanukaFilesCount)
		}
		fmt.Println()
	}

	// Warning about git history.
	fmt.Println(ui.Warning.Sprint("⚠") + " Warning: After revocation, " + result.DisplayName + " may still have access to old secrets from git history.")
	fmt.Println()

	fmt.Println(ui.Info.Sprint("No changes made.") + " Run without --dry-run to execute.")
}

// GetRevokeCmd returns the revoke command for use in tests.
func GetRevokeCmd() *cobra.Command {
	return revokeCmd
}
