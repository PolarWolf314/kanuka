package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var purgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Purges all secrets, including from the git history",
	Run: func(cmd *cobra.Command, args []string) {
		Logger.Infof("Starting purge command")
		Logger.WarnfAlways("Purge command is not yet implemented")
		fmt.Println("Purging secrets... (Placeholder)")
		Logger.Debugf("Purge command completed (placeholder)")
	},
}
