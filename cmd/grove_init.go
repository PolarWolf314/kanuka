package cmd

import (
	"github.com/PolarWolf314/kanuka/internal/grove"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	initContainers bool
)

var groveInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a development environment with devenv.nix",
	Long: `Initialize a new development environment by creating devenv.nix and kanuka.toml files.
This sets up the foundation for managing packages and development shell environments.

Use --containers to also initialize container support for building OCI containers.`,
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

		// Initialize container support if requested
		var containerInitialized bool
		if initContainers {
			GroveLogger.Debugf("Initializing container support")
			if err := grove.AddContainerConfigToDevenvNix(); err != nil {
				return GroveLogger.ErrorfAndReturn("Failed to add container configuration to devenv.nix: %v", err)
			}
			if err := grove.AddContainerProfilesToKanukaToml(); err != nil {
				return GroveLogger.ErrorfAndReturn("Failed to add container profiles to kanuka.toml: %v", err)
			}
			containerInitialized = true
			GroveLogger.Infof("Container support initialized")
		}

		GroveLogger.Infof("Grove init command completed successfully")

		// Build success message
		var finalMessage string
		filesCreated := []string{}
		if !devenvYamlExists {
			filesCreated = append(filesCreated, "devenv.yaml")
		}
		if !devenvNixExists {
			filesCreated = append(filesCreated, "devenv.nix")
		}
		filesCreated = append(filesCreated, "kanuka.toml")

		// Main success message
		finalMessage = color.GreenString("✓") + " Development environment initialized!\n"

		// Files created message
		if len(filesCreated) == 3 {
			finalMessage += color.CyanString("→") + " Created kanuka.toml, devenv.yaml, and devenv.nix\n"
		} else {
			finalMessage += color.CyanString("→") + " kanuka.toml created, existing files preserved\n"
		}

		// Container message
		if containerInitialized {
			finalMessage += color.CyanString("→") + " Container support enabled\n"
		}

		// Next steps
		finalMessage += color.CyanString("→") + " Run " + color.YellowString("kanuka grove add <package>") + " to add packages\n"
		finalMessage += color.CyanString("→") + " Run " + color.YellowString("kanuka grove enter") + " to enter your environment"

		// Additional container steps
		if containerInitialized {
			finalMessage += "\n" + color.CyanString("→") + " Run " + color.YellowString("kanuka grove container build") + " to create containers"
		} else {
			finalMessage += "\n" + color.CyanString("→") + " Use " + color.YellowString("kanuka grove container init") + " to add container support later"
		}

		spinner.FinalMSG = finalMessage
		return nil
	},
}

func init() {
	groveInitCmd.Flags().BoolVar(&initContainers, "containers", false, "initialize container support")
}
