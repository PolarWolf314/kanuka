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

func init() {
	registerCmd.Flags().StringVarP(&username, "user", "u", "", "username to register for access")
	registerCmd.Flags().StringVarP(&customFilePath, "file", "f", "", "the path to a custom public key — will add public key to the project")
	registerCmd.Flags().StringVar(&publicKeyText, "pubkey", "", "OpenSSH or PEM public key content to be saved with the specified username")
}

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Registers a new user to be given access to the repository's secrets",
	Run: func(cmd *cobra.Command, args []string) {
		Logger.Infof("Starting register command")
		spinner, cleanup := startSpinner("Registering user for access...", verbose)
		defer cleanup()

		// Check for required flags
		Logger.Debugf("Checking command flags: username=%s, customFilePath=%s, publicKeyText provided=%t", username, customFilePath, publicKeyText != "")
		if username == "" && customFilePath == "" && publicKeyText == "" {
			Logger.WarnfUser("No required flags specified")
			finalMessage := color.RedString("✗") + " Either " + color.YellowString("--user") + ", " + color.YellowString("--file") + ", or " + color.YellowString("--pubkey") + " must be specified.\n" +
				"Please run " + color.YellowString("kanuka secrets register --help") + " to see the available commands.\n"
			spinner.FinalMSG = finalMessage
			return
		}

		// When using --pubkey, username is required
		if publicKeyText != "" && username == "" {
			Logger.WarnfUser("Public key provided but username missing")
			finalMessage := color.RedString("✗") + " When using " + color.YellowString("--pubkey") + ", the " + color.YellowString("--user") + " flag is required.\n" +
				"Please specify a username with " + color.YellowString("--user") + ".\n"
			spinner.FinalMSG = finalMessage
			return
		}

		Logger.Debugf("Initializing project settings")
		if err := configs.InitProjectSettings(); err != nil {
			Logger.Fatalf("failed to init project settings: %v", err)
			return
		}

		switch {
		case publicKeyText != "":
			Logger.Infof("Handling public key text registration for user: %s", username)
			handlePubkeyTextRegistration(spinner)
		case customFilePath != "":
			Logger.Infof("Handling custom file registration from: %s", customFilePath)
			handleCustomFileRegistration(spinner)
		default:
			Logger.Infof("Handling user registration for: %s", username)
			handleUserRegistration(spinner)
		}
	},
}

func handlePubkeyTextRegistration(spinner *spinner.Spinner) {
	projectPath := configs.ProjectKanukaSettings.ProjectPath
	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath
	Logger.Debugf("Project path: %s, Public key path: %s", projectPath, projectPublicKeyPath)

	if projectPath == "" {
		Logger.WarnfUser("Kanuka has not been initialized")
		finalMessage := color.RedString("✗") + " Kanuka has not been initialized\n" +
			color.CyanString("→") + " Please run " + color.YellowString("kanuka secrets init") + " instead\n"
		spinner.FinalMSG = finalMessage
		return
	}

	// Validate and parse the public key text
	Logger.Debugf("Parsing public key text for user: %s", username)
	publicKey, err := secrets.ParsePublicKeyText(publicKeyText)
	if err != nil {
		Logger.Errorf("Invalid public key format provided: %v", err)
		finalMessage := color.RedString("✗") + " Invalid public key format provided\n" +
			color.RedString("Error: ") + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return
	}
	Logger.Infof("Public key parsed successfully")

	// Save the public key to a file
	pubKeyFilePath := filepath.Join(projectPublicKeyPath, username+".pub")
	Logger.Debugf("Saving public key to: %s", pubKeyFilePath)
	if err := secrets.SavePublicKeyToFile(publicKey, pubKeyFilePath); err != nil {
		Logger.Errorf("Failed to save public key to %s: %v", pubKeyFilePath, err)
		finalMessage := color.RedString("✗") + " Failed to save public key to " + color.YellowString(pubKeyFilePath) + "\n" +
			color.RedString("Error: ") + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return
	}
	Logger.Infof("Public key saved successfully")

	// Now register the user with the newly saved public key
	Logger.Debugf("Registering user %s with public key", username)
	if err := registerUserWithPublicKey(username, publicKey); err != nil {
		Logger.Errorf("Failed to register user %s with public key: %v", username, err)
		finalMessage := color.RedString("✗") + " Failed to register user with the provided public key\n" +
			color.RedString("Error: ") + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return
	}

	Logger.Infof("Public key registration completed successfully for user: %s", username)
	finalMessage := color.GreenString("✓") + " Public key for " + color.YellowString(username) + " has been saved and registered successfully!\n" +
		color.CyanString("→") + " They now have access to decrypt the repository's secrets\n"
	spinner.FinalMSG = finalMessage
}

