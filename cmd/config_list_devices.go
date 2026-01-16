package cmd

import (
	"fmt"
	"sort"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/ui"
	"github.com/spf13/cobra"
)

var (
	listDevicesUserEmail string
)

func init() {
	listDevicesCmd.Flags().StringVarP(&listDevicesUserEmail, "user", "u", "", "filter by user email")
	ConfigCmd.AddCommand(listDevicesCmd)
}

// resetListDevicesState resets the list-devices command's global state for testing.
func resetListDevicesState() {
	listDevicesUserEmail = ""
}

var listDevicesCmd = &cobra.Command{
	Use:   "list-devices",
	Short: "List all devices in the project",
	Long: `Lists all devices registered in the project configuration.

This command displays all users and their devices that are registered
in the current project. You can filter by a specific user email.

Examples:
  # List all devices in the project
  kanuka config list-devices

  # List devices for a specific user
  kanuka config list-devices --user alice@example.com`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ConfigLogger.Infof("Starting list-devices command")
		ConfigLogger.Debugf("Flags: user=%s", listDevicesUserEmail)

		spinner, cleanup := startSpinnerWithFlags("Loading devices...", configVerbose, configDebug)
		defer cleanup()

		// Initialize project settings.
		ConfigLogger.Debugf("Initializing project settings")
		if err := configs.InitProjectSettings(); err != nil {
			ConfigLogger.Infof("Failed to initialize project settings: %v", err)
			spinner.FinalMSG = ui.Error.Sprint("✗") + " Failed to initialize project settings\n"
			fmt.Println(ui.Info.Sprint("→") + " Make sure you're in a Kānuka project directory")
			return nil
		}

		if configs.ProjectKanukaSettings.ProjectPath == "" {
			ConfigLogger.Infof("Not in a Kanuka project directory")
			spinner.FinalMSG = ui.Error.Sprint("✗") + " Not in a Kānuka project directory\n"
			fmt.Println(ui.Info.Sprint("→") + " Run this command from within a Kānuka project")
			return nil
		}

		ConfigLogger.Debugf("Project path: %s", configs.ProjectKanukaSettings.ProjectPath)

		// Load project config.
		ConfigLogger.Debugf("Loading project config")
		projectConfig, err := configs.LoadProjectConfig()
		if err != nil {
			return ConfigLogger.ErrorfAndReturn("Failed to load project config: %v", err)
		}

		ConfigLogger.Infof("Project config loaded: %d devices found", len(projectConfig.Devices))

		if len(projectConfig.Devices) == 0 {
			spinner.FinalMSG = ui.Warning.Sprint("⚠") + " No devices found in this project\n"
			return nil
		}

		// Group devices by email.
		devicesByEmail := make(map[string][]deviceInfo)
		for uuid, device := range projectConfig.Devices {
			info := deviceInfo{
				UUID:      uuid,
				Name:      device.Name,
				CreatedAt: device.CreatedAt.Format("Jan 2, 2006"),
			}
			devicesByEmail[device.Email] = append(devicesByEmail[device.Email], info)
		}

		ConfigLogger.Debugf("Devices grouped by %d unique emails", len(devicesByEmail))

		// Filter by user if specified.
		if listDevicesUserEmail != "" {
			ConfigLogger.Infof("Filtering devices by user: %s", listDevicesUserEmail)
			devices, exists := devicesByEmail[listDevicesUserEmail]
			if !exists {
				spinner.FinalMSG = ui.Error.Sprint("✗") + " User " + ui.Highlight.Sprint(listDevicesUserEmail) + " not found in this project\n"
				return nil
			}
			ConfigLogger.Debugf("Found %d devices for user %s", len(devices), listDevicesUserEmail)
			devicesByEmail = map[string][]deviceInfo{listDevicesUserEmail: devices}
		}

		// Sort emails for consistent output.
		var emails []string
		for email := range devicesByEmail {
			emails = append(emails, email)
		}
		sort.Strings(emails)

		// Print header.
		projectName := projectConfig.Project.Name
		if projectName != "" {
			fmt.Printf("Devices in project %s:\n\n", ui.Highlight.Sprint(projectName))
		} else {
			fmt.Print("Devices in this project:\n\n")
		}

		// Print devices grouped by email.
		for _, email := range emails {
			devices := devicesByEmail[email]
			fmt.Printf("  %s\n", ui.Highlight.Sprint(email))

			// Sort devices by name for consistent output.
			sort.Slice(devices, func(i, j int) bool {
				return devices[i].Name < devices[j].Name
			})

			for _, device := range devices {
				shortUUID := device.UUID
				if len(shortUUID) > 8 {
					shortUUID = shortUUID[:8] + "..."
				}
				fmt.Printf("    - %s (UUID: %s) - created: %s\n",
					ui.Highlight.Sprint(device.Name),
					ui.Muted.Sprint(shortUUID),
					device.CreatedAt)
			}
			fmt.Println()
		}

		spinner.FinalMSG = ui.Success.Sprint("✓") + " Devices listed successfully\n"
		return nil
	},
}

type deviceInfo struct {
	UUID      string
	Name      string
	CreatedAt string
}
