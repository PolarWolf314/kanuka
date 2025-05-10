package cmd

import (
	"kanuka/internal/secrets"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var username string

func init() {
	registerCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	registerCmd.Flags().StringVarP(&username, "user", "u", "", "username to register for access")
	if err := registerCmd.MarkFlagRequired("user"); err != nil {
		printError("Failed to mark --user flag as required", err)
		return
	}
}

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Registers a new user to be given access to the repository's secrets",
	Run: func(cmd *cobra.Command, args []string) {
		spinner, cleanup := startSpinner("Registering user for access...", verbose)
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

		// Check if specified user's public key exists
		pubKeyPath := filepath.Join(projectRoot, ".kanuka", "public_keys", username+".pub")

		targetUserPublicKey, err := secrets.LoadPublicKey(pubKeyPath)
		if err != nil {
			finalMessage := color.RedString("✗") + " Public key for user " + color.YellowString(username) + " not found\n" +
				username + " must first run: " + color.YellowString("kanuka secrets create\n")
			spinner.FinalMSG = finalMessage
			return
		}
			color.CyanString("→") + " They now have access to decrypt the repository's secrets\n"
		spinner.FinalMSG = finalMessage
	},
}
