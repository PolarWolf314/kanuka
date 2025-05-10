package cmd

import (
	"fmt"
	"kanuka/internal/secrets"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func init() {
	decryptCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}

var decryptCmd = &cobra.Command{
	Use:   "decrypt",
	Short: "Decrypts the .env.kanuka file back into .env using your Kanuka key",
	Run: func(cmd *cobra.Command, args []string) {
		spinner, cleanup := startSpinner("Decrypting environment files...", verbose)
		defer cleanup()

		kanukaExists, err := secrets.DoesProjectKanukaSettingsExist()
		if err != nil {
			printError("Failed to check if project kanuka settings exists", err)
			return
		}
		if !kanukaExists {
			finalMessage := color.RedString("✗") + " Kanuka has not been initialized\n" +
				color.CyanString("→") + " Please run " + color.YellowString("kanuka secrets init") + " instead\n"
			spinner.FinalMSG = finalMessage
			return
		}

		verboseLog("🚀 Starting decryption process...")

		// Step 1: Check for .env.kanuka file
		workingDirectory, err := os.Getwd()
		if err != nil {
			printError("Failed to get working directory", err)
			return
		}

		// TODO: In future, add config options to list which dirs to ignore. .kanuka/ ignored by default
		listOfKanukaFiles, err := secrets.FindEnvOrKanukaFiles(workingDirectory, []string{}, true)
		if err != nil {
			printError("Failed to find environment files", err)
			return
		}
		if len(listOfKanukaFiles) == 0 {
			finalMessage := color.RedString("✗") + " No environment files found in " + color.YellowString(workingDirectory) + "\n"
			spinner.FinalMSG = finalMessage
			return
		}

		verboseLog(fmt.Sprintf("✅ Found %d .env.kanuka files: %s", len(listOfKanukaFiles), secrets.FormatPaths(listOfKanukaFiles)))

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
			finalMessage := color.RedString("✗") + " Failed to decrypt your " +
				color.YellowString(".kanuka") + " file. Are you sure you have access?\n" +
				"Error: " + color.RedString(err.Error()) + "\n"

			spinner.FinalMSG = finalMessage
			return
		}
		verboseLog("🔓 Decrypted symmetric key")

		// Step 4: Decrypt all .kanuka files
		if err := secrets.DecryptFiles(symKey, listOfKanukaFiles, verbose); err != nil {
			printError("Failed to decrypt environment files", err)
			return
		}

		// we can be sure they exist if the previous function ran without errors
		listOfEnvFiles, err := secrets.FindEnvOrKanukaFiles(workingDirectory, []string{}, false)
		if err != nil {
			printError("Failed to find environment files", err)
			return
		}

		formattedListOfFiles := secrets.FormatPaths(listOfEnvFiles)

		finalMessage := color.GreenString("✓") + " Environment files decrypted successfully!\n" +
			"The following files were created:" + formattedListOfFiles +
			color.CyanString("→") + " Your environment files are now ready to use\n"

		spinner.FinalMSG = finalMessage
	},
}
