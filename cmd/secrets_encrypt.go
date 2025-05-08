package cmd

import (
	"fmt"
	"kanuka/internal/secrets"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func init() {
	encryptCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}

var encryptCmd = &cobra.Command{
	Use:   "encrypt",
	Short: "Encrypts the .env file into .env.kanuka using your Kanuka key",
	Run: func(cmd *cobra.Command, args []string) {
		_, cleanup := startSpinner("Encrypting environment files...", verbose)
		defer cleanup()

		kanukaExists, err := secrets.DoesProjectKanukaSettingsExist()
		if err != nil {
			printError("Failed to check if project kanuka settings exists", err)
			return
		}
		if !kanukaExists {
			printError(".kanuka/ doesn't exist", fmt.Errorf("please init the project first with `kanuka init`"))
			return
		}

		verboseLog("🚀 Starting encryption process...")

		// Step 1: Check for .env file
		workingDirectory, err := os.Getwd()
		if err != nil {
			printError("Failed to get working directory", err)
			return
		}

		// TODO: In future, add config options to list which dirs to ignore. .kanuka/ ignored by default
		listOfEnvFiles, err := secrets.FindEnvOrKanukaFiles(workingDirectory, []string{}, false)
		if err != nil {
			printError("Failed to find environment files", err)
			return
		}
		if len(listOfEnvFiles) == 0 {
			printError("No environment files found", fmt.Errorf("no .env files in %v", workingDirectory))
			return
		}

		verboseLog(fmt.Sprintf("✅ Found %d .env files: %s", len(listOfEnvFiles), secrets.FormatPaths(listOfEnvFiles)))

		// Step 2: Get project's encrypted symmetric key
		encryptedSymKey, err := secrets.GetUserProjectKanukaKey()
		if err != nil {
			printError("Failed to get user's .kanuka file", err)
			return
		}
		verboseLog("🔑 Loaded user's .kanuka key")

		privateKey, err := secrets.GetUserPrivateKey()
		if err != nil {
			printError("Failed to get user's private key", err)
			return
		}
		verboseLog("🔑 Loaded user's private key")

		// Step 3: Decrypt user's kanuka file (get symmetric key)
		symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
		if err != nil {
			printError("Failed to decrypt symmetric key", err)
			return
		}
		verboseLog("🔓 Decrypted symmetric key")

		// Step 4: Encrypt all env files
		if err := secrets.EncryptFiles(symKey, listOfEnvFiles, verbose); err != nil {
			printError("Failed to encrypt environment files", err)
			return
		}

		fmt.Println(color.GreenString("✓") + " Environment files encrypted successfully!")
		fmt.Println(color.CyanString("→") + " You can now safely commit all " + color.YellowString(".kanuka") + " files in your repository")
	},
}
