package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var encryptCmd = &cobra.Command{
	Use:   "encrypt",
	Short: "Encrypts all .env files",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Encrypting .env file... (Placeholder)")
	},
}
