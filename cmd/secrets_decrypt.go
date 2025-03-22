package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var decryptCmd = &cobra.Command{
	Use:   "decrypt",
	Short: "Decrypts the .env file",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Decrypting .env file... (Placeholder)")
	},
}
