package cmd

import (
	"fmt"
	"kanuka/internal/secrets"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/nacl/secretbox"
)

var decryptCmd = &cobra.Command{
	Use:   "decrypt",
	Short: "Decrypts the .env.kanuka file back into .env using your Kanuka key",
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("🚀 Starting decryption process...")

		// Step 1: Check for .env.kanuka file
		envKanukaPath := ".env.kanuka"
		if _, err := os.Stat(envKanukaPath); os.IsNotExist(err) {
			log.Fatalf("❌ .env.kanuka file not found")
		}
		log.Println("✅ Found .env.kanuka file")

		// Step 2: Find user kanuka file
		currentUser, err := user.Current()
		if err != nil {
			log.Fatalf("❌ Could not get current user: %v", err)
		}
		userKeyFile := filepath.Join(".kanuka", "secrets", fmt.Sprintf("%s.kanuka", currentUser.Username))
		if _, err := os.Stat(userKeyFile); os.IsNotExist(err) {
			log.Fatalf("❌ Kanuka user key file not found: %s", userKeyFile)
		}
		log.Println("✅ Found user's .kanuka file")

		// Step 3: Find private key
		wd, err := os.Getwd()
		if err != nil {
			log.Fatalf("❌ Failed to get working directory: %v", err)
		}
		projectName := filepath.Base(wd)
		log.Printf("📂 Current project: %s\n", projectName)

		privateKeyPath := filepath.Join(currentUser.HomeDir, ".kanuka", "keys", projectName)
		if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
			log.Fatalf("❌ Private key not found at: %s", privateKeyPath)
		}
		log.Println("✅ Found private key")

		// Step 4: Load private key
		privateKey, err := secrets.LoadPrivateKey(privateKeyPath)
		if err != nil {
			log.Fatalf("❌ Failed to load private key: %v", err)
		}
		log.Println("🔑 Loaded private key")

		// Step 5: Decrypt user's kanuka file (get symmetric key)
		encryptedSymKey, err := os.ReadFile(userKeyFile)
		if err != nil {
			log.Fatalf("❌ Failed to read user key file: %v", err)
		}

		symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
		if err != nil {
			log.Fatalf("❌ Failed to decrypt symmetric key: %v", err)
		}
		log.Println("🔓 Decrypted symmetric key")

		// Step 6: Decrypt .env.kanuka file using secretbox
		outputPath := ".env"
		if err := decryptFile(symKey, envKanukaPath, outputPath); err != nil {
			log.Fatalf("❌ Failed to decrypt .env.kanuka file: %v", err)
		}

		log.Println("✅ .env.kanuka successfully decrypted to .env 🎉")
	},
}

// ===== Helper functions =====

func decryptFile(symKey []byte, inputPath, outputPath string) error {
	if len(symKey) != 32 {
		return fmt.Errorf("symmetric key must be 32 bytes for secretbox")
	}

	var key [32]byte
	copy(key[:], symKey)

	ciphertext, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}

	var nonce [24]byte
	copy(nonce[:], ciphertext[:24])

	plaintext, ok := secretbox.Open(nil, ciphertext[24:], &nonce, &key)
	if !ok {
		return fmt.Errorf("failed to decrypt ciphertext with secretbox")
	}

	// #nosec G306 -- We want the decrypted .env file to be editable by the user
	return os.WriteFile(outputPath, plaintext, 0644)
}
