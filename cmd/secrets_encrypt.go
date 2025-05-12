package cmd

import (
	"kanuka/internal/secrets"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func init() {
	encryptCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
}

var encryptCmd = &cobra.Command{
	Use:   "encrypt",
	Short: "Encrypts the .env file into .env.kanuka using your Kanuka key",
	Run: func(cmd *cobra.Command, args []string) {
		spinner, cleanup := startSpinner("Encrypting environment files...", verbose)
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

		// Step 1: Check for .env file
		// TODO: In future, add config options to list which dirs to ignore. .kanuka/ ignored by default
		listOfEnvFiles, err := secrets.FindEnvOrKanukaFiles(projectRoot, []string{}, false)
		if err != nil {
			printError("Failed to find environment files", err)
			return
		}
		if len(listOfEnvFiles) == 0 {
			finalMessage := color.RedString("✗") + " No environment files found in " + color.YellowString(projectRoot) + "\n"
			spinner.FinalMSG = finalMessage
			return
		}

		// Step 2: Get project's encrypted symmetric key
		currentUsername, err := secrets.GetUsername()
		if err != nil {
			printError("Failed to get username", err)
			return
		}

		encryptedSymKey, err := secrets.GetProjectKanukaKey(currentUsername)
		if err != nil {
			finalMessage := color.RedString("✗") + " Failed to get your " +
				color.YellowString(".kanuka") + " file. Are you sure you have access?\n" +
				color.RedString("Error: ") + err.Error() + "\n"
			spinner.FinalMSG = finalMessage
			return
		}

		homeDir, err := os.UserHomeDir()
		if err != nil {
			printError("Failed to get user's home directory", err)
			return
		}
		projectName := filepath.Base(projectRoot)
		privateKeyPath := filepath.Join(homeDir, ".config", ".kanuka", "keys", projectName)

		privateKey, err := secrets.LoadPrivateKey(privateKeyPath)
		if err != nil {
			finalMessage := color.RedString("✗") + " Failed to get your private key file. Are you sure you have access?\n" +
				color.RedString("Error: ") + err.Error() + "\n"
			spinner.FinalMSG = finalMessage
			return
		}

		// Step 3: Decrypt user's kanuka file (get symmetric key)
		symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
		if err != nil {
			finalMessage := color.RedString("✗") + " Failed to decrypt your " +
				color.YellowString(".kanuka") + " file. Are you sure you have access?\n" +
				color.RedString("Error: ") + err.Error() + "\n"

			spinner.FinalMSG = finalMessage
			return
		}

		// Step 4: Encrypt all env files
		if err := secrets.EncryptFiles(symKey, listOfEnvFiles, verbose); err != nil {
			finalMessage := color.RedString("✗") + " Failed to encrypt the project's " +
				color.YellowString(".env") + " files. Are you sure you have access?\n" +
				color.RedString("Error: ") + err.Error() + "\n"
			spinner.FinalMSG = finalMessage
			return
		}

		// we can be sure they exist if the previous function ran without errors
		listOfKanukaFiles, err := secrets.FindEnvOrKanukaFiles(projectRoot, []string{}, true)
		if err != nil {
			printError("Failed to find environment files", err)
			return
		}

		formattedListOfFiles := secrets.FormatPaths(listOfKanukaFiles)

		finalMessage := color.GreenString("✓") + " Environment files encrypted successfully!\n" +
			"The following files were created: " + formattedListOfFiles +
			color.CyanString("→") + " You can now safely commit all " + color.YellowString(".kanuka") + " files in your repository\n"

		spinner.FinalMSG = finalMessage
	},
}
