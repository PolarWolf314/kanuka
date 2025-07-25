package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/grove"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	buildNoSync bool
)

var groveContainerBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build OCI container from Grove environment",
	Long: `Build an OCI-compliant container from your Grove development environment.

Uses devenv's container generation to create a standard container that includes
your packages, languages, and environment configuration. The container will be
named "shell" as required by devenv.

Note: Container building requires Linux. On macOS, use CI/CD or remote Linux systems.

Examples:
  kanuka grove container build                   # Build and auto-sync to Docker daemon
  kanuka grove container build --no-sync        # Build without syncing to Docker daemon`,
	RunE: func(cmd *cobra.Command, args []string) error {
		GroveLogger.Infof("Starting grove container build command")
		spinner, cleanup := startGroveSpinner("Building container from Grove environment...", groveVerbose)
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

		// Check if we're on macOS (devenv containers don't work on macOS)
		GroveLogger.Debugf("Checking platform compatibility")
		if runtime.GOOS == "darwin" {
			finalMessage := color.RedString("✗") + " Container building not supported on macOS\n" +
				color.CyanString("→") + " devenv containers require Linux container runtime\n" +
				color.CyanString("→") + " Alternatives:\n" +
				color.CyanString("  •") + " Use " + color.YellowString("devenv shell") + " for local development\n" +
				color.CyanString("  •") + " Build containers on Linux (CI/CD, remote server)\n" +
				color.CyanString("  •") + " Use a Linux VM or container for building"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Get the project name from devenv.nix for display purposes
		projectName, err := grove.GetContainerNameFromDevenvNix()
		if err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to get project name from devenv.nix: %v", err)
		}
		GroveLogger.Debugf("Project name from devenv.nix: %s", projectName)

		// Note: devenv always uses "shell" as the container name, regardless of project name
		GroveLogger.Debugf("devenv will build container with name: shell")

		// Build container using devenv - devenv always uses "shell" as the container name
		GroveLogger.Debugf("Building container with devenv")
		if err := buildContainerWithDevenv("shell"); err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to build container: %v", err)
		}

		GroveLogger.Infof("Container build completed successfully")

		// Sync container to Docker daemon (unless --no-sync is specified)
		var syncCompleted bool
		if !buildNoSync {
			GroveLogger.Debugf("Syncing container to Docker daemon")
			spinner.Suffix = " Syncing container to Docker daemon..."
			if err := syncContainerToDockerDaemon("shell"); err != nil {
				return GroveLogger.ErrorfAndReturn("Failed to sync container to Docker daemon: %v", err)
			}
			GroveLogger.Infof("Container sync completed successfully")
			syncCompleted = true
		} else {
			GroveLogger.Infof("Skipping container sync (--no-sync specified)")
		}

		// Build success message based on whether sync was performed
		var finalMessage string
		if syncCompleted {
			finalMessage = color.GreenString("✓") + " Container built and synced successfully!\n" +
				color.CyanString("→") + " Container: " + color.YellowString("shell") + " (from project: " + projectName + ")\n" +
				color.CyanString("→") + " Synced to Docker daemon - ready to run!\n" +
				color.CyanString("→") + " Test locally: " + color.YellowString("kanuka grove container enter") + "\n" +
				color.CyanString("→") + " Run with Docker: " + color.YellowString("docker run -it shell") + "\n" +
				color.CyanString("→") + " Push to registry: " + color.YellowString("docker push shell")
		} else {
			finalMessage = color.GreenString("✓") + " Container built successfully!\n" +
				color.CyanString("→") + " Container: " + color.YellowString("shell") + " (from project: " + projectName + ")\n" +
				color.YellowString("⚠") + "  " + color.YellowString("Container not synced to Docker daemon (--no-sync used)\n") +
				color.CyanString("→") + " Run " + color.YellowString("kanuka grove container sync") + " to sync to Docker daemon\n" +
				color.CyanString("→") + " Test locally: " + color.YellowString("kanuka grove container enter") + "\n" +
				color.CyanString("→") + " Push to registry: " + color.YellowString("docker push shell")
		}

		spinner.FinalMSG = finalMessage
		return nil
	},
}

// buildContainerWithDevenv builds the container using devenv's container build command.
func buildContainerWithDevenv(containerName string) error {
	GroveLogger.Debugf("Executing devenv container build with name: %s", containerName)

	// Build the devenv container build command (correct syntax: devenv container build <NAME>)
	cmd := exec.Command("devenv", "container", "build", containerName)

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
		return fmt.Errorf("devenv container build failed: %w", err)
	}

	return nil
}

// syncContainerToDockerDaemon syncs the container from Nix store to Docker daemon using devenv.
func syncContainerToDockerDaemon(containerName string) error {
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

func init() {
	groveContainerBuildCmd.Flags().BoolVar(&buildNoSync, "no-sync", false, "skip syncing container to Docker daemon")
}
