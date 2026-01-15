package cmd

import (
	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/utils"

	"github.com/PolarWolf314/kanuka/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	ConfigCmd.AddCommand(setDefaultDeviceCmd)
}

var setDefaultDeviceCmd = &cobra.Command{
	Use:   "set-default-device [device-name]",
	Short: "Set your default device name for new projects",
	Long: `Sets your default device name in your user configuration.

This command updates the default_device_name field in your user config file
(~/.config/kanuka/config.toml). This default name is used when you initialize
or register for new projects.

The device name must be alphanumeric with hyphens and underscores only.

Examples:
  # Set your default device name
  kanuka config set-default-device my-laptop`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ConfigLogger.Infof("Starting set-default-device command")
		spinner, cleanup := startSpinnerWithFlags("Setting default device name...", configVerbose, configDebug)
		defer cleanup()

		deviceName := args[0]
		ConfigLogger.Debugf("Device name argument: %s", deviceName)

		if !utils.IsValidDeviceName(deviceName) {
			finalMessage := ui.Error.Sprint("✗") + " Invalid device name: " + ui.Highlight.Sprint(deviceName) + "\n" +
				ui.Info.Sprint("→") + " Device name must be alphanumeric with hyphens and underscores only"
			spinner.FinalMSG = finalMessage
			return nil
		}

		userConfig, err := configs.LoadUserConfig()
		if err != nil {
			return ConfigLogger.ErrorfAndReturn("Failed to load user config: %v", err)
		}

		if userConfig.User.DefaultDeviceName == deviceName {
			finalMessage := ui.Warning.Sprint("⚠") + " Default device name is already set to " + ui.Highlight.Sprint(deviceName)
			spinner.FinalMSG = finalMessage
			return nil
		}

		userConfig.User.DefaultDeviceName = deviceName
		ConfigLogger.Debugf("Setting default device name to: %s", deviceName)

		if err := configs.SaveUserConfig(userConfig); err != nil {
			return ConfigLogger.ErrorfAndReturn("Failed to save user config: %v", err)
		}
		ConfigLogger.Infof("User config saved successfully")

		ConfigLogger.Infof("Default device name set successfully")
		finalMessage := ui.Success.Sprint("✓") + " Default device name set to " + ui.Highlight.Sprint(deviceName) + "\n"
		spinner.FinalMSG = finalMessage
		return nil
	},
}
