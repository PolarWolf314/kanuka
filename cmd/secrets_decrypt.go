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
	decryptCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}

var decryptCmd = &cobra.Command{
	Use:   "decrypt",
	Short: "Decrypts the .env.kanuka file back into .env using your Kanuka key",
	Run: func(cmd *cobra.Command, args []string) {
		// Create a new spinner
		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Suffix = " Decrypting environment files..."
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
		if !kanukaExists {
			printError(".kanuka/ doesn't exist", fmt.Errorf("please init the project first with `kanuka init`"))
			return
		}

		verboseLog("🚀 Starting decryption process...")

		// Step 1: Check for .env.kanuka file
		workingDirectory, err := os.Getwd()
		if err != nil {
			printError("Failed to get working directory", err)
			return
		}

		// TODO: In future, add config options to list which dirs to ignore. .kanuka/ ignored by default
		listOfKanukaFiles, err := secrets.FindEnvOrKanukaFiles(workingDirectory, []string{}, true)
		if err != nil {
			printError("Failed to find environment files", err)
			return
		}
		if len(listOfKanukaFiles) == 0 {
			printError("No environment files found", fmt.Errorf("no .env.kanuka files in %v", workingDirectory))
			return
		}

		verboseLog(fmt.Sprintf("✅ Found %d .env.kanuka files: %s", len(listOfKanukaFiles), secrets.FormatPaths(listOfKanukaFiles)))

		// Step 2: Get project's encrypted symmetric key
		encryptedSymKey, err := secrets.GetUserProjectKanukaKey()
		if err != nil {
			printError("Failed to get user's .kanuka file", err)
			return
		}
		verboseLog("🔑 Loaded user's .kanuka key")

		privateKey, err := secrets.GetUserPrivateKey()
		if err != nil {
			printError("Failed to get user's private key", err)
			return
		}
		verboseLog("🔑 Loaded user's private key")

		// Step 3: Decrypt user's kanuka file (get symmetric key)
		symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
		if err != nil {
			printError("Failed to decrypt symmetric key", err)
			return
		}
		verboseLog("🔓 Decrypted symmetric key")

		// Step 4: Decrypt all .kanuka files
		if err := secrets.DecryptFiles(symKey, listOfKanukaFiles, verbose); err != nil {
			printError("Failed to decrypt environment files", err)
			return
		}

		if !verbose {
			s.Stop()
		}
		fmt.Println(color.GreenString("✓") + " Environment files decrypted successfully!")
		fmt.Println(color.CyanString("→") + " Your .env files are now ready to use")
	},
}
