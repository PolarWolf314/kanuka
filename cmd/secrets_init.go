package cmd

import (
	"fmt"
	"kanuka/internal/secrets"
	"log"
	"os"
	"time"

	"github.com/briandowns/spinner"
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
		// Create a new spinner
		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Suffix = " Initializing Kanuka..."
		err := s.Color("cyan")
		if err != nil {
			printError("Failed to create a spinner", err)
		}

		// Only show spinner if not in verbose mode
		if !verbose {
			s.Start()
			// Ensure log output is discarded unless in verbose mode
			log.SetOutput(os.NewFile(0, os.DevNull))
		}

		// Function to run at the end to restore logging and stop spinner
		defer func() {
			if !verbose {
				log.SetOutput(os.Stdout)
				s.Stop()
			}
		}()

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

		if !verbose {
			s.Stop()
		}

		fmt.Println(color.GreenString("✓") + " Kanuka initialized successfully!")
		fmt.Println(color.CyanString("→") + " Run 'kanuka secrets encrypt' to encrypt your existing .env files")
	},
}
