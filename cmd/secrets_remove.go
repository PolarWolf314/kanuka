package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Removes access to the secret store",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Removing secret... (Placeholder)")
	},
}
