package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"kanuka/cmd"
)

var rootCmd = &cobra.Command{
	Use:   "kanuka",
	Short: "Kanuka - A CLI for package management, cloud provisioning, and secrets management.",
	Long: `Kanuka is a powerful command-line tool for managing infrastructure, 
handling project packages using a nix shell environment, and securely storing environment secrets.

Features:
  - Store and retrieve secrets securely
  - Enter a nix shell without having to worry about your environment
  - Provision cloud resources using Pulumi
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Welcome to Kānuka! Run 'kanuka --help' to see available commands.")
	},
}

func main() {
	rootCmd.AddCommand(cmd.SecretsCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
