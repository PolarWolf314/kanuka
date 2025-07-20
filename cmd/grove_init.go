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
		devenvNixExists, err := grove.DoesDevenvNixExist()
		if err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to check if devenv.nix exists: %v", err)
		}

		GroveLogger.Debugf("Checking if devenv.yaml already exists")
		devenvYamlExists, err := grove.DoesDevenvYamlExist()
		if err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to check if devenv.yaml exists: %v", err)
		}

		GroveLogger.Debugf("Creating kanuka.toml")
		if err := grove.CreateKanukaToml(); err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to create kanuka.toml: %v", err)
		}
		GroveLogger.Infof("kanuka.toml created successfully")

		if !devenvYamlExists {
			GroveLogger.Debugf("Creating devenv.yaml")
			if err := grove.CreateDevenvYaml(); err != nil {
				return GroveLogger.ErrorfAndReturn("Failed to create devenv.yaml: %v", err)
			}
			GroveLogger.Infof("devenv.yaml created successfully")
		} else {
			GroveLogger.Infof("devenv.yaml already exists, skipping creation")
		}

		if !devenvNixExists {
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
		filesCreated := []string{}
		if !devenvYamlExists {
			filesCreated = append(filesCreated, "devenv.yaml")
		}
		if !devenvNixExists {
			filesCreated = append(filesCreated, "devenv.nix")
		}
		filesCreated = append(filesCreated, "kanuka.toml")

		if len(filesCreated) == 3 {
			finalMessage = color.GreenString("✓") + " Development environment initialized!\n" +
				color.CyanString("→") + " Created kanuka.toml, devenv.yaml, and devenv.nix\n" +
				color.CyanString("→") + " Run " + color.YellowString("kanuka grove add <package>") + " to add packages"
		} else {
			preservedFiles := []string{}
			if devenvYamlExists {
				preservedFiles = append(preservedFiles, "devenv.yaml")
			}
			if devenvNixExists {
				preservedFiles = append(preservedFiles, "devenv.nix")
			}
			
			finalMessage = color.GreenString("✓") + " Development environment initialized!\n" +
				color.CyanString("→") + " kanuka.toml created, existing files preserved\n" +
				color.CyanString("→") + " Run " + color.YellowString("kanuka grove add <package>") + " to add packages"
		}

		spinner.FinalMSG = finalMessage
		return nil
	},
}
