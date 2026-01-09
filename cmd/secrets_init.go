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
	initYes         bool
	initProjectName string
)

func init() {
	initCmd.Flags().BoolVarP(&initYes, "yes", "y", false, "non-interactive mode (fail if user config is incomplete)")
	initCmd.Flags().StringVarP(&initProjectName, "name", "n", "", "project name (defaults to directory name)")
}

// resetInitCommandState resets the init command's global state for testing.
func resetInitCommandState() {
	initYes = false
	initProjectName = ""
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initializes the secrets store",
	RunE: func(cmd *cobra.Command, args []string) error {
		Logger.Infof("Starting init command")
		spinner, cleanup := startSpinner("Initializing Kānuka...", verbose)
		defer cleanup()

		Logger.Debugf("Checking if project kanuka settings already exist")
		kanukaExists, err := secrets.DoesProjectKanukaSettingsExist()
		if err != nil {
			return Logger.ErrorfAndReturn("Failed to check if project kanuka settings exists: %v", err)
		}
		if kanukaExists {
			finalMessage := ui.Error.Sprint("✗") + " Kānuka has already been initialized\n" +
				ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets create") + " instead"
			spinner.FinalMSG = finalMessage
			return nil
		}

		Logger.Debugf("Ensuring user settings")
		if err := secrets.EnsureUserSettings(); err != nil {
			return Logger.ErrorfAndReturn("Failed ensuring user settings: %v", err)
		}
		Logger.Infof("User settings ensured successfully")

		// Check if user config is complete (has email and UUID).
		Logger.Debugf("Checking if user config is complete")
		isComplete, err := IsUserConfigComplete()
		if err != nil {
			return Logger.ErrorfAndReturn("Failed to check user config: %v", err)
		}

		if !isComplete {
			Logger.Infof("User config is incomplete, need to run setup")

			// If --yes flag is set, fail with clear error.
			if initYes {
				spinner.FinalMSG = ui.Error.Sprint("✗") + " User configuration is incomplete\n" +
					ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka config init") + " first to set up your identity"
				return fmt.Errorf("user configuration required: run 'kanuka config init' first")
			}

			// Run config init inline.
			spinner.Stop()
			fmt.Println(ui.Warning.Sprint("⚠") + " User configuration not found.\n")
			fmt.Println("Running initial setup...")
			fmt.Println()

			setupPerformed, err := RunConfigInit(verbose, debug)
			if err != nil {
				return Logger.ErrorfAndReturn("Failed to set up user config: %v", err)
			}

			if !setupPerformed {
				// Config was already complete (shouldn't happen, but handle gracefully).
				Logger.Debugf("Config init reported no setup needed")
			}

			// Restart spinner for project initialization.
			fmt.Println("Initializing project...")
			spinner.Restart()
		}

		Logger.Debugf("Ensuring kanuka settings and creating .kanuka folders")
		if err := secrets.EnsureKanukaSettings(); err != nil {
			return Logger.ErrorfAndReturn("Failed to create .kanuka folders: %v", err)
		}
		Logger.Infof("Kanuka settings and folders created successfully")

		// Ensure user config has UUID
		Logger.Debugf("Ensuring user config with UUID")
		userConfig, err := configs.EnsureUserConfig()
		if err != nil {
			return Logger.ErrorfAndReturn("Failed to ensure user config: %v", err)
		}
		Logger.Infof("User config ensured with UUID: %s", userConfig.User.UUID)

		// Create and save project config with UUID
		Logger.Debugf("Creating project config with UUID")
		wd, err := os.Getwd()
		if err != nil {
			return Logger.ErrorfAndReturn("Failed to get working directory: %v", err)
		}

		// Determine project name.
		defaultProjectName := filepath.Base(wd)
		var projectName string

		if initProjectName != "" {
			// Use flag value if provided.
			projectName = strings.TrimSpace(initProjectName)
			Logger.Debugf("Using project name from flag: %s", projectName)
		} else if initYes {
			// Non-interactive mode: use default.
			projectName = defaultProjectName
			Logger.Debugf("Using default project name (non-interactive): %s", projectName)
		} else {
			// Interactive mode: prompt for project name.
			spinner.Stop()
			reader := bufio.NewReader(os.Stdin)
			fmt.Printf("Project name [%s]: ", defaultProjectName)
			input, readErr := reader.ReadString('\n')
			if readErr != nil {
				return Logger.ErrorfAndReturn("Failed to read project name: %v", readErr)
			}
			projectName = strings.TrimSpace(input)
			if projectName == "" {
				projectName = defaultProjectName
			}
			spinner.Restart()
		}

		// Validate project name.
		if projectName == "" {
			return Logger.ErrorfAndReturn("Project name cannot be empty")
		}
		Logger.Infof("Using project name: %s", projectName)

		projectConfig := &configs.ProjectConfig{
			Project: configs.Project{
				UUID: configs.GenerateProjectUUID(),
				Name: projectName,
			},
			Users:   make(map[string]string),
			Devices: make(map[string]configs.DeviceConfig),
		}

		// Add the initializing user to the project config
		// Generate device name from hostname (no existing devices for this user in new project)
		deviceName, err := utils.GenerateDeviceName([]string{})
		if err != nil {
			return Logger.ErrorfAndReturn("Failed to generate device name: %v", err)
		}
		Logger.Debugf("Generated device name: %s", deviceName)

		projectConfig.Users[userConfig.User.UUID] = userConfig.User.Email
		projectConfig.Devices[userConfig.User.UUID] = configs.DeviceConfig{
			Email:     userConfig.User.Email,
			Name:      deviceName,
			CreatedAt: time.Now().UTC(),
		}

		// Save project config - need to set ProjectPath first for SaveProjectConfig to work
		configs.ProjectKanukaSettings.ProjectPath = wd
		if err := configs.SaveProjectConfig(projectConfig); err != nil {
			return Logger.ErrorfAndReturn("Failed to save project config: %v", err)
		}
		Logger.Infof("Project config created with UUID: %s", projectConfig.Project.UUID)

		// Update user config with project entry
		Logger.Debugf("Updating user config with project entry")
		if userConfig.Projects == nil {
			userConfig.Projects = make(map[string]configs.UserProjectEntry)
		}
		userConfig.Projects[projectConfig.Project.UUID] = configs.UserProjectEntry{
			DeviceName:  deviceName,
			ProjectName: projectName,
		}
		if err := configs.SaveUserConfig(userConfig); err != nil {
			return Logger.ErrorfAndReturn("Failed to update user config with project: %v", err)
		}
		Logger.Infof("User config updated with project UUID: %s -> device: %s, project: %s", projectConfig.Project.UUID, deviceName, projectName)

		// Now initialize project settings (which loads the project config)
		Logger.Debugf("Initializing project settings")
		if err := configs.InitProjectSettings(); err != nil {
			return Logger.ErrorfAndReturn("Failed to init project settings: %v", err)
		}

		Logger.Debugf("Creating and saving RSA key pair")
		if err := secrets.CreateAndSaveRSAKeyPair(verbose); err != nil {
			return Logger.ErrorfAndReturn("Failed to generate and save RSA key pair: %v", err)
		}
		Logger.Infof("RSA key pair created and saved successfully")

		Logger.Debugf("Copying user public key to project")
		destPath, err := secrets.CopyUserPublicKeyToProject()
		_ = destPath // explicitly ignore destPath for now
		if err != nil {
			return Logger.ErrorfAndReturn("Failed to copy public key to project: %v", err)
		}
		Logger.Infof("User public key copied to project successfully")

		Logger.Debugf("Creating and saving encrypted symmetric key")
		if err := secrets.CreateAndSaveEncryptedSymmetricKey(verbose); err != nil {
			return Logger.ErrorfAndReturn("Failed to create encrypted symmetric key: %v", err)
		}
		Logger.Infof("Encrypted symmetric key created and saved successfully")

		Logger.Infof("Init command completed successfully")

		// Log to audit trail.
		auditEntry := audit.LogWithUser("init")
		auditEntry.ProjectName = projectName
		auditEntry.ProjectUUID = projectConfig.Project.UUID
		auditEntry.DeviceName = deviceName
		audit.Log(auditEntry)

		spinner.Stop()
		// Security reminder about .env files
		Logger.WarnfUser("Remember to never commit .env files to version control - only commit .kanuka files")
		spinner.Restart()

		finalMessage := ui.Success.Sprint("✓") + " Kānuka initialized successfully!\n\n" +
			ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets encrypt") + " to encrypt your existing .env files\n\n" +
			ui.Info.Sprint("Tip:") + " Working in a monorepo? You have two options:\n" +
			"  1. Keep this single .kanuka at the root and use selective encryption:\n" +
			"     " + ui.Code.Sprint("kanuka secrets encrypt services/api/.env") + "\n" +
			"  2. Initialize separate .kanuka stores in each service:\n" +
			"     " + ui.Code.Sprint("cd services/api && kanuka secrets init")

		spinner.FinalMSG = finalMessage
		return nil
	},
}
