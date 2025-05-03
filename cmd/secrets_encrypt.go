package cmd

import (
	"crypto/rand"
	"fmt"
	"io"
	"kanuka/internal/secrets"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/nacl/secretbox"
)

var encryptCmd = &cobra.Command{
	Use:   "encrypt",
	Short: "Encrypts the .env file into .env.kanuka using your Kanuka key",
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("🚀 Starting encryption process...")

		// Step 1: Check for .env file
		envPath := ".env"
		if _, err := os.Stat(envPath); os.IsNotExist(err) {
			log.Fatalf("❌ .env file not found")
		}
		log.Println("✅ Found .env file")

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

		// Step 6: Encrypt .env file using secretbox
		outputPath := ".env.kanuka"
		if err := encryptFile(symKey, envPath, outputPath); err != nil {
			log.Fatalf("❌ Failed to encrypt .env file: %v", err)
		}

		log.Println("✅ .env successfully encrypted to .env.kanuka 🎉")
	},
}

// ===== Helper functions =====

func encryptFile(symKey []byte, inputPath, outputPath string) error {
	if len(symKey) != 32 {
		return fmt.Errorf("symmetric key must be 32 bytes for secretbox")
	}

	var key [32]byte
	copy(key[:], symKey)

	plaintext, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}

	var nonce [24]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return err
	}

	ciphertext := secretbox.Seal(nonce[:], plaintext, &nonce, &key)

	return os.WriteFile(outputPath, ciphertext, 0600)
}
