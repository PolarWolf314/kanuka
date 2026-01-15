package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/secrets"
	"github.com/PolarWolf314/kanuka/internal/utils"

	"github.com/PolarWolf314/kanuka/internal/ui"
	"github.com/spf13/cobra"
)

var (
	configInitEmail      string
	configInitName       string
	configInitDeviceName string
)

func init() {
	configInitCmd.Flags().StringVarP(&configInitEmail, "email", "e", "", "your email address")
	configInitCmd.Flags().StringVarP(&configInitName, "name", "n", "", "your display name (optional)")
	configInitCmd.Flags().StringVar(&configInitDeviceName, "device", "", "default device name (defaults to hostname)")
	ConfigCmd.AddCommand(configInitCmd)
}

// resetConfigInitState resets the config init command's global state for testing.
func resetConfigInitState() {
	configInitEmail = ""
	configInitName = ""
	configInitDeviceName = ""
}

// promptForInput prompts the user for input with an optional default value.
func promptForInput(reader *bufio.Reader, prompt, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)
	if input == "" && defaultValue != "" {
		return defaultValue, nil
	}
	return input, nil
}

// RunConfigInit runs the config init logic and returns whether setup was performed.
// This is exported so it can be called from secrets init.
func RunConfigInit(verbose, debug bool) (bool, error) {
	// Load existing user config if any.
	userConfig, err := configs.LoadUserConfig()
	if err != nil {
		return false, fmt.Errorf("failed to load user config: %w", err)
	}

	// Check if config is already complete.
	if userConfig.User.Email != "" && userConfig.User.UUID != "" {
		return false, nil // Already configured, no setup needed
	}

	// Need to run setup.
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(ui.Info.Sprint("Welcome to Kanuka!") + " Let's set up your identity.\n")

	// Prompt for email.
	var email string
	if configInitEmail != "" {
		email = configInitEmail
		if verbose {
			fmt.Println("[info] Using email from flag: " + email)
		}
	} else {
		defaultEmail := userConfig.User.Email
		promptedEmail, err := promptForInput(reader, "Email address", defaultEmail)
		if err != nil {
			return false, err
		}
		email = promptedEmail
	}

	// Validate email.
	if !utils.IsValidEmail(email) {
		return false, fmt.Errorf("invalid email format: %s", email)
	}
	if verbose {
		fmt.Println("[info] Email validated successfully")
	}

	// Prompt for display name (optional).
	var displayName string
	if configInitName != "" {
		displayName = configInitName
		if verbose {
			fmt.Println("[info] Using display name from flag: " + displayName)
		}
	} else {
		defaultName := userConfig.User.Name
		promptedName, err := promptForInput(reader, "Display name (optional)", defaultName)
		if err != nil {
			return false, err
		}
		displayName = promptedName
	}

	// Prompt for default device name.
	var deviceName string
	if configInitDeviceName != "" {
		deviceName = utils.SanitizeDeviceName(configInitDeviceName)
		if verbose {
			fmt.Println("[info] Using device name from flag: " + deviceName)
		}
	} else {
		// Generate default from hostname.
		defaultDevice, _ := utils.GenerateDeviceName([]string{})
		if userConfig.User.DefaultDeviceName != "" {
			defaultDevice = userConfig.User.DefaultDeviceName
		}
		promptedDevice, err := promptForInput(reader, "Default device name", defaultDevice)
		if err != nil {
			return false, err
		}
		deviceName = utils.SanitizeDeviceName(promptedDevice)
	}

	// Validate device name.
	if !utils.IsValidDeviceName(deviceName) {
		return false, fmt.Errorf("invalid device name: %s (must be alphanumeric with hyphens and underscores)", deviceName)
	}
	if verbose {
		fmt.Println("[info] Device name validated: " + deviceName)
	}

	// Update user config.
	userConfig.User.Email = email
	userConfig.User.Name = displayName
	userConfig.User.DefaultDeviceName = deviceName

	// Generate UUID if not present.
	if userConfig.User.UUID == "" {
		userConfig.User.UUID = configs.GenerateUserUUID()
		if verbose {
			fmt.Println("[info] Generated new user UUID: " + userConfig.User.UUID)
		}
	}

	// Initialize projects map if nil.
	if userConfig.Projects == nil {
		userConfig.Projects = make(map[string]configs.UserProjectEntry)
	}

	// Save user config.
	if err := configs.SaveUserConfig(userConfig); err != nil {
		return false, fmt.Errorf("failed to save user config: %w", err)
	}
	if verbose {
		fmt.Println("[info] User configuration saved to " + configs.UserKanukaSettings.UserConfigsPath + "/config.toml")
	}

	// Display summary.
	fmt.Println()
	fmt.Println(ui.Success.Sprint("✓") + " User configuration saved to " + ui.Path.Sprint(configs.UserKanukaSettings.UserConfigsPath+"/config.toml"))
	fmt.Println()
	fmt.Println("Your settings:")
	fmt.Println("  Email:   " + ui.Highlight.Sprint(email))
	if displayName != "" {
		fmt.Println("  Name:    " + ui.Highlight.Sprint(displayName))
	}
	fmt.Println("  Device:  " + ui.Highlight.Sprint(deviceName))
	fmt.Println("  User ID: " + ui.Highlight.Sprint(userConfig.User.UUID))
	fmt.Println()

	return true, nil
}

