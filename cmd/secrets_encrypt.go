package cmd

import (
	"errors"
	"fmt"
	"path/filepath"

	kerrors "github.com/PolarWolf314/kanuka/internal/errors"
	"github.com/PolarWolf314/kanuka/internal/ui"
	"github.com/PolarWolf314/kanuka/internal/utils"
	"github.com/PolarWolf314/kanuka/internal/workflows"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
)

var (
	encryptDryRun          bool
	encryptPrivateKeyStdin bool
)

func init() {
	encryptCmd.Flags().BoolVar(&encryptDryRun, "dry-run", false, "preview encryption without making changes")
	encryptCmd.Flags().BoolVar(&encryptPrivateKeyStdin, "private-key-stdin", false, "read private key from stdin instead of from disk")
}

func resetEncryptCommandState() {
	encryptDryRun = false
	encryptPrivateKeyStdin = false
}

var encryptCmd = &cobra.Command{
	Use:   "encrypt [files...]",
	Short: "Encrypts .env files into .kanuka files using your Kānuka key",
	Long: `Encrypts environment files using your Kānuka key.

If no files are specified, encrypts all .env files in the current directory
and subdirectories (excluding the .kanuka/ directory).

You can specify individual files, directories, or glob patterns:

  kanuka secrets encrypt                      # All .env files
  kanuka secrets encrypt .env                 # Single file
  kanuka secrets encrypt .env .env.local      # Multiple files
  kanuka secrets encrypt "services/*/.env"    # Glob pattern
  kanuka secrets encrypt services/api/        # Directory

Use --dry-run to preview which files would be encrypted without making changes.

Use --private-key-stdin to read your private key from stdin instead of from disk.
This is useful for piping keys from secret managers (e.g., HashiCorp Vault, 1Password).

Examples:
  # Encrypt all .env files
  kanuka secrets encrypt

  # Encrypt specific files
  kanuka secrets encrypt .env .env.local

  # Encrypt with glob pattern (quote to prevent shell expansion)
  kanuka secrets encrypt "services/*/.env"

  # Preview which files would be encrypted
  kanuka secrets encrypt --dry-run

  # Encrypt using a key piped from a secret manager
  vault read -field=private_key secret/kanuka | kanuka secrets encrypt --private-key-stdin`,
	RunE: runEncrypt,
}

func runEncrypt(cmd *cobra.Command, args []string) error {
	Logger.Infof("Starting encrypt command")
	spinner, cleanup := startSpinner("Encrypting environment files...", verbose)
	defer cleanup()

	opts := workflows.EncryptOptions{
		FilePatterns: args,
		DryRun:       encryptDryRun,
	}

	if encryptPrivateKeyStdin {
		Logger.Debugf("Reading private key from stdin")
		keyData, err := utils.ReadStdin()
		if err != nil {
			Logger.Errorf("Failed to read private key from stdin: %v", err)
			spinner.FinalMSG = ui.Error.Sprint("✗") + " Failed to read private key from stdin: " + err.Error()
			return nil
		}
		opts.PrivateKeyData = keyData
	}

	result, err := workflows.Encrypt(cmd.Context(), opts)
	if err != nil {
		Logger.Errorf("Encrypt workflow failed: %v", err)
		spinner.FinalMSG = formatEncryptError(err, encryptPrivateKeyStdin)
		spinner.Stop()
		return nil
	}

	if result.DryRun {
		return printEncryptDryRun(spinner, result.SourceFiles, result.ProjectPath)
	}

	formattedListOfFiles := utils.FormatPaths(result.EncryptedFiles)
	Logger.Infof("Encrypt command completed successfully. Created %d .kanuka files", len(result.EncryptedFiles))

	spinner.FinalMSG = ui.Success.Sprint("✓") + " Environment files encrypted successfully!" +
		"\nThe following files were created: " + formattedListOfFiles +
		"\n" + ui.Info.Sprint("→") + " You can now safely commit all " + ui.Path.Sprint(".kanuka") + " files to version control" +
		"\n\n" + ui.Info.Sprint("Note:") + " Encryption is non-deterministic for security reasons." +
		"\n       Re-encrypting unchanged files will produce different output."

	return nil
}

func formatEncryptError(err error, fromStdin bool) string {
	switch {
	case errors.Is(err, kerrors.ErrProjectNotInitialized):
		return ui.Error.Sprint("✗") + " Kānuka has not been initialized" +
			"\n" + ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " first"

	case errors.Is(err, kerrors.ErrNoFilesFound):
		return ui.Error.Sprint("✗") + " No environment files found"

	case errors.Is(err, kerrors.ErrNoAccess):
		return ui.Error.Sprint("✗") + " Failed to get your " +
			ui.Path.Sprint(".kanuka") + " file. Are you sure you have access?" +
			"\n\n" + ui.Info.Sprint("→") + " You don't have access to this project. Ask someone with access to run:" +
			"\n   " + ui.Code.Sprint("kanuka secrets register --user <your-email>")

	case errors.Is(err, kerrors.ErrPrivateKeyNotFound):
		return ui.Error.Sprint("✗") + " Failed to get your private key file. Are you sure you have access?" +
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

	case errors.Is(err, kerrors.ErrEncryptFailed):
		return ui.Error.Sprint("✗") + " Failed to encrypt project's " +
			ui.Path.Sprint(".env") + " files." +
			"\n\n" + ui.Error.Sprint("Error: ") + err.Error()

	default:
		return ui.Error.Sprint("✗") + " " + err.Error()
	}
}

func printEncryptDryRun(spinner *spinner.Spinner, envFiles []string, projectPath string) error {
	spinner.Stop()

	fmt.Println()
	fmt.Println(ui.Warning.Sprint("[dry-run]") + fmt.Sprintf(" Would encrypt %d environment file(s)", len(envFiles)))
	fmt.Println()

	fmt.Println("Files that would be created:")
	for _, envFile := range envFiles {
		relPath, err := filepath.Rel(projectPath, envFile)
		if err != nil {
			relPath = envFile
		}
		kanukaFile := relPath + ".kanuka"
		fmt.Printf("  %s → %s\n", ui.Path.Sprint(relPath), ui.Success.Sprint(kanukaFile))
	}
	fmt.Println()

	fmt.Println(ui.Info.Sprint("No changes made.") + " Run without --dry-run to execute.")

	spinner.FinalMSG = ""
	return nil
}

// GetEncryptCmd returns the encrypt command for testing.
func GetEncryptCmd() *cobra.Command {
	return encryptCmd
}
