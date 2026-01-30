package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	kerrors "github.com/PolarWolf314/kanuka/internal/errors"
	"github.com/PolarWolf314/kanuka/internal/ui"
	"github.com/PolarWolf314/kanuka/internal/utils"
	"github.com/PolarWolf314/kanuka/internal/workflows"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
)

var decryptDryRun bool
var decryptPrivateKeyStdin bool

func init() {
	decryptCmd.Flags().BoolVar(&decryptDryRun, "dry-run", false, "preview decryption without making changes")
	decryptCmd.Flags().BoolVar(&decryptPrivateKeyStdin, "private-key-stdin", false, "read private key from stdin instead of from disk")
}

func resetDecryptCommandState() {
	decryptDryRun = false
	decryptPrivateKeyStdin = false
}

var decryptCmd = &cobra.Command{
	Use:   "decrypt [files...]",
	Short: "Decrypts .kanuka files back into .env files using your Kānuka key",
	Long: `Decrypts encrypted files using your Kānuka key.

If no files are specified, decrypts all .kanuka files in the current directory
and subdirectories (excluding the .kanuka/ directory).

You can specify individual files, directories, or glob patterns:

  kanuka secrets decrypt                          # All .kanuka files
  kanuka secrets decrypt .env.kanuka              # Single file
  kanuka secrets decrypt "services/*/.env.kanuka" # Glob pattern
  kanuka secrets decrypt services/api/            # Directory

Use --dry-run to preview which files would be decrypted and detect any existing
files that would be overwritten.

Use --private-key-stdin to read your private key from stdin instead of from disk.
This is useful for piping keys from secret managers (e.g., HashiCorp Vault, 1Password).

Examples:
  # Decrypt all .kanuka files
  kanuka secrets decrypt

  # Decrypt specific files
  kanuka secrets decrypt .env.kanuka .env.local.kanuka

  # Decrypt with glob pattern (quote to prevent shell expansion)
  kanuka secrets decrypt "services/*/.env.kanuka"

  # Preview which files would be decrypted
  kanuka secrets decrypt --dry-run

  # Decrypt using a key piped from a secret manager
  vault read -field=private_key secret/kanuka | kanuka secrets decrypt --private-key-stdin`,
	RunE: runDecrypt,
}

func runDecrypt(cmd *cobra.Command, args []string) error {
	Logger.Infof("Starting decrypt command")
	spinner, cleanup := startSpinner("Decrypting environment files...", verbose)
	defer cleanup()

	opts := workflows.DecryptOptions{
		FilePatterns: args,
		DryRun:       decryptDryRun,
	}

	if decryptPrivateKeyStdin {
		Logger.Debugf("Reading private key from stdin")
		keyData, err := utils.ReadStdin()
		if err != nil {
			Logger.Errorf("Failed to read private key from stdin: %v", err)
			spinner.FinalMSG = ui.Error.Sprint("✗") + " Failed to read private key from stdin: " + err.Error()
			return nil
		}
		opts.PrivateKeyData = keyData
	}

	result, err := workflows.Decrypt(cmd.Context(), opts)
	if err != nil {
		Logger.Errorf("Decrypt workflow failed: %v", err)
		spinner.FinalMSG = formatDecryptError(err, decryptPrivateKeyStdin)
		spinner.Stop()
		return nil
	}

	if result.DryRun {
		return printDecryptDryRun(spinner, result.SourceFiles, result.ProjectPath)
	}

	formattedListOfFiles := utils.FormatPaths(result.DecryptedFiles)
	Logger.Infof("Decrypt command completed successfully. Created %d environment files", len(result.DecryptedFiles))

	spinner.Stop()
	Logger.WarnfUser("Decrypted .env files contain sensitive data - ensure they're in your .gitignore")
	spinner.Restart()

	spinner.FinalMSG = ui.Success.Sprint("✓") + " Environment files decrypted successfully!" +
		"\nThe following files were created:" + formattedListOfFiles +
		"\n" + ui.Info.Sprint("→") + " Your environment files are now ready to use"

	return nil
}

