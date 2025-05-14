package cmd

import (
	"kanuka/internal/configs"
	"kanuka/internal/secrets"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	username       string
	customFilePath string
)

func init() {
	registerCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	registerCmd.Flags().StringVarP(&username, "user", "u", "", "username to register for access")
	registerCmd.Flags().StringVarP(&customFilePath, "file", "f", "", "the path to a custom public key — will add public key to the project")
}

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Registers a new user to be given access to the repository's secrets",
	Run: func(cmd *cobra.Command, args []string) {
		spinner, cleanup := startSpinner("Registering user for access...", verbose)
		defer cleanup()

		if username == "" && customFilePath == "" {
			finalMessage := color.RedString("✗") + " Either " + color.YellowString("--user") + " or " + color.YellowString("--file") + " must be specified.\n" +
				"Please run " + color.YellowString("kanuka secrets register --help") + " to see the available commands.\n"
			spinner.FinalMSG = finalMessage
			return
		}

		if err := configs.InitProjectSettings(); err != nil {
			printError("failed to init project settings", err)
			return
		}
		currentUsername := configs.UserKanukaSettings.Username
		currentUserKeysPath := configs.UserKanukaSettings.UserKeysPath

		projectName := configs.ProjectKanukaSettings.ProjectName
		projectPath := configs.ProjectKanukaSettings.ProjectPath
		projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath

		if projectPath == "" {
			finalMessage := color.RedString("✗") + " Kanuka has not been initialized\n" +
				color.CyanString("→") + " Please run " + color.YellowString("kanuka secrets init") + " instead\n"
			spinner.FinalMSG = finalMessage
			return
		}

		// Check if target user's public key exists
		targetPubkeyPath := filepath.Join(projectPublicKeyPath, username+".pub")

		if customFilePath != "" {
			if !strings.HasSuffix(customFilePath, ".pub") {
				finalMessage := color.RedString("✗ ") + color.YellowString(customFilePath) + " is not a valid path to a public key file.\n"
				spinner.FinalMSG = finalMessage
				return
			}
			targetPubkeyPath = customFilePath
		}

		// TODO: In the future, differentiate between FileNotFound Error and InvalidKey Error
		targetUserPublicKey, err := secrets.LoadPublicKey(targetPubkeyPath)
		if err != nil {
			if customFilePath != "" {
				finalMessage := color.RedString("✗") + " Public key could not be loaded from " + color.YellowString(customFilePath) + "\n\n" +
					color.RedString("Error: ") + err.Error() + "\n"
				spinner.FinalMSG = finalMessage
				return
			}

			finalMessage := color.RedString("✗") + " Public key for user " + color.YellowString(username) + " not found\n" +
				username + " must first run: " + color.YellowString("kanuka secrets create\n")
			spinner.FinalMSG = finalMessage
			return
		}

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
			printError("Failed to encrypt symmetric key for target user", err)
			return
		}

		// Save encrypted symmetric key for target user
		targetName := username
		if customFilePath != "" {
			targetName = strings.TrimSuffix(filepath.Base(customFilePath), ".pub")
		}
		if err := secrets.SaveKanukaKeyToProject(targetName, targetEncryptedSymKey); err != nil {
			printError("Failed to save encrypted key for target user", err)
			return
		}

		finalMessage := color.GreenString("✓") + " Public key " + color.YellowString(targetName+".pub") + " has been registered successfully!\n" +
			color.CyanString("→") + " They now have access to decrypt the repository's secrets\n"
		spinner.FinalMSG = finalMessage
	},
}
