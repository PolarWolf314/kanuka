package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/grove"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var groveContainerSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync container from Nix store to Docker daemon",
	Long: `Sync the Grove container from the Nix store to the Docker daemon.

This command runs 'devenv container copy shell' to transfer the latest
container state from the Nix store to Docker's local registry. This is
useful when the environment has changed without rebuilding the container,
or when you want to manually sync after a build.

Note: This command requires the container to be built first with
'kanuka grove container build'.

Examples:
  kanuka grove container sync                     # Sync the shell container`,
	RunE: func(cmd *cobra.Command, args []string) error {
		GroveLogger.Infof("Starting grove container sync command")
		spinner, cleanup := startGroveSpinner("Syncing container to Docker daemon...", groveVerbose)
		defer cleanup()

		// Check if we're in a Grove project
		GroveLogger.Debugf("Checking if kanuka.toml exists")
		exists, err := grove.DoesKanukaTomlExist()
		if err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to check project status: %v", err)
		}
		if !exists {
			finalMessage := color.RedString("✗") + " Not in a grove project\n" +
				color.CyanString("→") + " Run " + color.YellowString("kanuka grove init") + " first"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Check if devenv.nix exists
		GroveLogger.Debugf("Checking if devenv.nix exists")
		devenvExists, err := grove.DoesDevenvNixExist()
		if err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to check devenv.nix: %v", err)
		}
		if !devenvExists {
			finalMessage := color.RedString("✗") + " devenv.nix not found\n" +
				color.CyanString("→") + " Run " + color.YellowString("kanuka grove init") + " to create it"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Check if container support is initialized
		GroveLogger.Debugf("Checking if container support exists")
		containerExists, err := grove.DoesContainerConfigExist()
		if err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to check container configuration: %v", err)
		}
		if !containerExists {
			finalMessage := color.RedString("✗") + " Container support not initialized\n" +
				color.CyanString("→") + " Run " + color.YellowString("kanuka grove container init") + " first"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Check if devenv is installed
		GroveLogger.Debugf("Checking if devenv is available")
		if _, err := exec.LookPath("devenv"); err != nil {
			finalMessage := color.RedString("✗") + " devenv not found\n" +
				color.CyanString("→") + " Install devenv: " + color.YellowString("nix profile install nixpkgs#devenv")
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Get the project name for display purposes
		projectName, err := grove.GetContainerNameFromDevenvNix()
		if err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to get project name from devenv.nix: %v", err)
		}
		GroveLogger.Debugf("Project name from devenv.nix: %s (container will be synced as 'shell')", projectName)

		// Sync container to Docker daemon
		GroveLogger.Debugf("Syncing container to Docker daemon")
		if err := syncContainerToDockerDaemonSync("shell"); err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to sync container to Docker daemon: %v", err)
		}

		GroveLogger.Infof("Container sync completed successfully")

		finalMessage := color.GreenString("✓") + " Container synced successfully!\n" +
			color.CyanString("→") + " Container: " + color.YellowString("shell") + " (from project: " + projectName + ")\n" +
			color.CyanString("→") + " Synced to Docker daemon - ready to run!\n" +
			color.CyanString("→") + " Test locally: " + color.YellowString("kanuka grove container enter") + "\n" +
			color.CyanString("→") + " Run with Docker: " + color.YellowString("docker run -it shell")

		spinner.FinalMSG = finalMessage
		return nil
	},
}

// syncContainerToDockerDaemonSync syncs the container from Nix store to Docker daemon using devenv.
// This is a separate function to avoid import cycles with the build command.
func syncContainerToDockerDaemonSync(containerName string) error {
	GroveLogger.Debugf("Executing devenv container copy with name: %s", containerName)

	// Build the devenv container copy command
	cmd := exec.Command("devenv", "container", "copy", containerName)

	// Set up command environment
	cmd.Dir, _ = os.Getwd()
	cmd.Env = os.Environ()

	// In verbose mode, show the output
	if groveVerbose || groveDebug {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		GroveLogger.Infof("Running: %s", strings.Join(cmd.Args, " "))
	}

	// Execute the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("devenv container copy failed: %w", err)
	}

	return nil
}