func formatDecryptError(err error, fromStdin bool) string {
	switch {
	case errors.Is(err, kerrors.ErrProjectNotInitialized):
		return ui.Error.Sprint("✗") + " Kānuka has not been initialized" +
			"\n" + ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " first"

	case errors.Is(err, kerrors.ErrNoFilesFound):
		return ui.Error.Sprint("✗") + " No encrypted environment (.kanuka) files found"

	case errors.Is(err, kerrors.ErrNoAccess):
		return ui.Error.Sprint("✗") + " Failed to obtain your " +
			ui.Path.Sprint(".kanuka") + " file. Are you sure you have access?" +
			"\n" + err.Error() +
			"\n\n" + ui.Info.Sprint("→") + " You don't have access to this project. Ask someone with access to run:" +
			"\n   " + ui.Code.Sprint("kanuka secrets register --user <your-email>")

	case errors.Is(err, kerrors.ErrPrivateKeyNotFound):
		return ui.Error.Sprint("✗") + " Failed to get your private key file. Are you sure you have access?" +
			"\n" + err.Error() +
			"\n\n" + ui.Info.Sprint("→") + " You don't have access to this project. Ask someone with access to run:" +
			"\n   " + ui.Code.Sprint("kanuka secrets register --user <your-email>")

	case errors.Is(err, kerrors.ErrInvalidPrivateKey):
		if fromStdin {
			return ui.Error.Sprint("✗") + " Failed to parse private key from stdin" +
				"\n" + ui.Info.Sprint("→") + " Ensure your private key is in valid format (PEM or OpenSSH)"
		}
		return ui.Error.Sprint("✗") + " Failed to parse private key" +
			"\n" + ui.Info.Sprint("→") + " Ensure your private key is in valid format (PEM or OpenSSH)"

	case errors.Is(err, kerrors.ErrKeyDecryptFailed):
		return ui.Error.Sprint("✗") + " Failed to decrypt your " +
			ui.Path.Sprint(".kanuka") + " file. Are you sure you have access?" +
			"\n\n" + ui.Info.Sprint("→") + " Your encrypted key file appears to be corrupted." +
			"\n   Try asking the project administrator to revoke and re-register your access."

	case errors.Is(err, kerrors.ErrDecryptFailed):
		return ui.Error.Sprint("✗") + " Failed to decrypt the project's " +
			ui.Path.Sprint(".kanuka") + " files." +
			"\n\n" + ui.Error.Sprint("Error: ") + err.Error()

	default:
		return ui.Error.Sprint("✗") + " " + err.Error()
	}
}

func printDecryptDryRun(s *spinner.Spinner, kanukaFiles []string, projectPath string) error {
	s.Stop()

	fmt.Println()
	fmt.Println(ui.Warning.Sprint("[dry-run]") + fmt.Sprintf(" Would decrypt %d encrypted file(s)", len(kanukaFiles)))
	fmt.Println()

	fmt.Println("Files that would be created:")

	overwriteCount := 0
	for _, kanukaFile := range kanukaFiles {
		relPath, err := filepath.Rel(projectPath, kanukaFile)
		if err != nil {
			relPath = kanukaFile
		}

		envRelPath := strings.TrimSuffix(relPath, ".kanuka")
		envFullPath := strings.TrimSuffix(kanukaFile, ".kanuka")

		status := ui.Success.Sprint("new file")
		if _, err := os.Stat(envFullPath); err == nil {
			status = ui.Warning.Sprint("exists - would be overwritten")
			overwriteCount++
		}

		fmt.Printf("  %s → %s (%s)\n", ui.Path.Sprint(relPath), envRelPath, status)
	}
	fmt.Println()

	if overwriteCount > 0 {
		fmt.Printf(ui.Warning.Sprint("⚠")+" Warning: %d existing file(s) would be overwritten.\n", overwriteCount)
		fmt.Println()
	}

	fmt.Println(ui.Info.Sprint("No changes made.") + " Run without --dry-run to execute.")

	s.FinalMSG = ""
	return nil
}

// GetDecryptCmd returns the decrypt command for testing.
func GetDecryptCmd() *cobra.Command {
	return decryptCmd
}
