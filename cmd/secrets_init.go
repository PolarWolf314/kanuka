package cmd

import (
	"github.com/PolarWolf314/kanuka/internal/secrets"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func init() {
	initCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initializes the secrets store",
	Run: func(cmd *cobra.Command, args []string) {
		spinner, cleanup := startSpinner("Initializing Kanuka...", verbose)
		defer cleanup()

		kanukaExists, err := secrets.DoesProjectKanukaSettingsExist()
		if err != nil {
			printError("Failed to check if project kanuka settings exists", err)
			return
		}
		if kanukaExists {
			finalMessage := color.RedString("✗") + " Kanuka has already been initialized\n" +
				color.CyanString("→") + " Please run " + color.YellowString("kanuka secrets create") + " instead\n"
			spinner.FinalMSG = finalMessage
			return
		}

		if err := secrets.EnsureUserSettings(); err != nil {
			printError("Failed ensuring user settings", err)
			return
		}

		if err := secrets.EnsureKanukaSettings(); err != nil {
			printError("Failed to create .kanuka folders", err)
			return
		}

		if err := secrets.CreateAndSaveRSAKeyPair(verbose); err != nil {
			printError("Failed to generate and save RSA key pair", err)
			return
		}

		destPath, err := secrets.CopyUserPublicKeyToProject()
		_ = destPath // explicity ignore destPath for now
		if err != nil {
			printError("Failed to copy public key to project", err)
			return
		}

		if err := secrets.CreateAndSaveEncryptedSymmetricKey(verbose); err != nil {
			printError("Failed to create encrypted symmetric key", err)
			return
		}

		finalMessage := color.GreenString("✓") + " Kanuka initialized successfully!\n" +
			color.CyanString("→") + " Run " + color.YellowString("kanuka secrets encrypt") + " to encrypt your existing .env files\n"

		spinner.FinalMSG = finalMessage
	},
}
