package cmd

import (
	"path/filepath"

	"github.com/PolarWolf314/kanuka/internal/secrets"
	"github.com/PolarWolf314/kanuka/internal/utils"

	"github.com/PolarWolf314/kanuka/internal/configs"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var decryptCmd = &cobra.Command{
	Use:   "decrypt",
	Short: "Decrypts the .env.kanuka file back into .env using your Kanuka key",
	Run: func(cmd *cobra.Command, args []string) {
		Logger.Infof("Starting decrypt command")
		spinner, cleanup := startSpinner("Decrypting environment files...", verbose)
		defer cleanup()

		Logger.Debugf("Initializing project settings")
		if err := configs.InitProjectSettings(); err != nil {
			Logger.Errorf("Failed to initialize project settings: %v", err)
			printError("failed to init project settings", err)
			return
		}
		projectName := configs.ProjectKanukaSettings.ProjectName
		projectPath := configs.ProjectKanukaSettings.ProjectPath
		Logger.Debugf("Project name: %s, Project path: %s", projectName, projectPath)

		if projectPath == "" {
			Logger.Warnf("Kanuka has not been initialized")
			finalMessage := color.RedString("✗") + " Kanuka has not been initialized\n" +
				color.CyanString("→") + " Please run " + color.YellowString("kanuka secrets init") + " instead\n"
			spinner.FinalMSG = finalMessage
			return
		}

		// TODO: In future, add config options to list which dirs to ignore. .kanuka/ ignored by default
		Logger.Debugf("Searching for .kanuka files in project path")
		listOfKanukaFiles, err := secrets.FindEnvOrKanukaFiles(projectPath, []string{}, true)
		if err != nil {
			Logger.Errorf("Failed to find environment files: %v", err)
			printError("Failed to find environment files", err)
			return
		}
		Logger.Debugf("Found %d .kanuka files", len(listOfKanukaFiles))
		if len(listOfKanukaFiles) == 0 {
			Logger.Warnf("No encrypted environment files found in %s", projectPath)
			finalMessage := color.RedString("✗") + " No encrypted environment (" + color.YellowString(".kanuka") + ") files found in " + color.YellowString(projectPath) + "\n"
			spinner.FinalMSG = finalMessage
			return
		}

		username := configs.UserKanukaSettings.Username
		userKeysPath := configs.UserKanukaSettings.UserKeysPath
		Logger.Debugf("Username: %s, User keys path: %s", username, userKeysPath)

		Logger.Debugf("Getting project kanuka key for user: %s", username)
		encryptedSymKey, err := secrets.GetProjectKanukaKey(username)
		if err != nil {
			Logger.Errorf("Failed to obtain kanuka key for user %s: %v", username, err)
			finalMessage := color.RedString("✗") + " Failed to obtain your " +
				color.YellowString(".kanuka") + " file. Are you sure you have access?\n" +
				color.RedString("Error: ") + err.Error() + "\n"
			spinner.FinalMSG = finalMessage
			return
		}

		privateKeyPath := filepath.Join(userKeysPath, projectName)
		Logger.Debugf("Loading private key from: %s", privateKeyPath)
		privateKey, err := secrets.LoadPrivateKey(privateKeyPath)
		if err != nil {
			Logger.Errorf("Failed to load private key from %s: %v", privateKeyPath, err)
			finalMessage := color.RedString("✗") + " Failed to get your private key file. Are you sure you have access?\n" +
				color.RedString("Error: ") + err.Error() + "\n"
			spinner.FinalMSG = finalMessage
			return
		}
		Logger.Infof("Private key loaded successfully")

		Logger.Debugf("Decrypting symmetric key with private key")
		symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
		if err != nil {
			Logger.Errorf("Failed to decrypt symmetric key: %v", err)
			finalMessage := color.RedString("✗") + " Failed to decrypt your " +
				color.YellowString(".kanuka") + " file. Are you sure you have access?\n" +
				color.RedString("Error: ") + err.Error() + "\n"

			spinner.FinalMSG = finalMessage
			return
		}
		Logger.Infof("Symmetric key decrypted successfully")

		Logger.Infof("Decrypting %d files", len(listOfKanukaFiles))
		if err := secrets.DecryptFiles(symKey, listOfKanukaFiles, verbose); err != nil {
			Logger.Errorf("Failed to decrypt files: %v", err)
			finalMessage := color.RedString("✗") + " Failed to decrypt the project's " +
				color.YellowString(".kanuka") + " files. Are you sure you have access?\n" +
				color.RedString("Error: ") + err.Error() + "\n"
			spinner.FinalMSG = finalMessage
			return
		}

		// we can be sure they exist if the previous function ran without errors
		Logger.Debugf("Finding decrypted environment files")
		listOfEnvFiles, err := secrets.FindEnvOrKanukaFiles(projectPath, []string{}, false)
		if err != nil {
			Logger.Errorf("Failed to find environment files after decryption: %v", err)
			printError("Failed to find environment files", err)
			return
		}

		formattedListOfFiles := utils.FormatPaths(listOfEnvFiles)
		Logger.Infof("Decrypt command completed successfully. Created %d environment files", len(listOfEnvFiles))

		finalMessage := color.GreenString("✓") + " Environment files decrypted successfully!\n" +
			"The following files were created:" + formattedListOfFiles +
			color.CyanString("→") + " Your environment files are now ready to use\n"

		spinner.FinalMSG = finalMessage
	},
}
