package cmd

import (
	"bufio"
	"crypto/rsa"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/audit"
	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/secrets"
	"github.com/PolarWolf314/kanuka/internal/ui"
	"github.com/PolarWolf314/kanuka/internal/utils"
	"github.com/briandowns/spinner"
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

// loadRevokePrivateKey loads the private key for the revoke command.
// If --private-key-stdin was used, it uses the stored key data; otherwise loads from disk.
func loadRevokePrivateKey(projectUUID string) (*rsa.PrivateKey, error) {
	if revokePrivateKeyStdin {
		return secrets.LoadPrivateKeyFromBytesWithTTYPrompt(revokePrivateKeyData)
	}
	privateKeyPath := configs.GetPrivateKeyPath(projectUUID)
	return secrets.LoadPrivateKey(privateKeyPath)
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
	RunE: func(cmd *cobra.Command, args []string) error {
		Logger.Infof("Starting revoke command")
		spinner, cleanup := startSpinner("Revoking access...", verbose)
		defer cleanup()

		// Read private key from stdin early, before any other code can consume stdin
		if revokePrivateKeyStdin {
			Logger.Debugf("Reading private key from stdin")
			keyData, err := utils.ReadStdin()
			if err != nil {
				return Logger.ErrorfAndReturn("failed to read private key from stdin: %v", err)
			}
			revokePrivateKeyData = keyData
			Logger.Infof("Read %d bytes of private key data from stdin", len(keyData))
			Logger.Debugf("Checking command flags: revokeUserEmail=%s, revokeFilePath=%s, revokeDevice=%s, revokeYes=%t",
				revokeUserEmail, revokeFilePath, revokeDevice, revokeYes)

			// Check --device requires --user FIRST
			if revokeDevice != "" && revokeUserEmail == "" {
				finalMessage := ui.Error.Sprint("✗") + " The " + ui.Flag.Sprint("--device") + " flag requires " + ui.Flag.Sprint("--user") + " flag.\n" +
					"Run " + ui.Code.Sprint("kanuka secrets revoke --help") + " to see the available commands.\n"
				spinner.FinalMSG = finalMessage
				return nil
			}

			// Then do the general check
			if revokeUserEmail == "" && revokeFilePath == "" {
				finalMessage := ui.Error.Sprint("✗") + " Either " + ui.Flag.Sprint("--user") + " or " + ui.Flag.Sprint("--file") + " flag is required.\n" +
					"Run " + ui.Code.Sprint("kanuka secrets revoke --help") + " to see the available commands.\n"
				spinner.FinalMSG = finalMessage
				return nil
			}
		}

		if revokeUserEmail != "" && revokeFilePath != "" {
			finalMessage := ui.Error.Sprint("✗") + " Cannot specify both " + ui.Flag.Sprint("--user") + " and " + ui.Flag.Sprint("--file") + " flags.\n" +
				"Run " + ui.Code.Sprint("kanuka secrets revoke --help") + " to see the available commands.\n"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Validate email format if provided
		if revokeUserEmail != "" && !utils.IsValidEmail(revokeUserEmail) {
			finalMessage := ui.Error.Sprint("✗") + " Invalid email format: " + ui.Highlight.Sprint(revokeUserEmail) + "\n" +
				ui.Info.Sprint("→") + " Please provide a valid email address"
			spinner.FinalMSG = finalMessage
			return nil
		}

		Logger.Debugf("Initializing project settings")
		if err := configs.InitProjectSettings(); err != nil {
			return Logger.ErrorfAndReturn("failed to init project settings: %v", err)
		}

		projectPath := configs.ProjectKanukaSettings.ProjectPath
		if projectPath == "" {
			finalMessage := ui.Error.Sprint("✗") + " Kānuka has not been initialized\n" +
				ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " first\n"
			spinner.FinalMSG = finalMessage
			return nil
		}

		exists, err := secrets.DoesProjectKanukaSettingsExist()
		if err != nil {
			return Logger.ErrorfAndReturn("failed to check if project exists: %v", err)
		}
		if !exists {
			finalMessage := ui.Error.Sprint("✗") + " Kānuka project not found\n" +
				ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " first\n"
			spinner.FinalMSG = finalMessage
			return nil
		}

		ctx, err := getFilesToRevoke(spinner)
		if err != nil {
			return err
		}

		if ctx == nil || len(ctx.Files) == 0 {
			return nil
		}

		return revokeFiles(spinner, ctx)
	},
}

type fileToRevoke struct {
	Path string
	Name string
}

type revokeContext struct {
	DisplayName  string
	Files        []fileToRevoke
	UUIDsRevoked []string // UUIDs to remove from project config
}

func getFilesToRevoke(spinner *spinner.Spinner) (*revokeContext, error) {
	if revokeUserEmail != "" {
		return getFilesByUserEmail(spinner)
	}
	return getFilesByPath(spinner)
}

func getFilesByUserEmail(spinner *spinner.Spinner) (*revokeContext, error) {
	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath
	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath

	Logger.Debugf("Project public key path: %s, Project secrets path: %s", projectPublicKeyPath, projectSecretsPath)

	// Load project config to look up user by email
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		if strings.Contains(err.Error(), "toml:") {
			return nil, fmt.Errorf("failed to load project config: .kanuka/config.toml is not valid TOML\n\nTo fix this issue:\n  1. Restore the file from git: git checkout .kanuka/config.toml\n  2. Or contact your project administrator for assistance\n\nDetails: %v", err)
		}
		return nil, Logger.ErrorfAndReturn("failed to load project config: %v", err)
	}

	// Get all devices for this email
	devices := projectConfig.GetDevicesByEmail(revokeUserEmail)
	if len(devices) == 0 {
		finalMessage := ui.Error.Sprint("✗") + " User " + ui.Highlight.Sprint(revokeUserEmail) + " not found in this project\n" +
			ui.Info.Sprint("→") + " No devices found for this user\n"
		spinner.FinalMSG = finalMessage
		return nil, nil
	}

	// If --device is specified, find that specific device
	if revokeDevice != "" {
		targetUserUUID, found := projectConfig.GetUserUUIDByEmailAndDevice(revokeUserEmail, revokeDevice)
		if !found {
			finalMessage := ui.Error.Sprint("✗") + " Device " + ui.Highlight.Sprint(revokeDevice) + " not found for user " + ui.Highlight.Sprint(revokeUserEmail) + "\n" +
				ui.Info.Sprint("→") + " Available devices:\n"
			for _, device := range devices {
				finalMessage += "    - " + ui.Highlight.Sprint(device.Name) + "\n"
			}
			spinner.FinalMSG = finalMessage
			return nil, nil
		}

		// Return files for this specific device
		return getFilesForUUID(spinner, targetUserUUID, revokeUserEmail+" ("+revokeDevice+")")
	}

	// No --device specified, handle all devices for this user
	// If multiple devices and no --yes, prompt for confirmation
	if len(devices) > 1 && !revokeYes {
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
			return nil, Logger.ErrorfAndReturn("Failed to read response: %v", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			finalMessage := ui.Warning.Sprint("⚠") + " Revocation cancelled\n"
			spinner.FinalMSG = finalMessage
			spinner.Restart()
			return nil, nil
		}

		spinner.Restart()
	}

	// Collect all files and UUIDs for all devices
	var allFiles []fileToRevoke
	var allUUIDs []string
	for userUUID := range devices {
		allUUIDs = append(allUUIDs, userUUID)
		publicKeyPath := filepath.Join(projectPublicKeyPath, userUUID+".pub")
		kanukaKeyPath := filepath.Join(projectSecretsPath, userUUID+".kanuka")

		if _, err := os.Stat(publicKeyPath); err == nil {
			allFiles = append(allFiles, fileToRevoke{Path: publicKeyPath, Name: userUUID + ".pub"})
		}
		if _, err := os.Stat(kanukaKeyPath); err == nil {
			allFiles = append(allFiles, fileToRevoke{Path: kanukaKeyPath, Name: userUUID + ".kanuka"})
		}
	}

	if len(allFiles) == 0 {
		finalMessage := ui.Error.Sprint("✗") + " No files found for user " + ui.Highlight.Sprint(revokeUserEmail) + "\n"
		spinner.FinalMSG = finalMessage
		return nil, nil
	}

	return &revokeContext{
		DisplayName:  revokeUserEmail,
		Files:        allFiles,
		UUIDsRevoked: allUUIDs,
	}, nil
}

func getFilesForUUID(spinner *spinner.Spinner, userUUID, displayName string) (*revokeContext, error) {
	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath
	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath

	publicKeyPath := filepath.Join(projectPublicKeyPath, userUUID+".pub")
	kanukaKeyPath := filepath.Join(projectSecretsPath, userUUID+".kanuka")

	Logger.Debugf("Checking for user files: %s, %s", publicKeyPath, kanukaKeyPath)

	publicKeyExists := false
	kanukaKeyExists := false

	if _, err := os.Stat(publicKeyPath); err == nil {
		publicKeyExists = true
	} else if !os.IsNotExist(err) {
		Logger.Errorf("Failed to check public key file %s: %v", publicKeyPath, err)
		finalMessage := ui.Error.Sprint("✗") + " Failed to check user's public key file\n" +
			ui.Error.Sprint("Error: ") + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return nil, nil
	}

	if _, err := os.Stat(kanukaKeyPath); err == nil {
		kanukaKeyExists = true
	} else if !os.IsNotExist(err) {
		Logger.Errorf("Failed to check kanuka key file %s: %v", kanukaKeyPath, err)
		finalMessage := ui.Error.Sprint("✗") + " Failed to check user's kanuka key file\n" +
			ui.Error.Sprint("Error: ") + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return nil, nil
	}

	if !publicKeyExists && !kanukaKeyExists {
		Logger.Infof("User %s does not exist in the project", displayName)
		finalMessage := ui.Error.Sprint("✗") + " User " + ui.Highlight.Sprint(displayName) + " does not exist in this project\n" +
			ui.Info.Sprint("→") + " No files found for this user\n"
		spinner.FinalMSG = finalMessage
		return nil, nil
	}

	var files []fileToRevoke
	if publicKeyExists {
		files = append(files, fileToRevoke{Path: publicKeyPath, Name: userUUID + ".pub"})
	}
	if kanukaKeyExists {
		files = append(files, fileToRevoke{Path: kanukaKeyPath, Name: userUUID + ".kanuka"})
	}

	return &revokeContext{
		DisplayName:  displayName,
		Files:        files,
		UUIDsRevoked: []string{userUUID},
	}, nil
}

func getFilesByPath(spinner *spinner.Spinner) (*revokeContext, error) {
	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath
	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath

	Logger.Debugf("Project secrets path: %s, Project public key path: %s", projectSecretsPath, projectPublicKeyPath)

	absFilePath, err := filepath.Abs(revokeFilePath)
	if err != nil {
		finalMessage := ui.Error.Sprint("✗") + " Failed to resolve file path: " + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return nil, nil
	}

	Logger.Debugf("Absolute file path: %s", absFilePath)

	fileInfo, err := os.Stat(absFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			Logger.Infof("File does not exist: %s", absFilePath)
			finalMessage := ui.Error.Sprint("✗") + " File " + ui.Path.Sprint(absFilePath) + " does not exist\n"
			spinner.FinalMSG = finalMessage
			return nil, nil
		}
		Logger.Errorf("Failed to check file %s: %v", absFilePath, err)
		finalMessage := ui.Error.Sprint("✗") + " Failed to check file\n" +
			ui.Error.Sprint("Error: ") + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return nil, nil
	}

	if fileInfo.IsDir() {
		finalMessage := ui.Error.Sprint("✗") + " Path " + ui.Path.Sprint(absFilePath) + " is a directory, not a file\n"
		spinner.FinalMSG = finalMessage
		return nil, nil
	}

	absProjectSecretsPath, err := filepath.Abs(projectSecretsPath)
	if err != nil {
		finalMessage := ui.Error.Sprint("✗") + " Failed to resolve project secrets path: " + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return nil, nil
	}

	if filepath.Dir(absFilePath) != absProjectSecretsPath {
		finalMessage := ui.Error.Sprint("✗") + " File " + ui.Path.Sprint(absFilePath) + " is not in the project secrets directory\n" +
			ui.Info.Sprint("→") + " Expected directory: " + ui.Path.Sprint(absProjectSecretsPath) + "\n"
		spinner.FinalMSG = finalMessage
		return nil, nil
	}

	if filepath.Ext(absFilePath) != ".kanuka" {
		finalMessage := ui.Error.Sprint("✗") + " File " + ui.Path.Sprint(absFilePath) + " is not a .kanuka file\n"
		spinner.FinalMSG = finalMessage
		return nil, nil
	}

	baseName := filepath.Base(absFilePath)
	userUUID := baseName[:len(baseName)-len(".kanuka")]

	Logger.Debugf("Extracted user UUID from file: %s", userUUID)

	// Try to find email for display purposes
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		Logger.Warnf("Could not load project config for display name lookup: %v", err)
	}
	displayName := userUUID
	if projectConfig != nil {
		if email, exists := projectConfig.Users[userUUID]; exists && email != "" {
			displayName = email
		}
	}

	var files []fileToRevoke
	files = append(files, fileToRevoke{Path: absFilePath, Name: baseName})

	publicKeyPath := filepath.Join(projectPublicKeyPath, userUUID+".pub")
	if _, err := os.Stat(publicKeyPath); err == nil {
		files = append(files, fileToRevoke{Path: publicKeyPath, Name: userUUID + ".pub"})
	} else if !os.IsNotExist(err) {
		Logger.Errorf("Failed to check public key file %s: %v", publicKeyPath, err)
		finalMessage := ui.Error.Sprint("✗") + " Failed to check public key file\n" +
			ui.Error.Sprint("Error: ") + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return nil, nil
	}

	return &revokeContext{
		DisplayName:  displayName,
		Files:        files,
		UUIDsRevoked: []string{userUUID},
	}, nil
}

