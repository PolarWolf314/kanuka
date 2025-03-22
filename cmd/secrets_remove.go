package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Removes a secret from the .env file",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Removing secret... (Placeholder)")
	},
}
