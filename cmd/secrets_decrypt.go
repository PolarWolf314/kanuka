package cmd

import (
	"fmt"
	"kanuka/internal/secrets"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func init() {
	decryptCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
}

var decryptCmd = &cobra.Command{
	Use:   "decrypt",
	Short: "Decrypts the .env.kanuka file back into .env using your Kanuka key",
	Run: func(cmd *cobra.Command, args []string) {
		spinner, cleanup := startSpinner("Decrypting environment files...", verbose)
		defer cleanup()

		projectRoot, err := secrets.FindProjectKanukaRoot()
		if err != nil {
			printError("Failed to obtain project root", err)
			return
		}
		if projectRoot == "" {
			finalMessage := color.RedString("✗") + " Kanuka has not been initialized\n" +
				color.CyanString("→") + " Please run " + color.YellowString("kanuka secrets init") + " instead\n"
			spinner.FinalMSG = finalMessage
			return
		}

		verboseLog("🚀 Starting decryption process...")

		// Step 1: Check for .kanuka files
		// TODO: In future, add config options to list which dirs to ignore. .kanuka/ ignored by default
		listOfKanukaFiles, err := secrets.FindEnvOrKanukaFiles(projectRoot, []string{}, true)
		if err != nil {
			printError("Failed to find environment files", err)
			return
		}
		if len(listOfKanukaFiles) == 0 {
			finalMessage := color.RedString("✗") + " No encrypted environment (" + color.YellowString(".kanuka") + ") files found in " + color.YellowString(projectRoot) + "\n"
			spinner.FinalMSG = finalMessage
			return
		}

		verboseLog(fmt.Sprintf("✅ Found %d .env.kanuka files: %s", len(listOfKanukaFiles), secrets.FormatPaths(listOfKanukaFiles)))

		// Step 2: Get project's encrypted symmetric key
		currentUsername, err := secrets.GetUsername()
		if err != nil {
			printError("Failed to get username", err)
			return
		}

		encryptedSymKey, err := secrets.GetProjectKanukaKey(currentUsername)
		if err != nil {
			finalMessage := color.RedString("✗") + " Failed to obtain your " +
				color.YellowString(".kanuka") + " file. Are you sure you have access?\n" +
				color.RedString("Error: ") + err.Error() + "\n"
			spinner.FinalMSG = finalMessage
			return
		}
		verboseLog("🔑 Loaded user's .kanuka key")

		homeDir, err := os.UserHomeDir()
		if err != nil {
			printError("Failed to get user's home directory", err)
			return
		}
		projectName := filepath.Base(projectRoot)
		privateKeyPath := filepath.Join(homeDir, ".kanuka", "keys", projectName)

		privateKey, err := secrets.LoadPrivateKey(privateKeyPath)
		if err != nil {
			finalMessage := color.RedString("✗") + " Failed to get your private key file. Are you sure you have access?\n" +
				color.RedString("Error: ") + err.Error() + "\n"
			spinner.FinalMSG = finalMessage
			return
		}
		verboseLog("🔑 Loaded user's private key")

		// Step 3: Decrypt user's kanuka file (get symmetric key)
		symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
		if err != nil {
			finalMessage := color.RedString("✗") + " Failed to decrypt your " +
				color.YellowString(".kanuka") + " file. Are you sure you have access?\n" +
				color.RedString("Error: ") + err.Error() + "\n"

			spinner.FinalMSG = finalMessage
			return
		}

		verboseLog("🔓 Decrypted symmetric key")

		// Step 4: Decrypt all .kanuka files
		if err := secrets.DecryptFiles(symKey, listOfKanukaFiles, verbose); err != nil {
			finalMessage := color.RedString("✗") + " Failed to decrypt the project's " +
				color.YellowString(".kanuka") + " files. Are you sure you have access?\n" +
				color.RedString("Error: ") + err.Error() + "\n"
			spinner.FinalMSG = finalMessage
			return
		}

		// we can be sure they exist if the previous function ran without errors
		listOfEnvFiles, err := secrets.FindEnvOrKanukaFiles(projectRoot, []string{}, false)
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
