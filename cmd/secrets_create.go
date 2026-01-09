package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PolarWolf314/kanuka/internal/audit"
	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/secrets"
	"github.com/PolarWolf314/kanuka/internal/ui"
	"github.com/PolarWolf314/kanuka/internal/utils"

	"github.com/spf13/cobra"
)

var (
	force         bool
	createEmail   string
	createDevName string
)

func init() {
	createCmd.Flags().BoolVarP(&force, "force", "f", false, "force key creation")
	createCmd.Flags().StringVarP(&createEmail, "email", "e", "", "your email address for identification")
	createCmd.Flags().StringVar(&createDevName, "device-name", "", "custom device name (auto-generated from hostname if not specified)")
}

// resetCreateCommandState resets the create command's global state for testing.
func resetCreateCommandState() {
	force = false
	createEmail = ""
	createDevName = ""
}

// promptForEmail prompts the user for their email address.
func promptForEmail(reader *bufio.Reader) (string, error) {
	fmt.Print("Enter your email: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read email: %w", err)
	}
	return strings.TrimSpace(email), nil
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates and adds your public key, and gives instructions on how to gain access",
	Long: `Creates a new RSA key pair for accessing the project's encrypted secrets.

This command generates a unique cryptographic identity for you on this device,
identified by your email address. Each device you use gets its own key pair.

The command will:
  1. Generate an RSA key pair (stored locally in ~/.local/share/kanuka/keys/)
  2. Copy your public key to the project's .kanuka/public_keys/ directory
  3. Register your device in the project configuration

After running this command, you need to:
  1. Commit the new .kanuka/public_keys/<uuid>.pub file
  2. Ask someone with access to run: kanuka secrets register --user <your-email>

Examples:
  # Create keys with email prompt
  kanuka secrets create

  # Create keys with email specified
  kanuka secrets create --email alice@example.com

  # Create keys with custom device name
  kanuka secrets create --email alice@example.com --device-name macbook-pro

  # Force recreate keys (overwrites existing)
  kanuka secrets create --force`,
	RunE: func(cmd *cobra.Command, args []string) error {
		Logger.Infof("Starting create command")
		spinner, cleanup := startSpinner("Creating Kānuka file...", verbose)
		defer cleanup()

		Logger.Debugf("Initializing project settings")
		if err := configs.InitProjectSettings(); err != nil {
			return Logger.ErrorfAndReturn("failed to init project settings: %v", err)
		}
		projectPath := configs.ProjectKanukaSettings.ProjectPath
		Logger.Debugf("Project path: %s", projectPath)

		if projectPath == "" {
			finalMessage := ui.Error.Sprint("✗") + " Kānuka has not been initialized\n" +
				ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " instead"
			spinner.FinalMSG = finalMessage
			return nil
		}

		Logger.Debugf("Ensuring user settings")
		if err := secrets.EnsureUserSettings(); err != nil {
			return Logger.ErrorfAndReturn("Failed ensuring user settings: %v", err)
		}

		// Ensure user config has UUID
		Logger.Debugf("Ensuring user config with UUID")
		userConfig, err := configs.EnsureUserConfig()
		if err != nil {
			return Logger.ErrorfAndReturn("Failed to ensure user config: %v", err)
		}
		userUUID := userConfig.User.UUID
		Logger.Debugf("Current user UUID: %s", userUUID)

		// Handle email: use flag, existing config, or prompt
		userEmail := createEmail
		if userEmail == "" {
			userEmail = userConfig.User.Email
		}

		// If still no email, prompt for it
		if userEmail == "" {
			spinner.Stop()
			reader := bufio.NewReader(os.Stdin)
			promptedEmail, err := promptForEmail(reader)
			if err != nil {
				return Logger.ErrorfAndReturn("Failed to read email: %v", err)
			}
			userEmail = promptedEmail
			spinner.Restart()
		}

		// Validate email format
		if !utils.IsValidEmail(userEmail) {
			finalMessage := ui.Error.Sprint("✗") + " Invalid email format: " + ui.Highlight.Sprint(userEmail) + "\n" +
				ui.Info.Sprint("→") + " Please provide a valid email address"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Update user config with email if changed
		if userConfig.User.Email != userEmail {
			userConfig.User.Email = userEmail
			if err := configs.SaveUserConfig(userConfig); err != nil {
				return Logger.ErrorfAndReturn("Failed to save user config: %v", err)
			}
			Logger.Infof("User email updated to: %s", userEmail)
		}

		// Load project config to check existing devices for this user
		Logger.Debugf("Loading project config for device name validation")
		projectConfig, err := configs.LoadProjectConfig()
		if err != nil {
			return Logger.ErrorfAndReturn("Failed to load project config: %v", err)
		}

		// Determine device name: use flag, or auto-generate from hostname
		var deviceName string
		existingDeviceNames := projectConfig.GetDeviceNamesByEmail(userEmail)

		if createDevName != "" {
			// User provided a device name - sanitize and validate uniqueness
			deviceName = utils.SanitizeDeviceName(createDevName)
			Logger.Debugf("Using user-provided device name: %s (sanitized from: %s)", deviceName, createDevName)

			// Check if device name is already taken by this user
			if projectConfig.IsDeviceNameTakenByEmail(userEmail, deviceName) {
				finalMessage := ui.Error.Sprint("✗") + " Device name " + ui.Highlight.Sprint(deviceName) + " is already in use for " + ui.Highlight.Sprint(userEmail) + "\n" +
					ui.Info.Sprint("→") + " Choose a different device name with " + ui.Flag.Sprint("--device-name")
				spinner.FinalMSG = finalMessage
				return nil
			}
		} else {
			// Auto-generate device name from hostname
			deviceName, err = utils.GenerateDeviceName(existingDeviceNames)
			if err != nil {
				return Logger.ErrorfAndReturn("Failed to generate device name: %v", err)
			}
			Logger.Debugf("Auto-generated device name: %s", deviceName)
		}

		// If force flag is active, then ignore checking for existing symmetric key
		if !force {
			Logger.Debugf("Force flag not set, checking for existing public key")
			projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath
			// Check for public key using user UUID
			userPublicKeyPath := filepath.Join(projectPublicKeyPath, userUUID+".pub")
			Logger.Debugf("Checking for existing public key at: %s", userPublicKeyPath)

			// We are explicitly ignoring errors, because an error means the key doesn't exist, which is what we want.
			userPublicKey, _ := secrets.LoadPublicKey(userPublicKeyPath)

			if userPublicKey != nil {
				finalMessage := ui.Error.Sprint("✗ ") + ui.Path.Sprint(userUUID+".pub ") + "already exists\n" +
					"To override, run: " + ui.Code.Sprint("kanuka secrets create --force")
				spinner.FinalMSG = finalMessage
				return nil
			}
		} else {
			Logger.Infof("Force flag set, will override existing keys if present")
			spinner.Stop()
			Logger.WarnfUser("Using --force flag will overwrite existing keys - ensure you have backups")
			spinner.Restart()
		}

		Logger.Debugf("Creating and saving RSA key pair")
		if err := secrets.CreateAndSaveRSAKeyPair(verbose); err != nil {
			return Logger.ErrorfAndReturn("Failed to generate and save RSA key pair: %v", err)
		}
		Logger.Infof("RSA key pair created successfully")

		Logger.Debugf("Copying user public key to project")
		destPath, err := secrets.CopyUserPublicKeyToProject()
		if err != nil {
			return Logger.ErrorfAndReturn("Failed to copy public key to project: %v", err)
		}
		Logger.Infof("Public key copied to: %s", destPath)

		// Add/update user in project config
		Logger.Debugf("Updating project config with user info")
		projectConfig.Users[userUUID] = userEmail
		projectConfig.Devices[userUUID] = configs.DeviceConfig{
			Email:     userEmail,
			Name:      deviceName,
			CreatedAt: time.Now().UTC(),
		}

		if err := configs.SaveProjectConfig(projectConfig); err != nil {
			return Logger.ErrorfAndReturn("Failed to save project config: %v", err)
		}
		Logger.Infof("Project config updated successfully")

		// Update user config with project entry
		Logger.Debugf("Updating user config with project entry")
		if userConfig.Projects == nil {
			userConfig.Projects = make(map[string]configs.UserProjectEntry)
		}
		userConfig.Projects[projectConfig.Project.UUID] = configs.UserProjectEntry{
			DeviceName:  deviceName,
			ProjectName: projectConfig.Project.Name,
		}
		if err := configs.SaveUserConfig(userConfig); err != nil {
			return Logger.ErrorfAndReturn("Failed to update user config with project: %v", err)
		}
		Logger.Infof("User config updated with project UUID: %s -> device: %s, project: %s", projectConfig.Project.UUID, deviceName, projectConfig.Project.Name)

		didKanukaExist := true

		projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath
		// Use user UUID for kanuka key path
		userKanukaKeyPath := filepath.Join(projectSecretsPath, userUUID+".kanuka")
		Logger.Debugf("Attempting to remove existing kanuka key at: %s", userKanukaKeyPath)

		if err := os.Remove(userKanukaKeyPath); err != nil {
			didKanukaExist = false
			Logger.Debugf("No existing kanuka key found (this is expected for new users)")
			// Explicitly ignore error as we want to idempotently delete the file
			_ = err
		} else {
			Logger.Infof("Removed existing kanuka key file")
		}

		deletedMessage := ""
		if didKanukaExist {
			deletedMessage = "    deleted: " + ui.Error.Sprint(userKanukaKeyPath) + "\n"
		}

		Logger.Infof("Create command completed successfully for user: %s (%s)", userEmail, userUUID)

		// Log to audit trail.
		auditEntry := audit.LogWithUser("create")
		auditEntry.DeviceName = deviceName
		audit.Log(auditEntry)

		finalMessage := ui.Success.Sprint("✓") + " Keys created for " + ui.Highlight.Sprint(userEmail) + " (device: " + ui.Highlight.Sprint(deviceName) + ")\n" +
			"    created: " + ui.Path.Sprint(destPath) + "\n" + deletedMessage +
			ui.Info.Sprint("To gain access to the secrets in this project:\n") +
			"  1. Commit your " + ui.Path.Sprint(".kanuka/public_keys/"+userUUID+".pub") + " file to your version control system\n" +
			"  2. Ask someone with permissions to grant you access with:\n" +
			"     " + ui.Code.Sprint("kanuka secrets register --user "+userEmail)

		spinner.FinalMSG = finalMessage
		return nil
	},
}
