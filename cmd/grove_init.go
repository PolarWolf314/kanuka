package cmd

import (
	"github.com/PolarWolf314/kanuka/internal/grove"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var groveInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a development environment with devenv.nix",
	Long: `Initialize a new development environment by creating devenv.nix and kanuka.toml files.
This sets up the foundation for managing packages and development shell environments.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		GroveLogger.Infof("Starting grove init command")
		spinner, cleanup := startGroveSpinner("Initializing development environment...", groveVerbose)
		defer cleanup()

		GroveLogger.Debugf("Checking if kanuka.toml already exists")
		exists, err := grove.DoesKanukaTomlExist()
		if err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to check if kanuka.toml exists: %v", err)
		}
		if exists {
			finalMessage := color.RedString("✗") + " Development environment already initialized\n" +
				color.CyanString("→") + " kanuka.toml already exists in this project"
			spinner.FinalMSG = finalMessage
			return nil
		}

		GroveLogger.Debugf("Checking if devenv.nix already exists")
		devenvExists, err := grove.DoesDevenvNixExist()
		if err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to check if devenv.nix exists: %v", err)
		}

		GroveLogger.Debugf("Creating kanuka.toml")
		if err := grove.CreateKanukaToml(); err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to create kanuka.toml: %v", err)
		}
		GroveLogger.Infof("kanuka.toml created successfully")

		if !devenvExists {
			GroveLogger.Debugf("Creating devenv.nix")
			if err := grove.CreateDevenvNix(); err != nil {
				return GroveLogger.ErrorfAndReturn("Failed to create devenv.nix: %v", err)
			}
			GroveLogger.Infof("devenv.nix created successfully")
		} else {
			GroveLogger.Infof("devenv.nix already exists, skipping creation")
		}

		GroveLogger.Infof("Grove init command completed successfully")

		var finalMessage string
		if devenvExists {
			finalMessage = color.GreenString("✓") + " Development environment initialized!\n" +
				color.CyanString("→") + " kanuka.toml created, existing devenv.nix preserved\n" +
				color.CyanString("→") + " Run " + color.YellowString("kanuka grove add <package>") + " to add packages"
		} else {
			finalMessage = color.GreenString("✓") + " Development environment initialized!\n" +
				color.CyanString("→") + " Created kanuka.toml and devenv.nix\n" +
				color.CyanString("→") + " Run " + color.YellowString("kanuka grove add <package>") + " to add packages"
		}

		spinner.FinalMSG = finalMessage
		return nil
	},
}