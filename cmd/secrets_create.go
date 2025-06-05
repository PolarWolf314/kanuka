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
	createCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	createCmd.Flags().BoolVarP(&force, "force", "f", false, "force key creation")
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates and adds your public key, and gives instructions on how to gain access",
	Run: func(cmd *cobra.Command, args []string) {
		spinner, cleanup := startSpinner("Creating Kanuka file...", verbose)
		defer cleanup()

		if err := configs.InitProjectSettings(); err != nil {
			printError("failed to init project settings", err)
			return
		}
		projectPath := configs.ProjectKanukaSettings.ProjectPath

		if projectPath == "" {
			finalMessage := color.RedString("✗") + " Kanuka has not been initialized\n" +
				color.CyanString("→") + " Please run " + color.YellowString("kanuka secrets init") + " instead\n"
			spinner.FinalMSG = finalMessage
			return
		}

		if err := secrets.EnsureUserSettings(); err != nil {
			printError("Failed ensuring user settings", err)
			return
		}

		currentUsername := configs.UserKanukaSettings.Username
		// If force flag is active, then ignore checking for existing symmetric key
		if !force {
			projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath
			userPublicKeyPath := filepath.Join(projectPublicKeyPath, currentUsername+".pub")

			// We are explicitly ignoring errors, because an error means the key doesn't exist, which is what we want.
			userPublicKey, _ := secrets.LoadPublicKey(userPublicKeyPath)

			if userPublicKey != nil {
				finalMessage := color.RedString("✗ ") + color.YellowString(currentUsername+".pub ") + "already exists\n" +
					"To override, run: " + color.YellowString("kanuka secrets create --force\n")
				spinner.FinalMSG = finalMessage
				return
			}
		}

		if err := secrets.CreateAndSaveRSAKeyPair(verbose); err != nil {
			printError("Failed to generate and save RSA key pair", err)
			return
		}

		destPath, err := secrets.CopyUserPublicKeyToProject()
		if err != nil {
			printError("Failed to copy public key to project", err)
			return
		}

		didKanukaExist := true

		username := configs.UserKanukaSettings.Username
		projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath
		userKanukaKeyPath := filepath.Join(projectSecretsPath, username+".kanuka")

		if err := os.Remove(userKanukaKeyPath); err != nil {
			didKanukaExist = false
			// Explicity ignore error as we want to idempotently delete the file
			_ = err
		}

		deletedMessage := ""
		if didKanukaExist {
			deletedMessage = "    deleted: " + color.RedString(userKanukaKeyPath) + "\n"
		}

		finalMessage := color.GreenString("✓") + " The following changes were made:\n" +
			"    created: " + color.YellowString(destPath) + "\n" + deletedMessage +
			color.CyanString("To gain access to the secrets in this project:\n") +
			"  1. " + color.WhiteString("Commit your") + color.YellowString(" .kanuka/public_keys/"+currentUsername+".pub ") + color.WhiteString("file to your version control system\n") +
			"  2. " + color.WhiteString("Ask someone with permissions to grant you access with:\n") +
			"     " + color.YellowString("kanuka secrets add "+currentUsername+"\n")

		spinner.FinalMSG = finalMessage
	},
}
