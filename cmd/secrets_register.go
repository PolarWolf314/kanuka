package cmd

import (
	"crypto/rsa"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/secrets"
	"github.com/PolarWolf314/kanuka/internal/utils"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	registerUserEmail       string
	customFilePath          string
	publicKeyText           string
	registerDryRun          bool
	registerPrivateKeyStdin bool
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

func init() {
	RegisterCmd.Flags().StringVarP(&registerUserEmail, "user", "u", "", "user email to register for access")
	RegisterCmd.Flags().StringVarP(&customFilePath, "file", "f", "", "the path to a custom public key — will add public key to the project")
	RegisterCmd.Flags().StringVar(&publicKeyText, "pubkey", "", "OpenSSH or PEM public key content to be saved with the specified user email")
	RegisterCmd.Flags().BoolVar(&registerDryRun, "dry-run", false, "preview registration without making changes")
	RegisterCmd.Flags().BoolVar(&registerPrivateKeyStdin, "private-key-stdin", false, "read private key from stdin instead of from disk")
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
			finalMessage := color.RedString("✗") + " Either " + color.YellowString("--user") + ", " + color.YellowString("--file") + ", or " + color.YellowString("--pubkey") + " must be specified.\n" +
				"Run " + color.YellowString("kanuka secrets register --help") + " to see the available commands"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// When using --pubkey, user email is required
		if publicKeyText != "" && registerUserEmail == "" {
			finalMessage := color.RedString("✗") + " When using " + color.YellowString("--pubkey") + ", the " + color.YellowString("--user") + " flag is required.\n" +
				"Specify a user email with " + color.YellowString("--user")
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Validate email format if provided
		if registerUserEmail != "" && !utils.IsValidEmail(registerUserEmail) {
			finalMessage := color.RedString("✗") + " Invalid email format: " + color.YellowString(registerUserEmail) + "\n" +
				color.CyanString("→") + " Please provide a valid email address"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Check if pubkey flag was explicitly used but with empty content
		// Only validate pubkey emptiness if we're in the pubkey text registration path
		if publicKeyText != "" {
			// We're already in the pubkey path, so this validation is handled below
		} else if cmd.Flags().Changed("pubkey") {
			// The pubkey flag was explicitly set but is empty
			finalMessage := color.RedString("✗") + " Invalid public key format provided\n" +
				color.RedString("Error: ") + "public key text cannot be empty"
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
				finalMessage := color.RedString("✗") + " Failed to read private key from stdin\n" +
					color.RedString("Error: ") + err.Error()
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
		finalMessage := color.RedString("✗") + " Kānuka has not been initialized\n" +
			color.CyanString("→") + " Run " + color.YellowString("kanuka secrets init") + " instead"
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
		finalMessage := color.RedString("✗") + " User " + color.YellowString(registerUserEmail) + " not found in project\n" +
			"They must first run: " + color.YellowString("kanuka secrets create --email "+registerUserEmail)
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Validate and parse the public key text
	Logger.Debugf("Parsing public key text for user: %s (UUID: %s)", registerUserEmail, targetUserUUID)
	publicKey, err := secrets.ParsePublicKeyText(publicKeyText)
	if err != nil {
		Logger.Errorf("Invalid public key format provided: %v", err)
		finalMessage := color.RedString("✗") + " Invalid public key format provided\n" +
			color.RedString("Error: ") + err.Error()
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
		finalMessage := color.RedString("✗") + " Couldn't get your Kānuka key from " + color.YellowString(currentKanukaKeyPath) + "\n\n" +
			"Are you sure you have access?\n\n" +
			color.RedString("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}

	privateKeyPath := configs.GetPrivateKeyPath(projectUUID)
	privateKey, err := loadRegisterPrivateKey(projectUUID)
	if err != nil {
		errorSource := "from " + color.YellowString(privateKeyPath)
		if registerPrivateKeyStdin {
			errorSource = "from stdin"
		}
		finalMessage := color.RedString("✗") + " Couldn't get your private key " + errorSource + "\n\n" +
			"Are you sure you have access?\n\n" +
			color.RedString("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Validate decryption works
	_, err = secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		currentKanukaKeyPath := filepath.Join(projectSecretsPath, currentUserUUID+".kanuka")
		finalMessage := color.RedString("✗") + " Failed to decrypt your Kānuka key using your private key: \n" +
			"    Kānuka key path: " + color.YellowString(currentKanukaKeyPath) + "\n" +
			"    Private key path: " + color.YellowString(privateKeyPath) + "\n\n" +
			"Are you sure you have access?\n\n" +
			color.RedString("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
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
		finalMessage := color.RedString("✗") + " Failed to save public key to " + color.YellowString(pubKeyFilePath) + "\n" +
			color.RedString("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}
	Logger.Infof("Public key saved successfully")

	// Now register the user with the newly saved public key
	Logger.Debugf("Registering user %s (UUID: %s) with public key", registerUserEmail, targetUserUUID)
	if err := registerUserWithPublicKey(targetUserUUID, publicKey); err != nil {
		Logger.Errorf("Failed to register user %s with public key: %v", registerUserEmail, err)
		finalMessage := color.RedString("✗") + " Failed to register user with the provided public key\n" +
			color.RedString("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}

	Logger.Infof("Public key registration completed successfully for user: %s", registerUserEmail)
	finalMessage := color.GreenString("✓") + " " + color.YellowString(registerUserEmail) + " has been granted access successfully!\n\n" +
		"Files created:\n" +
		"  Public key:    " + color.CyanString(pubKeyFilePath) + "\n" +
		"  Encrypted key: " + color.CyanString(kanukaFilePath) + "\n\n" +
		color.CyanString("→") + " They now have access to decrypt the repository's secrets"
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
		finalMessage := color.RedString("✗") + " Kānuka has not been initialized\n" +
			color.CyanString("→") + " Run " + color.YellowString("kanuka secrets init") + " instead"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Look up user UUID by email
	targetUserUUID, found := projectConfig.GetUserUUIDByEmail(registerUserEmail)
	if !found {
		finalMessage := color.RedString("✗") + " User " + color.YellowString(registerUserEmail) + " not found in project\n" +
			"They must first run: " + color.YellowString("kanuka secrets create --email "+registerUserEmail)
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
		finalMessage := color.RedString("✗") + " Public key for user " + color.YellowString(registerUserEmail) + " not found\n" +
			"They must first run: " + color.YellowString("kanuka secrets create --email "+registerUserEmail)
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
		finalMessage := color.RedString("✗") + " Couldn't get your Kānuka key from " + color.YellowString(kanukaKeyPath) + "\n\n" +
			"Are you sure you have access?\n\n" +
			color.RedString("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Get current user's private key using project UUID
	privateKeyPath := configs.GetPrivateKeyPath(projectUUID)
	Logger.Debugf("Loading private key from: %s", privateKeyPath)

	privateKey, err := loadRegisterPrivateKey(projectUUID)
	if err != nil {
		Logger.Errorf("Failed to load private key: %v", err)
		errorSource := "from " + color.YellowString(privateKeyPath)
		if registerPrivateKeyStdin {
			errorSource = "from stdin"
		}
		finalMessage := color.RedString("✗") + " Couldn't get your private key " + errorSource + "\n\n" +
			"Are you sure you have access?\n\n" +
			color.RedString("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Decrypt symmetric key with current user's private key
	Logger.Debugf("Decrypting symmetric key with current user's private key")
	_, err = secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		Logger.Errorf("Failed to decrypt symmetric key: %v", err)
		finalMessage := color.RedString("✗") + " Failed to decrypt your Kānuka key using your private key: \n" +
			"    Kānuka key path: " + color.YellowString(kanukaKeyPath) + "\n" +
			"    Private key path: " + color.YellowString(privateKeyPath) + "\n\n" +
			"Are you sure you have access?\n\n" +
			color.RedString("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Compute path for output
	targetKanukaFilePath := filepath.Join(projectSecretsPath, targetUserUUID+".kanuka")

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
	finalMessage := color.GreenString("✓") + " " + color.YellowString(registerUserEmail) + " has been granted access successfully!\n\n" +
		"Files created:\n" +
		"  Public key:    " + color.CyanString(targetPubkeyPath) + "\n" +
		"  Encrypted key: " + color.CyanString(targetKanukaFilePath) + "\n\n" +
		color.CyanString("→") + " They now have access to decrypt the repository's secrets"
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
		finalMessage := color.RedString("✗") + " Kānuka has not been initialized\n" +
			color.CyanString("→") + " Run " + color.YellowString("kanuka secrets init") + " instead"
		spinner.FinalMSG = finalMessage
		return nil
	}

	if !strings.HasSuffix(customFilePath, ".pub") {
		finalMessage := color.RedString("✗ ") + color.YellowString(customFilePath) + " is not a valid path to a public key file"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Load the custom public key
	Logger.Debugf("Loading public key from custom file: %s", customFilePath)
	targetUserPublicKey, err := secrets.LoadPublicKey(customFilePath)
	if err != nil {
		Logger.Errorf("Failed to load public key from %s: %v", customFilePath, err)
		finalMessage := color.RedString("✗") + " Public key could not be loaded from " + color.YellowString(customFilePath) + "\n\n" +
			color.RedString("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}
	Logger.Infof("Public key loaded successfully from custom file")

	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath
	kanukaKeyPath := filepath.Join(projectSecretsPath, currentUserUUID+".kanuka")

	encryptedSymKey, err := secrets.GetProjectKanukaKey(currentUserUUID)
	if err != nil {
		finalMessage := color.RedString("✗") + " Couldn't get your Kānuka key from " + color.YellowString(kanukaKeyPath) + "\n\n" +
			"Are you sure you have access?\n\n" +
			color.RedString("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Get current user's private key using project UUID
	privateKeyPath := configs.GetPrivateKeyPath(projectUUID)

	privateKey, err := loadRegisterPrivateKey(projectUUID)
	if err != nil {
		errorSource := "from " + color.YellowString(privateKeyPath)
		if registerPrivateKeyStdin {
			errorSource = "from stdin"
		}
		finalMessage := color.RedString("✗") + " Couldn't get your private key " + errorSource + "\n\n" +
			"Are you sure you have access?\n\n" +
			color.RedString("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Decrypt symmetric key with current user's private key
	_, err = secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		finalMessage := color.RedString("✗") + " Failed to decrypt your Kānuka key using your private key: \n" +
			"    Kānuka key path: " + color.YellowString(kanukaKeyPath) + "\n" +
			"    Private key path: " + color.YellowString(privateKeyPath) + "\n\n" +
			"Are you sure you have access?\n\n" +
			color.RedString("Error: ") + err.Error()
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
	finalMessage := color.GreenString("✓") + " " + color.YellowString(displayName) + " has been granted access successfully!\n\n" +
		"Files created:\n" +
		"  Public key:    " + color.CyanString(customFilePath) + " (provided)\n" +
		"  Encrypted key: " + color.CyanString(targetKanukaFilePath) + "\n\n" +
		color.CyanString("→") + " They now have access to decrypt the repository's secrets"
	spinner.FinalMSG = finalMessage
	return nil
}

// printRegisterDryRun prints a preview of what would be created during registration.
func printRegisterDryRun(spinner *spinner.Spinner, displayName, pubKeyPath, kanukaPath string, pubKeyWouldBeCreated bool) {
	spinner.Stop()

	fmt.Println(color.YellowString("[dry-run]") + " Would register " + color.CyanString(displayName))
	fmt.Println()

	fmt.Println("Files that would be created:")
	if pubKeyWouldBeCreated {
		fmt.Println("  - " + color.GreenString(pubKeyPath))
	}
	fmt.Println("  - " + color.GreenString(kanukaPath))
	fmt.Println()

	fmt.Println("Prerequisites verified:")
	fmt.Println("  " + color.GreenString("✓") + " User exists in project config")
	fmt.Println("  " + color.GreenString("✓") + " Public key found at " + pubKeyPath)
	fmt.Println("  " + color.GreenString("✓") + " Current user has access to decrypt symmetric key")
	fmt.Println()

	fmt.Println(color.CyanString("No changes made.") + " Run without --dry-run to execute.")
}

// printRegisterDryRunForFile prints a preview for --file registration mode.
func printRegisterDryRunForFile(spinner *spinner.Spinner, displayName, pubKeyPath, kanukaPath string) {
	spinner.Stop()

	fmt.Println(color.YellowString("[dry-run]") + " Would register " + color.CyanString(displayName))
	fmt.Println()

	fmt.Println("Files that would be created:")
	fmt.Println("  - " + color.GreenString(kanukaPath))
	fmt.Println()

	fmt.Println("Prerequisites verified:")
	fmt.Println("  " + color.GreenString("✓") + " Public key loaded from " + pubKeyPath)
	fmt.Println("  " + color.GreenString("✓") + " Current user has access to decrypt symmetric key")
	fmt.Println()

	fmt.Println(color.CyanString("No changes made.") + " Run without --dry-run to execute.")
}
