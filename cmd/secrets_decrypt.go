package cmd

import (
	"kanuka/internal/secrets"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var decryptCmd = &cobra.Command{
	Use:   "decrypt",
	Short: "Decrypts the .env.kanuka file back into .env using your Kanuka key",
	Run: func(cmd *cobra.Command, args []string) {
		kanukaExists, err := secrets.DoesProjectKanukaSettingsExist()
		if err != nil {
			log.Fatalf("❌ Failed to check if project kanuka settings exists: %v", err)
		}
		if !kanukaExists {
			log.Fatalf("❌ .kanuka/ doesn't exist. Please init the project")
		}

		log.Println("🚀 Starting decryption process...")

		// Step 1: Check for .env.kanuka file
		workingDirectory, err := os.Getwd()
		if err != nil {
			log.Fatalf("❌ Failed to get working directory: %v", err)
		}

		// TODO: In future, add config options to list which dirs to ignore. .kanuka/ ignored by default
		listOfKanukaFiles, err := secrets.FindEnvOrKanukaFiles(workingDirectory, []string{}, true)
		if err != nil {
			log.Fatalf("❌ Failed to find environment files: %v", err)
		}
		if len(listOfKanukaFiles) == 0 {
			log.Fatalf("❌ No environment files found in %v", workingDirectory)
		}

		// Step 2: Get project's encrypted symmetric key
		encryptedSymKey, err := secrets.GetUserProjectKanukaKey()
		if err != nil {
			log.Fatalf("❌ Failed to get user's .kanuka file: %v", err)
		}
		log.Println("🔑 Loaded user's .kanuka key")

		privateKey, err := secrets.GetUserPrivateKey()
		if err != nil {
			log.Fatalf("❌ Failed to get user's private key: %v", err)
		}
		log.Println("🔑 Loaded user's private key")

		// Step 3: Decrypt user's kanuka file (get symmetric key)
		symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
		if err != nil {
			log.Fatalf("❌ Failed to decrypt symmetric key: %v", err)
		}
		log.Println("🔓 Decrypted symmetric key")

		// Step 4: Decrypt all .kanuka files
		if err := secrets.DecryptFiles(symKey, listOfKanukaFiles); err != nil {
			log.Fatalf("❌ Failed to encrypt environment files: %v", err)
		}
		// Above method handles logging
	},
}
