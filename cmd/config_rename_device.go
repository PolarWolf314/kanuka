package cmd

import (
	"github.com/PolarWolf314/kanuka/internal/configs"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	renameDeviceUserEmail string
	renameDeviceOldName   string
)

func init() {
	renameDeviceCmd.Flags().StringVarP(&renameDeviceUserEmail, "user", "u", "", "user email (required)")
	renameDeviceCmd.Flags().StringVar(&renameDeviceOldName, "old-name", "", "old device name (required if user has multiple devices)")
	_ = renameDeviceCmd.MarkFlagRequired("user")
	ConfigCmd.AddCommand(renameDeviceCmd)
}

// resetRenameDeviceState resets the rename-device command's global state for testing.
func resetRenameDeviceState() {
	renameDeviceUserEmail = ""
	renameDeviceOldName = ""
}

var renameDeviceCmd = &cobra.Command{
	Use:   "rename-device [new-name]",
	Short: "Rename a device in the project",
	Long: `Renames a device in the project configuration.

This command updates the device name in the project's config.toml file.
You must specify the user email whose device you want to rename.

If the user has multiple devices, you must specify which device to rename
using the --old-name flag. If the user has only one device, the --old-name
flag is optional.

The new device name must be alphanumeric with hyphens and underscores only.

Examples:
  # Rename the only device for a user
  kanuka config rename-device --user alice@example.com new-laptop

  # Rename a specific device when user has multiple
  kanuka config rename-device --user alice@example.com --old-name macbook personal-macbook`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ConfigLogger.Infof("Starting rename-device command")
		spinner, cleanup := startSpinnerWithFlags("Renaming device...", configVerbose, configDebug)
		defer cleanup()

		newName := args[0]
		ConfigLogger.Debugf("New device name: %s, user: %s, old-name: %s", newName, renameDeviceUserEmail, renameDeviceOldName)

		// Validate new device name format.
		if !isValidDeviceName(newName) {
			finalMessage := color.RedString("✗") + " Invalid device name: " + color.YellowString(newName) + "\n" +
				color.CyanString("→") + " Device name must be alphanumeric with hyphens and underscores only"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Initialize project settings.
		if err := configs.InitProjectSettings(); err != nil {
			finalMessage := color.RedString("✗") + " Failed to initialize project settings\n" +
				color.CyanString("→") + " Make sure you're in a Kānuka project directory"
			spinner.FinalMSG = finalMessage
			return nil
		}

		if configs.ProjectKanukaSettings.ProjectPath == "" {
			finalMessage := color.RedString("✗") + " Not in a Kānuka project directory\n" +
				color.CyanString("→") + " Run this command from within a Kānuka project"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Load project config.
		projectConfig, err := configs.LoadProjectConfig()
		if err != nil {
			return ConfigLogger.ErrorfAndReturn("Failed to load project config: %v", err)
		}

		// Get all devices for this user.
		devices := projectConfig.GetDevicesByEmail(renameDeviceUserEmail)
		if len(devices) == 0 {
			finalMessage := color.RedString("✗") + " User " + color.YellowString(renameDeviceUserEmail) + " not found in this project\n" +
				color.CyanString("→") + " No devices found for this user"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Determine which device to rename.
		var targetUUID string
		var oldDeviceName string

		if len(devices) == 1 {
			// Only one device, use it.
			for uuid, device := range devices {
				targetUUID = uuid
				oldDeviceName = device.Name
			}
			ConfigLogger.Debugf("User has one device, using UUID: %s, name: %s", targetUUID, oldDeviceName)

			// If old-name was provided, verify it matches.
			if renameDeviceOldName != "" && renameDeviceOldName != oldDeviceName {
				finalMessage := color.RedString("✗") + " Device " + color.YellowString(renameDeviceOldName) + " not found for user " + color.YellowString(renameDeviceUserEmail) + "\n" +
					color.CyanString("→") + " The only device is: " + color.CyanString(oldDeviceName)
				spinner.FinalMSG = finalMessage
				return nil
			}
		} else {
			// Multiple devices, require --old-name.
			if renameDeviceOldName == "" {
				finalMessage := color.RedString("✗") + " User " + color.YellowString(renameDeviceUserEmail) + " has multiple devices\n" +
					color.CyanString("→") + " Specify which device to rename with " + color.YellowString("--old-name") + "\n" +
					color.CyanString("→") + " Available devices:\n"
				for _, device := range devices {
					finalMessage += "    - " + color.YellowString(device.Name) + "\n"
				}
				spinner.FinalMSG = finalMessage
				return nil
			}

			// Find the device by old name.
			found := false
			for uuid, device := range devices {
				if device.Name == renameDeviceOldName {
					targetUUID = uuid
					oldDeviceName = device.Name
					found = true
					break
				}
			}

			if !found {
				finalMessage := color.RedString("✗") + " Device " + color.YellowString(renameDeviceOldName) + " not found for user " + color.YellowString(renameDeviceUserEmail) + "\n" +
					color.CyanString("→") + " Available devices:\n"
				for _, device := range devices {
					finalMessage += "    - " + color.YellowString(device.Name) + "\n"
				}
				spinner.FinalMSG = finalMessage
				return nil
			}
		}

		// Check if new name is same as old name.
		if newName == oldDeviceName {
			finalMessage := color.YellowString("⚠") + " Device is already named " + color.CyanString(newName)
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Check if new name is already taken by this user.
		if projectConfig.IsDeviceNameTakenByEmail(renameDeviceUserEmail, newName) {
			finalMessage := color.RedString("✗") + " Device name " + color.YellowString(newName) + " is already in use for " + color.CyanString(renameDeviceUserEmail) + "\n" +
				color.CyanString("→") + " Choose a different device name"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Update the device name.
		device := projectConfig.Devices[targetUUID]
		device.Name = newName
		projectConfig.Devices[targetUUID] = device

		if err := configs.SaveProjectConfig(projectConfig); err != nil {
			return ConfigLogger.ErrorfAndReturn("Failed to save project config: %v", err)
		}

		// If the device being renamed belongs to the current user, also update their user config.
		userConfig, err := configs.LoadUserConfig()
		if err != nil {
			ConfigLogger.Debugf("Could not load user config to check if device is owned by current user: %v", err)
		} else if userConfig.User.UUID == targetUUID {
			// This is the current user's device, update their [projects] section.
			projectUUID := projectConfig.Project.UUID
			if projectUUID != "" && userConfig.Projects != nil {
				userConfig.Projects[projectUUID] = newName
				if err := configs.SaveUserConfig(userConfig); err != nil {
					ConfigLogger.Debugf("Could not update user config with new device name: %v", err)
				} else {
					ConfigLogger.Infof("Updated user config [projects] with new device name")
				}
			}
		}

		ConfigLogger.Infof("Device renamed successfully from %s to %s", oldDeviceName, newName)
		finalMessage := color.GreenString("✓") + " Device " + color.YellowString(oldDeviceName) + " renamed to " + color.CyanString(newName) + " for " + color.YellowString(renameDeviceUserEmail) + "\n"
		spinner.FinalMSG = finalMessage
		return nil
	},
}
