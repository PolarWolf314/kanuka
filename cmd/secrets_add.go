package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Adds a secret to the .env file",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Adding secret... (Placeholder)")
	},
}
