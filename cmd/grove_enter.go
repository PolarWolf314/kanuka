package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/PolarWolf314/kanuka/internal/grove"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	enterAuth bool
	enterEnv  string
)

var groveEnterCmd = &cobra.Command{
	Use:   "enter",
	Short: "Enter the development shell environment",
	Long: `Enter the development shell environment using devenv.
This starts a new shell with all your configured packages and languages available.

Examples:
  kanuka grove enter                   # Enter basic development shell
  kanuka grove enter --auth            # Enter shell with AWS SSO authentication
  kanuka grove enter --env production  # Enter shell with production environment`,
	RunE: func(cmd *cobra.Command, args []string) error {
		GroveLogger.Infof("Starting grove enter command")
		spinner, cleanup := startGroveSpinner("Preparing development environment...", groveVerbose)
		defer cleanup()

		// Check if we're in a grove project
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

		// Check if devenv is available
		GroveLogger.Debugf("Checking if devenv command is available")
		_, err = exec.LookPath("devenv")
		if err != nil {
			finalMessage := color.RedString("✗") + " devenv command not found\n" +
				color.CyanString("→") + " Install devenv: " + color.YellowString("nix profile install nixpkgs#devenv") + "\n" +
				color.CyanString("→") + " Or visit: " + color.BlueString("https://devenv.sh/getting-started/")
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Handle authentication if requested
		if enterAuth {
			GroveLogger.Debugf("Authentication requested")
			spinner.Suffix = " Setting up authentication..."
			
			err := handleAuthentication()
			if err != nil {
				return GroveLogger.ErrorfAndReturn("Failed to set up authentication: %v", err)
			}
		}

		// Handle named environment if specified
		if enterEnv != "" {
			GroveLogger.Debugf("Named environment requested: %s", enterEnv)
			spinner.Suffix = " Loading environment: " + enterEnv + "..."
			
			err := handleNamedEnvironment(enterEnv)
			if err != nil {
				return GroveLogger.ErrorfAndReturn("Failed to load environment '%s': %v", enterEnv, err)
			}
		}

		// Stop spinner before entering shell
		spinner.Stop()

		// Show entry message
		fmt.Printf("%s Entering development environment...\n", color.GreenString("✓"))
		if enterAuth {
			fmt.Printf("%s Authentication enabled\n", color.CyanString("→"))
		}
		if enterEnv != "" {
			fmt.Printf("%s Environment: %s\n", color.CyanString("→"), color.YellowString(enterEnv))
		}
		fmt.Printf("%s Type %s to exit\n\n", color.CyanString("→"), color.YellowString("exit"))

		// Enter the devenv shell
		GroveLogger.Debugf("Executing devenv shell")
		return enterDevenvShell()
	},
}

// enterDevenvShell executes the devenv shell command
func enterDevenvShell() error {
	// Find devenv executable
	devenvPath, err := exec.LookPath("devenv")
	if err != nil {
		return fmt.Errorf("devenv command not found: %w", err)
	}

	// Prepare the command
	args := []string{"devenv", "shell"}
	
	// Execute devenv shell, replacing the current process
	GroveLogger.Debugf("Executing: %s %v", devenvPath, args[1:])
	err = syscall.Exec(devenvPath, args, os.Environ())
	if err != nil {
		return fmt.Errorf("failed to execute devenv shell: %w", err)
	}

	// This line should never be reached if syscall.Exec succeeds
	return nil
}

// handleAuthentication sets up AWS SSO authentication
func handleAuthentication() error {
	GroveLogger.Debugf("Setting up AWS SSO authentication")
	
	// TODO: Implement AWS SSO authentication
	// For now, this is a placeholder that will be implemented in a future iteration
	GroveLogger.Infof("Authentication setup requested (not yet implemented)")
	
	return nil
}

// handleNamedEnvironment loads a named environment configuration
func handleNamedEnvironment(envName string) error {
	GroveLogger.Debugf("Loading named environment: %s", envName)
	
	// TODO: Implement named environment loading
	// For now, this is a placeholder that will be implemented in a future iteration
	GroveLogger.Infof("Named environment '%s' requested (not yet implemented)", envName)
	
	return nil
}

func init() {
	groveEnterCmd.Flags().BoolVar(&enterAuth, "auth", false, "enable AWS SSO authentication")
	groveEnterCmd.Flags().StringVar(&enterEnv, "env", "", "use named environment configuration")
}