package cmd

import (
	"strings"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/utils"

	"github.com/PolarWolf314/kanuka/internal/ui"
	"github.com/spf13/cobra"
)

var (
	setProjectDeviceUUID string
)

func init() {
	setProjectDeviceCmd.Flags().StringVar(&setProjectDeviceUUID, "project-uuid", "", "project UUID (defaults to current project)")
	ConfigCmd.AddCommand(setProjectDeviceCmd)
}

// resetSetProjectDeviceState resets the set-project-device command's global state for testing.
func resetSetProjectDeviceState() {
	setProjectDeviceUUID = ""
}

var setProjectDeviceCmd = &cobra.Command{
	Use:   "set-project-device [device-name]",
	Short: "Set your device name for a project",
	Long: `Sets your device name for a project in both user and project configuration.

This command updates your device name in:
  - Your user config file (~/.config/kanuka/config.toml)
  - The project's config.toml file

This is the command to use when you want to change your device name for
an existing project.

The device name must be alphanumeric with hyphens and underscores only.
If no project UUID is specified, the current project is used.

Examples:
  # Set device name for the current project
  kanuka config set-project-device my-laptop

  # Set device name for a specific project
  kanuka config set-project-device --project-uuid 550e8400-e29b-41d4-a716-446655440000 workstation`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ConfigLogger.Infof("Starting set-project-device command")
		spinner, cleanup := startSpinnerWithFlags("Setting device name...", configVerbose, configDebug)
		defer cleanup()

		deviceName := args[0]
		ConfigLogger.Debugf("Device name argument: %s", deviceName)

		// Validate device name format.
		if !utils.IsValidDeviceName(deviceName) {
			finalMessage := ui.Error.Sprint("✗") + " Invalid device name: " + ui.Highlight.Sprint(deviceName) + "\n" +
				ui.Info.Sprint("→") + " Device name must be alphanumeric with hyphens and underscores only"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Determine project UUID.
		var projectUUID string
		if setProjectDeviceUUID != "" {
			projectUUID = setProjectDeviceUUID
			ConfigLogger.Debugf("Using provided project UUID: %s", projectUUID)
		} else {
			// Try to get from current project.
			ConfigLogger.Debugf("No project UUID provided, checking current project")
			if err := configs.InitProjectSettings(); err != nil {
				finalMessage := ui.Error.Sprint("✗") + " Failed to initialize project settings: " + err.Error() + "\n" +
					ui.Info.Sprint("→") + " Use " + ui.Flag.Sprint("--project-uuid") + " to specify a project"
				spinner.FinalMSG = finalMessage
				return nil
			}

			if configs.ProjectKanukaSettings.ProjectPath == "" {
				finalMessage := ui.Error.Sprint("✗") + " Not in a Kānuka project directory\n" +
					ui.Info.Sprint("→") + " Use " + ui.Flag.Sprint("--project-uuid") + " to specify a project"
				spinner.FinalMSG = finalMessage
				return nil
			}

			projectConfig, err := configs.LoadProjectConfig()
			if err != nil {
				if strings.Contains(err.Error(), "toml:") {
					return ConfigLogger.ErrorfAndReturn("Failed to load project config: .kanuka/config.toml is not valid TOML\n\n"+
						"To fix this issue:\n"+
						"  1. Restore the file from git: git checkout .kanuka/config.toml\n"+
						"  2. Or contact your project administrator for assistance\n\n"+
						"Details: %v", err)
				}
				return ConfigLogger.ErrorfAndReturn("Failed to load project config: %v", err)
			}
			projectUUID = projectConfig.Project.UUID
			ConfigLogger.Debugf("Using current project UUID: %s", projectUUID)
		}

		if projectUUID == "" {
			finalMessage := ui.Error.Sprint("✗") + " Could not determine project UUID\n" +
				ui.Info.Sprint("→") + " Use " + ui.Flag.Sprint("--project-uuid") + " to specify a project"
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
			userConfig.Projects = make(map[string]configs.UserProjectEntry)
		}

		// Check if there's an existing device name for this project.
		existingEntry, hasExisting := userConfig.Projects[projectUUID]
		if hasExisting && existingEntry.DeviceName == deviceName {
			finalMessage := ui.Warning.Sprint("⚠") + " Device name is already set to " + ui.Highlight.Sprint(deviceName) + " for this project"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Get project name to store.
		projectName := ""
		if configs.ProjectKanukaSettings.ProjectPath != "" {
			projectConfig, err := configs.LoadProjectConfig()
			if err == nil && projectConfig.Project.Name != "" {
				projectName = projectConfig.Project.Name
			}
		}

		// Set the device name, preserving existing project name if available.
		if hasExisting && existingEntry.ProjectName != "" {
			projectName = existingEntry.ProjectName
		}
		userConfig.Projects[projectUUID] = configs.UserProjectEntry{
			DeviceName:  deviceName,
			ProjectName: projectName,
		}
		ConfigLogger.Debugf("Setting device name for project %s to %s", projectUUID, deviceName)

		if err := configs.SaveUserConfig(userConfig); err != nil {
			return ConfigLogger.ErrorfAndReturn("Failed to save user config: %v", err)
		}
		ConfigLogger.Infof("User config saved successfully")

		if configs.ProjectKanukaSettings.ProjectPath != "" {
			ConfigLogger.Debugf("Updating project config")
			projectConfig, err := configs.LoadProjectConfig()
			if err != nil {
				if strings.Contains(err.Error(), "toml:") {
					ConfigLogger.Errorf("Failed to load project config: %v", err)
					finalMessage := ui.Error.Sprint("✗") + " Failed to load project configuration.\n\n" +
						ui.Info.Sprint("→") + " The .kanuka/config.toml file is not valid TOML.\n" +
						"   " + ui.Code.Sprint(err.Error()) + "\n\n" +
						"   To fix this issue:\n" +
						"   1. Restore the file from git: " + ui.Code.Sprint("git checkout .kanuka/config.toml") + "\n" +
						"   2. Or contact your project administrator for assistance"
					spinner.FinalMSG = finalMessage
					spinner.Stop()
					return nil
				}
				return ConfigLogger.ErrorfAndReturn("Failed to load project config: %v", err)
			}

			if deviceConfig, exists := projectConfig.Devices[userConfig.User.UUID]; exists {
				deviceConfig.Name = deviceName
				projectConfig.Devices[userConfig.User.UUID] = deviceConfig

				if err := configs.SaveProjectConfig(projectConfig); err != nil {
					return ConfigLogger.ErrorfAndReturn("Failed to save project config: %v", err)
				}
				ConfigLogger.Infof("Project config updated successfully")
			} else {
				ConfigLogger.Warnf("Device not found in project config - only user config updated")
			}
		}

		ConfigLogger.Infof("Device name set successfully")

		// Build success message.
		var finalMessage string
		if hasExisting && existingEntry.DeviceName != "" {
			finalMessage = ui.Success.Sprint("✓") + " Device name updated from " + ui.Highlight.Sprint(existingEntry.DeviceName) + " to " + ui.Highlight.Sprint(deviceName)
		} else {
			finalMessage = ui.Success.Sprint("✓") + " Device name set to " + ui.Highlight.Sprint(deviceName)
		}

		// Try to get project name for display.
		if configs.ProjectKanukaSettings.ProjectPath != "" {
			projectConfig, err := configs.LoadProjectConfig()
			if err == nil && projectConfig.Project.Name != "" {
				finalMessage += " for project " + ui.Highlight.Sprint(projectConfig.Project.Name)
			}
		}

		spinner.FinalMSG = finalMessage
		return nil
	},
}
