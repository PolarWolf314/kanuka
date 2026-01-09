package cmd

import (
	"crypto/rsa"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/audit"
	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/secrets"
	"github.com/PolarWolf314/kanuka/internal/utils"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
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
	Use:   "decrypt",
	Short: "Decrypts the .env.kanuka file back into .env using your Kānuka key",
	Long: `Decrypts all .env.kanuka files in the project back into .env files.

This command discovers all .kanuka files in your project (recursively, excluding
the .kanuka/ directory) and decrypts each one using your private key.
The decrypted files are saved without the .kanuka extension (e.g., .env.kanuka → .env).

Use --dry-run to preview which files would be decrypted and detect any existing
files that would be overwritten.

Use --private-key-stdin to read your private key from stdin instead of from disk.
This is useful for piping keys from secret managers (e.g., HashiCorp Vault, 1Password).

Examples:
  # Decrypt all .kanuka files
  kanuka secrets decrypt

  # Preview which files would be decrypted
  kanuka secrets decrypt --dry-run

  # Decrypt using a key piped from a secret manager
  vault read -field=private_key secret/kanuka | kanuka secrets decrypt --private-key-stdin`,
	RunE: func(cmd *cobra.Command, args []string) error {
		Logger.Infof("Starting decrypt command")
		spinner, cleanup := startSpinner("Decrypting environment files...", verbose)
		defer cleanup()

		Logger.Debugf("Initializing project settings")
		if err := configs.InitProjectSettings(); err != nil {
			return Logger.ErrorfAndReturn("failed to init project settings: %v", err)
		}
		projectName := configs.ProjectKanukaSettings.ProjectName
		projectPath := configs.ProjectKanukaSettings.ProjectPath
		Logger.Debugf("Project name: %s, Project path: %s", projectName, projectPath)

		if projectPath == "" {
			finalMessage := color.RedString("✗") + " Kānuka has not been initialized\n" +
				color.CyanString("→") + " Run " + color.YellowString("kanuka secrets init") + " instead"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// TODO: In future, add config options to list which dirs to ignore. .kanuka/ ignored by default
		Logger.Debugf("Searching for .kanuka files in project path")
		listOfKanukaFiles, err := secrets.FindEnvOrKanukaFiles(projectPath, []string{}, true)
		if err != nil {
			return Logger.ErrorfAndReturn("Failed to find environment files: %v", err)
		}
		Logger.Debugf("Found %d .kanuka files", len(listOfKanukaFiles))
		if len(listOfKanukaFiles) == 0 {
			finalMessage := color.RedString("✗") + " No encrypted environment (" + color.YellowString(".kanuka") + ") files found in " + color.YellowString(projectPath)
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Performance warning for large number of files
		if len(listOfKanukaFiles) > 20 {
			Logger.Warnf("Processing %d encrypted files - this may take a moment", len(listOfKanukaFiles))
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
			finalMessage := color.RedString("✗") + " Failed to obtain your " +
				color.YellowString(".kanuka") + " file. Are you sure you have access?\n" +
				color.RedString("Error: ") + err.Error()
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Load private key - either from stdin or from disk
		var privateKey *rsa.PrivateKey
		if decryptPrivateKeyStdin {
			Logger.Debugf("Reading private key from stdin")
			keyData, err := utils.ReadStdin()
			if err != nil {
				Logger.Errorf("Failed to read private key from stdin: %v", err)
				finalMessage := color.RedString("✗") + " Failed to read private key from stdin\n" +
					color.RedString("Error: ") + err.Error()
				spinner.FinalMSG = finalMessage
				return nil
			}
			privateKey, err = secrets.LoadPrivateKeyFromBytesWithTTYPrompt(keyData)
			if err != nil {
				Logger.Errorf("Failed to parse private key from stdin: %v", err)
				finalMessage := color.RedString("✗") + " Failed to parse private key from stdin\n" +
					color.RedString("Error: ") + err.Error()
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
				finalMessage := color.RedString("✗") + " Failed to get your private key file. Are you sure you have access?\n" +
					color.RedString("Error: ") + err.Error()
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
					spinner.Restart()
				}
			}
		}

		Logger.Debugf("Decrypting symmetric key with private key")
		symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
		if err != nil {
			Logger.Errorf("Failed to decrypt symmetric key: %v", err)
			finalMessage := color.RedString("✗") + " Failed to decrypt your " +
				color.YellowString(".kanuka") + " file. Are you sure you have access?\n" +
				color.RedString("Error: ") + err.Error()

			spinner.FinalMSG = finalMessage
			return nil
		}
		Logger.Infof("Symmetric key decrypted successfully")

		// If dry-run, print preview and exit early.
		if decryptDryRun {
			return printDecryptDryRun(spinner, listOfKanukaFiles, projectPath)
		}

		Logger.Infof("Decrypting %d files", len(listOfKanukaFiles))
		if err := secrets.DecryptFiles(symKey, listOfKanukaFiles, verbose); err != nil {
			Logger.Errorf("Failed to decrypt files: %v", err)
			finalMessage := color.RedString("✗") + " Failed to decrypt the project's " +
				color.YellowString(".kanuka") + " files. Are you sure you have access?\n" +
				color.RedString("Error: ") + err.Error()
			spinner.FinalMSG = finalMessage
			return nil
		}

		// we can be sure they exist if the previous function ran without errors
		Logger.Debugf("Finding decrypted environment files")
		listOfEnvFiles, err := secrets.FindEnvOrKanukaFiles(projectPath, []string{}, false)
		if err != nil {
			return Logger.ErrorfAndReturn("Failed to find environment files after decryption: %v", err)
		}

		formattedListOfFiles := utils.FormatPaths(listOfEnvFiles)
		Logger.Infof("Decrypt command completed successfully. Created %d environment files", len(listOfEnvFiles))

		// Log to audit trail.
		auditEntry := audit.LogWithUser("decrypt")
		auditEntry.Files = listOfEnvFiles
		audit.Log(auditEntry)

		spinner.Stop()
		// Security reminder
		Logger.WarnfUser("Decrypted .env files contain sensitive data - ensure they're in your .gitignore")
		spinner.Restart()

		finalMessage := color.GreenString("✓") + " Environment files decrypted successfully!\n" +
			"The following files were created:" + formattedListOfFiles +
			color.CyanString("→") + " Your environment files are now ready to use"

		spinner.FinalMSG = finalMessage
		return nil
	},
}

func printDecryptDryRun(s *spinner.Spinner, kanukaFiles []string, projectPath string) error {
	s.Stop()

	fmt.Println()
	fmt.Println(color.YellowString("[dry-run]") + fmt.Sprintf(" Would decrypt %d encrypted file(s)", len(kanukaFiles)))
	fmt.Println()

	fmt.Println("Files that would be created:")

	overwriteCount := 0
	for _, kanukaFile := range kanukaFiles {
		// Get relative path for cleaner output.
		relPath, err := filepath.Rel(projectPath, kanukaFile)
		if err != nil {
			relPath = kanukaFile
		}

		// Remove .kanuka extension to get target .env file.
		envRelPath := strings.TrimSuffix(relPath, ".kanuka")
		envFullPath := strings.TrimSuffix(kanukaFile, ".kanuka")

		// Check if target file exists.
		status := color.GreenString("new file")
		if _, err := os.Stat(envFullPath); err == nil {
			status = color.YellowString("exists - would be overwritten")
			overwriteCount++
		}

		fmt.Printf("  %s → %s (%s)\n", color.CyanString(relPath), envRelPath, status)
	}
	fmt.Println()

	if overwriteCount > 0 {
		fmt.Printf(color.YellowString("⚠")+" Warning: %d existing file(s) would be overwritten.\n", overwriteCount)
		fmt.Println()
	}

	fmt.Println(color.CyanString("No changes made.") + " Run without --dry-run to execute.")

	s.FinalMSG = ""
	return nil
}
