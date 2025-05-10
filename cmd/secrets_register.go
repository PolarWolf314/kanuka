package cmd

import (
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

		finalMessage := color.GreenString("✓") + " User " + color.YellowString("") + " has been registered successfully!\n" +
			color.CyanString("→") + " They now have access to decrypt the repository's secrets\n"
		spinner.FinalMSG = finalMessage
	},
}
