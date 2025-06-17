package cmd

import (
	"os"
	"path/filepath"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/secrets"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var force bool

func init() {
	createCmd.Flags().BoolVarP(&force, "force", "f", false, "force key creation")
}

// resetCreateCommandState resets the create command's global state for testing.
func resetCreateCommandState() {
	force = false
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates and adds your public key, and gives instructions on how to gain access",
	RunE: func(cmd *cobra.Command, args []string) error {
		Logger.Infof("Starting create command")
		spinner, cleanup := startSpinner("Creating Kanuka file...", verbose)
		defer cleanup()

		Logger.Debugf("Initializing project settings")
		if err := configs.InitProjectSettings(); err != nil {
			return Logger.ErrorfAndReturn("failed to init project settings: %v", err)
		}
		projectPath := configs.ProjectKanukaSettings.ProjectPath
		Logger.Debugf("Project path: %s", projectPath)

		if projectPath == "" {
			finalMessage := color.RedString("✗") + " Kanuka has not been initialized\n" +
				color.CyanString("→") + " Please run " + color.YellowString("kanuka secrets init") + " instead\n"
			spinner.FinalMSG = finalMessage
			return nil
		}

		Logger.Debugf("Ensuring user settings")
		if err := secrets.EnsureUserSettings(); err != nil {
			return Logger.ErrorfAndReturn("Failed ensuring user settings: %v", err)
		}

		currentUsername := configs.UserKanukaSettings.Username
		Logger.Debugf("Current username: %s", currentUsername)

		// If force flag is active, then ignore checking for existing symmetric key
		if !force {
			Logger.Debugf("Force flag not set, checking for existing public key")
			projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath
			userPublicKeyPath := filepath.Join(projectPublicKeyPath, currentUsername+".pub")
			Logger.Debugf("Checking for existing public key at: %s", userPublicKeyPath)

			// We are explicitly ignoring errors, because an error means the key doesn't exist, which is what we want.
			userPublicKey, _ := secrets.LoadPublicKey(userPublicKeyPath)

			if userPublicKey != nil {
				finalMessage := color.RedString("✗ ") + color.YellowString(currentUsername+".pub ") + "already exists\n" +
					"To override, run: " + color.YellowString("kanuka secrets create --force\n")
				spinner.FinalMSG = finalMessage
				return nil
			}
		} else {
			Logger.Infof("Force flag set, will override existing keys if present")
			Logger.WarnfUser("Using --force flag will overwrite existing keys - ensure you have backups")
		}

		Logger.Debugf("Creating and saving RSA key pair")
		if err := secrets.CreateAndSaveRSAKeyPair(verbose); err != nil {
			return Logger.ErrorfAndReturn("Failed to generate and save RSA key pair: %v", err)
		}
		Logger.Infof("RSA key pair created successfully")

		Logger.Debugf("Copying user public key to project")
		destPath, err := secrets.CopyUserPublicKeyToProject()
		if err != nil {
			return Logger.ErrorfAndReturn("Failed to copy public key to project: %v", err)
		}
		Logger.Infof("Public key copied to: %s", destPath)

		didKanukaExist := true

		username := configs.UserKanukaSettings.Username
		projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath
		userKanukaKeyPath := filepath.Join(projectSecretsPath, username+".kanuka")
		Logger.Debugf("Attempting to remove existing kanuka key at: %s", userKanukaKeyPath)

		if err := os.Remove(userKanukaKeyPath); err != nil {
			didKanukaExist = false
			Logger.Debugf("No existing kanuka key found (this is expected for new users)")
			// Explicity ignore error as we want to idempotently delete the file
			_ = err
		} else {
			Logger.Infof("Removed existing kanuka key file")
		}

		deletedMessage := ""
		if didKanukaExist {
			deletedMessage = "    deleted: " + color.RedString(userKanukaKeyPath) + "\n"
		}

		Logger.Infof("Create command completed successfully for user: %s", currentUsername)
		finalMessage := color.GreenString("✓") + " The following changes were made:\n" +
			"    created: " + color.YellowString(destPath) + "\n" + deletedMessage +
			color.CyanString("To gain access to the secrets in this project:\n") +
			"  1. " + color.WhiteString("Commit your") + color.YellowString(" .kanuka/public_keys/"+currentUsername+".pub ") + color.WhiteString("file to your version control system\n") +
			"  2. " + color.WhiteString("Ask someone with permissions to grant you access with:\n") +
			"     " + color.YellowString("kanuka secrets add "+currentUsername+"\n")

		spinner.FinalMSG = finalMessage
		return nil
	},
}
