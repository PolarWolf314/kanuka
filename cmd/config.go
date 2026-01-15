package cmd

import (
	"github.com/PolarWolf314/kanuka/internal/configs"
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
  - Set your default device name for new projects
  - Set your device name for an existing project
  - List all devices in the project

Examples:
  # Initialize your user configuration
  kanuka config init

  # List all devices in the project
  kanuka config list-devices

  # Set your default device name
  kanuka config set-default-device my-laptop

  # Set your device name for the current project
  kanuka config set-project-device my-laptop`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			ConfigLogger = logger.Logger{
				Verbose: configVerbose,
				Debug:   configDebug,
			}
			ConfigLogger.Debugf("Initializing config command with verbose=%t, debug=%t", configVerbose, configDebug)

			// Update key metadata access time if in a project.
			updateConfigProjectAccessTime()
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
	resetConfigShowState()
	resetSetProjectDeviceState()
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

// updateConfigProjectAccessTime updates the key metadata access time if running inside a project.
// This is called from PersistentPreRun to track when the project was last accessed.
// Errors are silently ignored as this is a non-critical operation.
func updateConfigProjectAccessTime() {
	// Try to find project root - if not in a project, this will fail silently.
	if err := configs.InitProjectSettings(); err != nil {
		// Not in a project or project not initialized - this is fine.
		return
	}

	// Load project config to get project UUID.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil || projectConfig.Project.UUID == "" {
		return
	}

	// Update the access time - errors are non-critical.
	_ = configs.UpdateKeyMetadataAccessTime(projectConfig.Project.UUID)
}
