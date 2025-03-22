package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new user to be given access to the repository's secrets",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Adding secret... (Placeholder)")
	},
}
