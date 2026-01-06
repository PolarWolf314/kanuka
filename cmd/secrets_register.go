package cmd

import (
	"crypto/rsa"
	"path/filepath"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/secrets"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	username       string
	customFilePath string
	publicKeyText  string
)

// resetRegisterCommandState resets all register command global variables to their default values for testing.
func resetRegisterCommandState() {
	username = ""
	customFilePath = ""
	publicKeyText = ""
}

func init() {
	RegisterCmd.Flags().StringVarP(&username, "user", "u", "", "user UUID to register for access (from their public key filename)")
	RegisterCmd.Flags().StringVarP(&customFilePath, "file", "f", "", "the path to a custom public key — will add public key to the project")
	RegisterCmd.Flags().StringVar(&publicKeyText, "pubkey", "", "OpenSSH or PEM public key content to be saved with the specified user UUID")
}

var RegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Registers a new user to be given access to the repository's secrets",
	RunE: func(cmd *cobra.Command, args []string) error {
		Logger.Infof("Starting register command")
		spinner, cleanup := startSpinner("Registering user for access...", verbose)
		defer cleanup()

		// Check for required flags
		Logger.Debugf("Checking command flags: username=%s, customFilePath=%s, publicKeyText provided=%t", username, customFilePath, publicKeyText != "")
		if username == "" && customFilePath == "" && publicKeyText == "" {
			finalMessage := color.RedString("✗") + " Either " + color.YellowString("--user") + ", " + color.YellowString("--file") + ", or " + color.YellowString("--pubkey") + " must be specified.\n" +
				"Run " + color.YellowString("kanuka secrets register --help") + " to see the available commands"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// When using --pubkey, username is required
		if publicKeyText != "" && username == "" {
			finalMessage := color.RedString("✗") + " When using " + color.YellowString("--pubkey") + ", the " + color.YellowString("--user") + " flag is required.\n" +
				"Specify a user UUID with " + color.YellowString("--user")
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

		switch {
		case publicKeyText != "":
			Logger.Infof("Handling public key text registration for user: %s", username)
			return handlePubkeyTextRegistration(spinner)
		case customFilePath != "":
			Logger.Infof("Handling custom file registration from: %s", customFilePath)
			return handleCustomFileRegistration(spinner)
		default:
			Logger.Infof("Handling user registration for: %s", username)
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

	// Validate and parse the public key text
	Logger.Debugf("Parsing public key text for user: %s", username)
	publicKey, err := secrets.ParsePublicKeyText(publicKeyText)
	if err != nil {
		Logger.Errorf("Invalid public key format provided: %v", err)
		finalMessage := color.RedString("✗") + " Invalid public key format provided\n" +
			color.RedString("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}
	Logger.Infof("Public key parsed successfully")

	// username here is the user UUID
	targetUserUUID := username

	// Save the public key to a file using user UUID
	pubKeyFilePath := filepath.Join(projectPublicKeyPath, targetUserUUID+".pub")
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
	Logger.Debugf("Registering user %s with public key", targetUserUUID)
	if err := registerUserWithPublicKey(targetUserUUID, publicKey); err != nil {
		Logger.Errorf("Failed to register user %s with public key: %v", targetUserUUID, err)
		finalMessage := color.RedString("✗") + " Failed to register user with the provided public key\n" +
			color.RedString("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}

	Logger.Infof("Public key registration completed successfully for user: %s", targetUserUUID)
	finalMessage := color.GreenString("✓") + " Public key for " + color.YellowString(targetUserUUID) + " has been saved and registered successfully!\n" +
		color.CyanString("→") + " They now have access to decrypt the repository's secrets"
	spinner.FinalMSG = finalMessage
	return nil
}

func registerUserWithPublicKey(targetUserUUID string, targetPublicKey *rsa.PublicKey) error {
	currentUserKeysPath := configs.UserKanukaSettings.UserKeysPath

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
	privateKeyPath := filepath.Join(currentUserKeysPath, projectUUID)
	Logger.Debugf("Loading private key from: %s", privateKeyPath)
	privateKey, err := secrets.LoadPrivateKey(privateKeyPath)
	if err != nil {
		Logger.Errorf("Failed to load private key from %s: %v", privateKeyPath, err)
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
	currentUserKeysPath := configs.UserKanukaSettings.UserKeysPath

	projectPath := configs.ProjectKanukaSettings.ProjectPath
	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath

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

	Logger.Debugf("Current user UUID: %s, Project UUID: %s, Project path: %s", currentUserUUID, projectUUID, projectPath)

	if projectPath == "" {
		finalMessage := color.RedString("✗") + " Kānuka has not been initialized\n" +
			color.CyanString("→") + " Run " + color.YellowString("kanuka secrets init") + " instead"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// username here is the target user's UUID
	targetUserUUID := username

	// Check if target user's public key exists (using their UUID)
	targetPubkeyPath := filepath.Join(projectPublicKeyPath, targetUserUUID+".pub")
	Logger.Debugf("Looking for target user's public key at: %s", targetPubkeyPath)

	// TODO: In the future, differentiate between FileNotFound Error and InvalidKey Error
	targetUserPublicKey, err := secrets.LoadPublicKey(targetPubkeyPath)
	if err != nil {
		Logger.Errorf("Failed to load public key for user %s from %s: %v", targetUserUUID, targetPubkeyPath, err)
		finalMessage := color.RedString("✗") + " Public key for user " + color.YellowString(targetUserUUID) + " not found\n" +
			"They must first run: " + color.YellowString("kanuka secrets create")
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
	privateKeyPath := filepath.Join(currentUserKeysPath, projectUUID)
	Logger.Debugf("Loading private key from: %s", privateKeyPath)

	privateKey, err := secrets.LoadPrivateKey(privateKeyPath)
	if err != nil {
		Logger.Errorf("Failed to load private key from %s: %v", privateKeyPath, err)
		finalMessage := color.RedString("✗") + " Couldn't get your private key from " + color.YellowString(privateKeyPath) + "\n\n" +
			"Are you sure you have access?\n\n" +
			color.RedString("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Decrypt symmetric key with current user's private key
	Logger.Debugf("Decrypting symmetric key with current user's private key")
	symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
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

	// Encrypt symmetric key with target user's public key
	Logger.Debugf("Encrypting symmetric key with target user's public key")
	targetEncryptedSymKey, err := secrets.EncryptWithPublicKey(symKey, targetUserPublicKey)
	if err != nil {
		return Logger.ErrorfAndReturn("Failed to encrypt symmetric key for target user: %v", err)
	}

	// Save encrypted symmetric key for target user using their UUID
	Logger.Debugf("Saving kanuka key for target user: %s", targetUserUUID)
	if err := secrets.SaveKanukaKeyToProject(targetUserUUID, targetEncryptedSymKey); err != nil {
		return Logger.ErrorfAndReturn("Failed to save encrypted key for target user: %v", err)
	}

	Logger.Infof("User registration completed successfully for: %s", targetUserUUID)
	finalMessage := color.GreenString("✓") + " Public key " + color.YellowString(targetUserUUID+".pub") + " has been registered successfully!\n" +
		color.CyanString("→") + " They now have access to decrypt the repository's secrets"
	spinner.FinalMSG = finalMessage
	return nil
}

func handleCustomFileRegistration(spinner *spinner.Spinner) error {
	currentUserKeysPath := configs.UserKanukaSettings.UserKeysPath

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
	privateKeyPath := filepath.Join(currentUserKeysPath, projectUUID)

	privateKey, err := secrets.LoadPrivateKey(privateKeyPath)
	if err != nil {
		finalMessage := color.RedString("✗") + " Couldn't get your private key from " + color.YellowString(privateKeyPath) + "\n\n" +
			"Are you sure you have access?\n\n" +
			color.RedString("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Decrypt symmetric key with current user's private key
	symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		finalMessage := color.RedString("✗") + " Failed to decrypt your Kānuka key using your private key: \n" +
			"    Kānuka key path: " + color.YellowString(kanukaKeyPath) + "\n" +
			"    Private key path: " + color.YellowString(privateKeyPath) + "\n\n" +
			"Are you sure you have access?\n\n" +
			color.RedString("Error: ") + err.Error()
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Encrypt symmetric key with target user's public key
	targetEncryptedSymKey, err := secrets.EncryptWithPublicKey(symKey, targetUserPublicKey)
	if err != nil {
		return Logger.ErrorfAndReturn("Failed to encrypt symmetric key for target user: %v", err)
	}

	// Save encrypted symmetric key for target user
	// The target user UUID is the filename without .pub extension
	targetUserUUID := strings.TrimSuffix(filepath.Base(customFilePath), ".pub")
	Logger.Debugf("Saving kanuka key for target user: %s (from custom file)", targetUserUUID)
	if err := secrets.SaveKanukaKeyToProject(targetUserUUID, targetEncryptedSymKey); err != nil {
		return Logger.ErrorfAndReturn("Failed to save encrypted key for target user: %v", err)
	}

	Logger.Infof("Custom file registration completed successfully for: %s", targetUserUUID)
	finalMessage := color.GreenString("✓") + " Public key " + color.YellowString(targetUserUUID+".pub") + " has been registered successfully!\n" +
		color.CyanString("→") + " They now have access to decrypt the repository's secrets"
	spinner.FinalMSG = finalMessage
	return nil
}
