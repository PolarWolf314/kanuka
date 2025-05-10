package cmd

import (
	"fmt"
	"kanuka/internal/secrets"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var force bool

func init() {
	createCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	createCmd.Flags().BoolVarP(&force, "force", "f", false, "force key creation")
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates and adds your public key, and gives instructions on how to gain access",
	Run: func(cmd *cobra.Command, args []string) {
		spinner, cleanup := startSpinner("Creating Kanuka file...", verbose)
		defer cleanup()

		projectRoot, err := secrets.FindProjectKanukaRoot()
		if err != nil {
			printError("Failed to check if project kanuka settings exists", err)
			return
		}
		if projectRoot == "" {
			finalMessage := color.RedString("✗") + " Kanuka has not been initialized\n" +
				color.CyanString("→") + " Please run " + color.YellowString("kanuka secrets init") + " instead\n"
			spinner.FinalMSG = finalMessage
			return
		}

		verboseLog("Adding your public key...")

		if err := secrets.EnsureUserSettings(); err != nil {
			printError("Failed ensuring user settings", err)
			return
		}

		username, err := secrets.GetUsername()
		if err != nil {
			printError("Failed to get username", err)
			return
		}

		// If force flag is active, then ignore checking for existing symmetric key
		if !force {
			// We are explicitly ignoring errors, because an error means the key doesn't exist, which is what we want.
			encryptedSymmetricKey, _ := secrets.GetUserProjectKanukaKey()

			if encryptedSymmetricKey != nil {
				finalMessage := color.RedString("✗ ") + color.YellowString(username+".kanuka ") + "already exists\n" +
					"To override, run: " + color.YellowString("kanuka secrets create --force\n")
				spinner.FinalMSG = finalMessage
				return
			}

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

		finalMessage := color.GreenString("✓") + " Your public key has been added!\n" +
			color.CyanString("To gain access to the secrets in this project:\n") +
			"  1. " + color.WhiteString("Commit your") + color.YellowString(" .kanuka/public_keys/"+username+".pub ") + color.WhiteString("file to your version control system\n") +
			"  2. " + color.WhiteString("Ask someone with permissions to grant you access with:\n") +
			"     " + color.YellowString("kanuka secrets add "+username+"\n")

		spinner.FinalMSG = finalMessage
	},
}
