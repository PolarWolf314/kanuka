package cmd

import (
	"github.com/PolarWolf314/kanuka/internal/grove"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var groveContainerInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize container support for Grove environment",
	Long: `Initialize container support by adding nix2container input to devenv.yaml.

This enables you to build OCI containers from your Grove development environment
using 'kanuka grove container build'. Container configuration is handled directly
by devenv using the name field in devenv.nix.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		GroveLogger.Infof("Starting grove container init command")
		spinner, cleanup := startGroveSpinner("Initializing container support...", groveVerbose)
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

		// Check if container support is already initialized
		GroveLogger.Debugf("Checking if container support already exists")
		containerExists, err := grove.DoesContainerConfigExist()
		if err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to check container configuration: %v", err)
		}
		if containerExists {
			finalMessage := color.RedString("✗") + " Container support already initialized\n" +
				color.CyanString("→") + " Use " + color.YellowString("kanuka grove container build") + " to build containers"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Add required nix2container input first
		GroveLogger.Debugf("Adding nix2container input (required for containers)")
		if err := grove.AddNix2ContainerInput(); err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to add nix2container input: %v", err)
		}
		GroveLogger.Infof("nix2container input added")

		// Initialize container support
		GroveLogger.Debugf("Adding container configuration to devenv.nix")
		if err := grove.AddContainerConfigToDevenvNix(); err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to add container configuration to devenv.nix: %v", err)
		}
		GroveLogger.Infof("Container configuration added to devenv.nix")


		GroveLogger.Infof("Grove container init command completed successfully")

		finalMessage := color.GreenString("✓") + " Container support initialized!\n" +
			color.CyanString("→") + " Updated devenv.yaml with nix2container input\n" +
			color.CyanString("→") + " devenv.nix already has name field for containers\n" +
			color.CyanString("→") + " Container configuration handled directly by devenv\n" +
			color.YellowString("⚠") + "  " + color.YellowString("Do not modify the nix2container input - required for containers\n") +
			color.CyanString("→") + " Run " + color.YellowString("kanuka grove container build") + " to create your first container"

		spinner.FinalMSG = finalMessage
		return nil
	},
}