func registerUserWithPublicKey(targetUsername string, targetPublicKey *rsa.PublicKey) error {
	currentUsername := configs.UserKanukaSettings.Username
	currentUserKeysPath := configs.UserKanukaSettings.UserKeysPath
	projectName := configs.ProjectKanukaSettings.ProjectName
	Logger.Debugf("Registering user %s with current user %s, project %s", targetUsername, currentUsername, projectName)

	// Get the current user's encrypted symmetric key
	Logger.Debugf("Getting project kanuka key for current user: %s", currentUsername)
	encryptedSymKey, err := secrets.GetProjectKanukaKey(currentUsername)
	if err != nil {
		Logger.Errorf("Failed to get project kanuka key for user %s: %v", currentUsername, err)
		return err
	}

	// Get current user's private key
	privateKeyPath := filepath.Join(currentUserKeysPath, projectName)
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

	// Save encrypted symmetric key for target user
	Logger.Debugf("Saving kanuka key for target user: %s", targetUsername)
	if err := secrets.SaveKanukaKeyToProject(targetUsername, targetEncryptedSymKey); err != nil {
		Logger.Errorf("Failed to save kanuka key for target user %s: %v", targetUsername, err)
		return err
	}

	Logger.Infof("Successfully registered user %s with public key", targetUsername)
	return nil
}

func handleUserRegistration(spinner *spinner.Spinner) {
	currentUsername := configs.UserKanukaSettings.Username
	currentUserKeysPath := configs.UserKanukaSettings.UserKeysPath

	projectName := configs.ProjectKanukaSettings.ProjectName
	projectPath := configs.ProjectKanukaSettings.ProjectPath
	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath
	Logger.Debugf("Current user: %s, Project: %s, Project path: %s", currentUsername, projectName, projectPath)

	if projectPath == "" {
		Logger.WarnfUser("Kanuka has not been initialized")
		finalMessage := color.RedString("✗") + " Kanuka has not been initialized\n" +
			color.CyanString("→") + " Please run " + color.YellowString("kanuka secrets init") + " instead\n"
		spinner.FinalMSG = finalMessage
		return
	}

	// Check if target user's public key exists
	targetPubkeyPath := filepath.Join(projectPublicKeyPath, username+".pub")
	Logger.Debugf("Looking for target user's public key at: %s", targetPubkeyPath)

	// TODO: In the future, differentiate between FileNotFound Error and InvalidKey Error
	targetUserPublicKey, err := secrets.LoadPublicKey(targetPubkeyPath)
	if err != nil {
		Logger.Errorf("Failed to load public key for user %s from %s: %v", username, targetPubkeyPath, err)
		finalMessage := color.RedString("✗") + " Public key for user " + color.YellowString(username) + " not found\n" +
			username + " must first run: " + color.YellowString("kanuka secrets create\n")
		spinner.FinalMSG = finalMessage
		return
	}
	Logger.Infof("Target user's public key loaded successfully")

	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath
	kanukaKeyPath := filepath.Join(projectSecretsPath, currentUsername+".kanuka")
	Logger.Debugf("Getting kanuka key from: %s", kanukaKeyPath)

	encryptedSymKey, err := secrets.GetProjectKanukaKey(currentUsername)
	if err != nil {
		Logger.Errorf("Failed to get kanuka key for current user %s: %v", currentUsername, err)
		finalMessage := color.RedString("✗") + " Couldn't get your Kanuka key from " + color.YellowString(kanukaKeyPath) + "\n\n" +
			"Are you sure you have access?\n\n" +
			color.RedString("Error: ") + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return
	}

	// Get current user's private key
	privateKeyPath := filepath.Join(currentUserKeysPath, projectName)
	Logger.Debugf("Loading private key from: %s", privateKeyPath)

	privateKey, err := secrets.LoadPrivateKey(privateKeyPath)
	if err != nil {
		Logger.Errorf("Failed to load private key from %s: %v", privateKeyPath, err)
		finalMessage := color.RedString("✗") + " Couldn't get your private key from " + color.YellowString(privateKeyPath) + "\n\n" +
			"Are you sure you have access?\n\n" +
			color.RedString("Error: ") + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return
	}

	// Decrypt symmetric key with current user's private key
	Logger.Debugf("Decrypting symmetric key with current user's private key")
	symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		Logger.Errorf("Failed to decrypt symmetric key: %v", err)
		finalMessage := color.RedString("✗") + " Failed to decrypt your Kanuka key using your private key: \n" +
			"    Kanuka key path: " + color.YellowString(kanukaKeyPath) + "\n" +
			"    Private key path: " + color.YellowString(privateKeyPath) + "\n\n" +
			"Are you sure you have access?\n\n" +
			color.RedString("Error: ") + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return
	}

	// Encrypt symmetric key with target user's public key
	Logger.Debugf("Encrypting symmetric key with target user's public key")
	targetEncryptedSymKey, err := secrets.EncryptWithPublicKey(symKey, targetUserPublicKey)
	if err != nil {
		Logger.Fatalf("Failed to encrypt symmetric key for target user: %v", err)
		return
	}

	// Save encrypted symmetric key for target user
	Logger.Debugf("Saving kanuka key for target user: %s", username)
	if err := secrets.SaveKanukaKeyToProject(username, targetEncryptedSymKey); err != nil {
		Logger.Fatalf("Failed to save encrypted key for target user: %v", err)
		return
	}

	Logger.Infof("User registration completed successfully for: %s", username)
	finalMessage := color.GreenString("✓") + " Public key " + color.YellowString(username+".pub") + " has been registered successfully!\n" +
		color.CyanString("→") + " They now have access to decrypt the repository's secrets\n"
	spinner.FinalMSG = finalMessage
}

