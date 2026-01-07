package cmd

import (
	logger "github.com/PolarWolf314/kanuka/internal/logging"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	configVerbose bool
	configDebug   bool
	ConfigLogger  logger.Logger

	// ConfigCmd is the top-level config command.
	ConfigCmd = &cobra.Command{
		Use:   "config",
		Short: "Manage Kānuka configuration",
		Long: `Provides commands for managing user and project configuration settings.

Use these commands to:
  - Initialize your user identity (config init)
  - Set your device name for a project
  - Rename devices in the project
  - List all devices in the project

Examples:
  # Initialize your user configuration
  kanuka config init

  # List all devices in the project
  kanuka config list-devices

  # Set your device name for the current project
  kanuka config set-device-name my-laptop

  # Rename a device in the project
  kanuka config rename-device --user alice@example.com --new-name workstation`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			ConfigLogger = logger.Logger{
				Verbose: configVerbose,
				Debug:   configDebug,
			}
			ConfigLogger.Debugf("Initializing config command with verbose=%t, debug=%t", configVerbose, configDebug)
		},
	}
)

func init() {
	ConfigCmd.PersistentFlags().BoolVarP(&configVerbose, "verbose", "v", false, "enable verbose output")
	ConfigCmd.PersistentFlags().BoolVarP(&configDebug, "debug", "d", false, "enable debug output")
}

// GetConfigCmd returns the ConfigCmd for testing.
func GetConfigCmd() *cobra.Command {
	return ConfigCmd
}

// ResetConfigState resets all config command global variables to their default values for testing.
func ResetConfigState() {
	configVerbose = false
	configDebug = false
	resetConfigInitState()
	resetSetDeviceNameState()
	resetRenameDeviceState()
	resetListDevicesState()
	resetConfigCobraFlagState()
}

// resetConfigCobraFlagState resets the flag state for all config commands to prevent test pollution.
func resetConfigCobraFlagState() {
	if ConfigCmd != nil && ConfigCmd.Flags() != nil {
		ConfigCmd.Flags().VisitAll(func(flag *pflag.Flag) {
			flag.Changed = false
		})
	}
}
