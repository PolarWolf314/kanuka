package cmd

import (
	"kanuka/internal/secrets"
	"log"

	"github.com/spf13/cobra"
)

func init() {
	initCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initializes the secrets store",
	Run: func(cmd *cobra.Command, args []string) {
		kanukaExists, err := secrets.DoesProjectKanukaSettingsExist()
		if err != nil {
			log.Fatalf("❌ Failed to check if project kanuka settings exists: %v", err)
		}
		if kanukaExists {
			log.Fatalf("❌ .kanuka/ already exists. Please use `kanuka secrets create` instead")
		}

		log.Println("Starting Kanuka initialization...")

		if err := secrets.EnsureUserSettings(); err != nil {
			log.Fatalf("❌ Failed ensuring user settings: %v", err)
		}

		if err := secrets.EnsureKanukaSettings(); err != nil {
			log.Fatalf("❌ Failed to create .kanuka folders: %v", err)
		}
		log.Println("✅ Created .kanuka folders")

		if err := secrets.CreateAndSaveRSAKeyPair(); err != nil {
			log.Fatalf("❌ Failed to generate and save RSA key pair: %v", err)
		}
		// Above method handles printing comments

		destPath, err := secrets.CopyUserPublicKeyToProject()
		if err != nil {
			log.Fatalf("❌ Failed to copy public key to project: %v", err)
		}

		log.Printf("✅ Copied public key into %s", destPath)

		if err := secrets.CreateAndSaveEncryptedSymmetricKey(); err != nil {
			log.Fatalf("❌ Failed to create encrypted symmetric key: %v", err)
		}
		// Above method handles printing comments

		log.Println()
		log.Println("✨ Initialization complete!")
		log.Println("Go ahead and run `kanuka secrets encrypt` to encrypt your existing `.env` files!")
	},
}
