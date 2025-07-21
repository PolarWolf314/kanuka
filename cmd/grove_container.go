package cmd

import (
	"github.com/spf13/cobra"
)

var groveContainerCmd = &cobra.Command{
	Use:   "container",
	Short: "Manage containers for Grove environments",
	Long: `Build and manage OCI containers from your Grove development environment.
	
Leverages devenv's container generation capabilities to create standard OCI containers
that can be deployed with any container orchestration tool.`,
}

func init() {
	groveContainerCmd.AddCommand(groveContainerInitCmd)
	groveContainerCmd.AddCommand(groveContainerBuildCmd)
}