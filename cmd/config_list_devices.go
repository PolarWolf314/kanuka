package cmd

import (
	"fmt"
	"sort"

	"github.com/PolarWolf314/kanuka/internal/configs"

	"github.com/fatih/color"
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

		// Initialize project settings.
		if err := configs.InitProjectSettings(); err != nil {
			fmt.Println(color.RedString("✗") + " Failed to initialize project settings")
			fmt.Println(color.CyanString("→") + " Make sure you're in a Kānuka project directory")
			return nil
		}

		if configs.ProjectKanukaSettings.ProjectPath == "" {
			fmt.Println(color.RedString("✗") + " Not in a Kānuka project directory")
			fmt.Println(color.CyanString("→") + " Run this command from within a Kānuka project")
			return nil
		}

		// Load project config.
		projectConfig, err := configs.LoadProjectConfig()
		if err != nil {
			return ConfigLogger.ErrorfAndReturn("Failed to load project config: %v", err)
		}

		if len(projectConfig.Devices) == 0 {
			fmt.Println(color.YellowString("⚠") + " No devices found in this project")
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

		// Filter by user if specified.
		if listDevicesUserEmail != "" {
			devices, exists := devicesByEmail[listDevicesUserEmail]
			if !exists {
				fmt.Println(color.RedString("✗") + " User " + color.YellowString(listDevicesUserEmail) + " not found in this project")
				return nil
			}
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
			fmt.Printf("Devices in project %s:\n\n", color.CyanString(projectName))
		} else {
			fmt.Print("Devices in this project:\n\n")
		}

		// Print devices grouped by email.
		for _, email := range emails {
			devices := devicesByEmail[email]
			fmt.Printf("  %s\n", color.YellowString(email))

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
					color.CyanString(device.Name),
					color.WhiteString(shortUUID),
					device.CreatedAt)
			}
			fmt.Println()
		}

		return nil
	},
}

type deviceInfo struct {
	UUID      string
	Name      string
	CreatedAt string
}
