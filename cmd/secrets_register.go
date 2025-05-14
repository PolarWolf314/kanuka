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

		targetUserPublicKey, err := secrets.LoadPublicKey(targetPubkeyPath)
		if err != nil {
			finalMessage := color.RedString("✗") + " Public key for user " + color.YellowString(username) + " not found\n" +
				username + " must first run: " + color.YellowString("kanuka secrets create\n")
			spinner.FinalMSG = finalMessage
			return
		}

		encryptedSymKey, err := secrets.GetProjectKanukaKey(currentUsername)
		if err != nil {
			printError("Failed to get current user's .kanuka file", err)
			return
		}

		// Get current user's private key
		privateKeyPath := filepath.Join(currentUserKeysPath, projectName)

		privateKey, err := secrets.LoadPrivateKey(privateKeyPath)
		if err != nil {
			printError("Failed to get current user's private key", err)
			return
		}

		// Decrypt symmetric key with current user's private key
		symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
		if err != nil {
			printError("Failed to decrypt symmetric key", err)
			return
		}

		// Encrypt symmetric key with target user's public key
		targetEncryptedSymKey, err := secrets.EncryptWithPublicKey(symKey, targetUserPublicKey)
		if err != nil {
			printError("Failed to encrypt symmetric key for target user", err)
			return
		}

		// Save encrypted symmetric key for target user
		if err := secrets.SaveKanukaKeyToProject(username, targetEncryptedSymKey); err != nil {
			printError("Failed to save encrypted key for target user", err)
			return
		}

		finalMessage := color.GreenString("✓") + " User " + color.YellowString(username) + " has been registered successfully!\n" +
			color.CyanString("→") + " They now have access to decrypt the repository's secrets\n"
		spinner.FinalMSG = finalMessage
	},
}
