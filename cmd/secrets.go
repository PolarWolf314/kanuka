package cmd

import (
	"os"
	"path/filepath"

	"github.com/PolarWolf314/kanuka/internal/configs"
	logger "github.com/PolarWolf314/kanuka/internal/logging"
	"github.com/PolarWolf314/kanuka/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	verbose bool
	debug   bool
	Logger  logger.Logger

	SecretsCmd = &cobra.Command{
		Use:   "secrets",
		Short: "Manage secrets stored in the repository",
		Long:  `	Provides encryption, decryption, registration, revocation, and initialization of secrets.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			Logger = logger.Logger{
				Verbose: verbose,
				Debug:   debug,
			}
			Logger.Debugf("Initializing secrets command with verbose=%t, debug=%t", verbose, debug)

			// Update key metadata access time if in a project.
			updateProjectAccessTime()
		},
	}
)

func init() {
	SecretsCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	SecretsCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable debug output")

	SecretsCmd.AddCommand(encryptCmd)
	SecretsCmd.AddCommand(decryptCmd)
	SecretsCmd.AddCommand(createCmd)
	SecretsCmd.AddCommand(RegisterCmd)
	SecretsCmd.AddCommand(revokeCmd)
	SecretsCmd.AddCommand(initCmd)
	SecretsCmd.AddCommand(syncCmd)
	SecretsCmd.AddCommand(accessCmd)
	SecretsCmd.AddCommand(cleanCmd)
	SecretsCmd.AddCommand(statusCmd)
	SecretsCmd.AddCommand(doctorCmd)
	SecretsCmd.AddCommand(rotateCmd)
}

// Helper functions for testing

// GetSecretsCmd returns the SecretsCmd for testing.
func GetSecretsCmd() *cobra.Command {
	return SecretsCmd
}

// ResetGlobalState resets all global variables to their default values for testing.
func ResetGlobalState() {
	verbose = false
	debug = false
	// Reset the force flag from secrets_create.go
	resetCreateCommandState()
	// Reset the register command flags
	resetRegisterCommandState()
	// Reset the revoke command flags
	resetRevokeCommandState()
	// Reset the init command flags
	resetInitCommandState()
	// Reset the encrypt command flags
	resetEncryptCommandState()
	// Reset the decrypt command flags
	resetDecryptCommandState()
	// Reset the sync command flags
	resetSyncCommandState()
	// Reset the access command flags
	resetAccessCommandState()
	// Reset the clean command flags
	resetCleanCommandState()
	// Reset the status command flags
	resetStatusCommandState()
	// Reset the doctor command flags
	resetDoctorCommandState()
	// Reset the rotate command flags
	resetRotateCommandState()
	// Reset Cobra flag state to prevent pollution between tests
	resetCobraFlagState()
}

// resetCobraFlagState resets the flag state for all commands to prevent test pollution.
func resetCobraFlagState() {
	// Reset the register command flags specifically
	if RegisterCmd != nil && RegisterCmd.Flags() != nil {
		RegisterCmd.Flags().VisitAll(func(flag *pflag.Flag) {
			flag.Changed = false
		})
	}

	// Reset the revoke command flags specifically
	if revokeCmd != nil && revokeCmd.Flags() != nil {
		revokeCmd.Flags().VisitAll(func(flag *pflag.Flag) {
			flag.Changed = false
		})
	}

	// Reset the init command flags specifically
	if initCmd != nil && initCmd.Flags() != nil {
		initCmd.Flags().VisitAll(func(flag *pflag.Flag) {
			flag.Changed = false
		})
	}

	// Reset the encrypt command flags specifically
	if encryptCmd != nil && encryptCmd.Flags() != nil {
		encryptCmd.Flags().VisitAll(func(flag *pflag.Flag) {
			flag.Changed = false
		})
	}

	// Reset the decrypt command flags specifically
	if decryptCmd != nil && decryptCmd.Flags() != nil {
		decryptCmd.Flags().VisitAll(func(flag *pflag.Flag) {
			flag.Changed = false
		})
	}

	// Reset the sync command flags specifically
	if syncCmd != nil && syncCmd.Flags() != nil {
		syncCmd.Flags().VisitAll(func(flag *pflag.Flag) {
			flag.Changed = false
		})
	}

	// Reset the access command flags specifically
	if accessCmd != nil && accessCmd.Flags() != nil {
		accessCmd.Flags().VisitAll(func(flag *pflag.Flag) {
			flag.Changed = false
		})
	}

	// Reset the clean command flags specifically
	if cleanCmd != nil && cleanCmd.Flags() != nil {
		cleanCmd.Flags().VisitAll(func(flag *pflag.Flag) {
			flag.Changed = false
		})
	}

	// Reset the status command flags specifically
	if statusCmd != nil && statusCmd.Flags() != nil {
		statusCmd.Flags().VisitAll(func(flag *pflag.Flag) {
			flag.Changed = false
		})
	}

	// Reset the doctor command flags specifically
	if doctorCmd != nil && doctorCmd.Flags() != nil {
		doctorCmd.Flags().VisitAll(func(flag *pflag.Flag) {
			flag.Changed = false
		})
	}

	// Reset the rotate command flags specifically
	if rotateCmd != nil && rotateCmd.Flags() != nil {
		rotateCmd.Flags().VisitAll(func(flag *pflag.Flag) {
			flag.Changed = false
		})
	}

	// Reset the main secrets command flags
	if SecretsCmd != nil && SecretsCmd.Flags() != nil {
		SecretsCmd.Flags().VisitAll(func(flag *pflag.Flag) {
			flag.Changed = false
		})
	}
}

// SetVerbose sets the verbose flag for testing.
func SetVerbose(v bool) {
	verbose = v
}

// SetDebug sets the debug flag for testing.
func SetDebug(d bool) {
	debug = d
}

// SetLogger sets the logger for testing.
func SetLogger(l logger.Logger) {
	Logger = l
}

// updateProjectAccessTime updates the key metadata access time if running inside a project.
// This is called from PersistentPreRun to track when the project was last accessed.
// Errors are silently ignored as this is a non-critical operation.
// Important: This function avoids calling InitProjectSettings to prevent triggering
// legacy project migration during PersistentPreRun.
func updateProjectAccessTime() {
	// Find project root without initializing settings (which could trigger migration).
	projectPath, err := utils.FindProjectKanukaRoot()
	if err != nil || projectPath == "" {
		// Not in a project - this is fine.
		return
	}

	// Check if config.toml exists (only update access time for properly initialized projects).
	configPath := filepath.Join(projectPath, ".kanuka", "config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// No config.toml - project not properly initialized or is legacy.
		return
	}

	// Load project config directly to get project UUID.
	projectConfig := &configs.ProjectConfig{
		Users:   make(map[string]string),
		Devices: make(map[string]configs.DeviceConfig),
	}
	if err := configs.LoadTOML(configPath, projectConfig); err != nil {
		return
	}

	if projectConfig.Project.UUID == "" {
		return
	}

	// Update the access time - errors are non-critical.
	_ = configs.UpdateKeyMetadataAccessTime(projectConfig.Project.UUID)
}
