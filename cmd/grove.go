package cmd

import (
	logger "github.com/PolarWolf314/kanuka/internal/logging"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	groveVerbose bool
	groveDebug   bool
	GroveLogger  logger.Logger

	GroveCmd = &cobra.Command{
		Use:   "grove",
		Short: "Manage development environments using devenv.nix",
		Long:  `Provides package management and shell environment setup using the devenv ecosystem.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			GroveLogger = logger.Logger{
				Verbose: groveVerbose,
				Debug:   groveDebug,
			}
			GroveLogger.Debugf("Initializing grove command with verbose=%t, debug=%t", groveVerbose, groveDebug)
		},
	}
)

func init() {
	GroveCmd.PersistentFlags().BoolVarP(&groveVerbose, "verbose", "v", false, "enable verbose output")
	GroveCmd.PersistentFlags().BoolVar(&groveDebug, "debug", false, "enable debug output")

	GroveCmd.AddCommand(groveInitCmd)
	GroveCmd.AddCommand(groveAddCmd)
	GroveCmd.AddCommand(groveRemoveCmd)
	GroveCmd.AddCommand(groveListCmd)
	GroveCmd.AddCommand(groveEnterCmd)
	GroveCmd.AddCommand(groveSearchCmd)
}

// Helper functions for testing

// GetGroveCmd returns the GroveCmd for testing.
func GetGroveCmd() *cobra.Command {
	return GroveCmd
}

// ResetGroveGlobalState resets all global variables to their default values for testing.
func ResetGroveGlobalState() {
	groveVerbose = false
	groveDebug = false
	// Reset Cobra flag state to prevent pollution between tests
	resetGroveFlagState()
}

// resetGroveFlagState resets the flag state for grove commands to prevent test pollution.
func resetGroveFlagState() {
	// Reset the grove command flags
	if GroveCmd != nil && GroveCmd.Flags() != nil {
		GroveCmd.Flags().VisitAll(func(flag *pflag.Flag) {
			flag.Changed = false
		})
	}
}

// SetGroveVerbose sets the verbose flag for testing.
func SetGroveVerbose(v bool) {
	groveVerbose = v
}

// SetGroveDebug sets the debug flag for testing.
func SetGroveDebug(d bool) {
	groveDebug = d
}

// SetGroveLogger sets the logger for testing.
func SetGroveLogger(l logger.Logger) {
	GroveLogger = l
}

// GetGroveEnterCmd returns the groveEnterCmd for the dev alias.
func GetGroveEnterCmd() *cobra.Command {
	return groveEnterCmd
}
