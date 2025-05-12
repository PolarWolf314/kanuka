package cmd

import (
	"kanuka/internal/configs"
	"kanuka/internal/secrets"
	"kanuka/internal/utils"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func init() {
	encryptCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")

	configs.InitProjectSettings()
}

var encryptCmd = &cobra.Command{
	Use:   "encrypt",
	Short: "Encrypts the .env file into .env.kanuka using your Kanuka key",
	Run: func(cmd *cobra.Command, args []string) {
		spinner, cleanup := startSpinner("Encrypting environment files...", verbose)
		defer cleanup()

		projectName := configs.ProjectKanukaSettings.ProjectName
		projectPath := configs.ProjectKanukaSettings.ProjectPath

		if projectPath == "" {
			finalMessage := color.RedString("✗") + " Kanuka has not been initialized\n" +
				color.CyanString("→") + " Please run " + color.YellowString("kanuka secrets init") + " instead\n"
			spinner.FinalMSG = finalMessage
			return
		}

		// TODO: In future, add config options to list which dirs to ignore. .kanuka/ ignored by default
		listOfEnvFiles, err := secrets.FindEnvOrKanukaFiles(projectPath, []string{}, false)
		if err != nil {
			printError("Failed to find environment files", err)
			return
		}
		if len(listOfEnvFiles) == 0 {
			finalMessage := color.RedString("✗") + " No environment files found in " + color.YellowString(projectPath) + "\n"
			spinner.FinalMSG = finalMessage
			return
		}

		username := configs.UserKanukaSettings.Username
		userKeysPath := configs.UserKanukaSettings.UserKeysPath

		encryptedSymKey, err := secrets.GetProjectKanukaKey(username)
		if err != nil {
			finalMessage := color.RedString("✗") + " Failed to get your " +
				color.YellowString(".kanuka") + " file. Are you sure you have access?\n" +
				color.RedString("Error: ") + err.Error() + "\n"
			spinner.FinalMSG = finalMessage
			return
		}

		privateKeyPath := filepath.Join(userKeysPath, projectName)
		privateKey, err := secrets.LoadPrivateKey(privateKeyPath)
		if err != nil {
			finalMessage := color.RedString("✗") + " Failed to get your private key file. Are you sure you have access?\n" +
				color.RedString("Error: ") + err.Error() + "\n"
			spinner.FinalMSG = finalMessage
			return
		}

		symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
		if err != nil {
			finalMessage := color.RedString("✗") + " Failed to decrypt your " +
				color.YellowString(".kanuka") + " file. Are you sure you have access?\n" +
				color.RedString("Error: ") + err.Error() + "\n"

			spinner.FinalMSG = finalMessage
			return
		}

		if err := secrets.EncryptFiles(symKey, listOfEnvFiles, verbose); err != nil {
			finalMessage := color.RedString("✗") + " Failed to encrypt the project's " +
				color.YellowString(".env") + " files. Are you sure you have access?\n" +
				color.RedString("Error: ") + err.Error() + "\n"
			spinner.FinalMSG = finalMessage
			return
		}

		// we can be sure they exist if the previous function ran without errors
		listOfKanukaFiles, err := secrets.FindEnvOrKanukaFiles(projectPath, []string{}, true)
		if err != nil {
			printError("Failed to find environment files", err)
			return
		}

		formattedListOfFiles := utils.FormatPaths(listOfKanukaFiles)

		finalMessage := color.GreenString("✓") + " Environment files encrypted successfully!\n" +
			"The following files were created: " + formattedListOfFiles +
			color.CyanString("→") + " You can now safely commit all " + color.YellowString(".kanuka") + " files in your repository\n"

		spinner.FinalMSG = finalMessage
	},
}
