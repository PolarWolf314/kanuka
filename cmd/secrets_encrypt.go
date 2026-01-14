package cmd

import (
	"crypto/rsa"
	"fmt"
	"os"
	"path/filepath"

	"github.com/PolarWolf314/kanuka/internal/audit"
	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/secrets"
	"github.com/PolarWolf314/kanuka/internal/ui"
	"github.com/PolarWolf314/kanuka/internal/utils"

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
	RunE: func(cmd *cobra.Command, args []string) error {
		Logger.Infof("Starting encrypt command")
		spinner, cleanup := startSpinner("Encrypting environment files...", verbose)
		defer cleanup()

		Logger.Debugf("Initializing project settings")
		if err := configs.InitProjectSettings(); err != nil {
			return Logger.ErrorfAndReturn("failed to init project settings: %v", err)
		}
		projectName := configs.ProjectKanukaSettings.ProjectName
		projectPath := configs.ProjectKanukaSettings.ProjectPath
		Logger.Debugf("Project name: %s, Project path: %s", projectName, projectPath)

		if projectPath == "" {
			finalMessage := ui.Error.Sprint("✗") + " Kānuka has not been initialized\n" +
				ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " first\n"
			spinner.FinalMSG = finalMessage
			spinner.Stop()
			return nil
		}

		// TODO: In future, add config options to list which dirs to ignore. .kanuka/ ignored by default
		Logger.Debugf("Resolving files to encrypt")

		var listOfEnvFiles []string
		if len(args) > 0 {
			// Use user-provided file patterns.
			Logger.Debugf("User provided %d file pattern(s)", len(args))
			resolved, err := secrets.ResolveFiles(args, projectPath, true)
			if err != nil {
				Logger.Errorf("Failed to resolve file patterns: %v", err)
				finalMessage := ui.Error.Sprint("✗") + " " + err.Error()
				spinner.FinalMSG = finalMessage
				return nil
			}
			listOfEnvFiles = resolved
		} else {
			// Default: find all .env files.
			Logger.Debugf("Searching for .env files in project path")
			found, err := secrets.FindEnvOrKanukaFiles(projectPath, []string{}, false)
			if err != nil {
				return Logger.ErrorfAndReturn("Failed to find environment files: %v", err)
			}
			listOfEnvFiles = found
		}
		Logger.Debugf("Found %d .env files", len(listOfEnvFiles))
		if len(listOfEnvFiles) == 0 {
			finalMessage := ui.Error.Sprint("✗") + " No environment files found in " + ui.Path.Sprint(projectPath)
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Performance warning for large number of files
		if len(listOfEnvFiles) > 20 {
			Logger.Warnf("Processing %d environment files - this may take a moment", len(listOfEnvFiles))
		}

		// Load user config for user UUID
		userConfig, err := configs.EnsureUserConfig()
		if err != nil {
			return Logger.ErrorfAndReturn("failed to load user config: %v", err)
		}
		userUUID := userConfig.User.UUID

		// Load project config for project UUID
		projectConfig, err := configs.LoadProjectConfig()
		if err != nil {
			return Logger.ErrorfAndReturn("failed to load project config: %v", err)
		}
		projectUUID := projectConfig.Project.UUID

		Logger.Debugf("User UUID: %s", userUUID)

		Logger.Debugf("Getting project kanuka key for user: %s", userUUID)
		encryptedSymKey, err := secrets.GetProjectKanukaKey(userUUID)
		if err != nil {
			Logger.Errorf("Failed to obtain kanuka key for user %s: %v", userUUID, err)
			finalMessage := ui.Error.Sprint("✗") + " Failed to get your " +
				ui.Path.Sprint(".kanuka") + " file. Are you sure you have access?\n\n" +
				ui.Info.Sprint("→") + " You don't have access to this project. Ask someone with access to run:\n" +
				"   " + ui.Code.Sprint("kanuka secrets register --user <your-email>\n")
			spinner.FinalMSG = finalMessage
			spinner.Stop()
			return nil
		}

		// Load private key - either from stdin or from disk
		var privateKey *rsa.PrivateKey
		if encryptPrivateKeyStdin {
			Logger.Debugf("Reading private key from stdin")
			keyData, err := utils.ReadStdin()
			if err != nil {
				Logger.Errorf("Failed to read private key from stdin: %v", err)
				finalMessage := ui.Error.Sprint("✗") + " Failed to read private key from stdin\n" +
					ui.Error.Sprint("Error: ") + err.Error()
				spinner.FinalMSG = finalMessage
				return nil
			}
			privateKey, err = secrets.LoadPrivateKeyFromBytesWithTTYPrompt(keyData)
			if err != nil {
				Logger.Errorf("Failed to parse private key from stdin: %v", err)
				finalMessage := ui.Error.Sprint("✗") + " Failed to parse private key from stdin\n" +
					ui.Error.Sprint("Error: ") + err.Error()
				spinner.FinalMSG = finalMessage
				return nil
			}
			Logger.Infof("Private key loaded successfully from stdin")
		} else {
			privateKeyPath := configs.GetPrivateKeyPath(projectUUID)
			Logger.Debugf("Loading private key from: %s", privateKeyPath)
			privateKey, err = secrets.LoadPrivateKey(privateKeyPath)
			if err != nil {
				Logger.Errorf("Failed to load private key from %s: %v", privateKeyPath, err)
				finalMessage := ui.Error.Sprint("✗") + " Failed to get your private key file. Are you sure you have access?\n" +
					ui.Error.Sprint("Error: ") + err.Error()
				spinner.FinalMSG = finalMessage
				return nil
			}
			Logger.Infof("Private key loaded successfully")

			// Security warning: Check private key file permissions
			if fileInfo, err := os.Stat(privateKeyPath); err == nil {
				if fileInfo.Mode().Perm() != 0600 {
					spinner.Stop()
					Logger.WarnfAlways("Private key file has overly permissive permissions (%o), consider running 'chmod 600 %s'",
						fileInfo.Mode().Perm(), privateKeyPath)
					spinner.Start()
				}
			}
		}

		Logger.Debugf("Decrypting symmetric key with private key")
		symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
		if err != nil {
			Logger.Errorf("Failed to decrypt symmetric key: %v", err)
			finalMessage := ui.Error.Sprint("✗") + " Failed to decrypt your " +
				ui.Path.Sprint(".kanuka") + " file. Are you sure you have access?\n" +
				ui.Error.Sprint("Error: ") + err.Error()

			spinner.FinalMSG = finalMessage
			return nil
		}
		Logger.Infof("Symmetric key decrypted successfully")

		// If dry-run, print preview and exit early.
		if encryptDryRun {
			return printEncryptDryRun(spinner, listOfEnvFiles, projectPath)
		}

		Logger.Infof("Encrypting %d files", len(listOfEnvFiles))
		if err := secrets.EncryptFiles(symKey, listOfEnvFiles, verbose); err != nil {
			Logger.Errorf("Failed to encrypt files: %v", err)
			finalMessage := ui.Error.Sprint("✗") + " Failed to encrypt the project's " +
				ui.Path.Sprint(".env") + " files. Are you sure you have access?\n" +
				ui.Error.Sprint("Error: ") + err.Error()
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Convert .env files to .kanuka file paths for display.
		listOfKanukaFiles := make([]string, len(listOfEnvFiles))
		for i, envFile := range listOfEnvFiles {
			listOfKanukaFiles[i] = envFile + ".kanuka"
		}

		formattedListOfFiles := utils.FormatPaths(listOfKanukaFiles)
		Logger.Infof("Encrypt command completed successfully. Created %d .kanuka files", len(listOfKanukaFiles))

		// Log to audit trail.
		auditEntry := audit.LogWithUser("encrypt")
		auditEntry.Files = listOfKanukaFiles
		audit.Log(auditEntry)

		finalMessage := ui.Success.Sprint("✓") + " Environment files encrypted successfully!\n" +
			"The following files were created: " + formattedListOfFiles +
			ui.Info.Sprint("→") + " You can now safely commit all " + ui.Path.Sprint(".kanuka") + " files to version control\n\n" +
			ui.Info.Sprint("Note:") + " Encryption is non-deterministic for security reasons.\n" +
			"       Re-encrypting unchanged files will produce different output."

		spinner.FinalMSG = finalMessage
		return nil
	},
}

func printEncryptDryRun(spinner *spinner.Spinner, envFiles []string, projectPath string) error {
	spinner.Stop()

	fmt.Println()
	fmt.Println(ui.Warning.Sprint("[dry-run]") + fmt.Sprintf(" Would encrypt %d environment file(s)", len(envFiles)))
	fmt.Println()

	fmt.Println("Files that would be created:")
	for _, envFile := range envFiles {
		// Get relative path for cleaner output.
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