func printRevokeDryRun(spinner *spinner.Spinner, ctx *revokeContext) error {
	spinner.Stop()

	fmt.Println()
	fmt.Println(ui.Warning.Sprint("[dry-run]") + " Would revoke access for " + ui.Highlight.Sprint(ctx.DisplayName))
	fmt.Println()

	// List files that would be deleted.
	fmt.Println("Files that would be deleted:")
	for _, file := range ctx.Files {
		fmt.Println("  - " + ui.Error.Sprint(file.Path))
	}
	fmt.Println()

	// Show config changes.
	fmt.Println("Config changes:")
	for _, uuid := range ctx.UUIDsRevoked {
		fmt.Println("  - Remove user " + ui.Highlight.Sprint(uuid) + " from project")
	}
	fmt.Println()

	// Show re-encryption impact.
	allUsers, err := secrets.GetAllUsersInProject()
	if err == nil && len(allUsers) > len(ctx.UUIDsRevoked) {
		remainingCount := len(allUsers) - len(ctx.UUIDsRevoked)
		fmt.Println("Post-revocation actions:")
		fmt.Printf("  - Generate new encryption key\n")
		fmt.Printf("  - Re-encrypt symmetric key for %d remaining user(s)\n", remainingCount)

		// Count secret files that would be re-encrypted.
		projectPath := configs.ProjectKanukaSettings.ProjectPath
		if projectPath != "" {
			kanukaFiles, err := secrets.FindEnvOrKanukaFiles(projectPath, []string{}, true)
			if err == nil && len(kanukaFiles) > 0 {
				fmt.Printf("  - Re-encrypt %d secret file(s) with new key\n", len(kanukaFiles))
			}
		}
		fmt.Println()
	}

	// Warning about git history.
	fmt.Println(ui.Warning.Sprint("⚠") + " Warning: After revocation, " + ctx.DisplayName + " may still have access to old secrets from git history.")
	fmt.Println()

	fmt.Println(ui.Info.Sprint("No changes made.") + " Run without --dry-run to execute.")

	spinner.FinalMSG = ""
	return nil
}

