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
	"github.com/PolarWolf314/kanuka/internal/utils"

	"github.com/PolarWolf314/kanuka/internal/ui"
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
	// privateKeyData holds the private key data read from stdin (if --private-key-stdin is used).
	// This is stored so it can be used by multiple functions without re-reading stdin.
	registerPrivateKeyData []byte
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

// loadRegisterPrivateKey loads the private key for the register command.
// If --private-key-stdin was used, it uses the stored key data; otherwise loads from disk.
func loadRegisterPrivateKey(projectUUID string) (*rsa.PrivateKey, error) {
	if registerPrivateKeyStdin {
		return secrets.LoadPrivateKeyFromBytesWithTTYPrompt(registerPrivateKeyData)
	}
	privateKeyPath := configs.GetPrivateKeyPath(projectUUID)
	return secrets.LoadPrivateKey(privateKeyPath)
}

// fileExists checks if a file exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil && !info.IsDir()
}

// confirmRegisterOverwrite prompts the user to confirm overwriting an existing user's access.
// Returns true if the user confirms, false otherwise.
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

func init() {
	RegisterCmd.Flags().StringVarP(&registerUserEmail, "user", "u", "", "user email to register for access")
	RegisterCmd.Flags().StringVarP(&customFilePath, "file", "f", "", "the path to a custom public key — will add public key to the project")
	RegisterCmd.Flags().StringVar(&publicKeyText, "pubkey", "", "OpenSSH or PEM public key content to be saved with the specified user email")
	RegisterCmd.Flags().BoolVar(&registerDryRun, "dry-run", false, "preview registration without making changes")
	RegisterCmd.Flags().BoolVar(&registerPrivateKeyStdin, "private-key-stdin", false, "read private key from stdin instead of from disk")
	RegisterCmd.Flags().BoolVar(&registerForce, "force", false, "skip confirmation when updating existing user's access")
}

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
	RunE: func(cmd *cobra.Command, args []string) error {
		Logger.Infof("Starting register command")
		spinner, cleanup := startSpinner("Registering user for access...", verbose)
		defer cleanup()

		// Check for required flags
		Logger.Debugf("Checking command flags: registerUserEmail=%s, customFilePath=%s, publicKeyText provided=%t", registerUserEmail, customFilePath, publicKeyText != "")
		if registerUserEmail == "" && customFilePath == "" && publicKeyText == "" {
			finalMessage := ui.Error.Sprint("✗") + " Either " + ui.Flag.Sprint("--user") + ", " + ui.Flag.Sprint("--file") + ", or " + ui.Flag.Sprint("--pubkey") + " must be specified.\n" +
				"Run " + ui.Code.Sprint("kanuka secrets register --help") + " to see the available commands"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// When using --pubkey, user email is required
		if publicKeyText != "" && registerUserEmail == "" {
			finalMessage := ui.Error.Sprint("✗") + " When using " + ui.Flag.Sprint("--pubkey") + ", the " + ui.Flag.Sprint("--user") + " flag is required.\n" +
				"Specify a user email with " + ui.Flag.Sprint("--user")
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Validate email format if provided
		if registerUserEmail != "" && !utils.IsValidEmail(registerUserEmail) {
			finalMessage := ui.Error.Sprint("✗") + " Invalid email format: " + ui.Highlight.Sprint(registerUserEmail) + "\n" +
				ui.Info.Sprint("→") + " Please provide a valid email address"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Check if pubkey flag was explicitly used but with empty content
		// Only validate pubkey emptiness if we're in the pubkey text registration path
		if publicKeyText != "" {
			// We're already in the pubkey path, so this validation is handled below
		} else if cmd.Flags().Changed("pubkey") {
			// The pubkey flag was explicitly set but is empty
			finalMessage := ui.Error.Sprint("✗") + " Invalid public key format provided\n" +
				ui.Error.Sprint("Error: ") + "public key text cannot be empty"
			spinner.FinalMSG = finalMessage
			return nil
		}

		Logger.Debugf("Initializing project settings")
		if err := configs.InitProjectSettings(); err != nil {
			return Logger.ErrorfAndReturn("failed to init project settings: %v", err)
		}

		// If --private-key-stdin is set, read the key data now (before it gets consumed elsewhere)
		if registerPrivateKeyStdin {
			Logger.Debugf("Reading private key from stdin")
			keyData, err := utils.ReadStdin()
			if err != nil {
				Logger.Errorf("Failed to read private key from stdin: %v", err)
				finalMessage := ui.Error.Sprint("✗") + " Failed to read private key from stdin\n" +
					ui.Error.Sprint("Error: ") + err.Error()
				spinner.FinalMSG = finalMessage
				return nil
			}
			registerPrivateKeyData = keyData
			Logger.Infof("Private key data read from stdin (%d bytes)", len(keyData))
		}

		switch {
		case publicKeyText != "":
			Logger.Infof("Handling public key text registration for user: %s", registerUserEmail)
			return handlePubkeyTextRegistration(spinner)
		case customFilePath != "":
			Logger.Infof("Handling custom file registration from: %s", customFilePath)
			return handleCustomFileRegistration(spinner)
		default:
			Logger.Infof("Handling user registration for: %s", registerUserEmail)
			return handleUserRegistration(spinner)
		}
	},
}

func handlePubkeyTextRegistration(spinner *spinner.Spinner) error {
	projectPath := configs.ProjectKanukaSettings.ProjectPath
	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath
	Logger.Debugf("Project path: %s, Public key path: %s", projectPath, projectPublicKeyPath)

	if projectPath == "" {
		finalMessage := ui.Error.Sprint("✗") + " Kānuka has not been initialized\n" +
			ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " instead"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Load project config to look up user UUID by email
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		return Logger.ErrorfAndReturn("Failed to load project config: %v", err)
	}

	// Look up user UUID by email
	targetUserUUID, found := projectConfig.GetUserUUIDByEmail(registerUserEmail)
	if !found {
		finalMessage := ui.Error.Sprint("✗") + " User " + ui.Highlight.Sprint(registerUserEmail) + " not found in project\n" +
			"They must first run: " + ui.Code.Sprint("kanuka secrets create --email "+registerUserEmail)
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Validate and parse the public key text
	Logger.Debugf("Parsing public key text for user: %s (UUID: %s)", registerUserEmail, targetUserUUID)
	publicKey, err := secrets.ParsePublicKeyText(publicKeyText)
	if err != nil {
		Logger.Errorf("Invalid public key format provided: %v", err)
		finalMessage := ui.Error.Sprint("✗") + " Invalid public key format provided\n" +
			ui.Error.Sprint("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}
	Logger.Infof("Public key parsed successfully")

	// Compute paths for output
	pubKeyFilePath := filepath.Join(projectPublicKeyPath, targetUserUUID+".pub")
	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath
	kanukaFilePath := filepath.Join(projectSecretsPath, targetUserUUID+".kanuka")

	// Validate current user has access to decrypt symmetric key before making changes
	userConfig, err := configs.EnsureUserConfig()
	if err != nil {
		return Logger.ErrorfAndReturn("Failed to ensure user config: %v", err)
	}
	currentUserUUID := userConfig.User.UUID
	projectUUID := projectConfig.Project.UUID

	encryptedSymKey, err := secrets.GetProjectKanukaKey(currentUserUUID)
	if err != nil {
		currentKanukaKeyPath := filepath.Join(projectSecretsPath, currentUserUUID+".kanuka")
		finalMessage := ui.Error.Sprint("✗") + " Couldn't get your Kānuka key from " + ui.Path.Sprint(currentKanukaKeyPath) + "\n\n" +
			"Are you sure you have access?\n\n" +
			ui.Error.Sprint("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}

	privateKeyPath := configs.GetPrivateKeyPath(projectUUID)
	privateKey, err := loadRegisterPrivateKey(projectUUID)
	if err != nil {
		errorSource := "from " + ui.Path.Sprint(privateKeyPath)
		if registerPrivateKeyStdin {
			errorSource = "from stdin"
		}
		finalMessage := ui.Error.Sprint("✗") + " Couldn't get your private key " + errorSource + "\n\n" +
			"Are you sure you have access?\n\n" +
			ui.Error.Sprint("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Validate decryption works
	_, err = secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		currentKanukaKeyPath := filepath.Join(projectSecretsPath, currentUserUUID+".kanuka")
		finalMessage := ui.Error.Sprint("✗") + " Failed to decrypt your Kānuka key using your private key: \n" +
			"    Kānuka key path: " + ui.Path.Sprint(currentKanukaKeyPath) + "\n" +
			"    Private key path: " + ui.Path.Sprint(privateKeyPath) + "\n\n" +
			"Are you sure you have access?\n\n" +
			ui.Error.Sprint("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Check if user already has access (both public key AND .kanuka file exist)
	userAlreadyHasAccess := fileExists(pubKeyFilePath) && fileExists(kanukaFilePath)
	Logger.Debugf("User already has access: %t (pubkey: %s, kanuka: %s)", userAlreadyHasAccess, pubKeyFilePath, kanukaFilePath)

	// If user already has access and not forced, prompt for confirmation
	if userAlreadyHasAccess && !registerForce && !registerDryRun {
		if !confirmRegisterOverwrite(spinner, registerUserEmail) {
			spinner.FinalMSG = ui.Warning.Sprint("⚠") + " Registration cancelled.\n"
			return nil
		}
	}

	// If dry-run, print preview and exit early
	if registerDryRun {
		printRegisterDryRun(spinner, registerUserEmail, pubKeyFilePath, kanukaFilePath, true)
		return nil
	}

	// Save the public key to a file using user UUID
	Logger.Debugf("Saving public key to: %s", pubKeyFilePath)
	if err := secrets.SavePublicKeyToFile(publicKey, pubKeyFilePath); err != nil {
		Logger.Errorf("Failed to save public key to %s: %v", pubKeyFilePath, err)
		finalMessage := ui.Error.Sprint("✗") + " Failed to save public key to " + ui.Path.Sprint(pubKeyFilePath) + "\n" +
			ui.Error.Sprint("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}
	Logger.Infof("Public key saved successfully")

	// Now register the user with the newly saved public key
	Logger.Debugf("Registering user %s (UUID: %s) with public key", registerUserEmail, targetUserUUID)
	if err := registerUserWithPublicKey(targetUserUUID, publicKey); err != nil {
		Logger.Errorf("Failed to register user %s with public key: %v", registerUserEmail, err)
		finalMessage := ui.Error.Sprint("✗") + " Failed to register user with the provided public key\n" +
			ui.Error.Sprint("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}

	Logger.Infof("Public key registration completed successfully for user: %s", registerUserEmail)

	// Log to audit trail.
	auditEntry := audit.LogWithUser("register")
	auditEntry.TargetUser = registerUserEmail
	auditEntry.TargetUUID = targetUserUUID
	audit.Log(auditEntry)

	// Use different message for update vs new registration
	var successVerb, filesLabel string
	if userAlreadyHasAccess {
		successVerb = "access has been updated"
		filesLabel = "Files updated"
	} else {
		successVerb = "has been granted access"
		filesLabel = "Files created"
	}

	finalMessage := ui.Success.Sprint("✓") + " " + ui.Highlight.Sprint(registerUserEmail) + " " + successVerb + " successfully!\n\n" +
		filesLabel + ":\n" +
		"  Public key:    " + ui.Path.Sprint(pubKeyFilePath) + "\n" +
		"  Encrypted key: " + ui.Path.Sprint(kanukaFilePath) + "\n\n" +
		ui.Info.Sprint("→") + " They now have access to decrypt the repository's secrets"
	spinner.FinalMSG = finalMessage
	return nil
}

func registerUserWithPublicKey(targetUserUUID string, targetPublicKey *rsa.PublicKey) error {
	// Get current user's UUID
	userConfig, err := configs.EnsureUserConfig()
	if err != nil {
		return err
	}
	currentUserUUID := userConfig.User.UUID

	// Load project config to get project UUID
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		return err
	}
	projectUUID := projectConfig.Project.UUID

	Logger.Debugf("Registering user %s with current user %s, project %s", targetUserUUID, currentUserUUID, projectUUID)

	// Get the current user's encrypted symmetric key using their UUID
	Logger.Debugf("Getting project kanuka key for current user: %s", currentUserUUID)
	encryptedSymKey, err := secrets.GetProjectKanukaKey(currentUserUUID)
	if err != nil {
		Logger.Errorf("Failed to get project kanuka key for user %s: %v", currentUserUUID, err)
		return err
	}

	// Get current user's private key using project UUID
	privateKeyPath := configs.GetPrivateKeyPath(projectUUID)
	Logger.Debugf("Loading private key from: %s", privateKeyPath)
	privateKey, err := loadRegisterPrivateKey(projectUUID)
	if err != nil {
		Logger.Errorf("Failed to load private key: %v", err)
		return err
	}

	// Decrypt symmetric key with current user's private key
	Logger.Debugf("Decrypting symmetric key with current user's private key")
	symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		Logger.Errorf("Failed to decrypt symmetric key: %v", err)
		return err
	}

	// Encrypt symmetric key with target user's public key
	Logger.Debugf("Encrypting symmetric key with target user's public key")
	targetEncryptedSymKey, err := secrets.EncryptWithPublicKey(symKey, targetPublicKey)
	if err != nil {
		Logger.Errorf("Failed to encrypt symmetric key with target user's public key: %v", err)
		return err
	}

	// Save encrypted symmetric key for target user using their UUID
	Logger.Debugf("Saving kanuka key for target user: %s", targetUserUUID)
	if err := secrets.SaveKanukaKeyToProject(targetUserUUID, targetEncryptedSymKey); err != nil {
		Logger.Errorf("Failed to save kanuka key for target user %s: %v", targetUserUUID, err)
		return err
	}

	Logger.Infof("Successfully registered user %s with public key", targetUserUUID)
	return nil
}

func handleUserRegistration(spinner *spinner.Spinner) error {
	projectPath := configs.ProjectKanukaSettings.ProjectPath
	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath

	// Get current user's UUID
	userConfig, err := configs.EnsureUserConfig()
	if err != nil {
		return Logger.ErrorfAndReturn("Failed to ensure user config: %v", err)
	}
	currentUserUUID := userConfig.User.UUID

	// Load project config to get project UUID and look up target user
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		return Logger.ErrorfAndReturn("Failed to load project config: %v", err)
	}
	projectUUID := projectConfig.Project.UUID

	Logger.Debugf("Current user UUID: %s, Project UUID: %s, Project path: %s", currentUserUUID, projectUUID, projectPath)

	if projectPath == "" {
		finalMessage := ui.Error.Sprint("✗") + " Kānuka has not been initialized\n" +
			ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " instead"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Look up user UUID by email
	targetUserUUID, found := projectConfig.GetUserUUIDByEmail(registerUserEmail)
	if !found {
		finalMessage := ui.Error.Sprint("✗") + " User " + ui.Highlight.Sprint(registerUserEmail) + " not found in project\n" +
			"They must first run: " + ui.Code.Sprint("kanuka secrets create --email "+registerUserEmail)
		spinner.FinalMSG = finalMessage
		return nil
	}

	Logger.Debugf("Found target user UUID: %s for email: %s", targetUserUUID, registerUserEmail)

	// Check if target user's public key exists (using their UUID)
	targetPubkeyPath := filepath.Join(projectPublicKeyPath, targetUserUUID+".pub")
	Logger.Debugf("Looking for target user's public key at: %s", targetPubkeyPath)

	// TODO: In the future, differentiate between FileNotFound Error and InvalidKey Error
	targetUserPublicKey, err := secrets.LoadPublicKey(targetPubkeyPath)
	if err != nil {
		Logger.Errorf("Failed to load public key for user %s from %s: %v", registerUserEmail, targetPubkeyPath, err)
		finalMessage := ui.Error.Sprint("✗") + " Public key for user " + ui.Highlight.Sprint(registerUserEmail) + " not found\n" +
			"They must first run: " + ui.Code.Sprint("kanuka secrets create --email "+registerUserEmail)
		spinner.FinalMSG = finalMessage
		return nil
	}
	Logger.Infof("Target user's public key loaded successfully")

	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath
	kanukaKeyPath := filepath.Join(projectSecretsPath, currentUserUUID+".kanuka")
	Logger.Debugf("Getting kanuka key from: %s", kanukaKeyPath)

	encryptedSymKey, err := secrets.GetProjectKanukaKey(currentUserUUID)
	if err != nil {
		Logger.Errorf("Failed to get kanuka key for current user %s: %v", currentUserUUID, err)
		finalMessage := ui.Error.Sprint("✗") + " Couldn't get your Kānuka key from " + ui.Path.Sprint(kanukaKeyPath) + "\n\n" +
			"Are you sure you have access?\n\n" +
			ui.Error.Sprint("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Get current user's private key using project UUID
	privateKeyPath := configs.GetPrivateKeyPath(projectUUID)
	Logger.Debugf("Loading private key from: %s", privateKeyPath)

	privateKey, err := loadRegisterPrivateKey(projectUUID)
	if err != nil {
		Logger.Errorf("Failed to load private key: %v", err)
		errorSource := "from " + ui.Path.Sprint(privateKeyPath)
		if registerPrivateKeyStdin {
			errorSource = "from stdin"
		}
		finalMessage := ui.Error.Sprint("✗") + " Couldn't get your private key " + errorSource + "\n\n" +
			"Are you sure you have access?\n\n" +
			ui.Error.Sprint("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Decrypt symmetric key with current user's private key
	Logger.Debugf("Decrypting symmetric key with current user's private key")
	_, err = secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		Logger.Errorf("Failed to decrypt symmetric key: %v", err)
		finalMessage := ui.Error.Sprint("✗") + " Failed to decrypt your Kānuka key using your private key: \n" +
			"    Kānuka key path: " + ui.Path.Sprint(kanukaKeyPath) + "\n" +
			"    Private key path: " + ui.Path.Sprint(privateKeyPath) + "\n\n" +
			"Are you sure you have access?\n\n" +
			ui.Error.Sprint("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Compute path for output
	targetKanukaFilePath := filepath.Join(projectSecretsPath, targetUserUUID+".kanuka")

	// Check if user already has access (both public key AND .kanuka file exist)
	userAlreadyHasAccess := fileExists(targetPubkeyPath) && fileExists(targetKanukaFilePath)
	Logger.Debugf("User already has access: %t (pubkey: %s, kanuka: %s)", userAlreadyHasAccess, targetPubkeyPath, targetKanukaFilePath)

	// If user already has access and not forced, prompt for confirmation
	if userAlreadyHasAccess && !registerForce && !registerDryRun {
		if !confirmRegisterOverwrite(spinner, registerUserEmail) {
			spinner.FinalMSG = ui.Warning.Sprint("⚠") + " Registration cancelled.\n"
			return nil
		}
	}

	// If dry-run, print preview and exit early
	if registerDryRun {
		printRegisterDryRun(spinner, registerUserEmail, targetPubkeyPath, targetKanukaFilePath, false)
		return nil
	}

	// Re-decrypt symmetric key for actual use (we verified it works above)
	symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		return Logger.ErrorfAndReturn("Failed to decrypt symmetric key: %v", err)
	}

	// Encrypt symmetric key with target user's public key
	Logger.Debugf("Encrypting symmetric key with target user's public key")
	targetEncryptedSymKey, err := secrets.EncryptWithPublicKey(symKey, targetUserPublicKey)
	if err != nil {
		return Logger.ErrorfAndReturn("Failed to encrypt symmetric key for target user: %v", err)
	}

	// Save encrypted symmetric key for target user using their UUID
	Logger.Debugf("Saving kanuka key for target user: %s (UUID: %s)", registerUserEmail, targetUserUUID)
	if err := secrets.SaveKanukaKeyToProject(targetUserUUID, targetEncryptedSymKey); err != nil {
		return Logger.ErrorfAndReturn("Failed to save encrypted key for target user: %v", err)
	}

	Logger.Infof("User registration completed successfully for: %s", registerUserEmail)

	// Log to audit trail.
	auditEntry := audit.LogWithUser("register")
	auditEntry.TargetUser = registerUserEmail
	auditEntry.TargetUUID = targetUserUUID
	audit.Log(auditEntry)

	// Use different message for update vs new registration
	var successVerb, filesLabel string
	if userAlreadyHasAccess {
		successVerb = "access has been updated"
		filesLabel = "Files updated"
	} else {
		successVerb = "has been granted access"
		filesLabel = "Files created"
	}

	finalMessage := ui.Success.Sprint("✓") + " " + ui.Highlight.Sprint(registerUserEmail) + " " + successVerb + " successfully!\n\n" +
		filesLabel + ":\n" +
		"  Public key:    " + ui.Path.Sprint(targetPubkeyPath) + "\n" +
		"  Encrypted key: " + ui.Path.Sprint(targetKanukaFilePath) + "\n\n" +
		ui.Info.Sprint("→") + " They now have access to decrypt the repository's secrets"
	spinner.FinalMSG = finalMessage
	return nil
}

func handleCustomFileRegistration(spinner *spinner.Spinner) error {
	projectPath := configs.ProjectKanukaSettings.ProjectPath
	Logger.Debugf("Custom file path: %s", customFilePath)

	// Get current user's UUID
	userConfig, err := configs.EnsureUserConfig()
	if err != nil {
		return Logger.ErrorfAndReturn("Failed to ensure user config: %v", err)
	}
	currentUserUUID := userConfig.User.UUID

	// Load project config to get project UUID
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		return Logger.ErrorfAndReturn("Failed to load project config: %v", err)
	}
	projectUUID := projectConfig.Project.UUID

	Logger.Debugf("Current user UUID: %s, Project UUID: %s", currentUserUUID, projectUUID)

	if projectPath == "" {
		finalMessage := ui.Error.Sprint("✗") + " Kānuka has not been initialized\n" +
			ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " instead"
		spinner.FinalMSG = finalMessage
		return nil
	}

	if !strings.HasSuffix(customFilePath, ".pub") {
		finalMessage := ui.Error.Sprint("✗ ") + ui.Path.Sprint(customFilePath) + " is not a valid path to a public key file"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Load the custom public key
	Logger.Debugf("Loading public key from custom file: %s", customFilePath)
	targetUserPublicKey, err := secrets.LoadPublicKey(customFilePath)
	if err != nil {
		Logger.Errorf("Failed to load public key from %s: %v", customFilePath, err)
		finalMessage := ui.Error.Sprint("✗") + " Public key could not be loaded from " + ui.Path.Sprint(customFilePath) + "\n\n" +
			ui.Error.Sprint("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}
	Logger.Infof("Public key loaded successfully from custom file")

	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath
	kanukaKeyPath := filepath.Join(projectSecretsPath, currentUserUUID+".kanuka")

	encryptedSymKey, err := secrets.GetProjectKanukaKey(currentUserUUID)
	if err != nil {
		finalMessage := ui.Error.Sprint("✗") + " Couldn't get your Kānuka key from " + ui.Path.Sprint(kanukaKeyPath) + "\n\n" +
			"Are you sure you have access?\n\n" +
			ui.Error.Sprint("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Get current user's private key using project UUID
	privateKeyPath := configs.GetPrivateKeyPath(projectUUID)

	privateKey, err := loadRegisterPrivateKey(projectUUID)
	if err != nil {
		errorSource := "from " + ui.Path.Sprint(privateKeyPath)
		if registerPrivateKeyStdin {
			errorSource = "from stdin"
		}
		finalMessage := ui.Error.Sprint("✗") + " Couldn't get your private key " + errorSource + "\n\n" +
			"Are you sure you have access?\n\n" +
			ui.Error.Sprint("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Decrypt symmetric key with current user's private key
	_, err = secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		finalMessage := ui.Error.Sprint("✗") + " Failed to decrypt your Kānuka key using your private key: \n" +
			"    Kānuka key path: " + ui.Path.Sprint(kanukaKeyPath) + "\n" +
			"    Private key path: " + ui.Path.Sprint(privateKeyPath) + "\n\n" +
			"Are you sure you have access?\n\n" +
			ui.Error.Sprint("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}

	// The target user UUID is the filename without .pub extension
	targetUserUUID := strings.TrimSuffix(filepath.Base(customFilePath), ".pub")

	// Try to find email for display purposes
	targetEmail := projectConfig.Users[targetUserUUID]
	displayName := targetEmail
	if displayName == "" {
		displayName = targetUserUUID
	}

	// Compute path for output
	targetKanukaFilePath := filepath.Join(projectSecretsPath, targetUserUUID+".kanuka")

	// Check if user already has access (both public key AND .kanuka file exist)
	// For custom file, we check the custom file path and target kanuka path
	userAlreadyHasAccess := fileExists(customFilePath) && fileExists(targetKanukaFilePath)
	Logger.Debugf("User already has access: %t (pubkey: %s, kanuka: %s)", userAlreadyHasAccess, customFilePath, targetKanukaFilePath)

	// If user already has access and not forced, prompt for confirmation
	if userAlreadyHasAccess && !registerForce && !registerDryRun {
		if !confirmRegisterOverwrite(spinner, displayName) {
			spinner.FinalMSG = ui.Warning.Sprint("⚠") + " Registration cancelled.\n"
			return nil
		}
	}

	// If dry-run, print preview and exit early
	if registerDryRun {
		printRegisterDryRunForFile(spinner, displayName, customFilePath, targetKanukaFilePath)
		return nil
	}

	// Re-decrypt symmetric key for actual use (we verified it works above)
	symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		return Logger.ErrorfAndReturn("Failed to decrypt symmetric key: %v", err)
	}

	// Encrypt symmetric key with target user's public key
	targetEncryptedSymKey, err := secrets.EncryptWithPublicKey(symKey, targetUserPublicKey)
	if err != nil {
		return Logger.ErrorfAndReturn("Failed to encrypt symmetric key for target user: %v", err)
	}

	Logger.Debugf("Saving kanuka key for target user: %s (from custom file)", targetUserUUID)
	if err := secrets.SaveKanukaKeyToProject(targetUserUUID, targetEncryptedSymKey); err != nil {
		return Logger.ErrorfAndReturn("Failed to save encrypted key for target user: %v", err)
	}

	Logger.Infof("Custom file registration completed successfully for: %s", displayName)

	// Log to audit trail.
	auditEntry := audit.LogWithUser("register")
	auditEntry.TargetUser = displayName
	auditEntry.TargetUUID = targetUserUUID
	audit.Log(auditEntry)

	// Use different message for update vs new registration
	var successVerb, filesLabel string
	if userAlreadyHasAccess {
		successVerb = "access has been updated"
		filesLabel = "Files updated"
	} else {
		successVerb = "has been granted access"
		filesLabel = "Files created"
	}

	finalMessage := ui.Success.Sprint("✓") + " " + ui.Highlight.Sprint(displayName) + " " + successVerb + " successfully!\n\n" +
		filesLabel + ":\n" +
		"  Public key:    " + ui.Path.Sprint(customFilePath) + " (provided)\n" +
		"  Encrypted key: " + ui.Path.Sprint(targetKanukaFilePath) + "\n\n" +
		ui.Info.Sprint("→") + " They now have access to decrypt the repository's secrets"
	spinner.FinalMSG = finalMessage
	return nil
}

// printRegisterDryRun prints a preview of what would be created during registration.
func printRegisterDryRun(spinner *spinner.Spinner, displayName, pubKeyPath, kanukaPath string, pubKeyWouldBeCreated bool) {
	spinner.Stop()

	fmt.Println(ui.Warning.Sprint("[dry-run]") + " Would register " + ui.Highlight.Sprint(displayName))
	fmt.Println()

	fmt.Println("Files that would be created:")
	if pubKeyWouldBeCreated {
		fmt.Println("  - " + ui.Success.Sprint(pubKeyPath))
	}
	fmt.Println("  - " + ui.Success.Sprint(kanukaPath))
	fmt.Println()

	fmt.Println("Prerequisites verified:")
	fmt.Println("  " + ui.Success.Sprint("✓") + " User exists in project config")
	fmt.Println("  " + ui.Success.Sprint("✓") + " Public key found at " + pubKeyPath)
	fmt.Println("  " + ui.Success.Sprint("✓") + " Current user has access to decrypt symmetric key")
	fmt.Println()

	fmt.Println(ui.Info.Sprint("No changes made.") + " Run without --dry-run to execute.")
}

// printRegisterDryRunForFile prints a preview for --file registration mode.
func printRegisterDryRunForFile(spinner *spinner.Spinner, displayName, pubKeyPath, kanukaPath string) {
	spinner.Stop()

	fmt.Println(ui.Warning.Sprint("[dry-run]") + " Would register " + ui.Highlight.Sprint(displayName))
	fmt.Println()

	fmt.Println("Files that would be created:")
	fmt.Println("  - " + ui.Success.Sprint(kanukaPath))
	fmt.Println()

	fmt.Println("Prerequisites verified:")
	fmt.Println("  " + ui.Success.Sprint("✓") + " Public key loaded from " + pubKeyPath)
	fmt.Println("  " + ui.Success.Sprint("✓") + " Current user has access to decrypt symmetric key")
	fmt.Println()

	fmt.Println(ui.Info.Sprint("No changes made.") + " Run without --dry-run to execute.")
}
