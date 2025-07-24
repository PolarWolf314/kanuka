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
	enterShell string
	enterName  string
)

var groveContainerEnterCmd = &cobra.Command{
	Use:   "enter",
	Short: "Enter container interactively for testing",
	Long: `Enter a built Grove container interactively for testing and debugging.

This command starts the container and gives you an interactive shell inside it.
Useful for testing your container before deployment.

Note: Container entering requires the container to be built first and requires
a container runtime (Docker/Podman) to be available.

Examples:
  kanuka grove container enter                    # Enter container with default shell
  kanuka grove container enter --shell bash      # Enter with specific shell
  kanuka grove container enter --name myapp      # Enter specific container by name`,
	RunE: func(cmd *cobra.Command, args []string) error {
		GroveLogger.Infof("Starting grove container enter command")
		spinner, cleanup := startGroveSpinner("Entering container...", groveVerbose)
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

		// Check if we're on macOS (containers don't work on macOS)
		GroveLogger.Debugf("Checking platform compatibility")
		if runtime.GOOS == "darwin" {
			finalMessage := color.RedString("✗") + " Container entering not supported on macOS\n" +
				color.CyanString("→") + " Containers require Linux container runtime\n" +
				color.CyanString("→") + " Alternatives:\n" +
				color.CyanString("  •") + " Use " + color.YellowString("kanuka grove enter") + " for local development\n" +
				color.CyanString("  •") + " Enter containers on Linux (CI/CD, remote server)\n" +
				color.CyanString("  •") + " Use a Linux VM for container testing"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Check if Docker or Podman is available
		GroveLogger.Debugf("Checking for container runtime")
		containerRuntime, err := detectContainerRuntime()
		if err != nil {
			finalMessage := color.RedString("✗") + " No container runtime found\n" +
				color.CyanString("→") + " Install Docker: " + color.YellowString("https://docs.docker.com/get-docker/") + "\n" +
				color.CyanString("→") + " Or install Podman: " + color.YellowString("https://podman.io/getting-started/installation")
			spinner.FinalMSG = finalMessage
			return nil
		}
		GroveLogger.Infof("Using container runtime: %s", containerRuntime)

		// Get container name
		var finalContainerName string
		if enterName != "" {
			finalContainerName = enterName
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

		// Check if container image exists
		GroveLogger.Debugf("Checking if container image exists: %s", finalContainerName)
		imageExists, err := checkContainerImageExists(containerRuntime, finalContainerName)
		if err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to check container image: %v", err)
		}
		if !imageExists {
			finalMessage := color.RedString("✗") + " Container image not found: " + color.YellowString(finalContainerName) + "\n" +
				color.CyanString("→") + " Build the container first: " + color.YellowString("kanuka grove container build")
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Stop spinner before entering interactive mode
		cleanup()

		// Enter container interactively
		GroveLogger.Infof("Entering container: %s", finalContainerName)
		if err := enterContainerInteractively(containerRuntime, finalContainerName, enterShell); err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to enter container: %v", err)
		}

		// This won't be reached if the container session was successful
		// as the process will be replaced by the container shell
		return nil
	},
}

// detectContainerRuntime detects available container runtime (Docker or Podman)
func detectContainerRuntime() (string, error) {
	// Check for Docker first
	if _, err := exec.LookPath("docker"); err == nil {
		// Verify Docker is running
		cmd := exec.Command("docker", "info")
		if err := cmd.Run(); err == nil {
			return "docker", nil
		}
	}

	// Check for Podman
	if _, err := exec.LookPath("podman"); err == nil {
		return "podman", nil
	}

	return "", fmt.Errorf("no container runtime found (docker or podman)")
}

// checkContainerImageExists checks if the container image exists locally
func checkContainerImageExists(runtime, imageName string) (bool, error) {
	GroveLogger.Debugf("Checking if image exists: %s %s", runtime, imageName)

	cmd := exec.Command(runtime, "images", "-q", imageName)
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check container images: %w", err)
	}

	// If output is not empty, image exists
	return strings.TrimSpace(string(output)) != "", nil
}

// enterContainerInteractively starts the container and enters it interactively
func enterContainerInteractively(runtime, imageName, shell string) error {
	GroveLogger.Debugf("Entering container with runtime: %s, image: %s, shell: %s", runtime, imageName, shell)

	// Determine shell to use
	containerShell := shell
	if containerShell == "" {
		containerShell = "/bin/bash" // Default to bash
	}

	// Build the container run command
	args := []string{
		"run",
		"--rm",         // Remove container when it exits
		"-it",          // Interactive with TTY
		imageName,      // Image name
		containerShell, // Shell to run
	}

	// Create the command
	cmd := exec.Command(runtime, args...)

	// Connect stdin, stdout, stderr to allow interactive use
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set up command environment
	cmd.Env = os.Environ()

	// Show the command being run in verbose mode
	if groveVerbose || groveDebug {
		GroveLogger.Infof("Running: %s %s", runtime, strings.Join(args, " "))
	}

	// Execute the command (this will replace the current process)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("container session failed: %w", err)
	}

	return nil
}

func init() {
	groveContainerEnterCmd.Flags().StringVar(&enterShell, "shell", "", "shell to use inside container (default: /bin/bash)")
	groveContainerEnterCmd.Flags().StringVar(&enterName, "name", "", "container name (default: name from devenv.nix)")
}
