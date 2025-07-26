package main

import (
	"fmt"
	"os"

	"github.com/PolarWolf314/kanuka/cmd"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kanuka",
	Short: "Kānuka - A CLI for package management, cloud provisioning, and secrets management.",
	Long: `Kānuka is a powerful command-line tool for managing infrastructure, 
handling project packages using a nix shell environment, and securely storing environment secrets.

Features:
  - Store and retrieve secrets securely
  - Enter a nix shell without having to worry about your environment
  - Provision cloud resources using Pulumi
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Welcome to Kānuka! Run 'kanuka --help' to see available commands.")
		return nil
	},
}

func main() {
	rootCmd.AddCommand(cmd.SecretsCmd)
	rootCmd.AddCommand(cmd.GroveCmd)

	// Add 'kanuka dev' as an alias for 'kanuka grove enter'
	devCmd := &cobra.Command{
		Use:   "dev",
		Short: "Enter the development shell environment (alias for 'grove enter')",
		Long:  `Enter the development shell environment using devenv. This is an alias for 'kanuka grove enter'.`,
		RunE: func(devCmd *cobra.Command, args []string) error {
			// Get the flags from dev command
			authFlag, _ := devCmd.Flags().GetBool("auth")
			envFlag, _ := devCmd.Flags().GetString("env")

			// Get the grove enter command and set its flags
			enterCmd := cmd.GetGroveEnterCmd()

			// Set the flags on the enter command
			if err := enterCmd.Flags().Set("auth", fmt.Sprintf("%t", authFlag)); err != nil {
				return err
			}
			if envFlag != "" {
				if err := enterCmd.Flags().Set("env", envFlag); err != nil {
					return err
				}
			}

			// Execute grove enter command (errors are handled gracefully within the command)
			return enterCmd.RunE(enterCmd, args)
		},
	}

	// Copy flags from grove enter command
	devCmd.Flags().Bool("auth", false, "enable AWS SSO authentication")
	devCmd.Flags().String("env", "", "use named environment configuration")

	rootCmd.AddCommand(devCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
