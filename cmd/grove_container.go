package cmd

import (
	"github.com/spf13/cobra"
)

var groveContainerCmd = &cobra.Command{
	Use:   "container",
	Short: "Manage containers for Grove environments",
	Long: `Build and manage OCI containers from your Grove development environment.
	
Leverages devenv's container generation capabilities to create standard OCI containers
that can be deployed with any container orchestration tool. Containers are automatically
synced to the Docker daemon for immediate use.

Available commands:
  init   - Initialize container support
  build  - Build container (with automatic sync to Docker daemon)
  sync   - Manually sync container from Nix store to Docker daemon
  enter  - Enter container interactively

Note: Container building requires Linux. On macOS, use CI/CD or remote Linux systems.`,
}

func init() {
	groveContainerCmd.AddCommand(groveContainerInitCmd)
	groveContainerCmd.AddCommand(groveContainerBuildCmd)
	groveContainerCmd.AddCommand(groveContainerSyncCmd)
	groveContainerCmd.AddCommand(groveContainerEnterCmd)
}
