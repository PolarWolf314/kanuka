package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/PolarWolf314/kanuka/internal/audit"
	kerrors "github.com/PolarWolf314/kanuka/internal/errors"
	"github.com/PolarWolf314/kanuka/internal/ui"
	"github.com/PolarWolf314/kanuka/internal/workflows"
	"github.com/spf13/cobra"
)

var (
	logLimit     int
	logReverse   bool
	logUser      string
	logOperation string
	logSince     string
	logUntil     string
	logOneline   bool
	logJSON      bool
)

func init() {
	logCmd.Flags().IntVarP(&logLimit, "number", "n", 0, "limit number of entries shown")
	logCmd.Flags().BoolVar(&logReverse, "reverse", false, "show most recent entries first")
	logCmd.Flags().StringVar(&logUser, "user", "", "filter by user email")
	logCmd.Flags().StringVar(&logOperation, "operation", "", "filter by operation type (comma-separated)")
	logCmd.Flags().StringVar(&logSince, "since", "", "show entries after date (YYYY-MM-DD)")
	logCmd.Flags().StringVar(&logUntil, "until", "", "show entries before date (YYYY-MM-DD)")
	logCmd.Flags().BoolVar(&logOneline, "oneline", false, "compact one-line format")
	logCmd.Flags().BoolVar(&logJSON, "json", false, "output as JSON array")

	SecretsCmd.AddCommand(logCmd)
}

// resetLogCommandState resets the log command's global state for testing.
func resetLogCommandState() {
	logLimit = 0
	logReverse = false
	logUser = ""
	logOperation = ""
	logSince = ""
	logUntil = ""
	logOneline = false
	logJSON = false
}

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "View the audit log",
	Long: `Displays the audit log of secrets operations.

Shows who performed what operation and when. Use filters to narrow down
the results.

Examples:
  kanuka secrets log                              # View full log
  kanuka secrets log -n 10                        # Last 10 entries
  kanuka secrets log --reverse                    # Most recent first
  kanuka secrets log --user alice@example.com     # Filter by user
  kanuka secrets log --operation encrypt,decrypt  # Filter by operation
  kanuka secrets log --since 2024-01-01           # Filter by date
  kanuka secrets log --json                       # JSON output`,
	RunE: runLog,
}

func runLog(cmd *cobra.Command, args []string) error {
	Logger.Infof("Starting log command")

	spinner, cleanup := startSpinner("Loading audit log...", verbose)
	defer cleanup()

	opts := workflows.LogOptions{
		Limit:      logLimit,
		Reverse:    logReverse,
		User:       logUser,
		Operations: logOperation,
		Since:      logSince,
		Until:      logUntil,
	}

	result, err := workflows.Log(context.Background(), opts)
	if err != nil {
		spinner.FinalMSG = formatLogError(err)
		if isLogUnexpectedError(err) {
			return err
		}
		return nil
	}

	Logger.Debugf("Parsed %d entries from audit log", result.TotalEntriesBeforeFilter)
	Logger.Debugf("After filtering: %d entries", len(result.Entries))

	if len(result.Entries) == 0 {
		if result.TotalEntriesBeforeFilter == 0 {
			spinner.FinalMSG = ""
			fmt.Println("No audit log entries found.")
		} else {
			spinner.FinalMSG = ""
			fmt.Println("No audit log entries found matching the filters.")
		}
		return nil
	}

	// Output.
	spinner.FinalMSG = ""
	if logJSON {
		if err := outputLogJSON(result.Entries); err != nil {
			return err
		}
		return nil
	}

	if logOneline {
		outputLogOneline(result.Entries)
		return nil
	}

	outputLogDefault(result.Entries)
	return nil
}

// formatLogError formats a log error for display to the user.
func formatLogError(err error) string {
	switch {
	case errors.Is(err, kerrors.ErrProjectNotInitialized):
		return ui.Error.Sprint("✗") + " Kānuka has not been initialized\n" +
			ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " first"

	case errors.Is(err, kerrors.ErrNoFilesFound):
		return ui.Info.Sprint("ℹ") + " No audit log found. Operations will be logged after running any secrets command.\n"

	case errors.Is(err, kerrors.ErrInvalidDateFormat):
		return ui.Error.Sprint("✗") + " " + err.Error()

	default:
		return ui.Error.Sprint("✗") + " Failed to read audit log: " + err.Error()
	}
}

// isLogUnexpectedError returns true if the error is unexpected and should cause a non-zero exit.
func isLogUnexpectedError(err error) bool {
	switch {
	case errors.Is(err, kerrors.ErrProjectNotInitialized),
		errors.Is(err, kerrors.ErrNoFilesFound),
		errors.Is(err, kerrors.ErrInvalidDateFormat):
		return false
	default:
		return true
	}
}

func outputLogJSON(entries []audit.Entry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal entries to JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func outputLogOneline(entries []audit.Entry) {
	for _, e := range entries {
		date := workflows.FormatDate(e.Timestamp)
		details := workflows.FormatDetailsOneline(e)
		fmt.Printf("%s %s %s %s\n", date, e.User, e.Operation, details)
	}
}

func outputLogDefault(entries []audit.Entry) {
	for _, e := range entries {
		datetime := workflows.FormatDateTime(e.Timestamp)
		details := workflows.FormatDetails(e)
		fmt.Printf("%-19s  %-25s  %-10s  %s\n", datetime, e.User, e.Operation, details)
	}
}
