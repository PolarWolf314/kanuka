package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var purgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Purges all secrets, including from the git history",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Purging secrets... (Placeholder)")
	},
}
