package cmd

import (
	"os"
	"path/filepath"
	"time"

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
		spinner, cleanup := startSpinner("Creating Kānuka file...", verbose)
		defer cleanup()

		Logger.Debugf("Initializing project settings")
		if err := configs.InitProjectSettings(); err != nil {
			return Logger.ErrorfAndReturn("failed to init project settings: %v", err)
		}
		projectPath := configs.ProjectKanukaSettings.ProjectPath
		Logger.Debugf("Project path: %s", projectPath)

		if projectPath == "" {
			finalMessage := color.RedString("✗") + " Kānuka has not been initialized\n" +
				color.CyanString("→") + " Run " + color.YellowString("kanuka secrets init") + " instead"
			spinner.FinalMSG = finalMessage
			return nil
		}

		Logger.Debugf("Ensuring user settings")
		if err := secrets.EnsureUserSettings(); err != nil {
			return Logger.ErrorfAndReturn("Failed ensuring user settings: %v", err)
		}

		// Ensure user config has UUID
		Logger.Debugf("Ensuring user config with UUID")
		userConfig, err := configs.EnsureUserConfig()
		if err != nil {
			return Logger.ErrorfAndReturn("Failed to ensure user config: %v", err)
		}
		userUUID := userConfig.User.UUID
		Logger.Debugf("Current user UUID: %s", userUUID)

		currentUsername := configs.UserKanukaSettings.Username

		// If force flag is active, then ignore checking for existing symmetric key
		if !force {
			Logger.Debugf("Force flag not set, checking for existing public key")
			projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath
			// Check for public key using user UUID
			userPublicKeyPath := filepath.Join(projectPublicKeyPath, userUUID+".pub")
			Logger.Debugf("Checking for existing public key at: %s", userPublicKeyPath)

			// We are explicitly ignoring errors, because an error means the key doesn't exist, which is what we want.
			userPublicKey, _ := secrets.LoadPublicKey(userPublicKeyPath)

			if userPublicKey != nil {
				finalMessage := color.RedString("✗ ") + color.YellowString(userUUID+".pub ") + "already exists\n" +
					"To override, run: " + color.YellowString("kanuka secrets create --force")
				spinner.FinalMSG = finalMessage
				return nil
			}
		} else {
			Logger.Infof("Force flag set, will override existing keys if present")
			spinner.Stop()
			Logger.WarnfUser("Using --force flag will overwrite existing keys - ensure you have backups")
			spinner.Restart()
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

		// Update project config with user info
		Logger.Debugf("Updating project config with user info")
		projectConfig, err := configs.LoadProjectConfig()
		if err != nil {
			return Logger.ErrorfAndReturn("Failed to load project config: %v", err)
		}

		// Add/update user in project config
		projectConfig.Users[userUUID] = userConfig.User.Email
		projectConfig.Devices[userUUID] = configs.DeviceConfig{
			Email:     userConfig.User.Email,
			Name:      currentUsername, // Use system username as device name
			CreatedAt: time.Now().UTC(),
		}

		if err := configs.SaveProjectConfig(projectConfig); err != nil {
			return Logger.ErrorfAndReturn("Failed to save project config: %v", err)
		}
		Logger.Infof("Project config updated successfully")

		didKanukaExist := true

		projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath
		// Use user UUID for kanuka key path
		userKanukaKeyPath := filepath.Join(projectSecretsPath, userUUID+".kanuka")
		Logger.Debugf("Attempting to remove existing kanuka key at: %s", userKanukaKeyPath)

		if err := os.Remove(userKanukaKeyPath); err != nil {
			didKanukaExist = false
			Logger.Debugf("No existing kanuka key found (this is expected for new users)")
			// Explicitly ignore error as we want to idempotently delete the file
			_ = err
		} else {
			Logger.Infof("Removed existing kanuka key file")
		}

		deletedMessage := ""
		if didKanukaExist {
			deletedMessage = "    deleted: " + color.RedString(userKanukaKeyPath) + "\n"
		}

		Logger.Infof("Create command completed successfully for user: %s (UUID: %s)", currentUsername, userUUID)
		finalMessage := color.GreenString("✓") + " The following changes were made:\n" +
			"    created: " + color.YellowString(destPath) + "\n" + deletedMessage +
			color.CyanString("To gain access to the secrets in this project:\n") +
			"  1. " + color.WhiteString("Commit your") + color.YellowString(" .kanuka/public_keys/"+userUUID+".pub ") + color.WhiteString("file to your version control system\n") +
			"  2. " + color.WhiteString("Ask someone with permissions to grant you access with:\n") +
			"     " + color.YellowString("kanuka secrets register --user "+userConfig.User.Email)

		spinner.FinalMSG = finalMessage
		return nil
	},
}
