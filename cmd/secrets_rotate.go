package cmd

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PolarWolf314/kanuka/internal/audit"
	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/secrets"
	"github.com/PolarWolf314/kanuka/internal/ui"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
)

var (
	rotateForce bool
)

func init() {
	rotateCmd.Flags().BoolVar(&rotateForce, "force", false, "skip confirmation prompt")
}

// resetRotateCommandState resets the rotate command's global state for testing.
func resetRotateCommandState() {
	rotateForce = false
}

// confirmRotate prompts the user to confirm the keypair rotation.
// Returns true if the user confirms, false otherwise.
func confirmRotate(s *spinner.Spinner) bool {
	s.Stop()

	fmt.Printf("\n%s This will generate a new keypair and replace your current one.\n", ui.Warning.Sprint("Warning:"))
	fmt.Println("  Your old private key will no longer work for this project.")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Do you want to continue? [y/N]: ")
	response, err := reader.ReadString('\n')
	if err != nil {
		Logger.Errorf("Failed to read response: %v", err)
		s.Restart()
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))

	s.Restart()
	return response == "y" || response == "yes"
}

var rotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Rotate your keypair for this project",
	Long: `Generates a new keypair and replaces your current one.

This command is useful when you want to rotate your keys for security purposes,
such as when your private key may have been compromised.

The command will:
  1. Generate a new RSA keypair
  2. Decrypt the symmetric key with your old private key
  3. Re-encrypt the symmetric key with your new public key
  4. Update your public key in the project
  5. Save the new private key to your key directory

After running this command:
  - Your old private key will no longer work for this project
  - Other users do NOT need to take any action
  - You should commit the updated .kanuka/public_keys/<uuid>.pub file

Examples:
  # Rotate your keypair (with confirmation prompt)
  kanuka secrets rotate

  # Rotate without confirmation prompt
  kanuka secrets rotate --force`,
	RunE: func(cmd *cobra.Command, args []string) error {
		Logger.Infof("Starting rotate command")
		spinner, cleanup := startSpinner("Rotating keypair...", verbose)
		defer cleanup()

		Logger.Debugf("Initializing project settings")
		if err := configs.InitProjectSettings(); err != nil {
			return Logger.ErrorfAndReturn("failed to init project settings: %v", err)
		}

		projectPath := configs.ProjectKanukaSettings.ProjectPath
		Logger.Debugf("Project path: %s", projectPath)

		if projectPath == "" {
			finalMessage := ui.Error.Sprint("✗") + " Kanuka has not been initialized\n" +
				ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " instead"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Ensure user config has UUID
		Logger.Debugf("Ensuring user config with UUID")
		userConfig, err := configs.EnsureUserConfig()
		if err != nil {
			return Logger.ErrorfAndReturn("Failed to ensure user config: %v", err)
		}
		userUUID := userConfig.User.UUID
		Logger.Debugf("Current user UUID: %s", userUUID)

		// Load project config
		Logger.Debugf("Loading project config")
		projectConfig, err := configs.LoadProjectConfig()
		if err != nil {
			return Logger.ErrorfAndReturn("Failed to load project config: %v", err)
		}
		projectUUID := projectConfig.Project.UUID
		Logger.Debugf("Project UUID: %s", projectUUID)

		// Check if user has access to this project
		projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath
		userKanukaKeyPath := filepath.Join(projectSecretsPath, userUUID+".kanuka")

		if _, err := os.Stat(userKanukaKeyPath); os.IsNotExist(err) {
			finalMessage := ui.Error.Sprint("✗") + " You don't have access to this project\n" +
				ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets create") + " and ask someone to register you"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Load current private key
		privateKeyPath := configs.GetPrivateKeyPath(projectUUID)
		Logger.Debugf("Loading private key from: %s", privateKeyPath)

		oldPrivateKey, err := secrets.LoadPrivateKey(privateKeyPath)
		if err != nil {
			finalMessage := ui.Error.Sprint("✗") + " Couldn't load your private key from " + ui.Path.Sprint(privateKeyPath) + "\n\n" +
				ui.Error.Sprint("Error: ") + err.Error()
			spinner.FinalMSG = finalMessage
			return nil
		}
		Logger.Infof("Old private key loaded successfully")

		// Get and decrypt symmetric key
		Logger.Debugf("Getting encrypted symmetric key")
		encryptedSymKey, err := secrets.GetProjectKanukaKey(userUUID)
		if err != nil {
			finalMessage := ui.Error.Sprint("✗") + " Couldn't get your Kanuka key from " + ui.Path.Sprint(userKanukaKeyPath) + "\n\n" +
				ui.Error.Sprint("Error: ") + err.Error()
			spinner.FinalMSG = finalMessage
			return nil
		}

		Logger.Debugf("Decrypting symmetric key with old private key")
		symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, oldPrivateKey)
		if err != nil {
			finalMessage := ui.Error.Sprint("✗") + " Failed to decrypt your Kanuka key\n\n" +
				ui.Error.Sprint("Error: ") + err.Error()
			spinner.FinalMSG = finalMessage
			return nil
		}
		Logger.Infof("Symmetric key decrypted successfully")

		// Confirmation prompt (unless --force)
		if !rotateForce {
			if !confirmRotate(spinner) {
				spinner.FinalMSG = ui.Warning.Sprint("⚠") + " Keypair rotation cancelled."
				return nil
			}
		}

		// Generate new keypair
		Logger.Debugf("Generating new RSA keypair")
		newPrivateKey, newPublicKey, err := generateNewKeypair()
		if err != nil {
			return Logger.ErrorfAndReturn("Failed to generate new keypair: %v", err)
		}
		Logger.Infof("New keypair generated successfully")

		// Re-encrypt symmetric key with new public key
		Logger.Debugf("Encrypting symmetric key with new public key")
		newEncryptedSymKey, err := secrets.EncryptWithPublicKey(symKey, newPublicKey)
		if err != nil {
			return Logger.ErrorfAndReturn("Failed to encrypt symmetric key with new public key: %v", err)
		}
		Logger.Infof("Symmetric key re-encrypted successfully")

		// Save new private key to user's key directory
		Logger.Debugf("Saving new private key to: %s", privateKeyPath)
		if err := savePrivateKey(newPrivateKey, privateKeyPath); err != nil {
			return Logger.ErrorfAndReturn("Failed to save new private key: %v", err)
		}
		Logger.Infof("New private key saved successfully")

		// Save new public key to user's key directory
		publicKeyPath := configs.GetPublicKeyPath(projectUUID)
		Logger.Debugf("Saving new public key to: %s", publicKeyPath)
		if err := secrets.SavePublicKeyToFile(newPublicKey, publicKeyPath); err != nil {
			return Logger.ErrorfAndReturn("Failed to save new public key: %v", err)
		}
		Logger.Infof("New public key saved to user key directory")

		// Copy new public key to project
		projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath
		projectPubKeyPath := filepath.Join(projectPublicKeyPath, userUUID+".pub")
		Logger.Debugf("Copying new public key to project: %s", projectPubKeyPath)
		if err := secrets.SavePublicKeyToFile(newPublicKey, projectPubKeyPath); err != nil {
			return Logger.ErrorfAndReturn("Failed to copy public key to project: %v", err)
		}
		Logger.Infof("New public key copied to project")

		// Save new encrypted symmetric key
		Logger.Debugf("Saving new encrypted symmetric key")
		if err := secrets.SaveKanukaKeyToProject(userUUID, newEncryptedSymKey); err != nil {
			return Logger.ErrorfAndReturn("Failed to save new encrypted symmetric key: %v", err)
		}
		Logger.Infof("New encrypted symmetric key saved")

		// Update key metadata
		Logger.Debugf("Updating key metadata")
		metadata := &configs.KeyMetadata{
			ProjectName:    projectConfig.Project.Name,
			ProjectPath:    configs.ProjectKanukaSettings.ProjectPath,
			CreatedAt:      time.Now(),
			LastAccessedAt: time.Now(),
		}
		if err := configs.SaveKeyMetadata(projectUUID, metadata); err != nil {
			// Non-critical - just log the error
			Logger.Errorf("Failed to update key metadata: %v", err)
		}

		Logger.Infof("Keypair rotation completed successfully")

		// Log to audit trail.
		auditEntry := audit.LogWithUser("rotate")
		audit.Log(auditEntry)

		finalMessage := ui.Success.Sprint("✓") + " Keypair rotated successfully\n\n" +
			"Your new public key has been added to the project.\n" +
			"Other users do not need to take any action.\n\n" +
			ui.Info.Sprint("→") + " Commit the updated " + ui.Path.Sprint(".kanuka/public_keys/"+userUUID+".pub") + " file"
		spinner.FinalMSG = finalMessage
		return nil
	},
}

// generateNewKeypair generates a new RSA keypair.
func generateNewKeypair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}
	return privateKey, &privateKey.PublicKey, nil
}

// savePrivateKey saves an RSA private key to a file in PEM format.
func savePrivateKey(privateKey *rsa.PrivateKey, filePath string) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privPem := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}
	pemBytes := pem.EncodeToMemory(privPem)

	if err := os.WriteFile(filePath, pemBytes, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	return nil
}
