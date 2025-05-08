package cmd

import (
	"fmt"
	"kanuka/internal/secrets"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func init() {
	initCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initializes the secrets store",
	Run: func(cmd *cobra.Command, args []string) {
		_, cleanup := startSpinner("Initialising Kanuka...", verbose)
		defer cleanup()

		kanukaExists, err := secrets.DoesProjectKanukaSettingsExist()
		if err != nil {
			printError("Failed to check if project kanuka settings exists", err)
			return
		}
		if kanukaExists {
			printError(".kanuka/ already exists", fmt.Errorf("please use `kanuka secrets create` instead"))
			return
		}

		verboseLog("Starting Kanuka initialization...")

		if err := secrets.EnsureUserSettings(); err != nil {
			printError("Failed ensuring user settings", err)
			return
		}

		if err := secrets.EnsureKanukaSettings(); err != nil {
			printError("Failed to create .kanuka folders", err)
			return
		}
		verboseLog("✅ Created .kanuka folders")

		if err := secrets.CreateAndSaveRSAKeyPair(verbose); err != nil {
			printError("Failed to generate and save RSA key pair", err)
			return
		}

		destPath, err := secrets.CopyUserPublicKeyToProject()
		if err != nil {
			printError("Failed to copy public key to project", err)
			return
		}
		verboseLog(fmt.Sprintf("✅ Copied public key into %s", destPath))

		if err := secrets.CreateAndSaveEncryptedSymmetricKey(verbose); err != nil {
			printError("Failed to create encrypted symmetric key", err)
			return
		}

		fmt.Println(color.GreenString("✓") + " Kanuka initialized successfully!")
		fmt.Println(color.CyanString("→") + " Run 'kanuka secrets encrypt' to encrypt your existing .env files")
	},
}
