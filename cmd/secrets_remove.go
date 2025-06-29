package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Removes access to the secret store",
	RunE: func(cmd *cobra.Command, args []string) error {
		Logger.Infof("Starting remove command")
		Logger.WarnfAlways("Remove command is not yet implemented")
		fmt.Println("Removing secret... (Placeholder)")
		Logger.Debugf("Remove command completed (placeholder)")
		return nil
	},
}
