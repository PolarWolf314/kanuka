package main

import (
	"fmt"
	"os"

	"github.com/PolarWolf314/kanuka/cmd"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kanuka",
	Short: "Kānuka - A CLI for secure secrets management.",
	Long: `Kānuka is a powerful command-line tool for securely storing and managing environment secrets.

Features:
  - Encrypt and decrypt environment files
  - Manage user access to project secrets
  - Secure key management using public-key cryptography
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Welcome to Kānuka! Run 'kanuka --help' to see available commands.")
		return nil
	},
}

func main() {
	rootCmd.AddCommand(cmd.SecretsCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
