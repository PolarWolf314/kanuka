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
	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
)

var (
	registerUserEmail       string
	customFilePath          string
	publicKeyText           string
	registerDryRun          bool
	registerPrivateKeyStdin bool
	registerForce           bool
	registerPrivateKeyData  []byte
)

// resetRegisterCommandState resets all register command global variables to their default values for testing.
func resetRegisterCommandState() {
	registerUserEmail = ""
	customFilePath = ""
	publicKeyText = ""
	registerDryRun = false
	registerPrivateKeyStdin = false
	registerForce = false
	registerPrivateKeyData = nil
}

func init() {
	RegisterCmd.Flags().StringVarP(&registerUserEmail, "user", "u", "", "user email to register for access")
	RegisterCmd.Flags().StringVarP(&customFilePath, "file", "f", "", "the path to a custom public key — will add public key to the project")
	RegisterCmd.Flags().StringVar(&publicKeyText, "pubkey", "", "OpenSSH or PEM public key content to be saved with the specified user email")
	RegisterCmd.Flags().BoolVar(&registerDryRun, "dry-run", false, "preview registration without making changes")
	RegisterCmd.Flags().BoolVar(&registerPrivateKeyStdin, "private-key-stdin", false, "read private key from stdin instead of from disk")
	RegisterCmd.Flags().BoolVar(&registerForce, "force", false, "skip confirmation when updating existing user's access")
}

// RegisterCmd is the register command.
var RegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Registers a new user to be given access to the repository's secrets",
	Long: `Grants a user access to the project's encrypted secrets.

This command encrypts the project's symmetric key with the target user's
public key, allowing them to decrypt secrets. You must have access to the
project's secrets yourself before you can grant access to others.

Methods to register a user:
  1. By email: --user <email> (user must have run 'secrets create' first)
  2. By public key file: --file <path-to-.pub-file>
  3. By public key text: --pubkey <key-content> --user <email>

After running this command, the user will immediately have access to decrypt
secrets once they pull the latest changes from the repository.

Use --dry-run to preview what would be created without making changes.

Use --private-key-stdin to read your private key from stdin instead of from disk.
This is useful for piping keys from secret managers (e.g., HashiCorp Vault, 1Password).

Examples:
  # Register a user by their email address
  kanuka secrets register --user alice@example.com

  # Register a user with a public key file
  kanuka secrets register --file ./alice-key.pub

  # Register a user with public key text (useful for automation)
  kanuka secrets register --user alice@example.com --pubkey "ssh-rsa AAAA..."

  # Preview registration without making changes
  kanuka secrets register --user alice@example.com --dry-run

  # Register using a key piped from a secret manager
  vault read -field=private_key secret/kanuka | kanuka secrets register --user alice@example.com --private-key-stdin`,
	RunE: runRegister,
}

