package cmd

import (
	"github.com/PolarWolf314/kanuka/internal/secrets"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initializes the secrets store",
	RunE: func(cmd *cobra.Command, args []string) error {
		Logger.Infof("Starting init command")
		spinner, cleanup := startSpinner("Initializing Kānuka...", verbose)
		defer cleanup()

		Logger.Debugf("Checking if project kanuka settings already exist")
		kanukaExists, err := secrets.DoesProjectKanukaSettingsExist()
		if err != nil {
			return Logger.ErrorfAndReturn("Failed to check if project kanuka settings exists: %v", err)
		}
		if kanukaExists {
			finalMessage := color.RedString("✗") + " Kānuka has already been initialized\n" +
				color.CyanString("→") + " Please run " + color.YellowString("kanuka secrets create") + " instead"
			spinner.FinalMSG = finalMessage
			return nil
		}

		Logger.Debugf("Ensuring user settings")
		if err := secrets.EnsureUserSettings(); err != nil {
			return Logger.ErrorfAndReturn("Failed ensuring user settings: %v", err)
		}
		Logger.Infof("User settings ensured successfully")

		Logger.Debugf("Ensuring kanuka settings and creating .kanuka folders")
		if err := secrets.EnsureKanukaSettings(); err != nil {
			return Logger.ErrorfAndReturn("Failed to create .kanuka folders: %v", err)
		}
		Logger.Infof("Kanuka settings and folders created successfully")

		Logger.Debugf("Creating and saving RSA key pair")
		if err := secrets.CreateAndSaveRSAKeyPair(verbose); err != nil {
			return Logger.ErrorfAndReturn("Failed to generate and save RSA key pair: %v", err)
		}
		Logger.Infof("RSA key pair created and saved successfully")

		Logger.Debugf("Copying user public key to project")
		destPath, err := secrets.CopyUserPublicKeyToProject()
		_ = destPath // explicity ignore destPath for now
		if err != nil {
			return Logger.ErrorfAndReturn("Failed to copy public key to project: %v", err)
		}
		Logger.Infof("User public key copied to project successfully")

		Logger.Debugf("Creating and saving encrypted symmetric key")
		if err := secrets.CreateAndSaveEncryptedSymmetricKey(verbose); err != nil {
			return Logger.ErrorfAndReturn("Failed to create encrypted symmetric key: %v", err)
		}
		Logger.Infof("Encrypted symmetric key created and saved successfully")

		Logger.Infof("Init command completed successfully")

		spinner.Stop()
		// Security reminder about .env files
		Logger.WarnfUser("Remember to never commit .env files to version control - only commit .kanuka files")
		spinner.Restart()

		finalMessage := color.GreenString("✓") + " Kānuka initialized successfully!\n" +
			color.CyanString("→") + " Run " + color.YellowString("kanuka secrets encrypt") + " to encrypt your existing .env files"

		spinner.FinalMSG = finalMessage
		return nil
	},
}
