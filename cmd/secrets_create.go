package cmd

import (
	"kanuka/internal/secrets"
	"log"

	"github.com/spf13/cobra"
)

func init() {
	createCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates and adds your public key, and gives instructions on how to gain access",
	Run: func(cmd *cobra.Command, args []string) {
		kanukaExists, err := secrets.DoesProjectKanukaSettingsExist()
		if err != nil {
			log.Fatalf("❌ Failed to check if project kanuka settings exists: %v", err)
		}
		if !kanukaExists {
			log.Fatalf("❌ .kanuka/ doesn't exist. Please init the project first")
		}

		log.Println("Adding your public key...")
		if err := secrets.EnsureUserSettings(); err != nil {
			log.Fatalf("❌ Failed ensuring user settings: %v", err)
		}

		if err := secrets.CreateAndSaveRSAKeyPair(); err != nil {
			log.Fatalf("❌ Failed to generate and save RSA key pair: %v", err)
		}
		// Above method handles printing comments

		destPath, err := secrets.CopyUserPublicKeyToProject()
		if err != nil {
			log.Fatalf("❌ Failed to copy public key to project: %v", err)
		}

		log.Printf("✅ Copied public key into %s", destPath)

		username, err := secrets.GetUsername()
		if err != nil {
			log.Fatalf("❌ Failed to get username: %v", err)
		}

		log.Println()
		log.Println("✨ Your public key has been added!")
		log.Println("To gain access to the secrets in this project, do the following:")
		log.Println("    1. Commit your `.kanuka/public_keys/" + username + ".pub` file to Git.")
		log.Println("    2. Ask someone with permissions to grant you access with `kanuka secrets add " + username + "`")
	},
}
