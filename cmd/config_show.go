package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/secrets"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	configShowProject bool
	configShowJSON    bool
)

func init() {
	configShowCmd.Flags().BoolVarP(&configShowProject, "project", "p", false, "show project configuration instead of user configuration")
	configShowCmd.Flags().BoolVar(&configShowJSON, "json", false, "output in JSON format")
	ConfigCmd.AddCommand(configShowCmd)
}

// resetConfigShowState resets the config show command's global state for testing.
func resetConfigShowState() {
	configShowProject = false
	configShowJSON = false
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration",
	Long: `Displays the current Kanuka configuration.

By default, shows user configuration from ~/.config/kanuka/config.toml.
Use --project to show project configuration from .kanuka/config.toml.

Examples:
  # Show user configuration
  kanuka config show

  # Show project configuration (must be in a project directory)
  kanuka config show --project

  # Output in JSON format
  kanuka config show --json
  kanuka config show --project --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ConfigLogger.Infof("Starting config show command")
		ConfigLogger.Debugf("Flags: project=%t, json=%t", configShowProject, configShowJSON)

		if configShowProject {
			ConfigLogger.Infof("Showing project configuration")
			return showProjectConfig()
		}
		ConfigLogger.Infof("Showing user configuration")
		return showUserConfig()
	},
}

// showUserConfig displays the user configuration.
func showUserConfig() error {
	// Ensure user settings are initialized.
	ConfigLogger.Debugf("Ensuring user settings are initialized")
	if err := secrets.EnsureUserSettings(); err != nil {
		return ConfigLogger.ErrorfAndReturn("Failed to initialize user settings: %v", err)
	}

	ConfigLogger.Debugf("Loading user config from %s", configs.UserKanukaSettings.UserConfigsPath)
	userConfig, err := configs.LoadUserConfig()
	if err != nil {
		return ConfigLogger.ErrorfAndReturn("Failed to load user config: %v", err)
	}

	// Check if config exists.
	if userConfig.User.Email == "" && userConfig.User.UUID == "" {
		ConfigLogger.Infof("No user configuration found")
		if configShowJSON {
			fmt.Println("{}")
			return nil
		}
		fmt.Println(color.YellowString("⚠") + " No user configuration found.")
		fmt.Println()
		fmt.Println(color.CyanString("→") + " Run " + color.YellowString("kanuka config init") + " to set up your identity")
		return nil
	}

	ConfigLogger.Infof("User config loaded successfully (email: %s, UUID: %s)", userConfig.User.Email, userConfig.User.UUID)
	ConfigLogger.Debugf("User has %d project entries", len(userConfig.Projects))

	if configShowJSON {
		ConfigLogger.Debugf("Outputting user config as JSON")
		return outputUserConfigJSON(userConfig)
	}

	return outputUserConfigText(userConfig)
}

// outputUserConfigJSON outputs user config in JSON format.
func outputUserConfigJSON(config *configs.UserConfig) error {
	output, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return ConfigLogger.ErrorfAndReturn("Failed to marshal config to JSON: %v", err)
	}
	fmt.Println(string(output))
	return nil
}

// outputUserConfigText outputs user config in human-readable format.
func outputUserConfigText(config *configs.UserConfig) error {
	fmt.Println(color.CyanString("User Configuration") + " (~/.config/kanuka/config.toml):")
	fmt.Println()
	fmt.Printf("  %-14s %s\n", "Email:", color.GreenString(config.User.Email))
	if config.User.Name != "" {
		fmt.Printf("  %-14s %s\n", "Name:", color.GreenString(config.User.Name))
	}
	fmt.Printf("  %-14s %s\n", "User ID:", color.YellowString(config.User.UUID))
	if config.User.DefaultDeviceName != "" {
		fmt.Printf("  %-14s %s\n", "Default Device:", color.GreenString(config.User.DefaultDeviceName))
	}

	if len(config.Projects) > 0 {
		fmt.Println()
		fmt.Println(color.CyanString("Projects:"))

		// Sort project UUIDs for consistent output.
		var projectUUIDs []string
		for uuid := range config.Projects {
			projectUUIDs = append(projectUUIDs, uuid)
		}
		sort.Strings(projectUUIDs)

		for _, uuid := range projectUUIDs {
			entry := config.Projects[uuid]
			// Truncate UUID for display.
			shortUUID := uuid
			if len(uuid) > 8 {
				shortUUID = uuid[:8] + "..."
			}
			if entry.ProjectName != "" {
				fmt.Printf("  %s → %s (%s)\n", color.YellowString(shortUUID), color.GreenString(entry.DeviceName), color.CyanString(entry.ProjectName))
			} else {
				fmt.Printf("  %s → %s\n", color.YellowString(shortUUID), color.GreenString(entry.DeviceName))
			}
		}
	}

	return nil
}

// showProjectConfig displays the project configuration.
func showProjectConfig() error {
	// Check if we're in a project directory.
	ConfigLogger.Debugf("Checking if in a Kanuka project directory")
	exists, err := secrets.DoesProjectKanukaSettingsExist()
	if err != nil {
		return ConfigLogger.ErrorfAndReturn("Failed to check project settings: %v", err)
	}

	if !exists {
		ConfigLogger.Infof("Not in a Kanuka project directory")
		if configShowJSON {
			fmt.Println("{\"error\": \"not in a project directory\"}")
			return nil
		}
		fmt.Println(color.RedString("✗") + " Not in a Kanuka project directory")
		fmt.Println()
		fmt.Println(color.CyanString("→") + " Run " + color.YellowString("kanuka secrets init") + " to initialize a project")
		return nil
	}

	// Initialize project settings.
	ConfigLogger.Debugf("Initializing project settings")
	if err := configs.InitProjectSettings(); err != nil {
		return ConfigLogger.ErrorfAndReturn("Failed to initialize project settings: %v", err)
	}

	ConfigLogger.Debugf("Loading project config from %s/.kanuka/config.toml", configs.ProjectKanukaSettings.ProjectPath)
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		return ConfigLogger.ErrorfAndReturn("Failed to load project config: %v", err)
	}

	ConfigLogger.Infof("Project config loaded successfully (name: %s, UUID: %s)", projectConfig.Project.Name, projectConfig.Project.UUID)
	ConfigLogger.Debugf("Project has %d users and %d devices", len(projectConfig.Users), len(projectConfig.Devices))

	if configShowJSON {
		ConfigLogger.Debugf("Outputting project config as JSON")
		return outputProjectConfigJSON(projectConfig)
	}

	return outputProjectConfigText(projectConfig)
}

// outputProjectConfigJSON outputs project config in JSON format.
func outputProjectConfigJSON(config *configs.ProjectConfig) error {
	output, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return ConfigLogger.ErrorfAndReturn("Failed to marshal config to JSON: %v", err)
	}
	fmt.Println(string(output))
	return nil
}

// outputProjectConfigText outputs project config in human-readable format.
func outputProjectConfigText(config *configs.ProjectConfig) error {
	fmt.Println(color.CyanString("Project Configuration") + " (.kanuka/config.toml):")
	fmt.Println()
	fmt.Printf("  %-14s %s\n", "Project ID:", color.YellowString(config.Project.UUID))
	fmt.Printf("  %-14s %s\n", "Project Name:", color.GreenString(config.Project.Name))

	if len(config.Devices) > 0 {
		fmt.Println()
		fmt.Println(color.CyanString("Users:"))

		// Group devices by email using the shared deviceInfo type.
		devicesByEmail := make(map[string][]deviceInfo)
		for uuid, device := range config.Devices {
			createdStr := ""
			if !device.CreatedAt.IsZero() {
				createdStr = device.CreatedAt.Format("Jan 2, 2006")
			}
			devicesByEmail[device.Email] = append(devicesByEmail[device.Email], deviceInfo{
				UUID:      uuid,
				Name:      device.Name,
				CreatedAt: createdStr,
			})
		}

		// Sort emails for consistent output.
		var emails []string
		for email := range devicesByEmail {
			emails = append(emails, email)
		}
		sort.Strings(emails)

		for _, email := range emails {
			devices := devicesByEmail[email]
			// Sort devices by name.
			sort.Slice(devices, func(i, j int) bool {
				return devices[i].Name < devices[j].Name
			})

			// Get user UUID (first device's UUID will have it in the Users map).
			userUUID := ""
			for uuid, userEmail := range config.Users {
				if userEmail == email {
					userUUID = uuid
					break
				}
			}
			shortUUID := userUUID
			if len(userUUID) > 8 {
				shortUUID = userUUID[:8] + "..."
			}

			fmt.Printf("  %s (%s)\n", color.GreenString(email), color.YellowString(shortUUID))
			for _, device := range devices {
				createdDisplay := ""
				if device.CreatedAt != "" {
					createdDisplay = fmt.Sprintf(" (created: %s)", device.CreatedAt)
				}
				fmt.Printf("    - %s%s\n", device.Name, color.HiBlackString(createdDisplay))
			}
		}
	}

	return nil
}