// IsUserConfigComplete checks if the user config has all required fields.
func IsUserConfigComplete() (bool, error) {
	userConfig, err := configs.LoadUserConfig()
	if err != nil {
		return false, err
	}
	return userConfig.User.Email != "" && userConfig.User.UUID != "", nil
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize your user configuration",
	Long: `Sets up your Kanuka user identity.

This command creates or updates your user configuration file at
~/.config/kanuka/config.toml with your identity information.

The command will prompt for:
  - Email address (required, used as your identifier)
  - Display name (optional, for future audit log features)
  - Default device name (defaults to your hostname)

You can also provide values via flags for non-interactive usage.

Examples:
  # Interactive setup
  kanuka config init

  # Non-interactive setup
  kanuka config init --email alice@example.com --device macbook

  # With all options
  kanuka config init --email alice@example.com --name "Alice Smith" --device workstation`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ConfigLogger.Infof("Starting config init command")
		ConfigLogger.Debugf("Flags: email=%s, name=%s, device=%s", configInitEmail, configInitName, configInitDeviceName)

		// Ensure user settings directory exists.
		ConfigLogger.Debugf("Ensuring user settings directory exists")
		if err := secrets.EnsureUserSettings(); err != nil {
			return ConfigLogger.ErrorfAndReturn("Failed to initialize user settings: %v", err)
		}
		ConfigLogger.Infof("User settings directory initialized")

		// Check if already configured and no flags provided.
		ConfigLogger.Debugf("Checking if user config is complete")
		isComplete, err := IsUserConfigComplete()
		if err != nil {
			return ConfigLogger.ErrorfAndReturn("Failed to check user config: %v", err)
		}

		// If already complete and no flags provided, show current config.
		if isComplete && configInitEmail == "" && configInitName == "" && configInitDeviceName == "" {
			ConfigLogger.Infof("User config already exists, showing current configuration")
			userConfig, err := configs.LoadUserConfig()
			if err != nil {
				return ConfigLogger.ErrorfAndReturn("Failed to load user config: %v", err)
			}

			fmt.Println(ui.Success.Sprint("✓") + " User configuration already exists\n")
			fmt.Println("Your settings:")
			fmt.Println("  Email:   " + ui.Highlight.Sprint(userConfig.User.Email))
			if userConfig.User.Name != "" {
				fmt.Println("  Name:    " + ui.Highlight.Sprint(userConfig.User.Name))
			}
			if userConfig.User.DefaultDeviceName != "" {
				fmt.Println("  Device:  " + ui.Highlight.Sprint(userConfig.User.DefaultDeviceName))
			}
			fmt.Println("  User ID: " + ui.Highlight.Sprint(userConfig.User.UUID))
			fmt.Println()
			fmt.Println(ui.Info.Sprint("→") + " Run with flags to update: " + ui.Code.Sprint("kanuka config init --email new@email.com"))
			return nil
		}

		// If flags are provided, update directly without prompts.
		if configInitEmail != "" || configInitName != "" || configInitDeviceName != "" {
			ConfigLogger.Infof("Flags provided, updating configuration directly")
			ConfigLogger.Debugf("Loading existing user config")
			userConfig, err := configs.LoadUserConfig()
			if err != nil {
				return ConfigLogger.ErrorfAndReturn("Failed to load user config: %v", err)
			}

			// Update only provided fields.
			if configInitEmail != "" {
				if !utils.IsValidEmail(configInitEmail) {
					fmt.Println(ui.Error.Sprint("✗") + " Invalid email format: " + ui.Highlight.Sprint(configInitEmail))
					return nil
				}
				ConfigLogger.Infof("Updating email to: %s", configInitEmail)
				userConfig.User.Email = configInitEmail
			}

			if configInitName != "" {
				ConfigLogger.Infof("Updating display name to: %s", configInitName)
				userConfig.User.Name = configInitName
			}

			if configInitDeviceName != "" {
				deviceName := utils.SanitizeDeviceName(configInitDeviceName)
				if !utils.IsValidDeviceName(deviceName) {
					fmt.Println(ui.Error.Sprint("✗") + " Invalid device name: " + ui.Highlight.Sprint(configInitDeviceName))
					return nil
				}
				ConfigLogger.Infof("Updating device name to: %s", deviceName)
				userConfig.User.DefaultDeviceName = deviceName
			}

			// Generate UUID if not present.
			if userConfig.User.UUID == "" {
				userConfig.User.UUID = configs.GenerateUserUUID()
				ConfigLogger.Infof("Generated new user UUID: %s", userConfig.User.UUID)
			}

			// Initialize projects map if nil.
			if userConfig.Projects == nil {
				userConfig.Projects = make(map[string]configs.UserProjectEntry)
			}

			// Save.
			ConfigLogger.Debugf("Saving user config to disk")
			if err := configs.SaveUserConfig(userConfig); err != nil {
				return ConfigLogger.ErrorfAndReturn("Failed to save user config: %v", err)
			}
			ConfigLogger.Infof("User config saved successfully")

			fmt.Println(ui.Success.Sprint("✓") + " User configuration updated\n")
			fmt.Println("Your settings:")
			fmt.Println("  Email:   " + ui.Highlight.Sprint(userConfig.User.Email))
			if userConfig.User.Name != "" {
				fmt.Println("  Name:    " + ui.Highlight.Sprint(userConfig.User.Name))
			}
			if userConfig.User.DefaultDeviceName != "" {
				fmt.Println("  Device:  " + ui.Highlight.Sprint(userConfig.User.DefaultDeviceName))
			}
			fmt.Println("  User ID: " + ui.Highlight.Sprint(userConfig.User.UUID))
			return nil
		}

		// Run interactive setup.
		ConfigLogger.Infof("Running interactive setup")
		_, err = RunConfigInit(configVerbose, configDebug)
		if err != nil {
			fmt.Println(ui.Error.Sprint("✗") + " " + err.Error())
			return nil
		}

		return nil
	},
}
