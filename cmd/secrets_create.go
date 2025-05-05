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
	createCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates and adds your public key, and gives instructions on how to gain access",
	Run: func(cmd *cobra.Command, args []string) {
		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Suffix = " Adding your public key..."
		err := s.Color("cyan")
		if err != nil {
			printError("Failed to create a spinner", err)
		}

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
			printError(".kanuka/ doesn't exist", fmt.Errorf("please init the project first"))
			return
		}

		verboseLog("Adding your public key...")

		if err := secrets.EnsureUserSettings(); err != nil {
			printError("Failed ensuring user settings", err)
			return
		}

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

		username, err := secrets.GetUsername()
		if err != nil {
			printError("Failed to get username", err)
			return
		}

		if !verbose {
			s.Stop()
		}

		fmt.Println(color.GreenString("✓") + " Your public key has been added!")
		fmt.Println()
		fmt.Println(color.CyanString("To gain access to the secrets in this project:"))
		fmt.Println("  1. " + color.WhiteString("Commit your") + color.YellowString(" .kanuka/public_keys/"+username+".pub ") + color.WhiteString("file to Git"))
		fmt.Println("  2. " + color.WhiteString("Ask someone with permissions to grant you access with:"))
		fmt.Println("   " + color.YellowString("kanuka secrets add "+username))
	},
}
