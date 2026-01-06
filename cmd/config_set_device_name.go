package cmd

import (
	"regexp"

	"github.com/PolarWolf314/kanuka/internal/configs"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	setDeviceProjectUUID string
)

func init() {
	setDeviceNameCmd.Flags().StringVar(&setDeviceProjectUUID, "project-uuid", "", "project UUID (defaults to current project)")
	ConfigCmd.AddCommand(setDeviceNameCmd)
}

// resetSetDeviceNameState resets the set-device-name command's global state for testing.
func resetSetDeviceNameState() {
	setDeviceProjectUUID = ""
}

var setDeviceNameCmd = &cobra.Command{
	Use:   "set-device-name [device-name]",
	Short: "Set your device name for a project",
	Long: `Sets your preferred device name for a project in your local user configuration.

This command stores a device name preference in your user config file
(~/.config/kanuka/config.toml). This name is used when you create keys
for a project.

The device name must be alphanumeric with hyphens and underscores only.
If no project UUID is specified, the current project is used.

Examples:
  # Set device name for the current project
  kanuka config set-device-name my-laptop

  # Set device name for a specific project
  kanuka config set-device-name --project-uuid 550e8400-e29b-41d4-a716-446655440000 workstation`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ConfigLogger.Infof("Starting set-device-name command")
		spinner, cleanup := startSpinner("Setting device name...", configVerbose)
		defer cleanup()

		deviceName := args[0]
		ConfigLogger.Debugf("Device name argument: %s", deviceName)

		// Validate device name format.
		if !isValidDeviceName(deviceName) {
			finalMessage := color.RedString("✗") + " Invalid device name: " + color.YellowString(deviceName) + "\n" +
				color.CyanString("→") + " Device name must be alphanumeric with hyphens and underscores only"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Determine project UUID.
		var projectUUID string
		if setDeviceProjectUUID != "" {
			projectUUID = setDeviceProjectUUID
			ConfigLogger.Debugf("Using provided project UUID: %s", projectUUID)
		} else {
			// Try to get from current project.
			ConfigLogger.Debugf("No project UUID provided, checking current project")
			if err := configs.InitProjectSettings(); err != nil {
				finalMessage := color.RedString("✗") + " Failed to initialize project settings: " + err.Error() + "\n" +
					color.CyanString("→") + " Use " + color.YellowString("--project-uuid") + " to specify a project"
				spinner.FinalMSG = finalMessage
				return nil
			}

			if configs.ProjectKanukaSettings.ProjectPath == "" {
				finalMessage := color.RedString("✗") + " Not in a Kānuka project directory\n" +
					color.CyanString("→") + " Use " + color.YellowString("--project-uuid") + " to specify a project"
				spinner.FinalMSG = finalMessage
				return nil
			}

			projectConfig, err := configs.LoadProjectConfig()
			if err != nil {
				return ConfigLogger.ErrorfAndReturn("Failed to load project config: %v", err)
			}
			projectUUID = projectConfig.Project.UUID
			ConfigLogger.Debugf("Using current project UUID: %s", projectUUID)
		}

		if projectUUID == "" {
			finalMessage := color.RedString("✗") + " Could not determine project UUID\n" +
				color.CyanString("→") + " Use " + color.YellowString("--project-uuid") + " to specify a project"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Load user config.
		ConfigLogger.Debugf("Loading user config")
		userConfig, err := configs.LoadUserConfig()
		if err != nil {
			return ConfigLogger.ErrorfAndReturn("Failed to load user config: %v", err)
		}

		// Initialize projects map if nil.
		if userConfig.Projects == nil {
			userConfig.Projects = make(map[string]string)
		}

		// Check if there's an existing device name for this project.
		oldName, hasExisting := userConfig.Projects[projectUUID]
		if hasExisting && oldName == deviceName {
			finalMessage := color.YellowString("⚠") + " Device name is already set to " + color.CyanString(deviceName) + " for this project"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Set the device name.
		userConfig.Projects[projectUUID] = deviceName
		ConfigLogger.Debugf("Setting device name for project %s to %s", projectUUID, deviceName)

		if err := configs.SaveUserConfig(userConfig); err != nil {
			return ConfigLogger.ErrorfAndReturn("Failed to save user config: %v", err)
		}

		ConfigLogger.Infof("Device name set successfully")

		// Build success message.
		var finalMessage string
		if hasExisting {
			finalMessage = color.GreenString("✓") + " Device name updated from " + color.YellowString(oldName) + " to " + color.CyanString(deviceName)
		} else {
			finalMessage = color.GreenString("✓") + " Device name set to " + color.CyanString(deviceName)
		}

		// Try to get project name for display.
		if configs.ProjectKanukaSettings.ProjectPath != "" {
			projectConfig, err := configs.LoadProjectConfig()
			if err == nil && projectConfig.Project.Name != "" {
				finalMessage += " for project " + color.YellowString(projectConfig.Project.Name)
			}
		}

		spinner.FinalMSG = finalMessage + "\n"
		return nil
	},
}

// isValidDeviceName checks if a device name is valid (alphanumeric, hyphens, underscores).
func isValidDeviceName(name string) bool {
	if name == "" {
		return false
	}
	// Match alphanumeric characters, hyphens, and underscores only.
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)
	return validPattern.MatchString(name)
}
