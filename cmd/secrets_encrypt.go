package cmd

import (
	"os"
	"path/filepath"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/secrets"
	"github.com/PolarWolf314/kanuka/internal/utils"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var encryptCmd = &cobra.Command{
	Use:   "encrypt",
	Short: "Encrypts the .env file into .env.kanuka using your Kānuka key",
	RunE: func(cmd *cobra.Command, args []string) error {
		Logger.Infof("Starting encrypt command")
		spinner, cleanup := startSpinner("Encrypting environment files...", verbose)
		defer cleanup()

		Logger.Debugf("Initializing project settings")
		if err := configs.InitProjectSettings(); err != nil {
			return Logger.ErrorfAndReturn("failed to init project settings: %v", err)
		}
		projectName := configs.ProjectKanukaSettings.ProjectName
		projectPath := configs.ProjectKanukaSettings.ProjectPath
		Logger.Debugf("Project name: %s, Project path: %s", projectName, projectPath)

		if projectPath == "" {
			finalMessage := color.RedString("✗") + " Kānuka has not been initialized\n" +
				color.CyanString("→") + " Run " + color.YellowString("kanuka secrets init") + " instead"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// TODO: In future, add config options to list which dirs to ignore. .kanuka/ ignored by default
		Logger.Debugf("Searching for .env files in project path")
		listOfEnvFiles, err := secrets.FindEnvOrKanukaFiles(projectPath, []string{}, false)
		if err != nil {
			return Logger.ErrorfAndReturn("Failed to find environment files: %v", err)
		}
		Logger.Debugf("Found %d .env files", len(listOfEnvFiles))
		if len(listOfEnvFiles) == 0 {
			finalMessage := color.RedString("✗") + " No environment files found in " + color.YellowString(projectPath)
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Performance warning for large number of files
		if len(listOfEnvFiles) > 20 {
			Logger.Warnf("Processing %d environment files - this may take a moment", len(listOfEnvFiles))
		}

		username := configs.UserKanukaSettings.Username
		userKeysPath := configs.UserKanukaSettings.UserKeysPath
		Logger.Debugf("Username: %s, User keys path: %s", username, userKeysPath)

		Logger.Debugf("Getting project kanuka key for user: %s", username)
		encryptedSymKey, err := secrets.GetProjectKanukaKey(username)
		if err != nil {
			Logger.Errorf("Failed to obtain kanuka key for user %s: %v", username, err)
			finalMessage := color.RedString("✗") + " Failed to get your " +
				color.YellowString(".kanuka") + " file. Are you sure you have access?\n" +
				color.RedString("Error: ") + err.Error()
			spinner.FinalMSG = finalMessage
			return nil
		}

		privateKeyPath := filepath.Join(userKeysPath, projectName)
		Logger.Debugf("Loading private key from: %s", privateKeyPath)
		privateKey, err := secrets.LoadPrivateKey(privateKeyPath)
		if err != nil {
			Logger.Errorf("Failed to load private key from %s: %v", privateKeyPath, err)
			finalMessage := color.RedString("✗") + " Failed to get your private key file. Are you sure you have access?\n" +
				color.RedString("Error: ") + err.Error()
			spinner.FinalMSG = finalMessage
			return nil
		}
		Logger.Infof("Private key loaded successfully")

		// Security warning: Check private key file permissions
		if fileInfo, err := os.Stat(privateKeyPath); err == nil {
			if fileInfo.Mode().Perm() != 0600 {
				spinner.Stop()
				Logger.WarnfAlways("Private key file has overly permissive permissions (%o), consider running 'chmod 600 %s'",
					fileInfo.Mode().Perm(), privateKeyPath)
				spinner.Start()
			}
		}

		Logger.Debugf("Decrypting symmetric key with private key")
		symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
		if err != nil {
			Logger.Errorf("Failed to decrypt symmetric key: %v", err)
			finalMessage := color.RedString("✗") + " Failed to decrypt your " +
				color.YellowString(".kanuka") + " file. Are you sure you have access?\n" +
				color.RedString("Error: ") + err.Error()

			spinner.FinalMSG = finalMessage
			return nil
		}
		Logger.Infof("Symmetric key decrypted successfully")

		Logger.Infof("Encrypting %d files", len(listOfEnvFiles))
		if err := secrets.EncryptFiles(symKey, listOfEnvFiles, verbose); err != nil {
			Logger.Errorf("Failed to encrypt files: %v", err)
			finalMessage := color.RedString("✗") + " Failed to encrypt the project's " +
				color.YellowString(".env") + " files. Are you sure you have access?\n" +
				color.RedString("Error: ") + err.Error()
			spinner.FinalMSG = finalMessage
			return nil
		}

		// we can be sure they exist if the previous function ran without errors
		Logger.Debugf("Finding encrypted .kanuka files")
		listOfKanukaFiles, err := secrets.FindEnvOrKanukaFiles(projectPath, []string{}, true)
		if err != nil {
			return Logger.ErrorfAndReturn("Failed to find environment files after encryption: %v", err)
		}

		formattedListOfFiles := utils.FormatPaths(listOfKanukaFiles)
		Logger.Infof("Encrypt command completed successfully. Created %d .kanuka files", len(listOfKanukaFiles))

		finalMessage := color.GreenString("✓") + " Environment files encrypted successfully!\n" +
			"The following files were created: " + formattedListOfFiles +
			color.CyanString("→") + " You can now safely commit all " + color.YellowString(".kanuka") + " files to version control"

		spinner.FinalMSG = finalMessage
		return nil
	},
}
