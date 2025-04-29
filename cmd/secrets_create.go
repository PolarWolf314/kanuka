package cmd

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates and adds your public key, and gives instructions on how to gain access",
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("Adding your public key...")

		// Step 1: Get current working directory
		wd, err := os.Getwd()
		if err != nil {
			log.Fatalf("❌ Failed to get working directory: %v", err)
		}
		projectName := filepath.Base(wd)
		log.Printf("📂 Current project: %s\n", projectName)

		currentUser, err := user.Current()
		if err != nil {
			log.Fatalf("❌ Failed to get current user: %v", err)
		}
		username := currentUser.Username

		// Step 2: Ensure ~/.kanuka/keys exists
		keysDir := filepath.Join(currentUser.HomeDir, ".kanuka", "keys")
		if err := os.MkdirAll(keysDir, 0700); err != nil {
			log.Fatalf("❌ Failed to create keys directory: %v", err)
		}

		// Generate key pair
		privateKeyPath := filepath.Join(keysDir, projectName)
		publicKeyPath := privateKeyPath + ".pub"

		if err := generateRSAKeyPair(privateKeyPath, publicKeyPath); err != nil {
			log.Fatalf("❌ Failed to generate RSA key pair: %v", err)
		}
		log.Println("✅ Generated RSA public/private key pair")

		// Step 3: Check if .kanuka folder exists
		// TODO: Check that a user doesn't already exist, or at some point use a yaml/toml
		kanukaDir := filepath.Join(wd, ".kanuka")
		publicKeysDir := filepath.Join(kanukaDir, "public_keys")

		if _, err := os.Stat(kanukaDir); os.IsNotExist(err) {
			log.Fatalf("❌ .kanuka folder does not exist! Run kanuka secrets init instead")
			return
		}

		// Step 4: Copy public key into project
		destPublicKey := filepath.Join(publicKeysDir, fmt.Sprintf("%s.pub", username))
		if err := copyFile(publicKeyPath, destPublicKey); err != nil {
			log.Fatalf("❌ Failed to copy public key into project: %v", err)
		}
		log.Println("✅ Copied public key into project")

		// Step 5: Give instructions for how to add access
		log.Println()
		log.Println("✨ Your public key has been added!")
		log.Println("To gain access to the secrets in this project, do the following:")
		log.Println("1. Commit your `.kanuka/public_keys/" + username + ".pub` file to Git.")
		log.Println("2. Ask someone with access to encrypt the symmetric key for you using kanuka secrets add " + username)
	},
}
