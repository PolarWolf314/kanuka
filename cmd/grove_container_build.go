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
	buildName    string
	buildProfile string
)

var groveContainerBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build OCI container from Grove environment",
	Long: `Build an OCI-compliant container from your Grove development environment.

Uses devenv's container generation to create a standard container that includes
your packages, languages, and environment configuration.

Note: Container building requires Linux. On macOS, use CI/CD or remote Linux systems.

Examples:
  kanuka grove container build                   # Build with default profile
  kanuka grove container build --name myapp      # Build with custom container name
  kanuka grove container build --profile minimal # Build with minimal profile`,
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

		// Validate and get profile configuration
		GroveLogger.Debugf("Validating container profile: %s", buildProfile)
		profileConfig, err := grove.GetContainerProfile(buildProfile)
		if err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to get container profile '%s': %v", buildProfile, err)
		}
		GroveLogger.Infof("Using container profile: %s", profileConfig.Name)

		// Use custom name if provided, otherwise get the name from devenv.nix
		var finalContainerName string
		if buildName != "" {
			finalContainerName = buildName
			GroveLogger.Debugf("Using custom container name: %s", finalContainerName)
		} else {
			// Get container name from devenv.nix
			defaultName, err := grove.GetContainerNameFromDevenvNix()
			if err != nil {
				return GroveLogger.ErrorfAndReturn("Failed to get container name from devenv.nix: %v", err)
			}
			finalContainerName = defaultName
			GroveLogger.Debugf("Using container name from devenv.nix: %s", finalContainerName)
		}

		// Apply profile configuration and custom name to devenv.nix (temporarily)
		GroveLogger.Debugf("Applying profile configuration and container name")
		cleanup_profile, err := grove.ApplyContainerProfileAndName(profileConfig, finalContainerName)
		if err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to apply container configuration: %v", err)
		}
		defer cleanup_profile() // Ensure we restore original devenv.nix

		// Build container using devenv
		GroveLogger.Debugf("Building container with devenv")
		if err := buildContainerWithDevenv(finalContainerName); err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to build container: %v", err)
		}

		GroveLogger.Infof("Container build completed successfully")

		finalMessage := color.GreenString("✓") + " Container built successfully!\n" +
			color.CyanString("→") + " Container: " + color.YellowString(finalContainerName) + "\n" +
			color.CyanString("→") + " Profile: " + color.WhiteString(profileConfig.Name) + "\n" +
			color.CyanString("→") + " Test locally: " + color.YellowString("kanuka grove container enter") + "\n" +
			color.CyanString("→") + " Run with Docker: " + color.YellowString("docker run -it "+finalContainerName) + "\n" +
			color.CyanString("→") + " Push to registry: " + color.YellowString("docker push "+finalContainerName)

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

func init() {
	groveContainerBuildCmd.Flags().StringVar(&buildName, "name", "", "container name (default: name from devenv.nix)")
	groveContainerBuildCmd.Flags().StringVar(&buildProfile, "profile", "default", "container profile to use")
}
