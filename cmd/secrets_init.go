package cmd

import (
	"github.com/PolarWolf314/kanuka/internal/secrets"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initializes the secrets store",
	Run: func(cmd *cobra.Command, args []string) {
		Logger.Infof("Starting init command")
		spinner, cleanup := startSpinner("Initializing Kanuka...", verbose)
		defer cleanup()

		Logger.Debugf("Checking if project kanuka settings already exist")
		kanukaExists, err := secrets.DoesProjectKanukaSettingsExist()
		if err != nil {
			Logger.Errorf("Failed to check if project kanuka settings exist: %v", err)
			printError("Failed to check if project kanuka settings exists", err)
			return
		}
		if kanukaExists {
			Logger.Warnf("Kanuka has already been initialized")
			finalMessage := color.RedString("✗") + " Kanuka has already been initialized\n" +
				color.CyanString("→") + " Please run " + color.YellowString("kanuka secrets create") + " instead\n"
			spinner.FinalMSG = finalMessage
			return
		}

		Logger.Debugf("Ensuring user settings")
		if err := secrets.EnsureUserSettings(); err != nil {
			Logger.Errorf("Failed to ensure user settings: %v", err)
			printError("Failed ensuring user settings", err)
			return
		}
		Logger.Infof("User settings ensured successfully")

		Logger.Debugf("Ensuring kanuka settings and creating .kanuka folders")
		if err := secrets.EnsureKanukaSettings(); err != nil {
			Logger.Errorf("Failed to create .kanuka folders: %v", err)
			printError("Failed to create .kanuka folders", err)
			return
		}
		Logger.Infof("Kanuka settings and folders created successfully")

		Logger.Debugf("Creating and saving RSA key pair")
		if err := secrets.CreateAndSaveRSAKeyPair(verbose); err != nil {
			Logger.Errorf("Failed to generate and save RSA key pair: %v", err)
			printError("Failed to generate and save RSA key pair", err)
			return
		}
		Logger.Infof("RSA key pair created and saved successfully")

		Logger.Debugf("Copying user public key to project")
		destPath, err := secrets.CopyUserPublicKeyToProject()
		_ = destPath // explicity ignore destPath for now
		if err != nil {
			Logger.Errorf("Failed to copy public key to project: %v", err)
			printError("Failed to copy public key to project", err)
			return
		}
		Logger.Infof("User public key copied to project successfully")

		Logger.Debugf("Creating and saving encrypted symmetric key")
		if err := secrets.CreateAndSaveEncryptedSymmetricKey(verbose); err != nil {
			Logger.Errorf("Failed to create encrypted symmetric key: %v", err)
			printError("Failed to create encrypted symmetric key", err)
			return
		}
		Logger.Infof("Encrypted symmetric key created and saved successfully")

		Logger.Infof("Init command completed successfully")
		finalMessage := color.GreenString("✓") + " Kanuka initialized successfully!\n" +
			color.CyanString("→") + " Run " + color.YellowString("kanuka secrets encrypt") + " to encrypt your existing .env files\n"

		spinner.FinalMSG = finalMessage
	},
}