func handleCustomFileRegistration(spinner *spinner.Spinner) {
	currentUsername := configs.UserKanukaSettings.Username
	currentUserKeysPath := configs.UserKanukaSettings.UserKeysPath

	projectName := configs.ProjectKanukaSettings.ProjectName
	projectPath := configs.ProjectKanukaSettings.ProjectPath
	Logger.Debugf("Current user: %s, Project: %s, Custom file path: %s", currentUsername, projectName, customFilePath)

	if projectPath == "" {
		Logger.WarnfUser("Kanuka has not been initialized")
		finalMessage := color.RedString("✗") + " Kanuka has not been initialized\n" +
			color.CyanString("→") + " Please run " + color.YellowString("kanuka secrets init") + " instead\n"
		spinner.FinalMSG = finalMessage
		return
	}

	if !strings.HasSuffix(customFilePath, ".pub") {
		Logger.WarnfUser("Invalid file path provided: %s (must end with .pub)", customFilePath)
		finalMessage := color.RedString("✗ ") + color.YellowString(customFilePath) + " is not a valid path to a public key file.\n"
		spinner.FinalMSG = finalMessage
		return
	}

	// Load the custom public key
	Logger.Debugf("Loading public key from custom file: %s", customFilePath)
	targetUserPublicKey, err := secrets.LoadPublicKey(customFilePath)
	if err != nil {
		Logger.Errorf("Failed to load public key from %s: %v", customFilePath, err)
		finalMessage := color.RedString("✗") + " Public key could not be loaded from " + color.YellowString(customFilePath) + "\n\n" +
			color.RedString("Error: ") + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return
	}
	Logger.Infof("Public key loaded successfully from custom file")

	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath
	kanukaKeyPath := filepath.Join(projectSecretsPath, currentUsername+".kanuka")

	encryptedSymKey, err := secrets.GetProjectKanukaKey(currentUsername)
	if err != nil {
		finalMessage := color.RedString("✗") + " Couldn't get your Kanuka key from " + color.YellowString(kanukaKeyPath) + "\n\n" +
			"Are you sure you have access?\n\n" +
			color.RedString("Error: ") + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return
	}

	// Get current user's private key
	privateKeyPath := filepath.Join(currentUserKeysPath, projectName)

	privateKey, err := secrets.LoadPrivateKey(privateKeyPath)
	if err != nil {
		finalMessage := color.RedString("✗") + " Couldn't get your private key from " + color.YellowString(privateKeyPath) + "\n\n" +
			"Are you sure you have access?\n\n" +
			color.RedString("Error: ") + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return
	}

	// Decrypt symmetric key with current user's private key
	symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
	if err != nil {
		finalMessage := color.RedString("✗") + " Failed to decrypt your Kanuka key using your private key: \n" +
			"    Kanuka key path: " + color.YellowString(kanukaKeyPath) + "\n" +
			"    Private key path: " + color.YellowString(privateKeyPath) + "\n\n" +
			"Are you sure you have access?\n\n" +
			color.RedString("Error: ") + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return
	}

	// Encrypt symmetric key with target user's public key
	targetEncryptedSymKey, err := secrets.EncryptWithPublicKey(symKey, targetUserPublicKey)
	if err != nil {
		Logger.Fatalf("Failed to encrypt symmetric key for target user: %v", err)
		return
	}

	// Save encrypted symmetric key for target user
	targetName := strings.TrimSuffix(filepath.Base(customFilePath), ".pub")
	Logger.Debugf("Saving kanuka key for target user: %s (from custom file)", targetName)
	if err := secrets.SaveKanukaKeyToProject(targetName, targetEncryptedSymKey); err != nil {
		Logger.Fatalf("Failed to save encrypted key for target user: %v", err)
		return
	}

	Logger.Infof("Custom file registration completed successfully for: %s", targetName)
	finalMessage := color.GreenString("✓") + " Public key " + color.YellowString(targetName+".pub") + " has been registered successfully!\n" +
		color.CyanString("→") + " They now have access to decrypt the repository's secrets\n"
	spinner.FinalMSG = finalMessage
}