func revokeFiles(spinner *spinner.Spinner, ctx *revokeContext) error {
	if len(ctx.Files) == 0 {
		return nil
	}

	// If dry-run, print preview and exit early.
	if revokeDryRun {
		return printRevokeDryRun(spinner, ctx)
	}

	displayName := ctx.DisplayName
	filesToRevoke := ctx.Files

	// Load user config for current user UUID
	userConfig, err := configs.EnsureUserConfig()
	if err != nil {
		return Logger.ErrorfAndReturn("failed to load user config: %v", err)
	}
	currentUserUUID := userConfig.User.UUID

	// Load project config for project UUID and updating
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		if strings.Contains(err.Error(), "toml:") {
			Logger.Errorf("Failed to load project config: %v", err)
			finalMessage := ui.Error.Sprint("✗") + " Failed to load project configuration.\n\n" +
				ui.Info.Sprint("→") + " The .kanuka/config.toml file is not valid TOML.\n" +
				"   " + ui.Code.Sprint(err.Error()) + "\n\n" +
				"   To fix this issue:\n" +
				"   1. Restore the file from git: " + ui.Code.Sprint("git checkout .kanuka/config.toml") + "\n" +
				"   2. Or contact your project administrator for assistance"
			spinner.FinalMSG = finalMessage
			spinner.Stop()
			return nil
		}
		return Logger.ErrorfAndReturn("failed to load project config: %v", err)
	}
	projectUUID := projectConfig.Project.UUID

	Logger.Debugf("Current user UUID: %s, Project UUID: %s", currentUserUUID, projectUUID)

	var revokedFiles []string
	var revokeErrors []error

	for _, file := range filesToRevoke {
		Logger.Debugf("Revoking file: %s", file.Path)
		if err := os.Remove(file.Path); err != nil {
			Logger.Errorf("Failed to revoke file %s: %v", file.Path, err)
			revokeErrors = append(revokeErrors, err)
		} else {
			revokedFiles = append(revokedFiles, file.Name)
			Logger.Infof("Successfully revoked file: %s", file.Name)
		}
	}

	if len(revokeErrors) > 0 {
		finalMessage := ui.Error.Sprint("✗") + " Failed to completely revoke files for " + ui.Highlight.Sprint(displayName) + "\n"
		for _, err := range revokeErrors {
			finalMessage += ui.Error.Sprint("Error: ") + err.Error() + "\n"
		}
		if len(revokedFiles) > 0 {
			finalMessage += ui.Warning.Sprint("Warning: ") + "Some files were revoked successfully\n"
		}
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Remove revoked UUIDs from project config
	for _, uuid := range ctx.UUIDsRevoked {
		Logger.Debugf("Removing UUID %s from project config", uuid)
		projectConfig.RemoveDevice(uuid)
	}

	// Save updated project config
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		Logger.Errorf("Failed to update project config: %v", err)
		finalMessage := ui.Error.Sprint("✗") + " Files were revoked but failed to update project config: " + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return nil
	}
	Logger.Infof("Project config updated - removed %d device(s)", len(ctx.UUIDsRevoked))

	allUsers, err := secrets.GetAllUsersInProject()
	if err != nil {
		Logger.Errorf("Failed to get list of users: %v", err)
		finalMessage := ui.Error.Sprint("✗") + " Files were revoked but failed to rotate key: " + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	if len(allUsers) > 0 {
		spinner.Suffix = " Re-encrypting secrets for remaining users..."
		Logger.Infof("Re-encrypting secrets for %d remaining users", len(allUsers))

		privateKey, err := loadRevokePrivateKey(projectUUID)
		if err != nil {
			Logger.Errorf("Failed to load private key: %v", err)
			keySource := "from disk"
			if revokePrivateKeyStdin {
				keySource = "from stdin"
			}
			finalMessage := ui.Error.Sprint("✗") + " Files were revoked but failed to load private key " + keySource + ": " + err.Error() + "\n"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Use SyncSecrets to re-encrypt all secrets with a new key.
		// This ensures the revoked user cannot decrypt any secrets, even if they
		// previously copied the encrypted files and had the old symmetric key.
		syncOpts := secrets.SyncOptions{
			ExcludeUsers: ctx.UUIDsRevoked,
			Verbose:      verbose,
			Debug:        debug,
		}

		syncResult, err := secrets.SyncSecrets(privateKey, syncOpts)
		if err != nil {
			Logger.Errorf("Failed to re-encrypt secrets: %v", err)
			finalMessage := ui.Error.Sprint("✗") + " Files were revoked but failed to re-encrypt secrets: " + err.Error() + "\n"
			spinner.FinalMSG = finalMessage
			return nil
		}

		Logger.Infof("Secrets re-encrypted successfully: %d secrets for %d users", syncResult.SecretsProcessed, syncResult.UsersProcessed)
	}

	Logger.Infof("Files revocation completed successfully for: %s", displayName)

	// Log to audit trail.
	auditEntry := audit.LogWithUser("revoke")
	auditEntry.TargetUser = displayName
	if len(ctx.UUIDsRevoked) > 0 {
		auditEntry.TargetUUID = ctx.UUIDsRevoked[0]
	}
	if revokeDevice != "" {
		auditEntry.Device = revokeDevice
	}
	audit.Log(auditEntry)

	finalMessage := ui.Success.Sprint("✓") + " Access for " + ui.Highlight.Sprint(displayName) + " has been revoked successfully!\n" +
		ui.Info.Sprint("→") + " Revoked: "
	for i, file := range revokedFiles {
		if i > 0 {
			finalMessage += ", "
		}
		finalMessage += ui.Highlight.Sprint(file)
	}
	finalMessage += "\n"
	if len(allUsers) > 0 {
		finalMessage += ui.Info.Sprint("→") + " All secrets have been re-encrypted with a new key\n"
	}
	finalMessage += ui.Warning.Sprint("⚠") + ui.Error.Sprint(" Warning: ") + ui.Highlight.Sprint(displayName) + " may still have access to old secrets from their local git history.\n" +
		ui.Info.Sprint("→") + " If necessary, rotate your actual secret values after this revocation.\n"
	spinner.FinalMSG = finalMessage
	return nil
}
