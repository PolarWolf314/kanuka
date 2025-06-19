package cmd

import (
	logger "github.com/PolarWolf314/kanuka/internal/logging"
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
		Long:  `Provides encryption, decryption, registration, removal, initialization, and purging of secrets.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			Logger = logger.Logger{
				Verbose: verbose,
				Debug:   debug,
			}
			Logger.Debugf("Initializing secrets command with verbose=%t, debug=%t", verbose, debug)
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
	SecretsCmd.AddCommand(removeCmd)
	SecretsCmd.AddCommand(initCmd)
	SecretsCmd.AddCommand(purgeCmd)
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
	// Reset the remove command flags
	resetRemoveCommandState()
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

	// Reset the remove command flags specifically
	if removeCmd != nil && removeCmd.Flags() != nil {
		removeCmd.Flags().VisitAll(func(flag *pflag.Flag) {
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
