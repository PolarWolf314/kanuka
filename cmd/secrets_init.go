package cmd

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initializes the secrets store",
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("Starting Kanuka initialization...")

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
		kanukaDir := filepath.Join(wd, ".kanuka")
		secretsDir := filepath.Join(kanukaDir, "secrets")
		publicKeysDir := filepath.Join(kanukaDir, "public_keys")

		kanukaExists := true
		if _, err := os.Stat(kanukaDir); os.IsNotExist(err) {
			kanukaExists = false
			log.Println("📁 .kanuka folder does not exist, creating it...")

			if err := os.MkdirAll(secretsDir, 0755); err != nil {
				log.Fatalf("❌ Failed to create .kanuka/secrets: %v", err)
			}
			if err := os.MkdirAll(publicKeysDir, 0755); err != nil {
				log.Fatalf("❌ Failed to create .kanuka/public_keys: %v", err)
			}
			log.Println("✅ Created .kanuka folders")
		}

		// Step 4: Copy public key into project
		destPublicKey := filepath.Join(publicKeysDir, fmt.Sprintf("%s.pub", username))
		if err := copyFile(publicKeyPath, destPublicKey); err != nil {
			log.Fatalf("❌ Failed to copy public key into project: %v", err)
		}
		log.Println("✅ Copied public key into project")

		if !kanukaExists {
			// Step 5: Create symmetric key in memory
			symKey := make([]byte, 32) // AES-256
			if _, err := rand.Read(symKey); err != nil {
				log.Fatalf("❌ Failed to generate symmetric key: %v", err)
			}
			log.Println("🔐 Symmetric key generated")

			// Step 6: Encrypt symmetric key with user's public key
			pubKey, err := loadPublicKey(destPublicKey)
			if err != nil {
				log.Fatalf("❌ Failed to load project public key: %v", err)
			}
			encryptedSymKey, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, symKey)
			if err != nil {
				log.Fatalf("❌ Failed to encrypt symmetric key: %v", err)
			}
			log.Println("🔒 Encrypted symmetric key with project public key")

			// Step 7: Save encrypted symmetric key
			encryptedSymPath := filepath.Join(secretsDir, fmt.Sprintf("%s.kanuka", username))
			if err := os.WriteFile(encryptedSymPath, encryptedSymKey, 0644); err != nil {
				log.Fatalf("❌ Failed to save encrypted symmetric key: %v", err)
			}
			log.Println("✅ Saved encrypted symmetric key into project")
		}

		// Step 8: Run `kanuka encrypt`
		log.Println("🚀 Running `kanuka encrypt` automatically...")
		kanuka_executable_path, err := os.Executable()
		if err != nil {
			log.Fatalf("❌ Failed to get executable path for Kanuka: %v", err)
		}
		if err := exec.Command(kanuka_executable_path, "secrets encrypt").Run(); err != nil {
			log.Fatalf("❌ Failed to run encrypt command: %v", err)
		}
		log.Println("✅ Secrets encrypted successfully")

		// Step 9a: Give instructions for access
		log.Println()
		log.Println("✨ Initialization complete! To give access to others:")
		log.Println("1. Commit your `.kanuka/public_keys/" + username + ".pub` file to Git.")
		log.Println("2. Ask someone with access to encrypt the symmetric key for you.")
		log.Println()
	},
}

// ===== Helper functions =====

func generateRSAKeyPair(privatePath, publicPath string) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	// Save private key
	privFile, err := os.Create(privatePath)
	if err != nil {
		return err
	}
	defer privFile.Close()

	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privPem := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}
	if err := pem.Encode(privFile, privPem); err != nil {
		return err
	}

	// Save public key
	pubFile, err := os.Create(publicPath)
	if err != nil {
		return err
	}
	defer pubFile.Close()

	pubASN1, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}
	pubPem := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubASN1,
	}
	return pem.Encode(pubFile, pubPem)
}

func loadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("failed to decode PEM block containing public key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}
	return rsaPub, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
