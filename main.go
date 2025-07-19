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

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