func runRegister(cmd *cobra.Command, args []string) error {
	Logger.Infof("Starting register command")
	spinner, cleanup := startSpinner("Registering user for access...", verbose)
	defer cleanup()

	// Check for required flags.
	if registerUserEmail == "" && customFilePath == "" && publicKeyText == "" {
		finalMessage := ui.Error.Sprint("✗") + " Either " + ui.Flag.Sprint("--user") + ", " + ui.Flag.Sprint("--file") + ", or " + ui.Flag.Sprint("--pubkey") + " must be specified." +
			"\nRun " + ui.Code.Sprint("kanuka secrets register --help") + " to see the available commands"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// When using --pubkey, user email is required.
	if publicKeyText != "" && registerUserEmail == "" {
		finalMessage := ui.Error.Sprint("✗") + " When using " + ui.Flag.Sprint("--pubkey") + ", the " + ui.Flag.Sprint("--user") + " flag is required." +
			"\nSpecify a user email with " + ui.Flag.Sprint("--user")
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Validate email format if provided.
	if registerUserEmail != "" && !utils.IsValidEmail(registerUserEmail) {
		finalMessage := ui.Error.Sprint("✗") + " Invalid email format: " + ui.Highlight.Sprint(registerUserEmail) +
			"\n" + ui.Info.Sprint("→") + " Please provide a valid email address"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Check if pubkey flag was explicitly used but with empty content.
	if publicKeyText == "" && cmd.Flags().Changed("pubkey") {
		finalMessage := ui.Error.Sprint("✗") + " Invalid public key format provided" +
			"\n" + ui.Error.Sprint("Error: ") + "public key text cannot be empty"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Read private key from stdin early.
	if registerPrivateKeyStdin {
		Logger.Debugf("Reading private key from stdin")
		keyData, err := utils.ReadStdin()
		if err != nil {
			Logger.Errorf("Failed to read private key from stdin: %v", err)
			finalMessage := ui.Error.Sprint("✗") + " Failed to read private key from stdin" +
				"\n" + ui.Error.Sprint("Error: ") + err.Error()
			spinner.FinalMSG = finalMessage
			return nil
		}
		registerPrivateKeyData = keyData
		Logger.Infof("Private key data read from stdin (%d bytes)", len(keyData))
	}

	// Determine registration mode.
	var mode workflows.RegisterMode
	switch {
	case publicKeyText != "":
		mode = workflows.RegisterModePubkeyText
	case customFilePath != "":
		mode = workflows.RegisterModeFile
	default:
		mode = workflows.RegisterModeEmail
	}

	// Handle overwrite confirmation for existing users (interactive - must stay in cmd layer).
	if !registerForce && !registerDryRun {
		_, alreadyHasAccess, err := workflows.CheckUserExistsForRegistration(registerUserEmail)
		if err == nil && alreadyHasAccess {
			if !confirmRegisterOverwrite(spinner, registerUserEmail) {
				spinner.FinalMSG = ui.Warning.Sprint("⚠") + " Registration cancelled."
				return nil
			}
		}
	}

	ctx := context.Background()
	opts := workflows.RegisterOptions{
		Mode:           mode,
		UserEmail:      registerUserEmail,
		PublicKeyText:  publicKeyText,
		FilePath:       customFilePath,
		DryRun:         registerDryRun,
		PrivateKeyData: registerPrivateKeyData,
		Force:          registerForce,
		Verbose:        verbose,
		Debug:          debug,
	}

	result, err := workflows.Register(ctx, opts)
	if err != nil {
		spinner.FinalMSG = formatRegisterError(err, registerUserEmail, customFilePath)
		// Return nil for expected errors, return error for unexpected ones.
		if errors.Is(err, kerrors.ErrProjectNotInitialized) ||
			errors.Is(err, kerrors.ErrUserNotFound) ||
			errors.Is(err, kerrors.ErrNoAccess) ||
			errors.Is(err, kerrors.ErrPublicKeyNotFound) ||
			errors.Is(err, kerrors.ErrInvalidFileType) ||
			strings.Contains(err.Error(), "invalid public key format") {
			return nil
		}
		return err
	}

	if result.DryRun {
		spinner.FinalMSG = ""
		spinner.Stop()
		printRegisterDryRun(result)
		return nil
	}

	spinner.FinalMSG = formatRegisterSuccess(result)
	return nil
}

func formatRegisterError(err error, userEmail, filePath string) string {
	switch {
	case errors.Is(err, kerrors.ErrProjectNotInitialized):
		return ui.Error.Sprint("✗") + " Kānuka has not been initialized\n" +
			ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " instead"

	case errors.Is(err, kerrors.ErrUserNotFound):
		if userEmail != "" {
			return ui.Error.Sprint("✗") + " User " + ui.Highlight.Sprint(userEmail) + " not found in project\n" +
				"They must first run: " + ui.Code.Sprint("kanuka secrets create --email "+userEmail)
		}
		return ui.Error.Sprint("✗") + " User not found in project\n" +
			ui.Info.Sprint("→") + " " + err.Error()

	case errors.Is(err, kerrors.ErrNoAccess):
		return ui.Error.Sprint("✗") + " You don't have access to this project\n" +
			ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets create") + " to generate your keys"

	case errors.Is(err, kerrors.ErrPublicKeyNotFound):
		if userEmail != "" {
			return ui.Error.Sprint("✗") + " Public key for user " + ui.Highlight.Sprint(userEmail) + " not found" +
				"\nThey must first run: " + ui.Code.Sprint("kanuka secrets create --email "+userEmail)
		}
		return ui.Error.Sprint("✗") + " Public key not found\n" +
			ui.Info.Sprint("→") + " " + err.Error()

	case errors.Is(err, kerrors.ErrInvalidFileType):
		if filePath != "" {
			return ui.Error.Sprint("✗ ") + ui.Path.Sprint(filePath) + " is not a valid path to a public key file." +
				"\n\n" + ui.Info.Sprint("→") + " Public key file must be named <uuid>.pub" +
				"\n\nExample:" +
				"\n  mv /tmp/mykey.pub /tmp/550e8400-e29b-41d4-a716-4466554400000.pub" +
				"\n  kanuka secrets register --file /tmp/550e8400-e29b-41d4-a716-4466554400000.pub" +
				"\n\nOr:" +
				"\n  kanuka secrets register --user user@example.com --pubkey \"$(cat /tmp/mykey.pub)\""
		}
		return ui.Error.Sprint("✗") + " Invalid file type\n" +
			ui.Info.Sprint("→") + " " + err.Error()

	case strings.Contains(err.Error(), "toml:"):
		return ui.Error.Sprint("✗") + " Failed to load project configuration.\n\n" +
			ui.Info.Sprint("→") + " The .kanuka/config.toml file is not valid TOML.\n" +
			"   " + ui.Code.Sprint(err.Error()) + "\n\n" +
			"   To fix this issue:\n" +
			"   1. Restore the file from git: " + ui.Code.Sprint("git checkout .kanuka/config.toml") + "\n" +
			"   2. Or contact your project administrator for assistance"

	case strings.Contains(err.Error(), "invalid public key format"):
		return ui.Error.Sprint("✗") + " Invalid public key format provided" +
			"\n" + ui.Error.Sprint("Error: ") + err.Error()

	default:
		return ui.Error.Sprint("✗") + " Registration failed: " + err.Error()
	}
}

func formatRegisterSuccess(result *workflows.RegisterResult) string {
	var successVerb string
	if len(result.FilesUpdated) > 0 {
		successVerb = "access has been updated"
	} else {
		successVerb = "has been granted access"
	}

	finalMessage := ui.Success.Sprint("✓") + " " + ui.Highlight.Sprint(result.DisplayName) + " " + successVerb + " successfully!\n\n"

	if len(result.FilesCreated) > 0 {
		finalMessage += "Files created:\n"
		for _, f := range result.FilesCreated {
			label := "  Encrypted key: "
			if f.Type == "public_key" {
				label = "  Public key:    "
			}
			finalMessage += label + ui.Path.Sprint(f.Path) + "\n"
		}
		finalMessage += "\n"
	}

	if len(result.FilesUpdated) > 0 {
		finalMessage += "Files updated:\n"
		for _, f := range result.FilesUpdated {
			label := "  Encrypted key: "
			if f.Type == "public_key" {
				label = "  Public key:    "
			}
			finalMessage += label + ui.Path.Sprint(f.Path) + "\n"
		}
		finalMessage += "\n"
	}

	finalMessage += ui.Info.Sprint("→") + " They now have access to decrypt the repository's secrets"
	return finalMessage
}

func printRegisterDryRun(result *workflows.RegisterResult) {
	fmt.Println(ui.Warning.Sprint("[dry-run]") + " Would register " + ui.Highlight.Sprint(result.DisplayName))
	fmt.Println()

	fmt.Println("Files that would be created:")
	if result.Mode == workflows.RegisterModePubkeyText {
		fmt.Println("  - " + ui.Success.Sprint(result.PubKeyPath))
	}
	fmt.Println("  - " + ui.Success.Sprint(result.KanukaFilePath))
	fmt.Println()

	fmt.Println("Prerequisites verified:")
	fmt.Println("  " + ui.Success.Sprint("✓") + " User exists in project config")
	if result.Mode == workflows.RegisterModeFile {
		fmt.Println("  " + ui.Success.Sprint("✓") + " Public key loaded from file")
	} else {
		fmt.Println("  " + ui.Success.Sprint("✓") + " Public key found at " + result.PubKeyPath)
	}
	fmt.Println("  " + ui.Success.Sprint("✓") + " Current user has access to decrypt symmetric key")
	fmt.Println()

	fmt.Println(ui.Info.Sprint("No changes made.") + " Run without --dry-run to execute.")
}

// confirmRegisterOverwrite prompts the user to confirm overwriting an existing user's access.
func confirmRegisterOverwrite(s *spinner.Spinner, userEmail string) bool {
	s.Stop()

	fmt.Printf("\n%s Warning: %s already has access to this project.\n", ui.Warning.Sprint("⚠"), ui.Highlight.Sprint(userEmail))
	fmt.Println("  Continuing will replace their existing key.")
	fmt.Println("  If they generated a new keypair, this is expected.")
	fmt.Println("  If not, they may lose access.")
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

// GetRegisterCmd returns the register command for use in tests.
func GetRegisterCmd() *cobra.Command {
	return RegisterCmd
}
